// Package luacanon serialises Go values in the canonical form of
// tools/canon.lua, byte for byte: every table is a JSON object with sorted
// stringified keys, whole numbers print without a decimal point, other
// numbers as %.14g, functions as {"__fn":true}. Used only by the game-data
// archive comparison.
package luacanon

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Fn stands for a Lua function value ({"__fn":true} in canon).
type Fn struct{}

// adapters convert domain values (e.g. structured mods) into encodable
// shapes before reflection sees them.
var adapters []func(any) (any, bool)

// RegisterAdapter installs a conversion tried on every non-scalar value.
func RegisterAdapter(fn func(any) (any, bool)) {
	adapters = append(adapters, fn)
}

// Encode canonicalises v. Supported: nil, bool, integers, floats, string,
// Fn, maps (string or numeric keys), slices/arrays (1-based keys), pointers
// (nil pointers vanish), and structs via `lua:"key"` tags (fields without a
// tag are skipped; ",omitempty" drops zero values, mirroring keys Data.lua
// only sets when true).
func Encode(v any) string {
	var b strings.Builder
	enc(&b, reflect.ValueOf(v))
	return b.String()
}

// EncodeExact is Encode with round-trippable floats (%.17g), matching
// canon.lua's encodeExact -- for fixture records, which are replay input
// rather than compared canon. Not safe concurrently with Encode.
func EncodeExact(v any) string {
	floatDigits = 17
	defer func() { floatDigits = 14 }()
	return Encode(v)
}

// floatDigits is the significand precision numStr formats non-integral
// floats at; EncodeExact widens it for one encode.
var floatDigits = 14

func enc(b *strings.Builder, rv reflect.Value) {
	if !rv.IsValid() {
		b.WriteString("null")
		return
	}
	if rv.CanInterface() {
		k := rv.Kind()
		if (k == reflect.Pointer || k == reflect.Interface || k == reflect.Struct) && !((k == reflect.Pointer || k == reflect.Interface) && rv.IsNil()) {
			for _, ad := range adapters {
				if conv, ok := ad(rv.Interface()); ok {
					enc(b, reflect.ValueOf(conv))
					return
				}
			}
		}
	}
	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			b.WriteString("null")
			return
		}
		enc(b, rv.Elem())
	case reflect.Bool:
		if rv.Bool() {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(b, "%d", rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprintf(b, "%d", rv.Uint())
	case reflect.Float32, reflect.Float64:
		b.WriteString(numStr(rv.Float()))
	case reflect.String:
		b.WriteString(Quote(rv.String()))
	case reflect.Func:
		b.WriteString(`{"__fn":true}`)
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(Fn{}) {
			b.WriteString(`{"__fn":true}`)
			return
		}
		writeObject(b, structPairs(rv))
	case reflect.Map:
		var pairs []kv
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			var ks string
			switch k.Kind() {
			case reflect.String:
				ks = k.String()
			case reflect.Int, reflect.Int64, reflect.Int32:
				ks = strconv.FormatInt(k.Int(), 10)
			case reflect.Float64:
				ks = numKey(k.Float())
			default:
				panic("luacanon: unsupported map key kind " + k.Kind().String())
			}
			if omitted(iter.Value()) {
				continue
			}
			pairs = append(pairs, kv{ks, iter.Value()})
		}
		writeObject(b, pairs)
	case reflect.Slice, reflect.Array:
		var pairs []kv
		for i := 0; i < rv.Len(); i++ {
			if omitted(rv.Index(i)) {
				continue
			}
			pairs = append(pairs, kv{strconv.Itoa(i + 1), rv.Index(i)})
		}
		writeObject(b, pairs)
	default:
		panic("luacanon: unsupported kind " + rv.Kind().String())
	}
}

type kv struct {
	k string
	v reflect.Value
}

// omitted reports whether a value stands for an absent Lua key: nil
// pointers, interfaces, functions, slices and maps vanish (a present-but-
// empty Lua table is a non-nil empty slice/map).
func omitted(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Slice, reflect.Map:
		return rv.IsNil()
	}
	return false
}

func structPairs(rv reflect.Value) []kv {
	var pairs []kv
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("lua")
		if tag == "" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		fv := rv.Field(i)
		if omitted(fv) {
			continue
		}
		if opts == "omitempty" && fv.IsZero() {
			continue
		}
		if name == "@array" {
			// The slice field is the table's array part.
			for j := 0; j < fv.Len(); j++ {
				if !omitted(fv.Index(j)) {
					pairs = append(pairs, kv{strconv.Itoa(j + 1), fv.Index(j)})
				}
			}
			continue
		}
		pairs = append(pairs, kv{name, fv})
	}
	return pairs
}

func writeObject(b *strings.Builder, pairs []kv) {
	sort.Slice(pairs, func(a, c int) bool { return pairs[a].k < pairs[c].k })
	b.WriteByte('{')
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(Quote(p.k))
		b.WriteByte(':')
		enc(b, p.v)
	}
	b.WriteByte('}')
}

// numStr matches canon.lua's num(): quoted tostring for nan/inf, %d for
// whole numbers within 1e15, else %.14g.
func numStr(v float64) string {
	if math.IsNaN(v) {
		return `"nan"`
	}
	if math.IsInf(v, 1) {
		return `"inf"`
	}
	if math.IsInf(v, -1) {
		return `"-inf"`
	}
	if v == math.Trunc(v) && v < 1e15 && v > -1e15 {
		return fmt.Sprintf("%d", int64(v))
	}
	return strconv.FormatFloat(v, 'g', floatDigits, 64)
}

// numKey is Lua tostring() for a number key (%.14g).
func numKey(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

// Quote matches canon.lua's quote().
func Quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\b':
			b.WriteString("\\b")
		case '\f':
			b.WriteString("\\f")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if c < 0x20 {
				fmt.Fprintf(&b, "\\u%04x", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
