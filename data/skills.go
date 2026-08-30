// data.skills and data.skillStatMap: the granted effects, from the skills
// document plus the Go-maintained template tables.

package data

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// StatMapTable exposes the Go-maintained skillStatMap without Load — the
// export pipeline builds minion mods from it.
func StatMapTable() map[string]*StatMapEntry { return skillStatMap }

// StatMapEntry is one skillStatMap / per-skill statMap entry: the mods a
// skill stat maps to, plus its scale keys (Value replaces the stat value;
// otherwise the stat value scales by Mult/Div and Base).
type StatMapEntry struct {
	Mods                   []SkillMod
	Div, Mult, Base, Value util.Opt[float64]
	SkillFlag              string // a skill flag the stat sets when non-zero
}

// SkillMod is one element of a statMap entry: a mod, a group of mods
// with its own scale, or one of the reference's typo records.
type SkillMod struct {
	Mod   *modparser.Mod
	Group *StatMapGroup
	Typo  *TypoMod
}

// StatMapGroup is a nested mod list inside a statMap entry. Source and
// Tags are written only by the statMap metatable's blind processMod pass,
// which treats the group as a mod.
type StatMapGroup struct {
	Mods      []*modparser.Mod
	Div, Mult util.Opt[float64]
	Source    string
	Tags      []modparser.Tag
}

// TypoMod is a template mod record whose helper call misplaced its
// arguments (a tag in the flags slot, a nil type, stray positional
// numbers): the reference keeps the resulting plain table, which no
// skill's stat ever reaches. Kept verbatim for the archive comparison.
type TypoMod struct {
	Name, Type   string // Type "" = the nil type slot
	Value        modparser.Value
	FlagsTag     modparser.Tag // the tag sitting in the flags slot, if any
	Flags        float64
	KeywordFlags float64
	StrayNums    []float64       // positional numbers after keywordFlags
	StrayTags    []modparser.Tag // positional tags, plus any processMod appends
	Source       string
}

// CallbackKind names the hand-written Lua callbacks a template attaches to
// a granted effect. The bodies live in calc/skillfuncs.go; a listed
// callback without a ported body panics when reached.
type CallbackKind uint8

const (
	CallbackInitial CallbackKind = iota + 1
	CallbackPreSkillType
	CallbackPreDamage
	CallbackPostCrit
	CallbackPreDot
	CallbackExplosiveArrow
)

var callbackNames = [...]string{"", "initialFunc", "preSkillTypeFunc", "preDamageFunc", "postCritFunc", "preDotFunc", "explosiveArrowFunc"}

// String is the template's key for the callback.
func (k CallbackKind) String() string { return callbackNames[k] }

// SkillCustom carries the template's hand-written keys beyond the
// generated fields.
type SkillCustom struct {
	FromItem, FromTree, Legacy, MinionHasItemSet, HideFromGemList bool
	Parts                                                         []SkillPart
	MinionList                                                    []string // nil = absent; empty = the build's spectre list
	AddMinionList                                                 []string
	AddFlags                                                      map[string]bool
	MinionUses                                                    map[string]bool
	Callbacks                                                     map[CallbackKind]bool
}

// SkillPart is one multi-part skill part: its name and the skill flags it
// sets (true) or clears (false).
type SkillPart struct {
	Name  string
	Flags map[string]bool
}

// genMod builds a mod for the converted tables.
func genMod(name string, typ modparser.ModType, value modparser.Value, flags modparser.ModFlag, kw modparser.KeywordFlag, source string, tags ...modparser.Tag) *modparser.Mod {
	m := &modparser.Mod{Name: name, Type: typ, Value: value, Flags: flags, KeywordFlags: kw}
	if source != "" {
		m.Source = source
		m.SourceSet = true
	}
	m.Tags = append(m.Tags, tags...)
	return m
}

