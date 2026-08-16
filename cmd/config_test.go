package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xqsit94/shelp/pkg/paths"
)

func configEnv(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(paths.ConfigDirEnv, dir)
	t.Setenv("SHELP_URL", "")
	t.Setenv("SHELP_API_KEY", "")
	t.Setenv("SHELP_MODEL", "")
	t.Setenv("SHELP_TEMPERATURE", "")
	t.Setenv("SHELP_MAX_TOKENS", "")

	return dir
}

func readConfigFile(t *testing.T, dir string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, paths.ConfigFileName))
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse config file: %v", err)
	}

	return parsed
}

func TestConfigSetAndUnsetSamplingParameters(t *testing.T) {
	dir := configEnv(t)

	if _, _, err := execRoot(t, "config", "set", "temperature", "0.2"); err != nil {
		t.Fatalf("config set temperature returned error: %v", err)
	}
	if _, _, err := execRoot(t, "config", "set", "max-tokens", "256"); err != nil {
		t.Fatalf("config set max-tokens returned error: %v", err)
	}

	stored := readConfigFile(t, dir)
	if stored["temperature"] != 0.2 {
		t.Errorf("temperature = %v, want 0.2", stored["temperature"])
	}
	if stored["max_tokens"] != float64(256) {
		t.Errorf("max_tokens = %v, want 256", stored["max_tokens"])
	}

	if _, _, err := execRoot(t, "config", "unset", "temperature"); err != nil {
		t.Fatalf("config unset temperature returned error: %v", err)
	}

	stored = readConfigFile(t, dir)
	if _, ok := stored["temperature"]; ok {
		t.Errorf("temperature = %v, want it removed", stored["temperature"])
	}
	if stored["max_tokens"] != float64(256) {
		t.Errorf("max_tokens = %v, want it kept", stored["max_tokens"])
	}

	if _, _, err := execRoot(t, "config", "unset", "max-tokens"); err != nil {
		t.Fatalf("config unset max-tokens returned error: %v", err)
	}

	if stored = readConfigFile(t, dir); len(stored) != 3 {
		t.Errorf("config file = %v, want only the three string settings", stored)
	}
}

func TestConfigSetRejectsInvalidSamplingParameters(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		value string
	}{
		{"temperature not a number", []string{"config", "set", "temperature"}, "hot"},
		{"temperature above range", []string{"config", "set", "temperature"}, "2.5"},
		{"temperature below range", []string{"config", "set", "temperature"}, "-1"},
		{"max tokens not a number", []string{"config", "set", "max-tokens"}, "many"},
		{"max tokens zero", []string{"config", "set", "max-tokens"}, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := configEnv(t)

			if _, _, err := execRoot(t, append(tt.args, tt.value)...); err == nil {
				t.Fatalf("%v %q returned no error", tt.args, tt.value)
			}
			if _, err := os.Stat(filepath.Join(dir, paths.ConfigFileName)); !os.IsNotExist(err) {
				t.Errorf("config file was written: %v", err)
			}
		})
	}
}
