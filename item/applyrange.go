// Port of Modules/ItemTools.lua's value machinery: applyValueScalar,
// formatValue, isZeroValueLine and applyRange (with its modScalability
// combination search and the high-precision fallback).
package item

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

var antonyms = map[string]string{
	"increased": "reduced",
	"reduced":   "increased",
	"more":      "less",
	"less":      "more",
}

// antonymFunc ports the file-local antonymFunc: "-25% increased" ->
// "25% reduced"; unknown words keep the minus.
func antonymFunc(num, word string) string {
	if antonym, ok := antonyms[word]; ok {
		return num + " " + antonym
	}
	return "-" + num + " " + word
}

var (
	// %-(%d+%.?%d*%%) (%a+) — the strippedLine antonym pass.
	antonymDecRe = regexp.MustCompile(`-(\d+\.?\d*%) ([A-Za-z]+)`)
	// %-(%d+%%) (%a+) — the fallback path's integer-percent antonym pass.
	antonymIntRe = regexp.MustCompile(`-(\d+%) ([A-Za-z]+)`)
)

// applyValueScalar ports itemLib.applyValueScalar. numbers < 0 means no
// limit (Lua nil). precision == nil means unset.
func applyValueScalar(line string, valueScalar float64, baseValueScalar float64, numbers int, precision *int) string {
	if valueScalar == 1 && baseValueScalar == 1 {
		return line
	}
	power := 1.0
	if precision != nil {
		power = math.Pow(10, float64(*precision))
	}
	scaleValue := func(num string) string {
		value, _ := strconv.ParseFloat(num, 64)
		if baseValueScalar != 1 {
			value = util.RoundHalfUp(value*baseValueScalar*power, 0) / power
		}
		bump := 0.001
		if precision != nil {
			bump = 0
		}
		return util.FormatIntOrG14(math.Floor(value*valueScalar*power+bump) / power)
	}
	if precision != nil {
		// line:gsub("(%d+%.?%d*)", scaleValue, numbers)
		return gsubLimitFunc(line, `\d+\.?\d*`, numbers, func(caps []string) string {
			return scaleValue(caps[0])
		})
	}
	// line:gsub("(%d+)([^%.])", ..., numbers): digits followed by a non-dot
	// character; the suffix character is consumed with the match.
	return gsubLimitFunc(line, `(\d+)([^.])`, numbers, func(caps []string) string {
		return scaleValue(caps[1]) + caps[2]
	})
}

