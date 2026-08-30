package modparser

import (
	"fmt"
	"sort"

	"github.com/MissingL-tter/missingPassives/internal/util"
)

// TagKind is a tag's reference type ({type = "Condition"} and the rest).
type TagKind uint8

const (
	TagCondition TagKind = iota + 1
	TagActorCondition
	TagMultiplier
	TagMultiplierThreshold
	TagPerStat
	TagPercentStat
	TagStatThreshold
	TagSkillName
	TagSkillType
	TagSkillId
	TagSkillPart
	TagSocketedIn
	TagSlotName
	TagSlotNumber
	TagInSlot
	TagDisablesItem
	TagGlobalEffect
	TagItemCondition
	TagDistanceRamp
	TagMeleeProximity
	TagLimit
	TagModFlagOr
	TagKeywordFlagAnd
	TagMonsterTag
	TagBaseFlag
	TagGlobal
	TagIgnoreCond
	TagCryogenesisAddedDamage
	TagRaw
)

var tagKindNames = [...]string{"", "Condition", "ActorCondition", "Multiplier", "MultiplierThreshold",
	"PerStat", "PercentStat", "StatThreshold", "SkillName", "SkillType", "SkillId", "SkillPart",
	"SocketedIn", "SlotName", "SlotNumber", "InSlot", "DisablesItem", "GlobalEffect", "ItemCondition",
	"DistanceRamp", "MeleeProximity", "Limit", "ModFlagOr", "KeywordFlagAnd", "MonsterTag", "BaseFlag",
	"Global", "IgnoreCond", "Cryogenesis Added Damage", ""}

// String is the reference's type text ("" for a raw tag, whose type is a
// param like any other).
func (k TagKind) String() string { return tagKindNames[k] }

// TagKindByName maps the reference's type text to the kind.
var TagKindByName = func() map[string]TagKind {
	m := map[string]TagKind{}
	for i := 1; i < int(TagRaw); i++ {
		m[tagKindNames[i]] = TagKind(i)
	}
	return m
}()

// Tag is one entry of a modifier's tag array. Every kind is a pointer
// struct: the evaluator writes a computed div back into the shared tag
// (ModStore.lua EvalMod, tag.div from divVar) and later evaluations see it.
type Tag interface {
	Kind() TagKind
	// Clone copies the tag; slices are fresh.
	Clone() Tag
	// Params lists the set fields under their reference key names, sorted
	// by name — the shape formatTag and the serialisers read. Values are
	// Str, Num, Bool, StrList, NumList, SkillTypeList, SkillTypeID,
	// ModFlag, KeywordFlag or Pairs.
	Params() []Param
	// ReplaceStrings rewrites every string field (item slot templates:
	// {SlotName}, {Hand}, {OtherSlotNum}).
	ReplaceStrings(f func(string) string)
}

// ParamValue is the closed set a Param's value ranges over: the scalar
// Value kinds (Str, Num, Bool), the list kinds (StrList, NumList,
// SkillTypeList, Pairs), the integer field types (SkillTypeID, ModFlag,
// KeywordFlag), and the references ValueParams lists (*Mod, a nested
// Value, Conqueror, JewelNodeFn).
type ParamValue interface{ isParamValue() }

// StrList, NumList and SkillTypeList are the list-valued params (varList,
// statList, skillPartList, skillTypeList and the rest).
type StrList []string
type NumList []float64
type SkillTypeList []SkillTypeID

func (StrList) isParamValue()       {}
func (NumList) isParamValue()       {}
func (SkillTypeList) isParamValue() {}
func (SkillTypeID) isParamValue()   {}
func (ModFlag) isParamValue()       {}
func (KeywordFlag) isParamValue()   {}
func (JewelNodeFn) isParamValue()   {}
func (JewelFnRef) isParamValue()    {}

// JewelFnRef is the func= param of a JewelFunc value: the node function plus
// the identity FormatValue spells for it (see JewelFn.ID).
type JewelFnRef struct {
	ID string
	Fn JewelNodeFn
}

// Param is one set tag field: reference key and value.
type Param struct {
	Name  string
	Value ParamValue
}

// ParamOf classifies a value decoded from JSON or from the archive canon
// as a param. It is the codec's ingestion boundary — the one place a param
// value arrives untyped; the listings themselves are typed.
func ParamOf(v any) ParamValue {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return Str(t)
	case float64:
		return Num(t)
	case int:
		return Num(float64(t))
	case bool:
		return Bool(t)
	case []any:
		return listParamOf(t)
	case ParamValue:
		return t
	}
	panic(fmt.Sprintf("modparser: %T is not a param value", v))
}

