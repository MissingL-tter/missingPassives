// Ports of the hand-written skill callbacks that Data/Skills/*.lua attaches
// to granted effects (initialFunc, preSkillTypeFunc, preDamageFunc,
// postCritFunc, preDotFunc). The data tables list each one in the skill's
// Custom.Callbacks; runSkillFunc consults this registry first and panics
// on anything still unported, so a corpus build can never silently skip a
// callback.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// skillFunc is one ported callback. It gets the same reach the Lua closure
// has: the active skill, the pass-independent output, and the environment.
type skillFunc func(env *Env, c *offenceCtx)

// skillFuncKey names one callback of one granted effect.
type skillFuncKey struct {
	ID   string
	Kind data.CallbackKind
}

var skillFuncs = map[skillFuncKey]skillFunc{
	{"Cyclone", data.CallbackInitial}:                 cycloneInitialFunc("Skill:Cyclone"),
	{"CycloneAltX", data.CallbackInitial}:             cycloneInitialFunc("Skill:CycloneAltX"),
	{"VaalCyclone", data.CallbackInitial}:             cycloneInitialFunc("Skill:Cyclone"),
	{"BloodSacramentUnique", data.CallbackInitial}:    bloodSacramentInitialFunc,
	{"EnemyExplode", data.CallbackPreDamage}:          enemyExplodePreDamageFunc,
	{"StormBrand", data.CallbackPreDamage}:            brandHitTimeOverride,
	{"PenanceBrandAltX", data.CallbackPreDamage}:      brandHitTimeOverride,
	{"HeraldOfTheBreach", data.CallbackPreDamage}:     heraldOfTheBreachPreDamageFunc,
	{"RighteousFire", data.CallbackPreDamage}:         righteousFirePreDamageFunc,
	{"BlazingSalvo", data.CallbackPreDamage}:          blazingSalvoPreDamageFunc,
	{"ShrapnelBallista", data.CallbackPreDamage}:      shrapnelBallistaPreDamageFunc,
	{"ExplosiveTrap", data.CallbackPreDamage}:         explosiveTrapPreDamageFunc,
	{"IceSpearAltX", data.CallbackPreDamage}:          iceSpearAltXPreDamageFunc,
	{"BladeBlast", data.CallbackPreDamage}:            bladeBlastPreDamageFunc,
	{"TornadoShot", data.CallbackPreDamage}:           tornadoShotPreDamageFunc,
	{"ToxicRain", data.CallbackPreDamage}:             toxicRainPreDamageFunc,
	{"Earthquake", data.CallbackPreDamage}:            earthquakePreDamageFunc,
	{"MoltenStrike", data.CallbackPreDamage}:          moltenStrikePreDamageFunc(false),
	{"MoltenStrikeAltX", data.CallbackPreDamage}:      moltenStrikePreDamageFunc(true),
	{"LightningTendrilsAltX", data.CallbackPreDamage}: lightningTendrilsAltXPreDamageFunc,
	{"LightningTendrilsAltX", data.CallbackPostCrit}:  lightningTendrilsAltXPostCritFunc,
	{"MoltenShell", data.CallbackPreDamage}:           moltenShellPreDamageFunc("MoltenShellDamageMitigated"),
	{"VaalMoltenShell", data.CallbackPreDamage}:       moltenShellPreDamageFunc("VaalMoltenShellDamageMitigated"),
	{"HeraldOfAsh", data.CallbackPreDamage}:           heraldOfAshPreDamageFunc,
	{"HeraldOfThunder", data.CallbackPreDamage}:       repeatFrequencyOverride("HeraldStormFrequency"),
	{"VoidSphere", data.CallbackPreDamage}:            repeatFrequencyOverride("VoidSphereFrequency"),
	{"Barrage", data.CallbackPreDamage}:               barragePreDamageFunc,
	{"Tornado", data.CallbackPreDamage}:               tornadoPreDamageFunc,
	{"LancingSteelAltX", data.CallbackPreDamage}:      lancingSteelAltXPreDamageFunc,
	{"RighteousFireAltX", data.CallbackPreDamage}:     righteousFireAltXPreDamageFunc,
	{"BrandSupport", data.CallbackPreDamage}:          brandHitTimeOverride,
	{"ToxicRainAltY", data.CallbackPreDamage}:         toxicRainPreDamageFunc,
	{"BladefallAltZ", data.CallbackPreDamage}:         bladefallAltZPreDamageFunc,
	{"ForbiddenRite", data.CallbackPreDamage}:         forbiddenRitePreDamageFunc,
}

