package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vigo999/ms-cli/agent/context"
	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
)

// SmartAgent uses LLM for task planning and execution.
type SmartAgent struct {
	provider llm.Provider
	registry *tools.Registry
	context  *context.Manager
	budget   *context.Budget

	maxSteps int
	timeout  time.Duration

	systemPrompt string
}

// NewSmartAgent creates a new LLM-powered agent.
func NewSmartAgent(provider llm.Provider, registry *tools.Registry) *SmartAgent {
	a := &SmartAgent{
		provider: provider,
		registry: registry,
		context:  context.NewManager(128000),
		budget:   context.NewBudget(32768, 10.0),
		maxSteps: 15,
		timeout:  5 * time.Minute,
	}

	a.setupSystemPrompt()
	a.context.SetSystemMessage(a.systemPrompt)

	return a
}

// SetMaxSteps sets the maximum execution steps.
func (a *SmartAgent) SetMaxSteps(n int) {
	a.maxSteps = n
}

// SetTimeout sets the execution timeout.
func (a *SmartAgent) SetTimeout(d time.Duration) {
	a.timeout = d
}

// setupSystemPrompt configures the system message for the agent.
func (a *SmartAgent) setupSystemPrompt() {
	a.systemPrompt = `You are an AI assistant that helps users complete tasks by using available tools.

When given a task:
1. Analyze what needs to be done
2. Break it down into steps
3. Use tools when needed by calling the appropriate function
4. Provide clear explanations of your actions
5. Report results to the user

Available tools:
- fs_read: Read file contents
- fs_write: Write/create files  
- fs_edit: Edit files by replacing text
- fs_glob: Find files matching patterns
- fs_grep: Search text in files
- shell_exec: Execute shell commands

Guidelines:
- Always check if files exist before reading
- Use glob to find files when paths are not specified
- Prefer editing over rewriting for small changes
- Commands run with a 30-second timeout
- Be concise but thorough in your responses`
}

// Run executes a task using LLM planning.
func (a *SmartAgent) Run(task loop.Task) ([]loop.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	events := make([]loop.Event, 0)

	// Initial event
	events = append(events, loop.Event{
		Type:    "task_start",
		Message: task.Description,
	})

	// Add user message to context
	a.context.AddMessage("user", task.Description)

	// Main loop
	for step := 1; step <= a.maxSteps; step++ {
		select {
		case <-ctx.Done():
			events = append(events, loop.Event{
				Type:    "error",
				Message: "execution timed out",
			})
			return events, ctx.Err()
		default:
		}

		// Check budget
		if err := a.budget.CheckLimit(); err != nil {
			events = append(events, loop.Event{
				Type:    "error",
				Message: fmt.Sprintf("budget exceeded: %v", err),
			})
			return events, err
		}

		// Build messages for LLM
		messages := a.buildMessages()
		toolDefs := a.buildToolDefinitions()

		// Call LLM
		resp, err := a.provider.Complete(ctx, llm.CompletionRequest{
			Model:       "gpt-4",
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   4000,
			Temperature: 0.2,
		})

		if err != nil {
			events = append(events, loop.Event{
				Type:    "error",
				Message: fmt.Sprintf("LLM error: %v", err),
			})
			return events, err
		}

		// Record usage
		a.budget.RecordUsage(resp.Usage.TotalTokens, context.EstimateCost(resp.Usage.TotalTokens, "gpt-4"))

		// Process response
		if len(resp.Choices) == 0 {
			events = append(events, loop.Event{
				Type:    "error",
				Message: "no response from LLM",
			})
			return events, fmt.Errorf("no response")
		}

		choice := resp.Choices[0]

		// Add assistant message to context
		a.context.AddMessage("assistant", choice.Message.Content)

		// Check for tool calls
		if len(choice.ToolCalls) > 0 {
			// Process tool calls
			for _, tc := range choice.ToolCalls {
				event, done := a.executeToolCall(ctx, tc)
				events = append(events, event)

				if done {
					return events, nil
				}
			}
		} else if choice.FinishReason == "stop" {
			// Task complete
			events = append(events, loop.Event{
				Type:    "complete",
				Message: choice.Message.Content,
			})
			return events, nil
		} else {
			// Just a thought, continue
			events = append(events, loop.Event{
				Type:    "thought",
				Message: choice.Message.Content,
			})
		}
	}

	events = append(events, loop.Event{
		Type:    "error",
		Message: fmt.Sprintf("max steps (%d) exceeded", a.maxSteps),
	})
	return events, fmt.Errorf("max steps exceeded")
}

