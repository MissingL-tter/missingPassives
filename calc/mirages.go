// Port of .archive/src/Modules/CalcMirages.lua. Every mirage path builds a
// whole sub-environment (calcs.copyActiveSkill + a nested calcs.perform);
// the paths no corpus build reaches are ported as dispatch only and panic
// rather than guess. The return value is what CalcPerform gates the offence
// call on.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// MirageResult is env.player.mainSkill.mirage.
type MirageResult struct {
	Name          string
	Count         float64
	SkillPart     util.Opt[float64]
	SkillPartName string
	InfoMessage2  string
	Output        modstore.Output
	MinionOutput  modstore.Output
}

// mirageConfig is the `config` table calculateMirage drives.
type mirageConfig struct {
	calcMainSkillOffence    bool
	mirageSkill             *ActiveSkill
	compareFunc             func(skill *ActiveSkill, mirageSkill *ActiveSkill) *ActiveSkill
	preCalcFunc             func(newSkill *ActiveSkill, newEnv *Env)
	postCalcFunc            func(newSkill *ActiveSkill, newEnv *Env)
	mirageSkillNotFoundFunc func()
}

// RunMirages ports calcs.mirages: it reports whether the mirage machinery
// took over the main skill's calculation (in which case the caller skips
// calcs.offence).
func (env *Env) RunMirages() bool {
	main := env.PlayerMainSkill
	if (main.SkillCfg.SkillCond != nil && main.SkillCfg.SkillCond["usedByMirage"]) || main.SkillFlags["disable"] {
		return false
	}

	var config *mirageConfig
	switch {
	case main.SkillData.Flag("triggeredByMirageArcher"):
		config = env.mirageArcherConfig()
	case main.ActiveEffect.GrantedEffect.Name == "Reflection":
		config = env.saviourConfig()
	case main.ActiveEffect.GrantedEffect.Name == "Tawhoa's Chosen":
		config = env.tawhoasChosenConfig()
	case main.SkillData.Flag("triggeredBySacredWisps"):
		config = env.sacredWispsConfig()
	case main.SkillData.Flag("triggeredByGeneralsCry"):
		env.generalsCryMirage()
	}
	return env.calculateMirage(config)
}

// calculateMirage ports the local of the same name. A nil config short-
// circuits to nil, which CalcPerform reads as "offence still runs".
func (env *Env) calculateMirage(config *mirageConfig) bool {
	if config == nil {
		return false
	}
	mirageSkill := config.mirageSkill
	if config.compareFunc != nil {
		for _, skill := range env.PlayerActiveSkills {
			if skill.SkillCfg.SkillCond == nil || !skill.SkillCfg.SkillCond["usedByMirage"] {
				mirageSkill = config.compareFunc(skill, mirageSkill)
			}
		}
	}

	if mirageSkill != nil {
		newSkill, newEnv := env.copyActiveSkill(ModeCalculator, mirageSkill)
		if newSkill.SkillCfg.SkillCond == nil {
			newSkill.SkillCfg.SkillCond = map[string]bool{}
		}
		newSkill.SkillCfg.SkillCond["usedByMirage"] = true
		delete(newSkill.SkillFlags, "multiPart")
		delete(newSkill.SkillFlags, "haveMinion")
		if newEnv.LimitedSkills == nil {
			newEnv.LimitedSkills = map[string]bool{}
		}
		newEnv.LimitedSkills[newEnv.cacheSkillUUID(newSkill)] = true
		newSkill.SkillData.Set("mirageUses", env.PlayerMainSkill.SkillData.Get("storedUses"))
		newSkill.SkillTypes[modparser.SkillTypeOtherThingUsesSkill] = true

		config.preCalcFunc(newSkill, newEnv)

		newEnv.PlayerMainSkill = newSkill
		// `calcs.perform(newEnv)` inherits whatever calcs.defence/offence
		// currently are: stubbed no-ops during the dump's checkpoint phase,
		// the real stages inside the cache driver.
		if newEnv.Replay.StubHandoff {
			newEnv.Perform()
		} else {
			newEnv.PerformFull(false)
		}
		config.postCalcFunc(newSkill, newEnv)
	} else {
		config.mirageSkillNotFoundFunc()
	}
	return !config.calcMainSkillOffence
}

