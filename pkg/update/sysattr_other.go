//go:build !windows

package update

import "syscall"

// detachedSysProcAttr is a no-op on non-Windows platforms.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return nil
}
