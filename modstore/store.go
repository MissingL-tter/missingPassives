// Port of .archive/src/Classes/ModStore.lua — the base class of the modifier
// containers. DB (ModDB.lua) and List (ModList.lua) embed it.
//
// Fidelity notes: values follow Lua truthiness (nil/false are the only falsy
// values). Quirks preserved solely for archive parity are tagged #EVAL —
// candidates to fix once proven non-load-bearing.

package modstore

import (
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// Store is what both containers implement: the shared public surface plus
// the internals the base methods dispatch through (the Lua class system's
// virtual calls).
type Store interface {
	base() *ModStore
	// The ModStore aggregation surface, promoted from the embedded base so
	// callers (calcLib) can hold either container behind one type.
	Sum(modType modparser.ModType, cfg *Cfg, names ...string) float64
	More(cfg *Cfg, names ...string) float64
	Flag(cfg *Cfg, names ...string) bool
	Override(cfg *Cfg, names ...string) (modparser.Value, bool)
	List(cfg *Cfg, names ...string) []modparser.Value
	Tabulate(modType modparser.ModType, cfg *Cfg, names ...string) []TabEntry
	TabulateAll(cfg *Cfg, names ...string) []TabEntry
	GetCondition(varName string, cfg *Cfg) bool
	GetMultiplier(varName string, cfg *Cfg) float64
	AddMod(mod *modparser.Mod)
	AddList(list []*modparser.Mod)
	// Exported only because the mod-store differential drives these two
	// directly, mirroring dump_modstore.lua's `replace` case
	// (`if not db:ReplaceModInternal(m) then db:AddMod(m) end`).
	ReplaceModInternal(mod *modparser.Mod) bool
	ConvertModInternal(oldName string, mod *modparser.Mod) bool
	sumInternal(ctx Store, modType modparser.ModType, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) float64
	moreInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) float64
	flagInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) bool
	overrideInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) modparser.Value
	listInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) []modparser.Value
	tabulateInternal(ctx Store, modType modparser.ModType, hasType bool, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) []TabEntry
}

// TabEntry is one Tabulate result: { value = ..., mod = ... }.
type TabEntry struct {
	Value modparser.Value
	Mod   *modparser.Mod
}

// CondValue is one condition entry: a bool, or the class-name string the
// Forbidden jewels store (only ever tested for truthiness).
type CondValue struct {
	isClass bool
	on      bool
	class   string
}

func CondBool(on bool) CondValue       { return CondValue{on: on} }
func CondClass(class string) CondValue { return CondValue{isClass: true, class: class} }

// True is Lua truthiness: false only for a false bool (or an absent key).
func (c CondValue) True() bool { return c.isClass || c.on }

// Class reads the class-name string, if that is what the entry holds.
func (c CondValue) Class() (string, bool) { return c.class, c.isClass }

// Conditions is the store's direct condition table (Lua conditions).
type Conditions map[string]CondValue

func (c Conditions) Set(name string, on bool)           { c[name] = CondBool(on) }
func (c Conditions) SetClass(name string, class string) { c[name] = CondClass(class) }

// Get reads a condition's truthiness (absent is false).
func (c Conditions) Get(name string) bool { return c[name].True() }

// ModStore carries the shared state: parent chain, actor, and the direct
// multiplier/condition maps.
type ModStore struct {
	self        Store
	Parent      Store
	Actor       *Actor
	Multipliers map[string]float64
	Conditions  Conditions
}

