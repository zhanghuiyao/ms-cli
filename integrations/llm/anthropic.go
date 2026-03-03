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

// AnthropicProvider implements Provider for Anthropic Claude API.
type AnthropicProvider struct {
	APIKey   string
	Endpoint string
	Client   *http.Client
}

// NewAnthropicProvider creates a new Anthropic Claude provider.
func NewAnthropicProvider(apiKey, endpoint string) *AnthropicProvider {
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1"
	}
	return &AnthropicProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Complete sends a completion request to Anthropic.
func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	url := p.Endpoint + "/messages"

	// Convert to Anthropic format
	anthropicReq := p.toAnthropicRequest(req)

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", p.APIKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return p.toCompletionResponse(&anthropicResp), nil
}

// Stream sends a streaming completion request.
func (p *AnthropicProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	url := p.Endpoint + "/messages"

	anthropicReq := p.toAnthropicRequest(req)
	anthropicReq["stream"] = true

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", p.APIKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

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

// readStream reads SSE stream.
func (p *AnthropicProvider) readStream(body io.ReadCloser, ch chan<- StreamChunk) {
	defer body.Close()
	defer close(ch)

	decoder := json.NewDecoder(body)
	for {
		var chunk struct {
			Type    string `json:"type"`
			Delta   *struct {
				Text string `json:"text"`
			} `json:"delta"`
			ContentBlock *struct {
				Text string `json:"text"`
			} `json:"content_block"`
			Message *anthropicResponse `json:"message"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				return
			}
			ch <- StreamChunk{Error: err, Done: true}
			return
		}

		switch chunk.Type {
		case "content_block_delta":
			if chunk.Delta != nil {
				ch <- StreamChunk{Content: chunk.Delta.Text}
			}
		case "message_stop":
			ch <- StreamChunk{Done: true}
			return
		}
	}
}

// toAnthropicRequest converts generic request to Anthropic format.
func (p *AnthropicProvider) toAnthropicRequest(req CompletionRequest) map[string]interface{} {
	// Convert messages to Anthropic format
	var system string
	var messages []map[string]string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			system = msg.Content
		} else {
			role := msg.Role
			if role == "tool" {
				role = "assistant" // Claude doesn't have "tool" role
			}
			messages = append(messages, map[string]string{
				"role":    role,
				"content": msg.Content,
			})
		}
	}

	result := map[string]interface{}{
		"model":     req.Model,
		"max_tokens": req.MaxTokens,
		"messages":  messages,
	}

	if system != "" {
		result["system"] = system
	}

	if req.Temperature > 0 {
		result["temperature"] = req.Temperature
	}

	// Convert tools to Anthropic format
	if len(req.Tools) > 0 {
		result["tools"] = p.toAnthropicTools(req.Tools)
	}

	return result
}

// toAnthropicTools converts tool definitions.
func (p *AnthropicProvider) toAnthropicTools(tools []ToolDefinition) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		}
	}
	return result
}

// toCompletionResponse converts Anthropic response to generic format.
func (p *AnthropicProvider) toCompletionResponse(resp *anthropicResponse) *CompletionResponse {
	result := &CompletionResponse{
		Model:   resp.Model,
		Choices: make([]struct {
			Index        int         `json:"index"`
			Message      Message     `json:"message"`
			ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
			FinishReason string      `json:"finish_reason"`
		}, 1),
	}

	if len(resp.Content) > 0 {
		content := resp.Content[0]
		if content.Type == "text" {
			result.Choices[0].Message = Message{
				Role:    "assistant",
				Content: content.Text,
			}
		} else if content.Type == "tool_use" {
			result.Choices[0].ToolCalls = []ToolCall{{
				ID:   content.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      content.Name,
					Arguments: string(content.Input),
				},
			}}
		}
	}

	result.Usage.PromptTokens = resp.Usage.InputTokens
	result.Usage.CompletionTokens = resp.Usage.OutputTokens
	result.Usage.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens

	return result
}

// anthropicRequest represents Anthropic API request.
type anthropicRequest struct {
	Model      string                   `json:"model"`
	MaxTokens  int                      `json:"max_tokens"`
	System     string                   `json:"system,omitempty"`
	Messages   []map[string]string      `json:"messages"`
	Tools      []map[string]interface{} `json:"tools,omitempty"`
	Stream     bool                     `json:"stream,omitempty"`
}

// anthropicResponse represents Anthropic API response.
type anthropicResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type  string `json:"type"`
		Text  string `json:"text,omitempty"`
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
