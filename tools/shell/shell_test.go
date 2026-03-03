package shell

import (
	"context"
	"testing"
	"time"
)

func TestExecTool(t *testing.T) {
	tool := NewExecTool()
	tool.SetTimeout(5 * time.Second)

	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "echo hello",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Output)
	}
	if result.Output != "hello" {
		t.Errorf("expected 'hello', got '%s'", result.Output)
	}
}

func TestExecToolInvalidCommand(t *testing.T) {
	tool := NewExecTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "nonexistent_command_12345",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should fail but not error
	if result.Success {
		t.Error("expected failure for invalid command")
	}
}

func TestExecToolEmptyCommand(t *testing.T) {
	tool := NewExecTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for empty command")
	}
}

func TestExecToolTimeout(t *testing.T) {
	tool := NewExecTool()
	tool.SetTimeout(100 * time.Millisecond)

	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "sleep 10",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure due to timeout")
	}
	if result.Error == nil {
		t.Error("expected timeout error in result")
	}
}

func TestExecToolDangerousCommand(t *testing.T) {
	tool := NewExecTool()

	dangerousCommands := []string{
		"rm -rf /",
		":(){ :|: & };:",
		"mkfs",
	}

	for _, cmd := range dangerousCommands {
		result, err := tool.Execute(context.Background(), map[string]any{
			"command": cmd,
		})

		if err != nil {
			t.Errorf("unexpected error for '%s': %v", cmd, err)
		}
		if result.Success {
			t.Errorf("expected failure for dangerous command: %s", cmd)
		}
	}
}
