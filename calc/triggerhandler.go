// CalcTriggers.lua L401-899: defaultTriggerHandler, the shared body every
// trigger config funnels into.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// defaultTriggerHandler ports the local of the same name (L401).
func (env *Env) defaultTriggerHandler(config *triggerConfig) {
	actor := config.actor
	output := actor.output
	source := config.source
	triggeredSkills := config.triggeredSkills
	trigRate := config.trigRate
	uuid := ""
	d := env.Data
	main := actor.mainSkill

	// Only attacks using the granting weapon can activate source-weapon triggers
	sourceWeaponFlag := ""
	if config.sourceWeapon && main.SocketGroup != nil {
		if slot := str(main.SocketGroup.KV["slot"]); slot != "" {
			if strings.HasPrefix(slot, "Weapon 2") {
				sourceWeaponFlag = "weapon2Attack"
			} else {
				sourceWeaponFlag = "weapon1Attack"
			}
		}
	}

	// Find trigger skill and triggered skills
	if config.triggeredSkillCond != nil || config.triggerSkillCond != nil {
		for _, skill := range env.PlayerActiveSkills {
			if config.triggerSkillCond != nil && config.triggerSkillCond(env, skill) &&
				(sourceWeaponFlag == "" || skill.SkillFlags[sourceWeaponFlag]) &&
				(!isTriggered(skill) || main.SkillFlags["globalTrigger"] || config.allowTriggered) &&
				skill != main {
				source, trigRate, uuid = env.findTriggerSkill(skill, source, trigRate, nil)
			}
			if config.triggeredSkillCond != nil && config.triggeredSkillCond(env, skill) {
				triggeredSkills = append(triggeredSkills, env.packageSkillDataForSimulation(skill))
			}
		}
	}
	if !(len(triggeredSkills) > 0 || config.triggeredSkillCond == nil) {
		return
	}
	if source == nil && !(main.SkillFlags["globalTrigger"] && config.triggeredSkillCond != nil) {
		delete(main.SkillData, "triggered")
		return
	}
	main.SkillData["triggered"] = true

	if truthy(main.SkillData["triggeredByBrand"]) {
		panic("triggers: Arcanist Brand activation-frequency path unported")
	}

	// Dual wield triggers
	sourceWeaponTrigger := config.sourceWeapon && source != nil && source.SkillFlags["bothWeaponAttack"]
	itemSupportTrigger := main.TriggeredBy != nil && main.TriggeredBy.GrantedEffect.Support &&
		env.geFromItem(main.TriggeredBy.GrantedEffect)
	if trigRate != nil && source != nil && truthy(env.Player.WeaponData1["type"]) && truthy(env.Player.WeaponData2["type"]) &&
		!truthy(source.SkillData["doubleHitsWhenDualWielding"]) &&
		(source.SkillTypes[modparser.SkillType.Melee] || source.SkillTypes[modparser.SkillType.Attack]) &&
		(sourceWeaponTrigger || itemSupportTrigger) {
		halved := *trigRate / 2
		trigRate = &halved
	}

	// `ignoresTickRate = ignoresTickRate or (storedUses and storedUses > 1)`.
	// With storedUses present but 1, the right side is a real `false`, so the
	// key is WRITTEN false rather than left absent.
	if !truthy(main.SkillData["ignoresTickRate"]) {
		su, hasSU := main.SkillData["storedUses"]
		switch {
		case truthy(su):
			main.SkillData["ignoresTickRate"] = anyNum(su) > 1
		case hasSU && su != nil: // storedUses is itself false
			main.SkillData["ignoresTickRate"] = su
		default:
			delete(main.SkillData, "ignoresTickRate")
		}
	}

	cached := env.GlobalCache[uuid]

	// Account for source unleash
	if source != nil && cached != nil && source.SkillModList.Flag(nil, "HasSeals") &&
		source.SkillTypes[modparser.SkillType.CanRapidFire] {
		unleashDpsMult := 1.0
		if v, ok := cached.ActiveSkillData["dpsMultiplier"]; ok && truthy(v) {
			unleashDpsMult = anyNum(v)
		}
		scaled := *trigRate * unleashDpsMult
		trigRate = &scaled
		main.SkillFlags["HasSeals"] = true
		main.SkillData["ignoresTickRate"] = true
	}

	// Account for skills that can hit multiple times per use
	if source != nil && cached != nil && source.SkillPartName != "" &&
		strings.Contains(source.SkillPartName, "All") && strings.Contains(source.SkillPartName, "Projectiles") &&
		source.SkillFlags["projectile"] {
		multiHitDpsMult := 1.0
		if v := cached.out("ProjectileCount"); truthy(v) {
			multiHitDpsMult = anyNum(v)
		}
		scaled := *trigRate * multiHitDpsMult
		trigRate = &scaled
	}

	if truthy(main.SkillData["triggeredByManaSpent"]) {
		panic("triggers: Kitava's Thirst mana-spent path unported")
	}
	if truthy(main.SkillData["triggeredByBattleMageCry"]) {
		panic("triggers: Battlemage's Cry uptime path unported")
	}
	if main.ActiveEffect.GrantedEffect.Name == "Combust" {
		panic("triggers: Infernal Cry / Combust uptime path unported")
	}
	if truthy(main.SkillData["triggeredByManaforged"]) {
		panic("triggers: Manaforged Arrows path unported")
	}

	icdr := Mod(main.SkillModList, main.SkillCfg, "CooldownRecovery")
	addedCooldownVal := main.SkillModList.Sum("BASE", main.SkillCfg, "CooldownRecovery")
	var addedCooldown *float64
	if addedCooldownVal != 0 {
		addedCooldown = &addedCooldownVal
	}
	var cooldownOverride *float64
	if ov := main.SkillModList.Override(main.SkillCfg, "CooldownRecovery"); truthy(ov) {
		n := anyNum(ov)
		cooldownOverride = &n
	}
	// #EVAL: the guard is on actor.mainSkill.triggeredBy but the read is on
	// env.player.mainSkill.triggeredBy — the same table for the player actor.
	var triggerCD *float64
	if main.TriggeredBy != nil {
		triggerCD = triggeredByCooldown(env.PlayerMainSkill.TriggeredBy)
	}
	if triggerCD == nil && source != nil && source.TriggeredBy != nil {
		triggerCD = triggeredByCooldown(source.TriggeredBy)
	}
	var triggeredCD *float64
	if v, ok := main.SkillData["cooldown"]; ok && truthy(v) {
		n := anyNum(v)
		triggeredCD = &n
	}

	num := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	addsCastTime := anyNum(output["addsCastTime"])

	triggeredCDAdjusted := (num(triggeredCD) + num(addedCooldown)) / icdr
	triggerCDAdjusted := (num(triggerCD) + addsCastTime) / icdr
	triggeredCDTickRounded := math.Ceil(triggeredCDAdjusted*d.Misc.ServerTickRate) / d.Misc.ServerTickRate
	if truthy(main.SkillData["ignoresTickRate"]) {
		triggeredCDTickRounded = triggeredCDAdjusted
	}
	// #EVAL: the reference's `actor.mainSkill.triggeredBy.ignoresTickRate`
	// guard is dead — nothing anywhere in the archive sets that field.
	triggerCDTickRounded := math.Ceil(triggerCDAdjusted*d.Misc.ServerTickRate) / d.Misc.ServerTickRate
	actionCooldownTickRounded := math.Max(triggerCDTickRounded, triggeredCDTickRounded)
	if cooldownOverride != nil {
		actionCooldownTickRounded = math.Ceil(*cooldownOverride*d.Misc.ServerTickRate) / d.Misc.ServerTickRate
	}

	// `source == mainSkill and triggerRateCapOverride or m_huge`
	output["TriggerRateCap"] = math.Inf(1)
	if source == main && truthy(main.SkillData["triggerRateCapOverride"]) {
		output["TriggerRateCap"] = anyNum(main.SkillData["triggerRateCapOverride"])
	}
	if actionCooldownTickRounded != 0 {
		output["TriggerRateCap"] = 1 / actionCooldownTickRounded
	}
	if config.triggerName == "Doom Blast" {
		panic("triggers: Doom Blast source paths unported")
	}
	if env.PlayerMainSkill.ActiveEffect.GrantedEffect.Name == "Doom Blast" {
		panic("triggers: Doom Blast / Vixen's Entrapment path unported")
	}
	switch {
	case trigRate != nil && !main.SkillFlags["globalTrigger"] && !config.ignoreSourceRate:
		output["EffectiveSourceRate"] = *trigRate
	default:
		output["EffectiveSourceRate"] = output["TriggerRateCap"]
		main.SkillFlags["globalTrigger"] = true
	}

	if outNum(output, "EffectiveSourceRate") != 0 && !env.PlayerMainSkill.SkillFlags["skipEffectiveRate"] {
		triggerChance := 100.0

		// Accuracy and crit chance
		if source != nil && (source.SkillTypes[modparser.SkillType.Melee] || source.SkillTypes[modparser.SkillType.Attack]) &&
			cached != nil && !config.triggerOnUse {
			sourceHitChance := 0.0
			if cached.HitChance != nil {
				sourceHitChance = *cached.HitChance
			}
			dualRolls := truthy(env.Player.WeaponData1["type"]) && truthy(env.Player.WeaponData2["type"]) &&
				truthy(source.SkillData["doubleHitsWhenDualWielding"])
			if sourceHitChance != 100 {
				if dualRolls {
					// Some skills hit with both weapons at once; each rolls
					// accuracy independently.
					mainHandHit := anyNum(cached.OutputMainHand["HitChance"])
					offHandHit := anyNum(cached.OutputOffHand["HitChance"])
					bothHit := mainHandHit * offHandHit / 100
					effectiveHitChance := bothHit + mainHandHit*(100-offHandHit)/100 + (100-mainHandHit)*offHandHit/100
					triggerChance = triggerChance * effectiveHitChance / 100
				} else {
					triggerChance = triggerChance * sourceHitChance / 100
				}
			}
			if truthy(main.SkillData["triggerOnCrit"]) {
				if config.triggerChance == nil {
					if v, ok := main.SkillData["chanceToTriggerOnCrit"]; ok && truthy(v) {
						n := anyNum(v)
						config.triggerChance = &n
					} else if v, ok := cached.MainSkillData["chanceToTriggerOnCrit"]; ok && truthy(v) {
						n := anyNum(v)
						config.triggerChance = &n
					}
				}
				sourceCritChance := 0.0
				if cached.CritChance != nil {
					sourceCritChance = *cached.CritChance
				}
				if sourceCritChance != 100 {
					if dualRolls {
						mainHandCrit := anyNum(cached.OutputMainHand["CritChance"])
						offHandCrit := anyNum(cached.OutputOffHand["CritChance"])
						bothHit := mainHandCrit * offHandCrit / 100
						effectiveCritChance := bothHit + mainHandCrit*(100-offHandCrit)/100 + (100-mainHandCrit)*offHandCrit/100
						triggerChance = triggerChance * effectiveCritChance / 100
					} else {
						triggerChance = triggerChance * sourceCritChance / 100
					}
				}
			}
		}

		// Trigger chance
		if config.triggerChance != nil && *config.triggerChance != 100 {
			triggerChance = triggerChance * *config.triggerChance / 100
		}

		// #EVAL: the reference's second arm compares two freshly built
		// tables (`triggeredSkills[1] == packageSkillDataForSimulation(...)`),
		// which is never true in Lua, so this reduces to
		// `ignoresTickRate and not config.triggeredSkillCond`.
		switch {
		case truthy(main.SkillData["ignoresTickRate"]) && config.triggeredSkillCond == nil:
			overlaps := 1.0
			if config.stagesAreOverlaps != nil {
				panic("triggers: stagesAreOverlaps path unported")
			}
			if config.overlaps != nil {
				overlaps = *config.overlaps
			}
			output["SkillTriggerRate"] = math.Min(outNum(output, "TriggerRateCap"), outNum(output, "EffectiveSourceRate")*overlaps)
		case main.SkillFlags["globalTrigger"] && config.triggeredSkillCond == nil:
			// Trigger does not use source rate breakpoints
			output["SkillTriggerRate"] = output["EffectiveSourceRate"]
		default:
			// Triggers like Cast on Crit go through the simulation
			rotation := triggeredSkills
			if config.triggeredSkillCond == nil {
				rotation = []*simSkill{env.packageSkillDataForSimulation(main)}
			}
			var simCD *float64
			if !truthy(main.SkillData["triggeredByBrand"]) {
				if triggerCD != nil {
					simCD = triggerCD
				} else {
					simCD = triggeredCD
				}
			}
			rate := env.calcMultiSpellRotationImpact(rotation, outNum(output, "EffectiveSourceRate"), simCD, triggerChance, actor)
			if actor.db.Flag(nil, "HaveTriggerBots") && main.SkillTypes[modparser.SkillType.Spell] {
				rate = 2 * rate
			}
			if config.stagesAreOverlaps != nil {
				panic("triggers: stagesAreOverlaps path unported")
			}
			output["SkillTriggerRate"] = rate
		}
	} else {
		output["SkillTriggerRate"] = 0.0
	}
	main.SkillData["triggerRate"] = output["SkillTriggerRate"]

	// Account for Trigger-related INC/MORE modifiers
	output["Speed"] = main.SkillData["triggerRate"]
	if source != nil {
		addTriggerIncMoreMods(main, source)
	} else {
		addTriggerIncMoreMods(main, main)
	}
	if source != nil && source != main {
		main.SkillData["triggerSourceUUID"] = env.cacheSkillUUID(source)
	}
}

// triggeredByCooldown is
// `triggeredBy.grantedEffect.levels[triggeredBy.level].cooldown`.
func triggeredByCooldown(triggeredBy *ActiveEffect) *float64 {
	if triggeredBy == nil || triggeredBy.GrantedEffect == nil {
		return nil
	}
	lvl := triggeredBy.GrantedEffect.Levels[triggeredBy.Level]
	if lvl == nil {
		return nil
	}
	if v, ok := lvl.Extra["cooldown"]; ok {
		return &v
	}
	return nil
}
