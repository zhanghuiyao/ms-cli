package context

import (
	"fmt"
	"sync"
)

// Budget tracks token and cost usage.
type Budget struct {
	maxTokens   int
	maxCost     float64

	usedTokens  int
	usedCost    float64
	sessionRuns int

	mu sync.RWMutex
}

// NewBudget creates a new budget tracker.
func NewBudget(maxTokens int, maxCost float64) *Budget {
	return &Budget{
		maxTokens: maxTokens,
		maxCost:   maxCost,
	}
}

// RecordUsage records token and cost usage.
func (b *Budget) RecordUsage(tokens int, cost float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.usedTokens += tokens
	b.usedCost += cost
}

// RecordSession increments session counter.
func (b *Budget) RecordSession() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessionRuns++
}

// CheckLimit returns error if budget exceeded.
func (b *Budget) CheckLimit() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.maxTokens > 0 && b.usedTokens >= b.maxTokens {
		return fmt.Errorf("token budget exceeded: %d/%d", b.usedTokens, b.maxTokens)
	}
	if b.maxCost > 0 && b.usedCost >= b.maxCost {
		return fmt.Errorf("cost budget exceeded: $%.2f/$%.2f", b.usedCost, b.maxCost)
	}
	return nil
}

// GetUsage returns current usage stats.
func (b *Budget) GetUsage() UsageStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return UsageStats{
		UsedTokens:   b.usedTokens,
		MaxTokens:    b.maxTokens,
		UsedCost:     b.usedCost,
		MaxCost:      b.maxCost,
		SessionRuns:  b.sessionRuns,
	}
}

// UsageStats holds budget usage information.
type UsageStats struct {
	UsedTokens   int
	MaxTokens    int
	UsedCost     float64
	MaxCost      float64
	SessionRuns  int
}

// PercentUsed returns percentage of token budget used.
func (s UsageStats) PercentUsed() float64 {
	if s.MaxTokens <= 0 {
		return 0
	}
	return float64(s.UsedTokens) / float64(s.MaxTokens) * 100
}

// CostPercentUsed returns percentage of cost budget used.
func (s UsageStats) CostPercentUsed() float64 {
	if s.MaxCost <= 0 {
		return 0
	}
	return s.UsedCost / s.MaxCost * 100
}

// Reset resets all usage counters.
func (b *Budget) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.usedTokens = 0
	b.usedCost = 0
}

// EstimateCost estimates cost for tokens based on model.
func EstimateCost(tokens int, model string) float64 {
	// Rough estimates per 1K tokens
	rates := map[string]struct {
		Input  float64
		Output float64
	}{
		"gpt-4":          {0.03, 0.06},
		"gpt-4-turbo":    {0.01, 0.03},
		"gpt-3.5-turbo":  {0.0005, 0.0015},
		"deepseek-r1":    {0.0005, 0.002},
	}

	rate, ok := rates[model]
	if !ok {
		// Default to gpt-3.5 rates
		rate = rates["gpt-3.5-turbo"]
	}

	// Assume 70% input, 30% output split
	inputTokens := float64(tokens) * 0.7
	outputTokens := float64(tokens) * 0.3

	cost := (inputTokens / 1000 * rate.Input) + (outputTokens / 1000 * rate.Output)
	return cost
}
