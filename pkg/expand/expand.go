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

package expand

import (
	"strings"

	"github.com/kyra/make/pkg/misc"
	"github.com/kyra/make/pkg/types"
)

var (
	VariableBuffer string
	ReadingFile    *types.Floc
	ExpandingVar   **types.Floc
)

func VariableBufferOutput(ptr string, s string, length int) string {
	ptr += s[:length]
	return ptr
}

func InitializeVariableOutput() string {
	return ""
}

func InstallVariableBuffer(bufp *string, lenp *int) {
	VariableBuffer = *bufp
}

func RestoreVariableBuffer(buf string, len int) {
	VariableBuffer = buf
}

func VariableExpandString(line string, s string, length int) string {
	_ = line
	// Expand variables in string s
	var result strings.Builder
	i := 0
	for i < length {
		if s[i] == '$' {
			if i+1 < length {
				if s[i+1] == '$' {
					result.WriteByte('$')
					i += 2
					continue
				}
				// Variable reference
				ref, newI := parseVariableRef(s, i+1, length)
				expanded := lookupAndExpand(ref)
				result.WriteString(expanded)
				i = newI
				continue
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

func VariableExpand(line string) string {
	return VariableExpandString(line, line, len(line))
}

func VariableExpandForFile(line string, file *types.File) string {
	_ = file
	return VariableExpandString(line, line, len(line))
}

func AllocatedVariableExpandForFile(line string, file *types.File) string {
	return VariableExpandForFile(line, file)
}

func ExpandArgument(str string, end string) string {
	return VariableExpandString("", str, len(str))
}

func RecursivelyExpandForFile(v *types.Variable, file *types.File) string {
	_ = file
	if v == nil {
		return ""
	}
	if v.Recursive {
		return VariableExpandString("", v.Value, len(v.Value))
	}
	return v.Value
}

func HandleFunction(op *string, sp *string) int {
	return 0
}

// parseVariableRef parses a variable reference starting at s[i]
// Returns the variable name/key and the new index
func parseVariableRef(s string, start, length int) (string, int) {
	if start >= length {
		return "", start
	}

	// Simple variable: $X or ${X} or $(X)
	if s[start] == '(' || s[start] == '{' {
		closeChar := byte(')')
		if s[start] == '{' {
			closeChar = '}'
		}
		end := strings.IndexByte(s[start+1:], closeChar)
		if end < 0 {
			return s[start+1:], length
		}
		return s[start+1 : start+1+end], start + 2 + end
	}

	// Single char variable: $@, $<, etc.
	if start < length {
		return s[start : start+1], start + 1
	}

	return "", start
}

// lookupAndExpand looks up a variable and returns its expanded value
func lookupAndExpand(ref string) string {
	if ref == "" {
		return ""
	}

	// Handle function calls $(func ...)
	if idx := strings.IndexByte(ref, ' '); idx > 0 {
		funcName := ref[:idx]
		args := strings.TrimSpace(ref[idx+1:])
		return callFunction(funcName, args)
	}

	// Simple variable lookup
	v := lookupVariable(ref, len(ref))
	if v == nil {
		return ""
	}

	if v.Recursive {
		return VariableExpandString("", v.Value, len(v.Value))
	}
	return v.Value
}

// lookupVariable looks up a variable in the current scope
func lookupVariable(name string, length int) *types.Variable {
	// This is a temporary stub - will be connected to the variable package
	return nil
}

// callFunction calls a built-in make function
func callFunction(name, args string) string {
	// This is a stub - functions are implemented in the function package
	return ""
}

// Helper functions

func FindNextToken(s *string, length *int) string {
	for *s != "" && misc.Isspace((*s)[0]) {
		*s = (*s)[1:]
	}
	if *s == "" {
		return ""
	}
	end := 0
	for end < len(*s) && !misc.Isspace((*s)[end]) {
		end++
	}
	result := (*s)[:end]
	if length != nil {
		*length = end
	}
	if end < len(*s) {
		*s = (*s)[end+1:]
	} else {
		*s = ""
	}
	return result
}

// Register the Expand function with the variable package
func init() {
	// This will be set when variable package imports this
}

// SetLookupFn allows variable package to register its lookup
var LookupVariableFn func(name string, length int) *types.Variable

func init() {
	// Later connected by the main package
}
