package analyzer

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/mrwalker511/walk/internal/scrubber"
	"github.com/mrwalker511/walk/internal/tokenizer"
)

// WarningSeverity indicates how serious a warning is.
type WarningSeverity string

const (
	SeverityInfo    WarningSeverity = "info"
	SeverityWarning WarningSeverity = "warning"
	SeverityError   WarningSeverity = "error"
)

// Warning is a single analysis finding.
type Warning struct {
	Severity WarningSeverity
	Code     string
	Message  string
	Hint     string
}

// Report is the full output of an analysis run.
type Report struct {
	TokenCount      int
	WordCount       int
	LineCount       int
	EstimatedOutput int
	Model           string
	InputCost       float64
	OutputCost      float64
	TotalCost       float64
	Warnings        []Warning
	Secrets         []scrubber.Finding
	HasSecrets      bool
	CompressionHint string
}

// Analyze inspects text and returns a detailed report.
func Analyze(text, model string, entropyThreshold float64) Report {
	toks := tokenizer.Count(text)
	words := tokenizer.CountWords(text)
	lines := tokenizer.CountLines(text)
	estOut := tokenizer.EstimatedOutputTokens(toks)

	inputCost := tokenizer.Cost(toks, model, tokenizer.Input)
	outputCost := tokenizer.Cost(estOut, model, tokenizer.Output)

	var warnings []Warning

	// Dead weight: check for repeated phrases
	warnings = append(warnings, detectRepetition(text)...)

	// Bloat: very long lines (likely pasted raw data)
	warnings = append(warnings, detectLongLines(text)...)

	// Context fill warning
	if toks > 60_000 {
		warnings = append(warnings, Warning{
			Severity: SeverityWarning,
			Code:     "CONTEXT_FILL",
			Message:  fmt.Sprintf("Context is %d tokens — above 60%% fill threshold", toks),
			Hint:     "Run 'walk compress' to reduce before sending",
		})
	}

	// Secret scan
	scrubResult := scrubber.Scrub(text, entropyThreshold)

	for _, f := range scrubResult.Findings {
		warnings = append(warnings, Warning{
			Severity: SeverityError,
			Code:     "SECRET_" + strings.ToUpper(string(f.Type)),
			Message:  fmt.Sprintf("Potential %s detected at line %d: %s", f.Type, f.Line, f.Match),
			Hint:     "Run 'walk scrub' to redact before sending",
		})
	}

	hint := ""
	if toks > 1000 {
		hint = fmt.Sprintf("Run 'walk compress' to reduce ~%d tokens before sending", toks/4)
	}

	return Report{
		TokenCount:      toks,
		WordCount:       words,
		LineCount:       lines,
		EstimatedOutput: estOut,
		Model:           model,
		InputCost:       inputCost,
		OutputCost:      outputCost,
		TotalCost:       inputCost + outputCost,
		Warnings:        warnings,
		Secrets:         scrubResult.Findings,
		HasSecrets:      scrubResult.HasSecrets,
		CompressionHint: hint,
	}
}

// detectRepetition finds duplicate n-gram chunks using FNV hashing.
func detectRepetition(text string) []Warning {
	lines := strings.Split(text, "\n")
	seen := make(map[uint64]int) // hash -> first line number
	var warnings []Warning

	const windowSize = 5 // lines per chunk
	for i := 0; i <= len(lines)-windowSize; i++ {
		chunk := strings.Join(lines[i:i+windowSize], "\n")
		chunk = strings.TrimSpace(chunk)
		if len(strings.Fields(chunk)) < 10 {
			continue // skip trivially short chunks
		}
		h := hash(chunk)
		if first, ok := seen[h]; ok {
			warnings = append(warnings, Warning{
				Severity: SeverityWarning,
				Code:     "DUPLICATE_BLOCK",
				Message:  fmt.Sprintf("Duplicate content block detected (first at line %d, repeated near line %d)", first+1, i+1),
				Hint:     "Remove repeated instructions to save tokens",
			})
		} else {
			seen[h] = i
		}
	}
	return warnings
}

// detectLongLines warns about very long lines (>500 chars) which often indicate pasted raw data.
func detectLongLines(text string) []Warning {
	var warnings []Warning
	for i, line := range strings.Split(text, "\n") {
		if len(line) > 500 {
			warnings = append(warnings, Warning{
				Severity: SeverityInfo,
				Code:     "LONG_LINE",
				Message:  fmt.Sprintf("Line %d is %d chars — may be raw pasted data", i+1, len(line)),
				Hint:     "Consider summarising or removing raw data before sending",
			})
		}
	}
	return warnings
}

func hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(s))))
	return h.Sum64()
}
