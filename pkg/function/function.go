package function

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/expand"
	"github.com/amyinfo/gmake/pkg/read"
	"github.com/amyinfo/gmake/pkg/types"
)

var funcTable = map[string]struct {
	minArgs int
	maxArgs int
	fn      func(args []string) string
}{
	"shell":    {0, 1, fnShell},
	"subst":    {3, 3, fnSubst},
	"patsubst": {3, 3, fnPatSubst},
	"strip":    {0, 1, fnStrip},
	"sort":     {0, 1, fnSort},
	"firstword": {0, 1, fnFirstword},
	"lastword":  {0, 1, fnLastword},
	"words":    {0, 1, fnWords},
	"word":     {2, 2, fnWord},
	"wordlist": {3, 3, fnWordlist},
	"join":     {2, 2, fnJoin},
	"addprefix": {2, 2, fnAddPrefix},
	"addsuffix": {2, 2, fnAddSuffix},
	"dir":      {0, 1, fnDir},
	"notdir":   {0, 1, fnNotdir},
	"suffix":   {0, 1, fnSuffix},
	"basename": {0, 1, fnBasename},
	"filter":   {2, 2, fnFilter},
	"filter-out": {2, 2, fnFilterOut},
	"findstring": {2, 2, fnFindstring},
	"if":       {2, 3, fnIf},
	"or":       {1, 0, fnOr},
	"and":      {1, 0, fnAnd},
	"foreach":  {3, 3, fnForeach},
	"call":     {1, 0, fnCall},
	"value":    {0, 1, fnValue},
	"eval":     {0, 1, fnEval},
	"origin":   {0, 1, fnOrigin},
	"flavor":   {0, 1, fnFlavor},
	"info":     {0, 1, fnInfo},
	"warning":  {0, 1, fnWarning},
	"error":    {0, 1, fnError},
	"wildcard": {0, 1, fnWildcard},
	"realpath": {0, 1, fnRealpath},
	"abspath":  {0, 1, fnAbspath},
	"file":     {1, 2, fnFile},
}

func CallFunction(name, args string) string {
	entry, ok := funcTable[name]
	if !ok {
		return ""
	}

	var arglist []string
	if args != "" {
		if entry.maxArgs == 0 {
			arglist = []string{args}
		} else {
			arglist = splitArgs(args, entry.maxArgs)
		}
	}

	if len(arglist) < entry.minArgs {
		return ""
	}

	return entry.fn(arglist)
}

func splitArgs(s string, max int) []string {
	var args []string
	depth := 0
	start := 0

	for i := 0; i < len(s) && (max <= 0 || len(args) < max-1); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(s[start:]))
	return args
}

func fnShell(args []string) string {
	if len(args) == 0 || args[0] == "" {
		return ""
	}
	cmd := exec.Command("sh", "-c", args[0])
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	result := strings.TrimRight(string(output), "\n")
	result = strings.ReplaceAll(result, "\n", " ")
	return result
}

func fnSubst(args []string) string {
	if len(args) < 3 {
		return ""
	}
	from := args[0]
	to := args[1]
	text := args[2]
	if from == "" {
		return text
	}
	return strings.ReplaceAll(text, from, to)
}

func fnPatSubst(args []string) string {
	if len(args) < 3 {
		return ""
	}
	pattern := args[0]
	replacement := args[1]
	text := args[2]

	words := strings.Fields(text)
	var result []string
	for _, w := range words {
		result = append(result, applyPattern(w, pattern, replacement))
	}
	return strings.Join(result, " ")
}

