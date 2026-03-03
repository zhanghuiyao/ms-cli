package context

import (
	"testing"
)

func TestManager(t *testing.T) {
	m := NewManager(1000)
	m.SetReserveTokens(100)

	// Add system message
	m.SetSystemMessage("You are a helpful assistant.")

	// Add messages
	m.AddMessage("user", "Hello")
	m.AddMessage("assistant", "Hi there!")

	// Build messages
	msgs := m.BuildMessages()
	if len(msgs) != 3 { // system + user + assistant
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}

	// Check token count
	if m.GetTokenCount() <= 0 {
		t.Error("expected positive token count")
	}
}

func TestShouldCompact(t *testing.T) {
	m := NewManager(1000)
	m.SetReserveTokens(100)
	m.SetThreshold(0.8)

	// Add system message
	m.SetSystemMessage("System prompt here.")

	// Should not compact initially
	if m.ShouldCompact() {
		t.Error("should not need compaction initially")
	}

	// Add many messages to trigger compaction
	for i := 0; i < 50; i++ {
		m.AddMessage("user", "This is a long message that should consume tokens.")
		m.AddMessage("assistant", "This is another long response that should consume tokens.")
	}

	// Should compact now
	if !m.ShouldCompact() {
		t.Error("should need compaction after many messages")
	}
}

func TestCompact(t *testing.T) {
	m := NewManager(1000)
	m.SetReserveTokens(100)

	// Add system message
	m.SetSystemMessage("System prompt.")

	// Add messages
	m.AddMessage("user", "Question 1")
	m.AddMessage("assistant", "Answer 1")
	m.AddMessage("user", "Question 2")
	m.AddMessage("assistant", "Answer 2")

	initialCount := m.GetTokenCount()

	// Compact
	m.Compact()

	// Should have fewer tokens after compaction
	if m.GetTokenCount() > initialCount {
		t.Error("token count should decrease after compaction")
	}

	// Should still have messages
	msgs := m.BuildMessages()
	if len(msgs) == 0 {
		t.Error("should still have messages after compaction")
	}
}

func TestBudget(t *testing.T) {
	b := NewBudget(1000, 5.0)

	// Record usage
	b.RecordUsage(100, 0.01)
	b.RecordUsage(200, 0.02)

	// Check usage
	stats := b.GetUsage()
	if stats.UsedTokens != 300 {
		t.Errorf("expected 300 tokens, got %d", stats.UsedTokens)
	}
	if stats.UsedCost != 0.03 {
		t.Errorf("expected $0.03 cost, got %f", stats.UsedCost)
	}

	// Should be within limits
	if err := b.CheckLimit(); err != nil {
		t.Errorf("unexpected limit error: %v", err)
	}
}

func TestBudgetExceeded(t *testing.T) {
	b := NewBudget(100, 1.0)

	// Record usage exceeding limit
	b.RecordUsage(150, 0.0)

	// Should exceed limit
	if err := b.CheckLimit(); err == nil {
		t.Error("expected limit error")
	}
}
