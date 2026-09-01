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

// EqualWithin compares two canonical encodings. Identical text is equal.
// Otherwise both are parsed and walked leaf by leaf: numbers agree once
// quantized to CompareDigits, everything else - strings, booleans, the
// shape itself - must match exactly.
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
