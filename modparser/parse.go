package modparser

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

// The scan tables, built once. specialModList is assembled exactly as the
// reference does at ModParser.lua:5887-5891: literal entries, the gem-loop and
// keystone entries, then every key anchored with ^...$.
var (
	formsT     = newScanTable("formList", formList, false)
	namesT     = newScanTable("modNameList", modNameList, true)
	modFlagsT  = newScanTable("modFlagList", modFlagList, true)
	preFlagsT  = newScanTable("preFlagList", preFlagList, false)
	tagsT      = newScanTable("modTagList", modTagList, false)
	suffixT    = newScanTable("suffixTypes", suffixTypes, true)
	penT       = newScanTable("penTypes", penTypes, true)
	costT      = newScanTable("costTypes", costTypes, true)
	baseCostT  = newScanTable("baseCostTypes", baseCostTypes, true)
	flagTypesT = newScanTable("flagTypes", flagTypes, false)

	skillsT    = newScanTable("skillNameList", skillNameList, false)
	preSkillsT = newScanTable("preSkillNameList", preSkillNameList, false)

	specialT = buildSpecialTable()

	jewelFuncKeys = sortedJewelKeys()
)

func buildSpecialTable() *scanTable {
	// The reference anchors its literal entries (and the keystone entries added
	// just before) at ModParser.lua:5887-5891, and only AFTERWARDS the gem loop
	// at 6045 adds per-skill entries carrying their own partial anchoring.
	merged := map[string]any{}
	for _, part := range []map[string]any{
		specialModListData, specialModListHand, keystoneSpecialMods(),
	} {
		for k, v := range part {
			merged[k] = v
		}
	}
	anchored := make(map[string]any, len(merged)+len(gemSpecialMods))
	for k, v := range merged {
		anchored["^"+k+"$"] = v
	}
	for k, v := range gemSpecialMods {
		anchored[k] = v
	}
	return newScanTable("specialModList", anchored, false)
}

