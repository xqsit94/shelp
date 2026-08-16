package prompt

import (
	"fmt"
	"io"
	"os"

	"github.com/xqsit94/shelp/internal/safety"
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

// IsTerminalWriter reports whether w is a terminal, so that plain text can be
// written when the output is piped or captured.
func IsTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// DisplayCommandPlan prints one tree row per command with its risk level, so
// unattended runs still show what is about to happen.
func DisplayCommandPlan(commands []string) {
	fmt.Println()
	fmt.Println(TitleBoldStyle.Foreground(ColorInfo).Render(fmt.Sprintf("Generated Commands (%d)", len(commands))))

	for i, command := range commands {
		branch := TreeBranch
		if i == len(commands)-1 {
			branch = TreeLastBranch
		}

		risk := safety.AssessRisk(command)
		label := string(risk)
		if safety.IsBlocked(command) {
			label += " (blocked)"
		}

		fmt.Printf("%s %s  %s %s\n",
			TreeStyle.Render(branch),
			HighlightCommand(Oneline(command)),
			safety.GetRiskEmoji(risk),
			getRiskStyle(string(risk)).Render(label),
		)
	}
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

// Diagnostics go to stderr so that piped or captured stdout only ever carries
// commands and command output.
func DisplayError(message string) {
	fmt.Fprintln(os.Stderr, DangerStyle.Render("  "+IconError+" "+message))
}

func DisplayWarning(message string) {
	fmt.Fprintln(os.Stderr, warningStyle.Render("  "+IconWarning+" "+message))
}
