package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xqsit94/shelp/internal/safety"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
)

func DisplayCommand(cmd string, index, total int) {
	risk := safety.AssessRisk(cmd)
	riskColor := safety.GetRiskColor(risk)
	riskEmoji := safety.GetRiskEmoji(risk)

	if total > 1 {
		fmt.Printf("\n%sCommand %d of %d:%s\n", colorDim, index, total, colorReset)
	} else {
		fmt.Printf("\n%sGenerated command:%s\n", colorDim, colorReset)
	}

	boxWidth := len(cmd) + 4
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	fmt.Printf("┌%s┐\n", strings.Repeat("─", boxWidth))
	fmt.Printf("│ %s%s%s%s │\n", colorCyan, colorBold, cmd, colorReset)
	fmt.Printf("└%s┘\n", strings.Repeat("─", boxWidth))

	fmt.Printf("Risk: %s%s %s%s\n", riskColor, riskEmoji, risk, colorReset)
}

func ConfirmExecution(cmd string) bool {
	if safety.IsBlocked(cmd) {
		fmt.Printf("%s🚫 This command has been blocked for safety reasons.%s\n", safety.GetRiskColor(safety.RiskDanger), colorReset)
		return false
	}

	fmt.Print("\nExecute this command? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func PromptInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(input)
}

func ConfirmYesNo(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func DisplayOutput(output string, isError bool) {
	if output == "" {
		return
	}

	if isError {
		fmt.Printf("%s%s%s\n", safety.GetRiskColor(safety.RiskDanger), output, colorReset)
	} else {
		fmt.Printf("%s%s%s\n", colorDim, output, colorReset)
	}
}

func DisplaySuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

func DisplayError(message string) {
	fmt.Printf("❌ %s\n", message)
}

func DisplayWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

func DisplayInfo(message string) {
	fmt.Printf("💡 %s\n", message)
}
