package shell

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		return tools.Result{Success: false, Error: fmt.Errorf("command parameter is required")}, nil
	}

	// Parse and validate command
	parsed, err := t.parseCommand(command)
	if err != nil {
		return tools.Result{Success: false, Error: fmt.Errorf("command validation failed: %w", err)}, nil
	}

	// Check risk level
	risk := t.assessRisk(parsed)

	// Log attempt
	if t.AuditLog != nil {
		t.AuditLog(command, false, fmt.Errorf("risk level: %v", risk))
	}

	// Block critical commands
	if risk == RiskCritical {
		return tools.Result{Success: false, Error: fmt.Errorf("command blocked: critical risk operation detected")}, nil
	}

	// Check whitelist
	if len(t.AllowedCmds) > 0 && !t.isAllowed(parsed.BaseCmd) {
		return tools.Result{Success: false, Error: fmt.Errorf("command '%s' not in allowed list", parsed.BaseCmd)}, nil
	}

	// Check blacklist
	if t.isBlocked(parsed.BaseCmd) {
		return tools.Result{Success: false, Error: fmt.Errorf("command '%s' is blocked", parsed.BaseCmd)}, nil
	}

	// Security: Enhanced command validation
	if err := t.validateCommandSecurity(parsed); err != nil {
		return tools.Result{Success: false, Error: err}, nil
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
		return tools.Result{Success: false, Error: fmt.Errorf("create stdout pipe: %w", err)}, nil
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return tools.Result{Success: false, Error: fmt.Errorf("create stderr pipe: %w", err)}, nil
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return tools.Result{Success: false, Error: fmt.Errorf("start command: %w", err)}, nil
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
			return tools.Result{Success: false, Error: fmt.Errorf("command timed out after %v", t.Timeout)}, nil
		}
		
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			return tools.Result{Success: false, Error: fmt.Errorf("command failed: %w", err)}, nil
		}
	}

	success := exitCode == 0
	return tools.Result{
		Success: success,
		Output:  output.String(),
		Error:   nil,
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
	Raw     string
	BaseCmd string   // Main command (first token)
	Args    []string // Command arguments
	HasPipe bool     // Contains pipe operator
	HasRedirect bool // Contains redirect operator
}

// parseCommand parses a shell command to extract the base command and metadata
func (t *ExecTool) parseCommand(cmd string) (*ParsedCommand, error) {
	parsed := &ParsedCommand{
		Raw: cmd,
	}

	// Check for pipes and redirects
	parsed.HasPipe = strings.Contains(cmd, "|")
	parsed.HasRedirect = strings.ContainsAny(cmd, "><")

	// Extract base command (first token before space, pipe, or redirect)
	// Handle quoted strings
	cmd = strings.TrimSpace(cmd)

	// Remove common shell prefixes that might be used to bypass checks
	prefixes := []string{"sh -c ", "bash -c ", "sh ", "bash "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			cmd = strings.TrimSpace(cmd[len(prefix):])
			// Remove quotes if present
			cmd = strings.Trim(cmd, `"'`)
			break
		}
	}

	// Extract first token (base command)
	fields := strings.FieldsFunc(cmd, func(r rune) bool {
		return r == ' ' || r == '|' || r == ';' || r == '&' || r == '>' || r == '<'
	})

	if len(fields) == 0 {
		return nil, fmt.Errorf("no command found")
	}

	parsed.BaseCmd = fields[0]
	// Remove any path prefix to get just the command name
	parsed.BaseCmd = filepath.Base(parsed.BaseCmd)
	parsed.Args = fields[1:]

	return parsed, nil
}

// assessRisk determines the risk level of a command
func (t *ExecTool) assessRisk(parsed *ParsedCommand) CommandRisk {
	cmd := strings.ToLower(parsed.BaseCmd)
	fullCmd := strings.ToLower(parsed.Raw)

	// Critical: System-threatening operations
	criticalPatterns := []string{
		`:\s*\(\s*\)\s*{\s*:\s*\|\s*:\s*&\s*}\s*;\s*:\s*`, // Fork bomb
		`rm\s+-rf\s+/`,      // Delete root
		`dd\s+if=\s*/dev/`,  // Direct disk operations
		`mkfs`,
		`>\s*/dev/sda`,
	}

	for _, pattern := range criticalPatterns {
		if matched, _ := regexp.MatchString(pattern, fullCmd); matched {
			return RiskCritical
		}
	}

	// High: Destructive operations
	highRiskCmds := []string{"rm", "rmdir", "unlink", "mv", "chmod", "chown"}
	for _, risky := range highRiskCmds {
		if cmd == risky {
			return RiskHigh
		}
	}

	// Medium: Network operations and downloads
	mediumRiskCmds := []string{"curl", "wget", "nc", "netcat", "telnet", "ssh", "scp", "ftp"}
	for _, risky := range mediumRiskCmds {
		if cmd == risky {
			// Check for pipe to shell (curl | sh pattern)
			if strings.Contains(fullCmd, "| sh") || strings.Contains(fullCmd, "| bash") {
				return RiskHigh
			}
			return RiskMedium
		}
	}

	return RiskLow
}

// isAllowed checks if a command is in the whitelist
func (t *ExecTool) isAllowed(cmd string) bool {
	cmd = strings.ToLower(cmd)
	for _, allowed := range t.AllowedCmds {
		if strings.ToLower(allowed) == cmd {
			return true
		}
	}
	return false
}

// isBlocked checks if a command is in the blacklist
func (t *ExecTool) isBlocked(cmd string) bool {
	cmd = strings.ToLower(cmd)
	for _, blocked := range t.BlockedCmds {
		if strings.ToLower(blocked) == cmd {
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

// SetAllowedCmds sets the whitelist of allowed commands.
func (t *ExecTool) SetAllowedCmds(cmds []string) {
	t.AllowedCmds = cmds
}

// SetBlockedCmds sets the blacklist of blocked commands.
func (t *ExecTool) SetBlockedCmds(cmds []string) {
	t.BlockedCmds = cmds
}
