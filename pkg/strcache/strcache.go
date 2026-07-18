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

package strcache

import (
	"sync"
)

var (
	mu      sync.Mutex
	strings = make(map[string]string)
)

func Init() {
	mu.Lock()
	defer mu.Unlock()
	strings = make(map[string]string)
}

func PrintStats(prefix string) {
	// Stats printing for maintainer mode
}

func Iscached(str string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := strings[str]
	return ok
}

func Add(str string) string {
	if str == "" {
		return ""
	}
	mu.Lock()
	defer mu.Unlock()
	if cached, ok := strings[str]; ok {
		return cached
	}
	cached := str
	strings[cached] = cached
	return cached
}

func AddLen(str string, length int) string {
	if length == 0 {
		return ""
	}
	// If len is less than actual length, take substring
	if length < len(str) {
		str = str[:length]
	}
	return Add(str)
}