// listParamOf types a decoded list by its elements. An empty one has no
// element type; it decodes as an empty NumList, which the readers take for
// either list kind.
func listParamOf(l []any) ParamValue {
	if len(l) == 0 {
		return NumList{}
	}
	switch l[0].(type) {
	case string:
		out := make(StrList, len(l))
		for i, e := range l {
			s, ok := e.(string)
			if !ok {
				panic(fmt.Sprintf("modparser: mixed param list element %T", e))
			}
			out[i] = s
		}
		return out
	case float64:
		out := make(NumList, len(l))
		for i, e := range l {
			n, ok := e.(float64)
			if !ok {
				panic(fmt.Sprintf("modparser: mixed param list element %T", e))
			}
			out[i] = n
		}
		return out
	case []any:
		out := make(Pairs, len(l))
		for i, e := range l {
			pair, ok := e.([]any)
			if !ok || len(pair) != 2 {
				panic(fmt.Sprintf("modparser: %T is not a param pair", e))
			}
			x, okX := pair[0].(float64)
			y, okY := pair[1].(float64)
			if !okX || !okY {
				panic("modparser: non-numeric param pair")
			}
			out[i] = [2]float64{x, y}
		}
		return out
	}
	panic(fmt.Sprintf("modparser: %T is not a param list element", l[0]))
}

// CondTag is Condition (a flag on the actor, or a skill condition) and
// ActorCondition (the same on another actor).
type CondTag struct {
	IsActor    bool // ActorCondition
	Actor      string
	Var        string
	VarList    []string
	Neg        bool
	Unscalable bool // ScaleAddMod honours unscalable on any tag; only synthetic fixtures set it here
}

// MultiplierTag is Multiplier (scale by a multiplier) and
// MultiplierThreshold (require a multiplier to reach a threshold).
type MultiplierTag struct {
	IsThreshold    bool // MultiplierThreshold
	Actor          string
	Var            string
	VarList        []string
	Div            util.Opt[float64]
	DivVar         string
	Limit          util.Opt[float64]
	LimitVar       string
	LimitStat      string
	LimitActor     string
	LimitTotal     bool
	LimitNegTotal  bool
	Base           util.Opt[float64]
	NoFloor        bool
	Invert         bool
	GlobalLimit    util.Opt[float64]
	GlobalLimitKey string
	Threshold      util.Opt[float64]
	ThresholdVar   string
	ThresholdActor string
	Upper          bool
	Equals         bool
}

// StatTag is PerStat, PercentStat and StatThreshold.
type StatTag struct {
	StatKind            TagKind // TagPerStat, TagPercentStat or TagStatThreshold
	Actor               string
	Stat                string
	StatList            []string
	Div                 util.Opt[float64]
	DivVar              string
	Percent             util.Opt[float64]
	PercentVar          string
	Floor               bool
	Limit               util.Opt[float64]
	LimitVar            string
	LimitTotal          bool
	Base                util.Opt[float64]
	GlobalLimit         util.Opt[float64]
	GlobalLimitKey      string
	Threshold           util.Opt[float64]
	ThresholdStat       string
	ThresholdPercent    util.Opt[float64]
	ThresholdPercentVar string
	Upper               bool
}

// SkillNameTag matches the active skill by name.
type SkillNameTag struct {
	SkillName           string
	SkillNameList       []string
	IncludeTransfigured bool
	SummonSkill         bool
	SkillID             string // one transcribed entry names the skill by id
	Neg                 bool
}

// SkillTypeTag matches the active skill's types. A zero SkillType is an
// absent one (the reference's nil for an unknown SkillType.X).
type SkillTypeTag struct {
	SkillType     SkillTypeID
	SkillTypeList []SkillTypeID
	Neg           bool
}

// SkillIDTag matches the active skill's granted effect id.
type SkillIDTag struct{ SkillID string }

// SkillPartTag matches the active skill part.
type SkillPartTag struct {
	Part     util.Opt[float64]
	PartList []float64
	Neg      bool
}

// SlotTag is SocketedIn, SlotName, SlotNumber, InSlot and DisablesItem:
// the item-slot family.
type SlotTag struct {
	SlotKind        TagKind
	SlotName        string
	SlotNameList    []string
	Num             float64 // SlotNumber / InSlot
	Keyword         string  // SocketedIn gem keyword
	SocketColor     string
	Sockets         []float64         // SocketedIn socket numbers
	SocketsAll      bool              // sockets = "all"
	SocketCount     util.Opt[float64] // sockets = N: fewer than N gems of the colour
	ExcludeItemType string            // DisablesItem
	Neg             bool
}

