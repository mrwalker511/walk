package analyzer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeBasic(t *testing.T) {
	text := "This is a sample prompt with some content for testing the analyzer output."
	report := Analyze(text, "claude-sonnet-4-5", 100)

	assert.Greater(t, report.TokenCount, 0)
	assert.Greater(t, report.WordCount, 0)
	assert.Equal(t, 1, report.LineCount)
	assert.Greater(t, report.EstimatedOutput, 0)
	assert.False(t, report.HasSecrets)
	assert.Equal(t, "claude-sonnet-4-5", report.Model)
}

func TestAnalyzeCostCalculation(t *testing.T) {
	// 1000 tokens at claude-sonnet-4-5 = $0.003 input
	// Build a text that produces ~1000 tokens
	word := "word "
	text := strings.Repeat(word, 800) // ~800 words ≈ ~1000 tokens
	report := Analyze(text, "claude-sonnet-4-5", 100)

	assert.Greater(t, report.InputCost, 0.0)
	assert.Greater(t, report.OutputCost, 0.0)
	assert.Greater(t, report.TotalCost, report.InputCost)
}

func TestAnalyzeDetectsSecrets(t *testing.T) {
	text := "API key: sk-testABCDEFGHIJKLMNOPQRSTUVWXYZ and more text"
	report := Analyze(text, "claude-sonnet-4-5", 100)

	assert.True(t, report.HasSecrets)
	require.NotEmpty(t, report.Warnings)

	hasSecretWarning := false
	for _, w := range report.Warnings {
		if w.Code == "SECRET_API_KEY" {
			hasSecretWarning = true
			assert.Equal(t, SeverityError, w.Severity)
			break
		}
	}
	assert.True(t, hasSecretWarning)
}

func TestAnalyzeDetectsRepetition(t *testing.T) {
	block := "You are a helpful assistant.\nPlease be concise.\nDo not make up facts.\nAlways cite sources.\nBe professional.\n"
	// Repeat the block so it appears twice
	text := block + "\nSome other content here.\n" + block
	report := Analyze(text, "claude-sonnet-4-5", 100)

	hasDuplicateWarning := false
	for _, w := range report.Warnings {
		if w.Code == "DUPLICATE_BLOCK" {
			hasDuplicateWarning = true
			break
		}
	}
	assert.True(t, hasDuplicateWarning, "should detect duplicate block")
}

func TestAnalyzeContextFillWarning(t *testing.T) {
	// Build a very large text to exceed 60k tokens
	line := strings.Repeat("word ", 100)
	text := strings.Repeat(line+"\n", 700) // ~70000 tokens
	report := Analyze(text, "claude-sonnet-4-5", 100)

	hasContextWarning := false
	for _, w := range report.Warnings {
		if w.Code == "CONTEXT_FILL" {
			hasContextWarning = true
			break
		}
	}
	assert.True(t, hasContextWarning)
}

func TestAnalyzeLongLine(t *testing.T) {
	longLine := strings.Repeat("x", 600)
	text := "Normal line\n" + longLine + "\nAnother normal line"
	report := Analyze(text, "claude-sonnet-4-5", 100)

	hasLongLine := false
	for _, w := range report.Warnings {
		if w.Code == "LONG_LINE" {
			hasLongLine = true
			break
		}
	}
	assert.True(t, hasLongLine)
}

func TestAnalyzeCompressionHint(t *testing.T) {
	// Small text should not get compression hint
	small := "Hello world"
	reportSmall := Analyze(small, "claude-sonnet-4-5", 100)
	assert.Empty(t, reportSmall.CompressionHint)

	// Large text should get compression hint
	large := strings.Repeat("word ", 1500)
	reportLarge := Analyze(large, "claude-sonnet-4-5", 100)
	assert.NotEmpty(t, reportLarge.CompressionHint)
}

func TestAnalyzeUnknownModel(t *testing.T) {
	text := "some text"
	report := Analyze(text, "unknown-model-xyz", 100)
	// Should not panic, costs should be 0
	assert.Equal(t, 0.0, report.InputCost)
	assert.Equal(t, 0.0, report.OutputCost)
}
