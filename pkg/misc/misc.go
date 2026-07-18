package misc

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kyra/make/pkg/config"
)

var stopcharMap [256]uint16

func init() {
	for i := 0; i < 256; i++ {
		ch := byte(i)
		switch ch {
		case 0:
			stopcharMap[i] = config.MapNul
		case ' ', '\t':
			stopcharMap[i] = config.MapBlank
		case '\n':
			stopcharMap[i] = config.MapNewline
		case '#':
			stopcharMap[i] = config.MapComment
		case ';':
			stopcharMap[i] = config.MapSemi
		case '=':
			stopcharMap[i] = config.MapEquals
		case ':':
			stopcharMap[i] = config.MapColon
		case '$':
			stopcharMap[i] = config.MapVariable
		case '|':
			stopcharMap[i] = config.MapPipe
		case '.':
			stopcharMap[i] = config.MapDot
		case ',':
			stopcharMap[i] = config.MapComma
		case '/':
			stopcharMap[i] = config.MapDirsep
		}
	}
}

func StopSet(c byte, mask uint16) bool {
	return (stopcharMap[c] & mask) != 0
}

func Isdirsep(c byte) bool {
	return StopSet(c, config.MapDirsep)
}

func Isblank(c byte) bool {
	return StopSet(c, config.MapBlank)
}

func Isspace(c byte) bool {
	return StopSet(c, config.MapSpace)
}

func EndOfToken(c byte) bool {
	return StopSet(c, config.MapSpace|config.MapNul)
}

func NextToken(s string) string {
	for len(s) > 0 && Isspace(s[0]) {
		s = s[1:]
	}
	return s
}

func CollapseContinuations(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '\n' {
			i++ // skip the \n
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

func FindPercent(s string) int {
	return strings.IndexByte(s, '%')
}

func FindPercentCached(s *string) int {
	return strings.IndexByte(*s, '%')
}

func Lindex(s string, limit int, ch int) int {
	for i := 0; i < limit && i < len(s); i++ {
		if int(s[i]) == ch {
			return i
		}
	}
	return -1
}

func AlphaCompare(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func PrintSpaces(n uint) {
	_, _ = os.Stdout.WriteString(strings.Repeat(" ", int(n)))
}

func Xmalloc(size uintptr) interface{} {
	// In Go we just let the GC handle it
	return nil
}

func Xstrdup(s string) string {
	return s
}

func Xstrndup(s string, n int) string {
	if n > len(s) {
		n = len(s)
	}
	return s[:n]
}

func Strneq(a, b string, n int) bool {
	if len(a) < n || len(b) < n {
		return false
	}
	return a[:n] == b[:n]
}

func Streq(a, b string) bool {
	return a == b
}

func FindNextToken(s *string, length *int) string {
	for *s != "" && Isspace((*s)[0]) {
		*s = (*s)[1:]
	}
	if *s == "" {
		return ""
	}
	end := 0
	for end < len(*s) && !Isspace((*s)[end]) {
		end++
	}
	result := (*s)[:end]
	if end < len(*s) {
		*s = (*s)[end+1:]
	} else {
		*s = ""
	}
	if length != nil {
		*length = end
	}
	return result
}

func EndOfTokenStr(s string) string {
	for i := 0; i < len(s); i++ {
		if EndOfToken(s[i]) {
			return s[i:]
		}
	}
	return ""
}

func StripWhitespace(beg, end *string) string {
	*beg = NextToken(*beg)
	if *end != "" {
		for len(*end) > 0 && Isspace((*end)[len(*end)-1]) {
			*end = (*end)[:len(*end)-1]
		}
	}
	if *end <= *beg {
		return ""
	}
	return (*beg)[:len(*end)-len(*beg)]
}

func GetTmpdir() string {
	return config.DefaultTmpdir
}

func MakeToui(s string, end **string) uint {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		if end != nil {
			*end = &s
		}
		return 0
	}
	// Advance past digits
	digits := 0
	for digits < len(s) && s[digits] >= '0' && s[digits] <= '9' {
		digits++
	}
	if end != nil {
		remainder := s[digits:]
		*end = &remainder
	}
	return uint(v)
}

func MakeLltoa(val int64, buf []byte) string {
	return strconv.FormatInt(val, 10)
}

func MakeUlltoa(val uint64, buf []byte) string {
	return strconv.FormatUint(val, 10)
}

func MakeSeed(seed uint) {
	// In Go, crypto/rand is used for randomness
	_ = seed
}

func MakeRand() uint {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return uint(binary.LittleEndian.Uint32(b[:]))
}

func MakePid() int {
	return os.Getpid()
}

func Writebuf(fd int, buf []byte) (int, error) {
	err := os.WriteFile(fmt.Sprintf("/proc/self/fd/%d", fd), buf, 0644)
	if err != nil {
		return 0, err
	}
	return len(buf), nil
}

func Readbuf(fd int, buf []byte) (int, error) {
	f := os.NewFile(uintptr(fd), "")
	defer func() { _ = f.Close() }()
	return f.Read(buf)
}

func FindNextTokenStr(s *string) string {
	return FindNextToken(s, nil)
}
