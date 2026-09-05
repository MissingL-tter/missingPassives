// Port of .archive/src/Export/statdesc.lua: the stat-description parser and
// the describeStats/describeMod formatting engine most export scripts run on.

package export

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// statVal is one stat's value during describeStats: min/max plus the format
// each transform selects. Entries alias the caller's stats map, as the Lua
// tables do, so transforms mutate the caller's values.
type statVal struct {
	min, max   float64
	minS, maxS string // set by mod_value_to_item_class (fmt "s")
	fmt        string
	minZ, maxZ bool
}

type limitVal struct {
	kind byte // 'n' number, '#', '!' (negated, kind on [0] only), 0 absent
	num  float64
}

// descSpecial is one descriptor-line transform ("negate 1", "canonical_line"):
// the name and its numeric stat-slot argument when it has one.
type descSpecial struct {
	name   string
	arg    float64
	hasArg bool
}

// descLine is one language line of a descriptor ("1|# ... "text" specials").
type descLine struct {
	text     string
	limits   [][2]limitVal
	specials []descSpecial
	quality  map[string]bool // desc[quality] = true for gem_quality lines
}

// statDescriptor is one "description" block; Lang holds the English lines
// (descriptor[1] in the Lua — later `lang` blocks parse into detached
// tables).
type statDescriptor struct {
	lang  []*descLine
	order float64
	name  string
	stats []string
}

var (
	reInclude    = regexp.MustCompile(`include "Metadata/StatDescriptions/(.+)"$`)
	reNoDesc     = regexp.MustCompile(`no_description ([0-9A-Za-z_+\-%]+)`)
	reDescName   = regexp.MustCompile(`description ([0-9A-Za-z_]+)`)
	reStatsLine  = regexp.MustCompile(`[0-9]+\s+([0-9A-Za-z_+\-% ]+)`)
	reStatWord   = regexp.MustCompile(`[0-9A-Za-z_+\-%]+`)
	reLangLine   = regexp.MustCompile(`lang "(.+)"`)
	reDescLine   = regexp.MustCompile(`([0-9\-#| !]+)\s*([0-9A-Za-z_]*)\s*"(.*?)"\s*(.*)`)
	reLimitTok   = regexp.MustCompile(`[!0-9\-#|]+`)
	reLimitNum   = regexp.MustCompile(`^-?[0-9]+$`)
	reLimitNeg   = regexp.MustCompile(`^!(-?[0-9]+)$`)
	reLimitRange = regexp.MustCompile(`([0-9\-#]+)\|([0-9\-#]+)`)
	reSpecialTok = regexp.MustCompile(`[0-9A-Za-z%_]+`)
	reLine       = regexp.MustCompile(`[^\r\n]+`)

	reEscTag    = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	reEscPlain  = regexp.MustCompile(`\[([^|\]]+)\]`)
	reEscChoice = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)

	reFmtNum   = regexp.MustCompile(`\{([0-9])\}`)
	reFmtColon = regexp.MustCompile(`\{([0-9]?):(\+?)d?\}`)
	reDescSeg  = regexp.MustCompile(`([^\\]+)\\n`)

	// %b{} in the Lua; the format strings never nest braces.
	reBraceTok = regexp.MustCompile(`[+\-]?(\{[^{}]*\})`)
	reDigit    = regexp.MustCompile(`[0-9]`)
)

// escapeGGGString matches Common.lua's escapeGGGString.
func escapeGGGString(text string) string {
	line := reEscTag.ReplaceAllString(text, "$1")
	line = reEscPlain.ReplaceAllString(line, "$1")
	line = reEscChoice.ReplaceAllString(line, "$1")
	return line
}

// round matches Common.lua's round.
func round(val float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(val*p+0.5) / p
}

type statDescState struct {
	current     map[string]*statDescriptor
	byFile      map[string]map[string]*statDescriptor
	currentFile string
}

func (x *Ctx) statdesc() *statDescState {
	if x.sd == nil {
		x.sd = &statDescState{byFile: map[string]map[string]*statDescriptor{}}
	}
	return x.sd
}

