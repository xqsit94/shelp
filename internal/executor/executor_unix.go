//go:build !windows

package executor

import (
	"os"
	"os/exec"
	"syscall"
)

const (
	defaultShell  = "bash"
	fallbackShell = "sh"
)

func cancelProcess(cmd *exec.Cmd) error {
	return cmd.Process.Signal(os.Interrupt)
}

// A command killed by a signal carries no exit status of its own, so report it
// the way a shell does.
func exitCodeOf(err *exec.ExitError) int {
	if status, ok := err.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return err.ExitCode()
}
