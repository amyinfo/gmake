package strcache

import (
	"sync"
)

var (
	mu       sync.Mutex
	strings  = make(map[string]string)
	hits     uint64
	misses   uint64
	adds     uint64
)

func Init() {
	mu.Lock()
	defer mu.Unlock()
	strings = make(map[string]string)
	hits = 0
	misses = 0
	adds = 0
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
		hits++
		return cached
	}
	misses++
	adds++
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
