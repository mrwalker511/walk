package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrwalker511/walk/internal/config"
	"github.com/mrwalker511/walk/internal/session"
)

// captureStdout temporarily replaces os.Stdout with a pipe and returns the captured output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

// resetGlobals resets all package-level cobra flag vars to their zero values.
func resetGlobals() {
	cfgDir = ""
	jsonOut = false
	quiet = false
	dryRun = false
	model = ""
	globalCfg = nil
	scrubOutput = ""
	budgetSet = ""
	budgetReset = false
	budgetStatus = false
	reportSession = "last"
	reportFormat = ""
}

// newTestConfig returns a *config.Config with Session paths pointing to t.TempDir().
func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		Session: config.Session{
			DBPath:       filepath.Join(dir, "sessions.db"),
			AuditLog:     filepath.Join(dir, "audit.log"),
			AuditEnabled: false,
		},
		Budget: config.Budget{
			DailyLimit:    10.00,
			WarnAtPercent: 80,
			HardStop:      false,
		},
		Providers: config.Providers{
			Anthropic: config.ProviderConfig{DefaultModel: "claude-sonnet-4-5"},
		},
		Scrubber: config.Scrubber{EntropyThreshold: 4.5},
	}
}

// --- Pure function tests ---

func TestFormatTokens(t *testing.T) {
	assert.Equal(t, "0", formatTokens(0))
	assert.Equal(t, "999", formatTokens(999))
	assert.Equal(t, "1,000", formatTokens(1000))
	assert.Equal(t, "1,001", formatTokens(1001))
	assert.Equal(t, "999,999", formatTokens(999999))
	assert.Equal(t, "1,000,000", formatTokens(1_000_000))
	assert.Equal(t, "1,234,567", formatTokens(1_234_567))
}

func TestErrorHint(t *testing.T) {
	assert.Contains(t, errorHint(errors.New("config not found")), "walk init")
	assert.Contains(t, errorHint(errors.New("llama server unavailable")), "llama-server")
	assert.Contains(t, errorHint(errors.New("permission denied: file")), "privileges")
	assert.Equal(t, "", errorHint(errors.New("some completely unknown error")))
}

func TestContains(t *testing.T) {
	assert.True(t, contains("hello world", "world"))
	assert.True(t, contains("hello", "hello"))
	assert.False(t, contains("hello", "xyz"))
	assert.False(t, contains("", "x"))
	assert.True(t, contains("abc", ""))
}

func TestPrintSavings(t *testing.T) {
	t.Cleanup(resetGlobals)

	// quiet=true → no output regardless of token count
	quiet = true
	out := captureStdout(t, func() { printSavings(5000, 0.015, "claude-sonnet-4-5") })
	assert.Empty(t, out)

	// zero tokens → no output
	quiet = false
	out = captureStdout(t, func() { printSavings(0, 0.0, "claude-sonnet-4-5") })
	assert.Empty(t, out)

	// normal: token count and model name appear in output
	out = captureStdout(t, func() { printSavings(5000, 0.015, "claude-sonnet-4-5") })
	assert.Contains(t, out, "5,000")
	assert.Contains(t, out, "claude-sonnet-4-5")

	// empty model name defaults to "unknown"
	out = captureStdout(t, func() { printSavings(100, 0.001, "") })
	assert.Contains(t, out, "unknown")
}

// --- walk analyze ---

func TestRunAnalyze(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(f, []byte("This is a test prompt for token analysis."), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runAnalyze(nil, []string{f}))
	})
	assert.Contains(t, out, "Tokens")
	assert.Contains(t, out, "Model")
}

func TestRunAnalyzeJSON(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(f, []byte("Hello world."), 0644))
	jsonOut = true

	out := captureStdout(t, func() {
		assert.NoError(t, runAnalyze(nil, []string{f}))
	})
	assert.Contains(t, out, "token_count")
	assert.Contains(t, out, "word_count")
}

func TestRunAnalyzeWithModel(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(f, []byte("Short prompt."), 0644))
	model = "claude-sonnet-4-5"

	out := captureStdout(t, func() {
		assert.NoError(t, runAnalyze(nil, []string{f}))
	})
	assert.Contains(t, out, "claude-sonnet-4-5")
}