// gsubLimitFunc is string.gsub with a replacement limit (limit < 0 = all).
// caps[0] is the whole match; caps[i] the submatches.
func gsubLimitFunc(s, pattern string, limit int, repl func(caps []string) string) string {
	re := regexp.MustCompile(pattern)
	var sb strings.Builder
	pos := 0
	count := 0
	for pos <= len(s) {
		if limit >= 0 && count >= limit {
			break
		}
		loc := re.FindStringSubmatchIndex(s[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		sb.WriteString(s[pos:start])
		caps := make([]string, len(loc)/2)
		for i := 0; i < len(loc)/2; i++ {
			if loc[2*i] < 0 {
				caps[i] = ""
			} else {
				caps[i] = s[pos+loc[2*i] : pos+loc[2*i+1]]
			}
		}
		sb.WriteString(repl(caps))
		count++
		if end == start { // empty match safety; the patterns here never hit it
			if start < len(s) {
				sb.WriteByte(s[start])
			}
			pos = start + 1
		} else {
			pos = end
		}
	}
	if pos < len(s) {
		sb.WriteString(s[pos:])
	}
	return sb.String()
}

// formatValue ports itemLib.formatValue. value arrives as the captured
// string (Lua coerces on arithmetic).
func formatValue(valueStr string, baseValueScalar, valueScalar float64, precision int, displayPrecision *int, ifRequired bool) string {
	value, _ := strconv.ParseFloat(valueStr, 64)
	value = roundSymmetric(value * float64(precision))
	if baseValueScalar != 1 {
		value = alwaysPositiveRound(value * baseValueScalar)
	}
	if valueScalar != 1 {
		value = floorSymmetric(value * valueScalar)
	}
	value = value / float64(precision)
	if displayPrecision != nil {
		value = roundSymmetricDec(value, *displayPrecision)
	}
	if displayPrecision != nil && !ifRequired {
		return strconv.FormatFloat(value, 'f', *displayPrecision, 64)
	} else if displayPrecision != nil {
		return util.FormatIntOrG14(value)
	}
	// tostring(roundSymmetric(value, min(2, floor(log10(precision)+0.001))))
	dec := imin(2, int(math.Floor(math.Log10(float64(precision))+0.001)))
	return util.FormatIntOrG14(roundSymmetricDec(value, dec))
}

var (
	zeroLeadRe  = regexp.MustCompile(`^\+?0%? `)
	zeroMidRe   = regexp.MustCompile(` \+?0%? `)
	zeroToPosRe = regexp.MustCompile(`0 to [1-9]`)
	zeroPctToRe = regexp.MustCompile(`0% to \d+%`)
)

// isZeroValueLine ports itemLib.isZeroValueLine.
func isZeroValueLine(line string) bool {
	if zeroLeadRe.MatchString(line) {
		return true
	}
	if zeroMidRe.MatchString(line) && !zeroToPosRe.MatchString(line) && !zeroPctToRe.MatchString(line) {
		return true
	}
	return strings.Contains(line, " 0-0 ") || strings.Contains(line, " 0 to 0 ")
}

// rangeShellRe is ([%+-]?)%((%-?%d+%.?%d*)%-(%-?%d+%.?%d*)%): an optional
// sign then a "(min-max)" shell.
var rangeShellRe = regexp.MustCompile(`([+-]?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)

// plusRangeShellRe is (%+?)%(...%-...%): the fallback path's variant that
// only consumes a plus sign.
var plusRangeShellRe = regexp.MustCompile(`(\+?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)

// applyRange ports itemLib.applyRange for a numeric range (the table form
// is only reachable from crafting UI paths). valueScalar and
// baseValueScalar are neutral at 1 (Lua nil coerces there).
func applyRange(line string, rng float64, valueScalar, baseValueScalar float64) string {
	var values []string
	strippedLine := gsubLimitFunc(line, rangeShellRe.String(), -1, func(caps []string) string {
		sign, minS, maxS := caps[1], caps[2], caps[3]
		minV, _ := strconv.ParseFloat(minS, 64)
		maxV, _ := strconv.ParseFloat(maxS, 64)
		value := minV + rng*(maxV-minV)
		if sign == "-" {
			value = -value
		}
		if sign == "+" && value > 0 {
			return sign + util.FormatIntOrG14(value)
		}
		return util.FormatIntOrG14(value)
	})
	strippedLine = gsubLimitFunc(strippedLine, antonymDecRe.String(), -1, func(caps []string) string {
		return antonymFunc(caps[1], caps[2])
	})
	strippedLine = gsubLimitFunc(strippedLine, signedNumRe.String(), -1, func(caps []string) string {
		values = append(values, caps[0])
		return "#"
	})

	scalableLine, scalableValues, found := findScalableLine(strippedLine, values)

	if found {
		key := strings.ReplaceAll(scalableLine, "+#", "#")
		for i, scalability := range data.ModScalability[key] {
			if i >= len(scalableValues) {
				break
			}
			var precision *int
			var displayPrecision *int
			ifRequired := false
			setP := func(p int) { precision = &p }
			setDP := func(p int) { displayPrecision = &p }
			for _, format := range scalability.Formats {
				switch format {
				case "divide_by_two_0dp":
					setP(2)
					setDP(0)
					ifRequired = true
				case "divide_by_three":
					setP(3)
				case "divide_by_four":
					setP(4)
				case "divide_by_five":
					setP(5)
				case "divide_by_six":
					setP(6)
				case "divide_by_ten_0dp":
					setP(10)
					setDP(0)
				case "divide_by_ten_1dp":
					setP(10)
					setDP(1)
				case "divide_by_ten_1dp_if_required":
					setP(10)
					setDP(1)
					ifRequired = true
				case "divide_by_twelve":
					setP(12)
				case "divide_by_fifteen_0dp":
					setP(15)
					setDP(0)
				case "divide_by_twenty":
					setP(20)
				case "divide_by_twenty_then_double_0dp":
					setP(10)
					setDP(0)
				case "divide_by_one_hundred", "divide_by_one_hundred_and_negate":
					setP(100)
				case "divide_by_one_hundred_0dp":
					setP(100)
					setDP(0)
				case "divide_by_one_hundred_1dp":
					setP(100)
					setDP(1)
				case "divide_by_one_hundred_2dp":
					setP(100)
					setDP(2)
				case "divide_by_one_hundred_2dp_if_required":
					setP(100)
					setDP(2)
					ifRequired = true
				case "divide_by_one_thousand":
					setP(1000)
				case "divide_by_ten_thousand_1dp":
					setP(10000)
					setDP(1)
				case "per_minute_to_per_second":
					setP(60)
				case "per_minute_to_per_second_0dp":
					setP(60)
					setDP(0)
				case "per_minute_to_per_second_1dp":
					setP(60)
					setDP(1)
				case "per_minute_to_per_second_2dp":
					setP(60)
					setDP(2)
				case "per_minute_to_per_second_2dp_if_required":
					setP(60)
					setDP(2)
					ifRequired = true
				case "milliseconds_to_seconds", "milliseconds_to_seconds_halved":
					setP(1000)
				case "milliseconds_to_seconds_0dp":
					setP(1000)
					setDP(0)
				case "milliseconds_to_seconds_1dp":
					setP(1000)
					setDP(1)
				case "milliseconds_to_seconds_2dp":
					setP(1000)
					setDP(2)
				case "locations_to_metres":
					setP(10)
					setDP(1)
				case "milliseconds_to_seconds_2dp_if_required":
					setP(1000)
					setDP(2)
					ifRequired = true
				case "deciseconds_to_seconds":
					setP(10)
				}
			}
			p := 1
			if precision != nil {
				p = *precision
			}
			if scalability.IsScalable && (baseValueScalar != 1 || valueScalar != 1) {
				scalableValues[i] = formatValue(scalableValues[i], baseValueScalar, valueScalar, p, displayPrecision, ifRequired)
			} else {
				scalableValues[i] = formatValue(scalableValues[i], 1, 1, p, displayPrecision, ifRequired)
			}
		}
		for _, replacement := range scalableValues {
			scalableLine = strings.Replace(scalableLine, "#", replacement, 1)
		}
		return scalableLine
	}

	// Fallback: the pre-scalability-data method.
	precisionSame := true
	testLine := line
	if strings.Contains(line, "-") {
		testLine = gsubLimitFunc(line, plusRangeShellRe.String(), -1, func(caps []string) string {
			plus, minS, maxS := caps[1], caps[2], caps[3]
			minV, _ := strconv.ParseFloat(minS, 64)
			maxV, _ := strconv.ParseFloat(maxS, 64)
			maxPrecision := minV + rng*(maxV-minV)
			minPrecision := math.Floor(maxPrecision + 0.5)
			if minPrecision != maxPrecision {
				precisionSame = false
			}
			if minPrecision < 0 {
				return util.FormatIntOrG14(minPrecision)
			}
			return plus + util.FormatIntOrG14(minPrecision)
		})
		testLine = gsubLimitFunc(testLine, antonymIntRe.String(), -1, func(caps []string) string {
			return antonymFunc(caps[1], caps[2])
		})
	}
	if precisionSame && valueScalar == 1 && baseValueScalar == 1 {
		return testLine
	}

	var precision *int
	mods, extra, parsed := modparser.Parse(testLine)
	if parsed && extra == "" {
		for _, mod := range mods {
			subMod := mod
			if ref, ok := mod.Value.(modparser.ModRef); ok && ref.Mod != nil {
				subMod = ref.Mod
			}
			if _, isNum := subMod.Value.(modparser.Num); isNum {
				if byType, ok := data.HighPrecisionMods[subMod.Name]; ok {
					if p, ok := byType[subMod.Type]; ok {
						pp := p
						precision = &pp
					}
				}
			}
		}
	}
	if precision == nil && regexp.MustCompile(`\d+\.\d*`).MatchString(line) {
		p := data.DefaultHighPrecision
		precision = &p
	}
	numbers := 0
	out := gsubLimitFunc(line, plusRangeShellRe.String(), -1, func(caps []string) string {
		numbers++
		plus, minS, maxS := caps[1], caps[2], caps[3]
		minV, _ := strconv.ParseFloat(minS, 64)
		maxV, _ := strconv.ParseFloat(maxS, 64)
		power := 1.0
		if precision != nil {
			power = math.Pow(10, float64(*precision))
		}
		numVal := math.Floor((minV+rng*(maxV-minV))*power+0.5) / power
		if numVal < 0 {
			return util.FormatIntOrG14(numVal)
		}
		return plus + util.FormatIntOrG14(numVal)
	})
	out = gsubLimitFunc(out, antonymIntRe.String(), -1, func(caps []string) string {
		return antonymFunc(caps[1], caps[2])
	})
	if numbers == 0 && regexp.MustCompile(`\d+\.?\d*%? `).MatchString(out) {
		numbers = 1
	}
	return applyValueScalar(out, valueScalar, baseValueScalar, numbers, precision)
}

// findScalableLine ports the nested findScalableLine: search substitution
// combinations, largest count first, for a key data.modScalability knows.
func findScalableLine(line string, values []string) (string, []string, bool) {
	replaceNth := func(input, replacement string, n int) string {
		count := 0
		return gsubLimitFunc(input, `#`, -1, func(caps []string) string {
			count++
			if count == n {
				return replacement
			}
			return caps[0]
		})
	}

	var check func(start, numSubstitutions int, indices []int) (string, []string, bool)
	check = func(start, numSubstitutions int, indices []int) (string, []string, bool) {
		if len(indices) == numSubstitutions {
			modifiedLine := line
			substituted := 0
			for _, i := range indices {
				modifiedLine = replaceNth(modifiedLine, values[i-1], i-substituted)
				substituted++
			}
			key := strings.ReplaceAll(modifiedLine, "+#", "#")
			if _, ok := data.ModScalability[key]; ok {
				used := map[int]bool{}
				for _, index := range indices {
					used[index] = true
				}
				var remaining []string
				for i, value := range values {
					if !used[i+1] {
						remaining = append(remaining, value)
					}
				}
				return modifiedLine, remaining, true
			}
			return "", nil, false
		}
		for j := start; j <= len(values); j++ {
			indices = append(indices, j)
			if ml, rv, ok := check(j+1, numSubstitutions, indices); ok {
				return ml, rv, ok
			}
			indices = indices[:len(indices)-1]
		}
		return "", nil, false
	}

	for i := len(values); i >= 1; i-- {
		if ml, rv, ok := check(1, i, nil); ok {
			return ml, rv, ok
		}
	}
	key := strings.ReplaceAll(line, "+#", "#")
	if _, ok := data.ModScalability[key]; ok {
		return line, values, true
	}
	return "", nil, false
}
