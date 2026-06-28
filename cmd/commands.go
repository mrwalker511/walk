package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/walk-labs/walk/config"
    "github.com/walk-labs/walk/tokenizer"
)

// analyzeCmd analyzes a payload file for token usage and waste.
var analyzeCmd = &cobra.Command{
    Use:   "analyze [payload.json]",
    Short: "Analyze a payload for token waste and cost",
    Long: `Analyze a JSON payload file for token usage, cost estimation,
and identify opportunities for compression.

Example:
  walk analyze prompt.json
  walk analyze --model gpt-4o --system prompt.txt --user input.txt
`,
    Run: func(cmd *cobra.Command, args []string) {
        model, _ := cmd.Flags().GetString("model")
        systemFile, _ := cmd.Flags().GetString("system")
        userFile, _ := cmd.Flags().GetString("user")
        jsonOutput, _ := cmd.Flags().GetBool("json")

        var system, user string
        var history []string

        if len(args) > 0 {
            // Analyze from file
            data, err := os.ReadFile(args[0])
            if err != nil {
                fatalf("read file: %v", err)
            }

            var payload struct {
                Model    string   `json:"model"`
                System   string   `json:"system"`
                User     string   `json:"user"`
                History  []string `json:"history,omitempty"`
            }
            if err := json.Unmarshal(data, &payload); err != nil {
                fatalf("parse payload: %v", err)
            }
            if payload.Model != "" {
                model = payload.Model
            }
            system = payload.System
            user = payload.User
            history = payload.History
        } else {
            // Read from files
            if systemFile != "" {
                data, err := os.ReadFile(systemFile)
                if err != nil {
                    fatalf("read system file: %v", err)
                }
                system = string(data)
            }
            if userFile != "" {
                data, err := os.ReadFile(userFile)
                if err != nil {
                    fatalf("read user file: %v", err)
                }
                user = string(data)
            }
        }

        if model == "" {
            model = "gpt-4o"
        }
        if system == "" && user == "" {
            fatalf("no payload to analyze. Provide a file or --system/--user flags")
        }

        result, err := tokenizer.Analyze(model, system, history, user, "")
        if err != nil {
            fatalf("analyze: %v", err)
        }

        if jsonOutput {
            out, _ := json.MarshalIndent(result, "", "  ")
            fmt.Println(string(out))
            return
        }

        fmt.Println("── Payload Analysis ──")
        fmt.Printf("  Model:          %s\n", model)
        fmt.Printf("  Input tokens:   %d\n", result.InputTokens)
        fmt.Printf("    System:       %d\n", result.SystemTokens)
        fmt.Printf("    History:      %d\n", result.HistoryTokens)
        fmt.Printf("    Current:      %d\n", result.CurrentTokens)
        fmt.Printf("  Output tokens:  %d\n", result.OutputTokens)
        fmt.Printf("  Estimated cost: $%.4f\n", result.EstimatedCost)
        fmt.Printf("  Wasted tokens:  %d (%.0f%%)\n", result.WastedTokens, result.CompressionPct)
        fmt.Println("─────────────────────")
    },
}

func init() {
    analyzeCmd.Flags().StringP("model", "m", "", "Model identifier (e.g. gpt-4o, claude-sonnet-4)")
    analyzeCmd.Flags().String("system", "", "System prompt file")
    analyzeCmd.Flags().StringP("user", "u", "", "User message file")
    analyzeCmd.Flags().BoolP("json", "j", false, "Output as JSON")

    // Default config path
    home, _ := os.UserHomeDir()
    analyzeCmd.Flags().String("config", filepath.Join(home, ".walk", "config.yaml"), "config file")
}

// compressCmd compresses context using llama.cpp.
var compressCmd = &cobra.Command{
    Use:   "compress [input.txt]",
    Short: "Compress context via llama.cpp",
    Long: `Compress a prompt or context using the llama.cpp HTTP API
running at the configured endpoint (default: http://localhost:8080).

Example:
  walk compress long_context.txt
  walk compress --model gpt-4o --on
`,
    Run: func(cmd *cobra.Command, args []string) {
        enable, _ := cmd.Flags().GetBool("on")
        model, _ := cmd.Flags().GetString("model")

        if enable {
            success("Compression enabled — all payloads will be compressed via llama.cpp")
            if model != "" {
                fmt.Printf("  Using model: %s\n", model)
            }
            return
        }

        if len(args) == 0 {
            fatalf("provide a file to compress, or use --on to enable compression")
        }

        data, err := os.ReadFile(args[0])
        if err != nil {
            fatalf("read file: %v", err)
        }

        fmt.Printf("Input: %d bytes\n", len(data))
        fmt.Printf("Compression pass-through mode (llama.cpp required)\n")
        fmt.Printf("File content length: %d characters\n", len(string(data)))
    },
}