// cycloneInitialFunc ports the Cyclone family's initialFunc: the melee range
// the skill's area scales with. The three copies differ only in mod source.
func cycloneInitialFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		skillModList := activeSkill.SkillModList
		actor := c.actor
		rng := 0.0
		wd1, wd2 := weaponOf(actor.ms.WeaponData1), weaponOf(actor.ms.WeaponData2)
		if activeSkill.SkillFlags["weapon1Attack"] && wd1 != nil && wd1.Range != 0 {
			weapon1RangeBonus := skillModList.Sum(modparser.Base, activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
				10*skillModList.Sum(modparser.Base, activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre") +
				wd1.RangeBonus.Or(0)
			if activeSkill.SkillFlags["weapon2Attack"] && wd2 != nil && wd2.Range != 0 {
				// dual wield average
				rng = (weapon1RangeBonus + skillModList.Sum(modparser.Base, activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum(modparser.Base, activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre") +
					wd2.RangeBonus.Or(0)) / 2
			} else {
				// primary hand attack
				rng = weapon1RangeBonus
			}
		} else {
			// unarmed
			rng = skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "UnarmedRange") +
				10*skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "UnarmedRangeMetre")
		}
		skillModList.AddMod(newModS("Multiplier:AdditionalMeleeRange", modparser.Base, modparser.Num(rng), source))
	}
}

// bloodSacramentInitialFunc ports the Blood Sacrament (Sanguimancy) callback.
func bloodSacramentInitialFunc(env *Env, c *offenceCtx) {
	if c.output.N("LifeReservedPercent") >= 100 {
		return
	}
	skillData := c.skillData
	lifeReservedPercent := 3.0
	if skillData.Flag("LifeReservedPercent") {
		lifeReservedPercent = skillData.N("LifeReservedPercent")
	}
	// `skillData.LifeReservedBase or math.huge`
	lifeReserved := math.Inf(1)
	if skillData.Flag("LifeReservedBase") {
		lifeReserved = skillData.N("LifeReservedBase")
	}
	c.skillModList.AddMod(newModS("Multiplier:ChannelledLifeReservedPercentPerStage", modparser.Base, modparser.Num(lifeReservedPercent), "Blood Sacrament"))
	c.skillModList.AddMod(newModS("Multiplier:ChannelledLifeReservedPerStage", modparser.Base, modparser.Num(lifeReserved), "Blood Sacrament"))
}

