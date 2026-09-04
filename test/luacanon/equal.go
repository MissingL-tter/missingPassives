package luacanon

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CompareDigits is how many significant digits two numbers must agree to.
//
// It is set by the reference, not chosen. PoB writes its data files and
// its ModCache as %.14g text and reads them back, so wherever a number
// reaches the archive that way it carries 14 significant digits and no
// more. Asking for closer agreement asks the reference for precision it
// does not have.
//
// The dumps record every double whole, so nothing is discarded before
// anything looks at it. Only the comparison quantizes.
const CompareDigits = 14

// sameToCompareDigits reports whether two numbers agree once both are
// rendered to CompareDigits significant figures - "the same number, as
// precisely as the reference knows it".
func sameToCompareDigits(a, b float64) bool {
	if a == b {
		return true
	}
	return strconv.FormatFloat(a, 'g', CompareDigits, 64) ==
		strconv.FormatFloat(b, 'g', CompareDigits, 64)
}

// NumericDiff is one leaf that disagreed beyond CompareDigits.
type NumericDiff struct {
	Path      string
	Got, Want any
}

func (d NumericDiff) String() string {
	return fmt.Sprintf("%s: %v vs archive %v", d.Path, d.Got, d.Want)
}

// FalseIsAbsent lists the keys the product holds as a plain bool where the
// reference left either false or no key at all. Every reference reader
// tests them for truthiness, so the two states are one: a missing key on
// either side compares as false. `enabled` is deliberately not here - a
// gem saved disabled is a stored false the port renders as one.
var FalseIsAbsent = map[string]bool{
	// skills.Gem / skills.SocketGroup
	"enableGlobal1": true, "enableGlobal2": true, "matchesSocket": true,
	"fromItem": true, "includeInFullDPS": true, "slotEnabled": true,
	// calc.Buff
	"applyNotPlayer": true, "applyMinions": true, "applyAllies": true,
	"allowTotemBuff": true,
	// calc.ConfigInput and the defence output it feeds
	"conditionLowEnergyShield": true, "CappingES": true,
	// modparser.ItemCondTag
	"corruptedCond": true, "shaperCond": true, "elderCond": true,
}

// EqualWithin compares two canonical encodings. Identical text is equal.
// Otherwise both are parsed and walked leaf by leaf: numbers agree once
// quantized to CompareDigits, everything else - strings, booleans, the
// shape itself - must match exactly, except that a FalseIsAbsent key
// present as false on one side and missing on the other is not a
// difference.
//
// It returns the leaves that disagreed and, separately, how many numbers
// agreed only after quantizing, so a caller can report what it let through
// rather than passing silently.
func EqualWithin(got, want string) (diffs []NumericDiff, tolerated int, err error) {
	if got == want {
		return nil, 0, nil
	}
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		return nil, 0, fmt.Errorf("parse got: %w", err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		return nil, 0, fmt.Errorf("parse want: %w", err)
	}
	c := &comparison{}
	c.walk("", g, w)
	return c.diffs, c.tolerated, nil
}

type comparison struct {
	diffs     []NumericDiff
	tolerated int
}

func (c *comparison) report(path string, got, want any) {
	c.diffs = append(c.diffs, NumericDiff{Path: path, Got: got, Want: want})
}

