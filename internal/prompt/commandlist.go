package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xqsit94/shelp/internal/safety"
)

type CommandItem struct {
	Command  string
	Risk     safety.RiskLevel
	Selected bool
}

type commandListModel struct {
	commands      []CommandItem
	cursor        int
	confirmed     bool
	cancelled     bool
	regenerate    bool
	regenerating  bool
	originalQuery string
	textInput     textinput.Model
}

func newCommandListModel(commands []string, originalQuery string) commandListModel {
	items := make([]CommandItem, len(commands))
	for i, cmd := range commands {
		items[i] = CommandItem{
			Command:  cmd,
			Risk:     safety.AssessRisk(cmd),
			Selected: !safety.IsBlocked(cmd),
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
	if m.regenerating {
		return m.updateRegenerateMode(msg)
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
		case "r":
			m.regenerating = true
			m.textInput.Focus()
			return m, textinput.Blink
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.regenerate = true
			return m, tea.Quit
		case "esc":
			m.regenerating = false
			m.textInput.SetValue("")
			m.textInput.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m commandListModel) View() string {
	if m.confirmed || m.cancelled || m.regenerate {
		return ""
	}

	if m.regenerating {
		return m.viewRegenerateMode()
	}

	var b strings.Builder

	title := fmt.Sprintf("Generated Commands (%d)", len(m.commands))
	cmdTitleStyle := titleBoldStyle.Foreground(colorInfo)

	b.WriteString("\n" + cmdTitleStyle.Render(title) + "\n")

	var content strings.Builder
	for i, item := range m.commands {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("› ")
		}

		checkbox := checkboxUncheckedStyle.Render("○")
		if item.Selected {
			checkbox = checkboxCheckedStyle.Render("●")
		}
		if safety.IsBlocked(item.Command) {
			checkbox = dangerStyle.Render("⊘")
		}

		riskEmoji := safety.GetRiskEmoji(item.Risk)
		riskStyle := getRiskStyle(string(item.Risk))

		cmdStyle := commandTextStyle
		if m.cursor == i {
			cmdStyle = cmdStyle.Bold(true)
		}

		line := fmt.Sprintf("%s%s %s  %s %s",
			cursor,
			checkbox,
			cmdStyle.Render(item.Command),
			riskEmoji,
			riskStyle.Render(string(item.Risk)),
		)
		content.WriteString(line + "\n")
	}

	box := commandBoxStyle.
		Width(GetTerminalWidth() - 2).
		Render(content.String())

	b.WriteString(box + "\n\n")

	selectedCount := 0
	for _, item := range m.commands {
		if item.Selected {
			selectedCount++
		}
	}

	b.WriteString(hintStyle.Render(fmt.Sprintf("  %d of %d selected\n\n", selectedCount, len(m.commands))))

	b.WriteString(helpStyle.Render("  ↑/↓: navigate • space: toggle • a: all • n: none • r: regenerate • enter: execute • q: quit"))

	return b.String()
}

func (m commandListModel) viewRegenerateMode() string {
	var b strings.Builder

	regenTitleStyle := titleBoldStyle.Foreground(colorPrimary)

	b.WriteString("\n" + regenTitleStyle.Render("Refine your request") + "\n")

	queryPreview := m.originalQuery
	if len(queryPreview) > 60 {
		queryPreview = queryPreview[:57] + "..."
	}

	b.WriteString(hintStyle.Render(fmt.Sprintf("  Original: \"%s\"\n\n", queryPreview)))

	b.WriteString(infoStyle.Render("  Add to your request (or press Enter to retry):\n"))
	b.WriteString("  " + m.textInput.View() + "\n\n")

	b.WriteString(helpStyle.Render("  enter: regenerate • esc: cancel"))

	return b.String()
}

type CommandListResult struct {
	SelectedCommands []string
	Cancelled        bool
	Regenerate       bool
	NewQuery         string
}

func SelectCommands(commands []string, originalQuery string) CommandListResult {
	if len(commands) == 0 {
		return CommandListResult{Cancelled: true}
	}

	if len(commands) == 1 {
		choice := ConfirmExecutionInteractive(commands[0])
		switch choice {
		case ConfirmExecute:
			return CommandListResult{SelectedCommands: commands}
		case ConfirmRegenerate:
			return CommandListResult{Regenerate: true, NewQuery: originalQuery}
		default:
			return CommandListResult{Cancelled: true}
		}
	}

	m := newCommandListModel(commands, originalQuery)
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
		refinement := strings.TrimSpace(result.textInput.Value())
		newQuery := originalQuery
		if refinement != "" {
			newQuery = originalQuery + ", " + refinement
		}
		return CommandListResult{Regenerate: true, NewQuery: newQuery}
	}

	var selected []string
	for _, item := range result.commands {
		if item.Selected {
			selected = append(selected, item.Command)
		}
	}

	return CommandListResult{SelectedCommands: selected}
}
