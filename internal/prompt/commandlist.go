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

const maxQueryPreview = 60

const (
	rowsPerCommand = 2
	listChromeRows = 8
	minVisibleRows = 1
)

func visibleRange(count, cursor, availableRows int) (start, end int) {
	if availableRows <= 0 || count*rowsPerCommand <= availableRows {
		return 0, count
	}

	visible := max(availableRows/rowsPerCommand, minVisibleRows)

	start = cursor - visible/2
	start = min(max(start, 0), count-visible)

	return start, start + visible
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
	keys          listKeyMap
	editKeys      inputKeyMap
	refineKeys    inputKeyMap
	help          help.Model
	width         int
	height        int
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

	width := GetTerminalWidth()

	ti := textinput.New()
	ti.Placeholder = "add refinement here..."
	ti.CharLimit = 200
	ti.Width = width - 6

	h := newHelpModel(width)

	return commandListModel{
		commands:      items,
		cursor:        0,
		originalQuery: originalQuery,
		textInput:     ti,
		keys:          defaultListKeyMap(),
		editKeys:      newInputKeyMap("save"),
		refineKeys:    newInputKeyMap("regenerate"),
		help:          h,
		width:         width,
	}
}

func (m commandListModel) Init() tea.Cmd {
	return nil
}

func (m commandListModel) setSize(width, height int) commandListModel {
	m.width = width
	m.height = height
	m.help.Width = width
	m.textInput.Width = width - 6
	return m
}

func (m commandListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		return m.setSize(size.Width, size.Height), nil
	}

	switch m.mode {
	case listModeRegenerate:
		return m.updateRegenerateMode(msg)
	case listModeEdit:
		return m.updateEditMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.commands)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Toggle):
			if !safety.IsBlocked(m.commands[m.cursor].Command) {
				m.commands[m.cursor].Selected = !m.commands[m.cursor].Selected
			}
		case key.Matches(msg, m.keys.All):
			for i := range m.commands {
				if !safety.IsBlocked(m.commands[i].Command) {
					m.commands[i].Selected = true
				}
			}
		case key.Matches(msg, m.keys.None):
			for i := range m.commands {
				m.commands[i].Selected = false
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Edit):
			m.mode = listModeEdit
			m.textInput.Placeholder = "edit the command..."
			m.textInput.CharLimit = 0
			m.textInput.SetValue(m.commands[m.cursor].Command)
			m.textInput.CursorEnd()
			return m, tea.Batch(m.textInput.Focus(), textinput.Blink)
		case key.Matches(msg, m.keys.Regenerate):
			m.mode = listModeRegenerate
			m.textInput.Placeholder = "add refinement here..."
			m.textInput.CharLimit = 200
			m.textInput.SetValue("")
			return m, tea.Batch(m.textInput.Focus(), textinput.Blink)
		case key.Matches(msg, m.keys.Execute):
			m.confirmed = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Quit):
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

	b.WriteByte('\n')
	writeLine(&b, cmdTitleStyle.Render(title))

	width := m.width

	start, end := visibleRange(len(m.commands), m.cursor, m.height-listChromeRows)
	if start > 0 {
		writeLine(&b, hintStyle.Render(fmt.Sprintf("  ↑ %d more", start)))
	}

	for i, item := range m.commands[start:end] {
		i += start
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

		gutter := "  "
		if isActive {
			gutter = cursorStyle.Render("❯ ")
		}

		checkbox := checkboxUncheckedStyle.Render("[ ]")
		if item.Selected {
			checkbox = checkboxCheckedStyle.Render("[✓]")
		}
		if safety.IsBlocked(item.Command) {
			checkbox = checkboxBlockedStyle.Render("[⊘]")
		}

		prefix := fmt.Sprintf("%s%s %s ", gutter, connectorStyle.Render(branch), checkbox)
		commandLine := IndentUnder(prefix, HighlightCommand(item.Command))
		writeLine(&b, TruncateLines(commandLine, width))

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

		riskLine := fmt.Sprintf("  %s     %s %s",
			connectorStyle.Render(verticalLine),
			riskEmoji,
			riskText,
		)
		if item.Explanation != "" {
			riskLine += ExplanationStyle.Render(" — " + item.Explanation)
		}
		writeLine(&b, Truncate(riskLine, width))
	}

	if end < len(m.commands) {
		writeLine(&b, hintStyle.Render(fmt.Sprintf("  ↓ %d more", len(m.commands)-end)))
	}

	b.WriteString("\n")

	selectedCount := 0
	for _, item := range m.commands {
		if item.Selected {
			selectedCount++
		}
	}

	b.WriteString(hintStyle.Render(fmt.Sprintf("  %d of %d selected", selectedCount, len(m.commands))))
	b.WriteString("\n\n")

	b.WriteString(m.helpView(m.keys))

	return b.String()
}

func (m commandListModel) helpView(k help.KeyMap) string {
	return renderHelp(m.help, k, m.width)
}

func (m commandListModel) viewRegenerateMode() string {
	var b strings.Builder

	regenTitleStyle := TitleBoldStyle.Foreground(ColorPrimary)

	b.WriteByte('\n')
	writeLine(&b, regenTitleStyle.Render("Refine your request"))

	queryPreview := Truncate(m.originalQuery, maxQueryPreview)

	b.WriteString(hintStyle.Render(fmt.Sprintf("  Original: %q", queryPreview)))
	b.WriteString("\n\n")

	b.WriteString(infoStyle.Render("  Add to your request (or press Enter to retry):"))
	b.WriteString("\n  ")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	b.WriteString(m.helpView(m.refineKeys))

	return b.String()
}

func (m commandListModel) viewEditMode() string {
	var b strings.Builder

	b.WriteByte('\n')
	writeLine(&b, TitleBoldStyle.Foreground(ColorPrimary).Render("Edit command"))
	b.WriteString(hintStyle.Render(fmt.Sprintf("  %d of %d", m.cursor+1, len(m.commands))))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.helpView(m.editKeys))

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
