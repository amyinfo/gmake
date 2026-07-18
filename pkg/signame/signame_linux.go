//go:build linux

package signame

import "syscall"

func init() {
	signalNames[syscall.SIGPWR] = "Power Fail/Restart"
}
