package prompt

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	IconSuccess = "✓"
	IconError   = "✕"
	IconWarning = "▲"
)

func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func DisplayRunning(index, total int, command string) {
	prefix := fmt.Sprintf("%s %s ",
		TreeStyle.Render(TreeBranch),
		hintStyle.Render(fmt.Sprintf("[%d/%d]", index, total)),
	)

	fmt.Println(IndentUnder(prefix, HighlightCommand(command)))
	fmt.Println()
}

func DisplaySuccess(message string) {
	fmt.Println(SuccessStyle.Render("  " + IconSuccess + " " + message))
}

func DisplayError(message string) {
	fmt.Println(DangerStyle.Render("  " + IconError + " " + message))
}

func DisplayWarning(message string) {
	fmt.Println(warningStyle.Render("  " + IconWarning + " " + message))
}