// copyActiveSkill ports calcs.copyActiveSkill (CalcActiveSkill L164): a fresh
// instance of one skill inside its own environment, so the mirage's copy can
// be modified without touching the player's.
func (env *Env) copyActiveSkill(mode CalcMode, skill *ActiveSkill) (*ActiveSkill, *Env) {
	activeEffect := &ActiveEffect{
		GrantedEffect: skill.ActiveEffect.GrantedEffect,
		Level:         skill.ActiveEffect.Level,
		Quality:       skill.ActiveEffect.Quality,
	}
	if src := skill.ActiveEffect.SrcInstance; src != nil {
		activeEffect.Level = src.Level
		activeEffect.Quality = src.Quality
		activeEffect.SrcInstance = src
		activeEffect.GemData = src.GemData
	}

	newSkill := env.createActiveSkill(activeEffect, skill.SupportList, skill.Actor, skill.SocketGroup, skill.SummonSkill)
	newEnv := initEnvOverride(env.Build, mode, env.Replay, env.OverrideConditions)
	newEnv.buildActiveSkillModList(newSkill)
	newSkill.SkillModList = modstore.NewList(newSkill.BaseSkillModList)
	if newSkill.Minion != nil {
		// `newSkill.minion.modDB = new("ModDB")` with the minion itself as
		// the actor; this port's minion actor bridge is built here the same
		// way Perform builds it.
		m := newSkill.Minion
		m.DB = modstore.NewDB(nil)
		m.Ms = &modstore.Actor{
			DB:       m.DB,
			Level:    m.Level,
			Output:   modstore.Output{},
			Resolver: gemIds{},
			ItemList: map[string]modstore.Item{},
			MinionData: &modstore.MinionData{
				MonsterTags: m.MinionData.MonsterTags,
				DamageFixup: m.MinionData.DamageFixup,
			},
		}
		m.DB.Actor = m.Ms
		env.createMinionSkills(newSkill)
		newSkill.SkillPartName = m.MainSkill.ActiveEffect.GrantedEffect.Name
	}
	return newSkill, newEnv
}

// mirageArcherConfig ports the triggeredByMirageArcher branch: the archers
// use the player's own skill at reduced damage and speed, and the player's
// own offence still runs on top.
func (env *Env) mirageArcherConfig() *mirageConfig {
	main := env.PlayerMainSkill
	return &mirageConfig{
		calcMainSkillOffence: true,
		mirageSkill:          main,
		preCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			moreDamage := newSkill.SkillModList.Sum(modparser.Base, newSkill.SkillCfg, "MirageArcherLessDamage")
			moreAttackSpeed := newSkill.SkillModList.Sum(modparser.Base, newSkill.SkillCfg, "MirageArcherLessAttackSpeed")
			mirageCount := newSkill.SkillModList.Sum(modparser.Base, main.SkillCfg, "MirageArcherMaxCount")

			main.Mirage = &MirageResult{
				Name:  newSkill.ActiveEffect.GrantedEffect.Name,
				Count: mirageCount,
			}
			if main.InfoMessage == "" {
				main.InfoMessage = util.FormatG14(mirageCount) + " Mirage Archers using " + newSkill.ActiveEffect.GrantedEffect.Name
			}

			// Add new modifiers to new skill (which already has all the old
			// skill's modifiers). `mainSkill.ModFlags` / `.KeywordFlags`
			// are never assigned anywhere in the reference, so both nils
			// reach NewMod as no flags at all.
			newSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(moreDamage), "Mirage Archer"))
			newSkill.SkillModList.AddMod(newModS("Speed", modparser.More, modparser.Num(moreAttackSpeed), "Mirage Archer"))

			// Does not use player resources
			newSkill.SkillModList.AddMod(newModS("HasNoCost", modparser.Flag, modparser.Bool(true), "Used by mirage"))

			if newSkill.SkillPartName != "" {
				main.Mirage.SkillPart = newSkill.SkillPart
				main.Mirage.SkillPartName = newSkill.SkillPartName
				main.Mirage.InfoMessage2 = newSkill.ActiveEffect.GrantedEffect.Name
			} else {
				main.Mirage.SkillPartName = ""
			}
		},
		postCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			main.Mirage.Output = newEnv.Player.Output
			main.SkillFlags["mirageArcher"] = true
			if newSkill.Minion != nil {
				main.Mirage.MinionOutput = newEnv.Minion.Output
			}
		},
		mirageSkillNotFoundFunc: func() {
			if main.InfoMessage2 == "" {
				main.InfoMessage2 = "No Mirage Archer active skill found"
			}
		},
	}
}

