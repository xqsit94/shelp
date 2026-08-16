package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xqsit94/shelp/pkg/paths"
)

func TestMaskedAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{"empty", "", ""},
		{"very short", "abc", "***"},
		{"just under eight", "abcdefg", "*******"},
		{"exactly eight", "abcdefgh", "****efgh"},
		{"eleven", "sk-abcdefgh", "****efgh"},
		{"just under sixteen", "abcdefghijklmno", "****lmno"},
		{"exactly sixteen", "abcdefghijklmnop", "abcd" + strings.Repeat("*", 8) + "mnop"},
		{"long", "sk-or-v1-0123456789abcdef", "sk-o" + strings.Repeat("*", 17) + "cdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{APIKey: tt.apiKey}
			if got := cfg.MaskedAPIKey(); got != tt.want {
				t.Errorf("MaskedAPIKey() for %q = %q, want %q", tt.apiKey, got, tt.want)
			}
		})
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"complete", Config{AIURL: "https://x", APIKey: "k", Model: "m"}, true},
		{"missing url", Config{APIKey: "k", Model: "m"}, false},
		{"missing key", Config{AIURL: "https://x", Model: "m"}, false},
		{"missing model", Config{AIURL: "https://x", APIKey: "k"}, false},
		{"empty", Config{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() for %+v = %v, want %v", tt.cfg, got, tt.want)
			}
		})
	}
}

// isolate points config at a scratch directory and clears the environment
// overrides so ambient SHELP_* values cannot leak into a test.
func isolate(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(paths.ConfigDirEnv, dir)
	t.Setenv(EnvURL, "")
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvModel, "")
	t.Setenv(EnvTemperature, "")
	t.Setenv(EnvMaxTokens, "")
	t.Setenv(EnvProfile, "")

	return dir
}

func saveProfiles(t *testing.T, active string, profiles map[string]Profile) {
	t.Helper()

	if err := SaveFile(&File{ActiveProfile: active, Profiles: profiles}); err != nil {
		t.Fatalf("SaveFile() returned error: %v", err)
	}
}

func writeConfigFile(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, paths.ConfigFileName), []byte(content), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	isolate(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if want := (Config{Profile: DefaultProfile}); *cfg != want {
		t.Errorf("Load() = %+v, want %+v", *cfg, want)
	}
}

func TestSaveFileLoadResetRoundTrip(t *testing.T) {
	dir := isolate(t)

	want := Profile{
		AIURL:  "https://openrouter.ai/api/v1/chat/completions",
		APIKey: "sk-or-v1-secret",
		Model:  "anthropic/claude-3.5-sonnet",
	}
	saveProfiles(t, DefaultProfile, map[string]Profile{DefaultProfile: want})

	configPath := filepath.Join(dir, paths.ConfigFileName)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file mode = %04o, want 0600", perm)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if wantCfg := (Config{Profile: DefaultProfile, AIURL: want.AIURL, APIKey: want.APIKey, Model: want.Model}); *got != wantCfg {
		t.Errorf("Load() = %+v, want %+v", *got, wantCfg)
	}

	if err := Reset(); err != nil {
		t.Fatalf("Reset() returned error: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Errorf("config file still present after Reset(): %v", err)
	}

	if err := Reset(); err != nil {
		t.Errorf("Reset() on missing file returned error: %v", err)
	}

	got, err = Load()
	if err != nil {
		t.Fatalf("Load() after Reset() returned error: %v", err)
	}
	if want := (Config{Profile: DefaultProfile}); *got != want {
		t.Errorf("Load() after Reset() = %+v, want %+v", *got, want)
	}
}

func TestLoadFileMigratesV1Schema(t *testing.T) {
	dir := isolate(t)

	writeConfigFile(t, dir, `{"ai_url":"https://file","api_key":"file-key","model":"file-model","temperature":0.2}`)

	file, err := LoadFile()
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	if file.ActiveProfile != DefaultProfile {
		t.Errorf("ActiveProfile = %q, want %q", file.ActiveProfile, DefaultProfile)
	}

	profile, ok := file.Get(DefaultProfile)
	if !ok {
		t.Fatalf("profiles = %v, want the v1 values under %q", file.Profiles, DefaultProfile)
	}
	if profile.AIURL != "https://file" || profile.APIKey != "file-key" || profile.Model != "file-model" {
		t.Errorf("profile = %+v, want the v1 values", profile)
	}
	if profile.Temperature == nil || *profile.Temperature != 0.2 {
		t.Errorf("Temperature = %v, want 0.2", profile.Temperature)
	}

	// The next write upgrades the file to the v2 schema.
	if _, err := UpdateProfile("", func(profile *Profile) { profile.Model = "new-model" }); err != nil {
		t.Fatalf("UpdateProfile() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, paths.ConfigFileName))
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	var stored File
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("parse config file: %v", err)
	}
	if stored.ActiveProfile != DefaultProfile {
		t.Errorf("active_profile = %q, want %q", stored.ActiveProfile, DefaultProfile)
	}
	if got := stored.Profiles[DefaultProfile].Model; got != "new-model" {
		t.Errorf("model = %q, want %q", got, "new-model")
	}
	if got := stored.Profiles[DefaultProfile].AIURL; got != "https://file" {
		t.Errorf("ai_url = %q, want it kept", got)
	}
}

func TestLoadProfilePrecedence(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		env       string
		want      string
		wantModel string
	}{
		{"requested wins", "work", "home", "work", "work-model"},
		{"env over active", "", "home", "home", "home-model"},
		{"active over default", "", "", "personal", "personal-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t)
			saveProfiles(t, "personal", map[string]Profile{
				"work":     {AIURL: "https://work", APIKey: "k", Model: "work-model"},
				"home":     {AIURL: "https://home", APIKey: "k", Model: "home-model"},
				"personal": {AIURL: "https://personal", APIKey: "k", Model: "personal-model"},
			})
			t.Setenv(EnvProfile, tt.env)

			cfg, err := LoadProfile(tt.requested)
			if err != nil {
				t.Fatalf("LoadProfile(%q) returned error: %v", tt.requested, err)
			}
			if cfg.Profile != tt.want {
				t.Errorf("Profile = %q, want %q", cfg.Profile, tt.want)
			}
			if cfg.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", cfg.Model, tt.wantModel)
			}
		})
	}
}

