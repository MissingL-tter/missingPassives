// Port of .archive/src/Export/Scripts/bases.lua.

package export

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/luapat"
)

func init() {
	outs := []string{"Data/Rares.lua"}
	for _, name := range basesItemTypes {
		outs = append(outs, "Data/Bases/"+name+".lua")
	}
	Scripts = append(Scripts, Script{Name: "bases", Outs: outs, Run: scriptBases})
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

func scriptBases(x *Ctx) error {
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
					// table.remove(tags, nil) pops the last element.
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

	baseDirective := func(args string, out *OutFile) {
		baseTypeId := args
		displayName := ""
		if m := reBaseArgs.FindStringSubmatch(args); m != nil {
			baseTypeId, displayName = m[1], m[2]
		}
		baseItemType := x.Dat("BaseItemTypes").GetRow("Id", baseTypeId)
		if baseItemType == nil {
			return // the Lua printfs "Invalid Id"
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
		out.W("itemBases[\"", displayName, "\"] = {\n")
		out.W("\ttype = \"", state.typ, "\",\n")
		if state.subType != nil && len(*state.subType) > 0 {
			out.W("\tsubType = \"", *state.subType, "\",\n")
		}
		hidden := state.forceHide && !strings.Contains(baseTypeId, "Talisman") && !state.forceShow
		if hidden {
			out.W("\thidden = true,\n")
		}
		if state.socketLimit != nil {
			out.W("\tsocketLimit = ", luaNum(*state.socketLimit), ",\n")
		}
		out.W("\ttags = { ")
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
		for _, tag := range sortedTags {
			out.W(tag, " = true, ")
		}
		out.W("},\n")
		if state.influenceBaseTag != "" {
			out.W("\tinfluenceTags = { ")
			for i, suffix := range []string{"shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie", "cleansing", "tangle"} {
				if i != 0 {
					out.W(", ")
				}
				out.W(suffix, " = \"", state.influenceBaseTag, "_", suffix, "\"")
			}
			out.W(" },\n")
		}
		var implicitLines, implicitModTypes, implicitModIds []string
		implicitRows := listRows(baseItemType.Get("ImplicitMods"))
		for _, mod := range implicitRows {
			modDesc := x.DescribeMod(mod)
			for _, line := range modDesc.Lines {
				implicitLines = append(implicitLines, line)
				implicitModTypes = append(implicitModTypes, modDesc.ModTags)
			}
			if len(modDesc.Lines) > 0 {
				implicitModIds = append(implicitModIds, luaStr(mod.Get("Id")))
			}
		}
		if len(implicitLines) > 0 {
			out.W("\timplicit = \"", strings.Join(implicitLines, "\\n"), "\",\n")
		}
		out.W("\timplicitModTypes = { ")
		for _, t := range implicitModTypes {
			out.W("{ ", t, " }, ")
		}
		out.W("},\n")
		if len(implicitModIds) > 0 {
			out.W("\timplicitIds = ", stringifyInlineStrings(implicitModIds), ",\n")
		}
		var enchantLines, enchantModTypes, enchantModIds []string
		enchantRows := listRows(baseItemType.Get("EnchantMods"))
		for _, mod := range enchantRows {
			modDesc := x.DescribeMod(mod)
			for _, line := range modDesc.Lines {
				enchantLines = append(enchantLines, line)
				enchantModTypes = append(enchantModTypes, modDesc.ModTags)
			}
			if len(modDesc.Lines) > 0 {
				enchantModIds = append(enchantModIds, luaStr(mod.Get("Id")))
			}
		}
		if len(enchantLines) > 0 {
			out.W("\tenchant = \"", strings.Join(enchantLines, "\\n"), "\",\n")
			if len(enchantModIds) > 0 {
				out.W("\tenchantIds = ", stringifyInlineStrings(enchantModIds), ",\n")
			}
			out.W("\tenchantModTypes = { ")
			for _, t := range enchantModTypes {
				out.W("{ ", t, " }, ")
			}
			out.W("},\n")
		}
		if len(enchantRows) > 0 {
			out.W("\tcannotBeAnointed = true,\n")
		}
		itemValueSum := int64(0)
		weaponType := x.Dat("WeaponTypes").GetRow("BaseItemType", baseItemType)
		if weaponType != nil {
			out.W("\tweapon = { ")
			out.W("PhysicalMin = ", weaponType.Get("DamageMin").(int64), ", PhysicalMax = ", weaponType.Get("DamageMax").(int64), ", ")
			out.W("CritChanceBase = ", luaNum(float64(weaponType.Get("CritChance").(int64))/100), ", ")
			out.W("AttackRateBase = ", luaNum(round(1000/float64(weaponType.Get("Speed").(int64)), 2)), ", ")
			out.W("Range = ", weaponType.Get("Range").(int64), ", ")
			out.W("},\n")
			itemValueSum = weaponType.Get("DamageMin").(int64) + weaponType.Get("DamageMax").(int64)
		}
		armourType := x.Dat("ArmourTypes").GetRow("BaseItemType", baseItemType)
		if armourType != nil {
			out.W("\tarmour = { ")
			if shield := x.Dat("ShieldTypes").GetRow("BaseItemType", baseItemType); shield != nil {
				out.W("BlockChance = ", shield.Get("Block").(int64), ", ")
			}
			for _, part := range []struct{ col, name string }{
				{"Armour", "Armour"}, {"Evasion", "Evasion"}, {"EnergyShield", "EnergyShield"},
			} {
				mn := armourType.Get(part.col + "Min").(int64)
				mx := armourType.Get(part.col + "Max").(int64)
				if mn > 0 {
					out.W(part.name, "BaseMin = ", mn, ", ")
					out.W(part.name, "BaseMax = ", mx, ", ")
					itemValueSum += mn + mx
				}
				if part.col == "EnergyShield" {
					if mp := armourType.Get("MovementPenalty").(int64); mp != 0 {
						out.W("MovementPenalty = ", -mp, ", ")
					}
					wn := armourType.Get("WardMin").(int64)
					wx := armourType.Get("WardMax").(int64)
					if wn > 0 {
						out.W("WardBaseMin = ", wn, ", ")
						out.W("WardBaseMax = ", wx, ", ")
						itemValueSum += wn + wx
					}
				}
			}
			out.W("},\n")
		}
		flask := x.Dat("Flasks").GetRow("BaseItemType", baseItemType)
		if flask != nil {
			compCharges := x.Dat("ComponentCharges").GetRow("BaseItemType", luaStr(baseItemType.Get("Id")))
			out.W("\tflask = { ")
			if v := flask.Get("LifePerUse").(int64); v > 0 {
				out.W("life = ", v, ", ")
			}
			if v := flask.Get("ManaPerUse").(int64); v > 0 {
				out.W("mana = ", v, ", ")
			}
			out.W("duration = ", luaNum(float64(flask.Get("RecoveryTime").(int64))/10), ", ")
			out.W("chargesUsed = ", compCharges.Get("PerUse").(int64), ", ")
			out.W("chargesMax = ", compCharges.Get("Max").(int64), ", ")
			if buff, ok := flask.Get("Buff").(*Row); ok {
				stats := map[string]*statVal{}
				mags := flask.Get("BuffMagnitudes").([]any)
				for i, stat := range listRows(buff.Get("Stats")) {
					v := float64(mags[i].(int64))
					stats[luaStr(stat.Get("Id"))] = &statVal{min: v, max: v}
				}
				for _, stat := range listRows(buff.Get("GrantedFlags")) {
					stats[luaStr(stat.Get("Id"))] = &statVal{min: 1, max: 1}
				}
				out.W("buff = { \"", strings.Join(x.DescribeStats(stats).Lines, "\", \""), "\" }, ")
			}
			out.W("},\n")
		}
		tincture := x.Dat("tinctures").GetRow("BaseItemType", baseItemType)
		if tincture != nil {
			out.W("\ttincture = { manaBurn = ", luaNum(float64(tincture.Get("ManaBurn").(int64))/1000),
				", cooldown = ", luaNum(float64(tincture.Get("CoolDown").(int64))/1000), " },\n")
		}
		out.W("\treq = { ")
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
			out.W("level = ", reqLevel, ", ")
		}
		if compAtt := x.Dat("ComponentAttributeRequirements").GetRow("BaseItemType", luaStr(baseItemType.Get("Id"))); compAtt != nil {
			for _, attr := range []struct{ col, name string }{{"Str", "str"}, {"Dex", "dex"}, {"Int", "int"}} {
				if v := compAtt.Get(attr.col).(int64); v > 0 {
					out.W(attr.name, " = ", v, ", ")
				}
			}
		}
		out.W("},\n")
		if ft, ok := baseItemType.Get("FlavourTextKey").(*Row); ok {
			if text := luaStr(ft.Get("Text")); text != "" {
				cleanedLines := cleanAndSplit(text)
				if len(cleanedLines) > 0 {
					out.W("\tflavourText = {\n")
					for _, line := range cleanedLines {
						out.W("\t\t\"", line, "\",\n")
					}
					out.W("\t},\n")
				}
			}
		}
		out.W("}\n")
		if !hidden {
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
	}

	directives := map[string]func(args string, out *OutFile){}
	directives["type"] = func(args string, out *OutFile) { state.typ = args }
	directives["subType"] = func(args string, out *OutFile) { s := args; state.subType = &s }
	directives["influenceBaseTag"] = func(args string, out *OutFile) { state.influenceBaseTag = args }
	directives["forceShow"] = func(args string, out *OutFile) { state.forceShow = args == "true" }
	directives["forceHide"] = func(args string, out *OutFile) { state.forceHide = args == "true" }
	directives["socketLimit"] = func(args string, out *OutFile) {
		if n, err := strconv.ParseFloat(args, 64); err == nil {
			state.socketLimit = &n
		} else {
			state.socketLimit = nil
		}
	}
	directives["base"] = baseDirective
	directives["baseMatch"] = func(argstr string, out *OutFile) {
		key := "Id"
		args := strings.Fields(argstr)
		value := args[0]
		if len(args) > 1 {
			key = args[0]
			value = args[1]
		}
		pat, err := luapat.Convert(value)
		if err != nil {
			panic(fmt.Sprintf("baseMatch: bad pattern %q: %v", value, err))
		}
		re := regexp.MustCompile(pat)
		for _, baseItemType := range x.Dat("BaseItemTypes").GetRowListMatch(key, re.MatchString) {
			id := luaStr(baseItemType.Get("Id"))
			if !strings.Contains(id, "Royale") {
				baseDirective(id, out)
			}
		}
	}
	directives["baseGroup"] = func(args string, out *OutFile) {
		if m := reBaseGroupArg.FindStringSubmatch(args); m != nil {
			baseMods[m[1]] = m[2]
		}
	}
	writeValues := func(out *OutFile, values string) {
		for _, line := range reCommaField.FindAllString(values, -1) {
			out.W(reLeadingSpace.ReplaceAllString(line, ""), "\n")
		}
	}
	directives["setBestBase"] = func(args string, out *OutFile) {
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
		out.W("[[\n")
		out.W(itemName, "\n")
		out.W(base, "\n")
		if !strings.Contains(values, "Crafted: true") {
			out.W("Crafted: true\n")
		}
		if values != " " {
			writeValues(out, values)
		} else if _, ok := baseMods[itemName]; ok {
			writeValues(out, values)
		}
		out.W("]],")
	}
	directives["setBase"] = func(args string, out *OutFile) {
		var baseName, itemName, values string
		if m := reSetBase3.FindStringSubmatch(args); m != nil {
			baseName, itemName, values = m[1], m[2], m[3]
		} else if m := reSetBase2.FindStringSubmatch(args); m != nil {
			baseName, values = m[1], m[2]
		}
		all, ok := basesAll[baseName]
		if baseName != "" && !ok {
			return // the Lua prints "Missing base"
		}
		out.W("[[\n")
		baseClass := all.class
		groupName := baseClass
		if itemName != "" {
			formatted := reFmtS.ReplaceAllString(itemName, baseClass)
			formatted = strings.ReplaceAll(formatted, "One Handed", "1H")
			formatted = strings.ReplaceAll(formatted, "Two Handed", "2H")
			out.W(formatted, "\n")
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
				out.W(groupName, "\n")
			} else {
				out.W(baseClass, "\n")
			}
		}
		out.W(baseName, "\n")
		if !strings.Contains(values, "Crafted: true") {
			out.W("Crafted: true\n")
		}
		if values != " " {
			writeValues(out, values)
		} else if group, ok := baseMods[groupName]; ok {
			writeValues(out, group)
		}
		out.W("]],")
	}

	for _, name := range append(append([]string{}, basesItemTypes...), "Rares") {
		state = &baseState{}
		outDir := "../Data/Bases/"
		if name == "Rares" {
			outDir = "../Data/"
		}
		out, err := x.ProcessTemplateFile(name, "Bases/", outDir, directives)
		if err != nil {
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

// stringifyInlineStrings is Utils.stringifyInline over an array of strings.
func stringifyInlineStrings(list []string) string {
	var b strings.Builder
	b.WriteString("{ ")
	for _, s := range list {
		b.WriteString(luaQ(s))
		b.WriteString(", ")
	}
	b.WriteString("}")
	return b.String()
}