// GrantedEffect is one data.skills entry. nil slices and maps are keys the
// reference's table lacks; empty ones are present-but-empty tables.
type GrantedEffect struct {
	Name                     string
	Id                       string
	ModSource                string
	Hidden                   bool
	Description              *string
	Color                    float64
	BaseTypeName             *string
	FlavourText              []string
	BaseEffectiveness        *float64
	IncrementalEffectiveness *float64

	Support bool
	// Support type expressions in postfix order; 0 = a type the port does
	// not know (the exporter's Unknown<n>).
	RequireSkillTypes []modparser.SkillTypeID
	AddSkillTypes     []modparser.SkillTypeID
	ExcludeSkillTypes []modparser.SkillTypeID
	IsTrigger         bool
	SupportGemsOnly   bool
	IgnoreMinionTypes bool
	PlusVersionOf     *string

	SkillTypes        map[modparser.SkillTypeID]bool
	MinionSkillTypes  map[modparser.SkillTypeID]bool
	SkillTotemId      *float64
	CastTime          *float64
	CannotBeSupported bool

	WeaponTypes          map[string]bool
	StatDescriptionScope string

	BaseFlags     map[string]bool
	BaseMods      []SkillMod       // mods, plus the one template typo record
	LevelMods     []*modparser.Mod // full-custom skills only
	QualityStats  []schema.StatValue
	ConstantStats []schema.StatValue
	Stats         []string
	NotMinionStat []string
	Levels        map[int]*SkillLevel

	StatMap map[string]*StatMapEntry
	// StatMapOwner is the reference's `statMap._grantedEffect` backref
	// (Data.lua:1039). When two skills share one statMap table the
	// backref is last-writer-wins under pairs(), so the reference picks
	// a different owner per process; both sides settle it in sorted id
	// order instead. nil means the skill owns its own statMap.
	StatMapOwner    *GrantedEffect
	HasGlobalEffect bool

	Custom SkillCustom
	// FullCustom marks a skill defined entirely by hand in the template
	// (no #skill directive): only the fields the template sets exist.
	FullCustom bool
}

// SkillLevel is one levels[n] table.
type SkillLevel struct {
	Values            []float64
	Extra             map[string]float64
	StatInterpolation []float64
	Cost              map[string]float64 // nil = absent; the templates write present-empty tables
}

// skillTemplate is one Go-maintained template fragment: a full-custom
// skill's whole definition, or the statMap, custom keys and hand-written
// overrides (Levels, Stats, SkillTypes) laid over a generated skill.
type skillTemplate struct {
	Full                                    bool
	StatMap                                 map[string]*StatMapEntry
	StatMapAlias, BaseModsAlias, PartsAlias string // tables shared with another skill
	Skill                                   GrantedEffect
}

// LevelCount is the reference's #levels: the contiguous run from level 1.
func (ge *GrantedEffect) LevelCount() int {
	n := 0
	for ge.Levels[n+1] != nil {
		n++
	}
	return n
}

// LevelData is levels[level] for a Lua number key: nil unless level is a
// whole number the table holds.
func (ge *GrantedEffect) LevelData(level float64) *SkillLevel {
	if level != math.Trunc(level) {
		return nil
	}
	return ge.Levels[int(level)]
}

// ---- clones (copyTable over the template and shared-map entries)

func cloneMods(list []*modparser.Mod) []*modparser.Mod {
	if list == nil {
		return nil
	}
	out := make([]*modparser.Mod, len(list))
	for i, m := range list {
		out[i] = m.Clone()
	}
	return out
}

func cloneTags(list []modparser.Tag) []modparser.Tag {
	if list == nil {
		return nil
	}
	out := make([]modparser.Tag, len(list))
	for i, t := range list {
		out[i] = t.Clone()
	}
	return out
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	return append(make([]T, 0, len(s)), s...)
}

// Clone deep-copies the entry.
func (e *StatMapEntry) Clone() *StatMapEntry {
	out := *e
	out.Mods = make([]SkillMod, len(e.Mods))
	for i, m := range e.Mods {
		out.Mods[i] = m.clone()
	}
	return &out
}

func cloneSkillMods(list []SkillMod) []SkillMod {
	if list == nil {
		return nil
	}
	out := make([]SkillMod, len(list))
	for i, m := range list {
		out[i] = m.clone()
	}
	return out
}

