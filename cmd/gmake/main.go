// GNU Make in Go - Entry Point
// Port of GNU make 4.4 main.c
package main

import (
	"fmt"
	goos "os"
	"runtime"

	"github.com/kyra/make/pkg/config"
	"github.com/kyra/make/pkg/debug"
	_ "github.com/kyra/make/pkg/getopt"
	"github.com/kyra/make/pkg/misc"
	_ "github.com/kyra/make/pkg/os"
	"github.com/kyra/make/pkg/strcache"
	"github.com/kyra/make/pkg/types"
	"github.com/kyra/make/pkg/variable"
)

var (
	// Makefile search order
	makefiles = []string{"GNUmakefile", "makefile", "Makefile"}

	dbLevel int
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

	if config.PrintDataBaseFlag {
		// Will print database after reading makefiles
	}

	// Set up default variables
	setDefaultVariables()

	// Read makefiles
	goals := readMakefiles()

	if goals == nil {
		// No targets and no makefile
		noRuleMsg := "*** No targets specified and no makefile found."
		fmt.Fprintln(goos.Stderr, noRuleMsg)
		return config.MakeFailure
	}

	// Build the dependency graph
	snapDeps()

	// Update the goals
	status := updateGoals(goals)

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
}

func decodeSwitches(argv []string) {
	// Parse command-line options
	i := 0
	for i < len(argv) {
		arg := argv[i]
		if arg[0] != '-' {
			// Non-option argument: target
			config.Goals = append(config.Goals, arg)
			i++
			continue
		}

		if arg == "--" {
			// Rest are targets
			config.Goals = append(config.Goals, argv[i+1:]...)
			return
		}

		switch arg {
		case "-b":
			// Ignored for compatibility
		case "-B":
			config.AlwaysMakeFlag = true
		case "-d":
			config.DbLevel = debug.All
		case "--debug":
			if i+1 < len(argv) && argv[i+1][0] != '-' {
				i++
				parseDebugLevels(argv[i])
			} else {
				config.DbLevel = debug.Basic
			}
		case "-e":
			config.EnvOverrides = true
		case "-f":
			i++
			if i >= len(argv) {
				fatal("option '-f' requires an argument")
			}
			config.Makefiles = append(config.Makefiles, argv[i])
			case "-h", "--help":
				printUsage()
				goos.Exit(config.MakeSuccess)
		case "-i":
			config.IgnoreErrorsFlag = true
		case "-I":
			i++
			if i >= len(argv) {
				fatal("option '-I' requires an argument")
			}
			config.IncludeDirs = append(config.IncludeDirs, argv[i])
		case "-j":
			i++
			if i < len(argv) && argv[i][0] != '-' {
				slots := misc.MakeToui(argv[i], nil)
				if slots > 0 {
					config.JobSlots = slots
				}
			} else {
				i--
				config.JobSlots = 9999 // effectively unlimited
			}
		case "-k":
			config.KeepGoingFlag = true
		case "-l":
			i++
			if i < len(argv) && argv[i][0] != '-' {
				var load float64
				fmt.Sscanf(argv[i], "%f", &load)
				config.MaxLoadAverage = load
			}
		case "-n":
			config.JustPrintFlag = true
		case "-o":
			i++
			if i >= len(argv) {
				fatal("option '-o' requires an argument")
			}
			config.OldFile = argv[i]
		case "-p":
			config.PrintDataBaseFlag = true
		case "-q":
			config.QuestionFlag = true
		case "-r":
			config.NoBuiltinRulesFlag = true
		case "-R":
			config.NoBuiltinVariablesFlag = true
		case "-s":
			config.RunSilent = true
		case "-S":
			config.KeepGoingFlag = false
		case "-t":
			config.TouchFlag = true
			case "-v", "--version":
				printVersion()
				goos.Exit(config.MakeSuccess)
		case "-w":
			config.PrintDirectory = true
		case "--no-print-directory":
			config.PrintDirectory = false
		case "-C":
			i++
			if i >= len(argv) {
				fatal("option '-C' requires an argument")
			}
			goos.Chdir(argv[i])
		case "-W":
			i++
			if i >= len(argv) {
				fatal("option '-W' requires an argument")
			}
			config.WhatIf = argv[i]
		case "--warn-undefined-variables":
			config.WarnUndefinedVariables = true
		case "--shuffle":
			i++
			if i < len(argv) && argv[i][0] != '-' {
				config.ShuffleMode = argv[i]
			} else {
				i--
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
			if len(arg) > 1 && arg[1] != '-' {
				// Short option cluster: -j4 etc.
				opt := arg[1]
				_ = opt
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
}

func readMakefiles() []*types.Goaldep {
	// Read all makefiles
	var goals []*types.Goaldep

	if len(config.Makefiles) > 0 {
		// -f option used
		for _, mf := range config.Makefiles {
			g := readMakefile(mf, types.RMNoFlag)
			if g != nil {
				goals = append(goals, g...)
			}
		}
		return goals
	}

	// Try default makefiles
	for _, mf := range makefiles {
		if _, err := goos.Stat(mf); err == nil {
			g := readMakefile(mf, types.RMNoFlag)
			if g != nil {
				return g
			}
		}
	}

	return nil
}

func readMakefile(name string, flags int) []*types.Goaldep {
	// TODO: Call the read package's read_makefile
	if config.DbLevel&debug.Makefiles != 0 {
		fmt.Fprintf(goos.Stderr, "Reading makefile '%s'\n", name)
	}
	return nil
}

func snapDeps() {
	// TODO: Call snap_deps from the file/remake packages
}

func updateGoals(goals []*types.Goaldep) types.UpdateStatus {
	// TODO: Call update_goal_chain from the remake package
	return types.UpdateSuccess
}

func printDataBase() {
	// TODO: Print the internal database
}

func fatal(msg string) {
	fmt.Fprintf(goos.Stderr, "%s: %s\n", config.Program, msg)
	goos.Exit(config.MakeFailure)
}

func initHashFiles() {
	// TODO: Initialize the file hash table (init_hash_files from file.c)
}

func init() {
	_ = misc.StopSet
	_ = debug.DB
}

// These will be removed once all packages are properly integrated
var _ = goos.Getwd
