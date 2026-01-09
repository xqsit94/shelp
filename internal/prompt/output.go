package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultViewportHeight = 10
	maxViewportHeight     = 20
)

type outputModel struct {
	viewport viewport.Model
	content  string
	title    string
	isError  bool
	ready    bool
	done     bool
}

func newOutputModel(content string, isError bool) outputModel {
	lines := strings.Count(content, "\n") + 1
	height := lines
	if height > maxViewportHeight {
		height = maxViewportHeight
	}
	if height < 3 {
		height = 3
	}

	return outputModel{
		content: content,
		isError: isError,
		title:   "Output",
	}
}

func (m outputModel) Init() tea.Cmd {
	return nil
}

func (m outputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc", "enter":
			m.done = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			lines := strings.Count(m.content, "\n") + 1
			height := lines
			if height > maxViewportHeight {
				height = maxViewportHeight
			}
			if height < 3 {
				height = 3
			}

			m.viewport = viewport.New(msg.Width-4, height)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m outputModel) View() string {
	if m.done {
		return ""
	}

	if !m.ready {
		return "Loading..."
	}

	var titleColor lipgloss.Color
	var contentStyle lipgloss.Style

	if m.isError {
		titleColor = colorDanger
		contentStyle = errorTextStyle
		m.title = "Error"
	} else {
		titleColor = colorBorder
		contentStyle = outputTextStyle
		m.title = "Output"
	}

	titleRendered := titleBoldStyle.
		Foreground(titleColor).
		Render(m.title)

	content := contentStyle.Render(m.viewport.View())

	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		percent := m.viewport.ScrollPercent() * 100
		scrollInfo = hintStyle.Render(fmt.Sprintf(" (%.0f%%)", percent))
	}

	box := boxBase.
		BorderForeground(titleColor).
		Render(content)

	help := helpStyle.Render("↑/↓: scroll • q/enter: close")

	return "\n" + titleRendered + scrollInfo + "\n" + box + "\n" + help
}

func DisplayOutputInteractive(output string, isError bool) {
	if output == "" {
		return
	}

	lines := strings.Count(output, "\n") + 1
	if lines <= defaultViewportHeight {
		if isError {
			DisplayOutput(output, true)
		} else {
			DisplayOutput(output, false)
		}
		return
	}

	m := newOutputModel(strings.TrimSpace(output), isError)
	p := tea.NewProgram(m, tea.WithAltScreen())
	p.Run()
}
