package remake

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/amyinfo/gmake/pkg/commands"
	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/debug"
	"github.com/amyinfo/gmake/pkg/dep"
	"github.com/amyinfo/gmake/pkg/expand"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/implicit"
	"github.com/amyinfo/gmake/pkg/job"
	"github.com/amyinfo/gmake/pkg/misc"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/vpath"
)

var CommandsStarted uint
var DefaultFile *types.File
var goalDeps []*types.Goaldep
var goalDep *types.Dep
var considered uint

const fileTimestampsPerS = 1000000000

func startUpdating(f *types.File) {
	base := f
	if f.DoubleColon != nil {
		base = f.DoubleColon
	}
	base.Updating = true
}

func finishUpdating(f *types.File) {
	base := f
	if f.DoubleColon != nil {
		base = f.DoubleColon
	}
	base.Updating = false
}

func isUpdating(f *types.File) bool {
	base := f
	if f.DoubleColon != nil {
		base = f.DoubleColon
	}
	return base.Updating
}

func rehashFile(f *types.File, name string) {
	if name == "" {
		return
	}
	f.Hname = name
}

func checkRenamed(*types.File) {}

func msg(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

func errorMsg(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func fatalMsg(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}

func perrorWithName(call, name string) {
	fmt.Fprintf(os.Stderr, "%s%s\n", call, name)
}

func checkAlsoMake(f *types.File) {
	mtime := config.FileTimestamp(f.LastMtime)
	if mtime == config.UnknownMtime {
		mtime = nameMtime(f.Name)
	}
	if config.IsOrdinaryMtime(mtime) && mtime > config.FileTimestamp(f.MtimeBeforeUpdate) {
		for ad := f.AlsoMake; ad != nil; ad = ad.Next {
			if ad.File.LastMtime == uint64(config.NonexistentMtime) {
errorMsg("warning: pattern recipe did not update peer target '%s'.", ad.File.Name)
			}
		}
	}
}

func nameMtime(name string) config.FileTimestamp {
	st, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return config.NonexistentMtime
		}
		perrorWithName("stat: ", name)
		return config.NonexistentMtime
	}
	return config.FileTimestamp(st.ModTime().UnixNano())
}

func UpdateGoalChain(goaldeps *types.Goaldep) types.UpdateStatus {
	t := config.TouchFlag
	q := config.QuestionFlag
	n := config.JustPrintFlag
	status := types.UpdateNone

	goalsOrig := dep.CopyDepChain(&goaldeps.Dep)
	goals := goalsOrig

	goalDeps = nil
	for g := goaldeps; g != nil; g = (*types.Goaldep)(unsafe.Pointer(g.Next)) {
		goalDeps = append(goalDeps, g)
	}
	considered++

	for goals != nil {
		job.StartWaitingJobs()
		job.ReapChildren(1, 0)

		var lastgoal *types.Dep
		gu := goals
		for gu != nil {
			g := gu
			goalDep = g

			var stop bool
			var anyNotUpdated bool

			f := g.File.DoubleColon
			if f == nil {
				f = g.File
			}
			for ; f != nil; f = f.Prev {
				f.Dontcare = false
				checkRenamed(f)

				ocommandsStarted := CommandsStarted
				fail := UpdateFile(f, 1)
				checkRenamed(f)

				if CommandsStarted > ocommandsStarted {
					g.Changed = true
				}

				stop = false
				if (fail != types.UpdateNone || f.Updated) && status < types.UpdateQuestion {
					if f.UpdateStatus != 0 {
						status = types.UpdateStatus(f.UpdateStatus)
						if config.QuestionFlag && !config.KeepGoingFlag {
							stop = true
						}
					} else {
						mtime := config.FileTimestamp(f.LastMtime)
						checkRenamed(f)
						if f.Updated && mtime != config.FileTimestamp(f.MtimeBeforeUpdate) {
							status = types.UpdateSuccess
						}
					}
				}
				anyNotUpdated = anyNotUpdated || !f.Updated
				if stop {
					break
				}
			}

			f = g.File
			if stop || !anyNotUpdated {
				if types.UpdateStatus(f.UpdateStatus) == types.UpdateSuccess && !g.Changed &&
					!config.RunSilent && !config.QuestionFlag {
					if f.Phony || f.Cmds == nil {
						msg("Nothing to be done for '%s'.", f.Name)
					} else {
						msg("'%s' is up to date.", f.Name)
					}
				}
				if lastgoal == nil {
					goals = gu.Next
				} else {
					lastgoal.Next = gu.Next
				}
				if lastgoal == nil {
					gu = goals
				} else {
					gu = lastgoal.Next
				}
				if stop {
					break
				}
			} else {
				lastgoal = gu
				gu = gu.Next
			}
		}
		if gu == nil {
			considered++
		}
	}
	dep.FreeDepChain(goalsOrig)
	config.TouchFlag = t
	config.QuestionFlag = q
	config.JustPrintFlag = n
	return status
}

