package ui

import (
	"strings"
	"testing"

	"github.com/vigo999/ms-cli/ui/model"
)

func TestMessageSearch(t *testing.T) {
	messages := []model.Message{
		{Kind: model.MsgUser, Content: "Hello world"},
		{Kind: model.MsgAgent, Content: "Hi there!"},
		{Kind: model.MsgTool, Content: "tool output", ToolName: "test"},
		{Kind: model.MsgAgent, Content: "Hello again"},
	}

	search := NewMessageSearch(messages)

	// Test basic search
	results := search.Search(SearchOptions{
		Query: "hello",
	})

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Test case sensitive search
	results = search.Search(SearchOptions{
		Query:         "Hello",
		CaseSensitive: true,
	})

	if len(results) != 2 {
		t.Errorf("expected 2 case-sensitive results, got %d", len(results))
	}

	// Test whole word search
	results = search.Search(SearchOptions{
		Query:     "wor",
		WholeWord: true,
	})

	if len(results) != 0 {
		t.Error("expected 0 whole word matches for 'wor'")
	}

	// Test max results
	results = search.Search(SearchOptions{
		Query:      "o",
		MaxResults: 2,
	})

	if len(results) != 2 {
		t.Errorf("expected 2 results with limit, got %d", len(results))
	}
}

func TestSearchByTool(t *testing.T) {
	messages := []model.Message{
		{Kind: model.MsgTool, ToolName: "fs_read", Content: "content 1"},
		{Kind: model.MsgTool, ToolName: "shell_exec", Content: "content 2"},
		{Kind: model.MsgTool, ToolName: "fs_read", Content: "content 3"},
	}

	search := NewMessageSearch(messages)
	results := search.SearchByTool("fs_read")

	if len(results) != 2 {
		t.Errorf("expected 2 fs_read results, got %d", len(results))
	}
}

func TestSearchByKind(t *testing.T) {
	messages := []model.Message{
		{Kind: model.MsgUser, Content: "user message"},
		{Kind: model.MsgAgent, Content: "agent message"},
		{Kind: model.MsgTool, Content: "tool message"},
	}

	search := NewMessageSearch(messages)
	results := search.SearchByKind(model.MsgUser)

	if len(results) != 1 {
		t.Errorf("expected 1 user message, got %d", len(results))
	}

	if !strings.Contains(results[0].Message.Content, "user") {
		t.Error("expected user message content")
	}
}

func TestGetLastNMessages(t *testing.T) {
	messages := []model.Message{
		{Content: "message 1"},
		{Content: "message 2"},
		{Content: "message 3"},
		{Content: "message 4"},
		{Content: "message 5"},
	}

	search := NewMessageSearch(messages)
	last3 := search.GetLastNMessages(3)

	if len(last3) != 3 {
		t.Errorf("expected 3 messages, got %d", len(last3))
	}

	if last3[0].Content != "message 3" {
		t.Errorf("expected 'message 3', got '%s'", last3[0].Content)
	}
}

func TestGetLastNMessagesMoreThanAvailable(t *testing.T) {
	messages := []model.Message{
		{Content: "message 1"},
		{Content: "message 2"},
	}

	search := NewMessageSearch(messages)
	result := search.GetLastNMessages(10)

	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}
}

func TestGetMessageCount(t *testing.T) {
	messages := []model.Message{
		{Content: "message 1"},
		{Content: "message 2"},
	}

	search := NewMessageSearch(messages)
	if search.GetMessageCount() != 2 {
		t.Errorf("expected count 2, got %d", search.GetMessageCount())
	}
}

func TestGetToolUsage(t *testing.T) {
	messages := []model.Message{
		{Kind: model.MsgTool, ToolName: "fs_read"},
		{Kind: model.MsgTool, ToolName: "fs_read"},
		{Kind: model.MsgTool, ToolName: "shell_exec"},
	}

	search := NewMessageSearch(messages)
	usage := search.GetToolUsage()

	if usage["fs_read"] != 2 {
		t.Errorf("expected fs_read count 2, got %d", usage["fs_read"])
	}

	if usage["shell_exec"] != 1 {
		t.Errorf("expected shell_exec count 1, got %d", usage["shell_exec"])
	}
}

func TestExtractMatchText(t *testing.T) {
	search := NewMessageSearch(nil)
	
	content := "This is a long message with the search term in the middle of it"
	result := search.extractMatchText(content, "search term", false)

	if !strings.Contains(result, "search term") {
		t.Error("result should contain the search term")
	}
}

func TestEmptySearch(t *testing.T) {
	messages := []model.Message{
		{Content: "message 1"},
	}

	search := NewMessageSearch(messages)
	results := search.Search(SearchOptions{
		Query: "",
	})

	if results != nil {
		t.Error("expected nil results for empty query")
	}
}
