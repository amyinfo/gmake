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

package main

import (
	"fmt"
	goos "os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/debug"
	"github.com/amyinfo/gmake/pkg/expand"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/function"
	_ "github.com/amyinfo/gmake/pkg/getopt"
	"github.com/amyinfo/gmake/pkg/misc"
	_ "github.com/amyinfo/gmake/pkg/os"
	"github.com/amyinfo/gmake/pkg/read"
	"github.com/amyinfo/gmake/pkg/remake"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
)

var (
	// Makefile search order
	makefiles = []string{"GNUmakefile", "makefile", "Makefile"}
)

func main() {
	goos.Exit(realMain())
}

func realMain() int {
	// Initialize
	initMake()

	// Parse command-line arguments
	argv := goos.Args[1:]
	decodeSwitches(argv)

	// Handle special flags
	if config.PrintVersionFlag {
		printVersion()
		return config.MakeSuccess
	}

	// Set up default variables
	setDefaultVariables()

	// Read makefiles
	readMakefiles()

	// Build the dependency graph
	snapDeps()

	targets := determineTargets()
	if targets == nil {
		fmt.Fprintln(goos.Stderr, "*** No targets specified and no makefile found.")
		return config.MakeSuccess
	}
	goals := buildGoalChain(targets)

	status := remake.UpdateGoalChain(goals)

	// Map internal status to OS exit code:
	//   UpdateSuccess(1) → MakeSuccess(0)
	//   UpdateFailed(3)  → MakeFailure(2)
	//   UpdateQuestion(2) → MakeTrouble(1) when -q, MakeSuccess(0) otherwise
	switch status {
	case types.UpdateSuccess:
		status = types.UpdateStatus(config.MakeSuccess)
	case types.UpdateQuestion:
		if config.QuestionFlag {
			status = types.UpdateStatus(config.MakeTrouble)
		} else {
			status = types.UpdateStatus(config.MakeSuccess)
		}
	default:
		status = types.UpdateStatus(config.MakeFailure)
	}

	// Print database if requested
	if config.PrintDataBaseFlag {
		printDataBase()
	}

	// Print debugging stats
	if config.DbLevel > 0 {
		strcache.PrintStats("")
	}

	return int(status)
}

func initMake() {
	runtime.GOMAXPROCS(0)

	config.VersionString = "4.4.0.Go"
	config.Program = "gmake"
	config.StartingDirectory, _ = goos.Getwd()
	config.DbLevel = 0

	strcache.Init()

	// Initialize file hash table
	initHashFiles()

	// Initialize global variable set
	variable.InitHashGlobalVariableSet()
	variable.InitHashFunctionTable()

	// Set up default variables
	variable.DefineVariableCname("MAKE", config.Program,
		types.OriginDefault, false)
	variable.DefineVariableCname("MAKEFLAGS", "",
		types.OriginDefault, false)
	variable.DefineVariableCname("GNUMAKEFLAGS", "",
		types.OriginDefault, false)
	variable.DefineVariableCname("MAKELEVEL", "0",
		types.OriginDefault, false)
	variable.DefineVariableCname("SHELL", config.DefaultShell,
		types.OriginDefault, false)

	function.Init()
}

