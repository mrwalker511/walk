package scrubber

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ValidPattern(t *testing.T) {
	s, err := New([]string{`sk-[a-zA-Z0-9]+`})
	require.NoError(t, err)
	require.NotNil(t, s)
}

func TestNew_InvalidPattern(t *testing.T) {
	_, err := New([]string{`[invalid`})
	assert.Error(t, err)
}

func TestScrub_APIKey(t *testing.T) {
	s, err := New([]string{`sk-[a-zA-Z0-9]{20,}`})
	require.NoError(t, err)

	input := "My API key is sk-proj-ABCDEF1234567890abcdef1234 and it's secret"
	result := s.ScrubText(input)

	assert.NotEqual(t, input, result.Cleaned)
	assert.NotContains(t, result.Cleaned, "sk-proj-ABCDEF1234567890abcdef1234")
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, 1, result.Findings[0].Count)
}

func TestScrub_GitHubToken(t *testing.T) {
	s, err := New([]string{`ghp_[a-zA-Z0-9]{36}`})
	require.NoError(t, err)

	input := "token=ghp_abcdefghijklmnopqrstuvwxyz0123456789abc"
	result := s.ScrubText(input)

	assert.NotContains(t, result.Cleaned, "ghp_abcdefghijklmnopqrstuvwxyz0123456789abc")
}

func TestScrub_MultipleFindings(t *testing.T) {
	s, err := New([]string{`sk-[a-zA-Z0-9]+`})
	require.NoError(t, err)

	input := "key1=sk-abc key2=sk-def key3=sk-ghi"
	result := s.ScrubText(input)

	assert.Len(t, result.Findings, 1)
	assert.Equal(t, 3, result.Findings[0].Count)
}

func TestScrub_CleanText(t *testing.T) {
	s, err := New([]string{`sk-[a-zA-Z0-9]+`})
	require.NoError(t, err)

	input := "This is a clean text with no secrets"
	result := s.ScrubText(input)

	assert.Equal(t, input, result.Cleaned)
	assert.Len(t, result.Findings, 0)
}

func TestScrub_JSONPayload(t *testing.T) {
	s, err := New([]string{`sk-[a-zA-Z0-9]+`})
	require.NoError(t, err)

	input := []byte(`{"model":"gpt-4","api_key":"sk-proj-secret123","messages":[{"role":"user","content":"hi"}]}`)
	result, err := s.ScrubJSON(input)
	require.NoError(t, err)

	assert.NotContains(t, result.Cleaned, "sk-proj-secret123")
	assert.Contains(t, result.Cleaned, "gpt-4")
	assert.Contains(t, result.Cleaned, "hi")
}

func TestScrubReport_NoFindings(t *testing.T) {
	report := ScrubReport([]Finding{})
	assert.Contains(t, report, "No secrets")
}

func TestScrubReport_WithFindings(t *testing.T) {
	report := ScrubReport([]Finding{
		{RuleName: "api-keys", Count: 2, Sample: "sk-ab...cd"},
	})
	assert.Contains(t, report, "api-keys")
	assert.Contains(t, report, "2 occurrence(s)")
}

func TestAddRule(t *testing.T) {
	s, err := New([]string{})
	require.NoError(t, err)

	err = s.AddRule("custom", `AKIA[0-9A-Z]{16}`, RedactFull)
	require.NoError(t, err)

	result := s.ScrubText("key=AKIA1234567890ABCDEF")
	assert.Contains(t, result.Cleaned, "[REDACTED]")
	assert.Len(t, result.Findings, 1)
}