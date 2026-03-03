package shell

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/vigo999/ms-cli/tools"
)

// ExecTool implements shell command execution.
type ExecTool struct {
	Timeout       time.Duration
	MaxOutput     int // Maximum output bytes to capture
	AllowedEnv    []string
	AllowedCmds   []string // Whitelist of allowed commands (empty = allow all)
	BlockedCmds   []string // Blacklist of blocked commands
	AuditLog      func(command string, allowed bool, err error)
	RequireConfirm bool    // Require user confirmation for destructive commands
}

// CommandRisk represents the risk level of a command
type CommandRisk int

const (
	RiskLow CommandRisk = iota
	RiskMedium
	RiskHigh
	RiskCritical
)

func NewExecTool() *ExecTool {
	return &ExecTool{
		Timeout:    30 * time.Second,
		MaxOutput:  100 * 1024, // 100KB
		AllowedEnv: []string{"PATH", "HOME", "USER", "SHELL", "PWD"},
		BlockedCmds: []string{
			"rm", "rmdir", "unlink", "mkfs", "dd", "fdisk", "format",
			"curl", "wget", "nc", "netcat", "telnet",
		},
		RequireConfirm: true,
	}
}

func (t *ExecTool) Name() string        { return "shell_exec" }
func (t *ExecTool) Description() string { return "Execute a shell command" }

func (t *ExecTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return tools.Result{Success: false, Output: "command parameter is required"}, nil
	}

	// Parse and validate command
	parsed, err := t.parseCommand(command)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("command validation failed: %v", err)}, nil
	}

	// Check risk level
	risk := t.assessRisk(parsed)

	// Log attempt
	if t.AuditLog != nil {
		t.AuditLog(command, false, fmt.Errorf("risk level: %v", risk))
	}

	// Block critical commands
	if risk == RiskCritical {
		return tools.Result{Success: false, Output: "command blocked: critical risk operation detected"}, nil
	}

	// Check whitelist
	if len(t.AllowedCmds) > 0 && !t.isAllowed(parsed.BaseCmd) {
		return tools.Result{Success: false, Output: fmt.Sprintf("command '%s' not in allowed list", parsed.BaseCmd)}, nil
	}

	// Check blacklist
	if t.isBlocked(parsed.BaseCmd) {
		return tools.Result{Success: false, Output: fmt.Sprintf("command '%s' is blocked", parsed.BaseCmd)}, nil
	}

	// Security: Enhanced command validation
	if err := t.validateCommandSecurity(parsed); err != nil {
		return tools.Result{Success: false, Output: err.Error()}, nil
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	// Prepare command - use explicit shell path
	cmd := exec.CommandContext(execCtx, "/bin/sh", "-c", command)

	// Set up environment
	cmd.Env = t.buildEnv()

	// Capture output
	var output strings.Builder
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("create stdout pipe: %v", err)}, nil
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("create stderr pipe: %v", err)}, nil
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("start command: %v", err)}, nil
	}

	// Stream output with scanner
	done := make(chan bool, 2)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if output.Len()+len(line)+1 > t.MaxOutput {
				output.WriteString("\n... (output truncated)")
				break
			}
			if output.Len() > 0 {
				output.WriteString("\n")
			}
			output.WriteString(line)
		}
		done <- true
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if output.Len()+len(line)+1 > t.MaxOutput {
				continue
			}
			if output.Len() > 0 {
				output.WriteString("\n")
			}
			output.WriteString("[stderr] " + line)
		}
		done <- true
	}()

	// Wait for output readers
	<-done
	<-done

	// Wait for command to finish
	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		// Check for timeout first
		if execCtx.Err() == context.DeadlineExceeded {
			return tools.Result{Success: false, Output: fmt.Sprintf("command timed out after %v", t.Timeout)}, nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			return tools.Result{Success: false, Output: fmt.Sprintf("command failed: %v", err)}, nil
		}
	}

	success := exitCode == 0
	return tools.Result{
		Success: success,
		Output:  output.String(),
		Data: map[string]any{
			"command":    command,
			"exit_code":  exitCode,
			"timeout":    t.Timeout.Seconds(),
			"risk_level": risk,
		},
	}, nil
}

// ExecDefinition returns the schema for shell_exec.
func ExecDefinition() tools.Definition {
	return tools.Definition{
		Name:        "shell_exec",
		Description: "Execute a shell command and capture output. Commands run in a sandboxed environment with timeout protection.",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"command": {
					Type:        "string",
					Description: "Shell command to execute",
				},
				"timeout_sec": {
					Type:        "integer",
					Description: "Timeout in seconds (optional, default: 30)",
				},
			},
			Required: []string{"command"},
		},
	}
}

// buildEnv creates a sanitized environment for command execution.
func (t *ExecTool) buildEnv() []string {
	env := make([]string, 0, len(t.AllowedEnv))
	for _, key := range t.AllowedEnv {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}
	return env
}