func applyPattern(word, pattern, replacement string) string {
	if !strings.Contains(pattern, "%") {
		if word == pattern {
			return replacement
		}
		return word
	}

	pct := strings.IndexByte(pattern, '%')
	rpct := strings.IndexByte(replacement, '%')

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

func fnStrip(args []string) string {
	if len(args) == 0 {
		return ""
	}
	fields := strings.Fields(args[0])
	return strings.Join(fields, " ")
}

func fnSort(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	sort.Strings(words)
	return strings.Join(words, " ")
}

func fnFirstword(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	if len(words) == 0 {
		return ""
	}
	return words[0]
}

func fnLastword(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}

func fnWords(args []string) string {
	if len(args) == 0 {
		return "0"
	}
	words := strings.Fields(args[0])
	return itoa(len(words))
}

func fnWord(args []string) string {
	if len(args) < 2 {
		return ""
	}
	n := atoi(args[0])
	words := strings.Fields(args[1])
	if n < 1 || n > len(words) {
		return ""
	}
	return words[n-1]
}

func fnWordlist(args []string) string {
	if len(args) < 3 {
		return ""
	}
	start := atoi(args[0])
	end := atoi(args[1])
	words := strings.Fields(args[2])
	if start < 1 {
		start = 1
	}
	if end > len(words) {
		end = len(words)
	}
	if start > end || start > len(words) {
		return ""
	}
	return strings.Join(words[start-1:end], " ")
}

func fnJoin(args []string) string {
	if len(args) < 2 {
		return ""
	}
	list1 := strings.Fields(args[0])
	list2 := strings.Fields(args[1])
	var result []string
	max := len(list1)
	if len(list2) > max {
		max = len(list2)
	}
	for i := 0; i < max; i++ {
		var w1, w2 string
		if i < len(list1) {
			w1 = list1[i]
		}
		if i < len(list2) {
			w2 = list2[i]
		}
		result = append(result, w1+w2)
	}
	return strings.Join(result, " ")
}

func fnAddPrefix(args []string) string {
	if len(args) < 2 {
		return ""
	}
	prefix := args[0]
	words := strings.Fields(args[1])
	for i, w := range words {
		words[i] = prefix + w
	}
	return strings.Join(words, " ")
}

func fnAddSuffix(args []string) string {
	if len(args) < 2 {
		return ""
	}
	suffix := args[0]
	words := strings.Fields(args[1])
	for i, w := range words {
		words[i] = w + suffix
	}
	return strings.Join(words, " ")
}

func fnDir(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	for i, w := range words {
		dir := filepath.Dir(w)
		if dir == "." {
			dir = "./"
		} else {
			dir += "/"
		}
		words[i] = dir
	}
	return strings.Join(words, " ")
}

func fnNotdir(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	for i, w := range words {
		words[i] = filepath.Base(w)
	}
	return strings.Join(words, " ")
}

func fnSuffix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	for i, w := range words {
		ext := filepath.Ext(w)
		words[i] = ext
	}
	return strings.Join(words, " ")
}

func fnBasename(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	for i, w := range words {
		ext := filepath.Ext(w)
		words[i] = w[:len(w)-len(ext)]
	}
	return strings.Join(words, " ")
}

