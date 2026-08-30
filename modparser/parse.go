package modparser

import (
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/MissingL-tter/missingPassives/internal/util"
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

func buildSpecialTable() *scanTable[modsValue] {
	// The reference anchors its literal entries (and the keystone entries added
	// just before) at ModParser.lua:5887-5891, and only AFTERWARDS the gem loop
	// at 6045 adds per-skill entries carrying their own partial anchoring.
	merged := map[string]modsValue{}
	for _, part := range []map[string]modsValue{
		specialModListData, specialModListHand, keystoneSpecialMods(),
	} {
		for k, v := range part {
			merged[k] = v
		}
	}
	anchored := make(map[string]modsValue, len(merged)+len(gemSpecialMods))
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

// value is what a JewelFunc modifier carries: the function (or its factory
// result) plus the type label and the identity FormatValue spells.
func (e jewelFuncEntry) value(key string, c caps) JewelFn {
	if e.factory != nil {
		return JewelFn{Func: e.factory(c), Type: e.typ, ID: JewelFnID(key, c...)}
	}
	return JewelFn{Func: e.nodeFn, Type: e.typ, ID: JewelFnID(key)}
}

// JewelFnID builds a JewelFn.ID from the table key that produced the
// function and the captures its factory closed over. '/' and '}' are folded
// out because the mod text separates params with '/' and closes a tag with
// '}'.
func JewelFnID(key string, caps ...string) string {
	if len(caps) > 0 {
		key += "(" + strings.Join(caps, ",") + ")"
	}
	return jewelIDUnsafe.Replace(key)
}

var jewelIDUnsafe = strings.NewReplacer("/", "_", "}", "_")

// Parse runs the reference's two-pass protocol over one line of modifier text:
// pass 1, then pass 2 when pass 1 produced modifiers but left a remainder.
// recognised is false when the line was not understood (an expected state:
// garbage item text); extra is the unconsumed remainder ("" when everything
// was consumed).
//
// Results are cached per line and deep-copied on every return, exactly as the
// reference's cache wrapper does — callers (SetSource among them) mutate what
// they are given.
func Parse(line string) (mods []*Mod, extra string, recognised bool) {
	parseCacheMu.Lock()
	entry, hit := parseCache[line]
	if !hit {
		// A line the shipped Data/ModCache.lua covers is served from the
		// file, never parsed — exactly PoB's preloaded cache (modcache.go).
		if dec, ok := cacheLookup(line); ok {
			entry = dec
			parseCache[line] = entry
			hit = true
		}
	}
	parseCacheMu.Unlock()
	if !hit {
		entry = parseMod(line, 1)
		if entry.recognised && entry.extra != "" {
			entry = parseMod(line, 2)
		}
		parseCacheMu.Lock()
		parseCache[line] = entry
		parseCacheMu.Unlock()
	}
	if !entry.recognised {
		return nil, entry.extra, false
	}
	out := make([]*Mod, len(entry.mods))
	for i, m := range entry.mods {
		out[i] = m.Clone()
	}
	return out, entry.extra, true
}

type parseResult struct {
	mods       []*Mod
	extra      string
	recognised bool
}

var (
	parseCache   = map[string]parseResult{}
	parseCacheMu sync.Mutex
)

func unrecognised(extra string) parseResult { return parseResult{extra: extra} }

func recognised(mods []*Mod, extra string) parseResult {
	if mods == nil {
		mods = []*Mod{}
	}
	return parseResult{mods: mods, extra: extra, recognised: true}
}

// parseMod ports ModParser.lua's parseMod (line 6580). order decides whether
// the skill name is looked for after (1) or before (2) the modifier name.
func parseMod(line string, order int) parseResult {
	lineLower := strings.ToLower(line)

	// Check if the line describes a jewel radius function; parametric entries
	// are tried as patterns first, then the whole line as an exact key —
	// ModParser.lua:6582-6592.
	for _, pattern := range jewelFuncKeys {
		entry := jewelFuncList[pattern]
		if entry.re == nil {
			continue
		}
		if c := entry.re.FindStringSubmatch(lineLower); c != nil && len(c) > 1 && c[1] != "" {
			return recognised([]*Mod{mod("JewelFunc", List, entry.value(pattern, caps(c[1:])))}, "")
		}
	}
	if entry, ok := jewelFuncList[lineLower]; ok {
		return recognised([]*Mod{mod("JewelFunc", List, entry.value(lineLower, nil))}, "")
	}
	if v, ok := clusterJewelSkills[lineLower]; ok {
		return recognised(v, "")
	}
	if unsupportedModList[lineLower] {
		return recognised([]*Mod{}, line)
	}

	// Check if this is a special modifier — ModParser.lua:6600.
	specialVal, specialFound, specialLine, specialCaps := scan(line, specialT)
	if specialFound && len(specialLine) == 0 {
		result := specialVal.modsFor(caps(specialCaps))
		if result == nil {
			return unrecognised("")
		}
		return recognised(result, "")
	}

	// Check for add-to-cluster-jewel special — ModParser.lua:6610 (original
	// case, not the lowered line).
	if m := addToClusterRe.FindStringSubmatch(line); m != nil {
		return recognised([]*Mod{mod("AddToClusterJewelNode", List, Str(m[1]))}, "")
	}

	line = line + " "

	// Flag/tag specifications at the start of the line — ModParser.lua:6618.
	var preFlag *PatternEntry
	if v, found, rest, preFlagCaps := scan(line, preFlagsT); found {
		preFlag = v.entryFor(caps(preFlagCaps))
		line = rest
	}

	// Skill name at the start of the line.
	var skillTag *PatternEntry
	if v, found, rest, _ := scan(line, preSkillsT); found {
		skillTag, line = v, rest
	}

	// Modifier form.
	form, found, line, formCaps := scan(line, formsT)
	if !found {
		return unrecognised(remainder(line))
	}

	// Tags (per-charge, conditionals) — up to two.
	var modTag, modTag2 *PatternEntry
	if v, found, rest, tagCaps := scan(line, tagsT); found {
		modTag, line = v.entryFor(caps(tagCaps)), rest
		if v2, found2, rest2, tagCaps2 := scan(line, tagsT); found2 {
			modTag2, line = v2.entryFor(caps(tagCaps2)), rest2
		}
	}

	// Modifier name and skill name — ModParser.lua:6656.
	if order == 2 && skillTag == nil {
		if v, found, rest, _ := scan(line, skillsT); found {
			skillTag, line = v, rest
		}
	}
	var modName *PatternEntry
	scanName := func(t *scanTable[nameValue]) bool {
		v, found, rest, _ := scan(line, t)
		line = rest
		if found {
			modName = v.nameEntry()
		}
		return found
	}
	var flagType flagTypeValue
	switch form {
	case formPen:
		if !scanName(penT) {
			return recognised([]*Mod{}, remainder(line))
		}
		_, _, line, _ = scan(line, namesT)
	case formBaseCost:
		if !scanName(baseCostT) {
			return recognised([]*Mod{}, remainder(line))
		}
		_, _, line, _ = scan(line, namesT)
	case formTotalCost:
		if !scanName(costT) {
			return recognised([]*Mod{}, remainder(line))
		}
		_, _, line, _ = scan(line, namesT)
	case formFlag:
		v, found, rest, _ := scan(line, flagTypesT)
		if !found {
			return unrecognised(remainder(rest))
		}
		flagType, line = v, rest
		scanName(namesT)
	default:
		scanName(namesT)
	}
	if order == 1 && skillTag == nil {
		if v, found, rest, _ := scan(line, skillsT); found {
			skillTag, line = v, rest
		}
	}

	// Scan for flags.
	var modFlag *PatternEntry
	if v, found, rest, _ := scan(line, modFlagsT); found {
		modFlag, line = v, rest
	}

	// Find modifier value and type according to form — ModParser.lua:6699.
	var modValue Value = Str(cap1(formCaps, 1))
	if n, ok := util.Tonumber(cap1(formCaps, 1)); ok {
		modValue = Num(n)
	}
	var modValues []Value // per-name values (DMG forms, DOUBLED)
	modType := Base
	var modTypes []ModType // per-name types when DOUBLED
	var modSuffix string
	suffixSet := false
	var modExtraTags *PatternEntry

	scanSuffix := func() {
		v, found, rest, _ := scan(line, suffixT)
		if found {
			modSuffix = v
			suffixSet = true
		}
		line = rest
	}
	localHand := &PatternEntry{Tag: &CondTag{Var: "{Hand}Attack"}}

	switch form {
	case formInc:
		modType = Inc
	case formRed:
		modValue = negateNum(modValue)
		modType = Inc
	case formMore:
		modType = More
	case formLess:
		modValue = negateNum(modValue)
		modType = More
	case formBase:
		scanSuffix()
	case formGain:
		modType = Base
		scanSuffix()
	case formLose:
		modValue = negateNum(modValue)
		modType = Base
		scanSuffix()
	case formGrants: // local
		modType = Base
		modExtraTags = localHand
		scanSuffix()
	case formGrantsGlobal:
		modType = Base
		scanSuffix()
	case formRemoves: // local
		modValue = negateNum(modValue)
		modType = Base
		modExtraTags = localHand
		scanSuffix()
	case formChance:
		// value and type already correct
	case formRegenPercent:
		modName = nameEntryOf(regenTypes[cap1(formCaps, 2)])
		modSuffix, suffixSet = "Percent", true
	case formRegenFlat:
		modName = nameEntryOf(regenTypes[cap1(formCaps, 2)])
	case formDegenPercent:
		modName = nameEntryOf(degenTypes[cap1(formCaps, 2)])
		modSuffix, suffixSet = "Percent", true
	case formDegenFlat:
		modName = nameEntryOf(degenTypes[cap1(formCaps, 2)])
	case formDegen:
		damageType, ok := dmgTypes[cap1(formCaps, 2)]
		if !ok {
			return recognised([]*Mod{}, remainder(line))
		}
		modName = name(damageType + "Degen").nameEntry()
		modSuffix, suffixSet = "", true
	case formDmg, formDmgAttacks, formDmgSpells, formDmgBoth:
		damageType, ok := dmgTypes[cap1(formCaps, 3)]
		if !ok {
			return recognised([]*Mod{}, remainder(line))
		}
		n1, _ := util.Tonumber(cap1(formCaps, 1))
		n2, _ := util.Tonumber(cap1(formCaps, 2))
		modValues = []Value{Num(n1), Num(n2)}
		modName = nameList{damageType + "Min", damageType + "Max"}.nameEntry()
		if modFlag == nil {
			switch form {
			case formDmgAttacks:
				modFlag = &PatternEntry{KeywordFlags: KeywordAttack}
			case formDmgSpells:
				modFlag = &PatternEntry{KeywordFlags: KeywordSpell}
			case formDmgBoth:
				modFlag = &PatternEntry{KeywordFlags: KeywordAttack | KeywordSpell}
			}
		}
	case formFlag:
		if t, isMod := flagType.(FlagTypeMod); isMod {
			modName = name(t.Name).nameEntry()
			modType = t.Type
			modValue = t.Value
		} else {
			modName = name(string(flagType.(flagName))).nameEntry()
			modType = Flag
			modValue = Bool(true)
		}
	case formOverride:
		modType = Override
	case formDoubled:
		// One MORE mod plus a limited multiplier so the doubling cannot stack —
		// ModParser.lua:6795.
		if modName != nil && len(modName.Names) > 0 {
			modNameString := modName.Names[0]
			// The reference writes into its (shared) name table entry.
			for len(modName.Names) < 2 {
				modName.Names = append(modName.Names, "")
			}
			modName.Names[1] = "Multiplier:" + modNameString + "Doubled"
			modTypes = []ModType{More, Override}
			modValues = []Value{Num(100), Num(1)}
			modExtraTags = &PatternEntry{PerModTags: [][]Tag{{
				&MultiplierTag{Var: modNameString + "Doubled", GlobalLimit: opt(100), GlobalLimitKey: modNameString + "DoubledLimit"},
			}}}
		}
	}

	if modName == nil {
		return recognised([]*Mod{}, remainder(line))
	}

	// Combine flags and tags — ModParser.lua:6817.
	ctl := &PatternEntry{}
	var tagList []Tag
	var perModTags [][]Tag
	for _, data := range []*PatternEntry{modName, preFlag, modFlag, modTag, modTag2, skillTag, modExtraTags} {
		if data == nil {
			continue
		}
		ctl.merge(data)
		switch {
		case data.PerModTags != nil:
			perModTags = data.PerModTags
		case data.Tag != nil:
			tagList = append(tagList, data.Tag.Clone())
		case data.TagList != nil:
			for _, tag := range data.TagList {
				tagList = append(tagList, tag.Clone())
			}
		}
	}

	// Generate modifier list — ModParser.lua:6875.
	suffix := modSuffix
	if !suffixSet {
		suffix = ctl.ModSuffix
	}
	var modList []*Mod
	for i, nameStr := range modName.Names {
		typ := modType
		if modTypes != nil && i < len(modTypes) {
			typ = modTypes[i]
		}
		value := modValue
		if modValues != nil {
			value = nil
			if i < len(modValues) {
				value = modValues[i]
			}
		}
		m := &Mod{Name: nameStr + suffix, Type: typ, Value: value, Flags: ctl.Flags, KeywordFlags: ctl.KeywordFlags}
		m.Tags = append(m.Tags, tagList...)
		if i < len(perModTags) {
			m.Tags = append(m.Tags, perModTags[i]...)
		}
		modList = append(modList, m)
	}

	if len(modList) > 0 {
		// Special handling for various modifier types — ModParser.lua:6890.
		switch {
		case ctl.AddToAura:
			for i, effectMod := range modList {
				if ctl.OnlyAddToBanners {
					modList[i] = mod("ExtraAuraEffect", List, ModRef{Mod: effectMod}, &SkillTypeTag{SkillType: SkillTypeBanner})
				} else {
					modList[i] = mod("ExtraAuraEffect", List, ModRef{Mod: effectMod})
				}
			}
		case ctl.NewAura:
			for i, effectMod := range modList {
				tags := effectMod.Tags
				effectMod.Tags = nil
				modList[i] = mod("ExtraAura", List, ModRef{Mod: effectMod, OnlyAllies: ctl.NewAuraOnlyAllies}, tags...)
			}
		case ctl.AddToMinion:
			for i, effectMod := range modList {
				var tags []Tag
				if ctl.PlayerTag != nil {
					tags = append(tags, ctl.PlayerTag)
				}
				if ctl.AddToMinionTag != nil {
					tags = append(tags, ctl.AddToMinionTag)
				}
				tags = append(tags, ctl.PlayerTagList...)
				modList[i] = mod("MinionModifier", List, ModRef{Mod: effectMod}, tags...)
			}
		case ctl.AddToSkill != nil:
			for i, effectMod := range modList {
				modList[i] = mod("ExtraSkillMod", List, ModRef{Mod: effectMod}, ctl.AddToSkill)
			}
		case ctl.ApplyToEnemy:
			for i, effectMod := range modList {
				var tags []Tag
				if ctl.PlayerTag != nil {
					tags = append(tags, ctl.PlayerTag)
				}
				tags = append(tags, ctl.PlayerTagList...)
				newMod := effectMod
				if len(effectMod.Tags) > 0 && ctl.ActorEnemy {
					cp := *effectMod
					cp.Tags = CloneTags(effectMod.Tags)
					if t0, isCond := cp.Tags[0].(*CondTag); isCond {
						t0.Actor = "enemy"
					}
					newMod = &cp
				}
				modList[i] = mod("EnemyModifier", List, ModRef{Mod: newMod}, tags...)
			}
		}
	}
	return recognised(modList, remainder(line))
}

// nameEntryOrNil is a map lookup's nil-safe view.
func nameEntryOf(n nameValue) *PatternEntry {
	if n == nil {
		return nil
	}
	return n.nameEntry()
}

// remainder mirrors `line:match("%S") and line`: nil when only whitespace.
func remainder(line string) string {
	if strings.TrimLeft(line, " \t\n\v\f\r") == "" {
		return ""
	}
	return line
}

func negateNum(v Value) Value {
	if f, ok := v.(Num); ok {
		return -f
	}
	return v
}

var addToClusterRe = regexp.MustCompile(`^Added Small Passive Skills also grant: (.+)$`)
