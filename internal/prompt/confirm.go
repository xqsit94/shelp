package prompt

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xqsit94/shelp/internal/safety"
)

type ConfirmChoice int

const (
	ConfirmExecute ConfirmChoice = iota
	ConfirmRegenerate
	ConfirmSkip
	ConfirmCancel
)

type confirmModel struct {
	command  string
	risk     safety.RiskLevel
	choices  []string
	cursor   int
	selected ConfirmChoice
	done     bool
}

func newConfirmModel(command string) confirmModel {
	risk := safety.AssessRisk(command)
	return confirmModel{
		command: command,
		risk:    risk,
		choices: []string{"Execute", "Regenerate", "Cancel"},
		cursor:  0,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = ConfirmChoice(m.cursor)
			m.done = true
			return m, tea.Quit
		case "y", "Y":
			m.selected = ConfirmExecute
			m.done = true
			return m, tea.Quit
		case "r", "R":
			m.selected = ConfirmRegenerate
			m.done = true
			return m, tea.Quit
		case "n", "N", "q", "ctrl+c", "esc":
			m.selected = ConfirmCancel
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}

	riskEmoji := safety.GetRiskEmoji(m.risk)
	riskStyle := getRiskStyle(string(m.risk))

	s := "\n"
	s += RenderCommandBox("Generated Command", m.command)
	s += "\n"
	s += fmt.Sprintf("Risk: %s %s\n\n", riskEmoji, riskStyle.Render(string(m.risk)))

	choiceIcons := []string{"▶", "↻", "×"}
	for i, choice := range m.choices {
		cursor := "  "
		style := unselectedStyle
		icon := hintStyle.Render(choiceIcons[i])
		if m.cursor == i {
			cursor = cursorStyle.Render("› ")
			style = selectedStyle
			icon = cursorStyle.Render(choiceIcons[i])
		}
		s += cursor + icon + " " + style.Render(choice) + "\n"
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • enter: select • y: execute • r: regenerate • q: cancel")

	return s
}

func ConfirmExecutionInteractive(cmd string) ConfirmChoice {
	if safety.IsBlocked(cmd) {
		fmt.Println(dangerStyle.Render("This command has been blocked for safety reasons."))
		return ConfirmCancel
	}

	m := newConfirmModel(cmd)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ConfirmCancel
	}

	return finalModel.(confirmModel).selected
}

type confirmYesNoModel struct {
	prompt   string
	cursor   int
	choices  []string
	selected bool
	done     bool
}

func newConfirmYesNoModel(prompt string) confirmYesNoModel {
	return confirmYesNoModel{
		prompt:  prompt,
		choices: []string{"Yes", "No"},
		cursor:  0,
	}
}

func (m confirmYesNoModel) Init() tea.Cmd {
	return nil
}

func (m confirmYesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor == 0
			m.done = true
			return m, tea.Quit
		case "y", "Y":
			m.selected = true
			m.done = true
			return m, tea.Quit
		case "n", "N", "q", "ctrl+c", "esc":
			m.selected = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmYesNoModel) View() string {
	if m.done {
		return ""
	}

	s := m.prompt + "\n\n"

	for i, choice := range m.choices {
		style := unselectedStyle
		if m.cursor == i {
			style = selectedStyle
		}
		if i == m.cursor {
			s += cursorStyle.Render("[") + style.Render(choice) + cursorStyle.Render("]")
		} else {
			s += " " + style.Render(choice) + " "
		}
		s += "  "
	}

	s += "\n\n" + helpStyle.Render("←/→: navigate • enter: select • y/n: quick select")

	return s
}

func ConfirmYesNoInteractive(prompt string) bool {
	m := newConfirmYesNoModel(prompt)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false
	}

	return finalModel.(confirmYesNoModel).selected
}
