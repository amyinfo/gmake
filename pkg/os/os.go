// gmake - Go port of GNU Make
//
// Copyright (C) 1988-2022 Free Software Foundation, Inc.
// Copyright (C) 2026 amyinfo
//
// gmake is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// gmake is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with gmake.  If not, see <https://www.gnu.org/licenses/>.

package os

import (
	goos "os"
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
	if _, err := goos.Stdin.Stat(); err == nil {
		state |= IOStdinOK
	}
	if _, err := goos.Stdout.Stat(); err == nil {
		state |= IOStdoutOK
	}
	if _, err := goos.Stderr.Stat(); err == nil {
		state |= IOStderrOK
	}
	return state
}

// FdInherit sets close-on-exec flag.
func FdInherit(fd int) {
	setCloseOnExec(fd, true)
}

// FdNoinherit clears close-on-exec flag.
func FdNoinherit(fd int) {
	setCloseOnExec(fd, false)
}

// FdSetAppend sets the O_APPEND flag on a file descriptor.
func FdSetAppend(fd int) {
	setAppend(fd, true)
}

// OsAnontmp creates an anonymous temporary file.
func OsAnontmp() int {
	f, err := goos.CreateTemp("", "makeXXXXXX")
	if err != nil {
		return -1
	}
	fd := int(f.Fd())
	_ = f.Close()
	_ = goos.Remove(f.Name())
	return fd
}

// ——— Jobserver ———

// Jobserver uses a channel-based semaphore in this Go port.
var (
	jobserverTokensCh chan struct{}
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
	jobserverTokensCh = make(chan struct{}, jobSlots-1)
	for i := 0; i < jobSlots-1; i++ {
		jobserverTokensCh <- struct{}{}
	}
	jobserverEnabled_ = true
	return 1
}

func JobserverParseAuth(auth string) uint {
	_ = auth
	// In a single-process Go port, the jobserver is always local.
	jobserverEnabled_ = true
	if jobserverTokensCh == nil {
		jobserverTokensCh = make(chan struct{}, 1)
	}
	return 1
}

func JobserverGetAuth() string {
	return ""
}

func JobserverClear() {
	jobserverEnabled_ = false
	jobserverTokensCh = nil
}

func JobserverAcquireAll() uint {
	if !jobserverEnabled_ || jobserverTokensCh == nil {
		return 0
	}
	count := uint(0)
	for {
		select {
		case <-jobserverTokensCh:
			count++
		default:
			JobserverClear()
			return count
		}
	}
}

func JobserverRelease(isFatal int) {
	if !jobserverEnabled_ || jobserverTokensCh == nil {
		return
	}
	jobserverTokensCh <- struct{}{}
}

func JobserverPreChild(r int) {}
func JobserverPostChild(r int) {}
func JobserverPreAcquire()    {}

func JobserverAcquire(timeout int) uint {
	if !jobserverEnabled_ || jobserverTokensCh == nil {
		return 1
	}
	if timeout != 0 {
		select {
		case <-jobserverTokensCh:
			return 1
		default:
			return 0
		}
	}
	<-jobserverTokensCh
	return 1
}

func JobserverSignal() {}

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

func OsyncRelease() {}

func GetBadStdin() int {
	f, err := goos.OpenFile("/dev/null", goos.O_RDONLY, 0)
	if err != nil {
		return -1
	}
	return int(f.Fd())
}

// Export for main.go
var Stderr = goos.Stderr

