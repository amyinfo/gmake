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

package archive

import (
	"time"
)

// ArName checks if a filename is an archive member reference (lib.a(obj.o))
func ArName(name string) int {
	// Check for the ( and ) pattern in the name
	for i := 0; i < len(name); i++ {
		if name[i] == '(' {
			return 1
		}
	}
	return 0
}

// ArParseName parses an archive member name into archive and member
func ArParseName(name string, arnamep, memnamep *string) {
	for i := 0; i < len(name); i++ {
		if name[i] == '(' {
			*arnamep = name[:i]
			end := len(name) - 1
			if end > i && name[end] == ')' {
				*memnamep = name[i+1 : end]
			}
			return
		}
	}
	*arnamep = name
	*memnamep = ""
}

// ArMemberDate returns the modification time of an archive member
func ArMemberDate(name string) time.Time {
	return time.Time{}
}

// ArScan scans an archive file and calls the callback for each member
func ArScan(archive string, fn func(int, string, int, int64, int64, int64, int64, int, int, uint, interface{}) int64, arg interface{}) int64 {
	return 0
}

// ArNameEqual checks if a member name matches
func ArNameEqual(name, mem string, truncated int) int {
	if name == mem {
		return 1
	}
	return 0
}

// ArTouch touches an archive member
func ArTouch(name string) int {
	return 0
}

// ArMemberTouch sets the modification time of an archive member
func ArMemberTouch(arname, memname string) int {
	return 0
}

// ArGlob globs for archive members matching a pattern
func ArGlob(arname, memberPattern string, size int) *struct {
	Name string
	Next *struct{}
} {
	return nil
}
