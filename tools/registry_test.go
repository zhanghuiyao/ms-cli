package tools

import (
	"context"
	"testing"
)

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	// Create a mock tool
	mockTool := &mockTool{name: "test_tool"}

	// Register
	reg.Register(mockTool)

	// List
	tools := reg.List()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	// Get
	tool, err := reg.Get("test_tool")
	if err != nil {
		t.Error("expected to find tool")
	}
	if tool.Name() != "test_tool" {
		t.Errorf("expected name 'test_tool', got %s", tool.Name())
	}

	// Execute
	result, err := reg.Execute(context.Background(), "test_tool", map[string]any{"key": "value"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestRegistryNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "mock tool" }
func (m *mockTool) Execute(ctx context.Context, params map[string]any) (Result, error) {
	return Result{Success: true, Output: "mock result"}, nil
}
