// Package luarender renders schema documents back into the byte-exact
// Data/*.lua files the reference Lua exporter produced. It exists only for
// the differential test against the archive: the runtime never touches Lua,
// and when the archive comparison stops being the contract this package is
// deleted whole.
package luarender

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Templates supplies the hand-maintained template files some renders
// interleave with generated data, by path relative to the Lua src/ tree
// (e.g. "Export/Uniques/axe.lua").
type Templates interface {
	Read(rel string) (string, error)
}

// RenderFn turns one script's JSON document into its output files
// (path relative to the Lua src/ tree -> exact contents).
type RenderFn func(raw json.RawMessage, tpl Templates) (map[string]string, error)

// Renderers maps script name -> renderer.
var Renderers = map[string]RenderFn{}

// register adds a typed renderer, wrapping it with the JSON decode so the
// test exercises the same round-trip the real pipeline would.
func register[T any](name string, fn func(d T, tpl Templates) (map[string]string, error)) {
	if _, dup := Renderers[name]; dup {
		panic("luarender: duplicate renderer " + name)
	}
	Renderers[name] = func(raw json.RawMessage, tpl Templates) (map[string]string, error) {
		var d T
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("%s: decoding document: %w", name, err)
		}
		return fn(d, tpl)
	}
}

// B accumulates one output file, with the reference's out:write semantics.
type B struct{ strings.Builder }

// W mirrors Lua's out:write(...): strings and numbers only.
func (b *B) W(args ...any) {
	for _, a := range args {
		switch v := a.(type) {
		case string:
			b.WriteString(v)
		case int:
			b.WriteString(strconv.Itoa(v))
		case int64:
			b.WriteString(strconv.FormatInt(v, 10))
		case float64:
			b.WriteString(luaNum(v))
		default:
			panic(fmt.Sprintf("luarender: W: unsupported type %T", a))
		}
	}
}

// luaNum formats a float64 the way LuaJIT's tostring/write does (%.14g,
// inf/nan spelled Lua-style).
func luaNum(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if math.IsNaN(f) {
		return "nan"
	}
	return strconv.FormatFloat(f, 'g', 14, 64)
}

// luaStr mirrors tostring() for the value kinds renders hit.
func luaStr(v any) string {
	switch t := v.(type) {
	case nil:
		return "nil"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return luaNum(t)
	}
	panic(fmt.Sprintf("luarender: luaStr: unsupported type %T", v))
}
