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
	Data    map[string]any
}

// RegistryError represents an error from the tool registry.
type RegistryError struct {
	Op   string // Operation being performed
	Tool string // Tool name if applicable
	Err  error  // Underlying error
}

func (e *RegistryError) Error() string {
	if e.Tool != "" {
		return fmt.Sprintf("registry %s for tool %s: %v", e.Op, e.Tool, e.Err)
	}
	return fmt.Sprintf("registry %s: %v", e.Op, e.Err)
}

func (e *RegistryError) Unwrap() error {
	return e.Err
}

// IsNotFound returns true if the error is a "tool not found" error.
func IsNotFound(err error) bool {
	if e, ok := err.(*RegistryError); ok {
		return e.Op == "lookup" && e.Err == ErrToolNotFound
	}
	return false
}

// Common errors
var (
	ErrToolNotFound = fmt.Errorf("tool not found")
	ErrInvalidParams = fmt.Errorf("invalid parameters")
)

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
func (r *Registry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, &RegistryError{Op: "lookup", Tool: name, Err: ErrToolNotFound}
	}
	return tool, nil
}

// MustGet retrieves a tool by name, panicking if not found.
func (r *Registry) MustGet(name string) Tool {
	tool, err := r.Get(name)
	if err != nil {
		panic(err)
	}
	return tool
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
	tool, err := r.Get(name)
	if err != nil {
		return Result{}, err
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
