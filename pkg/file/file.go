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

// Package file implements target file management for GNU Make.
// Port of src/file.c
package file

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
)

// SnappedDeps remembers whether snap_deps has been invoked.
var SnappedDeps bool

// allSecondary tracks whether .SECONDARY with no prereqs was given.
var allSecondary bool

// noIntermediates tracks whether .NOTINTERMEDIATE with no prereqs was given.
var noIntermediates bool

// files is the hash table of all file records.
var files = make(map[string]*types.File)

// fileDirs tracks files that are directories (for double-colon entries).
var fileDirs []*types.File

// InitHashFiles initializes the file hash table.
// Port of init_hash_files() from file.c line 1272-1276
func InitHashFiles() {
	files = make(map[string]*types.File)
	fileDirs = nil
	SnappedDeps = false
	allSecondary = false
	noIntermediates = false
}

// LookupFile looks up a file record by name.
// Port of lookup_file() from file.c lines 74-141
func LookupFile(name string) *types.File {
	if name == "" {
		return nil
	}

	// Strip leading "./" sequences (Unix path normalization)
	for len(name) > 2 && name[0] == '.' && os.IsPathSeparator(name[1]) {
		name = name[2:]
		for len(name) > 0 && os.IsPathSeparator(name[0]) {
			name = name[1:]
		}
	}
	if name == "" {
		name = "./"
	}

	// Case-insensitive target comparison would go here on VMS/Windows
	// For Unix, just use the name as-is

	f := files[name]
	return f
}

// EnterFile looks up a file record for name, creating one if it doesn't exist.
// Port of enter_file() from file.c lines 148-204
func EnterFile(name string) *types.File {
	if name == "" {
		return nil
	}

	name = strcache.Add(name)

	if f, ok := files[name]; ok && f.DoubleColon == nil {
		f.Builtin = false
		return f
	}

	newf := &types.File{
		Name:         name,
		Hname:        name,
		UpdateStatus: types.UpdateNone,
		LastMtime:    uint64(config.UnknownMtime),
		MtimeBeforeUpdate: uint64(config.UnknownMtime),
		Last:         new(types.File),
	}

	if f, ok := files[name]; ok && f.DoubleColon != nil {
		newf.DoubleColon = f
		if f.Last != nil {
			f.Last.Prev = newf
		}
		f.Last = newf
	} else {
		newf.Last = newf
		files[name] = newf
	}

	return newf
}

// RehashFile renames a file in the hash table.
// Port of rehash_file() from file.c lines 210-341
func RehashFile(fromFile *types.File, toHname string) {
	if fromFile == nil {
		return
	}
	fromFile.Builtin = false

	// If it's already that name, we're done
	if fromFile.Hname == toHname {
		return
	}

	// Follow the rename chain to the end
	origFrom := fromFile
	for fromFile.Renamed != nil {
		fromFile = fromFile.Renamed
	}

	// Remove the from file from the hash
	oldName := fromFile.Hname
	delete(files, oldName)

	// Find where the newly renamed file will go
	toFile, exists := files[toHname]

	// Change the hash name
	fromFile.Hname = toHname
	for f := fromFile.DoubleColon; f != nil; f = f.Prev {
		f.Hname = toHname
	}

	// If the new name doesn't exist yet, just insert
	if !exists {
		files[toHname] = fromFile
		return
	}

	// toFile already exists - merge fromFile into it
	mergeFile(fromFile, toFile)
	fromFile.Renamed = toFile
	origFrom.Renamed = toFile
}

