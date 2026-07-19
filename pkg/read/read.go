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

package read

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/misc"
	"github.com/amyinfo/gmake/pkg/rule"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
)

// ReadingFile is the makefile currently being read.
var ReadingFile *types.Floc

// readFiles is the chain of files read by readAllMakefiles.
var readFiles []*types.Goaldep

// includeDirectories is the list of directories to search for include files.
var includeDirectories []string

// maxIncludeLength tracks the max length of include directory paths.
var maxInclLen int

const (
	wordBogus int = iota
	wordEol
	wordStatic
	wordVariable
	wordColon
	wordDcolon
	wordSemicolon
	wordVarassign
	wordAmpcolon
	wordAmpdcolon
)

// ReadAllMakefiles reads all the makefiles and returns targets to rebuild.
func ReadAllMakefiles(makefiles []string) []*types.Goaldep {
	numMakefiles := 0

	// Create MAKEFILE_LIST variable
	variable.DefineVariableCname("MAKEFILE_LIST", "", types.OriginFile, false)

	if config.IsDb(config.DbBasic) {
		fmt.Fprintf(os.Stderr, "%s: Reading makefiles...\n", config.Program)
	}

	// Handle MAKEFILES variable
	if value := lookupVariable("MAKEFILES"); value != "" {
		save := config.WarnUndefinedVariables
		config.WarnUndefinedVariables = false
		expanded := variable.Expand(value)
		config.WarnUndefinedVariables = save

		p := expanded
		for {
			token := misc.FindNextTokenStr(&p)
			if token == "" {
				break
			}
			evalMakefile(token, types.RMIncluded|types.RMDontcare|types.RMNoDefaultGoal)
		}
	}

	// Read makefiles specified with -f switches
	if len(makefiles) > 0 {
		for _, mf := range makefiles {
			d := evalMakefile(mf, types.RMNoFlag)
			if d != nil {
				_ = d
				numMakefiles++
			}
		}
	}

	// If no -f switches, try default names
	if numMakefiles == 0 {
		defaultMakefiles := []string{"GNUmakefile", "makefile", "Makefile"}
		found := ""
		for _, mf := range defaultMakefiles {
			if fileExists(mf) {
				found = mf
				break
			}
		}
		if found != "" {
			evalMakefile(found, types.RMNoFlag)
		} else {
			for _, mf := range defaultMakefiles {
				d := &types.Goaldep{
					Dep: types.Dep{
						File: file.EnterFile(strcache.Add(mf)),
						Flags: byte(types.RMDontcare),
					},
				}
				readFiles = append(readFiles, d)
			}
		}
	}

	return readFiles
}

