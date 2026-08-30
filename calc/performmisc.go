// CalcPerform.lua L612-1120: doActorMisc (misc buffs/debuffs),
// doActorCharges, actionSpeedMod.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// luaOr returns a when the Lua `a or b` keeps a (a non-nil): here the
// Override result when truthy, else the fallback.
func overrideOr(db *modstore.DB, name string, fallback float64) float64 {
	if ov, ok := db.Override(nil, name); ok {
		return valueNum(ov)
	}
	return fallback
}

// doActorMisc ports CalcPerform's doActorMisc.
func (env *Env) doActorMisc(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output
	condList := modDB.Conditions
	flr := func(v float64) float64 { return math.Floor(v) }

	if env.ModeCombat {
		if env.PlayerMainSkill.BaseSkillModList.Flag(nil, "Cruelty") {
			modDB.Multipliers["Cruelty"] = overrideOr(modDB, "Cruelty", 40)
		}
		// Minimum Rage
		if modDB.Sum(modparser.Base, nil, "MinimumRage") > modDB.Multipliers["Rage"] {
			modDB.Multipliers["Rage"] = modDB.Sum(modparser.Base, nil, "MinimumRage")
		}
		// allied fortify: parent link or party members (party is nil for
		// the replay's corpus; the parent link covers minions)
		alliedFortify := 0.0
		if modDB.Flag(nil, "YourFortifyEqualToParent") && actor.parent != nil {
			alliedFortify = actor.parent.output.N("FortificationStacks")
		}
		if modDB.Sum(modparser.Base, nil, "MinimumFortification") > 0 || alliedFortify > 0 {
			condList.Set("Fortified", true)
		}
		// Fortify
		if modDB.Flag(nil, "Fortified") || modDB.Sum(modparser.Base, nil, "Multiplier:Fortification") > 0 {
			var skillModList modstore.Store = modDB
			var skillCfg *modstore.Cfg
			if actor.mainSkill != nil {
				skillModList = actor.mainSkill.SkillModList
				skillCfg = actor.mainSkill.SkillCfg
			}
			maxStacks := math.Max(overrideOr(modDB, "MaximumFortification", modDB.Sum(modparser.Base, skillCfg, "MaximumFortification")), alliedFortify)
			minStacks := modDB.Sum(modparser.Base, nil, "MinimumFortification")
			if modDB.Flag(nil, "Condition:HaveMaxFortification") {
				minStacks = maxStacks
			}
			minStacks = math.Min(minStacks, maxStacks)
			stacksBase := maxStacks
			if minStacks > 0 {
				stacksBase = minStacks
			}
			stacks := math.Min(overrideOr(modDB, "FortificationStacks", stacksBase), maxStacks)
			increasedDuration := skillModList.Sum(modparser.Inc, nil, "FortifyDuration")
			output.SetN("MaximumFortification", maxStacks)
			output.SetN("MinimumFortification", minStacks)
			output.SetN("RemovableFortification", math.Min(maxStacks-minStacks, overrideOr(modDB, "FortificationStacks", maxStacks)-minStacks))
			output.SetN("FortificationStacks", stacks)
			output.SetN("FortificationStacksOver20", math.Min(math.Max(0, stacks-20), maxStacks-20))
			fortifyDurBase := data.Misc.FortifyBaseDuration
			if ov, ok := skillModList.Override(nil, "FortifyDuration"); ok {
				fortifyDurBase = valueNum(ov)
			}
			output.SetN("FortifyDuration", fortifyDurBase*(1+increasedDuration/100))
			output.SetStr("FortificationEffect", "0") // string, shown for Willowgift
			if !modDB.Flag(nil, "Condition:NoFortificationMitigation") {
				output.SetN("FortificationEffect", stacks)
				modDB.AddMod(newModS("DamageTakenWhenHit", modparser.More, modparser.Num(-stacks), "Fortification"))
			}
			if stacks >= maxStacks {
				modDB.AddMod(newModS("Condition:HaveMaximumFortification", modparser.Flag, modparser.Bool(true), ""))
			}
			modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + 1
		}
		if modDB.Flag(nil, "Onslaught") {
			// Silver flask detection adds flask effect
			onslaughtFromFlask := false
			flaskEffectInc := -100.0
			for item := range env.Flasks {
				if strings.Contains(deref(item.In.BaseName), "Silver Flask") {
					onslaughtFromFlask = true
					curFlaskEffectInc := item.In.FlaskData.EffectInc + modDB.Sum(modparser.Inc, &modstore.Cfg{Actor: "player"}, "FlaskEffect")
					if item.In.Rarity == "MAGIC" {
						curFlaskEffectInc += modDB.Sum(modparser.Inc, &modstore.Cfg{Actor: "player"}, "MagicUtilityFlaskEffect")
					}
					if flaskEffectInc < curFlaskEffectInc/100 {
						flaskEffectInc = curFlaskEffectInc / 100
					}
				}
			}
			onslaughtEffectInc := modDB.Sum(modparser.Inc, nil, "OnslaughtEffect", "BuffEffectOnSelf") / 100
			var effect float64
			if onslaughtFromFlask {
				effect = flr(20 * (1 + flaskEffectInc + onslaughtEffectInc))
			} else {
				effect = flr(20 * (1 + onslaughtEffectInc))
			}
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(effect), "Onslaught", modparser.FlagAttack, modparser.KeywordNone))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(effect), "Onslaught", modparser.FlagCast, modparser.KeywordNone))
			modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(effect), "Onslaught"))
		}
		if condList.Get("AffectedByArcaneSurge") || modDB.Flag(nil, "Condition:ArcaneSurge") {
			condList.Set("AffectedByArcaneSurge", true)
			effect := 1 + modDB.Sum(modparser.Inc, nil, "ArcaneSurgeEffect", "BuffEffectOnSelf")/100
			effect = effect + modDB.Sum(modparser.Inc, &modstore.Cfg{Actor: "player"}, "FlaskEffect")/100*modDB.Sum(modparser.Base, nil, "FlaskEffectToArcaneSurgeEffect")/100
			manaRegen := 30.0
			if v, ok := modDB.Max(nil, "ArcaneSurgeManaRegen"); ok {
				manaRegen = v
			}
			modDB.AddMod(newModS("ManaRegen", modparser.Inc, modparser.Num(manaRegen*effect), "Arcane Surge"))
			castSpeedBase := 20.0
			if v, ok := modDB.Max(nil, "ArcaneSurgeCastSpeed"); ok {
				castSpeedBase = v
			}
			arcaneSurgeCastSpeed := castSpeedBase * effect
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(arcaneSurgeCastSpeed), "Arcane Surge", modparser.FlagCast, modparser.KeywordNone))
			if modDB.Flag(nil, "ArcaneSurgeCastSpeedToMovementSpeed") {
				modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(arcaneSurgeCastSpeed), "Arcane Surge"))
			}
			arcaneSurgeDamage := 0.0
			if v, ok := modDB.Max(nil, "ArcaneSurgeDamage"); ok {
				arcaneSurgeDamage = v
			}
			if arcaneSurgeDamage != 0 {
				modDB.AddMod(newModSF("Damage", modparser.More, modparser.Num(arcaneSurgeDamage*effect), "Arcane Surge", modparser.FlagSpell, modparser.KeywordNone))
			}
			arcaneSurgeLifeRegen := modDB.Sum(modparser.Base, nil, "ArcaneSurgeAlsoLifeRegen")
			if arcaneSurgeLifeRegen > 0 {
				modDB.AddMod(newModS("LifeRegen", modparser.Inc, modparser.Num(arcaneSurgeLifeRegen*effect), "Arcane Surge"))
			}
		}
		if modDB.Flag(nil, "Fanaticism") && actor.mainSkill != nil && actor.mainSkill.SkillFlags["selfCast"] {
			effect := flr(75 * (1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newModSF("Speed", modparser.More, modparser.Num(effect), "Fanaticism", modparser.FlagCast, modparser.KeywordNone))
			modDB.AddMod(newModSF("Cost", modparser.More, modparser.Num(-effect), "Fanaticism", modparser.FlagCast, modparser.KeywordNone))
			modDB.AddMod(newModSF("AreaOfEffect", modparser.Inc, modparser.Num(effect), "Fanaticism", modparser.FlagCast, modparser.KeywordNone))
		}
		if modDB.Flag(nil, "Condition:CanGainSpiritInfusion") {
			globalEffectTag := &modparser.GlobalEffectTag{EffectType: "Buff", EffectName: "Spirit Infusion", Unscalable: true}
			multiplierTag := &modparser.MultiplierTag{Var: "SpiritInfusion"}
			modDB.AddMod(newModS("EnergyShieldRechargeFaster", modparser.Inc, modparser.Num(15.0), "Spirit Infusion", multiplierTag, globalEffectTag))
			modDB.AddMod(newModSF("Damage", modparser.More, modparser.Num(5.0), "Spirit Infusion", modparser.FlagNone, modparser.KeywordSpell, &modparser.SkillTypeTag{SkillType: modparser.SkillTypeChannel}, multiplierTag, globalEffectTag))
			modDB.AddMod(newModSF("Cost", modparser.More, modparser.Num(10.0), "Spirit Infusion", modparser.FlagNone, modparser.KeywordSpell, &modparser.SkillTypeTag{SkillType: modparser.SkillTypeChannel}, multiplierTag, globalEffectTag))
		}
		if modDB.Flag(nil, "UnholyMight") {
			effect := 1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")/100
			modDB.AddMod(newModS("PhysicalDamageConvertToChaos", modparser.Base, modparser.Num(flr(100*effect)), "Unholy Might"))
			modDB.AddMod(newModS("Condition:CanWither", modparser.Flag, modparser.Bool(true), "Unholy Might"))
		}
		if modDB.Flag(nil, "ShepherdOfSouls") {
			modDB.AddMod(newModS("SoulCost", modparser.More, modparser.Num(-80.0), "Shepherd of Souls", &modparser.SkillTypeTag{SkillType: modparser.SkillTypeVaal}, &modparser.SkillTypeTag{SkillType: modparser.SkillTypeAura, Neg: true}))
			modDB.AddMod(newModS("SoulCost", modparser.Inc, modparser.Num(100.0), "Shepherd of Souls", &modparser.SkillTypeTag{SkillType: modparser.SkillTypeVaal}, &modparser.SkillTypeTag{SkillType: modparser.SkillTypeAura, Neg: true}, &modparser.MultiplierTag{Var: "VaalSkillsUsedInPast8Seconds"}))
		}
		if modDB.Flag(nil, "ChaoticMight") {
			effect := flr(30 * (1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newModS("PhysicalDamageGainAsChaos", modparser.Base, modparser.Num(effect), "Chaotic Might"))
		}
		if modDB.Flag(nil, "Tailwind") {
			effect := flr(8 * (1 + modDB.Sum(modparser.Inc, nil, "TailwindEffectOnSelf", "BuffEffectOnSelf")/100))
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(effect), "Tailwind"))
		}
		if modDB.Flag(nil, "Condition:TotemTailwind") {
			modDB.AddMod(newModS("TotemActionSpeed", modparser.Inc, modparser.Num(8.0), "Tailwind"))
		}
		if modDB.Flag(nil, "Adrenaline") {
			effectMod := 1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")/100
			modDB.AddMod(newModS("Damage", modparser.Inc, modparser.Num(flr(100*effectMod)), "Adrenaline"))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(flr(25*effectMod)), "Adrenaline", modparser.FlagAttack, modparser.KeywordNone))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(flr(25*effectMod)), "Adrenaline", modparser.FlagCast, modparser.KeywordNone))
			modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(flr(25*effectMod)), "Adrenaline"))
			modDB.AddMod(newModS("PhysicalDamageReduction", modparser.Base, modparser.Num(flr(10*effectMod)), "Adrenaline"))
		}
		if modDB.Flag(nil, "Condition:WildSavagery") && modDB.Flag(nil, "WildSavagery") {
			modDB.AddMod(newModS("PhysicalDamage", modparser.Inc, modparser.Num(100.0), "Wild Savagery"))
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(10.0), "Wild Savagery"))
			modDB.AddMod(newModS("IgnoreEnemyPhysicalDamageReduction", modparser.Flag, modparser.Bool(true), "Wild Savagery"))
			modDB.AddMod(newModS("StunImmune", modparser.Flag, modparser.Bool(true), "Wild Savagery"))
		}
		if modDB.Flag(nil, "Convergence") {
			effect := flr(30 * (1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newModS("ElementalDamage", modparser.More, modparser.Num(effect), "Convergence"))
		}
		if modDB.Flag(nil, "HerEmbrace") {
			condList.Set("HerEmbrace", true)
			modDB.AddMod(newModS("AvoidStun", modparser.Base, modparser.Num(100.0), "Her Embrace"))
			modDB.AddMod(newModSF("PhysicalDamageGainAsFire", modparser.Base, modparser.Num(123.0), "Her Embrace", modparser.FlagSword, modparser.KeywordNone))
			modDB.AddMod(newModS("AvoidFreeze", modparser.Base, modparser.Num(100.0), "Her Embrace"))
			modDB.AddMod(newModS("AvoidChill", modparser.Base, modparser.Num(100.0), "Her Embrace"))
			modDB.AddMod(newModS("AvoidIgnite", modparser.Base, modparser.Num(100.0), "Her Embrace"))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(20.0), "Her Embrace", modparser.FlagAttack, modparser.KeywordNone))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(20.0), "Her Embrace", modparser.FlagCast, modparser.KeywordNone))
			modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(20.0), "Her Embrace"))
		}
		if modDB.Flag(nil, "Condition:OnConsecratedGround") {
			effect := 1 + modDB.Sum(modparser.Inc, nil, "ConsecratedGroundEffect")/100
			modDB.AddMod(newModS("LifeRegenPercent", modparser.Base, modparser.Num(5*effect), "Consecrated Ground"))
			modDB.AddMod(newModS("CurseEffectOnSelf", modparser.Inc, modparser.Num(-50*effect), "Consecrated Ground"))
			modDB.AddMod(newModS("Accuracy", modparser.Inc, modparser.Num(flr(modDB.Sum(modparser.Inc, nil, "ConsecratedGroundAlsoAccuracy")*effect)), "Consecrated Ground"))
		}
		if modDB.Flag(nil, "Condition:PhantasmalMight") {
			limit := output.N("ActivePhantasmLimit")
			if limit == 0 {
				limit = 1
			}
			modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + limit - 1
		}
		if modDB.Flag(nil, "Elusive") {
			maxSkillInc := 0.0
			if v, ok := modDB.Max(&modstore.Cfg{Source: "Skill"}, "ElusiveEffect"); ok {
				maxSkillInc = v
			}
			inc := modDB.Sum(modparser.Inc, nil, "ElusiveEffect", "BuffEffectOnSelf")
			if actor.mainSkill.SkillModList.Flag(nil, "SupportedByNightblade") {
				inc = inc + modDB.Sum(modparser.Inc, nil, "NightbladeSupportedElusiveEffect")
			}
			inc = inc + maxSkillInc
			elusiveEffectMod := (1 + inc/100) * modDB.More(nil, "ElusiveEffect", "BuffEffectOnSelf") * 100
			elusiveEffectMinThreshold := overrideOr(modDB, "ElusiveEffectMinThreshold", 0)
			elusiveEffectIncreaseDuration := modDB.Sum(modparser.Base, nil, "ElusiveEffectIncreaseDuration")
			peakElusiveEffect := elusiveEffectMod
			if elusiveEffectIncreaseDuration > 0 {
				elusiveEffectChangeRate := 20 / (1 + modDB.Sum(modparser.Inc, nil, "ElusiveEffectLossSlower")/100)
				peakElusiveEffect = elusiveEffectMod + elusiveEffectChangeRate*elusiveEffectIncreaseDuration
				elusiveEffectDecreaseDuration := (peakElusiveEffect - elusiveEffectMinThreshold) / elusiveEffectChangeRate
				totalElusiveEffectDuration := elusiveEffectIncreaseDuration + elusiveEffectDecreaseDuration
				averageIncreaseEffect := (elusiveEffectMod + peakElusiveEffect) / 2
				averageDecreaseEffect := (peakElusiveEffect + elusiveEffectMinThreshold) / 2
				output.SetN("ElusiveEffectMod", (averageIncreaseEffect*elusiveEffectIncreaseDuration+averageDecreaseEffect*elusiveEffectDecreaseDuration)/totalElusiveEffectDuration)
			} else {
				output.SetN("ElusiveEffectMod", (elusiveEffectMod+elusiveEffectMinThreshold)/2)
			}
			modDB.AddMod(newModS("ElusiveEffect", modparser.Inc, modparser.Num(maxSkillInc), "Max Skill Effect"))
			if ov, ok := modDB.Override(nil, "ElusiveEffect"); ok {
				output.SetN("ElusiveEffectMod", math.Min(valueNum(ov), peakElusiveEffect))
			}
			effect := output.N("ElusiveEffectMod") / 100
			condList.Set("Elusive", true)
			modDB.AddMod(newModS("AvoidAllDamageFromHitsChance", modparser.Base, modparser.Num(flr(15*effect)), "Elusive"))
			modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(flr(30*effect)), "Elusive"))
		}
		if _, ok := modDB.Max(nil, "WitherEffectStack"); ok {
			modDB.AddMod(newModS("Condition:CanWither", modparser.Flag, modparser.Bool(true), "Config"))
			effect, _ := modDB.Max(nil, "WitherEffectStack")
			enemyDB.AddMod(newModS("ChaosDamageTaken", modparser.Inc, modparser.Num(effect), "Withered", &modparser.MultiplierTag{Var: "WitheredStack", Limit: opt(15.0)}))
		}
		if modDB.Flag(nil, "Condition:CanBeWithered") {
			effect := 6 * (100 + modDB.Sum(modparser.Inc, nil, "WitherEffectOnSelf")) / 100 * modDB.More(nil, "WitherEffectOnSelf")
			modDB.AddMod(newModS("ChaosDamageTaken", modparser.Inc, modparser.Num(effect), "Withered", &modparser.MultiplierTag{Var: "WitheredStack", Limit: opt(15.0)}))
		}
		if modDB.Flag(nil, "Excommunicated") {
			modDB.AddMod(newModS("ChaosDamage", modparser.More, modparser.Num(-100.0), "Excommunicated"))
		}
		if modDB.Flag(nil, "Blind") && !modDB.Flag(nil, "CannotBeBlinded") {
			if !modDB.Flag(nil, "UnaffectedByBlind") {
				effect := 1 + modDB.Sum(modparser.Inc, nil, "BlindEffect", "BuffEffectOnSelf")/100
				if ov, ok := modDB.Override(nil, "BlindEffect"); ok {
					effect = math.Min(valueNum(ov)/100, effect)
				}
				modDB.AddMod(newModS("Accuracy", modparser.More, modparser.Num(flr(-20*effect)), "Blind"))
				modDB.AddMod(newModS("Evasion", modparser.More, modparser.Num(flr(-20*effect)), "Blind"))
			}
		}
		if modDB.Flag(nil, "Chill") {
			ail := data.NonDamagingAilment["Chill"]
			// `m_max(Sum(SelfChillOverride), Override(ChillVal)) or default`.
			// Override returns NO VALUES when unset (ModDB.lua:219 falls off
			// the end), so that collapses to the one-argument m_max — i.e.
			// just the Sum — and the `or ailmentData.Chill.default` tail is
			// dead, because a number (even 0) is truthy. #EVAL
			chillValue := modDB.Sum(modparser.Base, nil, "SelfChillOverride")
			if ov, ok := modDB.Override(nil, "ChillVal"); ok {
				chillValue = math.Max(chillValue, valueNum(ov))
			}
			totalChillSelfEffect := Mod(modDB, nil, "SelfChillEffect")
			avoidChill := 0.0
			if modDB.Flag(nil, "ChillImmune", "ElementalAilmentImmune") {
				avoidChill = 100
			} else {
				sum := modDB.Sum(modparser.Base, nil, "AvoidChill", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
					sum += modDB.Sum(modparser.Base, nil, "AvoidShock")
				}
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					sum += modDB.Sum(modparser.Base, nil, "SpellSuppressionChance") / 2
				}
				avoidChill = flr(math.Min(sum, 100))
			}
			chillMax := ail.Max
			if ov, ok := modDB.Override(nil, "ChillMax"); ok {
				chillMax = valueNum(ov)
			}
			effect := 0.0
			if avoidChill != 100 {
				effect = math.Min(math.Max(flr(chillValue*totalChillSelfEffect), 0), chillMax)
			}
			if modDB.Flag(nil, "SkitterbotBonechill") {
				sign := 1.0
				if !modDB.Flag(nil, "SelfChillEffectIsReversed") {
					sign = -1
				}
				modDB.AddMod(newModS("ColdDamageTaken", modparser.Inc, modparser.Num(effect*-sign), "Bonechill"))
			}
			sign := -1.0
			if modDB.Flag(nil, "SelfChillEffectIsReversed") {
				sign = 1
			}
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(effect*sign), "Chill"))
		}
		if modDB.Flag(nil, "Shock") {
			ail := data.NonDamagingAilment["Shock"]
			// Same shape as chill above: the Override contributes only when
			// it exists, and the `or default` tail is dead. #EVAL
			shockValue := modDB.Sum(modparser.Base, nil, "SelfShockOverride")
			if ov, ok := modDB.Override(nil, "ShockVal"); ok {
				shockValue = math.Max(shockValue, valueNum(ov))
			}
			totalShockSelfEffect := Mod(modDB, nil, "SelfShockEffect")
			avoidShock := 0.0
			if modDB.Flag(nil, "ShockImmune", "ElementalAilmentImmune") {
				avoidShock = 100
			} else {
				sum := modDB.Sum(modparser.Base, nil, "AvoidShock", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					sum += modDB.Sum(modparser.Base, nil, "SpellSuppressionChance") / 2
				}
				avoidShock = flr(math.Min(sum, 100))
			}
			shockMax := ail.Max
			if ov, ok := modDB.Override(nil, "ShockMax"); ok {
				shockMax = valueNum(ov)
			}
			effect := 0.0
			if avoidShock != 100 {
				effect = math.Min(math.Max(flr(shockValue*totalShockSelfEffect), 0), shockMax)
			}
			modDB.AddMod(newModS("DamageTaken", modparser.Inc, modparser.Num(effect), "Shock"))
		}
		if modDB.Flag(nil, "Scorch") {
			ail := data.NonDamagingAilment["Scorch"]
			scorchValue := math.Max(modDB.Sum(modparser.Base, nil, "SelfScorchOverride"), overrideNum(modDB, nil, "ScorchVal"))
			totalScorchSelfEffect := Mod(modDB, nil, "SelfScorchEffect")
			avoidScorch := 0.0
			if modDB.Flag(nil, "ScorchImmune", "ElementalAilmentImmune") {
				avoidScorch = 100
			} else {
				sum := modDB.Sum(modparser.Base, nil, "AvoidScorch", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
					sum += modDB.Sum(modparser.Base, nil, "AvoidShock")
				}
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					sum += modDB.Sum(modparser.Base, nil, "SpellSuppressionChance") / 2
				}
				avoidScorch = flr(math.Min(sum, 100))
			}
			scorchMax := ail.Max
			if ov, ok := modDB.Override(nil, "ScorchMax"); ok {
				scorchMax = valueNum(ov)
			}
			effect := 0.0
			if avoidScorch != 100 {
				effect = math.Min(math.Max(flr(scorchValue*totalScorchSelfEffect), 0), scorchMax)
			}
			modDB.AddMod(newModS("ElementalResist", modparser.Base, modparser.Num(-effect), "Scorch"))
		}
		if modDB.Flag(nil, "Freeze") {
			effect := math.Max(flr(70*Mod(modDB, nil, "SelfChillEffect")), 0)
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(-effect), "Freeze"))
		}
		if modDB.Flag(nil, "CanLeechLifeOnFullLife") && !modDB.Flag(nil, "GhostReaver") {
			condList.Set("Leeching", true)
			condList.Set("LeechingLife", true)
		}
		if modDB.Flag(nil, "CanLeechEnergyShieldOnFullEnergyShield") {
			condList.Set("Leeching", true)
			condList.Set("LeechingEnergyShield", true)
		}
		if modDB.Flag(nil, "Condition:CanGainRage") || modDB.Sum(modparser.Base, nil, "RageRegen") > 0 {
			// skillCfg is an undefined local in the reference: nil
			maxStacks := flr(modDB.Sum(modparser.Base, nil, "MaximumRage") * modDB.More(nil, "MaximumRage"))
			minStacks := math.Min(modDB.Sum(modparser.Base, nil, "MinimumRage"), maxStacks)
			rageConfig := modDB.Sum(modparser.Base, nil, "Multiplier:RageStack")
			stacks := math.Min(rageConfig, maxStacks)
			if minStacks > 0 && stacks < minStacks {
				stacks = minStacks
			}
			stacks = math.Max(stacks, 0)
			output.SetN("RageEffect", flr(stacks*Mod(modDB, nil, "RageEffect")))
			modDB.AddMod(newModS("Multiplier:RageEffect", modparser.Base, modparser.Num(output.N("RageEffect")), "Base"))
			output.SetN("Rage", stacks)
			output.SetN("MaximumRage", maxStacks)
			modDB.AddMod(newModS("Multiplier:Rage", modparser.Base, modparser.Num(stacks), "Base"))
			if modDB.Flag(nil, "Condition:RageSpellDamage") {
				modDB.AddMod(newModSF("Damage", modparser.More, modparser.Num(output.N("RageEffect")), "Rage", modparser.FlagSpell, modparser.KeywordNone))
			} else {
				modDB.AddMod(newModSF("Damage", modparser.More, modparser.Num(output.N("RageEffect")), "Rage", modparser.FlagAttack, modparser.KeywordNone))
			}
			if stacks == maxStacks {
				modDB.AddMod(newModS("Condition:HaveMaximumRage", modparser.Flag, modparser.Bool(true), ""))
			}
			output.SetN("InherentRageLossDelay", 2+modDB.Sum(modparser.Base, nil, "InherentRageLossDelay"))
			if !modDB.Flag(nil, "InherentRageLossIsPrevented") {
				output.SetN("InherentRageLoss", 10*(1+modDB.Sum(modparser.Inc, nil, "InherentRageLoss")/100))
			} else {
				output.SetN("InherentRageLoss", 0.0)
			}
		}
		if env.ConfigInput.MultiplierManaBurnStacks > 0 {
			maxManaBurn := modDB.Sum(modparser.Base, nil, "MaxManaBurnStacks")
			if maxManaBurn == 0 {
				maxManaBurn = 9999
			}
			manaBurnStacks := math.Min(env.ConfigInput.MultiplierManaBurnStacks, maxManaBurn)
			modDB.AddMod(newModS("Multiplier:ManaBurnStacks", modparser.Base, modparser.Num(manaBurnStacks), "Config"))
			manaBurnStacks = manaBurnStacks + modDB.Sum(modparser.Base, &modstore.Cfg{Actor: "player"}, "EffectiveManaBurnStacks")
			if modDB.Flag(nil, "Condition:WeepingWoundsInsteadOfManaBurn") {
				modDB.AddMod(newModS("Multiplier:WeepingWoundsStacks", modparser.Base, modparser.Num(manaBurnStacks), "Config"))
			} else {
				modDB.AddMod(newModS("Multiplier:EffectiveManaBurnStacks", modparser.Base, modparser.Num(manaBurnStacks), "Config"))
			}
		}
		if modDB.Sum(modparser.Base, nil, "CoveredInAshEffect") > 0 {
			effect := modDB.Sum(modparser.Base, nil, "CoveredInAshEffect")
			enemyDB.AddMod(newModS("FireDamageTaken", modparser.Inc, modparser.Num(math.Min(effect, 20)), "Covered in Ash"))
		}
		if modDB.Sum(modparser.Base, nil, "CoveredInFrostEffect") > 0 {
			effect := modDB.Sum(modparser.Base, nil, "CoveredInFrostEffect")
			enemyDB.AddMod(newModS("ColdDamageTaken", modparser.Inc, modparser.Num(math.Min(effect, 20)), "Covered in Frost"))
		}
		if modDB.Flag(nil, "HasMalediction") {
			modDB.AddMod(newModS("DamageTaken", modparser.Inc, modparser.Num(10.0), "Malediction"))
			modDB.AddMod(newModS("Damage", modparser.Inc, modparser.Num(-10.0), "Malediction"))
		}
		if modDB.Flag(nil, "HasMaddeningPresence") {
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(-10.0), "Maddening Presence"))
			modDB.AddMod(newModS("Damage", modparser.Inc, modparser.Num(-10.0), "Maddening Presence"))
		}
		if modDB.Flag(nil, "HasShapersPresence") {
			modDB.AddMod(newModS("BuffExpireFaster", modparser.More, modparser.Num(-20.0), "Shapers Presence"))
		}
		if modDB.Flag(nil, "Condition:CanHaveSoulEater") {
			max := overrideOr(modDB, "SoulEaterMax", modDB.Sum(modparser.Base, nil, "SoulEaterMax"))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(5.0), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.MultiplierTag{Var: "SoulEaterStack", Limit: opt(max)}))
			modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(5.0), "Base", modparser.FlagCast, modparser.KeywordNone, &modparser.MultiplierTag{Var: "SoulEaterStack", Limit: opt(max)}))
		}
	}

	// Process enemy modifiers
	applyEnemyModifiers(actor, false)
}

