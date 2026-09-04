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
	{"Cyclone", data.CallbackInitial}:                      cycloneInitialFunc("Skill:Cyclone"),
	{"CycloneAltX", data.CallbackInitial}:                  cycloneInitialFunc("Skill:CycloneAltX"),
	{"VaalCyclone", data.CallbackInitial}:                  cycloneInitialFunc("Skill:Cyclone"),
	{"BloodSacramentUnique", data.CallbackInitial}:         bloodSacramentInitialFunc,
	{"EnemyExplode", data.CallbackPreDamage}:               enemyExplodePreDamageFunc,
	{"StormBrand", data.CallbackPreDamage}:                 brandHitTimeOverride,
	{"PenanceBrandAltX", data.CallbackPreDamage}:           brandHitTimeOverride,
	{"HeraldOfTheBreach", data.CallbackPreDamage}:          heraldOfTheBreachPreDamageFunc,
	{"RighteousFire", data.CallbackPreDamage}:              righteousFirePreDamageFunc,
	{"BlazingSalvo", data.CallbackPreDamage}:               blazingSalvoPreDamageFunc,
	{"ShrapnelBallista", data.CallbackPreDamage}:           shrapnelBallistaPreDamageFunc,
	{"ExplosiveTrap", data.CallbackPreDamage}:              explosiveTrapPreDamageFunc,
	{"IceSpearAltX", data.CallbackPreDamage}:               iceSpearAltXPreDamageFunc,
	{"BladeBlast", data.CallbackPreDamage}:                 bladeBlastPreDamageFunc,
	{"TornadoShot", data.CallbackPreDamage}:                tornadoShotPreDamageFunc,
	{"ToxicRain", data.CallbackPreDamage}:                  toxicRainPreDamageFunc,
	{"Earthquake", data.CallbackPreDamage}:                 durationIncDamage("EarthquakeDurationIncDamage", "Skill:Earthquake"),
	{"MoltenStrike", data.CallbackPreDamage}:               moltenStrikePreDamageFunc(false),
	{"MoltenStrikeAltX", data.CallbackPreDamage}:           moltenStrikePreDamageFunc(true),
	{"LightningTendrilsAltX", data.CallbackPreDamage}:      lightningTendrilsAltXPreDamageFunc,
	{"LightningTendrilsAltX", data.CallbackPostCrit}:       lightningTendrilsAltXPostCritFunc,
	{"MoltenShell", data.CallbackPreDamage}:                moltenShellPreDamageFunc("MoltenShellDamageMitigated"),
	{"VaalMoltenShell", data.CallbackPreDamage}:            moltenShellPreDamageFunc("VaalMoltenShellDamageMitigated"),
	{"HeraldOfAsh", data.CallbackPreDamage}:                heraldOfAshPreDamageFunc,
	{"HeraldOfThunder", data.CallbackPreDamage}:            repeatFrequencyOverride("HeraldStormFrequency"),
	{"VoidSphere", data.CallbackPreDamage}:                 repeatFrequencyOverride("VoidSphereFrequency"),
	{"Barrage", data.CallbackPreDamage}:                    barragePreDamageFunc,
	{"Tornado", data.CallbackPreDamage}:                    tornadoPreDamageFunc,
	{"LancingSteelAltX", data.CallbackPreDamage}:           lancingSteelPreDamageFunc("Skill:LancingSteelAltX"),
	{"RighteousFireAltX", data.CallbackPreDamage}:          righteousFireAltXPreDamageFunc,
	{"BrandSupport", data.CallbackPreDamage}:               brandHitTimeOverride,
	{"ToxicRainAltY", data.CallbackPreDamage}:              toxicRainPreDamageFunc,
	{"BladefallAltZ", data.CallbackPreDamage}:              bladefallAltZPreDamageFunc,
	{"ForbiddenRite", data.CallbackPreDamage}:              forbiddenRitePreDamageFunc,
	{"ArmageddonBrand", data.CallbackPreDamage}:            brandHitTimeOverride,
	{"ArmageddonBrandAltY", data.CallbackPreDamage}:        brandHitTimeOverride,
	{"PenanceBrandAltY", data.CallbackPreDamage}:           brandHitTimeOverride,
	{"StormBrandAltX", data.CallbackPreDamage}:             brandHitTimeOverride,
	{"PenanceBrand", data.CallbackPreDamage}:               penanceBrandPreDamageFunc,
	{"WintertideBrand", data.CallbackPreDamage}:            wintertideBrandPreDamageFunc,
	{"VoidSphereAltX", data.CallbackPreDamage}:             repeatFrequencyOverride("VoidSphereFrequency"),
	{"StaticStrike", data.CallbackPreDamage}:               repeatFrequencyOverrideParts("StaticStrikeFrequency", 2),
	{"Hydrosphere", data.CallbackPreDamage}:                repeatFrequencyOverrideParts("HydroSphereFrequency", 1, 2, 3),
	{"GalvanicField", data.CallbackPreDamage}:              repeatFrequencyHitTime,
	{"GalvanicFieldAltX", data.CallbackPreDamage}:          repeatFrequencyHitTime,
	{"WinterOrb", data.CallbackPreDamage}:                  winterOrbPreDamageFunc,
	{"OrbOfStorms", data.CallbackPreDamage}:                orbOfStormsPreDamageFunc,
	{"BladeVortex", data.CallbackPreDamage}:                bladeVortexPreDamageFunc,
	{"VaalBladeVortex", data.CallbackPreDamage}:            vaalBladeVortexPreDamageFunc,
	{"Cremation", data.CallbackPreDamage}:                  cremationHitTime(true),
	{"CremationAltX", data.CallbackPreDamage}:              cremationHitTime(true),
	{"CremationAltY", data.CallbackPreDamage}:              cremationHitTime(false),
	{"Vortex", data.CallbackPreDamage}:                     cooldownHitTime,
	{"VortexAltX", data.CallbackPreDamage}:                 cooldownHitTime,
	{"LightningBolt", data.CallbackPreDamage}:              cooldownHitTime,
	{"GAZombieCorpseGroundImpact", data.CallbackPreDamage}: summonSpeedHitTime,
	{"MinionInstability", data.CallbackPreDamage}:          summonSpeedHitTime,
	{"StormRain", data.CallbackPreDamage}:                  stormRainPreDamageFunc(stormRainBeamOverlap),
	{"StormRainAltX", data.CallbackPreDamage}:              stormRainPreDamageFunc(stormRainActiveArrows),
	{"StormRainAltY", data.CallbackPreDamage}:              stormRainPreDamageFunc(stormRainAllowedArrows),
	{"TornadoAltY", data.CallbackPreDamage}:                tornadoAltYPreDamageFunc,
	{"DivineIre", data.CallbackPreDamage}:                  stageHitTimeMultiplier("Multiplier:DivineIreStage", 2),
	{"DivineIreAltX", data.CallbackPreDamage}:              stageHitTimeMultiplier("Multiplier:DivineIreofHolyLightningStage", 2),
	{"DivineIreAltY", data.CallbackPreDamage}:              stageHitTimeMultiplier("Multiplier:DivineIreofDisintegrationStage"),
	{"Flameblast", data.CallbackPreDamage}:                 stageHitTimeMultiplierFloored("Multiplier:FlameblastStage", "Multiplier:FlameblastMinimumStage", 0, 1),
	{"FlameblastAltX", data.CallbackPreDamage}:             stageHitTimeMultiplierFloored("Multiplier:FlameblastofCelerityStage", "Multiplier:FlameblastMinimumStage", 0, 1),
	{"FlameblastAltY", data.CallbackPreDamage}:             stageHitTimeMultiplierFloored("Multiplier:FlameblastofContractionStage", "Multiplier:FlameblastMinimumStage", 0, 1),
	{"Incinerate", data.CallbackPreDamage}:                 stageHitTimeMultiplierFloored("Multiplier:IncinerateStage", "Multiplier:IncinerateMinimumStage", 0.4175, 0.5825, 2),
	{"IncinerateAltX", data.CallbackPreDamage}:             stageHitTimeMultiplierFloored("Multiplier:IncinerateofExpanseStage", "Multiplier:IncinerateMinimumStage", 0.4175, 0.5825, 2),
	{"IncinerateAltY", data.CallbackPreDamage}:             stageHitTimeMultiplierFloored("Multiplier:IncinerateofVentingStage", "Multiplier:IncinerateMinimumStage", 0.4175, 0.5825, 2),
	{"ScourgeArrow", data.CallbackPreDamage}:               scourgeArrowPreDamageFunc,
	{"BarrageAltX", data.CallbackPreDamage}:                barragePreDamageFunc,
	{"BlastRain", data.CallbackPreDamage}:                  barragePreDamageFunc,
	{"VaalLightningArrow", data.CallbackPreDamage}:         barragePreDamageFunc,
	{"AzmeriHydraBarrage", data.CallbackPreDamage}:         azmeriHydraBarragePreDamageFunc,
	{"IceSpear", data.CallbackPreDamage}:                   iceSpearAltXPreDamageFunc,
	{"ToxicRainAltX", data.CallbackPreDamage}:              toxicRainPreDamageFunc,
	{"ShrapnelBallistaAltX", data.CallbackPreDamage}:       shrapnelBallistaPreDamageFunc,
	{"ExplosiveTrapAltX", data.CallbackPreDamage}:          explosiveTrapPreDamageFunc,
	{"BladeBlastAltX", data.CallbackPreDamage}:             bladeBlastPreDamageFunc,
	{"Perforate", data.CallbackPreDamage}:                  perforatePreDamageFunc,
	{"PerforateAltX", data.CallbackPreDamage}:              perforatePreDamageFunc,
	{"PerforateAltY", data.CallbackPreDamage}:              perforatePreDamageFunc,
	{"BladeFlurry", data.CallbackPreDamage}:                bladeFlurryPreDamageFunc,
	{"BladeFlurryAltX", data.CallbackPreDamage}:            bladeFlurryPreDamageFunc,
	{"Spark", data.CallbackPreDamage}:                      sparkPreDamageFunc,
	{"SparkAltX", data.CallbackPreDamage}:                  sparkPreDamageFunc,
	{"SparkAltY", data.CallbackPreDamage}:                  sparkPreDamageFunc,
	{"BallLightning", data.CallbackPreDamage}:              ballLightningPreDamageFunc,
	{"BallLightningAltX", data.CallbackPreDamage}:          ballLightningAltXPreDamageFunc,
	{"BallLightningAltY", data.CallbackPreDamage}:          ballLightningAltYPreDamageFunc,
	{"StormBurst", data.CallbackPreDamage}:                 stormBurstPreDamageFunc,
	{"StormBurstAltX", data.CallbackPreDamage}:             stormBurstAltXPreDamageFunc,
	{"FlamethrowerTrap", data.CallbackPreDamage}:           flamethrowerTrapPreDamageFunc(false),
	{"FlamethrowerTrapAltX", data.CallbackPreDamage}:       flamethrowerTrapPreDamageFunc(true),
	{"InfernalBlow", data.CallbackPreDamage}:               infernalBlowPreDamageFunc("Skill:InfernalBlow"),
	{"InfernalBlowAltX", data.CallbackPreDamage}:           infernalBlowPreDamageFunc("Skill:InfernalBlowAltX"),
	{"LancingSteel", data.CallbackPreDamage}:               lancingSteelPreDamageFunc("Skill:LancingSteel"),
	{"KineticFusillade", data.CallbackPreDamage}:           kineticFusilladePreDamageFunc("Skill:KineticFusillade"),
	{"KineticFusilladeAltX", data.CallbackPreDamage}:       kineticFusilladePreDamageFunc("Skill:KineticFusilladeAltX"),
	{"KineticFusillade", data.CallbackPostCrit}:            kineticFusilladePostCritFunc,
	{"KineticFusilladeAltX", data.CallbackPostCrit}:        kineticFusilladePostCritFunc,
	{"SeismicTrap", data.CallbackPreDamage}:                trapWavePreDamageFunc(trapWaveShape{frequencyStats: []string{"TrapThrowingSpeed", "SeismicPulseFrequency"}, pulses: true, parts: 6}),
	{"SeismicTrapAltX", data.CallbackPreDamage}:            trapWavePreDamageFunc(trapWaveShape{parts: 3}),
	{"LightningSpireTrap", data.CallbackPreDamage}:         trapWavePreDamageFunc(trapWaveShape{radiusFromData: true, frequencyStats: []string{"TrapThrowingSpeed"}, pulses: true, parts: 6}),
	{"LightningSpireTrapAltX", data.CallbackPreDamage}:     trapWavePreDamageFunc(trapWaveShape{radiusFromData: true, pulses: true, throwTimeTraps: true, parts: 6}),
	{"LightningSpireTrapAltY", data.CallbackPreDamage}:     trapWavePreDamageFunc(trapWaveShape{radiusFromData: true, pulses: true, parts: 6}),
	{"Bodyswap", data.CallbackPreDamage}:                   bodyswapPreDamageFunc,
	{"BodyswapAltX", data.CallbackPreDamage}:               bodyswapPreDamageFunc,
	{"DeathWish", data.CallbackPreDamage}:                  deathWishPreDamageFunc,
	{"DarkPact", data.CallbackPreDamage}:                   darkPactPreDamageFunc,
	{"DarkPactAltX", data.CallbackPreDamage}:               darkPactAltXPreDamageFunc,
	{"VaalRighteousFire", data.CallbackPreDamage}:          vaalRighteousFirePreDamageFunc,
	{"ForbiddenRiteAltX", data.CallbackPreDamage}:          forbiddenRiteAltXPreDamageFunc,
	{"Manabond", data.CallbackPreDamage}:                   manabondPreDamageFunc,
	{"PoisonousConcoction", data.CallbackPreDamage}:        poisonousConcoctionPreDamageFunc,
	{"PoisonousConcoctionAltX", data.CallbackPreDamage}:    poisonousConcoctionPreDamageFunc,
	{"AvengingFlame", data.CallbackPreDamage}:              avengingFlamePreDamageFunc,
	{"FreezingPulse", data.CallbackPreDamage}:              freezingPulsePreDamageFunc,
	{"EyeOfWinter", data.CallbackPreDamage}:                eyeOfWinterPreDamageFunc("Skill:EyeOfWinter"),
	{"EyeOfWinterAltX", data.CallbackPreDamage}:            eyeOfWinterPreDamageFunc("Skill:EyeOfWinterAltX"),
	{"EyeOfWinterAltY", data.CallbackPreDamage}:            eyeOfWinterPreDamageFunc("Skill:EyeOfWinterAltY"),
	{"ChargedDash", data.CallbackPreDamage}:                chargedDashPreDamageFunc("Skill:ChargedDash"),
	{"ChargedDashAltX", data.CallbackPreDamage}:            chargedDashPreDamageFunc("Skill:ChargedDashAltX"),
	{"FrostBombAltY", data.CallbackPreDamage}:              frostBombAltYPreDamageFunc,
	{"VoltaxicBurst", data.CallbackPreDamage}:              durationIncDamage("VoltaxicDurationIncDamage", "Skill:VoltaxicBurst"),
	{"WaveOfConviction", data.CallbackPreDamage}:           waveOfConvictionPreDamageFunc("Skill:WaveOfConviction"),
	{"WaveOfConvictionAltY", data.CallbackPreDamage}:       waveOfConvictionPreDamageFunc("Skill:WaveOfConvictionAltY"),
	{"Bane", data.CallbackPreSkillType}:                    banePreSkillTypeFunc,
	{"BaneAltX", data.CallbackPreSkillType}:                banePreSkillTypeFunc,
	{"RageVortex", data.CallbackPreSkillType}:              rageVortexPreSkillTypeFunc,
	{"FrozenSweep", data.CallbackPreDamage}:                frozenSweepPreDamageFunc("Frozen Legion"),
	{"FrozenSweepAltX", data.CallbackPreDamage}:            frozenSweepPreDamageFunc("Frozen Legion of Rallying"),
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
	enemyRadius := configuredEnemyRadius(skillModList, skillCfg)
	fullRadius := output.N("AreaOfEffectRadiusSecondary")
	overlapChance := 0.0
	marginWidth := skillData.N("radiusTertiaryBaseMargin")*2 + 1
	occurrences := c.radiusTertiaryOccurrences
	for _, smallRadius := range sortedNumKeys(occurrences) {
		overlapChance += areaHitChance(enemyRadius, smallRadius, fullRadius) * occurrences[smallRadius] / marginWidth
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

// durationIncDamage ports Earthquake's (act_str.lua) and Voltaxic Burst's
// (act_int.lua L20445) preDamageFunc: the delayed hit gains increased
// damage per 100ms of the delay. The two differ only in the stat and the
// mod source.
func durationIncDamage(incName, source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		skillModList := activeSkill.SkillModList
		duration := math.Floor(activeSkill.SkillData.N("duration") * c.output.N("DurationMod") * 10)
		skillModList.AddMod(newModS("Damage", modparser.Inc, modparser.Num(skillModList.Sum(modparser.Inc, activeSkill.SkillCfg, incName)*duration), source))
	}
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

// lancingSteelPreDamageFunc ports the Lancing Steel pair's preDamageFunc
// (act_dex.lua L10231 and its Spraying twin): every projectile past the
// first deals less damage, folded into one average multiplier over the
// count. The two copies differ only in the mod source.
func lancingSteelPreDamageFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		if activeSkill.SkillPart.V != 2 {
			return
		}
		percentReducedProjectiles := (output.N("ProjectileCount") - 1) / output.N("ProjectileCount")
		mult := (activeSkill.SkillModList.More(activeSkill.SkillCfg, "LancingSteelSubsequentDamage") - 1) * 100 * percentReducedProjectiles
		activeSkill.SkillData.SetN("dpsMultiplier", output.N("ProjectileCount"))
		activeSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(mult), source))
	}
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

