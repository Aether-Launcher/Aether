//go:build !windows

package servers

import (
	"os/exec"
)

// hideConsole is a no-op outside Windows.
func hideConsole(cmd *exec.Cmd) {
}
