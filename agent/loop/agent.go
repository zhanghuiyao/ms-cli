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

// Step represents a single step in the agent's execution.
type Step struct {
	Number      int
	Thought     string
	Action      *Action
	Observation string
	Timestamp   time.Time
}

// Action represents a tool call to be executed.
type Action struct {
	Tool   string
	Params map[string]any
}

// AgentLoop manages the ReAct execution cycle.
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

// Run executes a task and returns a sequence of events.
func (a *AgentLoop) Run(task Task) ([]Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	steps := make([]Step, 0)
	events := make([]Event, 0)

	// Initial event: task received
	events = append(events, Event{
		Type:    "task_start",
		Message: task.Description,
	})

	// ReAct loop
	for stepNum := 1; stepNum <= a.maxSteps; stepNum++ {
		select {
		case <-ctx.Done():
			events = append(events, Event{
				Type:    "error",
				Message: "execution timed out",
			})
			return events, ctx.Err()
		default:
		}

		// Build context from previous steps
		contextStr := a.buildContext(task, steps)

		// Generate thought and action (simulated for now)
		thought, action, err := a.think(ctx, contextStr)
		if err != nil {
			events = append(events, Event{
				Type:    "error",
				Message: fmt.Sprintf("thinking failed: %v", err),
			})
			return events, err
		}

		step := Step{
			Number:    stepNum,
			Thought:   thought,
			Action:    action,
			Timestamp: time.Now(),
		}

		// Emit thinking event
		events = append(events, Event{
			Type:    "thought",
			Message: thought,
		})

		// If no action, we're done
		if action == nil {
			events = append(events, Event{
				Type:    "complete",
				Message: thought,
			})
			return events, nil
		}

		// Execute action
		events = append(events, Event{
			Type:    "action_start",
			Message: fmt.Sprintf("%s: %v", action.Tool, action.Params),
		})

		result, err := a.executeAction(ctx, action)
		if err != nil {
			step.Observation = fmt.Sprintf("Error: %v", err)
			events = append(events, Event{
				Type:    "action_error",
				Message: err.Error(),
			})
		} else {
			step.Observation = result.Output
			events = append(events, Event{
				Type:    "action_result",
				Message: result.Output,
			})
		}

		steps = append(steps, step)

		// Check if we're done (based on action)
		if a.isComplete(action, result) {
			events = append(events, Event{
				Type:    "complete",
				Message: "Task completed successfully",
			})
			return events, nil
		}
	}

	events = append(events, Event{
		Type:    "error",
		Message: fmt.Sprintf("exceeded maximum steps (%d)", a.maxSteps),
	})
	return events, fmt.Errorf("max steps exceeded")
}

