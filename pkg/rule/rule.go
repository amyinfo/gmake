package rule

import (
	"fmt"
	"strings"

	"github.com/amyinfo/gmake/pkg/dep"
	"github.com/amyinfo/gmake/pkg/dir"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
)

var PatternRules *types.Rule
var LastPatternRule *types.Rule
var NumPatternRules uint
var MaxPatternTargets uint
var MaxPatternDeps uint
var MaxPatternDepLength uint
var SuffixFile *types.File
var maxsuffix uint

func freeRule(rule *types.Rule, lastRule *types.Rule) {
	next := rule.Next
	dep.FreeDepChain(rule.Deps)
	rule.Targets = nil
	rule.Suffixes = nil
	rule.Lens = nil
	rule.Defn = ""
	if PatternRules == rule {
		if lastRule != nil {
			panic("freeRule: inconsistent state")
		} else {
			PatternRules = next
		}
	} else if lastRule != nil {
		lastRule.Next = next
	}
	if LastPatternRule == rule {
		LastPatternRule = lastRule
	}
}

func GetRuleDefn(r *types.Rule) string {
	if r.Defn != "" {
		return r.Defn
	}
	result := ""
	for i := uint(0); i < uint(r.Num); i++ {
		if i > 0 {
			result += " "
		}
		result += r.Targets[i]
	}
	result += ":"
	if r.Terminal != 0 {
		result += ":"
	}
	for d := r.Deps; d != nil; d = d.Next {
		if d.IgnoreMtime {
			continue
		}
		if d.WaitHere {
			result += " .WAIT"
		}
		result += " " + dep.DepName(d)
	}
	firstOO := true
	for d := r.Deps; d != nil; d = d.Next {
		if d.IgnoreMtime {
			if firstOO {
				result += " |"
				firstOO = false
			}
			if d.WaitHere {
				result += " .WAIT"
			}
			result += " " + dep.DepName(d)
		}
	}
	r.Defn = result
	return r.Defn
}

func SnapImplicitRules() {
	prereqs := file.ExpandExtraPrereqs(variable.LookupVariable(".EXTRA_PREREQS", len(".EXTRA_PREREQS")))
	preDeps := uint(0)
	MaxPatternDepLength = 0

	for d := prereqs; d != nil; d = d.Next {
		dn := dep.DepName(d)
		l := uint(len(dn))
		if d.Need2ndExpansion {
			for idx := 0; idx >= 0; {
				pos := strings.IndexByte(dn[idx:], '%')
				if pos < 0 {
					break
				}
				l += 4
				idx += pos + 1
			}
		}
		if l > MaxPatternDepLength {
			MaxPatternDepLength = l
		}
		preDeps++
	}

	NumPatternRules = 0
	MaxPatternTargets = 0
	MaxPatternDeps = 0

	for rule := PatternRules; rule != nil; rule = rule.Next {
		ndeps := preDeps
		var lastdep *types.Dep
		NumPatternRules++
		if uint(rule.Num) > MaxPatternTargets {
			MaxPatternTargets = uint(rule.Num)
		}
		for d := rule.Deps; d != nil; d = d.Next {
			dname := dep.DepName(d)
			len_ := uint(len(dname))
			p := strings.LastIndexByte(dname, '/')
			p2 := -1
			if p >= 0 {
				p2 = strings.IndexByte(dname[p:], '%')
				if p2 >= 0 {
					p2 += p
				}
			}
			ndeps++
			if len_ > MaxPatternDepLength {
				MaxPatternDepLength = len_
			}
			if d.Next == nil {
				lastdep = d
			}
			if p2 >= 0 {
				start := p
				if start == 0 {
					start = 1
				}
				d.Changed = dir.DirFileExistsP(dname[:uint(start)], "") == 0
			} else {
				d.Changed = false
			}
		}
		if prereqs != nil {
			if lastdep != nil {
				lastdep.Next = dep.CopyDepChain(prereqs)
			} else {
				rule.Deps = dep.CopyDepChain(prereqs)
			}
		}
		if ndeps > MaxPatternDeps {
			MaxPatternDeps = ndeps
		}
	}
	dep.FreeDepChain(prereqs)
}

func convertSuffixRule(target string, source string, cmds *types.Commands) {
	names := make([]string, 1)
	percents := make([]string, 1)
	if target == "" {
		names[0] = strcache.Add("(%.o)")
		percents[0] = names[0][1:]
	} else {
		names[0] = strcache.Add("%" + target)
		percents[0] = names[0]
	}
	var deps *types.Dep
	if source != "" {
		deps = dep.AllocDep()
		deps.Name_ = strcache.Add("%" + source)
	}
	CreatePatternRule(names, percents, 1, false, deps, cmds, false)
}