func (m SkillMod) clone() SkillMod {
	switch {
	case m.Mod != nil:
		return SkillMod{Mod: m.Mod.Clone()}
	case m.Group != nil:
		g := *m.Group
		g.Mods = cloneMods(m.Group.Mods)
		g.Tags = cloneTags(m.Group.Tags)
		return SkillMod{Group: &g}
	case m.Typo != nil:
		t := *m.Typo
		t.Value = modparser.CloneValue(m.Typo.Value)
		if m.Typo.FlagsTag != nil {
			t.FlagsTag = m.Typo.FlagsTag.Clone()
		}
		t.StrayNums = cloneSlice(m.Typo.StrayNums)
		t.StrayTags = cloneTags(m.Typo.StrayTags)
		return SkillMod{Typo: &t}
	}
	return m
}

func cloneStatMap(m map[string]*StatMapEntry) map[string]*StatMapEntry {
	if m == nil {
		return nil
	}
	out := make(map[string]*StatMapEntry, len(m))
	for k, e := range m {
		out[k] = e.Clone()
	}
	return out
}

func (l *SkillLevel) clone() *SkillLevel {
	return &SkillLevel{
		Values:            cloneSlice(l.Values),
		Extra:             cloneMap(l.Extra),
		StatInterpolation: cloneSlice(l.StatInterpolation),
		Cost:              cloneMap(l.Cost),
	}
}

func cloneLevels(m map[int]*SkillLevel) map[int]*SkillLevel {
	if m == nil {
		return nil
	}
	out := make(map[int]*SkillLevel, len(m))
	for k, l := range m {
		out[k] = l.clone()
	}
	return out
}

func (c SkillCustom) clone() SkillCustom {
	out := c
	if c.Parts != nil {
		out.Parts = make([]SkillPart, len(c.Parts))
		for i, p := range c.Parts {
			out.Parts[i] = SkillPart{Name: p.Name, Flags: cloneMap(p.Flags)}
		}
	}
	out.MinionList = cloneSlice(c.MinionList)
	out.AddMinionList = cloneSlice(c.AddMinionList)
	out.AddFlags = cloneMap(c.AddFlags)
	out.MinionUses = cloneMap(c.MinionUses)
	out.Callbacks = cloneMap(c.Callbacks)
	return out
}

// clone copies a template's hand-written skill (full-custom skills carry
// no StatMap/Id/ModSource yet).
func (ge *GrantedEffect) clone() *GrantedEffect {
	out := *ge
	out.FlavourText = cloneSlice(ge.FlavourText)
	out.RequireSkillTypes = cloneSlice(ge.RequireSkillTypes)
	out.AddSkillTypes = cloneSlice(ge.AddSkillTypes)
	out.ExcludeSkillTypes = cloneSlice(ge.ExcludeSkillTypes)
	out.SkillTypes = cloneMap(ge.SkillTypes)
	out.MinionSkillTypes = cloneMap(ge.MinionSkillTypes)
	out.WeaponTypes = cloneMap(ge.WeaponTypes)
	out.BaseFlags = cloneMap(ge.BaseFlags)
	out.BaseMods = cloneSkillMods(ge.BaseMods)
	out.LevelMods = cloneMods(ge.LevelMods)
	out.QualityStats = cloneSlice(ge.QualityStats)
	out.ConstantStats = cloneSlice(ge.ConstantStats)
	out.Stats = cloneSlice(ge.Stats)
	out.NotMinionStat = cloneSlice(ge.NotMinionStat)
	out.Levels = cloneLevels(ge.Levels)
	out.StatMap = cloneStatMap(ge.StatMap)
	out.Custom = ge.Custom.clone()
	return &out
}

// ---- load