func TestLoadProfileDefaultsWithoutActiveProfile(t *testing.T) {
	isolate(t)
	saveProfiles(t, "", map[string]Profile{DefaultProfile: {AIURL: "https://x", APIKey: "k", Model: "m"}})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Profile != DefaultProfile {
		t.Errorf("Profile = %q, want %q", cfg.Profile, DefaultProfile)
	}
}

func TestLoadUnknownProfile(t *testing.T) {
	isolate(t)
	saveProfiles(t, "work", map[string]Profile{
		"work": {AIURL: "https://work", APIKey: "k", Model: "m"},
		"home": {AIURL: "https://home", APIKey: "k", Model: "m"},
	})

	_, err := LoadProfile("missing")
	if err == nil {
		t.Fatal("LoadProfile() returned no error for an unknown profile")
	}
	if want := `unknown profile "missing" (available: home, work)`; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestLoadUnknownProfileWithoutConfigFile(t *testing.T) {
	isolate(t)
	t.Setenv(EnvURL, "https://env")
	t.Setenv(EnvAPIKey, "env-key")
	t.Setenv(EnvModel, "env-model")

	cfg, err := LoadProfile("work")
	if err != nil {
		t.Fatalf("LoadProfile() returned error: %v", err)
	}
	if !cfg.IsConfigured() {
		t.Errorf("IsConfigured() = false for env-only config %+v", *cfg)
	}
	if cfg.Profile != "work" {
		t.Errorf("Profile = %q, want %q", cfg.Profile, "work")
	}
}

func TestUpdateProfileWritesResolvedProfileOnly(t *testing.T) {
	isolate(t)
	saveProfiles(t, "work", map[string]Profile{
		"work":     {AIURL: "https://work", APIKey: "work-key", Model: "work-model"},
		"personal": {AIURL: "https://personal", APIKey: "personal-key", Model: "personal-model"},
	})
	t.Setenv(EnvModel, "env-model")

	name, err := UpdateProfile("", func(profile *Profile) { profile.Model = "next-model" })
	if err != nil {
		t.Fatalf("UpdateProfile() returned error: %v", err)
	}
	if name != "work" {
		t.Errorf("UpdateProfile() = %q, want %q", name, "work")
	}

	file, err := LoadFile()
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	if got := file.Profiles["work"].Model; got != "next-model" {
		t.Errorf("work model = %q, want %q", got, "next-model")
	}
	if got := file.Profiles["personal"].Model; got != "personal-model" {
		t.Errorf("personal model = %q, want it untouched", got)
	}
}

func TestUpdateProfileCreatesFirstProfileActive(t *testing.T) {
	isolate(t)

	name, err := UpdateProfile("work", func(profile *Profile) { profile.Model = "work-model" })
	if err != nil {
		t.Fatalf("UpdateProfile() returned error: %v", err)
	}
	if name != "work" {
		t.Errorf("UpdateProfile() = %q, want %q", name, "work")
	}

	file, err := LoadFile()
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	if file.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want %q", file.ActiveProfile, "work")
	}

	if _, err := UpdateProfile("side", func(profile *Profile) { profile.Model = "side-model" }); err != nil {
		t.Fatalf("UpdateProfile() returned error: %v", err)
	}

	file, err = LoadFile()
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	if file.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want it unchanged", file.ActiveProfile)
	}
	if want := []string{"side", "work"}; strings.Join(file.Names(), ",") != strings.Join(want, ",") {
		t.Errorf("Names() = %v, want %v", file.Names(), want)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	isolate(t)

	stored := Profile{AIURL: "https://file", APIKey: "file-key", Model: "file-model"}
	saveProfiles(t, DefaultProfile, map[string]Profile{DefaultProfile: stored})

	t.Setenv(EnvURL, "https://env")
	t.Setenv(EnvModel, "  env-model  ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	want := Config{
		Profile: DefaultProfile,
		AIURL:   "https://env",
		APIKey:  "file-key",
		Model:   "env-model",
		FromEnv: Sources{AIURL: true, Model: true},
	}
	if *cfg != want {
		t.Errorf("Load() = %+v, want %+v", *cfg, want)
	}

	file, err := LoadFile()
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	if got := file.Profiles[DefaultProfile]; got != stored {
		t.Errorf("LoadFile() profile = %+v, want %+v", got, stored)
	}
}

