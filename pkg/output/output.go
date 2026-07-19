package output

import (
	"fmt"
	"os"

	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/types"
)

const outputNone = -1

type Output struct {
	Out     int
	Err     int
	Syncout bool
}

var OutputContext *Output
var savedContext []*Output

func OutputSet(out *Output) {
	savedContext = append(savedContext, OutputContext)
	OutputContext = out
}

func OutputUnset() {
	if len(savedContext) > 0 {
		OutputContext = savedContext[len(savedContext)-1]
		savedContext = savedContext[:len(savedContext)-1]
	} else {
		OutputContext = nil
	}
}

func IsOutputSet(out *Output) bool {
	return out != nil && (out.Out >= 0 || out.Err >= 0)
}

func writeOutput(out *Output, isErr int, msg string) {
	if out != nil && out.Syncout {
		fd := out.Err
		if isErr == 0 {
			fd = out.Out
		}
		if fd != outputNone {
			_, _ = os.Stdout.WriteString(msg)
			return
		}
	}

	f := os.Stdout
	if isErr != 0 {
		f = os.Stderr
	}
	_, _ = fmt.Fprint(f, msg)
}

func logWorkingDirectory(entering int) int {
	var msg string
	if config.Makelevel == 0 {
		if config.StartingDirectory == "" {
			if entering != 0 {
				msg = config.Program + ": Entering an unknown directory\n"
			} else {
				msg = config.Program + ": Leaving an unknown directory\n"
			}
		} else {
			if entering != 0 {
				msg = fmt.Sprintf("%s: Entering directory '%s'\n", config.Program, config.StartingDirectory)
			} else {
				msg = fmt.Sprintf("%s: Leaving directory '%s'\n", config.Program, config.StartingDirectory)
			}
		}
	} else {
		if config.StartingDirectory == "" {
			if entering != 0 {
				msg = fmt.Sprintf("%s[%d]: Entering an unknown directory\n", config.Program, config.Makelevel)
			} else {
				msg = fmt.Sprintf("%s[%d]: Leaving an unknown directory\n", config.Program, config.Makelevel)
			}
		} else {
			if entering != 0 {
				msg = fmt.Sprintf("%s[%d]: Entering directory '%s'\n", config.Program, config.Makelevel, config.StartingDirectory)
			} else {
				msg = fmt.Sprintf("%s[%d]: Leaving directory '%s'\n", config.Program, config.Makelevel, config.StartingDirectory)
			}
		}
	}

	writeOutput(nil, 0, msg)
	return 1
}

func OutputInit(out *Output) {
	if out != nil {
		out.Out = outputNone
		out.Err = outputNone
		out.Syncout = config.OutputSync != config.OutputSyncNone
		return
	}
}

func OutputClose(out *Output) {
	if out == nil {
		return
	}
	OutputDump(out)
	OutputInit(out)
}

func OutputStart() {
	if OutputContext != nil && OutputContext.Syncout {
		if !IsOutputSet(OutputContext) {
			setupTmpfile(OutputContext)
		}
	}

	if config.OutputSync == config.OutputSyncNone || config.OutputSync == config.OutputSyncRecurse {
		if config.PrintDirectory {
			logWorkingDirectory(1)
		}
	}
}

func Outputs(isErr int, msg string) {
	if msg == "" {
		return
	}
	OutputStart()
	writeOutput(OutputContext, isErr, msg)
}

func setupTmpfile(out *Output) {
}

func OutputDump(out *Output) {
	outfdNotEmpty := out.Out != outputNone
	errfdNotEmpty := out.Err != outputNone

	if outfdNotEmpty || errfdNotEmpty {
		traced := 0

		if config.PrintDirectory && config.OutputSync != config.OutputSyncRecurse {
			traced = logWorkingDirectory(1)
		}

		_ = outfdNotEmpty
		_ = errfdNotEmpty

		if traced != 0 {
			logWorkingDirectory(0)
		}
	}
}

func Message(prefix int, lenHint int, format string, args ...interface{}) {
	var msg string
	if prefix != 0 {
		if config.Makelevel == 0 {
			msg = config.Program + ": "
		} else {
			msg = fmt.Sprintf("%s[%d]: ", config.Program, config.Makelevel)
		}
	}
	msg += fmt.Sprintf(format, args...)
	msg += "\n"
	Outputs(0, msg)
}

func Error(flocp *types.Floc, lenHint int, format string, args ...interface{}) {
	var msg string
	if flocp != nil && flocp.Filenm != "" {
		msg = fmt.Sprintf("%s:%d: ", flocp.Filenm, flocp.Lineno+flocp.Offset)
	} else if config.Makelevel == 0 {
		msg = config.Program + ": "
	} else {
		msg = fmt.Sprintf("%s[%d]: ", config.Program, config.Makelevel)
	}
	msg += fmt.Sprintf(format, args...)
	msg += "\n"
	Outputs(1, msg)
}

func Fatal(flocp *types.Floc, lenHint int, format string, args ...interface{}) {
	var msg string
	if flocp != nil && flocp.Filenm != "" {
		msg = fmt.Sprintf("%s:%d: *** ", flocp.Filenm, flocp.Lineno+flocp.Offset)
	} else if config.Makelevel == 0 {
		msg = config.Program + ": *** "
	} else {
		msg = fmt.Sprintf("%s[%d]: *** ", config.Program, config.Makelevel)
	}
	msg += fmt.Sprintf(format, args...)
	msg += ".  Stop.\n"
	Outputs(1, msg)
	os.Exit(2)
}

func PerrorWithName(str, name string) {
	errMsg := fmt.Sprintf("%s%s: %s", str, name, "(error)")
	Error(nil, 0, "%s", errMsg)
}

func PfatalWithName(name string) {
	Fatal(nil, 0, "%s: %s", name, "(error)")
}

func OutOfMemory() {
	_, _ = os.Stderr.WriteString(config.Program + ": *** virtual memory exhausted\n")
	os.Exit(2)
}
