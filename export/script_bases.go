// Port of .archive/src/Export/Scripts/bases.lua.

package export

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "bases", Build: buildBases})
}

var basesItemTypes = []string{
	"axe", "bow", "claw", "dagger", "fishing", "mace", "staff", "sword", "wand",
	"helmet", "body", "gloves", "boots", "shield", "quiver",
	"amulet", "ring", "belt", "jewel", "flask", "tincture", "graft",
}

var (
	reBaseArgs     = regexp.MustCompile(`([0-9A-Za-z/_]+) (.+)`)
	reItExtends    = regexp.MustCompile(`extends "(.+)"`)
	reItRemoveTag  = regexp.MustCompile(`remove_tag = "(.+)"`)
	reItTag        = regexp.MustCompile(`tag = "(.+)"`)
	reBestBase4    = regexp.MustCompile(`^([^,]+), ([^,]+), ([^,]+), \[([^\]]+)\]`)
	reBestBase3    = regexp.MustCompile(`^([^)]+), ([^)]+), \[ ([^)]+)\]`)
	reBaseGroupArg = regexp.MustCompile(`^([^)]+), \[ ([^)]+)\]`)
	reSetBase3     = regexp.MustCompile(`^([^,]+), ([^,]+), \[([^\]]+)\]`)
	reSetBase2     = regexp.MustCompile(`([^,]+), \[([^\]]+)\]`)
	reCommaField   = regexp.MustCompile(`[^,]+`)
	reLeadingSpace = regexp.MustCompile(`^ `)
	reFmtS         = regexp.MustCompile(`%s`)
)