func TestLoadEnvOnlyIsConfigured(t *testing.T) {
	isolate(t)

	t.Setenv(EnvURL, "https://env")
	t.Setenv(EnvAPIKey, "env-key")
	t.Setenv(EnvModel, "env-model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if !cfg.IsConfigured() {
		t.Errorf("IsConfigured() = false for env-only config %+v", *cfg)
	}
	if cfg.FromEnv != (Sources{AIURL: true, APIKey: true, Model: true}) {
		t.Errorf("FromEnv = %+v, want all true", cfg.FromEnv)
	}
}

func TestSaveLoadSamplingParameters(t *testing.T) {
	isolate(t)

	temperature, maxTokens := 0.2, 256
	saveProfiles(t, DefaultProfile, map[string]Profile{
		DefaultProfile: {Temperature: &temperature, MaxTokens: &maxTokens},
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Temperature == nil || *cfg.Temperature != temperature {
		t.Errorf("Temperature = %v, want %v", cfg.Temperature, temperature)
	}
	if cfg.MaxTokens == nil || *cfg.MaxTokens != maxTokens {
		t.Errorf("MaxTokens = %v, want %v", cfg.MaxTokens, maxTokens)
	}
	if cfg.FromEnv != (Sources{}) {
		t.Errorf("FromEnv = %+v, want none", cfg.FromEnv)
	}
}

func TestSaveFileOmitsUnsetSamplingParameters(t *testing.T) {
	dir := isolate(t)

	saveProfiles(t, DefaultProfile, map[string]Profile{DefaultProfile: {AIURL: "https://x"}})

	data, err := os.ReadFile(filepath.Join(dir, paths.ConfigFileName))
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	for _, field := range []string{"temperature", "max_tokens"} {
		if strings.Contains(string(data), field) {
			t.Errorf("config file contains %q, want it omitted:\n%s", field, data)
		}
	}
}

func TestLoadSamplingEnvOverrides(t *testing.T) {
	isolate(t)

	t.Setenv(EnvTemperature, "0.2")
	t.Setenv(EnvMaxTokens, "  256  ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Temperature == nil || *cfg.Temperature != 0.2 {
		t.Errorf("Temperature = %v, want 0.2", cfg.Temperature)
	}
	if cfg.MaxTokens == nil || *cfg.MaxTokens != 256 {
		t.Errorf("MaxTokens = %v, want 256", cfg.MaxTokens)
	}
	if want := (Sources{Temperature: true, MaxTokens: true}); cfg.FromEnv != want {
		t.Errorf("FromEnv = %+v, want %+v", cfg.FromEnv, want)
	}
}

func TestLoadRejectsInvalidSamplingEnv(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{"temperature not a number", EnvTemperature, "hot"},
		{"temperature below range", EnvTemperature, "-0.1"},
		{"temperature above range", EnvTemperature, "2.5"},
		{"temperature nan", EnvTemperature, "NaN"},
		{"max tokens not a number", EnvMaxTokens, "many"},
		{"max tokens zero", EnvMaxTokens, "0"},
		{"max tokens negative", EnvMaxTokens, "-1"},
		{"max tokens fractional", EnvMaxTokens, "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t)
			t.Setenv(tt.env, tt.value)

			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load() = %+v, want an error for %s=%q", cfg, tt.env, tt.value)
			}
			if !strings.Contains(err.Error(), tt.env) {
				t.Errorf("error = %q, want it to name %s", err, tt.env)
			}
		})
	}
}

func TestLoadAcceptsSamplingRangeBounds(t *testing.T) {
	for _, value := range []string{"0", "2", "1.25"} {
		t.Run(value, func(t *testing.T) {
			isolate(t)
			t.Setenv(EnvTemperature, value)

			if _, err := Load(); err != nil {
				t.Errorf("Load() with %s=%q returned error: %v", EnvTemperature, value, err)
			}
		})
	}
}

func TestInsecureURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"https", "https://openrouter.ai/api/v1/chat/completions", false},
		{"http remote", "http://example.com/v1/chat/completions", true},
		{"http remote with port", "http://192.168.1.10:8080/v1", true},
		{"http localhost", "http://localhost:11434/v1/chat/completions", false},
		{"http loopback ipv4", "http://127.0.0.1:1234/v1", false},
		{"http loopback ipv6", "http://[::1]:1234/v1", false},
		{"uppercase scheme", "HTTP://example.com/v1", true},
		{"padded", "  http://example.com/v1  ", true},
		{"empty", "", false},
		{"no scheme", "example.com/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InsecureURL(tt.url); got != tt.want {
				t.Errorf("InsecureURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