// penanceBrandPreDamageFunc ports Penance Brand's preDamageFunc
// (act_int.lua L13809): the brand shape, times the energy the brand
// accumulates before it explodes.
func penanceBrandPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("repeatFrequency")*
		skillModList.Sum(modparser.Base, skillCfg, "Multiplier:PenanceBrandMaxEnergy")/
		(1+skillModList.Sum(modparser.Inc, skillCfg, "Speed", "BrandActivationFrequency")/100)/
		skillModList.More(skillCfg, "BrandActivationFrequency"))
}

// wintertideBrandPreDamageFunc ports Wintertide Brand's preDamageFunc
// (act_int.lua L21107): the brand shape, and for the "Average Damage" part
// one MORE multiplier averaging the stages the debuff climbs through over
// its attached duration plus the Wintertide's End uptime.
func wintertideBrandPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData.SetFlag("countsAttachedBrandsInDamage", activeSkill.SkillPart.V == 1)
	skillData.SetN("hitTimeOverride", skillData.N("repeatFrequency")/
		(1+skillModList.Sum(modparser.Inc, skillCfg, "Speed", "BrandActivationFrequency")/100)/
		skillModList.More(skillCfg, "BrandActivationFrequency"))
	if activeSkill.SkillPart.V != 1 {
		return
	}
	hitTime := skillData.N("hitTimeOverride")
	skillMaxStages := skillModList.Sum(modparser.Base, skillCfg, "Multiplier:WintertideBrandMaxStages")
	debuffDurationMult := 1 / math.Max(data.Misc.BuffExpirationSlowCap, Mod(c.enemyDB, skillCfg, "BuffExpireFaster"))
	// The reference passes an empty env, so calcSkillDuration's own
	// effective-mode debuff scaling never applies here.
	duration := calcSkillDurationMode(false, skillModList, skillCfg, skillData, nil) * debuffDurationMult
	maxStages := math.Min(math.Floor(duration/hitTime), skillMaxStages)
	timeToReachMaxStages := maxStages * hitTime
	timeAtMaxStages := math.Max(duration-timeToReachMaxStages, 0)
	damagePerStage := skillModList.Sum(modparser.Base, skillCfg, "Multiplier:WintertideBrandDamagePerStage")
	// Each activation adds one stage; average the completed stages over the
	// attached duration
	averageStages := (maxStages*(maxStages-1)/2*hitTime + maxStages*timeAtMaxStages) / duration
	averageDamageMultiplier := averageStages * damagePerStage
	endDurationBase := skillData.N("durationTertiary") + skillModList.Sum(modparser.Base, skillCfg, "Duration", "TertiaryDuration")
	endDurationMod := math.Max(Mod(skillModList, skillCfg, "Duration", "TertiaryDuration"), 0)
	endDuration := math.Ceil(endDurationBase*endDurationMod*debuffDurationMult*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	endUptime := math.Min(endDuration/duration, 1)
	attachedBrandCount := skillData.N("attachedBrandCount")
	// Attached Wintertide debuffs stack, but only the strongest
	// maximum-stage Wintertide's End debuff deals damage
	endDamageMultiplier := maxStages * damagePerStage
	dpsMultiplier := attachedBrandCount*(100+averageDamageMultiplier) + endUptime*(100+endDamageMultiplier) - 100
	skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(dpsMultiplier), "Wintertide Brand Average Multiplier"))
}

