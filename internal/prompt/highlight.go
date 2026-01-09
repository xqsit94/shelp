package prompt

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func HighlightCommand(command string) string {
	lexer := lexers.Get("bash")
	if lexer == nil {
		return command
	}

	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	formatter := formatters.Get("terminal256")

	iterator, err := lexer.Tokenise(nil, command)
	if err != nil {
		return command
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return command
	}

	return strings.TrimSuffix(buf.String(), "\n")
}
