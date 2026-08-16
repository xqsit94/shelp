package executor

import (
	"bytes"
	"context"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

func lookPathIn(installed ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		if slices.Contains(installed, name) {
			return "/fake/" + name, nil
		}
		return "", exec.ErrNotFound
	}
}

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  string
	}{
		{"zsh", "/bin/zsh", "zsh"},
		{"fish", "/usr/bin/fish", "fish"},
		{"bash", "/bin/bash", "bash"},
		{"sh", "/bin/sh", "sh"},
		{"unset", "", "bash"},
		{"unknown shell", "/opt/homebrew/bin/nu", "bash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.shell)
			if got := DetectShell(); got != tt.want {
				t.Errorf("DetectShell() with SHELL=%q = %q, want %q", tt.shell, got, tt.want)
			}
		})
	}
}

func TestDetectShellForGOOS(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		shellEnv  string
		installed []string
		want      string
	}{
		{"unix zsh", "darwin", "/bin/zsh", nil, "zsh"},
		{"unix fish", "linux", "/usr/bin/fish", nil, "fish"},
		{"unix unknown", "linux", "/opt/bin/nu", nil, "bash"},
		{"unix unset", "linux", "", nil, "bash"},
		{"windows prefers pwsh", "windows", "", []string{"pwsh", "powershell"}, "pwsh"},
		{"windows falls back to powershell", "windows", "", []string{"powershell"}, "powershell"},
		{"windows falls back to cmd", "windows", "", nil, "cmd"},
		{"windows ignores SHELL", "windows", "/bin/zsh", nil, "cmd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectShell(tt.goos, tt.shellEnv, lookPathIn(tt.installed...))
			if got != tt.want {
				t.Errorf("detectShell(%q, %q) = %q, want %q", tt.goos, tt.shellEnv, got, tt.want)
			}
		})
	}
}

func TestShellArgs(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  []string
	}{
		{"sh", "sh", []string{"-c", "echo hi"}},
		{"bash", "bash", []string{"-c", "echo hi"}},
		{"zsh", "zsh", []string{"-c", "echo hi"}},
		{"fish", "fish", []string{"-c", "echo hi"}},
		{"pwsh", "pwsh", []string{"-NoProfile", "-Command", "echo hi"}},
		{"powershell", "powershell", []string{"-NoProfile", "-Command", "echo hi"}},
		{"cmd", "cmd", []string{"/C", "echo hi"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args := shellArgs(tt.shell, "echo hi")
			if name != tt.shell {
				t.Errorf("shellArgs(%q) name = %q, want %q", tt.shell, name, tt.shell)
			}
			if !slices.Equal(args, tt.want) {
				t.Errorf("shellArgs(%q) args = %q, want %q", tt.shell, args, tt.want)
			}
		})
	}
}

func TestExecuteExitCode(t *testing.T) {
	result, err := Execute(t.Context(), "exit 3", "sh", Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", result.ExitCode)
	}
	if result.Interrupted {
		t.Error("Interrupted = true, want false")
	}
}

func TestExecuteStreamsOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer

	result, err := Execute(t.Context(), "echo hi; echo oops >&2", "sh", Options{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if got := stdout.String(); got != "hi\n" {
		t.Errorf("stdout = %q, want %q", got, "hi\n")
	}
	if got := stderr.String(); got != "oops\n" {
		t.Errorf("stderr = %q, want %q", got, "oops\n")
	}
}

func TestExecuteReadsStdin(t *testing.T) {
	var stdout bytes.Buffer

	opts := Options{Stdin: strings.NewReader("from stdin\n"), Stdout: &stdout, Stderr: &bytes.Buffer{}}
	if _, err := Execute(t.Context(), "cat", "sh", opts); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := stdout.String(); got != "from stdin\n" {
		t.Errorf("stdout = %q, want %q", got, "from stdin\n")
	}
}

func TestExecuteBlockedCommand(t *testing.T) {
	result, err := Execute(t.Context(), "rm -rf /", "sh", Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if err == nil {
		t.Fatalf("Execute returned no error, got %+v", result)
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error = %v, want it to mention the command was blocked", err)
	}
}

func TestExecuteUnknownShellFallsBackToSh(t *testing.T) {
	var stdout bytes.Buffer

	result, err := Execute(t.Context(), "echo fallback", "definitely-not-a-shell", Options{Stdout: &stdout, Stderr: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if got := stdout.String(); got != "fallback\n" {
		t.Errorf("stdout = %q, want %q", got, "fallback\n")
	}
}

func TestExecuteCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()

	result, err := Execute(ctx, "sleep 5", "sh", Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= 5*time.Second {
		t.Errorf("took %s, want the command to be interrupted", elapsed)
	}
	if !result.Interrupted {
		t.Error("Interrupted = false, want true")
	}
	if result.ExitCode == 0 {
		t.Error("ExitCode = 0, want a non-zero code for an interrupted command")
	}
}
