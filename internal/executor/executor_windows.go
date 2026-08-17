//go:build windows

package executor

import "os/exec"

const (
	defaultShell  = "cmd"
	fallbackShell = "cmd"
)

// Windows has no interrupt signal to deliver, so a cancelled command is killed.
func cancelProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func exitCodeOf(err *exec.ExitError) int {
	return err.ExitCode()
}
