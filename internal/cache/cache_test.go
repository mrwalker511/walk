package cache

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeBasic(t *testing.T) {
	text := "You are a helpful assistant. Always be concise.\nUser: What is the capital of France?"
	analysis := Analyze(text, "claude-sonnet-4-5")

	assert.NotEmpty(t, analysis.Sections)
	assert.Greater(t, analysis.StableTokens+analysis.DynamicTokens, 0)
}

func TestAnalyzeDetectsDynamicSection(t *testing.T) {
	text := "You are an expert.\nAlways answer carefully.\nUser: Help me with my code please"
	analysis := Analyze(text, "claude-sonnet-4-5")

	hasDynamic := false
	for _, s := range analysis.Sections {
		if s.Type == SectionDynamic {
			hasDynamic = true
			break
		}
	}
	assert.True(t, hasDynamic)
}

func TestAnalyzeDetectsStableSection(t *testing.T) {
	text := "You are a helpful coding assistant.\nAlways write tests.\nUser: write a function"
	analysis := Analyze(text, "claude-sonnet-4-5")

	hasStable := false
	for _, s := range analysis.Sections {
		if s.Type == SectionStable {
			hasStable = true
			break
		}
	}
	assert.True(t, hasStable)
}

func TestAnalyzeSavingsCalculation(t *testing.T) {
	// Stable-heavy text — should show Anthropic savings
	text := "You are an expert assistant.\nAlways be accurate.\nContext: this is all stable system content.\nInstructions: follow these rules carefully."
	analysis := Analyze(text, "claude-sonnet-4-5")

	// If any stable tokens exist, savings should be > 0
	if analysis.StableTokens > 0 {
		assert.Greater(t, analysis.EstimatedSavingsAnthropic, 0.0)
		assert.Greater(t, analysis.EstimatedSavingsOpenAI, 0.0)
	}
}

func TestAnalyzeReorderRecommendation(t *testing.T) {
	// Dynamic content comes before stable content — bad ordering
	text := "User: What should I do?\nYou are a helpful assistant.\nAlways follow these rules."
	analysis := Analyze(text, "claude-sonnet-4-5")

	// Should recommend reordering since dynamic is before stable
	assert.True(t, analysis.ReorderRecommended)
	require.NotEmpty(t, analysis.Recommendations)
}

func TestAnalyzeSectionTokens(t *testing.T) {
	text := "You are a helpful assistant.\nUser: help me"
	analysis := Analyze(text, "claude-sonnet-4-5")

	for _, s := range analysis.Sections {
		assert.Greater(t, s.Tokens, 0)
	}
}

func TestClassifySections(t *testing.T) {
	lines := []string{
		"You are an expert.",
		"Always be concise.",
		"User: What is the answer?",
	}
	sections := classifySections(lines)
	require.NotEmpty(t, sections)

	// First section should be stable
	assert.Equal(t, SectionStable, sections[0].Type)
}

func TestAnalyzeNoReorderNeeded(t *testing.T) {
	// Stable before dynamic — correct ordering, should not recommend reorder.
	text := "You are a helpful assistant.\nAlways be concise.\nUser: What is the capital of France?"
	analysis := Analyze(text, "claude-sonnet-4-5")
	assert.False(t, analysis.ReorderRecommended)
}

func TestAnalyzeEmptyText(t *testing.T) {
	// Should not panic and produce sensible zero values.
	analysis := Analyze("", "claude-sonnet-4-5")
	assert.Equal(t, 0, analysis.StableTokens)
	assert.Equal(t, 0, analysis.DynamicTokens)
	assert.Equal(t, 0.0, analysis.EstimatedSavingsAnthropic)
	assert.Equal(t, 0.0, analysis.EstimatedSavingsOpenAI)
	assert.False(t, analysis.ReorderRecommended)
}

func TestAnalyzeAllStable(t *testing.T) {
	// Text with only stable keywords — no dynamic sections, no reorder needed.
	text := "You are an expert assistant.\nAlways answer accurately.\nContext: system-level guidance."
	analysis := Analyze(text, "claude-sonnet-4-5")
	assert.Greater(t, analysis.StableTokens, 0)
	assert.Equal(t, 0, analysis.DynamicTokens)
	assert.False(t, analysis.ReorderRecommended)
}

func TestAnalyzeZeroSavingsNoStableTokens(t *testing.T) {
	// Pure dynamic content — no stable tokens, savings must be zero.
	text := "User: Can you help me?\nHuman: I need assistance please."
	analysis := Analyze(text, "claude-sonnet-4-5")
	assert.Equal(t, 0, analysis.StableTokens)
	assert.Equal(t, 0.0, analysis.EstimatedSavingsAnthropic)
	assert.Equal(t, 0.0, analysis.EstimatedSavingsOpenAI)
}

func TestAnalyzeDynamicHeavyRecommendation(t *testing.T) {
	// When dynamic tokens outnumber stable tokens, a third recommendation is added.
	// One stable line, many dynamic lines.
	text := "You are helpful.\n" +
		"User: question one\nUser: question two\nUser: question three\n" +
		"Human: please respond\nHuman: need help\nHuman: can you assist"
	analysis := Analyze(text, "claude-sonnet-4-5")
	if analysis.DynamicTokens > analysis.StableTokens && analysis.StableTokens > 0 {
		found := false
		for _, r := range analysis.Recommendations {
			if strings.Contains(r, "static system prompt") {
				found = true
			}
		}
		assert.True(t, found, "should recommend extracting more instructions to static system prompt")
	}
}

func TestAnalyzeSavingsFormula(t *testing.T) {
	// Verify savings formula: Anthropic uses (3.00-0.30)/1M, OpenAI uses (2.50-1.25)/1M.
	// Use a purely stable text so stableToks is well-defined.
	text := "You are a helpful assistant.\nAlways be concise and accurate.\nContext: stable system prompt here."
	analysis := Analyze(text, "claude-sonnet-4-5")
	if analysis.StableTokens > 0 {
		expectedAnthropic := float64(analysis.StableTokens) * (3.00 - 0.30) / 1_000_000
		expectedOpenAI := float64(analysis.StableTokens) * (2.50 - 1.25) / 1_000_000
		assert.InDelta(t, expectedAnthropic, analysis.EstimatedSavingsAnthropic, 1e-9)
		assert.InDelta(t, expectedOpenAI, analysis.EstimatedSavingsOpenAI, 1e-9)
	}
}

