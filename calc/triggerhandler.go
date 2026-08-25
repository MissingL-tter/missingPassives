// CalcTriggers.lua L401-899: defaultTriggerHandler, the shared body every
// trigger config funnels into.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
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
	uuid := config.uuid
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
				source, trigRate, uuid = env.findTriggerSkill(skill, source, trigRate, config.comparer)
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
		if v, ok := cached.activeSkillData("dpsMultiplier"); ok && truthy(v) {
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

	// Special handling for Kitava's Thirst: repeated hits do not consume
	// mana and do not trigger it.
	if truthy(main.SkillData["triggeredByManaSpent"]) && source != nil && trigRate != nil {
		repeats := 1 + source.SkillModList.Sum("BASE", nil, "RepeatCount")
		scaled := *trigRate / repeats
		trigRate = &scaled
	}
	// Battlemage's Cry uptime
	if truthy(main.SkillData["triggeredByBattleMageCry"]) && cached != nil && source != nil &&
		source.SkillTypes[modparser.SkillType.Melee] && trigRate != nil {
		ceilB := func(x float64) float64 {
			return data.Misc.ServerTickTime * math.Ceil(x/data.Misc.ServerTickTime)
		}
		battleMageExertsCount := anyNum(cached.out("BattleCryExertsCount"))
		battleMageDuration := ceilB(anyNum(cached.out("BattleMageCryDuration")))
		battleMageCastTime := anyNum(cached.out("BattleMageCryCastTime"))
		battleMageCooldown := ceilB(anyNum(cached.out("BattleMageCryCooldown")))
		// Cap the number of hits that happen during the duration; they
		// happen every battlemage cooldown + duration.
		battleMageHits := math.Max(math.Min(*trigRate*battleMageDuration, battleMageExertsCount), 0)
		scaled := battleMageHits / (battleMageCastTime + battleMageCooldown)
		trigRate = &scaled
	}
	// Infernal Cry uptime
	if main.ActiveEffect.GrantedEffect.Name == "Combust" && cached != nil && source != nil &&
		source.SkillTypes[modparser.SkillType.Melee] && trigRate != nil {
		ceilB := func(x float64) float64 {
			return data.Misc.ServerTickTime * math.Ceil(x/data.Misc.ServerTickTime)
		}
		infernalCryExertsCount := anyNum(cached.out("InfernalExertsCount"))
		infernalCryDuration := ceilB(anyNum(cached.out("InfernalCryDuration")))
		infernalCryCastTime := anyNum(cached.out("InfernalCryCastTime"))
		infernalCryCooldown := ceilB(anyNum(cached.out("InfernalCryCooldown")))
		// Cap the number of hits that happen during the duration; they
		// happen every Infernal Cry cooldown + duration.
		infernalCryHits := math.Max(math.Min(*trigRate*infernalCryDuration, infernalCryExertsCount), 0)
		scaled := infernalCryHits / (infernalCryCastTime + infernalCryCooldown)
		trigRate = &scaled
	}
	// Handling for mana spending rate for Manaforged Arrows Support
	if truthy(main.SkillData["triggeredByManaforged"]) && trigRate != nil && *trigRate > 0 {
		triggeredUUID := env.cacheSkillUUID(main)
		if env.GlobalCache[triggeredUUID] == nil {
			env.BuildActiveSkill(env.Mode, main, triggeredUUID, triggeredUUID)
		}
		triggeredManaCost := anyNum(env.GlobalCache[triggeredUUID].out("ManaCostRaw"))
		if triggeredManaCost > 0 {
			manaSpentThreshold := triggeredManaCost * anyNum(main.SkillData["ManaForgedArrowsPercentThreshold"])
			sourceManaCost := 0.0
			if cached != nil {
				sourceManaCost = anyNum(cached.out("ManaCostRaw"))
			}
			if sourceManaCost > 0 {
				scaled := *trigRate / math.Ceil(manaSpentThreshold/sourceManaCost)
				trigRate = &scaled
			} else {
				zero := 0.0
				trigRate = &zero
			}
		}
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

	if truthy(main.SkillData["triggeredByBrand"]) && main.TriggeredBy != nil {
		// The brand's activation interval stands in for the trigger CD; the
		// icdr multiplication cancels out the division below -- brand
		// activation rate is not affected by icdr.
		n := anyNum(main.TriggeredBy.MainSkill.SkillData["repeatFrequency"]) /
			main.TriggeredBy.ActivationFreqMore / main.TriggeredBy.ActivationFreqInc * icdr
		triggerCD = &n
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
	triggeredCDTickRounded := math.Ceil(triggeredCDAdjusted*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	if truthy(main.SkillData["ignoresTickRate"]) {
		triggeredCDTickRounded = triggeredCDAdjusted
	}
	// triggeredBy.ignoresTickRate has exactly one writer: the Arcanist
	// Brand config (L1347). (An earlier #EVAL note claimed the field was
	// dead; that held only for the paths ported at the time.)
	triggerCDTickRounded := math.Ceil(triggerCDAdjusted*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	if main.TriggeredBy != nil && main.TriggeredBy.IgnoresTickRate {
		triggerCDTickRounded = triggerCDAdjusted
	}
	actionCooldownTickRounded := math.Max(triggerCDTickRounded, triggeredCDTickRounded)
	if cooldownOverride != nil {
		actionCooldownTickRounded = math.Ceil(*cooldownOverride*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
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
		switch str(env.ConfigInput["doomBlastSource"]) {
		case "expiration":
			panic("triggers: Doom Blast expiration source unported (no corpus build sets it)")
		case "hexblast":
			panic("triggers: Doom Blast hexblast source unported (no corpus build sets it)")
		}
	}
	switch {
	case env.PlayerMainSkill.ActiveEffect.GrantedEffect.Name == "Doom Blast" && str(env.ConfigInput["doomBlastSource"]) == "vixen":
		// A curse socketed in Vixen's Entrapment is triggered on its own
		// cooldown, so the effective rate is that rotation, not the cast
		// rate of the curse.
		gloves, _ := env.Player.ItemList["Gloves"].(*Item)
		if gloves == nil || gloves.In.Title == nil || !strings.Contains(*gloves.In.Title, "Vixen's Entrapment") {
			output["VixenModeNoVixenGlovesWarn"] = true
		}
		env.ModDB.AddMod(newMod("UsesCurseOverlaps", "FLAG", true, "Config"))
		var vixensCD *float64
		if vixens := data.Skills["SupportUniqueCastCurseOnCurse"]; vixens != nil {
			cd := anyNum(vixens.Levels[1].Extra["cooldown"]) / icdr
			vixensCD = &cd
		}
		rate := env.calcMultiSpellRotationImpact(
			[]*simSkill{{uuid: env.cacheSkillUUID(env.PlayerMainSkill), icdr: &icdr}},
			num(trigRate), vixensCD, 100, actor)
		output["EffectiveSourceRate"] = rate
		output["VixensTooMuchCastSpeedWarn"] = vixensCD != nil && *vixensCD > (1/num(trigRate))
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
					mainHandHit := anyNum(cached.outputMainHand("HitChance"))
					offHandHit := anyNum(cached.outputOffHand("HitChance"))
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
					} else if v, ok := cached.mainSkillData("chanceToTriggerOnCrit"); ok && truthy(v) {
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
						mainHandCrit := anyNum(cached.outputMainHand("CritChance"))
						offHandCrit := anyNum(cached.outputOffHand("CritChance"))
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

// cwcHandler ports CWCHandler (CalcTriggers.lua L219): Cast While Channelling
// triggers its linked spells on a fixed interval while the channel runs, so
// the source is found by support-compatibility rather than by skill type.
func (env *Env) cwcHandler() {
	main := env.PlayerMainSkill
	if main.SkillFlags["minion"] || main.SkillFlags["disable"] {
		return
	}
	var triggeredSkills []*simSkill
	var source, disabledSource *ActiveSkill
	triggerName := "Cast While Channeling"
	output := env.Player.Output
	for _, skill := range env.PlayerActiveSkills {
		slotMatch := env.slotMatch(skill)
		if source == nil {
			canSupport := main.TriggeredBy != nil && main.TriggeredBy.GemData != nil &&
				env.canGrantedEffectSupportActiveSkill(main.TriggeredBy.GemData.GrantedEffect, skill, false)
			if truthy(skill.SkillData["triggerTime"]) && canSupport && skill != main && slotMatch && !isTriggered(skill) {
				source = skill
			} else if disabledSource == nil && canSupport && skill.SkillFlags["disable"] &&
				skill.SkillTypes[modparser.SkillType.Channel] && skill != main && slotMatch {
				// A channelling skill is socketed but unusable (commonly a
				// support gem restricting its weapon types). Remember it so
				// we don't fall through to the Self-Cast estimate below,
				// which would report a *higher* DPS for a setup that cannot
				// actually trigger at all.
				disabledSource = skill
			}
		}
		if truthy(skill.SkillData["triggeredWhileChannelling"]) && slotMatch {
			triggeredSkills = append(triggeredSkills, env.packageSkillDataForSimulation(skill))
		}
	}
	if source == nil && disabledSource != nil {
		main.SkillFlags["disable"] = true
		main.DisableReason = disabledSource.ActiveEffect.GrantedEffect.Name + " is disabled"
		main.InfoMessage = triggerName + " Triggering Skill is disabled"
	} else if source == nil || len(triggeredSkills) < 1 {
		delete(main.SkillData, "triggered")
		main.InfoMessage2 = "DPS reported assuming Self-Cast"
		main.InfoMessage = "No " + triggerName + " Triggering Skill Found"
	} else {
		if act := processAddedCastTime(main); act != nil {
			output["addsCastTime"] = *act
		}

		icdr := Mod(main.SkillModList, main.SkillCfg, "CooldownRecovery")
		triggerInterval := anyNum(source.SkillData["triggerTime"])
		triggerRateOfTrigger := 1 / triggerInterval
		cooldownOverride := main.SkillModList.Override(main.SkillCfg, "CooldownRecovery")
		if truthy(cooldownOverride) {
			main.SkillFlags["hasOverride"] = true
		}

		// `cooldownOverride or m_max(triggeredCD or 0, addsCastTime or 0) / icdr`
		triggeredTotalCooldown := math.Max(anyNum(main.SkillData["cooldown"]), anyNum(output["addsCastTime"])) / icdr
		if truthy(cooldownOverride) {
			triggeredTotalCooldown = anyNum(cooldownOverride)
		}
		triggeredCDAdjusted := math.Ceil(triggeredTotalCooldown*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
		effCDTriggeredSkill := math.Ceil(triggeredCDAdjusted*triggerRateOfTrigger) / triggerRateOfTrigger

		output["TriggerRateCap"] = math.Min(1/effCDTriggeredSkill, triggerRateOfTrigger)
		zero := 0.0
		rate := env.calcMultiSpellRotationImpact(triggeredSkills, triggerRateOfTrigger, &zero, 100, env.playerPA)
		if env.ModDB.Flag(nil, "HaveTriggerBots") && main.SkillTypes[modparser.SkillType.Spell] {
			rate = 2 * rate
		}
		output["SkillTriggerRate"] = rate

		// Account for Trigger-related INC/MORE modifiers
		addTriggerIncMoreMods(main, main)
		output["ChannelTimeToTrigger"] = triggerInterval
		main.SkillData["triggered"] = true
		main.SkillFlags["globalTrigger"] = true
		main.SkillData["triggerRate"] = output["SkillTriggerRate"]
		main.SkillData["triggerSourceUUID"] = env.cacheSkillUUID(source)
		main.InfoMessage = triggerName + "'s Trigger: " + source.ActiveEffect.GrantedEffect.Name
	}
}

// helmetFocusHandler ports the local of the same name (CalcTriggers L135):
// "trigger when you Focus" helmet enchants. Skills trigger only on
// activation, so the effective rate is one per focus duration + cooldown.
func (env *Env) helmetFocusHandler() {
	main := env.PlayerMainSkill
	if main.SkillFlags["minion"] || main.SkillFlags["disable"] || main.TriggeredBy == nil {
		return
	}
	output := env.Player.Output
	main.SkillData["triggered"] = true
	triggerCD := triggeredByCooldown(main.TriggeredBy)
	icdrFocus := Mod(main.SkillModList, main.SkillCfg, "FocusCooldownRecovery")
	icdrSkill := Mod(main.SkillModList, main.SkillCfg, "CooldownRecovery")

	// Next possible activation is duration + cooldown.
	skillFocus := data.Skills["Focus"]
	focusDuration := anyNum(skillFocus.ConstantStats[0][1]) / 1000
	focusCD := skillFocus.Levels[1].Extra["cooldown"] / icdrFocus
	focusTotalCD := focusDuration + focusCD

	// The skill's own cooldown still applies to focus triggers.
	modActionCooldown := math.Max(anyNum(main.SkillData["cooldown"]), num64(triggerCD)/icdrSkill)
	rateCapAdjusted := math.Ceil(modActionCooldown*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	triggerRate := math.Inf(1)
	if rateCapAdjusted != 0 {
		triggerRate = 1 / rateCapAdjusted
	}
	output["TriggerRateCap"] = triggerRate
	output["SkillTriggerRate"] = 1 / focusTotalCD

	// Account for Trigger-related INC/MORE modifiers
	addTriggerIncMoreMods(main, main)
	main.InfoMessage = "Assuming perfect focus Re-Use"
	main.SkillData["triggerRate"] = output["SkillTriggerRate"]
	main.SkillFlags["globalTrigger"] = true
}
