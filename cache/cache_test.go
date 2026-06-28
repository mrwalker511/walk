package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempCache(t *testing.T) *Cache {
	t.Helper()
	dir, err := os.MkdirTemp("", "walk-cache-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	c, err := New(dir, 100)
	require.NoError(t, err)
	return c
}

func TestNew_CreatesDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "walk-cache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cacheDir := filepath.Join(dir, "subdir")
	c, err := New(cacheDir, 100)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.DirExists(t, cacheDir)
}

func TestHashContent_Deterministic(t *testing.T) {
	h1 := HashContent("hello world")
	h2 := HashContent("hello world")
	assert.Equal(t, h1, h2)

	h3 := HashContent("hello world!")
	assert.NotEqual(t, h1, h3)
}

func TestSetGet(t *testing.T) {
	c := tempCache(t)

	hash := HashContent("test prompt")
	entry := &Entry{
		PromptHash:   hash,
		Response:     "test response",
		InputTokens:  50,
		OutputTokens: 20,
		Provider:     "openai",
		Model:        "gpt-4o",
	}

	err := c.Set(hash, entry)
	require.NoError(t, err)

	retrieved, ok := c.Get(hash)
	assert.True(t, ok)
	require.NotNil(t, retrieved)
	assert.Equal(t, "test response", retrieved.Response)
}

func TestGet_Miss(t *testing.T) {
	c := tempCache(t)

	_, ok := c.Get("nonexistent-hash")
	assert.False(t, ok)
}

func TestSetGetChunk(t *testing.T) {
	c := tempCache(t)

	chunk, err := c.SetChunk("repeated context block", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, chunk.Hash)

	retrieved, ok := c.GetChunk(chunk.Hash)
	assert.True(t, ok)
	require.NotNil(t, retrieved)
	assert.Equal(t, "repeated context block", retrieved.Content)
}

func TestStats_Empty(t *testing.T) {
	c := tempCache(t)
	stats := c.Stats()
	assert.Equal(t, 0, stats.Entries)
	assert.Equal(t, 0, stats.Chunks)
}

func TestStats_AfterSet(t *testing.T) {
	c := tempCache(t)

	_ = c.Set("hash1", &Entry{PromptHash: "hash1"})
	_ = c.Set("hash2", &Entry{PromptHash: "hash2"})
	_, _ = c.SetChunk("chunk1", 5)

	stats := c.Stats()
	assert.Equal(t, 2, stats.Entries)
	assert.Equal(t, 1, stats.Chunks)
}

func TestClear(t *testing.T) {
	c := tempCache(t)

	_ = c.Set("hash1", &Entry{PromptHash: "hash1"})
	_ = c.Set("hash2", &Entry{PromptHash: "hash2"})

	err := c.Clear()
	require.NoError(t, err)

	stats := c.Stats()
	assert.Equal(t, 0, stats.Entries)
}

func TestFindSimilarChunks(t *testing.T) {
	c := tempCache(t)

	_, _ = c.SetChunk("The quick brown fox jumps over the lazy dog", 10)
	_, _ = c.SetChunk("Completely unrelated content about weather", 8)

	results := c.FindSimilarChunks("quick brown fox", 0.3)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestJaccardSimilarity(t *testing.T) {
	assert.InDelta(t, 1.0, jaccardSimilarity("a b c", "a b c"), 0.01)
	assert.InDelta(t, 0.0, jaccardSimilarity("a b c", "d e f"), 0.01)
	assert.InDelta(t, 0.5, jaccardSimilarity("a b c", "a b d"), 0.01)
}

func TestEvictLRU(t *testing.T) {
	c := tempCache(t)

	// Add enough entries to trigger eviction
	for i := 0; i < 11000; i++ {
		hash := HashContent(string(rune(i)))
		_ = c.Set(hash, &Entry{PromptHash: hash})
	}

	stats := c.Stats()
	assert.LessOrEqual(t, stats.Entries, 10000)
}