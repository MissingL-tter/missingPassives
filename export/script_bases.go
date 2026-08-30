// Port of .archive/src/Export/Scripts/bases.lua.

package export

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
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
	reItExtends   = regexp.MustCompile(`extends "(.+)"`)
	reItRemoveTag = regexp.MustCompile(`remove_tag = "(.+)"`)
	reItTag       = regexp.MustCompile(`tag = "(.+)"`)
	reFmtS        = regexp.MustCompile(`%s`)
)

func buildBases(x *Ctx) (schema.Document, error) {
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
	baseMods := map[string][]string{}

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

	var (
		baseItemTypes, weaponTypes, armourTypes, shieldTypes, flasks *DatFile
		componentCharges, tinctures, componentAttributes             *DatFile
	)
	for name, dst := range map[string]**DatFile{
		"BaseItemTypes":                  &baseItemTypes,
		"WeaponTypes":                    &weaponTypes,
		"ArmourTypes":                    &armourTypes,
		"ShieldTypes":                    &shieldTypes,
		"Flasks":                         &flasks,
		"ComponentCharges":               &componentCharges,
		"tinctures":                      &tinctures,
		"ComponentAttributeRequirements": &componentAttributes,
	} {
		var err error
		if *dst, err = x.Dat(name); err != nil {
			return nil, err
		}
	}

	buildBase := func(baseTypeId, displayName string) (*schema.ItemBase, error) {
		baseItemType := baseItemTypes.GetRow("Id", baseTypeId)
		if baseItemType == nil {
			return nil, nil // the Lua printfs "Invalid Id"
		}
		baseItemTags := getBaseItemTags(baseItemType.Str("BaseType"))
		if displayName == "" {
			displayName = baseItemType.Str("Name")
		}
		displayName = strings.ReplaceAll(displayName, "\xc3\xb6", "o")
		displayName = strings.TrimSpace(displayName)
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
		for _, tag := range baseItemType.Refs("Tags") {
			combinedTags[tag.Str("Id")] = true
		}
		sortedTags := make([]string, 0, len(combinedTags))
		for t := range combinedTags {
			sortedTags = append(sortedTags, t)
		}
		sort.Strings(sortedTags)
		b.Tags = sortedTags
		b.InfluenceBaseTag = state.influenceBaseTag

		implicitRows := baseItemType.Refs("ImplicitMods")
		for _, mod := range implicitRows {
			modDesc, err := x.DescribeMod(mod)
			if err != nil {
				return nil, err
			}
			for _, line := range modDesc.Lines {
				b.Implicit = append(b.Implicit, line)
				b.ImplicitModTypes = append(b.ImplicitModTypes, modDesc.ModTags)
			}
			if len(modDesc.Lines) > 0 {
				b.ImplicitIds = append(b.ImplicitIds, mod.Str("Id"))
			}
		}
		enchantRows := baseItemType.Refs("EnchantMods")
		for _, mod := range enchantRows {
			modDesc, err := x.DescribeMod(mod)
			if err != nil {
				return nil, err
			}
			for _, line := range modDesc.Lines {
				b.Enchant = append(b.Enchant, line)
				b.EnchantModTypes = append(b.EnchantModTypes, modDesc.ModTags)
			}
			if len(modDesc.Lines) > 0 {
				b.EnchantIds = append(b.EnchantIds, mod.Str("Id"))
			}
		}
		b.CannotBeAnointed = len(enchantRows) > 0

		itemValueSum := int64(0)
		weaponType := weaponTypes.GetRow("BaseItemType", baseItemType)
		if weaponType != nil {
			b.Weapon = &schema.WeaponBase{
				PhysicalMin:    weaponType.Int("DamageMin"),
				PhysicalMax:    weaponType.Int("DamageMax"),
				CritChanceBase: float64(weaponType.Int("CritChance")) / 100,
				AttackRateBase: round(1000/float64(weaponType.Int("Speed")), 2),
				Range:          weaponType.Int("Range"),
			}
			itemValueSum = b.Weapon.PhysicalMin + b.Weapon.PhysicalMax
		}
		armourType := armourTypes.GetRow("BaseItemType", baseItemType)
		if armourType != nil {
			ab := &schema.ArmourBase{}
			if shield := shieldTypes.GetRow("BaseItemType", baseItemType); shield != nil {
				v := shield.Int("Block")
				ab.BlockChance = &v
			}
			set := func(minP, maxP **int64, col string) {
				mn := armourType.Int(col + "Min")
				mx := armourType.Int(col + "Max")
				if mn > 0 {
					*minP, *maxP = &mn, &mx
					itemValueSum += mn + mx
				}
			}
			set(&ab.ArmourMin, &ab.ArmourMax, "Armour")
			set(&ab.EvasionMin, &ab.EvasionMax, "Evasion")
			set(&ab.EnergyShieldMin, &ab.EnergyShieldMax, "EnergyShield")
			if mp := armourType.Int("MovementPenalty"); mp != 0 {
				v := -mp
				ab.MovementPenalty = &v
			}
			set(&ab.WardMin, &ab.WardMax, "Ward")
			b.Armour = ab
		}
		flask := flasks.GetRow("BaseItemType", baseItemType)
		if flask != nil {
			compCharges := componentCharges.GetRow("BaseItemType", baseItemType.Str("Id"))
			fb := &schema.FlaskBase{
				Duration:    float64(flask.Int("RecoveryTime")) / 10,
				ChargesUsed: compCharges.Int("PerUse"),
				ChargesMax:  compCharges.Int("Max"),
			}
			if v := flask.Int("LifePerUse"); v > 0 {
				fb.Life = &v
			}
			if v := flask.Int("ManaPerUse"); v > 0 {
				fb.Mana = &v
			}
			if buff := flask.Ref("Buff"); buff != nil {
				fb.HasBuff = true
				stats := map[string]*statVal{}
				mags := flask.Ints("BuffMagnitudes")
				for i, stat := range buff.Refs("Stats") {
					v := float64(mags[i])
					stats[stat.Str("Id")] = &statVal{min: v, max: v}
				}
				for _, stat := range buff.Refs("GrantedFlags") {
					stats[stat.Str("Id")] = &statVal{min: 1, max: 1}
				}
				lines, err := x.DescribeStats(stats)
				if err != nil {
					return nil, err
				}
				fb.Buff = lines.Lines
			}
			b.Flask = fb
		}
		tincture := tinctures.GetRow("BaseItemType", baseItemType)
		if tincture != nil {
			b.Tincture = &schema.TinctureBase{
				ManaBurn: float64(tincture.Int("ManaBurn")) / 1000,
				Cooldown: float64(tincture.Int("CoolDown")) / 1000,
			}
		}
		reqLevel := int64(1)
		dropLevel := baseItemType.Int("DropLevel")
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
			lvl := int64(math.Floor(float64(mod.Int("Level")) * 0.8))
			if lvl > reqLevel {
				reqLevel = lvl
			}
		}
		if reqLevel > 1 {
			b.ReqLevel = &reqLevel
		}
		if compAtt := componentAttributes.GetRow("BaseItemType", baseItemType.Str("Id")); compAtt != nil {
			for _, attr := range []struct {
				col string
				dst **int64
			}{{"Str", &b.ReqStr}, {"Dex", &b.ReqDex}, {"Int", &b.ReqInt}} {
				if v := compAtt.Int(attr.col); v > 0 {
					*attr.dst = &v
				}
			}
		}
		if ft := baseItemType.Ref("FlavourTextKey"); ft != nil {
			if text := ft.Str("Text"); text != "" {
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
		return b, nil
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

	addBase := func(id, name string) error {
		var ev []schema.ItemBase
		b, err := buildBase(id, name)
		if err != nil {
			return err
		}
		if b != nil {
			ev = append(ev, *b)
		}
		*curEvents = append(*curEvents, ev)
		return nil
	}
	addBaseMatch := func(column, pattern string) error {
		if column == "" {
			column = "Id"
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("bases: #baseMatch %q: %w", pattern, err)
		}
		var ev []schema.ItemBase
		for _, baseItemType := range baseItemTypes.GetRowListMatch(column, re.MatchString) {
			id := baseItemType.Str("Id")
			if !strings.Contains(id, "Royale") {
				b, err := buildBase(id, "")
				if err != nil {
					return err
				}
				if b != nil {
					ev = append(ev, *b)
				}
			}
		}
		*curEvents = append(*curEvents, ev)
		return nil
	}
	setBestBase := func(d *setBestBaseDirective) {
		itemName := d.Name
		if itemName == "" {
			itemName = d.SubType + " " + d.Class
		}
		base := bases[d.Class][d.SubType].displayName
		lines := []string{itemName, base}
		if !slices.Contains(d.Mods, "Crafted: true") {
			lines = append(lines, "Crafted: true")
		}
		if len(d.Mods) > 0 {
			lines = append(lines, d.Mods...)
		} else if _, ok := baseMods[itemName]; ok {
			lines = append(lines, "") // the reference re-splits its blank list, not the group
		}
		doc.Rares = append(doc.Rares, &schema.RareItem{Lines: lines})
		streamBlob(lines)
	}
	setBase := func(d *setBaseDirective) {
		baseName, itemName := d.Base, d.Name
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
		if !slices.Contains(d.Mods, "Crafted: true") {
			lines = append(lines, "Crafted: true")
		}
		if len(d.Mods) > 0 {
			lines = append(lines, d.Mods...)
		} else if group, ok := baseMods[groupName]; ok {
			lines = append(lines, group...)
		}
		doc.Rares = append(doc.Rares, &schema.RareItem{Lines: lines})
		streamBlob(lines)
	}

	for _, name := range append(append([]string{}, basesItemTypes...), "Rares") {
		state = &baseState{}
		var events [][]schema.ItemBase
		curEvents = &events
		tpl, err := readTemplate("Bases/", name, baseDirectives)
		if err != nil {
			return nil, err
		}
		inRares = name == "Rares"
		for _, d := range tpl.Directives {
			switch d := d.(type) {
			case *typeDirective:
				state.typ = d.Name
			case *subTypeDirective:
				s := d.Name
				state.subType = &s
			case *influenceBaseTagDirective:
				state.influenceBaseTag = d.Tag
			case *forceShowDirective:
				state.forceShow = d.Value
			case *forceHideDirective:
				state.forceHide = d.Value
			case *socketLimitDirective:
				n := d.Limit
				state.socketLimit = &n
			case *baseItemDirective:
				err = addBase(d.ID, d.Name)
			case *baseMatchDirective:
				err = addBaseMatch(d.Column, d.Pattern)
			case *baseGroupDirective:
				baseMods[d.Name] = d.Mods
			case *setBestBaseDirective:
				setBestBase(d)
			case *setBaseDirective:
				setBase(d)
			}
			if err != nil {
				return nil, err
			}
		}
		if !inRares {
			doc.Types[name] = events
			continue
		}
		// The rare list is the directive-generated best-base blobs (in
		// directive order) followed by the template's hand-written
		// blocks — the same order the generated file carries them.
		inRares = false
		f, err := splitUniqueFile(raresStream)
		if err != nil {
			return nil, fmt.Errorf("Rares: %w", err)
		}
		for _, sec := range f.Sections {
			doc.RareBlobs = append(doc.RareBlobs, sec.Items...)
		}
		doc.RareBlobs = append(doc.RareBlobs, tpl.Items...)
	}
	return doc, nil
}
