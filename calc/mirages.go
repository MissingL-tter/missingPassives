// Port of .archive/src/Modules/CalcMirages.lua. Every mirage path builds a
// whole sub-environment (calcs.copyActiveSkill + a nested calcs.perform) and
// none of them is reachable by the corpus, so the dispatch is ported exactly
// and each active path panics rather than guessing. The return value is what
// CalcPerform gates the offence call on.
package calc

// RunMirages ports calcs.mirages: it reports whether the mirage machinery
// took over the main skill's calculation (in which case the caller skips
// calcs.offence).
func (env *Env) RunMirages() bool {
	main := env.PlayerMainSkill
	if (main.SkillCfg.SkillCond != nil && main.SkillCfg.SkillCond["usedByMirage"]) || main.SkillFlags["disable"] {
		return false
	}

	// The reference builds a `config` in this chain and ends with
	// `return calculateMirage(env, config)`, which returns nil when no
	// branch matched.
	switch {
	case truthy(main.SkillData["triggeredByMirageArcher"]):
		panic("mirages: Mirage Archer unported (no corpus build reaches it)")
	case main.ActiveEffect.GrantedEffect.Name == "Reflection":
		panic("mirages: The Saviour / Reflection unported (no corpus build reaches it)")
	case main.ActiveEffect.GrantedEffect.Name == "Tawhoa's Chosen":
		panic("mirages: Tawhoa's Chosen unported (no corpus build reaches it)")
	case truthy(main.SkillData["triggeredBySacredWisps"]):
		panic("mirages: Sacred Wisps unported (no corpus build reaches it)")
	case truthy(main.SkillData["triggeredByGeneralsCry"]):
		panic("mirages: General's Cry unported (no corpus build reaches it)")
	}
	// calculateMirage returns nil for a nil config.
	return false
}
