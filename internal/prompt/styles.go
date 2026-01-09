package prompt

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	colorCyanLg   = lipgloss.Color("#00FFFF")
	colorGreen    = lipgloss.Color("#00FF00")
	colorYellow   = lipgloss.Color("#FFFF00")
	colorRed      = lipgloss.Color("#FF0000")
	colorGray     = lipgloss.Color("#666666")
	colorWhite    = lipgloss.Color("#FFFFFF")
	colorDimWhite = lipgloss.Color("#AAAAAA")
	colorPurple   = lipgloss.Color("#874BFD")

	subtle    = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1)

	commandBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyanLg).
			Padding(0, 1)

	outputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGray).
			Padding(0, 1)

	errorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorRed).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorDimWhite).
			Bold(false)

	commandTextStyle = lipgloss.NewStyle().
				Foreground(colorCyanLg).
				Bold(true)

	outputTextStyle = lipgloss.NewStyle().
			Foreground(colorDimWhite)

	errorTextStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorCyanLg)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorDimWhite).
			Width(10)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	riskSafeStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	riskCautionStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	riskDangerStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorCyanLg).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(colorDimWhite)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorCyanLg).
			Bold(true)

	checkboxCheckedStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	checkboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(colorGray)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(colorCyanLg)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(colorGray)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorCyanLg).
			Padding(0, 1)

	inputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(colorCyanLg).
				Padding(0, 1)

	stepIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorDimWhite)

	stepActiveStyle = lipgloss.NewStyle().
			Foreground(colorCyanLg).
			Bold(true)

	welcomeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(1, 2)
)

func getRiskStyle(risk string) lipgloss.Style {
	switch risk {
	case "safe":
		return riskSafeStyle
	case "caution":
		return riskCautionStyle
	case "danger":
		return riskDangerStyle
	default:
		return lipgloss.NewStyle()
	}
}

func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func RenderTitledBox(title, content string, borderColor lipgloss.Color) string {
	width := GetTerminalWidth() - 2

	titleRendered := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true).
		Render(title)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(content)

	return titleRendered + "\n" + box
}

func RenderCommandBox(title, command string) string {
	return RenderTitledBox(title, commandTextStyle.Render(command), colorCyanLg)
}

func RenderOutputBox(content string) string {
	return RenderTitledBox("Output", outputTextStyle.Render(content), colorGray)
}

func RenderErrorBox(content string) string {
	return RenderTitledBox("Error", errorTextStyle.Render(content), colorRed)
}
