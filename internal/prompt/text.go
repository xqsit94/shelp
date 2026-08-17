package prompt

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func Truncate(s string, max int) string {
	return ansi.Truncate(s, max, "…")
}

// TruncateLines truncates every line of s, so that a block that was indented
// under a tree prefix stays inside the terminal on all of its lines. Without
// it the terminal clips the overflow silently and the command that is about to
// run is shown as a prefix of itself.
func TruncateLines(s string, max int) string {
	if max <= 0 {
		return s
	}
	if !strings.Contains(s, "\n") {
		return Truncate(s, max)
	}

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = Truncate(line, max)
	}
	return strings.Join(lines, "\n")
}

// Oneline collapses a multi-line command into a single line so it fits in a
// tree row.
func Oneline(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// IndentLines leaves the first line untouched and pads the rest, so that a
// multi-line command lines up under the first line instead of under the tree.
func IndentLines(s string, indent int) string {
	if indent <= 0 || !strings.Contains(s, "\n") {
		return s
	}
	return strings.ReplaceAll(s, "\n", "\n"+strings.Repeat(" ", indent))
}

func IndentUnder(prefix, body string) string {
	return prefix + IndentLines(body, ansi.StringWidth(prefix))
}