func (c *comparison) walk(path string, got, want any) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			c.report(path, got, want)
			return
		}
		// A Lua array reaches the canon as an object keyed "1".."n", so
		// walking it by key would compare position by position and fail on
		// any reordering whatever the values are. Compare as a multiset:
		// same elements, same counts, order free.
		if ge, gok := arrayElems(g); gok {
			if we, wok := arrayElems(w); wok {
				c.multiset(path, ge, we)
				return
			}
		}
		keys := map[string]bool{}
		for k := range g {
			keys[k] = true
		}
		for k := range w {
			keys[k] = true
		}
		for _, k := range sortedKeys(keys) {
			gv, inGot := g[k]
			wv, inWant := w[k]
			switch {
			case !inGot:
				if !(FalseIsAbsent[k] && wv == false) {
					c.report(join(path, k), nil, wv)
				}
			case !inWant:
				if !(FalseIsAbsent[k] && gv == false) {
					c.report(join(path, k), gv, nil)
				}
			default:
				c.walk(join(path, k), gv, wv)
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			c.report(path, got, want)
			return
		}
		c.multiset(path, g, w)
	case float64:
		g, ok := got.(float64)
		if !ok {
			c.report(path, got, want)
			return
		}
		if !sameToCompareDigits(g, w) {
			c.report(path, g, w)
			return
		}
		if g != w {
			c.tolerated++
		}
	default:
		if got != want {
			c.report(path, got, want)
		}
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// FormatDiffs renders up to n disagreements for a failure message.
func FormatDiffs(diffs []NumericDiff, n int) string {
	var b strings.Builder
	for i, d := range diffs {
		if i == n {
			fmt.Fprintf(&b, "\n  ... and %d more", len(diffs)-n)
			break
		}
		b.WriteString("\n  " + d.String())
	}
	return b.String()
}

// arrayElems reports an object's values in key order when its keys are
// exactly "1".."n" - the shape a Lua array takes in the canon - and
// whether it had that shape at all.
func arrayElems(m map[string]any) ([]any, bool) {
	if len(m) == 0 {
		return nil, false
	}
	out := make([]any, len(m))
	for k, v := range m {
		i, err := strconv.Atoi(k)
		if err != nil || i < 1 || i > len(m) {
			return nil, false
		}
		if out[i-1] != nil {
			return nil, false
		}
		out[i-1] = v
	}
	return out, true
}

// multiset compares two sequences ignoring order: same elements, same
// counts. Elements are keyed by a deterministic rendering that quantizes
// numbers the same way the leaf comparison does, so a last-digit
// difference inside an element does not read as a different element.
func (c *comparison) multiset(path string, got, want []any) {
	have := map[string][]any{}
	for _, g := range got {
		k := elemKey(g)
		have[k] = append(have[k], g)
	}
	var missing []any
	for _, w := range want {
		k := elemKey(w)
		if len(have[k]) == 0 {
			missing = append(missing, w)
			continue
		}
		have[k] = have[k][1:]
	}
	for _, w := range missing {
		c.report(path+"[]", nil, w)
	}
	for _, left := range have {
		for _, g := range left {
			c.report(path+"[]", g, nil)
		}
	}
}

// elemKey renders a decoded value deterministically, with numbers at
// CompareDigits so it agrees with the leaf comparison.
func elemKey(v any) string {
	var b strings.Builder
	writeElemKey(&b, v)
	return b.String()
}

func writeElemKey(b *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		fmt.Fprintf(b, "%t", t)
	case float64:
		b.WriteString(strconv.FormatFloat(t, 'g', CompareDigits, 64))
	case string:
		b.WriteString(strconv.Quote(t))
	case []any:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			writeElemKey(b, e)
		}
		b.WriteByte(']')
	case map[string]any:
		// An array-shaped element is itself order-free, so render its
		// members sorted by their own key rather than by position -
		// otherwise a reordering nested inside an element makes it read
		// as a different element.
		if elems, ok := arrayElems(t); ok {
			rendered := make([]string, len(elems))
			for i, e := range elems {
				rendered[i] = elemKey(e)
			}
			sort.Strings(rendered)
			b.WriteByte('[')
			for i, r := range rendered {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteString(r)
			}
			b.WriteByte(']')
			return
		}
		keys := make([]string, 0, len(t))
		for k, v := range t {
			if FalseIsAbsent[k] && v == false {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Quote(k))
			b.WriteByte(':')
			writeElemKey(b, t[k])
		}
		b.WriteByte('}')
	default:
		fmt.Fprintf(b, "%v", v)
	}
}

// SameCanon reports whether two canonical encodings agree: numbers to
// CompareDigits, arrays (including a Lua array's "1".."n" object form) as
// multisets. It is the comparison every differential should use for canon
// text - a positional `!=` fails on any reordering whatever the values are,
// which is a defect in the comparison rather than a disagreement between
// the programs (knowledge.md 4.6). Text that will not decode falls back to
// exact equality so non-JSON payloads are still compared.
func SameCanon(got, want string) bool {
	if got == want {
		return true
	}
	diffs, _, err := EqualWithin(got, want)
	if err != nil {
		return false
	}
	return len(diffs) == 0
}
