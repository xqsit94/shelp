package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xqsit94/shelp/internal/config"
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
	t.Setenv("SHELP_PROFILE", "")
	t.Setenv("SHELP_NO_HISTORY", "1")

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

func readProfile(t *testing.T, dir, name string) map[string]any {
	t.Helper()

	stored := readConfigFile(t, dir)

	profiles, ok := stored["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("config file = %v, want a profiles object", stored)
	}

	profile, ok := profiles[name].(map[string]any)
	if !ok {
		t.Fatalf("profiles = %v, want a %q profile", profiles, name)
	}

	return profile
}

func TestConfigSetAndUnsetSamplingParameters(t *testing.T) {
	dir := configEnv(t)

	if _, _, err := execRoot(t, "config", "set", "temperature", "0.2"); err != nil {
		t.Fatalf("config set temperature returned error: %v", err)
	}
	if _, _, err := execRoot(t, "config", "set", "max-tokens", "256"); err != nil {
		t.Fatalf("config set max-tokens returned error: %v", err)
	}

	stored := readProfile(t, dir, config.DefaultProfile)
	if stored["temperature"] != 0.2 {
		t.Errorf("temperature = %v, want 0.2", stored["temperature"])
	}
	if stored["max_tokens"] != float64(256) {
		t.Errorf("max_tokens = %v, want 256", stored["max_tokens"])
	}

	if _, _, err := execRoot(t, "config", "unset", "temperature"); err != nil {
		t.Fatalf("config unset temperature returned error: %v", err)
	}

	stored = readProfile(t, dir, config.DefaultProfile)
	if _, ok := stored["temperature"]; ok {
		t.Errorf("temperature = %v, want it removed", stored["temperature"])
	}
	if stored["max_tokens"] != float64(256) {
		t.Errorf("max_tokens = %v, want it kept", stored["max_tokens"])
	}

	if _, _, err := execRoot(t, "config", "unset", "max-tokens"); err != nil {
		t.Fatalf("config unset max-tokens returned error: %v", err)
	}

	if stored = readProfile(t, dir, config.DefaultProfile); len(stored) != 3 {
		t.Errorf("profile = %v, want only the three string settings", stored)
	}
}

func TestConfigSetWritesTheResolvedProfile(t *testing.T) {
	dir := configEnv(t)
	t.Setenv("SHELP_MODEL", "env-model")

	if _, _, err := execRoot(t, "config", "set", "url", "https://default"); err != nil {
		t.Fatalf("config set url returned error: %v", err)
	}
	if _, _, err := execRoot(t, "--profile", "work", "config", "set", "model", "work-model"); err != nil {
		t.Fatalf("config set model returned error: %v", err)
	}

	stored := readConfigFile(t, dir)
	if stored["active_profile"] != config.DefaultProfile {
		t.Errorf("active_profile = %v, want %q", stored["active_profile"], config.DefaultProfile)
	}

	if got := readProfile(t, dir, config.DefaultProfile); got["ai_url"] != "https://default" || got["model"] != "" {
		t.Errorf("default profile = %v, want only the URL and no environment value", got)
	}
	if got := readProfile(t, dir, "work"); got["model"] != "work-model" || got["ai_url"] != "" {
		t.Errorf("work profile = %v, want only the model", got)
	}
}

func TestConfigSetMigratesV1File(t *testing.T) {
	dir := configEnv(t)

	v1 := `{"ai_url":"https://file","api_key":"file-key","model":"old-model"}`
	if err := os.WriteFile(filepath.Join(dir, paths.ConfigFileName), []byte(v1), 0600); err != nil {
		t.Fatalf("write v1 config file: %v", err)
	}

	if _, _, err := execRoot(t, "config", "set", "model", "new-model"); err != nil {
		t.Fatalf("config set model returned error: %v", err)
	}

	stored := readConfigFile(t, dir)
	if stored["active_profile"] != config.DefaultProfile {
		t.Errorf("active_profile = %v, want %q", stored["active_profile"], config.DefaultProfile)
	}

	profile := readProfile(t, dir, config.DefaultProfile)
	if profile["model"] != "new-model" {
		t.Errorf("model = %v, want %q", profile["model"], "new-model")
	}
	if profile["api_key"] != "file-key" {
		t.Errorf("api_key = %v, want the v1 value", profile["api_key"])
	}
}

