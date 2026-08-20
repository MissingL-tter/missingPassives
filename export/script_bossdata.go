// Port of .archive/src/Export/Scripts/bossData.lua.

package export

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func init() {
	Scripts = append(Scripts, Script{Name: "bossData", Outs: []string{"Data/BossSkills.lua", "Data/Bosses.lua"}, Run: scriptBossData})
}

var bossMonsterBaseDamage = []float64{
	4.9899997711182, 5.5599999427795, 6.1599998474121, 6.8099999427795, 7.5, 8.2299995422363, 9, 9.8199996948242, 10.699999809265, 11.619999885559,
	12.60000038147, 13.640000343323, 14.739999771118, 15.909999847412, 17.139999389648, 18.450000762939, 19.829999923706, 21.290000915527, 22.840000152588, 24.469999313354,
	26.190000534058, 28.010000228882, 29.940000534058, 31.959999084473, 34.110000610352, 36.360000610352, 38.75, 41.259998321533, 43.909999847412, 46.700000762939,
	49.650001525879, 52.75, 56.009998321533, 59.450000762939, 63.080001831055, 66.889999389648, 70.910003662109, 75.129997253418, 79.580001831055, 84.26000213623,
	89.180000305176, 94.349998474121, 99.800003051758, 105.51999664307, 111.5299987793, 117.86000061035, 124.5, 131.49000549316, 138.83000183105, 146.5299987793,
	154.63000488281, 163.13999938965, 172.07000732422, 181.44999694824, 191.30000305176, 201.63000488281, 212.47999572754, 223.86999511719, 235.83000183105, 248.36999511719,
	261.5299987793, 275.32998657227, 289.82000732422, 305.01000976562, 320.94000244141, 337.64999389648, 355.17999267578, 373.54998779297, 392.80999755859, 413.01000976562,
	434.17999267578, 456.36999511719, 479.61999511719, 504, 529.53997802734, 556.29998779297, 584.34997558594, 613.72998046875, 644.5, 676.75,
	710.52001953125, 745.89001464844, 782.94000244141, 821.72998046875, 862.35998535156, 904.90002441406, 949.44000244141, 996.07000732422, 1044.8900146484, 1096,
	1149.5, 1205.5, 1264.1099853516, 1325.4499511719, 1389.6400146484, 1456.8199462891, 1527.1199951172, 1600.6800537109, 1677.6400146484, 1758.1700439453,
}

// mbd is monsterBaseDamage[level] with Lua nil-for-missing semantics.
func mbd(level float64) (float64, bool) {
	i := int(level)
	if float64(i) != level || i < 1 || i > len(bossMonsterBaseDamage) {
		return 0, false
	}
	return bossMonsterBaseDamage[i-1], true
}

type bossOldMethodEntry struct {
	damage         map[string][2]float64
	uberDamage     float64
	hasUberMult    bool
	uberMapBoss    float64
	hasUberMapBoss bool
}

var bossOldMethod = map[string]bossOldMethodEntry{
	"AtlasBossFlickerSlam":           {damage: map[string][2]float64{"Physical": {1769.847, 0}}},
	"CleansingFireWall":              {damage: map[string][2]float64{"Fire": {3304.677, 20}}},
	"GSConsumeBossDisintegrateBeam":  {damage: map[string][2]float64{"Lightning": {3735.061, 50}}},
	"MavenSuperFireProjectileImpact": {damage: map[string][2]float64{"Fire": {4955.383, 0}}, uberDamage: 201, hasUberMult: true},
	"MavenMemoryGame":                {damage: map[string][2]float64{"Physical": {8626.344, 0}}},
}

var bossDamageTypes = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

