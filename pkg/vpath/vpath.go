package vpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyra/make/pkg/config"
	"github.com/kyra/make/pkg/strcache"
)

type vpathEntry struct {
	Pattern string
	Paths   []string
}

var (
	vpaths    []vpathEntry
	envVpaths []string
)

func BuildVpathLists() {
	// Build VPATH lists from .VPATH variable and environment
	vpaths = nil
	if envPath := os.Getenv("VPATH"); envPath != "" {
		envVpaths = append(envVpaths, strings.Fields(envPath)...)
	}
}

func ConstructVpathList(pattern string, dirpath string) {
	if dirpath == "" {
		// Remove pattern
		for i, v := range vpaths {
			if v.Pattern == pattern {
				vpaths = append(vpaths[:i], vpaths[i+1:]...)
				return
			}
		}
		return
	}

	dirs := strings.Fields(dirpath)
	entry := vpathEntry{
		Pattern: strcache.Add(pattern),
		Paths:   make([]string, len(dirs)),
	}
	for i, d := range dirs {
		entry.Paths[i] = strcache.Add(d)
	}

	// Replace or append
	for i, v := range vpaths {
		if v.Pattern == pattern {
			vpaths[i] = entry
			return
		}
	}
	vpaths = append(vpaths, entry)
}

func VpathSearch(file string, mtimePtr *config.FileTimestamp, vpathIdx, pathIdx *uint) string {
	// Search VPATH for the given file
	// Returns the full path if found, empty string otherwise

	// First check if file exists as-is
	if _, err := os.Stat(file); err == nil {
		return ""
	}

	// Try each VPATH directory
	searchDirs := vpaths
	if len(searchDirs) == 0 {
		searchDirs = []vpathEntry{{Pattern: "%", Paths: envVpaths}}
	}

	for _, vp := range searchDirs {
		for _, dir := range vp.Paths {
			fullPath := filepath.Join(dir, file)
			if fi, err := os.Stat(fullPath); err == nil {
				if mtimePtr != nil {
					*mtimePtr = config.FileTimestamp(fi.ModTime().Unix())
				}
				return fullPath
			}
		}
	}

	return ""
}

func GpathSearch(file string, length int) int {
	// GPATH search for the file
	_ = length
	return 0
}

func PrintVpathDataBase() {
	for _, vp := range vpaths {
		fmt.Fprintf(os.Stdout, "# VPATH %s: %s\n", vp.Pattern, strings.Join(vp.Paths, " "))
	}
}
