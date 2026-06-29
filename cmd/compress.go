package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/compressor"
	"github.com/mrwalker511/walk/internal/config"
	"github.com/mrwalker511/walk/internal/router"
	"github.com/mrwalker511/walk/internal/tokenizer"
)

var (
	compressFile  string
	compressLocal bool
)

var compressCmd = &cobra.Command{
	Use:   "compress",
	Short: "Compress content using llama.cpp before sending to a cloud LLM",
	Long: `Summarise or compress content to reduce token usage.

Examples:
  cat context.md | walk compress
  walk compress --file prompt.md
  walk compress --local --file big_context.md`,
	RunE: runCompress,
}

func init() {
	compressCmd.Flags().StringVarP(&compressFile, "file", "f", "", "input file (default: stdin)")
	compressCmd.Flags().BoolVar(&compressLocal, "local", false, "force local llama.cpp (error if unavailable)")
	rootCmd.AddCommand(compressCmd)
}

func runCompress(cmd *cobra.Command, args []string) error {
	var fileArgs []string
	if compressFile != "" {
		fileArgs = []string{compressFile}
	}
	text, err := readInput(fileArgs)
	if err != nil {
		return err
	}

	if dryRun {
		toks := tokenizer.Count(text)
		fmt.Printf("[dry-run] Would compress %s tokens using llama.cpp\n", formatTokens(toks))
		return nil
	}

	cfg := globalCfg
	if cfg == nil {
		cfg, _ = loadDefaultConfig()
	}

	r := router.New(cfg)
	decision, err := r.Route(context.Background(), compressLocal)
	if err != nil {
		return err
	}

	if decision.Destination == router.DestCloud {
		return fmt.Errorf("compression requires llama.cpp (local model): %s (hint: start with 'llama-server --model /path/to/model.gguf --port 8080')", decision.Reason)
	}

	c := compressor.New(decision.Endpoint, decision.Model, cfg.LocalModel.TimeoutSeconds)
	result, err := c.Compress(context.Background(), text)
	if err != nil {
		return err
	}

	if jsonOut {
		return printCompressJSON(result)
	}
	return printCompressTable(result)
}

func printCompressTable(r compressor.Result) error {
	if !quiet {
		fmt.Printf("Original:    %s tokens\n", formatTokens(r.OriginalTokens))
		fmt.Printf("Compressed:  %s tokens\n", formatTokens(r.CompressedTokens))
		fmt.Printf("Ratio:       %.0f%% of original\n", r.CompressionRatio*100)
	}

	fmt.Print(r.Compressed)
	if !quiet {
		fmt.Println()
	}

	printSavings(r.TokensSaved, 0, r.Model)
	return nil
}

type compressJSON struct {
	OriginalTokens   int     `json:"original_tokens"`
	CompressedTokens int     `json:"compressed_tokens"`
	CompressionRatio float64 `json:"compression_ratio"`
	TokensSaved      int     `json:"tokens_saved"`
	Model            string  `json:"model"`
	Compressed       string  `json:"compressed"`
}

func printCompressJSON(r compressor.Result) error {
	out := compressJSON{
		OriginalTokens:   r.OriginalTokens,
		CompressedTokens: r.CompressedTokens,
		CompressionRatio: r.CompressionRatio,
		TokensSaved:      r.TokensSaved,
		Model:            r.Model,
		Compressed:       r.Compressed,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func loadDefaultConfig() (*config.Config, error) {
	return config.Load()
}