func mergeFile(fromFile, toFile *types.File) {
	// Merge commands
	if fromFile.Cmds != nil {
		if toFile.Cmds == nil {
			toFile.Cmds = fromFile.Cmds
		} else if fromFile.Cmds != toFile.Cmds {
			// Warn about conflicting recipes
			l := len(fromFile.Name)
			if toFile.Cmds.Fileinfo.Filenm != "" {
				fmt.Fprintf(os.Stderr, "%s: Recipe was specified for file '%s' at %s:%d,\n",
					config.Program, fromFile.Name, toFile.Cmds.Fileinfo.Filenm, toFile.Cmds.Fileinfo.Lineno)
			} else {
				fmt.Fprintf(os.Stderr, "%s: Recipe for file '%s' was found by implicit rule search,\n",
					config.Program, fromFile.Name)
			}
			fmt.Fprintf(os.Stderr, "%s: but '%s' is now considered the same file as '%s'.\n",
				config.Program, fromFile.Name, toFile.Hname)
			fmt.Fprintf(os.Stderr, "%s: Recipe for '%s' will be ignored in favor of the one for '%s'.\n",
				config.Program, fromFile.Name, toFile.Hname)
			_ = l
		}
	}

	// Merge dependencies
	if toFile.Deps == nil {
		toFile.Deps = fromFile.Deps
	} else {
		deps := toFile.Deps
		for deps.Next != nil {
			deps = deps.Next
		}
		deps.Next = fromFile.Deps
	}

	// Merge variable sets (simplified)
	if fromFile.Variables != nil && toFile.Variables == nil {
		toFile.Variables = fromFile.Variables
	}

	// Merge flags
	if toFile.DoubleColon != nil && fromFile.IsTarget && fromFile.DoubleColon == nil {
		fmt.Fprintf(os.Stderr, "%s: *** can't rename single-colon '%s' to double-colon '%s'.\n",
			config.Program, fromFile.Name, toFile.Hname)
		os.Exit(config.MakeFailure)
	}
	if toFile.DoubleColon == nil && fromFile.DoubleColon != nil {
		if toFile.IsTarget {
			fmt.Fprintf(os.Stderr, "%s: *** can't rename double-colon '%s' to single-colon '%s'.\n",
				config.Program, fromFile.Name, toFile.Hname)
			os.Exit(config.MakeFailure)
		} else {
			toFile.DoubleColon = fromFile.DoubleColon
		}
	}

	if fromFile.LastMtime > toFile.LastMtime {
		toFile.LastMtime = fromFile.LastMtime
	}
	toFile.MtimeBeforeUpdate = fromFile.MtimeBeforeUpdate

	// Merge boolean flags
	if fromFile.Precious {
		toFile.Precious = true
	}
	if fromFile.Loaded {
		toFile.Loaded = true
	}
	if fromFile.TriedImplicit {
		toFile.TriedImplicit = true
	}
	if fromFile.Updating {
		toFile.Updating = true
	}
	if fromFile.Updated {
		toFile.Updated = true
	}
	if fromFile.IsTarget {
		toFile.IsTarget = true
	}
	if fromFile.CmdTarget {
		toFile.CmdTarget = true
	}
	if fromFile.Phony {
		toFile.Phony = true
	}
	if fromFile.IsExplicit {
		toFile.IsExplicit = true
	}
	if fromFile.Secondary {
		toFile.Secondary = true
	}
	if fromFile.Notintermediate {
		toFile.Notintermediate = true
	}
	if fromFile.IgnoreVpath {
		toFile.IgnoreVpath = true
	}
	if fromFile.Snapped {
		toFile.Snapped = true
	}
	toFile.Builtin = false
}

// RenameFile renames a file and updates its name field.
// Port of rename_file() from file.c lines 347-356
func RenameFile(fromFile *types.File, toHname string) {
	RehashFile(fromFile, toHname)
	for f := fromFile; f != nil; f = f.Prev {
		f.Name = f.Hname
	}
}

// RemoveIntermediates removes all nonprecious intermediate files.
// Port of remove_intermediates() from file.c lines 363-440
func RemoveIntermediates(sig bool) {
	// If there's no way we will ever remove anything, punt early
	if config.QuestionFlag || config.TouchFlag || allSecondary {
		return
	}
	if sig && config.JustPrintFlag {
		return
	}

	var doneany bool
	for _, f := range files {
		if f == nil {
			continue
		}
		// Is this file eligible for automatic deletion?
		if f.Intermediate && (f.Dontcare || !f.Precious) &&
			!f.Secondary && !f.Notintermediate && !f.CmdTarget {
			if f.UpdateStatus == types.UpdateNone {
				continue
			}
			var status error
			if !config.JustPrintFlag {
				status = os.Remove(f.Name)
				if status != nil && os.IsNotExist(status) {
					continue
				}
			}
			if !f.Dontcare {
				if sig {
					fmt.Fprintf(os.Stderr, "%s: *** Deleting intermediate file '%s'\n",
						config.Program, f.Name)
				} else {
					if !doneany {
						if config.IsDb(config.DbBasic) {
							fmt.Fprintf(os.Stderr, "Removing intermediate files...\n")
						}
					}
					if !config.RunSilent {
						if !doneany {
							fmt.Fprint(os.Stdout, "rm ")
							doneany = true
						} else {
							fmt.Fprint(os.Stdout, " ")
						}
						fmt.Fprint(os.Stdout, f.Name)
					}
				}
				if status != nil {
					fmt.Fprintf(os.Stderr, "\nunlink: %s: %v\n", f.Name, status)
					doneany = false
				}
			}
		}
	}

	if doneany && !sig {
		fmt.Fprintln(os.Stdout)
	}
}