// GlobalEffectTag marks a modifier as a buff/debuff/aura/curse effect.
type GlobalEffectTag struct {
	EffectType       string
	EffectName       string
	EffectCond       string
	ModCond          string
	EffectStackVar   string
	EffectStackLimit util.Opt[float64]
	Div              util.Opt[float64] // Warcry power scaling
	Limit            util.Opt[float64]
	WarcryPowerBonus util.Opt[float64] // written by the calc at buff time
	Unscalable       bool
	ApplyMinions     bool
	ApplyNotPlayer   bool
	AllowTotemBuff   bool
}

// ItemCondTag conditions on an equipped item.
type ItemCondTag struct {
	ItemSlot      string
	SearchCond    string
	RarityCond    string
	NameCond      string
	CorruptedCond util.Opt[bool] // present-with-false matches uncorrupted items
	ShaperCond    util.Opt[bool]
	ElderCond     util.Opt[bool]
	AllSlots      bool
	BothSlots     bool
	ExcludeSelf   bool
	Neg           bool
}

// DistanceRampTag scales by skill distance over a {dist, factor} ramp.
type DistanceRampTag struct{ Ramp Pairs }

// MeleeProximityTag scales by melee proximity; Ramp[0] is the factor.
type MeleeProximityTag struct{ Ramp []float64 }

// LimitTag caps the value.
type LimitTag struct {
	Limit    util.Opt[float64]
	LimitVar string
}

// ModFlagOrTag requires any of the flags on the query.
type ModFlagOrTag struct{ ModFlags ModFlag }

// KeywordAndTag requires all of the keyword flags on the query.
type KeywordAndTag struct{ KeywordFlags KeywordFlag }

// MonsterTag matches the minion's monster tags.
type MonsterTag struct {
	Name     string
	NameList []string
	Neg      bool
}

// BaseFlagTag matches a granted effect base flag.
type BaseFlagTag struct {
	BaseFlag string
	Neg      bool
}

// MarkerTag is a tag with only a type: Global, IgnoreCond, and the
// calc-internal "Cryogenesis Added Damage" marker.
type MarkerTag struct{ Marker TagKind }

// RawTag is a tag of no known kind: one parsed back from formatted text
// (ModTools parseTags, where every param is text except the literal "true"
// and a numeric threshold), or an upstream typo naming a type the
// evaluator has no case for. Fields hold the params in Param's value set.
type RawTag struct {
	Type   string
	Fields []Param
}

func (t *CondTag) Kind() TagKind {
	if t.IsActor {
		return TagActorCondition
	}
	return TagCondition
}
func (t *MultiplierTag) Kind() TagKind {
	if t.IsThreshold {
		return TagMultiplierThreshold
	}
	return TagMultiplier
}
func (t *StatTag) Kind() TagKind           { return t.StatKind }
func (t *SkillNameTag) Kind() TagKind      { return TagSkillName }
func (t *SkillTypeTag) Kind() TagKind      { return TagSkillType }
func (t *SkillIDTag) Kind() TagKind        { return TagSkillId }
func (t *SkillPartTag) Kind() TagKind      { return TagSkillPart }
func (t *SlotTag) Kind() TagKind           { return t.SlotKind }
func (t *GlobalEffectTag) Kind() TagKind   { return TagGlobalEffect }
func (t *ItemCondTag) Kind() TagKind       { return TagItemCondition }
func (t *DistanceRampTag) Kind() TagKind   { return TagDistanceRamp }
func (t *MeleeProximityTag) Kind() TagKind { return TagMeleeProximity }
func (t *LimitTag) Kind() TagKind          { return TagLimit }
func (t *ModFlagOrTag) Kind() TagKind      { return TagModFlagOr }
func (t *KeywordAndTag) Kind() TagKind     { return TagKeywordFlagAnd }
func (t *MonsterTag) Kind() TagKind        { return TagMonsterTag }
func (t *BaseFlagTag) Kind() TagKind       { return TagBaseFlag }
func (t *MarkerTag) Kind() TagKind         { return t.Marker }
func (t *RawTag) Kind() TagKind            { return TagRaw }

