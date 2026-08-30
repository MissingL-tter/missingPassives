package modparser

import (
	"strconv"

	"github.com/MissingL-tter/missingPassives/internal/util"
)

// Value is a modifier's value: a scalar, or one of the record shapes LIST
// modifiers carry. The set is closed; every kind has a codec form and a
// reference table shape (see Params/ValueFromParams). Every value is also
// a ParamValue, since a record field may hold a nested value.
type Value interface {
	ParamValue
	isValue()
}

// Num is a numeric value (Lua number; +Inf appears in tag limits).
type Num float64

// Bool is a boolean value (FLAG modifiers, {key,value} booleans).
type Bool bool

// Str is a text value (Keystone, ClusterJewelNotable, AddToClusterJewelNode).
type Str string

// ModRef is the {mod = ...} record: a modifier granted through another
// (ExtraAura, MinionModifier, EnemyModifier, ExtraSkillMod, NodeModifier).
type ModRef struct {
	Mod        *Mod
	OnlyAllies bool
	FromAllies bool
	MinionType string // MinionModifier's minion-type filter ({type = ...})
}

// PropertyModRef is the {value = mod} record SocketProperty/GroupProperty
// carry.
type PropertyModRef struct{ Mod *Mod }

// SkillRef is the granted-skill record (ExtraSkill, ExtraSupport,
// ExtraCurse, ExtraMinionSkill).
type SkillRef struct {
	SkillID       string // "" = unknown skill (the reference's nil)
	Level         util.Opt[float64]
	TriggerChance util.Opt[float64]
	Triggered     bool
	NoSupports    bool
	ApplyToPlayer bool
	Source        string
	MinionList    []string
}

// DataRef is the {key, value} record SkillData/JewelData/ArmourData/
// FlaskData/WeaponData/ExtraSkillStat carry. Value nil means the key is
// filled later (statMap scaling).
type DataRef struct {
	Key   string
	Value Value
	Merge string // "MAX" for SkillData entries merged by maximum
}

// GemPropertyRef is the GemProperty/SupportedGemProperty record.
type GemPropertyRef struct {
	Keyword        string
	KeywordList    []string
	Key            string
	Value          util.Opt[float64]
	KeyOfScaledMod string
}

// ExplodeRef is the ExplodeMod record.
type ExplodeRef struct {
	Type           string
	Chance, Amount float64
	KeyOfScaledMod string
}

// JewelFn is the JewelFunc/ExtraJewelFunc record: a radius-jewel node
// function plus its type label.
//
// ID identifies the function for FormatValue. A Go closure has no name and
// its address is not stable across runs, so an address cannot serve the
// purpose the text exists for — modKey is a cache key (CalcsTab.lua:611
// caches one power calculation per distinct mod set). Entries built from a
// pattern with captures fold the captures in, since those decide what the
// closure does.
type JewelFn struct {
	Type   string
	Func   JewelNodeFn
	Radius string
	ID     string
}

// ConqueredBy is the JewelData conqueredBy record of a timeless/abyss jewel.
type ConqueredBy struct {
	Seed      float64
	Conqueror *Conqueror // nil when the conqueror name is unknown
}

// Conqueror is one conquerorList record: the jewel family and the
// conqueror's slot in it (an index, or "<index>_v2" for the reworked
// variant).
type Conqueror struct {
	Kind  ConquerorKind
	Index int
	V2    bool
}

// IDText is the id as the reference concatenates it: the index, or
// "<index>_v2".
func (c Conqueror) IDText() string {
	if c.V2 {
		return strconv.Itoa(c.Index) + "_v2"
	}
	return strconv.Itoa(c.Index)
}

// ConquerorKind is a timeless or abyss jewel family.
type ConquerorKind uint8

const (
	ConquerorVaal ConquerorKind = iota + 1
	ConquerorKarui
	ConquerorMaraketh
	ConquerorTemplar
	ConquerorEternal
	ConquerorKalguur
	ConquerorAbyssMurderous
	ConquerorAbyssSearching
	ConquerorAbyssHypnotic
	ConquerorAbyssGhastly
	ConquerorAbyssSpecial
)

var conquerorKindNames = [...]string{"", "vaal", "karui", "maraketh", "templar", "eternal", "kalguur",
	"abyss_murderous", "abyss_searching", "abyss_hypnotic", "abyss_ghastly", "abyss_special"}

// String is the reference's type text.
func (k ConquerorKind) String() string { return conquerorKindNames[k] }

// ConquerorKindByName maps the reference's type text to the kind.
var ConquerorKindByName = func() map[string]ConquerorKind {
	m := map[string]ConquerorKind{}
	for i := 1; i < len(conquerorKindNames); i++ {
		m[conquerorKindNames[i]] = ConquerorKind(i)
	}
	return m
}()

// Pairs is a list of {x, y} pairs (DistanceRamp ramps, DMG min/max).
type Pairs [][2]float64

