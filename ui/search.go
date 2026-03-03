package ui

import (
	"strings"

	"github.com/vigo999/ms-cli/ui/model"
)

// SearchResult represents a match in the message history.
type SearchResult struct {
	MessageIndex int
	Message      model.Message
	MatchText    string
	Context      string
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	Query       string
	CaseSensitive bool
	WholeWord   bool
	MaxResults  int
}

// MessageSearch handles searching through chat history.
type MessageSearch struct {
	messages []model.Message
}

// NewMessageSearch creates a new message search.
func NewMessageSearch(messages []model.Message) *MessageSearch {
	return &MessageSearch{
		messages: messages,
	}
}

// SetMessages updates the message list.
func (s *MessageSearch) SetMessages(messages []model.Message) {
	s.messages = messages
}

// Search performs a search through messages.
func (s *MessageSearch) Search(opts SearchOptions) []SearchResult {
	if opts.Query == "" {
		return nil
	}

	if opts.MaxResults <= 0 {
		opts.MaxResults = 50
	}

	var results []SearchResult
	query := opts.Query

	if !opts.CaseSensitive {
		query = strings.ToLower(query)
	}

	for i, msg := range s.messages {
		content := msg.Content
		if !opts.CaseSensitive {
			content = strings.ToLower(content)
		}

		if s.matches(content, query, opts.WholeWord) {
			result := SearchResult{
				MessageIndex: i,
				Message:      msg,
				MatchText:    s.extractMatchText(msg.Content, opts.Query, opts.CaseSensitive),
				Context:      s.extractContext(msg.Content, opts.Query),
			}
			results = append(results, result)

			if len(results) >= opts.MaxResults {
				break
			}
		}
	}

	return results
}

// matches checks if content matches the query.
func (s *MessageSearch) matches(content, query string, wholeWord bool) bool {
	if !wholeWord {
		return strings.Contains(content, query)
	}

	// Whole word matching
	words := strings.Fields(content)
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}\n\t")
		if word == query {
			return true
		}
	}
	return false
}

// extractMatchText extracts the matching portion with surrounding context.
func (s *MessageSearch) extractMatchText(content, query string, caseSensitive bool) string {
	searchContent := content
	if !caseSensitive {
		searchContent = strings.ToLower(content)
		query = strings.ToLower(query)
	}

	idx := strings.Index(searchContent, query)
	if idx == -1 {
		return content
	}

	// Get surrounding context
	start := idx - 20
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 20
	if end > len(content) {
		end = len(content)
	}

	return content[start:end]
}

// extractContext extracts context around the match.
func (s *MessageSearch) extractContext(content, query string) string {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
			return strings.TrimSpace(line)
		}
	}

	return ""
}

// SearchByTool filters messages by tool name.
func (s *MessageSearch) SearchByTool(toolName string) []SearchResult {
	var results []SearchResult

	for i, msg := range s.messages {
		if msg.Kind == model.MsgTool && msg.ToolName == toolName {
			results = append(results, SearchResult{
				MessageIndex: i,
				Message:      msg,
				MatchText:    msg.Content,
			})
		}
	}

	return results
}

// SearchByKind filters messages by kind.
func (s *MessageSearch) SearchByKind(kind model.MessageKind) []SearchResult {
	var results []SearchResult

	for i, msg := range s.messages {
		if msg.Kind == kind {
			results = append(results, SearchResult{
				MessageIndex: i,
				Message:      msg,
				MatchText:    msg.Content,
			})
		}
	}

	return results
}

// GetLastNMessages returns the last N messages.
func (s *MessageSearch) GetLastNMessages(n int) []model.Message {
	if n >= len(s.messages) {
		return s.messages
	}
	return s.messages[len(s.messages)-n:]
}

// GetMessageCount returns total message count.
func (s *MessageSearch) GetMessageCount() int {
	return len(s.messages)
}

// GetToolUsage returns statistics about tool usage.
func (s *MessageSearch) GetToolUsage() map[string]int {
	usage := make(map[string]int)

	for _, msg := range s.messages {
		if msg.Kind == model.MsgTool {
			usage[msg.ToolName]++
		}
	}

	return usage
}
