package tools

import (
	"context"
	"testing"
)

// MockTool is a mock implementation of Tool for testing.
type MockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params map[string]any) (Result, error)
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return m.description }
func (m *MockTool) Execute(ctx context.Context, params map[string]any) (Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return Result{Success: true, Output: "mock output"}, nil
}

// NewMockRegistry creates a registry with mock tools for testing.
func NewMockRegistry() *Registry {
	r := NewRegistry()
	
	// Register mock tools
	r.Register(&MockTool{
		name:        "mock_read",
		description: "Mock read tool",
		executeFunc: func(ctx context.Context, params map[string]any) (Result, error) {
			path, _ := params["path"].(string)
			return Result{
				Success: true,
				Output:  "mock content of " + path,
				Data:    map[string]any{"path": path},
			}, nil
		},
	})
	
	r.Register(&MockTool{
		name:        "mock_write",
		description: "Mock write tool",
		executeFunc: func(ctx context.Context, params map[string]any) (Result, error) {
			return Result{Success: true, Output: "written"}, nil
		},
	})
	
	r.Register(&MockTool{
		name:        "mock_fail",
		description: "Mock tool that always fails",
		executeFunc: func(ctx context.Context, params map[string]any) (Result, error) {
			return Result{}, &RegistryError{Op: "execute", Tool: "mock_fail", Err: ErrInvalidParams}
		},
	})
	
	return r
}

// AssertResultSuccess checks that a result indicates success.
func AssertResultSuccess(t *testing.T, result Result, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Error("expected success result, got failure")
	}
}

// AssertResultError checks that an error is returned.
func AssertResultError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if expectedMsg != "" && err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}
