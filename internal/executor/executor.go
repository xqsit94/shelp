package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/xqsit94/shelp/internal/safety"
)

const waitDelay = 3 * time.Second

type Options struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Result struct {
	Command     string
	ExitCode    int
	Interrupted bool
}

func Execute(ctx context.Context, command, shell string, opts Options) (*Result, error) {
	if safety.IsBlocked(command) {
		return nil, fmt.Errorf("command blocked for safety reasons")
	}

	name, args := shellArgs(resolveShell(shell), command)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error { return cancelProcess(cmd) }
	cmd.WaitDelay = waitDelay

	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	cmd.Stdin = os.Stdin
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}

	cmd.Stdout = os.Stdout
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	}

	cmd.Stderr = os.Stderr
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	}

	err := cmd.Run()

	result := &Result{
		Command:     command,
		Interrupted: ctx.Err() != nil,
	}

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to execute command: %v", err)
		}

		result.ExitCode = exitCodeOf(exitErr)
	}

	return result, nil
}

// $SHELL may name a shell that is not installed here (fish or zsh on a bare
// server), so fall back to the one shell that is always present.
func resolveShell(shell string) string {
	if shell == "" {
		shell = defaultShell
	}

	if _, err := exec.LookPath(shell); err != nil {
		return fallbackShell
	}

	return shell
}

func shellArgs(shell, command string) (string, []string) {
	switch shell {
	case "pwsh", "powershell":
		return shell, []string{"-NoProfile", "-Command", command}
	case "cmd":
		return shell, []string{"/C", command}
	default:
		return shell, []string{"-c", command}
	}
}

func DetectShell() string {
	return detectShell(runtime.GOOS, os.Getenv("SHELL"), exec.LookPath)
}

// Windows has no $SHELL, so the best available PowerShell is preferred and
// cmd is the last resort.
func detectShell(goos, shellEnv string, lookPath func(string) (string, error)) string {
	if goos == "windows" {
		for _, shell := range []string{"pwsh", "powershell"} {
			if _, err := lookPath(shell); err == nil {
				return shell
			}
		}
		return "cmd"
	}

	for _, shell := range []string{"zsh", "fish", "bash", "sh"} {
		if strings.HasSuffix(shellEnv, "/"+shell) {
			return shell
		}
	}

	return "bash"
}
