package os

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	IOUnknown        = 0x0001
	IOCombinedOuterr = 0x0002
	IOStdinOK        = 0x0004
	IOStdoutOK       = 0x0008
	IOStderrOK       = 0x0010
)

func CheckIOState() uint {
	var state uint
	if fi, _ := os.Stdin.Stat(); fi != nil {
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			state |= IOStdinOK
		} else {
			state |= IOStdinOK
		}
	}
	if fi, _ := os.Stdout.Stat(); fi != nil {
		state |= IOStdoutOK
	}
	if fi, _ := os.Stderr.Stat(); fi != nil {
		state |= IOStderrOK
	}
	return state
}

func FdInherit(fd int) {
	syscall.CloseOnExec(fd)
}

func FdNoinherit(fd int) {
	syscall.CloseOnExec(fd)
}

func FdSetAppend(fd int) {
	var flags int = 0
	// Use raw syscall for fcntl
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, uintptr(unsafe.Pointer(&flags)))
	flags |= syscall.O_APPEND
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_SETFL, uintptr(flags))
}

func OsAnontmp() int {
	f, err := os.CreateTemp("", "makeXXXXXX")
	if err != nil {
		return -1
	}
	fd := f.Fd()
	f.Close()
	os.Remove(f.Name())
	return int(fd)
}

// ——— Jobserver implementation (POSIX pipes) ———

var (
	jobserverFDs   [2]int
	jobserverEnabled_ bool
)

func JobserverEnabled() uint {
	if jobserverEnabled_ {
		return 1
	}
	return 0
}

func JobserverSetup(jobSlots int, style string) uint {
	if jobSlots <= 1 {
		return 0
	}
	// Create a pipe for the jobserver
	p := [2]int{}
	err := syscall.Pipe(p[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "make: couldn't create jobserver pipe: %v\n", err)
		return 0
	}
	jobserverFDs = p

	// Write N-1 tokens to the pipe
	tokens := make([]byte, jobSlots-1)
	for i := range tokens {
		tokens[i] = '+'
	}
	syscall.Write(jobserverFDs[1], tokens)

	jobserverEnabled_ = true
	return 1
}

func JobserverParseAuth(auth string) uint {
	if auth == "" {
		return 0
	}
	_, err := fmt.Sscanf(auth, "%d,%d", &jobserverFDs[0], &jobserverFDs[1])
	if err != nil {
		return 0
	}
	jobserverEnabled_ = true
	return 1
}

func JobserverGetAuth() string {
	if !jobserverEnabled_ {
		return ""
	}
	return fmt.Sprintf("%d,%d", jobserverFDs[0], jobserverFDs[1])
}

func JobserverClear() {
	jobserverEnabled_ = false
	jobserverFDs = [2]int{}
}

func JobserverAcquireAll() uint {
	if !jobserverEnabled_ {
		return 0
	}
	count := uint(0)
	buf := make([]byte, 1024)
	for {
		n, err := syscall.Read(jobserverFDs[0], buf)
		if err != nil || n == 0 {
			break
		}
		count += uint(n)
	}
	JobserverClear()
	return count
}

func JobserverRelease(isFatal int) {
	if !jobserverEnabled_ {
		return
	}
	syscall.Write(jobserverFDs[1], []byte{'+'})
}

func JobserverPreChild(r int) {
	// No-op on POSIX
}

func JobserverPostChild(r int) {
	// No-op on POSIX
}

func JobserverPreAcquire() {
	// No-op on POSIX
}

func JobserverAcquire(timeout int) uint {
	if !jobserverEnabled_ {
		return 1 // No jobserver = always have a token
	}
	buf := make([]byte, 1)
	n, err := syscall.Read(jobserverFDs[0], buf)
	if err != nil || n == 0 {
		return 0
	}
	return 1
}

func JobserverSignal() {
	// No-op on POSIX
}

// ——— Output sync ———

var osyncEnabled_ bool

func OsyncEnabled() uint {
	if osyncEnabled_ {
		return 1
	}
	return 0
}

func OsyncSetup() {
	osyncEnabled_ = true
}

func OsyncGetMutex() string {
	if !osyncEnabled_ {
		return ""
	}
	return ""
}

func OsyncParseMutex(mutex string) uint {
	_ = mutex
	osyncEnabled_ = true
	return 1
}

func OsyncClear() {
	osyncEnabled_ = false
}

func OsyncAcquire() uint {
	return 1
}

func OsyncRelease() {
}

func GetBadStdin() int {
	f, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
	if err != nil {
		return -1
	}
	return int(f.Fd())
}
