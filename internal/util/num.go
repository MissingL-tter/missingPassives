package util

import (
	"math"
	"strconv"
	"strings"
)

// FormatG14 is C "%.14g" with inf/nan spelled the way LuaJIT prints them:
// tostring(number) without a shortcut for integral values.
func FormatG14(v float64) string {
	if math.IsInf(v, 1) {
		return "inf"
	}
	if math.IsInf(v, -1) {
		return "-inf"
	}
	if math.IsNaN(v) {
		return "nan"
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

// FormatIntOrG14 prints an integral value below 1e15 as a plain integer and
// anything else through FormatG14. Differs from FormatG14 only for -0 ("0")
// and integral magnitudes in [1e14, 1e15) (no exponent form); the archive
// fixtures were produced with this body.
func FormatIntOrG14(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return FormatG14(v)
}

// FormatInt prints an integral float as a decimal integer.
func FormatInt(v float64) string { return strconv.FormatInt(int64(v), 10) }

// Quantize14 is one number's trip through Data/ModCache.lua: written with
// %.14g and read back. Infinities and integral values below 1e15 survive
// verbatim.
func Quantize14(v float64) float64 {
	if math.IsInf(v, 0) || v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return v
	}
	n, _ := strconv.ParseFloat(FormatG14(v), 64)
	return n
}

// RoundHalfUp is Common.lua round(val, dec): floor(val*10^dec + 0.5)/10^dec.
func RoundHalfUp(v float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(v*p+0.5) / p
}

// Tonumber is Lua tonumber(string): surrounding whitespace ignored, decimal
// and exponent forms, and 0x hex integers (optionally signed).
//
// Known divergences from Lua 5.1 / LuaJIT: strconv.ParseFloat also accepts
// "inf"/"infinity"/"nan" in any case, hex floats with a p exponent
// ("0x1p4"), and underscore-separated digits after a base prefix;
// strings.TrimSpace also strips Unicode spaces, not just C isspace.
func Tonumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if h, ok := hexInt(s); ok {
		return h, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func hexInt(s string) (float64, bool) {
	neg := false
	switch s[0] {
	case '-':
		neg = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	if len(s) < 3 || s[0] != '0' || (s[1] != 'x' && s[1] != 'X') {
		return 0, false
	}
	n, err := strconv.ParseUint(s[2:], 16, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		return -float64(n), true
	}
	return float64(n), true
}
