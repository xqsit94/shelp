package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xqsit94/shelp/internal/safety"
)

type ConfirmChoice int

const (
	ConfirmExecute ConfirmChoice = iota
	ConfirmEdit
	ConfirmRegenerate
	ConfirmCancel
)

var confirmLabels = map[ConfirmChoice]string{
	ConfirmExecute:    "Execute",
	ConfirmEdit:       "Edit",
	ConfirmRegenerate: "Regenerate",
	ConfirmCancel:     "Cancel",
}

type confirmMode int

const (
	confirmModeMenu confirmMode = iota
	confirmModeEdit
	confirmModeRefine
)

type ConfirmResult struct {
	Choice     ConfirmChoice
	Command    string
	Refinement string
}

type confirmModel struct {
	command    string
	risk       safety.RiskLevel
	blocked    bool
	choices    []ConfirmChoice
	cursor     int
	selected   ConfirmChoice
	refinement string
	mode       confirmMode
	textInput  textinput.Model
	done       bool
}

func newConfirmModel(command string) confirmModel {
	ti := textinput.New()
	ti.Width = GetTerminalWidth() - 6

	m := confirmModel{
		command:   command,
		textInput: ti,
	}
	m.assess(command)

	return m
}

func (m *confirmModel) assess(command string) {
	m.command = command
	m.risk = safety.AssessRisk(command)
	m.blocked = safety.IsBlocked(command)
	m.cursor = 0

	m.choices = []ConfirmChoice{ConfirmExecute, ConfirmEdit, ConfirmRegenerate, ConfirmCancel}
	if m.blocked {
		m.choices = m.choices[1:]
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case confirmModeEdit:
		return m.updateEditMode(msg)
	case confirmModeRefine:
		return m.updateRefineMode(msg)
	}

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
			return m.choose(m.choices[m.cursor])
		case "y", "Y":
			if !m.blocked {
				return m.choose(ConfirmExecute)
			}
		case "e", "E":
			return m.choose(ConfirmEdit)
		case "r", "R":
			return m.choose(ConfirmRegenerate)
		case "n", "N", "q", "ctrl+c", "esc":
			return m.choose(ConfirmCancel)
		}
	}

	return m, nil
}

func (m confirmModel) choose(choice ConfirmChoice) (tea.Model, tea.Cmd) {
	switch choice {
	case ConfirmEdit:
		m.mode = confirmModeEdit
		m.textInput.Placeholder = "edit the command..."
		m.textInput.CharLimit = 0
		m.textInput.SetValue(m.command)
		m.textInput.CursorEnd()
		return m, tea.Batch(m.textInput.Focus(), textinput.Blink)
	case ConfirmRegenerate:
		m.mode = confirmModeRefine
		m.textInput.Placeholder = "add refinement here..."
		m.textInput.CharLimit = 200
		m.textInput.SetValue("")
		return m, tea.Batch(m.textInput.Focus(), textinput.Blink)
	default:
		m.selected = choice
		m.done = true
		return m, tea.Quit
	}
}

func (m confirmModel) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			if edited := strings.TrimSpace(m.textInput.Value()); edited != "" {
				m.assess(edited)
			}
			return m.backToMenu(), nil
		case "esc":
			return m.backToMenu(), nil
		case "ctrl+c":
			return m.choose(ConfirmCancel)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m confirmModel) updateRefineMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			m.refinement = strings.TrimSpace(m.textInput.Value())
			m.selected = ConfirmRegenerate
			m.done = true
			return m, tea.Quit
		case "esc":
			return m.backToMenu(), nil
		case "ctrl+c":
			return m.choose(ConfirmCancel)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m confirmModel) backToMenu() confirmModel {
	m.mode = confirmModeMenu
	m.textInput.SetValue("")
	m.textInput.Blur()
	return m
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}

	switch m.mode {
	case confirmModeEdit:
		return m.viewInput("Edit command", "enter: save • esc: cancel")
	case confirmModeRefine:
		return m.viewInput("Refine your request", "enter: regenerate • esc: cancel")
	}

	riskEmoji := safety.GetRiskEmoji(m.risk)
	riskStyle := getRiskStyle(string(m.risk))

	cmdTitleStyle := TitleBoldStyle.Foreground(ColorInfo)

	s := "\n"
	s += cmdTitleStyle.Render("Generated Command") + "\n"
	s += IndentUnder(TreeStyle.Render(TreeLastBranch)+" ", HighlightCommand(m.command)) + "\n"
	s += fmt.Sprintf("   %s %s\n", riskEmoji, riskStyle.Render(string(m.risk)))

	if m.blocked {
		s += DangerStyle.Render("   This command is blocked for safety reasons.") + "\n"
	}
	s += "\n"

	for i, choice := range m.choices {
		cursor := "  "
		style := unselectedStyle
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
			style = selectedStyle
		}
		s += cursor + style.Render(confirmLabels[choice]) + "\n"
	}

	help := "↑/↓: navigate • enter: select • y: execute • e: edit • r: regenerate • q: cancel"
	if m.blocked {
		help = "↑/↓: navigate • enter: select • e: edit • r: regenerate • q: cancel"
	}

	return s + "\n" + helpStyle.Render(help)
}

func (m confirmModel) viewInput(title, help string) string {
	var b strings.Builder

	b.WriteString("\n" + TitleBoldStyle.Foreground(ColorPrimary).Render(title) + "\n")
	b.WriteString(hintStyle.Render(fmt.Sprintf("  %s\n\n", Truncate(Oneline(m.command), 60))))
	b.WriteString("  " + m.textInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("  " + help))

	return b.String()
}

func ConfirmExecutionInteractive(cmd string) ConfirmResult {
	if !IsInteractive() {
		return ConfirmResult{Choice: ConfirmCancel, Command: cmd}
	}

	finalModel, err := tea.NewProgram(newConfirmModel(cmd)).Run()
	if err != nil {
		return ConfirmResult{Choice: ConfirmCancel, Command: cmd}
	}

	result := finalModel.(confirmModel)

	return ConfirmResult{
		Choice:     result.selected,
		Command:    result.command,
		Refinement: result.refinement,
	}
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
	if !IsInteractive() {
		return false
	}

	m := newConfirmYesNoModel(prompt)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false
	}

	return finalModel.(confirmYesNoModel).selected
}