// sacredWispsConfig ports the triggeredBySacredWisps branch: the wisps cast
// the player's own skill at reduced damage and a cast chance, and the
// player's own offence still runs on top.
func (env *Env) sacredWispsConfig() *mirageConfig {
	main := env.PlayerMainSkill
	return &mirageConfig{
		calcMainSkillOffence: true,
		mirageSkill:          main,
		preCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			lessDamage := newSkill.SkillModList.Sum(modparser.Base, main.SkillCfg, "SacredWispsLessDamage")
			var wispsMaxCount, wispsCastChance float64
			// Find Wisps summoning skill for cast chance and wisp count
			for _, skill := range env.PlayerActiveSkills {
				if skill.ActiveEffect.GrantedEffect.Name == "Summon Sacred Wisps" {
					wispsCastChance = skill.SkillModList.Sum(modparser.Base, main.SkillCfg, "SacredWispsChance")
					wispsMaxCount = skill.SkillModList.Sum(modparser.Base, main.SkillCfg, "SacredWispsMaxCount")
					break
				}
			}

			main.Mirage = &MirageResult{
				Name:  newSkill.ActiveEffect.GrantedEffect.Name,
				Count: wispsMaxCount,
			}
			if main.InfoMessage == "" {
				main.InfoMessage = util.FormatG14(wispsMaxCount) + " Sacred Wisps using " + newSkill.ActiveEffect.GrantedEffect.Name
			}

			// Add new modifiers to new skill (which already has all the old
			// skill's modifiers)
			newSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(lessDamage), "Used by Sacred Wisps"))
			newSkill.SkillModList.AddMod(newModS("Speed", modparser.More, modparser.Num(wispsCastChance-100), "Sacred Wisps cast chance"))

			// Does not use player resources
			newSkill.SkillModList.AddMod(newModS("HasNoCost", modparser.Flag, modparser.Bool(true), "Used by Sacred Wisps"))

			if newSkill.SkillPartName != "" {
				main.Mirage.SkillPart = newSkill.SkillPart
				main.Mirage.SkillPartName = newSkill.SkillPartName
				main.Mirage.InfoMessage2 = newSkill.ActiveEffect.GrantedEffect.Name
			} else {
				main.Mirage.SkillPartName = ""
			}
		},
		postCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			main.Mirage.Output = newEnv.Player.Output
			main.SkillFlags["wisp"] = true
			if newSkill.Minion != nil {
				main.Mirage.MinionOutput = newEnv.Minion.Output
			}
		},
		mirageSkillNotFoundFunc: func() {
			if main.InfoMessage2 == "" {
				main.InfoMessage2 = "No active skill for Sacred Wisps found"
			}
		},
	}
}

