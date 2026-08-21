// Port of .archive/src/Export/Scripts/skills.lua.

package export

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

func init() {
	outs := []string{"Data/Gems.lua"}
	for _, name := range skillTemplateFiles {
		outs = append(outs, "Data/Skills/"+name+".lua")
	}
	Scripts = append(Scripts, Script{Name: "skills", Outs: outs, Run: scriptSkills})
}

var skillTemplateFiles = []string{"act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int"}

var (
	reScopeLine = regexp.MustCompile(`([0-9A-Za-z_]+) "Metadata/StatDescriptions/([0-9A-Za-z_]+)\.txt"`)
	reScopeCopy = regexp.MustCompile(`copy ([0-9A-Za-z_]+) ([0-9A-Za-z_]+)`)
	reSkillHead = regexp.MustCompile(`([0-9A-Za-z_]+) (.+)`)
	reWord      = regexp.MustCompile(`[A-Za-z]+`)
)

// cleanAndSplitSkills is skills.lua's cleanAndSplit (no <default>/brace
// handling, unlike the flavour text exporter's).
func cleanAndSplitSkills(str string) []string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	var lines []string
	for _, line := range strings.Split(str, "\n") {
		line = luaTrim(line)
		if line != "" {
			lines = append(lines, strings.ReplaceAll(line, "\"", "\\\""))
		}
	}
	return lines
}

