// data.skills and data.skillStatMap: the granted effects, from the skills
// document plus the generated hand-fragment tables.

package data

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// StatMapEntry is one skillStatMap / per-skill statMap entry: a list of
// mods (or groups of mods) plus scaling keys (div, mult, value, skillFlag).
type StatMapEntry struct {
	Mods []any          // *modparser.Mod, []any group, or *modparser.D once mutated
	KV   map[string]any // div, mult, value, skillFlag, ...
}

// SkillCustom carries a skill's hand-written template fragment: its raw
// statMap and any custom keys (parts, minionList, flags, Unported functions).
type SkillCustom struct {
	// Full marks a skill whose entire block is hand-written passthrough
	// (no #skill directive).
	Full    bool
	StatMap map[string]*StatMapEntry
	// StatMapAlias shares another skill's statMap table (the templates
	// alias tables across skills).
	StatMapAlias string
	Keys         map[string]any
}

// SkillAlias marks a custom value that shares another skill's table.
type SkillAlias struct {
	Skill, Key string
}

// UnportedFn marks a Lua function not yet ported; the body lives in the
// Export/Skills templates and is ported by the calc modules on demand.
// `grep -rn UnportedFn data/*_gen.go` lists the outstanding ones.
type UnportedFn struct{}

// genMod builds a mod for the generated tables.
func genMod(name, typ string, value any, flags, kw int64, source string, tags ...any) *modparser.Mod {
	m := &modparser.Mod{Name: name, Type: typ, Value: value, Flags: flags, KeywordFlags: kw}
	if source != "" {
		m.Source = source
		m.SourceSet = true
	}
	m.Tags = append(m.Tags, tags...)
	return m
}

// GrantedEffect is one data.skills entry.
type GrantedEffect struct {
	Name                     string
	Id                       string
	ModSource                string
	Hidden                   bool
	Description              *string
	Color                    float64
	BaseTypeName             *string
	HasFlavour               bool
	FlavourText              []string
	BaseEffectiveness        *float64
	IncrementalEffectiveness *float64

	Support           bool
	RequireSkillTypes []any // SkillType numbers, nil holes for unknowns
	AddSkillTypes     []any
	ExcludeSkillTypes []any
	IsTrigger         bool
	SupportGemsOnly   bool
	IgnoreMinionTypes bool
	PlusVersionOf     *string

	SkillTypes        map[int64]bool
	MinionSkillTypes  map[int64]bool
	SkillTotemId      *float64
	CastTime          *float64
	CannotBeSupported bool

	WeaponTypes          map[string]bool
	StatDescriptionScope string

	HasBaseFlags  bool
	BaseFlags     map[string]bool
	BaseMods      []any
	QualityStats  [][]any
	ConstantStats [][]any
	HasStats      bool
	Stats         []string
	NotMinionStat []string
	HasLevels     bool
	Levels        map[float64]*SkillLevel

	StatMap         map[string]*StatMapEntry
	HasGlobalEffect bool

	// Custom carries the hand-written template keys (parts, minionList,
	// fromItem, Unported functions, ...).
	Custom map[string]any
	// FullCustom marks a skill defined entirely by hand in the template;
	// every field but Name/Id/ModSource/StatMap/HasGlobalEffect lives in
	// Custom.
	FullCustom bool
}

// SkillLevel is one levels[n] table.
type SkillLevel struct {
	Values            []float64
	Extra             map[string]float64
	StatInterpolation []float64
	Cost              map[string]float64
}

// resolveSkillType maps a "SkillType.X" identifier to its number, nil when
// the identifier is unknown (mapAST's Unknown<n> fallback).
func resolveSkillType(s string) any {
	if v, ok := modConstants[s]; ok {
		return v
	}
	return nil
}

