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

package dep

import (
	"github.com/amyinfo/gmake/pkg/types"
)

func AllocDep() *types.Dep {
	return &types.Dep{}
}

func FreeDepChain(d *types.Dep) {
	for d != nil {
		next := d.Next
		d.Next = nil
		d = next
	}
}

func CopyDepChain(d *types.Dep) *types.Dep {
	var head, tail *types.Dep
	for ; d != nil; d = d.Next {
		n := AllocDep()
		n.Name_ = d.Name_
		n.File = d.File
		n.Flags = d.Flags
		n.Changed = d.Changed
		n.IgnoreMtime = d.IgnoreMtime
		n.Staticpattern = d.Staticpattern
		n.Need2ndExpansion = d.Need2ndExpansion
		n.IgnoreAutomaticVars = d.IgnoreAutomaticVars
		n.IsExplicit = d.IsExplicit
		n.WaitHere = d.WaitHere
		n.Stem = d.Stem
		if head == nil {
			head = n
			tail = n
		} else {
			tail.Next = n
			tail = n
		}
	}
	return head
}

func DepName(d *types.Dep) string {
	return d.Name()
}

func FreeDep(d *types.Dep) {
	d.Next = nil
}

func PrintDeps(d *types.Dep) {
	for ; d != nil; d = d.Next {
		println(" ", DepName(d))
	}
}

func ParseSimpleSeq(ptr *string) *types.Dep {
	s := *ptr
	if s == "" || s == " " {
		return nil
	}
	d := AllocDep()
	d.Name_ = s
	*ptr = ""
	return d
}

func ParseFileSeq(ptr *string, flags int, dir string, parseFlags int) *types.Dep {
	s := *ptr
	if s == "" {
		return nil
	}
	d := AllocDep()
	d.Name_ = s
	*ptr = ""
	return d
}