// enemyExplodePreDamageFunc ports the EnemyExplode preDamageFunc
// (Data/Skills/other.lua L6076): which damage types the corpse explosion
// deals and the chance it happens.
func enemyExplodePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	explodeChance := 0.0
	part := activeSkill.SkillPart.V
	if part != 3 {
		activeEffectSource := activeSkill.ActiveEffect.SrcInstance.ExplodeSource.ExplodeKey()
		for _, entry := range skillModList.Tabulate(modparser.List, skillCfg, "ExplodeMod") {
			if entry.Mod.Source != activeEffectSource {
				continue
			}
			tag, ok := entry.Value.(modparser.ExplodeRef)
			if !ok {
				panic("calc: non-ExplodeRef value in ExplodeMod list (the Lua errors)")
			}
			typ := tag.Type
			amount := tag.Amount
			if typ == "RandomElement" {
				skillData.SetN("FireEffectiveExplodePercentage", amount/3)
				skillData.SetN("ColdEffectiveExplodePercentage", amount/3)
				skillData.SetN("LightningEffectiveExplodePercentage", amount/3)
			} else {
				skillData.SetN(typ+"EffectiveExplodePercentage", amount)
			}
			if part == 2 {
				explodeChance = 1
			} else {
				explodeChance = tag.Chance
			}
		}
	} else {
		// Every loop below is a commutative accumulation, so the reference's
		// pairs() order does not reach the result.
		type amountChance map[float64]float64
		typeAmountChances := map[string]amountChance{}
		for _, value := range skillModList.List(skillCfg, "ExplodeMod") {
			tag, ok := value.(modparser.ExplodeRef)
			if !ok {
				panic("calc: non-ExplodeRef value in ExplodeMod list (the Lua errors)")
			}
			typ := tag.Type
			ac := typeAmountChances[typ]
			if ac == nil {
				ac = amountChance{}
				typeAmountChances[typ] = ac
			}
			ac[tag.Amount] += tag.Chance
		}
		for typ, ac := range typeAmountChances {
			physExplodeChance := 0.0
			for amount, chance := range ac {
				amountXChance := amount * chance
				if typ == "RandomElement" {
					for _, ele := range []string{"Fire", "Cold", "Lightning"} {
						skillData.SetN(ele+"EffectiveExplodePercentage", skillData.N(ele+"EffectiveExplodePercentage")+amountXChance/3)
					}
				} else {
					skillData.SetN(typ+"EffectiveExplodePercentage", skillData.N(typ+"EffectiveExplodePercentage")+amountXChance)
				}
				if typ == "Physical" {
					physExplodeChance = 1 - ((1 - physExplodeChance) * (1 - chance))
				}
				explodeChance = 1 - ((1 - explodeChance) * (1 - chance))
			}
			if typ == "Physical" && physExplodeChance != 0 {
				skillModList.AddMod(newMod("CalcArmourAsThoughDealing", modparser.More, modparser.Num(100/math.Min(physExplodeChance, 1)-100)))
			}
		}
	}
	output.SetN("ExplodeChance", math.Min(explodeChance*100, 100))
}

// brandHitTimeOverride ports the brand family's preDamageFunc: the brand's
// activation frequency becomes the skill's hit time.
func brandHitTimeOverride(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("repeatFrequency")/
		(1+skillModList.Sum(modparser.Inc, skillCfg, "Speed", "BrandActivationFrequency")/100)/
		skillModList.More(skillCfg, "BrandActivationFrequency"))
}

// righteousFirePreDamageFunc ports Righteous Fire's preDamageFunc: the burn
// scales off the totem's or the character's own life and energy shield.
func righteousFirePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	if activeSkill.SkillFlags["totem"] && output.N("TotemLife") > 1 {
		skillData.SetN("FireDot", output.N("TotemLife")*skillData.N("RFLifeMultiplier")+
			output.N("TotemEnergyShield")*skillData.N("RFESMultiplier"))
	} else if output.N("LifeUnreserved") > 1 {
		skillData.SetN("FireDot", output.N("Life")*skillData.N("RFLifeMultiplier")+
			output.N("EnergyShield")*skillData.N("RFESMultiplier"))
	}
}

// blazingSalvoPreDamageFunc ports Blazing Salvo's preDamageFunc: the
// "All Projectiles" skill part multiplies DPS by the projectile count.
func blazingSalvoPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if activeSkill.SkillPart.V != 2 {
		return
	}
	mult := 1.0
	if activeSkill.SkillData.Has("dpsMultiplier") {
		mult = activeSkill.SkillData.N("dpsMultiplier")
	}
	activeSkill.SkillData.SetN("dpsMultiplier", mult*output.N("ProjectileCount"))
}

