package compressor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockLlamaServer(t *testing.T, response string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			resp := chatResponse{
				Choices: []struct {
					Message message `json:"message"`
				}{
					{Message: message{Role: "assistant", Content: response}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCompressSuccess(t *testing.T) {
	compressed := "Short summary of the content."
	srv := mockLlamaServer(t, compressed)

	c := NewWithClient(srv.URL, "gemma-4-27b-q8_0", srv.Client())

	original := "This is a much longer piece of text that contains a lot of words and should be summarised into something shorter and more concise for use in token-efficient prompts."
	result, err := c.Compress(context.Background(), original)

	require.NoError(t, err)
	assert.Equal(t, original, result.Original)
	assert.Equal(t, compressed, result.Compressed)
	assert.Greater(t, result.OriginalTokens, 0)
	assert.Greater(t, result.CompressedTokens, 0)
	assert.Greater(t, result.OriginalTokens, result.CompressedTokens)
	assert.Less(t, result.CompressionRatio, 1.0)
	assert.Greater(t, result.TokensSaved, 0)
}

func TestCompressServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewWithClient(srv.URL, "gemma", srv.Client())
	_, err := c.Compress(context.Background(), "some text")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCompressNoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: nil}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewWithClient(srv.URL, "gemma", srv.Client())
	_, err := c.Compress(context.Background(), "some text")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestCompressAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "model not loaded"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewWithClient(srv.URL, "gemma", srv.Client())
	_, err := c.Compress(context.Background(), "some text")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not loaded")
}

func TestCompressCompressionRatio(t *testing.T) {
	// Compressed output is shorter than original
	original := "word " + "word " + "word " // 3 words
	compressed := "word"                     // 1 word
	srv := mockLlamaServer(t, compressed)

	c := NewWithClient(srv.URL, "gemma", srv.Client())
	result, err := c.Compress(context.Background(), original)
	require.NoError(t, err)
	assert.Less(t, result.CompressionRatio, 1.0)
}

func TestNewConstructor(t *testing.T) {
	c := New("http://localhost:8080/v1", "gemma", 10)
	assert.NotNil(t, c)
	assert.Equal(t, "http://localhost:8080/v1", c.endpoint)
	assert.Equal(t, "gemma", c.model)
}

func TestNewDefaultTimeout(t *testing.T) {
	c := New("http://localhost:8080/v1", "gemma", 0)
	assert.NotNil(t, c)
}

func TestBenchmarkCompressStub(t *testing.T) {
	result, err := BenchmarkCompress("hello world")
	require.NoError(t, err)
	assert.Greater(t, result.OriginalTokens, 0)
	assert.Equal(t, 1.0, result.CompressionRatio)
}

func BenchmarkCompressLocal(b *testing.B) {
	text := "This is a sample text for benchmarking the compression function without network calls. "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BenchmarkCompress(text)
	}
}