// parseStatFile ports statdesc.lua's parseStatFile.
func (x *Ctx) parseStatFile(target map[string]*statDescriptor, order float64, fileName string) float64 {
	var curLang *[]*descLine
	curDescriptor := &statDescriptor{}

	var processLine func(line string)
	processLine = func(line string) {
		if m := reInclude.FindStringSubmatch(line); m != nil {
			text := convertUTF16to8([]byte(x.GetFile("Metadata/StatDescriptions/"+m[1])), 0)
			for _, l := range reLine.FindAllString(text, -1) {
				processLine(l)
			}
			return
		}
		if m := reNoDesc.FindStringSubmatch(line); m != nil {
			target[m[1]] = &statDescriptor{order: 0}
			return
		}
		if strings.Contains(line, "handed_description") ||
			(strings.Contains(line, "description") && !strings.Contains(line, "_description")) {
			curDescriptor = &statDescriptor{order: order}
			if m := reDescName.FindStringSubmatch(line); m != nil {
				curDescriptor.name = m[1]
			}
			curLang = &curDescriptor.lang
			order++
			return
		}
		if curDescriptor.stats == nil {
			if m := reStatsLine.FindStringSubmatch(line); m != nil {
				curDescriptor.stats = []string{}
				for _, stat := range reStatWord.FindAllString(m[1], -1) {
					curDescriptor.stats = append(curDescriptor.stats, stat)
					target[stat] = curDescriptor
				}
			}
			return
		}
		if reLangLine.MatchString(line) {
			// The Lua replaces curLang with a table it never attaches, so
			// non-English lines are parsed and dropped.
			detached := []*descLine{}
			curLang = &detached
			return
		}
		if strings.Contains(line, "table_only") {
			return
		}
		m := reDescLine.FindStringSubmatch(line)
		if m == nil {
			return
		}
		statLimits, quality, text, special := m[1], m[2], m[3], m[4]
		desc := &descLine{text: escapeGGGString(text)}
		for _, statLimit := range reLimitTok.FindAllString(statLimits, -1) {
			var limit [2]limitVal
			if statLimit == "#" {
				limit[0] = limitVal{kind: '#'}
				limit[1] = limitVal{kind: '#'}
			} else if reLimitNum.MatchString(statLimit) {
				n, _ := strconv.ParseFloat(statLimit, 64)
				limit[0] = limitVal{kind: 'n', num: n}
				limit[1] = limitVal{kind: 'n', num: n}
			} else if neg := reLimitNeg.FindStringSubmatch(statLimit); neg != nil {
				n, _ := strconv.ParseFloat(neg[1], 64)
				limit[0] = limitVal{kind: '!'}
				limit[1] = limitVal{kind: 'n', num: n}
			} else if r := reLimitRange.FindStringSubmatch(statLimit); r != nil {
				for i := 0; i < 2; i++ {
					if n, err := strconv.ParseFloat(r[i+1], 64); err == nil {
						limit[i] = limitVal{kind: 'n', num: n}
					} else {
						limit[i] = limitVal{kind: '#'} // "#" or unparseable
						if r[i+1] != "#" {
							limit[i] = limitVal{kind: 0}
						}
					}
				}
			}
			desc.limits = append(desc.limits, limit)
		}
		tokens := reSpecialTok.FindAllString(special, -1)
		for ti := 0; ti < len(tokens); {
			token := tokens[ti]
			if token == "canonical_line" {
				desc.specials = append(desc.specials, descSpecial{name: "canonical_line"})
				ti++
			} else if ti+1 < len(tokens) {
				sp := descSpecial{name: token}
				if n, err := strconv.ParseFloat(tokens[ti+1], 64); err == nil {
					sp.arg, sp.hasArg = n, true
				}
				desc.specials = append(desc.specials, sp)
				ti += 2
			} else {
				ti++
			}
		}
		if strings.Contains(quality, "gem_quality") {
			if desc.quality == nil {
				desc.quality = map[string]bool{}
			}
			desc.quality[quality] = true
		}
		*curLang = append(*curLang, desc)
	}

	text := convertUTF16to8([]byte(x.GetFile("Metadata/StatDescriptions/"+fileName)), 0)
	for _, l := range reLine.FindAllString(text, -1) {
		processLine(l)
	}
	return order
}

