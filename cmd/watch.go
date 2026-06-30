package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/session"
)

var (
	watchTool     string
	watchInterval int
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Live session token burn rate monitor",
	Long: `Monitor an active LLM session's token usage in real time.

Tracks token burn rate, enforces budget caps, and warns on context rot.

Examples:
  walk watch
  walk watch --tool claude-code
  walk watch --interval 5`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().StringVar(&watchTool, "tool", "", "tool to monitor: claude-code, codex")
	watchCmd.Flags().IntVar(&watchInterval, "interval", 3, "polling interval in seconds")
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	dbPath := expandHome("~/.walk/sessions.db")
	auditLog := expandHome("~/.walk/audit.log")
	if globalCfg != nil {
		dbPath = expandHome(globalCfg.Session.DBPath)
		auditLog = expandHome(globalCfg.Session.AuditLog)
	}

	db, err := session.Open(dbPath, auditLog)
	if err != nil {
		return fmt.Errorf("opening session db: %w (hint: run 'walk init' first)", err)
	}
	defer db.Close()

	dailyLimit := 10.00
	warnPercent := 80
	if globalCfg != nil {
		dailyLimit = globalCfg.Budget.DailyLimit
		warnPercent = globalCfg.Budget.WarnAtPercent
	}

	if !quiet {
		toolSuffix := ""
		if watchTool != "" && watchTool != "none" {
			toolSuffix = " " + watchTool
		}
		fmt.Printf("walk watch — monitoring%s (Ctrl+C to stop)\n", toolSuffix)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(watchInterval) * time.Second)
	defer ticker.Stop()

	var lastCost float64
	var lastTime time.Time

	for {
		select {
		case <-sigCh:
			if !quiet {
				fmt.Println("\nwatch stopped")
			}
			return nil
		case now := <-ticker.C:
			spend, err := db.TodaySpend()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error fetching spend: %v\n", err)
				continue
			}

			pct := 0.0
			if dailyLimit > 0 {
				pct = (spend.CostUSD / dailyLimit) * 100
			}

			burnRate := 0.0
			if !lastTime.IsZero() && spend.CostUSD > lastCost {
				elapsed := now.Sub(lastTime).Hours()
				burnRate = (spend.CostUSD - lastCost) / elapsed
			}
			lastCost = spend.CostUSD
			lastTime = now

			if jsonOut {
				out := map[string]interface{}{
					"spend_usd":     spend.CostUSD,
					"limit_usd":     dailyLimit,
					"used_percent":  pct,
					"tokens_today":  spend.TokensTotal,
					"burn_rate_usd_per_hour": burnRate,
					"timestamp":     now.Format(time.RFC3339),
				}
				enc := json.NewEncoder(os.Stdout)
				if err := enc.Encode(out); err != nil {
					return err
				}
			} else if !quiet {
				fmt.Printf("\r[%s] spend: $%.4f / $%.2f (%.1f%%) | tokens: %s",
					now.Format("15:04:05"),
					spend.CostUSD,
					dailyLimit,
					pct,
					formatTokens(int(spend.TokensTotal)),
				)
				if burnRate > 0 {
					fmt.Printf(" | burn: $%.3f/hr", burnRate)
					projTotal := spend.CostUSD + burnRate*8 // 8-hour projection
					fmt.Printf(" | 8h proj: $%.2f", projTotal)
				}
			}

			// Budget warnings
			if pct >= float64(warnPercent) && !quiet {
				fmt.Printf("\n⚠ Warning: %.0f%% of daily budget used\n", pct)
			}
			if pct >= 100 {
				fmt.Fprintln(os.Stderr, "\n✗ Daily budget exceeded")
				if globalCfg != nil && globalCfg.Budget.HardStop {
					os.Exit(1)
				}
			}
		}
	}
}