// shrapnelBallistaPreDamageFunc ports Shrapnel Ballista's preDamageFunc: the
// shotgunning overlap multiplies DPS, and splits that return add a
// conditional more-multiplier.
func shrapnelBallistaPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	if !skillModList.Flag(nil, "SequentialProjectiles") {
		mult := 1.0
		if skillData.Has("dpsMultiplier") {
			mult = skillData.N("dpsMultiplier")
		}
		// `overlap or (Rain and ProjectileCount or 1)`
		overlap := 1.0
		if skillData.Flag("ShrapnelBallistaProjectileOverlap") {
			overlap = skillData.N("ShrapnelBallistaProjectileOverlap")
		} else if activeSkill.SkillTypes[modparser.SkillTypeRain] {
			overlap = output.N("ProjectileCount")
		}
		skillData.SetN("dpsMultiplier", mult*math.Min(overlap, output.N("ProjectileCount")))
	}
	if splitCount := output.N("SplitCount"); splitCount > 0 {
		skillModList.AddMod(newModSF("DPS", modparser.More, modparser.Num(splitCount*100), "Split Return", modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "ReturningProjectile"}))
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
	if ov, ok := skillModList.Override(skillCfg, "EnemyRadius"); ok {
		enemyRadius = valueNum(ov)
	} else {
		enemyRadius = skillModList.Sum(modparser.Base, skillCfg, "EnemyRadius")
	}
	fullRadius := output.N("AreaOfEffectRadiusSecondary")
	overlapChance := 0.0
	marginWidth := skillData.N("radiusTertiaryBaseMargin")*2 + 1
	occurrences := c.radiusTertiaryOccurrences
	for _, smallRadius := range sortedNumKeys(occurrences) {
		overlapChance += hitChance(enemyRadius, smallRadius, fullRadius) * occurrences[smallRadius] / marginWidth
	}
	output.SetN("OverlapChance", overlapChance*100)
	smallExplosionsPerTrap := skillModList.Sum(modparser.Base, skillCfg, "SmallExplosions")
	output.SetN("SmallExplosionsPerTrap", smallExplosionsPerTrap)
	dpsMultiplier := 1.0
	switch activeSkill.SkillPart.V {
	case 2:
		dpsMultiplier = 1 + smallExplosionsPerTrap*overlapChance
	case 3:
		dpsMultiplier = 1 + smallExplosionsPerTrap
	}
	if dpsMultiplier != 1 {
		mult := 1.0
		if skillData.Has("dpsMultiplier") {
			mult = skillData.N("dpsMultiplier")
		}
		skillData.SetN("dpsMultiplier", mult*dpsMultiplier)
		outMult := 1.0
		if output.Has("SkillDPSMultiplier") {
			outMult = output.N("SkillDPSMultiplier")
		}
		output.SetN("SkillDPSMultiplier", outMult*dpsMultiplier)
	}
}

