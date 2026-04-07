package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Model     string
	BaseURL   string
	APIKey    string
	MaxTokens int
}

// Load returns a Config with values from config file, overridden by any non-zero flag values.
// Config file path: $LOG_INSIGHT_CONFIG env var, or ~/.kube/log-insight.yaml
func Load(model, baseURL, apiKey string, maxTokens int) Config {
	v := viper.New()
	v.SetDefault("base-url", "http://localhost:1234/v1")
	v.SetDefault("api-key", "test")
	v.SetDefault("max-tokens", 8192)

	cfgPath := os.Getenv("LOG_INSIGHT_CONFIG")
	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
	} else {
		home, _ := os.UserHomeDir()
		v.SetConfigFile(home + "/.kube/log-insight.yaml")
	}
	v.ReadInConfig() // ignore error — config file is optional

	cfg := Config{
		Model:     v.GetString("model"),
		BaseURL:   v.GetString("base-url"),
		APIKey:    v.GetString("api-key"),
		MaxTokens: v.GetInt("max-tokens"),
	}

	if model != "" {
		cfg.Model = model
	}
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if maxTokens != 0 {
		cfg.MaxTokens = maxTokens
	}

	return cfg
}