// SelfDamage is the {baseDamage|dmgMult, damageType} record the unique
// self-damage modifiers carry.
type SelfDamage struct {
	BaseDamage util.Opt[float64]
	DmgMult    util.Opt[float64]
	DamageType string
}

// AscendancyNodeRef is the GrantedAscendancyNode record.
type AscendancyNodeRef struct{ Name, Side string }

// LinkedSupportRef is the LinkedSupport record.
type LinkedSupportRef struct{ TargetSlotName string }

func (Num) isValue()               {}
func (Bool) isValue()              {}
func (Str) isValue()               {}
func (ModRef) isValue()            {}
func (PropertyModRef) isValue()    {}
func (SkillRef) isValue()          {}
func (DataRef) isValue()           {}
func (GemPropertyRef) isValue()    {}
func (ExplodeRef) isValue()        {}
func (JewelFn) isValue()           {}
func (ConqueredBy) isValue()       {}
func (Pairs) isValue()             {}
func (SelfDamage) isValue()        {}
func (AscendancyNodeRef) isValue() {}
func (LinkedSupportRef) isValue()  {}

func (Num) isParamValue()               {}
func (Bool) isParamValue()              {}
func (Str) isParamValue()               {}
func (ModRef) isParamValue()            {}
func (PropertyModRef) isParamValue()    {}
func (SkillRef) isParamValue()          {}
func (DataRef) isParamValue()           {}
func (GemPropertyRef) isParamValue()    {}
func (ExplodeRef) isParamValue()        {}
func (JewelFn) isParamValue()           {}
func (ConqueredBy) isParamValue()       {}
func (Pairs) isParamValue()             {}
func (SelfDamage) isParamValue()        {}
func (AscendancyNodeRef) isParamValue() {}
func (LinkedSupportRef) isParamValue()  {}
func (Conqueror) isParamValue()         {}
func (*Mod) isParamValue()              {}

// NumOf reads a value in an arithmetic context the way Lua does: numbers
// pass, numeric text converts, anything else fails.
func NumOf(v Value) (float64, bool) {
	switch t := v.(type) {
	case Num:
		return float64(t), true
	case Str:
		return util.Tonumber(string(t))
	}
	return 0, false
}

// Truthy is Lua truthiness over a value: nil and false are the only falsy
// values.
func Truthy(v Value) bool {
	if v == nil {
		return false
	}
	if f, ok := v.(Bool); ok {
		return bool(f)
	}
	return true
}

// CloneValue deep-copies a value: nested modifiers and slices are fresh.
func CloneValue(v Value) Value {
	switch t := v.(type) {
	case ModRef:
		t.Mod = t.Mod.Clone()
		return t
	case PropertyModRef:
		t.Mod = t.Mod.Clone()
		return t
	case SkillRef:
		t.MinionList = cloneStrings(t.MinionList)
		return t
	case DataRef:
		t.Value = CloneValue(t.Value)
		return t
	case GemPropertyRef:
		t.KeywordList = cloneStrings(t.KeywordList)
		return t
	case ConqueredBy:
		if t.Conqueror != nil {
			c := *t.Conqueror
			t.Conqueror = &c
		}
		return t
	case Pairs:
		return append(Pairs(nil), t...)
	}
	return v
}

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	return append([]string{}, s...)
}

// Mod mirrors the table modLib.createMod builds: name, type, value, bit
// flags, an optional source, and the tags in its array part. Tags may hold
// nil holes (a nil among a constructor's tags); ModTags stops at the first
// one as ipairs does, len(Tags) counts them as # does.
type Mod struct {
	Name         string
	Type         ModType
	Value        Value
	Flags        ModFlag
	KeywordFlags KeywordFlag
	Source       string
	SourceSet    bool
	SourceSlot   string // mod.sourceSlot, set by Item slot mod lists ("" = absent)
	Replaced     bool   // set by ModDB ReplaceMod bookkeeping (mod.replaced)
	Converted    bool   // set by ModDB ConvertMod bookkeeping (mod.converted)
	Tags         []Tag
}

// Clone deep-copies a mod, its tags and its value, the way the reference's
// copyTable does before handing out cached or table-stored mods.
func (m *Mod) Clone() *Mod {
	if m == nil {
		return nil
	}
	cp := *m
	cp.Value = CloneValue(m.Value)
	cp.Tags = CloneTags(m.Tags)
	return &cp
}

// CloneTags deep-copies a tag list, keeping nil holes.
func CloneTags(tags []Tag) []Tag {
	if tags == nil {
		return nil
	}
	out := make([]Tag, len(tags))
	for i, t := range tags {
		if t != nil {
			out[i] = t.Clone()
		}
	}
	return out
}

// ModTags returns a mod's tags up to the first hole, as ipairs sees them.
func ModTags(m *Mod) []Tag {
	for i, t := range m.Tags {
		if t == nil {
			return m.Tags[:i]
		}
	}
	return m.Tags
}

// tagArrayLen is the ipairs length of the tag array.
func tagArrayLen(m *Mod) int { return len(ModTags(m)) }
