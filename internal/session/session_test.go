package session

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"), filepath.Join(dir, "audit.log"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestStartEndSession(t *testing.T) {
	db := openTestDB(t)

	id, sessionUUID, err := db.StartSession("claude-sonnet-4-5", "test")
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
	assert.NotEmpty(t, sessionUUID)

	err = db.EndSession(id, 1000, 250, 100, 0.0035)
	require.NoError(t, err)

	rec, err := db.GetSession(id)
	require.NoError(t, err)
	assert.Equal(t, id, rec.ID)
	assert.Equal(t, sessionUUID, rec.SessionUUID)
	assert.Equal(t, "claude-sonnet-4-5", rec.Model)
	assert.Equal(t, int64(1000), rec.TokensIn)
	assert.Equal(t, int64(250), rec.TokensOut)
	assert.Equal(t, int64(100), rec.TokensCached)
	assert.InDelta(t, 0.0035, rec.CostUSD, 0.0001)
	assert.NotNil(t, rec.EndedAt)
}

func TestGetLastSession(t *testing.T) {
	db := openTestDB(t)

	id1, _, err := db.StartSession("gpt-4o", "first")
	require.NoError(t, err)
	require.NoError(t, db.EndSession(id1, 100, 25, 0, 0.001))

	time.Sleep(10 * time.Millisecond)

	id2, _, err := db.StartSession("claude-sonnet-4-5", "second")
	require.NoError(t, err)
	require.NoError(t, db.EndSession(id2, 500, 125, 0, 0.002))

	last, err := db.GetLastSession()
	require.NoError(t, err)
	assert.Equal(t, id2, last.ID)
	assert.Equal(t, "second", last.Tag)
}

func TestListSessions(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 3; i++ {
		id, _, err := db.StartSession("gpt-4o", "")
		require.NoError(t, err)
		require.NoError(t, db.EndSession(id, 100, 25, 0, 0.001))
	}

	records, err := db.ListSessions()
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestTodaySpend(t *testing.T) {
	db := openTestDB(t)

	// No sessions yet
	spend, err := db.TodaySpend()
	require.NoError(t, err)
	assert.Equal(t, 0.0, spend.CostUSD)

	// Add a session
	id, _, err := db.StartSession("claude-sonnet-4-5", "")
	require.NoError(t, err)
	require.NoError(t, db.EndSession(id, 1000, 250, 0, 0.006))

	spend, err = db.TodaySpend()
	require.NoError(t, err)
	assert.InDelta(t, 0.006, spend.CostUSD, 0.0001)
	assert.Equal(t, int64(1250), spend.TokensTotal)
}

func TestResetDailySpend(t *testing.T) {
	db := openTestDB(t)

	id, _, err := db.StartSession("claude-sonnet-4-5", "")
	require.NoError(t, err)
	require.NoError(t, db.EndSession(id, 1000, 250, 0, 0.006))

	require.NoError(t, db.ResetDailySpend())

	spend, err := db.TodaySpend()
	require.NoError(t, err)
	assert.Equal(t, 0.0, spend.CostUSD)
}

func TestAuditLog(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")
	db, err := Open(filepath.Join(dir, "test.db"), auditPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = db.AuditLog("sensitive payload content")
	require.NoError(t, err)

	content, err := os.ReadFile(auditPath)
	require.NoError(t, err)

	// Should contain a SHA-256 hash, not the plaintext
	assert.Contains(t, string(content), "sha256=")
	assert.NotContains(t, string(content), "sensitive payload content")
}

func TestTodaySpendIncludesCached(t *testing.T) {
	db := openTestDB(t)
	id, _, err := db.StartSession("claude-sonnet-4-5", "")
	require.NoError(t, err)
	// 500 in, 100 out, 200 cached — total should be 800
	require.NoError(t, db.EndSession(id, 500, 100, 200, 0.003))

	spend, err := db.TodaySpend()
	require.NoError(t, err)
	assert.Equal(t, int64(800), spend.TokensTotal)
}

func TestListSessionsEmpty(t *testing.T) {
	db := openTestDB(t)
	records, err := db.ListSessions()
	require.NoError(t, err)
	assert.NotNil(t, records)
	assert.Len(t, records, 0)
}

func TestAuditLogHashValue(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")
	db, err := Open(filepath.Join(dir, "test.db"), auditPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	payload := "test payload for hash verification"
	require.NoError(t, db.AuditLog(payload))

	content, err := os.ReadFile(auditPath)
	require.NoError(t, err)

	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	assert.Contains(t, string(content), "sha256="+expected)
	assert.NotContains(t, string(content), payload)
}

func TestAuditLogNoPath(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"), "")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Should not error when audit log path is empty
	assert.NoError(t, db.AuditLog("test payload"))
}
