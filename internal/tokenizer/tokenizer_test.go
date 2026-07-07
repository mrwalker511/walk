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
		{"sonnet-5 input 1M", 1_000_000, "claude-sonnet-5", Input, 2.00},
		{"sonnet-5 output 1M", 1_000_000, "claude-sonnet-5", Output, 10.00},
		{"sonnet-5 cached 1M", 1_000_000, "claude-sonnet-5", Cached, 0.20},
		{"haiku-4-5 input 1M", 1_000_000, "claude-haiku-4-5", Input, 1.00},
		{"haiku-4-5 output 1M", 1_000_000, "claude-haiku-4-5", Output, 5.00},
		{"haiku-4-5 cached 1M", 1_000_000, "claude-haiku-4-5", Cached, 0.10},
		{"opus-4-8 input 1M", 1_000_000, "claude-opus-4-8", Input, 5.00},
		{"opus-4-8 output 1M", 1_000_000, "claude-opus-4-8", Output, 25.00},
		{"opus-4-8 cached 1M", 1_000_000, "claude-opus-4-8", Cached, 0.50},
		{"fable-5 input 1M", 1_000_000, "claude-fable-5", Input, 10.00},
		{"fable-5 output 1M", 1_000_000, "claude-fable-5", Output, 50.00},
		{"fable-5 cached 1M", 1_000_000, "claude-fable-5", Cached, 1.00},
		{"gpt4o input 1M", 1_000_000, "gpt-4o", Input, 2.50},
		{"gpt4o output 1M", 1_000_000, "gpt-4o", Output, 10.00},
		{"gpt4o cached 1M", 1_000_000, "gpt-4o", Cached, 1.25},
		{"gpt4o-mini input 1M", 1_000_000, "gpt-4o-mini", Input, 0.15},
		{"gpt4o-mini output 1M", 1_000_000, "gpt-4o-mini", Output, 0.60},
		{"gpt4o-mini cached 1M", 1_000_000, "gpt-4o-mini", Cached, 0.075},
		{"gemini input 1M", 1_000_000, "gemini-2.5-flash", Input, 0.30},
		{"gemini output 1M", 1_000_000, "gemini-2.5-flash", Output, 2.50},
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

func TestCountTokens(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		model   string
		minToks int
	}{
		{"empty", "", "claude-sonnet-4-5", 0},
		{"single word", "hello", "gpt-4o", 1},
		{"sentence", "The quick brown fox jumps", "gpt-4o-mini", 5},
		{"model ignored", "hello world", "llama.cpp", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountTokens(tt.text, tt.model)
			assert.GreaterOrEqual(t, got, tt.minToks)
		})
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name      string
		tokens    int
		model     string
		direction string
		expected  float64
	}{
		{"claude input", 1_000_000, "claude-sonnet-4-5", "input", 3.00},
		{"claude output", 1_000_000, "claude-sonnet-4-5", "output", 15.00},
		{"claude cached", 1_000_000, "claude-sonnet-4-5", "cached", 0.30},
		{"gpt4o input", 1_000_000, "gpt-4o", "input", 2.50},
		{"gpt4o output", 1_000_000, "gpt-4o", "output", 10.00},
		{"gpt4o cached", 1_000_000, "gpt-4o", "cached", 1.25},
		{"gpt4o-mini input", 1_000_000, "gpt-4o-mini", "input", 0.15},
		{"gpt4o-mini output", 1_000_000, "gpt-4o-mini", "output", 0.60},
		{"gemini input", 1_000_000, "gemini-2.5-flash", "input", 0.30},
		{"gemini output", 1_000_000, "gemini-2.5-flash", "output", 2.50},
		{"llama free", 1_000_000, "llama.cpp", "input", 0.00},
		{"unknown model", 1_000_000, "unknown-x", "input", 0.00},
		{"unknown direction defaults to input", 1_000_000, "claude-sonnet-4-5", "inbound", 3.00},
		{"case insensitive Output", 1_000_000, "claude-sonnet-4-5", "Output", 15.00},
		{"zero tokens", 0, "claude-sonnet-4-5", "input", 0.00},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCost(tt.tokens, tt.model, tt.direction)
			assert.InDelta(t, tt.expected, got, 0.0001)
		})
	}
}

func TestIsCodeHeavy(t *testing.T) {
	assert.False(t, IsCodeHeavy(""))
	assert.False(t, IsCodeHeavy("This is just plain English text with normal words"))
	assert.True(t, IsCodeHeavy(`func main() { fmt.Println("hello") }; x := map[string]int{}`))
}

func TestKnownModels(t *testing.T) {
	models := KnownModels()
	assert.GreaterOrEqual(t, len(models), 6)
	assert.True(t, IsKnownModel("claude-sonnet-5"))
	assert.True(t, IsKnownModel("claude-haiku-4-5"))
	assert.True(t, IsKnownModel("claude-sonnet-4-5"))
	assert.True(t, IsKnownModel("llama.cpp"))
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