func decodeSwitches(argv []string) {
	i := 0
	for i < len(argv) {
		arg := argv[i]
		if arg == "--" {
			config.Goals = append(config.Goals, argv[i+1:]...)
			return
		}
		if arg[0] != '-' || arg == "-" {
			// Check for VAR=value command-line variable override
			if eq := strings.IndexByte(arg, '='); eq > 0 && !strings.HasPrefix(arg, "-") {
				name := arg[:eq]
				value := arg[eq+1:]
				variable.DefineVariableCname(name, value,
					types.OriginCommand, false)
				i++
				continue
			}
			config.Goals = append(config.Goals, arg)
			i++
			continue
		}

		// Long option: --xxxx
		if len(arg) > 2 && arg[1] == '-' {
			switch arg {
			case "--debug":
				if i+1 < len(argv) && argv[i+1][0] != '-' {
					i++
					parseDebugLevels(argv[i])
				} else {
					config.DbLevel = debug.Basic
				}
			case "--help":
				printUsage()
				goos.Exit(config.MakeSuccess)
			case "--version":
				printVersion()
				goos.Exit(config.MakeSuccess)
			case "--quiet", "--silent":
				config.RunSilent = true
			case "--warn-undefined-variables":
				config.WarnUndefinedVariables = true
			case "--shuffle":
				if i+1 < len(argv) && argv[i+1][0] != '-' {
					i++
					config.ShuffleMode = argv[i]
				} else {
					config.ShuffleMode = "random"
				}
			case "--jobserver-style":
				i++
				if i >= len(argv) {
					fatal("option '--jobserver-style' requires an argument")
				}
				config.JobserverStyle = argv[i]
			case "--output-sync":
				i++
				if i < len(argv) && argv[i][0] != '-' {
					switch argv[i] {
					case "none":
						config.OutputSync = config.OutputSyncNone
					case "line":
						config.OutputSync = config.OutputSyncLine
					case "target":
						config.OutputSync = config.OutputSyncTarget
					case "recurse":
						config.OutputSync = config.OutputSyncRecurse
					}
				}
			default:
				fatal("unknown option '" + arg + "'")
			}
			i++
			continue
		}

		// Short option(s): -x, -xf, -j4, -sf, etc.
		for j := 1; j < len(arg); j++ {
			opt := arg[j]
			switch opt {
			case 'b', 'm':
				// Ignored for compatibility
			case 'B':
				config.AlwaysMakeFlag = true
			case 'C':
				i++
				if i >= len(argv) {
					fatal("option '-C' requires an argument")
				}
				if err := goos.Chdir(argv[i]); err != nil {
					fatal("Cannot change directory to " + argv[i] + ": " + err.Error())
				}
			case 'd':
				config.DbLevel = debug.All
			case 'e':
				config.EnvOverrides = true
			case 'f':
				i++
				if i >= len(argv) {
					fatal("option '-f' requires an argument")
				}
				config.Makefiles = append(config.Makefiles, argv[i])
			case 'h':
				printUsage()
				goos.Exit(config.MakeSuccess)
			case 'i':
				config.IgnoreErrorsFlag = true
			case 'I':
				i++
				if i >= len(argv) {
					fatal("option '-I' requires an argument")
				}
				config.IncludeDirs = append(config.IncludeDirs, argv[i])
			case 'j':
				remaining := arg[j+1:]
				if remaining != "" {
					// -jN: N attached
					slots := misc.MakeToui(remaining, nil)
					if slots > 0 {
						config.JobSlots = slots
					}
					j = len(arg) // skip rest of arg
				} else if i+1 < len(argv) && argv[i+1][0] != '-' {
					// -j N: N is next arg
					i++
					slots := misc.MakeToui(argv[i], nil)
					if slots > 0 {
						config.JobSlots = slots
					}
				} else {
					config.JobSlots = 9999
				}
			case 'k':
				config.KeepGoingFlag = true
			case 'l':
				remaining := arg[j+1:]
				if remaining != "" {
					var load float64
					_, _ = fmt.Sscanf(remaining, "%f", &load)
					config.MaxLoadAverage = load
					j = len(arg)
				} else if i+1 < len(argv) && argv[i+1][0] != '-' {
					i++
					var load float64
					_, _ = fmt.Sscanf(argv[i], "%f", &load)
					config.MaxLoadAverage = load
				}
			case 'n':
				config.JustPrintFlag = true
			case 'o':
				i++
				if i >= len(argv) {
					fatal("option '-o' requires an argument")
				}
				config.OldFile = argv[i]
			case 'p':
				config.PrintDataBaseFlag = true
			case 'q':
				config.QuestionFlag = true
			case 'r':
				config.NoBuiltinRulesFlag = true
			case 'R':
				config.NoBuiltinVariablesFlag = true
			case 's':
				config.RunSilent = true
			case 'S':
				config.KeepGoingFlag = false
			case 't':
				config.TouchFlag = true
			case 'v':
				printVersion()
				goos.Exit(config.MakeSuccess)
			case 'w':
				config.PrintDirectory = true
			case 'W':
				i++
				if i >= len(argv) {
					fatal("option '-W' requires an argument")
				}
				config.WhatIf = argv[i]
			default:
				fatal("unknown option -- '" + string(opt) + "'")
			}
			if j >= len(arg) {
				break
			}
		}
		i++
	}
}

func parseDebugLevels(levels string) {
	for _, c := range levels {
		switch c {
		case 'a':
			config.DbLevel = debug.All
		case 'b':
			config.DbLevel |= debug.Basic
		case 'j':
			config.DbLevel |= debug.Jobs
		case 'i':
			config.DbLevel |= debug.Implicit
		case 'm':
			config.DbLevel |= debug.Makefiles
		case 'v':
			config.DbLevel |= debug.Verbose
		case 'p':
			config.DbLevel |= debug.Print
		case 'w':
			config.DbLevel |= debug.Why
		}
	}
}

