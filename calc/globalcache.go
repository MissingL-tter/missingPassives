// GlobalCache: the per-skill output cache Data/Global.lua declares and
// Common.lua's cacheData fills. It is produced by Calcs.lua's buildOutput
// driver (which runs a whole perform per skill), not by any of the stages
// ported here, so the replay consumes it as a fixture — the same boundary
// the tree alloc orders and the Energy Blade weapons already use.
//
// calcs.triggers is its only reader, and it reads a bounded set of fields;
// tools/dump_calc.lua captures exactly those, snapshotted at the moment the
// reference reaches the trigger stage.
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

	// Scalar slices of the cached env that CalcTriggers reaches into.
	Output          map[string]any `lua:"output"`
	OutputMainHand  map[string]any `lua:"outputMainHand"`
	OutputOffHand   map[string]any `lua:"outputOffHand"`
	MainSkillData   map[string]any `lua:"mainSkillData"`
	ActiveSkillData map[string]any `lua:"activeSkillData"`
}

// out reads a cached Env.player.output value.
func (c *CachedSkill) out(key string) any {
	if c == nil {
		return nil
	}
	return c.Output[key]
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