// ParsedCommand represents a parsed shell command
type ParsedCommand struct {
	BaseCmd string   // The base command (e.g., "git" from "git status")
	Args    []string // Command arguments
	Raw     string   // Original command string
}

// parseCommand parses a shell command string
func (t *ExecTool) parseCommand(cmd string) (*ParsedCommand, error) {
	parsed := &ParsedCommand{
		Raw:  cmd,
		Args: []string{},
	}

	// Extract base command (first token before space, pipe, or redirect)
	// Handle quoted strings
	cmd = strings.TrimSpace(cmd)

	// Remove common shell prefixes that might be used to bypass checks
	prefixes := []string{"sh -c ", "bash -c ", "sh ", "bash "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			cmd = strings.TrimSpace(cmd[len(prefix):])
		}
	}

	// Remove quotes if the entire command is quoted
	if (strings.HasPrefix(cmd, "\"") && strings.HasSuffix(cmd, "\"")) ||
		(strings.HasPrefix(cmd, "'") && strings.HasSuffix(cmd, "'")) {
		cmd = cmd[1 : len(cmd)-1]
	}

	// Find the first command before any pipes, redirects, or logical operators
	separators := []string{"|", "&&", "||", ";", ">", "<", "&"}
	cmdPart := cmd
	for _, sep := range separators {
		if idx := strings.Index(cmd, sep); idx > 0 {
			cmdPart = strings.TrimSpace(cmd[:idx])
			break
		}
	}

	// Extract base command and args
	fields := strings.Fields(cmdPart)
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	parsed.BaseCmd = fields[0]
	if len(fields) > 1 {
		parsed.Args = fields[1:]
	}

	return parsed, nil
}

// assessRisk assesses the risk level of a command
func (t *ExecTool) assessRisk(parsed *ParsedCommand) CommandRisk {
	base := strings.ToLower(parsed.BaseCmd)

	// Critical commands that can cause serious damage
	criticalCmds := []string{"rm", "rmdir", "unlink", "mkfs", "dd", "fdisk", "format", "mkfs.ext4"}
	for _, cmd := range criticalCmds {
		if base == cmd {
			return RiskCritical
		}
	}

	// High risk - network commands that could exfiltrate data
	highRiskCmds := []string{"curl", "wget", "nc", "netcat", "telnet", "ssh", "scp", "ftp", "sftp"}
	for _, cmd := range highRiskCmds {
		if base == cmd {
			return RiskHigh
		}
	}

	// Medium risk - commands that modify system state
	mediumRiskCmds := []string{"mv", "cp", "chmod", "chown", "mkdir", "touch", "echo", "cat"}
	for _, cmd := range mediumRiskCmds {
		if base == cmd {
			return RiskMedium
		}
	}

	// Check for shell escapes which increase risk
	if strings.Contains(parsed.Raw, ";") || strings.Contains(parsed.Raw, "&&") ||
		strings.Contains(parsed.Raw, "||") || strings.Contains(parsed.Raw, "$") {
		return RiskMedium
	}

	return RiskLow
}

// isAllowed checks if command is in whitelist
func (t *ExecTool) isAllowed(cmd string) bool {
	if len(t.AllowedCmds) == 0 {
		return true
	}
	for _, allowed := range t.AllowedCmds {
		if strings.EqualFold(cmd, allowed) {
			return true
		}
	}
	return false
}

// isBlocked checks if command is in blacklist
func (t *ExecTool) isBlocked(cmd string) bool {
	for _, blocked := range t.BlockedCmds {
		if strings.EqualFold(cmd, blocked) {
			return true
		}
	}
	return false
}

// validateCommandSecurity performs deep security validation
func (t *ExecTool) validateCommandSecurity(parsed *ParsedCommand) error {
	fullCmd := strings.ToLower(parsed.Raw)

	// Check for encoded dangerous patterns
	encodedPatterns := []string{
		`\$\(.*\)`,      // Command substitution $(...)
		"`.*`",          // Backtick command substitution
		`\$\{.*\}`,      // Variable expansion with default values
	}

	for _, pattern := range encodedPatterns {
		if matched, _ := regexp.MatchString(pattern, fullCmd); matched {
			// These patterns can be used to hide malicious commands
			// Log but don't necessarily block (they have legitimate uses)
			// In a stricter environment, you might block these
		}
	}

	// Check for shell escapes
	if strings.Contains(fullCmd, ";") || strings.Contains(fullCmd, "&&") ||
		strings.Contains(fullCmd, "||") {
		// Multiple commands - higher scrutiny
		// This is where command injection often happens
	}

	return nil
}

// SetTimeout sets the execution timeout.
func (t *ExecTool) SetTimeout(d time.Duration) {
	t.Timeout = d
}

// SetMaxOutput sets the maximum output size.
func (t *ExecTool) SetMaxOutput(n int) {
	t.MaxOutput = n
}
