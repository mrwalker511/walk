package scrubber

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrubAPIKey(t *testing.T) {
	text := "Use the key sk-abc123DEF456ghi789JKL012mno to authenticate"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, "sk-abc123DEF456ghi789JKL012mno")
	assert.Contains(t, result.Clean, "[REDACTED:API_KEY]")
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, TypeAPIKey, result.Findings[0].Type)
}

func TestScrubJWT(t *testing.T) {
	// Fake JWT structure (not a real token)
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	text := "Authorization: Bearer " + jwt
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, jwt)
	assert.Contains(t, result.Clean, "[REDACTED:JWT]")
}

func TestScrubAWSKey(t *testing.T) {
	text := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, "AKIAIOSFODNN7EXAMPLE")
	assert.Contains(t, result.Clean, "[REDACTED:AWS_KEY]")
}

func TestScrubEmail(t *testing.T) {
	text := "Contact user@example.com for support"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, "user@example.com")
	assert.Contains(t, result.Clean, "[REDACTED:EMAIL]")
}

func TestScrubSSN(t *testing.T) {
	text := "SSN: 123-45-6789"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, "123-45-6789")
	assert.Contains(t, result.Clean, "[REDACTED:SSN]")
}

func TestScrubPhone(t *testing.T) {
	text := "Call me at 555-867-5309"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, "555-867-5309")
	assert.Contains(t, result.Clean, "[REDACTED:PHONE]")
}

func TestScrubSSHKey(t *testing.T) {
	text := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA1234567890abcdef\n-----END RSA PRIVATE KEY-----"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	assert.NotContains(t, result.Clean, "BEGIN RSA PRIVATE KEY")
	assert.Contains(t, result.Clean, "[REDACTED:SSH_PRIVATE_KEY]")
}

func TestScrubClean(t *testing.T) {
	text := "This is a normal message with no secrets."
	result := Scrub(text, 100) // very high entropy threshold to suppress entropy scan

	assert.False(t, result.HasSecrets)
	assert.Equal(t, text, result.Clean)
	assert.Empty(t, result.Findings)
}

func TestScrubMultiple(t *testing.T) {
	text := "Email: admin@corp.com\nKey: sk-testABCDEFGHIJKLMNOPQRSTU\nSSN: 987-65-4321"
	result := Scrub(text, 100)

	require.True(t, result.HasSecrets)
	assert.GreaterOrEqual(t, len(result.Findings), 3)
	assert.NotContains(t, result.Clean, "admin@corp.com")
	assert.NotContains(t, result.Clean, "sk-testABCDEFGHIJKLMNOPQRSTU")
	assert.NotContains(t, result.Clean, "987-65-4321")
}

func TestScrubLineNumbers(t *testing.T) {
	lines := []string{
		"line 1 is normal",
		"line 2 has email admin@test.com here",
		"line 3 is normal",
	}
	text := strings.Join(lines, "\n")
	result := Scrub(text, 100)

	require.True(t, result.HasSecrets)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, 2, result.Findings[0].Line)
}

func TestShannonEntropy(t *testing.T) {
	// Low entropy (repeated chars)
	assert.Less(t, shannonEntropy("aaaaaaaaaa"), 1.0)
	// Medium entropy (English word)
	assert.Greater(t, shannonEntropy("hello"), 1.0)
	// High entropy (random-looking string)
	assert.Greater(t, shannonEntropy("aB3kQ9mNpR2xZvLw"), 3.5)
}

func TestScrubEntropyThreshold(t *testing.T) {
	// High-entropy token that looks like a secret
	highEntropy := "aB3kQ9mNpR2xZvLwYcDeFgHiJk"
	text := "token=" + highEntropy
	result := Scrub(text, 3.0) // low threshold to catch it

	assert.True(t, result.HasSecrets)
}

func TestScrubAnthropicKey(t *testing.T) {
	// ant[A-Za-z0-9_-]{30,} pattern — Anthropic-style key
	text := "key: antABCDEFGHIJKLMNOPQRSTUVWXYZabcd"
	result := Scrub(text, 0)

	require.True(t, result.HasSecrets)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, TypeAPIKey, result.Findings[0].Type)
	assert.NotContains(t, result.Clean, "antABCDEFGHIJKLMNOPQRSTUVWXYZabcd")
	assert.Contains(t, result.Clean, "[REDACTED:API_KEY]")
}

func TestScrubEntropyFindingType(t *testing.T) {
	// Verify that a high-entropy word produces a TypeHighEntropy finding
	highEntropy := "aB3kQ9mNpR2xZvLwYcDeFgHiJk"
	text := "token=" + highEntropy
	result := Scrub(text, 3.0)

	require.True(t, result.HasSecrets)
	types := make([]FindingType, 0, len(result.Findings))
	for _, f := range result.Findings {
		types = append(types, f.Type)
	}
	assert.Contains(t, types, TypeHighEntropy)
}

func TestScrubEntropyThresholdFallback(t *testing.T) {
	// entropyThreshold <= 0 should default to 4.5 — normal short words must not trigger it
	result := Scrub("hello world this is normal text", 0)
	assert.False(t, result.HasSecrets)
	assert.Empty(t, result.Findings)
}

func TestScrubRedactedField(t *testing.T) {
	// Finding.Redacted must hold the replacement string that appears in Clean
	result := Scrub("key: sk-abcdefghijklmnopqrstuvwxyz012345", 0)

	require.True(t, result.HasSecrets)
	require.NotEmpty(t, result.Findings)
	assert.NotEmpty(t, result.Findings[0].Redacted)
	assert.Contains(t, result.Clean, result.Findings[0].Redacted)
}