// skillDataOr is the `skillData.key or def` idiom: a present value wins,
// zero included (0 is truthy in Lua), and only an absent key falls back.
func skillDataOr(sd *SkillData, key string, def float64) float64 {
	if sd.Has(key) {
		return sd.N(key)
	}
	return def
}

// repeatFrequencyOverrideParts is repeatFrequencyOverride gated on the skill
// part: Static Strike's beams (part 2) and Hydrosphere's autopulse parts
// (1 to 3) pulse on their own interval; the other parts are cast normally.
func repeatFrequencyOverrideParts(incName string, parts ...float64) skillFunc {
	inner := repeatFrequencyOverride(incName)
	return func(env *Env, c *offenceCtx) {
		for _, part := range parts {
			if c.activeSkill.SkillPart.V == part {
				inner(env, c)
				return
			}
		}
	}
}

// repeatFrequencyHitTime ports the Galvanic Field pair's preDamageFunc: the
// field pulses on its bare repeat frequency, nothing speeds it up.
func repeatFrequencyHitTime(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("repeatFrequency"))
}

// winterOrbPreDamageFunc ports Winter Orb's preDamageFunc (act_int.lua
// L20983): the orb fires on its interval, sped up by cast speed and hit rate.
func winterOrbPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	rateMod := 1 + skillModList.Sum(modparser.Inc, skillCfg, "HitRate", "Speed")/100
	mult := skillModList.More(skillCfg, "HitRate")
	skillData.SetN("hitTimeOverride", skillData.N("repeatFrequency")/rateMod/mult)
}

// orbOfStormsPreDamageFunc ports Orb of Storms' preDamageFunc (act_int.lua
// L13621): the orb's bolt frequency scales with cast speed.
func orbOfStormsPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillData := activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("hitFrequency")/Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "Speed"))
}

// bladeVortexPreDamageFunc ports Blade Vortex's preDamageFunc (act_dex.lua
// L2209): more blades hit more often, the blade count coming from the part.
func bladeVortexPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillData := activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("hitFrequency")/
		(1+activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "Multiplier:BladeVortexBlade")*skillData.N("hitFrequencyPerBlade")))
}

// vaalBladeVortexPreDamageFunc ports Vaal Blade Vortex's preDamageFunc
// (act_dex.lua L2428): the same shape with the blade count a constant of the
// skill's own data.
func vaalBladeVortexPreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	skillData.SetN("hitTimeOverride", skillData.N("hitFrequency")/
		(1+skillData.N("VaalBladeVortexBlade")*skillDataOr(skillData, "hitFrequencyPerBlade", 0)))
}

