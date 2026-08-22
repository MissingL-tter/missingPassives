// Ports of the hand-written skill callbacks that Data/Skills/*.lua attaches
// to granted effects (initialFunc, preSkillTypeFunc, preDamageFunc,
// postCritFunc, preDotFunc). The generated data tables carry an UnportedFn
// marker for each one; runSkillFunc consults this registry first and panics
// on anything still unported, so a corpus build can never silently skip a
// callback.
package calc

import (
	"math"
	"strconv"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// skillFunc is one ported callback. It gets the same reach the Lua closure
// has: the active skill, the pass-independent output, and the environment.
type skillFunc func(env *Env, c *offenceCtx)

// skillFuncs is keyed "<grantedEffectId>:<callbackName>".
var skillFuncs = map[string]skillFunc{
	"Cyclone:initialFunc":                 cycloneInitialFunc("Skill:Cyclone"),
	"CycloneAltX:initialFunc":             cycloneInitialFunc("Skill:CycloneAltX"),
	"VaalCyclone:initialFunc":             cycloneInitialFunc("Skill:Cyclone"),
	"BloodSacramentUnique:initialFunc":    bloodSacramentInitialFunc,
	"EnemyExplode:preDamageFunc":          enemyExplodePreDamageFunc,
	"StormBrand:preDamageFunc":            brandHitTimeOverride,
	"PenanceBrandAltX:preDamageFunc":      brandHitTimeOverride,
	"HeraldOfTheBreach:preDamageFunc":     heraldOfTheBreachPreDamageFunc,
	"RighteousFire:preDamageFunc":         righteousFirePreDamageFunc,
	"BlazingSalvo:preDamageFunc":          blazingSalvoPreDamageFunc,
	"ShrapnelBallista:preDamageFunc":      shrapnelBallistaPreDamageFunc,
	"ExplosiveTrap:preDamageFunc":         explosiveTrapPreDamageFunc,
	"IceSpearAltX:preDamageFunc":          iceSpearAltXPreDamageFunc,
	"BladeBlast:preDamageFunc":            bladeBlastPreDamageFunc,
	"TornadoShot:preDamageFunc":           tornadoShotPreDamageFunc,
	"ToxicRain:preDamageFunc":             toxicRainPreDamageFunc,
	"Earthquake:preDamageFunc":            earthquakePreDamageFunc,
	"MoltenStrike:preDamageFunc":          moltenStrikePreDamageFunc(false),
	"MoltenStrikeAltX:preDamageFunc":      moltenStrikePreDamageFunc(true),
	"LightningTendrilsAltX:preDamageFunc": lightningTendrilsAltXPreDamageFunc,
	"LightningTendrilsAltX:postCritFunc":  lightningTendrilsAltXPostCritFunc,
}

// cycloneInitialFunc ports the Cyclone family's initialFunc: the melee range
// the skill's area scales with. The three copies differ only in mod source.
func cycloneInitialFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		skillModList := activeSkill.SkillModList
		actor := c.actor
		rng := 0.0
		if activeSkill.SkillFlags["weapon1Attack"] && truthy(actor.ms.WeaponData1["range"]) {
			weapon1RangeBonus := skillModList.Sum("BASE", activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
				10*skillModList.Sum("BASE", activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre") +
				anyNum(actor.ms.WeaponData1["rangeBonus"])
			if activeSkill.SkillFlags["weapon2Attack"] && truthy(actor.ms.WeaponData2["range"]) {
				// dual wield average
				rng = (weapon1RangeBonus + skillModList.Sum("BASE", activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum("BASE", activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre") +
					anyNum(actor.ms.WeaponData2["rangeBonus"])) / 2
			} else {
				// primary hand attack
				rng = weapon1RangeBonus
			}
		} else {
			// unarmed
			rng = skillModList.Sum("BASE", activeSkill.SkillCfg, "UnarmedRange") +
				10*skillModList.Sum("BASE", activeSkill.SkillCfg, "UnarmedRangeMetre")
		}
		skillModList.AddMod(newMod("Multiplier:AdditionalMeleeRange", "BASE", rng, source))
	}
}

// bloodSacramentInitialFunc ports the Blood Sacrament (Sanguimancy) callback.
func bloodSacramentInitialFunc(env *Env, c *offenceCtx) {
	if outNum(c.output, "LifeReservedPercent") >= 100 {
		return
	}
	skillData := c.skillData
	lifeReservedPercent := 3.0
	if truthy(skillData["LifeReservedPercent"]) {
		lifeReservedPercent = anyNum(skillData["LifeReservedPercent"])
	}
	// `skillData.LifeReservedBase or math.huge`
	lifeReserved := mathHuge
	if truthy(skillData["LifeReservedBase"]) {
		lifeReserved = anyNum(skillData["LifeReservedBase"])
	}
	c.skillModList.AddMod(newMod("Multiplier:ChannelledLifeReservedPercentPerStage", "BASE", lifeReservedPercent, "Blood Sacrament"))
	c.skillModList.AddMod(newMod("Multiplier:ChannelledLifeReservedPerStage", "BASE", lifeReserved, "Blood Sacrament"))
}

// mathHuge is Lua's math.huge.
var mathHuge = math.Inf(1)

// explodeSourceKey ports `explodeSource.modSource or "Tree:"..explodeSource.id`.
func explodeSourceKey(src any) string {
	switch t := src.(type) {
	case *Item:
		if t.In.ModSource != nil {
			return *t.In.ModSource
		}
	case *NodeInput:
		return "Tree:" + strconv.FormatInt(int64(t.ID), 10)
	}
	panic("offence: explode source without a modSource")
}

// enemyExplodePreDamageFunc ports the EnemyExplode preDamageFunc
// (Data/Skills/other.lua L6076): which damage types the corpse explosion
// deals and the chance it happens.
func enemyExplodePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	explodeChance := 0.0
	part := anyNum(activeSkill.SkillPart)
	if part != 3 {
		src := activeSkill.ActiveEffect.SrcInstance.ExplodeSource
		activeEffectSource := explodeSourceKey(src)
		for _, entry := range skillModList.Tabulate("LIST", skillCfg, "ExplodeMod") {
			if entry.Mod.Source != activeEffectSource {
				continue
			}
			tag, _ := entry.Value.(modparser.Tag)
			typ := str(tag["type"])
			amount := anyNum(tag["amount"])
			if typ == "RandomElement" {
				skillData["FireEffectiveExplodePercentage"] = amount / 3
				skillData["ColdEffectiveExplodePercentage"] = amount / 3
				skillData["LightningEffectiveExplodePercentage"] = amount / 3
			} else {
				skillData[typ+"EffectiveExplodePercentage"] = amount
			}
			if part == 2 {
				explodeChance = 1
			} else {
				explodeChance = anyNum(tag["chance"])
			}
		}
	} else {
		// Every loop below is a commutative accumulation, so the reference's
		// pairs() order does not reach the result.
		type amountChance map[float64]float64
		typeAmountChances := map[string]amountChance{}
		for _, value := range skillModList.List(skillCfg, "ExplodeMod") {
			tag, _ := value.(modparser.Tag)
			typ := str(tag["type"])
			ac := typeAmountChances[typ]
			if ac == nil {
				ac = amountChance{}
				typeAmountChances[typ] = ac
			}
			ac[anyNum(tag["amount"])] += anyNum(tag["chance"])
		}
		for typ, ac := range typeAmountChances {
			physExplodeChance := 0.0
			for amount, chance := range ac {
				amountXChance := amount * chance
				if typ == "RandomElement" {
					for _, ele := range []string{"Fire", "Cold", "Lightning"} {
						skillData[ele+"EffectiveExplodePercentage"] = anyNum(skillData[ele+"EffectiveExplodePercentage"]) + amountXChance/3
					}
				} else {
					skillData[typ+"EffectiveExplodePercentage"] = anyNum(skillData[typ+"EffectiveExplodePercentage"]) + amountXChance
				}
				if typ == "Physical" {
					physExplodeChance = 1 - ((1 - physExplodeChance) * (1 - chance))
				}
				explodeChance = 1 - ((1 - explodeChance) * (1 - chance))
			}
			if typ == "Physical" && physExplodeChance != 0 {
				skillModList.AddMod(newMod("CalcArmourAsThoughDealing", "MORE", 100/math.Min(physExplodeChance, 1)-100))
			}
		}
	}
	output["ExplodeChance"] = math.Min(explodeChance*100, 100)
}

// brandHitTimeOverride ports the brand family's preDamageFunc: the brand's
// activation frequency becomes the skill's hit time.
func brandHitTimeOverride(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData["hitTimeOverride"] = anyNum(skillData["repeatFrequency"]) /
		(1 + skillModList.Sum("INC", skillCfg, "Speed", "BrandActivationFrequency")/100) /
		skillModList.More(skillCfg, "BrandActivationFrequency")
}

// righteousFirePreDamageFunc ports Righteous Fire's preDamageFunc: the burn
// scales off the totem's or the character's own life and energy shield.
func righteousFirePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	if activeSkill.SkillFlags["totem"] && outNum(output, "TotemLife") > 1 {
		skillData["FireDot"] = outNum(output, "TotemLife")*anyNum(skillData["RFLifeMultiplier"]) +
			outNum(output, "TotemEnergyShield")*anyNum(skillData["RFESMultiplier"])
	} else if outNum(output, "LifeUnreserved") > 1 {
		skillData["FireDot"] = outNum(output, "Life")*anyNum(skillData["RFLifeMultiplier"]) +
			outNum(output, "EnergyShield")*anyNum(skillData["RFESMultiplier"])
	}
}

// blazingSalvoPreDamageFunc ports Blazing Salvo's preDamageFunc: the
// "All Projectiles" skill part multiplies DPS by the projectile count.
func blazingSalvoPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if anyNum(activeSkill.SkillPart) != 2 {
		return
	}
	mult := 1.0
	if truthy(activeSkill.SkillData["dpsMultiplier"]) {
		mult = anyNum(activeSkill.SkillData["dpsMultiplier"])
	}
	activeSkill.SkillData["dpsMultiplier"] = mult * outNum(output, "ProjectileCount")
}

