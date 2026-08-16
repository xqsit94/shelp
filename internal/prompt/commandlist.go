package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xqsit94/shelp/internal/safety"
)

// Suggestion mirrors ai.Suggestion so that the prompt package stays free of
// the AI client.
type Suggestion struct {
	Command     string
	Explanation string
}

type CommandItem struct {
	Command     string
	Explanation string
	Risk        safety.RiskLevel
	Selected    bool
}

type listMode int

const (
	listModeSelect listMode = iota
	listModeRegenerate
	listModeEdit
)

type commandListModel struct {
	commands      []CommandItem
	cursor        int
	confirmed     bool
	cancelled     bool
	regenerate    bool
	mode          listMode
	originalQuery string
	textInput     textinput.Model
}

func newCommandListModel(suggestions []Suggestion, originalQuery string) commandListModel {
	items := make([]CommandItem, len(suggestions))
	for i, suggestion := range suggestions {
		items[i] = CommandItem{
			Command:     suggestion.Command,
			Explanation: suggestion.Explanation,
			Risk:        safety.AssessRisk(suggestion.Command),
			Selected:    !safety.IsBlocked(suggestion.Command),
		}
	}

	ti := textinput.New()
	ti.Placeholder = "add refinement here..."
	ti.CharLimit = 200
	ti.Width = GetTerminalWidth() - 6

	return commandListModel{
		commands:      items,
		cursor:        0,
		originalQuery: originalQuery,
		textInput:     ti,
	}
}

func (m commandListModel) Init() tea.Cmd {
	return nil
}

func (m commandListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case listModeRegenerate:
		return m.updateRegenerateMode(msg)
	case listModeEdit:
		return m.updateEditMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.commands)-1 {
				m.cursor++
			}
		case " ":
			if !safety.IsBlocked(m.commands[m.cursor].Command) {
				m.commands[m.cursor].Selected = !m.commands[m.cursor].Selected
			}
		case "a":
			for i := range m.commands {
				if !safety.IsBlocked(m.commands[i].Command) {
					m.commands[i].Selected = true
				}
			}
		case "n":
			for i := range m.commands {
				m.commands[i].Selected = false
			}
		case "e":
			m.mode = listModeEdit
			m.textInput.Placeholder = "edit the command..."
			m.textInput.CharLimit = 0
			m.textInput.SetValue(m.commands[m.cursor].Command)
			m.textInput.CursorEnd()
			return m, tea.Batch(m.textInput.Focus(), textinput.Blink)
		case "r":
			m.mode = listModeRegenerate
			m.textInput.Placeholder = "add refinement here..."
			m.textInput.CharLimit = 200
			m.textInput.SetValue("")
			return m, tea.Batch(m.textInput.Focus(), textinput.Blink)
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m commandListModel) updateRegenerateMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			m.regenerate = true
			return m, tea.Quit
		case "esc":
			return m.backToList(), nil
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m commandListModel) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			if edited := strings.TrimSpace(m.textInput.Value()); edited != "" {
				item := &m.commands[m.cursor]
				item.Command = edited
				item.Explanation = ""
				item.Risk = safety.AssessRisk(edited)
				if safety.IsBlocked(edited) {
					item.Selected = false
				}
			}
			return m.backToList(), nil
		case "esc":
			return m.backToList(), nil
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m commandListModel) backToList() commandListModel {
	m.mode = listModeSelect
	m.textInput.SetValue("")
	m.textInput.Blur()
	return m
}

