package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type LocalModel struct {
	Provider       string `mapstructure:"provider"        yaml:"provider"`
	Endpoint       string `mapstructure:"endpoint"        yaml:"endpoint"`
	Model          string `mapstructure:"model"           yaml:"model"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
	Enabled        bool   `mapstructure:"enabled"         yaml:"enabled"`
}

type ProviderConfig struct {
	APIKey       string `mapstructure:"api_key"       yaml:"api_key"`
	DefaultModel string `mapstructure:"default_model" yaml:"default_model"`
}

type Providers struct {
	Anthropic ProviderConfig `mapstructure:"anthropic" yaml:"anthropic"`
	OpenAI    ProviderConfig `mapstructure:"openai"    yaml:"openai"`
}

type Budget struct {
	DailyLimit    float64 `mapstructure:"daily_limit"     yaml:"daily_limit"`
	SessionLimit  float64 `mapstructure:"session_limit"   yaml:"session_limit"`
	WarnAtPercent int     `mapstructure:"warn_at_percent" yaml:"warn_at_percent"`
	HardStop      bool    `mapstructure:"hard_stop"       yaml:"hard_stop"`
}

type Scrubber struct {
	Enabled          bool     `mapstructure:"enabled"           yaml:"enabled"`
	BlockOnDetect    bool     `mapstructure:"block_on_detect"   yaml:"block_on_detect"`
	Patterns         []string `mapstructure:"patterns"          yaml:"patterns"`
	EntropyThreshold float64  `mapstructure:"entropy_threshold" yaml:"entropy_threshold"`
}

type Cache struct {
	TrackHits         bool `mapstructure:"track_hits"         yaml:"track_hits"`
	OptimizeOnAnalyze bool `mapstructure:"optimize_on_analyze" yaml:"optimize_on_analyze"`
}

type Session struct {
	DBPath       string `mapstructure:"db_path"       yaml:"db_path"`
	AuditLog     string `mapstructure:"audit_log"     yaml:"audit_log"`
	AuditEnabled bool   `mapstructure:"audit_enabled" yaml:"audit_enabled"`
}

type Output struct {
	Color           bool   `mapstructure:"color"             yaml:"color"`
	ShowSavingsLine bool   `mapstructure:"show_savings_line" yaml:"show_savings_line"`
	DefaultFormat   string `mapstructure:"default_format"    yaml:"default_format"`
}

type Config struct {
	Walk       map[string]string `mapstructure:"walk"        yaml:"walk,omitempty"`
	LocalModel LocalModel        `mapstructure:"local_model" yaml:"local_model"`
	Providers  Providers         `mapstructure:"providers"   yaml:"providers"`
	Budget     Budget            `mapstructure:"budget"      yaml:"budget"`
	Scrubber   Scrubber          `mapstructure:"scrubber"    yaml:"scrubber"`
	Cache      Cache             `mapstructure:"cache"       yaml:"cache"`
	Session    Session           `mapstructure:"session"     yaml:"session"`
	Output     Output            `mapstructure:"output"      yaml:"output"`
}

// DefaultConfigDir returns the default config directory path.
var DefaultConfigDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".walk"
	}
	return filepath.Join(home, ".walk")
}

func setDefaults(v *viper.Viper) {
	configDir := DefaultConfigDir()

	v.SetDefault("walk.version", "1")

	v.SetDefault("local_model.provider", "llama.cpp")
	v.SetDefault("local_model.endpoint", "http://localhost:8080/v1")
	v.SetDefault("local_model.model", "gemma-4-27b-q8_0")
	v.SetDefault("local_model.timeout_seconds", 30)
	v.SetDefault("local_model.enabled", true)

	v.SetDefault("providers.anthropic.api_key", "${ANTHROPIC_API_KEY}")
	v.SetDefault("providers.anthropic.default_model", "claude-sonnet-4-5")
	v.SetDefault("providers.openai.api_key", "${OPENAI_API_KEY}")
	v.SetDefault("providers.openai.default_model", "gpt-4o")

	v.SetDefault("budget.daily_limit", 10.00)
	v.SetDefault("budget.session_limit", 2.00)
	v.SetDefault("budget.warn_at_percent", 80)
	v.SetDefault("budget.hard_stop", true)

	v.SetDefault("scrubber.enabled", true)
	v.SetDefault("scrubber.block_on_detect", true)
	v.SetDefault("scrubber.patterns", []string{"api_key", "jwt", "aws_credential", "ssh_key", "email", "ssn", "phone"})
	v.SetDefault("scrubber.entropy_threshold", 4.5)

	v.SetDefault("cache.track_hits", true)
	v.SetDefault("cache.optimize_on_analyze", true)

	v.SetDefault("session.db_path", filepath.Join(configDir, "sessions.db"))
	v.SetDefault("session.audit_log", filepath.Join(configDir, "audit.log"))
	v.SetDefault("session.audit_enabled", true)

	v.SetDefault("output.color", true)
	v.SetDefault("output.show_savings_line", true)
	v.SetDefault("output.default_format", "table")
}

// Load reads config from ~/.walk/config.yaml, applying defaults for missing keys.
func Load() (*Config, error) {
	return LoadFrom("")
}

// LoadFrom reads config from a specific directory (empty string uses default ~/.walk/).
func LoadFrom(dir string) (*Config, error) {
	if dir == "" {
		dir = DefaultConfigDir()
	}

	v := viper.New()
	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w (hint: run 'walk init' to create it)", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	ExpandVars(&cfg)
	return &cfg, nil
}

// ExpandVars resolves ${VAR_NAME} references in string fields using the environment.
func ExpandVars(cfg *Config) {
	cfg.Providers.Anthropic.APIKey = expand(cfg.Providers.Anthropic.APIKey)
	cfg.Providers.OpenAI.APIKey = expand(cfg.Providers.OpenAI.APIKey)
}

func expand(s string) string {
	return os.Expand(s, os.Getenv)
}

// EnsureDir creates ~/.walk/ if it doesn't exist.
func EnsureDir() (string, error) {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating config dir %s: %w", dir, err)
	}
	return dir, nil
}

// Write serializes cfg to dir/config.yaml using yaml.Marshal.
func Write(dir string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config to %s: %w", path, err)
	}
	return nil
}
