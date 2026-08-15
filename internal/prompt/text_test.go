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
