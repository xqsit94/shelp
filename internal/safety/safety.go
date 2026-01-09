package safety

import (
	"regexp"
	"strings"
)

type RiskLevel string

const (
	RiskSafe    RiskLevel = "safe"
	RiskCaution RiskLevel = "caution"
	RiskDanger  RiskLevel = "danger"
)

var blockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+(-[rRf]+\s+)*[~/]\s*$`),
	regexp.MustCompile(`rm\s+(-[rRf]+\s+)*/\s*$`),
	regexp.MustCompile(`rm\s+.*--no-preserve-root`),
	regexp.MustCompile(`:\s*\(\s*\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`),
	regexp.MustCompile(`dd\s+.*of\s*=\s*/dev/(sda|nvme|hd[a-z]|disk)`),
	regexp.MustCompile(`chmod\s+(-R\s+)?777\s+/\s*$`),
	regexp.MustCompile(`chmod\s+(-R\s+)?777\s+~/?\s*$`),
	regexp.MustCompile(`mkfs\.[a-z0-9]+\s+/dev/(sda|nvme|hd[a-z]|disk)`),
	regexp.MustCompile(`>\s*/dev/(sda|nvme|hd[a-z]|disk)`),
	regexp.MustCompile(`mv\s+/\s+`),
	regexp.MustCompile(`mv\s+~/?\s+/dev/null`),
	regexp.MustCompile(`wget\s+.*\|\s*(ba)?sh`),
	regexp.MustCompile(`curl\s+.*\|\s*(ba)?sh`),
	regexp.MustCompile(`echo\s+.*\|\s*base64\s+-d\s*\|\s*(ba)?sh`),
	regexp.MustCompile(`perl\s+-e\s*['"].*exec`),
	regexp.MustCompile(`python[23]?\s+-c\s*['"].*exec`),
	regexp.MustCompile(`sudo\s+rm\s+(-[rRf]+\s+)*/`),
}

var cautionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+(-[rRfv]+\s+)`),
	regexp.MustCompile(`sudo\s+`),
	regexp.MustCompile(`chmod\s+`),
	regexp.MustCompile(`chown\s+`),
	regexp.MustCompile(`dd\s+`),
	regexp.MustCompile(`mkfs\.`),
	regexp.MustCompile(`fdisk\s+`),
	regexp.MustCompile(`parted\s+`),
	regexp.MustCompile(`kill\s+`),
	regexp.MustCompile(`killall\s+`),
	regexp.MustCompile(`pkill\s+`),
	regexp.MustCompile(`systemctl\s+(stop|restart|disable)`),
	regexp.MustCompile(`service\s+.*\s+(stop|restart)`),
	regexp.MustCompile(`reboot`),
	regexp.MustCompile(`shutdown`),
	regexp.MustCompile(`init\s+[0-6]`),
	regexp.MustCompile(`>\s*/etc/`),
	regexp.MustCompile(`pip\s+install`),
	regexp.MustCompile(`npm\s+install\s+-g`),
	regexp.MustCompile(`brew\s+install`),
	regexp.MustCompile(`apt(-get)?\s+install`),
	regexp.MustCompile(`yum\s+install`),
	regexp.MustCompile(`dnf\s+install`),
}

func IsBlocked(command string) bool {
	normalizedCmd := strings.ToLower(strings.TrimSpace(command))
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(normalizedCmd) {
			return true
		}
	}
	return false
}

func AssessRisk(command string) RiskLevel {
	if IsBlocked(command) {
		return RiskDanger
	}

	normalizedCmd := strings.ToLower(strings.TrimSpace(command))
	for _, pattern := range cautionPatterns {
		if pattern.MatchString(normalizedCmd) {
			return RiskCaution
		}
	}

	return RiskSafe
}

func GetRiskEmoji(risk RiskLevel) string {
	switch risk {
	case RiskSafe:
		return "✅"
	case RiskCaution:
		return "⚠️"
	case RiskDanger:
		return "🚫"
	default:
		return "❓"
	}
}
