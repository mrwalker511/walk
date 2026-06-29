package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/mrwalker511/walk/internal/session"
)

var (
	reportSession string
	reportFormat  string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Post-session cost breakdown and savings summary",
	Long: `Show token usage and cost attribution for completed sessions.

Examples:
  walk report                          Show last session
  walk report --session last
  walk report --session all
  walk report --session 42
  walk report --format csv`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportSession, "session", "last", "session ID, 'last', or 'all'")
	reportCmd.Flags().StringVar(&reportFormat, "format", "", "output format: table, json, csv")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
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

	format := reportFormat
	if format == "" && globalCfg != nil {
		format = globalCfg.Output.DefaultFormat
	}
	if format == "" {
		format = "table"
	}
	if jsonOut {
		format = "json"
	}

	var records []session.SessionRecord

	switch reportSession {
	case "all":
		records, err = db.ListSessions()
		if err != nil {
			return fmt.Errorf("listing sessions: %w", err)
		}
	case "last", "":
		rec, err := db.GetLastSession()
		if err != nil {
			return fmt.Errorf("getting last session: %w (hint: complete a session first)", err)
		}
		records = []session.SessionRecord{*rec}
	default:
		id, err := strconv.ParseInt(reportSession, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid session id %q: %w (hint: use a number, 'last', or 'all')", reportSession, err)
		}
		rec, err := db.GetSession(id)
		if err != nil {
			return fmt.Errorf("getting session %d: %w", id, err)
		}
		records = []session.SessionRecord{*rec}
	}

	if len(records) == 0 {
		fmt.Println("No sessions found")
		return nil
	}

	switch format {
	case "json":
		return printReportJSON(records)
	case "csv":
		return printReportCSV(records)
	default:
		return printReportTable(records)
	}
}

func printReportTable(records []session.SessionRecord) error {
	fmt.Printf("%-6s %-12s %-20s %-10s %-10s %-10s %-10s\n",
		"ID", "Model", "Started", "Tokens In", "Tokens Out", "Cached", "Cost")
	fmt.Println(repeatStr("-", 90))
	totalCost := 0.0
	for _, r := range records {
		fmt.Printf("%-6d %-12s %-20s %-10s %-10s %-10s $%-9.4f\n",
			r.ID,
			truncate(r.Model, 12),
			r.StartedAt.Format("2006-01-02 15:04"),
			formatTokens(int(r.TokensIn)),
			formatTokens(int(r.TokensOut)),
			formatTokens(int(r.TokensCached)),
			r.CostUSD,
		)
		totalCost += r.CostUSD
	}
	if len(records) > 1 {
		fmt.Println(repeatStr("-", 90))
		fmt.Printf("%-6s %-12s %-20s %-10s %-10s %-10s $%-9.4f\n",
			"TOTAL", "", "", "", "", "", totalCost)
	}
	return nil
}

func printReportJSON(records []session.SessionRecord) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

func printReportCSV(records []session.SessionRecord) error {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"id", "model", "tag", "started_at", "tokens_in", "tokens_out", "tokens_cached", "cost_usd"})
	for _, r := range records {
		w.Write([]string{
			strconv.FormatInt(r.ID, 10),
			r.Model,
			r.Tag,
			r.StartedAt.Format("2006-01-02T15:04:05Z"),
			strconv.FormatInt(r.TokensIn, 10),
			strconv.FormatInt(r.TokensOut, 10),
			strconv.FormatInt(r.TokensCached, 10),
			fmt.Sprintf("%.6f", r.CostUSD),
		})
	}
	w.Flush()
	return w.Error()
}

func repeatStr(s string, n int) string {
	result := make([]byte, n*len(s))
	for i := 0; i < n; i++ {
		copy(result[i*len(s):], s)
	}
	return string(result)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
