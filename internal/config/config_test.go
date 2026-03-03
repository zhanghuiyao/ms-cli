package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Model.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %s", cfg.Model.Provider)
	}

	if cfg.Budget.MaxTokens != 32768 {
		t.Errorf("expected max tokens 32768, got %d", cfg.Budget.MaxTokens)
	}

	if cfg.Context.MaxTokens != 24000 {
		t.Errorf("expected context max tokens 24000, got %d", cfg.Context.MaxTokens)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp config file
	content := `
model:
  provider: anthropic
  api_key: test-key
budget:
  max_tokens: 1000
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Load config
	cfg, err := LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Model.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %s", cfg.Model.Provider)
	}

	if cfg.Model.APIKey != "test-key" {
		t.Errorf("expected API key 'test-key', got %s", cfg.Model.APIKey)
	}

	if cfg.Budget.MaxTokens != 1000 {
		t.Errorf("expected max tokens 1000, got %d", cfg.Budget.MaxTokens)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing provider",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Model.Provider = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "zero max tokens",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Budget.MaxTokens = 0
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("MSCLI_MODEL_PROVIDER", "test-provider")
	os.Setenv("MSCLI_MODEL_API_KEY", "env-key")
	defer func() {
		os.Unsetenv("MSCLI_MODEL_PROVIDER")
		os.Unsetenv("MSCLI_MODEL_API_KEY")
	}()

	cfg := DefaultConfig()
	cfg.loadFromEnv()

	if cfg.Model.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got %s", cfg.Model.Provider)
	}

	if cfg.Model.APIKey != "env-key" {
		t.Errorf("expected API key 'env-key', got %s", cfg.Model.APIKey)
	}
}
