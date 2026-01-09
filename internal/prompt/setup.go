package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SetupResult struct {
	AIURL     string
	APIKey    string
	Model     string
	Cancelled bool
}

type setupModel struct {
	focusIndex int
	inputs     []textinput.Model
	labels     []string
	placeholders []string
	step       int
	totalSteps int
	cancelled  bool
	done       bool
}

func newSetupModel() setupModel {
	labels := []string{"AI API URL", "API Key", "Model"}
	placeholders := []string{
		"https://openrouter.ai/api/v1/chat/completions",
		"sk-or-v1-...",
		"anthropic/claude-3.5-sonnet",
	}

	inputs := make([]textinput.Model, 3)

	for i := range inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.CharLimit = 256
		t.Width = 50

		if i == 1 {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '*'
		}

		inputs[i] = t
	}

	inputs[0].Focus()

	return setupModel{
		inputs:       inputs,
		labels:       labels,
		placeholders: placeholders,
		focusIndex:   0,
		step:         1,
		totalSteps:   3,
	}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "tab", "down":
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			m.step = m.focusIndex + 1
			return m, m.updateFocus()
		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			m.step = m.focusIndex + 1
			return m, m.updateFocus()
		case "enter":
			if m.focusIndex < len(m.inputs)-1 {
				m.focusIndex++
				m.step = m.focusIndex + 1
				return m, m.updateFocus()
			}
			if m.allFieldsFilled() {
				m.done = true
				return m, tea.Quit
			}
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *setupModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *setupModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m setupModel) allFieldsFilled() bool {
	for _, input := range m.inputs {
		if strings.TrimSpace(input.Value()) == "" {
			return false
		}
	}
	return true
}

func (m setupModel) View() string {
	if m.done || m.cancelled {
		return ""
	}

	var b strings.Builder

	welcomeTitle := lipgloss.NewStyle().
		Foreground(colorPurple).
		Bold(true).
		Render("Welcome to shelp!")

	welcomeSubtitle := lipgloss.NewStyle().
		Foreground(colorDimWhite).
		Render("Let's set up your AI provider.")

	stepText := fmt.Sprintf("Step %d of %d", m.step, m.totalSteps)
	stepIndicator := stepIndicatorStyle.Render(stepText)

	progressBar := m.renderProgressBar()

	header := lipgloss.JoinVertical(lipgloss.Left,
		welcomeTitle,
		welcomeSubtitle,
		"",
		stepIndicator,
		progressBar,
	)

	headerBox := welcomeBoxStyle.Render(header)
	b.WriteString("\n" + headerBox + "\n\n")

	for i, input := range m.inputs {
		labelStyle := lipgloss.NewStyle().Foreground(colorDimWhite)
		if i == m.focusIndex {
			labelStyle = labelStyle.Foreground(colorCyanLg).Bold(true)
		}

		b.WriteString("  " + labelStyle.Render(m.labels[i]+":") + "\n")

		inputBox := inputStyle
		if i == m.focusIndex {
			inputBox = inputFocusedStyle
		}
		b.WriteString("  " + inputBox.Render(input.View()) + "\n\n")
	}

	b.WriteString("\n" + helpStyle.Render("  tab/↓: next • shift+tab/↑: prev • enter: submit • esc: cancel"))

	return b.String()
}

func (m setupModel) renderProgressBar() string {
	width := 30
	filled := width * m.step / m.totalSteps

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString(progressBarStyle.Render("━"))
		} else {
			bar.WriteString(progressEmptyStyle.Render("░"))
		}
	}

	return bar.String()
}

func RunSetupWizard() SetupResult {
	m := newSetupModel()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return SetupResult{Cancelled: true}
	}

	result := finalModel.(setupModel)
	if result.cancelled {
		return SetupResult{Cancelled: true}
	}

	return SetupResult{
		AIURL:  strings.TrimSpace(result.inputs[0].Value()),
		APIKey: strings.TrimSpace(result.inputs[1].Value()),
		Model:  strings.TrimSpace(result.inputs[2].Value()),
	}
}