func ShowGoalError() {
	if int(goalDep.Flags)&(types.RMIncluded|types.RMDontcare) != types.RMIncluded {
		return
	}
	for _, goal := range goalDeps {
		if goalDep.File == goal.File {
			if goal.Error != 0 {
				errorMsg("%s: %s", goal.File.Name, fmt.Sprintf("%d", goal.Error))
				goal.Error = 0
			}
			return
		}
	}
}

func complain(f *types.File) {
	for d := f.Deps; d != nil; d = d.Next {
		if d.File.Updated && d.File.UpdateStatus > 0 && f.NoDiag {
			complain(d.File)
			return
		}
	}
	ShowGoalError()
	if f.Parent != nil {
		if !config.KeepGoingFlag {
			fatalMsg("No rule to make target '%s', needed by '%s'", f.Name, f.Parent.Name)
		}
		errorMsg("No rule to make target '%s', needed by '%s'", f.Name, f.Parent.Name)
	} else {
		if !config.KeepGoingFlag {
			fatalMsg("No rule to make target '%s'", f.Name)
		}
		errorMsg("No rule to make target '%s'", f.Name)
	}
	f.NoDiag = false
}

func UpdateFile(f *types.File, depth uint) types.UpdateStatus {
	status := types.UpdateSuccess
	f2 := f
	if f.DoubleColon != nil {
		f2 = f.DoubleColon
	}
	if f2.Considered == considered {
		if !(f2.Updated && f2.UpdateStatus > 0 && !f2.Dontcare && f2.NoDiag) {
			if f2.CommandState == types.CmdFinished {
				return types.UpdateStatus(f2.UpdateStatus)
			}
			return types.UpdateSuccess
		}
	}
	for ; f2 != nil; f2 = f2.Prev {
		f2.Considered = considered
		s := updateFile1(f2, depth)
		checkRenamed(f2)
		if s >= types.UpdateQuestion && !config.KeepGoingFlag {
			return s
		}
		if f2.CommandState == types.CmdRunning || f2.CommandState == types.CmdDepsRunning {
			return types.UpdateSuccess
		}
		if s > status {
			status = s
		}
	}
	return status
}

