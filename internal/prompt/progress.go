package prompt

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type executionModel struct {
	spinner  spinner.Model
	message  string
	command  string
	done     bool
}

func newExecutionModel(command, message string) executionModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorCyanLg)
	return executionModel{
		spinner: s,
		message: message,
		command: command,
	}
}

func (m executionModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m executionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case executionDoneMsg:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m executionModel) View() string {
	if m.done {
		return ""
	}

	cmdPreview := m.command
	if len(cmdPreview) > 50 {
		cmdPreview = cmdPreview[:47] + "..."
	}

	return fmt.Sprintf("\n%s %s\n%s\n",
		m.spinner.View(),
		infoStyle.Render(m.message),
		lipgloss.NewStyle().Foreground(colorGray).Render("  "+cmdPreview),
	)
}

type executionDoneMsg struct{}

type ExecutionProgress struct {
	program *tea.Program
}

func NewExecutionProgress(command string) *ExecutionProgress {
	m := newExecutionModel(command, "Executing...")
	p := tea.NewProgram(m)
	return &ExecutionProgress{program: p}
}

func (e *ExecutionProgress) Start() {
	go e.program.Run()
}

func (e *ExecutionProgress) Stop() {
	e.program.Send(executionDoneMsg{})
	e.program.Wait()
	fmt.Print("\r\033[K")
}

type batchProgressModel struct {
	spinner  spinner.Model
	current  int
	total    int
	command  string
	done     bool
}

func newBatchProgressModel(current, total int, command string) batchProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorCyanLg)
	return batchProgressModel{
		spinner: s,
		current: current,
		total:   total,
		command: command,
	}
}

func (m batchProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m batchProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case executionDoneMsg:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m batchProgressModel) View() string {
	if m.done {
		return ""
	}

	progressText := fmt.Sprintf("Executing %d of %d", m.current, m.total)

	cmdPreview := m.command
	if len(cmdPreview) > 50 {
		cmdPreview = cmdPreview[:47] + "..."
	}

	return fmt.Sprintf("\n%s %s\n%s\n",
		m.spinner.View(),
		infoStyle.Render(progressText),
		lipgloss.NewStyle().Foreground(colorGray).Render("  "+cmdPreview),
	)
}

type BatchExecutionProgress struct {
	program *tea.Program
}

func NewBatchExecutionProgress(current, total int, command string) *BatchExecutionProgress {
	m := newBatchProgressModel(current, total, command)
	p := tea.NewProgram(m)
	return &BatchExecutionProgress{program: p}
}

func (b *BatchExecutionProgress) Start() {
	go b.program.Run()
}

func (b *BatchExecutionProgress) Stop() {
	b.program.Send(executionDoneMsg{})
	b.program.Wait()
	fmt.Print("\r\033[K")
}
