package commands

import (
	"os"
	"os/exec"
	"strings"

	"github.com/amyinfo/gmake/pkg/expand"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
)

// OnCommandExecuted is called after each command line finishes execution.
// Set by the remake package to track CommandsStarted.
var OnCommandExecuted func()

const (
	CommandsRecur   = 1
	CommandsSilent  = 2
	CommandsNoError = 4
)

func SetFileVariables(f *types.File, stem string) {
	at := f.Name
	percent := ""

	if stem == "" {
		stem = f.Stem
	}
	star := stem

	less := ""
	for d := f.Deps; d != nil; d = d.Next {
		if !d.IgnoreMtime && !d.IgnoreAutomaticVars && !d.Need2ndExpansion {
			less = d.Name()
			break
		}
	}

	define := func(name, value string) {
		variable.DefineVariableForFile(name, len(name), value, types.OriginAutomatic, false, f)
	}

	define("<", less)
	define("*", star)
	define("@", at)
	define("%", percent)

	dedup := make(map[string]bool)
	caretParts := []string{}
	plusParts := []string{}
	qmarkParts := []string{}
	barParts := []string{}

	for d := f.Deps; d != nil; d = d.Next {
		if d.Need2ndExpansion || d.IgnoreAutomaticVars {
			continue
		}
		dn := d.Name()
		if dn == "" {
			continue
		}
		if dedup[dn] {
			continue
		}
		dedup[dn] = true

		if d.IgnoreMtime {
			barParts = append(barParts, dn)
		} else {
			plusParts = append(plusParts, dn)
			caretParts = append(caretParts, dn)
			if d.Changed {
				qmarkParts = append(qmarkParts, dn)
			}
		}
	}

	define("+", strings.Join(plusParts, " "))
	define("^", strings.Join(caretParts, " "))
	define("?", strings.Join(qmarkParts, " "))
	define("|", strings.Join(barParts, " "))
}

func ChopCommands(cmds *types.Commands) {
	if cmds == nil || cmds.CommandLines != nil {
		return
	}

	lines := splitLines(cmds.Commands)
	cmds.NCommandLines = uint16(len(lines))
	cmds.CommandLines = lines
	cmds.LinesFlags = make([]byte, len(lines))
	cmds.AnyRecurse = false

	for i, line := range lines {
		flags := byte(0)
		trimmed := line
		for {
			if trimmed == "" {
				break
			}
			ch := trimmed[0]
			switch ch {
			case '+':
				flags |= CommandsRecur
				trimmed = trimmed[1:]
			case '@':
				flags |= CommandsSilent
				trimmed = trimmed[1:]
			case '-':
				flags |= CommandsNoError
				trimmed = trimmed[1:]
			case ' ', '\t':
				trimmed = trimmed[1:]
			default:
				break
			}
			if ch != ' ' && ch != '\t' && ch != '+' && ch != '@' && ch != '-' {
				break
			}
		}

		if flags&CommandsRecur == 0 {
			if strings.Contains(trimmed, "$(MAKE)") || strings.Contains(trimmed, "${MAKE}") {
				flags |= CommandsRecur
			}
		}

		cmds.LinesFlags[i] = flags
		lines[i] = trimmed
		if flags&CommandsRecur != 0 {
			cmds.AnyRecurse = true
		}
	}
}

func splitLines(commands string) []string {
	if commands == "" {
		return nil
	}

	var lines []string
	p := commands
	for p != "" {
		end := strings.IndexByte(p, '\n')
		if end < 0 {
			lines = append(lines, p)
			break
		}

		if end > 0 && p[end-1] == '\\' {
			backslash := 1
			for b := end - 2; b >= 0 && p[b] == '\\'; b-- {
				backslash ^= 1
			}
			if backslash == 1 {
				end++
				for end < len(p) && p[end] != '\n' {
					end++
				}
				if end >= len(p) {
					lines = append(lines, p)
					break
				}
				lines = append(lines, p[:end])
				p = p[end+1:]
				continue
			}
		}

		lines = append(lines, p[:end])
		p = p[end+1:]
	}

	for i := range lines {
		if len(lines[i]) > 0 && lines[i][len(lines[i])-1] == '\r' {
			lines[i] = lines[i][:len(lines[i])-1]
		}
	}

	return lines
}

func ExecuteFileCommands(f *types.File) int {
	if f.Cmds == nil {
		file.SetCommandState(f, types.CmdRunning)
		f.UpdateStatus = types.UpdateSuccess
		NoticeFinishedFile(f)
		return 0
	}

	isEmpty := true
	for _, c := range f.Cmds.Commands {
		if c != ' ' && c != '\t' && c != '\n' && c != '-' && c != '@' && c != '+' {
			isEmpty = false
			break
		}
	}
	if isEmpty {
		file.SetCommandState(f, types.CmdRunning)
		f.UpdateStatus = types.UpdateSuccess
		NoticeFinishedFile(f)
		return 0
	}

	variable.InitializeFileVariables(f, 0)
	SetFileVariables(f, f.Stem)

	ChopCommands(f.Cmds)

	for i, line := range f.Cmds.CommandLines {
		expanded := expand.VariableExpand(line)
		flags := f.Cmds.LinesFlags[i]

		showCmd := flags&CommandsSilent == 0

		if showCmd && expanded != "" {
			os.Stdout.WriteString(expanded)
			os.Stdout.WriteString("\n")
		}

		cmd := exec.Command("sh", "-c", expanded)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if flags&CommandsNoError == 0 {
				f.UpdateStatus = types.UpdateFailed
				return 1
			}
		}
		if OnCommandExecuted != nil {
			OnCommandExecuted()
		}
	}

	file.SetCommandState(f, types.CmdRunning)
	f.UpdateStatus = types.UpdateSuccess
	NoticeFinishedFile(f)
	return 0
}

func NoticeFinishedFile(file *types.File) {
}