func getNextOrder(target map[string]*statDescriptor) float64 {
	nextOrder := 1.0
	for _, d := range target {
		if d.order >= nextOrder {
			nextOrder = d.order + 1
		}
	}
	return nextOrder
}

// LoadStatFile ports statdesc.lua's loadStatFile, including the multi-file
// append form loadStatFile(base, extra...).
func (x *Ctx) LoadStatFile(names ...string) {
	sd := x.statdesc()
	x.loadStatFile(names[0], false)
	for _, name := range names[1:] {
		x.loadStatFile(name, true)
	}
	_ = sd
}

func (x *Ctx) loadStatFile(fileName string, appendTo bool) {
	sd := x.statdesc()
	if appendTo && sd.current != nil {
		base := sd.current
		startOrder := getNextOrder(base)
		newDescriptor := map[string]*statDescriptor{}
		x.parseStatFile(newDescriptor, startOrder, fileName)
		cached := map[string]*statDescriptor{}
		copies := map[*statDescriptor]*statDescriptor{}
		normalised := map[*statDescriptor]bool{}
		for stat, descriptor := range newDescriptor {
			if descriptor.order > 0 && !normalised[descriptor] {
				descriptor.order = descriptor.order - startOrder + 1
				if descriptor.order < 1 {
					descriptor.order = 1
				}
				normalised[descriptor] = true
			}
			base[stat] = descriptor
			cp := copies[descriptor]
			if cp == nil {
				c := *descriptor // the Lua copy is shallow
				cp = &c
				if cp.order > 0 {
					// Archive parity: the Lua adjusts the cached copy's
					// order a second time (already-normalised input), leaving the
					// per-file cache with skewed orders.
					cp.order = cp.order - startOrder + 1
				}
				copies[descriptor] = cp
			}
			cached[stat] = cp
		}
		sd.byFile[fileName] = cached
		return
	}
	if c := sd.byFile[fileName]; c != nil {
		sd.current = c
		return
	}
	target := map[string]*statDescriptor{}
	sd.current = target
	sd.byFile[fileName] = target
	x.parseStatFile(target, 1, fileName)
}

// GetStatDescriptors ports getStatDescriptors.
func (x *Ctx) GetStatDescriptors(fileName string) map[string]*statDescriptor {
	sd := x.statdesc()
	if sd.byFile[fileName] == nil {
		x.loadStatFile(fileName, false)
	}
	return sd.byFile[fileName]
}

func matchLimit(lang []*descLine, val []*statVal) *descLine {
	for _, desc := range lang {
		match := true
		for i, limit := range desc.limits {
			if limit[0].kind == '!' {
				if val[i].min == limit[1].num {
					match = false
					break
				}
			} else if (limit[1].kind == 'n' && val[i].min > limit[1].num) ||
				(limit[0].kind == 'n' && val[i].min < limit[0].num) {
				match = false
				break
			}
		}
		if match {
			return desc
		}
	}
	return nil
}

// format is string.format("%"..prefix..fmt, v) for the format kinds
// describeStats produces: d, g, .2f, s, optionally with a "+" prefix.
func (v *statVal) format(which byte, prefix string) (string, error) {
	var num float64
	var str string
	isStr := false
	if which == 'n' {
		num, str, isStr = v.min, v.minS, v.fmt == "s"
	} else {
		num, str, isStr = v.max, v.maxS, v.fmt == "s"
	}
	if isStr {
		return str, nil
	}
	switch v.fmt {
	case "d":
		return fmt.Sprintf("%"+prefix+"d", int64(num)), nil
	case "g":
		return fmt.Sprintf("%"+prefix+".6g", num), nil
	case ".2f":
		return fmt.Sprintf("%"+prefix+".2f", num), nil
	}
	return "", fmt.Errorf("unknown stat format %q", v.fmt)
}

// formatRange is "(min-max)" with each end formatted by prefix.
func (v *statVal) formatRange(prefix string) (string, error) {
	mn, err := v.format('n', prefix)
	if err != nil {
		return "", err
	}
	mx, err := v.format('x', prefix)
	if err != nil {
		return "", err
	}
	return "(" + mn + "-" + mx + ")", nil
}