func TestRunAnalyzeMissingFile(t *testing.T) {
	t.Cleanup(resetGlobals)
	err := runAnalyze(nil, []string{"/nonexistent/file.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading file")
}

func TestRunAnalyzeQuiet(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(f, []byte("Some content."), 0644))
	quiet = true

	out := captureStdout(t, func() {
		assert.NoError(t, runAnalyze(nil, []string{f}))
	})
	assert.Empty(t, out)
}

// --- walk diff ---

func TestRunDiffTokensSaved(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	orig := filepath.Join(dir, "original.txt")
	opt := filepath.Join(dir, "optimized.txt")
	require.NoError(t, os.WriteFile(orig, []byte(strings.Repeat("word ", 200)), 0644))
	require.NoError(t, os.WriteFile(opt, []byte(strings.Repeat("word ", 50)), 0644))
	model = "claude-sonnet-4-5"

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{orig, opt}))
	})
	assert.Contains(t, out, "Tokens saved")
}

func TestRunDiffNoChange(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "same.txt")
	require.NoError(t, os.WriteFile(f, []byte("identical content"), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{f, f}))
	})
	assert.Contains(t, out, "No token difference")
}

func TestRunDiffTokensAdded(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	orig := filepath.Join(dir, "original.txt")
	bigger := filepath.Join(dir, "bigger.txt")
	require.NoError(t, os.WriteFile(orig, []byte("short"), 0644))
	require.NoError(t, os.WriteFile(bigger, []byte(strings.Repeat("word ", 200)), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{orig, bigger}))
	})
	assert.Contains(t, out, "Tokens added")
}

func TestRunDiffJSON(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(f, []byte("some text"), 0644))
	jsonOut = true

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{f, f}))
	})
	assert.Contains(t, out, "original_tokens")
	assert.Contains(t, out, "token_delta")
}

func TestRunDiffMissingFile(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(f, []byte("content"), 0644))

	err := runDiff(nil, []string{f, "/nonexistent.txt"})
	assert.Error(t, err)
}

func TestRunDiffHighlightsRemovedLines(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	orig := filepath.Join(dir, "original.txt")
	opt := filepath.Join(dir, "optimized.txt")
	require.NoError(t, os.WriteFile(orig, []byte("keep this line\nremove this line\n"), 0644))
	require.NoError(t, os.WriteFile(opt, []byte("keep this line\n"), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{orig, opt}))
	})
	assert.Contains(t, out, "=== Diff ===")
	assert.Contains(t, out, "remove this line")
}

func TestRunDiffJSONIncludesRemovedLines(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	orig := filepath.Join(dir, "original.txt")
	opt := filepath.Join(dir, "optimized.txt")
	require.NoError(t, os.WriteFile(orig, []byte("keep this line\nremove this line\n"), 0644))
	require.NoError(t, os.WriteFile(opt, []byte("keep this line\nadd this line\n"), 0644))
	jsonOut = true

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{orig, opt}))
	})
	assert.Contains(t, out, "removed_lines")
	assert.Contains(t, out, "remove this line")
	assert.Contains(t, out, "added_lines")
	assert.Contains(t, out, "add this line")
}

func TestRunDiffNoChangeNoHighlight(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "same.txt")
	require.NoError(t, os.WriteFile(f, []byte("identical content"), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runDiff(nil, []string{f, f}))
	})
	assert.NotContains(t, out, "=== Diff ===")
}

// --- walk scrub ---

func TestRunScrubClean(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "clean.txt")
	require.NoError(t, os.WriteFile(f, []byte("This is clean text with no secrets."), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runScrub(nil, []string{f}))
	})
	assert.Contains(t, out, "clean text")
}

func TestRunScrubDryRun(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "payload.txt")
	require.NoError(t, os.WriteFile(f, []byte("some safe text here"), 0644))
	dryRun = true

	assert.NoError(t, runScrub(nil, []string{f}))
}

func TestRunScrubJSON(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "clean.txt")
	require.NoError(t, os.WriteFile(f, []byte("No secrets here, just plain text."), 0644))
	jsonOut = true

	out := captureStdout(t, func() {
		assert.NoError(t, runScrub(nil, []string{f}))
	})
	assert.Contains(t, out, `"has_secrets"`)
	assert.Contains(t, out, `"findings"`)
	assert.Contains(t, out, `"clean"`)
	assert.Contains(t, out, "false")
}

func TestRunScrubOutputFile(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	src := filepath.Join(dir, "input.txt")
	dst := filepath.Join(dir, "output.txt")
	require.NoError(t, os.WriteFile(src, []byte("clean payload content"), 0644))
	scrubOutput = dst

	assert.NoError(t, runScrub(nil, []string{src}))
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Contains(t, string(data), "clean payload content")
}

