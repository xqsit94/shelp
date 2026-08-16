package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestInitSnippets(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  []string
	}{
		{
			name:  "zsh",
			shell: "zsh",
			want:  []string{"_shelp_widget() {", "zle -N _shelp_widget", "bindkey '^G' _shelp_widget", "$BUFFER", "CURSOR=", "zle -M", " && "},
		},
		{
			name:  "bash",
			shell: "bash",
			want:  []string{"_shelp_widget() {", `bind -x '"\C-g": _shelp_widget'`, "READLINE_LINE", "READLINE_POINT", " && "},
		},
		{
			name:  "fish",
			shell: "fish",
			want:  []string{"function _shelp_widget", `bind \cg _shelp_widget`, "commandline -r --", "commandline -f repaint", "string join ' && '"},
		},
		{
			name:  "powershell",
			shell: "powershell",
			want:  []string{"Set-PSReadLineKeyHandler -Chord 'Ctrl+g'", "GetBufferState", "Replace(0, $line.Length", "-join '; '"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := execRoot(t, "init", tt.shell)
			if err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want it empty", stderr)
			}

			for _, want := range append(tt.want, "shelp -p --", "Rebind:") {
				if !strings.Contains(stdout, want) {
					t.Errorf("snippet does not contain %q:\n%s", want, stdout)
				}
			}
			if strings.ContainsRune(stdout, '\x1b') {
				t.Error("snippet contains an escape sequence, want plain text")
			}
		})
	}
}

// A generated command line may start with a dash, so the query has to survive
// being handed back to shelp as an argument.
func TestInitSnippetsQuoteTheQuery(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		t.Run(shell, func(t *testing.T) {
			stdout, _, err := execRoot(t, "init", shell)
			if err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}
			if !strings.Contains(stdout, `command shelp -p -- "$query"`) {
				t.Errorf("snippet does not pass the query after --:\n%s", stdout)
			}
			if !strings.Contains(stdout, "2>/dev/tty") {
				t.Errorf("snippet does not send stderr to the terminal:\n%s", stdout)
			}
		})
	}
}

func TestInitUnknownShell(t *testing.T) {
	_, _, err := execRoot(t, "init", "tcsh")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}
	for _, shell := range initShells {
		if !strings.Contains(err.Error(), shell) {
			t.Errorf("error = %q, want it to list %q", err, shell)
		}
	}
}