func formatMinMax(v *statVal, prefix string) (string, error) {
	if v.fmt == "s" {
		if v.minS == v.maxS {
			return v.minS, nil
		}
		return "(" + v.minS + "-" + v.maxS + ")", nil
	}
	if v.min == v.max {
		return v.format('n', prefix)
	}
	if prefix == "+" {
		if v.max < 0 {
			neg := &statVal{min: -v.min, max: -v.max, fmt: v.fmt}
			r, err := neg.formatRange("")
			return "-" + r, err
		}
		r, err := v.formatRange("")
		return "+" + r, err
	}
	return v.formatRange(prefix)
}

// StatLines is describeStats' result: the description lines, their orders,
// and (from DescribeMod) the mod tags text.
type StatLines struct {
	Lines   []string
	Orders  []float64
	ModTags []string
}

// DescribeStats ports describeStats. stats maps stat ids to their values;
// entries are mutated by the description transforms exactly as in the Lua.
func (x *Ctx) DescribeStats(stats map[string]*statVal) (StatLines, error) {
	sd := x.statdesc()
	var out StatLines
	descriptors := map[*statDescriptor]bool{}
	for s, v := range stats {
		if s != "Type" && (v.min != 0 || v.max != 0) {
			if d := sd.current[s]; d != nil && d.stats != nil {
				descriptors[d] = true
			}
		}
	}
	descOrdered := make([]*statDescriptor, 0, len(descriptors))
	for d := range descriptors {
		descOrdered = append(descOrdered, d)
	}
	sort.Slice(descOrdered, func(a, b int) bool { return descOrdered[a].order < descOrdered[b].order })
	for _, descriptor := range descOrdered {
		val := make([]*statVal, len(descriptor.stats))
		for i, s := range descriptor.stats {
			if sv := stats[s]; sv != nil {
				val[i] = sv
			} else {
				val[i] = &statVal{}
			}
			val[i].fmt = "d"
		}
		desc := matchLimit(descriptor.lang, val)
		// Hack to handle ranges starting or ending at 0 where no descriptor
		// is defined for 0, as in the Lua.
		if desc == nil {
			for _, s := range val {
				if s.min == 0 && s.max > 0 {
					s.min = 1
					s.minZ = true
				} else if s.min < 0 && s.max == 0 {
					s.max = -1
					s.maxZ = true
				}
			}
			desc = matchLimit(descriptor.lang, val)
			for _, s := range val {
				if s.minZ {
					s.min = 0
				}
				if s.maxZ {
					s.max = 0
				}
			}
		}
		if desc == nil {
			continue
		}
		for _, spec := range desc.specials {
			vi := func() *statVal { return val[int(spec.arg)-1] }
			switch spec.name {
			case "negate":
				v := vi()
				v.max, v.min = -v.min, -v.max
			case "invert_chance":
				v := vi()
				v.max, v.min = 100-v.min, 100-v.max
			case "negate_and_double":
				v := vi()
				v.max, v.min = -2*v.min, -2*v.max
			case "passive_hash":
				v := vi()
				if v.min < 0 {
					v.min += 65536
					v.max += 65536
				}
			case "divide_by_two_0dp":
				v := vi()
				v.min, v.max = v.min/2, v.max/2
			case "divide_by_ten_0dp":
				v := vi()
				v.min, v.max = v.min/10, v.max/10
			case "divide_by_fifteen_0dp":
				v := vi()
				v.min, v.max = v.min/15, v.max/15
			case "divide_by_four":
				v := vi()
				v.min, v.max, v.fmt = v.min/4, v.max/4, "g"
			case "divide_by_five":
				v := vi()
				v.min, v.max, v.fmt = v.min/5, v.max/5, "g"
			case "divide_by_six":
				v := vi()
				v.min, v.max, v.fmt = v.min/6, v.max/6, "g"
			case "divide_by_ten_1dp_if_required", "divide_by_ten_1dp":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/10, 1), round(v.max/10, 1), "g"
			case "divide_by_twelve":
				v := vi()
				v.min, v.max, v.fmt = v.min/12, v.max/12, "g"
			case "divide_by_twenty":
				v := vi()
				v.min, v.max, v.fmt = v.min/20, v.max/20, "g"
			case "divide_by_one_hundred":
				v := vi()
				v.min, v.max, v.fmt = v.min/100, v.max/100, "g"
			case "divide_by_one_hundred_1dp":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/100, 1), round(v.max/100, 1), "g"
			case "divide_by_one_hundred_2dp", "divide_by_one_hundred_2dp_if_required":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/100, 2), round(v.max/100, 2), "g"
			case "divide_by_one_thousand":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/1000, 1), round(v.max/1000, 1), "g"
			case "per_minute_to_per_second":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/60, 1), round(v.max/60, 1), "g"
			case "permyriad_per_minute_to_%_per_second":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/60/100, 1), round(v.max/60/100, 1), "g"
			case "per_minute_to_per_second_2dp_if_required", "per_minute_to_per_second_2dp":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/60, 2), round(v.max/60, 2), "g"
			case "per_minute_to_per_second_0dp":
				v := vi()
				v.min, v.max = v.min/60, v.max/60
			case "per_minute_to_per_second_1dp":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/60, 1), round(v.max/60, 1), "g"
			case "milliseconds_to_seconds":
				v := vi()
				v.min, v.max, v.fmt = v.min/1000, v.max/1000, "g"
			case "milliseconds_to_seconds_0dp":
				v := vi()
				v.min, v.max = v.min/1000, v.max/1000
			case "milliseconds_to_seconds_2dp_if_required", "milliseconds_to_seconds_2dp":
				v := vi()
				v.min, v.max, v.fmt = round(v.min/1000, 2), round(v.max/1000, 2), "g"
			case "deciseconds_to_seconds":
				v := vi()
				v.min, v.max, v.fmt = v.min/10, v.max/10, ".2f"
			case "locations_to_metres":
				v := vi()
				v.min, v.max, v.fmt = v.min/10, v.max/10, "g"
			case "60%_of_value":
				v := vi()
				v.min, v.max = v.min*0.6, v.max*0.6
			case "30%_of_value":
				v := vi()
				v.min, v.max = v.min*0.3, v.max*0.3
			case "mod_value_to_item_class":
				// Archive parity: ItemClasses is never defined in the
				// Lua either; reaching this errored there too.
				return out, fmt.Errorf("mod_value_to_item_class hit for %s: ItemClasses is nil in the reference", descriptor.name)
			case "multiplicative_damage_modifier":
				v := vi()
				v.min, v.max = 100+v.min, 100+v.max
			case "multiply_by_four":
				v := vi()
				v.min, v.max = v.min*4, v.max*4
			case "times_one_point_five":
				v := vi()
				v.min, v.max = v.min*1.5, v.max*1.5
			case "times_twenty":
				v := vi()
				v.min, v.max = v.min*20, v.max*20
			case "double":
				v := vi()
				v.min, v.max = v.min*2, v.max*2
			case "plus_two_hundred":
				v := vi()
				v.min, v.max = v.min+200, v.max+200
			case "reminderstring", "canonical_line", "canonical_stat", "_stat":
				// no-op
			default:
				// The Lua ConPrintfs "Unknown description function"; ignore.
			}
		}
		var fmtErr error
		fmtVal := func(v *statVal, prefix string) string {
			s, err := formatMinMax(v, prefix)
			if err != nil && fmtErr == nil {
				fmtErr = err
			}
			return s
		}
		statDesc := reFmtNum.ReplaceAllStringFunc(desc.text, func(tok string) string {
			n, _ := strconv.Atoi(tok[1 : len(tok)-1])
			return fmtVal(val[n], "")
		})
		statDesc = replaceAll(statDesc, "{}", func() string {
			return fmtVal(val[0], "")
		})
		statDesc = reFmtColon.ReplaceAllStringFunc(statDesc, func(tok string) string {
			m := reFmtColon.FindStringSubmatch(tok)
			n := 0
			if m[1] != "" {
				n, _ = strconv.Atoi(m[1])
			}
			return fmtVal(val[n], m[2])
		})
		if fmtErr != nil {
			return out, fmtErr
		}
		statDesc = strings.ReplaceAll(statDesc, "%%", "%")
		// One division per line: `order += 0.1` accumulates error
		// (7258.4000000000015 by the fifth line), while (order*10+i)/10 is
		// the correctly rounded double of order + i/10 - order is always
		// integer-valued, so the numerator is exact.
		for i, seg := range reDescSeg.FindAllStringSubmatch(statDesc+"\\n", -1) {
			out.Lines = append(out.Lines, seg[1])
			out.Orders = append(out.Orders, (descriptor.order*10+float64(i))/10)
		}
	}
	return out, nil
}

