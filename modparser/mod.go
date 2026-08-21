package modparser

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Mod mirrors the table modLib.createMod builds: name, type, value, bit flags,
// an optional source, and a list of tags in its array part.
type Mod struct {
	Name         string
	Type         string
	Value        any
	Flags        int64
	KeywordFlags int64
	Source       string
	SourceSet    bool
	SourceSlot   string // mod.sourceSlot, set by Item slot mod lists ("" = absent)
	Replaced     bool // set by ModDB ReplaceMod bookkeeping (mod.replaced)
	Converted    bool // set by ModDB ConvertMod bookkeeping (mod.converted)
	Tags         []any
}

// Tag is a plain Lua-style tag table: { type = "Condition", var = "Fortified" }.
type Tag = map[string]any

// D is a mixed Lua table used as a pattern-table value: an array part (names,
// or per-mod tag descriptors) plus a hash part (tag, tagList, flags,
// keywordFlags, addToMinion, ...).
type D struct {
	Arr []any
	KV  map[string]any
}

type pair struct {
	k string
	v any
}

// p builds one key/value field of a d() table, mirroring Lua's `key = value`.
func p(k string, v any) pair { return pair{k, v} }

// d builds a mixed table the way a Lua constructor does: positional arguments
// fill the array part, p() pairs fill the hash part.
func d(items ...any) *D {
	out := &D{}
	for _, it := range items {
		if kv, ok := it.(pair); ok {
			if out.KV == nil {
				out.KV = make(map[string]any)
			}
			out.KV[kv.k] = kv.v
		} else {
			out.Arr = append(out.Arr, it)
		}
	}
	return out
}

func (t *D) get(k string) any {
	if t == nil || t.KV == nil {
		return nil
	}
	return t.KV[k]
}

// mod mirrors modLib.createMod exactly, including its positional quirks: an
// optional string source in position 1, then numbers for flags and keyword
// flags in positions 2 and 3, then the tags. A nil among the tags leaves a
// hole (the serialiser skips it but keeps the numbering), and trailing nils
// vanish, both exactly as a Lua table constructor behaves.
func mod(name, typ string, value any, rest ...any) *Mod {
	m := &Mod{Name: name, Type: typ, Value: value}
	tagStart := 0
	if len(rest) >= 1 {
		if s, ok := rest[0].(string); ok {
			m.Source = s
			m.SourceSet = true
			tagStart = 1
		}
	}
	if len(rest) >= 2 {
		if n, ok := asInt64(rest[1]); ok {
			m.Flags = n
			tagStart = 2
		}
	}
	if len(rest) >= 3 {
		if n, ok := asInt64(rest[2]); ok {
			m.KeywordFlags = n
			tagStart = 3
		}
	}
	m.Tags = append(m.Tags, rest[tagStart:]...)
	for len(m.Tags) > 0 && m.Tags[len(m.Tags)-1] == nil {
		m.Tags = m.Tags[:len(m.Tags)-1]
	}
	return m
}

// NewMod is modLib.createMod for the other packages: mod() with its
// positional quirks, exported.
func NewMod(name, typ string, value any, rest ...any) *Mod {
	return mod(name, typ, value, rest...)
}

// flag mirrors ModParser's local helper: a FLAG mod set to true.
func flag(name string, rest ...any) *Mod {
	args := append([]any{}, rest...)
	return mod(name, "FLAG", true, args...)
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	}
	return 0, false
}

// asciiLower mirrors Lua 5.1's string.lower under the C locale: only A-Z fold,
// and the byte length never changes, keeping match offsets aligned with the
// original line.
func asciiLower(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if b == nil {
				b = []byte(s)
			}
			b[i] = c + 32
		}
	}
	if b == nil {
		return s
	}
	return string(b)
}

// scanTable is one pattern table plus a fixed iteration order. The reference
// iterates with pairs(), which is unordered; sorted order keeps this port
// deterministic, and scan's own tie-break decides every case that matters.
// Non-plain tables hold Go regular expressions, compiled once here.
type scanTable struct {
	name   string
	m      map[string]any
	plain  bool
	keys   []string
	res    map[string]*regexp.Regexp
	tieLen map[string]int
}

func newScanTable(name string, m map[string]any, plain bool) *scanTable {
	t := &scanTable{name: name, m: m, plain: plain}
	t.keys = make([]string, 0, len(m))
	for k := range m {
		t.keys = append(t.keys, k)
	}
	sort.Strings(t.keys)
	t.tieLen = make(map[string]int, len(m))
	if !plain {
		t.res = make(map[string]*regexp.Regexp, len(m))
		for _, k := range t.keys {
			re, err := regexp.Compile(k)
			if err != nil {
				panic("table " + name + ": bad pattern " + k + ": " + err.Error())
			}
			t.res[k] = re
			t.tieLen[k] = literalWeight(k)
		}
	} else {
		for _, k := range t.keys {
			t.tieLen[k] = len(k)
		}
	}
	return t
}

