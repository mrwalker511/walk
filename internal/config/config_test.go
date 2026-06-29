package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFrom(dir)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8080/v1", cfg.LocalModel.Endpoint)
	assert.True(t, cfg.LocalModel.Enabled)
	assert.Equal(t, 30, cfg.LocalModel.TimeoutSeconds)

	assert.Equal(t, "claude-sonnet-4-5", cfg.Providers.Anthropic.DefaultModel)
	assert.Equal(t, "gpt-4o", cfg.Providers.OpenAI.DefaultModel)

	assert.Equal(t, 10.00, cfg.Budget.DailyLimit)
	assert.Equal(t, 2.00, cfg.Budget.SessionLimit)
	assert.Equal(t, 80, cfg.Budget.WarnAtPercent)
	assert.True(t, cfg.Budget.HardStop)

	assert.True(t, cfg.Scrubber.Enabled)
	assert.True(t, cfg.Scrubber.BlockOnDetect)
	assert.Equal(t, 4.5, cfg.Scrubber.EntropyThreshold)
	assert.Contains(t, cfg.Scrubber.Patterns, "api_key")
	assert.Contains(t, cfg.Scrubber.Patterns, "jwt")

	assert.Equal(t, "table", cfg.Output.DefaultFormat)
	assert.True(t, cfg.Output.ShowSavingsLine)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	yaml := `
budget:
  daily_limit: 5.00
  session_limit: 1.00
  warn_at_percent: 75
  hard_stop: false
output:
  default_format: json
  color: false
`
	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0600)
	require.NoError(t, err)

	cfg, err := LoadFrom(dir)
	require.NoError(t, err)

	assert.Equal(t, 5.00, cfg.Budget.DailyLimit)
	assert.Equal(t, 1.00, cfg.Budget.SessionLimit)
	assert.Equal(t, 75, cfg.Budget.WarnAtPercent)
	assert.False(t, cfg.Budget.HardStop)
	assert.Equal(t, "json", cfg.Output.DefaultFormat)
	assert.False(t, cfg.Output.Color)
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		LocalModel: LocalModel{
			Provider:       "llama.cpp",
			Endpoint:       "http://localhost:8080/v1",
			Model:          "gemma-4-27b-q8_0",
			TimeoutSeconds: 30,
			Enabled:        true,
		},
		Budget: Budget{
			DailyLimit:    7.50,
			SessionLimit:  1.50,
			WarnAtPercent: 80,
			HardStop:      true,
		},
		Scrubber: Scrubber{
			Enabled:          true,
			BlockOnDetect:    true,
			Patterns:         []string{"api_key", "jwt"},
			EntropyThreshold: 4.5,
		},
		Output: Output{
			Color:          true,
			ShowSavingsLine: true,
			DefaultFormat:  "table",
		},
	}

	err := Write(dir, cfg)
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(dir, "config.yaml"))
	assert.NoError(t, statErr)

	loaded, err := LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, 7.50, loaded.Budget.DailyLimit)
}

func TestExpandVars(t *testing.T) {
	t.Setenv("WALK_TEST_KEY", "sk-real-value")

	cfg := &Config{}
	cfg.Providers.Anthropic.APIKey = "${WALK_TEST_KEY}"
	cfg.Providers.OpenAI.APIKey = "${UNSET_VAR}"

	ExpandVars(cfg)

	assert.Equal(t, "sk-real-value", cfg.Providers.Anthropic.APIKey)
	assert.Equal(t, "", cfg.Providers.OpenAI.APIKey)
}

func TestLoadFromExpandsVars(t *testing.T) {
	t.Setenv("WALK_TEST_ANTHROPIC", "sk-expanded")
	dir := t.TempDir()
	yamlContent := `providers:
  anthropic:
    api_key: "${WALK_TEST_ANTHROPIC}"
`
	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yamlContent), 0600)
	require.NoError(t, err)

	cfg, err := LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "sk-expanded", cfg.Providers.Anthropic.APIKey)
}

func TestEnsureDir(t *testing.T) {
	orig := DefaultConfigDir
	defer func() { DefaultConfigDir = orig }()

	dir := t.TempDir()
	target := filepath.Join(dir, "newdir")
	DefaultConfigDir = func() string { return target }

	got, err := EnsureDir()
	require.NoError(t, err)
	assert.Equal(t, target, got)

	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
