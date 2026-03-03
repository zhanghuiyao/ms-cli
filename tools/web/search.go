package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vigo999/ms-cli/tools"
)

// WebSearchTool performs web searches using DuckDuckGo.
type WebSearchTool struct {
	client  *http.Client
	maxResults int
}

// NewWebSearchTool creates a new web search tool.
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		maxResults: 10,
	}
}

// SetMaxResults sets the maximum number of results.
func (t *WebSearchTool) SetMaxResults(n int) {
	t.maxResults = n
}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string { return "Search the web for information" }

func (t *WebSearchTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return tools.Result{Success: false, Output: "query parameter is required"}, nil
	}

	results, err := t.search(ctx, query)
	if err != nil {
		return tools.Result{Success: false, Output: err.Error()}, nil
	}

	// Format results
	output := t.formatResults(results)

	return tools.Result{
		Success: true,
		Output:  output,
		Data: map[string]any{
			"query":   query,
			"results": results,
			"count":   len(results),
		},
	}, nil
}

// WebSearchDefinition returns the schema for web_search.
func WebSearchDefinition() tools.Definition {
	return tools.Definition{
		Name:        "web_search",
		Description: "Search the web for current information. Returns titles, URLs, and snippets.",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"query": {
					Type:        "string",
					Description: "Search query",
				},
				"num_results": {
					Type:        "integer",
					Description: "Number of results (default: 10, max: 20)",
				},
			},
			Required: []string{"query"},
		},
	}
}

// SearchResult represents a web search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// search performs a web search using DuckDuckGo Instant Answer API.
func (t *WebSearchTool) search(ctx context.Context, query string) ([]SearchResult, error) {
	// Use DuckDuckGo HTML version (no API key required)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ms-cli/1.0)")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse HTML results (simplified)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return t.parseResults(string(body)), nil
}

// parseResults extracts search results from HTML (simplified).
func (t *WebSearchTool) parseResults(html string) []SearchResult {
	var results []SearchResult

	// Simple regex-like parsing (in production, use proper HTML parser)
	// This is a basic implementation for MVP
	lines := splitHTML(html)

	for i := 0; i < len(lines)-2 && len(results) < t.maxResults; i++ {
		line := lines[i]

		// Look for result titles
		if hasClass(line, "result__a") {
			result := SearchResult{}

			// Extract title and URL
			if title, url := extractLink(line); title != "" {
				result.Title = title
				result.URL = url
			}

			// Look for snippet in next lines
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if hasClass(lines[j], "result__snippet") {
					result.Snippet = stripTags(lines[j])
					break
				}
			}

			if result.Title != "" {
				results = append(results, result)
			}
		}
	}

	return results
}

// formatResults formats search results as text.
func (t *WebSearchTool) formatResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	var output string
	for i, r := range results {
		output += fmt.Sprintf("%d. %s\n   URL: %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet)
	}

	return output
}

// splitHTML splits HTML into lines.
func splitHTML(html string) []string {
	var result []string
	var current string

	for _, ch := range html {
		if ch == '<' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			current += string(ch)
		} else if ch == '>' {
			current += string(ch)
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// hasClass checks if HTML has a class.
func hasClass(html, class string) bool {
	return contains(html, fmt.Sprintf(`class="%s"`, class)) ||
		contains(html, fmt.Sprintf(`class='%s'`, class)) ||
		contains(html, fmt.Sprintf(`class="%s `, class)) ||
		contains(html, fmt.Sprintf(`class='%s `, class))
}

// extractLink extracts title and URL from link HTML.
func extractLink(html string) (string, string) {
	// Extract href
	url := extractAttr(html, "href")

	// Extract text content
	title := stripTags(html)

	return title, url
}

// extractAttr extracts an attribute value.
func extractAttr(html, attr string) string {
	prefix := attr + `="`
	start := indexOf(html, prefix)
	if start == -1 {
		prefix = attr + `='`
		start = indexOf(html, prefix)
	}
	if start == -1 {
		return ""
	}

	start += len(prefix)
	end := indexOf(html[start:], `"`)
	if end == -1 {
		end = indexOf(html[start:], `'`)
	}
	if end == -1 {
		return ""
	}

	return html[start : start+end]
}

// stripTags removes HTML tags.
func stripTags(html string) string {
	var result string
	inTag := false

	for _, ch := range html {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result += string(ch)
		}
	}

	// Decode common entities
	result = replaceEntities(result)

	return trimSpace(result)
}

// replaceEntities decodes HTML entities.
func replaceEntities(s string) string {
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&#39;":  "'",
	}

	for entity, char := range replacements {
		s = replaceAll(s, entity, char)
	}

	return s
}

// Helper functions for string manipulation without regexp.
func contains(s, substr string) bool {
	return indexOf(s, substr) != -1
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func replaceAll(s, old, new string) string {
	var result string
	start := 0

	for {
		idx := indexOf(s[start:], old)
		if idx == -1 {
			result += s[start:]
			break
		}
		result += s[start:start+idx] + new
		start += idx + len(old)
	}

	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
