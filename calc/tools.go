// Port of .archive/src/Modules/CalcTools.lua (calcLib). The functions keep
// the reference's evaluation order and Lua-truthiness edge cases; only the
// signatures are Go-shaped (explicit *data.Data instead of the Lua global).
//
// canGrantedEffectSupportActiveSkill is deferred to the skills stage - it
// reads the full ActiveSkill shape, which is defined when
// CalcActiveSkill.lua ports.
package calc

import (
	"math"
	"sort"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// Mod is calcLib.mod: (1 + Sum(INC)/100) * More.
func Mod(store modstore.Store, cfg *modstore.Cfg, names ...string) float64 {
	return (1 + store.Sum("INC", cfg, names...)/100) * store.More(cfg, names...)
}

// Mods is calcLib.mods.
func Mods(store modstore.Store, cfg *modstore.Cfg, names ...string) (inc, more float64) {
	return 1 + store.Sum("INC", cfg, names...)/100, store.More(cfg, names...)
}

// Val is calcLib.val.
func Val(store modstore.Store, name string, cfg *modstore.Cfg) float64 {
	baseVal := store.Sum("BASE", cfg, name)
	if baseVal != 0 {
		return baseVal * Mod(store, cfg, name)
	}
	return 0
}

// ActiveEffect is the active/support effect table (also the shape
// validateGemLevel and buildSkillInstanceStats see: grantedEffect, gemData,
// level, quality, actorLevel).
type ActiveEffect struct {
	GrantedEffect *data.GrantedEffect
	GemData       *data.Gem
	Level         float64
	Quality       float64
	ActorLevel    *float64

	GlobalQuality  float64
	ItemQuality    float64
	SupportQuality float64
	SocketQuality  float64
	SrcInstance    *SocketGemInput
	Superseded     bool
	IsSupporting   map[*SocketGemInput]bool
	GemCfg         *modstore.Cfg
	// Req is effect.req — set only when a GemProperty mod carries key
	// "req"; nil means unset (gemInstance.reqOverride stays nil).
	Req *float64
	// Extra catches applyGemMods keys beyond level/quality/req.
	Extra map[string]float64
	// Arcanist Brand back-fields: its trigger config stashes the brand's
	// activation numbers on the triggeredBy effect for the handler to read
	// (CalcTriggers.lua L1337-1348).
	MainSkill          *ActiveSkill
	ActivationFreqInc  float64
	ActivationFreqMore float64
	AttachedBrandCount float64
	IgnoresTickRate    bool

	// GemPropertyInfo collects the matched GemProperty entries (tooltip
	// source in the reference, feeds GemItem* mods).
	GemPropertyInfo []modstore.TabEntry
	// GrantedEffectLevel is effect.grantedEffectLevel.
	GrantedEffectLevel *data.SkillLevel
}

// luaLevelsLen is Lua's #levels on the per-gem-level table: the length of
// the contiguous run from key 1 (spectre-style tables keyed from a high
// monster level have no array part, so # is 0).
func luaLevelsLen(levels map[float64]*data.SkillLevel) float64 {
	n := 0.0
	for {
		if _, ok := levels[n+1]; !ok {
			return n
		}
		n++
	}
}

// ValidateGemLevel ports calcLib.validateGemLevel (mutates gi.Level).
func ValidateGemLevel(gi *ActiveEffect) {
	grantedEffect := gi.GrantedEffect
	if grantedEffect == nil {
		grantedEffect = gi.GemData.GrantedEffect
	}
	if grantedEffect.Levels[gi.Level] == nil {
		// Try limiting to the level range of the skill
		gi.Level = math.Max(1, gi.Level)
		if n := luaLevelsLen(grantedEffect.Levels); n > 0 {
			gi.Level = math.Min(n, gi.Level)
		}
	}
	if grantedEffect.Levels[gi.Level] == nil && gi.GemData != nil {
		gi.Level = gi.GemData.NaturalMaxLevel
	}
	if grantedEffect.Levels[gi.Level] == nil {
		// That failed, so just grab any level. The reference uses next(),
		// which is hash-order arbitrary; the lowest level keeps this port
		// deterministic.
		first, found := 0.0, false
		for lvl := range grantedEffect.Levels {
			if !found || lvl < first {
				first, found = lvl, true
			}
		}
		if found {
			gi.Level = first
		}
	}
}

// DoesTypeExpressionMatch ports calcLib.doesTypeExpressionMatch: a postfix
// boolean expression over skill type ids. Nil holes in checkTypes are keys
// pairs() never sees, so they are skipped.
func DoesTypeExpressionMatch(checkTypes []any, skillTypes, minionTypes map[int64]bool) bool {
	var stack []bool
	pop := func() bool {
		if len(stack) == 0 {
			return false // t_remove on empty = nil
		}
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return v
	}
	for _, st := range checkTypes {
		id, ok := st.(int64)
		if !ok {
			continue
		}
		switch id {
		case modparser.SkillType.OR:
			other := pop()
			if len(stack) > 0 {
				stack[len(stack)-1] = stack[len(stack)-1] || other
			}
		case modparser.SkillType.AND:
			other := pop()
			if len(stack) > 0 {
				stack[len(stack)-1] = stack[len(stack)-1] && other
			}
		case modparser.SkillType.NOT:
			if len(stack) > 0 {
				stack[len(stack)-1] = !stack[len(stack)-1]
			}
		default:
			stack = append(stack, skillTypes[id] || (minionTypes != nil && minionTypes[id]))
		}
	}
	for _, v := range stack {
		if v {
			return true
		}
	}
	return false
}

// GemIsType ports calcLib.gemIsType ("all", "strength", "melee", ...).
func GemIsType(d *data.Data, gem *data.Gem, typ string, includeTransfigured bool) bool {
	lowerName := luaLower(gem.Name)
	return typ == "all" ||
		(typ == "elemental" && (gem.Tags["fire"] || gem.Tags["cold"] || gem.Tags["lightning"])) ||
		(typ == "aoe" && gem.Tags["area"]) ||
		(typ == "trap or mine" && (gem.Tags["trap"] || gem.Tags["mine"])) ||
		((typ == "active skill" || typ == "grants_active_skill" || typ == "skill") && gem.Tags["grants_active_skill"] && !gem.Tags["support"]) ||
		(typ == "non-vaal" && !gem.Tags["vaal"]) ||
		(typ == "non-exceptional" && !gem.Tags["exceptional"]) ||
		typ == lowerName ||
		typ == stripVaalPrefix(lowerName) ||
		(includeTransfigured && IsGemIdSame(d, gem.Name, typ, true)) ||
		(typ != "active skill" && typ != "grants_active_skill" && typ != "skill" && gem.Tags[typ])
}

// GetGemStatRequirement ports calcLib.getGemStatRequirement (the in-game
// formula).
func GetGemStatRequirement(level float64, isSupport bool, multi float64) float64 {
	if multi == 0 {
		return 0
	}
	statType := 0.7
	if isSupport {
		statType = 0.5
	}
	req := round((20 + (level-3)*3) * math.Pow(multi/100, 0.9) * statType)
	if req < 14 {
		return 0
	}
	return req
}

// BuildSkillInstanceStats ports calcLib.buildSkillInstanceStats.
func BuildSkillInstanceStats(d *data.Data, gi *ActiveEffect, grantedEffect *data.GrantedEffect) map[string]float64 {
	stats := map[string]float64{}
	if gi.Quality > 0 && grantedEffect.QualityStats != nil {
		for _, stat := range grantedEffect.QualityStats {
			// math.modf keeps the integral part (truncates toward zero)
			stats[stat[0].(string)] += math.Trunc(anyNum(stat[1]) * gi.Quality)
		}
	}
	level := grantedEffect.Levels[gi.Level]
	var availableEffectiveness *float64
	actorLevel := 1.0
	if gi.ActorLevel != nil {
		actorLevel = *gi.ActorLevel
	} else if level != nil {
		if req, ok := level.Extra["levelRequirement"]; ok {
			actorLevel = req
		}
	}
	for i, stat := range grantedEffect.Stats {
		index := i + 1
		// Static value used as default (assumes statInterpolation == 1)
		statValue := 1.0
		if level != nil && index <= len(level.Values) {
			statValue = level.Values[index-1]
		}
		interp := 0.0
		if level != nil && level.StatInterpolation != nil && index <= len(level.StatInterpolation) {
			interp = level.StatInterpolation[index-1]
		}
		if interp == 3 {
			// Effectiveness interpolation
			if availableEffectiveness == nil {
				base, incr := 1.0, 0.0
				if grantedEffect.BaseEffectiveness != nil {
					base = *grantedEffect.BaseEffectiveness
				}
				if grantedEffect.IncrementalEffectiveness != nil {
					incr = *grantedEffect.IncrementalEffectiveness
				}
				e := (d.GameConstants["SkillDamageBaseEffectiveness"] +
					d.GameConstants["SkillDamageIncrementalEffectiveness"]*(actorLevel-1)) *
					base * math.Pow(1+incr, actorLevel-1)
				availableEffectiveness = &e
			}
			statValue = round(*availableEffectiveness * level.Values[index-1])
		} else if interp == 2 {
			// Linear interpolation between the ordered levels
			orderedLevels := make([]float64, 0, len(grantedEffect.Levels))
			for lvl := range grantedEffect.Levels {
				orderedLevels = append(orderedLevels, lvl)
			}
			sort.Float64s(orderedLevels)
			currentLevelIndex := 0
			for idx, lvl := range orderedLevels {
				if gi.Level == lvl {
					currentLevelIndex = idx + 1
				}
			}
			if len(orderedLevels) > 1 {
				nextLevelIndex := currentLevelIndex + 1
				if nextLevelIndex > len(orderedLevels) {
					nextLevelIndex = len(orderedLevels)
				}
				nextLevel := grantedEffect.Levels[orderedLevels[nextLevelIndex-1]]
				prevLevel := grantedEffect.Levels[orderedLevels[nextLevelIndex-2]]
				nextReq := nextLevel.Extra["levelRequirement"]
				prevReq := prevLevel.Extra["levelRequirement"]
				nextStat := nextLevel.Values[index-1]
				prevStat := prevLevel.Values[index-1]
				statValue = round(prevStat + (nextStat-prevStat)*(actorLevel-prevReq)/(nextReq-prevReq))
			} else {
				statValue = round(grantedEffect.Levels[orderedLevels[currentLevelIndex-1]].Values[index-1])
			}
		}
		stats[stat] += statValue
	}
	for _, stat := range grantedEffect.ConstantStats {
		stats[stat[0].(string)] += anyNum(stat[1])
	}
	return stats
}

// GetConvertedModTags ports calcLib.getConvertedModTags: correct the tags
// on conversion with multipliers so they carry over correctly. ipairs(mod)
// walks the mod's array part (the tags) and stops at the first hole.
func GetConvertedModTags(mod *modparser.Mod, multiplier float64, minionMods bool) []any {
	var modifiers []any
	for _, v := range mod.Tags {
		if v == nil {
			break
		}
		tag, isTag := v.(modparser.Tag)
		switch {
		case isTag && minionMods && tag["type"] == "ActorCondition" && tag["actor"] == "parent":
			modifiers = append(modifiers, modparser.Tag{"type": "Condition", "var": tag["var"]})
		case isTag && truthy(tag["limitTotal"]):
			// LimitTotal can apply to 'per stat' or 'multiplier', so just
			// copy the whole and update the limit
			cp := copyTagValue(tag).(modparser.Tag)
			cp["limit"] = anyNum(cp["limit"]) * multiplier
			modifiers = append(modifiers, cp)
		default:
			modifiers = append(modifiers, copyTagValue(v))
		}
	}
	return modifiers
}

// GetGameIdFromGemName ports calcLib.getGameIdFromGemName ("" = Lua nil).
func GetGameIdFromGemName(d *data.Data, gemName string, dropVaal bool) string {
	gemId, ok := d.GemForBaseName[luaLower(gemName)]
	if !ok {
		return ""
	}
	if dropVaal && d.Gems[gemId].VaalGem {
		return d.Gems[d.GemVaalGemIdForBaseGemId[gemId]].GameId
	}
	return d.Gems[gemId].GameId
}

// IsGemIdSame ports calcLib.isGemIdSame.
func IsGemIdSame(d *data.Data, gemName, typeName string, dropVaal bool) bool {
	gemNameId := GetGameIdFromGemName(d, gemName, dropVaal)
	typeId := GetGameIdFromGemName(d, typeName, dropVaal)
	return gemNameId != "" && typeId != "" && gemNameId == typeId
}

// --- shared helpers ---

// round is Common.lua's round without the dec argument.
func round(v float64) float64 { return math.Floor(v + 0.5) }

// luaLower is Lua 5.1 string.lower under the C locale: ASCII A-Z only.
func luaLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}

