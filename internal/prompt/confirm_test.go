package prompt

import (
	"strings"
	"testing"

	"github.com/xqsit94/shelp/internal/safety"
)

func TestConfirmChoiceOrder(t *testing.T) {
	m := newConfirmModel(Suggestion{Command: "ls -la"})

	want := []ConfirmChoice{ConfirmExecute, ConfirmEdit, ConfirmRegenerate, ConfirmCancel}
	if len(m.choices) != len(want) {
		t.Fatalf("choices = %v, want %v", m.choices, want)
	}
	for i, choice := range want {
		if m.choices[i] != choice {
			t.Errorf("choices[%d] = %v, want %v", i, m.choices[i], choice)
		}
		if confirmLabels[choice] == "" {
			t.Errorf("choice %v has no label", choice)
		}
	}
}

func TestConfirmExecuteOnEnter(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls -la"}), enter)

	if !m.done || m.selected != ConfirmExecute {
		t.Errorf("selected = %v, done = %v, want Execute", m.selected, m.done)
	}
}

func TestConfirmEdit(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls"}), key("e"), key(" -la"), enter)

	if m.command != "ls -la" {
		t.Errorf("command = %q, want %q", m.command, "ls -la")
	}
	if m.done {
		t.Error("model quit instead of returning to the menu")
	}
	if m.mode != confirmModeMenu {
		t.Errorf("mode = %v, want the menu", m.mode)
	}

	m = send(t, m, key("y"))

	if !m.done || m.selected != ConfirmExecute {
		t.Errorf("selected = %v, done = %v, want Execute", m.selected, m.done)
	}
}

func TestConfirmEditEscapeKeepsCommand(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls"}), key("e"), key(" -la"), esc)

	if m.command != "ls" {
		t.Errorf("command = %q, want it unchanged", m.command)
	}
	if m.mode != confirmModeMenu {
		t.Errorf("mode = %v, want the menu", m.mode)
	}
}

func TestConfirmEditIntoBlockedCommand(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls"}), key("e"), key(" && rm -rf /"), enter)

	if !m.blocked {
		t.Fatal("blocked = false after editing into a blocked command")
	}
	if m.risk != safety.RiskDanger {
		t.Errorf("risk = %q, want %q", m.risk, safety.RiskDanger)
	}
	if m.choices[0] != ConfirmEdit {
		t.Errorf("choices = %v, want Execute removed", m.choices)
	}

	m = send(t, m, key("y"))

	if m.done {
		t.Error("y executed a blocked command")
	}
}

func TestConfirmBlockedCommandMenu(t *testing.T) {
	m := newConfirmModel(Suggestion{Command: "rm -rf /"})

	if !m.blocked {
		t.Fatal("blocked = false for a blocked command")
	}

	m = send(t, m, enter)

	if m.mode != confirmModeEdit {
		t.Errorf("mode = %v, want the editor (Edit is the first choice)", m.mode)
	}
}

func TestConfirmRegenerateWithRefinement(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls"}), key("r"), key("use find"), enter)

	if !m.done || m.selected != ConfirmRegenerate {
		t.Fatalf("selected = %v, done = %v, want Regenerate", m.selected, m.done)
	}
	if m.refinement != "use find" {
		t.Errorf("refinement = %q, want %q", m.refinement, "use find")
	}
}

func TestConfirmRegenerateFromMenuCursor(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls"}), down, down, enter)

	if m.mode != confirmModeRefine {
		t.Fatalf("mode = %v, want the refinement input", m.mode)
	}

	m = send(t, m, enter)

	if !m.done || m.selected != ConfirmRegenerate || m.refinement != "" {
		t.Errorf("selected = %v, refinement = %q, want Regenerate with no refinement", m.selected, m.refinement)
	}
}

func TestConfirmCancel(t *testing.T) {
	m := send(t, newConfirmModel(Suggestion{Command: "ls"}), key("q"))

	if !m.done || m.selected != ConfirmCancel {
		t.Errorf("selected = %v, done = %v, want Cancel", m.selected, m.done)
	}
}

func TestConfirmViewShowsExplanation(t *testing.T) {
	m := newConfirmModel(Suggestion{Command: "du -sh .", Explanation: "Shows the size of this directory"})

	if view := m.View(); !strings.Contains(view, "Shows the size of this directory") {
		t.Errorf("view does not show the explanation:\n%s", view)
	}
}

func TestConfirmEditDropsExplanation(t *testing.T) {
	m := newConfirmModel(Suggestion{Command: "ls", Explanation: "Lists files"})

	m = send(t, m, key("e"), key(" -la"), enter)

	if m.explanation != "" {
		t.Errorf("explanation = %q, want it dropped after an edit", m.explanation)
	}
	if view := m.View(); strings.Contains(view, "Lists files") {
		t.Errorf("view still shows the old explanation:\n%s", view)
	}
}
