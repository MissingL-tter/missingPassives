// Lua-faithful helpers: the Common.lua rounding family, tostring formatting,
// and the handful of string operations Item.lua leans on.
package item

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// round is Common.lua round: floor(v + 0.5), or at dec places.
func round(v float64) float64 { return math.Floor(v + 0.5) }

func roundDec(v float64, dec int) float64 {
	factor := math.Pow(10, float64(dec))
	return math.Floor(v*factor+0.5) / factor
}

// roundSymmetric is Common.lua roundSymmetric.
func roundSymmetric(v float64) float64 {
	if v >= 0 {
		return math.Floor(v + 0.5)
	}
	return math.Ceil(v - 0.5)
}

func roundSymmetricDec(v float64, dec int) float64 {
	factor := math.Pow(10, float64(dec))
	if v >= 0 {
		return math.Floor(v*factor+0.5) / factor
	}
	return math.Ceil(v*factor-0.5) / factor
}

// floorSymmetric is Common.lua floorSymmetric: math.modf's integral part.
func floorSymmetric(v float64) float64 { return math.Trunc(v) }

// alwaysPositiveRound is Common.lua alwaysPositiveRound (no-dec form).
func alwaysPositiveRound(v float64) float64 { return floorSymmetric(v + 0.5) }

// luaNumStr is tostring() on a Lua number: %.14g with integer fast path.
func luaNumStr(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

// luaLower is string.lower: ASCII-only (C locale).
func luaLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sortedKeys[V any](m map[int]V) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// specToNumber ports the file-local specToNumber: tonumber of the leading
// [+-]?[%d%.]+ run.
func specToNumber(s string) *float64 {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	j := i
	for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
		j++
	}
	if j == i {
		return nil
	}
	n, err := strconv.ParseFloat(s[:j], 64)
	if err != nil {
		return nil
	}
	return &n
}

var numberRe = regexp.MustCompile(`\d+\.?\d*`)

// gsubNumberHash is line:gsub("%d+%.?%d*", "#").
func gsubNumberHash(line string) string {
	return numberRe.ReplaceAllString(line, "#")
}

// signedNumRe is the ubiquitous (%-?%d+%.?%d*) token.
var signedNumRe = regexp.MustCompile(`-?\d+\.?\d*`)

// trimRawLines is raw:gmatch("%s*([^\n]*%S)"): non-blank lines with
// surrounding whitespace stripped.
func trimRawLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.Trim(line, " \t\v\f\r")
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// sanitiseText ports Common.lua sanitiseText: strip <...> spans and map the
// unicode hyphen family to "-", but only when a byte 128-255 or '<' occurs.
func sanitiseText(text string) string {
	if !strings.ContainsFunc(text, func(r rune) bool { return r == '<' || r > 127 }) {
		// The reference returns nil here (the and-chain falls through);
		// every caller treats that as "unchanged".
		return text
	}
	text = stripBalanced(text, '<', '>')
	for _, hyphen := range []string{"‐", "‑", "‒", "–", "—", "―"} {
		text = strings.ReplaceAll(text, hyphen, "-")
	}
	return text
}

// stripBalanced is gsub("%b<>", ""): remove balanced open..close spans.
func stripBalanced(s string, open, close byte) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == open {
			depth := 1
			j := i + 1
			for j < len(s) && depth > 0 {
				if s[j] == open {
					depth++
				} else if s[j] == close {
					depth--
				}
				j++
			}
			if depth == 0 {
				i = j
				continue
			}
			// Unbalanced: no match, keep the byte.
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}

var (
	gggCurlyRe   = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	gggPlainRe   = regexp.MustCompile(`\[([^|\]]+)\]`)
	gggDisplayRe = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
)

// escapeGGGString ports Common.lua escapeGGGString: "[Critical|Critical
// Hit]" -> "Critical Hit", "<tag>{text}" -> "text".
func escapeGGGString(text string) string {
	text = gggCurlyRe.ReplaceAllString(text, "$1")
	text = gggPlainRe.ReplaceAllString(text, "$1")
	text = gggDisplayRe.ReplaceAllString(text, "$1")
	return text
}
