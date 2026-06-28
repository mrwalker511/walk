// Package session tracks token usage and costs across sessions
// using a local SQLite ledger.
package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Entry records a single token usage event.
type Entry struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	Timestamp    time.Time `json:"timestamp"`
	Label        string    `json:"label,omitempty"`
}

// SessionSummary aggregates usage for a session.
type SessionSummary struct {
	SessionID       string  `json:"session_id"`
	TotalInput      int     `json:"total_input"`
	TotalOutput     int     `json:"total_output"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	EntryCount      int     `json:"entry_count"`
	FirstEntry      time.Time `json:"first_entry"`
	LastEntry       time.Time `json:"last_entry"`
}

// Tracker maintains a SQLite ledger of token usage.
type Tracker struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// NewTracker opens or creates the SQLite database at the given path.
func NewTracker(dbPath string) (*Tracker, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("session dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// WAL mode for concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("wal mode: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cost REAL NOT NULL DEFAULT 0.0,
			timestamp TEXT NOT NULL DEFAULT (datetime('now')),
			label TEXT DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_id ON sessions(session_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_ts ON sessions(timestamp);
	`); err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &Tracker{db: db, path: dbPath}, nil
}

// Close closes the database.
func (t *Tracker) Close() error {
	return t.db.Close()
}

// Record adds a token usage entry to the ledger.
func (t *Tracker) Record(entry *Entry) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if entry.SessionID == "" {
		entry.SessionID = fmt.Sprintf("walk-%d", time.Now().Unix())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	_, err := t.db.Exec(
		`INSERT INTO sessions (session_id, provider, model, input_tokens, output_tokens, cost, timestamp, label)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.SessionID, entry.Provider, entry.Model,
		entry.InputTokens, entry.OutputTokens, entry.Cost,
		entry.Timestamp.Format(time.RFC3339), entry.Label,
	)
	return err
}

// GetSessionSummary returns aggregated stats for a session.
func (t *Tracker) GetSessionSummary(sessionID string) (*SessionSummary, error) {
	row := t.db.QueryRow(`
		SELECT
			session_id,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(input_tokens + output_tokens), 0),
			COALESCE(SUM(cost), 0),
			COUNT(*),
			MIN(timestamp),
			MAX(timestamp)
		FROM sessions
		WHERE session_id = ?
		GROUP BY session_id
	`, sessionID)

	s := &SessionSummary{}
	var firstStr, lastStr string
	err := row.Scan(&s.SessionID, &s.TotalInput, &s.TotalOutput, &s.TotalTokens,
		&s.TotalCost, &s.EntryCount, &firstStr, &lastStr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, err
	}

	s.FirstEntry, _ = time.Parse(time.RFC3339, firstStr)
	s.LastEntry, _ = time.Parse(time.RFC3339, lastStr)
	return s, nil
}

// ListSessions returns all session IDs with their summaries.
func (t *Tracker) ListSessions(limit int) ([]SessionSummary, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := t.db.Query(`
		SELECT
			session_id,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(input_tokens + output_tokens), 0),
			COALESCE(SUM(cost), 0),
			COUNT(*),
			MIN(timestamp),
			MAX(timestamp)
		FROM sessions
		GROUP BY session_id
		ORDER BY MAX(timestamp) DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []SessionSummary
	for rows.Next() {
		s := SessionSummary{}
		var firstStr, lastStr string
		if err := rows.Scan(&s.SessionID, &s.TotalInput, &s.TotalOutput, &s.TotalTokens,
			&s.TotalCost, &s.EntryCount, &firstStr, &lastStr); err != nil {
			return nil, err
		}
		s.FirstEntry, _ = time.Parse(time.RFC3339, firstStr)
		s.LastEntry, _ = time.Parse(time.RFC3339, lastStr)
		summaries = append(summaries, s)
	}

	if summaries == nil {
		return []SessionSummary{}, nil
	}
	return summaries, nil
}

// BudgetAlert checks if a session is approaching or exceeding budget limits.
type BudgetAlert struct {
	SessionID     string
	TokenLimit    int
	TokensUsed    int
	CostLimit     float64
	CostSpent     float64
	TokenExceeded bool
	CostExceeded  bool
}

// CheckBudget returns a budget alert if limits are exceeded.
func (t *Tracker) CheckBudget(sessionID string, tokenLimit int, costLimit float64) (*BudgetAlert, error) {
	session, err := t.GetSessionSummary(sessionID)
	if err != nil {
		return &BudgetAlert{
			SessionID: sessionID,
			TokenLimit: tokenLimit,
			CostLimit:  costLimit,
		}, nil
	}

	alert := &BudgetAlert{
		SessionID:  sessionID,
		TokenLimit: tokenLimit,
		TokensUsed: session.TotalTokens,
		CostLimit:  costLimit,
		CostSpent:  session.TotalCost,
	}

	if session.TotalTokens > tokenLimit {
		alert.TokenExceeded = true
	}
	if session.TotalCost > costLimit {
		alert.CostExceeded = true
	}

	return alert, nil
}

// TotalUsage returns cumulative usage across all sessions.
func (t *Tracker) TotalUsage() (*SessionSummary, error) {
	row := t.db.QueryRow(`
		SELECT
			'__total__',
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(input_tokens + output_tokens), 0),
			COALESCE(SUM(cost), 0),
			COUNT(*),
			MIN(timestamp),
			MAX(timestamp)
		FROM sessions
	`)

	s := &SessionSummary{}
	var firstStr, lastStr string
	err := row.Scan(&s.SessionID, &s.TotalInput, &s.TotalOutput, &s.TotalTokens,
		&s.TotalCost, &s.EntryCount, &firstStr, &lastStr)
	if err != nil {
		return &SessionSummary{}, nil
	}

	s.FirstEntry, _ = time.Parse(time.RFC3339, firstStr)
	s.LastEntry, _ = time.Parse(time.RFC3339, lastStr)
	return s, nil
}

// RecentEntries returns the most recent entries.
func (t *Tracker) RecentEntries(limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := t.db.Query(`
		SELECT id, session_id, provider, model, input_tokens, output_tokens, cost, timestamp, label
		FROM sessions
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		e := Entry{}
		var ts string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Provider, &e.Model,
			&e.InputTokens, &e.OutputTokens, &e.Cost, &ts, &e.Label); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		entries = append(entries, e)
	}

	return entries, nil
}