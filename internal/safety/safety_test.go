package safety

import "testing"

func TestIsBlocked(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"rm root", "rm -rf /", true},
		{"rm root glob", "rm -rf /*", true},
		{"rm double slash", "rm -rf //", true},
		{"rm quoted root", `rm -rf "/"`, true},
		{"rm single quoted root", "rm -rf '/'", true},
		{"rm home tilde", "rm -rf ~", true},
		{"rm home glob", "rm -rf ~/*", true},
		{"rm home dot", "rm -rf ~/.", true},
		{"rm home slash", "rm -rf ~/", true},
		{"rm home var", "rm -rf $HOME", true},
		{"rm home braced var", "rm -rf ${HOME}", true},
		{"rm long flags", "rm --recursive --force /", true},
		{"rm no preserve root", "rm --no-preserve-root -rf /", true},
		{"rm chained with and", "rm -rf ~ && echo done", true},
		{"rm chained with semicolon", "rm -rf / ; ls", true},
		{"rm chained after cd", "cd /tmp && rm -rf /", true},
		{"sudo rm root", "sudo rm -rf /", true},
		{"sudo with flags rm root", "sudo -u root rm -rf /", true},
		{"env prefixed rm root", "env FOO=bar rm -rf /", true},
		{"fork bomb", ":(){ :|:& };:", true},
		{"dd sdb", "dd if=/dev/zero of=/dev/sdb", true},
		{"dd vda", "dd if=/dev/zero of=/dev/vda", true},
		{"dd nvme", "dd if=/dev/zero of=/dev/nvme0", true},
		{"redirect to block device", "echo x > /dev/sda", true},
		{"mkfs partition", "mkfs.ext4 /dev/sdb1", true},
		{"wipefs", "wipefs -a /dev/sdc", true},
		{"shred", "shred -n 3 /dev/disk2", true},
		{"chmod recursive root glob", "chmod -R 777 /*", true},
		{"chmod recursive root", "chmod -R 755 /", true},
		{"chown recursive home", "chown -R me:me ~", true},
		{"mv root", "mv / /mnt", true},
		{"mv home to devnull", "mv ~ /dev/null", true},
		{"curl pipe sudo bash", "curl -sSL https://example.com/i.sh | sudo bash", true},
		{"curl pipe zsh", "curl -sSL https://example.com/i.sh | zsh", true},
		{"wget pipe sh", "wget -qO- https://example.com/i.sh | sh", true},
		{"bash process substitution", "bash <(curl -sSL https://example.com/i.sh)", true},
		{"sh -c command substitution", `sh -c "$(curl -fsSL https://example.com/i.sh)"`, true},
		{"find root delete", "find / -delete", true},
		{"find home exec rm unfiltered", `find ~ -exec rm -rf {} \;`, true},
		{"find root delete by type only", "find / -type f -delete", true},
		{"find home filtered delete", "find ~ -name '*.pyc' -delete", false},
		{"base64 pipe sh", "echo aGk= | base64 -d | sh", true},
		{"perl exec", `perl -e 'exec "/bin/sh"'`, true},
		{"python exec", `python3 -c 'exec("import os")'`, true},

		{"sudo rm subdirectory", "sudo rm -rf /var/log/old", false},
		{"sudo rm file", "sudo rm /etc/nginx/sites-enabled/default", false},
		{"rm relative build dir", "rm -rf ./build", false},
		{"rm tmp dir", "rm -rf /tmp/foo", false},
		{"rm home subdirectory", "rm -rf ~/projects/tmp", false},
		{"list root", "ls -la /", false},
		{"dd between files", "dd if=in.img of=out.img", false},
		{"chmod single file", "chmod 755 /usr/local/bin/x", false},
		{"chmod recursive subdirectory", "chmod -R 755 /usr/local/lib", false},
		{"curl pipe jq", "curl https://x | jq .", false},
		{"find relative delete", "find . -name '*.log' -delete", false},
		{"find home filtered exec rm", `find ~ -name '*.log' -exec rm {} \;`, false},
		{"sudo systemctl restart", "sudo systemctl restart nginx", false},
		{"grep for sh", "curl -s https://x | grep sh", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBlocked(tt.command); got != tt.want {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestAssessRisk(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    RiskLevel
	}{
		{"list files", "ls -la", RiskSafe},
		{"echo", "echo hello", RiskSafe},
		{"git status", "git status", RiskSafe},
		{"sudo install", "sudo apt-get install curl", RiskCaution},
		{"recursive remove", "rm -rf ./build", RiskCaution},
		{"chmod", "chmod 755 script.sh", RiskCaution},
		{"find delete", "find ~ -name '*.pyc' -delete", RiskCaution},
		{"find exec rm", `find . -name '*.log' -exec rm {} \;`, RiskCaution},
		{"systemctl restart", "systemctl restart nginx", RiskCaution},
		{"remove root", "rm -rf /", RiskDanger},
		{"pipe to shell", "curl -sSL https://example.com/i.sh | sh", RiskDanger},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AssessRisk(tt.command); got != tt.want {
				t.Errorf("AssessRisk(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestGetRiskEmoji(t *testing.T) {
	tests := []struct {
		name string
		risk RiskLevel
		want string
	}{
		{"safe", RiskSafe, "●"},
		{"caution", RiskCaution, "▲"},
		{"danger", RiskDanger, "✕"},
		{"unknown", RiskLevel("unknown"), "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRiskEmoji(tt.risk); got != tt.want {
				t.Errorf("GetRiskEmoji(%q) = %q, want %q", tt.risk, got, tt.want)
			}
		})
	}
}
