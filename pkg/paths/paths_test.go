package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	home := t.TempDir()
	override := t.TempDir()

	tests := []struct {
		name      string
		configDir string
		want      string
	}{
		{"home", "", filepath.Join(home, ConfigDirName)},
		{"override", override, override},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", home)
			t.Setenv(ConfigDirEnv, tt.configDir)

			if got := GetConfigDir(); got != tt.want {
				t.Errorf("GetConfigDir() = %q, want %q", got, tt.want)
			}
			if got, want := GetConfigPath(), filepath.Join(tt.want, ConfigFileName); got != want {
				t.Errorf("GetConfigPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestEnsureConfigDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "shelp")
	t.Setenv(ConfigDirEnv, dir)

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir() returned error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("config dir mode = %04o, want 0700", perm)
	}
}
