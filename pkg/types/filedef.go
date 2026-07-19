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

package types

import "os/exec"

// File represents a target file that the makefile says how to make.
// Port of struct file from filedef.h (lines 29-116)
type File struct {
	Name     string
	Hname    string    // hashed filename
	Vpath    string    // VPATH/vpath pathname
	Deps     *Dep      // all dependencies, including duplicates
	Cmds     *Commands // commands to execute
	Stem     string    // implicit stem for implicit rules
	AlsoMake *Dep      // targets made by making this

	Prev *File // previous entry for same filename (double-colon)
	Last *File // last entry for same filename

	Renamed *File // file renamed to

	Variables    *VariableSetList // variable sets for this file
	PatVariables *VariableSetList // pattern-specific variable reference
	Parent       *File            // immediate dependent causing remaking
	DoubleColon  *File            // first double-colon entry for same file

	LastMtime         uint64 // file's modtime if known
	MtimeBeforeUpdate uint64 // modtime before update
	Considered        uint   // considered on current scan

	CommandFlags  int          // flags OR'd for cmds
	UpdateStatus  UpdateStatus // last update attempt status
	CommandState  CmdState     // command execution state

	// Bit flags
	Builtin            bool // true if builtin rule
	Precious           bool // don't delete on quit
	Loaded             bool // true if loaded object
	Unloaded           bool // true if this loaded object was unloaded
	LowResolutionTime  bool // timestamp has only 1-second resolution
	TriedImplicit      bool // searched for implicit rule
	Updating           bool // updating deps
	Updated            bool // has been remade
	IsTarget           bool // described as target
	CmdTarget          bool // given on cmd line
	Phony              bool // phony target (.PHONY prereq)
	Intermediate       bool // intermediate file
	IsExplicit         bool // explicitly mentioned
	Secondary          bool // don't remove_intermediates
	Notintermediate    bool // prereq to .NOTINTERMEDIATE
	Dontcare           bool // no complaint if can't be remade
	IgnoreVpath        bool // threw out VPATH name
	PatSearched        bool // searched for pattern-specific vars
	NoDiag             bool // failed and no diagnostics issued
	WasShuffled        bool // already shuffled 'deps'
	Snapped            bool // deps have been secondary expanded
}

// Commands represents the shell commands to make a file.
// Port of struct commands from commands.h (lines 20-30)
type Commands struct {
	Fileinfo      Floc     // where commands were defined
	Commands      string   // commands text
	CommandLines  []string // commands chopped up into lines
	LinesFlags    []byte   // flag bits for each line
	NCommandLines uint16   // number of command lines
	RecipePrefix  byte     // recipe prefix char
	AnyRecurse    bool     // any line has COMMANDS_RECURSE bit set
}

// Dep represents one dependency of a file.
// Port of struct dep from dep.h (lines 45-63)
// Also the base for goaldep and nameseq
type Dep struct {
	Next  *Dep
	Name_ string
	File  *File
	Shuf  *Dep // shuffled pointer
	Stem  string

	Flags               byte
	Changed             bool
	IgnoreMtime         bool
	Staticpattern       bool
	Need2ndExpansion    bool
	IgnoreAutomaticVars bool
	IsExplicit          bool
	WaitHere            bool
}

// Name returns the dependency name - from the field or the file.
func (d *Dep) Name() string {
	if d.Name_ != "" {
		return d.Name_
	}
	if d.File != nil {
		return d.File.Name
	}
	return ""
}

// Goaldep is a dependency that represents a goal to be built.
// Port of struct goaldep from dep.h (lines 69-74)
type Goaldep struct {
	Dep
	Error int
	Floc  Floc
}

// Nameseq is used in chains of names for parsing and globbing.
// Port of struct nameseq from dep.h (lines 24-27)
type Nameseq struct {
	Next *Nameseq
	Name string
}

// Rule represents a pattern (implicit) rule.
// Port of struct rule from rule.h (lines 20-32)
type Rule struct {
	Next     *Rule
	Targets  []string   // targets of the rule
	Lens     []uint     // lengths of each target
	Suffixes []string   // suffixes (after '%')
	Deps     *Dep       // dependencies
	Cmds     *Commands  // commands
	Defn     string     // definition of the rule
	Num      uint16     // number of targets
	Terminal byte       // terminal (double-colon)
	InUse    byte       // in use by a parent pattern_search
}

// PatternSpec is used for installing pattern rules.
// Port of struct pspec from rule.h (lines 35-38)
type PatternSpec struct {
	Target, Dep, Commands string
}

// Variable represents one variable definition.
// Port of struct variable from variable.h (lines 62-88)
type Variable struct {
	Name       string
	Value      string
	Fileinfo   Floc
	Length     uint
	Recursive  bool // gets recursively re-evaluated
	Append     bool // appending target-specific
	Conditional bool // set with ?=
	PerTarget  bool // target-specific
	Special    bool // special variable
	Exportable bool // could be exported
	Expanding  bool // currently being expanded
	PrivateVar bool // no inheritance of target-specific
	ExpCount   uint16 // >1 allows self-referential expansions
	Flavor     VariableFlavor
	Origin     VariableOrigin
	Export     VariableExport
}

// VariableSet represents a set of variables.
// Port of struct variable_set from variable.h (lines 92-95)
type VariableSet struct {
	// Using a map for the variables - in C this is a hash table
	Variables map[string]*Variable
}

// VariableSetList represents a chain of variable sets.
// Port of struct variable_set_list from variable.h (lines 99-104)
type VariableSetList struct {
	Next         *VariableSetList
	Set          *VariableSet
	NextIsParent bool // true if next is a parent target
}

// PatternVar is used for pattern-specific variables.
// Port of struct pattern_var from variable.h (lines 108-115)
type PatternVar struct {
	Next     *PatternVar
	Suffix   string
	Target   string
	Len      uintptr
	Variable Variable
}

// Output represents output context for a child process.
// Port of struct output from output.h (lines 17-22)
type Output struct {
	Out     int
	Err     int
	SyncOut bool // synchronize output
}

// ChildBase is the base for child process structures.
// Port of struct childbase from job.h (lines 38-41)
type ChildBase struct {
	CmdName     string  // allocated copy of command run
	Environment []string // environment for commands
	Output      Output  // output for this child
}

// Child represents a running or dead child process.
// Port of struct child from job.h (lines 43-66)
type Child struct {
	ChildBase

	Next     *Child // link in chain
	File     *File  // file being remade

	ShBatchFile string   // script file for shell commands
	CmdLines    []string // expanded command lines
	CmdPtr      string   // ptr into command_lines[command_line]
	CmdLine     uint     // index into command_lines
	Pid         int      // child process ID

	Remote   bool // executing remotely
	Noerror  bool // commands contained '-'
	GoodStdin bool // has good stdin
	Deleted  bool // targets have been deleted
	Recursive bool // recursive command ('+' etc.)
	Jobslot  bool // reserved a job slot
	Dontcare bool // saved dontcare flag

	Cmd       *exec.Cmd
	Exited    chan struct{}
	Processed bool
}
