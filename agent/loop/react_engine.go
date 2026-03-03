package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vigo999/ms-cli/agent/context"
	"github.com/vigo999/ms-cli/agent/permission"
	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
)

// Step represents a single step in the ReAct loop.
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

// ReActEngine implements the ReAct (Reasoning + Acting) agent loop.
type ReActEngine struct {
	provider   llm.Provider
	registry   *tools.Registry
	ctxManager *context.Manager
	permission *permission.Service
	
	maxIterations int
	timeout       time.Duration
	systemPrompt  string
}

// NewReActEngine creates a new ReAct engine.
func NewReActEngine(provider llm.Provider, registry *tools.Registry) *ReActEngine {
	e := &ReActEngine{
		provider:      provider,
		registry:      registry,
		ctxManager:    context.NewManager(128000),
		maxIterations: 15,
		timeout:       5 * time.Minute,
	}
	e.setupSystemPrompt()
	e.ctxManager.SetSystemMessage(e.systemPrompt)
	return e
}

// SetMaxIterations sets the maximum number of iterations.
func (e *ReActEngine) SetMaxIterations(n int) {
	e.maxIterations = n
}

// SetTimeout sets the execution timeout.
func (e *ReActEngine) SetTimeout(d time.Duration) {
	e.timeout = d
}

// setupSystemPrompt configures the system message.
func (e *ReActEngine) setupSystemPrompt() {
	e.systemPrompt = `You are an AI assistant that helps users complete tasks by using available tools.

When given a task:
1. Analyze what needs to be done
2. Break it down into steps
3. Use tools when needed by calling the appropriate function
4. Provide clear explanations of your actions
5. Report results to the user

You have access to the following tools:
- fs_read: Read file contents
- fs_write: Write/create files  
- fs_edit: Edit files by replacing text
- fs_glob: Find files matching patterns
- fs_grep: Search text in files
- shell_exec: Execute shell commands

Respond in the following format:
Thought: [your reasoning about what to do]
Action: [tool_name]
Action Input: [JSON parameters]

When you have completed the task, respond with:
Thought: [final reasoning]
Final Answer: [your response to the user]`
}

// Execute runs the ReAct loop for a given task and returns events.
func (e *ReActEngine) Execute(task Task) (<-chan Event, error) {
	eventCh := make(chan Event, 10)

	go func() {
		defer close(eventCh)

		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		defer cancel()

		steps := []Step{}

		// Initial event
		eventCh <- Event{
			Type:    "task_start",
			Message: task.Description,
		}

		// Add user task to context
		e.ctxManager.AddMessage("user", task.Description)

		for i := 1; i <= e.maxIterations; i++ {
			select {
			case <-ctx.Done():
				eventCh <- Event{
					Type:    "error",
					Message: "execution timed out",
				}
				return
			default:
			}

			// Think: Generate thought and action
			thought, action, final, err := e.think(ctx, steps)
			if err != nil {
				eventCh <- Event{
					Type:    "error",
					Message: fmt.Sprintf("thinking failed: %v", err),
				}
				return
			}

			eventCh <- Event{
				Type:    "thought",
				Message: thought,
			}

			// Check if task is complete
			if final {
				eventCh <- Event{
					Type:    "complete",
					Message: thought,
				}
				return
			}

			// Act: Execute the action
			if action != nil {
				eventCh <- Event{
					Type:    "action_start",
					Message: fmt.Sprintf("%s: %v", action.Tool, action.Params),
				}

				observation, err := e.act(ctx, action)
				
				step := Step{
					Number:      i,
					Thought:     thought,
					Action:      action,
					Observation: observation,
					Timestamp:   time.Now(),
				}
				steps = append(steps, step)

				// Add to context for next iteration
				e.ctxManager.AddMessage("assistant", fmt.Sprintf("Thought: %s\nAction: %s\nObservation: %s", 
					thought, action.Tool, observation))

				if err != nil {
					eventCh <- Event{
						Type:    "action_error",
						Message: observation,
					}
				} else {
					eventCh <- Event{
						Type:    "action_result",
						Message: observation,
					}
				}
			}
		}

		eventCh <- Event{
			Type:    "error",
			Message: fmt.Sprintf("exceeded maximum iterations (%d)", e.maxIterations),
		}
	}()

	return eventCh, nil
}

// think generates the next thought and action using LLM.
func (e *ReActEngine) think(ctx context.Context, steps []Step) (thought string, action *Action, final bool, err error) {
	// Build messages from context
	messages := e.buildMessages()
	
	// Add tool definitions
	toolDefs := e.buildToolDefinitions()

	// Call LLM
	req := llm.CompletionRequest{
		Model:       "gpt-4",
		Messages:    messages,
		Tools:       toolDefs,
		MaxTokens:   4000,
		Temperature: 0.2,
	}

	resp, err := e.provider.Complete(ctx, req)
	if err != nil {
		return "", nil, false, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", nil, false, fmt.Errorf("no response from LLM")
	}

	content := resp.Choices[0].Message.Content
	
	// Check for tool calls
	if len(resp.Choices[0].ToolCalls) > 0 {
		tc := resp.Choices[0].ToolCalls[0]
		action = &Action{
			Tool: tc.Function.Name,
		}
		
		// Parse arguments
		var params map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", nil, false, fmt.Errorf("failed to parse tool arguments: %w", err)
		}
		action.Params = params
		
		// Extract thought from content (before tool call)
		thought = e.extractThought(content)
		return thought, action, false, nil
	}

	// No tool call - check if final answer
	if strings.Contains(content, "Final Answer:") {
		parts := strings.SplitN(content, "Final Answer:", 2)
		thought = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			thought = strings.TrimSpace(parts[1])
		}
		return thought, nil, true, nil
	}

	// Just a thought, no action
	return e.extractThought(content), nil, false, nil
}

// act executes a tool action.
func (e *ReActEngine) act(ctx context.Context, action *Action) (string, error) {
	// Check permission if service is configured
	if e.permission != nil {
		action := permission.Action{
			Tool:   action.Tool,
			Action: "execute",
			Params: action.Params,
		}
		result := e.permission.Check(action)
		if !result.Allowed {
			return fmt.Sprintf("Permission denied: %s", result.Message), fmt.Errorf("permission denied")
		}
	}

	// Execute tool
	result, err := e.registry.Execute(ctx, action.Tool, action.Params)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), err
	}

	if !result.Success {
		return fmt.Sprintf("Failed: %s", result.Output), fmt.Errorf("tool execution failed")
	}

	return result.Output, nil
}

// buildMessages builds the message list for LLM from context.
func (e *ReActEngine) buildMessages() []llm.Message {
	ctxMsgs := e.ctxManager.BuildMessages()
	messages := make([]llm.Message, 0, len(ctxMsgs))
	
	for _, msg := range ctxMsgs {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	return messages
}

// buildToolDefinitions creates tool definitions for LLM.
func (e *ReActEngine) buildToolDefinitions() []llm.ToolDefinition {
	defs := []llm.ToolDefinition{
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
							"description": "Content to write to the file",
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
							"description": "Exact text to find and replace",
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
	
	return defs
}

// extractThought extracts the thought from LLM response.
func (e *ReActEngine) extractThought(content string) string {
	content = strings.TrimSpace(content)
	
	// Remove "Thought:" prefix if present
	content = strings.TrimPrefix(content, "Thought:")
	content = strings.TrimSpace(content)
	
	// Extract just the thought part (before Action:)
	if idx := strings.Index(content, "Action:"); idx > 0 {
		content = content[:idx]
	}
	
	return strings.TrimSpace(content)
}
