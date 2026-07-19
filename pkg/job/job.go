package job

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/amyinfo/gmake/pkg/commands"
	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/debug"
	"github.com/amyinfo/gmake/pkg/expand"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/output"
	"github.com/amyinfo/gmake/pkg/shuffle"
	"github.com/amyinfo/gmake/pkg/signame"
	"github.com/amyinfo/gmake/pkg/types"
)

var (
	children        *types.Child
	jobSlotsUsed    uint
	goodStdinUsed   bool
	waitingJobs     *types.Child
	jobCounter   uint64
	deadChildren uint

	childrenMu  sync.Mutex
	childExited = make(chan struct{}, 16)
)

func childOutputPtr(typ *types.Output) *output.Output {
	return &output.Output{Out: typ.Out, Err: typ.Err, Syncout: typ.SyncOut}
}

func syncOutputFields(typ *types.Output, out *output.Output) {
	typ.Out = out.Out
	typ.Err = out.Err
	typ.SyncOut = out.Syncout
}

func initChildOutput(typ *types.Output) {
	out := childOutputPtr(typ)
	output.OutputInit(out)
	syncOutputFields(typ, out)
}

func closeChildOutput(typ *types.Output) {
	out := childOutputPtr(typ)
	output.OutputClose(out)
	syncOutputFields(typ, out)
}

func dumpChildOutput(typ *types.Output) {
	out := childOutputPtr(typ)
	output.OutputDump(out)
}

func pid2str(pid int) string {
	return fmt.Sprintf("%d", pid)
}

func ChildError(child *types.Child, exitCode, exitSig, coredump int, ignored bool) {
	if ignored && config.RunSilent {
		return
	}
	pre := "*** "
	post := ""
	dump := ""
	f := child.File
	flocp := &f.Cmds.Fileinfo
	if exitSig != 0 && coredump != 0 {
		dump = " (core dumped)"
	}
	if ignored {
		pre = ""
		post = " (ignored)"
	}
	var nm string
	if flocp.Filenm == "" {
		nm = "<builtin>"
	} else {
		nm = fmt.Sprintf("%s:%d", flocp.Filenm, flocp.Lineno+flocp.Offset)
	}
	smode := shuffle.GetMode()
	shuffleExtra := ""
	if smode != "" {
		shuffleExtra = " shuffle=" + smode
	}
	out := childOutputPtr(&child.Output)
	if !output.IsOutputSet(out) {
		initChildOutput(&child.Output)
	}
	output.OutputSet(out)
	if exitSig == 0 {
		fmt.Fprintf(os.Stderr, "%s[%s: %s] Error %d%s%s\n", pre, nm, f.Name, exitCode, post, shuffleExtra)
	} else {
		s := signame.SignalName(exitSig)
		fmt.Fprintf(os.Stderr, "%s[%s: %s] %s%s%s%s\n", pre, nm, f.Name, s, dump, post, shuffleExtra)
	}
	output.OutputUnset()
}

