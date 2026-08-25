// GlobalCache: the per-skill output cache Data/Global.lua declares and
// Common.lua's cacheData fills, at the end of every calcs.perform. Calcs.lua's
// buildOutput driver is what populates it for skills other than the main one,
// by running a whole perform per skill (calcs.buildActiveSkill).
//
// calcs.triggers is its only reader, and it reads a bounded set of fields.
package calc

import (
	"strconv"
	"strings"
)

// CachedSkill is one GlobalCache.cachedData[mode][uuid] entry.
type CachedSkill struct {
	Name                   string   `lua:"Name"`
	Speed                  *float64 `lua:"Speed"`
	HitSpeed               *float64 `lua:"HitSpeed"`
	ManaCost               *float64 `lua:"ManaCost"`
	LifeCost               *float64 `lua:"LifeCost"`
	ESCost                 *float64 `lua:"ESCost"`
	RageCost               *float64 `lua:"RageCost"`
	HitChance              *float64 `lua:"HitChance"`
	AccuracyHitChance      *float64 `lua:"AccuracyHitChance"`
	PreEffectiveCritChance *float64 `lua:"PreEffectiveCritChance"`
	CritChance             *float64 `lua:"CritChance"`
	TotalDPS               *float64 `lua:"TotalDPS"`

	// The reference stores the env and the skill themselves, so every read
	// through them is LIVE: a stage that runs after the entry was cached
	// changes what the entry appears to hold. The accessors below preserve
	// that -- do not snapshot these.
	ActiveSkill *ActiveSkill
	Env         *Env
}

// out reads a cached Env.player.output value, live.
func (c *CachedSkill) out(key string) any {
	if c == nil || c.Env == nil || c.Env.Player == nil {
		return nil
	}
	return c.Env.Player.Output[key]
}

// outputMainHand / outputOffHand are output.MainHand and output.OffHand of
// the cached env -- the per-weapon passes a dual-wielding source leaves
// behind.
func (c *CachedSkill) outputMainHand(key string) any { return c.outputSub("MainHand", key) }
func (c *CachedSkill) outputOffHand(key string) any  { return c.outputSub("OffHand", key) }

func (c *CachedSkill) outputSub(table, key string) any {
	if c == nil || c.Env == nil || c.Env.Player == nil {
		return nil
	}
	sub, _ := c.Env.Player.Output[table].(map[string]any)
	return sub[key]
}

// mainSkillData is the cached env's CURRENT main skill's data; activeSkillData
// is that of the skill the entry was cached for. They are the same table
// unless the env was re-performed for another skill.
func (c *CachedSkill) mainSkillData(key string) (any, bool) {
	if c == nil || c.Env == nil || c.Env.PlayerMainSkill == nil {
		return nil, false
	}
	v, ok := c.Env.PlayerMainSkill.SkillData[key]
	return v, ok
}

func (c *CachedSkill) activeSkillData(key string) (any, bool) {
	if c == nil || c.ActiveSkill == nil {
		return nil, false
	}
	v, ok := c.ActiveSkill.SkillData[key]
	return v, ok
}

// speedOrHitSpeed is `cached.HitSpeed or cached.Speed`.
func (c *CachedSkill) speedOrHitSpeed() (float64, bool) {
	if c == nil {
		return 0, false
	}
	if c.HitSpeed != nil {
		return *c.HitSpeed, true
	}
	if c.Speed != nil {
		return *c.Speed, true
	}
	return 0, false
}

// cacheSkillUUID ports Common.lua's cacheSkillUUID:
// "<name sans spaces>_<SLOT sans spaces>_<gem index>_<group index>".
func (env *Env) cacheSkillUUID(skill *ActiveSkill) string {
	strip := func(s string) string { return strings.Join(strings.Fields(s), "") }
	strName := strip(skill.ActiveEffect.GrantedEffect.Name)
	strSlotName := "NO_SLOT"
	if skill.SocketGroup != nil {
		if slot := str(skill.SocketGroup.KV["slot"]); slot != "" {
			strSlotName = strip(strings.ToUpper(slot))
		}
	}
	slotIdx, groupIdx := 1, 1
	if skill.SocketGroup != nil && skill.SocketGroup.GemList != nil && skill.ActiveEffect.SrcInstance != nil {
		// compares table identity, not names: the same gem can be socketed
		// twice in one slot
		for idx, gem := range skill.SocketGroup.GemList {
			if gem == skill.ActiveEffect.SrcInstance {
				slotIdx = idx + 1
				break
			}
		}
	}
	for i, group := range env.Build.SkillsTab.SocketGroups {
		if skill.SocketGroup == group {
			groupIdx = i + 1
			break
		}
	}
	return strName + "_" + strSlotName + "_" + strconv.Itoa(slotIdx) + "_" + strconv.Itoa(groupIdx)
}

// cached returns the cache entry for a skill, or nil when the fixture has
// none. The reference would call calcs.buildActiveSkill to fill a miss; the
// replay cannot, so callers that would need one panic instead of guessing.
func (env *Env) cached(skill *ActiveSkill) *CachedSkill {
	return env.GlobalCache[env.cacheSkillUUID(skill)]
}

// cacheData ports Common.lua's cacheData (L862), the last statement of
// calcs.perform: the main skill's headline numbers plus back-references to
// the env and skill that produced them.
func (env *Env) cacheData() {
	main := env.PlayerMainSkill
	output := env.Player.Output
	num := func(key string) *float64 {
		v, ok := output[key]
		if !ok || v == nil {
			return nil
		}
		n := anyNum(v)
		return &n
	}
	entry := &CachedSkill{
		Name:                   main.ActiveEffect.GrantedEffect.Name,
		Speed:                  num("Speed"),
		HitSpeed:               num("HitSpeed"),
		ManaCost:               num("ManaCost"),
		LifeCost:               num("LifeCost"),
		ESCost:                 num("ESCost"),
		RageCost:               num("RageCost"),
		HitChance:              num("HitChance"),
		AccuracyHitChance:      num("AccuracyHitChance"),
		PreEffectiveCritChance: num("PreEffectiveCritChance"),
		CritChance:             num("CritChance"),
		TotalDPS:               num("TotalDPS"),
		ActiveSkill:            main,
		Env:                    env,
	}
	if env.GlobalCache == nil {
		env.GlobalCache = map[string]*CachedSkill{}
	}
	env.GlobalCache[env.cacheSkillUUID(main)] = entry
}