// buildContext creates a context string from the task and previous steps.
func (a *AgentLoop) buildContext(task Task, steps []Step) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task: %s\n\n", task.Description))

	if len(steps) > 0 {
		sb.WriteString("Previous actions:\n")
		for _, step := range steps {
			sb.WriteString(fmt.Sprintf("%d. Thought: %s\n", step.Number, step.Thought))
			if step.Action != nil {
				sb.WriteString(fmt.Sprintf("   Action: %s\n", step.Action.Tool))
			}
			if step.Observation != "" {
				sb.WriteString(fmt.Sprintf("   Result: %s\n", step.Observation))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Available tools:\n")
	for _, name := range a.registry.List() {
		sb.WriteString(fmt.Sprintf("- %s\n", name))
	}

	return sb.String()
}

// think generates the next thought and action based on context.
// This is a simplified implementation - in production, this would call an LLM.
func (a *AgentLoop) think(ctx context.Context, context string) (string, *Action, error) {
	// Parse task description to determine simple actions
	// This is a rule-based fallback when no LLM is available

	lower := strings.ToLower(context)

	// Simple pattern matching for file operations
	if strings.Contains(lower, "read") || strings.Contains(lower, "show") {
		// Extract file path from task
		path := extractFilePath(context)
		if path != "" {
			return fmt.Sprintf("I need to read the file at %s", path), &Action{
				Tool: "fs_read",
				Params: map[string]any{
					"path": path,
				},
			}, nil
		}
	}

	if strings.Contains(lower, "list") || strings.Contains(lower, "find files") {
		pattern := "*"
		if strings.Contains(lower, ".go") {
			pattern = "*.go"
		} else if strings.Contains(lower, ".yaml") || strings.Contains(lower, ".yml") {
			pattern = "*.yaml"
		}
		return fmt.Sprintf("I'll search for files matching %s", pattern), &Action{
			Tool: "fs_glob",
			Params: map[string]any{
				"pattern": pattern,
			},
		}, nil
	}

	if strings.Contains(lower, "search") || strings.Contains(lower, "grep") {
		// Try to extract pattern
		pattern := extractPattern(context)
		if pattern != "" {
			return fmt.Sprintf("I'll search for '%s' in the codebase", pattern), &Action{
				Tool: "fs_grep",
				Params: map[string]any{
					"pattern": pattern,
					"path":    ".",
				},
			}, nil
		}
	}

	if strings.Contains(lower, "run") || strings.Contains(lower, "execute") {
		cmd := extractCommand(context)
		if cmd != "" {
			return fmt.Sprintf("I'll execute the command: %s", cmd), &Action{
				Tool: "shell_exec",
				Params: map[string]any{
					"command": cmd,
				},
			}, nil
		}
	}

	// Default: complete the task
	return "I've completed the analysis of the task", nil, nil
}

// executeAction runs a single tool action.
func (a *AgentLoop) executeAction(ctx context.Context, action *Action) (tools.Result, error) {
	return a.registry.Execute(ctx, action.Tool, action.Params)
}

// isComplete determines if the task is complete based on the action.
func (a *AgentLoop) isComplete(action *Action, result tools.Result) bool {
	// Task is complete if there's no action or action is "done"
	if action == nil {
		return true
	}
	return false
}

// Helper functions for simple pattern extraction

func extractFilePath(context string) string {
	// Look for quoted strings or paths after keywords
	patterns := []string{
		`"([^"]+\.(go|yaml|yml|json|md|txt))"`,
		`'([^']+\.(go|yaml|yml|json|md|txt))'`,
		`path[:\s]+(\S+\.(go|yaml|yml|json|md|txt))`,
		`file[:\s]+(\S+\.(go|yaml|yml|json|md|txt))`,
	}

	for _, pattern := range patterns {
		// Simple string matching for now
		if idx := strings.Index(context, ".go"); idx > 0 {
			start := idx
			for start > 0 && context[start-1] != ' ' && context[start-1] != '\n' {
				start--
			}
			end := idx + 3
			return strings.TrimSpace(context[start:end])
		}
	}

	// Try to find any file-like string
	words := strings.Fields(context)
	for _, word := range words {
		if strings.Contains(word, ".") && !strings.Contains(word, ":") {
			return strings.Trim(word, `"'.,`)
		}
	}

	return ""
}

func extractPattern(context string) string {
	// Look for quoted strings after "search for" or similar
	if idx := strings.Index(strings.ToLower(context), "search for"); idx >= 0 {
		after := context[idx+10:]
		after = strings.TrimSpace(after)
		if len(after) > 0 {
			// Get first word or quoted string
			if after[0] == '"' || after[0] == '\'' {
				quote := after[0]
				end := strings.Index(after[1:], string(quote))
				if end > 0 {
					return after[1 : end+1]
				}
			}
			fields := strings.Fields(after)
			if len(fields) > 0 {
				return strings.Trim(fields[0], `"'`)
			}
		}
	}
	return ""
}

func extractCommand(context string) string {
	// Look for command after "run" or "execute"
	lower := strings.ToLower(context)
	for _, prefix := range []string{"run ", "execute ", "command: ", "cmd: "} {
		if idx := strings.Index(lower, prefix); idx >= 0 {
			cmd := context[idx+len(prefix):]
			// Take until newline or end
			if nl := strings.Index(cmd, "\n"); nl > 0 {
				cmd = cmd[:nl]
			}
			return strings.TrimSpace(cmd)
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
