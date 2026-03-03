package executor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vigo999/ms-cli/agent"
	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
	"github.com/vigo999/ms-cli/tools/fs"
	"github.com/vigo999/ms-cli/tools/shell"
	"github.com/vigo999/ms-cli/tools/web"
	"github.com/vigo999/ms-cli/ui/model"
)

// Runner executes tasks using tools or LLM-powered agent.
type Runner struct {
	registry *tools.Registry
	useLLM   bool
	provider llm.Provider
}

// NewRunner creates a new executor with all tools registered.
func NewRunner() *Runner {
	r := &Runner{
		registry: tools.NewRegistry(),
		useLLM:   false,
	}

	// Register all file system tools
	r.registry.Register(&fs.ReadTool{})
	r.registry.Register(&fs.WriteTool{})
	r.registry.Register(&fs.EditTool{})
	r.registry.Register(&fs.GlobTool{})
	r.registry.Register(&fs.GrepTool{})

	// Register shell tool
	r.registry.Register(shell.NewExecTool())

	// Register web search tool
	r.registry.Register(web.NewWebSearchTool())

	return r
}

// NewSmartRunner creates a runner with LLM support.
func NewSmartRunner(apiKey, endpoint string) *Runner {
	return NewSmartRunnerWithProvider("openai", apiKey, endpoint)
}

// NewSmartRunnerWithProvider creates a runner with specific LLM provider.
func NewSmartRunnerWithProvider(provider, apiKey, endpoint string) *Runner {
	r := NewRunner()
	r.useLLM = true

	switch strings.ToLower(provider) {
	case "anthropic", "claude":
		if endpoint == "" {
			endpoint = "https://api.anthropic.com/v1"
		}
		r.provider = llm.NewAnthropicProvider(apiKey, endpoint)
	case "openai", "gpt":
		fallthrough
	default:
		if endpoint == "" {
			endpoint = "https://api.openai.com/v1"
		}
		r.provider = llm.NewOpenAIProvider(apiKey, endpoint)
	}

	return r
}

// NewSmartRunnerFromEnv creates a runner using environment variables.
func NewSmartRunnerFromEnv() *Runner {
	apiKey := os.Getenv("MSCLI_MODEL_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	endpoint := os.Getenv("MSCLI_MODEL_ENDPOINT")
	provider := os.Getenv("MSCLI_MODEL_PROVIDER")
	if provider == "" {
		provider = "openai"
	}

	if apiKey == "" {
		// Fall back to rule-based runner
		return NewRunner()
	}

	return NewSmartRunnerWithProvider(provider, apiKey, endpoint)
}

// SetProvider sets a custom LLM provider.
func (r *Runner) SetProvider(provider llm.Provider) {
	r.provider = provider
	r.useLLM = provider != nil
}

// Execute runs a task and emits events to the given channel.
func (r *Runner) Execute(task loop.Task, eventCh chan<- model.Event) error {
	if r.useLLM && r.provider != nil {
		return r.executeWithLLM(task, eventCh)
	}
	return r.executeWithRuleAgent(task, eventCh)
}

// executeWithLLM uses the smart agent with LLM.
func (r *Runner) executeWithLLM(task loop.Task, eventCh chan<- model.Event) error {
	// Create smart agent
	smartAgent := agent.NewSmartAgent(r.provider, r.registry)
	smartAgent.SetMaxSteps(15)

	// Run the agent
	events, err := smartAgent.Run(task)
	if err != nil {
		eventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "SmartAgent",
			Message:  fmt.Sprintf("execution failed: %v", err),
		}
		return err
	}

	// Convert and emit events
	for _, ev := range events {
		uiEvent := convertEvent(ev)
		eventCh <- uiEvent
	}

	return nil
}

// executeWithRuleAgent uses the rule-based agent (no LLM).
func (r *Runner) executeWithRuleAgent(task loop.Task, eventCh chan<- model.Event) error {
	ctx := context.Background()

	// Create agent loop with our registry
	agentLoop := loop.NewAgentLoop(r.registry)
	agentLoop.SetMaxSteps(15)

	// Run the agent
	events, err := agentLoop.Run(task)
	if err != nil {
		eventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "Agent",
			Message:  fmt.Sprintf("execution failed: %v", err),
		}
		return err
	}

	// Convert and emit events
	for _, ev := range events {
		uiEvent := convertEvent(ev)
		eventCh <- uiEvent
	}

	return nil
}

// convertEvent converts loop events to UI events.
func convertEvent(ev loop.Event) model.Event {
	switch ev.Type {
	case "task_start":
		return model.Event{
			Type:    model.AgentThinking,
			Message: ev.Message,
		}

	case "thought":
		return model.Event{
			Type:    model.AgentReply,
			Message: ev.Message,
		}

	case "action_start":
		// Determine tool type based on action
		return model.Event{
			Type:    model.CmdStarted,
			Message: ev.Message,
		}

	case "action_result":
		return model.Event{
			Type:    model.CmdOutput,
			Message: ev.Message,
		}

	case "action_error":
		return model.Event{
			Type:     model.ToolError,
			ToolName: "Tool",
			Message:  ev.Message,
		}

	case "complete":
		return model.Event{
			Type:    model.AnalysisReady,
			Message: ev.Message,
		}

	case "error":
		return model.Event{
			Type:     model.ToolError,
			ToolName: "Engine",
			Message:  ev.Message,
		}

	default:
		return model.Event{
			Type:    model.AgentReply,
			Message: ev.Message,
		}
	}
}

// GetRegistry returns the tool registry for inspection.
func (r *Runner) GetRegistry() *tools.Registry {
	return r.registry
}

// IsLLMEnabled returns true if LLM is configured.
func (r *Runner) IsLLMEnabled() bool {
	return r.useLLM
}