// --- walk cache analyze ---

func TestRunCacheAnalyze(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	content := "You are a helpful assistant.\nAlways be concise.\nUser: What is 2+2?"
	require.NoError(t, os.WriteFile(f, []byte(content), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runCacheAnalyze(nil, []string{f}))
	})
	assert.Contains(t, out, "Stable tokens")
	assert.Contains(t, out, "Dynamic tokens")
}

func TestRunCacheAnalyzeJSON(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(f, []byte("You are helpful.\nUser: help me"), 0644))
	jsonOut = true

	out := captureStdout(t, func() {
		assert.NoError(t, runCacheAnalyze(nil, []string{f}))
	})
	assert.Contains(t, out, "stable_tokens")
	assert.Contains(t, out, "reorder_recommended")
}

func TestRunCacheAnalyzeReorderWarning(t *testing.T) {
	t.Cleanup(resetGlobals)
	dir := t.TempDir()
	f := filepath.Join(dir, "prompt.txt")
	// Dynamic content before stable — should warn about reordering
	content := "User: What should I do?\nYou are a helpful assistant.\nAlways follow the rules."
	require.NoError(t, os.WriteFile(f, []byte(content), 0644))

	out := captureStdout(t, func() {
		assert.NoError(t, runCacheAnalyze(nil, []string{f}))
	})
	assert.Contains(t, out, "Reorder")
}

// --- walk budget ---

func TestRunBudgetStatus(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	budgetStatus = true

	out := captureStdout(t, func() {
		assert.NoError(t, runBudget(nil, []string{}))
	})
	assert.Contains(t, out, "Today's spend")
	assert.Contains(t, out, "Daily limit")
}

func TestRunBudgetStatusJSON(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	budgetStatus = true
	jsonOut = true

	out := captureStdout(t, func() {
		assert.NoError(t, runBudget(nil, []string{}))
	})
	assert.Contains(t, out, "spend_usd")
	assert.Contains(t, out, "limit_usd")
}

func TestRunBudgetSetDryRun(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	budgetSet = "5.00"
	dryRun = true

	out := captureStdout(t, func() {
		assert.NoError(t, runBudget(nil, []string{}))
	})
	assert.Contains(t, out, "dry-run")
	assert.Contains(t, out, "5.00")
}

func TestRunBudgetReset(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	budgetReset = true

	out := captureStdout(t, func() {
		assert.NoError(t, runBudget(nil, []string{}))
	})
	assert.Contains(t, out, "reset")
}

func TestRunBudgetDefaultsToStatus(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	// No flags set — should default to --status

	out := captureStdout(t, func() {
		assert.NoError(t, runBudget(nil, []string{}))
	})
	assert.Contains(t, out, "Today's spend")
}

func TestRunBudgetInvalidAmount(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	budgetSet = "not-a-number"

	err := runBudget(nil, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid budget amount")
}

func TestRunBudgetSetPersists(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	dir := t.TempDir()
	cfgDir = dir
	budgetSet = "7.50"

	assert.NoError(t, runBudget(nil, []string{}))

	persisted, err := config.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, 7.50, persisted.Budget.DailyLimit)
}

func TestRunBudgetSetDryRunDoesNotPersist(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	dir := t.TempDir()
	cfgDir = dir
	budgetSet = "9.00"
	dryRun = true

	assert.NoError(t, runBudget(nil, []string{}))

	_, err := os.Stat(filepath.Join(dir, "config.yaml"))
	assert.True(t, os.IsNotExist(err))
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	result := expandHome("~/foo/bar")
	assert.Equal(t, home+"/foo/bar", result)

	assert.Equal(t, "/absolute/path", expandHome("/absolute/path"))
	assert.Equal(t, "relative", expandHome("relative"))
}

// --- report helpers (pure functions) ---

func TestRepeatStr(t *testing.T) {
	assert.Equal(t, "---", repeatStr("-", 3))
	assert.Equal(t, "", repeatStr("-", 0))
	assert.Equal(t, "abab", repeatStr("ab", 2))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hello", truncate("hello", 5))
	// truncate adds an ellipsis rune when over limit
	result := truncate("hello world", 6)
	assert.Equal(t, 6, len([]rune(result))) // length check in rune terms
	assert.True(t, strings.HasPrefix(result, "hello"))
}

