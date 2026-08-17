package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/xqsit94/shelp/pkg/paths"
	"golang.org/x/term"
)

const (
	EnvURL         = "SHELP_URL"
	EnvAPIKey      = "SHELP_API_KEY"
	EnvModel       = "SHELP_MODEL"
	EnvTemperature = "SHELP_TEMPERATURE"
	EnvMaxTokens   = "SHELP_MAX_TOKENS"
	EnvProfile     = "SHELP_PROFILE"

	DefaultProfile = "default"
)

// Sources records which fields came from the environment rather than the file.
type Sources struct {
	AIURL       bool
	APIKey      bool
	Model       bool
	Temperature bool
	MaxTokens   bool
}

// Profile is one named provider as it is stored on disk.
type Profile struct {
	AIURL       string   `json:"ai_url"`
	APIKey      string   `json:"api_key"`
	Model       string   `json:"model"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

// File is the config file: a set of named profiles plus the one that is used
// when no profile is requested.
type File struct {
	ActiveProfile string             `json:"active_profile"`
	Profiles      map[string]Profile `json:"profiles"`

	// present separates "no config file yet" from "profile missing from the
	// file", so an env-only first run does not fail on an unknown profile.
	present bool
}

// Config is the effective configuration of one run: the resolved profile with
// the environment overrides applied.
//
// Temperature and MaxTokens are optional: when unset nothing is sent to the
// provider, because some OpenAI-compatible models reject the fields outright.
type Config struct {
	Profile     string
	AIURL       string
	APIKey      string
	Model       string
	Temperature *float64
	MaxTokens   *int

	FromEnv Sources
}

// Load reads the profile selected by SHELP_PROFILE or the active profile and
// applies the SHELP_* environment overrides on top of it.
func Load() (*Config, error) {
	return LoadProfile("")
}

// LoadProfile reads the named profile, falling back to the resolution order
// when name is empty.
func LoadProfile(name string) (*Config, error) {
	file, err := LoadFile()
	if err != nil {
		return nil, err
	}

	cfg, err := file.Config(name)
	if err != nil {
		return nil, err
	}

	if err := applyEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFile reads the config file only, so that writes never persist values that
// came from the environment.
func LoadFile() (*File, error) {
	data, err := os.ReadFile(paths.GetConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Profiles: map[string]Profile{}}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// The v1 schema stored a single unnamed profile at the top level.
	var stored struct {
		ActiveProfile string             `json:"active_profile"`
		Profiles      map[string]Profile `json:"profiles"`
		Profile
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	file := &File{ActiveProfile: stored.ActiveProfile, Profiles: stored.Profiles, present: true}
	if file.Profiles == nil {
		file.Profiles = map[string]Profile{}
		if stored.Profile != (Profile{}) {
			file.Profiles[DefaultProfile] = stored.Profile
		}
	}

	if _, ok := file.Profiles[file.ActiveProfile]; !ok && len(file.Profiles) > 0 {
		file.ActiveProfile = DefaultProfile
		if _, ok := file.Profiles[DefaultProfile]; !ok {
			file.ActiveProfile = file.Names()[0]
		}
	}

	return file, nil
}

func SaveFile(file *File) error {
	if err := paths.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	out := *file
	if out.Profiles == nil {
		out.Profiles = map[string]Profile{}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	if err := os.WriteFile(paths.GetConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (f *File) Get(name string) (Profile, bool) {
	profile, ok := f.Profiles[name]
	return profile, ok
}

// Set stores a profile, making it active when the file has no active profile
// yet, so the first profile created is the one runs use.
func (f *File) Set(name string, profile Profile) {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	f.Profiles[name] = profile

	if f.ActiveProfile == "" {
		f.ActiveProfile = name
	}
}

func (f *File) Delete(name string) {
	delete(f.Profiles, name)
}

func (f *File) Names() []string {
	names := make([]string, 0, len(f.Profiles))
	for name := range f.Profiles {
		names = append(names, name)
	}
	slices.Sort(names)

	return names
}

// Config returns the effective configuration of the requested profile, without
// the environment overrides.
func (f *File) Config(requested string) (*Config, error) {
	name := f.ResolveName(requested)

	profile, ok := f.Get(name)
	if !ok && f.present && len(f.Profiles) > 0 {
		return nil, fmt.Errorf("unknown profile %q (available: %s)", name, strings.Join(f.Names(), ", "))
	}

	return &Config{
		Profile:     name,
		AIURL:       profile.AIURL,
		APIKey:      profile.APIKey,
		Model:       profile.Model,
		Temperature: profile.Temperature,
		MaxTokens:   profile.MaxTokens,
	}, nil
}

// ResolveName applies the profile precedence: explicit request, then
// SHELP_PROFILE, then the active profile, then "default".
func (f *File) ResolveName(requested string) string {
	if name := strings.TrimSpace(requested); name != "" {
		return name
	}
	if name := strings.TrimSpace(os.Getenv(EnvProfile)); name != "" {
		return name
	}
	if f.ActiveProfile != "" {
		return f.ActiveProfile
	}

	return DefaultProfile
}

// UpdateProfile applies edit to the resolved profile and writes the file back,
// creating the profile when it does not exist yet. It returns the name written.
func UpdateProfile(requested string, edit func(*Profile)) (string, error) {
	file, err := LoadFile()
	if err != nil {
		return "", err
	}

	name := file.ResolveName(requested)
	profile, _ := file.Get(name)
	edit(&profile)
	file.Set(name, profile)

	if err := SaveFile(file); err != nil {
		return "", err
	}

	return name, nil
}

func applyEnv(cfg *Config) error {
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

	if value := strings.TrimSpace(os.Getenv(EnvTemperature)); value != "" {
		temperature, err := ParseTemperature(value)
		if err != nil {
			return fmt.Errorf("invalid %s: %v", EnvTemperature, err)
		}
		cfg.Temperature = &temperature
		cfg.FromEnv.Temperature = true
	}

	if value := strings.TrimSpace(os.Getenv(EnvMaxTokens)); value != "" {
		maxTokens, err := ParseMaxTokens(value)
		if err != nil {
			return fmt.Errorf("invalid %s: %v", EnvMaxTokens, err)
		}
		cfg.MaxTokens = &maxTokens
		cfg.FromEnv.MaxTokens = true
	}

	return nil
}

func ParseTemperature(value string) (float64, error) {
	temperature, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	// Negated so that NaN, which compares false against everything, is rejected.
	if err != nil || !(temperature >= 0 && temperature <= 2) {
		return 0, fmt.Errorf("%q is not a number between 0 and 2", value)
	}
	return temperature, nil
}

func ParseMaxTokens(value string) (int, error) {
	maxTokens, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || maxTokens <= 0 {
		return 0, fmt.Errorf("%q is not a positive integer", value)
	}
	return maxTokens, nil
}

func (c *Config) IsConfigured() bool {
	return c.AIURL != "" && c.APIKey != "" && c.Model != ""
}

func (c *Config) MaskedAPIKey() string {
	return MaskAPIKey(c.APIKey)
}

func MaskAPIKey(apiKey string) string {
	n := len(apiKey)
	switch {
	case n < 8:
		return strings.Repeat("*", n)
	case n < 16:
		return strings.Repeat("*", 4) + apiKey[n-4:]
	default:
		return apiKey[:4] + strings.Repeat("*", n-8) + apiKey[n-4:]
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
