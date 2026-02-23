//go:build !windows

package binkp

import "os/exec"

// getShellCommand returns an *exec.Cmd to run the given command string in a shell.
// On Unix-like systems, this uses "sh -c".
func getShellCommand(cmdStr string) *exec.Cmd {
	return exec.Command("sh", "-c", cmdStr)
}