// explosiveArrowFunc ports Explosive Arrow's granted-effect callback
// (act_dex.lua:6696), which CalcOffence calls by skill name rather than
// through the callback registry. It works out how many fuses the attack can
// keep on the target and how often those explode.
func (env *Env) explosiveArrowFunc(c *offenceCtx, output modstore.Output) {
	activeSkill, globalOutput := c.activeSkill, c.output
	// This doesn't apply to the "Arrow" skill part. That works like a
	// normal skill.
	part := activeSkill.SkillPart.V
	if part != 1 && part != 2 {
		return
	}

	modDB, enemyDB := c.modDB, c.enemyDB
	skillModList := activeSkill.SkillModList
	duration := env.calcSkillDuration(skillModList, activeSkill.SkillCfg, activeSkill.SkillData, enemyDB)
	fuseLimit := skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "ExplosiveArrowMaxFuseCount")
	activeTotems := 0.0
	if activeSkill.SkillFlags["totem"] {
		// Override returns no values when nothing matches, so the `or`
		// falls through to the limit sum.
		if ov, ok := modDB.Override(nil, "TotemsSummoned"); ok {
			activeTotems = valueNum(ov)
		} else {
			activeTotems = skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
		}
	}

	barrageProjectiles := 0.0
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		barrageProjectiles = skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "ProjectileCount")
		// cancel out the normal dps multiplier from barrage that applies to
		// most other skills
		activeSkill.SkillData.SetN("dpsMultiplier", activeSkill.SkillData.N("dpsMultiplier")/barrageProjectiles)
	}

	projectiles := 1.0
	if barrageProjectiles != 0 {
		projectiles = barrageProjectiles
	}
	fuseApplicationRate := (output.N("HitChance") / 100) * globalOutput.N("Speed") *
		activeSkill.SkillData.N("dpsMultiplier") * projectiles
	if activeSkill.SkillFlags["totem"] {
		fuseApplicationRate = fuseApplicationRate * activeTotems
	}

	// Calculate the max number of fuses you can sustain. Does not take into
	// account mines or traps.
	if part == 2 {
		maximum := math.Min(math.Floor(fuseApplicationRate*duration)+1, fuseLimit)
		skillModList.AddMod(newModS("Multiplier:ExplosiveArrowStage", modparser.Base, modparser.Num(maximum), "Base"))
		skillModList.AddMod(newModS("Multiplier:ExplosiveArrowStageAfterFirst", modparser.Base, modparser.Num(maximum-1), "Base"))
		globalOutput.SetN("MaxExplosiveArrowFuseCalculated", maximum)
	} else {
		globalOutput.Del("MaxExplosiveArrowFuseCalculated")
	}

	// Calculate explosion rate
	timeToMaxFuses := fuseLimit / fuseApplicationRate
	stageCount := 0.0
	if activeSkill.ActiveStageCount != nil {
		stageCount = *activeSkill.ActiveStageCount
	}
	if part == 2 || (part == 1 && stageCount+1 >= fuseLimit) {
		globalOutput.SetN("HitTime", math.Min(duration, timeToMaxFuses))
	} else {
		// Number of fuses is less than the limit, so the entire fuse
		// duration applies
		globalOutput.SetN("HitTime", duration)
	}
	globalOutput.SetN("HitSpeed", 1/globalOutput.N("HitTime"))
}

// iceSpearAltXPreDamageFunc ports Ice Spear of Splitting's preDamageFunc: the
// split parts hit once per projectile.
func iceSpearAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if part := activeSkill.SkillPart.V; part == 3 || part == 4 {
		mult := 1.0
		if activeSkill.SkillData.Has("dpsMultiplier") {
			mult = activeSkill.SkillData.N("dpsMultiplier")
		}
		activeSkill.SkillData.SetN("dpsMultiplier", mult*output.N("ProjectileCount"))
	}
}

// lightningTendrilsAltXPreDamageFunc ports Lightning Tendrils of
// Eccentricity's preDamageFunc: a DPS multiplier applied to the skill parts
// to reflect the DPS from each.
func lightningTendrilsAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	interval := activeSkill.SkillData.N("pulseInterval")
	switch activeSkill.SkillPart.V {
	case 2:
		activeSkill.SkillModList.AddMod(newModSF("DPS", modparser.More, modparser.Num(-(1/interval)*100), "Normal pulse", modparser.FlagNone, modparser.KeywordNone, &modparser.SkillPartTag{Part: opt(2.0)}))
	case 3:
		activeSkill.SkillModList.AddMod(newModSF("DPS", modparser.More, modparser.Num(-(interval-1)/interval*100), "Stronger pulse", modparser.FlagNone, modparser.KeywordNone, &modparser.SkillPartTag{Part: opt(3.0)}))
	}
}