// saviourConfig ports the Reflection (The Saviour) branch (CalcMirages
// L118): the mirage warriors copy the best one-hand-sword attack, and the
// mirage REPLACES the main skill's calculation outright.
func (env *Env) saviourConfig() *mirageConfig {
	main := env.PlayerMainSkill
	var usedSkillBestDps float64
	maxMirageWarriors := main.SkillModList.Sum(modparser.Base, main.SkillCfg, "SaviourMirageWarriorMaxCount")
	return &mirageConfig{
		compareFunc: func(skill *ActiveSkill, mirageSkill *ActiveSkill) *ActiveSkill {
			swordOneHand := modparser.FlagSword | modparser.FlagWeapon1H
			usedByMirage := skill.SkillCfg != nil && skill.SkillCfg.SkillCond != nil && skill.SkillCfg.SkillCond["usedByMirage"]
			// #EVAL: the reference also checks `SkillType.Totem`, which
			// Global.lua never defines -- the nil index always reads nil, so
			// that arm is dead.
			if skill != main && skill.SkillTypes[modparser.SkillTypeAttack] &&
				!skill.SkillTypes[modparser.SkillTypeSummonsTotem] &&
				skill.SkillCfg != nil && skill.SkillCfg.Flags != nil && *skill.SkillCfg.Flags&swordOneHand == swordOneHand &&
				!usedByMirage {
				uuid := env.cacheSkillUUID(skill)
				if env.GlobalCache[uuid] == nil {
					env.BuildActiveSkill(env.Mode, skill, uuid)
				}
				if c := env.GlobalCache[uuid]; c != nil && c.CritChance != nil && *c.CritChance > 0 {
					if mirageSkill == nil || (c.TotalDPS != nil && *c.TotalDPS > usedSkillBestDps) {
						if c.TotalDPS != nil {
							usedSkillBestDps = *c.TotalDPS
						}
						return c.ActiveSkill
					}
				}
			}
			return mirageSkill
		},
		preCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			moreDamage := main.SkillModList.Sum(modparser.Base, main.SkillCfg, "SaviourMirageWarriorLessDamage")
			// Add new modifiers to new skill (which already has all the old
			// skill's modifiers)
			newSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(moreDamage), "The Saviour"))
			w1, _ := env.Player.ItemList["Weapon 1"].(*Item)
			w2, _ := env.Player.ItemList["Weapon 2"].(*Item)
			if w1 != nil && w2 != nil && w1.In.Name == w2.In.Name {
				maxMirageWarriors = maxMirageWarriors / 2
			}
			newSkill.SkillModList.AddMod(newModS("QuantityMultiplier", modparser.Base, modparser.Num(maxMirageWarriors), "The Saviour Mirage Warriors"))
			// Does not use player resources
			newSkill.SkillModList.AddMod(newModS("HasNoCost", modparser.Flag, modparser.Bool(true), "Used by mirage"))
		},
		postCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			// The mirage REPLACES the main skill and its output.
			env.PlayerMainSkill = newSkill
			env.PlayerMainSkill.InfoMessage = util.FormatG14(maxMirageWarriors) + " Mirage Warriors using " + newSkill.ActiveEffect.GrantedEffect.Name
			env.Player.Output = newEnv.Player.Output
			if env.playerPA != nil {
				env.playerPA.takeOutput(newEnv.playerPA)
				env.playerPA.mainSkill = newSkill
			}
		},
		mirageSkillNotFoundFunc: func() {
			main.DisableReason = "No Saviour active skill found"
			main.SkillFlags["disable"] = true
		},
	}
}