func (d *Data) loadSkills(src gamedata.SkillsData, statMapCopies map[string][]string) {
	d.Skills = map[string]*GrantedEffect{}
	for _, name := range []string{"act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int"} {
		f := src.Files[name]
		if len(f.Skills) != len(f.Tails) {
			panic("data: skills template " + name + " has unpaired #skill/#mods directives")
		}
		for i, hdr := range f.Skills {
			if hdr.Invalid {
				panic("data: invalid skill " + hdr.GrantedId + " (unknown granted effect)")
			}
			d.Skills[hdr.GrantedId] = buildGrantedEffect(hdr, f.Tails[i])
		}
	}
	// Skills whose whole block is hand-written passthrough.
	for id, custom := range skillCustom {
		if !custom.Full {
			continue
		}
		ge := &GrantedEffect{FullCustom: true, Custom: deepCopyAny(custom.Keys).(map[string]any)}
		if name, ok := ge.Custom["name"].(string); ok {
			ge.Name = name
			delete(ge.Custom, "name")
		}
		if custom.StatMap != nil {
			ge.StatMap = map[string]*StatMapEntry{}
			for k, e := range custom.StatMap {
				ge.StatMap[k] = copyStatMapEntry(e)
			}
		}
		d.Skills[id] = ge
	}
	// Resolve cross-skill table aliases so mutation is shared, as in the
	// reference.
	for id, ge := range d.Skills {
		if custom := skillCustom[id]; custom != nil && custom.StatMapAlias != "" {
			target := d.Skills[custom.StatMapAlias]
			if target.StatMap == nil {
				target.StatMap = map[string]*StatMapEntry{}
			}
			ge.StatMap = target.StatMap
		}
		for k, v := range ge.Custom {
			if alias, ok := v.(SkillAlias); ok {
				target := d.Skills[alias.Skill]
				if alias.Key == "baseMods" && target.BaseMods != nil {
					ge.Custom[k] = target.BaseMods
				} else {
					ge.Custom[k] = target.Custom[alias.Key]
				}
			}
		}
	}

	// Post-process in sorted id order: shared tables' final mod sources are
	// last-writer-wins, and the archive dump makes the same deterministic
	// re-assignment (the reference's own pairs() order varies per process).
	ids := make([]string, 0, len(d.Skills))
	for id := range d.Skills {
		ids = append(ids, id)
	}
	sortStrings(ids)
	for _, id := range ids {
		ge := d.Skills[id]
		ge.Name = sanitiseText(ge.Name)
		ge.Id = id
		ge.ModSource = "Skill:" + id
		// Add sources for skill mods, and check for global effects
		processNamedOrGroup := func(m any, statName string) {
			switch t := m.(type) {
			case *modparser.Mod:
				processMod(ge, t, statName)
			case *modparser.D:
				if t.KV["name"] != nil {
					processTableMod(ge, t, statName)
					return
				}
				forEachTableValue(t, func(inner any) {
					if mod, ok := inner.(*modparser.Mod); ok {
						processMod(ge, mod, statName)
					}
				})
			case map[string]any:
				if t["name"] != nil {
					processMapMod(ge, t, statName)
					return
				}
				forEachTableValue(t, func(inner any) {
					if mod, ok := inner.(*modparser.Mod); ok {
						processMod(ge, mod, statName)
					}
				})
			default:
				forEachTableValue(m, func(inner any) {
					if mod, ok := inner.(*modparser.Mod); ok {
						processMod(ge, mod, statName)
					}
				})
			}
		}
		baseModsList := anyList(ge.BaseMods)
		if ge.FullCustom || ge.Custom["baseMods"] != nil || ge.BaseMods == nil {
			baseModsList = ge.Custom["baseMods"]
		}
		for _, list := range []any{baseModsList, ge.Custom["qualityMods"], ge.Custom["levelMods"]} {
			forEachTableValue(list, func(m any) { processNamedOrGroup(m, "") })
		}
		// Template statMap entries (group-aware processing).
		if ge.StatMap == nil {
			ge.StatMap = map[string]*StatMapEntry{}
		}
		for name, entry := range ge.StatMap {
			for _, modOrGroup := range entry.Mods {
				processNamedOrGroup(modOrGroup, name)
			}
		}
		// The boot's lazy statMap copies (replayed from the archive dump's
		// key lists): copyTable + the metatable's blind processMod pass.
		for _, key := range statMapCopies[id] {
			if _, exists := ge.StatMap[key]; exists {
				continue
			}
			base := skillStatMap[key]
			if base == nil {
				panic("data: statMap copy of unknown stat " + key)
			}
			entry := copyStatMapEntry(base)
			ge.StatMap[key] = entry
			for i, m := range entry.Mods {
				entry.Mods[i] = processModBlind(ge, m, key)
			}
		}
	}
}

