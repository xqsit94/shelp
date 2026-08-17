package prompt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var ErrCancelled = errors.New("cancelled")

var ShelpSpinner = spinner.Spinner{
	Frames: []string{
		"▰▱▱▱▱",
		"▰▰▱▱▱",
		"▰▰▰▱▱",
		"▰▰▰▰▱",
		"▰▰▰▰▰",
		"▱▰▰▰▰",
		"▱▱▰▰▰",
		"▱▱▱▰▰",
		"▱▱▱▱▰",
		"▱▱▱▱▱",
	},
	FPS: time.Second / 12,
}

type spinnerResultMsg[T any] struct {
	value T
	err   error
}

type spinnerModel[T any] struct {
	spinner  spinner.Model
	message  string
	fn       func(context.Context) (T, error)
	ctx      context.Context
	cancel   context.CancelFunc
	value    T
	err      error
	quitting bool
}

func (m spinnerModel[T]) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.run)
}

func (m spinnerModel[T]) run() tea.Msg {
	value, err := m.fn(m.ctx)
	return spinnerResultMsg[T]{value: value, err: err}
}

func (m spinnerModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m.cancelled()
		}
	case tea.InterruptMsg:
		return m.cancelled()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case spinnerResultMsg[T]:
		m.value = msg.value
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m spinnerModel[T]) cancelled() (tea.Model, tea.Cmd) {
	m.cancel()
	m.err = ErrCancelled
	m.quitting = true
	return m, tea.Quit
}

func (m spinnerModel[T]) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// RunWithSpinner runs fn while a spinner is on screen, cancelling fn's context
// when the user interrupts. Without a terminal there is nothing to animate and
// nothing to read keys from, so fn simply runs inline.
func RunWithSpinner[T any](ctx context.Context, message string, fn func(context.Context) (T, error)) (T, error) {
	if !IsInteractive() {
		return fn(ctx)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s := spinner.New()
	s.Spinner = ShelpSpinner
	s.Style = spinnerStyle

	finalModel, err := tea.NewProgram(spinnerModel[T]{
		spinner: s,
		message: message,
		fn:      fn,
		ctx:     ctx,
		cancel:  cancel,
	}).Run()
	if err != nil {
		var zero T
		if errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, tea.ErrInterrupted) {
			return zero, ErrCancelled
		}
		return zero, err
	}

	result := finalModel.(spinnerModel[T])

	return result.value, result.err
}