func updateFile1(f *types.File, depth uint) types.UpdateStatus {
	depStatus := types.UpdateNone
	noexist := false
	mustMake := false
	running := false

	debug.DBF(debug.Verbose, "Considering target file '%s'.\n", f.Name)

	if f.Updated {
		if f.UpdateStatus > 0 {
			if f.NoDiag && !f.Dontcare {
				complain(f)
			}
			return types.UpdateStatus(f.UpdateStatus)
		}
		debug.DBF(debug.Verbose, "File '%s' was considered already.\n", f.Name)
		return types.UpdateSuccess
	}

	switch f.CommandState {
	case types.CmdNotStarted, types.CmdDepsRunning:
	case types.CmdRunning:
		debug.DBF(debug.Verbose, "Still updating file '%s'.\n", f.Name)
		return types.UpdateSuccess
	case types.CmdFinished:
		debug.DBF(debug.Verbose, "Finished updating file '%s'.\n", f.Name)
		return types.UpdateStatus(f.UpdateStatus)
	default:
		return types.UpdateSuccess
	}

	f.NoDiag = f.Dontcare
	depth++
	startUpdating(f)
	ofile := f

	thisMtime := config.FileTimestamp(f.LastMtime)
	checkRenamed(f)
	// Resolve unknown mtime (file hasn't been stat'd yet)
	if thisMtime == config.UnknownMtime {
		thisMtime = config.FileTimestamp(fMtime(f, 0))
	}
	noexist = thisMtime == config.NonexistentMtime
	if noexist {
		debug.DBF(debug.Basic, "File '%s' does not exist.\n", f.Name)
	} else if config.IsOrdinaryMtime(thisMtime) && f.LowResolutionTime {
		ns := int64(thisMtime) % fileTimestampsPerS
		if ns != 0 {
			errorMsg("*** Warning: .LOW_RESOLUTION_TIME file '%s' has a high resolution time stamp", f.Name)
		}
		thisMtime += config.FileTimestamp(fileTimestampsPerS - 1 - ns)
	}

	for ad := f.AlsoMake; ad != nil && !noexist; ad = ad.Next {
		adfile := ad.File
		fmtime := config.FileTimestamp(adfile.LastMtime)
		noexist = fmtime == config.NonexistentMtime
		if noexist {
			checkRenamed(adfile)
		} else if fmtime < thisMtime {
			thisMtime = fmtime
		}
	}

	mustMake = noexist

	if !f.Phony && f.Cmds == nil && !f.TriedImplicit {
		implicit.TryImplicitRule(f, depth)
		f.TriedImplicit = true
	}
	if f.Cmds == nil && !f.IsTarget && DefaultFile != nil && DefaultFile.Cmds != nil {
		debug.DBF(debug.Implicit, "Using default recipe for '%s'.\n", f.Name)
		f.Cmds = DefaultFile.Cmds
	}

	amake := &types.Dep{File: f, Next: f.AlsoMake}
	for ad := amake; ad != nil; ad = ad.Next {
		if config.SecondExpansion {
			file.ExpandDeps(ad.File)
		}

		var lastd *types.Dep
		for du := ad.File.Deps; du != nil; {
			d := du
			if d.WaitHere && running {
				break
			}
			checkRenamed(d.File)
			mtime := d.File.LastMtime
			checkRenamed(d.File)

			if isUpdating(d.File) {
				errorMsg("Circular %s <- %s dependency dropped.", f.Name, d.File.Name)
				if lastd == nil {
					ad.File.Deps = du.Next
				} else {
					lastd.Next = du.Next
				}
				du = du.Next
				continue
			}

			d.File.Parent = f
			maybeMake := mustMake
			dontcare := false
			if config.RebuildingMakefiles {
				dontcare = d.File.Dontcare
				d.File.Dontcare = f.Dontcare
			}

			newStatus := checkDep(d.File, depth, thisMtime, &maybeMake)
			if newStatus > depStatus {
				depStatus = newStatus
			}
			if config.RebuildingMakefiles {
				d.File.Dontcare = dontcare
			}
			if !d.IgnoreMtime {
				mustMake = maybeMake
			}
			checkRenamed(d.File)

			for ff := d.File; ff != nil; ff = ff.Prev {
				if ff.DoubleColon != nil {
					ff = ff.DoubleColon
				}
				if ff.CommandState == types.CmdRunning || ff.CommandState == types.CmdDepsRunning {
					running = true
				}
				if ff == d.File {
					break
				}
			}

			if depStatus >= types.UpdateQuestion && !config.KeepGoingFlag {
				break
			}
			if !running {
				d.Changed = (f.LastMtime != mtime) || (mtime == uint64(config.NonexistentMtime))
			}
			lastd = du
			du = du.Next
		}
	}

	if mustMake || config.AlwaysMakeFlag {
		_ = false
		for du := f.Deps; du != nil; du = du.Next {
			d := du
			if d.WaitHere && running {
				break
			}
			if d.File.Intermediate {
				d.File.Considered = 0
				d.File.Parent = f
				dontcare := false
				if config.RebuildingMakefiles {
					dontcare = d.File.Dontcare
					d.File.Dontcare = f.Dontcare
				}
				mtime := d.File.LastMtime
				newStatus := UpdateFile(d.File, depth)
				if newStatus > depStatus {
					depStatus = newStatus
				}
				if config.RebuildingMakefiles {
					d.File.Dontcare = dontcare
				}
				checkRenamed(d.File)
				for ff := d.File; ff != nil; ff = ff.Prev {
					if ff.DoubleColon != nil {
						ff = ff.DoubleColon
					}
					if ff.CommandState == types.CmdRunning || ff.CommandState == types.CmdDepsRunning {
						running = true
					}
					if ff == d.File {
						break
					}
				}
				if depStatus >= types.UpdateQuestion && !config.KeepGoingFlag {
					break
				}
				if !running {
					d.Changed = (f.Phony && f.Cmds != nil) || d.File.LastMtime != mtime
				}
			}
		}
	}

	finishUpdating(f)
	finishUpdating(ofile)

	debug.DBF(debug.Verbose, "Finished prerequisites of target file '%s'.\n", f.Name)

	if running {
		file.SetCommandState(f, types.CmdDepsRunning)
		depth--
		debug.DBF(debug.Verbose, "The prerequisites of '%s' are being made.\n", f.Name)
		return types.UpdateSuccess
	}

	if depStatus >= types.UpdateQuestion {
		f.UpdateStatus = depStatus
		noticeFinishedFile(f)
		depth--
		if depth == 0 && config.KeepGoingFlag && !config.JustPrintFlag && !config.QuestionFlag {
			errorMsg("Target '%s' not remade because of errors.", f.Name)
		}
		return depStatus
	}

	if f.CommandState == types.CmdDepsRunning {
		file.SetCommandState(f, types.CmdNotStarted)
	}

	depsChanged := false
	for d := f.Deps; d != nil; d = d.Next {
		dMtime := d.File.LastMtime
		checkRenamed(d.File)
		if !d.IgnoreMtime {
			if dMtime == uint64(config.NonexistentMtime) && !d.File.Intermediate {
				mustMake = true
			}
			depsChanged = depsChanged || d.Changed
		}
		d.Changed = d.Changed || noexist || dMtime > uint64(thisMtime)
	}

	depth--

	if f.DoubleColon != nil && f.Deps == nil {
		mustMake = true
	} else if !noexist && f.IsTarget && !depsChanged && f.Cmds == nil && !config.AlwaysMakeFlag {
		mustMake = false
	} else if !mustMake && f.Cmds != nil && config.AlwaysMakeFlag {
		mustMake = true
	}

	if !mustMake {
		f.Secondary = true
		noticeFinishedFile(f)
		for ff := f; ff != nil; ff = ff.Prev {
			ff.Name = ff.Hname
		}
		return types.UpdateSuccess
	}

	if f.Name != f.Hname {
		f.IgnoreVpath = true
	}

	remakeFile(f)
	if f.CommandState != types.CmdFinished {
		debug.DBF(debug.Verbose, "Recipe of '%s' is being run.\n", f.Name)
		return types.UpdateSuccess
	}

	switch types.UpdateStatus(f.UpdateStatus) {
	case types.UpdateFailed:
		debug.DBF(debug.Basic, "Failed to remake target file '%s'.\n", f.Name)
	case types.UpdateSuccess:
		debug.DBF(debug.Basic, "Successfully remade target file '%s'.\n", f.Name)
	case types.UpdateQuestion:
		debug.DBF(debug.Basic, "Target file '%s' needs to be remade under -q.\n", f.Name)
	}

	f.Updated = true
	return types.UpdateStatus(f.UpdateStatus)
}

