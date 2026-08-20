package modparser

import (
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Canon serialises a parse result the same way tools/canon.lua serialises
// the reference's, so the two can be compared as strings. Every table becomes
// a JSON object with sorted keys; array indices are stringified ("1", "2", …);
// whole numbers print without a decimal point; functions print as
// {"__fn":true} since behaviour cannot be compared bytewise.
func Canon(v any) string {
	var sb strings.Builder
	writeCanon(&sb, v)
	return sb.String()
}

func writeCanon(sb *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		sb.WriteString("null")
	case bool:
		sb.WriteString(strconv.FormatBool(t))
	case int:
		sb.WriteString(strconv.Itoa(t))
	case int64:
		sb.WriteString(strconv.FormatInt(t, 10))
	case float64:
		sb.WriteString(canonNumber(t))
	case string:
		writeCanonString(sb, t)
	case *Mod:
		writeCanonMod(sb, t)
	case []any:
		// A pure array: numbered keys starting at 1, emitted in STRING-sorted
		// order (Lua-side canon sorts stringified keys, so "10" precedes "2").
		// A nil element vanishes, as a nil in a Lua constructor leaves a hole.
		keys := make([]string, 0, len(t))
		for i, e := range t {
			if e != nil {
				keys = append(keys, strconv.Itoa(i+1))
			}
		}
		sort.Strings(keys)
		sb.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeCanonString(sb, k)
			sb.WriteByte(':')
			idx, _ := strconv.Atoi(k)
			writeCanon(sb, t[idx-1])
		}
		sb.WriteByte('}')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			if t[k] == nil {
				continue // Lua tables have no nil-valued keys
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sb.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeCanonString(sb, k)
			sb.WriteByte(':')
			writeCanon(sb, t[k])
		}
		sb.WriteByte('}')
	case *D:
		writeCanonD(sb, t)
	default:
		if reflect.ValueOf(v).Kind() == reflect.Func {
			sb.WriteString(`{"__fn":true}`)
			return
		}
		writeCanonString(sb, "?unsupported?")
	}
}

// writeCanonMod emits a Mod as its Lua table shape: numbered tags plus the
// named fields, keys sorted. Source is omitted when unset, exactly as a nil
// key vanishes from a Lua table.
func writeCanonMod(sb *strings.Builder, m *Mod) {
	kv := map[string]any{
		"name":         m.Name,
		"type":         m.Type,
		"value":        m.Value,
		"flags":        m.Flags,
		"keywordFlags": m.KeywordFlags,
	}
	if m.SourceSet {
		kv["source"] = m.Source
	}
	keys := make([]string, 0, len(kv)+len(m.Tags))
	for i := range m.Tags {
		if m.Tags[i] != nil { // a nil tag is a hole in the Lua table
			keys = append(keys, strconv.Itoa(i+1))
		}
	}
	for k := range kv {
		if kv[k] == nil {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		writeCanonString(sb, k)
		sb.WriteByte(':')
		if idx, err := strconv.Atoi(k); err == nil {
			writeCanon(sb, m.Tags[idx-1])
		} else {
			writeCanon(sb, kv[k])
		}
	}
	sb.WriteByte('}')
}

// writeCanonD emits a mixed table: array part under "1".., hash part by key.
func writeCanonD(sb *strings.Builder, t *D) {
	keys := make([]string, 0, len(t.Arr)+len(t.KV))
	for i := range t.Arr {
		if t.Arr[i] != nil {
			keys = append(keys, strconv.Itoa(i+1))
		}
	}
	for k, v := range t.KV {
		if v != nil {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	sb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		writeCanonString(sb, k)
		sb.WriteByte(':')
		if idx, err := strconv.Atoi(k); err == nil && idx >= 1 && idx <= len(t.Arr) {
			writeCanon(sb, t.Arr[idx-1])
		} else {
			writeCanon(sb, t.KV[k])
		}
	}
	sb.WriteByte('}')
}

// canonNumber matches canon.lua: whole numbers print with no decimal point,
// everything else with %.14g; infinities and NaN print as Lua's tostring
// renders them, quoted.
func canonNumber(f float64) string {
	if math.IsInf(f, 1) {
		return `"inf"`
	}
	if math.IsInf(f, -1) {
		return `"-inf"`
	}
	if math.IsNaN(f) {
		return `"nan"`
	}
	if f == float64(int64(f)) && f < 1e15 && f > -1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return formatG14(f)
}

// formatG14 mimics C's %.14g, which Lua's string.format uses.
func formatG14(f float64) string {
	s := strconv.FormatFloat(f, 'g', 14, 64)
	// Go writes exponents as e+06 / e-06; C's %g writes e+06 too. Shapes align
	// for the magnitudes mod values take, so no further fixup is needed.
	return s
}

var canonEscapes = map[byte]string{
	'"': `\"`, '\\': `\\`, '\b': `\b`, '\f': `\f`, '\n': `\n`, '\r': `\r`, '\t': `\t`,
}

func writeCanonString(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if esc, ok := canonEscapes[c]; ok {
			sb.WriteString(esc)
			continue
		}
		if c < 0x20 || c == 0x7f {
			const hex = "0123456789abcdef"
			sb.WriteString(`\u00`)
			sb.WriteByte(hex[c>>4])
			sb.WriteByte(hex[c&0xf])
			continue
		}
		sb.WriteByte(c)
	}
	sb.WriteByte('"')
}