func lookupVariable(name string) string {
	v := variable.LookupVariable(name, len(name))
	if v == nil {
		return ""
	}
	if v.Recursive {
		return variable.Expand(v.Value)
	}
	return v.Value
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// evalMakefile evaluates one makefile.
func evalMakefile(filename string, flags int) *types.Goaldep {
	deps := &types.Goaldep{
		Floc: types.Floc{Filenm: filename, Lineno: 1},
	}
	readFiles = append(readFiles, deps)

	if config.IsDb(config.DbVerbose) {
		fmt.Fprintf(os.Stderr, "Reading makefile '%s'", filename)
		if flags&types.RMNoDefaultGoal != 0 {
			fmt.Fprintf(os.Stderr, " (no default goal)")
		}
		if flags&types.RMIncluded != 0 {
			fmt.Fprintf(os.Stderr, " (search path)")
		}
		if flags&types.RMDontcare != 0 {
			fmt.Fprintf(os.Stderr, " (don't care)")
		}
		fmt.Fprintf(os.Stderr, "...\n")
	}

	// Handle ~ expansion
	if flags&types.RMNoTilde == 0 && strings.HasPrefix(filename, "~") {
		expanded := tildeExpand(filename)
		if expanded != "" {
			filename = expanded
		}
	}

	// Try to open the file
	fp, err := os.Open(filename)
	deps.Error = 0
	if err != nil {
		deps.Error = 1
		// Search include directories if applicable
		if flags&types.RMIncluded != 0 && !filepath.IsAbs(filename) {
			for _, dir := range includeDirectories {
				fp2, err2 := os.Open(filepath.Join(dir, filename))
				if err2 == nil {
					fp = fp2
					filename = filepath.Join(dir, filename)
					deps.Error = 0
					break
				}
			}
		}
	}

	filename = strcache.Add(filename)
	fileEnt := file.LookupFile(filename)
	if fileEnt == nil {
		fileEnt = file.EnterFile(filename)
	}
	filename = fileEnt.Name
	if fileEnt != nil {
		deps.File = fileEnt
	}
	deps.Flags = byte(flags)

	if fp == nil {
		if deps.File != nil {
			deps.File.LastMtime = uint64(config.NonexistentMtime)
		}
		return deps
	}
	defer fp.Close()

	deps.Error = 0
	if deps.File != nil && deps.File.LastMtime == uint64(config.NonexistentMtime) {
		deps.File.LastMtime = 0
	}

	// Add to MAKEFILE_LIST
	variable.DoVariableDefinition(&deps.Floc, "MAKEFILE_LIST", filename,
		types.OriginFile, types.FlavorAppendValue, 0)

	// Evaluate the makefile
	curfile := ReadingFile
	ReadingFile = &deps.Floc

	eval(fp, flags&types.RMNoDefaultGoal == 0)

	ReadingFile = curfile

	return deps
}

// evalBuffer evaluates a buffer of makefile content (for $(eval)).
func EvalBuffer(buffer string, flocp *types.Floc) {
	var fi types.Floc
	if flocp != nil {
		fi = *flocp
	} else if ReadingFile != nil {
		fi = *ReadingFile
	} else {
		fi = types.Floc{Filenm: "", Lineno: 1}
	}

	curfile := ReadingFile
	ReadingFile = &fi

	// Evaluate the buffer content
	reader := strings.NewReader(buffer)
	eval(reader, true)

	ReadingFile = curfile
}

// eval is the main parsing loop.
func eval(r io.Reader, setDefault bool) {
	scanner := bufio.NewScanner(r)
	// Increase buffer for long lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var lineNum uint64
	var commands []string
	var inDefine bool
	var ignoring bool
	var inIgnoredDefine bool
	var ruleTargets []string
	var ruleDeps string
	var ruleTwoColon bool
	var commandStart uint64
	var targetStart uint64
	var pattern string
	var haveRule bool

	// For conditionals
	var condStack []condState

	isSpecialTarget := func(name string) bool {
		switch name {
		case ".PHONY", ".SUFFIXES", ".DEFAULT", ".PRECIOUS", ".INTERMEDIATE",
			".SECONDARY", ".SECONDEXPANSION", ".DELETE_ON_ERROR", ".IGNORE",
			".LOW_RESOLUTION_TIME", ".NOTPARALLEL", ".ONESHELL", ".POSIX",
			".EXPORT_ALL_VARIABLES", ".SILENT", ".EXEC", ".WAIT":
			return true
		}
		return false
	}

	flushRule := func() {
		if haveRule && len(ruleTargets) > 0 {
			recordFiles(ruleTargets, ruleDeps, pattern, ruleTwoColon, commands, commandStart, &types.Floc{Filenm: "", Lineno: targetStart})
			if config.DefaultGoalName == "" {
				for _, t := range ruleTargets {
					if !isSpecialTarget(t) {
						config.DefaultGoalName = t
						break
					}
				}
			}
		}
		ruleTargets = nil
		ruleDeps = ""
		commands = nil
		haveRule = false
		pattern = ""
		ruleTwoColon = false
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Handle line continuations (backslash at end of line)
		for strings.HasSuffix(line, "\\") && scanner.Scan() {
			line = line[:len(line)-1] + scanner.Text()
			lineNum++
		}

		// Strip trailing whitespace
		line = strings.TrimRight(line, " \t\r")

		// Skip UTF-8 BOM on first line
		if lineNum == 1 && len(line) >= 3 && line[0] == 0xEF && line[1] == 0xBB && line[2] == 0xBF {
			line = line[3:]
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for recipe line (starts with tab)
		if len(line) > 0 && line[0] == '\t' {
			if haveRule {
				if !ignoring {
					line = line[1:] // Strip tab
					commands = append(commands, line)
				}
			}
			continue
		}

		// Not a recipe line. Flush any pending rule.
		if haveRule {
			flushRule()
		}

		// Strip comments (but not inside variable references)
		line = stripComment(line)

		// Collapse whitespace
		collapsed := strings.TrimSpace(line)
		if collapsed == "" {
			continue
		}

		// Check for conditional directives
		if isConditional(collapsed) {
			handleConditional(collapsed, &condStack, &ignoring, &inIgnoredDefine)
			continue
		}

		// Check for endef
		if collapsed == "endef" {
			inDefine = false
			inIgnoredDefine = false
			continue
		}

		if inDefine || inIgnoredDefine {
			continue
		}

		// Check for override BEFORE parseAssignment so "override X = val"
		// is not mistaken for an assignment to variable "override X".
		if strings.HasPrefix(collapsed, "override ") {
			rest := strings.TrimSpace(collapsed[9:])
			if name, value, flavor := parseAssignment(rest); name != "" && !ignoring {
				variable.DoVariableDefinition(ReadingFile, name, value,
					types.OriginOverride, flavor, 0)
			}
			continue
		}

		// Check for variable assignment
		if name, value, flavor := parseAssignment(collapsed); name != "" {
			if !ignoring {
				variable.DoVariableDefinition(ReadingFile, name, value,
					types.OriginFile, flavor, 0)
			}
			continue
		}

		// Check for include directive
		if strings.HasPrefix(collapsed, "include ") || strings.HasPrefix(collapsed, "-include ") || strings.HasPrefix(collapsed, "sinclude ") {
			dontcare := strings.HasPrefix(collapsed, "-") || strings.HasPrefix(collapsed, "s")
			incFile := strings.TrimSpace(collapsed[strings.IndexByte(collapsed, ' '):])
			if !ignoring {
				// Expand the include filename
				expanded := variable.Expand(incFile)
				for _, f := range strings.Fields(expanded) {
					flags := types.RMIncluded | types.RMNoTilde
					if dontcare {
						flags |= types.RMDontcare
					}
					evalMakefile(f, flags)
				}
			}
			continue
		}

		// Check for export/unexport
		if strings.HasPrefix(collapsed, "export ") || collapsed == "export" {
			if !ignoring {
				if collapsed == "export" {
					config.ExportAllVariables = true
				} else {
					exportName := strings.TrimSpace(collapsed[7:])
					v := variable.LookupVariable(exportName, len(exportName))
					if v == nil {
						v = variable.DefineVariableInSet(exportName, len(exportName), "",
							types.OriginFile, false, nil, ReadingFile)
					}
					v.Export = types.ExportYes
				}
			}
			continue
		}
		if strings.HasPrefix(collapsed, "unexport ") || collapsed == "unexport" {
			if !ignoring {
				if collapsed == "unexport" {
					config.ExportAllVariables = false
				} else {
					exportName := strings.TrimSpace(collapsed[9:])
					v := variable.LookupVariable(exportName, len(exportName))
					if v == nil {
						v = variable.DefineVariableInSet(exportName, len(exportName), "",
							types.OriginFile, false, nil, ReadingFile)
					}
					v.Export = types.ExportNo
				}
			}
			continue
		}

		// Check for vpath
		if strings.HasPrefix(collapsed, "vpath ") {
			// Simplified vpath handling - just skip
			continue
		}

		// Check for define
		if strings.HasPrefix(collapsed, "define ") {
			if ignoring {
				inIgnoredDefine = true
			} else {
				inDefine = true
			}
			continue
		}

		// Check for private
		if strings.HasPrefix(collapsed, "private ") {
			continue
		}

		// Check pattern rules with %
		if strings.Contains(collapsed, "%") {
			// Check for :: or : separator
			separator := strings.Index(collapsed, "::")
			doubleColon := false
			sep := strings.Index(collapsed, ":")
			if separator >= 0 && (sep < 0 || separator < sep) {
				doubleColon = true
				sep = separator
			}
			if sep > 0 && sep < len(collapsed) {
				ruleTwoColon = doubleColon
				targets := strings.Fields(collapsed[:sep])
				ruleTargets = targets
				ruleDeps = ""
				if sep+1 < len(collapsed) {
					if doubleColon {
						ruleDeps = strings.TrimSpace(collapsed[sep+2:])
					} else {
						ruleDeps = strings.TrimSpace(collapsed[sep+1:])
					}
				}
				// Extract pattern suffix (text after %) from first target
				if len(targets) > 0 {
					if pct := strings.IndexByte(targets[0], '%'); pct >= 0 {
						pattern = targets[0][pct+1:]
					}
				}
				if !ignoring {
					haveRule = true
					targetStart = lineNum
				}
				continue
			}
		}

		// Check for rule with : or :: separator
		separator := findRuleSep(collapsed)
		if separator >= 0 {
			doubleColon := false
			if separator+1 < len(collapsed) && collapsed[separator+1] == ':' {
				doubleColon = true
			}
			targets := strings.Fields(collapsed[:separator])

			ruleTwoColon = doubleColon
			ruleTargets = targets
			var depsAndRecipe string
			if doubleColon {
				depsAndRecipe = strings.TrimSpace(collapsed[separator+2:])
			} else {
				depsAndRecipe = strings.TrimSpace(collapsed[separator+1:])
			}

			// Check for inline recipe (after ;)
			if semi := strings.IndexByte(depsAndRecipe, ';'); semi >= 0 {
				ruleDeps = strings.TrimSpace(depsAndRecipe[:semi])
				recipe := strings.TrimSpace(depsAndRecipe[semi+1:])
				if recipe != "" && !ignoring {
					commands = append(commands, recipe)
				}
			} else {
				ruleDeps = depsAndRecipe
			}

			if !ignoring {
				haveRule = true
				targetStart = lineNum
			}
			continue
		}

		// Not a directive or rule — expand for side effects (e.g. $(info ...), $(warning ...))
		if !ignoring && strings.Contains(collapsed, "$") {
			variable.Expand(collapsed)
		}
	}

	// Flush final rule
	if haveRule {
		flushRule()
	}
}

type condState struct {
	active   bool
	seenElse bool
}

func isConditional(line string) bool {
	words := []string{"ifdef ", "ifndef ", "ifeq ", "ifneq ", "else", "endif"}
	for _, w := range words {
		if strings.HasPrefix(line, w) {
			return true
		}
	}
	return false
}

func handleConditional(line string, stack *[]condState, ignoring *bool, inIgnoredDefine *bool) {
	if strings.HasPrefix(line, "ifdef ") || strings.HasPrefix(line, "ifndef ") {
		varName := strings.TrimSpace(line[6:])
		isDefined := variable.LookupVariable(varName, len(varName)) != nil
		negative := strings.HasPrefix(line, "ifndef")
		active := (isDefined != negative)

		var parentActive bool
		if len(*stack) > 0 {
			parentActive = (*stack)[len(*stack)-1].active
		} else {
			parentActive = true
		}

		*stack = append(*stack, condState{active: active && parentActive, seenElse: false})
		*ignoring = !(*stack)[len(*stack)-1].active
		*inIgnoredDefine = false
		return
	}

	if strings.HasPrefix(line, "ifeq ") || strings.HasPrefix(line, "ifneq ") {
		rest := strings.TrimSpace(line[5:])
		negative := strings.HasPrefix(line, "ifneq")

		var arg1, arg2 string
		if strings.HasPrefix(rest, "\"") {
			// "arg1", "arg2" format
			parts := splitQuoteArgs(rest)
			if len(parts) >= 2 {
				arg1 = parts[0]
				arg2 = parts[1]
			}
		} else if strings.HasPrefix(rest, "(") {
			// (arg1, arg2) format
			rest = strings.TrimPrefix(rest, "(")
			end := strings.Index(rest, ")")
			if end >= 0 {
				inner := rest[:end]
				comma := strings.IndexByte(inner, ',')
				if comma >= 0 {
					arg1 = strings.TrimSpace(inner[:comma])
					arg2 = strings.TrimSpace(inner[comma+1:])
				}
			}
		}

		// Expand arguments
		arg1 = variable.Expand(arg1)
		arg2 = variable.Expand(arg2)
		match := arg1 == arg2
		active := (match != negative)

		var parentActive bool
		if len(*stack) > 0 {
			parentActive = (*stack)[len(*stack)-1].active
		} else {
			parentActive = true
		}

		*stack = append(*stack, condState{active: active && parentActive, seenElse: false})
		*ignoring = !(*stack)[len(*stack)-1].active
		*inIgnoredDefine = false
		return
	}

	if strings.HasPrefix(line, "else") {
		if len(*stack) > 0 {
			top := &(*stack)[len(*stack)-1]
			if top.seenElse {
				// Already saw else - error
				fmt.Fprintf(os.Stderr, "%s: *** extraneous `else'.\n", config.Program)
				return
			}
			top.seenElse = true
			top.active = !top.active

			var anyActive bool
			for _, s := range *stack {
				if s.active {
					anyActive = true
					break
				}
			}
			*ignoring = !anyActive
			*inIgnoredDefine = false
		}
		return
	}

	if strings.HasPrefix(line, "endif") {
		if len(*stack) > 0 {
			*stack = (*stack)[:len(*stack)-1]
		}
		var anyActive bool
		for _, s := range *stack {
			if s.active {
				anyActive = true
				break
			}
		}
		*ignoring = !anyActive
		return
	}
}

func splitQuoteArgs(s string) []string {
	var args []string
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			end := strings.IndexByte(s[i+1:], '"')
			if end >= 0 {
				args = append(args, s[i+1:i+1+end])
				i = i + end + 1
			}
		} else if s[i] == ',' {
			// Separator
		}
	}
	return args
}