func noticeFinishedFile(f *types.File) {
	ran := f.CommandState == types.CmdRunning
	touched := false

	f.CommandState = types.CmdFinished
	f.Updated = true

	if config.TouchFlag && types.UpdateStatus(f.UpdateStatus) == types.UpdateSuccess {
		if f.Cmds != nil && f.Cmds.AnyRecurse {
			haveNonrecursing := false
			for i := uint16(0); i < f.Cmds.NCommandLines; i++ {
				if f.Cmds.LinesFlags[i]&commands.CommandsRecur == 0 {
					haveNonrecursing = true
					break
				}
			}
			if !haveNonrecursing {
				goto afterTouch
			}
		}
		if f.Phony {
			f.UpdateStatus = types.UpdateSuccess
		} else if f.Cmds != nil {
			f.UpdateStatus = touchFile(f)
			CommandsStarted++
			touched = true
		}
	}
afterTouch:

	if f.MtimeBeforeUpdate == uint64(config.UnknownMtime) {
		f.MtimeBeforeUpdate = f.LastMtime
	}

	if (ran && !f.Phony) || touched {
		i := 0
		if (config.QuestionFlag || config.JustPrintFlag || config.TouchFlag) && f.Cmds != nil {
			for i = int(f.Cmds.NCommandLines); i > 0; i-- {
				if f.Cmds.LinesFlags[i-1]&commands.CommandsRecur == 0 {
					break
				}
			}
		} else if f.IsTarget && f.Cmds == nil {
			i = 1
		}
		if i == 0 {
			f.LastMtime = uint64(config.UnknownMtime)
		} else {
			f.LastMtime = uint64(config.NewMtime)
		}
	}

	if f.DoubleColon != nil {
		maxMtime := f.LastMtime
		for ff := f.DoubleColon; ff != nil && ff.Updated; ff = ff.Prev {
			if maxMtime != uint64(config.UnknownMtime) &&
				(ff.LastMtime == uint64(config.UnknownMtime) || ff.LastMtime > maxMtime) {
				maxMtime = ff.LastMtime
			}
		}
		last := f.DoubleColon
		for ff := f.DoubleColon; ff != nil; ff = ff.Prev {
			last = ff
		}
		if last == nil {
			for ff := f.DoubleColon; ff != nil; ff = ff.Prev {
				ff.LastMtime = maxMtime
			}
		}
	}

	if ran && types.UpdateStatus(f.UpdateStatus) != types.UpdateNone {
		for d := f.AlsoMake; d != nil; d = d.Next {
			d.File.CommandState = types.CmdFinished
			d.File.Updated = true
			d.File.UpdateStatus = f.UpdateStatus
			if ran && !d.File.Phony {
				fMtime(d.File, 0)
			}
		}
		if f.TriedImplicit && f.AlsoMake != nil {
			checkAlsoMake(f)
		}
	} else if f.UpdateStatus == types.UpdateNone {
		f.UpdateStatus = types.UpdateSuccess
	}
}