// cremationHitTime ports the Cremation family's preDamageFunc (act_dex.lua
// L4353, L4471, L4590): the geyser fires on its own rate, sped up by the
// gem's fire-rate stat. The two base gems apply it only to the "Spell" part
// (1); Cremation of the Volcano has no parts and always does.
func cremationHitTime(partOneOnly bool) skillFunc {
	return func(env *Env, c *offenceCtx) {
		if partOneOnly && c.activeSkill.SkillPart.V != 1 {
			return
		}
		skillData := c.activeSkill.SkillData
		skillData.SetN("hitTimeOverride", skillData.N("cremationFireRate")/(1+skillDataOr(skillData, "cremationFireRateIncrease", 0)))
	}
}

// cooldownHitTime ports the Vortex pair's and Lightning Bolt's
// preDamageFunc: the skill fires once per cooldown.
func cooldownHitTime(env *Env, c *offenceCtx) {
	c.activeSkill.SkillData.Set("hitTimeOverride", c.output.Get("Cooldown"))
}

// summonSpeedHitTime ports Falling Slam's and Minion Instability's
// preDamageFunc (minion.lua L757, L1757): a minion skill fires once per
// summon of the skill that raised it.
func summonSpeedHitTime(env *Env, c *offenceCtx) {
	summonData := c.activeSkill.SummonSkill.SkillData
	c.activeSkill.SkillData.SetN("hitTimeOverride", 1/skillDataOr(summonData, "summonSpeed", 1))
}

// stormRainDPSMultiplier is how one Storm Rain variant counts the beams
// that overlap the target.
type stormRainDPSMultiplier func(c *offenceCtx) float64

func stormRainBeamOverlap(c *offenceCtx) float64 {
	return skillDataOr(c.activeSkill.SkillData, "beamOverlapMultiplier", 1)
}

func stormRainActiveArrows(c *offenceCtx) float64 {
	activeSkill := c.activeSkill
	return math.Min(skillDataOr(activeSkill.SkillData, "activeArrowMultiplier", 1),
		activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "StormRainAllowedStormArrows"))
}

// Max of 2 arrows, and each fires at each other, so 2 beams per tick
func stormRainAllowedArrows(c *offenceCtx) float64 {
	return c.activeSkill.SkillModList.Sum(modparser.Base, c.activeSkill.SkillCfg, "StormRainAllowedStormArrows")
}

// stormRainPreDamageFunc ports the Storm Rain family's preDamageFunc
// (act_dex.lua L11744, L11837, L11955): the "Beam" part (2) fires on the
// beam frequency, and the variants differ only in how many beams overlap.
func stormRainPreDamageFunc(beams stormRainDPSMultiplier) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		if activeSkill.SkillPart.V != 2 {
			return
		}
		skillData := activeSkill.SkillData
		skillData.SetN("hitTimeOverride", skillData.N("hitFrequency")/
			(1+activeSkill.SkillModList.Sum(modparser.Inc, activeSkill.SkillCfg, "StormRainBeamFrequency")/100))
		skillData.SetN("dpsMultiplier", beams(c))
	}
}

// tornadoAltYPreDamageFunc ports Tornado of Elemental Turbulence's
// preDamageFunc (act_dex.lua L18124): Tornado's interval, and one hit per
// stage.
func tornadoAltYPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillData := activeSkill.SkillData
	skillData.Set("hitTimeOverride", skillData.Get("damageInterval"))
	skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*
		activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "Multiplier:TornadoofElementalTurbulenceStage"))
}

// partGate reports whether the skill's part is one of parts; an empty list
// means the callback applies to every part.
func partGate(activeSkill *ActiveSkill, parts []float64) bool {
	if len(parts) == 0 {
		return true
	}
	for _, part := range parts {
		if activeSkill.SkillPart.V == part {
			return true
		}
	}
	return false
}

// stageHitTimeMultiplier ports the Divine Ire family's preDamageFunc
// (act_int.lua L4602, L4712, L4823): the release takes one channel tick
// per stage, so the stage count multiplies the hit time. The base gems
// apply it to the "Release" part (2) only.
func stageHitTimeMultiplier(stageName string, parts ...float64) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		if !partGate(activeSkill, parts) {
			return
		}
		activeSkill.SkillData.SetN("hitTimeMultiplier", activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, stageName))
	}
}

// stageHitTimeMultiplierFloored ports the Flameblast (act_int.lua L6800,
// L6905, L7008) and Incinerate (L10284, L10422, L10561) families'
// preDamageFunc: stages granted up front cost no channel time, and the
// first stage of Incinerate takes 0.5825x the time of the rest, so the
// multiplier is the stage count less the free stages less an offset, never
// below a floor. Incinerate applies it to the "Release" part (2) only.
func stageHitTimeMultiplierFloored(stageName, minimumName string, offset, floor float64, parts ...float64) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		if !partGate(activeSkill, parts) {
			return
		}
		skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
		activeSkill.SkillData.SetN("hitTimeMultiplier", math.Max(
			skillModList.Sum(modparser.Base, skillCfg, stageName)-skillModList.Sum(modparser.Base, skillCfg, minimumName)-offset, floor))
	}
}

// scourgeArrowPreDamageFunc ports Scourge Arrow's preDamageFunc
// (act_dex.lua L12987): the first stage takes 0.5x the time of the rest.
// The reference sums with `cfg`, an undeclared global, so the stage
// multiplier is read with no cfg.
func scourgeArrowPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	activeSkill.SkillData.SetN("hitTimeMultiplier", math.Max(activeSkill.SkillModList.Sum(modparser.Base, nil, "Multiplier:ScourgeArrowStage")-0.5, 0.5))
}

// azmeriHydraBarragePreDamageFunc ports the Hydra spectre's Barrage
// preDamageFunc (spectre.lua L7278): every projectile hits, with no part to
// choose.
func azmeriHydraBarragePreDamageFunc(env *Env, c *offenceCtx) {
	c.activeSkill.SkillData.SetN("dpsMultiplier", c.output.N("ProjectileCount"))
}

// outputOr is the `output.key or def` idiom on an output bag: a present
// value wins, zero included, and only an absent key falls back.
func outputOr(o modstore.Output, key string, def float64) float64 {
	if o.Has(key) {
		return o.N(key)
	}
	return def
}

// perforatePreDamageFunc ports the Perforate family's preDamageFunc
// (act_str.lua L8241, L8337, L8438): overlapping spikes can only raise DPS.
func perforatePreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	skillData.SetN("dpsMultiplier", math.Max(skillDataOr(skillData, "dpsMultiplier", 1), 1))
}

// bladeFlurryPreDamageFunc ports the Blade Flurry pair's preDamageFunc
// (act_dex.lua L1906, L2023): "Channel & Release" (part 2) averages the
// damage of every stage channelled through, then adds the release.
func bladeFlurryPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillData := activeSkill.SkillData
	if activeSkill.SkillPart.V != 2 || !(skillData.N("numStages") > 0) {
		return
	}
	numStages := skillData.N("numStages")
	channelMulti := 0.0
	for i := 1.0; i <= numStages; i++ {
		channelMulti = channelMulti + (0.8 + (0.2 * i))
	}
	channelMulti = channelMulti / (0.8 + (0.2 * numStages))
	skillData.SetN("dpsMultiplier", channelMulti/numStages+1)
}

// sparkPreDamageFunc ports the Spark family's preDamageFunc (act_int.lua
// L16721, L16811, L16902): "Maximum Hits" (part 2) lands one hit per
// 0.66s of the projectile's life.
func sparkPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if activeSkill.SkillPart.V != 2 {
		return
	}
	skillData := activeSkill.SkillData
	skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*(1+math.Floor(output.N("Duration")/0.66)))
	output.SetN("SkillDPSMultiplier", skillData.N("dpsMultiplier"))
}

