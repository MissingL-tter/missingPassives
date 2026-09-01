package config

// Converted once from Modules/ConfigOptions.lua's varList: one apply body
// per configuration option, each citing the reference line it came from.
// The bodies the conversion could not express - the map-affix dropdowns,
// the boss-skill preset and the stationary-condition compatibility branch
// - are hand-ported in apply_hand.go.

import (
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

var generatedApplies = map[Var]func(Value, *Tab){
	// ConfigOptions.lua L155
	"detonateDeadCorpseLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "corpseLife", Value: modparser.Num(n)})
	},
	// ConfigOptions.lua L171
	"conditionMoving": func(_ Value, t *Tab) {
		t.mod("Condition:Moving", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L174
	"conditionFullLife": func(_ Value, t *Tab) {
		t.mod("Condition:FullLife", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L177
	"conditionLowLife": func(_ Value, t *Tab) {
		t.mod("Condition:LowLife", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L180
	"conditionFullMana": func(_ Value, t *Tab) {
		t.mod("Condition:FullMana", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L183
	"conditionLowMana": func(_ Value, t *Tab) {
		t.mod("Condition:LowMana", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L186
	"conditionFullEnergyShield": func(_ Value, t *Tab) {
		t.mod("Condition:FullEnergyShield", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L189
	"conditionLowEnergyShield": func(_ Value, t *Tab) {
		t.mod("Condition:LowEnergyShield", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L192
	"conditionHaveEnergyShield": func(_ Value, t *Tab) {
		t.mod("Condition:HaveEnergyShield", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L195
	"conditionUnbrokenWard": func(_ Value, t *Tab) {
		t.mod("Condition:UnbrokenWard", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L198
	"minionsConditionFullLife": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:FullLife", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L201
	"minionsConditionLowLife": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:LowLife", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L204
	"minionsConditionFullEnergyShield": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:FullEnergyShield", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L207
	"minionsConditionCreatedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:MinionsCreatedRecently", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L212
	"lifeRegenMode": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "AVERAGE" {
			t.mod("Condition:LifeRegenBurstAvg", modparser.Flag, modparser.Bool(true))
		} else if s == "FULL" {
			t.mod("Condition:LifeRegenBurstFull", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L228
	"armourCalculationMode": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "MAX" {
			t.mod("Condition:ArmourMax", modparser.Flag, modparser.Bool(true))
		} else if s == "AVERAGE" {
			t.mod("Condition:ArmourAvg", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L235
	"warcryMode": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "MAX" {
			t.mod("Condition:WarcryMaxHit", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L240
	"pactMode": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "MAX" {
			t.mod("Condition:PactMaxHit", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L245
	"EVBypass": func(_ Value, t *Tab) {
		t.mod("Condition:EVBypass", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L259
	"arcaneCloakUsedRecentlyCheck": func(_ Value, t *Tab) {
		t.mod("Condition:ArcaneCloakUsedRecently", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L263
	"aspectOfTheAvianAviansMight": func(_ Value, t *Tab) {
		t.mod("Condition:AviansMightActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L266
	"aspectOfTheAvianAviansFlight": func(_ Value, t *Tab) {
		t.mod("Condition:AviansFlightActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L270
	"aspectOfTheCatCatsStealth": func(_ Value, t *Tab) {
		t.mod("Condition:CatsStealthActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L273
	"aspectOfTheCatCatsAgility": func(_ Value, t *Tab) {
		t.mod("Condition:CatsAgilityActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L277
	"overrideCrabBarriers": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("CrabBarriers", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L281
	"aspectOfTheSpiderWebStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("ExtraSkillMod", modparser.List, modparser.ModRef{Mod: modparser.NewMod("Multiplier:SpiderWebApplyStack", modparser.Base, modparser.Num(n))}, &modparser.SkillNameTag{SkillName: "Aspect of the Spider"})
		if n > 0 {
			t.mod("Condition:AspectOfTheSpiderActive", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L288
	"bannerPlanted": func(_ Value, t *Tab) {
		t.mod("Condition:BannerPlanted", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L291
	"bannerValour": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ValourStacks", modparser.Base, modparser.Num(n), &modparser.MarkerTag{Marker: modparser.TagIgnoreCond}, &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L295
	"barkskinStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:BarkskinStacks", modparser.Base, modparser.Num(math.Min(n, 10)))
		t.mod("Multiplier:MissingBarkskinStacks", modparser.Base, modparser.Num(math.Max(-n, -10)))
	},
	// ConfigOptions.lua L300
	"Unbound": func(_ Value, t *Tab) {
		t.mod("Condition:Unbound", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L304
	"bladestormInBloodstorm": func(_ Value, t *Tab) {
		t.mod("Condition:BladestormInBloodstorm", modparser.Flag, modparser.Bool(true), &modparser.SkillNameTag{SkillName: "Bladestorm", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L307
	"bladestormInSandstorm": func(_ Value, t *Tab) {
		t.mod("Condition:BladestormInSandstorm", modparser.Flag, modparser.Bool(true), &modparser.SkillNameTag{SkillName: "Bladestorm", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L317
	"bloodSacramentReservationEHP": func(_ Value, t *Tab) {
		t.mod("Condition:BloodSacramentReservationEHP", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L321
	"ActiveBrands": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ConfigActiveBrands", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L324
	"BrandsAttachedToEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ConfigBrandsAttachedToEnemy", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L330
	"BrandsInLastQuarter": func(_ Value, t *Tab) {
		t.mod("Condition:BrandLastQuarter", modparser.Flag, modparser.Bool(true))
		t.mod("Condition:BrandLastHalf", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L334
	"BrandsInLastHalf": func(_ Value, t *Tab) {
		t.mod("Condition:BrandLastHalf", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L338
	"carrionGolemNearbyMinion": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NearbyNonGolemMinion", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L341
	"carrionGolemEqualsChaosGolem": func(_ Value, t *Tab) {
		t.mod("Condition:CarrionEqualChaosGolem", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L344
	"chaosGolemEqualsStoneGolem": func(_ Value, t *Tab) {
		t.mod("Condition:ChaosEqualStoneGolem", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L347
	"cinderflameStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:CinderflameStacks", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L350
	"stoneGolemEqualsCarrionGolem": func(_ Value, t *Tab) {
		t.mod("Condition:StoneEqualCarrionGolem", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L354
	"closeCombatCombatRush": func(_ Value, t *Tab) {
		t.mod("Condition:CombatRushActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L358
	"ColdSnapBypassCD": func(_ Value, t *Tab) {
		t.mod("CooldownRecovery", modparser.Override, modparser.Num(0), &modparser.SkillNameTag{SkillName: "Cold Snap", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L372
	"overrideCruelty": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Cruelty", modparser.Override, modparser.Num(math.Min(n, 40)), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L376
	"channellingCycloneCheck": func(_ Value, t *Tab) {
		t.mod("Condition:ChannellingCyclone", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L380
	"darkPactSkeletonLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "skeletonLife", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Dark Bargain"})
	},
	// ConfigOptions.lua L384
	"divineSentinelPhysAsFire": func(_ Value, t *Tab) {
		t.mod("Condition:DivineSentinelPhysAsFire", modparser.Flag, modparser.Bool(true))
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:DivineSentinelPhysAsFire", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L388
	"divineSentinelPhysAsLightning": func(_ Value, t *Tab) {
		t.mod("Condition:DivineSentinelPhysAsLightning", modparser.Flag, modparser.Bool(true))
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:DivineSentinelPhysAsLightning", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L392
	"divineSentinelRegenLife": func(_ Value, t *Tab) {
		t.mod("Condition:DivineSentinelRegenLife", modparser.Flag, modparser.Bool(true))
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:DivineSentinelRegenLife", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L396
	"divineSentinelRegenMana": func(_ Value, t *Tab) {
		t.mod("Condition:DivineSentinelRegenMana", modparser.Flag, modparser.Bool(true))
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:DivineSentinelRegenMana", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L400
	"divineSentinelChaosResistance": func(_ Value, t *Tab) {
		t.mod("Condition:DivineSentinelChaosResistance", modparser.Flag, modparser.Bool(true))
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:DivineSentinelChaosResistance", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L404
	"divineSentinelSelfCurseEffect": func(_ Value, t *Tab) {
		t.mod("Condition:DivineSentinelSelfCurseEffect", modparser.Flag, modparser.Bool(true))
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:DivineSentinelSelfCurseEffect", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L410
	"curseOverlaps": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:CurseOverlaps", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L414
	"elementalArmyExposureType": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "Fire" {
			t.mod("FireExposureChance", modparser.Base, modparser.Num(100))
		} else if s == "Cold" {
			t.mod("ColdExposureChance", modparser.Base, modparser.Num(100))
		} else if s == "Lightning" {
			t.mod("LightningExposureChance", modparser.Base, modparser.Num(100))
		}
	},
	// ConfigOptions.lua L424
	"embraceMadnessActive": func(_ Value, t *Tab) {
		t.mod("Condition:AffectedByGloriousMadness", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L437
	"feedingFrenzyFeedingFrenzyActive": func(_ Value, t *Tab) {
		t.mod("Condition:FeedingFrenzyActive", modparser.Flag, modparser.Bool(true))
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Damage", modparser.More, modparser.Num(10), "Feeding Frenzy", true, modparser.FlagNone, modparser.KeywordNone)})
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("MovementSpeed", modparser.Inc, modparser.Num(10), "Feeding Frenzy", true, modparser.FlagNone, modparser.KeywordNone)})
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Speed", modparser.Inc, modparser.Num(10), "Feeding Frenzy", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L444
	"flameWallAddedDamage": func(_ Value, t *Tab) {
		t.mod("Condition:FlameWallAddedDamage", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L452
	"freshMeatBuffs": func(_ Value, t *Tab) {
		t.mod("Condition:FreshMeatActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L456
	"frostShieldStages": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:FrostShieldStage", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L460
	"Disgorged": func(_ Value, t *Tab) {
		t.mod("Condition:Disgorged", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L464
	"greaterHarbingerOfTimeSlipstream": func(_ Value, t *Tab) {
		t.mod("Condition:GreaterHarbingerOfTime", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L468
	"harbingerOfTimeSlipstream": func(_ Value, t *Tab) {
		t.mod("Condition:HarbingerOfTime", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L472
	"multiplierHexDoom": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:HexDoomStack", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L476
	"heraldOfAgonyVirulenceStack": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:VirulenceStack", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L480
	"hoaOverkill": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "hoaOverkill", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Herald of Ash"})
	},
	// ConfigOptions.lua L484
	"heraldOfTheHivePressure": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:OtherworldlyPressure", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L488
	"inventionMineTrapPlacedDuration": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:PlacedDuration", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L492
	"iceNovaCastOnFrostbolt": func(_ Value, t *Tab) {
		t.mod("Condition:CastOnFrostbolt", modparser.Flag, modparser.Bool(true), &modparser.SkillNameTag{SkillName: "Ice Nova of Frostbolts"})
	},
	// ConfigOptions.lua L496
	"infusedChannellingInfusion": func(_ Value, t *Tab) {
		t.mod("Condition:InfusionActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L500
	"innervateInnervation": func(_ Value, t *Tab) {
		t.mod("Condition:InnervationActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L504
	"intensifyIntensity": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:Intensity", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L507
	"OverloadedIntensity": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:OverloadedIntensity", modparser.Base, modparser.Num(math.Min(n, 3)))
	},
	// ConfigOptions.lua L511
	"multiplierLinkedTargets": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:LinkedTargets", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L514
	"linkedToMinion": func(_ Value, t *Tab) {
		t.mod("Condition:LinkedToMinion", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L517
	"linkedSourceRate": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("IntuitiveLinkSourceRate", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L521
	"manabondMissingUnreservedManaPercentage": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "ManabondMissingUnreservedManaPercentage", Value: modparser.Num(math.Max(math.Min(n, 100), 0))}, &modparser.SkillNameTag{SkillName: "Manabond"})
	},
	// ConfigOptions.lua L525
	"minionPactLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:MinionLife", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L529
	"conditionEnemyMalignantMadness": func(_ Value, t *Tab) {
		t.enemyMod("Condition:MalignantMadness", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L533
	"meatShieldEnemyNearYou": func(_ Value, t *Tab) {
		t.mod("Condition:MeatShieldEnemyNearYou", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L537
	"enemyHitMistyReflection": func(_ Value, t *Tab) {
		t.enemyMod("Condition:MistyReflection", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L541
	"MomentumStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:MomentumStacks", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L544
	"MomentumSwiftnessStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:MomentumStacksRemoved", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L556
	"perforateSpikeOverlap": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:PerforateSpikeOverlap", modparser.Base, modparser.Num(n), &modparser.SkillNameTag{SkillName: "Perforate", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L560
	"physicalAegisDepleted": func(_ Value, t *Tab) {
		t.mod("Condition:PhysicalAegisDepleted", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L564
	"deathmarkDeathmarkActive": func(_ Value, t *Tab) {
		t.mod("Condition:EnemyHasDeathmark", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L568
	"prideEffect": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "MAX" {
			t.mod("Condition:PrideMaxEffect", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L574
	"sacrificedRageCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:RageSacrificedStacks", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L585
	"conditionSummonedSpectreInPast8Sec": func(_ Value, t *Tab) {
		t.mod("Condition:SummonedSpectreInPast8Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L588
	"raiseSpectreBladeVortexBladeCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "dpsMultiplier", Value: modparser.Num(n)}, &modparser.SkillIDTag{SkillID: "DemonModularBladeVortexSpectre"})
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "dpsMultiplier", Value: modparser.Num(n)}, &modparser.SkillIDTag{SkillID: "GhostPirateBladeVortexSpectre"})
	},
	// ConfigOptions.lua L592
	"raiseSpectreKaomFireBeamTotemStage": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:KaomFireBeamTotemStage", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L595
	"raiseSpectreEnableSummonedUrsaRallyingCry": func(_ Value, t *Tab) {
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "enable", Value: modparser.Bool(true)}, &modparser.SkillIDTag{SkillID: "DropBearSummonedRallyingCry"})
	},
	// ConfigOptions.lua L598
	"raiseSpectreEnableSlashingHorrorEnrage": func(_ Value, t *Tab) {
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "enable", Value: modparser.Bool(false)}, &modparser.SkillIDTag{SkillID: "AzmeriDualStrikeDemonFireEnrage"})
	},
	// ConfigOptions.lua L601
	"raiseSpectreEnableSanguimancerDemonLowLife": func(_ Value, t *Tab) {
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "enable", Value: modparser.Bool(false)}, &modparser.SkillIDTag{SkillID: "ABTTAzmeriShepherdSpellDamage"})
	},
	// ConfigOptions.lua L605
	"raiseSpidersSpiderCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:RaisedSpiderConfig", modparser.Base, modparser.Num(n))
		t.mod("Multiplier:RaisedSpider", modparser.Base, modparser.Num(1), &modparser.MultiplierTag{Var: "RaisedSpiderConfig", LimitStat: "ActiveSpiderLimit"})
	},
	// ConfigOptions.lua L610
	"conditionSummonedZombieInPast8Sec": func(_ Value, t *Tab) {
		t.mod("Condition:SummonedZombieInPast8Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L613
	"animateWeaponLingeringBlade": func(_ Value, t *Tab) {
		t.mod("Condition:AnimatingLingeringBlades", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L617
	"returningProjectileHits": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ReturningProjectileHits", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L621
	"ShrapnelBallistaProjectileOverlap": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "ShrapnelBallistaProjectileOverlap", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Shrapnel Ballista", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L629
	"siphoningTrapAffectedEnemies": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:EnemyAffectedBySiphoningTrap", modparser.Base, modparser.Num(n))
		t.mod("Condition:SiphoningTrapSiphoning", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L634
	"configSnipeStages": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SnipeStage", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L643
	"configSpectralWolfCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SpectralWolfConfig", modparser.Base, modparser.Num(n))
		t.mod("Multiplier:SpectralWolfCount", modparser.Base, modparser.Num(1), &modparser.MultiplierTag{Var: "SpectralWolfConfig", LimitStat: "ActiveWolfLimit"})
	},
	// ConfigOptions.lua L648
	"bloodSandStance": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "SAND" {
			t.mod("Condition:SandStance", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L653
	"changedStance": func(_ Value, t *Tab) {
		t.mod("Condition:ChangedStanceRecently", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L657
	"shardsConsumed": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SteelShardConsumed", modparser.Base, modparser.Num(math.Min(n, 12)))
	},
	// ConfigOptions.lua L660
	"steelWards": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SteelWardCount", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L664
	"stormRainBeamOverlap": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "beamOverlapMultiplier", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Storm Rain"})
	},
	// ConfigOptions.lua L668
	"stormRainActiveArrows": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "activeArrowMultiplier", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Storm Rain of the Conduit"})
	},
	// ConfigOptions.lua L682
	"summonHolyRelicEnableHolyRelicBoon": func(_ Value, t *Tab) {
		t.mod("Condition:HolyRelicBoonActive", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L686
	"summonLightningGolemEnableWrath": func(_ Value, t *Tab) {
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "enable", Value: modparser.Bool(true)}, &modparser.SkillIDTag{SkillID: "LightningGolemWrath"})
	},
	// ConfigOptions.lua L690
	"summonReaperConsumeRecently": func(_ Value, t *Tab) {
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "enable", Value: modparser.Bool(true)}, &modparser.SkillIDTag{SkillID: "ReaperConsumeMinionForBuff"})
	},
	// ConfigOptions.lua L693
	"weepingBlackStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:WeepingBlackStacks", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L697
	"nearbyBleedingEnemies": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NearbyBleedingEnemies", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L701
	"tornadoShotSecondaryHitChance": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "tornadoShotSecondaryHitChance", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Tornado Shot"})
	},
	// ConfigOptions.lua L705
	"toxicRainPodOverlap": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "podOverlapMultiplier", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Toxic Rain", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L709
	"traumaStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:TraumaStacks", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L713
	"configResonanceCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ResonanceCount", modparser.Base, modparser.Num(math.Max(math.Min(n, 50), 0)))
	},
	// ConfigOptions.lua L717
	"configUnholyResonanceCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:UnholyResonanceCount", modparser.Base, modparser.Num(math.Max(math.Min(n, 50), 0)))
	},
	// ConfigOptions.lua L721
	"conditionInsane": func(_ Value, t *Tab) {
		t.mod("Condition:Insane", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L729
	"voltaxicBurstSpellsQueued": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:VoltaxicWaitingStages", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L733
	"vortexCastOnFrostbolt": func(_ Value, t *Tab) {
		t.mod("Condition:CastOnFrostbolt", modparser.Flag, modparser.Bool(true), &modparser.SkillNameTag{SkillName: "Vortex of Projection"})
	},
	// ConfigOptions.lua L737
	"multiplierWarcryPower": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("WarcryPower", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L741
	"waveOfConvictionExposureType": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "Fire" {
			t.mod("Condition:WaveOfConvictionFireExposureActive", modparser.Flag, modparser.Bool(true))
		} else if s == "Cold" {
			t.mod("Condition:WaveOfConvictionColdExposureActive", modparser.Flag, modparser.Bool(true))
		} else if s == "Lightning" {
			t.mod("Condition:WaveOfConvictionLightningExposureActive", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L750
	"multiplierWoCExpiredDuration": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:WoCDurationExpired", modparser.Base, modparser.Num(math.Min(n, 100)), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L753
	"absolutionSkillDamageCountedOnce": func(_ Value, t *Tab) {
		t.mod("Condition:AbsolutionSkillDamageCountedOnce", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L756
	"dominatingBlowSkillDamageCountedOnce": func(_ Value, t *Tab) {
		t.mod("Condition:DominatingBlowSkillDamageCountedOnce", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L759
	"holyStrikeSkillDamageCountedOnce": func(_ Value, t *Tab) {
		t.mod("Condition:HolyStrikeSkillDamageCountedOnce", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L763
	"MoltenShellDamageMitigated": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "MoltenShellDamageMitigated", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Molten Shell"})
	},
	// ConfigOptions.lua L767
	"VaalMoltenShellDamageMitigated": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SkillData", modparser.List, modparser.DataRef{Key: "VaalMoltenShellDamageMitigated", Value: modparser.Num(n)}, &modparser.SkillNameTag{SkillName: "Molten Shell"})
	},
	// ConfigOptions.lua L796
	"enemyRadius": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("EnemyRadius", modparser.Override, modparser.Num(math.Max(n, 1)))
	},
	// ConfigOptions.lua L799
	"TotalMinionLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalMinionLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L802
	"TotalSpectreLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalSpectreLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L805
	"TotalTotemLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalTotemLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L808
	"TotalRadianceSentinelLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalRadianceSentinelLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L811
	"TotalVoidSpawnLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalVoidSpawnLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L814
	"TotalStoneGolemLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalStoneGolemLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L817
	"TotalVaalRejuvenationTotemLife": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotalVaalRejuvenationTotemLife", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L832
	"multiplierSextant": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:Sextant", modparser.Base, modparser.Num(math.Min(n, 5)))
	},
	// ConfigOptions.lua L848
	"PvpScaling": func(_ Value, t *Tab) {
		t.mod("HasPvpScaling", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L852
	"playerCursedWithAssassinsMark": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "AssassinsMark", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L855
	"playerCursedWithConductivity": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Conductivity", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L858
	"playerCursedWithDespair": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Despair", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L861
	"playerCursedWithElementalWeakness": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "ElementalWeakness", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L864
	"playerCursedWithEnfeeble": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Enfeeble", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L867
	"playerCursedWithFlammability": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Flammability", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L870
	"playerCursedWithFrostbite": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Frostbite", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L873
	"playerCursedWithPoachersMark": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "PoachersMark", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L876
	"playerCursedWithProjectileWeakness": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "ProjectileWeakness", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L879
	"playerCursedWithPunishment": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Punishment", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L882
	"playerCursedWithTemporalChains": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "TemporalChains", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L885
	"playerCursedWithVulnerability": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "Vulnerability", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L888
	"playerCursedWithWarlordsMark": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modNS("ExtraCurse", modparser.List, modparser.SkillRef{SkillID: "WarlordsMark", Level: util.Some[float64](n), ApplyToPlayer: true})
	},
	// ConfigOptions.lua L894
	"usePowerCharges": func(_ Value, t *Tab) {
		t.mod("UsePowerCharges", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L897
	"overridePowerCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("PowerCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L900
	"useFrenzyCharges": func(_ Value, t *Tab) {
		t.mod("UseFrenzyCharges", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L903
	"overrideFrenzyCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("FrenzyCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L906
	"useEnduranceCharges": func(_ Value, t *Tab) {
		t.mod("UseEnduranceCharges", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L909
	"overrideEnduranceCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("EnduranceCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L912
	"useSiphoningCharges": func(_ Value, t *Tab) {
		t.mod("UseSiphoningCharges", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L915
	"overrideSiphoningCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SiphoningCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L918
	"useChallengerCharges": func(_ Value, t *Tab) {
		t.mod("UseChallengerCharges", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L921
	"overrideChallengerCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("ChallengerCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L924
	"useBlitzCharges": func(_ Value, t *Tab) {
		t.mod("UseBlitzCharges", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L927
	"overrideBlitzCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("BlitzCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L930
	"multiplierGaleForce": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:GaleForce", modparser.Base, modparser.Num(n), &modparser.MarkerTag{Marker: modparser.TagIgnoreCond}, &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanGainGaleForce"})
	},
	// ConfigOptions.lua L933
	"overrideInspirationCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("InspirationCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L936
	"useGhostShrouds": func(_ Value, t *Tab) {
		t.mod("UseGhostShrouds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L939
	"overrideGhostShrouds": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("GhostShrouds", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L942
	"waitForMaxSeals": func(_ Value, t *Tab) {
		t.mod("UseMaxUnleash", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L961
	"overrideBloodCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("BloodCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L964
	"overrideSpiritCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SpiritCharges", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L967
	"overrideSpiritInfusion": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("SpiritInfusion", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L970
	"minionsUsePowerCharges": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("UsePowerCharges", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L973
	"minionsUseFrenzyCharges": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("UseFrenzyCharges", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L976
	"minionsUseEnduranceCharges": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("UseEnduranceCharges", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L979
	"minionsOverridePowerCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("PowerCharges", modparser.Override, modparser.Num(n), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L982
	"minionsOverrideFrenzyCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("FrenzyCharges", modparser.Override, modparser.Num(n), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L985
	"minionsOverrideEnduranceCharges": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("EnduranceCharges", modparser.Override, modparser.Num(n), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L988
	"multiplierRampage": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:Rampage", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L991
	"multiplierSoulEater": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SoulEaterStack", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L994
	"multiplierMinionSoulEater": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Multiplier:SoulEaterStack", modparser.Base, modparser.Num(n), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L997
	"conditionFocused": func(_ Value, t *Tab) {
		t.mod("Condition:Focused", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1000
	"buffLifetap": func(_ Value, t *Tab) {
		t.mod("Condition:Lifetap", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.modSrc("FlaskLifeRecovery", modparser.Inc, modparser.Num(20), "Lifetap")
	},
	// ConfigOptions.lua L1004
	"buffOnslaught": func(_ Value, t *Tab) {
		t.mod("Condition:Onslaught", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1007
	"buffArcaneSurge": func(_ Value, t *Tab) {
		t.mod("Condition:ArcaneSurge", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1010
	"minionBuffOnslaught": func(_ Value, t *Tab) {
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:Onslaught", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L1013
	"buffUnholyMight": func(_ Value, t *Tab) {
		t.mod("Condition:UnholyMight", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.modSrc("Condition:CanWither", modparser.Flag, modparser.Bool(true), "Unholy Might", &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1017
	"minionbuffUnholyMight": func(_ Value, t *Tab) {
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:UnholyMight", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:CanWither", modparser.Flag, modparser.Bool(true), "Unholy Might", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L1021
	"buffChaoticMight": func(_ Value, t *Tab) {
		t.mod("Condition:ChaoticMight", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1024
	"buffSacrificialZeal": func(_ Value, t *Tab) {
		t.mod("Condition:SacrificialZeal", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1027
	"minionbuffChaoticMight": func(_ Value, t *Tab) {
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:ChaoticMight", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L1030
	"buffPhasing": func(_ Value, t *Tab) {
		t.mod("Condition:Phasing", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1033
	"buffFortification": func(_ Value, t *Tab) {
		t.mod("Condition:Fortified", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1036
	"overrideFortification": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("FortificationStacks", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1039
	"buffTailwind": func(_ Value, t *Tab) {
		t.mod("Condition:Tailwind", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1042
	"buffAdrenaline": func(_ Value, t *Tab) {
		t.mod("Condition:Adrenaline", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1045
	"conditionChangedStanceLastSecond": func(_ Value, t *Tab) {
		t.mod("Condition:StanceChangeLastSecond", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1048
	"buffAlchemistsGenius": func(_ Value, t *Tab) {
		t.mod("Condition:AlchemistsGenius", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanHaveAlchemistGenius"})
	},
	// ConfigOptions.lua L1051
	"buffVaalArcLuckyHits": func(_ Value, t *Tab) {
		t.mod("LuckyHits", modparser.Flag, modparser.Bool(true), &modparser.CondTag{VarList: []string{"Combat", "CanBeLucky"}}, &modparser.SkillNameTag{SkillName: "Arc", IncludeTransfigured: true})
	},
	// ConfigOptions.lua L1054
	"buffElusive": func(_ Value, t *Tab) {
		t.mod("Condition:Elusive", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanBeElusive"})
		t.mod("Elusive", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanBeElusive"})
	},
	// ConfigOptions.lua L1058
	"overrideBuffElusive": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("ElusiveEffect", modparser.Override, modparser.Num(n), &modparser.GlobalEffectTag{EffectType: "Buff"})
	},
	// ConfigOptions.lua L1061
	"buffDivinity": func(_ Value, t *Tab) {
		t.mod("Condition:Divinity", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1064
	"multiplierDefiance": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:Defiance", modparser.Base, modparser.Num(math.Min(n, 10)), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1067
	"multiplierRage": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:RageStack", modparser.Base, modparser.Num(n), &modparser.MarkerTag{Marker: modparser.TagIgnoreCond}, &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanGainRage"})
	},
	// ConfigOptions.lua L1070
	"buffWildSavagery": func(_ Value, t *Tab) {
		t.mod("Condition:WildSavagery", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1073
	"conditionLeeching": func(_ Value, t *Tab) {
		t.mod("Condition:Leeching", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1076
	"conditionLeechingLife": func(_ Value, t *Tab) {
		t.mod("Condition:LeechingLife", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Leeching", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1080
	"conditionLeechingEnergyShield": func(_ Value, t *Tab) {
		t.mod("Condition:LeechingEnergyShield", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Leeching", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1084
	"conditionLeechingMana": func(_ Value, t *Tab) {
		t.mod("Condition:LeechingMana", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Leeching", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1088
	"minionsConditionLeechingEnergyShield": func(_ Value, t *Tab) {
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:LeechingEnergyShield", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
		t.mod("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:Leeching", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone)})
	},
	// ConfigOptions.lua L1092
	"conditionUsingFlask": func(_ Value, t *Tab) {
		t.mod("Condition:UsingFlask", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1095
	"conditionUsedAmethystFlaskRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedAmethystFlaskRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "HaveAmethystFlask"})
	},
	// ConfigOptions.lua L1098
	"conditionUsedRubyFlaskRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedRubyFlaskRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "HaveRubyFlask"})
	},
	// ConfigOptions.lua L1101
	"conditionUsedSapphireFlaskRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedSapphireFlaskRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "HaveSapphireFlask"})
	},
	// ConfigOptions.lua L1104
	"conditionUsedTopazFlaskRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedTopazFlaskRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "HaveTopazFlask"})
	},
	// ConfigOptions.lua L1107
	"conditionUsingTincture": func(_ Value, t *Tab) {
		t.mod("Condition:UsingTincture", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1111
	"conditionHaveTotem": func(_ Value, t *Tab) {
		t.mod("Condition:HaveTotem", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1114
	"conditionSummonedTotemRecently": func(_ Value, t *Tab) {
		t.mod("Condition:SummonedTotemRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1117
	"TotemsSummoned": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TotemsSummoned", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:HaveTotem", modparser.Flag, modparser.Bool(n >= 1), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1121
	"conditionSummonedGolemInPast8Sec": func(_ Value, t *Tab) {
		t.mod("Condition:SummonedGolemInPast8Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1124
	"conditionSummonedGolemInPast10Sec": func(_ Value, t *Tab) {
		t.mod("Condition:SummonedGolemInPast10Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1127
	"multiplierNearbyAlly": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NearbyAlly", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1130
	"multiplierNearbyCorpse": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NearbyCorpse", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1133
	"multiplierSummonedMinion": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SummonedMinion", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1136
	"multiplierNonVaalSummonedMinion": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NonVaalSummonedMinion", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1139
	"multiplierPermanentMinion": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:PermanentMinion", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1142
	"conditionOnConsecratedGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnConsecratedGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:OnConsecratedGround", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L1146
	"conditionOnProfaneGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnProfaneGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1149
	"minionConditionOnProfaneGround": func(_ Value, t *Tab) {
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:OnProfaneGround", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
	},
	// ConfigOptions.lua L1152
	"conditionOnBrineGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnBrineGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("PhysicalDamageGainAsCold", modparser.Base, modparser.Num(10), &modparser.CondTag{Var: "OnBrineGround"})
		t.mod("PhysicalDamageGainAsLightning", modparser.Base, modparser.Num(10), &modparser.CondTag{Var: "OnBrineGround"})
	},
	// ConfigOptions.lua L1157
	"minionConditionOnBrineGround": func(_ Value, t *Tab) {
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("Condition:OnBrineGround", modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "Combat"})})
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("PhysicalDamageGainAsCold", modparser.Base, modparser.Num(10), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "OnBrineGround"})})
		t.modNS("MinionModifier", modparser.List, modparser.ModRef{Mod: modparser.NewModFull("PhysicalDamageGainAsLightning", modparser.Base, modparser.Num(10), "Config", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "OnBrineGround"})})
	},
	// ConfigOptions.lua L1162
	"conditionOnCausticGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnCausticGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1165
	"conditionOnFungalGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnFungalGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1168
	"conditionOnBurningGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnBurningGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Burning", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1172
	"conditionOnChilledGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnChilledGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Chilled", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1176
	"conditionOnShockedGround": func(_ Value, t *Tab) {
		t.mod("Condition:OnShockedGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Shocked", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1180
	"conditionBlinded": func(_ Value, t *Tab) {
		t.mod("Condition:Blinded", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1183
	"conditionBurning": func(_ Value, t *Tab) {
		t.mod("Condition:Burning", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1186
	"conditionIgnited": func(_ Value, t *Tab) {
		t.mod("Condition:Ignited", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1189
	"conditionScorched": func(_ Value, t *Tab) {
		t.mod("Condition:Scorched", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1192
	"conditionChilled": func(_ Value, t *Tab) {
		t.mod("Condition:Chilled", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1195
	"conditionChilledEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modSrc("ChillVal", modparser.Override, modparser.Num(n), "Chill", &modparser.CondTag{Var: "Chilled"})
	},
	// ConfigOptions.lua L1198
	"conditionFrozen": func(_ Value, t *Tab) {
		t.mod("Condition:Frozen", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1201
	"conditionBrittle": func(_ Value, t *Tab) {
		t.mod("Condition:Brittle", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1204
	"conditionShocked": func(_ Value, t *Tab) {
		t.mod("Condition:Shocked", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1207
	"conditionPlayerShockEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.modSrc("ShockVal", modparser.Override, modparser.Num(n), "Shock", &modparser.CondTag{Var: "Shocked"})
	},
	// ConfigOptions.lua L1210
	"conditionSapped": func(_ Value, t *Tab) {
		t.mod("Condition:Sapped", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1213
	"conditionBleeding": func(_ Value, t *Tab) {
		t.mod("Condition:Bleeding", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1216
	"conditionPoisoned": func(_ Value, t *Tab) {
		t.mod("Condition:Poisoned", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1219
	"conditionCanBeCurseImmune": func(_ Value, t *Tab) {
		t.mod("AvoidCurse", modparser.Base, modparser.Num(100), &modparser.CondTag{Var: "Combat"}, &modparser.GlobalEffectTag{EffectType: "Global", Unscalable: true})
	},
	// ConfigOptions.lua L1222
	"multiplierPoisonOnSelf": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:PoisonStack", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1228
	"multiplierNearbyEnemies": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NearbyEnemies", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:OnlyOneNearbyEnemy", modparser.Flag, modparser.Bool(n == 1), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1232
	"multiplierNearbyRareOrUniqueEnemies": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NearbyRareOrUniqueEnemies", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Multiplier:NearbyEnemies", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:AtMostOneNearbyRareOrUniqueEnemy", modparser.Flag, modparser.Bool(n <= 1), &modparser.CondTag{Var: "Combat"})
		t.enemyMod("Condition:NearbyRareOrUniqueEnemy", modparser.Flag, modparser.Bool(n >= 1), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1238
	"conditionHitRecently": func(_ Value, t *Tab) {
		t.mod("Condition:HitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1241
	"conditionHitSpellRecently": func(_ Value, t *Tab) {
		t.mod("Condition:HitSpellRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:HitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1245
	"conditionCritRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CritRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:SkillCritRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1249
	"conditionSkillCritRecently": func(_ Value, t *Tab) {
		t.mod("Condition:SkillCritRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1252
	"conditionCritWithHeraldSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CritWithHeraldSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1255
	"multiplierCritRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:CritRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:SkillCritRecently", modparser.Flag, modparser.Bool(1 <= n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1259
	"LostNonVaalBuffRecently": func(_ Value, t *Tab) {
		t.mod("Condition:LostNonVaalBuffRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1262
	"conditionNonCritRecently": func(_ Value, t *Tab) {
		t.mod("Condition:NonCritRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1265
	"conditionChannelling": func(_ Value, t *Tab) {
		t.mod("Condition:Channelling", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1268
	"multiplierChannelling": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ChannellingTime", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:Channelling", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1272
	"conditionHitRecentlyWithWeapon": func(_ Value, t *Tab) {
		t.mod("Condition:HitRecentlyWithWeapon", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1275
	"conditionKilledRecently": func(_ Value, t *Tab) {
		t.mod("Condition:KilledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1278
	"multiplierKilledRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:EnemyKilledRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:KilledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1282
	"conditionKilledLast3Seconds": func(_ Value, t *Tab) {
		t.mod("Condition:KilledLast3Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1285
	"conditionKilledPoisonedLast2Seconds": func(_ Value, t *Tab) {
		t.mod("Condition:KilledPoisonedLast2Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1288
	"conditionKilledTauntedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:KilledTauntedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1291
	"conditionTotemsNotSummonedInPastTwoSeconds": func(_ Value, t *Tab) {
		t.mod("Condition:NoSummonedTotemsInPastTwoSeconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1294
	"conditionTotemsKilledRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TotemsKilledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1297
	"conditionTotemsHitRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TotemsHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1300
	"conditionTotemsHitSpellRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TotemsHitSpellRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:TotemsHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1304
	"conditionUsedBrandRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedBrandRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1307
	"multiplierTotemsKilledRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:EnemyKilledByTotemsRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:TotemsKilledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1311
	"conditionMinionsKilledRecently": func(_ Value, t *Tab) {
		t.mod("Condition:MinionsKilledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1314
	"conditionMinionsDiedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:MinionsDiedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1317
	"multiplierMinionsKilledRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:EnemyKilledByMinionsRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:MinionsKilledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1321
	"conditionKilledAffectedByDoT": func(_ Value, t *Tab) {
		t.mod("Condition:KilledAffectedByDotRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1324
	"multiplierShockedEnemyKilledRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ShockedEnemyKilledRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1327
	"multiplierShockedNonShockedEnemyRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ShockedNonShockedEnemyRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1330
	"conditionFrozenEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:FrozenEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1333
	"conditionChilledEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:ChilledEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1336
	"conditionShatteredEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:ShatteredEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1339
	"conditionIgnitedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:IgnitedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1342
	"multiplierIgniteAppliedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:IgniteAppliedRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1345
	"conditionShockedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:ShockedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1348
	"conditionStunnedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:StunnedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1351
	"conditionStunnedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:StunnedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1354
	"conditionPoisonedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:PoisonedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1357
	"multiplierPoisonAppliedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:PoisonAppliedRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1360
	"multiplierLifeSpentRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:LifeSpentRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1363
	"multiplierManaSpentRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:ManaSpentRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1366
	"conditionWardBrokenPast2Seconds": func(_ Value, t *Tab) {
		t.mod("Condition:WardBrokenPast2Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1369
	"conditionBeenHitRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1372
	"multiplierBeenHitRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:BeenHitRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(1 <= n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1376
	"conditionBeenHitByAttackRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BeenHitByAttackRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1379
	"conditionBeenCritRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BeenCritRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1382
	"conditionConsumed12SteelShardsRecently": func(_ Value, t *Tab) {
		t.mod("Condition:Consumed12SteelShardsRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1385
	"conditionGainedPowerChargeRecently": func(_ Value, t *Tab) {
		t.mod("Condition:GainedPowerChargeRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1388
	"conditionGainedFrenzyChargeRecently": func(_ Value, t *Tab) {
		t.mod("Condition:GainedFrenzyChargeRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1391
	"conditionBeenSavageHitRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BeenSavageHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1395
	"conditionHitByFireDamageRecently": func(_ Value, t *Tab) {
		t.mod("Condition:HitByFireDamageRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1399
	"conditionHitByColdDamageRecently": func(_ Value, t *Tab) {
		t.mod("Condition:HitByColdDamageRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1403
	"conditionHitByLightningDamageRecently": func(_ Value, t *Tab) {
		t.mod("Condition:HitByLightningDamageRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1407
	"conditionHitBySpellDamageRecently": func(_ Value, t *Tab) {
		t.mod("Condition:HitBySpellDamageRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1411
	"conditionTakenFireDamageFromEnemyHitRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TakenFireDamageFromEnemyHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BeenHitRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1415
	"conditionBlockedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BlockedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1418
	"conditionBlockedAttackRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BlockedAttackRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BlockedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1422
	"conditionBlockedSpellRecently": func(_ Value, t *Tab) {
		t.mod("Condition:BlockedSpellRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:BlockedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1426
	"conditionEnergyShieldRechargeRecently": func(_ Value, t *Tab) {
		t.mod("Condition:EnergyShieldRechargeRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1429
	"conditionEnergyShieldRechargePastTwoSec": func(_ Value, t *Tab) {
		t.mod("Condition:EnergyShieldRechargePastTwoSec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1432
	"conditionStoppedTakingDamageOverTimeRecently": func(_ Value, t *Tab) {
		t.mod("Condition:StoppedTakingDamageOverTimeRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1435
	"conditionConvergence": func(_ Value, t *Tab) {
		t.mod("Condition:Convergence", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanGainConvergence"})
	},
	// ConfigOptions.lua L1438
	"buffPendulum": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "AREA" {
			t.mod("Condition:PendulumOfDestructionAreaOfEffect", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		} else if s == "DAMAGE" {
			t.mod("Condition:PendulumOfDestructionElementalDamage", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		}
	},
	// ConfigOptions.lua L1445
	"buffConflux": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "CHILLING" || s == "ALL" {
			t.mod("Condition:ChillingConflux", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		}
		if s == "SHOCKING" || s == "ALL" {
			t.mod("Condition:ShockingConflux", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		}
		if s == "IGNITING" || s == "ALL" {
			t.mod("Condition:IgnitingConflux", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		}
	},
	// ConfigOptions.lua L1456
	"highestDamageType": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s != "NONE" {
			t.mod("Condition:"+s+"IsHighestDamageType", modparser.Flag, modparser.Bool(true))
			t.mod("IsHighestDamageTypeOVERRIDE", modparser.Flag, modparser.Bool(true))
		}
	},
	// ConfigOptions.lua L1462
	"buffHeartstopper": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "HIT" {
			t.mod("Condition:HeartstopperHIT", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		} else if s == "DOT" {
			t.mod("Condition:HeartstopperDOT", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		} else if s == "AVERAGE" {
			t.mod("Condition:HeartstopperAVERAGE", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		}
	},
	// ConfigOptions.lua L1471
	"buffBastionOfHope": func(_ Value, t *Tab) {
		t.mod("Condition:BastionOfHopeActive", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1474
	"buffNgamahuFlamesAdvance": func(_ Value, t *Tab) {
		t.mod("Condition:NgamahuFlamesAdvance", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1477
	"buffHerEmbrace": func(_ Value, t *Tab) {
		t.mod("HerEmbrace", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanGainHerEmbrace"})
	},
	// ConfigOptions.lua L1483
	"conditionUsedSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1486
	"multiplierSkillUsedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:SkillUsedRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1490
	"conditionAttackedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:AttackedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1494
	"conditionCastSpellRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CastSpellRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1498
	"multiplierNonInstantSpellCastRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:NonInstantSpellCastRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1501
	"multiplierAppliedAilmentsRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:AppliedAilmentsRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1504
	"conditionLinkedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:LinkedRecently", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L1507
	"conditionStunnedWhileCastingRecently": func(_ Value, t *Tab) {
		t.mod("Condition:StunnedWhileCastingRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1510
	"conditionCastLast1Seconds": func(_ Value, t *Tab) {
		t.mod("Condition:CastLast1Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1513
	"multiplierCastLast8Seconds": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:CastLast8Seconds", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1516
	"conditionSuppressedRecently": func(_ Value, t *Tab) {
		t.mod("Condition:SuppressedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1519
	"multiplierHitsSuppressedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:HitsSuppressedRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:SuppressedRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1523
	"conditionUsedFireSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedFireSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1527
	"conditionUsedColdSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedColdSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1531
	"conditionUsedMinionSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedMinionSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1535
	"conditionUsedTravelSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedTravelSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedMovementSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1540
	"conditionUsedDashRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CastDashRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedTravelSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedMovementSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1546
	"conditionUsedMovementSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedMovementSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1550
	"conditionUsedVaalSkillRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedVaalSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1554
	"multiplierUsedVaalSkillInPast8Seconds": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:VaalSkillsUsedInPast8Seconds", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1557
	"conditionSoulGainPrevention": func(_ Value, t *Tab) {
		t.mod("Condition:SoulGainPrevention", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1563
	"conditionUsedWarcryRecently": func(_ Value, t *Tab) {
		t.mod("Condition:UsedWarcryRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedWarcryInPast8Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1568
	"conditionUsedWarcryInPast8Seconds": func(_ Value, t *Tab) {
		t.mod("Condition:UsedWarcryInPast8Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1571
	"multiplierAffectedByWarcryBuffDuration": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:AffectedByWarcryBuffDuration", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1574
	"DetonatedMinesRecently": func(_ Value, t *Tab) {
		t.mod("Condition:DetonatedMinesRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1577
	"multiplierMineDetonatedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:MineDetonatedRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1580
	"minesPerThrow": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("MineThrowCount", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1583
	"TriggeredTrapsRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TriggeredTrapsRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1586
	"multiplierTrapTriggeredRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:TrapTriggeredRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1589
	"conditionThrownTrapOrMineRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TrapOrMineThrownRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1592
	"trapsPerThrow": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("TrapThrowCount", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1595
	"conditionCursedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CursedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1598
	"conditionCastMarkRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CastMarkRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1601
	"conditionSpawnedCorpseRecently": func(_ Value, t *Tab) {
		t.mod("Condition:SpawnedCorpseRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1604
	"conditionConsumedCorpseRecently": func(_ Value, t *Tab) {
		t.mod("Condition:ConsumedCorpseRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1607
	"conditionConsumedCorpseInPast2Sec": func(_ Value, t *Tab) {
		t.mod("Condition:ConsumedCorpseInPast2Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1610
	"multiplierCorpseConsumedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:CorpseConsumedRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:ConsumedCorpseRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1614
	"conditionRavenousCorpseConsumed": func(_ Value, t *Tab) {
		t.mod("Condition:RavenousCorpseConsumed", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1617
	"multiplierWarcryUsedRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:WarcryUsedRecently", modparser.Base, modparser.Num(math.Min(n, 100)), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedWarcryRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedWarcryInPast8Seconds", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:UsedSkillRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1623
	"conditionTauntedEnemyRecently": func(_ Value, t *Tab) {
		t.mod("Condition:TauntedEnemyRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1626
	"conditionLostEnduranceChargeInPast8Sec": func(_ Value, t *Tab) {
		t.mod("Condition:LostEnduranceChargeInPast8Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1629
	"multiplierEnduranceChargesLostRecently": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:EnduranceChargesLostRecently", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.mod("Condition:LostEnduranceChargeInPast8Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1633
	"conditionBlockedHitFromUniqueEnemyInPast10Sec": func(_ Value, t *Tab) {
		t.mod("Condition:BlockedHitFromUniqueEnemyInPast10Sec", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1636
	"conditionKilledUniqueEnemy": func(_ Value, t *Tab) {
		t.mod("Condition:KilledUniqueEnemy", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1639
	"BlockedPast10Sec": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:BlockedPast10Sec", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1642
	"conditionImpaledRecently": func(_ Value, t *Tab) {
		t.mod("Condition:ImpaledRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1645
	"multiplierImpalesOnEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:ImpaleStacks", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1648
	"conditionCausedBleedingRecently": func(_ Value, t *Tab) {
		t.mod("Condition:CausedBleedingRecently", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1651
	"multiplierBleedsOnEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:BleedStacks", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.enemyMod("Condition:Bleeding", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1655
	"multiplierFragileRegrowth": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:FragileRegrowthCount", modparser.Base, modparser.Num(math.Min(n, 10)), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1658
	"conditionHaveArborix": func(_ Value, t *Tab) {
		t.mod("Condition:HaveIronReflexes", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Keystone", modparser.List, modparser.Str("Iron Reflexes"))
	},
	// ConfigOptions.lua L1662
	"conditionHaveAugyre": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "EleOverload" {
			t.mod("Condition:HaveElementalOverload", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
			t.mod("Keystone", modparser.List, modparser.Str("Elemental Overload"))
		} else if s == "ResTechnique" {
			t.mod("Condition:HaveResoluteTechnique", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
			t.mod("Keystone", modparser.List, modparser.Str("Resolute Technique"))
		}
	},
	// ConfigOptions.lua L1671
	"conditionHaveVulconus": func(_ Value, t *Tab) {
		t.mod("Condition:HaveAvatarOfFire", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
		t.mod("Keystone", modparser.List, modparser.Str("Avatar of Fire"))
	},
	// ConfigOptions.lua L1675
	"conditionHaveManaStorm": func(_ Value, t *Tab) {
		t.mod("Condition:SacrificeManaForLightning", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1681
	"EverlastingSacrifice": func(_ Value, t *Tab) {
		t.mod("ElementalResistMax", modparser.Base, modparser.Num(5), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "EverlastingSacrifice"})
		t.mod("ChaosResistMax", modparser.Base, modparser.Num(5), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "EverlastingSacrifice"})
	},
	// ConfigOptions.lua L1685
	"buffFanaticism": func(_ Value, t *Tab) {
		t.mod("Condition:Fanaticism", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "CanGainFanaticism"})
	},
	// ConfigOptions.lua L1688
	"conditionHitsAlwaysStun": func(_ Value, t *Tab) {
		t.mod("CullPercent", modparser.Max, modparser.Num(10), &modparser.CondTag{Var: "Combat"}, &modparser.CondTag{Var: "maceMasteryStunCullSpecced"})
	},
	// ConfigOptions.lua L1691
	"multiplierPvpTvalueOverride": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("MultiplierPvpTvalueOverride", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1694
	"multiplierPvpDamage": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("PvpDamageMultiplier", modparser.More, modparser.Num(n-100))
	},
	// ConfigOptions.lua L1697
	"buffAccelerationShrine": func(_ Value, t *Tab) {
		t.mod("Condition:AccelerationShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1700
	"buffBrutalShrine": func(_ Value, t *Tab) {
		t.mod("Condition:BrutalShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1703
	"buffDiamondShrine": func(_ Value, t *Tab) {
		t.mod("Condition:DiamondShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1706
	"buffDivineShrine": func(_ Value, t *Tab) {
		t.mod("Condition:DivineShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1709
	"buffEchoingShrine": func(_ Value, t *Tab) {
		t.mod("Condition:EchoingShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1712
	"buffGloomShrine": func(_ Value, t *Tab) {
		t.mod("Condition:GloomShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1715
	"buffGreaterFreezingShrine": func(_ Value, t *Tab) {
		t.mod("Condition:GreaterFreezingShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1718
	"buffGreaterShockingShrine": func(_ Value, t *Tab) {
		t.mod("Condition:GreaterShockingShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1721
	"buffGreaterSkeletalShrine": func(_ Value, t *Tab) {
		t.mod("Condition:GreaterSkeletalShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1724
	"buffImpenetrableShrine": func(_ Value, t *Tab) {
		t.mod("Condition:ImpenetrableShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1727
	"buffMassiveShrine": func(_ Value, t *Tab) {
		t.mod("Condition:MassiveShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1730
	"buffReplenishingShrine": func(_ Value, t *Tab) {
		t.mod("Condition:ReplenishingShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1733
	"buffResistanceShrine": func(_ Value, t *Tab) {
		t.mod("Condition:ResistanceShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1736
	"buffResonatingShrine": func(_ Value, t *Tab) {
		t.mod("Condition:ResonatingShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1739
	"buffLesserAccelerationShrine": func(_ Value, t *Tab) {
		t.mod("Condition:LesserAccelerationShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1742
	"buffLesserBrutalShrine": func(_ Value, t *Tab) {
		t.mod("Condition:LesserBrutalShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1745
	"buffLesserImpenetrableShrine": func(_ Value, t *Tab) {
		t.mod("Condition:LesserImpenetrableShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1748
	"buffLesserMassiveShrine": func(_ Value, t *Tab) {
		t.mod("Condition:LesserMassiveShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1751
	"buffLesserReplenishingShrine": func(_ Value, t *Tab) {
		t.mod("Condition:LesserReplenishingShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1754
	"buffLesserResistanceShrine": func(_ Value, t *Tab) {
		t.mod("Condition:LesserResistanceShrine", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Combat"})
	},
	// ConfigOptions.lua L1759
	"skillForkCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("ForkedCount", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1762
	"skillChainCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("ChainCount", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1765
	"skillPierceCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("PiercedCount", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1770
	"conditionAtCloseRange": func(_ Value, t *Tab) {
		t.mod("Condition:AtCloseRange", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1773
	"enemyMultiplierEnemyPresenceSeconds": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:EnemyPresenceSeconds", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1776
	"conditionEnemyMoving": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Moving", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1779
	"conditionEnemyFullLife": func(_ Value, t *Tab) {
		t.enemyMod("Condition:FullLife", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1782
	"conditionEnemyLowLife": func(_ Value, t *Tab) {
		t.enemyMod("Condition:LowLife", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1785
	"conditionEnemyCursed": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Cursed", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1788
	"conditionEnemyStunned": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Stunned", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1791
	"conditionEnemyBleeding": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Bleeding", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1794
	"overrideBleedStackPotential": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("BleedStackPotentialOverride", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1797
	"conditionSingleBleed": func(_ Value, t *Tab) {
		t.mod("Condition:SingleBleed", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1800
	"multiplierRuptureStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:RuptureStack", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1803
	"conditionEnemyPoisoned": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Poisoned", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1806
	"multiplierPoisonOnEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:PoisonStack", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1809
	"conditionNonPoisonedOnly": func(_ Value, t *Tab) {
		t.mod("Condition:NonPoisonedOnly", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1812
	"multiplierCurseExpiredOnEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:CurseExpired", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1815
	"multiplierCurseDurationExpiredOnEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:CurseDurationExpired", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1818
	"multiplierWitheredStackCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:WitheredStack", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1821
	"multiplierCorrosionStackCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:CorrosionStack", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
		t.enemyModSrc("Armour", modparser.Base, modparser.Num(-5000), "Corrosion", &modparser.MultiplierTag{Var: "CorrosionStack"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "CanCorrode"})
		t.enemyModSrc("Evasion", modparser.Base, modparser.Num(-1000), "Corrosion", &modparser.MultiplierTag{Var: "CorrosionStack"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "CanCorrode"})
	},
	// ConfigOptions.lua L1826
	"multiplierEnsnaredStackCount": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:EnsnareStackCount", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:Moving", modparser.Flag, modparser.Bool(true), &modparser.MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "EnsnareStackCount", Threshold: util.Some[float64](1)})
	},
	// ConfigOptions.lua L1830
	"conditionEnemyMaimed": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Maimed", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1833
	"conditionEnemyHindered": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Hindered", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1836
	"conditionEnemyExcommunicated": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Excommunicated", modparser.Flag, modparser.Bool(true), &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "CanExcommunicate"})
	},
	// ConfigOptions.lua L1839
	"conditionEnemyBlinded": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Blinded", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1842
	"overrideBuffBlinded": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("BlindEffect", modparser.Override, modparser.Num(n), &modparser.GlobalEffectTag{EffectType: "Buff"})
	},
	// ConfigOptions.lua L1845
	"conditionEnemyTaunted": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Taunted", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1848
	"conditionEnemyDebilitated": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Debilitated", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1851
	"conditionEnemyPacified": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Pacified", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1854
	"conditionEnemyBurning": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Burning", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1857
	"conditionEnemyIgnited": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Ignited", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1860
	"overrideIgniteStackPotential": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("IgniteStackPotentialOverride", modparser.Override, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1863
	"conditionEnemyScorched": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Scorched", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:ScorchedConfig", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1867
	"conditionScorchedEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("ScorchVal", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "ScorchedConfig"})
		t.enemyModSrc("DesiredScorchVal", modparser.Base, modparser.Num(n), "Scorch", &modparser.CondTag{Var: "ScorchedConfig", Neg: true})
	},
	// ConfigOptions.lua L1874
	"conditionEnemyOnScorchedGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Scorched", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:OnScorchedGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1878
	"conditionEnemyChilled": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Chilled", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:ChilledConfig", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1882
	"multiplierChilledByYouSeconds": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:ChilledByYouSeconds", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.enemyMod("Condition:ChilledByYou", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1886
	"conditionEnemyChilledEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyModSrc("ChillVal", modparser.Base, modparser.Num(n), "Chill", &modparser.CondTag{Var: "ChilledConfig"})
		t.enemyModSrc("DesiredChillVal", modparser.Base, modparser.Num(n), "Chill", &modparser.CondTag{Var: "ChilledConfig", Neg: true})
	},
	// ConfigOptions.lua L1890
	"conditionEnemyChilledByYourHits": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Chilled", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:ChilledByYourHits", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1894
	"HoarfrostStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("HoarfrostFreezeDuration", modparser.Inc, modparser.Num(n*20), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1897
	"multiplierBarnacleStacks": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:BarnacleStack", modparser.Base, modparser.Num(math.Min(n, 10)), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("SelfFreezeDuration", modparser.Inc, modparser.Num(5), &modparser.MultiplierTag{Var: "BarnacleStack"}, &modparser.CondTag{Var: "Effective"})
		t.enemyMod("SelfChillDuration", modparser.Inc, modparser.Num(5), &modparser.MultiplierTag{Var: "BarnacleStack"}, &modparser.CondTag{Var: "Effective"})
		t.enemyMod("PhysicalDamageConvertToCold", modparser.Base, modparser.Num(5), &modparser.MultiplierTag{Var: "BarnacleStack"}, &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1903
	"conditionEnemyFrozen": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Frozen", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1906
	"multiplierFrozenByYouSeconds": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:FrozenByYouSeconds", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Combat"})
		t.enemyMod("Condition:FrozenByYou", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1910
	"conditionEnemyBrittle": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Brittle", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:BrittleConfig", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1914
	"conditionBrittleEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("BrittleVal", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "BrittleConfig"})
		t.enemyModSrc("DesiredBrittleVal", modparser.Base, modparser.Num(n), "Brittle", &modparser.CondTag{Var: "BrittleConfig", Neg: true})
	},
	// ConfigOptions.lua L1918
	"conditionEnemyOnBrittleGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Brittle", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:OnBrittleGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1922
	"conditionEnemyShocked": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Shocked", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:ShockedConfig", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1926
	"conditionShockEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyModSrc("ShockVal", modparser.Base, modparser.Num(n), "Shock", &modparser.CondTag{Var: "ShockedConfig"})
		t.enemyModSrc("DesiredShockVal", modparser.Base, modparser.Num(n), "Shock", &modparser.CondTag{Var: "ShockedConfig", Neg: true})
	},
	// ConfigOptions.lua L1933
	"conditionEnemyOnShockedGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Shocked", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:OnShockedGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1937
	"conditionEnemySapped": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Sapped", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:SappedConfig", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1941
	"conditionSapEffect": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyModSrc("SapVal", modparser.Base, modparser.Num(n), "Sap", &modparser.CondTag{Var: "SappedConfig"})
		t.enemyModSrc("DesiredSapVal", modparser.Base, modparser.Num(n), "Sap", &modparser.CondTag{Var: "SappedConfig", Neg: true})
	},
	// ConfigOptions.lua L1945
	"conditionEnemyOnSappedGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Sapped", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("Condition:OnSappedGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1949
	"multiplierFreezeShockIgniteOnEnemy": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:FreezeShockIgniteOnEnemy", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1952
	"conditionEnemyFireExposure": func(_ Value, t *Tab) {
		t.enemyMod("FireExposure", modparser.Base, modparser.Num(-10), &modparser.CondTag{Var: "Effective"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "CanApplyFireExposure"})
	},
	// ConfigOptions.lua L1955
	"conditionEnemyColdExposure": func(_ Value, t *Tab) {
		t.enemyMod("ColdExposure", modparser.Base, modparser.Num(-10), &modparser.CondTag{Var: "Effective"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "CanApplyColdExposure"})
	},
	// ConfigOptions.lua L1958
	"conditionEnemyLightningExposure": func(_ Value, t *Tab) {
		t.enemyMod("LightningExposure", modparser.Base, modparser.Num(-10), &modparser.CondTag{Var: "Effective"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "CanApplyLightningExposure"})
	},
	// ConfigOptions.lua L1961
	"conditionEnemyIntimidated": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Intimidated", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1964
	"conditionEnemyCrushed": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Crushed", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1967
	"conditionEnemyHallowingFlame": func(_ Value, t *Tab) {
		t.enemyMod("Condition:HallowingFlame", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1976
	"conditionHallowingFlameMagnitude": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("HallowingFlameMagnitude", modparser.Override, modparser.Num(n))
	},
	// ConfigOptions.lua L1979
	"conditionNearLinkedTarget": func(_ Value, t *Tab) {
		t.enemyMod("Condition:NearLinkedTarget", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1982
	"conditionEnemyUnnerved": func(_ Value, t *Tab) {
		t.enemyMod("Condition:Unnerved", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1985
	"conditionEnemyCoveredInAsh": func(_ Value, t *Tab) {
		t.modSrc("CoveredInAshEffect", modparser.Base, modparser.Num(20), "Covered in Ash")
	},
	// ConfigOptions.lua L1988
	"conditionEnemyCoveredInFrost": func(_ Value, t *Tab) {
		t.modSrc("CoveredInFrostEffect", modparser.Base, modparser.Num(20), "Covered in Frost")
	},
	// ConfigOptions.lua L1991
	"conditionEnemyOnConsecratedGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:OnConsecratedGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1994
	"conditionEnemyHaveEnergyShield": func(_ Value, t *Tab) {
		t.enemyMod("Condition:HaveEnergyShield", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L1997
	"conditionEnemyOnProfaneGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:OnProfaneGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("CurseEffectOnSelf", modparser.Inc, modparser.Num(10), &modparser.CondTag{Var: "OnProfaneGround"})
		t.enemyMod("SelfCritChance", modparser.Inc, modparser.Num(100), &modparser.CondTag{Var: "OnProfaneGround"})
	},
	// ConfigOptions.lua L2002
	"multiplierEnemyAffectedByGraspingVines": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.mod("Multiplier:GraspingVinesAffectingEnemy", modparser.Base, modparser.Num(n), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L2005
	"conditionEnemyOnFungalGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:OnFungalGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L2008
	"conditionEnemyOnBrineGround": func(_ Value, t *Tab) {
		t.enemyMod("Condition:OnBrineGround", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
		t.enemyMod("SelfChillEffect", modparser.Inc, modparser.Num(30), &modparser.CondTag{Var: "OnBrineGround"})
		t.enemyMod("SelfBrittleEffect", modparser.Inc, modparser.Num(30), &modparser.CondTag{Var: "OnBrineGround"})
		t.enemyMod("SelfShockEffect", modparser.Inc, modparser.Num(30), &modparser.CondTag{Var: "OnBrineGround"})
		t.enemyMod("SelfSapEffect", modparser.Inc, modparser.Num(30), &modparser.CondTag{Var: "OnBrineGround"})
		t.enemyMod("Armour", modparser.Inc, modparser.Num(-25), &modparser.CondTag{Var: "OnBrineGround"})
		t.enemyMod("Evasion", modparser.Inc, modparser.Num(-25), &modparser.CondTag{Var: "OnBrineGround"})
	},
	// ConfigOptions.lua L2018
	"conditionEnemyInChillingArea": func(_ Value, t *Tab) {
		t.enemyMod("Condition:InChillingArea", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L2021
	"conditionEnemyInFrostGlobe": func(_ Value, t *Tab) {
		t.enemyMod("Condition:EnemyInFrostGlobe", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L2024
	"conditionEnemyLifeHigherThanPlayer": func(_ Value, t *Tab) {
		t.enemyMod("Condition:HigherLifePercentThanPlayer", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L2027
	"enemyConditionHitByFireDamage": func(_ Value, t *Tab) {
		t.enemyMod("Condition:HitByFireDamage", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L2030
	"enemyConditionHitByColdDamage": func(_ Value, t *Tab) {
		t.enemyMod("Condition:HitByColdDamage", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L2033
	"enemyConditionHitByLightningDamage": func(_ Value, t *Tab) {
		t.enemyMod("Condition:HitByLightningDamage", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L2036
	"enemyInRFOrScorchingRay": func(_ Value, t *Tab) {
		t.mod("Condition:InRFOrScorchingRay", modparser.Flag, modparser.Bool(true))
	},
	// ConfigOptions.lua L2040
	"conditionBetweenYouAndLinkedTarget": func(_ Value, t *Tab) {
		t.enemyMod("Condition:BetweenYouAndLinkedTarget", modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: "Effective"})
	},
	// ConfigOptions.lua L2043
	"conditionEnemyFireResZero": func(_ Value, t *Tab) {
		t.enemyMod("FireResist", modparser.Override, modparser.Num(0), &modparser.CondTag{Var: "Effective"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "HaveTrickstersSmile"})
	},
	// ConfigOptions.lua L2046
	"conditionEnemyColdResZero": func(_ Value, t *Tab) {
		t.enemyMod("ColdResist", modparser.Override, modparser.Num(0), &modparser.CondTag{Var: "Effective"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "HaveTrickstersSmile"})
	},
	// ConfigOptions.lua L2049
	"conditionEnemyLightningResZero": func(_ Value, t *Tab) {
		t.enemyMod("LightningResist", modparser.Override, modparser.Num(0), &modparser.CondTag{Var: "Effective"}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "HaveTrickstersSmile"})
	},
	// ConfigOptions.lua L2202
	"deliriousPercentage": func(v Value, t *Tab) {
		s, _ := StrOf(v)
		if s == "20Percent" {
			t.enemyModSrc("DamageTaken", modparser.More, modparser.Num(-16), "20% Delirious")
			t.enemyModSrc("Damage", modparser.Inc, modparser.Num(6), "20% Delirious")
		}
		if s == "40Percent" {
			t.enemyModSrc("DamageTaken", modparser.More, modparser.Num(-32), "40% Delirious")
			t.enemyModSrc("Damage", modparser.Inc, modparser.Num(12), "40% Delirious")
		}
		if s == "60Percent" {
			t.enemyModSrc("DamageTaken", modparser.More, modparser.Num(-48), "60% Delirious")
			t.enemyModSrc("Damage", modparser.Inc, modparser.Num(18), "60% Delirious")
		}
		if s == "80Percent" {
			t.enemyModSrc("DamageTaken", modparser.More, modparser.Num(-64), "80% Delirious")
			t.enemyModSrc("Damage", modparser.Inc, modparser.Num(24), "80% Delirious")
		}
		if s == "100Percent" {
			t.enemyModSrc("DamageTaken", modparser.More, modparser.Num(-80), "100% Delirious")
			t.enemyModSrc("Damage", modparser.Inc, modparser.Num(30), "100% Delirious")
		}
	},
	// ConfigOptions.lua L2224
	"enemyPhysicalReduction": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyModSrc("PhysicalDamageReduction", modparser.Base, modparser.Num(n), "EnemyConfig")
	},
	// ConfigOptions.lua L2239
	"enemyMaxResist": func(_ Value, t *Tab) {
		t.enemyModSrc("DoNotChangeMaxResFromConfig", modparser.Flag, modparser.Bool(true), "EnemyConfig")
	},
	// ConfigOptions.lua L2242
	"enemyBlockChance": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("BlockChance", modparser.Base, modparser.Num(n))
	},
	// ConfigOptions.lua L2342
	"enemyMultiplierPvpDamage": func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("MultiplierPvpDamage", modparser.Base, modparser.Num(n))
	},
}

// keep the imports honest when a regeneration drops the last user.
var _ = math.Min
var _ modparser.Value