func checkDep(f *types.File, depth uint, thisMtime config.FileTimestamp, mustMakePtr *bool) types.UpdateStatus {
	depStatus := types.UpdateSuccess
	depth++
	startUpdating(f)
	ofile := f

	if f.Phony || !f.Intermediate {
		depStatus = UpdateFile(f, depth)
		checkRenamed(f)
		mtime := config.FileTimestamp(f.LastMtime)
		checkRenamed(f)
		if mtime == config.NonexistentMtime || mtime > thisMtime {
			*mustMakePtr = true
		}
	} else {
		if !f.Phony && f.Cmds == nil && !f.TriedImplicit {
			implicit.TryImplicitRule(f, depth)
			f.TriedImplicit = true
		}
		if f.Cmds == nil && !f.IsTarget && DefaultFile != nil && DefaultFile.Cmds != nil {
			f.Cmds = DefaultFile.Cmds
		}
		checkRenamed(f)
		mtime := config.FileTimestamp(f.LastMtime)
		checkRenamed(f)
		if mtime != config.NonexistentMtime && mtime > thisMtime {
			*mustMakePtr = true
		} else {
			depsRunning := false
			if f.CommandState != types.CmdRunning {
				if f.CommandState == types.CmdDepsRunning {
					f.Considered = 0
				}
				file.SetCommandState(f, types.CmdNotStarted)
			}
			if config.SecondExpansion {
				file.ExpandDeps(f)
			}

			var ld *types.Dep
			for d := f.Deps; d != nil; {
				if isUpdating(d.File) {
				errorMsg("Circular %s <- %s dependency dropped.", f.Name, d.File.Name)
					if ld == nil {
						f.Deps = d.Next
						dep.FreeDep(d)
						d = f.Deps
					} else {
						ld.Next = d.Next
						dep.FreeDep(d)
						d = ld.Next
					}
					continue
				}
				d.File.Parent = f
				maybeMake := *mustMakePtr
				newStatus := checkDep(d.File, depth, thisMtime, &maybeMake)
				if newStatus > depStatus {
					depStatus = newStatus
				}
				if !d.IgnoreMtime {
					*mustMakePtr = maybeMake
				}
				checkRenamed(d.File)
				if depStatus >= types.UpdateQuestion && !config.KeepGoingFlag {
					break
				}
				if d.File.CommandState == types.CmdRunning || d.File.CommandState == types.CmdDepsRunning {
					depsRunning = true
				}
				ld = d
				d = d.Next
			}
			if depsRunning {
				file.SetCommandState(f, types.CmdDepsRunning)
			}
		}
	}

	finishUpdating(f)
	finishUpdating(ofile)
	return depStatus
}

