//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !windows

package os

func setCloseOnExec(fd int, inherit bool) {
}

func setAppend(fd int, enable bool) {
}
