package tokenizer

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Direction is the token flow direction for pricing.
type Direction int

const (
	Input  Direction = iota
	Output Direction = iota
	Cached Direction = iota
)

// ModelPricing holds per-1M-token rates in USD.
type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
	CachedPer1M float64
}

// PricingTable is the canonical model pricing from the spec.
var PricingTable = map[string]ModelPricing{
	"claude-sonnet-4-5": {InputPer1M: 3.00, OutputPer1M: 15.00, CachedPer1M: 0.30},
	"claude-haiku-3-5":  {InputPer1M: 0.80, OutputPer1M: 4.00, CachedPer1M: 0.08},
	"gpt-4o":            {InputPer1M: 2.50, OutputPer1M: 10.00, CachedPer1M: 1.25},
	"gpt-4o-mini":       {InputPer1M: 0.15, OutputPer1M: 0.60, CachedPer1M: 0.075},
	"gemini-2.5-flash":  {InputPer1M: 0.075, OutputPer1M: 0.30, CachedPer1M: 0},
	"llama.cpp":         {InputPer1M: 0, OutputPer1M: 0, CachedPer1M: 0},
}

// Count estimates the token count for text using a character-based approximation.
// Uses ~4 characters per token, consistent with BPE tokenizers for English prose.
func Count(text string) int {
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	charCount := 0
	for _, w := range words {
		charCount += len([]rune(w))
	}
	// ~4 chars per token; minimum 1 token per word
	byChars := int(math.Ceil(float64(charCount) / 4.0))
	if byChars < len(words) {
		byChars = len(words)
	}
	return byChars
}

// CountWords returns the raw word count.
func CountWords(text string) int {
	return len(strings.Fields(text))
}

// CountLines returns the line count.
func CountLines(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

// Cost returns the USD cost for a given token count, model, and direction.
// Returns 0 for unknown models (fail-open, don't block pipeline).
func Cost(tokens int, model string, dir Direction) float64 {
	p, ok := PricingTable[model]
	if !ok {
		return 0
	}
	rate := p.InputPer1M
	switch dir {
	case Output:
		rate = p.OutputPer1M
	case Cached:
		rate = p.CachedPer1M
	}
	return float64(tokens) * rate / 1_000_000
}

// EstimatedOutputTokens projects output token count as 25% of input (conservative default).
func EstimatedOutputTokens(inputTokens int) int {
	return int(math.Ceil(float64(inputTokens) * 0.25))
}

// FormatCost formats a USD cost value as a human-readable string.
func FormatCost(usd float64) string {
	if usd == 0 {
		return "$0.00"
	}
	if usd < 0.001 {
		return "<$0.001"
	}
	if usd < 0.01 {
		return fmt.Sprintf("$%.4f", usd)
	}
	if usd < 1.0 {
		return fmt.Sprintf("$%.3f", usd)
	}
	return fmt.Sprintf("$%.2f", usd)
}

// KnownModels returns the list of supported model names.
func KnownModels() []string {
	models := make([]string, 0, len(PricingTable))
	for m := range PricingTable {
		models = append(models, m)
	}
	return models
}

// IsKnownModel reports whether the model name has a known pricing entry.
func IsKnownModel(model string) bool {
	_, ok := PricingTable[model]
	return ok
}

// IsCodeHeavy returns true if > 20% of non-space chars are symbols,
// suggesting code content where tokenization is denser.
func IsCodeHeavy(text string) bool {
	runes := []rune(text)
	if len(runes) == 0 {
		return false
	}
	symCount, total := 0, 0
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			total++
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				symCount++
			}
		}
	}
	if total == 0 {
		return false
	}
	return float64(symCount)/float64(total) > 0.20
}
