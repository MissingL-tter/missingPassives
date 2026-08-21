// Port of .archive/src/Classes/ModStore.lua — the base class of the modifier
// containers. DB (ModDB.lua) and List (ModList.lua) embed it.
//
// Fidelity notes: values follow Lua truthiness (nil/false are the only falsy
// values). Quirks preserved solely for archive parity are tagged #EVAL —
// candidates to fix once proven non-load-bearing.

package modstore

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// Store is what both containers implement: the shared public surface plus
// the internals the base methods dispatch through (the Lua class system's
// virtual calls).
type Store interface {
	base() *ModStore
	AddMod(mod *modparser.Mod)
	AddList(list []*modparser.Mod)
	ReplaceModInternal(mod *modparser.Mod) bool
	ConvertModInternal(oldName string, mod *modparser.Mod) bool
	SumInternal(ctx Store, modType string, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) float64
	MoreInternal(ctx Store, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) float64
	FlagInternal(ctx Store, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) bool
	OverrideInternal(ctx Store, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) any
	ListInternal(ctx Store, result *[]any, cfg *Cfg, flags, keywordFlags int64, source string, names ...string)
	TabulateInternal(ctx Store, result *[]TabEntry, modType string, hasType bool, cfg *Cfg, flags, keywordFlags int64, source string, names ...string)
}

// TabEntry is one Tabulate result: { value = ..., mod = ... }.
type TabEntry struct {
	Value any
	Mod   *modparser.Mod
}

// ModStore carries the shared state: parent chain, actor, and the direct
// multiplier/condition maps.
type ModStore struct {
	self        Store
	Parent      Store
	Actor       *Actor
	Multipliers map[string]float64
	Conditions  map[string]bool
}

func newModStore(self, parent Store) ModStore {
	ms := ModStore{
		self:        self,
		Parent:      parent,
		Multipliers: map[string]float64{},
		Conditions:  map[string]bool{},
	}
	if parent != nil {
		ms.Actor = parent.base().Actor
	} else {
		ms.Actor = &Actor{}
	}
	return ms
}

func (ms *ModStore) base() *ModStore { return ms }

// getActor is ModStore.lua's getActor.
func (ms *ModStore) getActor(actorType string) *Actor {
	a := ms.Actor
	if actorType == "player" {
		if a.Player != nil {
			return a.Player
		}
		if a.ParentActor != nil && a.ParentActor.Player != nil {
			return a.ParentActor.Player
		}
		if a.Enemy != nil {
			return a.Enemy.Player
		}
		return nil
	}
	return a.byType(actorType)
}

// truthy is Lua truthiness.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

func toNum(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	panic("modstore: number expected")
}

// round is Common.lua's round.
func round(val float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(val*p+0.5) / p
}

// matchKeywordFlags is Data/Global.lua's MatchKeywordFlags (uncached).
func matchKeywordFlags(keywordFlags, modKeywordFlags int64) bool {
	matchAllMask := ^modparser.KeywordFlag.MatchAll
	matchAll := modKeywordFlags&modparser.KeywordFlag.MatchAll != 0
	modMasked := modKeywordFlags & matchAllMask
	keywordMasked := keywordFlags & matchAllMask
	if matchAll {
		return keywordMasked&modMasked == modMasked
	}
	return modMasked == 0 || keywordMasked&modMasked != 0
}

// sourceMatch is the `mod.source:match("[^:]+") == source` filter.
func sourceMatch(modSource, source string) bool {
	if source == "" {
		return true
	}
	first := modSource
	for i := 0; i < len(modSource); i++ {
		if modSource[i] == ':' {
			first = modSource[:i]
			break
		}
	}
	if first == "" {
		// "[^:]+" finds the first non-empty run of non-colon characters.
		rest := modSource
		for len(rest) > 0 && rest[0] == ':' {
			rest = rest[1:]
		}
		for i := 0; i < len(rest); i++ {
			if rest[i] == ':' {
				rest = rest[:i]
				break
			}
		}
		first = rest
	}
	return first == source
}

