package session

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	started_at   DATETIME NOT NULL,
	ended_at     DATETIME,
	model        TEXT NOT NULL DEFAULT '',
	tag          TEXT NOT NULL DEFAULT '',
	tokens_in    INTEGER NOT NULL DEFAULT 0,
	tokens_out   INTEGER NOT NULL DEFAULT 0,
	tokens_cached INTEGER NOT NULL DEFAULT 0,
	cost_usd     REAL NOT NULL DEFAULT 0.0,
	notes        TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS daily_spend (
	date         TEXT PRIMARY KEY,
	cost_usd     REAL NOT NULL DEFAULT 0.0,
	tokens_total INTEGER NOT NULL DEFAULT 0
);
`

// DB wraps an SQLite connection for session tracking.
type DB struct {
	db       *sql.DB
	auditLog string
}

// SessionRecord is a single session row.
type SessionRecord struct {
	ID           int64
	StartedAt    time.Time
	EndedAt      *time.Time
	Model        string
	Tag          string
	TokensIn     int64
	TokensOut    int64
	TokensCached int64
	CostUSD      float64
	Notes        string
}

// DailySpend is the aggregate spend for a calendar day.
type DailySpend struct {
	Date        string
	CostUSD     float64
	TokensTotal int64
}

// Open opens (or creates) the SQLite DB at dbPath and applies the schema.
func Open(dbPath, auditLogPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("creating db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db at %s: %w (hint: check path permissions)", dbPath, err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}

	return &DB{db: db, auditLog: auditLogPath}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// StartSession inserts a new session row and returns its ID.
func (d *DB) StartSession(model, tag string) (int64, error) {
	res, err := d.db.Exec(
		`INSERT INTO sessions (started_at, model, tag) VALUES (?, ?, ?)`,
		time.Now().UTC(), model, tag,
	)
	if err != nil {
		return 0, fmt.Errorf("starting session: %w", err)
	}
	return res.LastInsertId()
}

// EndSession records token usage and cost for a session.
func (d *DB) EndSession(id, tokensIn, tokensOut, tokensCached int64, costUSD float64) error {
	now := time.Now().UTC()
	_, err := d.db.Exec(
		`UPDATE sessions SET ended_at=?, tokens_in=?, tokens_out=?, tokens_cached=?, cost_usd=? WHERE id=?`,
		now, tokensIn, tokensOut, tokensCached, costUSD, id,
	)
	if err != nil {
		return fmt.Errorf("ending session %d: %w", id, err)
	}
	// Update daily spend
	date := now.Format("2006-01-02")
	_, err = d.db.Exec(`
		INSERT INTO daily_spend (date, cost_usd, tokens_total)
		VALUES (?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			cost_usd = cost_usd + excluded.cost_usd,
			tokens_total = tokens_total + excluded.tokens_total
	`, date, costUSD, tokensIn+tokensOut+tokensCached)
	return err
}

// GetSession retrieves a session by ID.
func (d *DB) GetSession(id int64) (*SessionRecord, error) {
	row := d.db.QueryRow(
		`SELECT id, started_at, ended_at, model, tag, tokens_in, tokens_out, tokens_cached, cost_usd, notes FROM sessions WHERE id=?`, id,
	)
	return scanSession(row)
}

// GetLastSession retrieves the most recently started session.
func (d *DB) GetLastSession() (*SessionRecord, error) {
	row := d.db.QueryRow(
		`SELECT id, started_at, ended_at, model, tag, tokens_in, tokens_out, tokens_cached, cost_usd, notes FROM sessions ORDER BY id DESC LIMIT 1`,
	)
	return scanSession(row)
}

// ListSessions returns all sessions, newest first.
func (d *DB) ListSessions() ([]SessionRecord, error) {
	rows, err := d.db.Query(
		`SELECT id, started_at, ended_at, model, tag, tokens_in, tokens_out, tokens_cached, cost_usd, notes FROM sessions ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]SessionRecord, 0)
	for rows.Next() {
		rec, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *rec)
	}
	return records, rows.Err()
}

// TodaySpend returns the running daily spend for today.
func (d *DB) TodaySpend() (DailySpend, error) {
	date := time.Now().UTC().Format("2006-01-02")
	row := d.db.QueryRow(`SELECT date, cost_usd, tokens_total FROM daily_spend WHERE date=?`, date)
	var spend DailySpend
	err := row.Scan(&spend.Date, &spend.CostUSD, &spend.TokensTotal)
	if err == sql.ErrNoRows {
		return DailySpend{Date: date}, nil
	}
	return spend, err
}

// ResetDailySpend zeroes the spend counter for today.
func (d *DB) ResetDailySpend() error {
	date := time.Now().UTC().Format("2006-01-02")
	_, err := d.db.Exec(`DELETE FROM daily_spend WHERE date=?`, date)
	return err
}

// AuditLog appends a SHA-256 hash of the payload to the audit log (never plaintext).
func (d *DB) AuditLog(payload string) error {
	if d.auditLog == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(d.auditLog), 0700); err != nil {
		return fmt.Errorf("creating audit log dir: %w", err)
	}
	f, err := os.OpenFile(d.auditLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}
	defer f.Close()

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	line := fmt.Sprintf("%s sha256=%s\n", time.Now().UTC().Format(time.RFC3339), hash)
	_, err = f.WriteString(line)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSession(s scanner) (*SessionRecord, error) {
	var rec SessionRecord
	var startedAt string
	var endedAt sql.NullString
	err := s.Scan(
		&rec.ID, &startedAt, &endedAt,
		&rec.Model, &rec.Tag,
		&rec.TokensIn, &rec.TokensOut, &rec.TokensCached,
		&rec.CostUSD, &rec.Notes,
	)
	if err != nil {
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
		rec.StartedAt = t
	} else if t, err := time.Parse("2006-01-02T15:04:05Z", startedAt); err == nil {
		rec.StartedAt = t
	} else {
		rec.StartedAt = time.Now()
	}
	if endedAt.Valid && endedAt.String != "" {
		if t, err := time.Parse(time.RFC3339, endedAt.String); err == nil {
			rec.EndedAt = &t
		}
	}
	return &rec, nil
}