func TestConfigProfileUseAndList(t *testing.T) {
	dir := configEnv(t)

	for _, name := range []string{"default", "work"} {
		if _, _, err := execRoot(t, "--profile", name, "config", "set", "model", name+"-model"); err != nil {
			t.Fatalf("config set model for %q returned error: %v", name, err)
		}
	}

	if _, _, err := execRoot(t, "config", "profile", "use", "work"); err != nil {
		t.Fatalf("config profile use returned error: %v", err)
	}

	stored := readConfigFile(t, dir)
	if stored["active_profile"] != "work" {
		t.Errorf("active_profile = %v, want %q", stored["active_profile"], "work")
	}

	stdout, _, err := execRoot(t, "config", "profile", "list")
	if err != nil {
		t.Fatalf("config profile list returned error: %v", err)
	}
	for _, want := range []string{"default", "work", "work-model", "*"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", stdout, want)
		}
	}
}

func TestConfigProfileUseUnknown(t *testing.T) {
	configEnv(t)

	if _, _, err := execRoot(t, "config", "set", "model", "m"); err != nil {
		t.Fatalf("config set model returned error: %v", err)
	}

	_, _, err := execRoot(t, "config", "profile", "use", "missing")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}
	if want := `unknown profile "missing" (available: default)`; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestConfigProfileRemove(t *testing.T) {
	dir := configEnv(t)

	for _, name := range []string{"default", "work"} {
		if _, _, err := execRoot(t, "--profile", name, "config", "set", "model", name+"-model"); err != nil {
			t.Fatalf("config set model for %q returned error: %v", name, err)
		}
	}

	if _, _, err := execRoot(t, "config", "profile", "remove", "default", "-y"); err == nil {
		t.Fatal("removing the active profile returned no error")
	}

	if _, _, err := execRoot(t, "config", "profile", "remove", "work", "-y"); err != nil {
		t.Fatalf("config profile remove returned error: %v", err)
	}

	stored := readConfigFile(t, dir)
	profiles, ok := stored["profiles"].(map[string]any)
	if !ok || len(profiles) != 1 {
		t.Fatalf("profiles = %v, want only the default profile", stored["profiles"])
	}
	if _, ok := profiles["work"]; ok {
		t.Errorf("profiles = %v, want work removed", profiles)
	}

	_, _, err := execRoot(t, "config", "profile", "remove", "default", "-y")
	if err == nil {
		t.Fatal("removing the last profile returned no error")
	}
}

func TestConfigProfileRename(t *testing.T) {
	dir := configEnv(t)

	if _, _, err := execRoot(t, "config", "set", "model", "m"); err != nil {
		t.Fatalf("config set model returned error: %v", err)
	}
	if _, _, err := execRoot(t, "config", "profile", "rename", "default", "work"); err != nil {
		t.Fatalf("config profile rename returned error: %v", err)
	}

	stored := readConfigFile(t, dir)
	if stored["active_profile"] != "work" {
		t.Errorf("active_profile = %v, want %q", stored["active_profile"], "work")
	}
	if got := readProfile(t, dir, "work"); got["model"] != "m" {
		t.Errorf("work profile = %v, want the renamed values", got)
	}
}

func TestConfigProfileAddWithoutTerminal(t *testing.T) {
	configEnv(t)

	_, _, err := execRoot(t, "config", "profile", "add", "work")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}
	if !strings.Contains(err.Error(), "shelp --profile work config set") {
		t.Errorf("error = %q, want it to name the non-interactive alternative", err)
	}
}

func TestRootUnknownProfileFails(t *testing.T) {
	server := fakeProvider(t, "echo hi")
	configureEnv(t, server)

	if _, _, err := execRoot(t, "config", "set", "model", "m"); err != nil {
		t.Fatalf("config set model returned error: %v", err)
	}

	t.Setenv("SHELP_PROFILE", "missing")

	_, _, err := execRoot(t, "-p", "say", "hi")
	if err == nil {
		t.Fatal("Execute() returned no error for an unknown profile")
	}
	if !strings.Contains(err.Error(), `unknown profile "missing"`) {
		t.Errorf("error = %q, want it to name the unknown profile", err)
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
