package prompt

import (
	"bytes"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Resolved once at package init: HasDarkBackground queries the terminal over
// stdin, which is unavailable once a Bubbletea program owns the TTY.
var (
	bashLexer       = newBashLexer()
	chromaStyle     = newChromaStyle()
	chromaFormatter = formatters.Get("terminal256")

	// chroma has no notion of NO_COLOR or of a piped stdout, so the colour
	// decision lipgloss already made for its own styles is reused here.
	highlightEnabled = lipgloss.ColorProfile() != termenv.Ascii
)

func newBashLexer() chroma.Lexer {
	lexer := lexers.Get("bash")
	if lexer == nil {
		return nil
	}
	return chroma.Coalesce(lexer)
}

func newChromaStyle() *chroma.Style {
	if lipgloss.HasDarkBackground() {
		return styles.Get("monokai")
	}
	return styles.Get("github")
}

// HighlightFor highlights only when w is a terminal, so redirected output stays plain.
func HighlightFor(w io.Writer, command string) string {
	if !IsTerminalWriter(w) {
		return command
	}
	return HighlightCommand(command)
}

func HighlightCommand(command string) string {
	if bashLexer == nil || !highlightEnabled {
		return command
	}

	iterator, err := bashLexer.Tokenise(nil, command)
	if err != nil {
		return command
	}

	var buf bytes.Buffer
	if err := chromaFormatter.Format(&buf, chromaStyle, iterator); err != nil {
		return command
	}

	return strings.TrimSuffix(buf.String(), "\n")
}
