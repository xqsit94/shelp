package prompt

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"shorter than limit", "hello", 10, "hello"},
		{"exactly at limit", "abcdefghij", 10, "abcdefghij"},
		{"one over limit", "abcdefghijk", 10, "abcdefghi…"},
		{"ascii command", "hello world, this is a long command line", 20, "hello world, this i…"},
		{"multibyte", `find . -name "→→→→→→→→→→→→→→→→→→→→→→→→"`, 20, `find . -name "→→→→→…`},
		{"wide runes", "日本語のテキストです、これは長い", 10, "日本語の…"},
		{"empty", "", 5, ""},
		{"zero limit", "abc", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.s, tt.max); got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestOneline(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"single line", "ls -la", "ls -la"},
		{"multi line", "cd project &&\n  npm test", "cd project && npm test"},
		{"tabs", "for f in *; do\n\techo $f\ndone", "for f in *; do echo $f done"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Oneline(tt.s); got != tt.want {
				t.Errorf("Oneline(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestIndentLines(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		indent int
		want   string
	}{
		{"single line", "ls -la", 4, "ls -la"},
		{"two lines", "a\nb", 2, "a\n  b"},
		{"three lines", "a\nb\nc", 1, "a\n b\n c"},
		{"zero indent", "a\nb", 0, "a\nb"},
		{"negative indent", "a\nb", -1, "a\nb"},
		{"empty", "", 4, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IndentLines(tt.s, tt.indent); got != tt.want {
				t.Errorf("IndentLines(%q, %d) = %q, want %q", tt.s, tt.indent, got, tt.want)
			}
		})
	}
}

func TestIndentUnder(t *testing.T) {
	got := IndentUnder("├─ ", "cd project\n&& npm test")
	want := "├─ cd project\n   && npm test"

	if got != want {
		t.Errorf("IndentUnder = %q, want %q", got, want)
	}
}
