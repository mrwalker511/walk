package compressor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient("http://localhost:8080")
	assert.Equal(t, "http://localhost:8080", c.Endpoint)
}

func TestTrimHistory_UnderLimit(t *testing.T) {
	history := []string{"short", "messages"}
	result, err := TrimHistory(history, 100000)
	assert.NoError(t, err)
	assert.Equal(t, history, result)
}

func TestTrimHistory_OverLimit(t *testing.T) {
	// Create a long message
	longMsg := ""
	for i := 0; i < 1000; i++ {
		longMsg += "this is a long message with lots of tokens to force trimming "
	}

	history := []string{longMsg, "short final message"}
	result, err := TrimHistory(history, 50)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(result), len(history))
}

func TestCompressInline_ShortText(t *testing.T) {
	result := compressInline("short text")
	assert.Equal(t, "short text", result)
}

func TestCompressInline_LongText(t *testing.T) {
	longText := ""
	for i := 0; i < 200; i++ {
		longText += "this is a long sentence that repeats to create a long text block. "
	}

	result := compressInline(longText)
	assert.Less(t, len(result), len(longText))
}

func TestDefaultCompressPrompt(t *testing.T) {
	assert.Contains(t, DefaultCompressPrompt, "compression")
	assert.Contains(t, DefaultCompressPrompt, "preserves")
}