// SplitPrereqs splits a string containing prerequisites into a dep list.
// Port of split_prereqs() from file.c lines 445-474
func SplitPrereqs(p string) *types.Dep {
	var newDeps *types.Dep
	var lastDep *types.Dep

	// Split on pipe for order-only prerequisites
	pipeIdx := -1
	depth := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '|' && depth == 0 {
			pipeIdx = i
			break
		}
	}

	var normalPart, ooPart string
	if pipeIdx >= 0 {
		normalPart = strings.TrimSpace(p[:pipeIdx])
		ooPart = strings.TrimSpace(p[pipeIdx+1:])
	} else {
		normalPart = strings.TrimSpace(p)
	}

	// Parse normal prerequisites
	if normalPart != "" {
		newDeps = parseFileSeq(normalPart, false)
	}

	// Parse order-only prerequisites
	if ooPart != "" {
		ood := parseFileSeq(ooPart, false)
		for d := ood; d != nil; d = d.Next {
			d.IgnoreMtime = true
		}
		if newDeps == nil {
			newDeps = ood
		} else {
			lastDep = newDeps
			for lastDep.Next != nil {
				lastDep = lastDep.Next
			}
			lastDep.Next = ood
		}
	}

	return newDeps
}

// parseFileSeq parses a space-separated list of filenames.
// Simplified version of parse_file_seq from C.
func parseFileSeq(s string, waitOk bool) *types.Dep {
	var head, tail *types.Dep
	words := strings.Fields(s)
	for _, w := range words {
		d := &types.Dep{Name_: strcache.Add(w)}
		if waitOk && w == ".WAIT" {
			d.WaitHere = true
			continue
		}
		if head == nil {
			head = d
			tail = d
		} else {
			tail.Next = d
			tail = d
		}
	}
	return head
}

// EnterPrereqs enters a list of prerequisites into the file database.
// Port of enter_prereqs() from file.c lines 478-557
func EnterPrereqs(deps *types.Dep, stem string) *types.Dep {
	if deps == nil {
		return nil
	}

	// If we have a stem, expand % patterns
	if stem != "" {
		var prev *types.Dep
		for dp := deps; dp != nil; {
			percent := strings.IndexByte(dp.Name_, '%')
			if percent >= 0 {
				var expanded string
				if stem == "" {
					// Empty stem: remove % 
					expanded = dp.Name_[:percent] + dp.Name_[percent+1:]
				} else {
					// Replace % with stem
					expanded = dp.Name_[:percent] + stem + dp.Name_[percent+1:]
				}
				if expanded == "" {
					// Remove empty dep
					if prev == nil {
						deps = dp.Next
					} else {
						prev.Next = dp.Next
					}
					dp = dp.Next
					continue
				}
				dp.Name_ = strcache.Add(expanded)
			}
			dp.Stem = stem
			dp.Staticpattern = true
			prev = dp
			dp = dp.Next
		}
	}

	// Enter them as files, unless they need 2nd expansion
	for d1 := deps; d1 != nil; d1 = d1.Next {
		if d1.Need2ndExpansion {
			continue
		}
		d1.File = LookupFile(d1.Name_)
		if d1.File == nil {
			d1.File = EnterFile(d1.Name_)
		}
		d1.Staticpattern = false
		d1.Name_ = ""
		if stem == "" {
			d1.File.IsExplicit = true
		}
	}

	return deps
}