// ballLightningPreDamageFunc ports Ball Lightning's preDamageFunc
// (act_int.lua L1087): "All Bolts in Range" (part 2) counts the bolt
// strikes that land while the ball travels past the target.
func ballLightningPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillCfg, skillData, skillFlags, skillModList := activeSkill.SkillCfg, activeSkill.SkillData, activeSkill.SkillFlags, activeSkill.SkillModList

	dpsMultiplier := 0.0
	switch activeSkill.SkillPart.V {
	case 1:
		// Compute DPS changes as if we get exactly 1 strike per ball.
		dpsMultiplier = 1
	case 2:
		// Compute DPS changes accounting for all strikes in range.

		// What's the bolt strike proc rate? Note that the interval is not
		// considered to be a cooldown, so it is unaffected by CDR mods.
		secsPerStrike := skillData.N("strikeInterval")
		// How many total bolt strikes proc per ball, ignoring whether the
		// enemy is in range? We assume that the first strike is at the end
		// of the first interval, based on Kitava self-poison testing.
		durationSecs := skillData.N("duration")
		maxStrikes := math.Floor(durationSecs / secsPerStrike)
		// How fast does the ball travel?
		baseBallDistPerSec := skillData.N("projectileSpeed")
		incSpeedMult, moreSpeedMult := Mods(skillModList, skillCfg, "ProjectileSpeed")
		netSpeedMult := incSpeedMult * moreSpeedMult
		ballDistPerSec := baseBallDistPerSec * netSpeedMult
		ballDistPerStrike := ballDistPerSec * secsPerStrike
		// How many times does the ball proc a bolt strike while it is in
		// range of the enemy? (The reference declares an enemy radius of 0
		// here and never reads it.)
		baseStrikeRadius := output.N("AreaOfEffectRadius")
		strikeRadius := baseStrikeRadius
		castDist := 0.0
		switch {
		case skillCfg.SkillDist != nil:
			// Advanced users can specify exactly the standoff distance
			// they'll use against single-target bosses.
			castDist = *skillCfg.SkillDist
		case skillFlags["triggered"]:
			// Cyclone is the most common trigger skill; assume the caster
			// averages one normal bolt strike radius away.
			castDist = math.Floor(baseStrikeRadius / 2)
		default:
			// Be nice and assume hand-casters are at the optimal distance
			// for normal bolt strikes.
			castDist = baseStrikeRadius
		}
		// 1 not 0 here: strike seems to happen at the end of the interval,
		// not start
		firstStrikeIdxThatHits := math.Max(1, math.Ceil((castDist-strikeRadius)/ballDistPerStrike))
		lastStrikeIdxThatHits := math.Floor(math.Min(data.Misc.ProjectileDistanceCap, castDist+strikeRadius) / ballDistPerStrike)
		numStrikes := math.Max(0, math.Min(maxStrikes, lastStrikeIdxThatHits+1-firstStrikeIdxThatHits))

		dpsMultiplier = numStrikes
		// output.NormalHitsPerCast is written only inside the breakdown,
		// which the port never builds.
	}
	if dpsMultiplier != 1 {
		skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*dpsMultiplier)
		output.SetN("SkillDPSMultiplier", outputOr(output, "SkillDPSMultiplier", 1)*dpsMultiplier)
	}
}

// ballLightningAltXPreDamageFunc ports Ball Lightning of Orbiting's
// preDamageFunc (act_int.lua L1252): parts 2 and 3 count every strike of
// the orbiting ball's life.
func ballLightningAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	dpsMultiplier := 1.0
	if part := activeSkill.SkillPart.V; part == 2 || part == 3 {
		skillData := activeSkill.SkillData
		numStrikes := math.Floor(skillData.N("duration") / skillData.N("strikeInterval"))

		dpsMultiplier = numStrikes
		if dpsMultiplier != 1 {
			skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*dpsMultiplier)
			output.SetN("SkillDPSMultiplier", outputOr(output, "SkillDPSMultiplier", 1)*dpsMultiplier)
		}
		output.SetN("NormalHitsPerCast", numStrikes)
	}
}

// ballLightningAltYPreDamageFunc ports Ball Lightning of Static's
// preDamageFunc (act_int.lua L1375): the stationary ball lands every
// strike of its life.
func ballLightningAltYPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	numStrikes := math.Floor(skillData.N("duration") / skillData.N("strikeInterval"))

	skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*numStrikes)
	output.SetN("NormalHitsPerCast", numStrikes)
	output.SetN("SkillDPSMultiplier", outputOr(output, "SkillDPSMultiplier", 1)*numStrikes)
}

// stormBurstRemainingDurationMore is the "Max Duration Explode" part shared
// by both Storm Burst gems: the orb's remaining jumps scale the explosion.
func stormBurstRemainingDurationMore(c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	remainingJumps := math.Floor((skillData.N("duration")*output.N("DurationMod") + 0.001) / skillData.N("repeatFrequency"))
	// Tested in-game and the skill grants more damage only after you have
	// at least 0.8s duration
	activeSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(math.Max(skillData.N("remainingDurationDamage")*remainingJumps-100, 0)), "Skill:StormBurst"))
}

// stormBurstPreDamageFunc ports Storm Burst's preDamageFunc (act_int.lua
// L18036): "Max Channelled Orbs" (part 2) counts the orb's ticks over its
// duration; "Max Duration Explode" (part 3) is the shared explosion.
func stormBurstPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	switch activeSkill.SkillPart.V {
	case 2:
		duration := skillData.N("duration") * output.N("DurationMod")
		// duration * 10 / (jump * 10), instead of duration / jump to avoid
		// floating point issues
		jumpPeriod := skillData.N("repeatFrequency") * 10
		// additional 1 tick upon spawn of orb
		skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*(1+math.Floor(duration*10/jumpPeriod)))
	case 3:
		stormBurstRemainingDurationMore(c)
	}
}

// stormBurstAltXPreDamageFunc ports Storm Burst of Repulsion's
// preDamageFunc (act_int.lua L18151): its part 2 is the explosion.
func stormBurstAltXPreDamageFunc(env *Env, c *offenceCtx) {
	if c.activeSkill.SkillPart.V == 2 {
		stormBurstRemainingDurationMore(c)
	}
}

// flamethrowerTrapPreDamageFunc ports the Flamethrower Trap pair's
// preDamageFunc (act_dex.lua L7688, L7806): the trap's flames tick on a
// fixed interval (slower for the "bad placement" parts 2 and 4) and the
// "Average # traps" parts (3 and 4) multiply by how many traps the cooldown
// keeps active. The Stability gem has no parts: good placement, averaged.
func flamethrowerTrapPreDamageFunc(stability bool) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		skillData := activeSkill.SkillData
		duration := output.N("Duration")
		cooldown := output.N("TrapCooldown")
		averageActiveTraps := duration / cooldown
		output.SetN("AverageActiveTraps", averageActiveTraps)
		part := activeSkill.SkillPart.V
		if !stability && (part == 2 || part == 4) {
			skillData.SetN("hitTimeOverride", 0.3)
		} else {
			skillData.SetN("hitTimeOverride", 0.1)
		}
		if stability || part == 3 || part == 4 {
			skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*averageActiveTraps)
		}
	}
}

// infernalBlowPreDamageFunc ports the Infernal Blow pair's preDamageFunc
// (act_str.lua L7052, L7178): the debuff explosion (parts 2 and 3) deals
// more damage per stack, and the six-stack part fires once per six hits.
func infernalBlowPreDamageFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
		effect := skillModList.Sum(modparser.Base, skillCfg, "DebuffEffect")
		part := activeSkill.SkillPart.V
		if part == 2 || part == 3 {
			skillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(effect), source, modparser.FlagNone, modparser.KeywordNone, &modparser.MultiplierTag{Var: "DebuffStack", Base: opt(-100 + effect)}))
		}
		if part == 3 {
			activeSkill.SkillData.SetN("dpsMultiplier", 1.0/6)
		}
	}
}