func (t *CondTag) Clone() Tag {
	c := *t
	c.VarList = cloneStrings(t.VarList)
	return &c
}
func (t *MultiplierTag) Clone() Tag {
	c := *t
	c.VarList = cloneStrings(t.VarList)
	return &c
}
func (t *StatTag) Clone() Tag {
	c := *t
	c.StatList = cloneStrings(t.StatList)
	return &c
}
func (t *SkillNameTag) Clone() Tag {
	c := *t
	c.SkillNameList = cloneStrings(t.SkillNameList)
	return &c
}
func (t *SkillTypeTag) Clone() Tag {
	c := *t
	if t.SkillTypeList != nil {
		c.SkillTypeList = append([]SkillTypeID{}, t.SkillTypeList...)
	}
	return &c
}
func (t *SkillIDTag) Clone() Tag { c := *t; return &c }
func (t *SkillPartTag) Clone() Tag {
	c := *t
	c.PartList = cloneFloats(t.PartList)
	return &c
}
func (t *SlotTag) Clone() Tag {
	c := *t
	c.SlotNameList = cloneStrings(t.SlotNameList)
	c.Sockets = cloneFloats(t.Sockets)
	return &c
}
func (t *GlobalEffectTag) Clone() Tag { c := *t; return &c }
func (t *ItemCondTag) Clone() Tag     { c := *t; return &c }
func (t *DistanceRampTag) Clone() Tag {
	c := *t
	c.Ramp = append(Pairs(nil), t.Ramp...)
	return &c
}
func (t *MeleeProximityTag) Clone() Tag {
	c := *t
	c.Ramp = cloneFloats(t.Ramp)
	return &c
}
func (t *LimitTag) Clone() Tag      { c := *t; return &c }
func (t *ModFlagOrTag) Clone() Tag  { c := *t; return &c }
func (t *KeywordAndTag) Clone() Tag { c := *t; return &c }
func (t *MonsterTag) Clone() Tag {
	c := *t
	c.NameList = cloneStrings(t.NameList)
	return &c
}
func (t *BaseFlagTag) Clone() Tag { c := *t; return &c }
func (t *MarkerTag) Clone() Tag   { c := *t; return &c }
func (t *RawTag) Clone() Tag {
	c := RawTag{Type: t.Type}
	if t.Fields != nil {
		c.Fields = append([]Param{}, t.Fields...)
	}
	return &c
}

func cloneFloats(s []float64) []float64 {
	if s == nil {
		return nil
	}
	return append([]float64{}, s...)
}

// paramList accumulates set fields; zero strings, unset options, false
// booleans and nil slices are absent keys.
type paramList []Param

func (p *paramList) str(name, v string) {
	if v != "" {
		*p = append(*p, Param{name, Str(v)})
	}
}
func (p *paramList) opt(name string, v util.Opt[float64]) {
	if v.Set {
		*p = append(*p, Param{name, Num(v.V)})
	}
}
func (p *paramList) num(name string, v float64) { *p = append(*p, Param{name, Num(v)}) }
func (p *paramList) optBool(name string, v util.Opt[bool]) {
	if v.Set {
		*p = append(*p, Param{name, Bool(v.V)})
	}
}
func (p *paramList) flag(name string, v bool) {
	if v {
		*p = append(*p, Param{name, Bool(true)})
	}
}
func (p *paramList) strs(name string, v []string) {
	if v != nil {
		*p = append(*p, Param{name, StrList(v)})
	}
}
func (p *paramList) nums(name string, v []float64) {
	if v != nil {
		*p = append(*p, Param{name, NumList(v)})
	}
}
func (p *paramList) done() []Param {
	sort.Slice(*p, func(i, j int) bool { return (*p)[i].Name < (*p)[j].Name })
	return *p
}

func (t *CondTag) Params() []Param {
	var p paramList
	p.str("actor", t.Actor)
	p.str("var", t.Var)
	p.strs("varList", t.VarList)
	p.flag("neg", t.Neg)
	p.flag("unscalable", t.Unscalable)
	return p.done()
}

func (t *MultiplierTag) Params() []Param {
	var p paramList
	p.str("actor", t.Actor)
	p.str("var", t.Var)
	p.strs("varList", t.VarList)
	p.opt("div", t.Div)
	p.str("divVar", t.DivVar)
	p.opt("limit", t.Limit)
	p.str("limitVar", t.LimitVar)
	p.str("limitStat", t.LimitStat)
	p.str("limitActor", t.LimitActor)
	p.flag("limitTotal", t.LimitTotal)
	p.flag("limitNegTotal", t.LimitNegTotal)
	p.opt("base", t.Base)
	p.flag("noFloor", t.NoFloor)
	p.flag("invert", t.Invert)
	p.opt("globalLimit", t.GlobalLimit)
	p.str("globalLimitKey", t.GlobalLimitKey)
	p.opt("threshold", t.Threshold)
	p.str("thresholdVar", t.ThresholdVar)
	p.str("thresholdActor", t.ThresholdActor)
	p.flag("upper", t.Upper)
	p.flag("equals", t.Equals)
	return p.done()
}

