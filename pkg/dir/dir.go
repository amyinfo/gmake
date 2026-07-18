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

package dir

import (
	"fmt"
	"os"
	"path/filepath"
)

var directories = make(map[string]bool)

func HashInitDirectories() {
	directories = make(map[string]bool)
}

func DirFileExistsP(dir, file string) int {
	// Check if file exists in directory
	fullPath := filepath.Join(dir, file)
	if _, err := os.Stat(fullPath); err == nil {
		return 1
	}
	return 0
}

func FileExistsP(name string) int {
	if _, err := os.Stat(name); err == nil {
		return 1
	}
	return 0
}

func FileImpossibleP(name string) int {
	// Check if this file was previously determined impossible to find
	// In original GNU make, this uses a hash table of "impossible" names
	return 0
}

func FileImpossible(name string) {
	// Mark a file as impossible to find
	// (not implemented in stub)
}

func DirName(name string) string {
	// Return the directory portion of a filename
	dir := filepath.Dir(name)
	if dir == "." {
		return ""
	}
	return dir + string(filepath.Separator)
}

func PrintDirDataBase() {
	fmt.Fprintf(os.Stderr, "# Directories: %d\n", len(directories))
}

func DirSetupGlob(g *Glob) {
	// Setup glob structure
	if g == nil {
		return
	}
}

// Glob structure wrapper
type Glob struct {
	GlobPathc  int
	GlobPathv  []string
	GlobOffs   int
	GlobFlags  int
}

const (
	GlobErr    = 1 << 0
	GlobMark   = 1 << 1
	GlobNosort = 1 << 2
	GlobNoCheck = 1 << 3
	GlobAlias  = 1 << 4
)

func GlobFunc(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