var (
	reBossMon3       = regexp.MustCompile(`([0-9A-Za-z_]+) (.+) \{([0-9A-Za-z_]+)\}`)
	reBossMon2       = regexp.MustCompile(`([0-9A-Za-z_]+) (.+)`)
	reBossSkill4     = regexp.MustCompile(`([0-9A-Za-z_]+) (.+) ([0-9A-Za-z_]+) ([0-9A-Za-z_]+)`)
	reSkillArgs      = regexp.MustCompile(`([0-9A-Za-z_]+) ([0-9A-Za-z_]+)`)
	reSkillIndex     = regexp.MustCompile(`skillIndex = ([0-9A-Za-z_]+),`)
	reSkillIndexUber = regexp.MustCompile(`skillIndexUber = ([0-9A-Za-z_]+),`)
	reGranted2       = regexp.MustCompile(`GrantedEffectId2 = ([0-9A-Za-z_]+),`)
	reGrantedUber    = regexp.MustCompile(`GrantedEffectIdUber = ([0-9A-Za-z_]+),`)
	reExtraMult      = regexp.MustCompile(`ExtraDamageMult = ([0-9]+),`)
	reStages         = regexp.MustCompile(`stages = ([0-9A-Za-z_]+),`)
	reSpeedMult      = regexp.MustCompile(`speedMult = ([0-9]+),`)
)