// doActorCharges ports CalcPerform's doActorCharges.
func (env *Env) doActorCharges(actor *performActor) {
	modDB := actor.db
	output := actor.output
	setMax := func(k string, v float64) { output.SetN(k, math.Max(v, 0)) }

	setMax("PowerChargesMin", modDB.Sum(modparser.Base, nil, "PowerChargesMin"))
	output.SetN("PowerChargesMax", overrideOr(modDB, "PowerChargesMax", math.Max(modDB.Sum(modparser.Base, nil, "PowerChargesMax"), 0)))
	output.SetN("PowerChargesDuration", math.Floor(modDB.Sum(modparser.Base, nil, "ChargeDuration")*Mod(modDB, nil, "PowerChargesDuration", "ChargeDuration")))
	if modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges") {
		source := modDB.Mods["MaximumFrenzyChargesIsMaximumPowerCharges"][0].Source
		modDB.ReplaceMod(newModS("FrenzyChargesMax", modparser.Override, modparser.Num(output.N("PowerChargesMax")), source))
	}
	setMax("FrenzyChargesMin", modDB.Sum(modparser.Base, nil, "FrenzyChargesMin"))
	fcBase := modDB.Sum(modparser.Base, nil, "FrenzyChargesMax")
	if modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges") {
		fcBase = output.N("PowerChargesMax")
	}
	output.SetN("FrenzyChargesMax", overrideOr(modDB, "FrenzyChargesMax", math.Max(fcBase, 0)))
	output.SetN("FrenzyChargesDuration", math.Floor(modDB.Sum(modparser.Base, nil, "ChargeDuration")*Mod(modDB, nil, "FrenzyChargesDuration", "ChargeDuration")))
	if modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges") {
		source := modDB.Mods["MaximumEnduranceChargesIsMaximumFrenzyCharges"][0].Source
		modDB.ReplaceMod(newModS("EnduranceChargesMax", modparser.Override, modparser.Num(output.N("FrenzyChargesMax")), source))
	}
	setMax("EnduranceChargesMin", modDB.Sum(modparser.Base, nil, "EnduranceChargesMin"))
	ecBase := modDB.Sum(modparser.Base, nil, "EnduranceChargesMax")
	if modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges") {
		ecBase = output.N("FrenzyChargesMax")
	}
	// (partyMembers link is nil for the replay corpus)
	output.SetN("EnduranceChargesMax", overrideOr(modDB, "EnduranceChargesMax", math.Max(ecBase, 0)))
	output.SetN("EnduranceChargesDuration", math.Floor(modDB.Sum(modparser.Base, nil, "ChargeDuration")*Mod(modDB, nil, "EnduranceChargesDuration", "ChargeDuration")))
	setMax("SiphoningChargesMax", modDB.Sum(modparser.Base, nil, "SiphoningChargesMax"))
	setMax("ChallengerChargesMax", modDB.Sum(modparser.Base, nil, "ChallengerChargesMax"))
	setMax("BlitzChargesMax", modDB.Sum(modparser.Base, nil, "BlitzChargesMax"))
	setMax("InspirationChargesMax", modDB.Sum(modparser.Base, nil, "InspirationChargesMax"))
	setMax("CrabBarriersMax", modDB.Sum(modparser.Base, nil, "CrabBarriersMax"))
	brutalMin := 0.0
	if modDB.Flag(nil, "MinimumEnduranceChargesEqualsMinimumBrutalCharges") {
		if modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges") {
			brutalMin = output.N("EnduranceChargesMax")
		} else {
			brutalMin = output.N("EnduranceChargesMin")
		}
	}
	setMax("BrutalChargesMin", brutalMin)
	brutalMax := 0.0
	if modDB.Flag(nil, "MaximumEnduranceChargesEqualsMaximumBrutalCharges") {
		brutalMax = output.N("EnduranceChargesMax")
	}
	setMax("BrutalChargesMax", brutalMax)
	setMax("BrineChargesMax", output.N("EnduranceChargesMax"))
	absMin := 0.0
	if modDB.Flag(nil, "MinimumPowerChargesEqualsMinimumAbsorptionCharges") {
		if modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges") {
			absMin = output.N("PowerChargesMax")
		} else {
			absMin = output.N("PowerChargesMin")
		}
	}
	setMax("AbsorptionChargesMin", absMin)
	absMax := 0.0
	if modDB.Flag(nil, "MaximumPowerChargesEqualsMaximumAbsorptionCharges") {
		absMax = output.N("PowerChargesMax")
	}
	setMax("AbsorptionChargesMax", absMax)
	afflMin := 0.0
	if modDB.Flag(nil, "MinimumFrenzyChargesEqualsMinimumAfflictionCharges") {
		if modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges") {
			afflMin = output.N("FrenzyChargesMax")
		} else {
			afflMin = output.N("FrenzyChargesMin")
		}
	}
	setMax("AfflictionChargesMin", afflMin)
	afflMax := 0.0
	if modDB.Flag(nil, "MaximumFrenzyChargesEqualsMaximumAfflictionCharges") {
		afflMax = output.N("FrenzyChargesMax")
	}
	setMax("AfflictionChargesMax", afflMax)
	setMax("BloodChargesMax", modDB.Sum(modparser.Base, nil, "BloodChargesMax"))
	setMax("SpiritChargesMax", modDB.Sum(modparser.Base, nil, "SpiritChargesMax"))
	sim := modDB.Sum(modparser.Base, nil, "SpiritInfusionsMax")
	if modDB.Flag(nil, "Condition:CanGainSpiritInfusion") {
		sim += 10
	}
	output.SetN("SpiritInfusionsMax", sim)

	// Initialize charges
	for _, k := range []string{"PowerCharges", "FrenzyCharges", "EnduranceCharges", "SiphoningCharges",
		"ChallengerCharges", "BlitzCharges", "InspirationCharges", "GhostShrouds", "BrutalCharges",
		"BrineCharges", "AbsorptionCharges", "AfflictionCharges", "BloodCharges", "SpiritCharges", "SpiritInfusions"} {
		output.SetN(k, 0.0)
	}

	if modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges") {
		output.Set("FrenzyChargesMin", output.Get("FrenzyChargesMax"))
	}
	if modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges") {
		output.Set("EnduranceChargesMin", output.Get("EnduranceChargesMax"))
	}
	if modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges") {
		output.Set("PowerChargesMin", output.Get("PowerChargesMax"))
	}
	if modDB.Flag(nil, "UsePowerCharges") {
		output.SetN("PowerCharges", overrideOr(modDB, "PowerCharges", output.N("PowerChargesMax")))
	}
	if modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		output.SetN("AbsorptionCharges", math.Max(output.N("PowerCharges"), math.Min(output.N("AbsorptionChargesMax"), output.N("AbsorptionChargesMin"))))
		output.SetN("PowerCharges", 0.0)
	} else {
		output.SetN("PowerCharges", math.Max(output.N("PowerCharges"), math.Min(output.N("PowerChargesMax"), output.N("PowerChargesMin"))))
	}
	output.SetN("RemovablePowerCharges", math.Max(output.N("PowerCharges")-output.N("PowerChargesMin"), 0))
	if modDB.Flag(nil, "UseFrenzyCharges") {
		output.SetN("FrenzyCharges", overrideOr(modDB, "FrenzyCharges", output.N("FrenzyChargesMax")))
	}
	if modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		output.SetN("AfflictionCharges", math.Max(output.N("FrenzyCharges"), math.Min(output.N("AfflictionChargesMax"), output.N("AfflictionChargesMin"))))
		output.SetN("FrenzyCharges", 0.0)
	} else {
		output.SetN("FrenzyCharges", math.Max(output.N("FrenzyCharges"), math.Min(output.N("FrenzyChargesMax"), output.N("FrenzyChargesMin"))))
	}
	output.SetN("RemovableFrenzyCharges", math.Max(output.N("FrenzyCharges")-output.N("FrenzyChargesMin"), 0))
	if modDB.Flag(nil, "UseEnduranceCharges") {
		output.SetN("EnduranceCharges", overrideOr(modDB, "EnduranceCharges", output.N("EnduranceChargesMax")))
		if modDB.Flag(nil, "CanGainBrineCharges") {
			output.Set("BrineCharges", output.Get("BrineChargesMax"))
		}
	}
	if modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		output.SetN("BrutalCharges", math.Max(output.N("EnduranceCharges"), math.Min(output.N("BrutalChargesMax"), output.N("BrutalChargesMin"))))
		output.SetN("EnduranceCharges", 0.0)
	} else {
		output.SetN("EnduranceCharges", math.Max(output.N("EnduranceCharges"), math.Min(output.N("EnduranceChargesMax"), output.N("EnduranceChargesMin"))))
	}
	output.SetN("RemovableEnduranceCharges", math.Max(output.N("EnduranceCharges")-output.N("EnduranceChargesMin"), 0))
	if modDB.Flag(nil, "UseSiphoningCharges") {
		output.SetN("SiphoningCharges", overrideOr(modDB, "SiphoningCharges", output.N("SiphoningChargesMax")))
	}
	if modDB.Flag(nil, "UseChallengerCharges") {
		output.SetN("ChallengerCharges", overrideOr(modDB, "ChallengerCharges", output.N("ChallengerChargesMax")))
	}
	if modDB.Flag(nil, "UseBlitzCharges") {
		output.SetN("BlitzCharges", overrideOr(modDB, "BlitzCharges", output.N("BlitzChargesMax")))
	}
	if actor == env.playerPA {
		output.SetN("InspirationCharges", overrideOr(modDB, "InspirationCharges", output.N("InspirationChargesMax")))
	}
	if modDB.Flag(nil, "UseGhostShrouds") {
		output.SetN("GhostShrouds", overrideOr(modDB, "GhostShrouds", 3))
	}
	output.SetN("BloodCharges", math.Min(overrideOr(modDB, "BloodCharges", output.N("BloodChargesMax")), output.N("BloodChargesMax")))
	output.SetN("SpiritCharges", math.Min(overrideOr(modDB, "SpiritCharges", 0), output.N("SpiritChargesMax")))
	output.SetN("SpiritInfusions", math.Min(overrideOr(modDB, "SpiritInfusion", 0), output.N("SpiritInfusionsMax")))

	output.SetN("CrabBarriers", math.Min(overrideOr(modDB, "CrabBarriers", output.N("CrabBarriersMax")), output.N("CrabBarriersMax")))
	if modDB.Flag(nil, "HaveMaximumPowerCharges") {
		output.Set("PowerCharges", output.Get("PowerChargesMax"))
	}
	if modDB.Flag(nil, "HaveMaximumFrenzyCharges") {
		output.Set("FrenzyCharges", output.Get("FrenzyChargesMax"))
	}
	if modDB.Flag(nil, "HaveMaximumEnduranceCharges") {
		output.Set("EnduranceCharges", output.Get("EnduranceChargesMax"))
	}
	output.SetN("TotalCharges", output.N("PowerCharges")+output.N("FrenzyCharges")+output.N("EnduranceCharges"))
	output.SetN("RemovableTotalCharges", output.N("RemovableEnduranceCharges")+output.N("RemovableFrenzyCharges")+output.N("RemovablePowerCharges"))
	for mult, key := range map[string]string{
		"PowerCharge": "PowerCharges", "PowerChargeMax": "PowerChargesMax",
		"RemovablePowerCharge": "RemovablePowerCharges", "FrenzyCharge": "FrenzyCharges",
		"RemovableFrenzyCharge": "RemovableFrenzyCharges", "EnduranceCharge": "EnduranceCharges",
		"RemovableEnduranceCharge": "RemovableEnduranceCharges", "TotalCharges": "TotalCharges",
		"RemovableTotalCharges": "RemovableTotalCharges", "SiphoningCharge": "SiphoningCharges",
		"ChallengerCharge": "ChallengerCharges", "BlitzCharge": "BlitzCharges",
		"InspirationCharge": "InspirationCharges", "GhostShroud": "GhostShrouds",
		"CrabBarrier": "CrabBarriers", "BrutalCharge": "BrutalCharges",
		"BrineCharge": "BrineCharges", "AbsorptionCharge": "AbsorptionCharges",
		"AfflictionCharge": "AfflictionCharges", "BloodCharge": "BloodCharges",
		"SpiritCharge": "SpiritCharges", "SpiritInfusion": "SpiritInfusions",
	} {
		modDB.Multipliers[mult] = output.N(key)
	}
}

// ActionSpeedMod ports calcs.actionSpeedMod.
func (env *Env) actionSpeedMod(actor *performActor) float64 {
	modDB := actor.db
	minimumActionSpeed := 0.0
	if v, ok := modDB.Max(nil, "MinimumActionSpeed"); ok {
		minimumActionSpeed = v
	}
	maximumActionSpeedReduction, hasMaxRed := modDB.Max(nil, "MaximumActionSpeedReduction")
	actionSpeedMod := 1 + (math.Max(-data.Misc.TemporalChainsEffectCap, modDB.Sum(modparser.Inc, nil, "TemporalChainsActionSpeed"))+modDB.Sum(modparser.Inc, nil, "ActionSpeed"))/100
	actionSpeedMod = math.Max(minimumActionSpeed/100, actionSpeedMod)
	if hasMaxRed {
		actionSpeedMod = math.Min((100-maximumActionSpeedReduction)/100, actionSpeedMod)
	}
	return actionSpeedMod
}
