package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Model       ModelConfig       `yaml:"model"`
	Budget      BudgetConfig      `yaml:"budget"`
	UI          UIConfig          `yaml:"ui"`
	Permissions PermissionConfig  `yaml:"permissions"`
	Context     ContextConfig     `yaml:"context"`
	Memory      MemoryConfig      `yaml:"memory"`
}

// ModelConfig holds LLM provider settings.
type ModelConfig struct {
	Provider    string  `yaml:"provider"`
	Endpoint    string  `yaml:"endpoint"`
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

// BudgetConfig holds token and cost limits.
type BudgetConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	MaxCostUSD  float64 `yaml:"max_cost_usd"`
}

// UIConfig holds UI settings.
type UIConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Theme         string `yaml:"theme"`
	ShowTokens    bool   `yaml:"show_tokens"`
}

// PermissionConfig holds permission settings.
type PermissionConfig struct {
	SkipRequests  bool     `yaml:"skip_requests"`
	AllowedTools  []string `yaml:"allowed_tools"`
	BlockedTools  []string `yaml:"blocked_tools"`
}

// ContextConfig holds context management settings.
type ContextConfig struct {
	MaxTokens          int     `yaml:"max_tokens"`
	CompactionThreshold float64 `yaml:"compaction_threshold"`
	ReserveTokens      int     `yaml:"reserve_tokens"`
}

// MemoryConfig holds memory persistence settings.
type MemoryConfig struct {
	MaxItems  int    `yaml:"max_items"`
	MaxBytes  int    `yaml:"max_bytes"`
	TTLHours  int    `yaml:"ttl_hours"`
	StorePath string `yaml:"store_path"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			Provider:    "openai",
			Endpoint:    "https://api.openai.com/v1",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		Budget: BudgetConfig{
			MaxTokens:  32768,
			MaxCostUSD: 10.0,
		},
		UI: UIConfig{
			Enabled:    true,
			Theme:      "default",
			ShowTokens: true,
		},
		Permissions: PermissionConfig{
			SkipRequests: false,
			AllowedTools: []string{},
			BlockedTools: []string{},
		},
		Context: ContextConfig{
			MaxTokens:           24000,
			CompactionThreshold: 0.85,
			ReserveTokens:       2000,
		},
		Memory: MemoryConfig{
			MaxItems:  200,
			MaxBytes:  2097152,
			TTLHours:  168,
			StorePath: ".cache/ms-cli/memory.json",
		},
	}
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	// Override with environment variables
	cfg.loadFromEnv()

	return cfg, nil
}

// loadFromEnv overrides config with environment variables.
func (c *Config) loadFromEnv() {
	if v := os.Getenv("MSCLI_MODEL_PROVIDER"); v != "" {
		c.Model.Provider = v
	}
	if v := os.Getenv("MSCLI_MODEL_ENDPOINT"); v != "" {
		c.Model.Endpoint = v
	}
	if v := os.Getenv("MSCLI_MODEL_API_KEY"); v != "" {
		c.Model.APIKey = v
	}
	if v := os.Getenv("MSCLI_MODEL_NAME"); v != "" {
		c.Model.Model = v
	}
	if v := os.Getenv("MSCLI_MEMORY_STORE_PATH"); v != "" {
		c.Memory.StorePath = v
	}
}

// FindConfig searches for config file in common locations.
func FindConfig() (*Config, error) {
	// Try locations in order of priority
	locations := []string{
		os.Getenv("MSCLI_CONFIG"), // Explicit env var
		"mscli.yaml",
		"configs/mscli.yaml",
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		locations = append(locations, filepath.Join(home, ".config", "ms-cli", "config.yaml"))
	}

	for _, path := range locations {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return LoadFromFile(path)
		}
	}

	// Return default config if no file found
	cfg := DefaultConfig()
	cfg.loadFromEnv()
	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Model.Provider == "" {
		return fmt.Errorf("model provider is required")
	}
	if c.Budget.MaxTokens <= 0 {
		return fmt.Errorf("budget.max_tokens must be positive")
	}
	if c.Context.MaxTokens <= 0 {
		return fmt.Errorf("context.max_tokens must be positive")
	}
	return nil
}