func touchFile(f *types.File) types.UpdateStatus {
	if !config.RunSilent {
		msg("touch %s", f.Name)
	}
	if config.JustPrintFlag {
		return types.UpdateSuccess
	}

	fd, err := os.OpenFile(f.Name, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		perrorWithName("touch: open: ", f.Name)
		return types.UpdateFailed
	}
	statbuf, err := fd.Stat()
	if err != nil {
		perrorWithName("touch: fstat: ", f.Name)
		fd.Close()
		return types.UpdateFailed
	}
	buf := make([]byte, 1)
	if _, err := fd.Read(buf); err != nil {
		perrorWithName("touch: read: ", f.Name)
		fd.Close()
		return types.UpdateFailed
	}
	if _, err := fd.Seek(0, 0); err != nil {
		perrorWithName("touch: lseek: ", f.Name)
		fd.Close()
		return types.UpdateFailed
	}
	if _, err := fd.Write(buf); err != nil {
		perrorWithName("touch: write: ", f.Name)
		fd.Close()
		return types.UpdateFailed
	}
	if statbuf.Size() == 0 {
		fd.Close()
		fd, err = os.OpenFile(f.Name, os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			perrorWithName("touch: open: ", f.Name)
			return types.UpdateFailed
		}
	}
	fd.Close()
	return types.UpdateSuccess
}

func remakeFile(f *types.File) {
	if f.Cmds == nil {
		if f.Phony {
			f.UpdateStatus = types.UpdateSuccess
		} else if f.IsTarget {
			f.UpdateStatus = types.UpdateSuccess
		} else {
			if !config.RebuildingMakefiles || !f.Dontcare {
				complain(f)
			}
			f.UpdateStatus = types.UpdateFailed
		}
	} else {
		commands.ChopCommands(f.Cmds)
		if !config.TouchFlag || f.Cmds.AnyRecurse {
			commands.ExecuteFileCommands(f)
		} else {
			f.UpdateStatus = types.UpdateSuccess
		}
	}
	noticeFinishedFile(f)
}