// generalsCryMirage ports the triggeredByGeneralsCry branch (CalcMirages
// L344): unlike the other paths it builds no config -- it rewrites the main
// skill in place (offence still runs on it) using the cached output of the
// skill's own solo build.
func (env *Env) generalsCryMirage() {
	main := env.PlayerMainSkill
	maxMirageWarriors := 0.0
	cooldown := 1.0
	var generalsCryActiveSkill *ActiveSkill
	uuid := env.cacheSkillUUID(main)

	// Prevent infinite recursion
	if env.LimitedSkills[uuid] {
		return
	}

	main.SkillTypes[modparser.SkillTypeTriggered] = true
	if main.SkillCfg.SkillCond == nil {
		main.SkillCfg.SkillCond = map[string]bool{}
	}
	main.SkillCfg.SkillCond["usedByMirage"] = true
	main.SkillTypes[modparser.SkillTypeOtherThingUsesSkill] = true

	if env.GlobalCache[uuid] == nil || env.Mode == ModeCalculator {
		env.BuildActiveSkill(env.Mode, main, uuid, uuid)
	}
	mainSkillOutputCache := env.GlobalCache[uuid]

	// Find the active General's Cry gem to get active properties
	for _, skill := range env.PlayerActiveSkills {
		if skill.ActiveEffect.GrantedEffect.Name == "General's Cry" && sameSocketSlot(skill, main) {
			cooldown, _ = env.calcSkillCooldown(skill.SkillModList, skill.SkillCfg, skill.SkillData)
			generalsCryActiveSkill = skill
			break
		}
	}

	// Scale dps with mirage quantity
	for _, value := range generalsCryActiveSkill.SkillModList.Tabulate(modparser.Base, generalsCryActiveSkill.SkillCfg, "GeneralsCryDoubleMaxCount") {
		m := value.Mod
		main.SkillModList.AddMod(newModSF("QuantityMultiplier", m.Type, m.Value, m.Source, m.Flags, m.KeywordFlags))
		maxMirageWarriors += valueNum(m.Value)
	}

	// Scale cooldown to have maximum number of Mirages at once: 0.3s for the
	// first mirage then 0.2s for each extra
	mirageSpawnTime := 0.3 + 0.2*maxMirageWarriors
	if main.SkillTypes[modparser.SkillTypeChannel] {
		mirageSpawnTime = mirageSpawnTime + 1
	} else {
		main.SkillData.SetN("hitTimeOverride", 1.0)
	}
	// Consistent with the info message; removing this could make the numbers
	// more accurate
	mirageSpawnTime = util.RoundHalfUp(mirageSpawnTime, 2)
	_ = mirageSpawnTime // info message only

	// Scale dps with GC's cooldown / attack time
	hitOrTime := mainSkillOutputCache.out("Time").Num()
	if v := mainSkillOutputCache.out("HitTime"); v.Truthy() {
		hitOrTime = v.Num()
	}
	cooldown = math.Max(cooldown, hitOrTime)
	main.SkillModList.AddMod(newModS("DPS", modparser.More, modparser.Num((1/cooldown-1)*100), "General's Cry Cooldown"))

	// Does not use player resources
	main.SkillModList.AddMod(newModS("HasNoCost", modparser.Flag, modparser.Bool(true), "Used by mirage"))

	// Supported Attacks Count as Exerted
	for _, spec := range []struct {
		typ       modparser.ModType
		name, out string
	}{
		{modparser.Inc, "ExertIncrease", "Damage"},
		{modparser.More, "ExertIncrease", "Damage"},
		{modparser.More, "ExertAttackIncrease", "Damage"},
		{modparser.More, "OverexertionExertAverageIncrease", "Damage"},
		{modparser.Base, "ExertDoubleDamageChance", "DoubleDamageChance"},
	} {
		for _, value := range main.SkillModList.Tabulate(spec.typ, main.SkillCfg, spec.name) {
			m := value.Mod
			main.SkillModList.AddMod(newModSF(spec.out, m.Type, m.Value, m.Source, m.Flags, m.KeywordFlags))
		}
	}
}

