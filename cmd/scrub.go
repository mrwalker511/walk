package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/scrubber"
)

var (
	scrubOutput string
)

var scrubCmd = &cobra.Command{
	Use:   "scrub [file]",
	Short: "Scan payload for secrets and PII before it leaves your machine",
	Long: `Scrub scans a file or stdin for API keys, JWTs, AWS credentials, SSH keys,
emails, SSNs, and phone numbers. Outputs a clean payload and redaction report.

Exits with code 1 if secrets are found (CI/CD friendly).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScrub,
}

func init() {
	scrubCmd.Flags().StringVarP(&scrubOutput, "output", "o", "", "write cleaned output to file (default: stdout)")
	rootCmd.AddCommand(scrubCmd)
}

func runScrub(cmd *cobra.Command, args []string) error {
	text, err := readInput(args)
	if err != nil {
		return err
	}

	threshold := scrubber.EntropyThreshold
	if globalCfg != nil {
		threshold = globalCfg.Scrubber.EntropyThreshold
	}

	result := scrubber.Scrub(text, threshold)

	if jsonOut {
		return printScrubJSON(result)
	}
	return printScrubTable(result, text)
}

func printScrubTable(result scrubber.Result, original string) error {
	if !quiet {
		if result.HasSecrets {
			fmt.Fprintf(os.Stderr, "✗ %d secret(s)/PII finding(s) detected:\n", len(result.Findings))
			for _, f := range result.Findings {
				fmt.Fprintf(os.Stderr, "  [%s] line %d: %s → %s\n", f.Type, f.Line, f.Match, f.Redacted)
			}
			fmt.Fprintln(os.Stderr, "")
		} else {
			fmt.Fprintln(os.Stderr, "✓ No secrets or PII detected")
		}
	}

	// Write clean output
	if dryRun {
		if !quiet {
			fmt.Fprintln(os.Stderr, "[dry-run] Would write redacted output (not writing)")
		}
	} else if scrubOutput != "" {
		if err := os.WriteFile(scrubOutput, []byte(result.Clean), 0600); err != nil {
			return fmt.Errorf("writing output file %s: %w", scrubOutput, err)
		}
	} else {
		fmt.Print(result.Clean)
	}

	if result.HasSecrets {
		os.Exit(1)
	}
	return nil
}

type scrubJSON struct {
	HasSecrets bool               `json:"has_secrets"`
	Findings   []scrubber.Finding `json:"findings"`
	Clean      string             `json:"clean"`
}

func printScrubJSON(result scrubber.Result) error {
	out := scrubJSON{
		HasSecrets: result.HasSecrets,
		Findings:   result.Findings,
		Clean:      result.Clean,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	if result.HasSecrets {
		os.Exit(1)
	}
	return nil
}
