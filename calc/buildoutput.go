// Port of Calcs.lua's output driver: the layer above the stages, which runs
// a whole calcs.perform per skill so that GlobalCache holds an entry for
// every active skill before anything reads one.
package calc

import "github.com/MissingL-tter/missingPassives/data"

// BuildActiveSkill ports calcs.buildActiveSkill (Calcs.lua L391): build a
// fresh environment, find the skill in it by uuid, and run a full perform on
// it. The point is the cacheData at the end of that perform -- the caller
// wants the cache entry, not the env.
//
// limitedProcessing carries uuids that must not recurse into another
// buildActiveSkill; the reference uses it to stop infinite loops.
func (env *Env) BuildActiveSkill(mode string, skill *ActiveSkill, targetUUID string, limitedProcessing ...string) {
	// Defensive, no reference counterpart: the reference's recursion
	// breaker is the limitedSkills check at the top of calcs.triggers, and
	// a bug in porting it once recursed unboundedly, allocating a full
	// environment per level until the machine ran out of memory. Depth 20
	// is far beyond anything the real dependency chains reach.
	if env.buildDepth >= 20 {
		panic("calc: BuildActiveSkill recursion depth exceeded (missing limitedSkills guard?)")
	}
	replay := *env.Replay
	replay.GlobalCache = nil
	fullEnv := initEnvOverride(env.Data, env.Build, mode, &replay, env.OverrideConditions)
	fullEnv.buildDepth = env.buildDepth + 1

	// env.limitedSkills contains a map of uuids that should be limited in
	// calculation, in order to prevent infinite recursion loops
	if fullEnv.LimitedSkills == nil {
		fullEnv.LimitedSkills = map[string]bool{}
	}
	for uuid := range env.LimitedSkills {
		fullEnv.LimitedSkills[uuid] = true
	}
	for _, uuid := range limitedProcessing {
		fullEnv.LimitedSkills[uuid] = true
	}

	if targetUUID == "" {
		targetUUID = env.cacheSkillUUID(skill)
	}
	for _, activeSkill := range fullEnv.PlayerActiveSkills {
		if fullEnv.cacheSkillUUID(activeSkill) == targetUUID {
			fullEnv.PlayerMainSkill = activeSkill
			fullEnv.GlobalCache = env.GlobalCache
			fullEnv.PerformFull(true)
			return
		}
	}
	// The reference ConPrintfs and carries on with no cache entry; the
	// callers all guard on the entry being present, so do the same.
}

// FillGlobalCache ports the cache-filling half of calcs.buildOutput
// (Calcs.lua L417-455): every active skill that nothing has cached yet gets
// its own environment and its own perform.
//
// The rest of buildOutput is display work -- the FullDPS roll-up, the cost
// warnings, and the conditions/multipliers discovery the config tab reads --
// none of which any ported stage consumes.
func (env *Env) FillGlobalCache(mode string) {
	if env.GlobalCache == nil {
		env.GlobalCache = map[string]*CachedSkill{}
	}
	for _, skill := range env.PlayerActiveSkills {
		uuid := env.cacheSkillUUID(skill)
		if env.GlobalCache[uuid] == nil {
			env.BuildActiveSkill(mode, skill, uuid)
		}
	}
}

// BuildOutput ports the driver itself: one env for the main skill, a full
// perform on it, then a cache entry for every other active skill.
func BuildOutput(d *data.Data, in *BuildInput, mode string, replay *ReplayInput) *Env {
	// The driver computes the cache; anything the fixture carried would
	// mask that.
	own := *replay
	own.GlobalCache = nil
	env := InitEnv(d, in, mode, &own)
	env.PerformFull(false)
	if mode == "MAIN" {
		env.FillGlobalCache(mode)
	}
	return env
}