// parseAssignment parses a variable assignment line.
// Returns name, value, flavor.
func parseAssignment(line string) (string, string, types.VariableFlavor) {
	depth := 0
	eqIdx := -1
	opLen := 0
	flavor := types.FlavorRecursive
	ruleColon := -1 // track first unquoted ':' that's not :: or :=

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '(' || ch == '{' {
			depth++
		} else if ch == ')' || ch == '}' {
			depth--
		}
		if depth != 0 {
			continue
		}
		// If we see a rule separator ':' before any '=', this is not an assignment
		if ch == ':' && ruleColon < 0 && eqIdx < 0 {
			if i+1 < len(line) && line[i+1] == '=' {
				// := is an assignment operator, not a rule separator
			} else if i+1 < len(line) && line[i+1] == ':' {
				// :: is a double-colon rule separator (unless it's ::=)
				if i+2 >= len(line) || line[i+2] != '=' {
					ruleColon = i
				}
			} else {
				ruleColon = i
			}
		}
		if i+2 < len(line) && line[i:i+3] == "::=" {
			eqIdx = i + 2
			opLen = 3
			flavor = types.FlavorSimple
			break
		}
		if i+1 < len(line) {
			op := line[i : i+2]
			switch op {
			case ":=", "+=", "?=", "!=":
				eqIdx = i + 1
				opLen = 2
				switch op {
				case ":=":
					flavor = types.FlavorSimple
				case "+=":
					flavor = types.FlavorAppend
				case "?=":
					flavor = types.FlavorConditional
				case "!=":
					flavor = types.FlavorShell
				}
				break
			}
		}
		if eqIdx >= 0 {
			break
		}
		if ch == '=' {
			// If we saw a rule colon before this =, this is a recipe, not an assignment
			if ruleColon >= 0 {
				return "", "", types.FlavorBogus
			}
			eqIdx = i
			opLen = 1
			flavor = types.FlavorRecursive
			break
		}
	}

	if eqIdx < 0 {
		return "", "", types.FlavorBogus
	}

	name := strings.TrimRight(line[:eqIdx-opLen+1], " \t")
	if name == "" {
		return "", "", types.FlavorBogus
	}

	value := strings.TrimLeft(line[eqIdx+1:], " \t")
	return name, value, flavor
}

