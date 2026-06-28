package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteDefaultConfig(t *testing.T) {
	// Backup and restore home
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	home, err := os.MkdirTemp("", "walk-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(home)
	os.Setenv("HOME", home)

	err = WriteDefaultConfig()
	require.NoError(t, err)

	configPath := home + "/.walk/config.yaml"
	assert.FileExists(t, configPath)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(string(data), "claude-code")
}

func TestWriteDefaultConfig_AlreadyExists(t *testing.T) {
	home, err := os.MkdirTemp("", "walk-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(home)
	os.Setenv("HOME", home)

	os.MkdirAll(home+"/.walk", 0755)
	os.WriteFile(home+"/.walk/config.yaml", []byte("existing"), 0644)

	err = WriteDefaultConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLoad_Defaults(t *testing.T) {
	home, err := os.MkdirTemp("", "walk-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(home)
	os.Setenv("HOME", home)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "claude-code", cfg.DefaultProvider)
	assert.True(t, cfg.Cache.Enabled)
	assert.True(t, cfg.Scrub.Enabled)
	assert.Equal(t, DefaultProxyPort, cfg.Proxy.Port)
}

func TestLoad_WithCustomConfig(t *testing.T) {
	home, err := os.MkdirTemp("", "walk-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(home)
	os.Setenv("HOME", home)

	// Write custom config
	configDir := home + "/.walk"
	os.MkdirAll(configDir, 0755)
	configContent := `
default_provider: codex
cache:
  enabled: false
budget:
  session_token_limit: 50000
scrub:
  enabled: false
proxy:
  port: 9090
`
	os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0644)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "codex", cfg.DefaultProvider)
	assert.False(t, cfg.Cache.Enabled)
	assert.False(t, cfg.Scrub.Enabled)
	assert.Equal(t, 50000, cfg.Budget.SessionTokenLimit)
	assert.Equal(t, 9090, cfg.Proxy.Port)
}