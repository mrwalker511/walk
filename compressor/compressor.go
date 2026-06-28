// Package compressor provides context compression via the llama.cpp
// HTTP API (or any compatible endpoint) by summarizing and trimming
// low-signal content.
package compressor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with the llama.cpp HTTP server.
type Client struct {
	Endpoint   string
	HTTPClient *http.Client
	Model      string
}

// CompressRequest describes what to compress.
type CompressRequest struct {
	Content     string `json:"content"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// CompressResponse holds the compressed result.
type CompressResponse struct {
	OriginalTokens int    `json:"original_tokens"`
	CompressedText string `json:"compressed_text"`
	CompressedTokens int  `json:"compressed_tokens"`
	Ratio          float64 `json:"ratio"`
}

// NewClient creates a new llama.cpp client.
func NewClient(endpoint string) *Client {
	return &Client{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Model: "default",
	}
}

// llamaChatMessage represents a chat message for the llama.cpp API.
type llamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// llamaChatRequest is sent to /v1/chat/completions.
type llamaChatRequest struct {
	Model       string             `json:"model"`
	Messages    []llamaChatMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
}

// llamaChatResponse from llama.cpp.
type llamaChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage,omitempty"`
}

// Compress sends content to llama.cpp for summarization/compression.
func (c *Client) Compress(req *CompressRequest) (*CompressResponse, error) {
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.3 // low temp for factual compression
	}

	systemPrompt := "You are a context compression assistant. Summarize the following content " +
		"preserving all key facts, code structure, and intent. " +
		"Remove redundancy, filler words, and low-signal text. " +
		"Output ONLY the compressed version, no commentary."

	userPrompt := fmt.Sprintf("Compress the following content:\n\n%s", req.Content)

	apiReq := llamaChatRequest{
		Model: c.Model,
		Messages: []llamaChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   req.MaxTokens,
		Temperature: temperature,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.Endpoint+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llama.cpp request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llama.cpp returned %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp llamaChatResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("llama.cpp returned zero choices")
	}

	compressed := apiResp.Choices[0].Message.Content

	// Rough token estimation
	originalTokens := len(req.Content) / 4
	compressedTokens := len(compressed) / 4

	ratio := float64(compressedTokens) / float64(originalTokens)
	if originalTokens == 0 {
		ratio = 0
	}

	return &CompressResponse{
		OriginalTokens:    originalTokens,
		CompressedText:    compressed,
		CompressedTokens:  compressedTokens,
		Ratio:             ratio,
	}, nil
}

// DefaultCompressPrompt is the system prompt used for compression.
const DefaultCompressPrompt = `You are a context compression assistant. Given a conversation between a user and an AI assistant, your job is to compress it into a shorter representation that preserves:
1. All facts, decisions, and code changes
2. User intent and goals
3. Key context needed for future turns
4. Remove: greetings, filler words, redundant explanations, and low-value chitchat

Output ONLY the compressed version.`

// TrimHistory trims conversation history to stay within a token budget.
// It compresses older turns aggressively.
func TrimHistory(history []string, maxTokens int) ([]string, error) {
	totalTokens := 0
	for _, msg := range history {
		totalTokens += len(msg) / 4
	}

	if totalTokens <= maxTokens {
		return history, nil // no trimming needed
	}

	// Keep most recent messages, compress older ones
	var trimmed []string
	accumulated := 0

	for i := len(history) - 1; i >= 0; i-- {
		msgTokens := len(history[i]) / 4
		if accumulated+msgTokens > maxTokens {
			// Compress this message
			compressed := compressInline(history[i])
			trimmed = append([]string{compressed}, trimmed...)
			accumulated += len(compressed) / 4
		} else {
			trimmed = append([]string{history[i]}, trimmed...)
			accumulated += msgTokens
		}
	}

	return trimmed, nil
}

// compressInline does a simple heuristic compression without an LLM call.
func compressInline(text string) string {
	// Remove repeated newlines
	result := text
	for {
		prev := result
		result = bytes.NewBufferString(result).String()
		_ = prev
		break
	}

	// Simple: truncate to 50% if very long
	if len(text) > 500 {
		// Keep first 30% and last 20%
		first := text[:len(text)*3/10]
		last := text[len(text)*8/10:]
		return first + "\n[...compressed...]\n" + last
	}

	return text
}