func fMtime(f *types.File, search int) uint64 {
	mtime := uint64(nameMtime(f.Name))
	if mtime == uint64(config.NonexistentMtime) && search != 0 && !f.IgnoreVpath {
		var ts config.FileTimestamp
		name := vpath.VpathSearch(f.Name, &ts, nil, nil)
		if name == "" && len(f.Name) > 1 && f.Name[0] == '-' && f.Name[1] == 'l' {
			n, _ := librarySearch(f.Name, nil)
			if n != "" {
				name = n
			}
		}
		if name != "" {
			if mtime != uint64(config.UnknownMtime) {
				f.LastMtime = mtime
			}
			rehashFile(f, name)
			checkRenamed(f)
			if ts != config.OldMtime && ts != config.NewMtime {
				mtime = uint64(nameMtime(name))
			}
		}
	}

	if !config.ClockSkewDetected &&
		mtime != uint64(config.NonexistentMtime) &&
		mtime != uint64(config.NewMtime) &&
		!f.Updated {

		nowMs := time.Now().UnixNano()
		adjustedMtime := int64(mtime)
		if nowMs > adjustedMtime {
			fromNow := float64(adjustedMtime-nowMs) / 1e9
			if fromNow >= 100.0 {
				fromNowStr := strconv.FormatFloat(fromNow, 'g', 2, 64)
				errorMsg("Warning: File '%s' has modification time %s s in the future",
					f.Name, fromNowStr)
				config.ClockSkewDetected = true
			}
		}
	}

	if f.DoubleColon != nil {
		f = f.DoubleColon
	}

	propagateTimestamp := f.Updated
	for ff := f; ff != nil; ff = ff.Prev {
		if mtime != uint64(config.NonexistentMtime) &&
			ff.CommandState == types.CmdNotStarted &&
			!ff.TriedImplicit && ff.Intermediate {
			ff.Intermediate = false
		}
		if ff.Updated == propagateTimestamp {
			ff.LastMtime = mtime
		}
	}
	return mtime
}

func librarySearch(lib string, mtimePtr *config.FileTimestamp) (string, config.FileTimestamp) {
	dirs := []string{"/lib", "/usr/lib", "/usr/local/lib"}
	libpatterns := expand.VariableExpand("$(.LIBPATTERNS)")

	if len(lib) < 2 || lib[0] != '-' || lib[1] != 'l' {
		return "", config.NonexistentMtime
	}
	lib = lib[2:]

	var result string
	var mtime config.FileTimestamp
	bestVpath := ^uint(0)
	_ = bestVpath

	p2 := libpatterns
	for {
		lenInt := 0
		p := misc.FindNextToken(&p2, &lenInt)
		if p == "" {
			break
		}
		cp := strings.IndexByte(p, '%')
		if cp < 0 {
			continue
		}
		libbuf := p[:cp] + lib + p[cp+1:]

		mtime = nameMtime(libbuf)
		if mtime != config.NonexistentMtime {
			if mtimePtr != nil {
				*mtimePtr = mtime
			}
			result = strcache.Add(libbuf)
			break
		}

		var vpathIndex, pathIndex uint
		if f := vpath.VpathSearch(libbuf, &mtime, &vpathIndex, &pathIndex); f != "" {
			if result == "" || vpathIndex < bestVpath {
				result = f
				if mtimePtr != nil {
					*mtimePtr = mtime
				}
			}
		}

		for _, dir := range dirs {
			buf := dir + "/" + libbuf
			if m := nameMtime(buf); m != config.NonexistentMtime {
				if result == "" {
					result = strcache.Add(buf)
					if mtimePtr != nil {
						*mtimePtr = m
					}
				}
			}
		}
	}

	return result, mtime
}

func init() {
	commands.OnCommandExecuted = func() {
		CommandsStarted++
	}
}

var _ = job.StartWaitingJobs
var _ = job.ReapChildren
