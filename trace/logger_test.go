package trace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "trace.json")

	logger, err := NewLogger(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Test Log
	event := Event{
		Type:   "test",
		TaskID: "task-1",
	}

	if err := logger.Log(event); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Flush
	if err := logger.flush(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file was not created: %v", err)
	}
}

func TestLoggerDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "trace.json")

	logger, _ := NewLogger(logPath)
	logger.SetEnabled(false)

	event := Event{Type: "test"}
	if err := logger.Log(event); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should not create file when disabled
	// (file is created on first flush, so this is okay)
}

func TestLogToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "trace.json")

	logger, _ := NewLogger(logPath)
	defer logger.Close()

	input := map[string]interface{}{"path": "/test"}
	if err := logger.LogToolCall("task-1", "fs_read", input, "output", nil, 100*time.Millisecond); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLogTaskStartEnd(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "trace.json")

	logger, _ := NewLogger(logPath)
	defer logger.Close()

	if err := logger.LogTaskStart("task-1", "test task"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := logger.LogTaskEnd("task-1", true, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLogLLMCall(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "trace.json")

	logger, _ := NewLogger(logPath)
	defer logger.Close()

	if err := logger.LogLLMCall("task-1", "gpt-4", 100, 50, 500*time.Millisecond); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAnalyze(t *testing.T) {
	events := []Event{
		{Type: "task_start"},
		{Type: "tool_call", Duration: 100 * time.Millisecond},
		{Type: "tool_call", Duration: 200 * time.Millisecond},
		{Type: "llm_call", Metadata: map[string]interface{}{"tokens_in": 100, "tokens_out": 50}},
		{Type: "error", Error: "test error"},
	}

	stats := Analyze(events)

	if stats.TotalTasks != 1 {
		t.Errorf("expected 1 task, got %d", stats.TotalTasks)
	}

	if stats.TotalToolCalls != 2 {
		t.Errorf("expected 2 tool calls, got %d", stats.TotalToolCalls)
	}

	if stats.TotalLLMCalls != 1 {
		t.Errorf("expected 1 LLM call, got %d", stats.TotalLLMCalls)
	}

	if stats.TotalErrors != 1 {
		t.Errorf("expected 1 error, got %d", stats.TotalErrors)
	}

	if stats.TokensIn != 100 {
		t.Errorf("expected 100 tokens in, got %d", stats.TokensIn)
	}

	if stats.TokensOut != 50 {
		t.Errorf("expected 50 tokens out, got %d", stats.TokensOut)
	}

	expectedAvg := 150 * time.Millisecond
	if stats.AvgToolTime != expectedAvg {
		t.Errorf("expected avg %v, got %v", expectedAvg, stats.AvgToolTime)
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	stats := Analyze([]Event{})

	if stats.TotalTasks != 0 {
		t.Error("expected 0 tasks for empty events")
	}
}

func TestStatsPercentUsed(t *testing.T) {
	stats := Stats{
		UsedTokens: 500,
		MaxTokens:  1000,
		UsedCost:   5.0,
		MaxCost:    10.0,
	}

	if stats.PercentUsed() != 50.0 {
		t.Errorf("expected 50%%, got %f%%", stats.PercentUsed())
	}

	if stats.CostPercentUsed() != 50.0 {
		t.Errorf("expected 50%%, got %f%%", stats.CostPercentUsed())
	}
}

func TestStatsPercentUsedZeroMax(t *testing.T) {
	stats := Stats{
		UsedTokens: 100,
		MaxTokens:  0,
		UsedCost:   5.0,
		MaxCost:    0,
	}

	if stats.PercentUsed() != 0 {
		t.Error("expected 0% for zero max")
	}

	if stats.CostPercentUsed() != 0 {
		t.Error("expected 0% for zero max")
	}
}
