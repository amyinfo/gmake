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

package getopt

import (
	"strings"
)

// Option represents a command-line option definition
type Option struct {
	Name    string // long option name
	HasArg  int    // 0=no, 1=required, 2=optional
	Flag    *int   // if non-nil, *Flag = Val on match
	Val     int    // value to return or set *Flag to
	Short   byte   // short option character (or 0)
}

const (
	NoArg      = 0
	RequiredArg = 1
	OptionalArg = 2
)

var (
	Optind     int = 1 // index into argv
	Optopt     byte    // option character found
	Optarg     string  // argument for option
	Opterr     int = 1 // error messages
)

type GetoptState struct {
	Optind   int
	Optopt   byte
	Optarg   string
	Optpos   int // position within current arg
	Options  []Option
	Order    int
	Argv     []string
}

const (
	_ = iota
	RequireOrder
	Permute
	ReturnInOrder
)

func GetoptLongOnly(state *GetoptState) int {
	return getoptInternal(state, false)
}

func GetoptLong(state *GetoptState) int {
	return getoptInternal(state, true)
}

func getoptInternal(state *GetoptState, longOnly bool) int {
RETRY:
	// No more options
	if state.Optind >= len(state.Argv) {
		return -1
	}

	arg := state.Argv[state.Optind]

	// Not an option
	if len(arg) < 2 || arg[0] != '-' {
		if state.Order == RequireOrder {
			return -1
		}
		state.Optind++
		goto RETRY
	}

	// End of options marker
	if arg == "--" {
		state.Optind++
		return -1
	}

	// Long option: --option or -option (longOnly)
	if len(arg) > 2 && arg[1] == '-' || longOnly && len(arg) > 1 {
		prefix := arg[1:]
		if arg[1] == '-' {
			prefix = arg[2:]
		}

		// Parse --option=value
		name := prefix
		eqIdx := strings.IndexByte(prefix, '=')
		if eqIdx >= 0 {
			name = prefix[:eqIdx]
		}

		for _, opt := range state.Options {
			if opt.Name == "" {
				continue
			}
			if opt.Name != name {
				continue
			}

			if eqIdx >= 0 {
				// --option=value form
				if opt.HasArg == NoArg {
					// No argument expected
					return '?'
				}
				state.Optarg = prefix[eqIdx+1:]
			} else if opt.HasArg == RequiredArg {
				// Next arg is the value
				state.Optind++
				if state.Optind >= len(state.Argv) {
					return '?'
				}
				state.Optarg = state.Argv[state.Optind]
			}

			state.Optind++

			if opt.Flag != nil {
				*(opt.Flag) = opt.Val
				return 0
			}
			return opt.Val
		}

		// Unknown long option
		state.Optind++
		return '?'
	}

	// Short option: -x
	optchar := arg[1]
	state.Optopt = optchar
	state.Optpos = 2

	// Find the option definition
	for _, opt := range state.Options {
		if opt.Short == optchar {
			if opt.HasArg == RequiredArg {
				if state.Optpos < len(arg) {
					// -xVALUE form
					state.Optarg = arg[state.Optpos:]
				} else {
					state.Optind++
					if state.Optind >= len(state.Argv) {
						return '?'
					}
					state.Optarg = state.Argv[state.Optind]
				}
			}

			state.Optind++
			if opt.Flag != nil {
				*(opt.Flag) = opt.Val
				return 0
			}
			return opt.Val
		}
	}

	// Unknown short option
	state.Optind++
	return '?'
}
