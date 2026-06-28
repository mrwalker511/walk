// Package tokenizer provides token counting and cost estimation
// for various LLM providers and models.
package tokenizer

import (
	"fmt"
	"strings"
)

// ModelTokenInfo contains token limits and pricing for a model.
type ModelTokenInfo struct {
	Model        string
	Provider     string
	ContextLimit int     // max tokens the model accepts
	InputCost    float64 // cost per 1K input tokens (USD)
	OutputCost   float64 // cost per 1K output tokens (USD)
}

// KnownModels maps model identifiers to their token/cost info.
// Updated as pricing changes.
var KnownModels = map[string]ModelTokenInfo{
	// Anthropic Claude
	"claude-sonnet-4-20250514":  {Model: "claude-sonnet-4-20250514", Provider: "anthropic", ContextLimit: 200000, InputCost: 0.003, OutputCost: 0.015},
	"claude-3-opus":             {Model: "claude-3-opus", Provider: "anthropic", ContextLimit: 200000, InputCost: 0.015, OutputCost: 0.075},
	"claude-3-haiku":            {Model: "claude-3-haiku", Provider: "anthropic", ContextLimit: 200000, InputCost: 0.00025, OutputCost: 0.00125},

	// OpenAI Codex / GPT
	"gpt-4o":                    {Model: "gpt-4o", Provider: "openai", ContextLimit: 128000, InputCost: 0.0025, OutputCost: 0.01},
	"gpt-4o-mini":               {Model: "gpt-4o-mini", Provider: "openai", ContextLimit: 128000, InputCost: 0.00015, OutputCost: 0.0006},
	"o1-preview":                {Model: "o1-preview", Provider: "openai", ContextLimit: 128000, InputCost: 0.015, OutputCost: 0.06},

	// Local / llama.cpp (free inference)
	"llama-3-8b":                {Model: "llama-3-8b", Provider: "local", ContextLimit: 8192, InputCost: 0, OutputCost: 0},
	"llama-3-70b":               {Model: "llama-3-70b", Provider: "local", ContextLimit: 8192, InputCost: 0, OutputCost: 0},
	"mistral-7b":                {Model: "mistral-7b", Provider: "local", ContextLimit: 32768, InputCost: 0, OutputCost: 0},
}

// CountResult holds the result of token counting.
type CountResult struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// EstimateCost calculates the cost of a token count for a given model.
func EstimateCost(modelID string, inputTokens, outputTokens int) (float64, error) {
	info, ok := KnownModels[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model: %s", modelID)
	}
	inputCost := (float64(inputTokens) / 1000) * info.InputCost
	outputCost := (float64(outputTokens) / 1000) * info.OutputCost
	return inputCost + outputCost, nil
}

// CountTokens estimates tokens from text.
// Uses a simple estimate: ~4 characters per token on average.
// For production, use provider-specific tokenizers (tiktoken, etc.).
func CountTokens(text string) int {
	// Simple but reasonable heuristic
	tokens := len(strings.Fields(text))
	tokens += len(text) / 4 // character-based adjustment
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

// AnalyzePayload analyzes a prompt+response pair for token usage.
type AnalyzeResult struct {
	InputTokens     int
	OutputTokens    int
	SystemTokens    int // tokens in system prompt
	HistoryTokens   int // tokens in conversation history
	CurrentTokens   int // tokens in current user message
	EstimatedCost   float64
	WastedTokens    int // tokens that could be trimmed
	CompressionPct  float64 // estimated compression ratio
}

// Analyze performs detailed analysis of a payload.
func Analyze(modelID string, systemPrompt string, history []string, userMessage string, response string) (*AnalyzeResult, error) {
	sysTokens := CountTokens(systemPrompt)
	histTokens := 0
	for _, h := range history {
		histTokens += CountTokens(h)
	}
	curTokens := CountTokens(userMessage)
	outTokens := CountTokens(response)
	totalIn := sysTokens + histTokens + curTokens

	cost, err := EstimateCost(modelID, totalIn, outTokens)
	if err != nil {
		return nil, err
	}

	// Estimate waste (20–40% typical bloat from redundant system prompts, repeated context)
	wasteRatio := 0.25
	if histTokens > 5000 {
		wasteRatio = 0.35
	}
	wasted := int(float64(totalIn) * wasteRatio)

	compressionPct := wasteRatio * 100

	return &AnalyzeResult{
		InputTokens:     totalIn,
		OutputTokens:    outTokens,
		SystemTokens:    sysTokens,
		HistoryTokens:   histTokens,
		CurrentTokens:   curTokens,
		EstimatedCost:   cost,
		WastedTokens:    wasted,
		CompressionPct:  compressionPct,
	}, nil
}