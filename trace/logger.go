package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event represents a traced event.
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	TaskID    string                 `json:"task_id,omitempty"`
	Tool      string                 `json:"tool,omitempty"`
	Action    string                 `json:"action,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    string                 `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Logger writes execution traces.
type Logger struct {
	path      string
	encoder   *json.Encoder
	file      *os.File
	mu        sync.Mutex
	enabled   bool
	buffer    []Event
	batchSize int
}

// NewLogger creates a new trace logger.
func NewLogger(path string) (*Logger, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create trace directory: %w", err)
	}

	return &Logger{
		path:      path,
		enabled:   true,
		buffer:    make([]Event, 0, 100),
		batchSize: 10,
	}, nil
}

// SetEnabled enables or disables logging.
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// SetBatchSize sets the number of events to buffer before writing.
func (l *Logger) SetBatchSize(n int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.batchSize = n
}

// Log records a single event.
func (l *Logger) Log(event Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.enabled {
		return nil
	}

	event.Timestamp = time.Now()
	l.buffer = append(l.buffer, event)

	if len(l.buffer) >= l.batchSize {
		return l.flush()
	}

	return nil
}

// LogToolCall records a tool invocation.
func (l *Logger) LogToolCall(taskID, tool string, input map[string]interface{}, output string, err error, duration time.Duration) error {
	event := Event{
		Type:     "tool_call",
		TaskID:   taskID,
		Tool:     tool,
		Input:    input,
		Output:   output,
		Duration: duration,
	}
	if err != nil {
		event.Error = err.Error()
	}
	return l.Log(event)
}

// LogTaskStart records task start.
func (l *Logger) LogTaskStart(taskID, description string) error {
	return l.Log(Event{
		Type:   "task_start",
		TaskID: taskID,
		Input: map[string]interface{}{
			"description": description,
		},
	})
}

// LogTaskEnd records task completion.
func (l *Logger) LogTaskEnd(taskID string, success bool, err error) error {
	event := Event{
		Type:   "task_end",
		TaskID: taskID,
		Metadata: map[string]interface{}{
			"success": success,
		},
	}
	if err != nil {
		event.Error = err.Error()
	}
	return l.Log(event)
}

// LogLLMCall records an LLM API call.
func (l *Logger) LogLLMCall(taskID, model string, tokensIn, tokensOut int, duration time.Duration) error {
	return l.Log(Event{
		Type:   "llm_call",
		TaskID: taskID,
		Metadata: map[string]interface{}{
			"model":       model,
			"tokens_in":   tokensIn,
			"tokens_out":  tokensOut,
			"duration_ms": duration.Milliseconds(),
		},
	})
}

// flush writes buffered events to file.
func (l *Logger) flush() error {
	if len(l.buffer) == 0 {
		return nil
	}

	// Open file if not open
	if l.file == nil {
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		l.file = f
		l.encoder = json.NewEncoder(f)
	}

	// Write events
	for _, event := range l.buffer {
		if err := l.encoder.Encode(event); err != nil {
			return err
		}
	}

	// Clear buffer
	l.buffer = l.buffer[:0]
	return nil
}

// Close flushes remaining events and closes the file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.flush(); err != nil {
		return err
	}

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// ReadTraces reads all traces from file.
func ReadTraces(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}

	var events []Event
	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			break
		}
		events = append(events, event)
	}

	return events, nil
}

// Stats aggregates trace statistics.
type Stats struct {
	TotalTasks     int
	TotalToolCalls int
	TotalLLMCalls  int
	TotalErrors    int
	AvgToolTime    time.Duration
	TokensIn       int
	TokensOut      int
	// Fields for backward compatibility with tests
	UsedTokens int
	MaxTokens  int
	UsedCost   float64
	MaxCost    float64
}

// PercentUsed returns percentage of token budget used.
func (s Stats) PercentUsed() float64 {
	if s.MaxTokens <= 0 {
		return 0
	}
	return float64(s.UsedTokens) / float64(s.MaxTokens) * 100
}

// CostPercentUsed returns percentage of cost budget used.
func (s Stats) CostPercentUsed() float64 {
	if s.MaxCost <= 0 {
		return 0
	}
	return s.UsedCost / s.MaxCost * 100
}

// Analyze computes statistics from traces.
func Analyze(events []Event) Stats {
	stats := Stats{}

	var totalToolTime time.Duration
	toolCount := 0

	for _, event := range events {
		switch event.Type {
		case "task_start":
			stats.TotalTasks++
		case "tool_call":
			stats.TotalToolCalls++
			totalToolTime += event.Duration
			toolCount++
		case "llm_call":
			stats.TotalLLMCalls++
			if in, ok := event.Metadata["tokens_in"].(int); ok {
				stats.TokensIn += in
			}
			if out, ok := event.Metadata["tokens_out"].(int); ok {
				stats.TokensOut += out
			}
		}
		if event.Error != "" {
			stats.TotalErrors++
		}
	}

	if toolCount > 0 {
		stats.AvgToolTime = totalToolTime / time.Duration(toolCount)
	}

	return stats
}
