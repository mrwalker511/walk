package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/session"
)

var (
	budgetSet    string
	budgetReset  bool
	budgetStatus bool
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage daily/session spend limits",
	Long: `Track and manage your LLM spend limits.

Examples:
  walk budget --status           Show today's spend vs. cap
  walk budget --set 5.00         Set $5.00 daily cap
  walk budget --reset            Reset today's spend counter`,
	RunE: runBudget,
}

func init() {
	budgetCmd.Flags().StringVar(&budgetSet, "set", "", "set daily budget cap in USD (e.g. --set 5.00)")
	budgetCmd.Flags().BoolVar(&budgetReset, "reset", false, "reset today's spend counter to zero")
	budgetCmd.Flags().BoolVar(&budgetStatus, "status", false, "show today's spend vs. cap")
	rootCmd.AddCommand(budgetCmd)
}

func runBudget(cmd *cobra.Command, args []string) error {
	dbPath := "~/.walk/sessions.db"
	auditLog := "~/.walk/audit.log"
	if globalCfg != nil {
		dbPath = globalCfg.Session.DBPath
		auditLog = globalCfg.Session.AuditLog
	}

	dbPath = expandHome(dbPath)
	auditLog = expandHome(auditLog)

	db, err := session.Open(dbPath, auditLog)
	if err != nil {
		return fmt.Errorf("opening session db: %w (hint: run 'walk init' first)", err)
	}
	defer func() { _ = db.Close() }()

	// Default to --status if no flag given
	if !budgetReset && budgetSet == "" {
		budgetStatus = true
	}

	if budgetSet != "" {
		limit, err := strconv.ParseFloat(budgetSet, 64)
		if err != nil {
			return fmt.Errorf("invalid budget amount %q: %w (hint: use a number like 5.00)", budgetSet, err)
		}
		if dryRun {
			fmt.Printf("[dry-run] Would set daily budget to $%.2f\n", limit)
			return nil
		}
		if globalCfg != nil {
			globalCfg.Budget.DailyLimit = limit
		}
		if !quiet {
			fmt.Printf("✓ Daily budget set to $%.2f\n", limit)
		}
	}

	if budgetReset {
		if dryRun {
			fmt.Println("[dry-run] Would reset today's spend counter")
			return nil
		}
		if err := db.ResetDailySpend(); err != nil {
			return fmt.Errorf("resetting daily spend: %w", err)
		}
		if !quiet {
			fmt.Println("✓ Today's spend reset to $0.00")
		}
	}

	if budgetStatus {
		spend, err := db.TodaySpend()
		if err != nil {
			return fmt.Errorf("fetching today's spend: %w", err)
		}

		dailyLimit := 10.00
		if globalCfg != nil {
			dailyLimit = globalCfg.Budget.DailyLimit
		}
		warnPercent := 80
		if globalCfg != nil {
			warnPercent = globalCfg.Budget.WarnAtPercent
		}

		pct := 0.0
		if dailyLimit > 0 {
			pct = (spend.CostUSD / dailyLimit) * 100
		}

		if jsonOut {
			return printBudgetJSON(spend.CostUSD, dailyLimit, pct, spend.TokensTotal)
		}

		if !quiet {
			fmt.Println("=== Budget Status ===")
			fmt.Printf("Today's spend:  $%.4f\n", spend.CostUSD)
			fmt.Printf("Daily limit:    $%.2f\n", dailyLimit)
			fmt.Printf("Used:           %.1f%%\n", pct)
			fmt.Printf("Tokens today:   %s\n", formatTokens(int(spend.TokensTotal)))
		}

		if pct >= float64(warnPercent) && !quiet {
			fmt.Printf("\n⚠ Warning: %.0f%% of daily budget used\n", pct)
		}
		if pct >= 100 && globalCfg != nil && globalCfg.Budget.HardStop {
			fmt.Fprintln(os.Stderr, "✗ Daily budget exceeded — hard stop enabled")
			os.Exit(1)
		}
	}
	return nil
}

type budgetStatusJSON struct {
	SpendUSD    float64 `json:"spend_usd"`
	LimitUSD    float64 `json:"limit_usd"`
	UsedPercent float64 `json:"used_percent"`
	TokensToday int64   `json:"tokens_today"`
}

func printBudgetJSON(spend, limit, pct float64, tokens int64) error {
	out := budgetStatusJSON{
		SpendUSD:    spend,
		LimitUSD:    limit,
		UsedPercent: pct,
		TokensToday: tokens,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}
