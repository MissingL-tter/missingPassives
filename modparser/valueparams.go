package modparser

import "github.com/MissingL-tter/missingPassives/internal/util"

// ValueKind is a record value's kind ("mod" and the rest), as the codec
// names it.
type ValueKind uint8

const (
	ValueKindMod ValueKind = iota + 1
	ValueKindPropertyMod
	ValueKindSkill
	ValueKindData
	ValueKindGemProperty
	ValueKindExplode
	ValueKindJewelFn
	ValueKindConqueredBy
	ValueKindPairs
	ValueKindSelfDamage
	ValueKindAscendancyNode
	ValueKindLinkedSupport
)

var valueKindNames = [...]string{"", "mod", "propertyMod", "skill", "data", "gemProperty", "explode",
	"jewelFn", "conqueredBy", "pairs", "selfDamage", "ascendancyNode", "linkedSupport"}

// String is the codec's kind text — the bytes the serialisers write.
func (k ValueKind) String() string { return valueKindNames[k] }

// ValueKindByName maps the codec's kind text to the kind.
var ValueKindByName = func() map[string]ValueKind {
	m := map[string]ValueKind{}
	for i := 1; i < len(valueKindNames); i++ {
		m[valueKindNames[i]] = ValueKind(i)
	}
	return m
}()

// ValueParams lists a record value's set fields under their reference key
// names, sorted, plus the record kind. Scalars (Num, Flag, Str) and nil are
// not records: ok is false. Beyond Tag.Params' value set, a param may hold
// *Mod ("mod"/"value"), Value (a DataRef's "value"), Conqueror
// ("conqueror") or JewelNodeFn ("func").
func ValueParams(v Value) (kind ValueKind, params []Param, ok bool) {
	var p paramList
	switch t := v.(type) {
	case ModRef:
		p = append(p, Param{"mod", t.Mod})
		p.flag("onlyAllies", t.OnlyAllies)
		p.flag("fromAllies", t.FromAllies)
		p.str("type", t.MinionType)
		return ValueKindMod, p.done(), true
	case PropertyModRef:
		return ValueKindPropertyMod, []Param{{"value", t.Mod}}, true
	case SkillRef:
		p.str("skillId", t.SkillID)
		p.opt("level", t.Level)
		p.opt("triggerChance", t.TriggerChance)
		p.flag("triggered", t.Triggered)
		p.flag("noSupports", t.NoSupports)
		p.flag("applyToPlayer", t.ApplyToPlayer)
		p.str("source", t.Source)
		p.strs("minionList", t.MinionList)
		return ValueKindSkill, p.done(), true
	case DataRef:
		p.str("key", t.Key)
		if t.Value != nil {
			p = append(p, Param{"value", t.Value})
		}
		p.str("merge", t.Merge)
		return ValueKindData, p.done(), true
	case GemPropertyRef:
		p.str("keyword", t.Keyword)
		p.strs("keywordList", t.KeywordList)
		p.str("key", t.Key)
		p.opt("value", t.Value)
		p.str("keyOfScaledMod", t.KeyOfScaledMod)
		return ValueKindGemProperty, p.done(), true
	case ExplodeRef:
		p.str("type", t.Type)
		p.num("chance", t.Chance)
		p.num("amount", t.Amount)
		p.str("keyOfScaledMod", t.KeyOfScaledMod)
		return ValueKindExplode, p.done(), true
	case JewelFn:
		p.str("type", t.Type)
		p.str("radius", t.Radius)
		if t.Func != nil {
			p = append(p, Param{"func", JewelFnRef{ID: t.ID, Fn: t.Func}})
		}
		return ValueKindJewelFn, p.done(), true
	case ConqueredBy:
		p.num("id", t.Seed)
		if t.Conqueror != nil {
			p = append(p, Param{"conqueror", *t.Conqueror})
		}
		return ValueKindConqueredBy, p.done(), true
	case Pairs:
		return ValueKindPairs, []Param{{"pairs", t}}, true
	case SelfDamage:
		p.opt("baseDamage", t.BaseDamage)
		p.opt("dmgMult", t.DmgMult)
		p.str("damageType", t.DamageType)
		return ValueKindSelfDamage, p.done(), true
	case AscendancyNodeRef:
		p.str("name", t.Name)
		p.str("side", t.Side)
		return ValueKindAscendancyNode, p.done(), true
	case LinkedSupportRef:
		p.str("targetSlotName", t.TargetSlotName)
		return ValueKindLinkedSupport, p.done(), true
	}
	return 0, nil, false
}

