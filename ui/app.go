package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vigo999/ms-cli/ui/components"
	"github.com/vigo999/ms-cli/ui/model"
	"github.com/vigo999/ms-cli/ui/panels"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	topBarHeight   = 3 // brand line + info line + divider
	chatLineHeight = 2
	hintBarHeight  = 2
	inputHeight    = 1
	verticalPad    = 2
)

var chatLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

// App is the TUI root model.
type App struct {
	state         model.State
	viewport      components.SelectableViewport
	viewportLines []string // 存储原始文本行用于选择
	input         components.AutoComplete
	spinner       components.Spinner
	thinking      components.ThinkingAnimator
	isThinking    bool
	streamingText string
	width         int
	height        int
	eventCh       <-chan model.Event
	userCh        chan<- string
}

// New creates a new App driven by the given event channel.
// userCh may be nil (demo mode) — user input won't be forwarded.
func New(ch <-chan model.Event, userCh chan<- string, version, workDir, repoURL string) App {
	return App{
		state:         model.NewState(version, workDir, repoURL),
		viewport:      components.NewSelectableViewport(80, 20),
		input:         components.NewAutoComplete(),
		spinner:       components.NewSpinner(),
		thinking:      components.NewThinkingAnimator(),
		isThinking:    false,
		eventCh:       ch,
		userCh:        userCh,
	}
}

func (a App) waitForEvent() tea.Msg {
	ev, ok := <-a.eventCh
	if !ok {
		return model.Event{Type: model.Done}
	}
	return ev
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Model.Tick,
		a.thinking.Init(), // 初始化 thinking 动画
		a.waitForEvent,
	)
}

func (a App) chatHeight() int {
	h := a.height - topBarHeight - chatLineHeight - hintBarHeight - inputHeight - verticalPad
	if h < 1 {
		return 1
	}
	return h
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.MouseMsg:
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		return a, cmd

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.viewport = a.viewport.SetSize(a.width-4, a.chatHeight())
		return a, nil

	case model.Event:
		return a.handleEvent(msg)

	case components.ThinkingMsg:
		// 处理 Thinking 动画 tick
		if a.isThinking {
			var cmd tea.Cmd
			a.thinking, cmd = a.thinking.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

	default:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// 同时更新 thinking 动画
		if a.isThinking {
			a.thinking, cmd = a.thinking.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return a, tea.Batch(cmds...)
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit

	case "tab":
		// 接受自动补全建议
		a.input = a.input.AcceptSuggestion()
		return a, nil

	case "shift+tab":
		// 向上选择建议
		a.input = a.input.PrevSuggestion()
		return a, nil

	case "up":
		// 如果自动补全显示，选择建议；否则滚动 viewport
		if a.input.Showing() {
			a.input = a.input.PrevSuggestion()
			return a, nil
		}
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		return a, cmd

	case "down":
		// 如果自动补全显示，选择建议；否则滚动 viewport
		if a.input.Showing() {
			a.input = a.input.NextSuggestion()
			return a, nil
		}
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		return a, cmd

	case "enter":
		// 如果有建议显示，先接受建议
		if a.input.Showing() {
			a.input = a.input.AcceptSuggestion()
			return a, nil
		}
		val := a.input.Value()
		if val == "" {
			return a, nil
		}
		a.state = a.state.WithMessage(model.Message{Kind: model.MsgUser, Content: val})
		a.input = a.input.Reset()
		a.updateViewport()
		if a.userCh != nil {
			select {
			case a.userCh <- val:
			default:
			}
		}
		return a, nil

	case "pgup", "pgdown", "home", "end":
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		return a, cmd

	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		return a, cmd
	}
}

func (a App) handleEvent(ev model.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {
	case model.AgentThinking:
		a.isThinking = true
		a.thinking.Start()
		mi := a.state.Model
		mi.Status = "Thinking..."
		a.state = a.state.WithModel(mi)
		a.state = a.state.WithMessage(model.Message{Kind: model.MsgThinking})
		return a, tea.Batch(a.waitForEvent, a.thinkingTick())

	case model.AgentReply:
		a.isThinking = false
		a.thinking.Stop()
		a.streamingText = ""
		mi := a.state.Model
		mi.Status = "Done"
		a.state = a.state.WithModel(mi)
		a.state = a.state.WithStepProgress(model.StepProgress{IsActive: false})
		a.state = a.replaceThinking(model.Message{Kind: model.MsgAgent, Content: ev.Message})

	case model.AgentStreaming:
		// 流式输出：追加到缓冲区并更新最后一条消息
		a.streamingText += ev.Message
		a.state = a.replaceOrUpdateStreaming(model.Message{Kind: model.MsgAgent, Content: a.streamingText})

	case model.SessionCleared:
		a.state = model.NewState(a.state.Version, a.state.WorkDir, a.state.RepoURL)
		a.state.Model.Name = ev.Message // 保留当前模型名称

	case model.CmdStarted:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: "Shell",
			Display:  model.DisplayExpanded,
			Content:  "$ " + ev.Message,
		})

	case model.CmdOutput:
		a.state = a.appendToLastTool(ev.Message)

	case model.CmdFinished:
		// output already in the tool block

	case model.ToolRead:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: "Read",
			Display:  model.DisplayCollapsed,
			Content:  ev.Message,
			Summary:  ev.Summary,
		})

	case model.ToolGrep:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: "Grep",
			Display:  model.DisplayCollapsed,
			Content:  ev.Message,
			Summary:  ev.Summary,
		})

	case model.ToolGlob:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: "Glob",
			Display:  model.DisplayCollapsed,
			Content:  ev.Message,
			Summary:  ev.Summary,
		})

	case model.ToolEdit:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: "Edit",
			Display:  model.DisplayExpanded,
			Content:  ev.Message,
		})

	case model.ToolWrite:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: "Write",
			Display:  model.DisplayExpanded,
			Content:  ev.Message,
		})

	case model.ToolError:
		a.state = a.state.WithMessage(model.Message{
			Kind:     model.MsgTool,
			ToolName: ev.ToolName,
			Display:  model.DisplayError,
			Content:  ev.Message,
		})

	case model.AnalysisReady:
		a.state = a.state.WithMessage(model.Message{Kind: model.MsgAgent, Content: ev.Message})

	case model.TokenUpdate:
		mi := a.state.Model
		mi.CtxUsed = ev.CtxUsed
		mi.TokensUsed = ev.TokensUsed
		a.state = a.state.WithModel(mi)

	case model.TaskUpdated:
		// no-op for now

	case model.StepUpdate:
		// 更新步骤进度
		a.state = a.state.WithStepProgress(model.StepProgress{
			CurrentStep: ev.CtxUsed,  // 复用字段
			TotalSteps:  ev.TokensUsed, // 复用字段
			StepName:    ev.Message,
			IsActive:    true,
		})

	case model.Done:
		return a, tea.Quit
	}

	a.updateViewport()
	return a, a.waitForEvent
}