func ReapChildren(block int, err int) {
	reapMore := true
	for (children != nil) && (block != 0 || reapMore) {
		if err != 0 && block != 0 {
			fmt.Fprintf(os.Stdout, "*** Waiting for unfinished jobs....\n")
		}
		func() {
			childrenMu.Lock()
			if deadChildren > 0 {
				deadChildren--
			}
			childrenMu.Unlock()
		}()
		var lastc *types.Child
		for c := children; c != nil; c = c.Next {
			if c.Pid < 0 {
				processChild(c, 0, 0, 127, 0, lastc, err)
				continue
			}
			lastc = c
		}
		var reaped bool
		func() {
			childrenMu.Lock()
			defer childrenMu.Unlock()
			for c := children; c != nil; c = c.Next {
				if c.Cmd == nil || c.Cmd.ProcessState != nil {
					continue
				}
				select {
				case <-c.Exited:
					if c.Cmd.ProcessState != nil {
						reaped = true
					}
				default:
				}
			}
		}()
		if !reaped {
			if block != 0 {
				select {
				case <-childExited:
					reaped = true
				case <-time.After(100 * time.Millisecond):
				}
			} else {
				select {
				case <-childExited:
					reaped = true
				default:
					reapMore = false
					if block == 0 {
						break
					}
				}
			}
		}
		childrenMu.Lock()
		c := children
		var prev *types.Child
		for c != nil {
			if c.Cmd != nil && c.Cmd.ProcessState != nil && !c.Processed {
				ws := c.Cmd.ProcessState
				exitCode := ws.ExitCode()
				exitSig, coredump := getExitSignalInfo(ws)
				config.CommandCount++
				c.Processed = true
				next := c.Next
				if prev == nil {
					children = next
				} else {
					prev.Next = next
				}
				childrenMu.Unlock()
				if jobCounter > 0 {
					jobCounter--
				}
				processChild(c, exitCode, exitSig, 0, coredump, prev, err)
				childrenMu.Lock()
				c = next
			} else {
				prev = c
				c = c.Next
			}
		}
		childrenMu.Unlock()
		block = 0
	}
}

func processChild(c *types.Child, exitCode, exitSig, forceExitCode, coredump int, lastc *types.Child, err int) {
	if forceExitCode != 0 {
		exitCode = forceExitCode
	}
	childFailed := config.MakeSuccess
	if exitSig == 0 && exitCode == 0 {
		childFailed = config.MakeSuccess
	} else if exitSig == 0 && exitCode == 1 && config.QuestionFlag && c.Recursive {
		childFailed = config.MakeTrouble
	} else {
		childFailed = config.MakeFailure
	}
	if c.ShBatchFile != "" {
		debug.DBF(config.DbJobs, "Cleaning up temp batch file %s\n", c.ShBatchFile)
		os.Remove(c.ShBatchFile)
		c.ShBatchFile = ""
	}
	if c.GoodStdin {
		goodStdinUsed = false
	}
	dontcare := c.Dontcare
	if childFailed != config.MakeSuccess && !c.Noerror && !config.IgnoreErrorsFlag {
		if !dontcare && childFailed == config.MakeFailure {
			ChildError(c, exitCode, exitSig, coredump, false)
		}
		if childFailed == config.MakeFailure {
			c.File.UpdateStatus = types.UpdateFailed
		} else {
			c.File.UpdateStatus = types.UpdateQuestion
		}
		deleteOnError := false
		f := file.LookupFile(".DELETE_ON_ERROR")
		if f != nil && f.IsTarget {
			deleteOnError = true
		}
		if exitSig != 0 || deleteOnError {
			deleteChildTargets(c)
		}
	} else {
		if childFailed != config.MakeSuccess {
			ChildError(c, exitCode, exitSig, coredump, true)
			childFailed = config.MakeSuccess
		}
		if jobNextCommand(c) {
			startJobCommand(c)
			if c.File.CommandState == types.CmdRunning {
				childrenMu.Lock()
				if lastc == nil {
					c.Next = children
					children = c
				} else {
					c.Next = lastc.Next
					lastc.Next = c
				}
				childrenMu.Unlock()
				return
			}
			if c.File.UpdateStatus != types.UpdateSuccess {
				deleteChildTargets(c)
			}
		} else {
			c.File.UpdateStatus = types.UpdateSuccess
		}
	}
	dumpChildOutput(&c.Output)
	commands.NoticeFinishedFile(c.File)
	if c.Jobslot {
		childrenMu.Lock()
		if jobSlotsUsed > 0 {
			jobSlotsUsed--
		}
		childrenMu.Unlock()
	}
	freeChild(c)
	if childFailed != config.MakeSuccess && !dontcare && !config.KeepGoingFlag {
		fmt.Fprintf(os.Stderr, "%s: *** [%s] Error %d\n", config.Program, c.File.Name, exitCode)
	}
}