func loadSkills(src schema.SkillsData, statMapCopies map[string][]string) error {
	Skills = map[string]*GrantedEffect{}
	for _, name := range []string{"act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int"} {
		f := src.Files[name]
		if len(f.Skills) != len(f.Tails) {
			return fmt.Errorf("data: skills template %s has unpaired #skill/#mods directives", name)
		}
		for i, hdr := range f.Skills {
			if hdr.Invalid {
				return fmt.Errorf("data: invalid skill %s (unknown granted effect)", hdr.GrantedId)
			}
			ge, err := buildGrantedEffect(hdr, f.Tails[i])
			if err != nil {
				return err
			}
			Skills[hdr.GrantedId] = ge
		}
	}
	// Skills whose whole block is hand-written.
	for id, tpl := range skillTemplates {
		if !tpl.Full {
			continue
		}
		ge := tpl.Skill.clone()
		ge.FullCustom = true
		ge.StatMap = cloneStatMap(tpl.StatMap)
		Skills[id] = ge
	}
	// Resolve cross-skill table aliases so mutation is shared, as in the
	// reference. statMapGroups collects the skills sharing each target's
	// statMap table so the backref owner can be settled deterministically.
	statMapGroups := map[string][]string{}
	for id, ge := range Skills {
		tpl := skillTemplates[id]
		if tpl == nil {
			continue
		}
		if tpl.StatMapAlias != "" {
			target := Skills[tpl.StatMapAlias]
			if target.StatMap == nil {
				target.StatMap = map[string]*StatMapEntry{}
			}
			ge.StatMap = target.StatMap
			statMapGroups[tpl.StatMapAlias] = append(statMapGroups[tpl.StatMapAlias], id)
		}
		if tpl.BaseModsAlias != "" {
			// The Lua shares the table outright (`baseMods = skills.X.baseMods`):
			// the mods' final source is the last writer's in sorted id order.
			ge.BaseMods = Skills[tpl.BaseModsAlias].BaseMods
		}
		if tpl.PartsAlias != "" {
			ge.Custom.Parts = Skills[tpl.PartsAlias].Custom.Parts
		}
	}

	// Post-process in sorted id order: shared tables' final mod sources are
	// last-writer-wins, and the archive dump makes the same deterministic
	// re-assignment (the reference's own pairs() order varies per process).
	ids := make([]string, 0, len(Skills))
	for id := range Skills {
		ids = append(ids, id)
	}
	sortStrings(ids)
	// Settle the shared-statMap backref: the reference's last pairs() writer
	// owns it, which in sorted order is the greatest member id.
	for target, members := range statMapGroups {
		owner := target
		for _, m := range members {
			if m > owner {
				owner = m
			}
		}
		for _, m := range append(members, target) {
			Skills[m].StatMapOwner = Skills[owner]
		}
	}
	for _, id := range ids {
		ge := Skills[id]
		ge.Name = sanitiseText(ge.Name)
		ge.Id = id
		ge.ModSource = "Skill:" + id
		// Add sources for skill mods, and check for global effects
		for _, m := range ge.BaseMods {
			processSkillMod(ge, m, "")
		}
		for _, m := range ge.LevelMods {
			processMod(ge, m, "")
		}
		if ge.StatMap == nil {
			ge.StatMap = map[string]*StatMapEntry{}
		}
		for name, entry := range ge.StatMap {
			for _, m := range entry.Mods {
				processSkillMod(ge, m, name)
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
				return fmt.Errorf("data: statMap copy of unknown stat %s", key)
			}
			entry := base.Clone()
			ge.StatMap[key] = entry
			for _, m := range entry.Mods {
				processModBlindInto(ge, m, key, &ge.HasGlobalEffect)
			}
		}
	}
	return nil
}