func scriptBossData(x *Ctx) error {
	unique := x.Dat("Mods").GetRow("Id", "MonsterUnique5").Get("Stat1Value").(Interval)[0]
	uniqueAttackPenalty := x.Dat("Mods").GetRow("Id", "MonsterUnique8").Get("Stat1Value").(Interval)[0]
	rarityDamageMult := map[string]float64{
		"Unique":       1 + float64(unique)/100,
		"UniqueAttack": (1 + float64(unique)/100) * (1 - float64(uniqueAttackPenalty)/100),
	}
	monsterMapDifficultyMult := map[float64]float64{}
	x.Dat("MonsterMapDifficulty").Rows(func(row *Row) bool {
		monsterMapDifficultyMult[float64(row.Get("AreaLevel").(int64))] = 1 + float64(row.Get("DamagePercentIncrease").(int64))/100
		return true
	})
	mmdm := func(level float64) (float64, bool) {
		v, ok := monsterMapDifficultyMult[level]
		return v, ok
	}

	type bossInfo struct {
		displayName string
		damageRange int64
		damageMult  int64
		critChance  int64
		earlierUber bool
		mapBoss     bool
		rarity      string
	}
	type skillInfo struct {
		skillData      *Row
		statSets       *Row
		statsPerLevel  []*Row
		statsPerLevel2 []*Row
		skillDataUber  *Row
		grantedId      string
		grantedId2     string
		index          int
		hasIndex       bool
		uberIndex      int
		hasUberIndex   bool
		stages         float64
		hasStages      bool
		speedMult      float64
		hasSpeedMult   bool
	}
	type skillState struct {
		boss                 *bossInfo
		skill                *skillInfo
		skillList            []string
		DamageData           map[string]any
		DamageType           string
		SkillExtraDamageMult float64
	}
	var state *skillState

	statRowsAndVals := func(spl *Row, statsCol, valsCol string) ([]*Row, []any) {
		return listRows(spl.Get(statsCol)), spl.Get(valsCol).([]any)
	}

	getDamageType := func() string {
		skill := state.skill
		damageType := "Untyped"
		isHit := false
		for _, is := range listRows(skill.statSets.Get("ImplicitStats")) {
			if luaStr(is.Get("Id")) == "base_is_projectile" {
				damageType = "Projectile"
				break
			}
		}
		activeSkill := skill.skillData.Get("ActiveSkill").(*Row)
		for _, st := range listRows(activeSkill.Get("SkillTypes")) {
			switch luaStr(st.Get("Id")) {
			case "Attack":
				if damageType != "Projectile" {
					damageType = "Melee"
				}
			case "Spell":
				if damageType == "Projectile" {
					damageType = "SpellProjectile"
				} else {
					damageType = "Spell"
				}
			case "Projectile":
				if damageType == "Spell" {
					damageType = "SpellProjectile"
				} else {
					damageType = "Projectile"
				}
			}
		}
		for _, cf := range listRows(activeSkill.Get("StatContextFlags")) {
			switch luaStr(cf.Get("Id")) {
			case "AttackHit":
				isHit = true
				if damageType != "Projectile" {
					damageType = "Melee"
				}
			case "SpellHit":
				isHit = true
				if damageType == "Projectile" {
					damageType = "SpellProjectile"
				} else {
					damageType = "Spell"
				}
			case "Projectile":
				if damageType == "Spell" {
					damageType = "SpellProjectile"
				} else {
					damageType = "Projectile"
				}
			}
		}
		if !isHit {
			for _, cf := range listRows(activeSkill.Get("StatContextFlags")) {
				if luaStr(cf.Get("Id")) == "DamageOverTime" {
					return "DamageOverTime"
				}
			}
		}
		return damageType
	}

	calcSkillDamage := func() {
		monsterLevel := float64(84)
		skill := state.skill
		boss := state.boss
		grantedId := skill.grantedId
		if _, ok := bossOldMethod[grantedId]; !ok {
			if _, ok2 := bossOldMethod[skill.grantedId2]; ok2 {
				grantedId = skill.grantedId2
			}
		}
		var rarityType string
		if state.DamageType == "Melee" || state.DamageType == "Projectile" {
			if boss.rarity == "" {
				panic("bossData: rarity unset for attack skill (the Lua would error)")
			}
			rarityType = boss.rarity + "Attack"
		} else {
			rarityType = boss.rarity
		}
		extraDamageMult := [2]float64{1, 1}
		levelIndexes := []int{skill.index}
		if skill.hasUberIndex {
			levelIndexes = append(levelIndexes, skill.uberIndex)
		}
		for i, levelIndex := range levelIndexes {
			spl := skill.statsPerLevel[levelIndex-1]
			addStats, addVals := statRowsAndVals(spl, "AdditionalStats", "AdditionalStatsValues")
			for j, as := range addStats {
				if luaStr(as.Get("Id")) == "active_skill_damage_+%_final" {
					extraDamageMult[i] = 1 + float64(addVals[j].(int64))/100
					break
				}
			}
		}
		if skill.hasStages {
			stageMulti := 1.0
			constStats, constVals := statRowsAndVals(skill.statSets, "ConstantStats", "ConstantStatsValues")
			for i, cs := range constStats {
				if luaStr(cs.Get("Id")) == "charged_blast_spell_damage_+%_final_per_stack" {
					stageMulti = float64(constVals[i].(int64)) / 100
					break
				}
			}
			extraDamageMult[0] *= 1 + stageMulti*skill.stages
			extraDamageMult[1] *= 1 + stageMulti*skill.stages
		}
		if om, ok := bossOldMethod[grantedId]; ok {
			mapMult := 1.0
			if boss.mapBoss {
				if v, found := mmdm(monsterLevel); found {
					mapMult = v
				} else {
					mapMult = 1
				}
			}
			base, _ := mbd(monsterLevel - 1)
			if base == 0 {
				base = 1
			}
			rdm, found := rarityDamageMult[rarityType]
			if !found {
				rdm = 1
			}
			baseDamageMult := state.SkillExtraDamageMult * extraDamageMult[0] * float64(boss.damageMult) / 100 * rdm * mapMult / base
			for _, damageType := range bossDamageTypes {
				if dv, has := om.damage[damageType]; has {
					damageRange := dv[1] / 100
					if dv[1] == 0 {
						damageRange = float64(boss.damageRange) / 100
					}
					damageMult := dv[0] * baseDamageMult
					state.DamageData[damageType+"DamageMultMin"] = damageMult * (1 - damageRange)
					state.DamageData[damageType+"DamageMultMax"] = damageMult * (1 + damageRange)
				}
			}
			mapRatio := 1.0
			if boss.mapBoss {
				a, _ := mmdm(monsterLevel + 1)
				b, _ := mmdm(monsterLevel)
				mapRatio = a / b
			}
			if extraDamageMult[0] != extraDamageMult[1] {
				state.DamageData["SkillUberDamageMult"] = 100 * extraDamageMult[1] / extraDamageMult[0] * mapRatio
			} else if om.hasUberMult {
				state.DamageData["SkillUberDamageMult"] = om.uberDamage * mapRatio
			} else if boss.mapBoss {
				lvl := monsterLevel + 1
				if om.hasUberMapBoss {
					lvl = om.uberMapBoss
				}
				a, _ := mmdm(lvl)
				b, _ := mmdm(monsterLevel)
				state.DamageData["SkillUberDamageMult"] = 100 * (a / b)
			}
		} else {
			// new method
			baseDamages := map[string]float64{}
			for i, levelIndex := range levelIndexes {
				var spl *Row
				if grantedId == skill.grantedId2 && skill.grantedId2 != "" {
					spl = skill.statsPerLevel2[levelIndex-1]
				} else {
					spl = skill.statsPerLevel[levelIndex-1]
				}
				floatStats, baseVals := statRowsAndVals(spl, "FloatStats", "BaseResolvedValues")
				suffix := luaStr(i + 1)
				for j, fs := range floatStats {
					id := luaStr(fs.Get("Id"))
					for _, dt := range bossDamageTypes {
						lower := strings.ToLower(dt)
						if id == "spell_minimum_base_"+lower+"_damage" {
							baseDamages["min"+dt+suffix] = 1 + float64(baseVals[j].(int64))
						} else if id == "spell_maximum_base_"+lower+"_damage" {
							baseDamages["max"+dt+suffix] = 1 + float64(baseVals[j].(int64))
						}
					}
				}
				if state.DamageType == "DamageOverTime" {
					for j, fs := range floatStats {
						id := luaStr(fs.Get("Id"))
						for _, dt := range bossDamageTypes {
							lower := strings.ToLower(dt)
							if id == "base_"+lower+"_damage_to_deal_per_minute" {
								baseDamages["min"+dt+suffix] = 1 + float64(baseVals[j].(int64))/60
								baseDamages["max"+dt+suffix] = 1 + float64(baseVals[j].(int64))/60
							}
						}
					}
				}
			}
			monsterLevel = skill.statsPerLevel[skill.index-1].Get("PlayerLevelReq").(float64)
			mapMult := 1.0
			if boss.mapBoss {
				if v, found := mmdm(monsterLevel); found {
					mapMult = v
				} else {
					mapMult = 1
				}
			}
			base, found := mbd(monsterLevel)
			if !found {
				base = 1
			}
			rdm, foundR := rarityDamageMult[rarityType]
			if !foundR {
				rdm = 1
			}
			damageMult := state.SkillExtraDamageMult * extraDamageMult[0] * rdm * mapMult / base
			for _, dt := range bossDamageTypes {
				mn, hasMin := baseDamages["min"+dt+"1"]
				mx, hasMax := baseDamages["max"+dt+"1"]
				if hasMin || hasMax {
					if !hasMin {
						mn = 0
					}
					if !hasMax {
						mx = 0
					}
					state.DamageData[dt+"DamageMultMin"] = damageMult * mn
					state.DamageData[dt+"DamageMultMax"] = damageMult * mx
				}
			}
			if skill.hasUberIndex {
				skillUber := 0.0
				skillBase := 0.0
				uberMonsterLevel := skill.statsPerLevel[skill.uberIndex-1].Get("PlayerLevelReq").(float64)
				for _, dt := range bossDamageTypes {
					mn1, hasMin1 := baseDamages["min"+dt+"1"]
					_, hasMax1 := baseDamages["max"+dt+"1"]
					if hasMin1 || hasMax1 {
						if !hasMin1 {
							panic("bossData: min damage missing (the Lua would error)")
						}
						mn2, hasMin2 := baseDamages["min"+dt+"2"]
						if !hasMin2 {
							panic("bossData: uber min damage missing (the Lua would error)")
						}
						skillBase += mn1
						skillUber += mn2
					}
				}
				ub, foundU := mbd(uberMonsterLevel)
				if !foundU {
					ub = 1
				}
				bb, foundB := mbd(monsterLevel)
				if !foundB {
					bb = 1
				}
				ratio := 1.0
				if boss.mapBoss {
					a, _ := mmdm(uberMonsterLevel)
					b, _ := mmdm(monsterLevel)
					ratio = a / b
				}
				uberMult := (skillUber / ub) / (skillBase / bb) * ratio
				if uberMult > 1.15 || uberMult < 0.85 {
					state.DamageData["SkillUberDamageMult"] = math.Ceil(uberMult * 100)
				}
			}
		}
	}

	getPenetration := func() bool {
		dd := state.DamageData
		for _, k := range []string{"PhysOverwhelm", "PhysUberOverwhelm", "LightningPen", "LightningUberPen", "ColdPen", "ColdUberPen", "FirePen", "FireUberPen", "ChaosPen", "ChaosUberPen"} {
			dd[k] = float64(0)
		}
		scan := func(levels []*Row) {
			for level, spl := range levels {
				addStats, addVals := statRowsAndVals(spl, "AdditionalStats", "AdditionalStatsValues")
				uber := ""
				if level > 0 {
					uber = "Uber"
				}
				for i, as := range addStats {
					switch luaStr(as.Get("Id")) {
					case "base_reduce_enemy_lightning_resistance_%":
						dd["Lightning"+uber+"Pen"] = float64(addVals[i].(int64))
					case "base_reduce_enemy_cold_resistance_%":
						dd["Cold"+uber+"Pen"] = float64(addVals[i].(int64))
					case "base_reduce_enemy_fire_resistance_%":
						dd["Fire"+uber+"Pen"] = float64(addVals[i].(int64))
					case "base_reduce_enemy_chaos_resistance_%":
						dd["Chaos"+uber+"Pen"] = float64(addVals[i].(int64))
					}
				}
			}
		}
		scan(state.skill.statsPerLevel)
		if state.skill.statsPerLevel2 != nil {
			scan(state.skill.statsPerLevel2)
		}
		zero := func(v any) bool { f, ok := v.(float64); return ok && f == 0 }
		if zero(dd["PhysOverwhelm"]) && !zero(dd["PhysUberOverwhelm"]) {
			dd["PhysOverwhelm"] = ""
		}
		for _, dt := range []string{"Lightning", "Cold", "Fire"} {
			if zero(dd[dt+"Pen"]) && !zero(dd[dt+"UberPen"]) {
				dd[dt+"Pen"] = ""
			}
		}
		return !zero(dd["PhysOverwhelm"]) || !zero(dd["LightningPen"]) || !zero(dd["ColdPen"]) || !zero(dd["FirePen"])
	}

	getSpeed := func() (float64, float64, bool, bool) {
		skill := state.skill
		speed := float64(skill.skillData.Get("CastTime").(int64))
		var uberSpeed float64
		hasUber := false
		if skill.skillDataUber != nil {
			uberSpeed = float64(skill.skillDataUber.Get("CastTime").(int64))
			hasUber = true
		}
		speedMult := [2]float64{0, 0}
		for level, spl := range skill.statsPerLevel {
			if level > 1 {
				break
			}
			addStats, addVals := statRowsAndVals(spl, "AdditionalStats", "AdditionalStatsValues")
			for i, as := range addStats {
				id := luaStr(as.Get("Id"))
				if id == "active_skill_attack_speed_+%_final" || id == "active_skill_cast_speed_+%_final" {
					speedMult[level] = 100 + float64(addVals[i].(int64))
					break
				}
			}
		}
		if skill.hasSpeedMult {
			speed = speed * skill.speedMult / 10000
			if hasUber {
				uberSpeed = uberSpeed * skill.speedMult / 10000
			}
		}
		if skill.hasStages {
			speed = speed * skill.stages
			if hasUber {
				uberSpeed = uberSpeed * skill.stages
			}
		}
		if speedMult[0] != 0 {
			if speedMult[0] != speedMult[1] {
				return math.Ceil(speed / speedMult[0] * 100), math.Ceil(speed / speedMult[1] * 100), true, true
			}
			speed = speed / speedMult[0] * 100
			if !hasUber {
				panic("bossData: uberSpeed nil in speed normalisation (the Lua would error)")
			}
			uberSpeed = uberSpeed / speedMult[0] * 100
		}
		if hasUber {
			return math.Ceil(speed), math.Ceil(uberSpeed), true, true
		}
		return math.Ceil(speed), 0, true, false
	}

	type addStatsSet struct {
		vals  map[string]any
		count int64
	}
	getAdditionalStats := func() (base, uber *addStatsSet) {
		skill := state.skill
		base = &addStatsSet{vals: map[string]any{}}
		uber = &addStatsSet{vals: map[string]any{}}
		for level, spl := range skill.statsPerLevel {
			if level > 1 {
				break
			}
			set := base
			if level == 1 {
				set = uber
			}
			addStats, addVals := statRowsAndVals(spl, "AdditionalStats", "AdditionalStatsValues")
			for i, as := range addStats {
				switch luaStr(as.Get("Id")) {
				case "global_reduce_enemy_block_%":
					set.vals["reduceEnemyBlock"] = float64(addVals[i].(int64))
					set.count++
				case "reduce_enemy_dodge_%":
					set.vals["reduceEnemyDodge"] = float64(addVals[i].(int64))
					set.count++
				}
			}
			for _, as := range listRows(spl.Get("AdditionalBooleanStats")) {
				switch luaStr(as.Get("Id")) {
				case "global_always_hit":
					set.vals["CannotBeEvaded"] = "\"flag\""
					set.count++
				case "cannot_be_blocked_or_dodged_or_suppressed":
					set.vals["CannotBeBlocked"] = "\"flag\""
					set.vals["CannotBeDodged"] = "\"flag\""
					set.vals["CannotBeSuppressed"] = "\"flag\""
					if level == 0 {
						set.count += 3
					} else {
						// The Lua adds to base.count here (a bug it keeps).
						set.count = base.count + 3
					}
				}
			}
		}
		for _, is := range listRows(skill.statSets.Get("ImplicitStats")) {
			if luaStr(is.Get("Id")) == "cannot_be_blocked_or_dodged_or_suppressed" {
				for _, set := range []*addStatsSet{base, uber} {
					set.vals["CannotBeBlocked"] = "\"flag\""
					set.vals["CannotBeDodged"] = "\"flag\""
					set.vals["CannotBeSuppressed"] = "\"flag\""
					set.count += 3
				}
			}
		}
		constStats, constVals := statRowsAndVals(skill.statSets, "ConstantStats", "ConstantStatsValues")
		for i, cs := range constStats {
			var name string
			switch luaStr(cs.Get("Id")) {
			case "skill_physical_damage_%_to_convert_to_lightning":
				name = "PhysicalDamageSkillConvertToLightning"
			case "skill_physical_damage_%_to_convert_to_cold":
				name = "PhysicalDamageSkillConvertToCold"
			case "skill_physical_damage_%_to_convert_to_fire":
				name = "PhysicalDamageSkillConvertToFire"
			case "skill_physical_damage_%_to_convert_to_chaos":
				name = "PhysicalDamageSkillConvertToChaos"
			}
			if name != "" {
				base.vals[name] = float64(constVals[i].(int64))
				base.count++
				uber.vals[name] = float64(constVals[i].(int64))
				uber.count++
			}
		}
		if base.count == 0 && uber.count == 0 {
			return nil, nil
		}
		return base, uber
	}

	writeStatSet := func(out *OutFile, set *addStatsSet) {
		keys := make([]string, 0, len(set.vals))
		for k := range set.vals {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				out.W(",")
			}
			out.W("\n\t\t\t\t", k, " = ")
			if s, ok := set.vals[k].(string); ok {
				out.W(s)
			} else {
				out.W(luaNum(set.vals[k].(float64)))
			}
		}
	}

	skillsDirectives := map[string]func(args string, out *OutFile){}
	skillsDirectives["boss"] = func(args string, out *OutFile) {
		m := reBossSkill4.FindStringSubmatch(args)
		bossData := x.Dat("MonsterVarieties").GetRow("Id", m[2])
		b := &bossInfo{
			displayName: m[1],
			damageRange: bossData.Get("Type").(*Row).Get("DamageSpread").(int64),
			damageMult:  bossData.Get("DamageMultiplier").(int64),
			critChance:  5,
		}
		if m[3] == "true" {
			b.earlierUber = true
		}
		if m[4] == "true" {
			b.mapBoss = true
		}
		for _, mod := range listRows(bossData.Get("Mods")) {
			if luaStr(mod.Get("Id")) == "MonsterMapBoss" {
				b.rarity = "Unique"
				break
			}
		}
		state.boss = b
	}
	skillsDirectives["skill"] = func(args string, out *OutFile) {
		m := reSkillArgs.FindStringSubmatch(args)
		displayName, grantedId := m[1], m[2]
		switch displayName {
		case "MemoryGame":
			displayName = "Memory Game"
		case "GroundDegen":
			displayName = "Ground Degen"
		}
		boss := state.boss
		state.skillList = append(state.skillList, boss.displayName+" "+displayName)
		skill := &skillInfo{grantedId: grantedId}
		skill.skillData = x.Dat("GrantedEffects").GetRow("Id", grantedId)
		skill.statSets = x.Dat("GrantedEffectStatSets").GetRow("Id", grantedId)
		skill.statsPerLevel = x.Dat("GrantedEffectStatSetsPerLevel").GetRowList("GrantedEffect", skill.skillData)
		state.skill = skill
		skill.index = 1
		skill.hasIndex = true
		if im := reSkillIndex.FindStringSubmatch(args); im != nil {
			if im[1] == "nil" {
				skill.hasIndex = false
			} else {
				n, _ := strconv.Atoi(im[1])
				skill.index = n
			}
		}
		if um := reSkillIndexUber.FindStringSubmatch(args); um != nil {
			if um[1] == "nil" {
				skill.hasUberIndex = false
			} else {
				n, _ := strconv.Atoi(um[1])
				skill.uberIndex = n
				skill.hasUberIndex = true
			}
		} else if skill.hasIndex {
			skill.uberIndex = skill.index + 1
			skill.hasUberIndex = true
		}
		if g2 := reGranted2.FindStringSubmatch(args); g2 != nil {
			skill.grantedId2 = g2[1]
			sd2 := x.Dat("GrantedEffects").GetRow("Id", g2[1])
			skill.statsPerLevel2 = x.Dat("GrantedEffectStatSetsPerLevel").GetRowList("GrantedEffect", sd2)
		}
		if gu := reGrantedUber.FindStringSubmatch(args); gu != nil {
			skill.skillDataUber = x.Dat("GrantedEffects").GetRow("Id", gu[1])
		}
		state.SkillExtraDamageMult = 1
		if em := reExtraMult.FindStringSubmatch(args); em != nil {
			n, _ := strconv.ParseFloat(em[1], 64)
			state.SkillExtraDamageMult = n / 100
		}
		if sm := reStages.FindStringSubmatch(args); sm != nil {
			n, _ := strconv.ParseFloat(sm[1], 64)
			skill.stages = n
			skill.hasStages = true
		}
		state.DamageData = map[string]any{}
		state.DamageType = getDamageType()
		calcSkillDamage()
		// output
		out.W("	[\"", boss.displayName, " ", displayName, "\"] = {\n")
		out.W("		DamageType = \"", state.DamageType, "\",\n")
		out.W("		DamageMultipliers = {\n")
		dCount := 0
		for _, dt := range bossDamageTypes {
			if mn, ok := state.DamageData[dt+"DamageMultMin"].(float64); ok {
				dCount++
				if dCount > 1 {
					out.W(",\n")
				}
				mx := state.DamageData[dt+"DamageMultMax"].(float64)
				out.W("			", dt, " = { ", luaNum(mn), ", ", luaNum((mx-mn)/100), " }")
			}
		}
		out.W("\n		}")
		if um, ok := state.DamageData["SkillUberDamageMult"].(float64); ok {
			out.W(",\n		UberDamageMultiplier = ", luaNum(um/100))
		}
		if getPenetration() {
			out.W(",\n		DamagePenetrations = {\n")
			dCount = 0
			writePen := func(keys []string, strip bool) {
				for _, penType := range keys {
					v := state.DamageData[penType]
					if f, ok := v.(float64); ok && f == 0 {
						continue
					}
					dCount++
					if dCount > 1 {
						out.W(",\n")
					}
					name := penType
					if strip {
						name = strings.ReplaceAll(name, "Uber", "")
					}
					out.W("			", name, " = ")
					if s, ok := v.(string); ok && s == "" {
						out.W("\"\"")
					} else {
						out.W(luaNum(v.(float64)))
					}
				}
			}
			writePen([]string{"PhysOverwhelm", "LightningPen", "ColdPen", "FirePen"}, false)
			out.W("\n		}")
			nonZero := func(k string) bool {
				f, ok := state.DamageData[k].(float64)
				return !(ok && f == 0)
			}
			if nonZero("PhysUberOverwhelm") || nonZero("LightningUberPen") || nonZero("ColdUberPen") || nonZero("FireUberPen") {
				out.W(",\n		UberDamagePenetrations = {\n")
				dCount = 0
				writePen([]string{"PhysUberOverwhelm", "LightningUberPen", "ColdUberPen", "FireUberPen"}, true)
				out.W("\n		}")
			}
		}
		if sm := reSpeedMult.FindStringSubmatch(args); sm != nil {
			n, _ := strconv.ParseFloat(sm[1], 64)
			state.skill.speedMult = n
			state.skill.hasSpeedMult = true
		}
		speed, uberSpeed, _, hasUberSpeed := getSpeed()
		if speed != 700 {
			out.W(",\n		speed = ", luaNum(speed))
		}
		if hasUberSpeed && uberSpeed != 700 {
			out.W(",\n		UberSpeed = ", luaNum(uberSpeed))
		}
		critChance := int64(math.Ceil(float64(skill.statsPerLevel[0].Get("AttackCritChance").(int64)) / 100))
		if critChance != 5 {
			out.W(",\n		critChance = ", critChance)
		}
		if boss.earlierUber {
			out.W(",\n		earlierUber = true")
		}
		baseSet, uberSet := getAdditionalStats()
		if baseSet != nil {
			out.W(",\n		additionalStats = {")
			if baseSet.count > 0 {
				out.W("\n			base = {")
				writeStatSet(out, baseSet)
				out.W("\n			}")
			}
			if uberSet.count > 0 {
				if baseSet.count > 0 {
					out.W(",")
				}
				out.W("\n			uber = {")
				writeStatSet(out, uberSet)
				out.W("\n			}")
			}
			out.W("\n		}")
		}
	}
	skillsDirectives["tooltip"] = func(args string, out *OutFile) {
		out.W(",\n		tooltip = ", args, "\n")
		out.W("	},\n")
		state.skill = nil
	}
	skillsDirectives["skillList"] = func(args string, out *OutFile) {
		out.W("},{\n")
		out.W("    { val = \"None\", label = \"None\" }")
		for _, skillName := range state.skillList {
			out.W(",\n    { val = \"", skillName, "\", label = \"", skillName, "\" }")
		}
		out.W("\n}")
		state.boss = nil
		state.skillList = nil
	}

	monstersDirectives := map[string]func(args string, out *OutFile){}
	monstersDirectives["boss"] = func(args string, out *OutFile) {
		var displayName, monsterId string
		isUber := false
		if m := reBossMon3.FindStringSubmatch(args); m != nil {
			displayName, monsterId = m[1], m[2]
			isUber = true
		} else if m := reBossMon2.FindStringSubmatch(args); m != nil {
			displayName, monsterId = m[1], m[2]
		}
		monsterType := x.Dat("MonsterTypes").GetRow("Id", monsterId)
		if monsterType == nil {
			return // the Lua prints "Invalid Type"
		}
		out.W("bosses[\"", displayName, "\"] = {\n")
		out.W("\tarmourMult = ", monsterType.Get("Armour").(int64), ",\n")
		out.W("\tevasionMult = ", monsterType.Get("Evasion").(int64), ",\n")
		if isUber {
			out.W("\tisUber = true,\n")
		} else {
			out.W("\tisUber = false,\n")
		}
		out.W("}\n")
	}

	state = &skillState{}
	out, err := x.ProcessTemplateFile("BossSkills", "Enemies/", "../Data/", skillsDirectives)
	if err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	out, err = x.ProcessTemplateFile("Bosses", "Enemies/", "../Data/", monstersDirectives)
	if err != nil {
		return err
	}
	return out.Close()
}
