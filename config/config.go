// Package config handles loading, merging, and providing access
// to walk configuration from YAML files and environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Defaults
const (
	DefaultProxyPort  = 9010
	DefaultConfigDir  = ".walk"
	DefaultConfigFile = "config.yaml"
	DefaultCacheDir   = "cache"
	DefaultLLamaEndpoint = "http://localhost:8080"
)

// Config is the top-level configuration structure.
type Config struct {
	DefaultProvider string              `mapstructure:"default_provider" yaml:"default_provider"`
	Providers       map[string]Provider `mapstructure:"providers" yaml:"providers"`
	Cache           CacheConfig         `mapstructure:"cache" yaml:"cache"`
	Budget          BudgetConfig        `mapstructure:"budget" yaml:"budget"`
	Scrub           ScrubConfig         `mapstructure:"scrub" yaml:"scrub"`
	Proxy           ProxyConfig         `mapstructure:"proxy" yaml:"proxy"`
}

// Provider configures a single LLM adapter.
type Provider struct {
	Adapter  string `mapstructure:"adapter" yaml:"adapter"`
	Model    string `mapstructure:"model" yaml:"model"`
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint"`
	APIKey   string `mapstructure:"api_key" yaml:"api_key,omitempty"`
}

// CacheConfig controls semantic caching behaviour.
type CacheConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	Dir       string `mapstructure:"dir" yaml:"dir"`
	MaxSizeMB int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
}

// BudgetConfig controls token and cost budgets per session.
type BudgetConfig struct {
	SessionTokenLimit int     `mapstructure:"session_token_limit" yaml:"session_token_limit"`
	CostAlert         float64 `mapstructure:"cost_alert" yaml:"cost_alert"`
	Enabled           bool    `mapstructure:"enabled" yaml:"enabled"`
}

// ScrubConfig controls secret/PII scrubbing.
type ScrubConfig struct {
	Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
	Patterns []string `mapstructure:"patterns" yaml:"patterns"`
}

// ProxyConfig controls the transparent proxy.
type ProxyConfig struct {
	Port int `mapstructure:"port" yaml:"port"`
}

// Load reads config from ~/.walk/config.yaml with viper.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	configDir := filepath.Join(home, DefaultConfigDir)
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.SetEnvPrefix("WALK")
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("default_provider", "claude-code")
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.dir", filepath.Join(configDir, DefaultCacheDir))
	v.SetDefault("cache.max_size_mb", 500)
	v.SetDefault("budget.enabled", true)
	v.SetDefault("budget.session_token_limit", 100000)
	v.SetDefault("budget.cost_alert", 0.50)
	v.SetDefault("scrub.enabled", true)
	v.SetDefault("scrub.patterns", []string{
		`sk-[a-zA-Z0-9]{20,}`,
		`ghp_[a-zA-Z0-9]{36}`,
		`gho_[a-zA-Z0-9]{36}`,
		`xox[bpras]-[a-zA-Z0-9-]{24,}`,
		`AKIA[0-9A-Z]{16}`,
	})
	v.SetDefault("proxy.port", DefaultProxyPort)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("config dir: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		// Config file missing is fine — use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Expand ~ in cache dir
	if len(cfg.Cache.Dir) > 0 && cfg.Cache.Dir[0] == '~' {
		cfg.Cache.Dir = filepath.Join(home, cfg.Cache.Dir[1:])
	}

	return &cfg, nil
}

// WriteDefaultConfig writes the default config to ~/.walk/config.yaml.
func WriteDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	configDir := filepath.Join(home, DefaultConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(configDir, DefaultConfigFile)

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config already exists at %s", configPath)
	}

	defaultCfg := `# walk configuration
default_provider: claude-code

providers:
  claude-code:
    adapter: claude-code
    model: claude-sonnet-4-20250514
  codex:
    adapter: codex
    model: gpt-4o
  llama:
    adapter: llama-cpp
    endpoint: http://localhost:8080

cache:
  enabled: true
  dir: ~/.walk/cache
  max_size_mb: 500

budget:
  enabled: true
  session_token_limit: 100000
  cost_alert: 0.50

scrub:
  enabled: true
  patterns:
    - sk-[a-zA-Z0-9]{20,}
    - ghp_[a-zA-Z0-9]{36}

proxy:
  port: 9010
`
	return os.WriteFile(configPath, []byte(defaultCfg), 0644)
}