package prompt

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/xqsit94/shelp/internal/safety"
)

func typed(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	down  = tea.KeyMsg{Type: tea.KeyDown}
)

func send[M tea.Model](t *testing.T, model M, msgs ...tea.Msg) M {
	t.Helper()

	var updated tea.Model = model
	for _, msg := range msgs {
		updated, _ = updated.Update(msg)
	}

	final, ok := updated.(M)
	if !ok {
		t.Fatalf("Update returned %T", updated)
	}

	return final
}

// suggested builds a list without explanations, which most tests do not care
// about.
func suggested(commands ...string) []Suggestion {
	suggestions := make([]Suggestion, len(commands))
	for i, command := range commands {
		suggestions[i] = Suggestion{Command: command}
	}
	return suggestions
}

func TestCommandListEdit(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, typed("e"), typed(" -la"), enter)

	if got := m.commands[0].Command; got != "ls -la" {
		t.Errorf("edited command = %q, want %q", got, "ls -la")
	}
	if !m.commands[0].Selected {
		t.Error("edited command was deselected")
	}
	if m.mode != listModeSelect {
		t.Errorf("mode = %v, want the list back", m.mode)
	}
	if m.commands[1].Command != "pwd" {
		t.Errorf("other command changed to %q", m.commands[1].Command)
	}
}

func TestCommandListEditReassessesRisk(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, down, typed("e"), typed(" && sudo rm -r /tmp/x"), enter)

	if got := m.commands[1].Risk; got != safety.RiskCaution {
		t.Errorf("risk = %q, want %q", got, safety.RiskCaution)
	}
	if !m.commands[1].Selected {
		t.Error("caution command was deselected")
	}
}

func TestCommandListEditBlockedDeselects(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, typed("e"), typed(" && rm -rf /"), enter)

	if got := m.commands[0].Command; got != "ls && rm -rf /" {
		t.Errorf("edited command = %q", got)
	}
	if m.commands[0].Risk != safety.RiskDanger {
		t.Errorf("risk = %q, want %q", m.commands[0].Risk, safety.RiskDanger)
	}
	if m.commands[0].Selected {
		t.Error("blocked command stayed selected")
	}
}

func TestCommandListEditEscapeKeepsCommand(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, typed("e"), typed(" -la"), esc)

	if got := m.commands[0].Command; got != "ls" {
		t.Errorf("command = %q, want it unchanged", got)
	}
	if m.mode != listModeSelect {
		t.Errorf("mode = %v, want the list back", m.mode)
	}
}

func TestCommandListRegenerateRefinement(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, typed("r"), typed("use find instead"), enter)

	if !m.regenerate {
		t.Fatal("regenerate = false, want true")
	}
	if got := m.textInput.Value(); got != "use find instead" {
		t.Errorf("refinement = %q, want %q", got, "use find instead")
	}
}

func TestCommandListBlockedStartsDeselected(t *testing.T) {
	m := newCommandListModel(suggested("rm -rf /", "ls"), "clean up")

	if m.commands[0].Selected {
		t.Error("blocked command started selected")
	}

	m = send(t, m, typed(" "))

	if m.commands[0].Selected {
		t.Error("blocked command could be toggled on")
	}
}

func TestCommandListEditDropsExplanation(t *testing.T) {
	m := newCommandListModel([]Suggestion{
		{Command: "ls", Explanation: "Lists files"},
		{Command: "pwd", Explanation: "Prints the working directory"},
	}, "list files")

	m = send(t, m, typed("e"), typed(" -la"), enter)

	if m.commands[0].Explanation != "" {
		t.Errorf("explanation = %q, want it dropped after an edit", m.commands[0].Explanation)
	}
	if want := "Prints the working directory"; m.commands[1].Explanation != want {
		t.Errorf("other explanation = %q, want %q", m.commands[1].Explanation, want)
	}
}

func TestCommandListViewShowsExplanation(t *testing.T) {
	m := newCommandListModel([]Suggestion{
		{Command: "ls -a", Explanation: "Lists files including hidden ones"},
		{Command: "pwd"},
	}, "list files")

	view := m.View()

	if !strings.Contains(view, "Lists files including hidden ones") {
		t.Errorf("view does not show the explanation:\n%s", view)
	}
	if strings.Contains(view, "safe — \n") {
		t.Errorf("view shows an empty explanation:\n%s", view)
	}
}

// The focused row has to be identifiable from the text alone: under NO_COLOR
// a colour-only cursor leaves the two views byte-identical.
func TestCommandListFocusIsVisibleWithoutColour(t *testing.T) {
	m := newCommandListModel([]Suggestion{
		{Command: "echo one"},
		{Command: "echo two"},
	}, "echo things")

	first := ansi.Strip(m.View())

	moved, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	second := ansi.Strip(moved.(commandListModel).View())

	if first == second {
		t.Errorf("moving the cursor did not change the plain-text view:\n%s", first)
	}
	if !strings.Contains(first, "❯ ") {
		t.Errorf("view has no cursor marker:\n%s", first)
	}
}