// ScaleAddMod ports ModStore:ScaleAddMod.
func (ms *ModStore) ScaleAddMod(mod *modparser.Mod, scale float64, replace bool) {
	unscalable := false
	for _, tag := range modparser.ModTags(mod) {
		if truthy(tag["unscalable"]) {
			unscalable = true
			break
		}
	}
	if scale == 1 || unscalable {
		ms.self.AddMod(mod)
		return
	}
	scaledMod := modparser.CopyMod(mod)
	subMod := scaledMod
	if valueTable, ok := scaledMod.Value.(modparser.Tag); ok {
		if vm, ok := valueTable["mod"].(*modparser.Mod); ok {
			subMod = vm
		} else if key, ok := valueTable["keyOfScaledMod"].(string); ok {
			valueTable[key] = round(toNum(valueTable[key])*scale, 2)
		}
	}
	if v, ok := numValue(subMod.Value); ok {
		precision, has := highPrecision(subMod.Name, subMod.Type)
		if !has && math.Floor(v) != v {
			precision, has = defaultHighPrecision, true
		}
		if has {
			power := math.Pow(10, float64(precision))
			subMod.Value = math.Floor(v*scale*power) / power
		} else {
			subMod.Value = math.Trunc(round(v*scale, 2))
		}
	}
	if replace {
		ms.self.ReplaceModInternal(scaledMod)
	} else {
		ms.self.AddMod(scaledMod)
	}
}

func numValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}

func highPrecision(name, modType string) (int, bool) {
	if m, ok := highPrecisionMods[name]; ok {
		if p, ok := m[modType]; ok {
			return p, true
		}
	}
	return 0, false
}

// CopyList ports ModStore:CopyList.
func (ms *ModStore) CopyList(modList []*modparser.Mod) {
	for _, mod := range modList {
		ms.self.AddMod(modparser.CopyMod(mod))
	}
}

// ScaleAddList ports ModStore:ScaleAddList.
func (ms *ModStore) ScaleAddList(modList []*modparser.Mod, scale float64, replace bool) {
	if scale == 1 {
		ms.self.AddList(modList)
		return
	}
	for _, mod := range modList {
		ms.ScaleAddMod(mod, scale, replace)
	}
}

// ReplaceMod ports ModStore:ReplaceMod (taking a built mod where the Lua
// builds one from createMod varargs).
func (ms *ModStore) ReplaceMod(mod *modparser.Mod) {
	if !ms.self.ReplaceModInternal(mod) {
		ms.self.AddMod(mod)
	}
}

// ConvertMod ports ModStore:ConvertMod.
func (ms *ModStore) ConvertMod(oldName string, mod *modparser.Mod) {
	if !ms.self.ConvertModInternal(oldName, mod) {
		ms.self.AddMod(mod)
	}
}

func cfgParts(cfg *Cfg) (flags, keywordFlags int64, source string) {
	if cfg != nil {
		if cfg.Flags != nil {
			flags = *cfg.Flags
		}
		if cfg.KeywordFlags != nil {
			keywordFlags = *cfg.KeywordFlags
		}
		source = cfg.Source
	}
	return
}

// Combine ports ModStore:Combine. MORE/FLAG/OVERRIDE/LIST/MAX dispatch to
// their aggregations; everything else sums.
func (ms *ModStore) Combine(modType string, cfg *Cfg, names ...string) any {
	switch modType {
	case "MORE":
		return ms.More(cfg, names...)
	case "FLAG":
		return ms.Flag(cfg, names...)
	case "OVERRIDE":
		return ms.Override(cfg, names...)
	case "LIST":
		return ms.List(cfg, names...)
	case "MAX":
		v, ok := ms.Max(cfg, names...)
		if !ok {
			return nil
		}
		return v
	default:
		return ms.Sum(modType, cfg, names...)
	}
}

func (ms *ModStore) Sum(modType string, cfg *Cfg, names ...string) float64 {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.SumInternal(ms.self, modType, cfg, flags, keywordFlags, source, names...)
}

func (ms *ModStore) More(cfg *Cfg, names ...string) float64 {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.MoreInternal(ms.self, cfg, flags, keywordFlags, source, names...)
}

