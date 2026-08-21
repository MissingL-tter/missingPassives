// CalcPerform.lua L612-1120: doActorMisc (misc buffs/debuffs),
// doActorCharges, actionSpeedMod.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// luaOr returns a when the Lua `a or b` keeps a (a non-nil): here the
// Override result when truthy, else the fallback.
func overrideOr(db *modstore.DB, name string, fallback float64) float64 {
	if ov := db.Override(nil, name); truthy(ov) {
		return anyNum(ov)
	}
	return fallback
}

// doActorMisc ports CalcPerform's doActorMisc.
func (env *Env) doActorMisc(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output
	condList := modDB.Conditions
	d := env.Data
	flr := func(v float64) float64 { return math.Floor(v) }

	if env.ModeCombat {
		if env.PlayerMainSkill.BaseSkillModList.Flag(nil, "Cruelty") {
			modDB.Multipliers["Cruelty"] = overrideOr(modDB, "Cruelty", 40)
		}
		// Minimum Rage
		if modDB.Sum("BASE", nil, "MinimumRage") > modDB.Multipliers["Rage"] {
			modDB.Multipliers["Rage"] = modDB.Sum("BASE", nil, "MinimumRage")
		}
		// allied fortify: parent link or party members (party is nil for
		// the replay's corpus; the parent link covers minions)
		alliedFortify := 0.0
		if modDB.Flag(nil, "YourFortifyEqualToParent") && actor.parent != nil {
			alliedFortify = outNum(actor.parent.output, "FortificationStacks")
		}
		if modDB.Sum("BASE", nil, "MinimumFortification") > 0 || alliedFortify > 0 {
			condList["Fortified"] = true
		}
		// Fortify
		if modDB.Flag(nil, "Fortified") || modDB.Sum("BASE", nil, "Multiplier:Fortification") > 0 {
			var skillModList modstore.Store = modDB
			var skillCfg *modstore.Cfg
			if actor.mainSkill != nil {
				skillModList = actor.mainSkill.SkillModList
				skillCfg = actor.mainSkill.SkillCfg
			}
			maxStacks := math.Max(overrideOr(modDB, "MaximumFortification", modDB.Sum("BASE", skillCfg, "MaximumFortification")), alliedFortify)
			minStacks := modDB.Sum("BASE", nil, "MinimumFortification")
			if modDB.Flag(nil, "Condition:HaveMaxFortification") {
				minStacks = maxStacks
			}
			minStacks = math.Min(minStacks, maxStacks)
			stacksBase := maxStacks
			if minStacks > 0 {
				stacksBase = minStacks
			}
			stacks := math.Min(overrideOr(modDB, "FortificationStacks", stacksBase), maxStacks)
			increasedDuration := skillModList.Sum("INC", nil, "FortifyDuration")
			output["MaximumFortification"] = maxStacks
			output["MinimumFortification"] = minStacks
			output["RemovableFortification"] = math.Min(maxStacks-minStacks, overrideOr(modDB, "FortificationStacks", maxStacks)-minStacks)
			output["FortificationStacks"] = stacks
			output["FortificationStacksOver20"] = math.Min(math.Max(0, stacks-20), maxStacks-20)
			fortifyDurBase := d.Misc.FortifyBaseDuration
			if ov := skillModList.Override(nil, "FortifyDuration"); truthy(ov) {
				fortifyDurBase = anyNum(ov)
			}
			output["FortifyDuration"] = fortifyDurBase * (1 + increasedDuration/100)
			output["FortificationEffect"] = "0" // string, shown for Willowgift
			if !modDB.Flag(nil, "Condition:NoFortificationMitigation") {
				output["FortificationEffect"] = stacks
				modDB.AddMod(newMod("DamageTakenWhenHit", "MORE", -stacks, "Fortification"))
			}
			if stacks >= maxStacks {
				modDB.AddMod(newMod("Condition:HaveMaximumFortification", "FLAG", true, ""))
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
					curFlaskEffectInc := anyNum(item.In.FlaskData["effectInc"]) + modDB.Sum("INC", &modstore.Cfg{Actor: "player"}, "FlaskEffect")
					if item.In.Rarity == "MAGIC" {
						curFlaskEffectInc += modDB.Sum("INC", &modstore.Cfg{Actor: "player"}, "MagicUtilityFlaskEffect")
					}
					if flaskEffectInc < curFlaskEffectInc/100 {
						flaskEffectInc = curFlaskEffectInc / 100
					}
				}
			}
			onslaughtEffectInc := modDB.Sum("INC", nil, "OnslaughtEffect", "BuffEffectOnSelf") / 100
			var effect float64
			if onslaughtFromFlask {
				effect = flr(20 * (1 + flaskEffectInc + onslaughtEffectInc))
			} else {
				effect = flr(20 * (1 + onslaughtEffectInc))
			}
			modDB.AddMod(newMod("Speed", "INC", effect, "Onslaught", modparser.ModFlag.Attack))
			modDB.AddMod(newMod("Speed", "INC", effect, "Onslaught", modparser.ModFlag.Cast))
			modDB.AddMod(newMod("MovementSpeed", "INC", effect, "Onslaught"))
		}
		if truthy(condList["AffectedByArcaneSurge"]) || modDB.Flag(nil, "Condition:ArcaneSurge") {
			condList["AffectedByArcaneSurge"] = true
			effect := 1 + modDB.Sum("INC", nil, "ArcaneSurgeEffect", "BuffEffectOnSelf")/100
			effect = effect + modDB.Sum("INC", &modstore.Cfg{Actor: "player"}, "FlaskEffect")/100*modDB.Sum("BASE", nil, "FlaskEffectToArcaneSurgeEffect")/100
			manaRegen := 30.0
			if v, ok := modDB.Max(nil, "ArcaneSurgeManaRegen"); ok {
				manaRegen = v
			}
			modDB.AddMod(newMod("ManaRegen", "INC", manaRegen*effect, "Arcane Surge"))
			castSpeedBase := 20.0
			if v, ok := modDB.Max(nil, "ArcaneSurgeCastSpeed"); ok {
				castSpeedBase = v
			}
			arcaneSurgeCastSpeed := castSpeedBase * effect
			modDB.AddMod(newMod("Speed", "INC", arcaneSurgeCastSpeed, "Arcane Surge", modparser.ModFlag.Cast))
			if modDB.Flag(nil, "ArcaneSurgeCastSpeedToMovementSpeed") {
				modDB.AddMod(newMod("MovementSpeed", "INC", arcaneSurgeCastSpeed, "Arcane Surge"))
			}
			arcaneSurgeDamage := 0.0
			if v, ok := modDB.Max(nil, "ArcaneSurgeDamage"); ok {
				arcaneSurgeDamage = v
			}
			if arcaneSurgeDamage != 0 {
				modDB.AddMod(newMod("Damage", "MORE", arcaneSurgeDamage*effect, "Arcane Surge", modparser.ModFlag.Spell))
			}
			arcaneSurgeLifeRegen := modDB.Sum("BASE", nil, "ArcaneSurgeAlsoLifeRegen")
			if arcaneSurgeLifeRegen > 0 {
				modDB.AddMod(newMod("LifeRegen", "INC", arcaneSurgeLifeRegen*effect, "Arcane Surge"))
			}
		}
		if modDB.Flag(nil, "Fanaticism") && actor.mainSkill != nil && actor.mainSkill.SkillFlags["selfCast"] {
			effect := flr(75 * (1 + modDB.Sum("INC", nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newMod("Speed", "MORE", effect, "Fanaticism", modparser.ModFlag.Cast))
			modDB.AddMod(newMod("Cost", "MORE", -effect, "Fanaticism", modparser.ModFlag.Cast))
			modDB.AddMod(newMod("AreaOfEffect", "INC", effect, "Fanaticism", modparser.ModFlag.Cast))
		}
		if modDB.Flag(nil, "Condition:CanGainSpiritInfusion") {
			globalEffectTag := modparser.Tag{"type": "GlobalEffect", "effectType": "Buff", "effectName": "Spirit Infusion", "unscalable": true}
			multiplierTag := modparser.Tag{"type": "Multiplier", "var": "SpiritInfusion"}
			modDB.AddMod(newMod("EnergyShieldRechargeFaster", "INC", 15.0, "Spirit Infusion", multiplierTag, globalEffectTag))
			modDB.AddMod(newMod("Damage", "MORE", 5.0, "Spirit Infusion", nil, modparser.KeywordFlag.Spell, modparser.Tag{"type": "SkillType", "skillType": modparser.SkillType.Channel}, multiplierTag, globalEffectTag))
			modDB.AddMod(newMod("Cost", "MORE", 10.0, "Spirit Infusion", nil, modparser.KeywordFlag.Spell, modparser.Tag{"type": "SkillType", "skillType": modparser.SkillType.Channel}, multiplierTag, globalEffectTag))
		}
		if modDB.Flag(nil, "UnholyMight") {
			effect := 1 + modDB.Sum("INC", nil, "BuffEffectOnSelf")/100
			modDB.AddMod(newMod("PhysicalDamageConvertToChaos", "BASE", flr(100*effect), "Unholy Might"))
			modDB.AddMod(newMod("Condition:CanWither", "FLAG", true, "Unholy Might"))
		}
		if modDB.Flag(nil, "ShepherdOfSouls") {
			modDB.AddMod(newMod("SoulCost", "MORE", -80.0, "Shepherd of Souls", modparser.Tag{"type": "SkillType", "skillType": modparser.SkillType.Vaal}, modparser.Tag{"type": "SkillType", "skillType": modparser.SkillType.Aura, "neg": true}))
			modDB.AddMod(newMod("SoulCost", "INC", 100.0, "Shepherd of Souls", modparser.Tag{"type": "SkillType", "skillType": modparser.SkillType.Vaal}, modparser.Tag{"type": "SkillType", "skillType": modparser.SkillType.Aura, "neg": true}, modparser.Tag{"type": "Multiplier", "var": "VaalSkillsUsedInPast8Seconds"}))
		}
		if modDB.Flag(nil, "ChaoticMight") {
			effect := flr(30 * (1 + modDB.Sum("INC", nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newMod("PhysicalDamageGainAsChaos", "BASE", effect, "Chaotic Might"))
		}
		if modDB.Flag(nil, "Tailwind") {
			effect := flr(8 * (1 + modDB.Sum("INC", nil, "TailwindEffectOnSelf", "BuffEffectOnSelf")/100))
			modDB.AddMod(newMod("ActionSpeed", "INC", effect, "Tailwind"))
		}
		if modDB.Flag(nil, "Condition:TotemTailwind") {
			modDB.AddMod(newMod("TotemActionSpeed", "INC", 8.0, "Tailwind"))
		}
		if modDB.Flag(nil, "Adrenaline") {
			effectMod := 1 + modDB.Sum("INC", nil, "BuffEffectOnSelf")/100
			modDB.AddMod(newMod("Damage", "INC", flr(100*effectMod), "Adrenaline"))
			modDB.AddMod(newMod("Speed", "INC", flr(25*effectMod), "Adrenaline", modparser.ModFlag.Attack))
			modDB.AddMod(newMod("Speed", "INC", flr(25*effectMod), "Adrenaline", modparser.ModFlag.Cast))
			modDB.AddMod(newMod("MovementSpeed", "INC", flr(25*effectMod), "Adrenaline"))
			modDB.AddMod(newMod("PhysicalDamageReduction", "BASE", flr(10*effectMod), "Adrenaline"))
		}
		if modDB.Flag(nil, "Condition:WildSavagery") && modDB.Flag(nil, "WildSavagery") {
			modDB.AddMod(newMod("PhysicalDamage", "INC", 100.0, "Wild Savagery"))
			modDB.AddMod(newMod("ActionSpeed", "INC", 10.0, "Wild Savagery"))
			modDB.AddMod(newMod("IgnoreEnemyPhysicalDamageReduction", "FLAG", true, "Wild Savagery"))
			modDB.AddMod(newMod("StunImmune", "FLAG", true, "Wild Savagery"))
		}
		if modDB.Flag(nil, "Convergence") {
			effect := flr(30 * (1 + modDB.Sum("INC", nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newMod("ElementalDamage", "MORE", effect, "Convergence"))
		}
		if modDB.Flag(nil, "HerEmbrace") {
			condList["HerEmbrace"] = true
			modDB.AddMod(newMod("AvoidStun", "BASE", 100.0, "Her Embrace"))
			modDB.AddMod(newMod("PhysicalDamageGainAsFire", "BASE", 123.0, "Her Embrace", modparser.ModFlag.Sword))
			modDB.AddMod(newMod("AvoidFreeze", "BASE", 100.0, "Her Embrace"))
			modDB.AddMod(newMod("AvoidChill", "BASE", 100.0, "Her Embrace"))
			modDB.AddMod(newMod("AvoidIgnite", "BASE", 100.0, "Her Embrace"))
			modDB.AddMod(newMod("Speed", "INC", 20.0, "Her Embrace", modparser.ModFlag.Attack))
			modDB.AddMod(newMod("Speed", "INC", 20.0, "Her Embrace", modparser.ModFlag.Cast))
			modDB.AddMod(newMod("MovementSpeed", "INC", 20.0, "Her Embrace"))
		}
		if modDB.Flag(nil, "Condition:OnConsecratedGround") {
			effect := 1 + modDB.Sum("INC", nil, "ConsecratedGroundEffect")/100
			modDB.AddMod(newMod("LifeRegenPercent", "BASE", 5*effect, "Consecrated Ground"))
			modDB.AddMod(newMod("CurseEffectOnSelf", "INC", -50*effect, "Consecrated Ground"))
			modDB.AddMod(newMod("Accuracy", "INC", flr(modDB.Sum("INC", nil, "ConsecratedGroundAlsoAccuracy")*effect), "Consecrated Ground"))
		}
		if modDB.Flag(nil, "Condition:PhantasmalMight") {
			limit := outNum(output, "ActivePhantasmLimit")
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
			inc := modDB.Sum("INC", nil, "ElusiveEffect", "BuffEffectOnSelf")
			if actor.mainSkill.SkillModList.Flag(nil, "SupportedByNightblade") {
				inc = inc + modDB.Sum("INC", nil, "NightbladeSupportedElusiveEffect")
			}
			inc = inc + maxSkillInc
			elusiveEffectMod := (1 + inc/100) * modDB.More(nil, "ElusiveEffect", "BuffEffectOnSelf") * 100
			elusiveEffectMinThreshold := overrideOr(modDB, "ElusiveEffectMinThreshold", 0)
			elusiveEffectIncreaseDuration := modDB.Sum("BASE", nil, "ElusiveEffectIncreaseDuration")
			peakElusiveEffect := elusiveEffectMod
			if elusiveEffectIncreaseDuration > 0 {
				elusiveEffectChangeRate := 20 / (1 + modDB.Sum("INC", nil, "ElusiveEffectLossSlower")/100)
				peakElusiveEffect = elusiveEffectMod + elusiveEffectChangeRate*elusiveEffectIncreaseDuration
				elusiveEffectDecreaseDuration := (peakElusiveEffect - elusiveEffectMinThreshold) / elusiveEffectChangeRate
				totalElusiveEffectDuration := elusiveEffectIncreaseDuration + elusiveEffectDecreaseDuration
				averageIncreaseEffect := (elusiveEffectMod + peakElusiveEffect) / 2
				averageDecreaseEffect := (peakElusiveEffect + elusiveEffectMinThreshold) / 2
				output["ElusiveEffectMod"] = (averageIncreaseEffect*elusiveEffectIncreaseDuration + averageDecreaseEffect*elusiveEffectDecreaseDuration) / totalElusiveEffectDuration
			} else {
				output["ElusiveEffectMod"] = (elusiveEffectMod + elusiveEffectMinThreshold) / 2
			}
			modDB.AddMod(newMod("ElusiveEffect", "INC", maxSkillInc, "Max Skill Effect"))
			if ov := modDB.Override(nil, "ElusiveEffect"); truthy(ov) {
				output["ElusiveEffectMod"] = math.Min(anyNum(ov), peakElusiveEffect)
			}
			effect := outNum(output, "ElusiveEffectMod") / 100
			condList["Elusive"] = true
			modDB.AddMod(newMod("AvoidAllDamageFromHitsChance", "BASE", flr(15*effect), "Elusive"))
			modDB.AddMod(newMod("MovementSpeed", "INC", flr(30*effect), "Elusive"))
		}
		if _, ok := modDB.Max(nil, "WitherEffectStack"); ok {
			modDB.AddMod(newMod("Condition:CanWither", "FLAG", true, "Config"))
			effect, _ := modDB.Max(nil, "WitherEffectStack")
			enemyDB.AddMod(newMod("ChaosDamageTaken", "INC", effect, "Withered", modparser.Tag{"type": "Multiplier", "var": "WitheredStack", "limit": 15.0}))
		}
		if modDB.Flag(nil, "Condition:CanBeWithered") {
			effect := 6 * (100 + modDB.Sum("INC", nil, "WitherEffectOnSelf")) / 100 * modDB.More(nil, "WitherEffectOnSelf")
			modDB.AddMod(newMod("ChaosDamageTaken", "INC", effect, "Withered", modparser.Tag{"type": "Multiplier", "var": "WitheredStack", "limit": 15.0}))
		}
		if modDB.Flag(nil, "Excommunicated") {
			modDB.AddMod(newMod("ChaosDamage", "MORE", -100.0, "Excommunicated"))
		}
		if modDB.Flag(nil, "Blind") && !modDB.Flag(nil, "CannotBeBlinded") {
			if !modDB.Flag(nil, "UnaffectedByBlind") {
				effect := 1 + modDB.Sum("INC", nil, "BlindEffect", "BuffEffectOnSelf")/100
				if ov := modDB.Override(nil, "BlindEffect"); truthy(ov) {
					effect = math.Min(anyNum(ov)/100, effect)
				}
				modDB.AddMod(newMod("Accuracy", "MORE", flr(-20*effect), "Blind"))
				modDB.AddMod(newMod("Evasion", "MORE", flr(-20*effect), "Blind"))
			}
		}
		if modDB.Flag(nil, "Chill") {
			ail := d.NonDamagingAilment["Chill"]
			// Lua: m_max(Sum, Override) — errors when the override is nil,
			// so the config always supplies ChillVal alongside the flag;
			// nil-as-0 here (the `or default` after m_max is dead code)
			chillValue := math.Max(modDB.Sum("BASE", nil, "SelfChillOverride"), anyNum(modDB.Override(nil, "ChillVal")))
			totalChillSelfEffect := Mod(modDB, nil, "SelfChillEffect")
			avoidChill := 0.0
			if modDB.Flag(nil, "ChillImmune", "ElementalAilmentImmune") {
				avoidChill = 100
			} else {
				sum := modDB.Sum("BASE", nil, "AvoidChill", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
					sum += modDB.Sum("BASE", nil, "AvoidShock")
				}
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					sum += modDB.Sum("BASE", nil, "SpellSuppressionChance") / 2
				}
				avoidChill = flr(math.Min(sum, 100))
			}
			chillMax := ail.Max
			if ov := modDB.Override(nil, "ChillMax"); truthy(ov) {
				chillMax = anyNum(ov)
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
				modDB.AddMod(newMod("ColdDamageTaken", "INC", effect*-sign, "Bonechill"))
			}
			sign := -1.0
			if modDB.Flag(nil, "SelfChillEffectIsReversed") {
				sign = 1
			}
			modDB.AddMod(newMod("ActionSpeed", "INC", effect*sign, "Chill"))
		}
		if modDB.Flag(nil, "Shock") {
			ail := d.NonDamagingAilment["Shock"]
			shockValue := math.Max(modDB.Sum("BASE", nil, "SelfShockOverride"), anyNum(modDB.Override(nil, "ShockVal")))
			totalShockSelfEffect := Mod(modDB, nil, "SelfShockEffect")
			avoidShock := 0.0
			if modDB.Flag(nil, "ShockImmune", "ElementalAilmentImmune") {
				avoidShock = 100
			} else {
				sum := modDB.Sum("BASE", nil, "AvoidShock", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					sum += modDB.Sum("BASE", nil, "SpellSuppressionChance") / 2
				}
				avoidShock = flr(math.Min(sum, 100))
			}
			shockMax := ail.Max
			if ov := modDB.Override(nil, "ShockMax"); truthy(ov) {
				shockMax = anyNum(ov)
			}
			effect := 0.0
			if avoidShock != 100 {
				effect = math.Min(math.Max(flr(shockValue*totalShockSelfEffect), 0), shockMax)
			}
			modDB.AddMod(newMod("DamageTaken", "INC", effect, "Shock"))
		}
		if modDB.Flag(nil, "Scorch") {
			ail := d.NonDamagingAilment["Scorch"]
			scorchValue := math.Max(modDB.Sum("BASE", nil, "SelfScorchOverride"), anyNum(modDB.Override(nil, "ScorchVal")))
			totalScorchSelfEffect := Mod(modDB, nil, "SelfScorchEffect")
			avoidScorch := 0.0
			if modDB.Flag(nil, "ScorchImmune", "ElementalAilmentImmune") {
				avoidScorch = 100
			} else {
				sum := modDB.Sum("BASE", nil, "AvoidScorch", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
					sum += modDB.Sum("BASE", nil, "AvoidShock")
				}
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					sum += modDB.Sum("BASE", nil, "SpellSuppressionChance") / 2
				}
				avoidScorch = flr(math.Min(sum, 100))
			}
			scorchMax := ail.Max
			if ov := modDB.Override(nil, "ScorchMax"); truthy(ov) {
				scorchMax = anyNum(ov)
			}
			effect := 0.0
			if avoidScorch != 100 {
				effect = math.Min(math.Max(flr(scorchValue*totalScorchSelfEffect), 0), scorchMax)
			}
			modDB.AddMod(newMod("ElementalResist", "BASE", -effect, "Scorch"))
		}
		if modDB.Flag(nil, "Freeze") {
			effect := math.Max(flr(70*Mod(modDB, nil, "SelfChillEffect")), 0)
			modDB.AddMod(newMod("ActionSpeed", "INC", -effect, "Freeze"))
		}
		if modDB.Flag(nil, "CanLeechLifeOnFullLife") && !modDB.Flag(nil, "GhostReaver") {
			condList["Leeching"] = true
			condList["LeechingLife"] = true
		}
		if modDB.Flag(nil, "CanLeechEnergyShieldOnFullEnergyShield") {
			condList["Leeching"] = true
			condList["LeechingEnergyShield"] = true
		}
		if modDB.Flag(nil, "Condition:CanGainRage") || modDB.Sum("BASE", nil, "RageRegen") > 0 {
			// skillCfg is an undefined local in the reference: nil
			maxStacks := flr(modDB.Sum("BASE", nil, "MaximumRage") * modDB.More(nil, "MaximumRage"))
			minStacks := math.Min(modDB.Sum("BASE", nil, "MinimumRage"), maxStacks)
			rageConfig := modDB.Sum("BASE", nil, "Multiplier:RageStack")
			stacks := math.Min(rageConfig, maxStacks)
			if minStacks > 0 && stacks < minStacks {
				stacks = minStacks
			}
			stacks = math.Max(stacks, 0)
			output["RageEffect"] = flr(stacks * Mod(modDB, nil, "RageEffect"))
			modDB.AddMod(newMod("Multiplier:RageEffect", "BASE", outNum(output, "RageEffect"), "Base"))
			output["Rage"] = stacks
			output["MaximumRage"] = maxStacks
			modDB.AddMod(newMod("Multiplier:Rage", "BASE", stacks, "Base"))
			if modDB.Flag(nil, "Condition:RageSpellDamage") {
				modDB.AddMod(newMod("Damage", "MORE", outNum(output, "RageEffect"), "Rage", modparser.ModFlag.Spell))
			} else {
				modDB.AddMod(newMod("Damage", "MORE", outNum(output, "RageEffect"), "Rage", modparser.ModFlag.Attack))
			}
			if stacks == maxStacks {
				modDB.AddMod(newMod("Condition:HaveMaximumRage", "FLAG", true, ""))
			}
			output["InherentRageLossDelay"] = 2 + modDB.Sum("BASE", nil, "InherentRageLossDelay")
			if !modDB.Flag(nil, "InherentRageLossIsPrevented") {
				output["InherentRageLoss"] = 10 * (1 + modDB.Sum("INC", nil, "InherentRageLoss")/100)
			} else {
				output["InherentRageLoss"] = 0.0
			}
		}
		if anyNum(env.ConfigInput["multiplierManaBurnStacks"]) > 0 {
			maxManaBurn := modDB.Sum("BASE", nil, "MaxManaBurnStacks")
			if maxManaBurn == 0 {
				maxManaBurn = 9999
			}
			manaBurnStacks := math.Min(anyNum(env.ConfigInput["multiplierManaBurnStacks"]), maxManaBurn)
			modDB.AddMod(newMod("Multiplier:ManaBurnStacks", "BASE", manaBurnStacks, "Config"))
			manaBurnStacks = manaBurnStacks + modDB.Sum("BASE", &modstore.Cfg{Actor: "player"}, "EffectiveManaBurnStacks")
			if modDB.Flag(nil, "Condition:WeepingWoundsInsteadOfManaBurn") {
				modDB.AddMod(newMod("Multiplier:WeepingWoundsStacks", "BASE", manaBurnStacks, "Config"))
			} else {
				modDB.AddMod(newMod("Multiplier:EffectiveManaBurnStacks", "BASE", manaBurnStacks, "Config"))
			}
		}
		if modDB.Sum("BASE", nil, "CoveredInAshEffect") > 0 {
			effect := modDB.Sum("BASE", nil, "CoveredInAshEffect")
			enemyDB.AddMod(newMod("FireDamageTaken", "INC", math.Min(effect, 20), "Covered in Ash"))
		}
		if modDB.Sum("BASE", nil, "CoveredInFrostEffect") > 0 {
			effect := modDB.Sum("BASE", nil, "CoveredInFrostEffect")
			enemyDB.AddMod(newMod("ColdDamageTaken", "INC", math.Min(effect, 20), "Covered in Frost"))
		}
		if modDB.Flag(nil, "HasMalediction") {
			modDB.AddMod(newMod("DamageTaken", "INC", 10.0, "Malediction"))
			modDB.AddMod(newMod("Damage", "INC", -10.0, "Malediction"))
		}
		if modDB.Flag(nil, "HasMaddeningPresence") {
			modDB.AddMod(newMod("ActionSpeed", "INC", -10.0, "Maddening Presence"))
			modDB.AddMod(newMod("Damage", "INC", -10.0, "Maddening Presence"))
		}
		if modDB.Flag(nil, "HasShapersPresence") {
			modDB.AddMod(newMod("BuffExpireFaster", "MORE", -20.0, "Shapers Presence"))
		}
		if modDB.Flag(nil, "Condition:CanHaveSoulEater") {
			max := overrideOr(modDB, "SoulEaterMax", modDB.Sum("BASE", nil, "SoulEaterMax"))
			modDB.AddMod(newMod("Speed", "INC", 5.0, "Base", modparser.ModFlag.Attack, modparser.Tag{"type": "Multiplier", "var": "SoulEaterStack", "limit": max}))
			modDB.AddMod(newMod("Speed", "INC", 5.0, "Base", modparser.ModFlag.Cast, modparser.Tag{"type": "Multiplier", "var": "SoulEaterStack", "limit": max}))
		}
	}

	// Process enemy modifiers
	applyEnemyModifiers(actor, false)
}

// doActorCharges ports CalcPerform's doActorCharges.
func (env *Env) doActorCharges(actor *performActor) {
	modDB := actor.db
	output := actor.output
	setMax := func(k string, v float64) { output[k] = math.Max(v, 0) }

	setMax("PowerChargesMin", modDB.Sum("BASE", nil, "PowerChargesMin"))
	output["PowerChargesMax"] = overrideOr(modDB, "PowerChargesMax", math.Max(modDB.Sum("BASE", nil, "PowerChargesMax"), 0))
	output["PowerChargesDuration"] = math.Floor(modDB.Sum("BASE", nil, "ChargeDuration") * Mod(modDB, nil, "PowerChargesDuration", "ChargeDuration"))
	if modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges") {
		source := modDB.Mods["MaximumFrenzyChargesIsMaximumPowerCharges"][0].Source
		modDB.ReplaceMod(newMod("FrenzyChargesMax", "OVERRIDE", outNum(output, "PowerChargesMax"), source))
	}
	setMax("FrenzyChargesMin", modDB.Sum("BASE", nil, "FrenzyChargesMin"))
	fcBase := modDB.Sum("BASE", nil, "FrenzyChargesMax")
	if modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges") {
		fcBase = outNum(output, "PowerChargesMax")
	}
	output["FrenzyChargesMax"] = overrideOr(modDB, "FrenzyChargesMax", math.Max(fcBase, 0))
	output["FrenzyChargesDuration"] = math.Floor(modDB.Sum("BASE", nil, "ChargeDuration") * Mod(modDB, nil, "FrenzyChargesDuration", "ChargeDuration"))
	if modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges") {
		source := modDB.Mods["MaximumEnduranceChargesIsMaximumFrenzyCharges"][0].Source
		modDB.ReplaceMod(newMod("EnduranceChargesMax", "OVERRIDE", outNum(output, "FrenzyChargesMax"), source))
	}
	setMax("EnduranceChargesMin", modDB.Sum("BASE", nil, "EnduranceChargesMin"))
	ecBase := modDB.Sum("BASE", nil, "EnduranceChargesMax")
	if modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges") {
		ecBase = outNum(output, "FrenzyChargesMax")
	}
	// (partyMembers link is nil for the replay corpus)
	output["EnduranceChargesMax"] = overrideOr(modDB, "EnduranceChargesMax", math.Max(ecBase, 0))
	output["EnduranceChargesDuration"] = math.Floor(modDB.Sum("BASE", nil, "ChargeDuration") * Mod(modDB, nil, "EnduranceChargesDuration", "ChargeDuration"))
	setMax("SiphoningChargesMax", modDB.Sum("BASE", nil, "SiphoningChargesMax"))
	setMax("ChallengerChargesMax", modDB.Sum("BASE", nil, "ChallengerChargesMax"))
	setMax("BlitzChargesMax", modDB.Sum("BASE", nil, "BlitzChargesMax"))
	setMax("InspirationChargesMax", modDB.Sum("BASE", nil, "InspirationChargesMax"))
	setMax("CrabBarriersMax", modDB.Sum("BASE", nil, "CrabBarriersMax"))
	brutalMin := 0.0
	if modDB.Flag(nil, "MinimumEnduranceChargesEqualsMinimumBrutalCharges") {
		if modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges") {
			brutalMin = outNum(output, "EnduranceChargesMax")
		} else {
			brutalMin = outNum(output, "EnduranceChargesMin")
		}
	}
	setMax("BrutalChargesMin", brutalMin)
	brutalMax := 0.0
	if modDB.Flag(nil, "MaximumEnduranceChargesEqualsMaximumBrutalCharges") {
		brutalMax = outNum(output, "EnduranceChargesMax")
	}
	setMax("BrutalChargesMax", brutalMax)
	setMax("BrineChargesMax", outNum(output, "EnduranceChargesMax"))
	absMin := 0.0
	if modDB.Flag(nil, "MinimumPowerChargesEqualsMinimumAbsorptionCharges") {
		if modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges") {
			absMin = outNum(output, "PowerChargesMax")
		} else {
			absMin = outNum(output, "PowerChargesMin")
		}
	}
	setMax("AbsorptionChargesMin", absMin)
	absMax := 0.0
	if modDB.Flag(nil, "MaximumPowerChargesEqualsMaximumAbsorptionCharges") {
		absMax = outNum(output, "PowerChargesMax")
	}
	setMax("AbsorptionChargesMax", absMax)
	afflMin := 0.0
	if modDB.Flag(nil, "MinimumFrenzyChargesEqualsMinimumAfflictionCharges") {
		if modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges") {
			afflMin = outNum(output, "FrenzyChargesMax")
		} else {
			afflMin = outNum(output, "FrenzyChargesMin")
		}
	}
	setMax("AfflictionChargesMin", afflMin)
	afflMax := 0.0
	if modDB.Flag(nil, "MaximumFrenzyChargesEqualsMaximumAfflictionCharges") {
		afflMax = outNum(output, "FrenzyChargesMax")
	}
	setMax("AfflictionChargesMax", afflMax)
	setMax("BloodChargesMax", modDB.Sum("BASE", nil, "BloodChargesMax"))
	setMax("SpiritChargesMax", modDB.Sum("BASE", nil, "SpiritChargesMax"))
	sim := modDB.Sum("BASE", nil, "SpiritInfusionsMax")
	if modDB.Flag(nil, "Condition:CanGainSpiritInfusion") {
		sim += 10
	}
	output["SpiritInfusionsMax"] = sim

	// Initialize charges
	for _, k := range []string{"PowerCharges", "FrenzyCharges", "EnduranceCharges", "SiphoningCharges",
		"ChallengerCharges", "BlitzCharges", "InspirationCharges", "GhostShrouds", "BrutalCharges",
		"BrineCharges", "AbsorptionCharges", "AfflictionCharges", "BloodCharges", "SpiritCharges", "SpiritInfusions"} {
		output[k] = 0.0
	}

	if modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges") {
		output["FrenzyChargesMin"] = output["FrenzyChargesMax"]
	}
	if modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges") {
		output["EnduranceChargesMin"] = output["EnduranceChargesMax"]
	}
	if modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges") {
		output["PowerChargesMin"] = output["PowerChargesMax"]
	}
	if modDB.Flag(nil, "UsePowerCharges") {
		output["PowerCharges"] = overrideOr(modDB, "PowerCharges", outNum(output, "PowerChargesMax"))
	}
	if modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		output["AbsorptionCharges"] = math.Max(outNum(output, "PowerCharges"), math.Min(outNum(output, "AbsorptionChargesMax"), outNum(output, "AbsorptionChargesMin")))
		output["PowerCharges"] = 0.0
	} else {
		output["PowerCharges"] = math.Max(outNum(output, "PowerCharges"), math.Min(outNum(output, "PowerChargesMax"), outNum(output, "PowerChargesMin")))
	}
	output["RemovablePowerCharges"] = math.Max(outNum(output, "PowerCharges")-outNum(output, "PowerChargesMin"), 0)
	if modDB.Flag(nil, "UseFrenzyCharges") {
		output["FrenzyCharges"] = overrideOr(modDB, "FrenzyCharges", outNum(output, "FrenzyChargesMax"))
	}
	if modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		output["AfflictionCharges"] = math.Max(outNum(output, "FrenzyCharges"), math.Min(outNum(output, "AfflictionChargesMax"), outNum(output, "AfflictionChargesMin")))
		output["FrenzyCharges"] = 0.0
	} else {
		output["FrenzyCharges"] = math.Max(outNum(output, "FrenzyCharges"), math.Min(outNum(output, "FrenzyChargesMax"), outNum(output, "FrenzyChargesMin")))
	}
	output["RemovableFrenzyCharges"] = math.Max(outNum(output, "FrenzyCharges")-outNum(output, "FrenzyChargesMin"), 0)
	if modDB.Flag(nil, "UseEnduranceCharges") {
		output["EnduranceCharges"] = overrideOr(modDB, "EnduranceCharges", outNum(output, "EnduranceChargesMax"))
		if modDB.Flag(nil, "CanGainBrineCharges") {
			output["BrineCharges"] = output["BrineChargesMax"]
		}
	}
	if modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		output["BrutalCharges"] = math.Max(outNum(output, "EnduranceCharges"), math.Min(outNum(output, "BrutalChargesMax"), outNum(output, "BrutalChargesMin")))
		output["EnduranceCharges"] = 0.0
	} else {
		output["EnduranceCharges"] = math.Max(outNum(output, "EnduranceCharges"), math.Min(outNum(output, "EnduranceChargesMax"), outNum(output, "EnduranceChargesMin")))
	}
	output["RemovableEnduranceCharges"] = math.Max(outNum(output, "EnduranceCharges")-outNum(output, "EnduranceChargesMin"), 0)
	if modDB.Flag(nil, "UseSiphoningCharges") {
		output["SiphoningCharges"] = overrideOr(modDB, "SiphoningCharges", outNum(output, "SiphoningChargesMax"))
	}
	if modDB.Flag(nil, "UseChallengerCharges") {
		output["ChallengerCharges"] = overrideOr(modDB, "ChallengerCharges", outNum(output, "ChallengerChargesMax"))
	}
	if modDB.Flag(nil, "UseBlitzCharges") {
		output["BlitzCharges"] = overrideOr(modDB, "BlitzCharges", outNum(output, "BlitzChargesMax"))
	}
	if actor == env.playerPA {
		output["InspirationCharges"] = overrideOr(modDB, "InspirationCharges", outNum(output, "InspirationChargesMax"))
	}
	if modDB.Flag(nil, "UseGhostShrouds") {
		output["GhostShrouds"] = overrideOr(modDB, "GhostShrouds", 3)
	}
	output["BloodCharges"] = math.Min(overrideOr(modDB, "BloodCharges", outNum(output, "BloodChargesMax")), outNum(output, "BloodChargesMax"))
	output["SpiritCharges"] = math.Min(overrideOr(modDB, "SpiritCharges", 0), outNum(output, "SpiritChargesMax"))
	output["SpiritInfusions"] = math.Min(overrideOr(modDB, "SpiritInfusion", 0), outNum(output, "SpiritInfusionsMax"))

	output["CrabBarriers"] = math.Min(overrideOr(modDB, "CrabBarriers", outNum(output, "CrabBarriersMax")), outNum(output, "CrabBarriersMax"))
	if modDB.Flag(nil, "HaveMaximumPowerCharges") {
		output["PowerCharges"] = output["PowerChargesMax"]
	}
	if modDB.Flag(nil, "HaveMaximumFrenzyCharges") {
		output["FrenzyCharges"] = output["FrenzyChargesMax"]
	}
	if modDB.Flag(nil, "HaveMaximumEnduranceCharges") {
		output["EnduranceCharges"] = output["EnduranceChargesMax"]
	}
	output["TotalCharges"] = outNum(output, "PowerCharges") + outNum(output, "FrenzyCharges") + outNum(output, "EnduranceCharges")
	output["RemovableTotalCharges"] = outNum(output, "RemovableEnduranceCharges") + outNum(output, "RemovableFrenzyCharges") + outNum(output, "RemovablePowerCharges")
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
		modDB.Multipliers[mult] = outNum(output, key)
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
	actionSpeedMod := 1 + (math.Max(-env.Data.Misc.TemporalChainsEffectCap, modDB.Sum("INC", nil, "TemporalChainsActionSpeed"))+modDB.Sum("INC", nil, "ActionSpeed"))/100
	actionSpeedMod = math.Max(minimumActionSpeed/100, actionSpeedMod)
	if hasMaxRed {
		actionSpeedMod = math.Min((100-maximumActionSpeedReduction)/100, actionSpeedMod)
	}
	return actionSpeedMod
}
