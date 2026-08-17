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

const (
	// Root or home directory in any spelling, optionally quoted.
	rootOrHome = `["']?(//|/\*|/|~/\*|~/\.|~/|~|\$\{home\}|\$home)["']?`
	// Whole-disk device nodes; partition suffixes are matched by the trailing context.
	blockDevice = `/dev/(sd[a-z]|vd[a-z]|xvd[a-z]|hd[a-z]|nvme\d|mmcblk\d|disk\d|rdisk\d)`
	// Any flag token, and the subset that makes rm destructive.
	anyFlag       = `(--[a-z-]+|--|-[a-z]+)`
	destructiveRm = `(--recursive|--force|--no-preserve-root|-[a-z]*[rf][a-z]*)`
	recursiveFlag = `(--recursive|-[a-z]*r[a-z]*)`
	// Operands before the destructive one: the target is rarely the last
	// argument, e.g. `rm -rf $HOME /tmp/cache`.
	anyOperand = `(\S+\s+)*`
	// A destructive target only counts as a whole argument, so /tmp/cache is
	// not read as /.
	operandEnd = `(\s|$)`
	// sudo and doas with their own options, e.g. `... | sudo -u root bash`.
	privPrefix = `((sudo|doas)\s+(-[a-z-]+\s+(\S+\s+)?)*)?`

	// Windows drive roots and profile directories, optionally quoted and with a
	// trailing separator or wildcard.
	windowsDrive = `([a-z]:|\$env:systemdrive|\$env:userprofile|\$home|~)`
	windowsRoot  = `["']?(` + windowsDrive + `(\\\*?|/\*?)?|\\\*?)["']?`
	// Any PowerShell or cmd flag, and the subset that deletes recursively or
	// without asking, including the abbreviations PowerShell accepts.
	windowsFlag   = `(-[a-z]+|/[a-z])`
	windowsDelete = `(remove-item|rmdir|erase|del|rd|ri|rm)`
	forcedDelete  = `(-r|-rec[a-z]*|-f|-for[a-z]*|/s|/q|/f)`
)

var blockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\s+(` + anyFlag + `\s+)*` + destructiveRm + `\s+` + anyOperand + rootOrHome + operandEnd),
	regexp.MustCompile(`\brm\s+.*--no-preserve-root`),
	regexp.MustCompile(`:\s*\(\s*\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`),
	regexp.MustCompile(`\bdd\s+.*\bof\s*=\s*` + blockDevice),
	regexp.MustCompile(`>\s*` + blockDevice),
	regexp.MustCompile(`\bmkfs[a-z0-9.]*\s+.*` + blockDevice),
	regexp.MustCompile(`\b(wipefs|shred)\s+.*` + blockDevice),
	regexp.MustCompile(`\b(chmod|chown)\s+(\S+\s+)*` + recursiveFlag + `\s+` + anyOperand + rootOrHome + operandEnd),
	regexp.MustCompile(`\bmv\s+/\s+`),
	regexp.MustCompile(`\bmv\s+~/?\s+/dev/null`),
	regexp.MustCompile(`\b(curl|wget)\s.*\|\s*` + privPrefix + `(ba|z|k|da)?sh\b`),
	regexp.MustCompile(`\b(ba|z)?sh\s+<\(\s*(curl|wget)`),
	regexp.MustCompile(`\b(ba|z)?sh\s+-c\s+["']?\$\(\s*(curl|wget)`),
	regexp.MustCompile(`\becho\s+.*\|\s*base64\s+-d\s*\|\s*` + privPrefix + `(ba)?sh`),
	regexp.MustCompile(`\bperl\s+-e\s*['"].*exec`),
	regexp.MustCompile(`\bpython[23]?\s+-c\s*['"].*exec`),
	regexp.MustCompile(`\b` + windowsDelete + `\s+(` + windowsFlag + `\s+)*` + forcedDelete + `\s+` + anyOperand + windowsRoot + operandEnd),
	regexp.MustCompile(`\b` + windowsDelete + `\s+(` + windowsFlag + `\s+)*` + windowsRoot + `\s+(` + windowsFlag + `\s+)*` + forcedDelete + `\b`),
	regexp.MustCompile(`^format\s+["']?[a-z]:`),
	regexp.MustCompile(`\b(format-volume|clear-disk|initialize-disk|diskpart)\b`),
}

var cautionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+(-[rRfv]+\s+)`),
	regexp.MustCompile(`find\s+.*(-delete|-exec\s+rm)`),
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
	regexp.MustCompile(`remove-item\s`),
	regexp.MustCompile(`stop-service\s`),
	regexp.MustCompile(`(restart|stop)-computer`),
	regexp.MustCompile(`set-executionpolicy\s`),
	regexp.MustCompile(`reg\s+delete\s`),
	regexp.MustCompile(`bcdedit`),
}

var assignmentPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*=`)

// find rooted at / or ~ that deletes is blocked unless it narrows the match
// with a name/path/type/time/size predicate.
var (
	unfilteredFindDelete = regexp.MustCompile(`\bfind\s+(/|~|\$home)\s.*(-delete|-exec\s+rm)`)
	findFilterPredicate  = regexp.MustCompile(`\s-(i?name|i?path|i?regex|mtime|mmin|newer|size|empty)\b`)
)

func IsBlocked(command string) bool {
	normalizedCmd := strings.ToLower(strings.TrimSpace(command))

	if matchesBlocked(normalizedCmd) {
		return true
	}

	for _, segment := range splitSegments(normalizedCmd) {
		if matchesBlocked(segment) {
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
		return "●"
	case RiskCaution:
		return "▲"
	case RiskDanger:
		return "✕"
	default:
		return "○"
	}
}

func matchesBlocked(command string) bool {
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(command) {
			return true
		}
	}
	return unfilteredFindDelete.MatchString(command) && !findFilterPredicate.MatchString(command)
}

// splitSegments breaks a command into simple commands on ; && || | & and
// newlines, then strips privilege and environment prefixes from each so the
// blocked patterns see the bare command. Quoting subtleties are ignored.
func splitSegments(command string) []string {
	parts := strings.FieldsFunc(command, func(r rune) bool {
		return r == ';' || r == '|' || r == '&' || r == '\n'
	})

	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if segment := stripPrefixes(part); segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

func stripPrefixes(segment string) string {
	fields := strings.Fields(segment)
	for len(fields) > 0 {
		switch {
		case fields[0] == "sudo" || fields[0] == "doas" || fields[0] == "env":
			fields = stripLeadingFlags(fields[1:])
		case assignmentPattern.MatchString(fields[0]):
			fields = fields[1:]
		default:
			return strings.Join(fields, " ")
		}
	}
	return ""
}

func stripLeadingFlags(fields []string) []string {
	for len(fields) > 0 && strings.HasPrefix(fields[0], "-") {
		flag := fields[0]
		fields = fields[1:]
		if len(fields) > 0 && (flag == "-u" || flag == "-g" || flag == "-p" || flag == "-c") {
			fields = fields[1:]
		}
	}
	return fields
}