func (ms *ModStore) Flag(cfg *Cfg, names ...string) bool {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.FlagInternal(ms.self, cfg, flags, keywordFlags, source, names...)
}

func (ms *ModStore) Override(cfg *Cfg, names ...string) any {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.OverrideInternal(ms.self, cfg, flags, keywordFlags, source, names...)
}

func (ms *ModStore) List(cfg *Cfg, names ...string) []any {
	flags, keywordFlags, source := cfgParts(cfg)
	result := []any{}
	ms.self.ListInternal(ms.self, &result, cfg, flags, keywordFlags, source, names...)
	return result
}

func (ms *ModStore) Tabulate(modType string, cfg *Cfg, names ...string) []TabEntry {
	flags, keywordFlags, source := cfgParts(cfg)
	result := []TabEntry{}
	ms.self.TabulateInternal(ms.self, &result, modType, modType != "", cfg, flags, keywordFlags, source, names...)
	return result
}

// TabulateAll is Tabulate with a nil modType in the Lua (match every type).
func (ms *ModStore) TabulateAll(cfg *Cfg, names ...string) []TabEntry {
	flags, keywordFlags, source := cfgParts(cfg)
	result := []TabEntry{}
	ms.self.TabulateInternal(ms.self, &result, "", false, cfg, flags, keywordFlags, source, names...)
	return result
}

// Max ports ModStore:Max; ok reports whether any value was found.
// #EVAL: archive parity — `val > (max or 0)` means all-negative candidates
// never register (Max of {-5,-2} is nil, not -2).
func (ms *ModStore) Max(cfg *Cfg, names ...string) (float64, bool) {
	var max float64
	found := false
	for _, entry := range ms.Tabulate("MAX", cfg, names...) {
		val := evalMod(ms.self, entry.Mod, cfg, nil)
		v := toNum(val)
		cur := 0.0
		if found {
			cur = max
		}
		if v > cur {
			max = v
			found = true
		}
	}
	return max, found
}

// Min ports ModStore:Min.
func (ms *ModStore) Min(cfg *Cfg, names ...string) (float64, bool) {
	var min float64
	found := false
	for _, entry := range ms.Tabulate("MIN", cfg, names...) {
		val := evalMod(ms.self, entry.Mod, cfg, nil)
		v := toNum(val)
		if !found || v < min {
			min = v
			found = true
		}
	}
	return min, found
}

// HasMod ports ModStore:HasMod.
// #EVAL: archive parity — only ModDB implements HasModInternal; calling it
// on (or through) a ModList errors in the reference, so this panics.
func (ms *ModStore) HasMod(modType string, cfg *Cfg, names ...string) bool {
	flags, keywordFlags, source := cfgParts(cfg)
	db, ok := ms.self.(*DB)
	if !ok {
		panic("modstore: HasMod on a non-DB store (the Lua errors too)")
	}
	return db.HasModInternal(modType, flags, keywordFlags, source, names...)
}

// GetCondition ports ModStore:GetCondition.
func (ms *ModStore) GetCondition(varName string, cfg *Cfg) bool {
	return getCondition(ms.self, varName, cfg, false)
}

func getCondition(s Store, varName string, cfg *Cfg, noMod bool) bool {
	b := s.base()
	if b.Conditions[varName] {
		return true
	}
	if b.Parent != nil && getCondition(b.Parent, varName, cfg, true) {
		return true
	}
	if !noMod {
		return b.Flag(cfg, "Condition:"+varName)
	}
	return false
}

// GetMultiplier ports ModStore:GetMultiplier.
func (ms *ModStore) GetMultiplier(varName string, cfg *Cfg) float64 {
	return getMultiplier(ms.self, varName, cfg, false)
}

func getMultiplier(s Store, varName string, cfg *Cfg, noMod bool) float64 {
	b := s.base()
	if !noMod {
		if ov := b.Override(cfg, "Multiplier:"+varName); truthy(ov) {
			return toNum(ov)
		}
	}
	result := b.Multipliers[varName]
	if b.Parent != nil {
		result += getMultiplier(b.Parent, varName, cfg, true)
	}
	if !noMod {
		result += b.Sum("BASE", cfg, "Multiplier:"+varName)
	}
	return result
}
