// Port of .archive/src/Export/Scripts/skills.lua.

package export

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/modparser"
)

func init() {
	Scripts = append(Scripts, Script{Name: "skills", Build: buildSkills})
}

var skillTemplateFiles = []string{"act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int"}

var (
	reScopeLine = regexp.MustCompile(`([0-9A-Za-z_]+) "Metadata/StatDescriptions/([0-9A-Za-z_]+)\.txt"`)
	reScopeCopy = regexp.MustCompile(`copy ([0-9A-Za-z_]+) ([0-9A-Za-z_]+)`)
)

// cleanAndSplitSkills is skills.lua's cleanAndSplit (no <default>/brace
// handling, unlike the flavour text exporter's).
func cleanAndSplitSkills(str string) []string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	var lines []string
	for _, line := range strings.Split(str, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// skillDesc is writeDesc's text cleanup.
func skillDesc(desc string) string {
	return escapeGGGString(strings.ReplaceAll(desc, "\r", ""))
}

func buildSkills(x *Ctx) (schema.Document, error) {
	var (
		activeSkillTypes, grantedEffects, gemEffects, skillGems        *DatFile
		statSetsPerLevel, perLevelDat, qualityStatsDat, itemExperience *DatFile
	)
	for name, dst := range map[string]**DatFile{
		"ActiveSkillType":               &activeSkillTypes,
		"GrantedEffects":                &grantedEffects,
		"GemEffects":                    &gemEffects,
		"SkillGems":                     &skillGems,
		"GrantedEffectStatSetsPerLevel": &statSetsPerLevel,
		"GrantedEffectsPerLevel":        &perLevelDat,
		"GrantedEffectQualityStats":     &qualityStatsDat,
		"ItemExperiencePerLevel":        &itemExperience,
	} {
		var err error
		if *dst, err = x.Dat(name); err != nil {
			return nil, err
		}
	}

	var skillTypeMap []string
	for row := range activeSkillTypes.Rows() {
		skillTypeMap = append(skillTypeMap, row.Str("Id"))
	}
	// mapAST is the ActiveSkillType row's id; a row without a named
	// constant keeps its 1-based row index.
	mapAST := func(ast *Row) modparser.SkillTypeID {
		if ast.ID < len(skillTypeMap) {
			if id, ok := modparser.SkillTypeByName[skillTypeMap[ast.ID]]; ok {
				return id
			}
		}
		return modparser.SkillTypeID(ast.ID + 1)
	}

	// This is here to fix name collisions like in the case of Barrage
	fullNameGems := map[string]bool{
		"Metadata/Items/Gems/SupportGemBarrage": true,
	}

	weaponClassMap := map[string]string{
		"Claw":                     "Claw",
		"Dagger":                   "Dagger",
		"Wand":                     "Wand",
		"One Hand Sword":           "One Handed Sword",
		"Thrusting One Hand Sword": "Thrusting One Handed Sword",
		"One Hand Axe":             "One Handed Axe",
		"One Hand Mace":            "One Handed Mace",
		"Bow":                      "Bow",
		"Fishing Rod":              "Fishing Rod",
		"Staff":                    "Staff",
		"Two Hand Sword":           "Two Handed Sword",
		"Two Hand Axe":             "Two Handed Axe",
		"Two Hand Mace":            "Two Handed Mace",
		"Sceptre":                  "Sceptre",
		"Unarmed":                  "None",
	}

	skillStatScope := map[string]string{}
	{
		text := convertUTF16to8([]byte(x.GetFile("Metadata/StatDescriptions/skillpopup_stat_filters.txt")), 0)
		for _, m := range reScopeLine.FindAllStringSubmatch(text, -1) {
			skillStatScope[m[1]] = m[2]
		}
		for _, m := range reScopeCopy.FindAllStringSubmatch(text, -1) {
			skillStatScope[m[1]] = skillStatScope[m[2]]
		}
	}

	gems := map[string]bool{}
	trueGemNames := map[string]string{}

	// #EVAL: archive parity — statInterpolation cells are aliased and
	// mutated across levels sharing a stat row, exactly as the Lua does.
	type interpHolder struct{ vals []int64 }
	interpHolders := map[*Row]*interpHolder{}
	interpFor := func(statRow *Row) *interpHolder {
		if h, ok := interpHolders[statRow]; ok {
			return h
		}
		h := &interpHolder{vals: statRow.Ints("StatInterpolations")}
		interpHolders[statRow] = h
		return h
	}

	type levelInfo struct {
		level  int64
		values []float64 // the level's array part
		extra  map[string]float64
		interp *interpHolder
		cost   map[string]int64
	}
	type skillInfo struct {
		isSupport           bool
		baseFlags           []string
		mods                []json.RawMessage
		levels              []*levelInfo
		stats               []string
		cannotGrantToMinion []string
		constantStats       []schema.StatValue
		qualityStats        []schema.StatValue
		addSkillTypes       []string
	}
	type skillsState struct {
		noGem         bool
		addSkillTypes []string
		skill         *skillInfo
	}
	var state *skillsState
	var curFile *schema.SkillFile

	// openSkill starts a skill from its granted-effect id and display name
	// (the id when the template gives none).
	openSkill := func(grantedId, displayName string) error {
		noName := displayName == ""
		if noName {
			displayName = grantedId
		}
		hdr := schema.SkillHeader{GrantedId: grantedId}
		granted := grantedEffects.RowByStr("Id", grantedId)
		if granted == nil {
			// the Lua ConPrintfs and leaves the previous skill state
			hdr.Invalid = true
			curFile.Skills = append(curFile.Skills, hdr)
			return nil
		}
		gemEffect := gemEffects.RowByRef("GrantedEffect", granted)
		secondaryEffect := false
		if gemEffect == nil {
			gemEffect = gemEffects.RowByRef("GrantedEffect2", granted)
			if gemEffect != nil {
				secondaryEffect = true
			}
		}
		var skillGem *Row
		if gemEffect != nil {
			gemEffectId := gemEffect.Str("Id")
			for gem := range skillGems.Rows() {
				for _, variant := range gem.Refs("GemVariants") {
					if gemEffectId == variant.Str("Id") {
						skillGem = gem
						trueGemNameObj := gemEffects.RowByStr("Id", gemEffectId)
						if name := trueGemNameObj.Str("Name"); name != "" {
							trueGemNames[gemEffectId] = name
						}
						break
					}
				}
				if skillGem != nil {
					break
				}
			}
		}
		skill := &skillInfo{}
		state.skill = skill
		isSupport := granted.Bool("IsSupport")
		if skillGem != nil && !state.noGem {
			gemEffectId := gemEffect.Str("Id")
			gems[gemEffectId] = true
			base := skillGem.Ref("BaseItemType")
			if isSupport {
				name := base.Str("Name")
				if !fullNameGems[base.Str("Id")] {
					name = strings.ReplaceAll(name, " Support", "")
				}
				hdr.Name = name
				if desc := gemEffect.Str("Description"); len(desc) > 0 {
					d := skillDesc(desc)
					hdr.Description = &d
				}
			} else {
				activeName := granted.Ref("ActiveSkill").Str("DisplayName")
				name := activeName
				if !secondaryEffect {
					if tn, ok := trueGemNames[gemEffectId]; ok {
						name = tn
					}
				}
				hdr.Name = name
				// Hybrid gems (e.g. Vaal gems) use the display name of the
				// active skill e.g. Vaal Summon Skeletons of Sorcery
				hdr.BaseTypeName = &activeName
			}
		} else {
			if noName && !isSupport {
				displayName = ""
				if gemEffect != nil {
					displayName = trueGemNames[gemEffect.Str("Id")]
				}
				if displayName == "" {
					displayName = granted.Ref("ActiveSkill").Str("DisplayName")
				}
			}
			hdr.Name = displayName
			hdr.Hidden = true
		}
		if skillGem != nil {
			if ft := skillGem.Ref("BaseItemType").Ref("FlavourTextKey"); ft != nil {
				hdr.HasFlavour = true
				hdr.FlavourText = cleanAndSplitSkills(ft.Str("Text"))
			}
		}
		state.noGem = false
		skill.addSkillTypes = state.addSkillTypes
		state.addSkillTypes = nil
		statSets := granted.Ref("GrantedEffectStatSets")
		hdr.Color = granted.Int("Attribute")
		if be := statSets.Float("BaseEffectiveness"); be != 1 {
			hdr.BaseEffectiveness = &be
		}
		if ie := statSets.Float("IncrementalEffectiveness"); ie != 0 {
			hdr.IncrementalEffectiveness = &ie
		}
		weaponTypesOf := func(classes []*Row) []string {
			weaponTypes := map[string]bool{}
			for _, class := range classes {
				if mapped, ok := weaponClassMap[class.Str("Id")]; ok {
					weaponTypes[mapped] = true
				}
			}
			keys := make([]string, 0, len(weaponTypes))
			for k := range weaponTypes {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return keys
		}
		if isSupport {
			skill.isSupport = true
			hdr.Support = true
			hdr.RequireSkillTypes = []modparser.SkillTypeID{}
			for _, typ := range granted.Refs("SupportTypes") {
				hdr.RequireSkillTypes = append(hdr.RequireSkillTypes, mapAST(typ))
			}
			hdr.AddSkillTypes = []modparser.SkillTypeID{}
			for _, typ := range granted.Refs("AddTypes") {
				id := mapAST(typ)
				if id == modparser.SkillTypeTriggered {
					hdr.IsTrigger = true
				}
				hdr.AddSkillTypes = append(hdr.AddSkillTypes, id)
			}
			hdr.ExcludeSkillTypes = []modparser.SkillTypeID{}
			for _, typ := range granted.Refs("ExcludeTypes") {
				hdr.ExcludeSkillTypes = append(hdr.ExcludeSkillTypes, mapAST(typ))
			}
			hdr.SupportGemsOnly = granted.Bool("SupportGemsOnly")
			hdr.IgnoreMinionTypes = granted.Bool("IgnoreMinionTypes")
			if pv := granted.Ref("PlusVersionOf"); pv != nil {
				id := pv.Str("Id")
				hdr.PlusVersionOf = &id
			}
			hdr.WeaponTypes = weaponTypesOf(granted.Refs("WeaponRestrictions"))
			hdr.StatDescriptionScope = "gem_stat_descriptions"
		} else {
			activeSkill := granted.Ref("ActiveSkill")
			if desc := activeSkill.Str("Description"); len(desc) > 0 {
				d := skillDesc(desc)
				hdr.Description = &d
			}
			hdr.SkillTypes = []modparser.SkillTypeID{}
			for _, typ := range activeSkill.Refs("SkillTypes") {
				hdr.SkillTypes = append(hdr.SkillTypes, mapAST(typ))
			}
			for _, typ := range skill.addSkillTypes {
				id, ok := modparser.SkillTypeByName[typ]
				if !ok {
					return fmt.Errorf("skills: %s: unknown #addSkillTypes %s", grantedId, typ)
				}
				hdr.SkillTypes = append(hdr.SkillTypes, id)
			}
			for _, typ := range activeSkill.Refs("MinionSkillTypes") {
				hdr.MinionSkillTypes = append(hdr.MinionSkillTypes, mapAST(typ))
			}
			hdr.WeaponTypes = weaponTypesOf(activeSkill.Refs("WeaponRestrictions"))
			scope := skillStatScope[activeSkill.Str("Id")]
			if scope == "" {
				scope = "skill_stat_descriptions"
			}
			hdr.StatDescriptionScope = scope
			if st := activeSkill.Int("SkillTotem"); st <= 21 {
				hdr.SkillTotemId = &st
			}
			ct := float64(granted.Int("CastTime")) / 1000
			hdr.CastTime = &ct
			hdr.CannotBeSupported = granted.Bool("CannotBeSupported")
		}
		curFile.Skills = append(curFile.Skills, hdr)

		statsPerLevel := statSetsPerLevel.RowsByRef("GrantedEffectStatSets", statSets)
		var statMapOrder []string
		statMap := map[string]bool{}
		perLevel := perLevelDat.RowsByRef("GrantedEffect", granted)
		n := len(perLevel)
		if len(statsPerLevel) > n {
			n = len(statsPerLevel)
		}
		for indx := 1; indx <= n; indx++ {
			levelRow := perLevel[0]
			if indx <= len(perLevel) {
				levelRow = perLevel[indx-1]
			}
			statRow := statsPerLevel[0]
			if indx <= len(statsPerLevel) {
				statRow = statsPerLevel[indx-1]
			}
			level := &levelInfo{extra: map[string]float64{}, cost: map[string]int64{}}
			if len(perLevel) == 1 {
				level.level = statRow.Int("GemLevel")
				level.extra["levelRequirement"] = statRow.Float("PlayerLevelReq")
			} else {
				level.level = levelRow.Int("Level")
				level.extra["levelRequirement"] = levelRow.Float("PlayerLevelReq")
			}
			costAmounts := levelRow.Ints("CostAmounts")
			for i, cost := range levelRow.Refs("CostTypes") {
				level.cost[cost.Str("Resource")] = costAmounts[i]
			}
			intExtra := func(col, name string, div float64) {
				if v := levelRow.Int(col); v != 0 {
					level.extra[name] = float64(v) / div
				}
			}
			intExtra("ManaReservationFlat", "manaReservationFlat", 1)
			intExtra("ManaReservationPercent", "manaReservationPercent", 100)
			intExtra("LifeReservationFlat", "lifeReservationFlat", 1)
			intExtra("LifeReservationPercent", "lifeReservationPercent", 100)
			if cm := levelRow.Int("CostMultiplier"); cm != 100 {
				level.extra["manaMultiplier"] = float64(cm - 100)
			}
			intExtra("AttackSpeedMultiplier", "attackSpeedMultiplier", 1)
			intExtra("AttackTime", "attackTime", 1)
			intExtra("Cooldown", "cooldown", 1000)
			intExtra("PvPDamageMultiplier", "PvPDamageMultiplier", 1)
			intExtra("StoredUses", "storedUses", 1)
			if v := levelRow.Int("VaalSouls"); v != 0 {
				level.cost["Soul"] = v
			}
			intExtra("VaalStoredUses", "vaalStoredUses", 1)
			intExtra("SoulGainPreventionDuration", "soulPreventionDuration", 1000)
			// stat based level info
			if de := statRow.Int("DamageEffectiveness"); de != 0 {
				level.extra["damageEffectiveness"] = float64(de)/10000 + 1
			}
			if cc := statRow.Int("AttackCritChance"); cc != 0 {
				level.extra["critChance"] = float64(cc) / 100
			}
			if cc := statRow.Int("OffhandCritChance"); cc != 0 {
				level.extra["critChance"] = float64(cc) / 100
			}
			if bm := statRow.Int("BaseMultiplier"); bm != 0 {
				level.extra["baseMultiplier"] = float64(bm)/10000 + 1
			}
			level.interp = interpFor(statRow)
			registerStat := func(id string, cgtm bool, first bool) {
				statMap[id] = true
				skill.stats = append(skill.stats, id)
				if first {
					statMapOrder = append(statMapOrder, id)
					if cgtm {
						skill.cannotGrantToMinion = append(skill.cannotGrantToMinion, id)
					}
				}
			}
			padMissing := func(statMapOrderIndex *int, id string) {
				for *statMapOrderIndex < len(statMapOrder) && statMapOrder[*statMapOrderIndex-1] != id {
					level.values = append(level.values, 0)
					if len(level.interp.vals) < len(statMapOrder) {
						idx := *statMapOrderIndex - 1
						level.interp.vals = append(level.interp.vals[:idx], append([]int64{0}, level.interp.vals[idx:]...)...)
					}
					*statMapOrderIndex++
				}
			}
			statMapOrderIndex := 1
			floatStats := statRow.Refs("FloatStats")
			floatVals := statRow.Floats("FloatStatsValues")
			interpBases := statRow.Refs("InterpolationBases")
			for i, stat := range floatStats {
				id := stat.Str("Id")
				if !statMap[id] || indx == 1 {
					registerStat(id, stat.Bool("CannotGrantToMinion"), indx == 1)
				} else if statMapOrderIndex-1 >= len(statMapOrder) || statMapOrder[statMapOrderIndex-1] != id {
					padMissing(&statMapOrderIndex, id)
				}
				statMapOrderIndex++
				level.values = append(level.values, floatVals[i]/math.Max(interpBases[i].Float("Value"), 0.00001))
			}
			addStats := statRow.Refs("AdditionalStats")
			addVals := statRow.Ints("AdditionalStatsValues")
			for i, stat := range addStats {
				id := stat.Str("Id")
				if !statMap[id] {
					registerStat(id, stat.Bool("CannotGrantToMinion"), indx == 1)
				} else if statMapOrderIndex-1 >= len(statMapOrder) || statMapOrder[statMapOrderIndex-1] != id {
					padMissing(&statMapOrderIndex, id)
				}
				statMapOrderIndex++
				level.values = append(level.values, float64(addVals[i]))
			}
			for _, stat := range statRow.Refs("AdditionalBooleanStats") {
				id := stat.Str("Id")
				if !statMap[id] {
					statMap[id] = true
					skill.stats = append(skill.stats, id)
					if stat.Bool("CannotGrantToMinion") {
						skill.cannotGrantToMinion = append(skill.cannotGrantToMinion, id)
					}
				}
			}
			skill.levels = append(skill.levels, level)
		}
		for _, stat := range statSets.Refs("ImplicitStats") {
			id := stat.Str("Id")
			if !statMap[id] {
				statMap[id] = true
				skill.stats = append(skill.stats, id)
			}
		}
		constStats := statSets.Refs("ConstantStats")
		constVals := statSets.Ints("ConstantStatsValues")
		for i, stat := range constStats {
			skill.constantStats = append(skill.constantStats, schema.StatValue{Id: stat.Str("Id"), Value: float64(constVals[i])})
		}
		for _, qsRow := range qualityStatsDat.RowsByRef("GrantedEffect", granted) {
			statVals := qsRow.Ints("StatValues")
			for j, stat := range qsRow.Refs("GrantedStats") {
				id := stat.Str("Id")
				if id != "dummy_stat_display_nothing" {
					skill.qualityStats = append(skill.qualityStats, schema.StatValue{Id: id, Value: float64(statVals[j]) / 1000})
				}
			}
		}
		return nil
	}
	// closeSkill emits the pending skill's tail; flags are the emission
	// switches.
	closeSkill := func(flags []string) {
		skill := state.skill
		tail := schema.SkillTail{
			ModsFlags:     append([]string(nil), flags...),
			Support:       skill.isSupport,
			BaseFlags:     append([]string(nil), skill.baseFlags...),
			BaseMods:      append([]json.RawMessage(nil), skill.mods...),
			Stats:         append([]string{}, skill.stats...),
			NotMinionStat: append([]string(nil), skill.cannotGrantToMinion...),
			QualityStats:  append([]schema.StatValue(nil), skill.qualityStats...),
			ConstantStats: append([]schema.StatValue(nil), skill.constantStats...),
		}
		for _, level := range skill.levels {
			l := schema.SkillLevel{Level: level.level}
			l.Values = append([]float64(nil), level.values...)
			if len(level.extra) > 0 {
				l.Extra = map[string]float64{}
				for k, v := range level.extra {
					l.Extra[k] = v
				}
			}
			l.Interp = append(l.Interp, level.interp.vals...)
			if len(level.cost) > 0 {
				l.Cost = map[string]int64{}
				for k, v := range level.cost {
					l.Cost[k] = v
				}
			}
			tail.Levels = append(tail.Levels, l)
		}
		curFile.Tails = append(curFile.Tails, tail)
		state.skill = nil
	}

	doc := schema.SkillsData{Files: map[string]schema.SkillFile{}}
	for _, name := range skillTemplateFiles {
		state = &skillsState{}
		var f schema.SkillFile
		curFile = &f
		tpl, err := readTemplate("Skills/", name, skillDirectives)
		if err != nil {
			return nil, err
		}
		for _, d := range tpl.Directives {
			switch d := d.(type) {
			case *skillHeadDirective:
				if err := openSkill(d.Granted, d.Name); err != nil {
					return nil, err
				}
			case *flagsDirective:
				state.skill.baseFlags = append(state.skill.baseFlags, d.Flags...)
			case *baseModDirective:
				state.skill.mods = append(state.skill.mods, d.Mods)
			case *modsDirective:
				closeSkill(d.Flags)
			case *noGemDirective:
				state.noGem = true
			case *addSkillTypesDirective:
				state.addSkillTypes = d.Types
			}
		}
		doc.Files[name] = f
	}

	for skillGem := range skillGems.Rows() {
		for _, ge := range skillGem.Refs("GemVariants") {
			gemEffectId := ge.Str("Id")
			if !gems[gemEffectId] {
				continue
			}
			// Some skills have an additional old version that exists in the
			// game files and messes up transfigured gem matching.
			delete(gems, gemEffectId)
			base := skillGem.Ref("BaseItemType")
			grantedEffect := ge.Ref("GrantedEffect")
			g := schema.GemDef{VariantId: gemEffectId}
			name := base.Str("Name")
			if !fullNameGems[base.Str("Id")] {
				if tn, ok := trueGemNames[gemEffectId]; ok {
					name = tn
				} else {
					name = strings.ReplaceAll(name, " Support", "")
				}
			}
			g.Name = name
			// Hybrid gems (e.g. Vaal gems) use the display name of the
			// active skill e.g. Vaal Summon Skeletons of Sorcery
			if !skillGem.Bool("IsSupport") {
				btn := grantedEffect.Ref("ActiveSkill").Str("DisplayName")
				g.BaseTypeName = &btn
			}
			g.GameId = base.Str("Id")
			g.GrantedEffectId = grantedEffect.Str("Id")
			if ge2 := ge.Ref("GrantedEffect2"); ge2 != nil {
				id := ge2.Str("Id")
				g.SecondaryGrantedEffectId = &id
			}
			if sn := ge.Str("SecondarySupportName"); len(sn) > 0 {
				g.SecondaryEffectName = &sn
			}
			g.VaalGem = skillGem.Bool("IsVaalGem")
			var tagNames []string
			g.Tags = []string{}
			for _, tag := range ge.Refs("Tags") {
				escaped := escapeGGGString(tag.Str("Name"))
				tag.SetCell("Name", escaped)
				g.Tags = append(g.Tags, tag.Str("Id"))
				if len(escaped) > 0 {
					tagNames = append(tagNames, escaped)
				}
			}
			g.TagString = strings.Join(tagNames, ", ")
			g.ReqStr = skillGem.Int("Str")
			g.ReqDex = skillGem.Int("Dex")
			g.ReqInt = skillGem.Int("Int")
			naturalMaxLevel := len(itemExperience.RowsByRef("ItemExperienceType", skillGem.Ref("GemLevelProgression")))
			if naturalMaxLevel == 0 {
				naturalMaxLevel = 1
			}
			g.NaturalMaxLevel = naturalMaxLevel
			doc.Gems = append(doc.Gems, g)
		}
	}
	return doc, nil
}
