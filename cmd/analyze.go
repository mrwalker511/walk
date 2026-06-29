package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/analyzer"
	"github.com/mrwalker511/walk/internal/tokenizer"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [file]",
	Short: "Inspect a prompt for token count, cost, and issues",
	Long: `Analyze a prompt or payload file (or stdin) before sending to an LLM.

Reports token count, estimated cost, dead-weight detection, repetition
fingerprinting, and secret/PII scanning.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	text, err := readInput(args)
	if err != nil {
		return err
	}

	m := "claude-sonnet-4-5"
	threshold := 4.5
	if globalCfg != nil {
		m = globalCfg.Providers.Anthropic.DefaultModel
		threshold = globalCfg.Scrubber.EntropyThreshold
	}
	if model != "" {
		m = model
	}

	report := analyzer.Analyze(text, m, threshold)

	if jsonOut {
		return printAnalyzeJSON(report)
	}
	return printAnalyzeTable(report)
}

func readInput(args []string) (string, error) {
	if len(args) == 0 {
		// Read from stdin
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w (hint: pipe content or provide a file path)", err)
		}
		return string(b), nil
	}
	b, err := os.ReadFile(args[0])
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w (hint: check the path exists and is readable)", args[0], err)
	}
	return string(b), nil
}

func printAnalyzeTable(r analyzer.Report) error {
	if !quiet {
		fmt.Println("=== walk analyze ===")
		fmt.Printf("Model:          %s\n", r.Model)
		fmt.Printf("Tokens:         %s (est. output: %s)\n", formatTokens(r.TokenCount), formatTokens(r.EstimatedOutput))
		fmt.Printf("Words:          %d\n", r.WordCount)
		fmt.Printf("Lines:          %d\n", r.LineCount)
		fmt.Printf("Input cost:     %s\n", tokenizer.FormatCost(r.InputCost))
		fmt.Printf("Output cost:    %s (est.)\n", tokenizer.FormatCost(r.OutputCost))
		fmt.Printf("Total cost:     %s\n", tokenizer.FormatCost(r.TotalCost))
	}

	if len(r.Warnings) > 0 {
		if !quiet {
			fmt.Printf("\nWarnings (%d):\n", len(r.Warnings))
		}
		for _, w := range r.Warnings {
			icon := "⚠"
			switch w.Severity {
			case analyzer.SeverityError:
				icon = "✗"
			case analyzer.SeverityInfo:
				icon = "ℹ"
			}
			fmt.Printf("  %s [%s] %s\n", icon, w.Code, w.Message)
			if w.Hint != "" && !quiet {
				fmt.Printf("    → %s\n", w.Hint)
			}
		}
	}

	if r.CompressionHint != "" && !quiet {
		fmt.Printf("\n%s\n", r.CompressionHint)
	}

	if r.HasSecrets {
		fmt.Fprintln(os.Stderr, "\nSecrets detected — run 'walk scrub' to redact before sending")
		os.Exit(1)
	}

	return nil
}

type analyzeJSON struct {
	Model           string              `json:"model"`
	TokenCount      int                 `json:"token_count"`
	WordCount       int                 `json:"word_count"`
	LineCount       int                 `json:"line_count"`
	EstimatedOutput int                 `json:"estimated_output_tokens"`
	InputCost       float64             `json:"input_cost_usd"`
	OutputCost      float64             `json:"output_cost_usd"`
	TotalCost       float64             `json:"total_cost_usd"`
	Warnings        []analyzer.Warning  `json:"warnings"`
	HasSecrets      bool                `json:"has_secrets"`
}

func printAnalyzeJSON(r analyzer.Report) error {
	out := analyzeJSON{
		Model:           r.Model,
		TokenCount:      r.TokenCount,
		WordCount:       r.WordCount,
		LineCount:       r.LineCount,
		EstimatedOutput: r.EstimatedOutput,
		InputCost:       r.InputCost,
		OutputCost:      r.OutputCost,
		TotalCost:       r.TotalCost,
		Warnings:        r.Warnings,
		HasSecrets:      r.HasSecrets,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