func printUsage() {
	fmt.Println("Usage: gmake [options] [target] ...")
	fmt.Println("Options:")
	fmt.Println("  -b, -m            Ignored for compatibility")
	fmt.Println("  -B, --always-make  Unconditionally make all targets")
	fmt.Println("  -d                Print lots of debugging information")
	fmt.Println("  --debug[=FLAGS]   Print various types of debugging information")
	fmt.Println("  -e, --environment-overrides")
	fmt.Println("                    Environment variables override makefiles")
	fmt.Println("  -f FILE           Read FILE as a makefile")
	fmt.Println("  -h, --help        Print this message and exit")
	fmt.Println("  -i, --ignore-errors")
	fmt.Println("                    Ignore errors from recipes")
	fmt.Println("  -I DIR            Search DIR for included makefiles")
	fmt.Println("  -j [N]            Allow N jobs at once (default 1)")
	fmt.Println("  -k, --keep-going  Keep going when some targets can't be made")
	fmt.Println("  -l [N]            Don't start multiple jobs if load > N")
	fmt.Println("  -n, --just-print  Print recipes without executing")
	fmt.Println("  -o FILE           Consider FILE to be very old")
	fmt.Println("  -p, --print-data-base")
	fmt.Println("                    Print make's internal database")
	fmt.Println("  -q, --question    Run no recipe; exit 0 if up to date")
	fmt.Println("  -r, --no-builtin-rules")
	fmt.Println("                    Disable built-in implicit rules")
	fmt.Println("  -R, --no-builtin-variables")
	fmt.Println("                    Disable built-in variable settings")
	fmt.Println("  -s, --silent      Silent operation")
	fmt.Println("  -S, --no-keep-going")
	fmt.Println("                    Turns off -k")
	fmt.Println("  -t, --touch       Touch targets instead of remaking them")
	fmt.Println("  -v, --version     Print version and exit")
	fmt.Println("  -w, --print-directory")
	fmt.Println("                    Print the current directory")
	fmt.Println("  -C DIR            Change to DIR before doing anything")
	fmt.Println("  -W FILE           Consider FILE to be infinitely new")
	fmt.Println("  --warn-undefined-variables")
	fmt.Println("                    Warn when an undefined variable is expanded")
}

func printVersion() {
	fmt.Printf("GNU Make %s (Go port)\n", config.VersionString)
	fmt.Println("Copyright (C) 1988-2022 Free Software Foundation, Inc.")
	fmt.Println("Go port (C) 2024")
	fmt.Println("This is free software; see the source for copying conditions.")
}

func setDefaultVariables() {
	// Set up the default variables from the C default.c
	for name, val := range config.DefaultVariables {
		variable.DefineVariableCname(name, val,
			types.OriginDefault, false)
	}

	// Override MAKE to the actual binary path (as GNU Make does)
	makePath := goos.Args[0]
	if abs, err := filepath.Abs(makePath); err == nil {
		makePath = abs
	}
	variable.DefineVariableCname("MAKE", makePath,
		types.OriginDefault, false)
}

func readMakefiles() []*types.Goaldep {
	variable.Expand = expand.VariableExpand
	read.ConstructIncludePath("")

	goals := read.ReadAllMakefiles(config.Makefiles)
	return goals
}

func snapDeps() {
	file.SnapDeps()
}

func buildGoalChain(targets []string) *types.Goaldep {
	var head, tail *types.Goaldep
	for _, t := range targets {
		f := file.EnterFile(strcache.Add(t))
		g := &types.Goaldep{
			Dep: types.Dep{File: f, Name_: f.Name},
		}
		f.IsTarget = true
		if head == nil {
			head = g
			tail = g
		} else {
			tail.Dep.Next = &g.Dep
			tail = g
		}
	}
	return head
}

func determineTargets() []string {
	if len(config.Goals) > 0 {
		return config.Goals
	}
	if config.DefaultGoalName != "" {
		return []string{config.DefaultGoalName}
	}
	allTargets := file.BuildTargetList("")
	if allTargets != "" {
		fields := strings.Fields(allTargets)
		if len(fields) > 0 {
			return fields[:1]
		}
	}
	return nil
}

func printDataBase() {
	// TODO: Print the internal database
}

func fatal(msg string) {
	fmt.Fprintf(goos.Stderr, "%s: %s\n", config.Program, msg)
	goos.Exit(config.MakeFailure)
}

func initHashFiles() {
	file.InitHashFiles()
}

func init() {
	_ = misc.StopSet
	_ = debug.DB
}

// These will be removed once all packages are properly integrated
var _ = goos.Getwd
