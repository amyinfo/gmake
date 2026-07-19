//go:build windows

package job

import (
	"os"
)

func setupChildHandler() {
	// Windows does not have SIGCHLD. Child process exit notification
	// is handled by the per-child goroutine in startJobCommand().
}

func getExitSignalInfo(ws *os.ProcessState) (int, int) {
	// Windows has no signal-based exit or core dump concept.
	return 0, 0
}
