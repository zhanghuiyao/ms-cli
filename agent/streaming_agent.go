package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
	"github.com/vigo999/ms-cli/ui/model"
)

// StreamingAgent executes tasks with streaming LLM responses.
type StreamingAgent struct {
	*SmartAgent
	eventCh chan<- model.Event
}

// NewStreamingAgent creates a new streaming agent.
func NewStreamingAgent(provider llm.Provider, registry *tools.Registry, eventCh chan<- model.Event) *StreamingAgent {
	return &StreamingAgent{
		SmartAgent: NewSmartAgent(provider, registry),
		eventCh:    eventCh,
	}
}

// RunStream executes a task with streaming output.
func (a *StreamingAgent) RunStream(task loop.Task) error {
	ctx := context.Background()

	// Initial event
	a.emit(model.Event{
		Type:    model.AgentThinking,
		Message: "Analyzing task...",
	})

	// Add user message to context
	a.context.AddMessage("user", task.Description)

	// Build messages
	messages := a.buildMessages()
	toolDefs := a.buildToolDefinitions()

	// Call LLM with streaming
	stream, err := a.provider.Stream(ctx, llm.CompletionRequest{
		Model:       "gpt-4",
		Messages:    messages,
		Tools:       toolDefs,
		MaxTokens:   4000,
		Temperature: 0.2,
	})

	if err != nil {
		a.emit(model.Event{
			Type:     model.ToolError,
			ToolName: "LLM",
			Message:  fmt.Sprintf("stream error: %v", err),
		})
		return err
	}

	// Process stream
	var fullContent strings.Builder
	var toolCalls []llm.ToolCall

	for chunk := range stream {
		if chunk.Error != nil {
			a.emit(model.Event{
				Type:     model.ToolError,
				ToolName: "LLM",
				Message:  fmt.Sprintf("chunk error: %v", chunk.Error),
			})
			return chunk.Error
		}

		if chunk.Content != "" {
			fullContent.WriteString(chunk.Content)
			a.emit(model.Event{
				Type:    model.AgentReply,
				Message: chunk.Content,
			})
		}

		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}

		if chunk.Done {
			break
		}
	}

	// Add assistant message to context
	content := fullContent.String()
	if content != "" {
		a.context.AddMessage("assistant", content)
	}

	// Process tool calls
	for _, tc := range toolCalls {
		a.executeToolCallAndStream(ctx, tc)
	}

	return nil
}

// executeToolCallAndStream executes a tool call and streams results.
func (a *StreamingAgent) executeToolCallAndStream(ctx context.Context, tc llm.ToolCall) {
	// Emit action start
	a.emit(model.Event{
		Type:    model.CmdStarted,
		Message: fmt.Sprintf("%s: %s", tc.Function.Name, tc.Function.Arguments),
	})

	// Execute tool (same as parent)
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
		a.context.AddMessage("tool", fmt.Sprintf("Parse error: %v", err))
		a.emit(model.Event{
			Type:     model.ToolError,
			ToolName: tc.Function.Name,
			Message:  fmt.Sprintf("parse error: %v", err),
		})
		return
	}

	result, err := a.registry.Execute(ctx, tc.Function.Name, params)

	var resultContent string
	if err != nil {
		resultContent = fmt.Sprintf("Error: %v", err)
	} else if result.Success {
		resultContent = result.Output
	} else if result.Error != nil {
		resultContent = fmt.Sprintf("Error: %v", result.Error)
	} else {
		resultContent = "No output"
	}

	// Add to context
	a.context.AddMessage("tool", fmt.Sprintf("%s result:\n%s", tc.Function.Name, resultContent))

	// Stream result
	if err != nil || !result.Success {
		a.emit(model.Event{
			Type:     model.ToolError,
			ToolName: tc.Function.Name,
			Message:  resultContent,
		})
	} else {
		a.emit(model.Event{
			Type:    model.CmdOutput,
			Message: resultContent,
		})
	}
}

// emit sends an event to the channel.
func (a *StreamingAgent) emit(ev model.Event) {
	select {
	case a.eventCh <- ev:
	default:
		// Channel full, drop event
	}
}
