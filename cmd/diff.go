package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mrwalker511/walk/internal/tokenizer"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
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

	unified, removed, added, err := renderDiffHighlight(args[0], args[1], origText, optText)
	if err != nil {
		return fmt.Errorf("computing diff: %w", err)
	}

	if jsonOut {
		return printDiffJSON(m, origToks, optToks, tokenDelta, origCost, optCost, costDelta, removed, added)
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

	if !quiet && unified != "" {
		fmt.Println()
		fmt.Println("=== Diff ===")
		printUnifiedDiff(unified)
	}

	return nil
}

// renderDiffHighlight computes a unified diff between orig and opt, returning
// the rendered diff text plus the removed and added lines (without their
// leading -/+ markers) for JSON output. Returns empty/nil when the texts are
// identical.
func renderDiffHighlight(fromFile, toFile, orig, opt string) (unified string, removed, added []string, err error) {
	if orig == opt {
		return "", nil, nil, nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(orig),
		B:        difflib.SplitLines(opt),
		FromFile: fromFile,
		ToFile:   toFile,
		Context:  3,
	}
	unified, err = difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return "", nil, nil, err
	}

	for _, line := range strings.Split(unified, "\n") {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "@@"):
			continue
		case strings.HasPrefix(line, "-"):
			removed = append(removed, line[1:])
		case strings.HasPrefix(line, "+"):
			added = append(added, line[1:])
		}
	}

	return unified, removed, added, nil
}

// printUnifiedDiff prints a unified diff string, colorizing removed/added
// lines when output.color is enabled in config.
func printUnifiedDiff(unified string) {
	color := globalCfg == nil || globalCfg.Output.Color
	for _, line := range strings.Split(strings.TrimRight(unified, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "@@"):
			fmt.Println(line)
		case strings.HasPrefix(line, "-"):
			if color {
				fmt.Printf("\x1b[31m%s\x1b[0m\n", line)
			} else {
				fmt.Println(line)
			}
		case strings.HasPrefix(line, "+"):
			if color {
				fmt.Printf("\x1b[32m%s\x1b[0m\n", line)
			} else {
				fmt.Println(line)
			}
		default:
			fmt.Println(line)
		}
	}
}

type diffJSON struct {
	Model           string   `json:"model"`
	OriginalTokens  int      `json:"original_tokens"`
	OptimizedTokens int      `json:"optimized_tokens"`
	TokenDelta      int      `json:"token_delta"`
	OriginalCost    float64  `json:"original_cost_usd"`
	OptimizedCost   float64  `json:"optimized_cost_usd"`
	CostDelta       float64  `json:"cost_delta_usd"`
	RemovedLines    []string `json:"removed_lines"`
	AddedLines      []string `json:"added_lines"`
}

func printDiffJSON(m string, origToks, optToks, tokenDelta int, origCost, optCost, costDelta float64, removed, added []string) error {
	out := diffJSON{
		Model:           m,
		OriginalTokens:  origToks,
		OptimizedTokens: optToks,
		TokenDelta:      tokenDelta,
		OriginalCost:    origCost,
		OptimizedCost:   optCost,
		CostDelta:       costDelta,
		RemovedLines:    removed,
		AddedLines:      added,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