func (t *StatTag) Params() []Param {
	var p paramList
	p.str("actor", t.Actor)
	p.str("stat", t.Stat)
	p.strs("statList", t.StatList)
	p.opt("div", t.Div)
	p.str("divVar", t.DivVar)
	p.opt("percent", t.Percent)
	p.str("percentVar", t.PercentVar)
	p.flag("floor", t.Floor)
	p.opt("limit", t.Limit)
	p.str("limitVar", t.LimitVar)
	p.flag("limitTotal", t.LimitTotal)
	p.opt("base", t.Base)
	p.opt("globalLimit", t.GlobalLimit)
	p.str("globalLimitKey", t.GlobalLimitKey)
	p.opt("threshold", t.Threshold)
	p.str("thresholdStat", t.ThresholdStat)
	p.opt("thresholdPercent", t.ThresholdPercent)
	p.str("thresholdPercentVar", t.ThresholdPercentVar)
	p.flag("upper", t.Upper)
	return p.done()
}

func (t *SkillNameTag) Params() []Param {
	var p paramList
	p.str("skillName", t.SkillName)
	p.strs("skillNameList", t.SkillNameList)
	p.flag("includeTransfigured", t.IncludeTransfigured)
	p.flag("summonSkill", t.SummonSkill)
	p.str("skillId", t.SkillID)
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *SkillTypeTag) Params() []Param {
	var p paramList
	if t.SkillType != 0 {
		p = append(p, Param{"skillType", t.SkillType})
	}
	if t.SkillTypeList != nil {
		p = append(p, Param{"skillTypeList", SkillTypeList(t.SkillTypeList)})
	}
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *SkillIDTag) Params() []Param {
	var p paramList
	p.str("skillId", t.SkillID)
	return p.done()
}

func (t *SkillPartTag) Params() []Param {
	var p paramList
	p.opt("skillPart", t.Part)
	p.nums("skillPartList", t.PartList)
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *SlotTag) Params() []Param {
	var p paramList
	p.str("slotName", t.SlotName)
	p.strs("slotNameList", t.SlotNameList)
	if t.SlotKind == TagSlotNumber || t.SlotKind == TagInSlot {
		p.num("num", t.Num)
	}
	p.str("keyword", t.Keyword)
	p.str("socketColor", t.SocketColor)
	switch {
	case t.SocketsAll:
		p.str("sockets", "all")
	case t.SocketCount.Set:
		p.num("sockets", t.SocketCount.V)
	default:
		p.nums("sockets", t.Sockets)
	}
	p.str("excludeItemType", t.ExcludeItemType)
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *GlobalEffectTag) Params() []Param {
	var p paramList
	p.str("effectType", t.EffectType)
	p.str("effectName", t.EffectName)
	p.str("effectCond", t.EffectCond)
	p.str("modCond", t.ModCond)
	p.str("effectStackVar", t.EffectStackVar)
	p.opt("effectStackLimit", t.EffectStackLimit)
	p.opt("div", t.Div)
	p.opt("limit", t.Limit)
	p.opt("warcryPowerBonus", t.WarcryPowerBonus)
	p.flag("unscalable", t.Unscalable)
	p.flag("applyMinions", t.ApplyMinions)
	p.flag("applyNotPlayer", t.ApplyNotPlayer)
	p.flag("allowTotemBuff", t.AllowTotemBuff)
	return p.done()
}

func (t *ItemCondTag) Params() []Param {
	var p paramList
	p.str("itemSlot", t.ItemSlot)
	p.str("searchCond", t.SearchCond)
	p.str("rarityCond", t.RarityCond)
	p.str("nameCond", t.NameCond)
	p.optBool("corruptedCond", t.CorruptedCond)
	p.optBool("shaperCond", t.ShaperCond)
	p.optBool("elderCond", t.ElderCond)
	p.flag("allSlots", t.AllSlots)
	p.flag("bothSlots", t.BothSlots)
	p.flag("excludeSelf", t.ExcludeSelf)
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *DistanceRampTag) Params() []Param   { return []Param{{"ramp", t.Ramp}} }
func (t *MeleeProximityTag) Params() []Param { return []Param{{"ramp", NumList(t.Ramp)}} }

func (t *LimitTag) Params() []Param {
	var p paramList
	p.opt("limit", t.Limit)
	p.str("limitVar", t.LimitVar)
	return p.done()
}

func (t *ModFlagOrTag) Params() []Param  { return []Param{{"modFlags", t.ModFlags}} }
func (t *KeywordAndTag) Params() []Param { return []Param{{"keywordFlags", t.KeywordFlags}} }

func (t *MonsterTag) Params() []Param {
	var p paramList
	p.str("monsterTag", t.Name)
	p.strs("monsterTagList", t.NameList)
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *BaseFlagTag) Params() []Param {
	var p paramList
	p.str("baseFlag", t.BaseFlag)
	p.flag("neg", t.Neg)
	return p.done()
}

func (t *MarkerTag) Params() []Param { return nil }