func buildBases(x *Ctx) (any, error) {
	x.LoadStatFile("tincture_stat_descriptions.txt")

	type baseState struct {
		typ, influenceBaseTag string
		subType               *string
		forceShow, forceHide  bool
		socketLimit           *float64
	}
	type bestBase struct {
		displayName  string
		itemValueSum int64
	}
	type allBase struct {
		class   string
		subType *string
	}
	bases := map[string]map[string]bestBase{}
	basesAll := map[string]allBase{}
	baseMods := map[string]string{}

	var getBaseItemTags func(base string) []string
	getBaseItemTags = func(base string) []string {
		if base == "nothing" { // base case
			return []string{}
		}
		raw := x.GetFile(base + ".it")
		if raw == "" {
			return nil
		}
		text := convertUTF16to8([]byte(raw), 0)
		tags := []string{}
		for _, line := range reLine.FindAllString(text, -1) {
			if m := reItExtends.FindStringSubmatch(line); m != nil {
				tags = append(tags, getBaseItemTags(m[1])...)
			} else if strings.Contains(line, "remove_tag") {
				var val string
				if m := reItRemoveTag.FindStringSubmatch(line); m != nil {
					val = m[1]
				}
				idx := -1
				for i, t := range tags {
					if val != "" && t == val {
						idx = i
						break
					}
				}
				if idx >= 0 {
					tags = append(tags[:idx], tags[idx+1:]...)
				} else if len(tags) > 0 {
					// #EVAL: archive parity — a remove_tag for an absent tag is
					// table.remove(tags, nil), which pops the LAST element.
					tags = tags[:len(tags)-1]
				}
			} else if strings.Contains(line, "tag") {
				if m := reItTag.FindStringSubmatch(line); m != nil {
					tags = append(tags, m[1])
				}
			}
		}
		return tags
	}

	var state *baseState

	buildBase := func(args string) *schema.ItemBase {
		baseTypeId := args
		displayName := ""
		if m := reBaseArgs.FindStringSubmatch(args); m != nil {
			baseTypeId, displayName = m[1], m[2]
		}
		baseItemType := x.Dat("BaseItemTypes").GetRow("Id", baseTypeId)
		if baseItemType == nil {
			return nil // the Lua printfs "Invalid Id"
		}
		baseItemTags := getBaseItemTags(luaStr(baseItemType.Get("BaseType")))
		if displayName == "" {
			displayName = luaStr(baseItemType.Get("Name"))
		}
		displayName = strings.ReplaceAll(displayName, "\xc3\xb6", "o")
		displayName = luaTrim(displayName)
		if displayName == "Energy Blade" {
			if state.typ == "One Handed Sword" {
				displayName = "Energy Blade One Handed"
			} else {
				displayName = "Energy Blade Two Handed"
			}
		}
		b := &schema.ItemBase{DisplayName: displayName, Type: state.typ}
		if state.subType != nil && len(*state.subType) > 0 {
			b.SubType = *state.subType
		}
		b.Hidden = state.forceHide && !strings.Contains(baseTypeId, "Talisman") && !state.forceShow
		b.SocketLimit = state.socketLimit
		combinedTags := map[string]bool{}
		for _, tag := range baseItemTags {
			combinedTags[tag] = true
		}
		for _, tag := range listRows(baseItemType.Get("Tags")) {
			combinedTags[luaStr(tag.Get("Id"))] = true
		}
		sortedTags := make([]string, 0, len(combinedTags))
		for t := range combinedTags {
			sortedTags = append(sortedTags, t)
		}
		sort.Strings(sortedTags)
		b.Tags = sortedTags
		b.InfluenceBaseTag = state.influenceBaseTag

		implicitRows := listRows(baseItemType.Get("ImplicitMods"))
		for _, mod := range implicitRows {
			modDesc := x.DescribeMod(mod)
			for _, line := range modDesc.Lines {
				b.Implicit = append(b.Implicit, line)
				b.ImplicitModTypes = append(b.ImplicitModTypes, modDesc.ModTags)
			}
			if len(modDesc.Lines) > 0 {
				b.ImplicitIds = append(b.ImplicitIds, luaStr(mod.Get("Id")))
			}
		}
		enchantRows := listRows(baseItemType.Get("EnchantMods"))
		for _, mod := range enchantRows {
			modDesc := x.DescribeMod(mod)
			for _, line := range modDesc.Lines {
				b.Enchant = append(b.Enchant, line)
				b.EnchantModTypes = append(b.EnchantModTypes, modDesc.ModTags)
			}
			if len(modDesc.Lines) > 0 {
				b.EnchantIds = append(b.EnchantIds, luaStr(mod.Get("Id")))
			}
		}
		b.CannotBeAnointed = len(enchantRows) > 0

		itemValueSum := int64(0)
		weaponType := x.Dat("WeaponTypes").GetRow("BaseItemType", baseItemType)
		if weaponType != nil {
			b.Weapon = &schema.WeaponBase{
				PhysicalMin:    weaponType.Get("DamageMin").(int64),
				PhysicalMax:    weaponType.Get("DamageMax").(int64),
				CritChanceBase: float64(weaponType.Get("CritChance").(int64)) / 100,
				AttackRateBase: round(1000/float64(weaponType.Get("Speed").(int64)), 2),
				Range:          weaponType.Get("Range").(int64),
			}
			itemValueSum = b.Weapon.PhysicalMin + b.Weapon.PhysicalMax
		}
		armourType := x.Dat("ArmourTypes").GetRow("BaseItemType", baseItemType)
		if armourType != nil {
			ab := &schema.ArmourBase{}
			if shield := x.Dat("ShieldTypes").GetRow("BaseItemType", baseItemType); shield != nil {
				v := shield.Get("Block").(int64)
				ab.BlockChance = &v
			}
			set := func(minP, maxP **int64, col string) {
				mn := armourType.Get(col + "Min").(int64)
				mx := armourType.Get(col + "Max").(int64)
				if mn > 0 {
					*minP, *maxP = &mn, &mx
					itemValueSum += mn + mx
				}
			}
			set(&ab.ArmourMin, &ab.ArmourMax, "Armour")
			set(&ab.EvasionMin, &ab.EvasionMax, "Evasion")
			set(&ab.EnergyShieldMin, &ab.EnergyShieldMax, "EnergyShield")
			if mp := armourType.Get("MovementPenalty").(int64); mp != 0 {
				v := -mp
				ab.MovementPenalty = &v
			}
			set(&ab.WardMin, &ab.WardMax, "Ward")
			b.Armour = ab
		}
		flask := x.Dat("Flasks").GetRow("BaseItemType", baseItemType)
		if flask != nil {
			compCharges := x.Dat("ComponentCharges").GetRow("BaseItemType", luaStr(baseItemType.Get("Id")))
			fb := &schema.FlaskBase{
				Duration:    float64(flask.Get("RecoveryTime").(int64)) / 10,
				ChargesUsed: compCharges.Get("PerUse").(int64),
				ChargesMax:  compCharges.Get("Max").(int64),
			}
			if v := flask.Get("LifePerUse").(int64); v > 0 {
				fb.Life = &v
			}
			if v := flask.Get("ManaPerUse").(int64); v > 0 {
				fb.Mana = &v
			}
			if buff, ok := flask.Get("Buff").(*Row); ok {
				fb.HasBuff = true
				stats := map[string]*statVal{}
				mags := flask.Get("BuffMagnitudes").([]any)
				for i, stat := range listRows(buff.Get("Stats")) {
					v := float64(mags[i].(int64))
					stats[luaStr(stat.Get("Id"))] = &statVal{min: v, max: v}
				}
				for _, stat := range listRows(buff.Get("GrantedFlags")) {
					stats[luaStr(stat.Get("Id"))] = &statVal{min: 1, max: 1}
				}
				fb.Buff = x.DescribeStats(stats).Lines
			}
			b.Flask = fb
		}
		tincture := x.Dat("tinctures").GetRow("BaseItemType", baseItemType)
		if tincture != nil {
			b.Tincture = &schema.TinctureBase{
				ManaBurn: float64(tincture.Get("ManaBurn").(int64)) / 1000,
				Cooldown: float64(tincture.Get("CoolDown").(int64)) / 1000,
			}
		}
		reqLevel := int64(1)
		dropLevel := baseItemType.Get("DropLevel").(int64)
		if weaponType != nil || armourType != nil {
			if dropLevel > 4 {
				reqLevel = dropLevel
			}
		}
		if flask != nil {
			if dropLevel > 2 {
				reqLevel = dropLevel
			}
		}
		for _, mod := range implicitRows {
			lvl := int64(math.Floor(float64(mod.Get("Level").(int64)) * 0.8))
			if lvl > reqLevel {
				reqLevel = lvl
			}
		}
		if reqLevel > 1 {
			b.ReqLevel = &reqLevel
		}
		if compAtt := x.Dat("ComponentAttributeRequirements").GetRow("BaseItemType", luaStr(baseItemType.Get("Id"))); compAtt != nil {
			for _, attr := range []struct {
				col string
				dst **int64
			}{{"Str", &b.ReqStr}, {"Dex", &b.ReqDex}, {"Int", &b.ReqInt}} {
				if v := compAtt.Get(attr.col).(int64); v > 0 {
					*attr.dst = &v
				}
			}
		}
		if ft, ok := baseItemType.Get("FlavourTextKey").(*Row); ok {
			if text := luaStr(ft.Get("Text")); text != "" {
				b.FlavourText = cleanAndSplit(text)
			}
		}

		if !b.Hidden {
			if bases[state.typ] == nil {
				bases[state.typ] = map[string]bestBase{}
			}
			subtype := ""
			if state.subType != nil {
				subtype = *state.subType
			}
			if prev, ok := bases[state.typ][subtype]; !ok || itemValueSum > prev.itemValueSum {
				bases[state.typ][subtype] = bestBase{displayName, itemValueSum}
			}
			basesAll[displayName] = allBase{class: state.typ, subType: state.subType}
		}
		return b
	}

	doc := schema.BasesData{Types: map[string][][]schema.ItemBase{}}
	var curEvents *[][]schema.ItemBase

	// While walking the Rares template, mirror the generated file's line
	// stream (directive blobs merged with the hand-written passthrough
	// blobs) to recover the complete rare list.
	inRares := false
	var raresStream []string
	streamBlob := func(lines []string) {
		if !inRares {
			return
		}
		if n := len(raresStream); n > 0 && raresStream[n-1] == "]]," {
			raresStream[n-1] = "]],[["
		} else {
			raresStream = append(raresStream, "[[")
		}
		raresStream = append(raresStream, lines...)
		raresStream = append(raresStream, "]],")
	}

	directives := map[string]func(args string){}
	directives["type"] = func(args string) { state.typ = args }
	directives["subType"] = func(args string) { s := args; state.subType = &s }
	directives["influenceBaseTag"] = func(args string) { state.influenceBaseTag = args }
	directives["forceShow"] = func(args string) { state.forceShow = args == "true" }
	directives["forceHide"] = func(args string) { state.forceHide = args == "true" }
	directives["socketLimit"] = func(args string) {
		if n, err := strconv.ParseFloat(args, 64); err == nil {
			state.socketLimit = &n
		} else {
			state.socketLimit = nil
		}
	}
	directives["base"] = func(args string) {
		var ev []schema.ItemBase
		if b := buildBase(args); b != nil {
			ev = append(ev, *b)
		}
		*curEvents = append(*curEvents, ev)
	}
	directives["baseMatch"] = func(argstr string) {
		key := "Id"
		args := strings.Fields(argstr)
		value := args[0]
		if len(args) > 1 {
			key = args[0]
			value = args[1]
		}
		re := regexp.MustCompile(value) // Go regex in the template
		var ev []schema.ItemBase
		for _, baseItemType := range x.Dat("BaseItemTypes").GetRowListMatch(key, re.MatchString) {
			id := luaStr(baseItemType.Get("Id"))
			if !strings.Contains(id, "Royale") {
				if b := buildBase(id); b != nil {
					ev = append(ev, *b)
				}
			}
		}
		*curEvents = append(*curEvents, ev)
	}
	directives["baseGroup"] = func(args string) {
		if m := reBaseGroupArg.FindStringSubmatch(args); m != nil {
			baseMods[m[1]] = m[2]
		}
	}
	valueLines := func(lines *[]string, values string) {
		for _, line := range reCommaField.FindAllString(values, -1) {
			*lines = append(*lines, reLeadingSpace.ReplaceAllString(line, ""))
		}
	}
	directives["setBestBase"] = func(args string) {
		var baseClass, baseSubType, itemNameOverride, values string
		if m := reBestBase4.FindStringSubmatch(args); m != nil {
			baseClass, baseSubType, itemNameOverride, values = m[1], m[2], m[3], m[4]
		} else if m := reBestBase3.FindStringSubmatch(args); m != nil {
			baseClass, baseSubType, values = m[1], m[2], m[3]
		}
		itemName := itemNameOverride
		if itemName == "" {
			itemName = baseSubType + " " + baseClass
		}
		base := bases[baseClass][baseSubType].displayName
		lines := []string{itemName, base}
		if !strings.Contains(values, "Crafted: true") {
			lines = append(lines, "Crafted: true")
		}
		if values != " " {
			valueLines(&lines, values)
		} else if _, ok := baseMods[itemName]; ok {
			valueLines(&lines, values)
		}
		doc.Rares = append(doc.Rares, &schema.RareItem{Lines: lines})
		streamBlob(lines)
	}
	directives["setBase"] = func(args string) {
		var baseName, itemName, values string
		if m := reSetBase3.FindStringSubmatch(args); m != nil {
			baseName, itemName, values = m[1], m[2], m[3]
		} else if m := reSetBase2.FindStringSubmatch(args); m != nil {
			baseName, values = m[1], m[2]
		}
		all, ok := basesAll[baseName]
		if baseName != "" && !ok {
			// the Lua prints "Missing base"
			doc.Rares = append(doc.Rares, nil)
			return
		}
		var lines []string
		baseClass := all.class
		groupName := baseClass
		if itemName != "" {
			formatted := reFmtS.ReplaceAllString(itemName, baseClass)
			formatted = strings.ReplaceAll(formatted, "One Handed", "1H")
			formatted = strings.ReplaceAll(formatted, "Two Handed", "2H")
			lines = append(lines, formatted)
			hand := ""
			if strings.Contains(baseClass, "One Handed") || strings.Contains(baseClass, "Claw") ||
				strings.Contains(baseClass, "Dagger") || strings.Contains(baseClass, "Sceptre") ||
				strings.Contains(baseClass, "Wand") {
				hand = "One Handed"
			} else if strings.Contains(baseClass, "Two Handed") || strings.Contains(baseClass, "Staff") {
				hand = "Two Handed"
			}
			groupName = reFmtS.ReplaceAllString(itemName, hand)
		} else {
			if all.subType != nil {
				groupName = *all.subType + " " + baseClass
				lines = append(lines, groupName)
			} else {
				lines = append(lines, baseClass)
			}
		}
		lines = append(lines, baseName)
		if !strings.Contains(values, "Crafted: true") {
			lines = append(lines, "Crafted: true")
		}
		if values != " " {
			valueLines(&lines, values)
		} else if group, ok := baseMods[groupName]; ok {
			valueLines(&lines, group)
		}
		doc.Rares = append(doc.Rares, &schema.RareItem{Lines: lines})
		streamBlob(lines)
	}

	for _, name := range append(append([]string{}, basesItemTypes...), "Rares") {
		state = &baseState{}
		var events [][]schema.ItemBase
		curEvents = &events
		if name == "Rares" {
			// The rare list is the directive-generated best-base blobs (in
			// directive order) followed by the template's hand-written
			// blocks — the same order the generated file carries them.
			inRares = true
			err := x.WalkTemplate(name, "Bases/", directives)
			inRares = false
			if err != nil {
				return nil, err
			}
			f := splitUniqueFile(raresStream)
			for _, sec := range f.Sections {
				doc.RareBlobs = append(doc.RareBlobs, sec.Items...)
			}
			tpl, err := readTemplate("Bases/", "Rares")
			if err != nil {
				return nil, err
			}
			doc.RareBlobs = append(doc.RareBlobs, tpl.Items...)
			continue
		}
		if err := x.WalkTemplate(name, "Bases/", directives); err != nil {
			return nil, err
		}
		doc.Types[name] = events
	}
	return doc, nil
}