// LazyStatMapCopy builds the skillStatMapMeta.__index result for a stat
// key that is not in the skill's own statMap: a deep copy of the shared
// skillStatMap entry run through the metatable's blind processMod pass
// (Data.lua L1004-1016). Returns nil when the shared map lacks the key.
// Pure: the caller memoizes (the calc replay keeps copies per env so the
// shared data — whose canon the game-data test compares — stays pristine).
// The second result reports whether the reference's processMod would have
// stamped hasGlobalEffect onto the owner — the reference writes the shared
// skill table here (Data.lua's metatable memoizes into it); this port keeps
// the loaded data immutable and hands the flag to the caller instead.
func LazyStatMapCopy(ge *GrantedEffect, key string) (*StatMapEntry, bool) {
	base := skillStatMap[key]
	if base == nil {
		return nil, false
	}
	// The metatable passes `t._grantedEffect`, the skill that owns the
	// statMap table — not necessarily the skill being looked up, since a
	// shared table has one backref for both.
	owner := ge
	if ge.StatMapOwner != nil {
		owner = ge.StatMapOwner
	}
	setsGlobal := false
	entry := base.Clone()
	for _, m := range entry.Mods {
		processModBlindInto(owner, m, key, &setsGlobal)
	}
	return entry, setsGlobal
}

// processSkillMod is Data.lua's load-time pass over a statMap entry:
// groups are walked into their mods, typo records are stamped as tables.
func processSkillMod(ge *GrantedEffect, m SkillMod, statName string) {
	switch {
	case m.Mod != nil:
		processMod(ge, m.Mod, statName)
	case m.Group != nil:
		for _, gm := range m.Group.Mods {
			processMod(ge, gm, statName)
		}
	case m.Typo != nil:
		processTypoInto(ge, m.Typo, statName, &ge.HasGlobalEffect)
	}
}

// processModBlindInto is the statMap metatable's pass, which treats every
// element — group or typo record included — as a mod (source key, appended
// ActorCondition tag). The hasGlobalEffect sink is exposed for the
// calc-time lazy-copy path.
func processModBlindInto(ge *GrantedEffect, m SkillMod, statName string, globalFlag *bool) {
	switch {
	case m.Mod != nil:
		processModInto(ge, m.Mod, statName, globalFlag)
	case m.Group != nil:
		g := m.Group
		g.Source = ge.ModSource
		for _, tag := range g.Tags {
			if _, ok := tag.(*modparser.GlobalEffectTag); ok {
				*globalFlag = true
				break
			}
		}
		if notMinionStatApplies(ge, statName) {
			g.Tags = append(g.Tags, parentActorCond())
		}
	case m.Typo != nil:
		processTypoInto(ge, m.Typo, statName, globalFlag)
	}
}

// processMod ports Data.lua's processMod for a proper mod.
func processMod(ge *GrantedEffect, mod *modparser.Mod, statName string) {
	processModInto(ge, mod, statName, &ge.HasGlobalEffect)
}

// processModInto is processMod with the hasGlobalEffect write directed at a
// sink: load-time callers pass the granted effect's own field, the calc-time
// lazy-copy path passes a local so the shared data stays immutable after
// Load.
func processModInto(ge *GrantedEffect, mod *modparser.Mod, statName string, globalFlag *bool) {
	mod.Source = ge.ModSource
	mod.SourceSet = true
	if ref, ok := mod.Value.(modparser.ModRef); ok && ref.Mod != nil {
		ref.Mod.Source = "Skill:" + ge.Id
		ref.Mod.SourceSet = true
	}
	for _, tag := range mod.Tags {
		if _, ok := tag.(*modparser.GlobalEffectTag); ok {
			*globalFlag = true
			break
		}
	}
	if statName != "" && notMinionStatApplies(ge, statName) {
		mod.Tags = append(mod.Tags, parentActorCond())
	}
}

// processTypoInto is processMod over a typo record: the table gets a
// source key, and any appended ActorCondition tag lands in its array part.
func processTypoInto(ge *GrantedEffect, t *TypoMod, statName string, globalFlag *bool) {
	t.Source = ge.ModSource
	if ref, ok := t.Value.(modparser.ModRef); ok && ref.Mod != nil {
		ref.Mod.Source = "Skill:" + ge.Id
		ref.Mod.SourceSet = true
	}
	for _, tag := range t.StrayTags {
		if _, ok := tag.(*modparser.GlobalEffectTag); ok {
			*globalFlag = true
			break
		}
	}
	if notMinionStatApplies(ge, statName) {
		t.StrayTags = append(t.StrayTags, parentActorCond())
	}
}