// shrapnelBallistaPreDamageFunc ports Shrapnel Ballista's preDamageFunc: the
// shotgunning overlap multiplies DPS, and splits that return add a
// conditional more-multiplier.
func shrapnelBallistaPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	if !skillModList.Flag(nil, "SequentialProjectiles") {
		mult := 1.0
		if truthy(skillData["dpsMultiplier"]) {
			mult = anyNum(skillData["dpsMultiplier"])
		}
		// `overlap or (Rain and ProjectileCount or 1)`
		overlap := 1.0
		if truthy(skillData["ShrapnelBallistaProjectileOverlap"]) {
			overlap = anyNum(skillData["ShrapnelBallistaProjectileOverlap"])
		} else if activeSkill.SkillTypes[modparser.SkillType.Rain] {
			overlap = outNum(output, "ProjectileCount")
		}
		skillData["dpsMultiplier"] = mult * math.Min(overlap, outNum(output, "ProjectileCount"))
	}
	if splitCount := outNum(output, "SplitCount"); splitCount > 0 {
		skillModList.AddMod(newMod("DPS", "MORE", splitCount*100, "Split Return", int64(0),
			modparser.Tag{"type": "Condition", "var": "ReturningProjectile"}))
	}
}

// explosiveTrapPreDamageFunc ports Explosive Trap's preDamageFunc: the small
// explosions land at a random radius, so how often one covers the enemy is a
// weighted average over the radii calcAreaOfEffect enumerated.
func explosiveTrapPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData, skillCfg := activeSkill.SkillModList, activeSkill.SkillData, activeSkill.SkillCfg
	// Not to be confused with attack hit chance: the share of the secondary
	// area in which a small explosion can spawn and still reach the enemy.
	// The -1 assumes PoE coordinates are integers and that areas sharing
	// only a point or vertex do not register damage.
	hitChance := func(enemyRadius, areaDamageRadius, areaSpreadRadius float64) float64 {
		damagingAreaRadius := areaDamageRadius + enemyRadius - 1
		return math.Min(damagingAreaRadius*damagingAreaRadius/(areaSpreadRadius*areaSpreadRadius), 1)
	}
	enemyRadius := 0.0
	if ov := skillModList.Override(skillCfg, "EnemyRadius"); truthy(ov) {
		enemyRadius = anyNum(ov)
	} else {
		enemyRadius = skillModList.Sum("BASE", skillCfg, "EnemyRadius")
	}
	fullRadius := outNum(output, "AreaOfEffectRadiusSecondary")
	overlapChance := 0.0
	marginWidth := anyNum(skillData["radiusTertiaryBaseMargin"])*2 + 1
	occurrences, _ := output["AreaOfEffectRadiusTertiaryOccurrences"].(map[float64]float64)
	for _, smallRadius := range sortedNumKeys(occurrences) {
		overlapChance += hitChance(enemyRadius, smallRadius, fullRadius) * occurrences[smallRadius] / marginWidth
	}
	output["OverlapChance"] = overlapChance * 100
	smallExplosionsPerTrap := skillModList.Sum("BASE", skillCfg, "SmallExplosions")
	output["SmallExplosionsPerTrap"] = smallExplosionsPerTrap
	dpsMultiplier := 1.0
	switch anyNum(activeSkill.SkillPart) {
	case 2:
		dpsMultiplier = 1 + smallExplosionsPerTrap*overlapChance
	case 3:
		dpsMultiplier = 1 + smallExplosionsPerTrap
	}
	if dpsMultiplier != 1 {
		mult := 1.0
		if truthy(skillData["dpsMultiplier"]) {
			mult = anyNum(skillData["dpsMultiplier"])
		}
		skillData["dpsMultiplier"] = mult * dpsMultiplier
		outMult := 1.0
		if truthy(output["SkillDPSMultiplier"]) {
			outMult = outNum(output, "SkillDPSMultiplier")
		}
		output["SkillDPSMultiplier"] = outMult * dpsMultiplier
	}
}

