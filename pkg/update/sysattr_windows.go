//go:build windows

package update

import "syscall"

// detachedSysProcAttr detaches the relaunched process from the parent's
// console so it survives the parent exiting.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:     true,
		CreationFlags:  0x00000008, // DETACHED_PROCESS
		NoInheritHandles: true,
	}
}
