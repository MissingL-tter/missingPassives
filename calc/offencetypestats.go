// CalcOffence.lua L1022-1456: the skill-type stats — minion limits, chain/
// projectile/pierce counts, melee range, area of effect, aura/curse/warcry/
// link effect, reservation multipliers, trap/mine/totem/brand/corpse stats.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// calcSkillCooldown ports the module-level function of the same name
// (L274). The Lua returns (cooldown, rounded, addedCooldown); only the
// first two are read outside the breakdown.
func (env *Env) calcSkillCooldown(skillModList *modstore.List, skillCfg *modstore.Cfg, skillData map[string]any) (cooldown float64, rounded bool) {
	d := env.Data
	cooldownOverride := skillModList.Override(skillCfg, "CooldownRecovery")
	addedCooldown := skillModList.Sum("BASE", skillCfg, "CooldownRecovery")
	if truthy(cooldownOverride) {
		cooldown = anyNum(cooldownOverride)
	} else {
		cooldown = (anyNum(skillData["cooldown"]) + addedCooldown) / math.Max(0, Mod(skillModList, skillCfg, "CooldownRecovery"))
	}
	// If a skill can store extra uses and has a cooldown, it doesn't round
	// the cooldown value to server ticks
	if anyNum(skillData["storedUses"]) > 1 || anyNum(skillData["VaalStoredUses"]) > 1 ||
		skillModList.Sum("BASE", skillCfg, "AdditionalCooldownUses") > 0 {
		return cooldown, false
	}
	cooldown = math.Ceil(cooldown*d.Misc.ServerTickRate) / d.Misc.ServerTickRate
	return cooldown, true
}

// calcWarcryCastTime ports the local of the same name (L289).
func (env *Env) calcWarcryCastTime(skillModList *modstore.List, skillCfg *modstore.Cfg, skillData map[string]any, actor *performActor) float64 {
	d := env.Data
	baseSpeed := 1 / skillModList.Sum("BASE", skillCfg, "WarcryCastTime")
	warcryCastTime := baseSpeed * Mod(skillModList, skillCfg, "WarcrySpeed") * env.actionSpeedMod(actor)
	warcryCastTime = math.Min(warcryCastTime, d.Misc.ServerTickRate)
	warcryCastTime = 1 / warcryCastTime
	if skillModList.Flag(skillCfg, "InstantWarcry") || truthy(skillData["triggeredByAutoexertion"]) {
		warcryCastTime = 0
	}
	return warcryCastTime
}

