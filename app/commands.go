package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vigo999/ms-cli/internal/project"
	"github.com/vigo999/ms-cli/ui/model"
)

// handleCommand dispatches slash commands.
func (a *Application) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	// 已有命令
	case "/roadmap":
		a.cmdRoadmap(parts[1:])
	case "/weekly":
		a.cmdWeekly(parts[1:])

	// === 新增命令 ===
	case "/model":
		a.cmdModel(parts[1:])
	case "/compact":
		a.cmdCompact()
	case "/clear":
		a.cmdClear()
	case "/new":
		a.cmdNew()
	case "/help":
		a.cmdHelp()
	case "/save":
		a.cmdSave(parts[1:])
	case "/load":
		a.cmdLoad(parts[1:])
	case "/history":
		a.cmdHistory()
	case "/tools":
		a.cmdTools()
	case "/temperature":
		a.cmdTemperature(parts[1:])
	case "/tokens":
		a.cmdTokens(parts[1:])

	default:
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: fmt.Sprintf("Unknown command: %s. Type /help for available commands.", parts[0]),
		}
	}
}

// ============ 模型命令 ============

func (a *Application) cmdModel(args []string) {
	if len(args) == 0 {
		// 显示当前模型信息
		msg := fmt.Sprintf(`Current Model: %s
Provider: %s
Endpoint: %s`,
			a.Config.Model.Model,
			a.Config.Model.Provider,
			a.Config.Model.Endpoint,
		)
		a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
		return
	}

	switch args[0] {
	case "list", "ls":
		models := []string{
			"gpt-4o",
			"gpt-4o-mini",
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"deepseek-chat",
			"deepseek-reasoner",
		}
		msg := "Available models:\n"
		for _, m := range models {
			marker := "  "
			if m == a.Config.Model.Model {
				marker = "• "
			}
			msg += marker + m + "\n"
		}
		a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}

	case "use", "set":
		if len(args) < 2 {
			a.EventCh <- model.Event{
				Type:    model.AgentReply,
				Message: "Usage: /model use \u003cmodel-name\u003e",
			}
			return
		}
		oldModel := a.Config.Model.Model
		a.Config.Model.Model = args[1]
		msg := fmt.Sprintf("Model changed: %s → %s", oldModel, args[1])
		a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}

	default:
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "Usage: /model [list|use \u003cmodel\u003e]",
		}
	}
}

// ============ 上下文管理命令 ============

func (a *Application) cmdCompact() {
	a.EventCh <- model.Event{Type: model.AgentThinking}
	
	// 获取当前 token 数
	beforeTokens := a.ContextManager.GetTokenCount()
	
	// 执行压缩
	a.ContextManager.Compact()
	
	afterTokens := a.ContextManager.GetTokenCount()
	saved := beforeTokens - afterTokens
	
	msg := fmt.Sprintf("Context compacted: %d → %d tokens (saved %d)",
		beforeTokens, afterTokens, saved)
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

func (a *Application) cmdClear() {
	// 清空上下文，但保留系统提示
	a.ContextManager.Clear()
	
	// 发送会话清空事件，UI 会重置消息列表但保留模型名称
	a.EventCh <- model.Event{
		Type:    model.SessionCleared,
		Message: a.Config.Model.Model,
	}
	
	a.EventCh <- model.Event{
		Type:    model.AgentReply,
		Message: "Session cleared. System prompt preserved.",
	}
}

func (a *Application) cmdNew() {
	// 创建新会话（简化版）
	a.ContextManager.Clear()
	
	a.EventCh <- model.Event{
		Type:    model.SessionCleared,
		Message: a.Config.Model.Model,
	}
	
	a.EventCh <- model.Event{
		Type:    model.AgentReply,
		Message: "New session started. Context cleared.",
	}
}

// ============ 会话持久化命令 ============

func (a *Application) cmdSave(args []string) {
	filename := "session.json"
	if len(args) > 0 {
		filename = args[0]
		if !strings.HasSuffix(filename, ".json") {
			filename += ".json"
		}
	}
	
	// 确保目录存在
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}
	
	// 保存会话数据
	data, err := a.ContextManager.Save()
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "save",
			Message:  err.Error(),
		}
		return
	}
	
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "save",
			Message:  err.Error(),
		}
		return
	}
	
	msg := fmt.Sprintf("Session saved to: %s", filename)
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

func (a *Application) cmdLoad(args []string) {
	if len(args) == 0 {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "Usage: /load \u003cfilename\u003e",
		}
		return
	}
	
	filename := args[0]
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}
	
	data, err := os.ReadFile(filename)
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "load",
			Message:  err.Error(),
		}
		return
	}
	
	err = a.ContextManager.Load(data)
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "load",
			Message:  err.Error(),
		}
		return
	}
	
	a.EventCh <- model.Event{
		Type:    model.SessionCleared,
		Message: a.Config.Model.Model,
	}
	
	msg := fmt.Sprintf("Session loaded from: %s", filename)
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