// anyList adapts []any so nil stays nil for forEachTableValue.
func anyList(l []any) any {
	if l == nil {
		return nil
	}
	return l
}

// forEachTableValue walks the values of a generic list ([]any) or map.
func forEachTableValue(v any, fn func(any)) {
	switch t := v.(type) {
	case nil:
	case []any:
		for _, e := range t {
			fn(e)
		}
	case map[string]any:
		for _, e := range t {
			fn(e)
		}
	case *modparser.D:
		for _, e := range t.Arr {
			fn(e)
		}
	}
}

// processMod ports Data.lua's processMod for a proper mod.
func processMod(ge *GrantedEffect, mod *modparser.Mod, statName string) {
	mod.Source = ge.ModSource
	mod.SourceSet = true
	if vm, ok := mod.Value.(map[string]any); ok {
		if inner, ok := vm["mod"].(*modparser.Mod); ok {
			inner.Source = "Skill:" + ge.Id
			inner.SourceSet = true
		}
	}
	for _, tag := range mod.Tags {
		if tm, ok := tag.(map[string]any); ok && tm["type"] == "GlobalEffect" {
			ge.HasGlobalEffect = true
			break
		}
	}
	if statName != "" && notMinionStatApplies(ge, statName) {
		mod.Tags = append(mod.Tags, map[string]any{"type": "ActorCondition", "actor": "parent", "neg": true})
	}
}

// processTableMod is Lua processMod over a table-shaped mod (a group, or a
// mod whose flags slot holds a tag — #EVAL: archive parity for both: the
// table gets a source key, and any appended ActorCondition tag lands in its
// array part).
func processTableMod(ge *GrantedEffect, grp *modparser.D, statName string) {
	if grp.KV == nil {
		grp.KV = map[string]any{}
	}
	grp.KV["source"] = ge.ModSource
	if vm, ok := grp.KV["value"].(map[string]any); ok {
		if inner, ok := vm["mod"].(*modparser.Mod); ok {
			inner.Source = "Skill:" + ge.Id
			inner.SourceSet = true
		}
	}
	for _, e := range grp.Arr {
		if tm, ok := e.(map[string]any); ok && tm["type"] == "GlobalEffect" {
			ge.HasGlobalEffect = true
			break
		}
	}
	if notMinionStatApplies(ge, statName) {
		grp.Arr = append(grp.Arr, map[string]any{"type": "ActorCondition", "actor": "parent", "neg": true})
	}
}

// processMapMod is Lua processMod over a hash-only mod table (a mod built
// with a nil type slot stays a plain table in the generated data).
func processMapMod(ge *GrantedEffect, m map[string]any, statName string) {
	m["source"] = ge.ModSource
	if vm, ok := m["value"].(map[string]any); ok {
		if inner, ok := vm["mod"].(*modparser.Mod); ok {
			inner.Source = "Skill:" + ge.Id
			inner.SourceSet = true
		}
	}
	if notMinionStatApplies(ge, statName) {
		// t_insert appends at the (empty) array part's index 1
		for i := 1; ; i++ {
			key := itoa(i)
			if m[key] == nil {
				m[key] = map[string]any{"type": "ActorCondition", "actor": "parent", "neg": true}
				break
			}
		}
	}
}

// processModBlind is the statMap metatable's pass, which treats groups as
// mods. Returns the possibly rewrapped element.
func processModBlind(ge *GrantedEffect, m any, statName string) any {
	switch t := m.(type) {
	case *modparser.Mod:
		processMod(ge, t, statName)
		return t
	case *modparser.D:
		processTableMod(ge, t, statName)
		return t
	case []any:
		grp := &modparser.D{Arr: t}
		processTableMod(ge, grp, statName)
		return grp
	case map[string]any:
		processMapMod(ge, t, statName)
		return t
	default:
		panic("data: unexpected statMap element")
	}
}

func notMinionStatApplies(ge *GrantedEffect, statName string) bool {
	if len(ge.NotMinionStat) == 0 || statName == "" {
		return false
	}
	if !ge.Support && !ge.SkillTypes[modConstants["SkillType.Buff"]] {
		return false
	}
	for _, n := range ge.NotMinionStat {
		if n == statName {
			return true
		}
	}
	return false
}

