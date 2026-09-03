// CalcTriggers.lua L401-899: defaultTriggerHandler, the shared body every
// trigger config funnels into.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
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
		if slot := main.SocketGroup.Slot; slot != "" {
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
		main.SkillData.Del("triggered")
		return
	}
	main.SkillData.SetFlag("triggered", true)

	// Dual wield triggers
	sourceWeaponTrigger := config.sourceWeapon && source != nil && source.SkillFlags["bothWeaponAttack"]
	itemSupportTrigger := main.TriggeredBy != nil && main.TriggeredBy.GrantedEffect.Support &&
		env.geFromItem(main.TriggeredBy.GrantedEffect)
	if trigRate != nil && source != nil && weaponType(weaponOf(env.Player.WeaponData1)) != "" && weaponType(weaponOf(env.Player.WeaponData2)) != "" &&
		!source.SkillData.Flag("doubleHitsWhenDualWielding") &&
		(source.SkillTypes[modparser.SkillTypeMelee] || source.SkillTypes[modparser.SkillTypeAttack]) &&
		(sourceWeaponTrigger || itemSupportTrigger) {
		halved := *trigRate / 2
		trigRate = &halved
	}

	// `ignoresTickRate = ignoresTickRate or (storedUses and storedUses > 1)`.
	// With storedUses present but 1, the right side is a real `false`, so the
	// key is WRITTEN false rather than left absent.
	if !main.SkillData.Flag("ignoresTickRate") {
		su := main.SkillData.Get("storedUses")
		switch {
		case su.Truthy():
			main.SkillData.SetFlag("ignoresTickRate", su.Num() > 1)
		case su.Kind != modstore.OutAbsent: // storedUses is itself false
			main.SkillData.Set("ignoresTickRate", su)
		default:
			main.SkillData.Del("ignoresTickRate")
		}
	}

	cached := env.GlobalCache[uuid]

	// Account for source unleash
	if source != nil && cached != nil && source.SkillModList.Flag(nil, "HasSeals") &&
		source.SkillTypes[modparser.SkillTypeCanRapidFire] {
		unleashDpsMult := 1.0
		if v := cached.activeSkillData("dpsMultiplier"); v.Truthy() {
			unleashDpsMult = v.Num()
		}
		scaled := *trigRate * unleashDpsMult
		trigRate = &scaled
		main.SkillFlags["HasSeals"] = true
		main.SkillData.SetFlag("ignoresTickRate", true)
	}

	// Account for skills that can hit multiple times per use
	if source != nil && cached != nil && source.SkillPartName != "" &&
		strings.Contains(source.SkillPartName, "All") && strings.Contains(source.SkillPartName, "Projectiles") &&
		source.SkillFlags["projectile"] {
		multiHitDpsMult := 1.0
		if v := cached.out("ProjectileCount"); v.Truthy() {
			multiHitDpsMult = v.Num()
		}
		scaled := *trigRate * multiHitDpsMult
		trigRate = &scaled
	}

	// Special handling for Kitava's Thirst: repeated hits do not consume
	// mana and do not trigger it.
	if main.SkillData.Flag("triggeredByManaSpent") && source != nil && trigRate != nil {
		repeats := 1 + source.SkillModList.Sum(modparser.Base, nil, "RepeatCount")
		scaled := *trigRate / repeats
		trigRate = &scaled
	}
	// Battlemage's Cry uptime
	if main.SkillData.Flag("triggeredByBattleMageCry") && cached != nil && source != nil &&
		source.SkillTypes[modparser.SkillTypeMelee] && trigRate != nil {
		ceilB := func(x float64) float64 {
			return data.Misc.ServerTickTime * math.Ceil(x/data.Misc.ServerTickTime)
		}
		battleMageExertsCount := cached.out("BattleCryExertsCount").Num()
		battleMageDuration := ceilB(cached.out("BattleMageCryDuration").Num())
		battleMageCastTime := cached.out("BattleMageCryCastTime").Num()
		battleMageCooldown := ceilB(cached.out("BattleMageCryCooldown").Num())
		// Cap the number of hits that happen during the duration; they
		// happen every battlemage cooldown + duration.
		battleMageHits := math.Max(math.Min(*trigRate*battleMageDuration, battleMageExertsCount), 0)
		scaled := battleMageHits / (battleMageCastTime + battleMageCooldown)
		trigRate = &scaled
	}
	// Infernal Cry uptime
	if main.ActiveEffect.GrantedEffect.Name == "Combust" && cached != nil && source != nil &&
		source.SkillTypes[modparser.SkillTypeMelee] && trigRate != nil {
		ceilB := func(x float64) float64 {
			return data.Misc.ServerTickTime * math.Ceil(x/data.Misc.ServerTickTime)
		}
		infernalCryExertsCount := cached.out("InfernalExertsCount").Num()
		infernalCryDuration := ceilB(cached.out("InfernalCryDuration").Num())
		infernalCryCastTime := cached.out("InfernalCryCastTime").Num()
		infernalCryCooldown := ceilB(cached.out("InfernalCryCooldown").Num())
		// Cap the number of hits that happen during the duration; they
		// happen every Infernal Cry cooldown + duration.
		infernalCryHits := math.Max(math.Min(*trigRate*infernalCryDuration, infernalCryExertsCount), 0)
		scaled := infernalCryHits / (infernalCryCastTime + infernalCryCooldown)
		trigRate = &scaled
	}
	// Handling for mana spending rate for Manaforged Arrows Support
	if main.SkillData.Flag("triggeredByManaforged") && trigRate != nil && *trigRate > 0 {
		triggeredUUID := env.cacheSkillUUID(main)
		if env.GlobalCache[triggeredUUID] == nil {
			env.BuildActiveSkill(env.Mode, main, triggeredUUID, triggeredUUID)
		}
		triggeredManaCost := env.GlobalCache[triggeredUUID].out("ManaCostRaw").Num()
		if triggeredManaCost > 0 {
			manaSpentThreshold := triggeredManaCost * main.SkillData.N("ManaForgedArrowsPercentThreshold")
			sourceManaCost := 0.0
			if cached != nil {
				sourceManaCost = cached.out("ManaCostRaw").Num()
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
	addedCooldownVal := main.SkillModList.Sum(modparser.Base, main.SkillCfg, "CooldownRecovery")
	var addedCooldown *float64
	if addedCooldownVal != 0 {
		addedCooldown = &addedCooldownVal
	}
	var cooldownOverride *float64
	if ov, ok := main.SkillModList.Override(main.SkillCfg, "CooldownRecovery"); ok {
		n := valueNum(ov)
		cooldownOverride = &n
	}
	// The guard is on actor.mainSkill.triggeredBy but the read is on
	// env.player.mainSkill.triggeredBy — the same table for the player actor.
	var triggerCD *float64
	if main.TriggeredBy != nil {
		triggerCD = triggeredByCooldown(env.PlayerMainSkill.TriggeredBy)
	}
	if triggerCD == nil && source != nil && source.TriggeredBy != nil {
		triggerCD = triggeredByCooldown(source.TriggeredBy)
	}
	var triggeredCD *float64
	if main.SkillData.Flag("cooldown") {
		n := main.SkillData.N("cooldown")
		triggeredCD = &n
	}

	if main.SkillData.Flag("triggeredByBrand") && main.TriggeredBy != nil {
		// The brand's activation interval stands in for the trigger CD; the
		// icdr multiplication cancels out the division below -- brand
		// activation rate is not affected by icdr.
		n := main.TriggeredBy.MainSkill.SkillData.N("repeatFrequency") /
			main.TriggeredBy.ActivationFreqMore / main.TriggeredBy.ActivationFreqInc * icdr
		triggerCD = &n
	}

	num := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	addsCastTime := output.N("addsCastTime")

	triggeredCDAdjusted := (num(triggeredCD) + num(addedCooldown)) / icdr
	triggerCDAdjusted := (num(triggerCD) + addsCastTime) / icdr
	triggeredCDTickRounded := math.Ceil(triggeredCDAdjusted*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	if main.SkillData.Flag("ignoresTickRate") {
		triggeredCDTickRounded = triggeredCDAdjusted
	}
	// triggeredBy.ignoresTickRate has exactly one writer: the Arcanist
	// Brand config (L1347). (An earlier note claimed the field was
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
	output.SetN("TriggerRateCap", math.Inf(1))
	if source == main && main.SkillData.Has("triggerRateCapOverride") {
		output.SetN("TriggerRateCap", main.SkillData.N("triggerRateCapOverride"))
	}
	if actionCooldownTickRounded != 0 {
		output.SetN("TriggerRateCap", 1/actionCooldownTickRounded)
	}
	if config.triggerName == "Doom Blast" {
		switch env.ConfigInput.DoomBlastSource {
		case "expiration":
			// The hexes fall off on their own; if they'd expire faster than
			// they're recast, expiration behaves like replacement (overlaps).
			expirationRate := 1 / env.GlobalCache[uuid].out("Duration").Num()
			if trigRate != nil && expirationRate > *trigRate {
				env.ModDB.AddMod(newModS("UsesCurseOverlaps", modparser.Flag, modparser.Bool(true), "Config"))
			} else {
				trigRate = &expirationRate
			}
		case "hexblast":
			// Hexblast consumes the hex: one blast per cast+recast cycle.
			var hexBlast *ActiveSkill
			var rate *float64
			for _, skill := range env.PlayerActiveSkills {
				if skill.ActiveEffect.GrantedEffect.Name == "Hexblast" && !isTriggered(skill) && skill != main {
					hexBlast, rate, uuid = env.findTriggerSkill(skill, hexBlast, rate, nil)
				}
			}
			if hexBlast != nil && trigRate != nil && rate != nil {
				combined := 1 / (1 / *trigRate + 1 / *rate)
				trigRate = &combined
			}
		}
	}
	switch {
	case env.PlayerMainSkill.ActiveEffect.GrantedEffect.Name == "Doom Blast" && env.ConfigInput.DoomBlastSource == "vixen":
		// A curse socketed in Vixen's Entrapment is triggered on its own
		// cooldown, so the effective rate is that rotation, not the cast
		// rate of the curse.
		gloves, _ := env.Player.ItemList["Gloves"].(*Item)
		if gloves == nil || gloves.In.Title == nil || !strings.Contains(*gloves.In.Title, "Vixen's Entrapment") {
			output.SetFlag("VixenModeNoVixenGlovesWarn", true)
		}
		env.ModDB.AddMod(newModS("UsesCurseOverlaps", modparser.Flag, modparser.Bool(true), "Config"))
		var vixensCD *float64
		if vixens := data.Skills["SupportUniqueCastCurseOnCurse"]; vixens != nil {
			cd := vixens.Levels[1].Extra["cooldown"] / icdr
			vixensCD = &cd
		}
		rate := env.calcMultiSpellRotationImpact(
			[]*simSkill{{uuid: env.cacheSkillUUID(env.PlayerMainSkill), icdr: &icdr}},
			num(trigRate), vixensCD, 100, actor)
		output.SetN("EffectiveSourceRate", rate)
		output.SetFlag("VixensTooMuchCastSpeedWarn", vixensCD != nil && *vixensCD > (1/num(trigRate)))
	case trigRate != nil && !main.SkillFlags["globalTrigger"] && !config.ignoreSourceRate:
		output.SetN("EffectiveSourceRate", *trigRate)
	default:
		output.Set("EffectiveSourceRate", output.Get("TriggerRateCap"))
		main.SkillFlags["globalTrigger"] = true
	}

	if output.N("EffectiveSourceRate") != 0 && !env.PlayerMainSkill.SkillFlags["skipEffectiveRate"] {
		triggerChance := 100.0

		// Accuracy and crit chance
		if source != nil && (source.SkillTypes[modparser.SkillTypeMelee] || source.SkillTypes[modparser.SkillTypeAttack]) &&
			cached != nil && !config.triggerOnUse {
			sourceHitChance := 0.0
			if cached.HitChance != nil {
				sourceHitChance = *cached.HitChance
			}
			dualRolls := weaponType(weaponOf(env.Player.WeaponData1)) != "" && weaponType(weaponOf(env.Player.WeaponData2)) != "" &&
				source.SkillData.Flag("doubleHitsWhenDualWielding")
			if sourceHitChance != 100 {
				if dualRolls {
					// Some skills hit with both weapons at once; each rolls
					// accuracy independently.
					mainHandHit := cached.outputMainHand("HitChance").Num()
					offHandHit := cached.outputOffHand("HitChance").Num()
					bothHit := mainHandHit * offHandHit / 100
					effectiveHitChance := bothHit + mainHandHit*(100-offHandHit)/100 + (100-mainHandHit)*offHandHit/100
					triggerChance = triggerChance * effectiveHitChance / 100
				} else {
					triggerChance = triggerChance * sourceHitChance / 100
				}
			}
			if main.SkillData.Flag("triggerOnCrit") {
				if config.triggerChance == nil {
					if main.SkillData.Flag("chanceToTriggerOnCrit") {
						n := main.SkillData.N("chanceToTriggerOnCrit")
						config.triggerChance = &n
					} else if v := cached.mainSkillData("chanceToTriggerOnCrit"); v.Truthy() {
						n := v.Num()
						config.triggerChance = &n
					}
				}
				sourceCritChance := 0.0
				if cached.CritChance != nil {
					sourceCritChance = *cached.CritChance
				}
				if sourceCritChance != 100 {
					if dualRolls {
						mainHandCrit := cached.outputMainHand("CritChance").Num()
						offHandCrit := cached.outputOffHand("CritChance").Num()
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

		// The reference's second arm compares two freshly built
		// tables (`triggeredSkills[1] == packageSkillDataForSimulation(...)`),
		// which is never true in Lua, so this reduces to
		// `ignoresTickRate and not config.triggeredSkillCond`.
		switch {
		case main.SkillData.Flag("ignoresTickRate") && config.triggeredSkillCond == nil:
			overlaps := 1.0
			if v, ok := env.stageOverlaps(config); ok {
				overlaps = v
			} else if config.overlaps != nil {
				overlaps = *config.overlaps
			}
			output.SetN("SkillTriggerRate", math.Min(output.N("TriggerRateCap"), output.N("EffectiveSourceRate")*overlaps))
		case main.SkillFlags["globalTrigger"] && config.triggeredSkillCond == nil:
			// Trigger does not use source rate breakpoints
			output.Set("SkillTriggerRate", output.Get("EffectiveSourceRate"))
		default:
			// Triggers like Cast on Crit go through the simulation
			rotation := triggeredSkills
			if config.triggeredSkillCond == nil {
				rotation = []*simSkill{env.packageSkillDataForSimulation(main)}
			}
			var simCD *float64
			if !main.SkillData.Flag("triggeredByBrand") {
				if triggerCD != nil {
					simCD = triggerCD
				} else {
					simCD = triggeredCD
				}
			}
			rate := env.calcMultiSpellRotationImpact(rotation, output.N("EffectiveSourceRate"), simCD, triggerChance, actor)
			if actor.db.Flag(nil, "HaveTriggerBots") && main.SkillTypes[modparser.SkillTypeSpell] {
				rate = 2 * rate
			}
			// stagesAreOverlaps is the skill part which makes the stages
			// behave as overlaps
			if hitsPerCast, ok := env.stageOverlaps(config); ok {
				rate = hitsPerCast * rate
			}
			output.SetN("SkillTriggerRate", rate)
		}
	} else {
		output.SetN("SkillTriggerRate", 0.0)
	}
	main.SkillData.Set("triggerRate", output.Get("SkillTriggerRate"))

	// Account for Trigger-related INC/MORE modifiers
	output.Set("Speed", main.SkillData.Get("triggerRate"))
	if source != nil {
		addTriggerIncMoreMods(main, source)
	} else {
		addTriggerIncMoreMods(main, main)
	}
	if source != nil && source != main {
		main.SkillData.SetStr("triggerSourceUUID", env.cacheSkillUUID(source))
	}
}

// triggeredByCooldown is
// `triggeredBy.grantedEffect.levels[triggeredBy.level].cooldown`.
func triggeredByCooldown(triggeredBy *ActiveEffect) *float64 {
	if triggeredBy == nil || triggeredBy.GrantedEffect == nil {
		return nil
	}
	lvl := triggeredBy.GrantedEffect.LevelData(triggeredBy.Level)
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
			if skill.SkillData.Has("triggerTime") && canSupport && skill != main && slotMatch && !isTriggered(skill) {
				source = skill
			} else if disabledSource == nil && canSupport && skill.SkillFlags["disable"] &&
				skill.SkillTypes[modparser.SkillTypeChannel] && skill != main && slotMatch {
				// A channelling skill is socketed but unusable (commonly a
				// support gem restricting its weapon types). Remember it so
				// we don't fall through to the Self-Cast estimate below,
				// which would report a *higher* DPS for a setup that cannot
				// actually trigger at all.
				disabledSource = skill
			}
		}
		if skill.SkillData.Flag("triggeredWhileChannelling") && slotMatch {
			triggeredSkills = append(triggeredSkills, env.packageSkillDataForSimulation(skill))
		}
	}
	if source == nil && disabledSource != nil {
		main.SkillFlags["disable"] = true
		main.DisableReason = disabledSource.ActiveEffect.GrantedEffect.Name + " is disabled"
		main.InfoMessage = triggerName + " Triggering Skill is disabled"
	} else if source == nil || len(triggeredSkills) < 1 {
		main.SkillData.Del("triggered")
		main.InfoMessage2 = "DPS reported assuming Self-Cast"
		main.InfoMessage = "No " + triggerName + " Triggering Skill Found"
	} else {
		if act := processAddedCastTime(main); act != nil {
			output.SetN("addsCastTime", *act)
		}

		icdr := Mod(main.SkillModList, main.SkillCfg, "CooldownRecovery")
		triggerInterval := source.SkillData.N("triggerTime")
		triggerRateOfTrigger := 1 / triggerInterval
		cooldownOverride, _ := main.SkillModList.Override(main.SkillCfg, "CooldownRecovery")
		if modparser.Truthy(cooldownOverride) {
			main.SkillFlags["hasOverride"] = true
		}

		// `cooldownOverride or m_max(triggeredCD or 0, addsCastTime or 0) / icdr`
		triggeredTotalCooldown := math.Max(main.SkillData.N("cooldown"), output.N("addsCastTime")) / icdr
		if modparser.Truthy(cooldownOverride) {
			triggeredTotalCooldown = valueNum(cooldownOverride)
		}
		triggeredCDAdjusted := math.Ceil(triggeredTotalCooldown*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
		effCDTriggeredSkill := math.Ceil(triggeredCDAdjusted*triggerRateOfTrigger) / triggerRateOfTrigger

		output.SetN("TriggerRateCap", math.Min(1/effCDTriggeredSkill, triggerRateOfTrigger))
		zero := 0.0
		rate := env.calcMultiSpellRotationImpact(triggeredSkills, triggerRateOfTrigger, &zero, 100, env.playerPA)
		if env.ModDB.Flag(nil, "HaveTriggerBots") && main.SkillTypes[modparser.SkillTypeSpell] {
			rate = 2 * rate
		}
		output.SetN("SkillTriggerRate", rate)

		// Account for Trigger-related INC/MORE modifiers
		addTriggerIncMoreMods(main, main)
		output.SetN("ChannelTimeToTrigger", triggerInterval)
		main.SkillData.SetFlag("triggered", true)
		main.SkillFlags["globalTrigger"] = true
		main.SkillData.Set("triggerRate", output.Get("SkillTriggerRate"))
		main.SkillData.SetStr("triggerSourceUUID", env.cacheSkillUUID(source))
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
	main.SkillData.SetFlag("triggered", true)
	triggerCD := triggeredByCooldown(main.TriggeredBy)
	icdrFocus := Mod(main.SkillModList, main.SkillCfg, "FocusCooldownRecovery")
	icdrSkill := Mod(main.SkillModList, main.SkillCfg, "CooldownRecovery")

	// Next possible activation is duration + cooldown.
	skillFocus := data.Skills["Focus"]
	focusDuration := skillFocus.ConstantStats[0].Value / 1000
	focusCD := skillFocus.Levels[1].Extra["cooldown"] / icdrFocus
	focusTotalCD := focusDuration + focusCD

	// The skill's own cooldown still applies to focus triggers.
	modActionCooldown := math.Max(main.SkillData.N("cooldown"), num64(triggerCD)/icdrSkill)
	rateCapAdjusted := math.Ceil(modActionCooldown*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	triggerRate := math.Inf(1)
	if rateCapAdjusted != 0 {
		triggerRate = 1 / rateCapAdjusted
	}
	output.SetN("TriggerRateCap", triggerRate)
	output.SetN("SkillTriggerRate", 1/focusTotalCD)

	// Account for Trigger-related INC/MORE modifiers
	addTriggerIncMoreMods(main, main)
	main.InfoMessage = "Assuming perfect focus Re-Use"
	main.SkillData.Set("triggerRate", output.Get("SkillTriggerRate"))
	main.SkillFlags["globalTrigger"] = true
}

// stageOverlaps evaluates `config.stagesAreOverlaps and mainSkill.skillPart
// == it and srcInstance.skillStageCount` (L804/L827). No configTable
// entry in the archive ever SETS stagesAreOverlaps — the hook is
// reference-dead — so this always reports false today; it exists so the
// expression is the reference's rather than a guess if an entry ever grows
// one.
func (env *Env) stageOverlaps(config *triggerConfig) (float64, bool) {
	if config.stagesAreOverlaps == nil {
		return 0, false
	}
	main := env.PlayerMainSkill
	if main.SkillPart.V != *config.stagesAreOverlaps || main.ActiveEffect.SrcInstance == nil {
		return 0, false
	}
	v := main.ActiveEffect.SrcInstance.SkillStageCount
	if !v.Set {
		return 0, false
	}
	return v.V, true
}