// ValueFromParams is ValueParams' inverse, taking the kind as the codec's
// text — the form the decoders read off the wire.
func ValueFromParams(kind string, params []Param) (Value, bool) {
	k, ok := ValueKindByName[kind]
	if !ok {
		return nil, false
	}
	return valueFromParams(k, params)
}

func valueFromParams(kind ValueKind, params []Param) (Value, bool) {
	r := paramReader{m: make(map[string]ParamValue, len(params))}
	for _, p := range params {
		r.m[p.Name] = p.Value
	}
	var v Value
	switch kind {
	case ValueKindMod:
		v = ModRef{Mod: r.mod("mod"), OnlyAllies: r.flag("onlyAllies"), FromAllies: r.flag("fromAllies"), MinionType: r.str("type")}
	case ValueKindPropertyMod:
		v = PropertyModRef{Mod: r.mod("value")}
	case ValueKindSkill:
		v = SkillRef{SkillID: r.str("skillId"), Level: r.opt("level"), TriggerChance: r.opt("triggerChance"), Triggered: r.flag("triggered"),
			NoSupports: r.flag("noSupports"), ApplyToPlayer: r.flag("applyToPlayer"), Source: r.str("source"), MinionList: r.strs("minionList")}
	case ValueKindData:
		d := DataRef{Key: r.str("key"), Merge: r.str("merge")}
		if inner, ok := r.take("value"); ok {
			d.Value, ok = inner.(Value)
			if !ok {
				r.bad = true
			}
		}
		v = d
	case ValueKindGemProperty:
		v = GemPropertyRef{Keyword: r.str("keyword"), KeywordList: r.strs("keywordList"), Key: r.str("key"), Value: r.opt("value"), KeyOfScaledMod: r.str("keyOfScaledMod")}
	case ValueKindExplode:
		v = ExplodeRef{Type: r.str("type"), Chance: r.num("chance"), Amount: r.num("amount"), KeyOfScaledMod: r.str("keyOfScaledMod")}
	case ValueKindJewelFn:
		j := JewelFn{Type: r.str("type"), Radius: r.str("radius")}
		if fn, ok := r.take("func"); ok {
			if ref, isRef := fn.(JewelFnRef); isRef {
				j.Func, j.ID = ref.Fn, ref.ID
			}
		}
		v = j
	case ValueKindConqueredBy:
		c := ConqueredBy{Seed: r.num("id")}
		if cq, ok := r.take("conqueror"); ok {
			if t, isConq := cq.(Conqueror); isConq {
				c.Conqueror = &t
			} else {
				r.bad = true
			}
		}
		v = c
	case ValueKindPairs:
		v = r.pairs("pairs")
	case ValueKindSelfDamage:
		v = SelfDamage{BaseDamage: r.opt("baseDamage"), DmgMult: r.opt("dmgMult"), DamageType: r.str("damageType")}
	case ValueKindAscendancyNode:
		v = AscendancyNodeRef{Name: r.str("name"), Side: r.str("side")}
	case ValueKindLinkedSupport:
		v = LinkedSupportRef{TargetSlotName: r.str("targetSlotName")}
	default:
		return nil, false
	}
	if len(r.m) > 0 || r.bad {
		return nil, false
	}
	return v, true
}

func (r *paramReader) mod(k string) *Mod {
	v, ok := r.take(k)
	if !ok {
		return nil
	}
	m, isMod := v.(*Mod)
	if !isMod {
		r.bad = true
	}
	return m
}

// opt helps the transcribed tables set optional numeric tag fields.
func opt(v float64) util.Opt[float64] { return util.Some(v) }
