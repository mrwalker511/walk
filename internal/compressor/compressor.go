package compressor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mrwalker511/walk/internal/tokenizer"
)

// Result holds the output of a compression operation.
type Result struct {
	Original        string
	Compressed      string
	OriginalTokens  int
	CompressedTokens int
	CompressionRatio float64 // < 1.0 means smaller
	TokensSaved     int
	Model           string
}

// Client is an HTTP client interface for testability.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// Compressor sends content to a llama.cpp server for summarisation.
type Compressor struct {
	endpoint string
	model    string
	client   Client
}

// chatRequest is the OpenAI-compatible request body.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// New creates a Compressor targeting the llama.cpp endpoint.
func New(endpoint, model string, timeoutSecs int) *Compressor {
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}
	return &Compressor{
		endpoint: endpoint,
		model:    model,
		client:   &http.Client{Timeout: time.Duration(timeoutSecs) * time.Second},
	}
}

// NewWithClient creates a Compressor with a custom HTTP client (for testing).
func NewWithClient(endpoint, model string, client Client) *Compressor {
	return &Compressor{endpoint: endpoint, model: model, client: client}
}

// Compress sends text to llama.cpp for summarisation and returns a Result.
func (c *Compressor) Compress(ctx context.Context, text string) (Result, error) {
	originalTokens := tokenizer.Count(text)

	prompt := fmt.Sprintf(
		"Summarise the following content as concisely as possible, preserving all key information and instructions. Remove redundancy and filler. Output only the summarised content, no preamble.\n\n---\n%s\n---",
		text,
	)

	reqBody := chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: originalTokens, // cap output at input size
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Result{}, fmt.Errorf("marshalling request: %w", err)
	}

	url := c.endpoint + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("calling llama.cpp at %s: %w (hint: start with 'llama-server --model /path/to/model.gguf --port 8080')", url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("llama.cpp returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return Result{}, fmt.Errorf("parsing response: %w", err)
	}

	if chatResp.Error != nil {
		return Result{}, fmt.Errorf("llama.cpp error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return Result{}, fmt.Errorf("llama.cpp returned no choices")
	}

	compressed := chatResp.Choices[0].Message.Content
	compressedTokens := tokenizer.Count(compressed)
	ratio := 1.0
	if originalTokens > 0 {
		ratio = float64(compressedTokens) / float64(originalTokens)
	}

	return Result{
		Original:         text,
		Compressed:       compressed,
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		CompressionRatio: ratio,
		TokensSaved:      originalTokens - compressedTokens,
		Model:            c.model,
	}, nil
}

// BenchmarkCompress is exported for use by go test -bench.
func BenchmarkCompress(text string) (Result, error) {
	// Stub for benchmark harness — real benchmarks use NewWithClient with a mock.
	return Result{
		Original:         text,
		Compressed:       text,
		OriginalTokens:   tokenizer.Count(text),
		CompressedTokens: tokenizer.Count(text),
		CompressionRatio: 1.0,
	}, nil
}
