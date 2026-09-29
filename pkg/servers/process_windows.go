package servers

import (
	"os/exec"
	"syscall"
)

// hideConsole prevents a console window from popping up per server process.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