func scriptSkills(x *Ctx) error {
	var skillTypeMap []string
	x.Dat("ActiveSkillType").Rows(func(row *Row) bool {
		skillTypeMap = append(skillTypeMap, luaStr(row.Get("Id")))
		return true
	})
	mapAST := func(ast *Row) string {
		if ast.Index-1 < len(skillTypeMap) {
			return "SkillType." + skillTypeMap[ast.Index-1]
		}
		return "SkillType.Unknown" + luaStr(ast.Index)
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
	type interpHolder struct{ vals []any }
	interpHolders := map[*Row]*interpHolder{}
	interpFor := func(statRow *Row) *interpHolder {
		if h, ok := interpHolders[statRow]; ok {
			return h
		}
		h := &interpHolder{vals: append([]any(nil), statRow.Get("StatInterpolations").([]any)...)}
		interpHolders[statRow] = h
		return h
	}

	type levelInfo struct {
		level  int64
		values []any // the level's array part
		extra  map[string]any
		interp *interpHolder
		cost   map[string]int64
	}
	type skillInfo struct {
		isSupport           bool
		baseFlags           []string
		mods                []string
		levels              []*levelInfo
		stats               []string
		cannotGrantToMinion []string
		constantStats       [][2]any
		qualityStats        [][2]any
		addSkillTypes       []string
	}
	type skillsState struct {
		noGem         bool
		addSkillTypes []string
		skill         *skillInfo
	}
	var state *skillsState

	writeDesc := func(out *OutFile, key, desc string) {
		s := strings.ReplaceAll(desc, "\"", "\\\"")
		s = strings.ReplaceAll(s, "\r", "")
		s = strings.ReplaceAll(s, "\n", "\\n")
		out.W("\t", key, " = \"", escapeGGGString(s), "\",\n")
	}

	directives := map[string]func(args string, out *OutFile){}
	directives["noGem"] = func(args string, out *OutFile) { state.noGem = true }
	directives["addSkillTypes"] = func(args string, out *OutFile) {
		state.addSkillTypes = reWord.FindAllString(args, -1)
	}
	directives["skill"] = func(args string, out *OutFile) {
		grantedId := args
		displayName := args
		if m := reSkillHead.FindStringSubmatch(args); m != nil {
			grantedId, displayName = m[1], m[2]
		}
		out.W("skills[\"", grantedId, "\"] = {\n")
		granted := x.Dat("GrantedEffects").GetRow("Id", grantedId)
		if granted == nil {
			return // the Lua ConPrintfs and leaves the previous skill state
		}
		gemEffect := x.Dat("GemEffects").GetRow("GrantedEffect", granted)
		secondaryEffect := false
		if gemEffect == nil {
			gemEffect = x.Dat("GemEffects").GetRow("GrantedEffect2", granted)
			if gemEffect != nil {
				secondaryEffect = true
			}
		}
		var skillGem *Row
		if gemEffect != nil {
			gemEffectId := luaStr(gemEffect.Get("Id"))
			x.Dat("SkillGems").Rows(func(gem *Row) bool {
				for _, variant := range listRows(gem.Get("GemVariants")) {
					if gemEffectId == luaStr(variant.Get("Id")) {
						skillGem = gem
						trueGemNameObj := x.Dat("GemEffects").GetRow("Id", gemEffectId)
						if name := luaStr(trueGemNameObj.Get("Name")); name != "" {
							trueGemNames[gemEffectId] = name
						}
						break
					}
				}
				return skillGem == nil
			})
		}
		skill := &skillInfo{}
		state.skill = skill
		isSupport := granted.Get("IsSupport").(bool)
		if skillGem != nil && !state.noGem {
			gemEffectId := luaStr(gemEffect.Get("Id"))
			gems[gemEffectId] = true
			base := skillGem.Get("BaseItemType").(*Row)
			if isSupport {
				name := luaStr(base.Get("Name"))
				if !fullNameGems[luaStr(base.Get("Id"))] {
					name = strings.ReplaceAll(name, " Support", "")
				}
				out.W("\tname = \"", name, "\",\n")
				if desc := luaStr(gemEffect.Get("Description")); len(desc) > 0 {
					writeDesc(out, "description", desc)
				}
			} else {
				activeName := luaStr(granted.Get("ActiveSkill").(*Row).Get("DisplayName"))
				name := activeName
				if !secondaryEffect {
					if tn, ok := trueGemNames[gemEffectId]; ok {
						name = tn
					}
				}
				out.W("\tname = \"", name, "\",\n")
				// Hybrid gems (e.g. Vaal gems) use the display name of the
				// active skill e.g. Vaal Summon Skeletons of Sorcery
				out.W("\tbaseTypeName = \"", activeName, "\",\n")
			}
		} else {
			if displayName == args && !isSupport {
				displayName = ""
				if gemEffect != nil {
					displayName = trueGemNames[luaStr(gemEffect.Get("Id"))]
				}
				if displayName == "" {
					displayName = luaStr(granted.Get("ActiveSkill").(*Row).Get("DisplayName"))
				}
			}
			out.W("\tname = \"", displayName, "\",\n")
			out.W("\thidden = true,\n")
		}
		if skillGem != nil {
			if ft, ok := skillGem.Get("BaseItemType").(*Row).Get("FlavourTextKey").(*Row); ok {
				out.W("\tflavourText = {")
				for _, line := range cleanAndSplitSkills(luaStr(ft.Get("Text"))) {
					out.W("\"", line, "\", ")
				}
				out.W("},\n")
			}
		}
		state.noGem = false
		skill.addSkillTypes = state.addSkillTypes
		state.addSkillTypes = nil
		statSets := granted.Get("GrantedEffectStatSets").(*Row)
		out.W("\tcolor = ", granted.Get("Attribute").(int64), ",\n")
		if be := statSets.Get("BaseEffectiveness").(float64); be != 1 {
			out.W("\tbaseEffectiveness = ", luaNum(be), ",\n")
		}
		if ie := statSets.Get("IncrementalEffectiveness").(float64); ie != 0 {
			out.W("\tincrementalEffectiveness = ", luaNum(ie), ",\n")
		}
		writeWeaponTypes := func(classes []*Row) {
			weaponTypes := map[string]bool{}
			for _, class := range classes {
				if mapped, ok := weaponClassMap[luaStr(class.Get("Id"))]; ok {
					weaponTypes[mapped] = true
				}
			}
			if len(weaponTypes) > 0 {
				out.W("\tweaponTypes = {\n")
				keys := make([]string, 0, len(weaponTypes))
				for k := range weaponTypes {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, typ := range keys {
					out.W("\t\t[\"", typ, "\"] = true,\n")
				}
				out.W("\t},\n")
			}
		}
		if isSupport {
			skill.isSupport = true
			out.W("\tsupport = true,\n")
			out.W("\trequireSkillTypes = { ")
			for _, typ := range listRows(granted.Get("SupportTypes")) {
				out.W(mapAST(typ), ", ")
			}
			out.W("},\n")
			out.W("\taddSkillTypes = { ")
			isTrigger := false
			for _, typ := range listRows(granted.Get("AddTypes")) {
				typeString := mapAST(typ)
				if typeString == "SkillType.Triggered" {
					isTrigger = true
				}
				out.W(typeString, ", ")
			}
			out.W("},\n")
			out.W("\texcludeSkillTypes = { ")
			for _, typ := range listRows(granted.Get("ExcludeTypes")) {
				out.W(mapAST(typ), ", ")
			}
			out.W("},\n")
			if isTrigger {
				out.W("\tisTrigger = true,\n")
			}
			if granted.Get("SupportGemsOnly").(bool) {
				out.W("\tsupportGemsOnly = true,\n")
			}
			if granted.Get("IgnoreMinionTypes").(bool) {
				out.W("\tignoreMinionTypes = true,\n")
			}
			if pv, ok := granted.Get("PlusVersionOf").(*Row); ok {
				out.W("\tplusVersionOf = \"", luaStr(pv.Get("Id")), "\",\n")
			}
			writeWeaponTypes(listRows(granted.Get("WeaponRestrictions")))
			out.W("\tstatDescriptionScope = \"gem_stat_descriptions\",\n")
		} else {
			activeSkill := granted.Get("ActiveSkill").(*Row)
			if desc := luaStr(activeSkill.Get("Description")); len(desc) > 0 {
				writeDesc(out, "description", desc)
			}
			out.W("\tskillTypes = { ")
			for _, typ := range listRows(activeSkill.Get("SkillTypes")) {
				out.W("[", mapAST(typ), "] = true, ")
			}
			for _, typ := range skill.addSkillTypes {
				out.W("[SkillType.", typ, "] = true, ")
			}
			out.W("},\n")
			minionTypes := listRows(activeSkill.Get("MinionSkillTypes"))
			if len(minionTypes) > 0 {
				out.W("\tminionSkillTypes = { ")
				for _, typ := range minionTypes {
					out.W("[", mapAST(typ), "] = true, ")
				}
				out.W("},\n")
			}
			writeWeaponTypes(listRows(activeSkill.Get("WeaponRestrictions")))
			scope := skillStatScope[luaStr(activeSkill.Get("Id"))]
			if scope == "" {
				scope = "skill_stat_descriptions"
			}
			out.W("\tstatDescriptionScope = \"", scope, "\",\n")
			if st := activeSkill.Get("SkillTotem").(int64); st <= 21 {
				out.W("\tskillTotemId = ", st, ",\n")
			}
			out.W("\tcastTime = ", luaNum(float64(granted.Get("CastTime").(int64))/1000), ",\n")
			if granted.Get("CannotBeSupported").(bool) {
				out.W("\tcannotBeSupported = true,\n")
			}
		}
		statsPerLevel := x.Dat("GrantedEffectStatSetsPerLevel").GetRowList("GrantedEffectStatSets", statSets)
		var statMapOrder []string
		statMap := map[string]bool{}
		perLevel := x.Dat("GrantedEffectsPerLevel").GetRowList("GrantedEffect", granted)
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
			level := &levelInfo{extra: map[string]any{}, cost: map[string]int64{}}
			if len(perLevel) == 1 {
				level.level = statRow.Get("GemLevel").(int64)
				level.extra["levelRequirement"] = statRow.Get("PlayerLevelReq").(float64)
			} else {
				level.level = levelRow.Get("Level").(int64)
				level.extra["levelRequirement"] = levelRow.Get("PlayerLevelReq").(float64)
			}
			costAmounts := levelRow.Get("CostAmounts").([]any)
			for i, cost := range listRows(levelRow.Get("CostTypes")) {
				level.cost[luaStr(cost.Get("Resource"))] = costAmounts[i].(int64)
			}
			intExtra := func(col, name string, transform func(int64) any) {
				if v := levelRow.Get(col).(int64); v != 0 {
					if transform != nil {
						level.extra[name] = transform(v)
					} else {
						level.extra[name] = v
					}
				}
			}
			intExtra("ManaReservationFlat", "manaReservationFlat", nil)
			intExtra("ManaReservationPercent", "manaReservationPercent", func(v int64) any { return float64(v) / 100 })
			intExtra("LifeReservationFlat", "lifeReservationFlat", nil)
			intExtra("LifeReservationPercent", "lifeReservationPercent", func(v int64) any { return float64(v) / 100 })
			if cm := levelRow.Get("CostMultiplier").(int64); cm != 100 {
				level.extra["manaMultiplier"] = cm - 100
			}
			intExtra("AttackSpeedMultiplier", "attackSpeedMultiplier", nil)
			intExtra("AttackTime", "attackTime", nil)
			intExtra("Cooldown", "cooldown", func(v int64) any { return float64(v) / 1000 })
			intExtra("PvPDamageMultiplier", "PvPDamageMultiplier", nil)
			intExtra("StoredUses", "storedUses", nil)
			if v := levelRow.Get("VaalSouls").(int64); v != 0 {
				level.cost["Soul"] = v
			}
			intExtra("VaalStoredUses", "vaalStoredUses", nil)
			intExtra("SoulGainPreventionDuration", "soulPreventionDuration", func(v int64) any { return float64(v) / 1000 })
			// stat based level info
			if de := statRow.Get("DamageEffectiveness").(int64); de != 0 {
				level.extra["damageEffectiveness"] = float64(de)/10000 + 1
			}
			if cc := statRow.Get("AttackCritChance").(int64); cc != 0 {
				level.extra["critChance"] = float64(cc) / 100
			}
			if cc := statRow.Get("OffhandCritChance").(int64); cc != 0 {
				level.extra["critChance"] = float64(cc) / 100
			}
			if bm := statRow.Get("BaseMultiplier").(int64); bm != 0 {
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
					level.values = append(level.values, int64(0))
					if len(level.interp.vals) < len(statMapOrder) {
						idx := *statMapOrderIndex - 1
						level.interp.vals = append(level.interp.vals[:idx], append([]any{"0"}, level.interp.vals[idx:]...)...)
					}
					*statMapOrderIndex++
				}
			}
			statMapOrderIndex := 1
			floatStats := listRows(statRow.Get("FloatStats"))
			floatVals := statRow.Get("FloatStatsValues").([]any)
			interpBases := listRows(statRow.Get("InterpolationBases"))
			for i, stat := range floatStats {
				id := luaStr(stat.Get("Id"))
				if !statMap[id] || indx == 1 {
					registerStat(id, stat.Get("CannotGrantToMinion").(bool), indx == 1)
				} else if statMapOrderIndex-1 >= len(statMapOrder) || statMapOrder[statMapOrderIndex-1] != id {
					padMissing(&statMapOrderIndex, id)
				}
				statMapOrderIndex++
				level.values = append(level.values, floatVals[i].(float64)/math.Max(interpBases[i].Get("Value").(float64), 0.00001))
			}
			addStats := listRows(statRow.Get("AdditionalStats"))
			addVals := statRow.Get("AdditionalStatsValues").([]any)
			for i, stat := range addStats {
				id := luaStr(stat.Get("Id"))
				if !statMap[id] {
					registerStat(id, stat.Get("CannotGrantToMinion").(bool), indx == 1)
				} else if statMapOrderIndex-1 >= len(statMapOrder) || statMapOrder[statMapOrderIndex-1] != id {
					padMissing(&statMapOrderIndex, id)
				}
				statMapOrderIndex++
				level.values = append(level.values, addVals[i].(int64))
			}
			for _, stat := range listRows(statRow.Get("AdditionalBooleanStats")) {
				id := luaStr(stat.Get("Id"))
				if !statMap[id] {
					statMap[id] = true
					skill.stats = append(skill.stats, id)
					if stat.Get("CannotGrantToMinion").(bool) {
						skill.cannotGrantToMinion = append(skill.cannotGrantToMinion, id)
					}
				}
			}
			skill.levels = append(skill.levels, level)
		}
		for _, stat := range listRows(statSets.Get("ImplicitStats")) {
			id := luaStr(stat.Get("Id"))
			if !statMap[id] {
				statMap[id] = true
				skill.stats = append(skill.stats, id)
			}
		}
		constStats := listRows(statSets.Get("ConstantStats"))
		constVals := statSets.Get("ConstantStatsValues").([]any)
		for i, stat := range constStats {
			skill.constantStats = append(skill.constantStats, [2]any{luaStr(stat.Get("Id")), constVals[i].(int64)})
		}
		for _, qsRow := range x.Dat("GrantedEffectQualityStats").GetRowList("GrantedEffect", granted) {
			statVals := qsRow.Get("StatValues").([]any)
			for j, stat := range listRows(qsRow.Get("GrantedStats")) {
				id := luaStr(stat.Get("Id"))
				if id != "dummy_stat_display_nothing" {
					skill.qualityStats = append(skill.qualityStats, [2]any{id, float64(statVals[j].(int64)) / 1000})
				}
			}
		}
	}
	directives["flags"] = func(args string, out *OutFile) {
		state.skill.baseFlags = append(state.skill.baseFlags, reWord.FindAllString(args, -1)...)
	}
	directives["baseMod"] = func(args string, out *OutFile) {
		state.skill.mods = append(state.skill.mods, args)
	}
	directives["mods"] = func(args string, out *OutFile) {
		skill := state.skill
		if !strings.Contains(args, "noBaseFlags") && !skill.isSupport {
			out.W("\tbaseFlags = {\n")
			for _, flag := range skill.baseFlags {
				out.W("\t\t", flag, " = true,\n")
			}
			out.W("\t},\n")
		}
		if !strings.Contains(args, "noBaseMods") && len(skill.mods) > 0 {
			out.W("\tbaseMods = {\n")
			for _, mod := range skill.mods {
				out.W("\t\t", mod, ",\n")
			}
			out.W("\t},\n")
		}
		if !strings.Contains(args, "noQualityStats") && len(skill.qualityStats) > 0 {
			out.W("\tqualityStats = {\n")
			for _, stat := range skill.qualityStats {
				out.W("\t\t{ \"", stat[0].(string), "\", ", luaStrAny(stat[1]), " },\n")
			}
			out.W("\t},\n")
		}
		if !strings.Contains(args, "noStats") {
			if len(skill.constantStats) > 0 {
				out.W("\tconstantStats = {\n")
				for _, stat := range skill.constantStats {
					out.W("\t\t{ \"", stat[0].(string), "\", ", luaStrAny(stat[1]), " },\n")
				}
				out.W("\t},\n")
			}
			out.W("\tstats = {\n")
			for _, stat := range skill.stats {
				out.W("\t\t\"", stat, "\",\n")
			}
			out.W("\t},\n")
			if len(skill.cannotGrantToMinion) > 0 {
				out.W("\tnotMinionStat = {\n")
				for _, stat := range skill.cannotGrantToMinion {
					out.W("\t\t\"", stat, "\",\n")
				}
				out.W("\t},\n")
			}
		}
		if !strings.Contains(args, "noLevels") {
			out.W("\tlevels = {\n")
			for _, level := range skill.levels {
				out.W("\t\t[", level.level, "] = { ")
				for _, v := range level.values {
					out.W(luaStrAny(v), ", ")
				}
				extraKeys := make([]string, 0, len(level.extra))
				for k := range level.extra {
					extraKeys = append(extraKeys, k)
				}
				sort.Strings(extraKeys)
				for _, k := range extraKeys {
					out.W(k, " = ", luaStrAny(level.extra[k]), ", ")
				}
				if len(level.interp.vals) > 0 {
					out.W("statInterpolation = { ")
					for _, t := range level.interp.vals {
						out.W(luaStrAny(t), ", ")
					}
					out.W("}, ")
				}
				if len(level.cost) > 0 {
					out.W("cost = { ")
					costKeys := make([]string, 0, len(level.cost))
					for k := range level.cost {
						costKeys = append(costKeys, k)
					}
					sort.Strings(costKeys)
					for _, k := range costKeys {
						out.W(k, " = ", level.cost[k], ", ")
					}
					out.W("}, ")
				}
				out.W("},\n")
			}
			out.W("\t},\n")
		}
		out.W("}")
		state.skill = nil
	}

	for _, name := range skillTemplateFiles {
		state = &skillsState{}
		out, err := x.ProcessTemplateFile(name, "Skills/", "../Data/Skills/", directives)
		if err != nil {
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}

	out := x.Out("Data/Gems.lua")
	out.W("-- This file is automatically generated, do not edit!\n")
	out.W("-- Gem data (c) Grinding Gear Games\n\nreturn {\n")
	x.Dat("SkillGems").Rows(func(skillGem *Row) bool {
		for _, ge := range listRows(skillGem.Get("GemVariants")) {
			gemEffectId := luaStr(ge.Get("Id"))
			if !gems[gemEffectId] {
				continue
			}
			// Some skills have an additional old version that exists in the
			// game files and messes up transfigured gem matching.
			delete(gems, gemEffectId)
			base := skillGem.Get("BaseItemType").(*Row)
			grantedEffect := ge.Get("GrantedEffect").(*Row)
			out.W("\t[\"", "Metadata/Items/Gems/SkillGem"+gemEffectId, "\"] = {\n")
			name := luaStr(base.Get("Name"))
			if !fullNameGems[luaStr(base.Get("Id"))] {
				if tn, ok := trueGemNames[gemEffectId]; ok {
					name = tn
				} else {
					name = strings.ReplaceAll(name, " Support", "")
				}
			}
			out.W("\t\tname = \"", name, "\",\n")
			// Hybrid gems (e.g. Vaal gems) use the display name of the
			// active skill e.g. Vaal Summon Skeletons of Sorcery
			if !skillGem.Get("IsSupport").(bool) {
				out.W("\t\tbaseTypeName = \"", luaStr(grantedEffect.Get("ActiveSkill").(*Row).Get("DisplayName")), "\",\n")
			}
			out.W("\t\tgameId = \"", luaStr(base.Get("Id")), "\",\n")
			out.W("\t\tvariantId = \"", gemEffectId, "\",\n")
			out.W("\t\tgrantedEffectId = \"", luaStr(grantedEffect.Get("Id")), "\",\n")
			if ge2, ok := ge.Get("GrantedEffect2").(*Row); ok {
				out.W("\t\tsecondaryGrantedEffectId = \"", luaStr(ge2.Get("Id")), "\",\n")
			}
			if sn := luaStr(ge.Get("SecondarySupportName")); len(sn) > 0 {
				out.W("\t\tsecondaryEffectName = \"", sn, "\",\n")
			}
			if skillGem.Get("IsVaalGem").(bool) {
				out.W("\t\tvaalGem = true,\n")
			}
			var tagNames []string
			out.W("\t\ttags = {\n")
			for _, tag := range listRows(ge.Get("Tags")) {
				escaped := escapeGGGString(luaStr(tag.Get("Name")))
				tag.SetCell("Name", escaped)
				out.W("\t\t\t", luaStr(tag.Get("Id")), " = true,\n")
				if len(escaped) > 0 {
					tagNames = append(tagNames, escaped)
				}
			}
			out.W("\t\t},\n")
			out.W("\t\ttagString = \"", strings.Join(tagNames, ", "), "\",\n")
			out.W("\t\treqStr = ", skillGem.Get("Str").(int64), ",\n")
			out.W("\t\treqDex = ", skillGem.Get("Dex").(int64), ",\n")
			out.W("\t\treqInt = ", skillGem.Get("Int").(int64), ",\n")
			naturalMaxLevel := len(x.Dat("ItemExperiencePerLevel").GetRowList("ItemExperienceType", skillGem.Get("GemLevelProgression")))
			if naturalMaxLevel == 0 {
				naturalMaxLevel = 1
			}
			out.W("\t\tnaturalMaxLevel = ", naturalMaxLevel, ",\n")
			out.W("\t},\n")
		}
		return true
	})
	out.W("}")
	return out.Close()
}
