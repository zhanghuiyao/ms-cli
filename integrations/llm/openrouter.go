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

// OpenRouterProvider implements Provider for OpenRouter API.
// OpenRouter provides unified access to multiple AI models through a single API.
type OpenRouterProvider struct {
	APIKey   string
	Endpoint string
	Client   *http.Client
	SiteURL  string // Optional: for rankings on openrouter.ai
	AppName  string // Optional: for rankings on openrouter.ai
}

// NewOpenRouterProvider creates a new OpenRouter provider.
func NewOpenRouterProvider(apiKey, endpoint string) *OpenRouterProvider {
	if endpoint == "" {
		endpoint = "https://openrouter.ai/api/v1"
	}
	return &OpenRouterProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: 120 * time.Second},
		AppName:  "ms-cli",
	}
}

// SetSiteURL sets the site URL for OpenRouter rankings.
func (p *OpenRouterProvider) SetSiteURL(url string) {
	p.SiteURL = url
}

// SetAppName sets the app name for OpenRouter rankings.
func (p *OpenRouterProvider) SetAppName(name string) {
	p.AppName = name
}

// Complete sends a completion request to OpenRouter.
func (p *OpenRouterProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	url := p.Endpoint + "/chat/completions"

	// OpenRouter uses OpenAI-compatible format
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
	httpReq.Header.Set("HTTP-Referer", p.SiteURL)
	httpReq.Header.Set("X-Title", p.AppName)

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
func (p *OpenRouterProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	url := p.Endpoint + "/chat/completions"

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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("HTTP-Referer", p.SiteURL)
	httpReq.Header.Set("X-Title", p.AppName)
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
func (p *OpenRouterProvider) readStream(body io.ReadCloser, ch chan<- StreamChunk) {
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

// GetAvailableModels returns the list of available models from OpenRouter.
func (p *OpenRouterProvider) GetAvailableModels(ctx context.Context) ([]ModelInfo, error) {
	url := p.Endpoint + "/models"

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

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

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data, nil
}

// ModelInfo represents model information from OpenRouter.
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Pricing  struct {
		Prompt     float64 `json:"prompt"`
		Completion float64 `json:"completion"`
	} `json:"pricing"`
}