func sortedJewelKeys() []string {
	keys := make([]string, 0, len(jewelFuncList))
	for k := range jewelFuncList {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// jewelEntryValue is what a JewelFunc modifier carries: the function (or its
// factory result) plus the type label — canonically {"func":…, "type":…}.
func (e jewelFuncEntry) tagValue(c caps) Tag {
	if e.factory != nil {
		return Tag{"func": e.factory(c), "type": e.typ}
	}
	return Tag{"func": e.nodeFn, "type": e.typ}
}

// Parse runs the reference's two-pass protocol over one line of modifier text:
// pass 1, then pass 2 when pass 1 produced modifiers but left a remainder.
// It returns the modifier list (nil when the line was not understood) and the
// unconsumed remainder ("" when everything was consumed).
//
// Results are cached per line and deep-copied on every return, exactly as the
// reference's cache wrapper does — callers (SetSource among them) mutate what
// they are given.
func Parse(line string) (mods []any, extra string) {
	parseCacheMu.Lock()
	entry, hit := parseCache[line]
	parseCacheMu.Unlock()
	if !hit {
		entry.mods, entry.extra = parseMod(line, 1)
		if entry.mods != nil && entry.extra != "" {
			entry.mods, entry.extra = parseMod(line, 2)
		}
		parseCacheMu.Lock()
		parseCache[line] = entry
		parseCacheMu.Unlock()
	}
	if entry.mods == nil {
		return nil, entry.extra
	}
	out := make([]any, len(entry.mods))
	for i, m := range entry.mods {
		if mm, ok := m.(*Mod); ok {
			out[i] = CopyMod(mm)
		} else {
			out[i] = copyAny(m)
		}
	}
	return out, entry.extra
}

type parseResult struct {
	mods  []any
	extra string
}

var (
	parseCache   = map[string]parseResult{}
	parseCacheMu sync.Mutex
)

// parseMod ports ModParser.lua's parseMod (line 6580). order decides whether
// the skill name is looked for after (1) or before (2) the modifier name.
func parseMod(line string, order int) ([]any, string) {
	lineLower := asciiLower(line)

	// Check if the line describes a jewel radius function; parametric entries
	// are tried as patterns first, then the whole line as an exact key —
	// ModParser.lua:6582-6592.
	for _, pattern := range jewelFuncKeys {
		entry := jewelFuncList[pattern]
		if entry.re == nil {
			continue
		}
		if c := entry.re.FindStringSubmatch(lineLower); c != nil && len(c) > 1 && c[1] != "" {
			return []any{mod("JewelFunc", "LIST", entry.tagValue(caps(c[1:])))}, ""
		}
	}
	if entry, ok := jewelFuncList[lineLower]; ok {
		return []any{mod("JewelFunc", "LIST", entry.tagValue(nil))}, ""
	}
	if v, ok := clusterJewelSkills[lineLower]; ok {
		return v.([]any), ""
	}
	if unsupportedModList[lineLower] {
		return []any{}, line
	}

	// Check if this is a special modifier — ModParser.lua:6600.
	specialVal, specialLine, specialCaps := scan(line, specialT)
	if specialVal != nil && len(specialLine) == 0 {
		if f, isFn := specialVal.(fn); isFn {
			result := f(caps(specialCaps))
			if result == nil {
				return nil, ""
			}
			return asModList(result), ""
		}
		return copyModList(asModList(specialVal)), ""
	}

	// Check for add-to-cluster-jewel special — ModParser.lua:6610 (original
	// case, not the lowered line).
	if m := addToClusterRe.FindStringSubmatch(line); m != nil {
		return []any{mod("AddToClusterJewelNode", "LIST", m[1])}, ""
	}

	line = line + " "

	// Flag/tag specifications at the start of the line — ModParser.lua:6618.
	preFlagVal, line, preFlagCaps := scan(line, preFlagsT)
	if f, isFn := preFlagVal.(fn); isFn {
		preFlagVal = f(caps(preFlagCaps))
	}

	// Skill name at the start of the line.
	skillTag, line, _ := scan(line, preSkillsT)

	// Modifier form.
	modFormVal, line, formCaps := scan(line, formsT)
	if modFormVal == nil {
		return nil, remainder(line)
	}
	modForm := modFormVal.(string)

	// Tags (per-charge, conditionals) — up to two.
	modTag, line, tagCaps := scan(line, tagsT)
	if f, isFn := modTag.(fn); isFn {
		modTag = f(caps(tagCaps))
	}
	var modTag2 any
	if modTag != nil {
		modTag2, line, tagCaps = scan(line, tagsT)
		if f, isFn := modTag2.(fn); isFn {
			modTag2 = f(caps(tagCaps))
		}
	}

	// Modifier name and skill name — ModParser.lua:6656.
	if order == 2 && skillTag == nil {
		skillTag, line, _ = scan(line, skillsT)
	}
	var modName any
	var flagVal any
	switch modForm {
	case "PEN":
		modName, line, _ = scan(line, penT)
		if modName == nil {
			return []any{}, remainder(line)
		}
		_, line, _ = scan(line, namesT)
	case "BASECOST":
		modName, line, _ = scan(line, baseCostT)
		if modName == nil {
			return []any{}, remainder(line)
		}
		_, line, _ = scan(line, namesT)
	case "TOTALCOST":
		modName, line, _ = scan(line, costT)
		if modName == nil {
			return []any{}, remainder(line)
		}
		_, line, _ = scan(line, namesT)
	case "FLAG":
		flagVal, line, _ = scan(line, flagTypesT)
		if flagVal == nil {
			return nil, remainder(line)
		}
		modName, line, _ = scan(line, namesT)
	default:
		modName, line, _ = scan(line, namesT)
	}
	if order == 1 && skillTag == nil {
		skillTag, line, _ = scan(line, skillsT)
	}

	// Scan for flags.
	modFlag, line, _ := scan(line, modFlagsT)

	// Find modifier value and type according to form — ModParser.lua:6699.
	var modValue any = cap1(formCaps, 1)
	if n, ok := tonumber(cap1(formCaps, 1)); ok {
		modValue = n
	}
	modType := "BASE"
	var modTypes []any // per-name types when DOUBLED
	var modSuffix string
	suffixSet := false
	var modExtraTags *D

	scanSuffix := func() {
		v, rest, _ := scan(line, suffixT)
		if v != nil {
			modSuffix = v.(string)
			suffixSet = true
		}
		line = rest
	}

	switch modForm {
	case "INC":
		modType = "INC"
	case "RED":
		modValue = negateNum(modValue)
		modType = "INC"
	case "MORE":
		modType = "MORE"
	case "LESS":
		modValue = negateNum(modValue)
		modType = "MORE"
	case "BASE":
		scanSuffix()
	case "GAIN":
		modType = "BASE"
		scanSuffix()
	case "LOSE":
		modValue = negateNum(modValue)
		modType = "BASE"
		scanSuffix()
	case "GRANTS": // local
		modType = "BASE"
		modExtraTags = d(p("tag", Tag{"type": "Condition", "var": "{Hand}Attack"}))
		scanSuffix()
	case "GRANTS_GLOBAL":
		modType = "BASE"
		scanSuffix()
	case "REMOVES": // local
		modValue = negateNum(modValue)
		modType = "BASE"
		modExtraTags = d(p("tag", Tag{"type": "Condition", "var": "{Hand}Attack"}))
		scanSuffix()
	case "CHANCE":
		// value and type already correct
	case "REGENPERCENT":
		modName = regenTypes[cap1(formCaps, 2)]
		modSuffix, suffixSet = "Percent", true
	case "REGENFLAT":
		modName = regenTypes[cap1(formCaps, 2)]
	case "DEGENPERCENT":
		modName = degenTypes[cap1(formCaps, 2)]
		modSuffix, suffixSet = "Percent", true
	case "DEGENFLAT":
		modName = degenTypes[cap1(formCaps, 2)]
	case "DEGEN":
		damageType, ok := dmgTypes[cap1(formCaps, 2)]
		if !ok {
			return []any{}, remainder(line)
		}
		modName = damageType.(string) + "Degen"
		modSuffix, suffixSet = "", true
	case "DMG", "DMGATTACKS", "DMGSPELLS", "DMGBOTH":
		damageType, ok := dmgTypes[cap1(formCaps, 3)]
		if !ok {
			return []any{}, remainder(line)
		}
		n1, _ := tonumber(cap1(formCaps, 1))
		n2, _ := tonumber(cap1(formCaps, 2))
		modValue = []any{n1, n2}
		modName = []any{damageType.(string) + "Min", damageType.(string) + "Max"}
		if modFlag == nil {
			switch modForm {
			case "DMGATTACKS":
				modFlag = d(p("keywordFlags", KeywordFlag.Attack))
			case "DMGSPELLS":
				modFlag = d(p("keywordFlags", KeywordFlag.Spell))
			case "DMGBOTH":
				modFlag = d(p("keywordFlags", KeywordFlag.Attack|KeywordFlag.Spell))
			}
		}
	case "FLAG":
		if t, isTag := flagVal.(Tag); isTag {
			modName = t["name"]
			modType, _ = t["type"].(string)
			modValue = t["value"]
		} else {
			modName = flagVal
			modType = "FLAG"
			modValue = true
		}
	case "OVERRIDE":
		modType = "OVERRIDE"
	case "DOUBLED":
		// One MORE mod plus a limited multiplier so the doubling cannot stack —
		// ModParser.lua:6795.
		var modNameString string
		switch n := modName.(type) {
		case *D:
			if len(n.Arr) > 0 {
				modNameString, _ = n.Arr[0].(string)
				for len(n.Arr) < 2 {
					n.Arr = append(n.Arr, nil)
				}
				n.Arr[1] = "Multiplier:" + modNameString + "Doubled"
			}
		case string:
			modNameString = n
			modName = d(n, "Multiplier:"+n+"Doubled")
		}
		if modNameString != "" {
			modTypes = []any{"MORE", "OVERRIDE"}
			modValue = []any{100.0, 1.0}
			modExtraTags = d(
				Tag{"tag": Tag{"type": "Multiplier", "var": modNameString + "Doubled", "globalLimit": 100.0, "globalLimitKey": modNameString + "DoubledLimit"}},
				p("tag", true),
			)
		}
	}

	if modName == nil {
		return []any{}, remainder(line)
	}

	// Combine flags and tags — ModParser.lua:6817.
	var flags, keywordFlags int64
	var tagList []any
	var perModTags [][]any
	misc := map[string]any{}
	for _, data := range []any{modName, preFlagVal, modFlag, modTag, modTag2, skillTag, modExtraTags} {
		dd := asTable(data)
		if dd == nil {
			continue
		}
		flags |= i64Field(dd.KV, "flags")
		keywordFlags |= i64Field(dd.KV, "keywordFlags")
		if tag, has := dd.KV["tag"]; has {
			if per, ok := perEntryTags(dd, "tag"); ok {
				perModTags = per
			} else if tm := asTag(tag); tm != nil {
				tagList = append(tagList, copyTag(tm))
			}
		} else if tl, has := dd.KV["tagList"]; has {
			if per, ok := perEntryTags(dd, "tagList"); ok {
				perModTags = per
			} else {
				for _, tag := range anyList(tl) {
					if tm := asTag(tag); tm != nil {
						tagList = append(tagList, copyTag(tm))
					}
				}
			}
		}
		for k, v := range dd.KV {
			misc[k] = v
		}
	}

	// Generate modifier list — ModParser.lua:6875.
	var nameList []any
	switch n := modName.(type) {
	case *D:
		nameList = n.Arr
	case []any:
		nameList = n
	default:
		nameList = []any{modName}
	}
	suffix := modSuffix
	if !suffixSet {
		if ms, ok := misc["modSuffix"].(string); ok {
			suffix = ms
		}
	}
	var modList []any
	for i, name := range nameList {
		nameStr, _ := name.(string)
		typ := modType
		if modTypes != nil && i < len(modTypes) {
			typ = modTypes[i].(string)
		}
		value := modValue
		if vl, isList := modValue.([]any); isList {
			if i < len(vl) {
				value = vl[i]
			} else {
				value = nil
			}
		}
		m := &Mod{Name: nameStr + suffix, Type: typ, Value: value, Flags: flags, KeywordFlags: keywordFlags}
		m.Tags = append(m.Tags, tagList...)
		if i < len(perModTags) {
			m.Tags = append(m.Tags, perModTags[i]...)
		}
		modList = append(modList, m)
	}

	if len(modList) > 0 {
		// Special handling for various modifier types — ModParser.lua:6890.
		switch {
		case truthy(misc["addToAura"]):
			for i, effectMod := range modList {
				if truthy(misc["onlyAddToBanners"]) {
					modList[i] = mod("ExtraAuraEffect", "LIST", Tag{"mod": effectMod},
						Tag{"type": "SkillType", "skillType": SkillType.Banner})
				} else {
					modList[i] = mod("ExtraAuraEffect", "LIST", Tag{"mod": effectMod})
				}
			}
		case truthy(misc["newAura"]):
			for i, effectMod := range modList {
				em := effectMod.(*Mod)
				tags := em.Tags
				em.Tags = nil
				value := Tag{"mod": em}
				if v, has := misc["newAuraOnlyAllies"]; has && v != nil {
					value["onlyAllies"] = v
				}
				modList[i] = mod("ExtraAura", "LIST", value, tags...)
			}
		case truthy(misc["addToMinion"]):
			for i, effectMod := range modList {
				var tags []any
				if t, has := misc["playerTag"]; has && t != nil {
					tags = append(tags, t)
				}
				if t, has := misc["addToMinionTag"]; has && t != nil {
					tags = append(tags, t)
				}
				tags = append(tags, anyList(misc["playerTagList"])...)
				modList[i] = mod("MinionModifier", "LIST", Tag{"mod": effectMod}, tags...)
			}
		case truthy(misc["addToSkill"]):
			for i, effectMod := range modList {
				modList[i] = mod("ExtraSkillMod", "LIST", Tag{"mod": effectMod}, misc["addToSkill"])
			}
		case truthy(misc["applyToEnemy"]):
			for i, effectMod := range modList {
				var tags []any
				if t, has := misc["playerTag"]; has && t != nil {
					tags = append(tags, t)
				}
				tags = append(tags, anyList(misc["playerTagList"])...)
				newMod := effectMod
				if em, isMod := effectMod.(*Mod); isMod && len(em.Tags) > 0 && truthy(misc["actorEnemy"]) {
					cp := *em
					cp.Tags = make([]any, len(em.Tags))
					for ti, t := range em.Tags {
						if tt, isTag := t.(Tag); isTag {
							cp.Tags[ti] = copyTag(tt)
						} else {
							cp.Tags[ti] = t
						}
					}
					if t0, isTag := cp.Tags[0].(Tag); isTag {
						t0["actor"] = "enemy"
					}
					newMod = &cp
				}
				modList[i] = mod("EnemyModifier", "LIST", Tag{"mod": newMod}, tags...)
			}
		}
	}
	return modList, remainder(line)
}

// remainder mirrors `line:match("%S") and line`: nil when only whitespace.
func remainder(line string) string {
	if strings.TrimLeft(line, " \t\n\v\f\r") == "" {
		return ""
	}
	return line
}

func negateNum(v any) any {
	if f, ok := v.(float64); ok {
		return -f
	}
	return v
}

// asTable views a scanned value as a mixed table: *D directly, Tag as hash-only.
func asTable(v any) *D {
	switch t := v.(type) {
	case *D:
		return t
	case Tag:
		return &D{KV: t}
	}
	return nil
}

func i64Field(kv map[string]any, key string) int64 {
	if kv == nil {
		return 0
	}
	if n, ok := asInt64(kv[key]); ok {
		return n
	}
	return 0
}

// perEntryTags handles the array-of-per-mod-tags shape (data[1].tag /
// data[1].tagList) — ModParser.lua:6826-6852.
func perEntryTags(dd *D, key string) ([][]any, bool) {
	if len(dd.Arr) == 0 {
		return nil, false
	}
	first := asTable(dd.Arr[0])
	if first == nil {
		return nil, false
	}
	if _, has := first.KV[key]; !has {
		return nil, false
	}
	var out [][]any
	for _, entry := range dd.Arr {
		e := asTable(entry)
		if e == nil {
			break
		}
		var tags []any
		if key == "tag" {
			if t := asTag(e.KV["tag"]); t != nil {
				tags = append(tags, copyTag(t))
			}
		} else {
			for _, t := range anyList(e.KV["tagList"]) {
				if tt := asTag(t); tt != nil {
					tags = append(tags, copyTag(tt))
				}
			}
		}
		out = append(out, tags)
	}
	return out, true
}

// anyList reads a list-ish value: []any directly, a *D's array part.
func anyList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case *D:
		return t.Arr
	}
	return nil
}

func copyTag(t Tag) Tag {
	out := make(Tag, len(t))
	for k, v := range t {
		out[k] = v
	}
	return out
}

// copyModList mirrors the copyTable the reference applies to data-valued
// special entries before returning them.
func copyModList(list []any) []any {
	out := make([]any, len(list))
	copy(out, list)
	return out
}

// asTag views any table-like value as a tag: a Tag map directly, or a *D whose
// hash part carries the fields (the transform stores tags whose strings contain
// literal braces that way).
func asTag(v any) Tag {
	switch t := v.(type) {
	case Tag:
		return t
	case *D:
		if len(t.Arr) == 0 && t.KV != nil {
			return Tag(t.KV)
		}
	}
	return nil
}

// asModList views a closure or table result as a modifier list. An empty Lua
// table arrives as an empty *D.
func asModList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case *D:
		return t.Arr
	case nil:
		return nil
	}
	return []any{v}
}

var addToClusterRe = regexp.MustCompile(`^Added Small Passive Skills also grant: (.+)$`)