func FreeChildbase(child *types.ChildBase) {
	child.CmdName = ""
	child.Environment = nil
}

func freeChild(child *types.Child) {
	closeChildOutput(&child.Output)
	if child.CmdLines != nil {
		child.CmdLines = nil
	}
	FreeChildbase(&child.ChildBase)
}

func startJobCommand(child *types.Child) {
	flags := child.File.CommandFlags
	if child.CmdLine > 0 {
		flags |= int(child.File.Cmds.LinesFlags[child.CmdLine-1])
	}
	p := child.CmdPtr
	if p == "" {
		jobNextCommand(child)
		if child.CmdPtr == "" {
			if jobNextCommand(child) {
				startJobCommand(child)
			} else {
				file.SetCommandState(child.File, types.CmdRunning)
				child.File.UpdateStatus = types.UpdateSuccess
				commands.NoticeFinishedFile(child.File)
			}
			return
		}
		p = child.CmdPtr
	}
	child.Noerror = (flags & commands.CommandsNoError) != 0
	i := 0
	for i < len(p) {
		switch p[i] {
		case '@':
			flags |= commands.CommandsSilent
		case '+':
			flags |= commands.CommandsRecur
		case '-':
			child.Noerror = true
		default:
			if p[i] != ' ' && p[i] != '\t' {
				goto doneParsing
			}
		}
		i++
	}
doneParsing:
	child.Recursive = (flags & commands.CommandsRecur) != 0
	if child.CmdLine > 0 {
		child.File.Cmds.LinesFlags[child.CmdLine-1] |= byte(flags & commands.CommandsRecur)
	}
	if child.File.Cmds.RecipePrefix != 0 {
		prefix := child.File.Cmds.RecipePrefix
		var sb strings.Builder
		j := i
		for j < len(p) {
			sb.WriteByte(p[j])
			if p[j] == '\n' && j+1 < len(p) && p[j+1] == byte(prefix) {
				j++
			}
			j++
		}
		p = sb.String()
		i = 0
	}
	var end int
	var argv []string
	if config.OneShell {
		argv = []string{config.DefaultShell, "-c", p[i:]}
		end = len(p)
	} else {
		argv, end = constructCommandArgv(p[i:])
	}
	if end == 0 || end >= len(p[i:]) {
		child.CmdPtr = ""
	} else {
		child.CmdPtr = p[i+end:]
	}
	if argv != nil && config.QuestionFlag && (flags&commands.CommandsRecur) == 0 {
		child.File.UpdateStatus = types.UpdateQuestion
		commands.NoticeFinishedFile(child.File)
		return
	}
	if config.TouchFlag && (flags&commands.CommandsRecur) == 0 {
		jobNextCommand(child)
		if child.CmdPtr != "" {
			startJobCommand(child)
		} else {
			file.SetCommandState(child.File, types.CmdRunning)
			child.File.UpdateStatus = types.UpdateSuccess
			commands.NoticeFinishedFile(child.File)
		}
		return
	}
	if argv == nil {
		if jobNextCommand(child) {
			startJobCommand(child)
		} else {
			file.SetCommandState(child.File, types.CmdRunning)
			child.File.UpdateStatus = types.UpdateSuccess
			commands.NoticeFinishedFile(child.File)
		}
		return
	}
	child.Output.SyncOut = config.OutputSync != config.OutputSyncNone &&
		(config.OutputSync == config.OutputSyncRecurse || (flags&commands.CommandsRecur) == 0)
	out := childOutputPtr(&child.Output)
	output.OutputSet(out)
	if !child.Output.SyncOut {
		dumpChildOutput(&child.Output)
	}
	if config.JustPrintFlag || config.IsDb(config.DbPrint) ||
		((flags&commands.CommandsSilent) == 0 && !config.RunSilent) {
		fmt.Fprintf(os.Stdout, "%s\n", p[i:])
	}
	config.CommandsStarted++
	if config.JustPrintFlag && (flags&commands.CommandsRecur) == 0 {
		if jobNextCommand(child) {
			startJobCommand(child)
		} else {
			file.SetCommandState(child.File, types.CmdRunning)
			child.File.UpdateStatus = types.UpdateSuccess
			commands.NoticeFinishedFile(child.File)
		}
		output.OutputUnset()
		return
	}
	output.OutputStart()
	os.Stdout.Sync()
	os.Stderr.Sync()
	child.GoodStdin = !goodStdinUsed
	if child.GoodStdin {
		goodStdinUsed = true
	}
	child.Deleted = false
	if child.Environment == nil {
		child.Environment = targetEnvironment(child.File)
	}
	childExecuteJob(child, argv)
	jobCounter++
	file.SetCommandState(child.File, types.CmdRunning)
	output.OutputUnset()
}

