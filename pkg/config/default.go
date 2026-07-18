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

package config

// Default variables and rules for GNU make

var DefaultTmpdir = "/tmp"

var DefaultVariables = map[string]string{
	"AR":            "ar",
	"ARFLAGS":       "rv",
	"AS":            "as",
	"ASFLAGS":       "",
	"CC":            "cc",
	"CFLAGS":        "",
	"CO":            "co",
	"COFLAGS":       "",
	"CPP":           "$(CC) -E",
	"CPPFLAGS":      "",
	"CXX":           "g++",
	"CXXFLAGS":      "",
	"FC":            "f77",
	"FFLAGS":        "",
	"GNUMAKEFLAGS":  "",
	"LDFLAGS":       "",
	"LEX":           "lex",
	"LFLAGS":        "",
	"YACC":          "yacc",
	"YFLAGS":        "",
	"MAKE":          "make",
	"MAKEINFO":      "makeinfo",
	"MAKEINFOFLAGS": "",
	"MAKECMDGOALS":  "",
	"MAKEFLAGS":     "",
	"MAKEFILES":     "",
	"MAKELEVEL":     "0",
	"MAKESHELL":     "/bin/sh",
	"RM":            "rm -f",
	"SHELL":         "/bin/sh",
	"TEXI2DVI":      "texi2dvi",
	"TEXI2PDF":      "texi2dvi --pdf",
}

// Default implicit rules
// These represent the built-in pattern rules from GNU make's default.c
var DefaultRules = []struct {
	Target     string
	Prereqs    string
	Commands   string
	Terminal   byte
}{
	// Compiling C programs
	{"%o", "%c", "$(CC) $(CFLAGS) $(CPPFLAGS) $(LDFLAGS) -o $@ $<", 0},
	// Compiling C++ programs
	{"%o", "%cc", "$(CXX) $(CXXFLAGS) $(CPPFLAGS) $(LDFLAGS) -o $@ $<", 0},
	// Compiling Pascal programs
	{"%o", "%p", "$(PC) $(PFLAGS) $(CPPFLAGS) -o $@ $<", 0},
	// Compiling Fortran programs
	{"%o", "%f", "$(FC) $(FFLAGS) -o $@ $<", 0},
	// Link single object file
	{"%", "%o", "$(CC) $(LDFLAGS) -o $@ $<", 0},
	// Yacc
	{"%c", "%y", "$(YACC) $(YFLAGS) $< && mv y.tab.c $@", 0},
	// Lex
	{"%c", "%l", "$(LEX) $(LFLAGS) $< && mv lex.yy.c $@", 0},
	// Archive
	{"%a", "%.o", "$(AR) $(ARFLAGS) $@ $%", 0},
	// Lint
	{"%ln", "%c", "$(LINT) $(LINTFLAGS) $(CPPFLAGS) -o $@ $<", 0},
}
