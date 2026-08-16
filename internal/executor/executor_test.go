package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

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