func childExecuteJob(child *types.Child, argv []string) {
	if len(argv) == 0 {
		return
	}
	var cmd *exec.Cmd
	if len(argv) == 1 {
		cmd = exec.Command(config.DefaultShell, "-c", argv[0])
	} else {
		cmd = exec.Command(argv[0], argv[1:]...)
	}
	if child.Environment != nil {
		cmd.Env = child.Environment
	}
	if !child.GoodStdin {
		devNull, err := os.Open("/dev/null")
		if err == nil {
			cmd.Stdin = devNull
		}
	}
	if child.Output.Out >= 0 {
		cmd.Stdout = os.NewFile(uintptr(child.Output.Out), "output-sync-out")
	}
	if child.Output.Err >= 0 {
		cmd.Stderr = os.NewFile(uintptr(child.Output.Err), "output-sync-err")
	}
	child.Cmd = cmd
	err := cmd.Start()
	if err != nil {
		if argv[0] != "" {
			fmt.Fprintf(os.Stderr, "%s: %s\n", argv[0], err.Error())
		}
		child.Pid = -1
		return
	}
	child.Pid = cmd.Process.Pid
	go func(c *types.Child) {
		c.Cmd.Wait()
		childrenMu.Lock()
		deadChildren++
		childrenMu.Unlock()
		select {
		case childExited <- struct{}{}:
		default:
		}
	}(child)
}

func startWaitingJob(c *types.Child) int {
	f := c.File
	if jobSlotsUsed > 0 && loadTooHigh() {
		file.SetCommandState(f, types.CmdRunning)
		childrenMu.Lock()
		c.Next = waitingJobs
		waitingJobs = c
		childrenMu.Unlock()
		return 0
	}
	startJobCommand(c)
	switch f.CommandState {
	case types.CmdRunning:
		childrenMu.Lock()
		c.Next = children
		if c.Pid > 0 {
			debug.DB(config.DbJobs, "Putting child %p (%s) PID %s on the chain.\n",
				c, c.File.Name, pid2str(c.Pid))
			jobSlotsUsed++
			c.Jobslot = true
		}
		children = c
		childrenMu.Unlock()
		return 1
	case types.CmdNotStarted:
		f.UpdateStatus = types.UpdateSuccess
		fallthrough
	case types.CmdFinished:
		commands.NoticeFinishedFile(f)
		freeChild(c)
		return 1
	default:
		return 1
	}
}

