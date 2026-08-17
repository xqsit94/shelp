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

// DisplayCommandPlan prints one tree row per command with its risk level and
// explanation, so unattended runs still show what is about to happen.
func DisplayCommandPlan(suggestions []Suggestion) {
	fmt.Println()
	fmt.Println(TitleBoldStyle.Foreground(ColorInfo).Render(fmt.Sprintf("Generated Commands (%d)", len(suggestions))))

	for i, suggestion := range suggestions {
		branch := TreeBranch
		if i == len(suggestions)-1 {
			branch = TreeLastBranch
		}

		risk := safety.AssessRisk(suggestion.Command)
		label := string(risk)
		if safety.IsBlocked(suggestion.Command) {
			label += " (blocked)"
		}

		row := fmt.Sprintf("%s %s  %s %s",
			TreeStyle.Render(branch),
			HighlightCommand(Oneline(suggestion.Command)),
			safety.GetRiskEmoji(risk),
			getRiskStyle(string(risk)).Render(label),
		)
		if suggestion.Explanation != "" {
			row += ExplanationStyle.Render(" — " + suggestion.Explanation)
		}

		fmt.Println(Truncate(row, GetTerminalWidth()))
	}
}

func DisplayRunning(index, total int, command string) {
	branch := TreeBranch
	if index == total {
		branch = TreeLastBranch
	}

	prefix := fmt.Sprintf("%s %s ",
		TreeStyle.Render(branch),
		hintStyle.Render(fmt.Sprintf("[%d/%d]", index, total)),
	)

	fmt.Println(IndentUnder(prefix, HighlightCommand(command)))
	fmt.Println()
}

// DisplayStepResult reports how a command ended as soon as it ends. Without it
// a failure is only visible in the closing summary, so a mid-run "continue?"
// question arrives with no indication of what went wrong.
func DisplayStepResult(exitCode int, interrupted bool, err error) {
	switch {
	case err != nil:
		DisplayError("Failed to run: " + err.Error())
	case interrupted:
		DisplayWarning("Interrupted.")
	case exitCode != 0:
		DisplayError(fmt.Sprintf("Exited with code %d.", exitCode))
	}
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

// DisplayHint follows an error with the thing to try next.
func DisplayHint(message string) {
	fmt.Fprintln(os.Stderr, hintStyle.Render("    "+message))
}
