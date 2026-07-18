package variable

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyra/make/pkg/config"
	"github.com/kyra/make/pkg/strcache"
	"github.com/kyra/make/pkg/types"
)

var (
	CurrentVariableSetList *types.VariableSetList
	DefaultGoalVar         *types.Variable
	ShellVar               types.Variable
	VariableBuffer         string
)

const EXP_COUNT_MAX = (1 << 15) - 1

func InitHashGlobalVariableSet() {
	// Initialize global variable set
	vs := &types.VariableSet{
		Variables: make(map[string]*types.Variable),
	}
	CurrentVariableSetList = &types.VariableSetList{
		Set: vs,
	}
}

func InitHashFunctionTable() {
	// Initialize function table
	// Functions are registered in function.go
}

func CreateNewVariableSet() *types.VariableSetList {
	vs := &types.VariableSet{
		Variables: make(map[string]*types.Variable),
	}
	return &types.VariableSetList{
		Set: vs,
	}
}

func FreeVariableSet(vsl *types.VariableSetList) {
	// Let GC handle it
}

func PushNewVariableScope() *types.VariableSetList {
	vs := CreateNewVariableSet()
	vs.Next = CurrentVariableSetList
	CurrentVariableSetList = vs
	return vs
}

func PopVariableScope() {
	if CurrentVariableSetList != nil {
		CurrentVariableSetList = CurrentVariableSetList.Next
	}
}

func LookupVariable(name string, length int) *types.Variable {
	if length > len(name) {
		length = len(name)
	}
	if length == 0 {
		return nil
	}
	name = name[:length]

	for vsl := CurrentVariableSetList; vsl != nil; vsl = vsl.Next {
		if vsl.Set != nil {
			if v, ok := vsl.Set.Variables[name]; ok {
				return v
			}
		}
		if !vsl.NextIsParent {
			break
		}
	}
	return nil
}

func LookupVariableInSet(name string, length int, set *types.VariableSet) *types.Variable {
	if length > len(name) {
		length = len(name)
	}
	if set == nil {
		return nil
	}
	return set.Variables[name[:length]]
}

func DefineVariableInSet(name string, length int, value string,
	origin types.VariableOrigin, recursive bool,
	set *types.VariableSet, flocp *types.Floc) *types.Variable {

	if length > len(name) {
		length = len(name)
	}
	n := name[:length]

	if set == nil {
		// Global variable set
		set = CurrentVariableSetList.Set
	}

	v := &types.Variable{
		Name:       strcache.Add(n),
		Value:      value,
		Length:     uint(length),
		Recursive:  recursive,
		Origin:     origin,
		Exportable: true,
	}

	if flocp != nil {
		v.Fileinfo = *flocp
	}

	set.Variables[n] = v
	return v
}

func DoVariableDefinition(flocp *types.Floc, name, value string,
	origin types.VariableOrigin, flavor types.VariableFlavor,
	targetVar int) *types.Variable {

	_ = targetVar
	v := LookupVariable(name, len(name))

	switch flavor {
	case types.FlavorRecursive:
		if v != nil && v.Origin == types.OriginAutomatic {
			return v
		}
		return DefineVariableInSet(name, len(name), value, origin, true, nil, flocp)

	case types.FlavorSimple:
		if v != nil && v.Origin == types.OriginAutomatic {
			return v
		}
		// Simple expansion: expand value now
		expanded := ExpandVariable(value)
		return DefineVariableInSet(name, len(name), expanded, origin, false, nil, flocp)

	case types.FlavorConditional:
		if v != nil {
			return v
		}
		return DefineVariableInSet(name, len(name), value, origin, true, nil, flocp)

	case types.FlavorAppend:
		if v != nil && v.Origin == types.OriginAutomatic {
			return v
		}
		if v == nil {
			return DefineVariableInSet(name, len(name), value, origin, true, nil, flocp)
		}
		// Append
		if v.Recursive {
			v.Value += value
		} else {
			// For simple variables, expand and append
			expanded := ExpandVariable(value)
			v.Value += expanded
		}
		return v

	case types.FlavorShell:
		// Execute shell command and capture output
		output, err := ShellEscape(value)
		if err == nil {
			value = output
		}
		if v != nil && v.Origin == types.OriginAutomatic {
			return v
		}
		return DefineVariableInSet(name, len(name), value, origin, false, nil, flocp)

	case types.FlavorExpand:
		expanded := ExpandVariable(value)
		return DefineVariableInSet(name, len(name), expanded, origin, true, nil, flocp)

	case types.FlavorAppendValue:
		if v != nil && v.Origin == types.OriginAutomatic {
			return v
		}
		var newval string
		if v != nil {
			newval = v.Value + value
		} else {
			newval = value
		}
		return DefineVariableInSet(name, len(name), newval, origin, v != nil && v.Recursive, nil, flocp)

	default:
		return DefineVariableInSet(name, len(name), value, origin, false, nil, flocp)
	}
}

