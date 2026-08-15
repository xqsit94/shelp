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

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if *cfg != (Config{}) {
		t.Errorf("Load() = %+v, want zero config", *cfg)
	}
}

func TestSaveLoadResetRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := &Config{
		AIURL:  "https://openrouter.ai/api/v1/chat/completions",
		APIKey: "sk-or-v1-secret",
		Model:  "anthropic/claude-3.5-sonnet",
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	configPath := filepath.Join(home, paths.ConfigDirName, paths.ConfigFileName)
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
