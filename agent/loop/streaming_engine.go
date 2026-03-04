package loop

import (
	"context"
	"fmt"
	"strings"
	"time"

	ctxmanager "github.com/vigo999/ms-cli/agent/context"
	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
)

// StreamingReActEngine implements a ReAct engine with streaming output support.
type StreamingReActEngine struct {
	provider    llm.Provider
	registry    *tools.Registry
	ctxManager  *ctxmanager.Manager
	modelName   string
	temperature float64
	maxTokens   int
	maxIterations int
	timeout       time.Duration
	systemPrompt  string
}

// NewStreamingReActEngine creates a new streaming ReAct engine.
func NewStreamingReActEngine(provider llm.Provider, registry *tools.Registry) *StreamingReActEngine {
	e := &StreamingReActEngine{
		provider:      provider,
		registry:      registry,
		ctxManager:    ctxmanager.NewManager(128000),
		modelName:     "gpt-4",
		temperature:   0.7,
		maxTokens:     4000,
		maxIterations: 15,
		timeout:       5 * time.Minute,
	}
	e.setupSystemPrompt()
	e.ctxManager.SetSystemMessage(e.systemPrompt)
	return e
}

// SetModelName sets the model name to use.
func (e *StreamingReActEngine) SetModelName(name string) {
	e.modelName = name
}

// SetTemperature sets the temperature for generation.
func (e *StreamingReActEngine) SetTemperature(t float64) {
	e.temperature = t
}

// SetMaxTokens sets the maximum tokens to generate.
func (e *StreamingReActEngine) SetMaxTokens(n int) {
	e.maxTokens = n
}

// SetMaxIterations sets the maximum number of iterations.
func (e *StreamingReActEngine) SetMaxIterations(n int) {
	e.maxIterations = n
}

// SetTimeout sets the execution timeout.
func (e *StreamingReActEngine) SetTimeout(d time.Duration) {
	e.timeout = d
}

// setupSystemPrompt configures the system message.
func (e *StreamingReActEngine) setupSystemPrompt() {
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

Respond clearly and concisely. When you complete a task, summarize what you did.`
}

// ExecuteStream runs the ReAct loop with streaming output.
// It sends events for: thinking_start, streaming_content, tool_call, tool_result, complete
func (e *StreamingReActEngine) ExecuteStream(task Task, eventCh chan<- Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

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
			return fmt.Errorf("timeout")
		default:
		}

		// Send thinking event
		eventCh <- Event{
			Type:    "thinking_start",
			Message: fmt.Sprintf("Step %d/%d", i, e.maxIterations),
		}

		// Stream the LLM response
		content, toolCalls, err := e.streamResponse(ctx, eventCh)
		if err != nil {
			eventCh <- Event{
				Type:    "error",
				Message: fmt.Sprintf("streaming failed: %v", err),
			}
			return err
		}

		// Handle tool calls
		if len(toolCalls) > 0 {
			tc := toolCalls[0]
			eventCh <- Event{
				Type:    "tool_call",
				Message: fmt.Sprintf("%s: %s", tc.Function.Name, tc.Function.Arguments),
			}

			// Execute tool
			var params map[string]any
			// Simple JSON parsing - in real code use proper unmarshaling
			params = make(map[string]any)
			
			result, err := e.registry.Execute(ctx, tc.Function.Name, params)
			
			var observation string
			if err != nil {
				observation = fmt.Sprintf("Error: %v", err)
				eventCh <- Event{
					Type:    "tool_error",
					Message: observation,
				}
			} else {
				observation = result.Output
				eventCh <- Event{
					Type:    "tool_result",
					Message: observation,
				}
			}

			// Add to context
			e.ctxManager.AddMessage("assistant", content)
			e.ctxManager.AddToolResult(tc.Function.Name, observation)
		} else {
			// No tool calls - task is complete
			eventCh <- Event{
				Type:    "complete",
				Message: content,
			}
			return nil
		}
	}

	eventCh <- Event{
		Type:    "error",
		Message: fmt.Sprintf("exceeded maximum iterations (%d)", e.maxIterations),
	}
	return fmt.Errorf("max iterations exceeded")
}

// streamResponse streams the LLM response and returns the full content.
func (e *StreamingReActEngine) streamResponse(ctx context.Context, eventCh chan<- Event) (string, []llm.ToolCall, error) {
	messages := e.buildMessages()
	toolDefs := e.buildToolDefinitions()

	req := llm.CompletionRequest{
		Model:       e.modelName,
		Messages:    messages,
		Tools:       toolDefs,
		MaxTokens:   e.maxTokens,
		Temperature: e.temperature,
	}

	stream, err := e.provider.Stream(ctx, req)
	if err != nil {
		return "", nil, err
	}

	var content strings.Builder
	var toolCalls []llm.ToolCall

	for chunk := range stream {
		if chunk.Error != nil {
			return content.String(), toolCalls, chunk.Error
		}

		if chunk.Done {
			break
		}

		// Send streaming content event
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
			eventCh <- Event{
				Type:    "streaming_content",
				Message: chunk.Content,
			}
		}

		// Accumulate tool calls
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
	}

	return content.String(), toolCalls, nil
}

// buildMessages builds the message list for LLM from context.
func (e *StreamingReActEngine) buildMessages() []llm.Message {
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
func (e *StreamingReActEngine) buildToolDefinitions() []llm.ToolDefinition {
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
}