// kineticFusilladePreDamageFunc ports the Kinetic Fusillade pair's
// preDamageFunc (act_int.lua L11009, L11227): "All Projectiles" (part 1)
// hits once per projectile, and each projectile deals more damage for
// every one fired before it, folded into one average multiplier.
func kineticFusilladePreDamageFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		if activeSkill.SkillPart.V != 1 {
			return
		}
		skillData := activeSkill.SkillData
		// Set base dpsMultiplier for projectile count
		skillData.SetN("dpsMultiplier", output.N("ProjectileCount"))

		// Calculate average damage scaling for sequential projectiles
		// Each projectile does more damage based on how many came before it
		moreDamagePerProj := skillDataOr(skillData, "damagePerProjectile", 0)
		if moreDamagePerProj != 0 && output.N("ProjectileCount") > 1 {
			// Average multiplier: sum of (0, X, 2X, ..., (n-1)X) / n
			// = X * (n-1)/2
			avgMoreMult := moreDamagePerProj * (output.N("ProjectileCount") - 1) / 2
			activeSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(avgMoreMult), source))

			// Store the average multiplier for display
			output.SetN("KineticFusilladeAvgMoreMult", avgMoreMult)
		}
	}
}

// kineticFusilladePostCritFunc ports the Kinetic Fusillade pair's
// postCritFunc (act_int.lua L11046, L11264): the projectiles hover before
// firing and recasting resets the timer, so attacking faster than the
// hover-plus-fire delay wastes projectiles; "All Projectiles" (part 1)
// scales its multiplier down by that waste.
func kineticFusilladePostCritFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData

	baseDelayBetweenProjectiles := skillData.N("delayPerProjectile")
	projectileCount := 1.0
	if activeSkill.SkillPart.V == 1 {
		projectileCount = output.N("ProjectileCount")
	}

	// Calculate effective attack rate accounting for delayed projectile
	// firing: totalTime = (hoverDelay + delayBetweenProj * nProj) * durationMod
	hoverDelay := skillData.N("duration")
	durationMod := output.N("DurationMod")
	baseTimeForAllProjectiles := baseDelayBetweenProjectiles * (projectileCount - 1)
	effectiveDelay := (hoverDelay + baseTimeForAllProjectiles) * durationMod
	// Testing in game showed playing in Lockstep rounded the duration to
	// server ticks but Predictive did not (the predictive rate is
	// breakdown-only)
	effectiveDelayRounded := math.Ceil(effectiveDelay/data.Misc.ServerTickTime) * data.Misc.ServerTickTime
	maxEffectiveAPS := 1 / effectiveDelayRounded

	output.SetN("KineticFusilladeMaxEffectiveAPS", maxEffectiveAPS)

	// Adjust dpsMultiplier if attacking too fast (only for "All
	// Projectiles" mode); `currentAPS and ...` skips an absent Speed
	if activeSkill.SkillPart.V == 1 && output.Has("Speed") && output.N("Speed") > maxEffectiveAPS {
		efficiencyRatio := maxEffectiveAPS / output.N("Speed")
		originalMultiplier := skillDataOr(skillData, "dpsMultiplier", output.N("ProjectileCount"))
		skillData.SetN("dpsMultiplier", originalMultiplier*efficiencyRatio)
	}
}

// areaHitChance is the trap skills' shared `hitChance` local: not to be
// confused with attack hit chance, it is the share of the spread area in
// which a damaging area can land and still reach the enemy. The -1 assumes
// PoE coordinates are integers and that areas sharing only a point or
// vertex do not register damage.
func areaHitChance(enemyRadius, areaDamageRadius, areaSpreadRadius float64) float64 {
	damagingAreaRadius := areaDamageRadius + enemyRadius - 1
	return math.Min(damagingAreaRadius*damagingAreaRadius/(areaSpreadRadius*areaSpreadRadius), 1)
}

// configuredEnemyRadius is `Override(cfg, "EnemyRadius") or Sum("BASE",
// cfg, "EnemyRadius")`: the config tab's enemy size.
func configuredEnemyRadius(skillModList *modstore.List, skillCfg *modstore.Cfg) float64 {
	if ov, ok := skillModList.Override(skillCfg, "EnemyRadius"); ok {
		return valueNum(ov)
	}
	return skillModList.Sum(modparser.Base, skillCfg, "EnemyRadius")
}

// trapWaveShape is how one member of the Seismic Trap / Lightning Spire
// Trap family differs from the others: whether the targeting radius ignores
// area modifiers (the spire traps), which stats speed the pulses (none
// means a fixed pulse rate and no WavePulseRate output), whether the trap's
// duration is counted in pulses at all (Seismic Trap of Swells never
// pulses), whether the active trap count comes from throw time instead of
// the trap cooldown (Lightning Spire Trap of Zapping), and how many parts
// the gem has.
type trapWaveShape struct {
	radiusFromData bool
	frequencyStats []string
	pulses         bool
	throwTimeTraps bool
	parts          int
}

// trapWavePreDamageFunc ports the trap-wave family's preDamageFunc
// (act_dex.lua L13378 Seismic Trap, L13619 Seismic Trap of Swells;
// act_int.lua L11667 Lightning Spire Trap, L11912 of Zapping, L12150 of
// Overloading): the trap fires waves on its own pulse rate, a wave lands
// close enough to hit with the area overlap chance, and the parts choose
// between one wave, every wave, and every wave of every active trap.
func trapWavePreDamageFunc(shape trapWaveShape) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		skillCfg, skillData, skillModList := activeSkill.SkillCfg, activeSkill.SkillData, activeSkill.SkillModList
		skillPart := activeSkill.SkillPart.V

		if shape.radiusFromData {
			// seemingly the only mechanical difference with seismic trap -
			// this one does not scale its total radius with AoE modifiers
			output.SetN("AreaOfEffectRadius", skillData.N("radius"))
		}

		averageActiveTraps := 0.0
		if shape.pulses {
			baseInterval := skillData.N("repeatInterval")
			wavePulseRate := 1 / baseInterval
			if shape.frequencyStats != nil {
				incFrequency := 1 + skillModList.Sum(modparser.Inc, skillCfg, shape.frequencyStats...)/100
				moreFrequency := skillModList.More(skillCfg, shape.frequencyStats...)
				wavePulseRate = incFrequency * moreFrequency / baseInterval
			}
			skillData.SetN("hitTimeOverride", 1/wavePulseRate)
			if shape.frequencyStats != nil {
				output.SetN("WavePulseRate", wavePulseRate)
			}
			incDuration := 1 + skillModList.Sum(modparser.Inc, skillCfg, "Duration")/100
			moreDuration := skillModList.More(skillCfg, "Duration")
			duration := skillData.N("duration") * incDuration * moreDuration
			pulses := math.Floor(duration * wavePulseRate)
			output.SetN("PulsesPerTrap", pulses)
			effectiveDuration := pulses / wavePulseRate
			if shape.throwTimeTraps {
				actionSpeedMod := 1 + skillModList.Sum(modparser.Inc, skillCfg, "ActionSpeed")/100
				baseSpeed := 1 / skillModList.Sum(modparser.Base, skillCfg, "TrapThrowingTime")
				throwSpeed := baseSpeed * Mod(skillModList, skillCfg, "TrapThrowingSpeed") * actionSpeedMod
				throwSpeed = math.Min(throwSpeed, data.Misc.ServerTickRate)
				throwTime := 1 / throwSpeed
				averageActiveTraps = effectiveDuration / throwTime
			} else {
				averageActiveTraps = effectiveDuration / output.N("TrapCooldown")
			}
			output.SetN("AverageActiveTraps", averageActiveTraps)
		}

		enemyRadius := configuredEnemyRadius(skillModList, skillCfg)
		waveRadius := output.N("AreaOfEffectRadiusSecondary")
		fullRadius := output.N("AreaOfEffectRadius")
		overlapChance := areaHitChance(enemyRadius, waveRadius, fullRadius)
		output.SetN("OverlapChance", overlapChance*100)

		maxWaves := skillModList.Sum(modparser.Base, skillCfg, "MaximumWaves")
		dpsMultiplier := 1.0
		switch {
		case skillPart == 2:
			dpsMultiplier = maxWaves * overlapChance
		case skillPart == 3:
			dpsMultiplier = maxWaves
		case shape.parts < 4:
			// the two-wave gem has no trap-count parts
		case skillPart == 4:
			dpsMultiplier = averageActiveTraps
		case skillPart == 5:
			dpsMultiplier = averageActiveTraps * maxWaves * overlapChance
		case skillPart == 6:
			dpsMultiplier = averageActiveTraps * maxWaves
		}
		if dpsMultiplier != 1 {
			skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*dpsMultiplier)
			output.SetN("SkillDPSMultiplier", outputOr(output, "SkillDPSMultiplier", 1)*dpsMultiplier)
		}
	}
}

