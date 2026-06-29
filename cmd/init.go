package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard",
	Long:  "Detect llama.cpp, configure API keys, set budget limits, and write ~/.walk/config.yaml",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if dryRun {
		fmt.Println("[dry-run] Would create ~/.walk/config.yaml with default settings")
		return nil
	}

	dir, err := config.EnsureDir()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	cfg := defaultInitConfig(dir)

	if !quiet {
		fmt.Println("walk init — setup wizard")
		fmt.Println("========================")
	}

	// Detect llama.cpp
	llamaEndpoint := "http://localhost:8080/v1"
	llamaHealthy := checkLlamaHealth(llamaEndpoint)
	if llamaHealthy {
		if !quiet {
			fmt.Printf("✓ llama.cpp detected at %s\n", llamaEndpoint)
		}
		cfg.LocalModel.Enabled = true
		cfg.LocalModel.Endpoint = llamaEndpoint
	} else {
		if !quiet {
			fmt.Printf("✗ llama.cpp not found at %s (local routing disabled)\n", llamaEndpoint)
		}
		cfg.LocalModel.Enabled = false
	}

	// Prompt for custom endpoint
	if !quiet {
		endpoint := prompt(reader, fmt.Sprintf("llama.cpp endpoint [%s]: ", llamaEndpoint))
		if endpoint != "" {
			cfg.LocalModel.Endpoint = endpoint
		}
	}

	// API keys — store as env var references only
	if !quiet {
		fmt.Println("\nAPI Keys (stored as ${ENV_VAR} references — never plaintext):")
		anthropicKey := prompt(reader, "Anthropic API key env var [ANTHROPIC_API_KEY]: ")
		if anthropicKey == "" {
			anthropicKey = "ANTHROPIC_API_KEY"
		}
		cfg.Providers.Anthropic.APIKey = "${" + strings.TrimPrefix(strings.TrimSuffix(anthropicKey, "}"), "${") + "}"

		openaiKey := prompt(reader, "OpenAI API key env var [OPENAI_API_KEY]: ")
		if openaiKey == "" {
			openaiKey = "OPENAI_API_KEY"
		}
		cfg.Providers.OpenAI.APIKey = "${" + strings.TrimPrefix(strings.TrimSuffix(openaiKey, "}"), "${") + "}"
	}

	// Budget
	if !quiet {
		fmt.Println("\nBudget:")
		dailyStr := prompt(reader, "Daily limit in USD [10.00]: ")
		if dailyStr != "" {
			if v, err := strconv.ParseFloat(dailyStr, 64); err == nil {
				cfg.Budget.DailyLimit = v
			}
		}

		sessionStr := prompt(reader, "Session limit in USD [2.00]: ")
		if sessionStr != "" {
			if v, err := strconv.ParseFloat(sessionStr, 64); err == nil {
				cfg.Budget.SessionLimit = v
			}
		}
	}

	if err := config.Write(dir, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if !quiet {
		fmt.Printf("\n✓ Config written to %s/config.yaml\n", dir)
		fmt.Println("\nRun 'walk analyze <file>' to start analyzing prompts.")
	}
	return nil
}

func defaultInitConfig(dir string) *config.Config {
	return &config.Config{
		LocalModel: config.LocalModel{
			Provider:       "llama.cpp",
			Endpoint:       "http://localhost:8080/v1",
			Model:          "gemma-4-27b-q8_0",
			TimeoutSeconds: 30,
			Enabled:        true,
		},
		Providers: config.Providers{
			Anthropic: config.ProviderConfig{
				APIKey:       "${ANTHROPIC_API_KEY}",
				DefaultModel: "claude-sonnet-4-5",
			},
			OpenAI: config.ProviderConfig{
				APIKey:       "${OPENAI_API_KEY}",
				DefaultModel: "gpt-4o",
			},
		},
		Budget: config.Budget{
			DailyLimit:    10.00,
			SessionLimit:  2.00,
			WarnAtPercent: 80,
			HardStop:      true,
		},
		Scrubber: config.Scrubber{
			Enabled:          true,
			BlockOnDetect:    true,
			Patterns:         []string{"api_key", "jwt", "aws_credential", "ssh_key", "email", "ssn", "phone"},
			EntropyThreshold: 4.5,
		},
		Cache: config.Cache{
			TrackHits:         true,
			OptimizeOnAnalyze: true,
		},
		Session: config.Session{
			DBPath:       dir + "/sessions.db",
			AuditLog:     dir + "/audit.log",
			AuditEnabled: true,
		},
		Output: config.Output{
			Color:           true,
			ShowSavingsLine: true,
			DefaultFormat:   "table",
		},
	}
}

func checkLlamaHealth(endpoint string) bool {
	base := strings.TrimSuffix(endpoint, "/v1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func prompt(r *bufio.Reader, text string) string {
	fmt.Print(text)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
