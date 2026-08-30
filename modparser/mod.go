package modparser

import (
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
)

// Constructors for the transcribed tables, mirroring modLib.createMod's
// positional forms: mod(name, type, value, tags...), with a source and/or
// flags in front of the tags. A nil among the tags is a hole (kept, so the
// numbering matches the reference's table); trailing nils vanish as they
// do from a Lua constructor.

func mod(name string, typ ModType, value Value, tags ...Tag) *Mod {
	return &Mod{Name: name, Type: typ, Value: value, Tags: trimTags(tags)}
}

func modf(name string, typ ModType, value Value, flags ModFlag, kw KeywordFlag, tags ...Tag) *Mod {
	return &Mod{Name: name, Type: typ, Value: value, Flags: flags, KeywordFlags: kw, Tags: trimTags(tags)}
}

func mods(name string, typ ModType, value Value, source string, tags ...Tag) *Mod {
	return &Mod{Name: name, Type: typ, Value: value, Source: source, SourceSet: true, Tags: trimTags(tags)}
}

func modsf(name string, typ ModType, value Value, source string, flags ModFlag, kw KeywordFlag, tags ...Tag) *Mod {
	return &Mod{Name: name, Type: typ, Value: value, Source: source, SourceSet: true, Flags: flags, KeywordFlags: kw, Tags: trimTags(tags)}
}

// flag mirrors ModParser's local helper: a FLAG mod set to true.
func flag(name string, tags ...Tag) *Mod { return mod(name, Flag, Bool(true), tags...) }

func flagf(name string, flags ModFlag, kw KeywordFlag, tags ...Tag) *Mod {
	return modf(name, Flag, Bool(true), flags, kw, tags...)
}

func trimTags(tags []Tag) []Tag {
	for len(tags) > 0 && tags[len(tags)-1] == nil {
		tags = tags[:len(tags)-1]
	}
	if len(tags) == 0 {
		return nil
	}
	return append([]Tag{}, tags...)
}

// NewMod is modLib.createMod for the other packages.
func NewMod(name string, typ ModType, value Value, tags ...Tag) *Mod {
	return mod(name, typ, value, tags...)
}

// NewModFull is createMod with every positional part.
func NewModFull(name string, typ ModType, value Value, source string, sourceSet bool, flags ModFlag, kw KeywordFlag, tags ...Tag) *Mod {
	m := modf(name, typ, value, flags, kw, tags...)
	m.Source, m.SourceSet = source, sourceSet
	return m
}

// scanTable is one pattern table plus a fixed iteration order. The reference
// iterates with pairs(), which is unordered; sorted order keeps this port
// deterministic, and scan's own tie-break decides every case that matters.
// Non-plain tables hold Go regular expressions, compiled once here.
type scanTable[V any] struct {
	name   string
	m      map[string]V
	plain  bool
	keys   []string
	res    map[string]*regexp.Regexp
	tieLen map[string]int
}

func newScanTable[V any](name string, m map[string]V, plain bool) *scanTable[V] {
	t := &scanTable[V]{name: name, m: m, plain: plain}
	t.keys = make([]string, 0, len(m))
	for k := range m {
		t.keys = append(t.keys, k)
	}
	sort.Strings(t.keys)
	t.tieLen = make(map[string]int, len(m))
	t.res = make(map[string]*regexp.Regexp, len(m))
	for _, k := range t.keys {
		pat := k
		if plain {
			pat = regexp.QuoteMeta(k)
			t.tieLen[k] = len(k)
		} else {
			t.tieLen[k] = literalWeight(k)
		}
		re, err := regexp.Compile("(?i)" + pat)
		if err != nil {
			panic("table " + name + ": bad pattern " + k + ": " + err.Error())
		}
		t.res[k] = re
	}
	return t
}

// scan ports ModParser's scan: find the best-matching entry within line and
// return its value plus line with the matched span cut out. Matching is
// case-insensitive on the original line; captures are lowercased, as the
// reference took them from its lowercased copy.
//
// Earliest match wins; ties go to the longest match. A full tie (same span)
// prefers the entry with the higher literalWeight — the more literal, more
// specific pattern — then the longer pattern text. The reference broke this
// tie by Lua pattern length, which the regex conversion does not preserve.
func scan[V any](line string, t *scanTable[V]) (value V, found bool, rest string, caps []string) {
	bestIndex, bestEndIndex := 0, 0
	bestPattern := ""
	bestTieLen := 0

	for _, pattern := range t.keys {
		loc := t.res[pattern].FindStringSubmatchIndex(line)
		if loc == nil {
			continue
		}
		index, endIndex := loc[0]+1, loc[1]
		var c []string
		for g := 1; g*2 < len(loc); g++ {
			if loc[g*2] < 0 {
				c = append(c, "")
			} else {
				c = append(c, strings.ToLower(line[loc[g*2]:loc[g*2+1]]))
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
			value = t.m[pattern]
			caps = c
		}
	}
	if !found {
		return value, false, line, nil
	}
	return value, true, line[:bestIndex-1] + line[bestEndIndex:], caps
}

// cap1 returns the i-th capture (1-based) or "" — Lua closures index into the
// capture table the same way and get nil past the end.
func cap1(caps []string, i int) string {
	if i >= 1 && i <= len(caps) {
		return caps[i-1]
	}
	return ""
}

// num is tonumber or 0, for closures that trust their capture is numeric.
func num(s string) float64 {
	f, _ := util.Tonumber(s)
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

// caps are a pattern's captures. The reference calls each table's closures
// with slightly different leading arguments (specialModList gets
// tonumber(cap1), preFlagList gets the captures directly), but every
// argument derives from the captures, so the port passes just those and
// each hand-ported body reads what its original signature bound.
//
// Mapping used when porting: a Lua closure `function(num, _, x, y)` on
// specialModList/modTagList reads num as c.n(1), x as c.s(2), y as c.s(3);
// on preFlagList `function(cond, x)` reads cond as c.s(1), x as c.s(2).
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
	f, _ := util.Tonumber(c.s(i))
	return f
}

// nok reports whether capture i parses as a number.
func (c caps) nok(i int) bool {
	_, ok := util.Tonumber(c.s(i))
	return ok
}

// str is a Str value from a capture.
func (c caps) str(i int) Str { return Str(c.s(i)) }

// v is a Num value from a capture.
func (c caps) v(i int) Num { return Num(c.n(i)) }
