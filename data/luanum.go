// Lua number semantics for values that pass through text.

package data

import (
	"math"
	"strconv"
	"strings"
)

// parseLuaNumber is tonumber() for the literal forms the data files hold:
// decimal floats and 0x hex integers.
func parseLuaNumber(s string) (float64, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseUint(s[2:], 16, 64)
		return float64(n), err
	}
	return strconv.ParseFloat(s, 64)
}

// luaIntString is tostring() for an integral float.
func luaIntString(v float64) string {
	return strconv.FormatInt(int64(v), 10)
}

// luaNumString is tostring() (%.14g).
func luaNumString(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return luaIntString(v)
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

// luaStringLiteral evaluates a quoted Lua string literal (template text the
// reference loads as code).
func luaStringLiteral(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		panic("data: not a quoted Lua string literal: " + s)
	}
	return luaUnescape(s[1 : len(s)-1])
}

// luaUnescape resolves the short escape sequences of a Lua string literal,
// for generated text the reference loads through one.
func luaUnescape(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '\\', '"', '\'':
			b.WriteByte(s[i])
		default:
			panic("data: unhandled Lua escape \\" + string(s[i]))
		}
	}
	return b.String()
}
