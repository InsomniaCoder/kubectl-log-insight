package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/InsomniaCoder/kubectl-insight-logs/internal/config"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := config.Load("", "", "", 0)
	if cfg.BaseURL != "http://localhost:1234/v1" {
		t.Errorf("expected default base-url, got %s", cfg.BaseURL)
	}
	if cfg.APIKey != "test" {
		t.Errorf("expected default api-key, got %s", cfg.APIKey)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("expected default max-tokens 8192, got %d", cfg.MaxTokens)
	}
}

func TestLoadConfig_FlagsOverrideDefaults(t *testing.T) {
	cfg := config.Load("mymodel", "http://custom:8080/v1", "mykey", 2048)
	if cfg.Model != "mymodel" {
		t.Errorf("expected model mymodel, got %s", cfg.Model)
	}
	if cfg.BaseURL != "http://custom:8080/v1" {
		t.Errorf("expected custom base-url, got %s", cfg.BaseURL)
	}
	if cfg.MaxTokens != 2048 {
		t.Errorf("expected 2048, got %d", cfg.MaxTokens)
	}
}

func TestLoadConfig_FileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "log-insight.yaml")
	os.WriteFile(cfgFile, []byte("base-url: http://from-file:9999/v1\napi-key: filekey\nmax-tokens: 1024\n"), 0644)
	os.Setenv("LOG_INSIGHT_CONFIG", cfgFile)
	defer os.Unsetenv("LOG_INSIGHT_CONFIG")

	cfg := config.Load("", "", "", 0)
	if cfg.BaseURL != "http://from-file:9999/v1" {
		t.Errorf("expected base-url from file, got %s", cfg.BaseURL)
	}
	if cfg.MaxTokens != 1024 {
		t.Errorf("expected 1024 from file, got %d", cfg.MaxTokens)
	}
}
