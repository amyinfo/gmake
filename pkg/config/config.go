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

// Package config provides the global configuration state for GNU Make.
// This corresponds to the global variables and defines in makeint.h, main.c, etc.
package config

import (
	"math"
	"os"
)

// ——————————————————— Return codes ———————————————————
const (
	MakeSuccess = 0
	MakeTrouble = 1
	MakeFailure = 2
)

// ——————————————————— Output sync modes ———————————————————
const (
	OutputSyncNone    = 0
	OutputSyncLine    = 1
	OutputSyncTarget  = 2
	OutputSyncRecurse = 3
)

// ——————————————————— Global flags ———————————————————
// Corresponds to the global flags declared in makeint.h (lines 736-742)
var (
	JustPrintFlag           bool
	RunSilent               bool
	IgnoreErrorsFlag        bool
	KeepGoingFlag           bool
	PrintDataBaseFlag       bool
	QuestionFlag            bool
	TouchFlag               bool
	AlwaysMakeFlag          bool
	EnvOverrides            bool
	NoBuiltinRulesFlag      bool
	NoBuiltinVariablesFlag  bool
	PrintVersionFlag        bool
	PrintDirectory          bool
	CheckSymlinkFlag        bool
	WarnUndefinedVariables  bool
	PosixPedantic           bool
	NotParallel             bool
	SecondExpansion         bool
	ClockSkewDetected       bool
	RebuildingMakefiles     bool
	OneShell                bool
	OutputSync              int
	VerifyFlag              bool
	CommandCount            uint64

	// makeint.h line 745
	DefaultShell = "/bin/sh"

	// makeint.h line 748
	BatchModeShell bool

	// makeint.h line 756
	CmdPrefix = '\t'

	// makeint.h line 760-762
	JobserverAuth   string
	JobSlots        uint
	MaxLoadAverage  float64

	// makeint.h line 764
	Program string

	// makeint.h line 807-809
	StartingDirectory string
	Makelevel         uint
	VersionString     string
	RemoteDescription string
	MakeHost          string

	// makeint.h line 811
	CommandsStarted uint

	// From filedef.h line 229
	SnappedDeps bool

	// From variable.h
	ExportAllVariables bool

	// Jobserver
	JobserverTokens uint
	JobSlotsUsed    uint

	// From main.c
	Environ = os.Environ()

	// Additional fields used by main.go
	Goals           []string
	Makefiles       []string
	IncludeDirs     []string
	OldFile         string
	WhatIf          string
	ShuffleMode     string
	JobserverStyle  string
	DefaultGoalName string
)

// GNUMAKEFLAGS_NAME / MAKEFLAGS_NAME
const (
	GNUMAKEFLAGSName = "GNUMAKEFLAGS"
	MAKEFLAGSName    = "MAKEFLAGS"

	RECIPEPREFIXName    = ".RECIPEPREFIX"
	RECIPEPREFIXDefault = '\t'

	JOBSERVERAuthOpt = "jobserver-auth"

	MAKELEVELName       = "MAKELEVEL"
	MAKELEVELLength     = 9
)

// Character classification constants (from makeint.h MAP_*)
const (
	MapNul     = 0x0001
	MapBlank   = 0x0002 // space, TAB
	MapNewline = 0x0004
	MapComment = 0x0008
	MapSemi    = 0x0010
	MapEquals  = 0x0020
	MapColon   = 0x0040
	MapVarsep  = 0x0080
	MapPipe    = 0x0100
	MapDot     = 0x0200
	MapComma   = 0x0400

	MapUserFunc = 0x2000
	MapVariable = 0x4000
	MapDirsep   = 0x8000

	MapSpace = MapBlank | MapNewline
)

// Path separator handling
var (
	PathSeparatorChar byte = ':'
	MapPathsep             = MapColon
)

// File timestamp type
type FileTimestamp uint64

// Special timestamp values (filedef.h lines 200-213)
const (
	UnknownMtime    FileTimestamp = 0
	NonexistentMtime FileTimestamp = 1
	OldMtime        FileTimestamp = 2
)

// Ordinary mtime range
var (
	OrdinaryMtimeMin = FileTimestamp(OldMtime + 1)
	OrdinaryMtimeMax = FileTimestamp(math.MaxUint64)
)

func IsOrdinaryMtime(t FileTimestamp) bool {
	return t >= OrdinaryMtimeMin
}

// NewMtime - infinitely new timestamp
var NewMtime = FileTimestamp(math.MaxUint64)

// Debug levels (from debug.h)
const (
	DbNone     = 0x000
	DbBasic    = 0x001
	DbVerbose  = 0x002
	DbJobs     = 0x004
	DbImplicit = 0x008
	DbPrint    = 0x010
	DbWhy      = 0x020
	DbMakefiles = 0x100
	DbAll      = 0xfff
)

var DbLevel int

func IsDb(level int) bool {
	return (level & DbLevel) != 0
}