// bodyswapPreDamageFunc ports the Bodyswap pair's preDamageFunc
// (act_int.lua L2298, L2413): the corpse explosion (part 1) adds fire from
// the caster's, or the totem's, life.
func bodyswapPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if activeSkill.SkillPart.V != 1 {
		return
	}
	skillData := activeSkill.SkillData
	life := output.N("Life")
	if activeSkill.SkillFlags["totem"] {
		life = output.N("TotemLife")
	}
	skillData.SetN("FireBonusMin", life*skillData.N("selfFireExplosionLifeMultiplier"))
	skillData.SetN("FireBonusMax", life*skillData.N("selfFireExplosionLifeMultiplier"))
}

// deathWishPreDamageFunc ports Death Wish's preDamageFunc (other.lua
// L1154): the minion explosion (part 2) adds fire from the caster's life.
func deathWishPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if activeSkill.SkillPart.V != 2 {
		return
	}
	skillData := activeSkill.SkillData
	skillData.SetN("FireBonusMin", output.N("Life")*skillData.N("selfFireExplosionLifeMultiplier"))
	skillData.SetN("FireBonusMax", output.N("Life")*skillData.N("selfFireExplosionLifeMultiplier"))
}

// darkPactPreDamageFunc ports Dark Bargain's preDamageFunc (act_int.lua
// L3889): the sacrifice adds chaos from the caster's (part 1) or the
// targeted skeleton's life.
func darkPactPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	var life float64
	if activeSkill.SkillPart.V == 1 {
		if activeSkill.SkillFlags["totem"] {
			life = output.N("TotemLife")
		} else {
			life = output.N("Life")
		}
	} else {
		life = skillDataOr(skillData, "skeletonLife", 0)
	}
	add := life * skillData.N("lifeDealtAsChaos") / 100
	skillData.SetN("ChaosMin", skillData.N("ChaosMin")+add)
	skillData.SetN("ChaosMax", skillData.N("ChaosMax")+add)
}

// darkPactAltXPreDamageFunc ports Dark Bargain of Trarthus' preDamageFunc
// (act_int.lua L4006): the sacrifice is a share of the caster's life, paid
// in full on the Ruin part (2) and averaged over the casts until Ruin
// otherwise.
func darkPactAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	life := output.N("Life")
	if activeSkill.SkillFlags["totem"] {
		life = output.N("TotemLife")
	}
	add := life * skillData.N("percentageLifeSacrificed") / 100 * (skillData.N("percentageSacrificedDealtAsChaos") / 100)
	if activeSkill.SkillPart.V == 2 {
		skillData.SetN("ChaosMin", skillData.N("ChaosMin")+math.Floor(add))
		skillData.SetN("ChaosMax", skillData.N("ChaosMax")+math.Floor(add))
	} else {
		avgCastsTillRuin := 7 / (1 + math.Min(100, skillDataOr(skillData, "additionalRuinChance", 0))/100)
		skillData.SetN("ChaosMin", skillData.N("ChaosMin")+math.Floor(add/avgCastsTillRuin))
		skillData.SetN("ChaosMax", skillData.N("ChaosMax")+math.Floor(add/avgCastsTillRuin))
	}
}

// vaalRighteousFirePreDamageFunc ports Vaal Righteous Fire's preDamageFunc
// (act_int.lua L15625): the burn is a share of the sacrificed life and
// energy shield.
func vaalRighteousFirePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	if activeSkill.SkillFlags["totem"] {
		skillData.SetN("FireDot", output.N("TotemLife")*skillData.N("percentSacrificed")*skillData.N("RFMultiplier"))
	} else {
		skillData.SetN("FireDot", (output.N("Life")+output.N("EnergyShield"))*skillData.N("percentSacrificed")*skillData.N("RFMultiplier"))
	}
}

// forbiddenRiteAltXPreDamageFunc ports Forbidden Rite of Soul Sacrifice's
// preDamageFunc (act_int.lua L7549): Forbidden Rite's shape with energy
// shield alone feeding both the hit and the self-damage.
func forbiddenRiteAltXPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	basetakenFlat := skillModList.Sum(modparser.Base, nil, "DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit")
	baseTakenInc := skillModList.Sum(modparser.Inc, nil, "DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit")
	baseTakenMore := skillModList.More(nil, "DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit")
	chaosDamageTaken := math.Max((1+baseTakenInc/100)*baseTakenMore, 0)
	chaosFlat := floorDec(math.Floor(basetakenFlat*chaosDamageTaken+0.5), 0)
	var energyShield, chaosResistance float64
	if activeSkill.SkillFlags["totem"] {
		energyShield = output.N("TotemEnergyShield")
		chaosResistance = output.N("TotemChaosResist")
	} else {
		energyShield = output.N("EnergyShield")
		chaosResistance = output.N("ChaosResist")
	}
	add := energyShield * skillData.N("energyShieldDealtAsChaos")
	selfDamageTakenES := math.Floor(math.Floor(energyShield*skillData.N("SelfDamageTakenES")+0.5) * (100 - chaosResistance) / 100 * chaosDamageTaken)
	skillData.SetN("ChaosMin", skillData.N("ChaosMin")+add)
	skillData.SetN("ChaosMax", skillData.N("ChaosMax")+add)
	if activeSkill.SkillPart.V == 2 {
		skillData.SetN("dpsMultiplier", skillDataOr(skillData, "dpsMultiplier", 1)*(output.N("ProjectileCount")+1))
	}
	output.SetN("FRDamageTaken", selfDamageTakenES+chaosFlat)
}

// manabondPreDamageFunc ports Manabond's preDamageFunc (act_int.lua
// L13527): a share of the missing unreserved mana becomes base lightning
// damage.
func manabondPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	missingUnreservedManaPercentage := skillDataOr(skillData, "ManabondMissingUnreservedManaPercentage", 100)
	manaGainedAsBaseLightningDamage := math.Floor((skillData.N("ManabondMissingManaGainPercent") / 100) * (missingUnreservedManaPercentage / 100) * outputOr(output, "ManaUnreserved", 0))
	activeSkill.SkillModList.AddMod(newModS("LightningMin", modparser.Base, modparser.Num(manaGainedAsBaseLightningDamage), "Manabond gain % missing unreserved mana as base lightning damage"))
	activeSkill.SkillModList.AddMod(newModS("LightningMax", modparser.Base, modparser.Num(manaGainedAsBaseLightningDamage), "Manabond gain % missing unreserved mana as base lightning damage"))
}

// poisonousConcoctionPreDamageFunc ports the Poisonous Concoction pair's
// preDamageFunc (act_dex.lua L17420, L17521): the consumed life flask
// charges add chaos damage. (`Sum(...) or 0`: Sum always returns a
// number, so the fallback is dead.)
func poisonousConcoctionPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	multiplier := activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "ChaosPerLifeFlaskPercent")
	addedFromFlask := outputOr(output, "LifeFlaskRecovery", 0) * (multiplier / 100)
	activeSkill.SkillModList.AddMod(newModS("ChaosMin", modparser.Base, modparser.Num(addedFromFlask), "Life Flask charges consumed"))
	activeSkill.SkillModList.AddMod(newModS("ChaosMax", modparser.Base, modparser.Num(addedFromFlask), "Life Flask charges consumed"))
}