// stripVaalPrefix is name:gsub("^vaal ", "") on an already-lowered name.
func stripVaalPrefix(s string) string {
	if len(s) >= 5 && s[:5] == "vaal " {
		return s[5:]
	}
	return s
}

// truthy is Lua truthiness: only nil and false are false.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	b, ok := v.(bool)
	return !ok || b
}

// anyNum reads a Lua number that Go may hold as float64 or int64.
func anyNum(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case nil:
		return 0 // `stat[2] or 0`
	}
	panic("calc: non-numeric value where the Lua holds a number")
}

// copyTagValue is Common.lua's copyTable (recursive) over the value shapes
// mod tags hold: Tag maps, array tables, D tables, and scalars.
func copyTagValue(v any) any {
	switch t := v.(type) {
	case modparser.Tag:
		out := modparser.Tag{}
		for k, e := range t {
			out[k] = copyTagValue(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = copyTagValue(e)
		}
		return out
	case *modparser.D:
		out := &modparser.D{}
		for _, e := range t.Arr {
			out.Arr = append(out.Arr, copyTagValue(e))
		}
		if t.KV != nil {
			out.KV = map[string]any{}
			for k, e := range t.KV {
				out.KV[k] = copyTagValue(e)
			}
		}
		return out
	default:
		return v
	}
}

// sortedNumKeys is the key order dump_calc's sortedPairs gives a table with
// only numeric keys: ascending. Float sums over such a table are not
// associative, so the replay has to walk them the same way.
func sortedNumKeys(m map[float64]float64) []float64 {
	keys := make([]float64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	return keys
}