// ExpandDeps expands and parses each dependency line.
// Port of expand_deps() from file.c lines 562-690
func ExpandDeps(f *types.File) {
	if f.Snapped {
		return
	}
	f.Snapped = true

	var initialized bool
	var changedDep bool

	var dp **types.Dep
	var d *types.Dep

	// Walk through dependencies, expanding any that need 2nd expansion
	dp = &f.Deps
	d = f.Deps
	for d != nil {
		if d.Name_ == "" || !d.Need2ndExpansion {
			dp = &d.Next
			d = d.Next
			continue
		}

		// If it's from a static pattern rule, convert % to $*
		if d.Staticpattern {
			nperc := strings.Count(d.Name_, "%")
			if nperc > 0 {
				// Replace % with $*
				name := d.Name_
				var result strings.Builder
				for _, ch := range name {
					if ch == '%' {
						result.WriteString("$*")
					} else {
						result.WriteRune(ch)
					}
				}
				d.Name_ = result.String()
			}
		}

		// Perform second expansion
		if !initialized {
			// initialize_file_variables(f, 0)
			f.Variables = &types.VariableSetList{
				Set: &types.VariableSet{
					Variables: make(map[string]*types.Variable),
				},
			}
			initialized = true
		}

		// Set file variables for expansion
		fstem := d.Stem
		if fstem == "" {
			fstem = f.Stem
		}
		_ = fstem

		// Expand the dependency name (simplified - no variable expansion yet)
		p := d.Name_

		// Parse expanded prerequisites
		newDeps := SplitPrereqs(p)

		// If no prereqs here, throw this one out
		if newDeps == nil {
			*dp = d.Next
			changedDep = true
			d = *dp
			continue
		}

		// Add newly parsed prerequisites
		next := d.Next
		changedDep = true
		*dp = newDeps
		for nd := newDeps; nd != nil; nd = nd.Next {
			nd.File = LookupFile(nd.Name_)
			if nd.File == nil {
				nd.File = EnterFile(nd.Name_)
			}
			nd.Name_ = ""
			nd.Stem = fstem
			if fstem == "" {
				nd.File.IsExplicit = true
			}
			dp = &nd.Next
		}
		*dp = next
		d = *dp
	}

	_ = changedDep
	// Note: shuffle support would go here but requires the shuffle package
}

// ExpandExtraPrereqs expands .EXTRA_PREREQS variable.
// Port of expand_extra_prereqs() from file.c lines 694-710
func ExpandExtraPrereqs(extra *types.Variable) *types.Dep {
	if extra == nil {
		return nil
	}
	prereqs := SplitPrereqs(extra.Value)
	for d := prereqs; d != nil; d = d.Next {
		d.File = LookupFile(d.Name_)
		if d.File == nil {
			d.File = EnterFile(d.Name_)
		}
		d.Name_ = ""
		d.IgnoreAutomaticVars = true
	}
	return prereqs
}