// lightningTendrilsAltXPostCritFunc ports its postCritFunc: an effective
// damage multiplier that folds in the 500% more damage on every 5th hit.
func lightningTendrilsAltXPostCritFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if activeSkill.SkillPart.V != 1 {
		return
	}
	interval := activeSkill.SkillData.N("pulseInterval")
	pulseDamage := activeSkill.SkillData.N("pulseDamage") / 100
	critChance := output.N("PreEffectiveCritChance") / 100
	effectiveCritChance := output.N("CritChance") / 100
	critMulti := output.N("CritMultiplier")
	averageMore := 100 * (((interval-1)*(1+critChance*(critMulti-1))+(1+pulseDamage)*critMulti)/
		(interval*((1-effectiveCritChance)+critMulti*effectiveCritChance)) - 1)
	activeSkill.SkillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(averageMore), "Average Pulse Damage", modparser.FlagNone, modparser.KeywordHit|modparser.KeywordAilment, &modparser.SkillPartTag{Part: opt(1.0)}))
}

// bladeBlastPreDamageFunc ports Blade Blast's preDamageFunc: one cast
// detonates every blade, and the "detonate all" part happens in one instant.
func bladeBlastPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	mult := 1.0
	if skillData.Has("dpsMultiplier") {
		mult = skillData.N("dpsMultiplier")
	}
	skillData.SetN("dpsMultiplier", mult*skillData.N("dpsBaseMultiplier"))
	if c.activeSkill.SkillPart.V == 2 {
		skillData.SetN("hitTimeOverride", 1.0)
	}
}

// heraldOfTheBreachPreDamageFunc ports Herald of the Breach's preDamageFunc:
// the pulse delay shortens with stacked Otherworldly Pressure.
func heraldOfTheBreachPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("repeatFrequency")/
		(1+skillModList.Sum(modparser.Inc, skillCfg, "PulseFrequencyPerPressure")/100))
}

// tornadoShotPreDamageFunc ports Tornado Shot's preDamageFunc: the secondary
// projectiles each get a chance to hit the same target.
func tornadoShotPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	if activeSkill.SkillPart.V != 2 || output.N("ReturnChance") != 0 {
		return
	}
	averageSecondaryProjectiles := output.N("ProjectileCount") + output.N("SplitCount")
	// if barrage then only shoots 1 projectile at a time, but those can
	// still split and still releases at least 1 secondary projectile
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "SingleProjectile") &&
		!skillModList.Flag(nil, "TriggeredBySnipe") {
		averageSecondaryProjectiles = 1 + output.N("SplitCount")
	}
	// default to 20% per secondary projectile, so 60% base, and 80% with
	// helm enchant
	secondary := 20 * skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "tornadoShotSecondaryProjectiles")
	if skillData.Flag("tornadoShotSecondaryHitChance") {
		secondary = skillData.N("tornadoShotSecondaryHitChance")
	}
	chanceForSecondaryProjectilesToHit := math.Min(secondary/100, 1)
	mult := 1.0
	if skillData.Has("dpsMultiplier") {
		mult = skillData.N("dpsMultiplier")
	}
	skillData.SetN("dpsMultiplier", mult*(1+chanceForSecondaryProjectilesToHit*averageSecondaryProjectiles))
}

// toxicRainPreDamageFunc ports Toxic Rain's preDamageFunc: only as many pods
// overlap the target as there are projectiles.
func toxicRainPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	overlap := 1.0
	if skillData.Flag("podOverlapMultiplier") {
		overlap = skillData.N("podOverlapMultiplier")
	}
	skillData.SetN("dpsMultiplier", math.Min(overlap, c.output.N("ProjectileCount")))
}

// earthquakePreDamageFunc ports Earthquake's preDamageFunc: the aftershock
// hits harder the longer the fissure lasts.
func earthquakePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList := activeSkill.SkillModList
	duration := math.Floor(activeSkill.SkillData.N("duration") * c.output.N("DurationMod") * 10)
	skillModList.AddMod(newModS("Damage", modparser.Inc, modparser.Num(skillModList.Sum(modparser.Inc, activeSkill.SkillCfg, "EarthquakeDurationIncDamage")*duration), "Skill:Earthquake"))
}