// thinkingTick creates a command that sends a ThinkingMsg periodically.
func (a App) thinkingTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return components.ThinkingMsg{}
	})
}

// replaceOrUpdateStreaming replaces a thinking message or updates the last streaming message.
func (a App) replaceOrUpdateStreaming(m model.Message) model.State {
	msgs := make([]model.Message, 0, len(a.state.Messages))
	
	// 如果最后一条是 thinking 或者是 agent 消息（流式），则替换
	for i, msg := range a.state.Messages {
		if i == len(a.state.Messages)-1 && (msg.Kind == model.MsgThinking || msg.Kind == model.MsgAgent) {
			msgs = append(msgs, m)
		} else {
			msgs = append(msgs, msg)
		}
	}
	
	// 如果消息列表为空，添加新消息
	if len(a.state.Messages) == 0 {
		msgs = append(msgs, m)
	}
	
	return model.State{
		Version:  a.state.Version,
		Model:    a.state.Model,
		Messages: msgs,
		WorkDir:  a.state.WorkDir,
		RepoURL:  a.state.RepoURL,
	}
}

func (a App) replaceThinking(m model.Message) model.State {
	msgs := make([]model.Message, 0, len(a.state.Messages))
	for _, msg := range a.state.Messages {
		if msg.Kind != model.MsgThinking {
			msgs = append(msgs, msg)
		}
	}
	msgs = append(msgs, m)
	return model.State{
		Version:  a.state.Version,
		Model:    a.state.Model,
		Messages: msgs,
		WorkDir:  a.state.WorkDir,
		RepoURL:  a.state.RepoURL,
	}
}

func (a App) appendToLastTool(line string) model.State {
	msgs := make([]model.Message, len(a.state.Messages))
	copy(msgs, a.state.Messages)

	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Kind == model.MsgTool {
			msgs[i] = model.Message{
				Kind:     model.MsgTool,
				ToolName: msgs[i].ToolName,
				Display:  msgs[i].Display,
				Content:  msgs[i].Content + "\n" + line,
			}
			break
		}
	}

	return model.State{
		Version:  a.state.Version,
		Model:    a.state.Model,
		Messages: msgs,
		WorkDir:  a.state.WorkDir,
		RepoURL:  a.state.RepoURL,
	}
}

func (a *App) updateViewport() {
	// 生成带样式的内容和纯文本内容
	styledContent := panels.RenderMessages(a.state.Messages, a.spinner.View())
	
	// 存储原始文本用于选择
	a.viewportLines = a.extractPlainTextLines(a.state.Messages)
	
	// 如果正在思考，在末尾添加动态 thinking 指示器
	if a.isThinking {
		styledContent += "\n\n  " + a.thinking.View()
		a.viewportLines = append(a.viewportLines, "", "  Thinking...")
	}
	
	a.viewport = a.viewport.SetContent(styledContent)
	// 同时设置原始行用于选择
	a.viewport = a.viewport.SetLines(a.viewportLines)
}

// extractPlainTextLines 从消息中提取纯文本行
func (a *App) extractPlainTextLines(messages []model.Message) []string {
	var lines []string
	for _, m := range messages {
		switch m.Kind {
		case model.MsgUser:
			lines = append(lines, "> "+m.Content)
		case model.MsgAgent:
			lines = append(lines, strings.Split(m.Content, "\n")...)
		case model.MsgTool:
			lines = append(lines, m.ToolName+": "+m.Content)
		}
	}
	return lines
}

func (a App) chatLine() string {
	return chatLineStyle.Render(strings.Repeat("─", a.width))
}

func (a App) View() string {
	topBar := panels.RenderTopBar(a.state, a.width)
	line := a.chatLine()
	chat := a.viewport.View()
	input := "  " + a.input.View()
	hintBar := panels.RenderHintBar(a.width)

	return lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		line,
		chat,
		line,
		input,
		hintBar,
	)
}