func (t *RawTag) Params() []Param {
	p := paramList(append([]Param{}, t.Fields...))
	return p.done()
}

func (t *CondTag) ReplaceStrings(f func(string) string) {
	t.Actor, t.Var = f(t.Actor), f(t.Var)
}
func (t *MultiplierTag) ReplaceStrings(f func(string) string) {
	t.Actor, t.Var, t.DivVar = f(t.Actor), f(t.Var), f(t.DivVar)
	t.LimitVar, t.LimitStat, t.LimitActor = f(t.LimitVar), f(t.LimitStat), f(t.LimitActor)
	t.GlobalLimitKey, t.ThresholdVar, t.ThresholdActor = f(t.GlobalLimitKey), f(t.ThresholdVar), f(t.ThresholdActor)
}
func (t *StatTag) ReplaceStrings(f func(string) string) {
	t.Actor, t.Stat, t.DivVar, t.PercentVar = f(t.Actor), f(t.Stat), f(t.DivVar), f(t.PercentVar)
	t.LimitVar, t.GlobalLimitKey = f(t.LimitVar), f(t.GlobalLimitKey)
	t.ThresholdStat, t.ThresholdPercentVar = f(t.ThresholdStat), f(t.ThresholdPercentVar)
}
func (t *SkillNameTag) ReplaceStrings(f func(string) string) {
	t.SkillName, t.SkillID = f(t.SkillName), f(t.SkillID)
}
func (t *SkillTypeTag) ReplaceStrings(func(string) string) {}
func (t *SkillIDTag) ReplaceStrings(f func(string) string) { t.SkillID = f(t.SkillID) }
func (t *SkillPartTag) ReplaceStrings(func(string) string) {}
func (t *SlotTag) ReplaceStrings(f func(string) string) {
	t.SlotName, t.Keyword, t.SocketColor, t.ExcludeItemType = f(t.SlotName), f(t.Keyword), f(t.SocketColor), f(t.ExcludeItemType)
}
func (t *GlobalEffectTag) ReplaceStrings(f func(string) string) {
	t.EffectType, t.EffectName, t.EffectCond = f(t.EffectType), f(t.EffectName), f(t.EffectCond)
	t.ModCond, t.EffectStackVar = f(t.ModCond), f(t.EffectStackVar)
}
func (t *ItemCondTag) ReplaceStrings(f func(string) string) {
	t.ItemSlot, t.SearchCond, t.RarityCond, t.NameCond = f(t.ItemSlot), f(t.SearchCond), f(t.RarityCond), f(t.NameCond)
}
func (t *DistanceRampTag) ReplaceStrings(func(string) string)   {}
func (t *MeleeProximityTag) ReplaceStrings(func(string) string) {}
func (t *LimitTag) ReplaceStrings(f func(string) string)        { t.LimitVar = f(t.LimitVar) }
func (t *ModFlagOrTag) ReplaceStrings(func(string) string)      {}
func (t *KeywordAndTag) ReplaceStrings(func(string) string)     {}
func (t *MonsterTag) ReplaceStrings(f func(string) string)      { t.Name = f(t.Name) }
func (t *BaseFlagTag) ReplaceStrings(f func(string) string)     { t.BaseFlag = f(t.BaseFlag) }
func (t *MarkerTag) ReplaceStrings(func(string) string)         {}
func (t *RawTag) ReplaceStrings(f func(string) string) {
	for i, p := range t.Fields {
		if s, ok := p.Value.(Str); ok {
			t.Fields[i].Value = Str(f(string(s)))
		}
	}
}

// TagTypeName is a tag's type text: the kind's name, or a raw tag's own.
func TagTypeName(t Tag) string {
	if r, ok := t.(*RawTag); ok {
		return r.Type
	}
	return t.Kind().String()
}

