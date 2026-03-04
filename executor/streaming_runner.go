package executor

import (
	"fmt"
	"os"
	"strings"

	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
	"github.com/vigo999/ms-cli/tools/fs"
	"github.com/vigo999/ms-cli/tools/shell"
	"github.com/vigo999/ms-cli/tools/web"
	"github.com/vigo999/ms-cli/ui/model"
)

// StreamingRunner executes tasks with streaming output support.
type StreamingRunner struct {
	registry *tools.Registry
	provider llm.Provider
	useLLM   bool
}

// NewStreamingRunner creates a new streaming runner.
func NewStreamingRunner(workDir string, provider llm.Provider) *StreamingRunner {
	if workDir == "" {
		wd, err := os.Getwd()
		if err == nil {
			workDir = wd
		} else {
			workDir = "."
		}
	}

	r := &StreamingRunner{
		registry: tools.NewRegistry(),
		provider: provider,
		useLLM:   provider != nil,
	}

	// Register all tools
	r.registry.Register(&fs.ReadTool{WorkDir: workDir})
	r.registry.Register(&fs.WriteTool{WorkDir: workDir})
	r.registry.Register(&fs.EditTool{WorkDir: workDir})
	r.registry.Register(&fs.GlobTool{})
	r.registry.Register(&fs.GrepTool{})
	r.registry.Register(shell.NewExecTool())
	r.registry.Register(web.NewWebSearchTool())

	return r
}

// ExecuteStream runs a task with streaming output.
func (r *StreamingRunner) ExecuteStream(task loop.Task, eventCh chan<- model.Event) error {
	if !r.useLLM || r.provider == nil {
		return r.executeWithRuleAgent(task, eventCh)
	}
	return r.executeWithStreamingAgent(task, eventCh)
}

// executeWithStreamingAgent uses the streaming ReAct engine.
func (r *StreamingRunner) executeWithStreamingAgent(task loop.Task, eventCh chan<- model.Event) error {
	engine := loop.NewStreamingReActEngine(r.provider, r.registry)
	engine.SetMaxIterations(15)
	
	// 从 provider 获取模型配置（如果可用）
	// 这里简化处理，实际应该从配置传递
	engine.SetModelName("gpt-4")
	engine.SetTemperature(0.7)
	engine.SetMaxTokens(4000)

	// Create internal event channel
	internalCh := make(chan loop.Event, 10)

	// Run engine in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.ExecuteStream(task, internalCh)
	}()

	// Forward events and handle streaming
	var streamingBuffer strings.Builder
	isStreaming := false

	for {
		select {
		case ev, ok := <-internalCh:
			if !ok {
				// Channel closed
				if isStreaming {
					// Send final accumulated content
					eventCh <- model.Event{
						Type:    model.AgentReply,
						Message: streamingBuffer.String(),
					}
				}
				return <-errCh
			}

			switch ev.Type {
			case "thinking_start":
				eventCh <- model.Event{
					Type:    model.AgentThinking,
					Message: ev.Message,
				}
				streamingBuffer.Reset()
				isStreaming = true

			case "streaming_content":
				streamingBuffer.WriteString(ev.Message)
				eventCh <- model.Event{
					Type:    model.AgentStreaming,
					Message: ev.Message,
				}

			case "tool_call":
				// End streaming for tool call
				if isStreaming && streamingBuffer.Len() > 0 {
					eventCh <- model.Event{
						Type:    model.AgentReply,
						Message: streamingBuffer.String(),
					}
					streamingBuffer.Reset()
					isStreaming = false
				}
				eventCh <- model.Event{
					Type:    model.CmdStarted,
					Message: ev.Message,
				}

			case "tool_result":
				eventCh <- model.Event{
					Type:    model.CmdOutput,
					Message: ev.Message,
				}

			case "tool_error":
				eventCh <- model.Event{
					Type:     model.ToolError,
					ToolName: "Tool",
					Message:  ev.Message,
				}

			case "complete":
				if isStreaming {
					// Final streaming content
					if streamingBuffer.Len() > 0 {
						eventCh <- model.Event{
							Type:    model.AgentReply,
							Message: streamingBuffer.String(),
						}
					}
					isStreaming = false
				}
				eventCh <- model.Event{
					Type:    model.AnalysisReady,
					Message: ev.Message,
				}

			case "error":
				eventCh <- model.Event{
					Type:     model.ToolError,
					ToolName: "Engine",
					Message:  ev.Message,
				}
			}

		case err := <-errCh:
			if isStreaming && streamingBuffer.Len() > 0 {
				eventCh <- model.Event{
					Type:    model.AgentReply,
					Message: streamingBuffer.String(),
				}
			}
			return err
		}
	}
}

// executeWithRuleAgent uses the rule-based agent (fallback).
func (r *StreamingRunner) executeWithRuleAgent(task loop.Task, eventCh chan<- model.Event) error {
	agentLoop := loop.NewAgentLoop(r.registry)
	agentLoop.SetMaxSteps(15)

	eventCh <- model.Event{Type: model.AgentThinking}

	events, err := agentLoop.Run(task)
	if err != nil {
		eventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "Agent",
			Message:  fmt.Sprintf("execution failed: %v", err),
		}
		return err
	}

	for _, ev := range events {
		uiEvent := convertEvent(ev)
		eventCh <- uiEvent
	}

	return nil
}

// GetRegistry returns the tool registry.
func (r *StreamingRunner) GetRegistry() *tools.Registry {
	return r.registry
}

// IsLLMEnabled returns true if LLM is configured.
func (r *StreamingRunner) IsLLMEnabled() bool {
	return r.useLLM
}