// scan ports ModParser's scan: find the best-matching entry within line and
// return its value plus line with the matched span cut out. Matching runs on
// an ASCII-lowercased copy while the remainder is cut from the original.
//
// Earliest match wins; ties go to the longest match. A full tie (same span)
// prefers the entry with the higher literalWeight — the more literal, more
// specific pattern — then the longer pattern text. The reference broke this
// tie by Lua pattern length, which the regex conversion does not preserve.
func scan(line string, t *scanTable) (value any, rest string, caps []string) {
	lineLower := asciiLower(line)
	var bestVal any
	bestIndex, bestEndIndex := 0, 0
	bestPattern := ""
	bestTieLen := 0
	var bestCaps []string
	found := false

	for _, pattern := range t.keys {
		var index, endIndex int
		var c []string
		if t.plain {
			idx := strings.Index(lineLower, pattern)
			if idx < 0 {
				continue
			}
			index, endIndex = idx+1, idx+len(pattern)
		} else {
			loc := t.res[pattern].FindStringSubmatchIndex(lineLower)
			if loc == nil {
				continue
			}
			index, endIndex = loc[0]+1, loc[1]
			for g := 1; g*2 < len(loc); g++ {
				if loc[g*2] < 0 {
					c = append(c, "")
				} else {
					c = append(c, lineLower[loc[g*2]:loc[g*2+1]])
				}
			}
		}
		tl := t.tieLen[pattern]
		better := !found || index < bestIndex ||
			(index == bestIndex && (endIndex > bestEndIndex ||
				(endIndex == bestEndIndex && (tl > bestTieLen ||
					(tl == bestTieLen && len(pattern) > len(bestPattern))))))
		if better {
			found = true
			bestIndex = index
			bestEndIndex = endIndex
			bestPattern = pattern
			bestTieLen = tl
			bestVal = t.m[pattern]
			bestCaps = c
		}
	}
	if !found {
		return nil, line, nil
	}
	return bestVal, line[:bestIndex-1] + line[bestEndIndex:], bestCaps
}

// cap returns the i-th capture (1-based) or "" — Lua closures index into the
// capture table the same way and get nil past the end.
func cap1(caps []string, i int) string {
	if i >= 1 && i <= len(caps) {
		return caps[i-1]
	}
	return ""
}

// tonumber mirrors Lua's tonumber for the shapes mod text produces.
func tonumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// num is tonumber or 0, for closures that trust their capture is numeric.
func num(s string) float64 {
	f, _ := tonumber(s)
	return f
}

// firstToUpper mirrors ModParser's helper: str:gsub("^%l", string.upper).
func firstToUpper(s string) string {
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

// literalWeight measures a pattern's specificity for scan's full-tie break: the
// length of everything outside its capture groups. The reference broke full
// ties by Lua pattern length, which mostly ordered a specific literal variant
// above its generic captured form; the regex conversion inflates class syntax
// and inverts raw lengths, while the outside-groups length preserves the
// intended ordering.
func literalWeight(pattern string) int {
	weight := 0
	depth := 0
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\\' && i+1 < len(pattern) {
			if depth == 0 {
				weight += 2
			}
			i++
			continue
		}
		switch c {
		case '(':
			depth++
			continue
		case ')':
			depth--
			continue
		}
		if depth == 0 {
			weight++
		}
	}
	return weight
}

// fn is a pattern-table closure. The reference calls each table's closures
// with slightly different leading arguments (specialModList gets
// tonumber(cap1), preFlagList gets the captures directly), but every argument
// derives from the captures, so the port passes just those and each hand-ported
// body reads what its original signature bound.
//
// Mapping used when porting: a Lua closure `function(num, _, x, y)` on
// specialModList/modTagList reads num as c.n(1), x as c.s(2), y as c.s(3);
// on preFlagList `function(cond, x)` reads cond as c.s(1), x as c.s(2).
type fn func(c caps) any

type caps []string

// s returns capture i (1-based) or "".
func (c caps) s(i int) string {
	if i >= 1 && i <= len(c) {
		return c[i-1]
	}
	return ""
}

// n returns tonumber(capture i) or 0.
func (c caps) n(i int) float64 {
	f, _ := tonumber(c.s(i))
	return f
}

// nok reports whether capture i parses as a number.
func (c caps) nok(i int) bool {
	_, ok := tonumber(c.s(i))
	return ok
}
