package prompt

import (
	"fmt"
	"strings"
)

const (
	maxRefineCommands    = 6
	maxRefineRefinements = 3
	refineChromeRows     = 11
	refineLabelWidth     = 10
)

// refineContext is the round being refined: what was asked, what has already
// been added to it, and what came back. It is shown while the next refinement
// is typed, because that request is judged against those commands.
type refineContext struct {
	Query       string
	Refinements []string
	Commands    []string
}

func (c refineContext) render(width, height int) string {
	var b strings.Builder

	if c.Query != "" {
		writeLine(&b, refineLine("Original", c.Query, width))
	}

	shown, hidden := lastN(c.Refinements, maxRefineRefinements)
	if hidden > 0 {
		writeLine(&b, refineNote(fmt.Sprintf("%d earlier refinements", hidden), width))
	}
	for _, refinement := range shown {
		writeLine(&b, refineLine("Refined", refinement, width))
	}

	if len(c.Commands) > 0 {
		b.WriteByte('\n')
		writeLine(&b, TruncateLines(hintStyle.Render("  It suggested:"), width))
		for _, line := range c.commandLines(height) {
			writeLine(&b, TruncateLines(line, width))
		}
	}

	return b.String()
}

func (c refineContext) commandLines(height int) []string {
	shown, hidden := firstN(c.Commands, c.commandBudget(height))

	lines := make([]string, 0, len(shown)+1)
	for i, command := range shown {
		branch := TreeBranch
		if hidden == 0 && i == len(shown)-1 {
			branch = TreeLastBranch
		}
		lines = append(lines, "    "+TreeStyle.Render(branch)+" "+unselectedStyle.Render(Oneline(command)))
	}
	if hidden > 0 {
		lines = append(lines, "    "+hintStyle.Render(fmt.Sprintf("… %d more", hidden)))
	}

	return lines
}

// commandBudget keeps the input and its key hints on screen: the context is
// what gets cut when the terminal is short, never the field being typed into.
func (c refineContext) commandBudget(height int) int {
	if height <= 0 {
		return maxRefineCommands
	}

	budget := height - refineChromeRows - min(len(c.Refinements), maxRefineRefinements)

	return min(max(budget, minVisibleRows), maxRefineCommands)
}

func refineLine(label, value string, width int) string {
	text := fmt.Sprintf("  %-*s%q", refineLabelWidth, label+":", Truncate(value, maxQueryPreview))
	return TruncateLines(hintStyle.Render(text), width)
}

func refineNote(text string, width int) string {
	return TruncateLines(hintStyle.Render("  "+strings.Repeat(" ", refineLabelWidth)+"… "+text), width)
}

func firstN(values []string, n int) (shown []string, hidden int) {
	if len(values) <= n {
		return values, 0
	}
	return values[:n], len(values) - n
}

func lastN(values []string, n int) (shown []string, hidden int) {
	if len(values) <= n {
		return values, 0
	}
	return values[len(values)-n:], len(values) - n
}
