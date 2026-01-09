package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xqsit94/shelp/internal/safety"
)

type Result struct {
	Command  string
	Output   string
	Error    string
	ExitCode int
}

func Execute(command, shell string) (*Result, error) {
	if safety.IsBlocked(command) {
		return nil, fmt.Errorf("command blocked for safety reasons")
	}

	var cmd *exec.Cmd
	switch shell {
	case "zsh":
		cmd = exec.Command("zsh", "-c", command)
	case "fish":
		cmd = exec.Command("fish", "-c", command)
	case "sh":
		cmd = exec.Command("sh", "-c", command)
	default:
		cmd = exec.Command("bash", "-c", command)
	}

	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Command:  command,
		Output:   strings.TrimSpace(stdout.String()),
		Error:    strings.TrimSpace(stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %v", err)
		}
	}

	return result, nil
}

func DetectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}

	if strings.HasSuffix(shell, "/zsh") {
		return "zsh"
	}
	if strings.HasSuffix(shell, "/fish") {
		return "fish"
	}
	if strings.HasSuffix(shell, "/bash") {
		return "bash"
	}
	if strings.HasSuffix(shell, "/sh") {
		return "sh"
	}

	return "bash"
}

func IsShellAvailable(shell string) bool {
	_, err := exec.LookPath(shell)
	return err == nil
}
