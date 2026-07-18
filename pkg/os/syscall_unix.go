//go:build linux || darwin || freebsd || netbsd || openbsd

package os

import (
	"syscall"
)

func setCloseOnExec(fd int, inherit bool) {
	if inherit {
		syscall.CloseOnExec(fd)
	} else {
		_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_SETFD, 0)
	}
}

func setAppend(fd int, enable bool) {
	if !enable {
		return
	}
	flags, _, _ := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
	flags |= syscall.O_APPEND
	_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_SETFL, flags)
}
