//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli

import (
	"os/exec"
	"syscall"
)

func signalFFplayProcessPause(cmd *exec.Cmd, pause bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	signal := syscall.SIGCONT
	if pause {
		signal = syscall.SIGSTOP
	}
	return cmd.Process.Signal(signal)
}
