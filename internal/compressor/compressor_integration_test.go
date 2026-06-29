//go:build integration

package compressor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressIntegration(t *testing.T) {
	// Requires llama.cpp running at localhost:8080
	c := New("http://localhost:8080/v1", "gemma-4-27b-q8_0", 30)
	text := "You are a helpful assistant that summarises documents. Please always cite sources. Never make up facts. Be concise and accurate. Use bullet points where appropriate."
	result, err := c.Compress(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, result.OriginalTokens, 0)
	t.Logf("Compression ratio: %.2f, saved %d tokens", result.CompressionRatio, result.TokensSaved)
}
