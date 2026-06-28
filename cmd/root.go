// Package cmd provides the cobra command hierarchy for walk.
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/walk-labs/walk/config"
)

// Simple ANSI color helpers (no external dependency)
func colorCyan(s string) string  { return "\033[36m" + s + "\033[0m" }
func colorGreen(s string) string { return "\033[32m" + s + "\033[0m" }
func colorRed(s string) string   { return "\033[31m" + s + "\033[0m" }
func colorYellow(s string) string { return "\033[33m" + s + "\033[0m" }

var (
    cfgFile string
    cfg     *config.Config
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
    Use:   "walk",
    Short: "LLM token optimizer — intercept, analyze, compress, and track",
    Long: `walk sits between your tools and their LLM providers, optimizing
every token payload. It analyzes waste, compresses context, caches
redundant chunks, tracks budgets, and scrubs secrets — all locally.`,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        var err error
        cfg, err = config.Load()
        return err
    },
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println(colorCyan("walk") + " — LLM token optimizer")
        fmt.Println()
        fmt.Println("Usage: walk [command]")
        fmt.Println()
        fmt.Println("Commands:")
        fmt.Println("  analyze   Analyze a payload for token waste")
        fmt.Println("  compress  Compress context via llama.cpp")
        fmt.Println("  watch     Start the transparent proxy")
        fmt.Println("  report    Show session budget and usage report")
        fmt.Println("  init      Create default configuration")
        fmt.Println("  cache     Manage semantic cache")
        fmt.Println()
        fmt.Println("Run 'walk [command] --help' for details.")
    },
}

// Execute runs the root command.
func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.walk/config.yaml)")

    rootCmd.AddCommand(analyzeCmd)
    rootCmd.AddCommand(compressCmd)
    rootCmd.AddCommand(watchCmd)
    rootCmd.AddCommand(reportCmd)
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(cacheCmd)
}

// verbose flag used by multiple commands
func init() {
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}

// fatalf prints an error and exits.
func fatalf(format string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, colorRed("error: ")+format+"\n", args...)
    os.Exit(1)
}

// success prints a success message.
func success(format string, args ...interface{}) {
    fmt.Printf(colorGreen("✓ ")+format+"\n", args...)
}

// warn prints a warning.
func warn(format string, args ...interface{}) {
    fmt.Printf(colorYellow("⚠ ")+format+"\n", args...)
}

// info prints an info message.
func info(format string, args ...interface{}) {
    fmt.Printf(colorCyan("ℹ ")+format, args...)
}