// moltenStrikePreDamageFunc ports the Molten Strike family's preDamageFunc:
// how often a ball lands close enough to hit the same target it was struck
// from. The Zenith transfiguration adds parts 5 and 6, the latter a weighted
// average of normal and every-fifth-attack balls.
func moltenStrikePreDamageFunc(zenith bool) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		skillModList, skillData, skillCfg := activeSkill.SkillModList, activeSkill.SkillData, activeSkill.SkillCfg
		skillPart := activeSkill.SkillPart.V
		// melee part doesn't need to calc balls
		if skillPart == 1 {
			return
		}

		enemyRadius := 0.0
		if ov, ok := skillModList.Override(skillCfg, "EnemyRadius"); ok {
			enemyRadius = valueNum(ov)
		} else {
			enemyRadius = skillModList.Sum(modparser.Base, skillCfg, "EnemyRadius")
		}
		ballRadius := output.N("AreaOfEffectRadius")
		innerRadius := output.N("AreaOfEffectRadiusSecondary")
		outerRadius := output.N("AreaOfEffectRadiusTertiary")

		// logic adapted from MoldyDwarf's calculator
		hitRange := enemyRadius + ballRadius - innerRadius
		landingRange := outerRadius - innerRadius
		overlapChance := math.Min(1, hitRange/landingRange)
		output.SetN("OverlapChance", overlapChance*100)

		numProjectiles := output.N("ProjectileCount")
		dpsMult := 1.0
		if skillPart == 3 || (zenith && (skillPart == 5 || skillPart == 6)) {
			dpsMult = overlapChance * numProjectiles
			if zenith && skillPart == 6 {
				// zenith: make an effective dpsMult for the weighted average
				// of normal and 5th attack balls
				fifthAttackMulti := 1 + skillData.N("FifthStrikeDamage")/100
				fifthAttackOverallMulti := fifthAttackMulti * overlapChance * (numProjectiles + skillData.N("FifthStrikeProjectiles"))
				dpsMult = 0.8*dpsMult + 0.2*fifthAttackOverallMulti
			}
		}
		if dpsMult != 1 {
			mult := 1.0
			if skillData.Has("dpsMultiplier") {
				mult = skillData.N("dpsMultiplier")
			}
			skillData.SetN("dpsMultiplier", mult*dpsMult)
			outMult := 1.0
			if output.Has("SkillDPSMultiplier") {
				outMult = output.N("SkillDPSMultiplier")
			}
			output.SetN("SkillDPSMultiplier", outMult*dpsMult)
		}
	}
}

// moltenShellPreDamageFunc ports the Molten Shell pair's preDamageFunc: the
// burst reflects a share of what the shell absorbed. The two copies differ
// only in the config key holding the mitigated total.
func moltenShellPreDamageFunc(mitigatedKey string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		skillData := c.activeSkill.SkillData
		add := skillData.N(mitigatedKey) * skillData.N("moltenShellReflect") / 100
		skillData.SetN("FireMin", add)
		skillData.SetN("FireMax", add)
	}
}

// heraldOfAshPreDamageFunc ports Herald of Ash's preDamageFunc: the burn is
// a share of the overkill damage.
func heraldOfAshPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	skillData.SetN("FireDot", skillData.N("hoaOverkill")*
		(1+skillData.N("hoaMoreBurn")/100)*skillData.N("hoaOverkillPercent"))
}

// repeatFrequencyOverride is the shared shape of Herald of Thunder's and
// Void Sphere's preDamageFunc: the skill pulses on its own interval, sped up
// by one INC stat.
func repeatFrequencyOverride(incName string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		activeSkill.SkillData.SetN("hitTimeOverride", activeSkill.SkillData.N("repeatFrequency")/
			(1+activeSkill.SkillModList.Sum(modparser.Inc, activeSkill.SkillCfg, incName)/100))
	}
}

// barragePreDamageFunc ports Barrage's preDamageFunc: the "all projectiles"
// part hits once per projectile.
func barragePreDamageFunc(env *Env, c *offenceCtx) {
	if c.activeSkill.SkillPart.V == 2 {
		c.activeSkill.SkillData.SetN("dpsMultiplier", c.output.N("ProjectileCount"))
	}
}