func fnFilter(args []string) string {
	if len(args) < 2 {
		return ""
	}
	patterns := strings.Fields(args[0])
	text := strings.Fields(args[1])
	var result []string
	for _, w := range text {
		if matchAny(w, patterns) {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}

func fnFilterOut(args []string) string {
	if len(args) < 2 {
		return ""
	}
	patterns := strings.Fields(args[0])
	text := strings.Fields(args[1])
	var result []string
	for _, w := range text {
		if !matchAny(w, patterns) {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}

func matchAny(word string, patterns []string) bool {
	for _, p := range patterns {
		if matchPattern(word, p) {
			return true
		}
	}
	return false
}

func matchPattern(word, pattern string) bool {
	if !strings.Contains(pattern, "%") {
		return word == pattern
	}
	pct := strings.IndexByte(pattern, '%')
	prefix := pattern[:pct]
	suffix := pattern[pct+1:]
	if !strings.HasPrefix(word, prefix) || !strings.HasSuffix(word, suffix) {
		return false
	}
	return true
}

func fnFindstring(args []string) string {
	if len(args) < 2 {
		return ""
	}
	if strings.Contains(args[1], args[0]) {
		return args[0]
	}
	return ""
}

func fnIf(args []string) string {
	if len(args) < 2 {
		return ""
	}
	condition := strings.TrimSpace(args[0])
	thenPart := args[1]
	elsePart := ""
	if len(args) > 2 {
		elsePart = args[2]
	}
	if condition != "" {
		return expand.VariableExpand(thenPart)
	}
	if elsePart != "" {
		return expand.VariableExpand(elsePart)
	}
	return ""
}

func fnOr(args []string) string {
	for _, a := range args {
		expanded := expand.VariableExpand(a)
		if strings.TrimSpace(expanded) != "" {
			return expanded
		}
	}
	return ""
}

func fnAnd(args []string) string {
	result := ""
	for _, a := range args {
		result = expand.VariableExpand(a)
		if strings.TrimSpace(result) == "" {
			return ""
		}
	}
	return result
}

func fnForeach(args []string) string {
	if len(args) < 3 {
		return ""
	}
	varName := strings.TrimSpace(args[0])
	list := strings.Fields(args[1])
	text := args[2]

	var result []string
	_ = varName
	for _, w := range list {
		expanded := strings.ReplaceAll(text, "$(varName)", w)
		expanded = strings.ReplaceAll(expanded, "${varName}", w)
		expanded = expand.VariableExpand(expanded)
		result = append(result, expanded)
	}
	return strings.Join(result, " ")
}

func fnCall(args []string) string {
	if len(args) == 0 {
		return ""
	}
	funcName := args[0]
	callArgs := args[1:]

	v := expand.LookupVariable(funcName, len(funcName))
	if v == nil {
		return ""
	}

	result := v.Value
	for i, a := range callArgs {
		param := "$(" + itoa(i+1) + ")"
		result = strings.ReplaceAll(result, param, a)
	}
	result = strings.ReplaceAll(result, "$(0)", funcName)

	return expand.VariableExpand(result)
}

func fnValue(args []string) string {
	if len(args) == 0 {
		return ""
	}
	v := expand.LookupVariable(args[0], len(args[0]))
	if v == nil {
		return ""
	}
	return v.Value
}

func fnEval(args []string) string {
	if len(args) == 0 {
		return ""
	}
	// TODO: actually evaluate the string as a makefile
	return ""
}

func fnOrigin(args []string) string {
	if len(args) == 0 {
		return ""
	}
	v := expand.LookupVariable(args[0], len(args[0]))
	if v == nil {
		return "undefined"
	}
	switch v.Origin {
	case types.OriginDefault:
		return "default"
	case types.OriginEnv:
		return "environment"
	case types.OriginFile:
		return "file"
	case types.OriginEnvOverride:
		return "environment override"
	case types.OriginCommand:
		return "command line"
	case types.OriginOverride:
		return "override"
	case types.OriginAutomatic:
		return "automatic"
	default:
		return "undefined"
	}
}

func fnFlavor(args []string) string {
	if len(args) == 0 {
		return ""
	}
	v := expand.LookupVariable(args[0], len(args[0]))
	if v == nil {
		return "undefined"
	}
	if v.Recursive {
		return "recursive"
	}
	return "simple"
}

func fnInfo(args []string) string {
	if len(args) == 0 {
		return ""
	}
	os.Stdout.WriteString(args[0])
	os.Stdout.WriteString("\n")
	return ""
}

func fnWarning(args []string) string {
	if len(args) == 0 {
		return ""
	}
	prefix := ""
	if read.ReadingFile != nil {
		prefix = fmt.Sprintf("%s:%d: ", read.ReadingFile.Filenm, read.ReadingFile.Lineno)
	}
	os.Stderr.WriteString(prefix + args[0] + "\n")
	return ""
}

func fnError(args []string) string {
	if len(args) == 0 {
		return ""
	}
	prefix := ""
	if read.ReadingFile != nil {
		prefix = fmt.Sprintf("%s:%d: ", read.ReadingFile.Filenm, read.ReadingFile.Lineno)
	}
	os.Stderr.WriteString(prefix + args[0] + "\n")
	os.Stderr.WriteString(config.Program + ": *** Stop.\n")
	os.Exit(2)
	return ""
}

func fnWildcard(args []string) string {
	if len(args) == 0 {
		return ""
	}
	patterns := strings.Fields(args[0])
	var matches []string
	for _, p := range patterns {
		m, err := filepath.Glob(p)
		if err == nil {
			matches = append(matches, m...)
		}
	}
	return strings.Join(matches, " ")
}

func fnRealpath(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	for i, w := range words {
		p, err := filepath.EvalSymlinks(w)
		if err == nil {
			p, err = filepath.Abs(p)
			if err == nil {
				words[i] = p
			}
		}
	}
	return strings.Join(words, " ")
}

func fnAbspath(args []string) string {
	if len(args) == 0 {
		return ""
	}
	words := strings.Fields(args[0])
	for i, w := range words {
		p, err := filepath.Abs(w)
		if err == nil {
			words[i] = p
		}
	}
	return strings.Join(words, " ")
}

func fnFile(args []string) string {
	if len(args) == 0 {
		return ""
	}
	mode := ">"
	fn := args[0]
	text := ""
	if len(args) > 1 {
		text = args[1]
	}
	if strings.HasPrefix(fn, ">") {
		mode = ">"
		fn = strings.TrimSpace(fn[1:])
	} else if strings.HasPrefix(fn, ">>") {
		mode = ">>"
		fn = strings.TrimSpace(fn[2:])
	}
	if fn == "" {
		return ""
	}
	switch mode {
	case ">":
		os.WriteFile(fn, []byte(text), 0666)
	case ">>":
		f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			defer f.Close()
			f.WriteString(text)
		}
	}
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func atoi(s string) int {
	n := 0
	neg := false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	if neg {
		n = -n
	}
	return n
}

func Init() {
	expand.FuncHandler = CallFunction
}
