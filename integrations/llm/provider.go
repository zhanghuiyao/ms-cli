package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolCall represents a tool invocation from the model.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// CompletionRequest is the request to the LLM.
type CompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

// CompletionResponse is the response from the LLM.
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int      `json:"index"`
		Message      Message  `json:"message"`
		ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
		FinishReason string   `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ToolDefinition defines a tool for function calling.
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function FunctionDef  `json:"function"`
}

// FunctionDef is the function schema.
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Provider is the interface for LLM providers.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

// StreamChunk represents a chunk of a streaming response.
type StreamChunk struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool
	Error     error
}

// OpenAIProvider implements Provider for OpenAI API.
type OpenAIProvider struct {
	APIKey   string
	Endpoint string
	Client   *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey, endpoint string) *OpenAIProvider {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Complete sends a completion request to OpenAI.
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	url := p.Endpoint + "/chat/completions"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// Stream sends a streaming completion request.
func (p *OpenAIProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	// Enable streaming
	reqMap := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"stream":      true,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
	if req.Tools != nil {
		reqMap["tools"] = req.Tools
	}

	body, err := json.Marshal(reqMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.Endpoint + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamChunk)
	go p.readStream(resp.Body, ch)

	return ch, nil
}

// readStream reads SSE stream and sends chunks to channel.
func (p *OpenAIProvider) readStream(body io.ReadCloser, ch chan<- StreamChunk) {
	defer body.Close()
	defer close(ch)

	decoder := json.NewDecoder(body)
	for {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				return
			}
			// Send error through channel
			select {
			case ch <- StreamChunk{Error: err, Done: true}:
			default:
			}
			return
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			select {
			case ch <- StreamChunk{
				Content:   choice.Delta.Content,
				ToolCalls: choice.Delta.ToolCalls,
				Done:      choice.FinishReason != "",
			}:
			default:
				return
			}
		}
	}
}

// MockProvider is a mock implementation for testing.
type MockProvider struct {
	Response *CompletionResponse
}

// Complete returns a mock response.
func (p *MockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if p.Response != nil {
		return p.Response, nil
	}
	return &CompletionResponse{
		Choices: []struct {
			Index        int     `json:"index"`
			Message      Message `json:"message"`
			ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
			FinishReason string  `json:"finish_reason"`
		}{{
			Message:      Message{Role: "assistant", Content: "Mock response"},
			FinishReason: "stop",
		}},
	}, nil
}

// Stream returns a mock stream.
func (p *MockProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{Content: "Mock response", Done: true}
	close(ch)
	return ch, nil
}