// parentActorCond is the tag processMod appends to minion-excluded stats.
func parentActorCond() modparser.Tag {
	return &modparser.CondTag{IsActor: true, Actor: "parent", Neg: true}
}

func notMinionStatApplies(ge *GrantedEffect, statName string) bool {
	if len(ge.NotMinionStat) == 0 || statName == "" {
		return false
	}
	if !ge.Support && !ge.SkillTypes[modparser.SkillTypeBuff] {
		return false
	}
	for _, n := range ge.NotMinionStat {
		if n == statName {
			return true
		}
	}
	return false
}

// decodeBaseMods reads one template line's structured mods: codec mods, or
// the exporter's {"kind":"mixed"} record for a mod() call with a tag in its
// flags slot (one template line has one).
func decodeBaseMods(blob []byte) ([]SkillMod, error) {
	var list []json.RawMessage
	if err := json.Unmarshal(blob, &list); err != nil {
		return nil, fmt.Errorf("data: bad baseMods: %w", err)
	}
	out := make([]SkillMod, 0, len(list))
	for _, e := range list {
		var probe struct {
			Kind string          `json:"kind"`
			Arr  []float64       `json:"arr"`
			KV   json.RawMessage `json:"kv"`
		}
		if err := json.Unmarshal(e, &probe); err != nil || probe.Kind != "mixed" {
			out = append(out, SkillMod{Mod: modparser.DecodeMod(e)})
			continue
		}
		typo, err := decodeTypoMod(probe.KV, probe.Arr)
		if err != nil {
			return nil, err
		}
		out = append(out, SkillMod{Typo: typo})
	}
	return out, nil
}

// decodeTypoMod reads a mixed record's hash part: the mod() fields with the
// flags slot holding a tag table ({type = ..., ...}) or a number.
func decodeTypoMod(kv json.RawMessage, nums []float64) (*TypoMod, error) {
	var rec struct {
		Name         string          `json:"name"`
		Type         string          `json:"type"`
		Value        json.RawMessage `json:"value"`
		Flags        json.RawMessage `json:"flags"`
		KeywordFlags float64         `json:"keywordFlags"`
	}
	if err := json.Unmarshal(kv, &rec); err != nil {
		return nil, fmt.Errorf("data: bad mixed baseMods record: %w", err)
	}
	t := &TypoMod{Name: rec.Name, Type: rec.Type, KeywordFlags: rec.KeywordFlags, StrayNums: nums}
	// The value and the flags-slot tag decode through the mod codec, as the
	// value and first tag of a carrier mod (the tag table's "type" key is
	// the codec's "kind").
	var tagJSON json.RawMessage
	if err := json.Unmarshal(rec.Flags, &t.Flags); err != nil {
		var tag map[string]json.RawMessage
		if err := json.Unmarshal(rec.Flags, &tag); err != nil {
			return nil, fmt.Errorf("data: bad mixed baseMods flags: %w", err)
		}
		tag["kind"] = tag["type"]
		delete(tag, "type")
		tagJSON, _ = json.Marshal(tag)
	}
	carrier := map[string]any{"name": "", "type": "BASE", "flags": 0, "keywordFlags": 0, "tags": []json.RawMessage{}}
	if rec.Value != nil {
		carrier["value"] = rec.Value
	}
	if tagJSON != nil {
		carrier["tags"] = []json.RawMessage{tagJSON}
	}
	raw, _ := json.Marshal(carrier)
	decoded := modparser.DecodeMod(raw)
	t.Value = decoded.Value
	if len(decoded.Tags) > 0 {
		t.FlagsTag = decoded.Tags[0]
	}
	return t, nil
}