func ConvertToPattern() {
	maxsuffix = 0
	for d := SuffixFile.Deps; d != nil; d = d.Next {
		l := uint(len(dep.DepName(d)))
		if l > maxsuffix {
			maxsuffix = l
		}
	}
	rulename := make([]byte, maxsuffix*2+1)
	for d := SuffixFile.Deps; d != nil; d = d.Next {
		slen := uint(len(dep.DepName(d)))
		convertSuffixRule(dep.DepName(d), "", nil)
		if d.File != nil && d.File.Cmds != nil {
			convertSuffixRule("", dep.DepName(d), d.File.Cmds)
		}
		copy(rulename, []byte(dep.DepName(d)))
		for d2 := SuffixFile.Deps; d2 != nil; d2 = d2.Next {
			s2len := uint(len(dep.DepName(d2)))
			if slen == s2len && dep.DepName(d) == dep.DepName(d2) {
				continue
			}
			copy(rulename[slen:], []byte(dep.DepName(d2)))
			rulename[slen+s2len] = 0
			rn := string(rulename[:slen+s2len])
			f := file.LookupFile(rn)
			if f == nil || f.Cmds == nil {
				continue
			}
			if s2len == 2 && rulename[slen] == '.' && rulename[slen+1] == 'a' {
				convertSuffixRule("", dep.DepName(d), f.Cmds)
			}
			convertSuffixRule(dep.DepName(d2), dep.DepName(d), f.Cmds)
		}
	}
}

func newPatternRule(rule *types.Rule, override bool) bool {
	rule.InUse = 0
	rule.Terminal = 0
	rule.Next = nil
	var lastRule *types.Rule
	r := PatternRules
	for r != nil {
		for i := uint(0); i < uint(rule.Num); i++ {
			j := uint(0)
			for j = 0; j < uint(r.Num); j++ {
				if rule.Targets[i] != r.Targets[j] {
					break
				}
			}
			if j == uint(r.Num) {
				d := rule.Deps
				d2 := r.Deps
				for d != nil && d2 != nil {
					if dep.DepName(d) != dep.DepName(d2) {
						break
					}
					d = d.Next
					d2 = d2.Next
				}
				if d == nil && d2 == nil {
					if override {
						freeRule(r, lastRule)
						if PatternRules == nil {
							PatternRules = rule
						} else {
							LastPatternRule.Next = rule
						}
						LastPatternRule = rule
						return true
					} else {
						freeRule(rule, nil)
						return false
					}
				}
			}
		}
		lastRule = r
		r = r.Next
	}

	if PatternRules == nil {
		PatternRules = rule
	} else {
		LastPatternRule.Next = rule
	}
	LastPatternRule = rule
	return true
}

func InstallPatternRule(target, dep_, commands_ string, terminal bool) {
	r := &types.Rule{
		Num:      1,
		Targets:  make([]string, 1),
		Suffixes: make([]string, 1),
		Lens:     make([]uint, 1),
		Defn:     "",
	}
	r.Lens[0] = uint(len(target))
	r.Targets[0] = target
	percentPos := strings.IndexByte(target, '%')
	if percentPos < 0 {
		panic("InstallPatternRule: target must contain %")
	}
	r.Suffixes[0] = target[percentPos+1:]
	ptr := dep_
	r.Deps = dep.ParseSimpleSeq(&ptr)
	if newPatternRule(r, false) {
		if terminal {
			r.Terminal = 1
		}
		r.Cmds = &types.Commands{
			Fileinfo:     types.Floc{},
			Commands:     commands_,
			CommandLines: nil,
			RecipePrefix: '\t',
		}
	}
}

func CreatePatternRule(targets []string, targetPercents []string, n uint16, terminal bool, deps *types.Dep, commands_ *types.Commands, override bool) {
	r := &types.Rule{
		Num:      n,
		Cmds:     commands_,
		Deps:     deps,
		Targets:  targets,
		Suffixes: targetPercents,
		Lens:     make([]uint, n),
		Defn:     "",
	}
	for i := uint(0); i < uint(n); i++ {
		r.Lens[i] = uint(len(targets[i]))
		if len(targetPercents[i]) == 0 {
			panic("CreatePatternRule: target must contain %")
		}
	}
	if newPatternRule(r, override) {
		if terminal {
			r.Terminal = 1
		}
	}
}

func PrintRule(r *types.Rule) {
	fmt.Println(GetRuleDefn(r))
	if r.Cmds != nil {
		fmt.Println("# commands:", r.Cmds.Commands)
	}
}

func PrintRuleDataBase() {
	fmt.Println("\n# Implicit Rules")
	rules := uint(0)
	term := uint(0)
	for r := PatternRules; r != nil; r = r.Next {
		rules++
		fmt.Println()
		PrintRule(r)
		if r.Terminal != 0 {
			term++
		}
	}
	if rules == 0 {
		fmt.Println("\n# No implicit rules.")
	} else {
		pct := float64(term) / float64(rules) * 100.0
		fmt.Printf("\n# %d implicit rules, %d (%.1f%%) terminal.\n", rules, term, pct)
	}
	if NumPatternRules != rules {
		if NumPatternRules != 0 {
			panic(fmt.Sprintf("BUG: num_pattern_rules is wrong! %d != %d", NumPatternRules, rules))
		}
	}
}
