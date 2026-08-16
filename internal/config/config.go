package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/xqsit94/shelp/pkg/paths"
	"golang.org/x/term"
)

const (
	EnvURL    = "SHELP_URL"
	EnvAPIKey = "SHELP_API_KEY"
	EnvModel  = "SHELP_MODEL"
)

// Sources records which fields came from the environment rather than the file.
type Sources struct {
	AIURL  bool
	APIKey bool
	Model  bool
}

type Config struct {
	AIURL  string `json:"ai_url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"`

	FromEnv Sources `json:"-"`
}

// Load reads the config file and applies the SHELP_* environment overrides on
// top of it.
func Load() (*Config, error) {
	cfg, err := LoadFile()
	if err != nil {
		return nil, err
	}

	applyEnv(cfg)

	return cfg, nil
}

// LoadFile reads the config file only, so that writes never persist values that
// came from the environment.
func LoadFile() (*Config, error) {
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

func applyEnv(cfg *Config) {
	overrides := []struct {
		name   string
		value  *string
		source *bool
	}{
		{EnvURL, &cfg.AIURL, &cfg.FromEnv.AIURL},
		{EnvAPIKey, &cfg.APIKey, &cfg.FromEnv.APIKey},
		{EnvModel, &cfg.Model, &cfg.FromEnv.Model},
	}

	for _, override := range overrides {
		if value := strings.TrimSpace(os.Getenv(override.name)); value != "" {
			*override.value = value
			*override.source = true
		}
	}
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
	n := len(c.APIKey)
	switch {
	case n < 8:
		return strings.Repeat("*", n)
	case n < 16:
		return strings.Repeat("*", 4) + c.APIKey[n-4:]
	default:
		return c.APIKey[:4] + strings.Repeat("*", n-8) + c.APIKey[n-4:]
	}
}

// InsecureURL reports whether the endpoint would send the API key in cleartext,
// which only matters once the request leaves the machine.
func InsecureURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "http") {
		return false
	}

	switch strings.ToLower(parsed.Hostname()) {
	case "", "localhost", "127.0.0.1", "::1":
		return false
	default:
		return true
	}
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
