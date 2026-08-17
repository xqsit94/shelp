package prompt

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	TreeBranch     = "├─"
	TreeLastBranch = "└─"
	TreeVertical   = "│"
)

var (
	// Primary - Brand violet
	ColorPrimary = lipgloss.Color("#7C3AED")

	// Semantic - Soft, professional
	colorSuccess = lipgloss.Color("#22C55E")
	colorWarning = lipgloss.Color("#F59E0B")
	colorDanger  = lipgloss.Color("#EF4444")
	ColorInfo    = lipgloss.Color("#06B6D4")

	// Neutral - Terminal-friendly
	colorText      = lipgloss.Color("#F9FAFB")
	colorTextDim   = lipgloss.Color("#9CA3AF")
	colorTextMuted = lipgloss.Color("#6B7280")
	colorBorder    = lipgloss.Color("#4B5563")
	colorBorderDim = lipgloss.Color("#374151")

	// Base box style - all boxes inherit from this pattern
	boxBase = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	welcomeBoxStyle = boxBase.
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// Text styles
	TitleBoldStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	// Semantic styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	DangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	infoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

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
			Foreground(ColorInfo).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	cursorStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
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
			Foreground(colorTextMuted)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	hintStyle = lipgloss.NewStyle().
			Foreground(colorTextMuted)

	// Command explanations are secondary to the command itself.
	ExplanationStyle = lipgloss.NewStyle().
				Foreground(colorTextMuted)

	// Tree connector styles
	TreeStyle = lipgloss.NewStyle().
			Foreground(colorBorder)

	treeConnectorActiveStyle = lipgloss.NewStyle().
					Foreground(ColorPrimary).
					Bold(true)

	// Progress/Spinner styles
	spinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(colorBorderDim)

	// Input styles
	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	inputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Step indicator styles
	stepIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorTextDim)

	// Table styles (for config display)
	TableLabelStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Padding(0, 1)

	TableValueStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	TableBorderStyle = lipgloss.NewStyle().
				Foreground(colorBorder)
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
