package prompt

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func Truncate(s string, max int) string {
	return ansi.Truncate(s, max, "…")
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
