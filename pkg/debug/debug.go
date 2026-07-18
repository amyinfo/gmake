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

package debug

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyra/make/pkg/config"
)

const (
	None     = 0x000
	Basic    = 0x001
	Verbose  = 0x002
	Jobs     = 0x004
	Implicit = 0x008
	Print    = 0x010
	Why      = 0x020
	Makefiles = 0x100
	All      = 0xfff
)

func IsDb(level int) bool {
	return (level & config.DbLevel) != 0
}

func DBS(level int, depth uint, format string, args ...interface{}) {
	if IsDb(level) {
		indent := strings.Repeat(" ", int(depth))
		msg := fmt.Sprintf(format, args...)
		_, _ = fmt.Fprintf(os.Stdout, "%s%s\n", indent, msg)
	}
}

func DBF(level int, format string, file string, args ...interface{}) {
	if IsDb(level) {
		msg := fmt.Sprintf(format, args...)
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", fmt.Sprintf(msg, file))
	}
}

func DB(level int, format string, args ...interface{}) {
	if IsDb(level) {
		_, _ = fmt.Fprintf(os.Stdout, format, args...)
	}
}