// SnapDeps marks files depended on by special targets.
// Port of snap_deps() from file.c lines 770-910
func SnapDeps() {
	// Remember that we've done this
	SnappedDeps = true

	// .PRECIOUS
	for f := LookupFile(".PRECIOUS"); f != nil; f = f.Prev {
		for d := f.Deps; d != nil; d = d.Next {
			for f2 := d.File; f2 != nil; f2 = f2.Prev {
				f2.Precious = true
			}
		}
	}

	// .LOW_RESOLUTION_TIME
	for f := LookupFile(".LOW_RESOLUTION_TIME"); f != nil; f = f.Prev {
		for d := f.Deps; d != nil; d = d.Next {
			for f2 := d.File; f2 != nil; f2 = f2.Prev {
				f2.LowResolutionTime = true
			}
		}
	}

	// .PHONY
	for f := LookupFile(".PHONY"); f != nil; f = f.Prev {
		for d := f.Deps; d != nil; d = d.Next {
			for f2 := d.File; f2 != nil; f2 = f2.Prev {
				f2.Phony = true
				f2.IsTarget = true
				f2.LastMtime = uint64(config.NonexistentMtime)
				f2.MtimeBeforeUpdate = uint64(config.NonexistentMtime)
			}
		}
	}

	// .NOTINTERMEDIATE
	for f := LookupFile(".NOTINTERMEDIATE"); f != nil; f = f.Prev {
		if f.Deps != nil {
			for d := f.Deps; d != nil; d = d.Next {
				for f2 := d.File; f2 != nil; f2 = f2.Prev {
					f2.Notintermediate = true
				}
			}
		} else {
			noIntermediates = true
		}
	}

	// .INTERMEDIATE
	for f := LookupFile(".INTERMEDIATE"); f != nil; f = f.Prev {
		for d := f.Deps; d != nil; d = d.Next {
			for f2 := d.File; f2 != nil; f2 = f2.Prev {
				if f2.Notintermediate {
					fmt.Fprintf(os.Stderr, "%s: *** %s cannot be both .NOTINTERMEDIATE and .INTERMEDIATE.\n",
						config.Program, f2.Name)
					os.Exit(config.MakeFailure)
				}
				f2.Intermediate = true
			}
		}
	}

	// .SECONDARY
	for f := LookupFile(".SECONDARY"); f != nil; f = f.Prev {
		if f.Deps != nil {
			for d := f.Deps; d != nil; d = d.Next {
				for f2 := d.File; f2 != nil; f2 = f2.Prev {
					if f2.Notintermediate {
						fmt.Fprintf(os.Stderr, "%s: *** %s cannot be both .NOTINTERMEDIATE and .SECONDARY.\n",
							config.Program, f2.Name)
						os.Exit(config.MakeFailure)
					}
					f2.Intermediate = true
					f2.Secondary = true
				}
			}
		} else {
			allSecondary = true
		}
	}

	if noIntermediates && allSecondary {
		fmt.Fprintf(os.Stderr, "%s: *** .NOTINTERMEDIATE and .SECONDARY are mutually exclusive.\n",
			config.Program)
		os.Exit(config.MakeFailure)
	}

	// .EXPORT_ALL_VARIABLES
	if f := LookupFile(".EXPORT_ALL_VARIABLES"); f != nil && f.IsTarget {
		config.ExportAllVariables = true
	}

	// .IGNORE
	if f := LookupFile(".IGNORE"); f != nil && f.IsTarget {
		if f.Deps == nil {
			config.IgnoreErrorsFlag = true
		} else {
			for d := f.Deps; d != nil; d = d.Next {
				for f2 := d.File; f2 != nil; f2 = f2.Prev {
					f2.CommandFlags |= types.CmdNoerror
				}
			}
		}
	}

	// .SILENT
	if f := LookupFile(".SILENT"); f != nil && f.IsTarget {
		if f.Deps == nil {
			config.RunSilent = true
		} else {
			for d := f.Deps; d != nil; d = d.Next {
				for f2 := d.File; f2 != nil; f2 = f2.Prev {
					f2.CommandFlags |= types.CmdSilent
				}
			}
		}
	}

	// .NOTPARALLEL
	if f := LookupFile(".NOTPARALLEL"); f != nil && f.IsTarget {
		if f.Deps == nil {
			config.NotParallel = true
		} else {
			for d := f.Deps; d != nil; d = d.Next {
				for f2 := d.File; f2 != nil; f2 = f2.Prev {
					if f2.Deps != nil {
						for d2 := f2.Deps.Next; d2 != nil; d2 = d2.Next {
							d2.WaitHere = true
						}
					}
				}
			}
		}
	}

	// Per-file snap operations for .EXTRA_PREREQS
	extra := lookupVariable(".EXTRA_PREREQS")
	prereqs := ExpandExtraPrereqs(extra)
	_ = prereqs

	// Apply snap per file
	for _, f := range files {
		snapFile(f)
	}
}

func lookupVariable(name string) *types.Variable {
	return nil // Will be connected to variable package
}

func snapFile(f *types.File) {
	// If we're not doing second expansion then reset updating
	if !config.SecondExpansion {
		f.Updating = false
	}

	// .SECONDARY with no deps marks all targets as intermediate
	if allSecondary && !f.Notintermediate {
		f.Intermediate = true
	}

	// .NOTINTERMEDIATE with no deps marks all targets
	if noIntermediates && !f.Intermediate && !f.Secondary {
		f.Notintermediate = true
	}

	// EXTRA_PREREQS handling - simplified
}

// SetCommandState sets the command state for a file and its also_make targets.
// Port of set_command_state() from file.c lines 916-926
func SetCommandState(file *types.File, state types.CmdState) {
	file.CommandState = state
	for d := file.AlsoMake; d != nil; d = d.Next {
		if d.File != nil && state > d.File.CommandState {
			d.File.CommandState = state
		}
	}
}

// FileTimestampCons converts an external file timestamp to internal form.
// Port of file_timestamp_cons() from file.c lines 930-950
func FileTimestampCons(fname string, stamp time.Time) uint64 {
	// Simplified: use UnixNano
	ts := uint64(stamp.UnixNano())
	return ts
}

// FileTimestampNow returns the current time as a file timestamp.
// Port of file_timestamp_now() from file.c lines 954-1000
func FileTimestampNow() uint64 {
	return uint64(time.Now().UnixNano())
}

// FileTimestampSprintf formats a file timestamp.
// Port of file_timestamp_sprintf() from file.c lines 1004-1034
func FileTimestampSprintf(ts uint64) string {
	sec := int64(ts / 1000000000)
	nsec := int64(ts % 1000000000)
	t := time.Unix(sec, nsec)
	return t.Format("2006-01-02 15:04:05")
}

