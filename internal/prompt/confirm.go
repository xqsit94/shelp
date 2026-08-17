package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	command       string
	explanation   string
	originalQuery string
	refinements   []string
	risk          safety.RiskLevel
	blocked       bool
	choices       []ConfirmChoice
	cursor        int
	selected      ConfirmChoice
	refinement    string
	mode          confirmMode
	textInput     textinput.Model
	done          bool
	keys          confirmKeyMap
	editKeys      inputKeyMap
	refineKeys    inputKeyMap
	help          help.Model
	width         int
	height        int
}

func newConfirmModel(suggestion Suggestion) confirmModel {
	width := GetTerminalWidth()

	ti := textinput.New()
	ti.Width = width - 6

	h := newHelpModel(width)

	m := confirmModel{
		command:     suggestion.Command,
		explanation: suggestion.Explanation,
		textInput:   ti,
		keys:        defaultConfirmKeyMap(),
		editKeys:    newInputKeyMap("save"),
		refineKeys:  newInputKeyMap("regenerate"),
		help:        h,
		width:       width,
	}
	m.assess(suggestion.Command)

	return m
}

func (m confirmModel) withRefineContext(originalQuery string, refinements []string) confirmModel {
	m.originalQuery = originalQuery
	m.refinements = refinements
	return m
}

func (m confirmModel) setSize(width, height int) confirmModel {
	m.width = width
	m.height = height
	m.help.Width = width
	m.textInput.Width = width - 6
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
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		return m.setSize(size.Width, size.Height), nil
	}

	switch m.mode {
	case confirmModeEdit:
		return m.updateEditMode(msg)
	case confirmModeRefine:
		return m.updateRefineMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Select):
			return m.choose(m.choices[m.cursor])
		case key.Matches(msg, m.keys.Execute):
			if !m.blocked {
				return m.choose(ConfirmExecute)
			}
		case key.Matches(msg, m.keys.Edit):
			return m.choose(ConfirmEdit)
		case key.Matches(msg, m.keys.Regenerate):
			return m.choose(ConfirmRegenerate)
		case key.Matches(msg, m.keys.Quit):
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
				m.explanation = ""
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
		return m.viewEdit()
	case confirmModeRefine:
		return m.viewRefine()
	}

	riskEmoji := safety.GetRiskEmoji(m.risk)
	riskStyle := getRiskStyle(string(m.risk))

	cmdTitleStyle := TitleBoldStyle.Foreground(ColorInfo)

	s := "\n"
	s += cmdTitleStyle.Render("Generated Command") + "\n"
	commandLine := IndentUnder(TreeStyle.Render(TreeLastBranch)+" ", HighlightCommand(m.command))
	s += TruncateLines(commandLine, m.width) + "\n"
	riskLine := fmt.Sprintf("   %s %s", riskEmoji, riskStyle.Render(string(m.risk)))
	if m.explanation != "" {
		riskLine += ExplanationStyle.Render(" — " + m.explanation)
	}
	s += Truncate(riskLine, m.width) + "\n"

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

	return s + "\n" + indentBlock(
		TruncateLines(m.help.ShortHelpView(m.keys.shortHelpFor(m.blocked)), m.width-helpIndent),
		helpIndent,
	)
}

func (m confirmModel) viewEdit() string {
	var b strings.Builder

	b.WriteByte('\n')
	writeLine(&b, TitleBoldStyle.Foreground(ColorPrimary).Render("Edit command"))
	writeLine(&b, TruncateLines(hintStyle.Render("  "+Truncate(Oneline(m.command), maxQueryPreview)), m.width))

	b.WriteByte('\n')
	writeLine(&b, "  "+m.textInput.View())

	b.WriteByte('\n')
	b.WriteString(renderHelp(m.help, m.editKeys, m.width))

	return b.String()
}

func (m confirmModel) viewRefine() string {
	var b strings.Builder

	b.WriteByte('\n')
	writeLine(&b, TitleBoldStyle.Foreground(ColorPrimary).Render("Refine your request"))
	b.WriteString(refineContext{
		Query:       m.originalQuery,
		Refinements: m.refinements,
		Commands:    []string{m.command},
	}.render(m.width, m.height))

	b.WriteByte('\n')
	writeLine(&b, TruncateLines(infoStyle.Render("  Add to your request (or press Enter to retry):"), m.width))
	writeLine(&b, "  "+m.textInput.View())

	b.WriteByte('\n')
	b.WriteString(renderHelp(m.help, m.refineKeys, m.width))

	return b.String()
}

func ConfirmExecutionInteractive(suggestion Suggestion, originalQuery string, refinements []string) ConfirmResult {
	if !IsInteractive() {
		return ConfirmResult{Choice: ConfirmCancel, Command: suggestion.Command}
	}

	model := newConfirmModel(suggestion).withRefineContext(originalQuery, refinements)

	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return ConfirmResult{Choice: ConfirmCancel, Command: suggestion.Command}
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
	keys     yesNoKeyMap
	help     help.Model
	width    int
}

func newConfirmYesNoModel(prompt string) confirmYesNoModel {
	width := GetTerminalWidth()
	h := newHelpModel(width)

	return confirmYesNoModel{
		width:   width,
		prompt:  prompt,
		choices: []string{"Yes", "No"},
		cursor:  0,
		keys:    defaultYesNoKeyMap(),
		help:    h,
	}
}

func (m confirmYesNoModel) Init() tea.Cmd {
	return nil
}

func (m confirmYesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.help.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Left):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Right):
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Select):
			m.selected = m.cursor == 0
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Yes):
			m.selected = true
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.No):
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

	s += "\n\n" + renderHelp(m.help, m.keys, m.width)

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
