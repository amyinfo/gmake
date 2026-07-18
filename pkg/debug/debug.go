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
		fmt.Fprintf(os.Stdout, "%s\n", fmt.Sprintf(msg, file))
	}
}

func DB(level int, format string, args ...interface{}) {
	if IsDb(level) {
		fmt.Fprintf(os.Stdout, format, args...)
	}
}
