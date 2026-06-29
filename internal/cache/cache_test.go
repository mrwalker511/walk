package cache

import (
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
