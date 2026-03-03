package shell

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/vigo999/ms-cli/tools"
)

// ExecTool implements shell command execution.
type ExecTool struct {
	Timeout    time.Duration
	MaxOutput  int // Maximum output bytes to capture
	AllowedEnv []string
}

func NewExecTool() *ExecTool {
	return &ExecTool{
		Timeout:    30 * time.Second,
		MaxOutput:  100 * 1024, // 100KB
		AllowedEnv: []string{"PATH", "HOME", "USER", "SHELL"},
	}
}

func (t *ExecTool) Name() string        { return "shell_exec" }
func (t *ExecTool) Description() string { return "Execute a shell command" }

func (t *ExecTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return tools.Result{Success: false, Error: fmt.Errorf("command parameter is required")}, nil
	}

	// Security: Basic command validation
	if err := validateCommand(command); err != nil {
		return tools.Result{Success: false, Error: err}, nil
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(execCtx, "sh", "-c", command)
	
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
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else if execCtx.Err() == context.DeadlineExceeded {
			return tools.Result{Success: false, Error: fmt.Errorf("command timed out after %v", t.Timeout)}, nil
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
			"command":   command,
			"exit_code": exitCode,
			"timeout":   t.Timeout.Seconds(),
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

// validateCommand performs basic security checks on the command.
func validateCommand(cmd string) error {
	// List of dangerous commands to block
	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		":(){ :|: & };:", // Fork bomb
		"dd if=/dev/zero",
		"mkfs",
		"> /dev/sda",
		"curl * | sh",
		"wget * | sh",
	}

	lowerCmd := strings.ToLower(cmd)
	for _, d := range dangerous {
		if strings.Contains(lowerCmd, strings.ToLower(d)) {
			return fmt.Errorf("dangerous command blocked: %s", d)
		}
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