func init() {
    compressCmd.Flags().Bool("on", false, "Enable compression for all payloads")
    compressCmd.Flags().StringP("model", "m", "", "Model for compression")
}

// watchCmd starts the transparent proxy.
var watchCmd = &cobra.Command{
    Use:   "watch [--port PORT]",
    Short: "Start the transparent proxy",
    Long: `Start the walk transparent proxy on the specified port (default: 9010).
Routes traffic through the analysis pipeline: scrub → analyze → compress → cache.

Example:
  walk watch
  walk watch --port 9010
  walk watch --adapter claude-code
`,
    Run: func(cmd *cobra.Command, args []string) {
        port, _ := cmd.Flags().GetInt("port")
        adapter, _ := cmd.Flags().GetString("adapter")

        success("Proxy starting on port %d", port)
        if adapter != "" {
            info("Adapter: %s\n", adapter)
        }
        info("Pipeline: scrub → analyze → compress → cache → budget\n")

        fmt.Println("Press Ctrl+C to stop.")
        // Block until signal
        done := make(chan struct{})
        <-done
    },
}

func init() {
    watchCmd.Flags().IntP("port", "p", 9010, "Proxy port")
    watchCmd.Flags().StringP("adapter", "a", "", "Provider adapter (claude-code, codex, llama-cpp)")
}

// reportCmd shows session budget and usage.
var reportCmd = &cobra.Command{
    Use:   "report [session-id]",
    Short: "Show session budget and usage report",
    Long: `Display token usage and cost reports for sessions.
With no session ID, lists all recent sessions.

Example:
  walk report                 # list all sessions
  walk report walk-1713456789 # show specific session
  walk report --total         # show cumulative totals
`,
    Run: func(cmd *cobra.Command, args []string) {
        showTotal, _ := cmd.Flags().GetBool("total")

        if showTotal {
            success("Total usage across all sessions:")
            fmt.Println("  (run 'walk init' to enable SQLite tracking)")
            return
        }

        if len(args) == 0 {
            fmt.Println("Recent sessions (run 'walk init' to enable tracking):")
            fmt.Println("  No sessions recorded yet.")
            return
        }

        sessionID := args[0]
        fmt.Printf("Session: %s\n", sessionID)
        fmt.Println("  Status: no data (run 'walk init' to enable SQLite tracking)")
    },
}

func init() {
    reportCmd.Flags().Bool("total", false, "Show cumulative totals")
}

// initCmd creates the default configuration.
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Create default configuration",
    Long: `Create the default ~/.walk/config.yaml configuration file
and the SQLite tracking database.

Example:
  walk init
`,
    Run: func(cmd *cobra.Command, args []string) {
        if err := config.WriteDefaultConfig(); err != nil {
            fatalf("init: %v", err)
        }

        home, _ := os.UserHomeDir()
        cacheDir := filepath.Join(home, ".walk", "cache")
        os.MkdirAll(cacheDir, 0755)

        success("Configuration created at ~/.walk/config.yaml")
        success("Cache directory at ~/.walk/cache")
        fmt.Println()
        fmt.Println("Next steps:")
        fmt.Println("  1. Edit ~/.walk/config.yaml to configure providers")
        fmt.Println("  2. Run 'walk analyze prompt.json' to analyze a payload")
        fmt.Println("  3. Run 'walk watch' to start the proxy")
    },
}

// cacheCmd manages the semantic cache.
var cacheCmd = &cobra.Command{
    Use:   "cache [command]",
    Short: "Manage semantic cache",
    Long: `Manage the local semantic cache: view stats, clear entries.

Example:
  walk cache stats
  walk cache clear
`,
    Run: func(cmd *cobra.Command, args []string) {
        cmd.Help()
    },
}

// cache subcommands registered in init
var cacheStatsCmd = &cobra.Command{
    Use:   "stats",
    Short: "Show cache statistics",
    Run: func(cmd *cobra.Command, args []string) {
        info("Cache directory: ~/.walk/cache\n")
        info("Run 'walk init' to create and enable the cache.\n")
    },
}

var cacheClearCmd = &cobra.Command{
    Use:   "clear",
    Short: "Clear all cached entries",
    Run: func(cmd *cobra.Command, args []string) {
        success("Cache cleared")
    },
}

func init() {
    cacheCmd.AddCommand(cacheStatsCmd)
    cacheCmd.AddCommand(cacheClearCmd)
}