// TagFromParams builds a tag of the named kind from its reference-keyed
// params — the inverse of Params. Values follow the same closed set; a
// number may also arrive as a numeric string, which coerces as Lua
// arithmetic would. An unknown kind becomes a RawTag; an unknown key on a
// known kind fails.
func TagFromParams(kind string, params []Param) (Tag, bool) {
	k, ok := TagKindByName[kind]
	if !ok {
		return &RawTag{Type: kind, Fields: append([]Param{}, params...)}, true
	}
	r := paramReader{m: make(map[string]ParamValue, len(params))}
	for _, p := range params {
		r.m[p.Name] = p.Value
	}
	var t Tag
	switch k {
	case TagCondition, TagActorCondition:
		t = &CondTag{IsActor: k == TagActorCondition, Actor: r.str("actor"), Var: r.str("var"), VarList: r.strs("varList"), Neg: r.flag("neg"), Unscalable: r.flag("unscalable")}
	case TagMultiplier, TagMultiplierThreshold:
		t = &MultiplierTag{IsThreshold: k == TagMultiplierThreshold, Actor: r.str("actor"), Var: r.str("var"), VarList: r.strs("varList"),
			Div: r.opt("div"), DivVar: r.str("divVar"), Limit: r.opt("limit"), LimitVar: r.str("limitVar"), LimitStat: r.str("limitStat"),
			LimitActor: r.str("limitActor"), LimitTotal: r.flag("limitTotal"), LimitNegTotal: r.flag("limitNegTotal"), Base: r.opt("base"),
			NoFloor: r.flag("noFloor"), Invert: r.flag("invert"), GlobalLimit: r.opt("globalLimit"), GlobalLimitKey: r.str("globalLimitKey"),
			Threshold: r.opt("threshold"), ThresholdVar: r.str("thresholdVar"), ThresholdActor: r.str("thresholdActor"), Upper: r.flag("upper"), Equals: r.flag("equals")}
	case TagPerStat, TagPercentStat, TagStatThreshold:
		t = &StatTag{StatKind: k, Actor: r.str("actor"), Stat: r.str("stat"), StatList: r.strs("statList"), Div: r.opt("div"), DivVar: r.str("divVar"),
			Percent: r.opt("percent"), PercentVar: r.str("percentVar"), Floor: r.flag("floor"), Limit: r.opt("limit"), LimitVar: r.str("limitVar"),
			LimitTotal: r.flag("limitTotal"), Base: r.opt("base"), GlobalLimit: r.opt("globalLimit"), GlobalLimitKey: r.str("globalLimitKey"),
			Threshold: r.opt("threshold"), ThresholdStat: r.str("thresholdStat"), ThresholdPercent: r.opt("thresholdPercent"),
			ThresholdPercentVar: r.str("thresholdPercentVar"), Upper: r.flag("upper")}
	case TagSkillName:
		t = &SkillNameTag{SkillName: r.str("skillName"), SkillNameList: r.strs("skillNameList"), IncludeTransfigured: r.flag("includeTransfigured"),
			SummonSkill: r.flag("summonSkill"), SkillID: r.str("skillId"), Neg: r.flag("neg")}
	case TagSkillType:
		t = &SkillTypeTag{SkillType: SkillTypeID(r.num("skillType")), SkillTypeList: r.skillTypes("skillTypeList"), Neg: r.flag("neg")}
	case TagSkillId:
		t = &SkillIDTag{SkillID: r.str("skillId")}
	case TagSkillPart:
		t = &SkillPartTag{Part: r.opt("skillPart"), PartList: r.nums("skillPartList"), Neg: r.flag("neg")}
	case TagSocketedIn, TagSlotName, TagSlotNumber, TagInSlot, TagDisablesItem:
		st := &SlotTag{SlotKind: k, SlotName: r.str("slotName"), SlotNameList: r.strs("slotNameList"), Num: r.num("num"), Keyword: r.str("keyword"),
			SocketColor: r.str("socketColor"), ExcludeItemType: r.str("excludeItemType"), Neg: r.flag("neg")}
		switch sv := r.m["sockets"].(type) {
		case Str:
			if sv == "all" {
				st.SocketsAll = true
				delete(r.m, "sockets")
			} else {
				st.SocketCount = r.opt("sockets")
			}
		case Num:
			st.SocketCount = r.opt("sockets")
		default:
			st.Sockets = r.nums("sockets")
		}
		t = st
	case TagGlobalEffect:
		t = &GlobalEffectTag{EffectType: r.str("effectType"), EffectName: r.str("effectName"), EffectCond: r.str("effectCond"), ModCond: r.str("modCond"),
			EffectStackVar: r.str("effectStackVar"), EffectStackLimit: r.opt("effectStackLimit"), Div: r.opt("div"), Limit: r.opt("limit"),
			WarcryPowerBonus: r.opt("warcryPowerBonus"), Unscalable: r.flag("unscalable"), ApplyMinions: r.flag("applyMinions"),
			ApplyNotPlayer: r.flag("applyNotPlayer"), AllowTotemBuff: r.flag("allowTotemBuff")}
	case TagItemCondition:
		t = &ItemCondTag{ItemSlot: r.str("itemSlot"), SearchCond: r.str("searchCond"), RarityCond: r.str("rarityCond"), NameCond: r.str("nameCond"),
			CorruptedCond: r.optBool("corruptedCond"), ShaperCond: r.optBool("shaperCond"), ElderCond: r.optBool("elderCond"), AllSlots: r.flag("allSlots"),
			BothSlots: r.flag("bothSlots"), ExcludeSelf: r.flag("excludeSelf"), Neg: r.flag("neg")}
	case TagDistanceRamp:
		t = &DistanceRampTag{Ramp: r.pairs("ramp")}
	case TagMeleeProximity:
		t = &MeleeProximityTag{Ramp: r.nums("ramp")}
	case TagLimit:
		t = &LimitTag{Limit: r.opt("limit"), LimitVar: r.str("limitVar")}
	case TagModFlagOr:
		t = &ModFlagOrTag{ModFlags: ModFlag(r.num("modFlags"))}
	case TagKeywordFlagAnd:
		t = &KeywordAndTag{KeywordFlags: KeywordFlag(r.num("keywordFlags"))}
	case TagMonsterTag:
		t = &MonsterTag{Name: r.str("monsterTag"), NameList: r.strs("monsterTagList"), Neg: r.flag("neg")}
	case TagBaseFlag:
		t = &BaseFlagTag{BaseFlag: r.str("baseFlag"), Neg: r.flag("neg")}
	case TagGlobal, TagIgnoreCond, TagCryogenesisAddedDamage:
		t = &MarkerTag{Marker: k}
	}
	if len(r.m) > 0 || r.bad {
		return nil, false
	}
	return t, true
}