// buildMessages converts context to LLM format.
func (a *SmartAgent) buildMessages() []llm.Message {
	ctxMsgs := a.context.BuildMessages()
	messages := make([]llm.Message, 0, len(ctxMsgs))

	for _, msg := range ctxMsgs {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

// buildToolDefinitions creates LLM tool schemas.
func (a *SmartAgent) buildToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        "fs_read",
				Description: "Read the contents of a file",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{
							"type":        "string",
							"description": "Path to the file to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        "fs_write",
				Description: "Write content to a file, creating it if needed",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{
							"type":        "string",
							"description": "Path to the file",
						},
						"content": map[string]string{
							"type":        "string",
							"description": "Content to write",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        "fs_edit",
				Description: "Edit a file by replacing exact text",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{
							"type":        "string",
							"description": "Path to the file",
						},
						"old_text": map[string]string{
							"type":        "string",
							"description": "Exact text to find (must match exactly)",
						},
						"new_text": map[string]string{
							"type":        "string",
							"description": "New text to replace with",
						},
					},
					"required": []string{"path", "old_text", "new_text"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        "fs_glob",
				Description: "Find files matching a glob pattern",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]string{
							"type":        "string",
							"description": "Glob pattern (e.g., '*.go', '**/*.yaml')",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        "fs_grep",
				Description: "Search for text patterns in files",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]string{
							"type":        "string",
							"description": "Text pattern to search for",
						},
						"path": map[string]string{
							"type":        "string",
							"description": "Directory to search in",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        "shell_exec",
				Description: "Execute a shell command",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]string{
							"type":        "string",
							"description": "Shell command to execute",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}

// executeToolCall executes a tool call from LLM.
func (a *SmartAgent) executeToolCall(ctx context.Context, tc llm.ToolCall) (loop.Event, bool) {
	// Parse arguments
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
		// Add error to context
		a.context.AddMessage("tool", fmt.Sprintf("Error parsing arguments: %v", err))
		return loop.Event{
			Type:    "action_error",
			Message: fmt.Sprintf("parse error: %v", err),
		}, false
	}

	// Emit action start
	actionStart := loop.Event{
		Type:    "action_start",
		Message: fmt.Sprintf("%s: %s", tc.Function.Name, tc.Function.Arguments),
	}

	// Execute tool
	result, err := a.registry.Execute(ctx, tc.Function.Name, params)

	// Build result message for context
	var resultContent string
	if err != nil {
		resultContent = fmt.Sprintf("Error: %v", err)
	} else if result.Success {
		resultContent = result.Output
	} else {
		resultContent = "Operation failed"
	}

	// Add to context
	a.context.AddMessage("tool", fmt.Sprintf("%s result:\n%s", tc.Function.Name, resultContent))

	// Return event
	if err != nil {
		return loop.Event{
			Type:    "action_error",
			Message: resultContent,
		}, false
	}
	if !result.Success {
		return loop.Event{
			Type:    "action_error",
			Message: resultContent,
		}, false
	}

	return loop.Event{
		Type:    "action_result",
		Message: resultContent,
	}, false
}

// GetUsage returns current usage stats.
func (a *SmartAgent) GetUsage() context.UsageStats {
	return a.budget.GetUsage()
}

// GetContextManager returns the context manager.
func (a *SmartAgent) GetContextManager() *context.Manager {
	return a.context
}