func (a *Application) cmdHistory() {
	// 列出保存的会话文件
	sessions, err := a.SessionManager.List()
	if err != nil || len(sessions) == 0 {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "No session history found.",
		}
		return
	}
	
	msg := "Session History:\n"
	for i, s := range sessions {
		msg += fmt.Sprintf("%d. %s - %s (%s)\n",
			i+1, s.Name, s.ID[:8], s.CreatedAt.Format("2006-01-02 15:04"))
	}
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

// ============ 工具管理命令 ============

func (a *Application) cmdTools() {
	tools := a.Runner.GetRegistry().List()
	
	msg := "Available Tools:\n"
	for _, name := range tools {
		msg += fmt.Sprintf("  • %s\n", name)
	}
	
	msg += "\nTools are automatically selected by the AI during task execution."
	
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

// ============ 生成参数命令 ============

func (a *Application) cmdTemperature(args []string) {
	if len(args) == 0 {
		msg := fmt.Sprintf("Current temperature: %.2f", a.Config.Model.Temperature)
		a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
		return
	}
	
	temp, err := strconv.ParseFloat(args[0], 64)
	if err != nil || temp < 0 || temp > 2 {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "Invalid temperature. Use a value between 0.0 and 2.0",
		}
		return
	}
	
	a.Config.Model.Temperature = temp
	msg := fmt.Sprintf("Temperature set to: %.2f", temp)
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

func (a *Application) cmdTokens(args []string) {
	if len(args) == 0 {
		msg := fmt.Sprintf("Current max_tokens: %d", a.Config.Model.MaxTokens)
		a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
		return
	}
	
	tokens, err := strconv.Atoi(args[0])
	if err != nil || tokens < 1 {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "Invalid token count. Use a positive integer.",
		}
		return
	}
	
	a.Config.Model.MaxTokens = tokens
	msg := fmt.Sprintf("Max tokens set to: %d", tokens)
	a.EventCh <- model.Event{Type: model.AgentReply, Message: msg}
}

// ============ 帮助命令 ============

func (a *Application) cmdHelp() {
	help := `Available Commands:

Session Management:
  /new              Start a new session (saves current to history)
  /clear            Clear current session (keeps system prompt)
  /save [file]      Save session to file (default: session.json)
  /load [file]      Load session from file
  /history          Show session history

Model Configuration:
  /model            Show current model
  /model list       List available models
  /model use [name] Switch to a different model
  /temperature [n]  Set temperature (0.0-2.0)
  /tokens [n]       Set max tokens

Context Management:
  /compact          Manually compress context

Project Commands:
  /roadmap status   Show project roadmap
  /weekly status    Show weekly update

Other:
  /tools            List available tools
  /help             Show this help message

Keybindings:
  Enter             Send message
  Ctrl+C            Quit
  PgUp/PgDn         Scroll chat
  Up/Down           Scroll chat
  Home/End          Jump to top/bottom
`
	a.EventCh <- model.Event{Type: model.AgentReply, Message: help}
}

// ============ 已有命令（保持不变）===========

// cmdRoadmap handles "/roadmap status [path]".
func (a *Application) cmdRoadmap(args []string) {
	if len(args) == 0 || args[0] != "status" {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "Usage: /roadmap status [path] (default: roadmap.yaml)",
		}
		return
	}

	path := "roadmap.yaml"
	if len(args) > 1 {
		path = args[1]
	}

	a.EventCh <- model.Event{Type: model.AgentThinking}

	rm, err := project.LoadRoadmapFromFile(path)
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "roadmap",
			Message:  err.Error(),
		}
		return
	}

	status, err := project.ComputeRoadmapStatus(rm, time.Now())
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "roadmap",
			Message:  err.Error(),
		}
		return
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	a.EventCh <- model.Event{
		Type:    model.AgentReply,
		Message: string(data),
	}
}

// cmdWeekly handles "/weekly status [path]".
func (a *Application) cmdWeekly(args []string) {
	if len(args) == 0 || args[0] != "status" {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: "Usage: /weekly status [path] (default: weekly.md)",
		}
		return
	}

	path := "weekly.md"
	if len(args) > 1 {
		path = args[1]
	}

	a.EventCh <- model.Event{Type: model.AgentThinking}

	wu, err := project.LoadWeeklyUpdateFromFile(path)
	if err != nil {
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "weekly",
			Message:  err.Error(),
		}
		return
	}

	data, _ := json.MarshalIndent(wu, "", "  ")
	a.EventCh <- model.Event{
		Type:    model.AgentReply,
		Message: string(data),
	}
}
