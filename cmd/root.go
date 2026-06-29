package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/config"
)

var (
	cfgDir   string
	jsonOut  bool
	quiet    bool
	dryRun   bool
	model    string
	globalCfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "walk",
	Short: "Optimize LLM token usage across agentic workflows",
	Long: `walk is a CLI tool that analyzes, compresses, and monitors LLM payloads
to reduce token usage and cost across agentic coding workflows.

Slow down. Use less. Save more.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config load for "init" — it creates the config
		if cmd.Name() == "init" {
			return nil
		}
		cfg, err := config.LoadFrom(cfgDir)
		if err != nil {
			// Non-fatal: continue with defaults
			cfg, _ = config.LoadFrom(os.TempDir())
		}
		globalCfg = cfg
		if model != "" {
			globalCfg.Providers.Anthropic.DefaultModel = model
		}
		return nil
	},
}

// Execute is the entry point called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if hint := errorHint(err); hint != "" {
			fmt.Fprintln(os.Stderr, "Hint:", hint)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgDir, "config-dir", "", "config directory (default: ~/.walk)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress decorative output (for CI)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would happen without making changes")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "override default model (e.g. claude-sonnet-4-5)")
}

func errorHint(err error) string {
	// Provide contextual hints for common errors
	msg := err.Error()
	switch {
	case contains(msg, "config"):
		return "run 'walk init' to create a configuration file"
	case contains(msg, "llama"):
		return "start llama.cpp with: llama-server --model /path/to/model.gguf --port 8080"
	case contains(msg, "permission denied"):
		return "check file permissions or run with appropriate privileges"
	}
	return ""
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func printSavings(tokensSaved int, costSaved float64, modelName string) {
	if quiet || tokensSaved <= 0 {
		return
	}
	if modelName == "" {
		modelName = "unknown"
	}
	fmt.Printf("Saved %s tokens (~$%.4f at %s)\n",
		formatTokens(tokensSaved), costSaved, modelName)
}

func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1_000_000, (n/1000)%1000, n%1000)
}
