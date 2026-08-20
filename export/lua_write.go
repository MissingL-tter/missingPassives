// Port of Common.lua's writeLuaTable/qFmt: serializes a Lua-shaped value
// tree (map[any]any tables with int/string/bool keys) byte-identically.

package export

import (
	"sort"
	"strings"
)

// luaTable is a Lua table under construction: int and string keys.
type luaTable map[any]any

func qFmt(s string) string {
	return "\"" + strings.ReplaceAll(strings.ReplaceAll(s, "\n", "\\n"), "\"", "\\\"") + "\""
}

var reBareKey = regexpMustBare()

func regexpMustBare() func(string) bool {
	return func(s string) bool {
		// ^%a[%a%d]*$ — letters and digits only, starting with a letter.
		if s == "" || s == "hexproof" {
			return false
		}
		for i, c := range s {
			isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
			isDigit := c >= '0' && c <= '9'
			if i == 0 && !isAlpha {
				return false
			}
			if !isAlpha && !isDigit {
				return false
			}
		}
		return true
	}
}

func luaTypeRank(k any) int {
	switch k.(type) {
	case bool:
		return 0 // "boolean"
	case int, int64, float64:
		return 1 // "number"
	case string:
		return 2 // "string"
	}
	return 3
}

func keyNum(k any) float64 {
	switch n := k.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

// writeLuaTable ports Common.lua's writeLuaTable (always called with an
// indent here, as the statdesc script does).
func writeLuaTable(o *OutFile, t luaTable, indent int) {
	o.W("{\n")
	keys := make([]any, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		ra, rb := luaTypeRank(keys[a]), luaTypeRank(keys[b])
		if ra != rb {
			return ra < rb
		}
		if ra == 1 {
			return keyNum(keys[a]) < keyNum(keys[b])
		}
		if ra == 2 {
			return keys[a].(string) < keys[b].(string)
		}
		return false
	})
	for i, k := range keys {
		v := t[k]
		o.W(strings.Repeat("\t", indent))
		if ks, ok := k.(string); ok && reBareKey(ks) {
			o.W(ks, "=")
		} else if luaTypeRank(k) == 1 {
			o.W("[", luaNum(keyNum(k)), "]=")
		} else {
			o.W("[", qFmt(k.(string)), "]=")
		}
		switch vv := v.(type) {
		case luaTable:
			writeLuaTable(o, vv, indent+1)
		case string:
			o.W(qFmt(vv))
		case bool:
			o.W(luaStr(vv))
		case float64:
			o.W(luaNum(vv))
		case int:
			o.W(vv)
		case int64:
			o.W(vv)
		default:
			panic("writeLuaTable: unsupported value")
		}
		if i < len(keys)-1 {
			o.W(",")
		}
		o.W("\n")
	}
	o.W(strings.Repeat("\t", indent-1))
	o.W("}")
}