// explosiveArrowFunc ports Explosive Arrow's granted-effect callback
// (act_dex.lua:6696), which CalcOffence calls by skill name rather than
// through the callback registry. It works out how many fuses the attack can
// keep on the target and how often those explode.
func (env *Env) explosiveArrowFunc(c *offenceCtx, output map[string]any) {
	activeSkill, globalOutput := c.activeSkill, c.output
	// This doesn't apply to the "Arrow" skill part. That works like a
	// normal skill.
	part := anyNum(activeSkill.SkillPart)
	if part != 1 && part != 2 {
		return
	}

	modDB, enemyDB := c.modDB, c.enemyDB
	skillModList := activeSkill.SkillModList
	duration := env.calcSkillDuration(skillModList, activeSkill.SkillCfg, activeSkill.SkillData, enemyDB)
	fuseLimit := skillModList.Sum("BASE", activeSkill.SkillCfg, "ExplosiveArrowMaxFuseCount")
	activeTotems := 0.0
	if activeSkill.SkillFlags["totem"] {
		// Override returns no values when nothing matches, so the `or`
		// falls through to the limit sum.
		if ov := modDB.Override(nil, "TotemsSummoned"); truthy(ov) {
			activeTotems = anyNum(ov)
		} else {
			activeTotems = skillModList.Sum("BASE", activeSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
		}
	}

	barrageProjectiles := 0.0
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		barrageProjectiles = skillModList.Sum("BASE", activeSkill.SkillCfg, "ProjectileCount")
		// cancel out the normal dps multiplier from barrage that applies to
		// most other skills
		activeSkill.SkillData["dpsMultiplier"] = anyNum(activeSkill.SkillData["dpsMultiplier"]) / barrageProjectiles
	}

	projectiles := 1.0
	if barrageProjectiles != 0 {
		projectiles = barrageProjectiles
	}
	fuseApplicationRate := (outNum(output, "HitChance") / 100) * outNum(globalOutput, "Speed") *
		anyNum(activeSkill.SkillData["dpsMultiplier"]) * projectiles
	if activeSkill.SkillFlags["totem"] {
		fuseApplicationRate = fuseApplicationRate * activeTotems
	}

	// Calculate the max number of fuses you can sustain. Does not take into
	// account mines or traps.
	if part == 2 {
		maximum := math.Min(math.Floor(fuseApplicationRate*duration)+1, fuseLimit)
		skillModList.AddMod(newMod("Multiplier:ExplosiveArrowStage", "BASE", maximum, "Base"))
		skillModList.AddMod(newMod("Multiplier:ExplosiveArrowStageAfterFirst", "BASE", maximum-1, "Base"))
		globalOutput["MaxExplosiveArrowFuseCalculated"] = maximum
	} else {
		delete(globalOutput, "MaxExplosiveArrowFuseCalculated")
	}

	// Calculate explosion rate
	timeToMaxFuses := fuseLimit / fuseApplicationRate
	stageCount := 0.0
	if activeSkill.ActiveStageCount != nil {
		stageCount = *activeSkill.ActiveStageCount
	}
	if part == 2 || (part == 1 && stageCount+1 >= fuseLimit) {
		globalOutput["HitTime"] = math.Min(duration, timeToMaxFuses)
	} else {
		// Number of fuses is less than the limit, so the entire fuse
		// duration applies
		globalOutput["HitTime"] = duration
	}
	globalOutput["HitSpeed"] = 1 / outNum(globalOutput, "HitTime")
}

