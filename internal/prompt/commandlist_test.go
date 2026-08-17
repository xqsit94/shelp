package prompt

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/xqsit94/shelp/internal/safety"
)

func key(s string) tea.KeyMsg {
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

	m = send(t, m, key("e"), key(" -la"), enter)

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

	m = send(t, m, down, key("e"), key(" && sudo rm -r /tmp/x"), enter)

	if got := m.commands[1].Risk; got != safety.RiskCaution {
		t.Errorf("risk = %q, want %q", got, safety.RiskCaution)
	}
	if !m.commands[1].Selected {
		t.Error("caution command was deselected")
	}
}

func TestCommandListEditBlockedDeselects(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, key("e"), key(" && rm -rf /"), enter)

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

	m = send(t, m, key("e"), key(" -la"), esc)

	if got := m.commands[0].Command; got != "ls" {
		t.Errorf("command = %q, want it unchanged", got)
	}
	if m.mode != listModeSelect {
		t.Errorf("mode = %v, want the list back", m.mode)
	}
}

func TestCommandListRegenerateRefinement(t *testing.T) {
	m := newCommandListModel(suggested("ls", "pwd"), "list files")

	m = send(t, m, key("r"), key("use find instead"), enter)

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

	m = send(t, m, key(" "))

	if m.commands[0].Selected {
		t.Error("blocked command could be toggled on")
	}
}

func TestCommandListEditDropsExplanation(t *testing.T) {
	m := newCommandListModel([]Suggestion{
		{Command: "ls", Explanation: "Lists files"},
		{Command: "pwd", Explanation: "Prints the working directory"},
	}, "list files")

	m = send(t, m, key("e"), key(" -la"), enter)

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
