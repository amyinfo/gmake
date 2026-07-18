package signame

import (
	"syscall"
)

var signalNames = map[syscall.Signal]string{
	syscall.SIGHUP:    "Hangup",
	syscall.SIGINT:    "Interrupt",
	syscall.SIGQUIT:   "Quit",
	syscall.SIGILL:    "Illegal Instruction",
	syscall.SIGTRAP:   "Trace/BPT Trap",
	syscall.SIGABRT:   "Abort",
	syscall.SIGBUS:    "Bus Error",
	syscall.SIGFPE:    "Arithmetic Exception",
	syscall.SIGKILL:   "Killed",
	syscall.SIGUSR1:   "User Signal 1",
	syscall.SIGSEGV:   "Segmentation Fault",
	syscall.SIGUSR2:   "User Signal 2",
	syscall.SIGPIPE:   "Broken Pipe",
	syscall.SIGALRM:   "Alarm Clock",
	syscall.SIGTERM:   "Terminated",
	syscall.SIGCHLD:   "Child Status",
	syscall.SIGCONT:   "Continued",
	syscall.SIGSTOP:   "Stopped (signal)",
	syscall.SIGTSTP:   "Stopped",
	syscall.SIGTTIN:   "Stopped (tty input)",
	syscall.SIGTTOU:   "Stopped (tty output)",
	syscall.SIGURG:    "Urgent IO condition",
	syscall.SIGXCPU:   "CPU limit exceeded",
	syscall.SIGXFSZ:   "File size limit exceeded",
	syscall.SIGVTALRM: "Virtual Timer Expired",
	syscall.SIGPROF:   "Profiling Timer Expired",
	syscall.SIGWINCH:  "Window size changed",
	syscall.SIGIO:     "I/O possible",
	syscall.SIGPWR:    "Power Fail/Restart",
	syscall.SIGSYS:    "Bad System Call",
}

func SignalName(sig int) string {
	if name, ok := signalNames[syscall.Signal(sig)]; ok {
		return name
	}
	return "Unknown Signal"
}
