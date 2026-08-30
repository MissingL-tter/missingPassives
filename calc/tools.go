// Port of .archive/src/Modules/CalcTools.lua (calcLib). The functions keep
// the reference's evaluation order and Lua-truthiness edge cases, reading
// the package-level game data the way the reference reads its data global.
//
// canGrantedEffectSupportActiveSkill is deferred to the skills stage - it
// reads the full ActiveSkill shape, which is defined when
// CalcActiveSkill.lua ports.
package calc

import (
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// Mod is calcLib.mod: (1 + Sum(INC)/100) * More.
func Mod(store modstore.Store, cfg *modstore.Cfg, names ...string) float64 {
	return (1 + store.Sum(modparser.Inc, cfg, names...)/100) * store.More(cfg, names...)
}

// Mods is calcLib.mods.
func Mods(store modstore.Store, cfg *modstore.Cfg, names ...string) (inc, more float64) {
	return 1 + store.Sum(modparser.Inc, cfg, names...)/100, store.More(cfg, names...)
}

// Val is calcLib.val.
func Val(store modstore.Store, name string, cfg *modstore.Cfg) float64 {
	baseVal := store.Sum(modparser.Base, cfg, name)
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

// ValidateGemLevel ports calcLib.validateGemLevel (mutates gi.Level).
func ValidateGemLevel(gi *ActiveEffect) {
	grantedEffect := gi.GrantedEffect
	if grantedEffect == nil {
		grantedEffect = gi.GemData.GrantedEffect
	}
	if grantedEffect.LevelData(gi.Level) == nil {
		// Try limiting to the level range of the skill (#levels)
		gi.Level = math.Max(1, gi.Level)
		if n := grantedEffect.LevelCount(); n > 0 {
			gi.Level = math.Min(float64(n), gi.Level)
		}
	}
	if grantedEffect.LevelData(gi.Level) == nil && gi.GemData != nil {
		gi.Level = gi.GemData.NaturalMaxLevel
	}
	if grantedEffect.LevelData(gi.Level) == nil {
		// That failed, so just grab any level: lowest key (the reference's
		// next() is hash-order arbitrary).
		// #EVAL: for a table like {6, 7} this ignores levelRequirement;
		// shipped data only has single-entry cases.
		first, found := 0, false
		for lvl := range grantedEffect.Levels {
			if !found || lvl < first {
				first, found = lvl, true
			}
		}
		if found {
			gi.Level = float64(first)
		}
	}
}

// DoesTypeExpressionMatch ports calcLib.doesTypeExpressionMatch: a postfix
// boolean expression over skill type ids. Unknown types (0) are the nil
// holes pairs() never sees, so they are skipped.
func DoesTypeExpressionMatch(checkTypes []modparser.SkillTypeID, skillTypes, minionTypes map[modparser.SkillTypeID]bool) bool {
	var stack []bool
	pop := func() bool {
		if len(stack) == 0 {
			return false // t_remove on empty = nil
		}
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return v
	}
	for _, id := range checkTypes {
		if id == 0 {
			continue
		}
		switch id {
		case modparser.SkillTypeOR:
			other := pop()
			if len(stack) > 0 {
				stack[len(stack)-1] = stack[len(stack)-1] || other
			}
		case modparser.SkillTypeAND:
			other := pop()
			if len(stack) > 0 {
				stack[len(stack)-1] = stack[len(stack)-1] && other
			}
		case modparser.SkillTypeNOT:
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
func GemIsType(gem *data.Gem, typ string, includeTransfigured bool) bool {
	lowerName := strings.ToLower(gem.Name)
	return typ == "all" ||
		(typ == "elemental" && (gem.Tags["fire"] || gem.Tags["cold"] || gem.Tags["lightning"])) ||
		(typ == "aoe" && gem.Tags["area"]) ||
		(typ == "trap or mine" && (gem.Tags["trap"] || gem.Tags["mine"])) ||
		((typ == "active skill" || typ == "grants_active_skill" || typ == "skill") && gem.Tags["grants_active_skill"] && !gem.Tags["support"]) ||
		(typ == "non-vaal" && !gem.Tags["vaal"]) ||
		(typ == "non-exceptional" && !gem.Tags["exceptional"]) ||
		typ == lowerName ||
		typ == stripVaalPrefix(lowerName) ||
		(includeTransfigured && IsGemIdSame(gem.Name, typ, true)) ||
		(typ != "active skill" && typ != "grants_active_skill" && typ != "skill" && gem.Tags[typ])
}

// BuildSkillInstanceStats ports calcLib.buildSkillInstanceStats.
func BuildSkillInstanceStats(gi *ActiveEffect, grantedEffect *data.GrantedEffect) map[string]float64 {
	stats := map[string]float64{}
	if gi.Quality > 0 && grantedEffect.QualityStats != nil {
		for _, stat := range grantedEffect.QualityStats {
			// math.modf keeps the integral part (truncates toward zero)
			stats[stat.Id] += math.Trunc(stat.Value * gi.Quality)
		}
	}
	level := grantedEffect.LevelData(gi.Level)
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
				e := (data.GameConstants["SkillDamageBaseEffectiveness"] +
					data.GameConstants["SkillDamageIncrementalEffectiveness"]*(actorLevel-1)) *
					base * math.Pow(1+incr, actorLevel-1)
				availableEffectiveness = &e
			}
			statValue = util.RoundHalfUp(*availableEffectiveness*level.Values[index-1], 0)
		} else if interp == 2 {
			// Linear interpolation between the ordered levels
			orderedLevels := make([]int, 0, len(grantedEffect.Levels))
			for lvl := range grantedEffect.Levels {
				orderedLevels = append(orderedLevels, lvl)
			}
			sort.Ints(orderedLevels)
			currentLevelIndex := 0
			for idx, lvl := range orderedLevels {
				if gi.Level == float64(lvl) {
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
				statValue = util.RoundHalfUp(prevStat+(nextStat-prevStat)*(actorLevel-prevReq)/(nextReq-prevReq), 0)
			} else {
				statValue = util.RoundHalfUp(grantedEffect.Levels[orderedLevels[currentLevelIndex-1]].Values[index-1], 0)
			}
		}
		stats[stat] += statValue
	}
	for _, stat := range grantedEffect.ConstantStats {
		stats[stat.Id] += stat.Value
	}
	return stats
}

// GetConvertedModTags ports calcLib.getConvertedModTags: correct the tags
// on conversion with multipliers so they carry over correctly. ipairs(mod)
// walks the mod's array part (the tags) and stops at the first hole.
func GetConvertedModTags(mod *modparser.Mod, multiplier float64, minionMods bool) []modparser.Tag {
	var modifiers []modparser.Tag
	for _, v := range mod.Tags {
		if v == nil {
			break
		}
		switch tag := v.(type) {
		case *modparser.CondTag:
			if minionMods && tag.IsActor && tag.Actor == "parent" {
				modifiers = append(modifiers, &modparser.CondTag{Var: tag.Var})
				continue
			}
			modifiers = append(modifiers, tag.Clone())
		case *modparser.MultiplierTag:
			// LimitTotal can apply to 'per stat' or 'multiplier', so just
			// copy the whole and update the limit
			cp := tag.Clone().(*modparser.MultiplierTag)
			if cp.LimitTotal {
				cp.Limit = opt(cp.Limit.Or(0) * multiplier)
			}
			modifiers = append(modifiers, cp)
		case *modparser.StatTag:
			cp := tag.Clone().(*modparser.StatTag)
			if cp.LimitTotal {
				cp.Limit = opt(cp.Limit.Or(0) * multiplier)
			}
			modifiers = append(modifiers, cp)
		default:
			modifiers = append(modifiers, v.Clone())
		}
	}
	return modifiers
}

// GetGameIdFromGemName ports calcLib.getGameIdFromGemName ("" = Lua nil).
func GetGameIdFromGemName(gemName string, dropVaal bool) string {
	gemId, ok := data.GemForBaseName[strings.ToLower(gemName)]
	if !ok {
		return ""
	}
	if dropVaal && data.Gems[gemId].VaalGem {
		return data.Gems[data.GemVaalGemIdForBaseGemId[gemId]].GameId
	}
	return data.Gems[gemId].GameId
}

// IsGemIdSame ports calcLib.isGemIdSame.
func IsGemIdSame(gemName, typeName string, dropVaal bool) bool {
	gemNameId := GetGameIdFromGemName(gemName, dropVaal)
	typeId := GetGameIdFromGemName(typeName, dropVaal)
	return gemNameId != "" && typeId != "" && gemNameId == typeId
}

// --- shared helpers ---

// stripVaalPrefix is name:gsub("^vaal ", "") on an already-lowered name.
func stripVaalPrefix(s string) string {
	if len(s) >= 5 && s[:5] == "vaal " {
		return s[5:]
	}
	return s
}

// valueNum is Lua arithmetic over a mod value: numbers and numeric text;
// nil is 0 (`value or 0`). Anything else is the Lua arithmetic error.
func valueNum(v modparser.Value) float64 {
	switch n := v.(type) {
	case modparser.Num:
		return float64(n)
	case modparser.Str:
		if f, ok := modparser.NumOf(n); ok {
			return f
		}
	case nil:
		return 0
	}
	panic("calc: non-numeric value where the Lua holds a number")
}

// modRefOf reads the mod of a {mod = ...} value (nil when it is none).
func modRefOf(v modparser.Value) *modparser.Mod {
	if ref, ok := v.(modparser.ModRef); ok {
		return ref.Mod
	}
	return nil
}

// firstTag is a mod's first tag as a tag list (empty when it has none).
func firstTag(m *modparser.Mod) []modparser.Tag {
	if len(m.Tags) > 0 && m.Tags[0] != nil {
		return []modparser.Tag{m.Tags[0]}
	}
	return nil
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

// weaponData presents an item's weapon data to the weapon-condition tags;
// it is the same live record (perform writes AddedUsing into it).
type weaponData struct{ wd *item.WeaponData }

func (w weaponData) CountsAsAll1H() bool { return w.wd.CountsAsAll1H }

func (w weaponData) AddedCond(cond string) (added, present bool) {
	return w.wd.AddedUsing.Added(strings.TrimPrefix(cond, "Using"))
}

// weaponRef wraps weapon data for an actor; nil stays nil.
func weaponRef(wd *item.WeaponData) modstore.WeaponData {
	if wd == nil {
		return nil
	}
	return weaponData{wd}
}

// weaponOf recovers the weapon data behind an actor's weapon-data view.
func weaponOf(w modstore.WeaponData) *item.WeaponData {
	v, _ := w.(weaponData)
	return v.wd
}

// weaponType is weaponData.type, "" for no weapon data.
func weaponType(wd *item.WeaponData) string {
	if wd == nil {
		return ""
	}
	return wd.Type
}

// weaponCrit is weaponData.CritChance, 0 when absent.
func weaponCrit(wd *item.WeaponData) float64 {
	if wd == nil {
		return 0
	}
	return wd.CritChance.Or(0)
}

// weaponAddedDagger is the AddedUsingDagger flag (CalcOffence.lua L900).
func weaponAddedDagger(wd *item.WeaponData) bool {
	return wd != nil && wd.AddedUsing.Dagger
}

// flaskPoolTotal is flaskData.<pool>Total, 0 when the pool is absent.
func flaskPoolTotal(fd *item.FlaskData, pool string) float64 {
	if fd == nil || fd.Pool(pool) == nil {
		return 0
	}
	return fd.Pool(pool).Total
}

// dmgOf is one damage type's Min/Max/DPS, zero for no weapon data.
func dmgOf(wd *item.WeaponData, dmgType string) item.DamageRange {
	if wd == nil {
		return item.DamageRange{}
	}
	return *wd.Damage(dmgType)
}

// gemRef is cfg.skillGem: the gem the SocketedIn keyword check asks.
type gemRef struct{ gem *data.Gem }

func (g gemRef) IsType(keyword string) bool { return GemIsType(g.gem, keyword, false) }

// gemIds is the game-data resolver installed on every actor.
type gemIds struct{}

func (gemIds) GetGameIdFromGemName(name string, dropVaal bool) (string, bool) {
	id := GetGameIdFromGemName(name, dropVaal)
	return id, id != ""
}

// overrideNum is `store:Override(...) or 0` read as a number.
func overrideNum(s modstore.Store, cfg *modstore.Cfg, names ...string) float64 {
	v, _ := s.Override(cfg, names...)
	return valueNum(v)
}

// hasOverride reports an OVERRIDE match (the value's truthiness).
func hasOverride(s modstore.Store, cfg *modstore.Cfg, names ...string) bool {
	_, ok := s.Override(cfg, names...)
	return ok
}

// condClass reads a class-name condition ("" when unset or a bool).
func condClass(c modstore.Conditions, name string) string {
	class, _ := c[name].Class()
	return class
}