// PrintPrereqs prints the dependency list.
// Port of print_prereqs() from file.c lines 1039-1060
func PrintPrereqs(deps *types.Dep) {
	var ood *types.Dep

	// Print normal dependencies; note any order-only deps
	for d := deps; d != nil; d = d.Next {
		if !d.IgnoreMtime {
			if d.WaitHere {
				fmt.Printf(" .WAIT %s", d.Name())
			} else {
				fmt.Printf(" %s", d.Name())
			}
		} else if ood == nil {
			ood = d
		}
	}

	// Print order-only deps
	if ood != nil {
		fmt.Printf(" | %s%s", waitStr(ood), depName(ood))
		for ood = ood.Next; ood != nil; ood = ood.Next {
			if ood.IgnoreMtime {
				fmt.Printf(" %s%s", waitStr(ood), depName(ood))
			}
		}
	}
	fmt.Println()
}

func waitStr(d *types.Dep) string {
	if d.WaitHere {
		return ".WAIT "
	}
	return ""
}

func depName(d *types.Dep) string {
	if d.Name_ != "" {
		return d.Name_
	}
	if d.File != nil {
		return d.File.Name
	}
	return ""
}

// PrintFileDataBase prints the data base of files.
// Port of print_file_data_base() from file.c lines 1179-1188
func PrintFileDataBase() {
	fmt.Println("\n# Files")

	// Collect and sort keys for deterministic output
	var keys []string
	for name := range files {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		f := files[name]
		if f != nil {
			printFile(f)
		}
	}

	fmt.Printf("\n# files hash-table stats:\n# %d entries\n", len(files))
}

func printFile(f *types.File) {
	if config.NoBuiltinRulesFlag && f.Builtin {
		return
	}

	fmt.Println()
	fmt.Printf("%s:%s", f.Name, colonStr(f.DoubleColon))
	PrintPrereqs(f.Deps)

	if f.Precious {
		fmt.Println("#  Precious file (prerequisite of .PRECIOUS).")
	}
	if f.Phony {
		fmt.Println("#  Phony target (prerequisite of .PHONY).")
	}
	if f.CmdTarget {
		fmt.Println("#  Command line target.")
	}
	if f.Dontcare {
		fmt.Println("#  A default, MAKEFILES, or -include/sinclude makefile.")
	}
	if f.Builtin {
		fmt.Println("#  Builtin rule")
	}
	if f.TriedImplicit {
		fmt.Println("#  Implicit rule search has been done.")
	} else {
		fmt.Println("#  Implicit rule search has not been done.")
	}
	if f.Stem != "" {
		fmt.Printf("#  Implicit/static pattern stem: '%s'\n", f.Stem)
	}
	if f.Intermediate {
		fmt.Println("#  File is an intermediate prerequisite.")
	}
	if f.Notintermediate {
		fmt.Println("#  File is a prerequisite of .NOTINTERMEDIATE.")
	}
	if f.Secondary {
		fmt.Println("#  File is secondary (prerequisite of .SECONDARY).")
	}
	if f.AlsoMake != nil {
		fmt.Print("#  Also makes:")
		for d := f.AlsoMake; d != nil; d = d.Next {
			fmt.Printf(" %s", d.Name())
		}
		fmt.Println()
	}

	switch {
	case f.LastMtime == 0:
		fmt.Println("#  Modification time never checked.")
	case f.LastMtime == 1:
		fmt.Println("#  File does not exist.")
	default:
		fmt.Printf("#  Last modified %s\n", FileTimestampSprintf(f.LastMtime))
	}

	if f.Updated {
		fmt.Println("#  File has been updated.")
	} else {
		fmt.Println("#  File has not been updated.")
	}

	switch f.CommandState {
	case types.CmdNotStarted, types.CmdFinished:
		switch f.UpdateStatus {
		case types.UpdateSuccess:
			fmt.Println("#  Successfully updated.")
		case types.UpdateFailed:
			fmt.Println("#  Failed to be updated.")
		}
	}
}

func colonStr(dc *types.File) string {
	if dc != nil {
		return ":"
	}
	return ""
}

// BuildTargetList builds a space-separated list of all target names.
// Port of build_target_list() from file.c lines 1227-1270
var lastTargetCount int

func BuildTargetList(value string) string {
	count := len(files)
	if count != lastTargetCount {
		var names []string
		for name, f := range files {
			if f != nil && f.IsTarget {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		value = strings.Join(names, " ")
		lastTargetCount = count
	}
	return value
}

// GetFiles returns the file map for iteration.
func GetFiles() map[string]*types.File {
	return files
}
