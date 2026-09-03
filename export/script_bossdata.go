// Port of .archive/src/Export/Scripts/bossData.lua.

package export

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "bossData", Build: buildBossData})
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

// penVal is one penetration/overwhelm value; blank stands for the
// reference's "" (a zero base value whose uber value is non-zero).
type penVal struct {
	n     float64
	blank bool
}

func buildBossData(x *Ctx) (schema.Document, error) {
	var doc schema.BossData
	var (
		mods, mapDifficulty, monsterVarieties, grantedEffects *DatFile
		statSetsDat, statSetsPerLevel, monsterTypes           *DatFile
	)
	for name, dst := range map[string]**DatFile{
		"Mods":                          &mods,
		"MonsterMapDifficulty":          &mapDifficulty,
		"MonsterVarieties":              &monsterVarieties,
		"GrantedEffects":                &grantedEffects,
		"GrantedEffectStatSets":         &statSetsDat,
		"GrantedEffectStatSetsPerLevel": &statSetsPerLevel,
		"MonsterTypes":                  &monsterTypes,
	} {
		var err error
		if *dst, err = x.Dat(name); err != nil {
			return nil, err
		}
	}
	unique := mods.RowByStr("Id", "MonsterUnique5").Ivl("Stat1Value")[0]
	uniqueAttackPenalty := mods.RowByStr("Id", "MonsterUnique8").Ivl("Stat1Value")[0]
	rarityDamageMult := map[string]float64{
		"Unique":       1 + float64(unique)/100,
		"UniqueAttack": (1 + float64(unique)/100) * (1 - float64(uniqueAttackPenalty)/100),
	}
	monsterMapDifficultyMult := map[float64]float64{}
	for row := range mapDifficulty.Rows() {
		monsterMapDifficultyMult[float64(row.Int("AreaLevel"))] = 1 + float64(row.Int("DamagePercentIncrease"))/100
	}
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
		DamageMult           map[string]float64 // <Type>DamageMultMin/Max, SkillUberDamageMult
		Pen                  map[string]penVal
		DamageType           string
		SkillExtraDamageMult float64
	}
	var state *skillState

	statRowsAndVals := func(spl *Row, statsCol, valsCol string) ([]*Row, []int64) {
		return spl.Refs(statsCol), spl.Ints(valsCol)
	}

	getDamageType := func() string {
		skill := state.skill
		damageType := "Untyped"
		isHit := false
		for _, is := range skill.statSets.Refs("ImplicitStats") {
			if is.Str("Id") == "base_is_projectile" {
				damageType = "Projectile"
				break
			}
		}
		activeSkill := skill.skillData.Ref("ActiveSkill")
		for _, st := range activeSkill.Refs("SkillTypes") {
			switch st.Str("Id") {
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
		for _, cf := range activeSkill.Refs("StatContextFlags") {
			switch cf.Str("Id") {
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
			for _, cf := range activeSkill.Refs("StatContextFlags") {
				if cf.Str("Id") == "DamageOverTime" {
					return "DamageOverTime"
				}
			}
		}
		return damageType
	}

	calcSkillDamage := func() error {
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
				return fmt.Errorf("%s: rarity unset for attack skill (the Lua would error)", grantedId)
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
				if as.Str("Id") == "active_skill_damage_+%_final" {
					extraDamageMult[i] = 1 + float64(addVals[j])/100
					break
				}
			}
		}
		if skill.hasStages {
			stageMulti := 1.0
			constStats, constVals := statRowsAndVals(skill.statSets, "ConstantStats", "ConstantStatsValues")
			for i, cs := range constStats {
				if cs.Str("Id") == "charged_blast_spell_damage_+%_final_per_stack" {
					stageMulti = float64(constVals[i]) / 100
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
					state.DamageMult[damageType+"DamageMultMin"] = damageMult * (1 - damageRange)
					state.DamageMult[damageType+"DamageMultMax"] = damageMult * (1 + damageRange)
				}
			}
			mapRatio := 1.0
			if boss.mapBoss {
				a, _ := mmdm(monsterLevel + 1)
				b, _ := mmdm(monsterLevel)
				mapRatio = a / b
			}
			if extraDamageMult[0] != extraDamageMult[1] {
				state.DamageMult["SkillUberDamageMult"] = 100 * extraDamageMult[1] / extraDamageMult[0] * mapRatio
			} else if om.hasUberMult {
				state.DamageMult["SkillUberDamageMult"] = om.uberDamage * mapRatio
			} else if boss.mapBoss {
				lvl := monsterLevel + 1
				if om.hasUberMapBoss {
					lvl = om.uberMapBoss
				}
				a, _ := mmdm(lvl)
				b, _ := mmdm(monsterLevel)
				state.DamageMult["SkillUberDamageMult"] = 100 * (a / b)
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
				suffix := strconv.Itoa(i + 1)
				for j, fs := range floatStats {
					id := fs.Str("Id")
					for _, dt := range bossDamageTypes {
						lower := strings.ToLower(dt)
						if id == "spell_minimum_base_"+lower+"_damage" {
							baseDamages["min"+dt+suffix] = 1 + float64(baseVals[j])
						} else if id == "spell_maximum_base_"+lower+"_damage" {
							baseDamages["max"+dt+suffix] = 1 + float64(baseVals[j])
						}
					}
				}
				if state.DamageType == "DamageOverTime" {
					for j, fs := range floatStats {
						id := fs.Str("Id")
						for _, dt := range bossDamageTypes {
							lower := strings.ToLower(dt)
							if id == "base_"+lower+"_damage_to_deal_per_minute" {
								baseDamages["min"+dt+suffix] = 1 + float64(baseVals[j])/60
								baseDamages["max"+dt+suffix] = 1 + float64(baseVals[j])/60
							}
						}
					}
				}
			}
			monsterLevel = skill.statsPerLevel[skill.index-1].Float("PlayerLevelReq")
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
					state.DamageMult[dt+"DamageMultMin"] = damageMult * mn
					state.DamageMult[dt+"DamageMultMax"] = damageMult * mx
				}
			}
			if skill.hasUberIndex {
				skillUber := 0.0
				skillBase := 0.0
				uberMonsterLevel := skill.statsPerLevel[skill.uberIndex-1].Float("PlayerLevelReq")
				for _, dt := range bossDamageTypes {
					mn1, hasMin1 := baseDamages["min"+dt+"1"]
					_, hasMax1 := baseDamages["max"+dt+"1"]
					if hasMin1 || hasMax1 {
						if !hasMin1 {
							return fmt.Errorf("%s: min %s damage missing (the Lua would error)", grantedId, dt)
						}
						mn2, hasMin2 := baseDamages["min"+dt+"2"]
						if !hasMin2 {
							return fmt.Errorf("%s: uber min %s damage missing (the Lua would error)", grantedId, dt)
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
					state.DamageMult["SkillUberDamageMult"] = math.Ceil(uberMult * 100)
				}
			}
		}
		return nil
	}

	getPenetration := func() bool {
		dd := state.Pen
		for _, k := range []string{"PhysOverwhelm", "PhysUberOverwhelm", "LightningPen", "LightningUberPen", "ColdPen", "ColdUberPen", "FirePen", "FireUberPen", "ChaosPen", "ChaosUberPen"} {
			dd[k] = penVal{}
		}
		scan := func(levels []*Row) {
			for level, spl := range levels {
				addStats, addVals := statRowsAndVals(spl, "AdditionalStats", "AdditionalStatsValues")
				uber := ""
				if level > 0 {
					uber = "Uber"
				}
				for i, as := range addStats {
					switch as.Str("Id") {
					case "base_reduce_enemy_lightning_resistance_%":
						dd["Lightning"+uber+"Pen"] = penVal{n: float64(addVals[i])}
					case "base_reduce_enemy_cold_resistance_%":
						dd["Cold"+uber+"Pen"] = penVal{n: float64(addVals[i])}
					case "base_reduce_enemy_fire_resistance_%":
						dd["Fire"+uber+"Pen"] = penVal{n: float64(addVals[i])}
					case "base_reduce_enemy_chaos_resistance_%":
						dd["Chaos"+uber+"Pen"] = penVal{n: float64(addVals[i])}
					}
				}
			}
		}
		scan(state.skill.statsPerLevel)
		if state.skill.statsPerLevel2 != nil {
			scan(state.skill.statsPerLevel2)
		}
		zero := func(k string) bool { v := dd[k]; return !v.blank && v.n == 0 }
		if zero("PhysOverwhelm") && !zero("PhysUberOverwhelm") {
			dd["PhysOverwhelm"] = penVal{blank: true}
		}
		for _, dt := range []string{"Lightning", "Cold", "Fire"} {
			if zero(dt+"Pen") && !zero(dt+"UberPen") {
				dd[dt+"Pen"] = penVal{blank: true}
			}
		}
		return !zero("PhysOverwhelm") || !zero("LightningPen") || !zero("ColdPen") || !zero("FirePen")
	}

	getSpeed := func() (speed, uberSpeed float64, hasUberSpeed bool, err error) {
		skill := state.skill
		speed = float64(skill.skillData.Int("CastTime"))
		hasUber := false
		if skill.skillDataUber != nil {
			uberSpeed = float64(skill.skillDataUber.Int("CastTime"))
			hasUber = true
		}
		speedMult := [2]float64{0, 0}
		for level, spl := range skill.statsPerLevel {
			if level > 1 {
				break
			}
			addStats, addVals := statRowsAndVals(spl, "AdditionalStats", "AdditionalStatsValues")
			for i, as := range addStats {
				id := as.Str("Id")
				if id == "active_skill_attack_speed_+%_final" || id == "active_skill_cast_speed_+%_final" {
					speedMult[level] = 100 + float64(addVals[i])
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
				return math.Ceil(speed / speedMult[0] * 100), math.Ceil(speed / speedMult[1] * 100), true, nil
			}
			speed = speed / speedMult[0] * 100
			if !hasUber {
				return 0, 0, false, fmt.Errorf("%s: uberSpeed nil in speed normalisation (the Lua would error)", skill.grantedId)
			}
			uberSpeed = uberSpeed / speedMult[0] * 100
		}
		if hasUber {
			return math.Ceil(speed), math.Ceil(uberSpeed), true, nil
		}
		return math.Ceil(speed), 0, false, nil
	}

	// addStatsSet holds the extra stat lines: a number each, or a flag.
	type addStatsSet struct {
		vals  map[string]schema.BossStatValue
		count int64
	}
	flagStat := schema.BossStatValue{Flag: true}
	getAdditionalStats := func() (base, uber *addStatsSet) {
		skill := state.skill
		base = &addStatsSet{vals: map[string]schema.BossStatValue{}}
		uber = &addStatsSet{vals: map[string]schema.BossStatValue{}}
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
				switch as.Str("Id") {
				case "global_reduce_enemy_block_%":
					set.vals["reduceEnemyBlock"] = schema.BossStatValue{Value: float64(addVals[i])}
					set.count++
				case "reduce_enemy_dodge_%":
					set.vals["reduceEnemyDodge"] = schema.BossStatValue{Value: float64(addVals[i])}
					set.count++
				}
			}
			for _, as := range spl.Refs("AdditionalBooleanStats") {
				switch as.Str("Id") {
				case "global_always_hit":
					set.vals["CannotBeEvaded"] = flagStat
					set.count++
				case "cannot_be_blocked_or_dodged_or_suppressed":
					set.vals["CannotBeBlocked"] = flagStat
					set.vals["CannotBeDodged"] = flagStat
					set.vals["CannotBeSuppressed"] = flagStat
					if level == 0 {
						set.count += 3
					} else {
						// Archive parity: the Lua adds to base.count here
						// instead of uber.count (a bug it keeps).
						set.count = base.count + 3
					}
				}
			}
		}
		for _, is := range skill.statSets.Refs("ImplicitStats") {
			if is.Str("Id") == "cannot_be_blocked_or_dodged_or_suppressed" {
				for _, set := range []*addStatsSet{base, uber} {
					set.vals["CannotBeBlocked"] = flagStat
					set.vals["CannotBeDodged"] = flagStat
					set.vals["CannotBeSuppressed"] = flagStat
					set.count += 3
				}
			}
		}
		constStats, constVals := statRowsAndVals(skill.statSets, "ConstantStats", "ConstantStatsValues")
		for i, cs := range constStats {
			var name string
			switch cs.Str("Id") {
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
				v := schema.BossStatValue{Value: float64(constVals[i])}
				base.vals[name] = v
				base.count++
				uber.vals[name] = v
				uber.count++
			}
		}
		if base.count == 0 && uber.count == 0 {
			return nil, nil
		}
		return base, uber
	}

	openBoss := func(d *bossHeadDirective) {
		bossData := monsterVarieties.RowByStr("Id", d.Monster)
		b := &bossInfo{
			displayName: d.Name,
			damageRange: bossData.Ref("Type").Int("DamageSpread"),
			damageMult:  bossData.Int("DamageMultiplier"),
			critChance:  5,
		}
		b.earlierUber = d.EarlierUber
		b.mapBoss = d.MapBoss
		for _, mod := range bossData.Refs("Mods") {
			if mod.Str("Id") == "MonsterMapBoss" {
				b.rarity = "Unique"
				break
			}
		}
		state.boss = b
	}
	addSkill := func(d *bossSkillEntry) error {
		displayName, grantedId := d.Name, d.Granted
		switch displayName {
		case "MemoryGame":
			displayName = "Memory Game"
		case "GroundDegen":
			displayName = "Ground Degen"
		}
		boss := state.boss
		state.skillList = append(state.skillList, boss.displayName+" "+displayName)
		skill := &skillInfo{grantedId: grantedId}
		skill.skillData = grantedEffects.RowByStr("Id", grantedId)
		skill.statSets = statSetsDat.RowByStr("Id", grantedId)
		skill.statsPerLevel = statSetsPerLevel.RowsByRef("GrantedEffect", skill.skillData)
		state.skill = skill
		skill.index = 1
		skill.hasIndex = true
		if d.SkillIndex.Set {
			skill.hasIndex = !d.SkillIndex.Nil
			if !d.SkillIndex.Nil {
				skill.index = d.SkillIndex.N
			}
		}
		if d.SkillIndexUber.Set {
			skill.hasUberIndex = !d.SkillIndexUber.Nil
			skill.uberIndex = d.SkillIndexUber.N
		} else if skill.hasIndex {
			skill.uberIndex = skill.index + 1
			skill.hasUberIndex = true
		}
		if d.Granted2 != "" {
			skill.grantedId2 = d.Granted2
			sd2 := grantedEffects.RowByStr("Id", d.Granted2)
			skill.statsPerLevel2 = statSetsPerLevel.RowsByRef("GrantedEffect", sd2)
		}
		if d.GrantedUber != "" {
			skill.skillDataUber = grantedEffects.RowByStr("Id", d.GrantedUber)
		}
		state.SkillExtraDamageMult = 1
		if d.ExtraDamageMult != nil {
			state.SkillExtraDamageMult = *d.ExtraDamageMult / 100
		}
		if d.Stages != nil {
			skill.stages = *d.Stages
			skill.hasStages = true
		}
		state.DamageMult = map[string]float64{}
		state.Pen = map[string]penVal{}
		state.DamageType = getDamageType()
		if err := calcSkillDamage(); err != nil {
			return err
		}
		bs := schema.BossSkill{
			Key:        boss.displayName + " " + displayName,
			DamageType: state.DamageType,
		}
		for _, dt := range bossDamageTypes {
			if mn, ok := state.DamageMult[dt+"DamageMultMin"]; ok {
				mx := state.DamageMult[dt+"DamageMultMax"]
				bs.DamageMultipliers = append(bs.DamageMultipliers, schema.BossDamageMult{
					Type: dt, Min: mn, Spread: (mx - mn) / 100,
				})
			}
		}
		if um, ok := state.DamageMult["SkillUberDamageMult"]; ok {
			v := um / 100
			bs.UberDamageMultiplier = &v
		}
		penEntries := func(keys []string, strip bool) []schema.PenEntry {
			var entries []schema.PenEntry
			for _, penType := range keys {
				v := state.Pen[penType]
				if !v.blank && v.n == 0 {
					continue
				}
				name := penType
				if strip {
					name = strings.ReplaceAll(name, "Uber", "")
				}
				e := schema.PenEntry{Name: name}
				if !v.blank {
					n := v.n
					e.Value = &n
				}
				entries = append(entries, e)
			}
			return entries
		}
		if getPenetration() {
			bs.HasPen = true
			bs.Pens = penEntries([]string{"PhysOverwhelm", "LightningPen", "ColdPen", "FirePen"}, false)
			nonZero := func(k string) bool {
				v := state.Pen[k]
				return v.blank || v.n != 0
			}
			if nonZero("PhysUberOverwhelm") || nonZero("LightningUberPen") || nonZero("ColdUberPen") || nonZero("FireUberPen") {
				bs.HasUberPen = true
				bs.UberPens = penEntries([]string{"PhysUberOverwhelm", "LightningUberPen", "ColdUberPen", "FireUberPen"}, true)
			}
		}
		if d.SpeedMult != nil {
			state.skill.speedMult = *d.SpeedMult
			state.skill.hasSpeedMult = true
		}
		speed, uberSpeed, hasUberSpeed, err := getSpeed()
		if err != nil {
			return err
		}
		bs.Speed = speed
		if hasUberSpeed {
			bs.UberSpeed = &uberSpeed
		}
		bs.CritChance = int64(math.Ceil(float64(skill.statsPerLevel[0].Int("AttackCritChance")) / 100))
		bs.EarlierUber = boss.earlierUber
		baseSet, uberSet := getAdditionalStats()
		if baseSet != nil {
			bs.HasAdditional = true
			bs.BaseCount = baseSet.count
			bs.UberCount = uberSet.count
			bs.BaseVals = baseSet.vals
			bs.UberVals = uberSet.vals
		}
		doc.Skills = append(doc.Skills, bs)
		return nil
	}

	state = &skillState{}
	skillsTpl, err := readTemplate("Enemies/", "BossSkills", bossSkillDirectives)
	if err != nil {
		return nil, err
	}
	for _, d := range skillsTpl.Directives {
		switch d := d.(type) {
		case *bossHeadDirective:
			openBoss(d)
		case *bossSkillEntry:
			if err := addSkill(d); err != nil {
				return nil, err
			}
		case *tooltipDirective:
			doc.Skills[len(doc.Skills)-1].Tooltip = d.Text
			state.skill = nil
		case *skillListDirective:
			doc.SkillLists = append(doc.SkillLists, append([]string{}, state.skillList...))
			state.boss = nil
			state.skillList = nil
		}
	}

	bossesTpl, err := readTemplate("Enemies/", "Bosses", bossMonsterDirectives)
	if err != nil {
		return nil, err
	}
	for _, d := range bossesTpl.Directives {
		b := d.(*bossMonsterEntry)
		monsterType := monsterTypes.RowByStr("Id", b.Monster)
		if monsterType == nil {
			doc.Bosses = append(doc.Bosses, nil) // the Lua prints "Invalid Type"
			continue
		}
		doc.Bosses = append(doc.Bosses, &schema.BossMonster{
			DisplayName: b.Name,
			ArmourMult:  monsterType.Int("Armour"),
			EvasionMult: monsterType.Int("Evasion"),
			IsUber:      b.Uber,
		})
	}
	return doc, nil
}