// A command longer than the terminal must never be shown as a silent prefix of
// itself: the user is approving exactly this text.
func TestCommandListTruncatesLongCommandsWithEllipsis(t *testing.T) {
	long := "find /var/log -type f -name '*.log' " + strings.Repeat("-o -name '*.gz' ", 20)
	m := newCommandListModel([]Suggestion{{Command: long}}, "find logs")

	width := GetTerminalWidth()

	var commandRow string
	for _, line := range strings.Split(m.View(), "\n") {
		if strings.Contains(line, "find /var/log") {
			commandRow = line
			break
		}
	}

	if commandRow == "" {
		t.Fatalf("command row not found in view:\n%s", m.View())
	}
	if got := ansi.StringWidth(commandRow); got > width {
		t.Errorf("command row width = %d, want at most %d", got, width)
	}
	if !strings.HasSuffix(ansi.Strip(commandRow), "…") {
		t.Errorf("truncated command row does not end in an ellipsis: %q", ansi.Strip(commandRow))
	}
}

// The bindings a user cannot finish the screen without have to survive the
// narrowest terminal we support.
func TestCommandListHelpKeepsEssentialKeysAtEveryWidth(t *testing.T) {
	m := newCommandListModel([]Suggestion{
		{Command: "echo one"},
		{Command: "echo two"},
	}, "echo things")

	for _, width := range []int{40, 60, 80, 100, 160} {
		sized := m.setSize(width, 24)
		help := ansi.Strip(sized.helpView(sized.keys))

		for _, want := range []string{"enter", "q"} {
			if !strings.Contains(help, want) {
				t.Errorf("width %d: help %q does not mention %q", width, help, want)
			}
		}
		if got := ansi.StringWidth(help); got > width {
			t.Errorf("width %d: help row width = %d", width, got)
		}
	}
}

// Every view has to stay inside the terminal at any width: the renderer clips
// silently, so an overflowing line is invisible rather than obviously wrong.
func TestViewsFitEveryWidth(t *testing.T) {
	suggestions := []Suggestion{
		{Command: "docker compose up -d --force-recreate --remove-orphans", Explanation: strings.Repeat("long ", 30)},
		{Command: "rm -rf /", Explanation: "Deletes everything"},
	}

	for _, width := range []int{40, 60, 80, 100, 160} {
		views := map[string]string{
			"list":    newCommandListModel(suggestions, strings.Repeat("query ", 30)).setSize(width, 24).View(),
			"confirm": newConfirmModel(suggestions[0]).setSize(width, 24).View(),
		}

		for name, view := range views {
			for i, line := range strings.Split(view, "\n") {
				if got := ansi.StringWidth(line); got > width {
					t.Errorf("%s view at width %d: line %d is %d wide: %q",
						name, width, i+1, got, ansi.Strip(line))
				}
			}
		}
	}
}

func TestVisibleRange(t *testing.T) {
	tests := []struct {
		name                string
		count, cursor, rows int
		wantStart, wantEnd  int
	}{
		{"everything fits", 3, 0, 20, 0, 3},
		{"no size known yet", 12, 5, 0, 0, 12},
		{"window follows cursor", 12, 5, 10, 3, 8},
		{"clamped at the top", 12, 0, 10, 0, 5},
		{"clamped at the bottom", 12, 11, 10, 7, 12},
		{"one row still shows one", 12, 6, 1, 6, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleRange(tt.count, tt.cursor, tt.rows)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("visibleRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.count, tt.cursor, tt.rows, start, end, tt.wantStart, tt.wantEnd)
			}
			if tt.cursor < start || tt.cursor >= end {
				t.Errorf("cursor %d is outside the visible window [%d, %d)", tt.cursor, start, end)
			}
		})
	}
}

// A list taller than the terminal used to push its own title and first items
// off the screen for good.
func TestCommandListScrollsInsteadOfOverflowing(t *testing.T) {
	suggestions := make([]Suggestion, 30)
	for i := range suggestions {
		suggestions[i] = Suggestion{Command: fmt.Sprintf("echo step-%02d", i)}
	}

	const height = 20
	m := newCommandListModel(suggestions, "do things").setSize(100, height)

	for _, cursor := range []int{0, 15, 29} {
		m.cursor = cursor
		view := m.View()
		lines := strings.Split(view, "\n")

		if len(lines) > height {
			t.Errorf("cursor %d: view is %d lines, want at most %d", cursor, len(lines), height)
		}
		if !strings.Contains(view, "Generated Commands (30)") {
			t.Errorf("cursor %d: title scrolled out of the view", cursor)
		}

		want := fmt.Sprintf("echo step-%02d", cursor)
		if !strings.Contains(ansi.Strip(view), "❯ "+TreeBranch+" [✓] "+want) &&
			!strings.Contains(ansi.Strip(view), "❯ "+TreeLastBranch+" [✓] "+want) {
			t.Errorf("cursor %d: focused row %q is not visible:\n%s", cursor, want, ansi.Strip(view))
		}
	}
}

func TestCommandListViewFitsTerminalWidth(t *testing.T) {
	m := newCommandListModel([]Suggestion{
		{Command: "ls", Explanation: strings.Repeat("very long explanation ", 20)},
	}, "list files")

	width := GetTerminalWidth()

	rows := 0
	for _, line := range strings.Split(m.View(), "\n") {
		if !strings.Contains(line, "very long explanation") {
			continue
		}
		rows++
		if got := ansi.StringWidth(line); got > width {
			t.Errorf("explanation row width = %d, want at most %d: %q", got, width, line)
		}
	}

	if rows != 1 {
		t.Fatalf("found %d explanation rows, want 1", rows)
	}
}