// offenceSkillTypeStats ports L1022-1456.
func (env *Env) offenceSkillTypeStats(c *offenceCtx) {
	actor, skillModList, skillCfg, skillData := c.actor, c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, enemyDB := c.skillFlags, c.output, c.enemyDB
	activeSkill := c.activeSkill
	d := env.Data

	// Calculate skill type stats
	if activeSkill.Minion != nil {
		if limit := activeSkill.Minion.MinionData.Limit; limit != "" {
			if ov := env.ModDB.Override(nil, limit); truthy(ov) {
				output["ActiveMinionLimit"] = math.Floor(anyNum(ov))
			} else {
				output["ActiveMinionLimit"] = math.Floor(Val(skillModList, limit, skillCfg) * skillModList.More(skillCfg, "ActiveMinionLimit"))
			}
		}
		output["SummonedMinionsPerCast"] = math.Floor(Val(skillModList, "MinionPerCastCount", skillCfg))
		if outNum(output, "SummonedMinionsPerCast") == 0 {
			output["SummonedMinionsPerCast"] = 1.0
		}
	}
	if skillFlags["chaining"] {
		if skillModList.Flag(skillCfg, "CannotChain") || skillModList.Flag(skillCfg, "NoAdditionalChains") {
			output["ChainMaxString"] = "Cannot chain"
		} else {
			names := []string{"ChainCountMax"}
			if !skillFlags["projectile"] {
				names = append(names, "BeamChainCountMax")
			}
			chainMax := skillModList.Sum("BASE", skillCfg, names...)
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddChainsInstead") {
				projCount := 0.0
				if !skillModList.Flag(skillCfg, "SingleProjectile") {
					projCount = math.Floor((skillModList.Sum("BASE", skillCfg, "ProjectileCount") - 1) * skillModList.More(skillCfg, "ProjectileCount"))
				}
				chainMax += projCount
			}
			chainMax *= skillModList.More(skillCfg, "ChainCountMax")
			output["ChainMax"] = chainMax
			output["ChainMaxString"] = chainMax
			output["Chain"] = math.Min(chainMax, skillModList.Sum("BASE", skillCfg, "ChainCount"))
			output["ChainRemaining"] = math.Max(0, chainMax-outNum(output, "Chain"))
		}
	}
	if skillFlags["projectile"] {
		if skillModList.Flag(nil, "PointBlank") {
			skillModList.AddMod(newMod("Damage", "MORE", 30.0, "Point Blank",
				modparser.ModFlag.Attack|modparser.ModFlag.Projectile,
				modparser.Tag{"type": "DistanceRamp", "ramp": rampTable([][2]float64{{10, 1}, {35, 0}, {120, -1}})}))
		}
		if skillModList.Flag(nil, "FarShot") {
			skillModList.AddMod(newMod("Damage", "MORE", 100.0, "Far Shot",
				modparser.ModFlag.Attack|modparser.ModFlag.Projectile,
				modparser.Tag{"type": "DistanceRamp", "ramp": rampTable([][2]float64{{10, -0.2}, {25, 0}, {70, 0.6}})}))
		}
		if skillModList.Flag(skillCfg, "NoAdditionalProjectiles") || skillModList.Flag(skillCfg, "SingleProjectile") {
			output["ProjectileCount"] = 1.0
		} else {
			projMin := skillModList.Sum("BASE", skillCfg, "ProjectileCountMinimum")
			projMax := math.Inf(1)
			if ov := skillModList.Override(skillCfg, "ProjectileCountMaximum"); truthy(ov) {
				projMax = anyNum(ov)
			}
			projBase := skillModList.Sum("BASE", skillCfg, "ProjectileCount")
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			proj := math.Floor(projBase * projMore)
			output["ProjectileCount"] = math.Max(math.Min(proj, projMax), projMin)
		}
		if skillModList.Flag(skillCfg, "AdditionalProjectilesAddBouncesInstead") {
			projBase := 0.0
			if !skillModList.Flag(skillCfg, "SingleProjectile") {
				projBase = skillModList.Sum("BASE", skillCfg, "ProjectileCount") + skillModList.Sum("BASE", skillCfg, "BounceCount") - 1
			}
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			output["BounceCount"] = math.Floor(projBase * projMore)
		}
		if skillModList.Flag(skillCfg, "CannotSplit") || activeSkill.SkillTypes[modparser.SkillType.ProjectileNumber] {
			// breakdown-only in the reference
		} else {
			splitCount := skillModList.Sum("BASE", skillCfg, "SplitCount") + enemyDB.Sum("BASE", skillCfg, "SelfSplitCount")
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddSplitsInstead") {
				addedSplits := 0.0
				if !skillModList.Flag(skillCfg, "SingleProjectile") {
					addedSplits = math.Floor((skillModList.Sum("BASE", skillCfg, "ProjectileCount") - 1) * skillModList.More(skillCfg, "ProjectileCount"))
				}
				splitCount += addedSplits
			}
			if skillModList.Flag(skillCfg, "AdditionalChainsAddSplitsInstead") {
				splitCount += skillModList.Sum("BASE", skillCfg, "ChainCountMax")
			}
			output["SplitCount"] = splitCount
			output["SplitCountString"] = splitCount
		}
		switch {
		case skillModList.Flag(skillCfg, "CannotFork"):
			output["ForkCountString"] = "Cannot fork"
		case skillModList.Flag(skillCfg, "ForkOnce"):
			skillFlags["forking"] = true
			cap := 1.0
			if skillModList.Flag(skillCfg, "ForkTwice") {
				cap = 2
			}
			forkMax := math.Min(skillModList.Sum("BASE", skillCfg, "ForkCountMax"), cap)
			output["ForkCountMax"] = forkMax
			output["ForkedCount"] = math.Min(forkMax, skillModList.Sum("BASE", skillCfg, "ForkedCount"))
			output["ForkCountString"] = forkMax
			output["ForkRemaining"] = math.Max(0, forkMax-outNum(output, "ForkedCount"))
		default:
			output["ForkCountString"] = "0"
		}
		if skillModList.Flag(skillCfg, "CannotPierce") {
			output["PierceCount"] = 0.0
			output["PierceCountString"] = "Cannot pierce"
		} else {
			if skillModList.Flag(skillCfg, "PierceAllTargets") || enemyDB.Flag(nil, "AlwaysPierceSelf") {
				output["PierceCount"] = 100.0
				output["PierceCountString"] = "All targets"
			} else {
				pc := skillModList.Sum("BASE", skillCfg, "PierceCount")
				output["PierceCount"] = pc
				output["PierceCountString"] = pc
			}
			if outNum(output, "PierceCount") > 0 {
				skillFlags["piercing"] = true
			}
			output["PiercedCount"] = math.Min(outNum(output, "PierceCount"), skillModList.Sum("BASE", skillCfg, "PiercedCount"))
		}
		output["ProjectileSpeedMod"] = Mod(skillModList, skillCfg, "ProjectileSpeed")
	}
	if skillFlags["melee"] {
		if skillFlags["weapon1Attack"] {
			if truthy(actor.ms.WeaponData1["range"]) {
				actor.weaponRange1 = anyNum(actor.ms.WeaponData1["range"]) +
					skillModList.Sum("BASE", activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum("BASE", activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre")
			} else {
				actor.weaponRange1 = 6 + skillModList.Sum("BASE", skillCfg, "UnarmedRange") +
					10*skillModList.Sum("BASE", skillCfg, "UnarmedRangeMetre")
			}
		}
		if skillFlags["weapon2Attack"] {
			if truthy(actor.ms.WeaponData2["range"]) {
				actor.weaponRange2 = anyNum(actor.ms.WeaponData2["range"]) +
					skillModList.Sum("BASE", activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum("BASE", activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre")
			} else {
				actor.weaponRange2 = 6 + skillModList.Sum("BASE", skillCfg, "UnarmedRange") +
					10*skillModList.Sum("BASE", skillCfg, "UnarmedRangeMetre")
			}
		}
		if activeSkill.SkillTypes[modparser.SkillType.MeleeSingleTarget] {
			rng := 100.0
			if skillFlags["weapon1Attack"] {
				rng = math.Min(rng, actor.weaponRange1)
			}
			if skillFlags["weapon2Attack"] {
				rng = math.Min(rng, actor.weaponRange2)
			}
			output["WeaponRange"] = rng + 2
			output["WeaponRangeMetre"] = outNum(output, "WeaponRange") / 10

			baseStrikeCount := 1.0
			output["StrikeTargets"] = baseStrikeCount + skillModList.Sum("BASE", skillCfg, "AdditionalStrikeTarget")
		}
	}
	if skillFlags["area"] || truthy(skillData["radius"]) || (skillFlags["mine"] && activeSkill.SkillTypes[modparser.SkillType.Aura]) {
		env.calcAreaOfEffect(c)
	}
	if activeSkill.SkillTypes[modparser.SkillType.Aura] {
		names := []string{"AuraEffect"}
		if !(truthy(skillData["auraCannotAffectSelf"]) || activeSkill.SkillTypes[modparser.SkillType.AuraAffectsEnemies]) {
			names = append(names, "SkillAuraEffectOnSelf")
		}
		output["AuraEffectMod"] = Mod(skillModList, skillCfg, names...)
	}
	if activeSkill.SkillTypes[modparser.SkillType.HasReservation] && !activeSkill.SkillTypes[modparser.SkillType.ReservationBecomesCost] {
		for _, pool := range []string{"Life", "Mana"} {
			output[pool+"ReservedMod"] = 0.0
			if Mod(skillModList, skillCfg, "SupportManaMultiplier") > 0 && Mod(skillModList, skillCfg, pool+"Reserved", "Reserved") > 0 {
				output[pool+"ReservedMod"] = Mod(skillModList, skillCfg, pool+"Reserved", "Reserved") *
					floorDec(Mod(skillModList, skillCfg, "SupportManaMultiplier"), 4) /
					math.Max(0, Mod(skillModList, skillCfg, pool+"ReservationEfficiency", "ReservationEfficiency"))
			}
		}
	}
	if activeSkill.SkillTypes[modparser.SkillType.Hex] || activeSkill.SkillTypes[modparser.SkillType.Mark] {
		output["CurseEffectMod"] = Mod(skillModList, skillCfg, "CurseEffect")
	}
	if activeSkill.SkillTypes[modparser.SkillType.Warcry] {
		fullDuration := env.calcSkillDuration(skillModList, skillCfg, activeSkill.SkillData, enemyDB)
		cooldownOverride := skillModList.Override(skillCfg, "CooldownRecovery")
		var actualCooldown float64
		if truthy(cooldownOverride) {
			actualCooldown = anyNum(cooldownOverride)
		} else {
			actualCooldown = (anyNum(activeSkill.SkillData["cooldown"]) + skillModList.Sum("BASE", skillCfg, "CooldownRecovery")) /
				Mod(skillModList, skillCfg, "CooldownRecovery")
		}
		uptime := math.Min(fullDuration/actualCooldown, 1)
		if env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
			uptime = 1
		}
		unscaledEffect := Mod(skillModList, skillCfg, "WarcryEffect", "BuffEffect")
		output["WarcryEffectMod"] = unscaledEffect * uptime
	}
	if activeSkill.SkillTypes[modparser.SkillType.Link] {
		output["LinkEffectMod"] = Mod(skillModList, skillCfg, "LinkEffect", "BuffEffect")
	}
	if activeSkill.SkillTypes[modparser.SkillType.Buff] && activeSkill.SkillTypes[modparser.SkillType.Herald] {
		output["HeraldBuffEffectMod"] = Mod(skillModList, skillCfg, "BuffEffect", "BuffEffectOnSelf")
	}
	if (skillFlags["trap"] || skillFlags["mine"]) && !(truthy(skillData["trapCooldown"]) || truthy(skillData["cooldown"])) {
		skillFlags["notAverage"] = true
		skillFlags["showAverage"] = false
		skillData["showAverage"] = false
	}
	if skillFlags["trap"] {
		baseSpeed := 1 / skillModList.Sum("BASE", skillCfg, "TrapThrowingTime")
		timeMod := Mod(skillModList, skillCfg, "SkillTrapThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		output["TrapThrowingSpeed"] = baseSpeed * Mod(skillModList, skillCfg, "TrapThrowingSpeed") * outNum(output, "ActionSpeedMod")
		trapThrowCount := Val(skillModList, "TrapThrowCount", skillCfg)
		if truthy(skillData["trapCooldown"]) || truthy(skillData["cooldown"]) {
			trapThrowCount = 1
		}
		if ov := env.ModDB.Override(nil, "TrapThrowCount"); truthy(ov) {
			output["TrapThrowCount"] = anyNum(ov)
		} else {
			output["TrapThrowCount"] = trapThrowCount
		}
		output["TrapThrowingSpeed"] = math.Min(outNum(output, "TrapThrowingSpeed"), d.Misc.ServerTickRate)
		output["TrapThrowingTime"] = 1 / outNum(output, "TrapThrowingSpeed")
		skillData["timeOverride"] = outNum(output, "TrapThrowingTime") / outNum(output, "TrapThrowCount")

		baseCooldown, hasBaseCooldown := 0.0, false
		if truthy(skillData["trapCooldown"]) {
			baseCooldown, hasBaseCooldown = anyNum(skillData["trapCooldown"]), true
		} else if truthy(skillData["cooldown"]) {
			baseCooldown, hasBaseCooldown = anyNum(skillData["cooldown"]), true
		}
		if hasBaseCooldown || skillModList.Sum("BASE", skillCfg, "CooldownRecovery") != 0 {
			if hasBaseCooldown {
				tc := baseCooldown / Mod(skillModList, skillCfg, "CooldownRecovery")
				output["TrapCooldown"] = math.Ceil(tc*d.Misc.ServerTickRate) / d.Misc.ServerTickRate
			} else {
				// Assign Trap Cooldown if the trap/skill does not have
				// cooldown but gains cooldown elsewhere
				cooldown, _ := env.calcSkillCooldown(skillModList, skillCfg, skillData)
				output["TrapCooldown"] = cooldown
			}
		}
		incArea, moreArea := Mods(skillModList, skillCfg, "TrapTriggerAreaOfEffect")
		areaMod := roundDec(roundDec(incArea*moreArea, 10), 2)
		output["TrapTriggerRadius"] = calcRadius(d.Misc.TrapTriggerRadiusBase, areaMod)
		output["TrapTriggerRadiusMetre"] = outNum(output, "TrapTriggerRadius") / 10
	} else if truthy(skillData["cooldown"]) || skillModList.Sum("BASE", skillCfg, "CooldownRecovery") != 0 {
		cooldown, _ := env.calcSkillCooldown(skillModList, skillCfg, skillData)
		output["Cooldown"] = cooldown
	}
	if truthy(skillData["storedUses"]) {
		baseUses := anyNum(skillData["storedUses"])
		additionalUses := skillModList.Sum("BASE", skillCfg, "AdditionalCooldownUses", "AdditionalUses")
		output["StoredUses"] = baseUses + additionalUses
	}
	if skillFlags["mine"] {
		baseSpeed := 1 / skillModList.Sum("BASE", skillCfg, "MineLayingTime")
		timeMod := Mod(skillModList, skillCfg, "SkillMineThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		output["MineLayingSpeed"] = baseSpeed * Mod(skillModList, skillCfg, "MineLayingSpeed") * outNum(output, "ActionSpeedMod")
		// Calculate additional mine throw
		mineThrowCount := Val(skillModList, "MineThrowCount", skillCfg)
		if truthy(skillData["trapCooldown"]) || truthy(skillData["cooldown"]) {
			mineThrowCount = 1
		}
		if ov := env.ModDB.Override(nil, "MineThrowCount"); truthy(ov) {
			output["MineThrowCount"] = anyNum(ov)
		} else {
			output["MineThrowCount"] = mineThrowCount
		}
		if outNum(output, "MineThrowCount") >= 1 {
			// Throwing Mines takes 10% more time for each *additional* Mine thrown
			output["MineLayingSpeed"] = outNum(output, "MineLayingSpeed") / (1 + (outNum(output, "MineThrowCount")-1)*0.1)
		}

		output["MineLayingSpeed"] = math.Min(outNum(output, "MineLayingSpeed"), d.Misc.ServerTickRate)
		output["MineLayingTime"] = 1 / outNum(output, "MineLayingSpeed")

		// Trap mine interaction where the Character throws mines, mine throws traps
		if skillFlags["trap"] {
			skillData["timeOverride"] = outNum(output, "MineLayingTime") / outNum(output, "MineThrowCount") / outNum(output, "TrapThrowCount")
		} else {
			skillData["timeOverride"] = outNum(output, "MineLayingTime") / outNum(output, "MineThrowCount")
		}

		incArea, moreArea := Mods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
		areaMod := roundDec(roundDec(incArea*moreArea, 10), 2)
		output["MineDetonationRadius"] = calcRadius(d.Misc.MineDetonationRadiusBase, areaMod)
		output["MineDetonationRadiusMetre"] = outNum(output, "MineDetonationRadius") / 10
		if activeSkill.SkillTypes[modparser.SkillType.Aura] {
			output["MineAuraRadius"] = calcRadius(d.Misc.MineAuraRadiusBase, outNum(output, "AreaOfEffectMod"))
			output["MineAuraRadiusMetre"] = outNum(output, "MineAuraRadius") / 10
		}
	}
	if skillFlags["totem"] {
		var baseSpeed float64
		if skillFlags["ballista"] {
			baseSpeed = 1 / skillModList.Sum("BASE", skillCfg, "BallistaPlacementTime")
		} else {
			baseSpeed = 1 / skillModList.Sum("BASE", skillCfg, "TotemPlacementTime")
		}
		output["TotemPlacementSpeed"] = baseSpeed * Mod(skillModList, skillCfg, "TotemPlacementSpeed") * outNum(output, "ActionSpeedMod")
		output["TotemPlacementTime"] = 1 / outNum(output, "TotemPlacementSpeed")
		output["ActiveTotemLimit"] = skillModList.Sum("BASE", skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
		if ov := env.ModDB.Override(nil, "TotemsSummoned"); truthy(ov) {
			output["TotemsSummoned"] = anyNum(ov)
		} else {
			output["TotemsSummoned"] = outNum(output, "ActiveTotemLimit")
		}
		life, lifeMod := env.calcTotemLife(activeSkill)
		output["TotemLife"], output["TotemLifeMod"] = life, lifeMod
		output["TotemEnergyShield"] = skillModList.Sum("BASE", skillCfg, "TotemEnergyShield")
		output["TotemBlockChance"] = skillModList.Sum("BASE", skillCfg, "TotemBlockChance")
		output["TotemArmour"] = skillModList.Sum("BASE", skillCfg, "TotemArmour")
	}
	if activeSkill.SkillTypes[modparser.SkillType.Brand] {
		output["BrandAttachmentRange"] = d.Misc.BrandAttachmentRangeBase * Mod(skillModList, skillCfg, "BrandAttachmentRange")
		output["BrandAttachmentRangeMetre"] = outNum(output, "BrandAttachmentRange") / 10
		output["ActiveBrandLimit"] = skillModList.Sum("BASE", skillCfg, "ActiveBrandLimit")
		if v, ok := skillData["attachedBrandCount"]; ok && v != nil {
			output["AttachedBrandCount"] = v
		} else {
			delete(output, "AttachedBrandCount")
		}
	}

	if skillFlags["warcry"] {
		output["WarcryCastTime"] = env.calcWarcryCastTime(skillModList, skillCfg, skillData, actor)
	}

	if skillFlags["corpse"] {
		output["CorpseLevel"] = skillModList.Sum("BASE", skillCfg, "CorpseLevel")
		// `output.CorpseLevel or 1` never falls through: CorpseLevel is the
		// Sum just assigned, and 0 is truthy in Lua.
		lvl := int(outNum(output, "CorpseLevel"))
		varietyMult := 1.0
		if v, ok := d.MonsterVarietyLifeMult[str(skillData["corpseMonsterVariety"])]; ok {
			varietyMult = v
		}
		mapMult := 1.0
		if v, ok := d.MapLevelLifeMult[int64(env.EnemyLevel)]; ok {
			mapMult = v
		}
		output["BaseCorpseLife"] = luaIndex("monsterLifeTable", d.MonsterLifeTable, lvl) * varietyMult * mapMult
		output["CorpseLifeInc"] = 1 + skillModList.Sum("INC", skillCfg, "CorpseLife")/100
		output["CorpseLife"] = outNum(output, "BaseCorpseLife") * outNum(output, "CorpseLifeInc")
	}

	env.offenceDuration(c)
}

// rampTable renders a Lua {{x,y},...} ramp as the nested list shape the
// modifier evaluator reads.
func rampTable(pairs [][2]float64) []any {
	out := make([]any, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, []any{p[0], p[1]})
	}
	return out
}

// luaIndex reads a 1-based Lua array slot. Out of range is nil in Lua, and
// every caller here multiplies straight away, so the reference errors too —
// panic rather than invent a value.
func luaIndex(name string, t []float64, i int) float64 {
	if i < 1 || i > len(t) {
		panic("offence: " + name + " index out of range (the Lua errors too)")
	}
	return t[i-1]
}
