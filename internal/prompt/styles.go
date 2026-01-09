package prompt

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	// Primary - Brand violet
	colorPrimary    = lipgloss.Color("#7C3AED")
	colorPrimaryDim = lipgloss.Color("#6D28D9")

	// Semantic - Soft, professional
	colorSuccess = lipgloss.Color("#22C55E")
	colorWarning = lipgloss.Color("#F59E0B")
	colorDanger  = lipgloss.Color("#EF4444")
	colorInfo    = lipgloss.Color("#06B6D4")

	// Neutral - Terminal-friendly
	colorText      = lipgloss.Color("#F9FAFB")
	colorTextDim   = lipgloss.Color("#9CA3AF")
	colorTextMuted = lipgloss.Color("#6B7280")
	colorBorder    = lipgloss.Color("#4B5563")
	colorBorderDim = lipgloss.Color("#374151")

	// Adaptive colors for light/dark terminal support
	subtleColor    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	highlightColor = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#7C3AED"}

	// Base box style - all boxes inherit from this pattern
	boxBase = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	boxStyle = boxBase.BorderForeground(subtleColor)

	commandBoxStyle = boxBase.BorderForeground(colorInfo)

	outputBoxStyle = boxBase.BorderForeground(colorBorder)

	errorBoxStyle = boxBase.BorderForeground(colorDanger)

	welcomeBoxStyle = boxBase.
			BorderForeground(colorPrimary).
			Padding(1, 2)

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	titleBoldStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	commandTextStyle = lipgloss.NewStyle().
				Foreground(colorInfo).
				Bold(true)

	outputTextStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	errorTextStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	// Semantic styles
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorInfo)

	// Label styles
	labelStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Width(12)

	labelActiveStyle = lipgloss.NewStyle().
				Foreground(colorInfo).
				Bold(true).
				Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorText)

	// Risk level styles
	riskSafeStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	riskCautionStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	riskDangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	// Interactive element styles
	selectedStyle = lipgloss.NewStyle().
			Foreground(colorInfo).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Checkbox styles
	checkboxCheckedStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	checkboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(colorTextMuted)

	checkboxBlockedStyle = lipgloss.NewStyle().
				Foreground(colorDanger)

	// Help/Hint styles
	helpStyle = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Italic(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(colorTextMuted)

	// Progress/Spinner styles
	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorPrimary)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(colorBorderDim)

	commandPreviewStyle = lipgloss.NewStyle().
				Foreground(colorTextMuted)

	// Input styles
	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	inputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Step indicator styles
	stepIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorTextDim)

	stepActiveStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Table styles (for config display)
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Padding(0, 1)

	tableLabelStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Padding(0, 1)

	tableValueStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	tableBorderStyle = lipgloss.NewStyle().
				Foreground(colorBorder)
)

// Exported styles for external packages (cmd)
var (
	ColorPrimary     = colorPrimary
	ColorInfo        = colorInfo
	ColorBorder      = colorBorder
	TitleBoldStyle   = titleBoldStyle
	TableHeaderStyle = tableHeaderStyle
	TableLabelStyle  = tableLabelStyle
	TableValueStyle  = tableValueStyle
	TableBorderStyle = tableBorderStyle
	HintStyle        = hintStyle
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

	titleRendered := titleBoldStyle.
		Foreground(borderColor).
		Render(title)

	box := boxBase.
		BorderForeground(borderColor).
		Width(width).
		Render(content)

	return titleRendered + "\n" + box
}

func RenderCommandBox(title, command string) string {
	return RenderTitledBox(title, commandTextStyle.Render(command), colorInfo)
}

func RenderOutputBox(content string) string {
	return RenderTitledBox("Output", outputTextStyle.Render(content), colorBorder)
}

func RenderErrorBox(content string) string {
	return RenderTitledBox("Error", errorTextStyle.Render(content), colorDanger)
}