func (m commandListModel) View() string {
	if m.confirmed || m.cancelled || m.regenerate {
		return ""
	}

	switch m.mode {
	case listModeRegenerate:
		return m.viewRegenerateMode()
	case listModeEdit:
		return m.viewEditMode()
	}

	var b strings.Builder

	title := fmt.Sprintf("Generated Commands (%d)", len(m.commands))
	cmdTitleStyle := TitleBoldStyle.Foreground(ColorInfo)

	b.WriteString("\n" + cmdTitleStyle.Render(title) + "\n")

	for i, item := range m.commands {
		isLast := i == len(m.commands)-1
		isActive := m.cursor == i

		branch := TreeBranch
		if isLast {
			branch = TreeLastBranch
		}

		connectorStyle := TreeStyle
		if isActive {
			connectorStyle = treeConnectorActiveStyle
		}

		checkbox := checkboxUncheckedStyle.Render("[○]")
		if item.Selected {
			checkbox = checkboxCheckedStyle.Render("[●]")
		}
		if safety.IsBlocked(item.Command) {
			checkbox = checkboxBlockedStyle.Render("[⊘]")
		}

		prefix := fmt.Sprintf("%s %s ", connectorStyle.Render(branch), checkbox)
		b.WriteString(IndentUnder(prefix, HighlightCommand(item.Command)) + "\n")

		verticalLine := TreeVertical
		if isLast {
			verticalLine = " "
		}

		riskEmoji := safety.GetRiskEmoji(item.Risk)
		riskStyle := getRiskStyle(string(item.Risk))

		riskText := riskStyle.Render(string(item.Risk))
		if safety.IsBlocked(item.Command) {
			riskText = riskStyle.Render(string(item.Risk) + " (blocked)")
		}

		riskLine := fmt.Sprintf("%s     %s %s",
			connectorStyle.Render(verticalLine),
			riskEmoji,
			riskText,
		)
		if item.Explanation != "" {
			riskLine += ExplanationStyle.Render(" — " + item.Explanation)
		}
		b.WriteString(Truncate(riskLine, GetTerminalWidth()) + "\n")
	}

	b.WriteString("\n")

	selectedCount := 0
	for _, item := range m.commands {
		if item.Selected {
			selectedCount++
		}
	}

	b.WriteString(hintStyle.Render(fmt.Sprintf("  %d of %d selected\n\n", selectedCount, len(m.commands))))

	b.WriteString(helpStyle.Render("  ↑/↓: navigate • space: toggle • a: all • n: none • e: edit • r: regenerate • enter: execute • q: quit"))

	return b.String()
}

func (m commandListModel) viewRegenerateMode() string {
	var b strings.Builder

	regenTitleStyle := TitleBoldStyle.Foreground(ColorPrimary)

	b.WriteString("\n" + regenTitleStyle.Render("Refine your request") + "\n")

	queryPreview := Truncate(m.originalQuery, 60)

	b.WriteString(hintStyle.Render(fmt.Sprintf("  Original: \"%s\"\n\n", queryPreview)))

	b.WriteString(infoStyle.Render("  Add to your request (or press Enter to retry):\n"))
	b.WriteString("  " + m.textInput.View() + "\n\n")

	b.WriteString(helpStyle.Render("  enter: regenerate • esc: cancel"))

	return b.String()
}

func (m commandListModel) viewEditMode() string {
	var b strings.Builder

	b.WriteString("\n" + TitleBoldStyle.Foreground(ColorPrimary).Render("Edit command") + "\n")
	b.WriteString(hintStyle.Render(fmt.Sprintf("  %d of %d\n\n", m.cursor+1, len(m.commands))))
	b.WriteString("  " + m.textInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("  enter: save • esc: cancel"))

	return b.String()
}

type CommandListResult struct {
	SelectedCommands []string
	Cancelled        bool
	Regenerate       bool
	Refinement       string
}

func SelectCommands(suggestions []Suggestion, originalQuery string) CommandListResult {
	if len(suggestions) == 0 || !IsInteractive() {
		return CommandListResult{Cancelled: true}
	}

	if len(suggestions) == 1 {
		result := ConfirmExecutionInteractive(suggestions[0])
		switch result.Choice {
		case ConfirmExecute:
			return CommandListResult{SelectedCommands: []string{result.Command}}
		case ConfirmRegenerate:
			return CommandListResult{Regenerate: true, Refinement: result.Refinement}
		default:
			return CommandListResult{Cancelled: true}
		}
	}

	m := newCommandListModel(suggestions, originalQuery)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return CommandListResult{Cancelled: true}
	}

	result := finalModel.(commandListModel)

	if result.cancelled {
		return CommandListResult{Cancelled: true}
	}

	if result.regenerate {
		return CommandListResult{Regenerate: true, Refinement: strings.TrimSpace(result.textInput.Value())}
	}

	var selected []string
	for _, item := range result.commands {
		if item.Selected {
			selected = append(selected, item.Command)
		}
	}

	return CommandListResult{SelectedCommands: selected}
}