func newModStore(self, parent Store) ModStore {
	ms := ModStore{
		self:        self,
		Parent:      parent,
		Multipliers: map[string]float64{},
		Conditions:  Conditions{},
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

// compareNum is a MAX/MIN candidate under Lua `<`/`>`: only a number
// compares; nil (a rejected re-evaluation), a bool or a string errors.
func compareNum(v modparser.Value) float64 {
	n, ok := numValue(v)
	if !ok {
		panic("modstore: MAX/MIN comparison on a non-number (the Lua errors)")
	}
	return n
}

// round is Common.lua's round.
func round(val float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(val*p+0.5) / p
}

// matchKeywordFlags is Data/Global.lua's MatchKeywordFlags (uncached).
func matchKeywordFlags(keywordFlags, modKeywordFlags modparser.KeywordFlag) bool {
	matchAllMask := ^modparser.KeywordMatchAll
	matchAll := modKeywordFlags&modparser.KeywordMatchAll != 0
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
		if tagUnscalable(tag) {
			unscalable = true
			break
		}
	}
	if scale == 1 || unscalable {
		ms.self.AddMod(mod)
		return
	}
	scaledMod := mod.Clone()
	subMod := scaledMod
	switch v := scaledMod.Value.(type) {
	case modparser.ModRef:
		subMod = v.Mod
	case modparser.ExplodeRef:
		// keyOfScaledMod names the scaled field: "chance" here.
		v.Chance = round(v.Chance*scale, 2)
		scaledMod.Value = v
	case modparser.GemPropertyRef:
		if v.KeyOfScaledMod == "value" {
			v.Value = optOf(round(v.Value.Or(0)*scale, 2))
			scaledMod.Value = v
		}
	}
	if v, ok := numValue(subMod.Value); ok {
		precision, has := highPrecision(subMod.Name, subMod.Type)
		if !has && math.Floor(v) != v {
			precision, has = defaultHighPrecision, true
		}
		if has {
			power := math.Pow(10, float64(precision))
			subMod.Value = modparser.Num(math.Floor(v*scale*power) / power)
		} else {
			subMod.Value = modparser.Num(math.Trunc(round(v*scale, 2)))
		}
	}
	if replace {
		ms.self.ReplaceModInternal(scaledMod)
	} else {
		ms.self.AddMod(scaledMod)
	}
}

// numValue reads a Num value (`type(v) == "number"`).
func numValue(v modparser.Value) (float64, bool) {
	n, ok := v.(modparser.Num)
	return float64(n), ok
}

func optOf(v float64) util.Opt[float64] { return util.Some(v) }

// tagUnscalable reads the unscalable flag ScaleAddMod checks on every tag.
func tagUnscalable(tag modparser.Tag) bool {
	switch t := tag.(type) {
	case *modparser.GlobalEffectTag:
		return t.Unscalable
	case *modparser.CondTag:
		return t.Unscalable
	}
	return false
}

func highPrecision(name string, modType modparser.ModType) (int, bool) {
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
		ms.self.AddMod(mod.Clone())
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

func cfgParts(cfg *Cfg) (flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string) {
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

func (ms *ModStore) Sum(modType modparser.ModType, cfg *Cfg, names ...string) float64 {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.sumInternal(ms.self, modType, cfg, flags, keywordFlags, source, names...)
}

func (ms *ModStore) More(cfg *Cfg, names ...string) float64 {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.moreInternal(ms.self, cfg, flags, keywordFlags, source, names...)
}

func (ms *ModStore) Flag(cfg *Cfg, names ...string) bool {
	flags, keywordFlags, source := cfgParts(cfg)
	return ms.self.flagInternal(ms.self, cfg, flags, keywordFlags, source, names...)
}

// Override returns the first truthy OVERRIDE value; ok is false when none.
func (ms *ModStore) Override(cfg *Cfg, names ...string) (modparser.Value, bool) {
	flags, keywordFlags, source := cfgParts(cfg)
	v := ms.self.overrideInternal(ms.self, cfg, flags, keywordFlags, source, names...)
	return v, v != nil
}

func (ms *ModStore) List(cfg *Cfg, names ...string) []modparser.Value {
	flags, keywordFlags, source := cfgParts(cfg)
	result := []modparser.Value{}
	return append(result, ms.self.listInternal(ms.self, cfg, flags, keywordFlags, source, names...)...)
}

func (ms *ModStore) Tabulate(modType modparser.ModType, cfg *Cfg, names ...string) []TabEntry {
	flags, keywordFlags, source := cfgParts(cfg)
	result := []TabEntry{}
	return append(result, ms.self.tabulateInternal(ms.self, modType, modType != 0, cfg, flags, keywordFlags, source, names...)...)
}

// TabulateAll is Tabulate with a nil modType in the Lua (match every type).
func (ms *ModStore) TabulateAll(cfg *Cfg, names ...string) []TabEntry {
	flags, keywordFlags, source := cfgParts(cfg)
	result := []TabEntry{}
	return append(result, ms.self.tabulateInternal(ms.self, 0, false, cfg, flags, keywordFlags, source, names...)...)
}

// Max ports ModStore:Max; ok reports whether any value was found.
// #EVAL: archive parity — `val > (max or 0)` means all-negative candidates
// never register (Max of {-5,-2} is nil, not -2).
func (ms *ModStore) Max(cfg *Cfg, names ...string) (float64, bool) {
	var max float64
	found := false
	for _, entry := range ms.Tabulate(modparser.Max, cfg, names...) {
		v := compareNum(evalMod(ms.self, entry.Mod, cfg, nil))
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
	for _, entry := range ms.Tabulate(modparser.Min, cfg, names...) {
		v := compareNum(evalMod(ms.self, entry.Mod, cfg, nil))
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
func (ms *ModStore) HasMod(modType modparser.ModType, cfg *Cfg, names ...string) bool {
	flags, keywordFlags, source := cfgParts(cfg)
	db, ok := ms.self.(*DB)
	if !ok {
		panic("modstore: HasMod on a non-DB store (the Lua errors too)")
	}
	return db.hasModInternal(modType, flags, keywordFlags, source, names...)
}

// GetCondition ports ModStore:GetCondition.
func (ms *ModStore) GetCondition(varName string, cfg *Cfg) bool {
	return getCondition(ms.self, varName, cfg, false)
}

func getCondition(s Store, varName string, cfg *Cfg, noMod bool) bool {
	b := s.base()
	if b.Conditions.Get(varName) {
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
		if ov, ok := b.Override(cfg, "Multiplier:"+varName); ok {
			return valueNum(ov)
		}
	}
	result := b.Multipliers[varName]
	if b.Parent != nil {
		result += getMultiplier(b.Parent, varName, cfg, true)
	}
	if !noMod {
		result += b.Sum(modparser.Base, cfg, "Multiplier:"+varName)
	}
	return result
}