// avengingFlamePreDamageFunc ports Avenging Flame's preDamageFunc
// (sup_str.lua L2539): the projectile adds fire from the life of the totem
// that triggered it, read live from that totem skill's cache entry.
func avengingFlamePreDamageFunc(env *Env, c *offenceCtx) {
	skillData := c.activeSkill.SkillData
	totemLife := 0.0
	if uuid := skillData.Str("triggerSourceUUID"); uuid != "" {
		// `cache and cache.Env.player.output.TotemLife or 0`: an absent
		// entry or an absent value both read 0
		totemLife = env.GlobalCache[uuid].out("TotemLife").Num()
	}
	add := totemLife * skillData.N("lifeDealtAsFire") / 100
	skillData.SetN("FireMax", skillDataOr(skillData, "FireMax", 0)+add)
	skillData.SetN("FireMin", skillDataOr(skillData, "FireMin", 0)+add)
}

// freezingPulsePreDamageFunc ports Freezing Pulse's preDamageFunc
// (act_int.lua L7686): the pulse loses damage and freeze chance with
// distance, both ramps stretched by projectile speed.
func freezingPulsePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	psm := output.N("ProjectileSpeedMod")
	activeSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(-50), "Skill:FreezingPulse", &modparser.DistanceRampTag{Ramp: modparser.Pairs{{0, 0}, {60 * psm, 1}}}))
	activeSkill.SkillModList.AddMod(newModS("EnemyFreezeChance", modparser.Base, modparser.Num(25), "Skill:FreezingPulse", &modparser.DistanceRampTag{Ramp: modparser.Pairs{{0, 1}, {15 * psm, 0}}}))
}

// eyeOfWinterPreDamageFunc ports the Eye of Winter family's preDamageFunc
// (act_int.lua L5518, L5611, L5704): the gem's own ramp stat scales damage
// with distance, stretched by projectile speed.
func eyeOfWinterPreDamageFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		activeSkill.SkillModList.AddMod(newModS("Damage", modparser.More,
			modparser.Num(activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "EyeOfWinterRamp")), source,
			&modparser.DistanceRampTag{Ramp: modparser.Pairs{{0, 0}, {60 * output.N("ProjectileSpeedMod"), 1}}}))
	}
}

// chargedDashPreDamageFunc ports the Charged Dash pair's preDamageFunc
// (act_dex.lua L3939, L4064): the final wave (part 3) deals more damage.
// The reference tags the mod `{ type = "Release Damage", skillPart = 3 }`,
// a tag type EvalMod never tests, so the tag filters nothing and the port
// adds the mod untagged (decision 2026-09-03, later.md).
func chargedDashPreDamageFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		if activeSkill.SkillPart.V != 3 {
			return
		}
		finalWaveDamageModifier := activeSkill.SkillModList.Sum(modparser.Inc, activeSkill.SkillCfg, "chargedDashFinalDamageModifier")
		activeSkill.SkillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(finalWaveDamageModifier), source, modparser.FlagAttack, modparser.KeywordNone))
	}
}

// frostBombAltYPreDamageFunc ports Frost Bomb of Forthcoming's
// preDamageFunc (act_int.lua L7933): the bomb's delay, in 100ms steps,
// feeds its own multiplier.
func frostBombAltYPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	duration := math.Floor(activeSkill.SkillData.N("duration") * output.N("DurationMod") * 10)
	activeSkill.SkillModList.AddMod(newModS("Multiplier:100msFrostBombDuration", modparser.Base, modparser.Num(duration), "Skill:FrostBombAltY"))
}

// waveOfConvictionPreDamageFunc ports the Wave of Conviction pair's
// preDamageFunc (act_int.lua L20795, L20889): the exposure's damage over
// time grows per 100ms of the wave's tick-rounded duration, scaled by how
// much of it has expired.
func waveOfConvictionPreDamageFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		skillData := activeSkill.SkillData
		if !skillData.Flag("duration") {
			return
		}
		duration := math.Floor(math.Ceil(skillData.N("duration")*data.Misc.ServerTickRate) / data.Misc.ServerTickRate * output.N("DurationMod") * 10)
		activeSkill.SkillModList.AddMod(newModSF("DotMultiplier", modparser.Base,
			modparser.Num(activeSkill.SkillModList.Sum(modparser.Inc, activeSkill.SkillCfg, "WaveOfConvictionDurationDotMulti")*duration/100), source,
			modparser.FlagNone, modparser.KeywordNone, &modparser.MultiplierTag{Var: "WoCDurationExpired"}))
	}
}

// banePreSkillTypeFunc ports the Bane pair's preSkillTypeFunc (act_int.lua
// L1470, L1647): the curses linked in Bane's socket group count towards its
// per-curse multiplier, up to the enemy's curse limit.
func banePreSkillTypeFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	curseCount := 0.0
	for _, skill := range c.actor.skills {
		if skill.SocketGroup == activeSkill.SocketGroup && skill.SkillModList.GetCondition("AppliedByBane", nil) {
			curseCount = curseCount + 1
			if curseCount == output.N("EnemyCurseLimit") {
				break
			}
		}
	}
	activeSkill.SkillModList.AddMod(newModS("Multiplier:CurseApplied", modparser.Base, modparser.Num(curseCount), "Base"))
}

// rageVortexPreSkillTypeFunc ports Rage Vortex's preSkillTypeFunc
// (act_str.lua L9019): the sacrifice part (2) spends a share of maximum
// rage, or the configured stacks when fewer.
func rageVortexPreSkillTypeFunc(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	if activeSkill.SkillPart.V != 2 {
		return
	}
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	maxRage := skillModList.Sum(modparser.Base, skillCfg, "MaximumRage")
	rageVortexSacrificePercentage := skillData.N("MaxRageVortexSacrificePercentage") / 100
	configOverride := skillModList.Sum(modparser.Base, skillCfg, "Multiplier:RageSacrificedStacks")
	maxSacrificedRage := math.Floor(rageVortexSacrificePercentage * maxRage)
	stacks := maxSacrificedRage
	if configOverride > 0 {
		stacks = math.Min(configOverride, maxSacrificedRage)
	}
	skillModList.AddMod(newModS("Multiplier:RageSacrificed", modparser.Base, modparser.Num(stacks), "Skill:RageVortex"))
}

// frozenSweepPreDamageFunc ports the Frozen Sweep pair's preDamageFunc
// (act_str.lua L4616, L4734): the sweep fires once per Frozen Legion
// cooldown, so the parent gem in the same slot supplies the cooldown and
// the statue count the "all statues" part (2) waits for.
func frozenSweepPreDamageFunc(parentName string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill, output := c.activeSkill, c.output
		skillData := activeSkill.SkillData
		skillData.SetFlag("showAverage", false)
		activeSkill.SkillFlags["showAverage"] = false
		activeSkill.SkillFlags["notAverage"] = true

		var parentSkill *ActiveSkill
		for _, skill := range c.actor.skills {
			if skill.ActiveEffect.GrantedEffect.Name == parentName && c.actor.mainSkill.SocketGroup.Slot == activeSkill.SocketGroup.Slot {
				parentSkill = skill
				break
			}
		}
		if parentSkill == nil {
			panic("calc: Frozen Sweep with no " + parentName + " in the skill list (the Lua errors)")
		}
		parentModList, parentCfg, parentData := parentSkill.SkillModList, parentSkill.SkillCfg, parentSkill.SkillData
		if parentModList.Flag(parentCfg, "DisableSkill") && !parentModList.Flag(parentCfg, "EnableSkill") {
			return
		}

		skillData.Set("cooldown", parentData.Get("cooldown"))
		var cooldown float64
		if ov, ok := parentModList.Override(parentCfg, "CooldownRecovery"); ok {
			cooldown = valueNum(ov)
		} else {
			cooldown = (parentData.N("cooldown") + parentModList.Sum(modparser.Base, parentCfg, "CooldownRecovery")) / math.Max(0, Mod(parentModList, parentCfg, "CooldownRecovery"))
		}
		output.SetN("Cooldown", math.Ceil(cooldown*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
		skillData.SetN("hitTimeOverride", output.N("Cooldown"))

		maxStatues := parentData.N("storedUses") + parentModList.Sum(modparser.Base, parentCfg, "AdditionalCooldownUses")
		switch activeSkill.SkillPart.V {
		case 1:
			skillData.SetN("averageBurstHits", 1)
		case 2:
			skillData.SetN("averageBurstHits", maxStatues)
		}
	}
}