// tornadoPreDamageFunc ports Tornado's preDamageFunc: it deals damage on its
// own interval while it lasts.
func tornadoPreDamageFunc(env *Env, c *offenceCtx) {
	c.activeSkill.SkillData.Set("hitTimeOverride", c.activeSkill.SkillData.Get("damageInterval"))
}

// lancingSteelAltXPreDamageFunc ports Lancing Steel of Spraying's
// preDamageFunc: every projectile past the first deals less damage, folded
// into one average multiplier over the count.
func lancingSteelAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if activeSkill.SkillPart.V != 2 {
		return
	}
	percentReducedProjectiles := (output.N("ProjectileCount") - 1) / output.N("ProjectileCount")
	mult := (activeSkill.SkillModList.More(activeSkill.SkillCfg, "LancingSteelSubsequentDamage") - 1) * 100 * percentReducedProjectiles
	activeSkill.SkillData.SetN("dpsMultiplier", output.N("ProjectileCount"))
	activeSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(mult), "Skill:LancingSteelAltX"))
}

// righteousFireAltXPreDamageFunc ports Righteous Fire of Arcane Devotion's
// preDamageFunc: the burn scales off mana instead of life.
func righteousFireAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if output.N("LifeUnreserved") > 1 {
		activeSkill.SkillData.SetN("FireDot", output.N("Mana")*activeSkill.SkillData.N("RFManaMultiplier"))
	}
}

// bladefallAltZPreDamageFunc ports Bladefall of Volleys' preDamageFunc: the
// volleys land on their own interval.
func bladefallAltZPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("hitFrequency")/(1+skillData.N("incVolleyFrequency")/100))
}

// forbiddenRitePreDamageFunc ports Forbidden Rite's preDamageFunc: the hit
// scales with the caster's life and energy shield, and the cast costs a
// chaos self-hit computed from the same pools.
func forbiddenRitePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	basetakenFlat := skillModList.Sum(modparser.Base, nil, "DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit")
	baseTakenInc := skillModList.Sum(modparser.Inc, nil, "DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit")
	baseTakenMore := skillModList.More(nil, "DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit")
	chaosDamageTaken := math.Max((1+baseTakenInc/100)*baseTakenMore, 0)
	chaosFlat := floorDec(math.Floor(basetakenFlat*chaosDamageTaken+0.5), 0)
	var life, energyShield, chaosResistance float64
	if activeSkill.SkillFlags["totem"] {
		life = output.N("TotemLife")
		energyShield = output.N("TotemEnergyShield")
		chaosResistance = output.N("TotemChaosResist")
	} else {
		life = output.N("Life")
		energyShield = output.N("EnergyShield")
		chaosResistance = output.N("ChaosResist")
	}
	add := life*skillData.N("lifeDealtAsChaos") + energyShield*skillData.N("energyShieldDealtAsChaos")
	selfDamageTakenLife := math.Floor(math.Floor(life*skillData.N("SelfDamageTakenLife")+0.5) * (100 - chaosResistance) / 100 * chaosDamageTaken)
	selfDamageTakenES := math.Floor(math.Floor(energyShield*skillData.N("SelfDamageTakenES")+0.5) * (100 - chaosResistance) / 100 * chaosDamageTaken)
	skillData.SetN("ChaosMin", skillData.N("ChaosMin")+add)
	skillData.SetN("ChaosMax", skillData.N("ChaosMax")+add)
	if activeSkill.SkillPart.V == 2 {
		mult := 1.0
		if skillData.Has("dpsMultiplier") {
			mult = skillData.N("dpsMultiplier")
		}
		skillData.SetN("dpsMultiplier", mult*(output.N("ProjectileCount")+1))
	}
	output.SetN("FRDamageTaken", selfDamageTakenLife+selfDamageTakenES+chaosFlat)
}
