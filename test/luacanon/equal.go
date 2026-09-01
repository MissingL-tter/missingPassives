package luacanon

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Tolerance is the relative agreement two numbers must reach to count as
// equal when their canonical text differs. It is relative, not absolute:
// one fixed epsilon is unsatisfiable at one end of the range and vacuous
// at the other. At a damage total of 5e6 the gap between adjacent doubles
// is about 1e-9, so an absolute 1e-14 could never be met; at a cast time
// of 1e-5 that same 1e-14 spans some six million doubles.
//
// The value is set by the reference, not by us. PoB's own data files and
// its ModCache are written as text at %.14g and read back, so wherever a
// number reaches the archive through one of those it carries exactly 14
// significant digits and no more. Rounding to 14 significant digits moves
// a value by at most half a unit in the last one, which in relative terms
// is worst when the leading digit is 1: 0.5e-13 / 1 = 5e-14. Asking for
// closer agreement than that asks the reference for precision it does not
// have. The largest such difference measured across the corpus is
// 3.7e-14, in monsterDamageTable.
//
// Everything the two sides actually compute agrees far better - the
// widest arithmetic drift measured is 1.6e-15, some thirty times inside
// this bound - so a failure here is a real disagreement, not rounding.
const Tolerance = 5e-14

// nearlyEqual reports whether two numbers agree to within Tolerance,
// scaled by the larger magnitude. An exact zero on one side alone has no
// scale to be relative to, so it fails - a term that should have vanished
// and did not is a real disagreement, not a rounding one.
func nearlyEqual(a, b float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	larger := math.Max(math.Abs(a), math.Abs(b))
	return diff <= larger*Tolerance
}

// NumericDiff is one leaf that disagreed beyond Tolerance.
type NumericDiff struct {
	Path      string
	Got, Want any
}

func (d NumericDiff) String() string {
	return fmt.Sprintf("%s: %v vs archive %v", d.Path, d.Got, d.Want)
}

// EqualWithin compares two canonical encodings. Identical text is equal.
// Otherwise both are parsed and walked leaf by leaf: numbers agree within
// Tolerance, everything else - strings, booleans, the shape itself - must
// match exactly.
//
// It returns the leaves that disagreed and, separately, how many numbers
// were equal only within tolerance, so a caller can report what it let
// through rather than passing silently.
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
				c.report(join(path, k), nil, wv)
			case !inWant:
				c.report(join(path, k), gv, nil)
			default:
				c.walk(join(path, k), gv, wv)
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			c.report(path, got, want)
			return
		}
		for i := range w {
			c.walk(fmt.Sprintf("%s[%d]", path, i), g[i], w[i])
		}
	case float64:
		g, ok := got.(float64)
		if !ok {
			c.report(path, got, want)
			return
		}
		if !nearlyEqual(g, w) {
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
