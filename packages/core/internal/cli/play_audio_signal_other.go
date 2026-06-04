//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package cli

import "os/exec"

func signalFFplayProcessPause(cmd *exec.Cmd, pause bool) error {
	return nil
}