func NewJob(f *types.File) {
	cmds := f.Cmds
	StartWaitingJobs()
	ReapChildren(0, 0)
	commands.ChopCommands(cmds)
	c := &types.Child{}
	initChildOutput(&c.Output)
	c.File = f
	c.Dontcare = f.Dontcare
	out := childOutputPtr(&c.Output)
	output.OutputSet(out)
	lines := make([]string, cmds.NCommandLines)
	for i := uint16(0); i < cmds.NCommandLines; i++ {
		cmds.Fileinfo.Offset = uint64(i)
		line := collapseBackslashNewline(cmds.CommandLines[i])
		lines[i] = expand.VariableExpandForFile(line, f)
	}
	cmds.Fileinfo.Offset = 0
	c.CmdLines = lines
	jobNextCommand(c)
	setupChildHandler()
	if config.JobSlots != 0 {
		for jobSlotsUsed == config.JobSlots {
			ReapChildren(1, 0)
		}
	}
	if config.IsDb(config.DbWhy) {
		nm := "<builtin>"
		if cmds.Fileinfo.Filenm != "" {
			nm = fmt.Sprintf("%s:%d", cmds.Fileinfo.Filenm, cmds.Fileinfo.Lineno)
		}
		fmt.Fprintf(os.Stdout, "%s: update target '%s' due to: %s\n", nm, c.File.Name,
			expand.VariableExpandForFile("$?", c.File))
	}
	startWaitingJob(c)
	if config.JobSlots == 1 || config.NotParallel {
		for f.CommandState == types.CmdRunning {
			ReapChildren(1, 0)
		}
	}
	output.OutputUnset()
}

func jobNextCommand(child *types.Child) bool {
	for child.CmdPtr == "" {
		if child.CmdLine >= uint(len(child.CmdLines)) {
			child.CmdPtr = ""
			if child.File.Cmds != nil {
				child.File.Cmds.Fileinfo.Offset = 0
			}
			return false
		}
		child.CmdPtr = child.CmdLines[child.CmdLine]
		child.CmdLine++
	}
	if child.File.Cmds != nil {
		child.File.Cmds.Fileinfo.Offset = uint64(child.CmdLine - 1)
	}
	return true
}

func loadTooHigh() bool {
	return config.MaxLoadAverage >= 0
}

func StartWaitingJobs() {
	if waitingJobs == nil {
		return
	}
	for {
		ReapChildren(0, 0)
		childrenMu.Lock()
		job := waitingJobs
		if job == nil {
			childrenMu.Unlock()
			break
		}
		waitingJobs = job.Next
		job.Next = nil
		childrenMu.Unlock()
		if startWaitingJob(job) == 0 {
			break
		}
		childrenMu.Lock()
		if waitingJobs == nil {
			childrenMu.Unlock()
			break
		}
		childrenMu.Unlock()
	}
}

func constructCommandArgv(cmdline string) ([]string, int) {
	trimmed := strings.TrimSpace(cmdline)
	if trimmed == "" {
		return nil, 0
	}
	return []string{config.DefaultShell, "-c", trimmed}, len(cmdline)
}

func collapseBackslashNewline(line string) string {
	var out strings.Builder
	in := line
	for {
		ref := strings.IndexByte(in, '$')
		if ref < 0 {
			out.WriteString(in)
			break
		}
		out.WriteString(in[:ref])
		in = in[ref+1:]
		if len(in) == 0 {
			out.WriteByte('$')
			break
		}
		ch := in[0]
		in = in[1:]
		if ch == '$' {
			out.WriteString("$$")
			continue
		}
		if ch == '(' || ch == '{' {
			closeparen := byte(')')
			if ch == '{' {
				closeparen = '}'
			}
			out.WriteByte('$')
			out.WriteByte(ch)
			count := 0
			for len(in) > 0 {
				c := in[0]
				in = in[1:]
				if c == '\\' && len(in) > 0 && in[0] == '\n' {
					in = in[1:]
					for len(in) > 0 && (in[0] == ' ' || in[0] == '\t') {
						in = in[1:]
					}
					continue
				}
				if c == closeparen {
					if count == 0 {
						out.WriteByte(c)
						break
					}
					count--
				} else if c == ch {
					count++
				}
				out.WriteByte(c)
			}
		} else {
			out.WriteByte('$')
			out.WriteByte(ch)
		}
	}
	return out.String()
}

func targetEnvironment(f *types.File) []string {
	return os.Environ()
}

func deleteChildTargets(c *types.Child) {
	if c.Deleted {
		return
	}
	c.Deleted = true
	f := c.File
	if f != nil && !f.Precious {
		f.UpdateStatus = types.UpdateFailed
	}
}
