// Port of .archive/src/Export/Scripts/skills.lua.

package export

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "skills", Build: buildSkills})
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

// skillDesc applies writeDesc's escaping.
func skillDesc(desc string) string {
	s := strings.ReplaceAll(desc, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return escapeGGGString(s)
}

func buildSkills(x *Ctx) (any, error) {
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
	var curFile *schema.SkillFile

	directives := map[string]func(args string){}
	directives["noGem"] = func(string) { state.noGem = true }
	directives["addSkillTypes"] = func(args string) {
		state.addSkillTypes = reWord.FindAllString(args, -1)
	}
	directives["skill"] = func(args string) {
		grantedId := args
		displayName := args
		if m := reSkillHead.FindStringSubmatch(args); m != nil {
			grantedId, displayName = m[1], m[2]
		}
		hdr := schema.SkillHeader{GrantedId: grantedId}
		granted := x.Dat("GrantedEffects").GetRow("Id", grantedId)
		if granted == nil {
			// the Lua ConPrintfs and leaves the previous skill state
			hdr.Invalid = true
			curFile.Skills = append(curFile.Skills, hdr)
			return
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
				hdr.Name = name
				if desc := luaStr(gemEffect.Get("Description")); len(desc) > 0 {
					d := skillDesc(desc)
					hdr.Description = &d
				}
			} else {
				activeName := luaStr(granted.Get("ActiveSkill").(*Row).Get("DisplayName"))
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
			if displayName == args && !isSupport {
				displayName = ""
				if gemEffect != nil {
					displayName = trueGemNames[luaStr(gemEffect.Get("Id"))]
				}
				if displayName == "" {
					displayName = luaStr(granted.Get("ActiveSkill").(*Row).Get("DisplayName"))
				}
			}
			hdr.Name = displayName
			hdr.Hidden = true
		}
		if skillGem != nil {
			if ft, ok := skillGem.Get("BaseItemType").(*Row).Get("FlavourTextKey").(*Row); ok {
				hdr.HasFlavour = true
				hdr.FlavourText = cleanAndSplitSkills(luaStr(ft.Get("Text")))
			}
		}
		state.noGem = false
		skill.addSkillTypes = state.addSkillTypes
		state.addSkillTypes = nil
		statSets := granted.Get("GrantedEffectStatSets").(*Row)
		hdr.Color = granted.Get("Attribute").(int64)
		if be := statSets.Get("BaseEffectiveness").(float64); be != 1 {
			hdr.BaseEffectiveness = &be
		}
		if ie := statSets.Get("IncrementalEffectiveness").(float64); ie != 0 {
			hdr.IncrementalEffectiveness = &ie
		}
		weaponTypesOf := func(classes []*Row) []string {
			weaponTypes := map[string]bool{}
			for _, class := range classes {
				if mapped, ok := weaponClassMap[luaStr(class.Get("Id"))]; ok {
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
			hdr.RequireSkillTypes = []string{}
			for _, typ := range listRows(granted.Get("SupportTypes")) {
				hdr.RequireSkillTypes = append(hdr.RequireSkillTypes, mapAST(typ))
			}
			hdr.AddSkillTypes = []string{}
			for _, typ := range listRows(granted.Get("AddTypes")) {
				typeString := mapAST(typ)
				if typeString == "SkillType.Triggered" {
					hdr.IsTrigger = true
				}
				hdr.AddSkillTypes = append(hdr.AddSkillTypes, typeString)
			}
			hdr.ExcludeSkillTypes = []string{}
			for _, typ := range listRows(granted.Get("ExcludeTypes")) {
				hdr.ExcludeSkillTypes = append(hdr.ExcludeSkillTypes, mapAST(typ))
			}
			hdr.SupportGemsOnly = granted.Get("SupportGemsOnly").(bool)
			hdr.IgnoreMinionTypes = granted.Get("IgnoreMinionTypes").(bool)
			if pv, ok := granted.Get("PlusVersionOf").(*Row); ok {
				id := luaStr(pv.Get("Id"))
				hdr.PlusVersionOf = &id
			}
			hdr.WeaponTypes = weaponTypesOf(listRows(granted.Get("WeaponRestrictions")))
			hdr.StatDescriptionScope = "gem_stat_descriptions"
		} else {
			activeSkill := granted.Get("ActiveSkill").(*Row)
			if desc := luaStr(activeSkill.Get("Description")); len(desc) > 0 {
				d := skillDesc(desc)
				hdr.Description = &d
			}
			hdr.SkillTypes = []string{}
			for _, typ := range listRows(activeSkill.Get("SkillTypes")) {
				hdr.SkillTypes = append(hdr.SkillTypes, mapAST(typ))
			}
			for _, typ := range skill.addSkillTypes {
				hdr.SkillTypes = append(hdr.SkillTypes, "SkillType."+typ)
			}
			for _, typ := range listRows(activeSkill.Get("MinionSkillTypes")) {
				hdr.MinionSkillTypes = append(hdr.MinionSkillTypes, mapAST(typ))
			}
			hdr.WeaponTypes = weaponTypesOf(listRows(activeSkill.Get("WeaponRestrictions")))
			scope := skillStatScope[luaStr(activeSkill.Get("Id"))]
			if scope == "" {
				scope = "skill_stat_descriptions"
			}
			hdr.StatDescriptionScope = scope
			if st := activeSkill.Get("SkillTotem").(int64); st <= 21 {
				hdr.SkillTotemId = &st
			}
			ct := float64(granted.Get("CastTime").(int64)) / 1000
			hdr.CastTime = &ct
			hdr.CannotBeSupported = granted.Get("CannotBeSupported").(bool)
		}
		curFile.Skills = append(curFile.Skills, hdr)

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
	directives["flags"] = func(args string) {
		state.skill.baseFlags = append(state.skill.baseFlags, reWord.FindAllString(args, -1)...)
	}
	directives["baseMod"] = func(args string) {
		state.skill.mods = append(state.skill.mods, args)
	}
	directives["mods"] = func(args string) {
		skill := state.skill
		tail := schema.SkillTail{
			ModsArgs:      args,
			Support:       skill.isSupport,
			BaseFlags:     append([]string(nil), skill.baseFlags...),
			BaseMods:      append([]string(nil), skill.mods...),
			Stats:         append([]string{}, skill.stats...),
			NotMinionStat: append([]string(nil), skill.cannotGrantToMinion...),
		}
		for _, stat := range skill.qualityStats {
			tail.QualityStats = append(tail.QualityStats, schema.StatValue{Id: stat[0].(string), Value: stat[1].(float64)})
		}
		for _, stat := range skill.constantStats {
			tail.ConstantStats = append(tail.ConstantStats, schema.StatValue{Id: stat[0].(string), Value: float64(stat[1].(int64))})
		}
		for _, level := range skill.levels {
			l := schema.SkillLevel{Level: level.level}
			for _, v := range level.values {
				switch n := v.(type) {
				case int64:
					l.Values = append(l.Values, float64(n))
				case float64:
					l.Values = append(l.Values, n)
				}
			}
			if len(level.extra) > 0 {
				l.Extra = map[string]float64{}
				for k, v := range level.extra {
					switch n := v.(type) {
					case int64:
						l.Extra[k] = float64(n)
					case float64:
						l.Extra[k] = n
					}
				}
			}
			for _, t := range level.interp.vals {
				l.Interp = append(l.Interp, luaStrAny(t))
			}
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
		if err := x.WalkTemplate(name, "Skills/", directives); err != nil {
			return nil, err
		}
		doc.Files[name] = f
	}

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
			g := schema.GemDef{VariantId: gemEffectId}
			name := luaStr(base.Get("Name"))
			if !fullNameGems[luaStr(base.Get("Id"))] {
				if tn, ok := trueGemNames[gemEffectId]; ok {
					name = tn
				} else {
					name = strings.ReplaceAll(name, " Support", "")
				}
			}
			g.Name = name
			// Hybrid gems (e.g. Vaal gems) use the display name of the
			// active skill e.g. Vaal Summon Skeletons of Sorcery
			if !skillGem.Get("IsSupport").(bool) {
				btn := luaStr(grantedEffect.Get("ActiveSkill").(*Row).Get("DisplayName"))
				g.BaseTypeName = &btn
			}
			g.GameId = luaStr(base.Get("Id"))
			g.GrantedEffectId = luaStr(grantedEffect.Get("Id"))
			if ge2, ok := ge.Get("GrantedEffect2").(*Row); ok {
				id := luaStr(ge2.Get("Id"))
				g.SecondaryGrantedEffectId = &id
			}
			if sn := luaStr(ge.Get("SecondarySupportName")); len(sn) > 0 {
				g.SecondaryEffectName = &sn
			}
			g.VaalGem = skillGem.Get("IsVaalGem").(bool)
			var tagNames []string
			g.Tags = []string{}
			for _, tag := range listRows(ge.Get("Tags")) {
				escaped := escapeGGGString(luaStr(tag.Get("Name")))
				tag.SetCell("Name", escaped)
				g.Tags = append(g.Tags, luaStr(tag.Get("Id")))
				if len(escaped) > 0 {
					tagNames = append(tagNames, escaped)
				}
			}
			g.TagString = strings.Join(tagNames, ", ")
			g.ReqStr = skillGem.Get("Str").(int64)
			g.ReqDex = skillGem.Get("Dex").(int64)
			g.ReqInt = skillGem.Get("Int").(int64)
			naturalMaxLevel := len(x.Dat("ItemExperiencePerLevel").GetRowList("ItemExperienceType", skillGem.Get("GemLevelProgression")))
			if naturalMaxLevel == 0 {
				naturalMaxLevel = 1
			}
			g.NaturalMaxLevel = naturalMaxLevel
			doc.Gems = append(doc.Gems, g)
		}
		return true
	})
	return doc, nil
}
