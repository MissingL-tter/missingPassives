// CalcOffence.lua L1022-1456: the skill-type stats — minion limits, chain/
// projectile/pierce counts, melee range, area of effect, aura/curse/warcry/
// link effect, reservation multipliers, trap/mine/totem/brand/corpse stats.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strconv"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// calcSkillCooldown ports the module-level function of the same name
// (L274). The Lua returns (cooldown, rounded, addedCooldown); only the
// first two are read outside the breakdown.
func (env *Env) calcSkillCooldown(skillModList *modstore.List, skillCfg *modstore.Cfg, skillData *SkillData) (cooldown float64, rounded bool) {
	cooldownOverride, _ := skillModList.Override(skillCfg, "CooldownRecovery")
	addedCooldown := skillModList.Sum(modparser.Base, skillCfg, "CooldownRecovery")
	if modparser.Truthy(cooldownOverride) {
		cooldown = valueNum(cooldownOverride)
	} else {
		cooldown = (skillData.N("cooldown") + addedCooldown) / math.Max(0, Mod(skillModList, skillCfg, "CooldownRecovery"))
	}
	// If a skill can store extra uses and has a cooldown, it doesn't round
	// the cooldown value to server ticks
	if skillData.N("storedUses") > 1 || skillData.N("VaalStoredUses") > 1 ||
		skillModList.Sum(modparser.Base, skillCfg, "AdditionalCooldownUses") > 0 {
		return cooldown, false
	}
	cooldown = math.Ceil(cooldown*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	return cooldown, true
}

// calcWarcryCastTime ports the local of the same name (L289).
func (env *Env) calcWarcryCastTime(skillModList *modstore.List, skillCfg *modstore.Cfg, skillData *SkillData, actor *performActor) float64 {
	baseSpeed := 1 / skillModList.Sum(modparser.Base, skillCfg, "WarcryCastTime")
	warcryCastTime := baseSpeed * Mod(skillModList, skillCfg, "WarcrySpeed") * env.actionSpeedMod(actor)
	warcryCastTime = math.Min(warcryCastTime, data.Misc.ServerTickRate)
	warcryCastTime = 1 / warcryCastTime
	if skillModList.Flag(skillCfg, "InstantWarcry") || skillData.Flag("triggeredByAutoexertion") {
		warcryCastTime = 0
	}
	return warcryCastTime
}

// offenceSkillTypeStats ports L1022-1456.
func (env *Env) offenceSkillTypeStats(c *offenceCtx) {
	actor, skillModList, skillCfg, skillData := c.actor, c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, enemyDB := c.skillFlags, c.output, c.enemyDB
	activeSkill := c.activeSkill

	// Calculate skill type stats
	if activeSkill.Minion != nil {
		if limit := activeSkill.Minion.MinionData.Limit; limit != "" {
			if ov, ok := env.ModDB.Override(nil, limit); ok {
				output.SetN("ActiveMinionLimit", math.Floor(valueNum(ov)))
			} else {
				output.SetN("ActiveMinionLimit", math.Floor(Val(skillModList, limit, skillCfg)*skillModList.More(skillCfg, "ActiveMinionLimit")))
			}
		}
		output.SetN("SummonedMinionsPerCast", math.Floor(Val(skillModList, "MinionPerCastCount", skillCfg)))
		if output.N("SummonedMinionsPerCast") == 0 {
			output.SetN("SummonedMinionsPerCast", 1.0)
		}
	}
	if skillFlags["chaining"] {
		if skillModList.Flag(skillCfg, "CannotChain") || skillModList.Flag(skillCfg, "NoAdditionalChains") {
			output.SetStr("ChainMaxString", "Cannot chain")
		} else {
			names := []string{"ChainCountMax"}
			if !skillFlags["projectile"] {
				names = append(names, "BeamChainCountMax")
			}
			chainMax := skillModList.Sum(modparser.Base, skillCfg, names...)
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddChainsInstead") {
				projCount := 0.0
				if !skillModList.Flag(skillCfg, "SingleProjectile") {
					projCount = math.Floor((skillModList.Sum(modparser.Base, skillCfg, "ProjectileCount") - 1) * skillModList.More(skillCfg, "ProjectileCount"))
				}
				chainMax += projCount
			}
			chainMax *= skillModList.More(skillCfg, "ChainCountMax")
			output.SetN("ChainMax", chainMax)
			output.SetN("ChainMaxString", chainMax)
			output.SetN("Chain", math.Min(chainMax, skillModList.Sum(modparser.Base, skillCfg, "ChainCount")))
			output.SetN("ChainRemaining", math.Max(0, chainMax-output.N("Chain")))
		}
	}
	if skillFlags["projectile"] {
		if skillModList.Flag(nil, "PointBlank") {
			skillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(30.0), "Point Blank", modparser.FlagAttack|modparser.FlagProjectile, modparser.KeywordNone, &modparser.DistanceRampTag{Ramp: modparser.Pairs{{10, 1}, {35, 0}, {120, -1}}}))
		}
		if skillModList.Flag(nil, "FarShot") {
			skillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(100.0), "Far Shot", modparser.FlagAttack|modparser.FlagProjectile, modparser.KeywordNone, &modparser.DistanceRampTag{Ramp: modparser.Pairs{{10, -0.2}, {25, 0}, {70, 0.6}}}))
		}
		if skillModList.Flag(skillCfg, "NoAdditionalProjectiles") || skillModList.Flag(skillCfg, "SingleProjectile") {
			output.SetN("ProjectileCount", 1.0)
		} else {
			projMin := skillModList.Sum(modparser.Base, skillCfg, "ProjectileCountMinimum")
			projMax := math.Inf(1)
			if ov, ok := skillModList.Override(skillCfg, "ProjectileCountMaximum"); ok {
				projMax = valueNum(ov)
			}
			projBase := skillModList.Sum(modparser.Base, skillCfg, "ProjectileCount")
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			proj := math.Floor(projBase * projMore)
			output.SetN("ProjectileCount", math.Max(math.Min(proj, projMax), projMin))
		}
		if skillModList.Flag(skillCfg, "AdditionalProjectilesAddBouncesInstead") {
			projBase := 0.0
			if !skillModList.Flag(skillCfg, "SingleProjectile") {
				projBase = skillModList.Sum(modparser.Base, skillCfg, "ProjectileCount") + skillModList.Sum(modparser.Base, skillCfg, "BounceCount") - 1
			}
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			output.SetN("BounceCount", math.Floor(projBase*projMore))
		}
		if skillModList.Flag(skillCfg, "CannotSplit") || activeSkill.SkillTypes[modparser.SkillTypeProjectileNumber] {
			// breakdown-only in the reference
		} else {
			splitCount := skillModList.Sum(modparser.Base, skillCfg, "SplitCount") + enemyDB.Sum(modparser.Base, skillCfg, "SelfSplitCount")
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddSplitsInstead") {
				addedSplits := 0.0
				if !skillModList.Flag(skillCfg, "SingleProjectile") {
					addedSplits = math.Floor((skillModList.Sum(modparser.Base, skillCfg, "ProjectileCount") - 1) * skillModList.More(skillCfg, "ProjectileCount"))
				}
				splitCount += addedSplits
			}
			if skillModList.Flag(skillCfg, "AdditionalChainsAddSplitsInstead") {
				splitCount += skillModList.Sum(modparser.Base, skillCfg, "ChainCountMax")
			}
			output.SetN("SplitCount", splitCount)
			output.SetN("SplitCountString", splitCount)
		}
		switch {
		case skillModList.Flag(skillCfg, "CannotFork"):
			output.SetStr("ForkCountString", "Cannot fork")
		case skillModList.Flag(skillCfg, "ForkOnce"):
			skillFlags["forking"] = true
			cap := 1.0
			if skillModList.Flag(skillCfg, "ForkTwice") {
				cap = 2
			}
			forkMax := math.Min(skillModList.Sum(modparser.Base, skillCfg, "ForkCountMax"), cap)
			output.SetN("ForkCountMax", forkMax)
			output.SetN("ForkedCount", math.Min(forkMax, skillModList.Sum(modparser.Base, skillCfg, "ForkedCount")))
			output.SetN("ForkCountString", forkMax)
			output.SetN("ForkRemaining", math.Max(0, forkMax-output.N("ForkedCount")))
		default:
			output.SetStr("ForkCountString", "0")
		}
		if skillModList.Flag(skillCfg, "CannotPierce") {
			output.SetN("PierceCount", 0.0)
			output.SetStr("PierceCountString", "Cannot pierce")
		} else {
			if skillModList.Flag(skillCfg, "PierceAllTargets") || enemyDB.Flag(nil, "AlwaysPierceSelf") {
				output.SetN("PierceCount", 100.0)
				output.SetStr("PierceCountString", "All targets")
			} else {
				pc := skillModList.Sum(modparser.Base, skillCfg, "PierceCount")
				output.SetN("PierceCount", pc)
				output.SetN("PierceCountString", pc)
			}
			if output.N("PierceCount") > 0 {
				skillFlags["piercing"] = true
			}
			output.SetN("PiercedCount", math.Min(output.N("PierceCount"), skillModList.Sum(modparser.Base, skillCfg, "PiercedCount")))
		}
		output.SetN("ProjectileSpeedMod", Mod(skillModList, skillCfg, "ProjectileSpeed"))
	}
	if skillFlags["melee"] {
		if skillFlags["weapon1Attack"] {
			if wd := weaponOf(actor.ms.WeaponData1); wd != nil && wd.Range != 0 {
				actor.weaponRange1 = wd.Range +
					skillModList.Sum(modparser.Base, activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum(modparser.Base, activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre")
			} else {
				actor.weaponRange1 = 6 + skillModList.Sum(modparser.Base, skillCfg, "UnarmedRange") +
					10*skillModList.Sum(modparser.Base, skillCfg, "UnarmedRangeMetre")
			}
		}
		if skillFlags["weapon2Attack"] {
			if wd := weaponOf(actor.ms.WeaponData2); wd != nil && wd.Range != 0 {
				actor.weaponRange2 = wd.Range +
					skillModList.Sum(modparser.Base, activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum(modparser.Base, activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre")
			} else {
				actor.weaponRange2 = 6 + skillModList.Sum(modparser.Base, skillCfg, "UnarmedRange") +
					10*skillModList.Sum(modparser.Base, skillCfg, "UnarmedRangeMetre")
			}
		}
		if activeSkill.SkillTypes[modparser.SkillTypeMeleeSingleTarget] {
			rng := 100.0
			if skillFlags["weapon1Attack"] {
				rng = math.Min(rng, actor.weaponRange1)
			}
			if skillFlags["weapon2Attack"] {
				rng = math.Min(rng, actor.weaponRange2)
			}
			output.SetN("WeaponRange", rng+2)
			output.SetN("WeaponRangeMetre", output.N("WeaponRange")/10)

			baseStrikeCount := 1.0
			output.SetN("StrikeTargets", baseStrikeCount+skillModList.Sum(modparser.Base, skillCfg, "AdditionalStrikeTarget"))
		}
	}
	if skillFlags["area"] || skillData.Has("radius") || (skillFlags["mine"] && activeSkill.SkillTypes[modparser.SkillTypeAura]) {
		env.calcAreaOfEffect(c)
	}
	if activeSkill.SkillTypes[modparser.SkillTypeAura] {
		names := []string{"AuraEffect"}
		if !(skillData.Flag("auraCannotAffectSelf") || activeSkill.SkillTypes[modparser.SkillTypeAuraAffectsEnemies]) {
			names = append(names, "SkillAuraEffectOnSelf")
		}
		output.SetN("AuraEffectMod", Mod(skillModList, skillCfg, names...))
	}
	if activeSkill.SkillTypes[modparser.SkillTypeHasReservation] && !activeSkill.SkillTypes[modparser.SkillTypeReservationBecomesCost] {
		for _, pool := range []string{"Life", "Mana"} {
			output.SetN(pool+"ReservedMod", 0.0)
			if Mod(skillModList, skillCfg, "SupportManaMultiplier") > 0 && Mod(skillModList, skillCfg, pool+"Reserved", "Reserved") > 0 {
				output.SetN(pool+"ReservedMod", Mod(skillModList, skillCfg, pool+"Reserved", "Reserved")*
					floorDec(Mod(skillModList, skillCfg, "SupportManaMultiplier"), 4)/
					math.Max(0, Mod(skillModList, skillCfg, pool+"ReservationEfficiency", "ReservationEfficiency")))
			}
		}
	}
	if activeSkill.SkillTypes[modparser.SkillTypeHex] || activeSkill.SkillTypes[modparser.SkillTypeMark] {
		output.SetN("CurseEffectMod", Mod(skillModList, skillCfg, "CurseEffect"))
	}
	if activeSkill.SkillTypes[modparser.SkillTypeWarcry] {
		fullDuration := env.calcSkillDuration(skillModList, skillCfg, activeSkill.SkillData, enemyDB)
		cooldownOverride, _ := skillModList.Override(skillCfg, "CooldownRecovery")
		var actualCooldown float64
		if modparser.Truthy(cooldownOverride) {
			actualCooldown = valueNum(cooldownOverride)
		} else {
			actualCooldown = (activeSkill.SkillData.N("cooldown") + skillModList.Sum(modparser.Base, skillCfg, "CooldownRecovery")) /
				Mod(skillModList, skillCfg, "CooldownRecovery")
		}
		uptime := math.Min(fullDuration/actualCooldown, 1)
		if env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
			uptime = 1
		}
		unscaledEffect := Mod(skillModList, skillCfg, "WarcryEffect", "BuffEffect")
		output.SetN("WarcryEffectMod", unscaledEffect*uptime)
	}
	if activeSkill.SkillTypes[modparser.SkillTypeLink] {
		output.SetN("LinkEffectMod", Mod(skillModList, skillCfg, "LinkEffect", "BuffEffect"))
	}
	if activeSkill.SkillTypes[modparser.SkillTypeBuff] && activeSkill.SkillTypes[modparser.SkillTypeHerald] {
		output.SetN("HeraldBuffEffectMod", Mod(skillModList, skillCfg, "BuffEffect", "BuffEffectOnSelf"))
	}
	if (skillFlags["trap"] || skillFlags["mine"]) && !(skillData.Flag("trapCooldown") || skillData.Has("cooldown")) {
		skillFlags["notAverage"] = true
		skillFlags["showAverage"] = false
		skillData.SetFlag("showAverage", false)
	}
	if skillFlags["trap"] {
		baseSpeed := 1 / skillModList.Sum(modparser.Base, skillCfg, "TrapThrowingTime")
		timeMod := Mod(skillModList, skillCfg, "SkillTrapThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		output.SetN("TrapThrowingSpeed", baseSpeed*Mod(skillModList, skillCfg, "TrapThrowingSpeed")*output.N("ActionSpeedMod"))
		trapThrowCount := Val(skillModList, "TrapThrowCount", skillCfg)
		if skillData.Flag("trapCooldown") || skillData.Has("cooldown") {
			trapThrowCount = 1
		}
		if ov, ok := env.ModDB.Override(nil, "TrapThrowCount"); ok {
			output.SetN("TrapThrowCount", valueNum(ov))
		} else {
			output.SetN("TrapThrowCount", trapThrowCount)
		}
		output.SetN("TrapThrowingSpeed", math.Min(output.N("TrapThrowingSpeed"), data.Misc.ServerTickRate))
		output.SetN("TrapThrowingTime", 1/output.N("TrapThrowingSpeed"))
		skillData.SetN("timeOverride", output.N("TrapThrowingTime")/output.N("TrapThrowCount"))

		baseCooldown, hasBaseCooldown := 0.0, false
		if skillData.Flag("trapCooldown") {
			baseCooldown, hasBaseCooldown = skillData.N("trapCooldown"), true
		} else if skillData.Has("cooldown") {
			baseCooldown, hasBaseCooldown = skillData.N("cooldown"), true
		}
		if hasBaseCooldown || skillModList.Sum(modparser.Base, skillCfg, "CooldownRecovery") != 0 {
			if hasBaseCooldown {
				tc := baseCooldown / Mod(skillModList, skillCfg, "CooldownRecovery")
				output.SetN("TrapCooldown", math.Ceil(tc*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
			} else {
				// Assign Trap Cooldown if the trap/skill does not have
				// cooldown but gains cooldown elsewhere
				cooldown, _ := env.calcSkillCooldown(skillModList, skillCfg, skillData)
				output.SetN("TrapCooldown", cooldown)
			}
		}
		incArea, moreArea := Mods(skillModList, skillCfg, "TrapTriggerAreaOfEffect")
		areaMod := util.RoundHalfUp(util.RoundHalfUp(incArea*moreArea, 10), 2)
		output.SetN("TrapTriggerRadius", calcRadius(data.Misc.TrapTriggerRadiusBase, areaMod))
		output.SetN("TrapTriggerRadiusMetre", output.N("TrapTriggerRadius")/10)
	} else if skillData.Has("cooldown") || skillModList.Sum(modparser.Base, skillCfg, "CooldownRecovery") != 0 {
		cooldown, _ := env.calcSkillCooldown(skillModList, skillCfg, skillData)
		output.SetN("Cooldown", cooldown)
	}
	if skillData.Has("storedUses") {
		baseUses := skillData.N("storedUses")
		additionalUses := skillModList.Sum(modparser.Base, skillCfg, "AdditionalCooldownUses", "AdditionalUses")
		output.SetN("StoredUses", baseUses+additionalUses)
	}
	if skillFlags["mine"] {
		baseSpeed := 1 / skillModList.Sum(modparser.Base, skillCfg, "MineLayingTime")
		timeMod := Mod(skillModList, skillCfg, "SkillMineThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		output.SetN("MineLayingSpeed", baseSpeed*Mod(skillModList, skillCfg, "MineLayingSpeed")*output.N("ActionSpeedMod"))
		// Calculate additional mine throw
		mineThrowCount := Val(skillModList, "MineThrowCount", skillCfg)
		if skillData.Flag("trapCooldown") || skillData.Has("cooldown") {
			mineThrowCount = 1
		}
		if ov, ok := env.ModDB.Override(nil, "MineThrowCount"); ok {
			output.SetN("MineThrowCount", valueNum(ov))
		} else {
			output.SetN("MineThrowCount", mineThrowCount)
		}
		if output.N("MineThrowCount") >= 1 {
			// Throwing Mines takes 10% more time for each *additional* Mine thrown
			output.SetN("MineLayingSpeed", output.N("MineLayingSpeed")/(1+(output.N("MineThrowCount")-1)*0.1))
		}

		output.SetN("MineLayingSpeed", math.Min(output.N("MineLayingSpeed"), data.Misc.ServerTickRate))
		output.SetN("MineLayingTime", 1/output.N("MineLayingSpeed"))

		// Trap mine interaction where the Character throws mines, mine throws traps
		if skillFlags["trap"] {
			skillData.SetN("timeOverride", output.N("MineLayingTime")/output.N("MineThrowCount")/output.N("TrapThrowCount"))
		} else {
			skillData.SetN("timeOverride", output.N("MineLayingTime")/output.N("MineThrowCount"))
		}

		incArea, moreArea := Mods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
		areaMod := util.RoundHalfUp(util.RoundHalfUp(incArea*moreArea, 10), 2)
		output.SetN("MineDetonationRadius", calcRadius(data.Misc.MineDetonationRadiusBase, areaMod))
		output.SetN("MineDetonationRadiusMetre", output.N("MineDetonationRadius")/10)
		if activeSkill.SkillTypes[modparser.SkillTypeAura] {
			output.SetN("MineAuraRadius", calcRadius(data.Misc.MineAuraRadiusBase, output.N("AreaOfEffectMod")))
			output.SetN("MineAuraRadiusMetre", output.N("MineAuraRadius")/10)
		}
	}
	if skillFlags["totem"] {
		var baseSpeed float64
		if skillFlags["ballista"] {
			baseSpeed = 1 / skillModList.Sum(modparser.Base, skillCfg, "BallistaPlacementTime")
		} else {
			baseSpeed = 1 / skillModList.Sum(modparser.Base, skillCfg, "TotemPlacementTime")
		}
		output.SetN("TotemPlacementSpeed", baseSpeed*Mod(skillModList, skillCfg, "TotemPlacementSpeed")*output.N("ActionSpeedMod"))
		output.SetN("TotemPlacementTime", 1/output.N("TotemPlacementSpeed"))
		output.SetN("ActiveTotemLimit", skillModList.Sum(modparser.Base, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit"))
		if ov, ok := env.ModDB.Override(nil, "TotemsSummoned"); ok {
			output.SetN("TotemsSummoned", valueNum(ov))
		} else {
			output.SetN("TotemsSummoned", output.N("ActiveTotemLimit"))
		}
		life, lifeMod := env.calcTotemLife(activeSkill)
		output.SetN("TotemLife", life)
		output.SetN("TotemLifeMod", lifeMod)
		output.SetN("TotemEnergyShield", skillModList.Sum(modparser.Base, skillCfg, "TotemEnergyShield"))
		output.SetN("TotemBlockChance", skillModList.Sum(modparser.Base, skillCfg, "TotemBlockChance"))
		output.SetN("TotemArmour", skillModList.Sum(modparser.Base, skillCfg, "TotemArmour"))
	}
	if activeSkill.SkillTypes[modparser.SkillTypeBrand] {
		output.SetN("BrandAttachmentRange", data.Misc.BrandAttachmentRangeBase*Mod(skillModList, skillCfg, "BrandAttachmentRange"))
		output.SetN("BrandAttachmentRangeMetre", output.N("BrandAttachmentRange")/10)
		output.SetN("ActiveBrandLimit", skillModList.Sum(modparser.Base, skillCfg, "ActiveBrandLimit"))
		output.Set("AttachedBrandCount", skillData.Get("attachedBrandCount"))
	}

	if skillFlags["warcry"] {
		output.SetN("WarcryCastTime", env.calcWarcryCastTime(skillModList, skillCfg, skillData, actor))
	}

	if skillFlags["corpse"] {
		output.SetN("CorpseLevel", skillModList.Sum(modparser.Base, skillCfg, "CorpseLevel"))
		// `output.CorpseLevel or 1` never falls through: CorpseLevel is the
		// Sum just assigned, and 0 is truthy in Lua.
		lvl := int(output.N("CorpseLevel"))
		varietyMult := 1.0
		if v, ok := data.MonsterVarietyLifeMult[skillData.Str("corpseMonsterVariety")]; ok {
			varietyMult = v
		}
		mapMult := 1.0
		if v, ok := data.MapLevelLifeMult[int64(env.EnemyLevel)]; ok {
			mapMult = v
		}
		output.SetN("BaseCorpseLife", monsterLifeAtLevel(lvl)*varietyMult*mapMult)
		output.SetN("CorpseLifeInc", 1+skillModList.Sum(modparser.Inc, skillCfg, "CorpseLife")/100)
		output.SetN("CorpseLife", output.N("BaseCorpseLife")*output.N("CorpseLifeInc"))
	}

	env.offenceDuration(c)
}

// monsterLifeAtLevel reads the per-level monster life table (entry 1 =
// level 1). A level off the table panics — the reference errors there too
// (nil arithmetic).
func monsterLifeAtLevel(level int) float64 {
	t := data.MonsterLifeTable
	if level < 1 || level > len(t) {
		panic("offence: monsterLifeTable has no level " + strconv.Itoa(level) + " (the reference errors too)")
	}
	return t[level-1]
}
