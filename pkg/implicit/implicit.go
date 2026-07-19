package implicit

import (
	"sort"
	"strings"

	"github.com/amyinfo/gmake/pkg/commands"
	"github.com/amyinfo/gmake/pkg/debug"
	"github.com/amyinfo/gmake/pkg/dep"
	"github.com/amyinfo/gmake/pkg/dir"
	"github.com/amyinfo/gmake/pkg/expand"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/rule"
	"github.com/amyinfo/gmake/pkg/shuffle"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
	"github.com/amyinfo/gmake/pkg/vpath"
)

type patdeps struct {
	Name               string
	Pattern            string
	File               *types.File
	IgnoreMtime        bool
	IgnoreAutomaticVars bool
	IsExplicit         bool
	WaitHere           bool
}

type tryrule struct {
	Rule             *types.Rule
	Stemlen          uint
	Matches          uint
	Order            uint
	CheckedLastslash byte
}

func TryImplicitRule(f *types.File, depth uint) int {
	debug.DBF(debug.Implicit, "Looking for an implicit rule for '%s'.\n", f.Name)
	if PatternSearch(f, 0, depth, 0, false) {
		return 1
	}
	return 0
}

func getNextWord(buffer string) (string, uint) {
	if len(buffer) == 0 {
		return "", 0
	}
	p := 0
	for p < len(buffer) && (buffer[p] == ' ' || buffer[p] == '\t') {
		p++
	}
	if p >= len(buffer) {
		return "", 0
	}
	beg := p
	for p < len(buffer) {
		c := buffer[p]
		switch c {
		case 0, ' ', '\t':
			return buffer[beg:p], uint(p - beg)
		case '$':
			p++
			if p < len(buffer) && buffer[p] == '$' {
				p++
				continue
			}
			if p >= len(buffer) {
				break
			}
			var closeparen byte
			switch buffer[p] {
			case '(':
				closeparen = ')'
			case '{':
				closeparen = '}'
			default:
				p++
				continue
			}
			p++
			count := 0
			for p < len(buffer) {
				if buffer[p] == c {
					count++
				} else if buffer[p] == closeparen {
					if count <= 0 {
						p++
						break
					}
					count--
				}
				p++
			}
			continue
		case '|':
			if beg == p {
				p++
				return buffer[beg:p], uint(p - beg)
			}
			return buffer[beg:p], uint(p - beg)
		}
		p++
	}
	return buffer[beg:p], uint(p - beg)
}

