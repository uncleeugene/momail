//go:build windows

package binkp

import "os/exec"

// getShellCommand returns an *exec.Cmd to run the given command string in a shell.
// On Windows, this uses "cmd /C".
func getShellCommand(cmdStr string) *exec.Cmd {
	return exec.Command("cmd", "/C", cmdStr)
}