// iceSpearAltXPreDamageFunc ports Ice Spear of Splitting's preDamageFunc: the
// split parts hit once per projectile.
func iceSpearAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if part := anyNum(activeSkill.SkillPart); part == 3 || part == 4 {
		mult := 1.0
		if truthy(activeSkill.SkillData["dpsMultiplier"]) {
			mult = anyNum(activeSkill.SkillData["dpsMultiplier"])
		}
		activeSkill.SkillData["dpsMultiplier"] = mult * outNum(output, "ProjectileCount")
	}
}

// lightningTendrilsAltXPreDamageFunc ports Lightning Tendrils of
// Eccentricity's preDamageFunc: a DPS multiplier applied to the skill parts
// to reflect the DPS from each.
func lightningTendrilsAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	interval := anyNum(activeSkill.SkillData["pulseInterval"])
	switch anyNum(activeSkill.SkillPart) {
	case 2:
		activeSkill.SkillModList.AddMod(newMod("DPS", "MORE", -(1/interval)*100, "Normal pulse", int64(0), int64(0),
			modparser.Tag{"type": "SkillPart", "skillPart": 2.0}))
	case 3:
		activeSkill.SkillModList.AddMod(newMod("DPS", "MORE", -(interval-1)/interval*100, "Stronger pulse", int64(0), int64(0),
			modparser.Tag{"type": "SkillPart", "skillPart": 3.0}))
	}
}

