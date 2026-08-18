//go:build !windows

package instance

import "os/exec"

func hideProcessWindow(cmd *exec.Cmd) {}
