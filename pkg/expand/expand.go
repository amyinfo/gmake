package expand

import (
	"strings"

	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
)

func VariableExpand(line string) string {
	return VariableExpandString(line, line, len(line))
}

func VariableExpandString(_ string, s string, length int) string {
	var buf strings.Builder

	if length <= 0 {
		return buf.String()
	}
	if length > len(s) {
		length = len(s)
	}

	pos := 0
	for pos < length {
		dollar := strings.IndexByte(s[pos:length], '$')
		if dollar < 0 {
			buf.WriteString(s[pos:length])
			break
		}

		buf.WriteString(s[pos : pos+dollar])
		pos += dollar + 1

		if pos >= length {
			buf.WriteByte('$')
			break
		}

		ch := s[pos]
		switch ch {
		case '$':
			buf.WriteByte('$')
			pos++

		case '(':
			pos++

			depth := 1
			end := pos
			for end < length && depth > 0 {
				switch s[end] {
				case '(':
					depth++
				case ')':
					depth--
				}
				if depth > 0 {
					end++
				}
			}
			if depth > 0 {
				buf.WriteString(s[pos-1 : length])
				pos = length
				break
			}

			ref := s[pos:end]
			pos = end + 1

			if strings.IndexByte(ref, '$') >= 0 {
				ref = VariableExpandString("", ref, len(ref))
			}

			colonIdx := findColon(ref)
			if colonIdx >= 0 {
				substPart := ref[colonIdx+1:]
				eqIdx := strings.IndexByte(substPart, '=')
				if eqIdx >= 0 {
					varName := ref[:colonIdx]
					pattern := substPart[:eqIdx]
					replace := substPart[eqIdx+1:]
					result := substReference(varName, pattern, replace)
					buf.WriteString(result)
					continue
				}
			}

			spaceIdx := strings.IndexAny(ref, " \t")
			if spaceIdx > 0 {
				funcName := ref[:spaceIdx]
				funcArgs := strings.TrimSpace(ref[spaceIdx+1:])
				result := callFunction(funcName, funcArgs)
				buf.WriteString(result)
				continue
			}

			val := referenceVariable(ref)
			buf.WriteString(val)

		case '{':
			pos++

			depth := 1
			end := pos
			for end < length && depth > 0 {
				switch s[end] {
				case '{':
					depth++
				case '}':
					depth--
				}
				if depth > 0 {
					end++
				}
			}
			if depth > 0 {
				buf.WriteString(s[pos-1 : length])
				pos = length
				break
			}

			ref := s[pos:end]
			pos = end + 1

			if strings.IndexByte(ref, '$') >= 0 {
				ref = VariableExpandString("", ref, len(ref))
			}

			colonIdx := findColon(ref)
			if colonIdx >= 0 {
				substPart := ref[colonIdx+1:]
				eqIdx := strings.IndexByte(substPart, '=')
				if eqIdx >= 0 {
					varName := ref[:colonIdx]
					pattern := substPart[:eqIdx]
					replace := substPart[eqIdx+1:]
					result := substReference(varName, pattern, replace)
					buf.WriteString(result)
					continue
				}
			}

			spaceIdx := strings.IndexAny(ref, " \t")
			if spaceIdx > 0 {
				funcName := ref[:spaceIdx]
				funcArgs := strings.TrimSpace(ref[spaceIdx+1:])
				result := callFunction(funcName, funcArgs)
				buf.WriteString(result)
				continue
			}

			val := referenceVariable(ref)
			buf.WriteString(val)

		default:
			val := referenceVariable(string(ch))
			buf.WriteString(val)
			pos++
		}
	}

	return buf.String()
}

func referenceVariable(name string) string {
	if name == "" {
		return ""
	}

	v := LookupVariable(name, len(name))
	if v == nil {
		return ""
	}
	_ = v

	if v.Recursive {
		return VariableExpandString("", v.Value, len(v.Value))
	}
	return v.Value
}

func substReference(varName, pattern, replacement string) string {
	v := LookupVariable(varName, len(varName))
	if v == nil {
		return ""
	}

	var value string
	if v.Recursive {
		value = VariableExpandString("", v.Value, len(v.Value))
	} else {
		value = v.Value
	}

	if value == "" || pattern == "" {
		return value
	}

	return patsubstExpand(value, pattern, replacement)
}

func patsubstExpand(value, pattern, replacement string) string {
	var result strings.Builder
	words := strings.Fields(value)
	for i, w := range words {
		if i > 0 {
			result.WriteByte(' ')
		}
		if pattern == "%" {
			result.WriteString(replacement)
		} else if strings.Contains(pattern, "%") {
			result.WriteString(applyPattern(w, pattern, replacement))
		} else if w == pattern {
			result.WriteString(replacement)
		} else {
			result.WriteString(w)
		}
	}
	return result.String()
}

func applyPattern(word, pattern, replacement string) string {
	pct := strings.IndexByte(pattern, '%')
	rpct := strings.IndexByte(replacement, '%')

	if pct < 0 {
		if word == pattern {
			return replacement
		}
		return word
	}

	prefix := pattern[:pct]
	suffix := pattern[pct+1:]

	if !strings.HasPrefix(word, prefix) || !strings.HasSuffix(word, suffix) {
		return word
	}

	stem := word[len(prefix) : len(word)-len(suffix)]

	if rpct < 0 {
		return replacement
	}
	return replacement[:rpct] + stem + replacement[rpct+1:]
}

func findColon(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

var LookupVariable func(name string, length int) *types.Variable

func init() {
	LookupVariable = variable.LookupVariable
}

func callFunction(name, args string) string {
	if FuncHandler != nil {
		return FuncHandler(name, args)
	}
	return ""
}

var FuncHandler func(name, args string) string

func VariableExpandForFile(line string, file *types.File) string {
	if file == nil {
		return VariableExpand(line)
	}
	saveVSL := variable.CurrentVariableSetList
	if file.Variables != nil {
		variable.CurrentVariableSetList = file.Variables
	}
	result := VariableExpand(line)
	variable.CurrentVariableSetList = saveVSL
	return result
}

func AllocatedVariableExpandForFile(line string, file *types.File) string {
	return VariableExpandForFile(line, file)
}

func ExpandArgument(str, end string) string {
	if str == end {
		return ""
	}
	if end == "" {
		return AllocatedVariableExpand(str)
	}
	return AllocatedVariableExpand(str)
}

func AllocatedVariableExpand(line string) string {
	return VariableExpand(line)
}

func RecursivelyExpandForFile(v *types.Variable, file *types.File) string {
	if v == nil {
		return ""
	}

	if v.Recursive {
		if file == nil {
			return VariableExpandString("", v.Value, len(v.Value))
		}
		return VariableExpandForFile(v.Value, file)
	}
	return v.Value
}

func InitializeVariableOutput() string {
	return ""
}

func VariableBufferOutput(ptr, s string, length int) string {
	return ptr + s[:length]
}

func InstallVariableBuffer(bufp *string, lenp *int) {
}

func RestoreVariableBuffer(buf string, length int) {
}

func HandleFunction(op *string, sp *string) int {
	return 0
}

func RecursivelyExpand(v *types.Variable) string {
	if v == nil {
		return ""
	}
	if v.Recursive {
		return VariableExpandString("", v.Value, len(v.Value))
	}
	return v.Value
}