// tawhoasChosenConfig ports the Tawhoa's Chosen branch (CalcMirages L163):
// the mirage chieftain repeats the best slam/strike, replacing the main
// skill's calculation, with the trigger rate standing in for attack speed.
func (env *Env) tawhoasChosenConfig() *mirageConfig {
	main := env.PlayerMainSkill
	var usedSkillBestDps, effectiveSourceRate float64
	var triggerRateCap, skillTriggerRate float64
	return &mirageConfig{
		compareFunc: func(skill *ActiveSkill, mirageSkill *ActiveSkill) *ActiveSkill {
			isDisabled := skill.SkillFlags["disable"]
			skillTypeMatch := (skill.SkillTypes[modparser.SkillTypeSlam] || skill.SkillTypes[modparser.SkillTypeMelee]) &&
				skill.SkillTypes[modparser.SkillTypeAttack]
			skillTypeExcludes := skill.SkillTypes[modparser.SkillTypeVaal] || skill.SkillTypes[modparser.SkillTypeSummonsTotem]
			usedByMirage := skill.SkillCfg != nil && skill.SkillCfg.SkillCond != nil && skill.SkillCfg.SkillCond["usedByMirage"]
			if skill != main && !isTriggered(skill) && !isDisabled && skillTypeMatch && !skillTypeExcludes && !usedByMirage {
				uuid := env.cacheSkillUUID(skill)
				if env.GlobalCache[uuid] == nil || env.Mode == ModeCalculator {
					env.BuildActiveSkill(env.Mode, skill, uuid)
				}
				c := env.GlobalCache[uuid]
				if mirageSkill == nil || (c != nil && c.TotalDPS != nil && *c.TotalDPS > usedSkillBestDps) {
					if c != nil {
						if c.TotalDPS != nil {
							usedSkillBestDps = *c.TotalDPS
						}
						if c.Speed != nil {
							effectiveSourceRate = *c.Speed
						}
						return c.ActiveSkill
					}
				}
			}
			return mirageSkill
		},
		preCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			icdrSkill := Mod(newSkill.SkillModList, newSkill.SkillCfg, "CooldownRecovery")

			triggeredCD := newSkill.SkillData.N("cooldown")
			triggeredCDAdjusted := triggeredCD / icdrSkill
			triggeredCDTickRounded := math.Ceil(triggeredCDAdjusted*data.Misc.ServerTickRate) / data.Misc.ServerTickRate

			triggerCD := main.SkillData.N("cooldown")
			triggerCDAdjusted := triggerCD / icdrSkill
			triggerCDTickRounded := math.Ceil(triggerCDAdjusted*data.Misc.ServerTickRate) / data.Misc.ServerTickRate

			actionCooldown := math.Max(triggeredCDTickRounded, triggerCDTickRounded)

			triggerRateCap = math.Inf(1)
			if actionCooldown != 0 {
				triggerRateCap = 1 / actionCooldown
			}

			skillTriggerRate = 0
			if effectiveSourceRate != 0 {
				sim := &simSkill{uuid: env.cacheSkillUUID(main), cd: &triggeredCD, icdr: &icdrSkill}
				skillTriggerRate = env.calcMultiSpellRotationImpact([]*simSkill{sim}, effectiveSourceRate, &triggerCD, 100, env.playerPA)
			}

			// Override attack speed with the trigger rate
			newSkill.SkillData.SetN("triggerRate", skillTriggerRate)
			newSkill.SkillData.SetFlag("triggered", true)
			newSkill.SkillFlags["triggered"] = true

			// Does not use player resources
			newSkill.SkillModList.AddMod(newModS("HasNoCost", modparser.Flag, modparser.Bool(true), "Used by Tawhoa's Chosen"))

			moreDamage := main.SkillModList.Sum(modparser.Base, main.SkillCfg, "ChieftainMirageChieftainMoreDamage")
			// Add new modifiers to new skill (which already has all the old
			// skill's modifiers)
			newSkill.SkillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(moreDamage), "Tawhoa's Chosen"))
		},
		postCalcFunc: func(newSkill *ActiveSkill, newEnv *Env) {
			env.PlayerMainSkill = newSkill
			env.PlayerMainSkill.InfoMessage = "Tawhoa's Chosen using " + newSkill.ActiveEffect.GrantedEffect.Name

			env.Player.Output = newEnv.Player.Output
			env.Player.Output.SetN("Speed", skillTriggerRate)
			env.Player.Output.SetN("TriggerRateCap", triggerRateCap)
			env.Player.Output.SetN("EffectiveSourceRate", effectiveSourceRate)
			env.Player.Output.SetN("SkillTriggerRate", skillTriggerRate)
			if env.playerPA != nil {
				env.playerPA.takeOutput(newEnv.playerPA)
				env.playerPA.mainSkill = newSkill
			}
		},
		mirageSkillNotFoundFunc: func() {
			main.DisableReason = "No Tawhoa's Chosen active skill found"
			main.SkillFlags["disable"] = true
		},
	}
}
