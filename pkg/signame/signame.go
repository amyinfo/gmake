package signame

import "syscall"

// SignalName returns the name of a signal number.
func SignalName(sig int) string {
	if name, ok := signalNames[syscall.Signal(sig)]; ok {
		return name
	}
	return "Unknown Signal"
}
