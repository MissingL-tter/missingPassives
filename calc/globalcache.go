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

	"github.com/MissingL-tter/missingPassives/modstore"
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
func (c *CachedSkill) out(key string) modstore.OutValue {
	if c == nil || c.Env == nil || c.Env.Player == nil {
		return modstore.OutValue{}
	}
	return c.Env.Player.Output.Get(key)
}

// outputMainHand / outputOffHand are output.MainHand and output.OffHand of
// the cached env -- the per-weapon passes a dual-wielding source leaves
// behind.
func (c *CachedSkill) outputMainHand(key string) modstore.OutValue {
	if c == nil || c.Env == nil || c.Env.playerPA == nil {
		return modstore.OutValue{}
	}
	return c.Env.playerPA.mainHand.Get(key)
}

func (c *CachedSkill) outputOffHand(key string) modstore.OutValue {
	if c == nil || c.Env == nil || c.Env.playerPA == nil {
		return modstore.OutValue{}
	}
	return c.Env.playerPA.offHand.Get(key)
}

// PlayerWeaponOutputs is the player's output.MainHand / output.OffHand
// (nil until an attack's offence ran).
func (env *Env) PlayerWeaponOutputs() (mainHand, offHand modstore.Output) {
	if env.playerPA == nil {
		return nil, nil
	}
	return env.playerPA.mainHand, env.playerPA.offHand
}

// mainSkillData is the cached env's CURRENT main skill's data; activeSkillData
// is that of the skill the entry was cached for. They are the same table
// unless the env was re-performed for another skill.
func (c *CachedSkill) mainSkillData(key string) modstore.OutValue {
	if c == nil || c.Env == nil || c.Env.PlayerMainSkill == nil {
		return modstore.OutValue{}
	}
	return c.Env.PlayerMainSkill.SkillData.Get(key)
}

func (c *CachedSkill) activeSkillData(key string) modstore.OutValue {
	if c == nil || c.ActiveSkill == nil {
		return modstore.OutValue{}
	}
	return c.ActiveSkill.SkillData.Get(key)
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
		if slot := skill.SocketGroup.Slot; slot != "" {
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
		if !output.Has(key) {
			return nil
		}
		n := output.N(key)
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