// replaceAll replaces every occurrence of the literal token with the callback
// result (gsub with a plain pattern and a function).
func replaceAll(s, token string, repl func() string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, token)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		b.WriteString(repl())
		s = s[i+len(token):]
	}
}

// DescribeModTags ports describeModTags: the tag ids, typed — the
// reference pre-joined them into Lua list-literal text, which the render
// test re-spells (lua-residue.md T4).
func (x *Ctx) DescribeModTags(modTags []*Row) []string {
	out := make([]string, 0, len(modTags))
	for _, row := range modTags {
		out = append(out, row.Str("Id"))
	}
	return out
}

// DescribeMod ports describeMod.
func (x *Ctx) DescribeMod(mod *Row) (StatLines, error) {
	stats := map[string]*statVal{}
	buffTemplateStats := map[string]bool{}
	// The Lua also reads BuffTemplate2, but that is a list column: indexing
	// .Stats on the list table yields nil, so only BuffTemplate1 contributes.
	if bt := mod.Ref("BuffTemplate1"); bt != nil {
		for _, sr := range bt.Refs("Stats") {
			buffTemplateStats[sr.Str("Id")] = true
		}
	}
	for i := 1; i <= 6; i++ {
		if sr := mod.Ref(fmt.Sprintf("Stat%d", i)); sr != nil {
			id := sr.Str("Id")
			if !buffTemplateStats[id] {
				iv := mod.Ivl(fmt.Sprintf("Stat%dValue", i))
				stats[id] = &statVal{min: float64(iv[0]), max: float64(iv[1])}
			}
		}
	}
	out, err := x.DescribeStats(stats)
	if err != nil {
		return out, err
	}
	out.ModTags = x.DescribeModTags(mod.Refs("ImplicitTags"))
	return out, nil
}

