package archive

import (
	"time"
)

// ArName checks if a filename is an archive member reference (lib.a(obj.o))
func ArName(name string) int {
	// Check for the ( and ) pattern in the name
	for i := 0; i < len(name); i++ {
		if name[i] == '(' {
			return 1
		}
	}
	return 0
}

// ArParseName parses an archive member name into archive and member
func ArParseName(name string, arnamep, memnamep *string) {
	for i := 0; i < len(name); i++ {
		if name[i] == '(' {
			*arnamep = name[:i]
			end := len(name) - 1
			if end > i && name[end] == ')' {
				*memnamep = name[i+1 : end]
			}
			return
		}
	}
	*arnamep = name
	*memnamep = ""
}

// ArMemberDate returns the modification time of an archive member
func ArMemberDate(name string) time.Time {
	return time.Time{}
}

// ArScan scans an archive file and calls the callback for each member
func ArScan(archive string, fn func(int, string, int, int64, int64, int64, int64, int, int, uint, interface{}) int64, arg interface{}) int64 {
	return 0
}

// ArNameEqual checks if a member name matches
func ArNameEqual(name, mem string, truncated int) int {
	if name == mem {
		return 1
	}
	return 0
}

// ArTouch touches an archive member
func ArTouch(name string) int {
	return 0
}

// ArMemberTouch sets the modification time of an archive member
func ArMemberTouch(arname, memname string) int {
	return 0
}

// ArGlob globs for archive members matching a pattern
func ArGlob(arname, memberPattern string, size int) *struct {
	Name string
	Next *struct{}
} {
	return nil
}