// copyStatMapEntry is copyTable over one skillStatMap entry.
func copyStatMapEntry(e *StatMapEntry) *StatMapEntry {
	out := &StatMapEntry{}
	for _, m := range e.Mods {
		out.Mods = append(out.Mods, deepCopyAny(m))
	}
	if e.KV != nil {
		out.KV = deepCopyAny(e.KV).(map[string]any)
	}
	return out
}

func deepCopyAny(v any) any {
	switch t := v.(type) {
	case *modparser.Mod:
		m := *t
		m.Tags = nil
		for _, tag := range t.Tags {
			m.Tags = append(m.Tags, deepCopyAny(tag))
		}
		if t.Value != nil {
			m.Value = deepCopyAny(t.Value)
		}
		return &m
	case map[string]any:
		out := map[string]any{}
		for k, e := range t {
			out[k] = deepCopyAny(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = deepCopyAny(e)
		}
		return out
	case *modparser.D:
		out := &modparser.D{}
		for _, e := range t.Arr {
			out.Arr = append(out.Arr, deepCopyAny(e))
		}
		if t.KV != nil {
			out.KV = deepCopyAny(t.KV).(map[string]any)
		}
		return out
	default:
		return v
	}
}

func buildGrantedEffect(hdr gamedata.SkillHeader, tail gamedata.SkillTail) *GrantedEffect {
	ge := &GrantedEffect{
		Name:                 luaUnescape(hdr.Name),
		Hidden:               hdr.Hidden,
		Color:                float64(hdr.Color),
		Support:              hdr.Support,
		IsTrigger:            hdr.IsTrigger,
		SupportGemsOnly:      hdr.SupportGemsOnly,
		IgnoreMinionTypes:    hdr.IgnoreMinionTypes,
		PlusVersionOf:        hdr.PlusVersionOf,
		SkillTotemId:         nil,
		CannotBeSupported:    hdr.CannotBeSupported,
		StatDescriptionScope: hdr.StatDescriptionScope,
	}
	if hdr.Description != nil {
		s := luaUnescape(*hdr.Description)
		ge.Description = &s
	}
	if hdr.BaseTypeName != nil {
		s := luaUnescape(*hdr.BaseTypeName)
		ge.BaseTypeName = &s
	}
	ge.HasFlavour = hdr.HasFlavour
	if hdr.HasFlavour {
		ge.FlavourText = unescapeAll(hdr.FlavourText)
		if ge.FlavourText == nil {
			ge.FlavourText = []string{}
		}
	}
	ge.BaseEffectiveness = hdr.BaseEffectiveness
	ge.IncrementalEffectiveness = hdr.IncrementalEffectiveness
	if hdr.Support {
		types := func(list []string) []any {
			out := []any{}
			for _, s := range list {
				out = append(out, resolveSkillType(s))
			}
			return out
		}
		ge.RequireSkillTypes = types(hdr.RequireSkillTypes)
		ge.AddSkillTypes = types(hdr.AddSkillTypes)
		ge.ExcludeSkillTypes = types(hdr.ExcludeSkillTypes)
	} else {
		ge.SkillTypes = map[int64]bool{}
		for _, s := range hdr.SkillTypes {
			if v, ok := resolveSkillType(s).(int64); ok {
				ge.SkillTypes[v] = true
			} else {
				panic("data: unknown skill type " + s)
			}
		}
		if len(hdr.MinionSkillTypes) > 0 {
			ge.MinionSkillTypes = map[int64]bool{}
			for _, s := range hdr.MinionSkillTypes {
				if v, ok := resolveSkillType(s).(int64); ok {
					ge.MinionSkillTypes[v] = true
				} else {
					panic("data: unknown minion skill type " + s)
				}
			}
		}
		ge.SkillTotemId = intPtrToFloat(hdr.SkillTotemId)
		ge.CastTime = hdr.CastTime
	}
	if len(hdr.WeaponTypes) > 0 {
		ge.WeaponTypes = map[string]bool{}
		for _, t := range hdr.WeaponTypes {
			ge.WeaponTypes[t] = true
		}
	}

	args := tail.ModsArgs
	noArg := func(flag string) bool { return strings.Contains(args, flag) }
	if !noArg("noBaseFlags") && !tail.Support {
		ge.HasBaseFlags = true
		ge.BaseFlags = map[string]bool{}
		for _, f := range tail.BaseFlags {
			ge.BaseFlags[f] = true
		}
	}
	if !noArg("noBaseMods") && len(tail.BaseMods) > 0 {
		ge.BaseMods = []any{}
		for _, line := range tail.BaseMods {
			if mod, ok := evalModLine(line); ok {
				ge.BaseMods = append(ge.BaseMods, mod)
			}
		}
	}
	if !noArg("noQualityStats") && len(tail.QualityStats) > 0 {
		for _, s := range tail.QualityStats {
			ge.QualityStats = append(ge.QualityStats, []any{luaUnescape(s.Id), s.Value})
		}
	}
	if !noArg("noStats") {
		if len(tail.ConstantStats) > 0 {
			for _, s := range tail.ConstantStats {
				ge.ConstantStats = append(ge.ConstantStats, []any{luaUnescape(s.Id), s.Value})
			}
		}
		ge.HasStats = true
		ge.Stats = unescapeAll(tail.Stats)
		if ge.Stats == nil {
			ge.Stats = []string{}
		}
		ge.NotMinionStat = tail.NotMinionStat
	}
	if !noArg("noLevels") {
		ge.HasLevels = true
		ge.Levels = map[float64]*SkillLevel{}
		for _, l := range tail.Levels {
			lvl := &SkillLevel{Values: l.Values}
			if lvl.Values == nil {
				lvl.Values = []float64{}
			}
			lvl.Extra = l.Extra
			for _, s := range l.Interp {
				n, err := parseLuaNumber(s)
				if err != nil {
					panic("data: bad statInterpolation value " + s)
				}
				lvl.StatInterpolation = append(lvl.StatInterpolation, n)
			}
			if l.Cost != nil {
				lvl.Cost = map[string]float64{}
				for k, v := range l.Cost {
					lvl.Cost[k] = float64(v)
				}
			}
			ge.Levels[float64(l.Level)] = lvl
		}
	}

	if custom := skillCustom[hdr.GrantedId]; custom != nil {
		if custom.StatMap != nil {
			ge.StatMap = map[string]*StatMapEntry{}
			for k, e := range custom.StatMap {
				ge.StatMap[k] = copyStatMapEntry(e)
			}
		}
		if custom.Keys != nil {
			ge.Custom = deepCopyAny(custom.Keys).(map[string]any)
		}
	}
	return ge
}

// --- canon shadows (used by the archive-comparison test) ---

// GrantedEffectCanon builds the plain-table shape of a skill for the
// archive canon (statMap._grantedEffect is elided on both sides).
func GrantedEffectCanon(ge *GrantedEffect) map[string]any {
	if ge.FullCustom {
		m := map[string]any{
			"name":      ge.Name,
			"id":        ge.Id,
			"modSource": ge.ModSource,
			"statMap":   ge.StatMap,
		}
		if ge.HasGlobalEffect {
			m["hasGlobalEffect"] = true
		}
		for k, v := range ge.Custom {
			m[k] = v
		}
		return m
	}
	m := map[string]any{
		"name":                 ge.Name,
		"id":                   ge.Id,
		"modSource":            ge.ModSource,
		"color":                ge.Color,
		"statDescriptionScope": ge.StatDescriptionScope,
		"statMap":              ge.StatMap,
	}
	if ge.Hidden {
		m["hidden"] = true
	}
	if ge.Description != nil {
		m["description"] = *ge.Description
	}
	if ge.BaseTypeName != nil {
		m["baseTypeName"] = *ge.BaseTypeName
	}
	if ge.HasFlavour {
		m["flavourText"] = ge.FlavourText
	}
	if ge.BaseEffectiveness != nil {
		m["baseEffectiveness"] = *ge.BaseEffectiveness
	}
	if ge.IncrementalEffectiveness != nil {
		m["incrementalEffectiveness"] = *ge.IncrementalEffectiveness
	}
	if ge.Support {
		m["support"] = true
		m["requireSkillTypes"] = ge.RequireSkillTypes
		m["addSkillTypes"] = ge.AddSkillTypes
		m["excludeSkillTypes"] = ge.ExcludeSkillTypes
		if ge.IsTrigger {
			m["isTrigger"] = true
		}
		if ge.SupportGemsOnly {
			m["supportGemsOnly"] = true
		}
		if ge.IgnoreMinionTypes {
			m["ignoreMinionTypes"] = true
		}
		if ge.PlusVersionOf != nil {
			m["plusVersionOf"] = *ge.PlusVersionOf
		}
	} else {
		m["skillTypes"] = ge.SkillTypes
		if ge.MinionSkillTypes != nil {
			m["minionSkillTypes"] = ge.MinionSkillTypes
		}
		if ge.SkillTotemId != nil {
			m["skillTotemId"] = *ge.SkillTotemId
		}
		if ge.CastTime != nil {
			m["castTime"] = *ge.CastTime
		}
		if ge.CannotBeSupported {
			m["cannotBeSupported"] = true
		}
	}
	if ge.WeaponTypes != nil {
		m["weaponTypes"] = ge.WeaponTypes
	}
	if ge.HasBaseFlags {
		m["baseFlags"] = ge.BaseFlags
	}
	if len(ge.BaseMods) > 0 {
		m["baseMods"] = ge.BaseMods
	}
	if len(ge.QualityStats) > 0 {
		m["qualityStats"] = ge.QualityStats
	}
	if len(ge.ConstantStats) > 0 {
		m["constantStats"] = ge.ConstantStats
	}
	if ge.HasStats {
		m["stats"] = ge.Stats
		if len(ge.NotMinionStat) > 0 {
			m["notMinionStat"] = ge.NotMinionStat
		}
	}
	if ge.HasLevels {
		m["levels"] = ge.Levels
	}
	if ge.HasGlobalEffect {
		m["hasGlobalEffect"] = true
	}
	for k, v := range ge.Custom {
		m[k] = v
	}
	return m
}

// StatMapEntryCanon merges mods and scaling keys into one table shadow.
func StatMapEntryCanon(e *StatMapEntry) map[string]any {
	m := map[string]any{}
	for i, mod := range e.Mods {
		m[itoa(i+1)] = mod
	}
	for k, v := range e.KV {
		m[k] = v
	}
	return m
}

// SkillLevelCanon merges a level's values, extras, interpolation and cost.
func SkillLevelCanon(l *SkillLevel) map[string]any {
	m := map[string]any{}
	for i, v := range l.Values {
		m[itoa(i+1)] = v
	}
	for k, v := range l.Extra {
		m[k] = v
	}
	if len(l.StatInterpolation) > 0 {
		m["statInterpolation"] = l.StatInterpolation
	}
	if len(l.Cost) > 0 {
		m["cost"] = l.Cost
	}
	return m
}

// DCanon merges a mixed table's array and hash parts.
func DCanon(t *modparser.D) map[string]any {
	m := map[string]any{}
	for i, v := range t.Arr {
		m[itoa(i+1)] = v
	}
	for k, v := range t.KV {
		m[k] = v
	}
	return m
}

func itoa(i int) string {
	return luaIntString(float64(i))
}

// sanitiseText ports Common.lua's sanitiseText.
func sanitiseText(text string) string {
	// only bytes 128-255 or '<' trigger the replacements
	trigger := false
	for i := 0; i < len(text); i++ {
		if text[i] >= 128 || text[i] == '<' {
			trigger = true
			break
		}
	}
	if !trigger {
		return text
	}
	s := stripBalancedAngles(text)
	for _, rep := range []struct{ from, to string }{
		{"\xe2\x80\x90", "-"}, {"\xe2\x80\x91", "-"}, {"\xe2\x80\x92", "-"},
		{"\xe2\x80\x93", "-"}, {"\xe2\x80\x94", "-"}, {"\xe2\x80\x95", "-"},
		{"\xe2\x88\x92", "-"},
		{"\xc3\xa4", "a"}, {"\xc3\xb6", "o"},
		{"\x96", "-"}, {"\x97", "-"}, {"\xe4", "a"}, {"\xf6", "o"},
	} {
		s = strings.ReplaceAll(s, rep.from, rep.to)
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			b.WriteByte('?')
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// stripBalancedAngles is gsub("%b<>", "").
func stripBalancedAngles(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			depth := 1
			j := i + 1
			for ; j < len(s); j++ {
				if s[j] == '<' {
					depth++
				} else if s[j] == '>' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			if j < len(s) {
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
