//go:build !windows

package job

import (
	"os"
	"os/signal"
	"syscall"
)

func setupChildHandler() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGCHLD)
		for range c {
			childrenMu.Lock()
			deadChildren++
			childrenMu.Unlock()
			select {
			case childExited <- struct{}{}:
			default:
			}
		}
	}()
}

func getExitSignalInfo(ws *os.ProcessState) (int, int) {
	if status, ok := ws.Sys().(syscall.WaitStatus); ok {
		exitSig := 0
		coredump := 0
		if status.Signaled() {
			exitSig = int(status.Signal())
		}
		if status.CoreDump() {
			coredump = 1
		}
		return exitSig, coredump
	}
	return 0, 0
}
