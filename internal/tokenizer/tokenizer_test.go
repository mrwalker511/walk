package tokenizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		minToks int
		maxToks int
	}{
		{"empty", "", 0, 0},
		{"single word", "hello", 1, 2},
		{"short sentence", "The quick brown fox", 4, 6},
		{"paragraph", "This is a sample paragraph with about twenty words in it to test token counting accuracy here.", 18, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Count(tt.text)
			assert.GreaterOrEqual(t, got, tt.minToks)
			assert.LessOrEqual(t, got, tt.maxToks)
		})
	}
}

func TestCountWords(t *testing.T) {
	assert.Equal(t, 0, CountWords(""))
	assert.Equal(t, 1, CountWords("hello"))
	assert.Equal(t, 4, CountWords("one two three four"))
	assert.Equal(t, 2, CountWords("  spaces  around  "))
}

func TestCountLines(t *testing.T) {
	assert.Equal(t, 0, CountLines(""))
	assert.Equal(t, 1, CountLines("single line"))
	assert.Equal(t, 3, CountLines("line1\nline2\nline3"))
	assert.Equal(t, 2, CountLines("line1\nline2"))
}

func TestCost(t *testing.T) {
	tests := []struct {
		name     string
		tokens   int
		model    string
		dir      Direction
		expected float64
	}{
		{"claude input 1M", 1_000_000, "claude-sonnet-4-5", Input, 3.00},
		{"claude output 1M", 1_000_000, "claude-sonnet-4-5", Output, 15.00},
		{"claude cached 1M", 1_000_000, "claude-sonnet-4-5", Cached, 0.30},
		{"gpt4o input 1M", 1_000_000, "gpt-4o", Input, 2.50},
		{"gpt4o-mini input 1M", 1_000_000, "gpt-4o-mini", Input, 0.15},
		{"llama free", 1_000_000, "llama.cpp", Input, 0.00},
		{"unknown model", 1_000_000, "unknown-model", Input, 0.00},
		{"zero tokens", 0, "claude-sonnet-4-5", Input, 0.00},
		{"1000 tokens claude input", 1000, "claude-sonnet-4-5", Input, 0.003},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Cost(tt.tokens, tt.model, tt.dir)
			assert.InDelta(t, tt.expected, got, 0.0001)
		})
	}
}

func TestEstimatedOutputTokens(t *testing.T) {
	assert.Equal(t, 25, EstimatedOutputTokens(100))
	assert.Equal(t, 250, EstimatedOutputTokens(1000))
	assert.Equal(t, 0, EstimatedOutputTokens(0))
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		usd      float64
		contains string
	}{
		{0, "$0.00"},
		{0.00001, "<$0.001"},
		{0.0035, "$0.0035"},
		{0.043, "$0.043"},
		{1.50, "$1.50"},
	}
	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			assert.Equal(t, tt.contains, FormatCost(tt.usd))
		})
	}
}

func TestIsKnownModel(t *testing.T) {
	assert.True(t, IsKnownModel("claude-sonnet-4-5"))
	assert.True(t, IsKnownModel("gpt-4o"))
	assert.True(t, IsKnownModel("llama.cpp"))
	assert.False(t, IsKnownModel("gpt-5"))
	assert.False(t, IsKnownModel(""))
}

func TestIsCodeHeavy(t *testing.T) {
	assert.False(t, IsCodeHeavy(""))
	assert.False(t, IsCodeHeavy("This is just plain English text with normal words"))
	assert.True(t, IsCodeHeavy(`func main() { fmt.Println("hello") }; x := map[string]int{}`))
}

func TestKnownModels(t *testing.T) {
	models := KnownModels()
	assert.Len(t, models, 6)
}

func BenchmarkCount(b *testing.B) {
	text := "The quick brown fox jumps over the lazy dog. " +
		"Pack my box with five dozen liquor jugs. " +
		"How vexingly quick daft zebras jump! "
	text = text + text + text + text + text // ~225 words
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Count(text)
	}
}
