package prompt

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// Resolved once at package init: HasDarkBackground queries the terminal over
// stdin, which is unavailable once a Bubbletea program owns the TTY.
var (
	bashLexer       = newBashLexer()
	chromaStyle     = newChromaStyle()
	chromaFormatter = formatters.Get("terminal256")
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

func HighlightCommand(command string) string {
	if bashLexer == nil {
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
