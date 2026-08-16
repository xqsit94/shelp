package config

import (
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

	return dir
}

func TestLoadMissingFile(t *testing.T) {
	isolate(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if *cfg != (Config{}) {
		t.Errorf("Load() = %+v, want zero config", *cfg)
	}
}

func TestSaveLoadResetRoundTrip(t *testing.T) {
	dir := isolate(t)

	want := &Config{
		AIURL:  "https://openrouter.ai/api/v1/chat/completions",
		APIKey: "sk-or-v1-secret",
		Model:  "anthropic/claude-3.5-sonnet",
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

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
	if *got != *want {
		t.Errorf("Load() = %+v, want %+v", *got, *want)
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
	if *got != (Config{}) {
		t.Errorf("Load() after Reset() = %+v, want zero config", *got)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	isolate(t)

	file := &Config{AIURL: "https://file", APIKey: "file-key", Model: "file-model"}
	if err := Save(file); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	t.Setenv(EnvURL, "https://env")
	t.Setenv(EnvModel, "  env-model  ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	want := Config{
		AIURL:   "https://env",
		APIKey:  "file-key",
		Model:   "env-model",
		FromEnv: Sources{AIURL: true, Model: true},
	}
	if *cfg != want {
		t.Errorf("Load() = %+v, want %+v", *cfg, want)
	}

	onDisk, err := LoadFile()
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	if *onDisk != *file {
		t.Errorf("LoadFile() = %+v, want %+v", *onDisk, *file)
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
