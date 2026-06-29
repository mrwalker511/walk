package cache

import (
	"fmt"
	"strings"

	"github.com/mrwalker511/walk/internal/tokenizer"
)

// SectionType classifies a text section by its cache-friendliness.
type SectionType string

const (
	SectionStable  SectionType = "stable"   // system prompt, instructions — cache this
	SectionDynamic SectionType = "dynamic"  // user input, variable content — don't cache
)

// Section is a labelled portion of the text.
type Section struct {
	Type    SectionType
	Content string
	Tokens  int
	LineStart int
	LineEnd   int
}

// Analysis is the result of cache analysis.
type Analysis struct {
	Sections          []Section
	StableTokens      int
	DynamicTokens     int
	EstimatedSavingsAnthropic float64
	EstimatedSavingsOpenAI    float64
	ReorderRecommended bool
	Recommendations   []string
}

// Analyze inspects text to identify stable vs. dynamic sections and
// recommends cache-friendly reordering.
func Analyze(text, model string) Analysis {
	lines := strings.Split(text, "\n")
	sections := classifySections(lines)

	stableToks, dynamicToks := 0, 0
	for i := range sections {
		sections[i].Tokens = tokenizer.Count(sections[i].Content)
		if sections[i].Type == SectionStable {
			stableToks += sections[i].Tokens
		} else {
			dynamicToks += sections[i].Tokens
		}
	}

	// Savings estimate: if stable content were cached, we pay cached rate instead of input rate
	// Anthropic: $3.00 → $0.30 per 1M (90% discount)
	// OpenAI: $2.50 → $1.25 per 1M (50% discount)
	savingsAnthropic := float64(stableToks) * (3.00 - 0.30) / 1_000_000
	savingsOpenAI := float64(stableToks) * (2.50 - 1.25) / 1_000_000

	var recommendations []string
	reorderNeeded := false

	// Check if dynamic content appears before stable content (bad ordering)
	if len(sections) >= 2 {
		firstDynIdx := -1
		lastStableIdx := -1
		for i, s := range sections {
			if s.Type == SectionDynamic && firstDynIdx < 0 {
				firstDynIdx = i
			}
			if s.Type == SectionStable {
				lastStableIdx = i
			}
		}
		if firstDynIdx >= 0 && lastStableIdx > firstDynIdx {
			reorderNeeded = true
			recommendations = append(recommendations,
				"Move stable content (system prompt, instructions) before dynamic content (user input) to maximise cache hits")
		}
	}

	if stableToks > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Cache the %d stable tokens to save ~%s per request (Anthropic) or ~%s (OpenAI)",
				stableToks,
				tokenizer.FormatCost(savingsAnthropic),
				tokenizer.FormatCost(savingsOpenAI),
			),
		)
	}

	if dynamicToks > stableToks && stableToks > 0 {
		recommendations = append(recommendations,
			"Consider extracting more instructions to a static system prompt to increase cache hit rate")
	}

	return Analysis{
		Sections:                  sections,
		StableTokens:              stableToks,
		DynamicTokens:             dynamicToks,
		EstimatedSavingsAnthropic: savingsAnthropic,
		EstimatedSavingsOpenAI:    savingsOpenAI,
		ReorderRecommended:        reorderNeeded,
		Recommendations:           recommendations,
	}
}

// classifySections applies heuristics to label lines as stable or dynamic.
func classifySections(lines []string) []Section {
	var sections []Section
	current := Section{Type: SectionStable, LineStart: 1}
	var currentLines []string

	stableKeywords := []string{
		"you are", "your role", "always", "never", "important:", "instructions:",
		"system:", "assistant:", "rules:", "guidelines:", "context:",
		"## system", "# system", "<system>", "</system>",
	}

	dynamicKeywords := []string{
		"user:", "human:", "question:", "request:", "<user>", "</user>",
		"please help", "can you", "i need", "i want",
	}

	flush := func(lineEnd int) {
		if len(currentLines) > 0 {
			current.Content = strings.Join(currentLines, "\n")
			current.LineEnd = lineEnd
			if strings.TrimSpace(current.Content) != "" {
				sections = append(sections, current)
			}
		}
	}

	for i, line := range lines {
		lower := strings.ToLower(line)

		isDynamic := false
		for _, kw := range dynamicKeywords {
			if strings.Contains(lower, kw) {
				isDynamic = true
				break
			}
		}

		isStable := false
		if !isDynamic {
			for _, kw := range stableKeywords {
				if strings.Contains(lower, kw) {
					isStable = true
					break
				}
			}
		}

		newType := current.Type
		if isDynamic {
			newType = SectionDynamic
		} else if isStable {
			newType = SectionStable
		}

		if newType != current.Type {
			flush(i)
			current = Section{Type: newType, LineStart: i + 1}
			currentLines = nil
		}
		currentLines = append(currentLines, line)
	}
	flush(len(lines))

	return sections
}
