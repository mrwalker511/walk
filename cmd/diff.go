package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/tokenizer"
)

var diffCmd = &cobra.Command{
	Use:   "diff <original> <optimized>",
	Short: "Side-by-side token comparison between two payload versions",
	Long: `Compare token counts and costs between an original and optimized payload.

Example:
  walk diff original.md optimized.md
  walk diff original.md optimized.md --model gpt-4o`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	origBytes, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading %s: %w (hint: check the file exists and is readable)", args[0], err)
	}
	optBytes, err := os.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("reading %s: %w (hint: check the file exists and is readable)", args[1], err)
	}

	origText := string(origBytes)
	optText := string(optBytes)

	m := "claude-sonnet-4-5"
	if globalCfg != nil {
		m = globalCfg.Providers.Anthropic.DefaultModel
	}
	if model != "" {
		m = model
	}

	origToks := tokenizer.Count(origText)
	optToks := tokenizer.Count(optText)
	tokenDelta := origToks - optToks

	origCost := tokenizer.Cost(origToks, m, tokenizer.Input)
	optCost := tokenizer.Cost(optToks, m, tokenizer.Input)
	costDelta := origCost - optCost

	if jsonOut {
		return printDiffJSON(m, origToks, optToks, tokenDelta, origCost, optCost, costDelta)
	}

	if !quiet {
		fmt.Println("=== walk diff ===")
		fmt.Printf("Model:          %s\n", m)
		fmt.Printf("%-16s %s\n", "File", "Tokens")
		fmt.Printf("%-16s %s\n", args[0], formatTokens(origToks))
		fmt.Printf("%-16s %s\n", args[1], formatTokens(optToks))
		fmt.Println()
	}

	if tokenDelta > 0 {
		fmt.Printf("Tokens saved:   %s (%.1f%% reduction)\n",
			formatTokens(tokenDelta),
			float64(tokenDelta)/float64(origToks)*100,
		)
		fmt.Printf("Cost saved:     %s at %s\n", tokenizer.FormatCost(costDelta), m)
		printSavings(tokenDelta, costDelta, m)
	} else if tokenDelta < 0 {
		fmt.Printf("⚠ Tokens added: %s (%.1f%% increase)\n",
			formatTokens(-tokenDelta),
			float64(-tokenDelta)/float64(origToks)*100,
		)
		fmt.Printf("Cost added:     %s at %s\n", tokenizer.FormatCost(-costDelta), m)
	} else {
		fmt.Println("No token difference")
	}

	return nil
}

type diffJSON struct {
	Model         string  `json:"model"`
	OriginalTokens int    `json:"original_tokens"`
	OptimizedTokens int   `json:"optimized_tokens"`
	TokenDelta    int     `json:"token_delta"`
	OriginalCost  float64 `json:"original_cost_usd"`
	OptimizedCost float64 `json:"optimized_cost_usd"`
	CostDelta     float64 `json:"cost_delta_usd"`
}

func printDiffJSON(m string, origToks, optToks, tokenDelta int, origCost, optCost, costDelta float64) error {
	out := diffJSON{
		Model:          m,
		OriginalTokens:  origToks,
		OptimizedTokens: optToks,
		TokenDelta:      tokenDelta,
		OriginalCost:   origCost,
		OptimizedCost:  optCost,
		CostDelta:      costDelta,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