// DescribeScalability ports describeScalability.
type scalability struct {
	isScalable bool
	formats    []string
}

func (x *Ctx) DescribeScalability(fileName string) (map[string][]scalability, error) {
	sd := x.statdesc()
	out := map[string][]scalability{}
	stats, err := x.Dat("stats")
	if err != nil {
		return nil, err
	}
	descs := sd.byFile[fileName]
	keys := make([]string, 0, len(descs))
	for k := range descs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, stat := range keys {
		statDescription := descs[stat]
		if statDescription.stats == nil {
			continue
		}
		var scal []bool
		for _, s := range statDescription.stats {
			scal = append(scal, stats.RowByStr("Id", s).Bool("IsScalable"))
		}
		for _, wordings := range statDescription.lang {
			wordingFormats := map[int][]string{}
			for _, format := range wordings.specials {
				if format.hasArg {
					wordingFormats[int(format.arg)] = append(wordingFormats[int(format.arg)], format.name)
				}
			}
			var inOrder []scalability
			strippedLine := reBraceTok.ReplaceAllStringFunc(wordings.text, func(tok string) string {
				statNum := 1
				if d := reDigit.FindString(tok); d != "" {
					n, _ := strconv.Atoi(d)
					statNum = n + 1
				}
				var isScal bool
				if statNum-1 < len(scal) {
					isScal = scal[statNum-1]
				}
				inOrder = append(inOrder, scalability{isScalable: isScal, formats: wordingFormats[statNum]})
				return "#"
			})
			if strings.HasPrefix(strippedLine, "DNT") {
				continue
			}
			if prior, ok := out[strippedLine]; ok {
				// Keep the wording with the fewest format oddities per slot.
				for j := range prior {
					pf := len(prior[j].formats)
					var nf int
					if j < len(inOrder) {
						nf = len(wordingFormats[j+1])
					}
					if pf > nf && j < len(inOrder) {
						prior[j] = inOrder[j]
					}
				}
			} else {
				out[strippedLine] = inOrder
			}
		}
	}
	return out, nil
}