// --- readInput stdin path ---

func TestReadInputStdin(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString("stdin content")
	require.NoError(t, err)
	_ = w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	text, err := readInput(nil)
	require.NoError(t, err)
	assert.Equal(t, "stdin content", text)
}

// --- walk report ---

// newTestSession creates a DB, starts and ends a session, and returns the DB.
func newTestSession(t *testing.T, cfg *config.Config) *session.DB {
	t.Helper()
	db, err := session.Open(cfg.Session.DBPath, cfg.Session.AuditLog)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	id, err := db.StartSession("claude-sonnet-4-5", "test")
	require.NoError(t, err)
	require.NoError(t, db.EndSession(id, 1000, 250, 100, 0.003))
	return db
}

func TestRunReportAll(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	_ = newTestSession(t, globalCfg)
	reportSession = "all"

	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	// Model is truncated to 12 chars in table view: "claude-sonn…"
	assert.Contains(t, out, "claude-sonn")
}

func TestRunReportLast(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	_ = newTestSession(t, globalCfg)
	reportSession = "last"

	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	assert.Contains(t, out, "claude-sonn")
}

func TestRunReportJSON(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	_ = newTestSession(t, globalCfg)
	reportSession = "last"
	reportFormat = "json"

	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	assert.Contains(t, out, "claude-sonnet-4-5")
	assert.Contains(t, out, "cache_hit_ratio")
	assert.Contains(t, out, "cache_savings_usd")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(out), "["))
}

func TestRunReportCSV(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	_ = newTestSession(t, globalCfg)
	reportSession = "last"
	reportFormat = "csv"

	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	assert.Contains(t, out, "id,model,tag")
	assert.Contains(t, out, "cache_hit_ratio,cache_savings_usd")
	assert.Contains(t, out, "claude-sonnet-4-5")
}

func TestRunReportTableShowsCacheMetrics(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	_ = newTestSession(t, globalCfg)
	reportSession = "last"
	reportFormat = "table"

	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	assert.Contains(t, out, "Hit%")
	assert.Contains(t, out, "Savings")
	// newTestSession: tokens_in=1000, tokens_cached=100 -> hit ratio 100/1100 = 9.1%
	assert.Contains(t, out, "9.1%")
}

func TestComputeCacheMetricsZeroDenominator(t *testing.T) {
	ratio, savings := computeCacheMetrics(session.SessionRecord{Model: "claude-sonnet-4-5"})
	assert.Equal(t, 0.0, ratio)
	assert.Equal(t, 0.0, savings)
}

func TestComputeCacheMetricsUnknownModel(t *testing.T) {
	ratio, savings := computeCacheMetrics(session.SessionRecord{
		Model: "unknown-model", TokensIn: 1000, TokensCached: 100,
	})
	assert.InDelta(t, 100.0/1100.0, ratio, 0.0001)
	assert.Equal(t, 0.0, savings)
}

func TestComputeCacheMetricsKnownModel(t *testing.T) {
	ratio, savings := computeCacheMetrics(session.SessionRecord{
		Model: "claude-sonnet-4-5", TokensIn: 900, TokensCached: 100,
	})
	assert.InDelta(t, 0.1, ratio, 0.0001)
	// 100 tokens * (3.00 - 0.30) / 1_000_000
	assert.InDelta(t, 100*(3.00-0.30)/1_000_000, savings, 1e-9)
}

func TestRunReportEmpty(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	reportSession = "all"

	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	assert.Contains(t, out, "No sessions found")
}

func TestRunReportInvalidSessionID(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	reportSession = "not-a-number"

	err := runReport(nil, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session id")
}

func TestRunReportMultipleSessions(t *testing.T) {
	t.Cleanup(resetGlobals)
	globalCfg = newTestConfig(t)
	db, err := session.Open(globalCfg.Session.DBPath, globalCfg.Session.AuditLog)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	for i := 0; i < 3; i++ {
		id, err := db.StartSession("claude-sonnet-4-5", "test")
		require.NoError(t, err)
		require.NoError(t, db.EndSession(id, 100, 25, 0, 0.0003))
	}

	reportSession = "all"
	reportFormat = "table"
	out := captureStdout(t, func() {
		assert.NoError(t, runReport(nil, []string{}))
	})
	// Table has a TOTAL row when multiple sessions are present
	assert.Contains(t, out, "TOTAL")
}
