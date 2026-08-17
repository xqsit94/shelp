package prompt

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

const helpIndent = 2

func renderHelp(h help.Model, k help.KeyMap, width int) string {
	return indentBlock(TruncateLines(h.View(k), width-helpIndent), helpIndent)
}

func newHelpModel(width int) help.Model {
	h := help.New()
	h.Width = width
	h.Styles = help.Styles{
		Ellipsis:       helpStyle,
		ShortKey:       helpKeyStyle,
		ShortDesc:      helpStyle,
		ShortSeparator: helpStyle,
		FullKey:        helpKeyStyle,
		FullDesc:       helpStyle,
		FullSeparator:  helpStyle,
	}
	return h
}

type listKeyMap struct {
	Execute    key.Binding
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Toggle     key.Binding
	All        key.Binding
	None       key.Binding
	Edit       key.Binding
	Regenerate key.Binding
	Help       key.Binding
}

func defaultListKeyMap() listKeyMap {
	return listKeyMap{
		Execute:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Toggle:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
		All:        key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all")),
		None:       key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "none")),
		Edit:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Regenerate: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
	}
}

func (k listKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Execute, k.Quit, k.Up, k.Down, k.Toggle, k.Edit, k.Regenerate, k.Help}
}

func (k listKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Toggle, k.All, k.None},
		{k.Edit, k.Regenerate},
		{k.Execute, k.Quit},
	}
}

type confirmKeyMap struct {
	Execute    key.Binding
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Select     key.Binding
	Edit       key.Binding
	Regenerate key.Binding
}

func defaultConfirmKeyMap() confirmKeyMap {
	return confirmKeyMap{
		Execute:    key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "run")),
		Quit:       key.NewBinding(key.WithKeys("q", "n", "N", "ctrl+c", "esc"), key.WithHelp("q", "cancel")),
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Select:     key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "select")),
		Edit:       key.NewBinding(key.WithKeys("e", "E"), key.WithHelp("e", "edit")),
		Regenerate: key.NewBinding(key.WithKeys("r", "R"), key.WithHelp("r", "retry")),
	}
}

func (k confirmKeyMap) shortHelpFor(blocked bool) []key.Binding {
	if blocked {
		return []key.Binding{k.Select, k.Quit, k.Up, k.Down, k.Edit, k.Regenerate}
	}
	return []key.Binding{k.Execute, k.Quit, k.Select, k.Up, k.Down, k.Edit, k.Regenerate}
}

func (k confirmKeyMap) ShortHelp() []key.Binding { return k.shortHelpFor(false) }

func (k confirmKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.Edit, k.Regenerate},
		{k.Execute, k.Quit},
	}
}

type yesNoKeyMap struct {
	Yes    key.Binding
	No     key.Binding
	Left   key.Binding
	Right  key.Binding
	Select key.Binding
}

func defaultYesNoKeyMap() yesNoKeyMap {
	return yesNoKeyMap{
		Yes:    key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "yes")),
		No:     key.NewBinding(key.WithKeys("n", "N", "q", "ctrl+c", "esc"), key.WithHelp("n", "no")),
		Left:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Select: key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "select")),
	}
}

func (k yesNoKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Yes, k.No, k.Left, k.Right, k.Select}
}

func (k yesNoKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Left, k.Right, k.Select}, {k.Yes, k.No}}
}

type inputKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
}

func newInputKeyMap(submit string) inputKeyMap {
	return inputKeyMap{
		Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", submit)),
		Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (k inputKeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Submit, k.Cancel} }

func (k inputKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

type setupKeyMap struct {
	Submit key.Binding
	Next   key.Binding
	Prev   key.Binding
	Cancel key.Binding
}

func defaultSetupKeyMap() setupKeyMap {
	return setupKeyMap{
		Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		Next:   key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("tab", "next")),
		Prev:   key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("shift+tab", "prev")),
		Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
	}
}

func (k setupKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Cancel, k.Next, k.Prev}
}

func (k setupKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Next, k.Prev}, {k.Submit, k.Cancel}}
}
