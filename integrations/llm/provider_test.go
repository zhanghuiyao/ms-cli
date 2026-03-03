package llm

import (
	"context"
	"testing"
)

func TestOpenAIProviderMock(t *testing.T) {
	// Use mock provider for testing
	mock := &MockProvider{
		Response: &CompletionResponse{
			Choices: []struct {
				Index        int         `json:"index"`
				Message      Message     `json:"message"`
				ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Message:      Message{Role: "assistant", Content: "Test response"},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
	}

	req := CompletionRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := mock.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Test response" {
		t.Errorf("expected 'Test response', got '%s'", resp.Choices[0].Message.Content)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestMockProviderStream(t *testing.T) {
	mock := &MockProvider{}

	req := CompletionRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	stream, err := mock.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunkCount := 0
	for chunk := range stream {
		chunkCount++
		if chunk.Done {
			break
		}
	}

	if chunkCount != 1 {
		t.Errorf("expected 1 chunk, got %d", chunkCount)
	}
}

func TestToolDefinition(t *testing.T) {
	tool := ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]string{
					"param1": "string",
				},
			},
		},
	}

	if tool.Type != "function" {
		t.Error("expected type 'function'")
	}

	if tool.Function.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got '%s'", tool.Function.Name)
	}
}

func TestMessageStruct(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", msg.Role)
	}

	if msg.Content != "Hello" {
		t.Errorf("expected content 'Hello', got '%s'", msg.Content)
	}
}
