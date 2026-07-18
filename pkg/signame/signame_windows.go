//go:build windows

package signame

import "syscall"

var signalNames = map[syscall.Signal]string{
	syscall.SIGHUP:  "Hangup",
	syscall.SIGINT:  "Interrupt",
	syscall.SIGQUIT: "Quit",
	syscall.SIGILL:  "Illegal Instruction",
	syscall.SIGTRAP: "Trace/BPT Trap",
	syscall.SIGABRT: "Abort",
	syscall.SIGBUS:  "Bus Error",
	syscall.SIGFPE:  "Arithmetic Exception",
	syscall.SIGKILL: "Killed",
	syscall.SIGSEGV: "Segmentation Fault",
	syscall.SIGPIPE: "Broken Pipe",
	syscall.SIGALRM: "Alarm Clock",
	syscall.SIGTERM: "Terminated",
}
