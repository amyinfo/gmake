//go:build windows

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

func setCloseOnExec(fd int, inherit bool) {
	// Windows handles inheritance via CreateProcess flags.
	// No-op for now; job objects and process flags handle this.
}

func setAppend(fd int, enable bool) {
	// Windows: FILE_APPEND_DATA is set via CreateFile flags.
	// No-op for now.
}
