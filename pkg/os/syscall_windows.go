//go:build windows

package os

func setCloseOnExec(fd int, inherit bool) {
	// Windows handles inheritance via CreateProcess flags.
	// No-op for now; job objects and process flags handle this.
}

func setAppend(fd int, enable bool) {
	// Windows: FILE_APPEND_DATA is set via CreateFile flags.
	// No-op for now.
}
