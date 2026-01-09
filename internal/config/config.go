package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/xqsit94/shelp/pkg/paths"
	"golang.org/x/term"
)

type Config struct {
	AIURL string `json:"ai_url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

func Load() (*Config, error) {
	configPath := paths.GetConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := paths.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	configPath := paths.GetConfigPath()
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (c *Config) IsConfigured() bool {
	return c.AIURL != "" && c.APIKey != "" && c.Model != ""
}

func (c *Config) MaskedAPIKey() string {
	if len(c.APIKey) <= 8 {
		return strings.Repeat("*", len(c.APIKey))
	}
	return c.APIKey[:4] + strings.Repeat("*", len(c.APIKey)-8) + c.APIKey[len(c.APIKey)-4:]
}

func PromptForAPIKey() (string, error) {
	fmt.Print("Enter API Key: ")
	keyBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %v", err)
	}
	fmt.Println()
	return strings.TrimSpace(string(keyBytes)), nil
}

func Reset() error {
	configPath := paths.GetConfigPath()
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %v", err)
	}
	return nil
}