func buildGrantedEffect(hdr schema.SkillHeader, tail schema.SkillTail) (*GrantedEffect, error) {
	ge := &GrantedEffect{
		Name:                 hdr.Name,
		Hidden:               hdr.Hidden,
		Color:                float64(hdr.Color),
		Support:              hdr.Support,
		IsTrigger:            hdr.IsTrigger,
		SupportGemsOnly:      hdr.SupportGemsOnly,
		IgnoreMinionTypes:    hdr.IgnoreMinionTypes,
		PlusVersionOf:        hdr.PlusVersionOf,
		CannotBeSupported:    hdr.CannotBeSupported,
		StatDescriptionScope: hdr.StatDescriptionScope,
	}
	ge.Description = hdr.Description
	ge.BaseTypeName = hdr.BaseTypeName
	if hdr.HasFlavour {
		ge.FlavourText = emptyIfNil(hdr.FlavourText)
	}
	ge.BaseEffectiveness = hdr.BaseEffectiveness
	ge.IncrementalEffectiveness = hdr.IncrementalEffectiveness
	if hdr.Support {
		ge.RequireSkillTypes = emptyIfNil(hdr.RequireSkillTypes)
		ge.AddSkillTypes = emptyIfNil(hdr.AddSkillTypes)
		ge.ExcludeSkillTypes = emptyIfNil(hdr.ExcludeSkillTypes)
	} else {
		ge.SkillTypes = map[modparser.SkillTypeID]bool{}
		for _, id := range hdr.SkillTypes {
			ge.SkillTypes[id] = true
		}
		if len(hdr.MinionSkillTypes) > 0 {
			ge.MinionSkillTypes = map[modparser.SkillTypeID]bool{}
			for _, id := range hdr.MinionSkillTypes {
				ge.MinionSkillTypes[id] = true
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
		ge.BaseFlags = map[string]bool{}
		for _, f := range tail.BaseFlags {
			ge.BaseFlags[f] = true
		}
	}
	if !noArg("noBaseMods") && len(tail.BaseMods) > 0 {
		ge.BaseMods = []SkillMod{}
		for _, blob := range tail.BaseMods {
			mods, err := decodeBaseMods(blob)
			if err != nil {
				return nil, fmt.Errorf("data: skill %s: %w", hdr.GrantedId, err)
			}
			ge.BaseMods = append(ge.BaseMods, mods...)
		}
	}
	if !noArg("noQualityStats") && len(tail.QualityStats) > 0 {
		for _, s := range tail.QualityStats {
			ge.QualityStats = append(ge.QualityStats, s)
		}
	}
	if !noArg("noStats") {
		for _, s := range tail.ConstantStats {
			ge.ConstantStats = append(ge.ConstantStats, s)
		}
		ge.Stats = emptyIfNil(tail.Stats)
		ge.NotMinionStat = tail.NotMinionStat
	}
	if !noArg("noLevels") {
		ge.Levels = map[int]*SkillLevel{}
		for _, l := range tail.Levels {
			lvl := &SkillLevel{Values: l.Values}
			if lvl.Values == nil {
				lvl.Values = []float64{}
			}
			lvl.Extra = l.Extra
			for _, s := range l.Interp {
				n, ok := util.Tonumber(s)
				if !ok {
					return nil, fmt.Errorf("data: skill %s: bad statInterpolation value %s", hdr.GrantedId, s)
				}
				lvl.StatInterpolation = append(lvl.StatInterpolation, n)
			}
			if l.Cost != nil {
				lvl.Cost = map[string]float64{}
				for k, v := range l.Cost {
					lvl.Cost[k] = float64(v)
				}
			}
			ge.Levels[int(l.Level)] = lvl
		}
	}

	// The template's hand-written fragment: its statMap, custom keys, and
	// the levels/stats/skillTypes the hand-written block supplies in place
	// of the generated ones.
	if tpl := skillTemplates[hdr.GrantedId]; tpl != nil {
		ge.StatMap = cloneStatMap(tpl.StatMap)
		ge.Custom = tpl.Skill.Custom.clone()
		if tpl.Skill.Levels != nil {
			ge.Levels = cloneLevels(tpl.Skill.Levels)
		}
		if tpl.Skill.Stats != nil {
			ge.Stats = cloneSlice(tpl.Skill.Stats)
		}
		if tpl.Skill.SkillTypes != nil {
			ge.SkillTypes = cloneMap(tpl.Skill.SkillTypes)
		}
	}
	return ge, nil
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