func ParseVariableDefinition(line string, v *types.Variable) string {
	// Parse a variable definition line
	// Returns the value portion, or empty string if not a variable definition

	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return ""
	}

	// Check for various assignment operators
	operators := []struct {
		op     string
		flavor types.VariableFlavor
	}{
		{":=", types.FlavorSimple},
		{"::=", types.FlavorSimple},
		{":::", types.FlavorExpand},
		{"+=", types.FlavorAppend},
		{"?=", types.FlavorConditional},
		{"!=", types.FlavorShell},
	}

	// Check for = first (recursive)
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return ""
	}

	name := strings.TrimRight(line[:eq], " \t")
	flavor := types.FlavorRecursive

	// Check for multi-char operators
	for _, op := range operators {
		if strings.HasSuffix(name, op.op) {
			name = strings.TrimRight(name[:len(name)-len(op.op)], " \t")
			flavor = op.flavor
			break
		}
	}

	if name == "" {
		return ""
	}

	value := strings.TrimLeft(line[eq+1:], " \t")
	v.Name = name
	v.Value = value
	v.Flavor = flavor
	v.Length = uint(len(name))

	return value
}

func AssignVariableDefinition(v *types.Variable, line string) *types.Variable {
	ParseVariableDefinition(line, v)
	return v
}

func TryVariableDefinition(flocp *types.Floc, line string,
	origin types.VariableOrigin, targetVar int) *types.Variable {

	var v types.Variable
	value := ParseVariableDefinition(line, &v)
	if value == "" {
		return nil
	}
	return DoVariableDefinition(flocp, v.Name, v.Value, origin, v.Flavor, targetVar)
}

func TargetEnvironment(file *types.File, recursive bool) []string {
	// Build environment for a target
	// This corresponds to target_environment() in variable.c
	var env []string

	// Collect variables from the file and its parents
	seen := make(map[string]bool)
	collectVars := func(vsl *types.VariableSetList) {
		for vsl != nil {
			if vsl.Set != nil {
				for name, v := range vsl.Set.Variables {
					if !seen[name] && v.Exportable {
						env = append(env, name+"="+v.Value)
						seen[name] = true
					}
				}
			}
			vsl = vsl.Next
		}
	}

	if file != nil {
		collectVars(file.Variables)
	}
	collectVars(CurrentVariableSetList)

	return env
}

func CreatePatternVar(target, suffix string) *types.PatternVar {
	return &types.PatternVar{
		Target: target,
		Suffix: suffix,
		Len:    uintptr(len(target)),
	}
}

func DefineAutomaticVariables() {
	// Define automatic variables $@, $<, $^, $+, $*, $?, $%, etc.
	// These are defined with origin o_automatic

	automatic := []string{
		"@", "<", "^", "+", "*", "?", "%",
		"D", "F", // directory/ file parts
		"*D", "*F",
		"<D", "<F",
		"@D", "@F",
		"^D", "^F",
		"+D", "+F",
		"?D", "?F",
	}

	for _, name := range automatic {
		DefineVariableInSet(name, len(name), "",
			types.OriginAutomatic, false, nil, nil)
	}
}

func InitializeFileVariables(file *types.File, reading int) {
	file.Variables = &types.VariableSetList{
		Set: &types.VariableSet{
			Variables: make(map[string]*types.Variable),
		},
	}
	_ = reading
}

func PrintFileVariables(file *types.File) {
	if file == nil || file.Variables == nil {
		return
	}
	for name, v := range file.Variables.Set.Variables {
		_, _ = fmt.Fprintf(os.Stdout, "# %s:\n", name)
		_, _ = fmt.Fprintf(os.Stdout, "%s = %s\n", name, v.Value)
	}
}

func PrintTargetVariables(file *types.File) {
	PrintFileVariables(file)
}

func MergeVariableSetLists(toList **types.VariableSetList, fromList *types.VariableSetList) {
	if fromList == nil || toList == nil {
		return
	}

	// Merge fromList into toList
	merged := &types.VariableSetList{
		Set:          fromList.Set,
		Next:         *toList,
		NextIsParent: true,
	}
	*toList = merged
}

func UndefineVariableInSet(name string, length int,
	origin types.VariableOrigin, set *types.VariableSet) {

	if length > len(name) {
		length = len(name)
	}
	n := name[:length]

	if set == nil {
		set = CurrentVariableSetList.Set
	}
	delete(set.Variables, n)
}

func WarnUndefined(name string, length int) {
	if config.WarnUndefinedVariables {
		fmt.Fprintf(os.Stderr, "warning: undefined variable '%.*s'\n", length, name)
	}
}

// ShellEscape expands a shell command and returns output
func ShellEscape(cmd string) (string, error) {
	// TODO: Implement actual shell execution
	return "", nil
}

// ExpandVariable performs variable expansion
// This is a simplified wrapper - the real expansion is in the expand package
func ExpandVariable(line string) string {
	// Call into the expand package
	return Expand(line)
}

// These will be set by the expand package to avoid circular imports
var Expand func(string) string

// DefineVariable is a convenience wrapper for define_variable macro
func DefineVariable(name string, length int, value string,
	origin types.VariableOrigin, recursive bool) *types.Variable {

	return DefineVariableInSet(name, length, value, origin, recursive, nil, nil)
}

// DefineVariableCname defines a variable with a constant name
func DefineVariableCname(name, value string,
	origin types.VariableOrigin, recursive bool) *types.Variable {

	return DefineVariableInSet(name, len(name), value, origin, recursive, nil, nil)
}

// UndefineVariableGlobal removes a variable from global set
func UndefineVariableGlobal(name string, length int, origin types.VariableOrigin) {
	UndefineVariableInSet(name, length, origin, nil)
}