// lightningTendrilsAltXPostCritFunc ports its postCritFunc: an effective
// damage multiplier that folds in the 500% more damage on every 5th hit.
func lightningTendrilsAltXPostCritFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if anyNum(activeSkill.SkillPart) != 1 {
		return
	}
	interval := anyNum(activeSkill.SkillData["pulseInterval"])
	pulseDamage := anyNum(activeSkill.SkillData["pulseDamage"]) / 100
	critChance := outNum(output, "PreEffectiveCritChance") / 100
	effectiveCritChance := outNum(output, "CritChance") / 100
	critMulti := outNum(output, "CritMultiplier")
	averageMore := 100 * (((interval-1)*(1+critChance*(critMulti-1))+(1+pulseDamage)*critMulti)/
		(interval*((1-effectiveCritChance)+critMulti*effectiveCritChance)) - 1)
	activeSkill.SkillModList.AddMod(newMod("Damage", "MORE", averageMore, "Average Pulse Damage", nil,
		modparser.KeywordFlag.Hit|modparser.KeywordFlag.Ailment,
		modparser.Tag{"type": "SkillPart", "skillPart": 1.0}))
}

// bladeBlastPreDamageFunc ports Blade Blast's preDamageFunc: one cast
// detonates every blade, and the "detonate all" part happens in one instant.
func bladeBlastPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	mult := 1.0
	if truthy(skillData["dpsMultiplier"]) {
		mult = anyNum(skillData["dpsMultiplier"])
	}
	skillData["dpsMultiplier"] = mult * anyNum(skillData["dpsBaseMultiplier"])
	if anyNum(c.activeSkill.SkillPart) == 2 {
		skillData["hitTimeOverride"] = 1.0
	}
}

// heraldOfTheBreachPreDamageFunc ports Herald of the Breach's preDamageFunc:
// the pulse delay shortens with stacked Otherworldly Pressure.
func heraldOfTheBreachPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData["hitTimeOverride"] = anyNum(skillData["repeatFrequency"]) /
		(1 + skillModList.Sum("INC", skillCfg, "PulseFrequencyPerPressure")/100)
}

