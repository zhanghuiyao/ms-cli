package context

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Message represents a message in the conversation.
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Tokens    int       `json:"tokens"`
}

// Manager handles context assembly, budget tracking, and compaction.
type Manager struct {
	mu            sync.RWMutex // Protects all fields below
	maxTokens     int
	reserveTokens int
	threshold     float64
	history       []Message
	systemMsg     *Message
	totalUsed     int
}

// NewManager creates a new context manager.
func NewManager(maxTokens int) *Manager {
	return &Manager{
		maxTokens:     maxTokens,
		reserveTokens: 2000,
		threshold:     0.85,
		history:       make([]Message, 0),
	}
}

// SetReserveTokens sets the reserve tokens for response.
func (m *Manager) SetReserveTokens(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reserveTokens = n
}

// SetThreshold sets the compaction threshold (0-1).
func (m *Manager) SetThreshold(t float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threshold = t
}

// SetSystemMessage sets the system message.
func (m *Manager) SetSystemMessage(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tokens := estimateTokens(content)
	// Subtract old system message tokens if exists
	if m.systemMsg != nil {
		m.totalUsed -= m.systemMsg.Tokens
	}
	m.systemMsg = &Message{
		Role:      "system",
		Content:   content,
		Timestamp: time.Now(),
		Tokens:    tokens,
	}
	m.totalUsed += tokens
}

// AddMessage adds a message to history.
func (m *Manager) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokens := estimateTokens(content)
	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Tokens:    tokens,
	}
	m.history = append(m.history, msg)
	m.totalUsed += tokens
}

// AddToolResult adds a tool execution result.
func (m *Manager) AddToolResult(toolName, result string) {
	content := fmt.Sprintf("Tool %s result:\n%s", toolName, result)
	m.AddMessage("tool", content)
}

// ShouldCompact returns true if context should be compacted.
func (m *Manager) ShouldCompact() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.shouldCompactLocked()
}

// shouldCompactLocked must be called with mu held
func (m *Manager) shouldCompactLocked() bool {
	available := m.maxTokens - m.reserveTokens
	return float64(m.totalUsed) > float64(available)*m.threshold
}

// Compact reduces context size by summarizing old messages.
func (m *Manager) Compact() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.compactLocked()
}

// compactLocked must be called with mu held
func (m *Manager) compactLocked() {
	if len(m.history) <= 2 {
		return // Keep minimum history
	}

	// Strategy: Keep recent messages, summarize older ones
	keepCount := len(m.history) / 3
	if keepCount < 2 {
		keepCount = 2
	}

	summarizeIdx := len(m.history) - keepCount
	toSummarize := m.history[:summarizeIdx]
	keep := m.history[summarizeIdx:]

	// Create summary (in real implementation, this would use LLM)
	summary := m.summarizeMessages(toSummarize)

	// Replace old messages with summary
	m.history = append([]Message{{
		Role:      "system",
		Content:   "[Earlier conversation summary]: " + summary,
		Timestamp: time.Now(),
		Tokens:    estimateTokens(summary),
	}}, keep...)

	// Recalculate total
	m.recalculateTokensLocked()
}

// summarizeMessages creates a summary of messages (placeholder).
func (m *Manager) summarizeMessages(msgs []Message) string {
	return fmt.Sprintf("(%d msgs)", len(msgs))
}

// recalculateTokensLocked must be called with mu held
func (m *Manager) recalculateTokensLocked() {
	m.totalUsed = 0
	if m.systemMsg != nil {
		m.totalUsed += m.systemMsg.Tokens
	}
	for _, msg := range m.history {
		m.totalUsed += msg.Tokens
	}
}

// BuildMessages returns the complete message list for LLM.
func (m *Manager) BuildMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, 0, len(m.history)+1)
	if m.systemMsg != nil {
		result = append(result, *m.systemMsg)
	}
	result = append(result, m.history...)
	return result
}

// BuildLLMMessages converts to LLM provider format.
func (m *Manager) BuildLLMMessages() []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
} {
	msgs := m.BuildMessages()
	result := make([]struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}, len(msgs))
	for i, msg := range msgs {
		result[i] = struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// GetTokenCount returns current token usage.
func (m *Manager) GetTokenCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalUsed
}

// GetRemainingTokens returns available tokens.
func (m *Manager) GetRemainingTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxTokens - m.totalUsed - m.reserveTokens
}

// Clear clears all history.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = make([]Message, 0)
	m.totalUsed = 0
	if m.systemMsg != nil {
		m.totalUsed = m.systemMsg.Tokens
	}
}

// Save persists context to JSON.
func (m *Manager) Save() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := struct {
		System    *Message  `json:"system,omitempty"`
		History   []Message `json:"history"`
		TotalUsed int       `json:"total_used"`
	}{
		System:    m.systemMsg,
		History:   m.history,
		TotalUsed: m.totalUsed,
	}
	return json.MarshalIndent(data, "", "  ")
}

// Load restores context from JSON.
func (m *Manager) Load(data []byte) error {
	var saved struct {
		System    *Message  `json:"system,omitempty"`
		History   []Message `json:"history"`
		TotalUsed int       `json:"total_used"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.systemMsg = saved.System
	m.history = saved.History
	m.totalUsed = saved.TotalUsed
	return nil
}

// estimateTokens provides a rough token estimate.
func estimateTokens(text string) int {
	// Simple approximation: ~4 characters per token on average
	return len(text) / 4
}

// truncate truncates string to max length with ellipsis.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
