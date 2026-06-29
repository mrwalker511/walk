package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/cache"
	"github.com/mrwalker511/walk/internal/tokenizer"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Prefix cache analysis and optimization",
}

var cacheAnalyzeCmd = &cobra.Command{
	Use:   "analyze [file]",
	Short: "Analyze a prompt for prefix cache optimization",
	Long: `Identify stable vs. dynamic sections and recommend cache-friendly reordering.

Example:
  walk cache analyze prompt.md
  cat prompt.md | walk cache analyze`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCacheAnalyze,
}

func init() {
	cacheCmd.AddCommand(cacheAnalyzeCmd)
	rootCmd.AddCommand(cacheCmd)
}

func runCacheAnalyze(cmd *cobra.Command, args []string) error {
	text, err := readInput(args)
	if err != nil {
		return err
	}

	m := "claude-sonnet-4-5"
	if globalCfg != nil {
		m = globalCfg.Providers.Anthropic.DefaultModel
	}
	if model != "" {
		m = model
	}

	analysis := cache.Analyze(text, m)

	if jsonOut {
		return printCacheJSON(analysis)
	}
	return printCacheTable(analysis, m)
}

func printCacheTable(a cache.Analysis, m string) error {
	if !quiet {
		fmt.Println("=== walk cache analyze ===")
		fmt.Printf("Stable tokens:   %s\n", formatTokens(a.StableTokens))
		fmt.Printf("Dynamic tokens:  %s\n", formatTokens(a.DynamicTokens))
		fmt.Printf("Est. savings (Anthropic): %s/request\n", tokenizer.FormatCost(a.EstimatedSavingsAnthropic))
		fmt.Printf("Est. savings (OpenAI):    %s/request\n", tokenizer.FormatCost(a.EstimatedSavingsOpenAI))
		fmt.Println()

		if a.ReorderRecommended {
			fmt.Println("⚠ Reorder recommended: stable content should come before dynamic content")
		} else {
			fmt.Println("✓ Content ordering is cache-friendly")
		}

		if len(a.Recommendations) > 0 {
			fmt.Printf("\nRecommendations (%d):\n", len(a.Recommendations))
			for _, rec := range a.Recommendations {
				fmt.Printf("  → %s\n", rec)
			}
		}

		fmt.Printf("\nSections (%d):\n", len(a.Sections))
		for i, s := range a.Sections {
			icon := "S"
			if s.Type == cache.SectionDynamic {
				icon = "D"
			}
			preview := s.Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			fmt.Printf("  [%d] [%s] %d tokens — %q\n", i+1, icon, s.Tokens, preview)
		}
	}
	return nil
}

type cacheJSON struct {
	StableTokens              int                `json:"stable_tokens"`
	DynamicTokens             int                `json:"dynamic_tokens"`
	EstimatedSavingsAnthropic float64            `json:"estimated_savings_anthropic_usd"`
	EstimatedSavingsOpenAI    float64            `json:"estimated_savings_openai_usd"`
	ReorderRecommended        bool               `json:"reorder_recommended"`
	Recommendations           []string           `json:"recommendations"`
	Sections                  []cache.Section    `json:"sections"`
}

func printCacheJSON(a cache.Analysis) error {
	out := cacheJSON{
		StableTokens:              a.StableTokens,
		DynamicTokens:             a.DynamicTokens,
		EstimatedSavingsAnthropic: a.EstimatedSavingsAnthropic,
		EstimatedSavingsOpenAI:    a.EstimatedSavingsOpenAI,
		ReorderRecommended:        a.ReorderRecommended,
		Recommendations:           a.Recommendations,
		Sections:                  a.Sections,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
