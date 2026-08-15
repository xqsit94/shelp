package prompt

import "github.com/charmbracelet/x/ansi"

func Truncate(s string, max int) string {
	return ansi.Truncate(s, max, "…")
}
