package tools

import (
	"context"
	"fmt"
)

// Tool is the interface for all executable tools.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (Result, error)
}

// Result is the output of a tool execution.
type Result struct {
	Success bool
	Output  string
	Error   error
	Data    map[string]any
}

// Registry holds all available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool by name with given parameters.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (Result, error) {
	tool, ok := r.Get(name)
	if !ok {
		return Result{}, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, params)
}

// Definition represents a tool's schema for LLM function calling.
type Definition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  Parameters  `json:"parameters"`
}

// Parameters defines the JSON schema for tool parameters.
type Parameters struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

// Property is a single parameter property.
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToDefinition converts a Tool to its schema definition.
func ToDefinition(tool Tool) Definition {
	return Definition{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters: Parameters{
			Type:       "object",
			Properties: make(map[string]Property),
			Required:   []string{},
		},
	}
}
