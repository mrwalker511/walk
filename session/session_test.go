package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudgetAlert_Defaults(t *testing.T) {
	alert := &BudgetAlert{
		SessionID:  "test-session",
		TokenLimit: 1000,
		TokensUsed: 500,
		CostLimit:  1.0,
		CostSpent:  0.25,
	}
	assert.False(t, alert.TokenExceeded)
	assert.False(t, alert.CostExceeded)
}

func TestBudgetAlert_Exceeded(t *testing.T) {
	alert := &BudgetAlert{
		SessionID:     "test-session",
		TokenLimit:    1000,
		TokensUsed:    1500,
		CostLimit:     1.0,
		CostSpent:     2.5,
		TokenExceeded: true,
		CostExceeded:  true,
	}
	assert.True(t, alert.TokenExceeded)
	assert.True(t, alert.CostExceeded)
}

func TestEntry_Defaults(t *testing.T) {
	e := &Entry{
		SessionID:    "sess-1",
		Provider:     "openai",
		Model:        "gpt-4o",
		InputTokens:  100,
		OutputTokens: 50,
		Cost:         0.00375,
	}
	assert.Equal(t, "sess-1", e.SessionID)
	assert.Equal(t, 100, e.InputTokens)
	assert.Equal(t, 50, e.OutputTokens)
}

func TestSessionSummary_Getters(t *testing.T) {
	s := &SessionSummary{
		SessionID:   "sess-1",
		TotalInput:  1000,
		TotalOutput: 500,
		TotalTokens: 1500,
		TotalCost:   0.05,
		EntryCount:  10,
	}
	assert.Equal(t, 1000, s.TotalInput)
	assert.Equal(t, 1500, s.TotalTokens)
	assert.Equal(t, 0.05, s.TotalCost)
}