// paramReader consumes params by key; anything left over is an unknown key.
type paramReader struct {
	m   map[string]ParamValue
	bad bool
}

func (r *paramReader) take(k string) (ParamValue, bool) {
	v, ok := r.m[k]
	if ok {
		delete(r.m, k)
	}
	return v, ok
}

func (r *paramReader) str(k string) string {
	v, ok := r.take(k)
	if !ok {
		return ""
	}
	s, isStr := v.(Str)
	if !isStr {
		r.bad = true
	}
	return string(s)
}

func (r *paramReader) flag(k string) bool {
	v, ok := r.take(k)
	if !ok {
		return false
	}
	b, isBool := v.(Bool)
	if !isBool {
		r.bad = true
	}
	return bool(b)
}

func (r *paramReader) optBool(k string) util.Opt[bool] {
	v, ok := r.take(k)
	if !ok {
		return util.Opt[bool]{}
	}
	b, isBool := v.(Bool)
	if !isBool {
		r.bad = true
	}
	return util.Some(bool(b))
}

// numOfParam reads a param in an arithmetic context: numeric text converts,
// as Lua arithmetic would.
func numOfParam(v ParamValue) (float64, bool) {
	switch n := v.(type) {
	case Num:
		return float64(n), true
	case SkillTypeID:
		return float64(n), true
	case ModFlag:
		return float64(n), true
	case KeywordFlag:
		return float64(n), true
	case Str:
		return util.Tonumber(string(n))
	}
	return 0, false
}

func (r *paramReader) num(k string) float64 {
	v, ok := r.take(k)
	if !ok {
		return 0
	}
	n, isNum := numOfParam(v)
	if !isNum {
		r.bad = true
	}
	return n
}

func (r *paramReader) opt(k string) util.Opt[float64] {
	v, ok := r.take(k)
	if !ok {
		return util.Opt[float64]{}
	}
	n, isNum := numOfParam(v)
	if !isNum {
		r.bad = true
	}
	return util.Some(n)
}

func (r *paramReader) strs(k string) []string {
	v, ok := r.take(k)
	if !ok {
		return nil
	}
	switch l := v.(type) {
	case StrList:
		return l
	case NumList:
		if len(l) == 0 {
			return []string{} // an empty decoded list has no element type
		}
	}
	r.bad = true
	return nil
}

func (r *paramReader) nums(k string) []float64 {
	v, ok := r.take(k)
	if !ok {
		return nil
	}
	switch l := v.(type) {
	case NumList:
		return l
	case StrList:
		out := make([]float64, len(l))
		for i, e := range l {
			n, isNum := util.Tonumber(e)
			if !isNum {
				r.bad = true
			}
			out[i] = n
		}
		return out
	}
	r.bad = true
	return nil
}

func (r *paramReader) skillTypes(k string) []SkillTypeID {
	v, ok := r.take(k)
	if !ok {
		return nil
	}
	if l, isList := v.(SkillTypeList); isList {
		return l
	}
	r.m[k] = v
	nums := r.nums(k)
	if nums == nil {
		return nil
	}
	out := make([]SkillTypeID, len(nums))
	for i, n := range nums {
		out[i] = SkillTypeID(n)
	}
	return out
}

func (r *paramReader) pairs(k string) Pairs {
	v, ok := r.take(k)
	if !ok {
		return nil
	}
	switch l := v.(type) {
	case Pairs:
		return l
	case NumList:
		if len(l) == 0 {
			return Pairs{} // an empty decoded list has no element type
		}
	}
	r.bad = true
	return nil
}
