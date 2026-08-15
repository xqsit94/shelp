package executor

import "testing"

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
