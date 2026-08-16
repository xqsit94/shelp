package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
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

	cmd := exec.CommandContext(ctx, resolveShell(shell), "-c", command)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
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

		result.ExitCode = exitErr.ExitCode()
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			result.ExitCode = 128 + int(status.Signal())
		}
	}

	return result, nil
}

// $SHELL may name a shell that is not installed here (fish or zsh on a bare
// server), so fall back to the one shell that is always present.
func resolveShell(shell string) string {
	if shell == "" {
		shell = "bash"
	}

	if _, err := exec.LookPath(shell); err != nil {
		return "sh"
	}

	return shell
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
