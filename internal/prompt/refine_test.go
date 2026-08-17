package prompt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func refineView(t *testing.T, suggestions []Suggestion, query string, refinements []string, width, height int) string {
	t.Helper()

	m := newCommandListModel(suggestions, query).withRefinements(refinements).setSize(width, height)

	return ansi.Strip(send(t, m, typed("r")).View())
}

func TestRefineShowsWhatWasSuggested(t *testing.T) {
	view := refineView(t, suggested("docker compose down", "docker compose up -d"), "restart the stack", nil, 100, 30)

	for _, want := range []string{"restart the stack", "docker compose down", "docker compose up -d"} {
		if !strings.Contains(view, want) {
			t.Errorf("refine view does not mention %q:\n%s", want, view)
		}
	}
}

func TestRefineShowsEarlierRefinements(t *testing.T) {
	view := refineView(t, suggested("ls"), "list files", []string{"use exa instead", "sort by size"}, 100, 30)

	for _, want := range []string{"list files", "use exa instead", "sort by size"} {
		if !strings.Contains(view, want) {
			t.Errorf("refine view does not mention %q:\n%s", want, view)
		}
	}
}

func TestRefineSummarisesRefinementsBeyondTheLimit(t *testing.T) {
	refinements := make([]string, maxRefineRefinements+2)
	for i := range refinements {
		refinements[i] = fmt.Sprintf("refinement-%d", i)
	}

	view := refineView(t, suggested("ls"), "list files", refinements, 100, 40)

	if !strings.Contains(view, "2 earlier refinements") {
		t.Errorf("refine view does not summarise the dropped refinements:\n%s", view)
	}
	if !strings.Contains(view, "refinement-4") {
		t.Errorf("refine view dropped the newest refinement:\n%s", view)
	}
	if strings.Contains(view, "refinement-0") {
		t.Errorf("refine view kept an oldest refinement it should have summarised:\n%s", view)
	}
}

func TestRefineKeepsTheInputVisibleWhenShort(t *testing.T) {
	suggestions := make([]Suggestion, 12)
	for i := range suggestions {
		suggestions[i] = Suggestion{Command: fmt.Sprintf("echo step-%02d", i)}
	}

	const height = 16
	view := refineView(t, suggestions, "do things", []string{"quietly"}, 100, height)

	if lines := strings.Split(view, "\n"); len(lines) > height {
		t.Errorf("refine view is %d lines, want at most %d:\n%s", len(lines), height, view)
	}
	if !strings.Contains(view, "more") {
		t.Errorf("refine view hid commands without saying so:\n%s", view)
	}
	for _, want := range []string{"Add to your request", "enter", "regenerate"} {
		if !strings.Contains(view, want) {
			t.Errorf("refine view dropped %q to make room for context:\n%s", want, view)
		}
	}
}

func TestRefineFitsEveryWidth(t *testing.T) {
	suggestions := []Suggestion{
		{Command: "docker compose up -d --force-recreate --remove-orphans --timeout 120"},
		{Command: "rm -rf /"},
	}
	refinements := []string{strings.Repeat("refined ", 20)}

	for _, width := range []int{40, 60, 80, 100, 160} {
		views := map[string]string{
			"list": refineView(t, suggestions, strings.Repeat("query ", 30), refinements, width, 24),
			"confirm": ansi.Strip(send(t,
				newConfirmModel(suggestions[0]).
					withRefineContext(strings.Repeat("query ", 30), refinements).
					setSize(width, 24),
				typed("r"),
			).View()),
		}

		for name, view := range views {
			for i, line := range strings.Split(view, "\n") {
				if got := ansi.StringWidth(line); got > width {
					t.Errorf("%s refine view at width %d: line %d is %d wide: %q", name, width, i+1, got, line)
				}
			}
		}
	}
}

func TestRefineContextCommandBudget(t *testing.T) {
	tests := []struct {
		name        string
		refinements int
		height      int
		want        int
	}{
		{"no size known yet", 0, 0, maxRefineCommands},
		{"tall terminal", 0, 40, maxRefineCommands},
		{"room for three", 0, refineChromeRows + 3, 3},
		{"refinements take room from commands", 2, refineChromeRows + 3, 1},
		{"never drops below one", 0, 1, minVisibleRows},
		{"more refinements than shown do not count twice", 10, refineChromeRows + 5, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := refineContext{Refinements: make([]string, tt.refinements)}
			if got := c.commandBudget(tt.height); got != tt.want {
				t.Errorf("commandBudget(%d) with %d refinements = %d, want %d",
					tt.height, tt.refinements, got, tt.want)
			}
		})
	}
}

func TestConfirmRefineShowsTheOriginalQuery(t *testing.T) {
	m := newConfirmModel(Suggestion{Command: "ls"}).
		withRefineContext("list files", []string{"sort by size"}).
		setSize(100, 30)

	view := ansi.Strip(send(t, m, typed("r")).View())

	for _, want := range []string{"list files", "sort by size", "ls"} {
		if !strings.Contains(view, want) {
			t.Errorf("confirm refine view does not mention %q:\n%s", want, view)
		}
	}
}

func TestConfirmRefineWithoutAQueryOmitsTheOriginalLine(t *testing.T) {
	m := newConfirmModel(Suggestion{Command: "ls"}).setSize(100, 30)

	view := ansi.Strip(send(t, m, typed("r")).View())

	if strings.Contains(view, "Original:") {
		t.Errorf("confirm refine view shows an empty original query:\n%s", view)
	}
	if !strings.Contains(view, "ls") {
		t.Errorf("confirm refine view does not mention the command:\n%s", view)
	}
}
