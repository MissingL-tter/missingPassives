// Port of .archive/src/Modules/CalcMirages.lua. Every mirage path builds a
// whole sub-environment (calcs.copyActiveSkill + a nested calcs.perform);
// the paths no corpus build reaches are ported as dispatch only and panic
// rather than guess. The return value is what CalcPerform gates the offence
// call on.
package calc

import (
	"strconv"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// MirageResult is env.player.mainSkill.mirage.
type MirageResult struct {
	Name          string
	Count         float64
	SkillPart     any
	SkillPartName string
	InfoMessage2  string
	Output        map[string]any
	MinionOutput  map[string]any
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
	case truthy(main.SkillData["triggeredByMirageArcher"]):
		config = env.mirageArcherConfig()
	case main.ActiveEffect.GrantedEffect.Name == "Reflection":
		panic("mirages: The Saviour / Reflection unported (no corpus build reaches it)")
	case main.ActiveEffect.GrantedEffect.Name == "Tawhoa's Chosen":
		panic("mirages: Tawhoa's Chosen unported (no corpus build reaches it)")
	case truthy(main.SkillData["triggeredBySacredWisps"]):
		config = env.sacredWispsConfig()
	case truthy(main.SkillData["triggeredByGeneralsCry"]):
		panic("mirages: General's Cry unported (no corpus build reaches it)")
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
		newSkill, newEnv := env.copyActiveSkill("CALCULATOR", mirageSkill)
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
		newSkill.SkillData["mirageUses"] = env.PlayerMainSkill.SkillData["storedUses"]
		newSkill.SkillTypes[modparser.SkillType.OtherThingUsesSkill] = true

		config.preCalcFunc(newSkill, newEnv)

		newEnv.PlayerMainSkill = newSkill
		newEnv.Perform()
		config.postCalcFunc(newSkill, newEnv)
	} else {
		config.mirageSkillNotFoundFunc()
	}
	return !config.calcMainSkillOffence
}

// copyActiveSkill ports calcs.copyActiveSkill (CalcActiveSkill L164): a fresh
// instance of one skill inside its own environment, so the mirage's copy can
// be modified without touching the player's.
func (env *Env) copyActiveSkill(mode string, skill *ActiveSkill) (*ActiveSkill, *Env) {
	activeEffect := &ActiveEffect{
		GrantedEffect: skill.ActiveEffect.GrantedEffect,
		Level:         skill.ActiveEffect.Level,
		Quality:       skill.ActiveEffect.Quality,
	}
	if src := skill.ActiveEffect.SrcInstance; src != nil {
		activeEffect.Level = anyNum(src.KV["level"])
		activeEffect.Quality = anyNum(src.KV["quality"])
		activeEffect.SrcInstance = src
		activeEffect.GemData = src.GemData
	}

	newSkill := env.createActiveSkill(activeEffect, skill.SupportList, skill.Actor, skill.SocketGroup, skill.SummonSkill)
	newEnv := initEnvOverride(env.Data, env.Build, mode, env.mirageReplay(), env.OverrideConditions)
	newEnv.buildActiveSkillModList(newSkill)
	newSkill.SkillModList = modstore.NewList(newSkill.BaseSkillModList)
	if newSkill.Minion != nil {
		// `newSkill.minion.modDB = new("ModDB")` with the minion itself as
		// the actor. This port splits the Lua minion table into Minion and
		// its modstore.Actor, and the actor is not built until Perform, so
		// the wiring here is not derivable from the reference alone.
		panic("mirages: copyActiveSkill minion branch unported (no corpus build reaches it)")
	}
	return newSkill, newEnv
}

// mirageReplay is the fixture the sub-environment's initEnv replays: the same
// build, but the node orders the dump recorded for that second initEnv.
func (env *Env) mirageReplay() *ReplayInput {
	sub := *env.Replay
	sub.AllocOrders = env.Replay.MirageAllocOrders
	sub.NodeOrders = env.Replay.MirageNodeOrders
	return &sub
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
			moreDamage := newSkill.SkillModList.Sum("BASE", newSkill.SkillCfg, "MirageArcherLessDamage")
			moreAttackSpeed := newSkill.SkillModList.Sum("BASE", newSkill.SkillCfg, "MirageArcherLessAttackSpeed")
			mirageCount := newSkill.SkillModList.Sum("BASE", main.SkillCfg, "MirageArcherMaxCount")

			main.Mirage = &MirageResult{
				Name:  newSkill.ActiveEffect.GrantedEffect.Name,
				Count: mirageCount,
			}
			if main.InfoMessage == "" {
				main.InfoMessage = luaNumStr(mirageCount) + " Mirage Archers using " + newSkill.ActiveEffect.GrantedEffect.Name
			}

			// Add new modifiers to new skill (which already has all the old
			// skill's modifiers). `mainSkill.ModFlags` / `.KeywordFlags`
			// are never assigned anywhere in the reference, so both nils
			// reach NewMod as no flags at all.
			newSkill.SkillModList.AddMod(newMod("Damage", "MORE", moreDamage, "Mirage Archer", nil, nil))
			newSkill.SkillModList.AddMod(newMod("Speed", "MORE", moreAttackSpeed, "Mirage Archer", nil, nil))

			// Does not use player resources
			newSkill.SkillModList.AddMod(newMod("HasNoCost", "FLAG", true, "Used by mirage"))

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

// luaNumStr is tostring() on a Lua number: %.14g.
func luaNumStr(v float64) string {
	return strconv.FormatFloat(v, 'g', 14, 64)
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
			lessDamage := newSkill.SkillModList.Sum("BASE", main.SkillCfg, "SacredWispsLessDamage")
			var wispsMaxCount, wispsCastChance float64
			// Find Wisps summoning skill for cast chance and wisp count
			for _, skill := range env.PlayerActiveSkills {
				if skill.ActiveEffect.GrantedEffect.Name == "Summon Sacred Wisps" {
					wispsCastChance = skill.SkillModList.Sum("BASE", main.SkillCfg, "SacredWispsChance")
					wispsMaxCount = skill.SkillModList.Sum("BASE", main.SkillCfg, "SacredWispsMaxCount")
					break
				}
			}

			main.Mirage = &MirageResult{
				Name:  newSkill.ActiveEffect.GrantedEffect.Name,
				Count: wispsMaxCount,
			}
			if main.InfoMessage == "" {
				main.InfoMessage = luaNumStr(wispsMaxCount) + " Sacred Wisps using " + newSkill.ActiveEffect.GrantedEffect.Name
			}

			// Add new modifiers to new skill (which already has all the old
			// skill's modifiers)
			newSkill.SkillModList.AddMod(newMod("Damage", "MORE", lessDamage, "Used by Sacred Wisps", nil, nil))
			newSkill.SkillModList.AddMod(newMod("Speed", "MORE", wispsCastChance-100, "Sacred Wisps cast chance", nil, nil))

			// Does not use player resources
			newSkill.SkillModList.AddMod(newMod("HasNoCost", "FLAG", true, "Used by Sacred Wisps"))

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