// tornadoShotPreDamageFunc ports Tornado Shot's preDamageFunc: the secondary
// projectiles each get a chance to hit the same target.
func tornadoShotPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	if anyNum(activeSkill.SkillPart) != 2 || outNum(output, "ReturnChance") != 0 {
		return
	}
	averageSecondaryProjectiles := outNum(output, "ProjectileCount") + outNum(output, "SplitCount")
	// if barrage then only shoots 1 projectile at a time, but those can
	// still split and still releases at least 1 secondary projectile
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "SingleProjectile") &&
		!skillModList.Flag(nil, "TriggeredBySnipe") {
		averageSecondaryProjectiles = 1 + outNum(output, "SplitCount")
	}
	// default to 20% per secondary projectile, so 60% base, and 80% with
	// helm enchant
	secondary := 20 * skillModList.Sum("BASE", activeSkill.SkillCfg, "tornadoShotSecondaryProjectiles")
	if truthy(skillData["tornadoShotSecondaryHitChance"]) {
		secondary = anyNum(skillData["tornadoShotSecondaryHitChance"])
	}
	chanceForSecondaryProjectilesToHit := math.Min(secondary/100, 1)
	mult := 1.0
	if truthy(skillData["dpsMultiplier"]) {
		mult = anyNum(skillData["dpsMultiplier"])
	}
	skillData["dpsMultiplier"] = mult * (1 + chanceForSecondaryProjectilesToHit*averageSecondaryProjectiles)
}

// toxicRainPreDamageFunc ports Toxic Rain's preDamageFunc: only as many pods
// overlap the target as there are projectiles.
func toxicRainPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	overlap := 1.0
	if truthy(skillData["podOverlapMultiplier"]) {
		overlap = anyNum(skillData["podOverlapMultiplier"])
	}
	skillData["dpsMultiplier"] = math.Min(overlap, outNum(c.output, "ProjectileCount"))
}

// earthquakePreDamageFunc ports Earthquake's preDamageFunc: the aftershock
// hits harder the longer the fissure lasts.
func earthquakePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList := activeSkill.SkillModList
	duration := math.Floor(anyNum(activeSkill.SkillData["duration"]) * outNum(c.output, "DurationMod") * 10)
	skillModList.AddMod(newMod("Damage", "INC",
		skillModList.Sum("INC", activeSkill.SkillCfg, "EarthquakeDurationIncDamage")*duration, "Skill:Earthquake"))
}

// moltenStrikePreDamageFunc ports the Molten Strike family's preDamageFunc:
// how often a ball lands close enough to hit the same target it was struck
// from. The Zenith transfiguration adds parts 5 and 6, the latter a weighted
// average of normal and every-fifth-attack balls.
func moltenStrikePreDamageFunc(zenith bool) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		skillModList, skillData, skillCfg := activeSkill.SkillModList, activeSkill.SkillData, activeSkill.SkillCfg
		skillPart := anyNum(activeSkill.SkillPart)
		// melee part doesn't need to calc balls
		if skillPart == 1 {
			return
		}

		enemyRadius := 0.0
		if ov := skillModList.Override(skillCfg, "EnemyRadius"); truthy(ov) {
			enemyRadius = anyNum(ov)
		} else {
			enemyRadius = skillModList.Sum("BASE", skillCfg, "EnemyRadius")
		}
		ballRadius := outNum(output, "AreaOfEffectRadius")
		innerRadius := outNum(output, "AreaOfEffectRadiusSecondary")
		outerRadius := outNum(output, "AreaOfEffectRadiusTertiary")

		// logic adapted from MoldyDwarf's calculator
		hitRange := enemyRadius + ballRadius - innerRadius
		landingRange := outerRadius - innerRadius
		overlapChance := math.Min(1, hitRange/landingRange)
		output["OverlapChance"] = overlapChance * 100

		numProjectiles := outNum(output, "ProjectileCount")
		dpsMult := 1.0
		if skillPart == 3 || (zenith && (skillPart == 5 || skillPart == 6)) {
			dpsMult = overlapChance * numProjectiles
			if zenith && skillPart == 6 {
				// zenith: make an effective dpsMult for the weighted average
				// of normal and 5th attack balls
				fifthAttackMulti := 1 + anyNum(skillData["FifthStrikeDamage"])/100
				fifthAttackOverallMulti := fifthAttackMulti * overlapChance * (numProjectiles + anyNum(skillData["FifthStrikeProjectiles"]))
				dpsMult = 0.8*dpsMult + 0.2*fifthAttackOverallMulti
			}
		}
		if dpsMult != 1 {
			mult := 1.0
			if truthy(skillData["dpsMultiplier"]) {
				mult = anyNum(skillData["dpsMultiplier"])
			}
			skillData["dpsMultiplier"] = mult * dpsMult
			outMult := 1.0
			if truthy(output["SkillDPSMultiplier"]) {
				outMult = outNum(output, "SkillDPSMultiplier")
			}
			output["SkillDPSMultiplier"] = outMult * dpsMult
		}
	}
}
