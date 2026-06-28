package tokenizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountTokens_Empty(t *testing.T) {
	assert.Equal(t, 1, CountTokens(""))
}

func TestCountTokens_Short(t *testing.T) {
	assert.Greater(t, CountTokens("hello world"), 0)
}

func TestCountTokens_Long(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog. " +
		"Pack my box with five dozen liquor jugs. " +
		"How vexingly quick daft zebras jump!"
	count := CountTokens(text)
	assert.Greater(t, count, 10)
	assert.Less(t, count, 50)
}

func TestEstimateCost_KnownModel(t *testing.T) {
	cost, err := EstimateCost("gpt-4o", 1000, 500)
	require.NoError(t, err)
	assert.InDelta(t, 0.0025+0.005, cost, 0.001)
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	_, err := EstimateCost("unknown-model", 100, 50)
	assert.Error(t, err)
}

func TestEstimateCost_FreeModel(t *testing.T) {
	cost, err := EstimateCost("llama-3-8b", 10000, 5000)
	require.NoError(t, err)
	assert.Equal(t, 0.0, cost)
}

func TestAnalyze_Basic(t *testing.T) {
	result, err := Analyze("gpt-4o",
		"You are a helpful assistant",
		[]string{"User: hi\nAssistant: hello"},
		"What is Go?",
		"Go is a programming language",
	)
	require.NoError(t, err)
	assert.Greater(t, result.InputTokens, 0)
	assert.Greater(t, result.OutputTokens, 0)
	assert.GreaterOrEqual(t, result.SystemTokens, 0)
	assert.GreaterOrEqual(t, result.HistoryTokens, 0)
	assert.GreaterOrEqual(t, result.CurrentTokens, 0)
	assert.Greater(t, result.EstimatedCost, 0.0)
	assert.GreaterOrEqual(t, result.WastedTokens, 0)
}

func TestKnownModels_NotEmpty(t *testing.T) {
	assert.Greater(t, len(KnownModels), 5)
}

func TestKnownModels_ClaudeSonnet(t *testing.T) {
	info, ok := KnownModels["claude-sonnet-4-20250514"]
	require.True(t, ok)
	assert.Equal(t, "anthropic", info.Provider)
	assert.Equal(t, 200000, info.ContextLimit)
}