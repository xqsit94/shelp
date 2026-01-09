package prompt

import (
	"fmt"
	"strings"
)

const (
	IconSuccess = "●"
	IconError   = "●"
	IconWarning = "●"
	IconInfo    = "●"
	IconArrow   = "→"
)

func DisplayOutput(output string, isError bool) {
	if output == "" {
		return
	}

	lines := strings.Split(output, "\n")
	var content strings.Builder
	for _, line := range lines {
		content.WriteString(line + "\n")
	}

	if isError {
		fmt.Println(RenderErrorBox(strings.TrimSpace(content.String())))
	} else {
		fmt.Println(RenderOutputBox(strings.TrimSpace(content.String())))
	}
}

func DisplaySuccess(message string) {
	fmt.Println(successStyle.Render("  " + IconSuccess + " " + message))
}

func DisplayError(message string) {
	fmt.Println(dangerStyle.Render("  " + IconError + " " + message))
}

func DisplayWarning(message string) {
	fmt.Println(warningStyle.Render("  " + IconWarning + " " + message))
}

func DisplayInfo(message string) {
	fmt.Println(infoStyle.Render("  " + IconInfo + " " + message))
}
