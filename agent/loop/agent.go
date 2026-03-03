package loop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vigo999/ms-cli/tools"
	"github.com/vigo999/ms-cli/tools/fs"
	"github.com/vigo999/ms-cli/tools/shell"
)

// AgentLoop manages the rule-based execution cycle (fallback when no LLM available).
type AgentLoop struct {
	registry *tools.Registry
	maxSteps int
	timeout  time.Duration
}

// NewAgentLoop creates a new agent loop with the given tool registry.
func NewAgentLoop(registry *tools.Registry) *AgentLoop {
	return &AgentLoop{
		registry: registry,
		maxSteps: 10,
		timeout:  5 * time.Minute,
	}
}

// SetMaxSteps sets the maximum number of execution steps.
func (a *AgentLoop) SetMaxSteps(n int) {
	a.maxSteps = n
}

// SetTimeout sets the execution timeout.
func (a *AgentLoop) SetTimeout(d time.Duration) {
	a.timeout = d
}

// Run executes a task and returns a sequence of events (simplified fallback implementation).
func (a *AgentLoop) Run(task Task) ([]Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	events := make([]Event, 0)

	// Initial event: task received
	events = append(events, Event{
		Type:    "task_start",
		Message: task.Description,
	})

	// Simple rule-based execution (fallback when no LLM available)
	result := a.executeRuleBased(ctx, task)
	events = append(events, result...)

	return events, nil
}

// executeRuleBased performs simple rule-based task execution.
func (a *AgentLoop) executeRuleBased(ctx context.Context, task Task) []Event {
	events := make([]Event, 0)
	lower := strings.ToLower(task.Description)

	// Simple pattern matching
	if strings.Contains(lower, "read") || strings.Contains(lower, "show") || strings.Contains(lower, "cat ") {
		path := a.extractPath(task.Description)
		if path != "" {
			events = append(events, Event{
				Type:    "thought",
				Message: fmt.Sprintf("I need to read the file at %s", path),
			})
			events = append(events, Event{
				Type:    "action_start",
				Message: fmt.Sprintf("fs_read: {\"path\": \"%s\"}", path),
			})

			result, err := a.registry.Execute(ctx, "fs_read", map[string]any{"path": path})
			if err != nil {
				events = append(events, Event{
					Type:    "action_error",
					Message: err.Error(),
				})
			} else {
				events = append(events, Event{
					Type:    "action_result",
					Message: result.Output,
				})
				events = append(events, Event{
					Type:    "complete",
					Message: fmt.Sprintf("File contents:\n%s", result.Output),
				})
			}
			return events
		}
	}

	if strings.Contains(lower, "list") || strings.Contains(lower, "find") || strings.Contains(lower, "ls") {
		events = append(events, Event{
			Type:    "thought",
			Message: "I'll search for files matching the pattern",
		})
		events = append(events, Event{
			Type:    "action_start",
			Message: "fs_glob: {\"pattern\": \"*\"}",
		})

		result, err := a.registry.Execute(ctx, "fs_glob", map[string]any{"pattern": "*"})
		if err != nil {
			events = append(events, Event{
				Type:    "action_error",
				Message: err.Error(),
			})
		} else {
			events = append(events, Event{
				Type:    "action_result",
				Message: result.Output,
			})
			events = append(events, Event{
				Type:    "complete",
				Message: fmt.Sprintf("Found files:\n%s", result.Output),
			})
		}
		return events
	}

	// Default response
	events = append(events, Event{
		Type:    "complete",
		Message: fmt.Sprintf("Task received: %s\n(I'm a rule-based fallback agent. Please configure an LLM for full functionality.)", task.Description),
	})

	return events
}

// extractPath extracts a file path from the task description.
func (a *AgentLoop) extractPath(desc string) string {
	words := strings.Fields(desc)
	for _, word := range words {
		if strings.Contains(word, ".") && !strings.Contains(word, ":") {
			return strings.Trim(word, `"',.!?`)
		}
	}
	return ""
}

// ToolDefinitions returns all tool definitions for LLM function calling.
func ToolDefinitions() []tools.Definition {
	return []tools.Definition{
		fs.ReadDefinition(),
		fs.WriteDefinition(),
		fs.EditDefinition(),
		fs.GlobDefinition(),
		fs.GrepDefinition(),
		shell.ExecDefinition(),
	}
}
