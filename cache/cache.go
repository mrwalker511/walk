// Package cache provides semantic chunk caching for LLM payloads.
// It stores and retrieves prompt chunks based on content hashes,
// reducing redundant LLM calls for repeated or similar contexts.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Chunk represents a cached prompt chunk.
type Chunk struct {
	Hash      string    `json:"hash"`
	Content   string    `json:"content"`
	Tokens    int       `json:"tokens"`
	CreatedAt time.Time `json:"created_at"`
	HitCount  int       `json:"hit_count"`
	LastUsed  time.Time `json:"last_used"`
	TTL       time.Duration `json:"ttl,omitempty"`
}

// Entry is a cache entry storing a compressed response for a prompt.
type Entry struct {
	PromptHash    string    `json:"prompt_hash"`
	Response      string    `json:"response"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	CreatedAt     time.Time `json:"created_at"`
	HitCount      int       `json:"hit_count"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
}

// Cache manages semantic caching on disk.
type Cache struct {
	dir         string
	maxSizeMB   int
	mu          sync.RWMutex
	entries     map[string]*Entry
	chunks      map[string]*Chunk
}

// New creates a new file-based cache.
func New(cacheDir string, maxSizeMB int) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("cache dir: %w", err)
	}

	c := &Cache{
		dir:       cacheDir,
		maxSizeMB: maxSizeMB,
		entries:   make(map[string]*Entry),
		chunks:    make(map[string]*Chunk),
	}

	// Load existing index
	if err := c.loadIndex(); err != nil {
		// Index missing is fine
		_ = err
	}

	return c, nil
}

// HashContent creates a SHA-256 hash of content.
func HashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// indexFile returns the path to the cache index file.
func (c *Cache) indexFile() string {
	return filepath.Join(c.dir, "index.json")
}

// loadIndex reads the cache index from disk.
func (c *Cache) loadIndex() error {
	data, err := os.ReadFile(c.indexFile())
	if err != nil {
		return err
	}

	var idx struct {
		Entries map[string]*Entry `json:"entries"`
		Chunks  map[string]*Chunk `json:"chunks"`
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		return err
	}

	if idx.Entries != nil {
		c.entries = idx.Entries
	}
	if idx.Chunks != nil {
		c.chunks = idx.Chunks
	}
	return nil
}

// saveIndex writes the cache index to disk.
func (c *Cache) saveIndex() error {
	idx := struct {
		Entries map[string]*Entry `json:"entries"`
		Chunks  map[string]*Chunk `json:"chunks"`
	}{
		Entries: c.entries,
		Chunks:  c.chunks,
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.indexFile(), data, 0644)
}

// Get retrieves a cached entry by prompt hash.
func (c *Cache) Get(promptHash string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[promptHash]
	if ok {
		entry.HitCount++
		entry.CreatedAt = time.Now()
	}
	return entry, ok
}

// Set stores an entry in the cache.
func (c *Cache) Set(promptHash string, entry *Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry.CreatedAt = time.Now()
	c.entries[promptHash] = entry

	// Enforce size limit lazily
	if len(c.entries) > 10000 {
		c.evictLRU()
	}

	return c.saveIndex()
}

// GetChunk retrieves a cached chunk by hash.
func (c *Cache) GetChunk(hash string) (*Chunk, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chunk, ok := c.chunks[hash]
	if ok {
		chunk.HitCount++
		chunk.LastUsed = time.Now()
	}
	return chunk, ok
}

// SetChunk stores a chunk in the cache.
func (c *Cache) SetChunk(content string, tokens int) (*Chunk, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := HashContent(content)
	chunk := &Chunk{
		Hash:      hash,
		Content:   content,
		Tokens:    tokens,
		CreatedAt: time.Now(),
		HitCount:  1,
		LastUsed:  time.Now(),
	}
	c.chunks[hash] = chunk

	_ = c.saveIndex()
	return chunk, nil
}

// FindSimilarChunks returns cached chunks that overlap with the given content.
// Uses content-based similarity (substring matching).
func (c *Cache) FindSimilarChunks(content string, minSimilarity float64) []*Chunk {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*Chunk
	contentLower := strings.ToLower(content)

	for _, chunk := range c.chunks {
		chunkLower := strings.ToLower(chunk.Content)
		similarity := jaccardSimilarity(contentLower, chunkLower)
		if similarity >= minSimilarity {
			results = append(results, chunk)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].HitCount > results[j].HitCount
	})

	if len(results) > 10 {
		results = results[:10]
	}

	return results
}

// Stats returns cache statistics.
type Stats struct {
	Entries      int `json:"entries"`
	Chunks       int `json:"chunks"`
	TotalHits    int `json:"total_hits"`
	EntriesSize  int64 `json:"entries_size"`
}

// Stats returns current cache statistics.
func (c *Cache) Stats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := 0
	for _, e := range c.entries {
		totalHits += e.HitCount
	}
	for _, ch := range c.chunks {
		totalHits += ch.HitCount
	}

	return &Stats{
		Entries:   len(c.entries),
		Chunks:    len(c.chunks),
		TotalHits: totalHits,
	}
}

// Clear removes all cache entries.
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*Entry)
	c.chunks = make(map[string]*Chunk)

	// Remove all cached files
	entries, _ := os.ReadDir(c.dir)
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != "index.json" {
			os.Remove(filepath.Join(c.dir, entry.Name()))
		}
	}

	return c.saveIndex()
}

// evictLRU removes the least recently used entries.
func (c *Cache) evictLRU() {
	type kv struct {
		key   string
		time  time.Time
	}

	var items []kv
	for k, v := range c.entries {
		items = append(items, kv{k, v.CreatedAt})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].time.Before(items[j].time)
	})

	// Remove oldest 20%
	remove := len(items) / 5
	for i := 0; i < remove && i < len(items); i++ {
		delete(c.entries, items[i].key)
	}
}

// jaccardSimilarity computes word-level Jaccard similarity.
func jaccardSimilarity(a, b string) float64 {
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}

	setA := make(map[string]struct{}, len(wordsA))
	for _, w := range wordsA {
		setA[w] = struct{}{}
	}

	setB := make(map[string]struct{}, len(wordsB))
	for _, w := range wordsB {
		setB[w] = struct{}{}
	}

	intersection := 0
	for w := range setA {
		if _, ok := setB[w]; ok {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}