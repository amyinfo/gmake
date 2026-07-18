//go:build !windows

package signame

import "syscall"

// signalNames is populated by init() functions in platform-specific files.
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

func init() {
	signalNames[syscall.SIGUSR1] = "User Signal 1"
	signalNames[syscall.SIGUSR2] = "User Signal 2"
	signalNames[syscall.SIGCHLD] = "Child Status"
	signalNames[syscall.SIGCONT] = "Continued"
	signalNames[syscall.SIGSTOP] = "Stopped (signal)"
	signalNames[syscall.SIGTSTP] = "Stopped"
	signalNames[syscall.SIGTTIN] = "Stopped (tty input)"
	signalNames[syscall.SIGTTOU] = "Stopped (tty output)"
	signalNames[syscall.SIGURG] = "Urgent IO condition"
	signalNames[syscall.SIGXCPU] = "CPU limit exceeded"
	signalNames[syscall.SIGXFSZ] = "File size limit exceeded"
	signalNames[syscall.SIGVTALRM] = "Virtual Timer Expired"
	signalNames[syscall.SIGPROF] = "Profiling Timer Expired"
	signalNames[syscall.SIGWINCH] = "Window size changed"
	signalNames[syscall.SIGIO] = "I/O possible"
	signalNames[syscall.SIGSYS] = "Bad System Call"
}