func PatternSearch(f *types.File, archive int, depth uint, recursions uint, allowCompatRules bool) bool {
	filename := f.Name
	namelen := uint(len(filename))

	lastslash := -1
	if archive == 0 && namelen > 1 {
		lastslash = strings.LastIndexByte(filename[:namelen-1], '/')
	}

	var pathlen uint
	if lastslash >= 0 {
		pathlen = uint(lastslash + 1)
	}

	var intFile *types.File

	maxDeps := rule.MaxPatternDeps
	deplist := make([]patdeps, maxDeps)
	pat := 0

	fileVarsInitialized := false
	specificRuleMatched := false
	foundCompatRule := false
	var stemlen uint

	tryrules := make([]tryrule, rule.NumPatternRules*rule.MaxPatternTargets)
	nrules := uint(0)

	for r := rule.PatternRules; r != nil; r = r.Next {
		if r.Deps != nil && r.Cmds == nil {
			continue
		}
		if r.InUse != 0 {
			debug.DBS(debug.Implicit, 0, "Avoiding implicit rule recursion\n")
			continue
		}
		for ti := uint(0); ti < uint(r.Num); ti++ {
			target := r.Targets[ti]
			suffix := r.Suffixes[ti]
			checkLastslash := byte(0)

			if recursions > 0 && len(target) > 1 && r.Terminal == 0 {
				continue
			}
			if int(r.Lens[ti]) > int(namelen) {
				continue
			}

			percentIdx := len(target) - len(suffix) - 1
			var stemStart uint
			if percentIdx < 0 {
				stemStart = 0
			} else {
				stemStart = uint(percentIdx)
			}
			stemlen = namelen - uint(r.Lens[ti]) + 1

			if lastslash >= 0 && strings.IndexByte(target, '/') < 0 {
				checkLastslash = 1
			}
			if checkLastslash != 0 {
				if pathlen > stemlen {
					continue
				}
				stemlen -= pathlen
				stemStart += pathlen
			}

			if checkLastslash != 0 {
				if percentIdx > 0 {
					targetPrefix := target[:percentIdx]
					fnamePrefix := filename[lastslash+1 : lastslash+1+len(targetPrefix)]
					if targetPrefix != fnamePrefix {
						continue
					}
				}
			} else {
				if percentIdx > 0 {
					targetPrefix := target[:percentIdx]
					if targetPrefix != filename[:len(targetPrefix)] {
						continue
					}
				}
			}

			afterStem := int(stemStart + stemlen)
			if afterStem+len(suffix) > int(namelen) || filename[afterStem:afterStem+len(suffix)] != suffix {
				continue
			}

			if len(target) > 1 {
				specificRuleMatched = true
			}
			if r.Deps == nil && r.Cmds == nil {
				continue
			}

			tryrules[nrules] = tryrule{
				Rule:             r,
				Matches:          ti,
				Stemlen:          stemlen + (uint(checkLastslash) * pathlen),
				Order:            nrules,
				CheckedLastslash: checkLastslash,
			}
			nrules++
		}
	}

	if nrules == 0 {
		return false
	}

	if nrules > 1 {
		sort.Slice(tryrules[:nrules], func(i, j int) bool {
			if tryrules[i].Stemlen != tryrules[j].Stemlen {
				return tryrules[i].Stemlen < tryrules[j].Stemlen
			}
			return tryrules[i].Order < tryrules[j].Order
		})
	}

	if specificRuleMatched {
		for ri := uint(0); ri < nrules; ri++ {
			if tryrules[ri].Rule != nil && tryrules[ri].Rule.Terminal == 0 {
				for j := uint(0); j < uint(tryrules[ri].Rule.Num); j++ {
					if len(tryrules[ri].Rule.Targets[j]) == 1 {
						tryrules[ri].Rule = nil
						break
					}
				}
			}
		}
	}

	var foundRule *types.Rule
	foundInIntermed := false
	var foundruleIdx uint

	for intermedOk := 0; intermedOk < 2; intermedOk++ {
		pat = 0
		if intermedOk != 0 {
			debug.DBS(debug.Implicit, 0, "Trying harder.\n")
		}

		for ri := uint(0); ri < nrules; ri++ {
			ru := tryrules[ri].Rule
			if ru == nil {
				continue
			}
			if intermedOk != 0 && ru.Terminal != 0 {
				continue
			}

			matches := tryrules[ri].Matches
			stemStart := uint(len(ru.Targets[matches]) - len(ru.Suffixes[matches]) - 1)
			stemlen = namelen - uint(ru.Lens[matches]) + 1
			checkLastslash := tryrules[ri].CheckedLastslash
			if checkLastslash != 0 {
				stemStart += pathlen
				stemlen -= pathlen
			}

			stemStr := filename[stemStart : stemStart+stemlen]

			debug.DBS(debug.Implicit, 0, "Trying pattern rule '%s' with stem '%.*s'.\n",
				rule.GetRuleDefn(ru), int(stemlen), stemStr)

			fullStemLen := stemlen
			if checkLastslash != 0 {
				fullStemLen += pathlen
			}

			stemBuf := make([]byte, fullStemLen)
			if checkLastslash != 0 {
				copy(stemBuf, filename[:pathlen])
				copy(stemBuf[pathlen:], stemStr)
			} else {
				copy(stemBuf, stemStr)
			}

			if ru.Deps == nil {
				foundRule = ru
				foundInIntermed = true
				break
			}

			ru.InUse = 1
			failed := false
			fileVariablesSet := false
			depsFound := 0
			orderOnly := false

			for dep_ := ru.Deps; dep_ != nil; {
				nptr := dep.DepName(dep_)
				if nptr == "" {
					dep_ = dep_.Next
					continue
				}

				for {
					if !dep_.Need2ndExpansion {
						cp := strings.IndexByte(nptr, '%')
						var expanded string
						var isExplicit bool
						if cp < 0 {
							expanded = nptr
							isExplicit = true
						} else {
							var b strings.Builder
							if checkLastslash != 0 {
								b.WriteString(filename[:pathlen])
							}
							b.WriteString(nptr[:cp])
							b.Write(stemBuf)
							b.WriteString(nptr[cp+1:])
							expanded = b.String()
							isExplicit = false
						}
						p := expanded
						dl := dep.ParseFileSeq(&p, 0, "", 0)
						for d := dl; d != nil; d = d.Next {
							depsFound++
							if pat >= len(deplist) {
								newList := make([]patdeps, pat+16)
								copy(newList, deplist)
								deplist = newList
							}
							deplist[pat] = patdeps{
								Name:               d.Name_,
								IgnoreMtime:        dep_.IgnoreMtime,
								IgnoreAutomaticVars: dep_.IgnoreAutomaticVars,
								WaitHere:           dep_.WaitHere,
								IsExplicit:         isExplicit,
							}
							pat++
						}
						dep.FreeDepChain(dl)
						nptr = ""
						break
					}

					word, wlen := getNextWord(nptr)
					if word == "" {
						break
					}
					nptr = nptr[wlen:]

					if !orderOnly && wlen == 1 && word[0] == '|' {
						orderOnly = true
						continue
					}

					isExplicit := true
					addDir := false
					depname := word
					cp := strings.IndexByte(word, '%')
					if cp >= 0 {
						isExplicit = false
						var b strings.Builder
						b.WriteString(word[:cp])
						if checkLastslash != 0 {
							b.WriteString("$(*F)")
							addDir = true
						} else {
							b.WriteString("$*")
						}
						b.WriteString(word[cp+1:])
						depname = b.String()
					}

					if !fileVarsInitialized {
						variable.InitializeFileVariables(f, 0)
						commands.SetFileVariables(f, string(stemBuf))
						fileVarsInitialized = true
					} else if !fileVariablesSet {
						variable.DefineVariableForFile("*", 1, string(stemBuf), types.OriginAutomatic, false, f)
						fileVariablesSet = true
					}

					expanded := expand.VariableExpandForFile(depname, f)
					p := expanded

					var pathdir string
					if addDir {
						pathdir = filename[:pathlen]
					}

					for p != "" {
						dl := dep.ParseFileSeq(&p, 0, pathdir, 0)
						for d := dl; d != nil; d = d.Next {
							depsFound++
							if orderOnly {
								d.IgnoreMtime = true
							}
							d.IsExplicit = isExplicit
							if pat >= len(deplist) {
								newList := make([]patdeps, pat+16)
								copy(newList, deplist)
								deplist = newList
							}
							deplist[pat] = patdeps{
								Name:               d.Name_,
								IgnoreMtime:        d.IgnoreMtime,
								IgnoreAutomaticVars: d.IgnoreAutomaticVars,
								WaitHere:           d.WaitHere,
								IsExplicit:         d.IsExplicit,
							}
							pat++
						}
						dep.FreeDepChain(dl)

						if len(p) > 0 && p[0] == '|' {
							orderOnly = true
							p = p[1:]
						}
					}
				}

				if failed {
					break
				}

				dep_ = dep_.Next
			}

			patIdx := 0
			for pi := 0; pi < pat; pi++ {
				pdep := &deplist[pi]
				if pdep.Name == "" {
					continue
				}

				df := file.LookupFile(pdep.Name)

				if df != nil && !df.IsExplicit && !pdep.IsExplicit {
					df.Intermediate = true
				}

				explicit := false
				if df != nil && df.IsTarget {
					explicit = true
				} else {
					for dp := f.Deps; dp != nil; dp = dp.Next {
						if pdep.Name == dep.DepName(dp) {
							explicit = true
							break
						}
					}
				}

				if explicit || pdep.IsExplicit {
					deplist[patIdx] = *pdep
					patIdx++
					continue
				}

				if dir.FileExistsP(pdep.Name) != 0 {
					deplist[patIdx] = *pdep
					patIdx++
					continue
				}

				if df != nil && allowCompatRules {
					deplist[patIdx] = *pdep
					patIdx++
					continue
				}

				if df != nil {
					debug.DBS(debug.Implicit, 0,
						"Prerequisite '%s' of rule does not qualify as ought to exist.\n", pdep.Name)
					foundCompatRule = true
				}

				{
					vname := vpath.VpathSearch(pdep.Name, nil, nil, nil)
					if vname != "" {
						debug.DBS(debug.Implicit, 0,
							"Found prerequisite '%s' as VPATH '%s'.\n", pdep.Name, vname)
						deplist[patIdx] = *pdep
						patIdx++
						continue
					}
				}

				if intermedOk != 0 {
					debug.DBS(debug.Implicit, 0,
						"Looking for a rule with intermediate file '%s'.\n", pdep.Name)

					intFile = &types.File{Name: pdep.Name}

					if PatternSearch(intFile, 0, depth+1, recursions+1, allowCompatRules) {
						deplist[patIdx] = *pdep
						deplist[patIdx].Pattern = intFile.Name
						intFile.Name = pdep.Name
						deplist[patIdx].File = intFile
						intFile = nil
						patIdx++
						continue
					}

					if df == nil {
						dir.FileImpossible(pdep.Name)
					}
				}

				if intermedOk != 0 {
					debug.DBS(debug.Implicit, 0,
						"Rejecting rule due to impossible prerequisite '%s'.\n", pdep.Name)
				} else {
					debug.DBS(debug.Implicit, 0, "Not found '%s'.\n", pdep.Name)
				}
				failed = true
				break
			}

			pat = patIdx
			ru.InUse = 0

			if !failed {
				foundRule = ru
				foundInIntermed = true
				break
			}
		}

		if foundInIntermed {
			break
		}
	}

	if foundRule == nil {
		if foundCompatRule {
			return PatternSearch(f, archive, depth, recursions, true)
		}
		debug.DBS(debug.Implicit, 0, "No implicit rule found for '%s'.\n", filename)
		return false
	}

	for ri := uint(0); ri < nrules; ri++ {
		if tryrules[ri].Rule == foundRule {
			foundruleIdx = ri
			break
		}
	}

	if recursions > 0 {
		f.Name = foundRule.Targets[tryrules[foundruleIdx].Matches]
	}

	for pi := pat - 1; pi >= 0; pi-- {
		pdep := &deplist[pi]
		if pdep.Name == "" {
			continue
		}
		if pdep.File != nil {
			imf := pdep.File
			f2 := file.LookupFile(imf.Name)
			if f2 == nil {
				f2 = file.EnterFile(imf.Name)
			}
			f2.Deps = imf.Deps
			f2.Cmds = imf.Cmds
			f2.Stem = imf.Stem
			f2.IsTarget = true
			f2.Intermediate = true
			f2.TriedImplicit = true
			for d := f2.Deps; d != nil; d = d.Next {
				d.File = file.EnterFile(dep.DepName(d))
				d.Name_ = ""
			}
		}

		if pdep.File == nil && tryrules[foundruleIdx].Rule.Terminal != 0 {
			d := f.Deps
			for d != nil {
				if dep.DepName(d) == pdep.Name {
					break
				}
				d = d.Next
			}
			if d != nil {
				d.Changed = true
			}
		}

		d := dep.AllocDep()
		d.IgnoreMtime = pdep.IgnoreMtime
		d.IsExplicit = pdep.IsExplicit
		d.IgnoreAutomaticVars = pdep.IgnoreAutomaticVars
		d.WaitHere = pdep.WaitHere
		s := strcache.Add(pdep.Name)
		if recursions > 0 {
			d.Name_ = s
		} else {
			d.File = file.LookupFile(s)
			if d.File == nil {
				d.File = file.EnterFile(s)
			}
		}
		d.Next = f.Deps
		f.Deps = d
		f.WasShuffled = false
	}

	if !f.WasShuffled {
		shuffle.ShuffleDepsRecursive(f.Deps)
	}

	if tryrules[foundruleIdx].CheckedLastslash == 0 {
		percentIdx := len(foundRule.Targets[tryrules[foundruleIdx].Matches]) -
			len(foundRule.Suffixes[tryrules[foundruleIdx].Matches]) - 1
		if percentIdx < 0 {
			percentIdx = 0
		}
		f.Stem = strcache.AddLen(filename[uint(percentIdx):], int(stemlen))
	} else {
		percentIdx := len(foundRule.Targets[tryrules[foundruleIdx].Matches]) -
			len(foundRule.Suffixes[tryrules[foundruleIdx].Matches]) - 1
		if percentIdx < 0 {
			percentIdx = 0
		}
		stemBuf := make([]byte, pathlen+stemlen)
		copy(stemBuf, filename[:pathlen])
		copy(stemBuf[pathlen:], filename[uint(percentIdx):uint(percentIdx)+stemlen])
		f.Stem = strcache.Add(string(stemBuf))
	}

	f.Cmds = foundRule.Cmds
	f.IsTarget = true

	if foundRule.Num > 1 {
		for ri := uint(0); ri < uint(foundRule.Num); ri++ {
			if ri != tryrules[foundruleIdx].Matches {
				tgt := foundRule.Targets[ri]
				suff := foundRule.Suffixes[ri]
				percentIdx := len(tgt) - len(suff) - 1
				nm := tgt[:percentIdx] + f.Stem + tgt[percentIdx+1:]

				newDep := dep.AllocDep()
				newDep.Name_ = strcache.Add(nm)
				newDep.File = file.EnterFile(newDep.Name_)
				newDep.Next = f.AlsoMake
				newDep.File.IsTarget = true
				f.AlsoMake = newDep
			}
		}
	}

	debug.DBS(debug.Implicit, 0, "Found implicit rule '%s' for '%s'.\n", rule.GetRuleDefn(foundRule), filename)
	return true
}
