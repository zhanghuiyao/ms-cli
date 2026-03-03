package context

import (
	"fmt"
	"strings"
)

// Compactor handles context compaction strategies.
type Compactor struct {
	minHistory    int  // Minimum messages to keep
	keepRecent    int  // Number of recent messages to always keep
	useLLMForSummary bool // Whether to use LLM for summarization
}

// NewCompactor creates a new compactor with default settings.
func NewCompactor() *Compactor {
	return &Compactor{
		minHistory:       4,
		keepRecent:       6,
		useLLMForSummary: false, // Default to simple summarization
	}
}

// SetMinHistory sets the minimum number of messages to preserve.
func (c *Compactor) SetMinHistory(n int) {
	c.minHistory = n
}

// SetKeepRecent sets the number of recent messages to keep intact.
func (c *Compactor) SetKeepRecent(n int) {
	c.keepRecent = n
}

// Compact performs compaction on the given messages.
// Returns the compacted messages and a summary of what was removed.
func (c *Compactor) Compact(messages []Message) ([]Message, string, error) {
	if len(messages) <= c.minHistory {
		return messages, "", nil // Nothing to compact
	}

	// Determine how many messages to keep vs summarize
	keepCount := c.keepRecent
	if keepCount > len(messages) {
		keepCount = len(messages) - 2
	}
	if keepCount < 2 {
		keepCount = 2
	}

	summarizeIdx := len(messages) - keepCount
	if summarizeIdx <= 0 {
		return messages, "", nil
	}

	toSummarize := messages[:summarizeIdx]
	keep := messages[summarizeIdx:]

	// Generate summary
	summary := c.summarize(toSummarize)

	// Create summary message
	summaryMsg := Message{
		Role:    "system",
		Content: fmt.Sprintf("[Earlier conversation summary]: %s", summary),
		Tokens:  estimateTokens(summary),
	}

	// Combine: summary + kept messages
	result := append([]Message{summaryMsg}, keep...)

	return result, summary, nil
}

// summarize creates a summary of messages.
func (c *Compactor) summarize(msgs []Message) string {
	if c.useLLMForSummary {
		// In a real implementation, this would call an LLM
		// For now, fall back to simple summarization
		return c.simpleSummarize(msgs)
	}
	return c.simpleSummarize(msgs)
}

// simpleSummarize creates a simple text summary.
func (c *Compactor) simpleSummarize(msgs []Message) string {
	var parts []string
	
	// Group by role
	userMsgs := 0
	assistantMsgs := 0
	toolCalls := 0
	
	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			userMsgs++
			// Capture first user message as topic
			if userMsgs == 1 && len(parts) == 0 {
				content := truncate(msg.Content, 80)
				parts = append(parts, fmt.Sprintf("Task: %s", content))
			}
		case "assistant":
			assistantMsgs++
		case "tool":
			toolCalls++
		}
	}
	
	// Add statistics
	stats := fmt.Sprintf("(%d user msgs, %d assistant msgs, %d tool calls)", 
		userMsgs, assistantMsgs, toolCalls)
	parts = append(parts, stats)
	
	return strings.Join(parts, "; ")
}

// CompactWithLLM performs LLM-based summarization (placeholder for future implementation).
func (c *Compactor) CompactWithLLM(messages []Message, summarizer func(string) (string, error)) ([]Message, error) {
	if len(messages) <= c.minHistory {
		return messages, nil
	}

	// Convert messages to text for summarization
	var text strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&text, "%s: %s\n", msg.Role, msg.Content)
	}

	summary, err := summarizer(text.String())
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	// Return just the summary as a system message
	return []Message{{
		Role:    "system",
		Content: fmt.Sprintf("[Conversation summary]: %s", summary),
		Tokens:  estimateTokens(summary),
	}}, nil
}