// stripComment removes comments from a line, respecting quoted strings.
func stripComment(line string) string {
	depth := 0
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '(' || ch == '{' {
			depth++
		} else if ch == ')' || ch == '}' {
			depth--
		} else if ch == '#' && depth == 0 {
			// Check if this is a literal \#
			if i > 0 && line[i-1] == '\\' {
				continue
			}
			return line[:i]
		}
	}
	return line
}

// findRuleSep finds the rule separator (:) in a line, respecting variable refs.
func findRuleSep(line string) int {
	depth := 0
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '(' || ch == '{' {
			depth++
		} else if ch == ')' || ch == '}' {
			depth--
		} else if ch == ':' && depth == 0 {
			// Check for :: (double colon)
			// But not := (variable assignment)
			if i+1 < len(line) && line[i+1] == '=' {
				continue // Skip :=
			}
			return i
		}
	}
	return -1
}

// recordFiles records target rules parsed from the Makefile.
func recordFiles(targets []string, depstr, pattern string, twoColon bool, commands []string, cmdStart uint64, flocp *types.Floc) {
	if len(targets) == 0 {
		return
	}

	// Parse dependencies
	var deps *types.Dep
	if depstr != "" {
		deps = file.SplitPrereqs(depstr)
	}

	// If we have a pattern, it's an implicit rule
	if pattern != "" {
		// Compute suffix after % for each target
		n := uint16(len(targets))
		targetPercents := make([]string, n)
		for i, t := range targets {
			if pct := strings.IndexByte(t, '%'); pct >= 0 {
				targetPercents[i] = t[pct+1:]
			}
		}
		// Build commands struct if we have commands
		var cmds *types.Commands
		if len(commands) > 0 {
			cmds = &types.Commands{
				Fileinfo:     *flocp,
				Commands:     strings.Join(commands, "\n"),
				RecipePrefix: byte(config.CmdPrefix),
			}
		}
		rule.CreatePatternRule(targets, targetPercents, n, false, deps, cmds, false)
		return
	}

	// Enter targets into file database
	var lastTarget *types.File
	for _, tname := range targets {
		tname = strcache.Add(tname)
		targetFile := file.EnterFile(tname)
		targetFile.IsTarget = true

		if twoColon {
			if targetFile.DoubleColon == nil {
				targetFile.DoubleColon = targetFile
			}
		}

		if lastTarget == nil {
			lastTarget = targetFile
		}

		if len(targets) > 1 && tname != targets[0] {
		}

		// Set dependencies (first target only to avoid duplication)
		if deps != nil && targetFile.Deps == nil {
			targetFile.Deps = deps
		}

		// Set commands on the first target
		if commands != nil && targetFile.Cmds == nil {
			targetFile.Cmds = &types.Commands{
				Fileinfo:     *flocp,
				Commands:     strings.Join(commands, "\n"),
				RecipePrefix: byte(config.CmdPrefix),
			}
			// Set recipe prefix if we saved it
			targetFile.CmdTarget = true
		}
	}

	// Enter each dependency as a file too
	for d := deps; d != nil; d = d.Next {
		if d.Name_ != "" {
			depFile := file.EnterFile(strcache.Add(d.Name_))
			d.File = depFile
		}
	}

}

// constructIncludePath sets up the include directory search path.
func ConstructIncludePath(dirnames string) {
	includeDirectories = append(includeDirectories, "/usr/gnu/include", "/usr/local/include", "/usr/include")
	if dirnames != "" {
		for _, dir := range strings.Fields(dirnames) {
			includeDirectories = append(includeDirectories, dir)
			if len(dir) > maxInclLen {
				maxInclLen = len(dir)
			}
		}
	}
}

// tildeExpand expands a leading ~ in a filename.
func tildeExpand(name string) string {
	if !strings.HasPrefix(name, "~") {
		return name
	}
	if len(name) == 1 || name[1] == '/' {
		home, err := os.UserHomeDir()
		if err != nil {
			return name
		}
		return home + name[1:]
	}
	// ~user expansion - simplified
	return name
}
