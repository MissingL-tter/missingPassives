package modparser

import "github.com/MissingL-tter/missingPassives/internal/util"

// Record value kinds, as the codec names them.
const (
	valueKindMod            = "mod"
	valueKindPropertyMod    = "propertyMod"
	valueKindSkill          = "skill"
	valueKindData           = "data"
	valueKindGemProperty    = "gemProperty"
	valueKindExplode        = "explode"
	valueKindJewelFn        = "jewelFn"
	valueKindConqueredBy    = "conqueredBy"
	valueKindPairs          = "pairs"
	valueKindSelfDamage     = "selfDamage"
	valueKindAscendancyNode = "ascendancyNode"
	valueKindLinkedSupport  = "linkedSupport"
)

// ValueParams lists a record value's set fields under their reference key
// names, sorted, plus the record kind. Scalars (Num, Flag, Str) and nil are
// not records: ok is false. Beyond Tag.Params' value set, a param may hold
// *Mod ("mod"/"value"), Value (a DataRef's "value"), Conqueror
// ("conqueror") or JewelNodeFn ("func").
func ValueParams(v Value) (kind string, params []Param, ok bool) {
	var p paramList
	switch t := v.(type) {
	case ModRef:
		p = append(p, Param{"mod", t.Mod})
		p.flag("onlyAllies", t.OnlyAllies)
		p.flag("fromAllies", t.FromAllies)
		p.str("type", t.MinionType)
		return valueKindMod, p.done(), true
	case PropertyModRef:
		return valueKindPropertyMod, []Param{{"value", t.Mod}}, true
	case SkillRef:
		p.str("skillId", t.SkillID)
		p.opt("level", t.Level)
		p.opt("triggerChance", t.TriggerChance)
		p.flag("triggered", t.Triggered)
		p.flag("noSupports", t.NoSupports)
		p.flag("applyToPlayer", t.ApplyToPlayer)
		p.str("source", t.Source)
		p.strs("minionList", t.MinionList)
		return valueKindSkill, p.done(), true
	case DataRef:
		p.str("key", t.Key)
		if t.Value != nil {
			p = append(p, Param{"value", t.Value})
		}
		p.str("merge", t.Merge)
		return valueKindData, p.done(), true
	case GemPropertyRef:
		p.str("keyword", t.Keyword)
		p.strs("keywordList", t.KeywordList)
		p.str("key", t.Key)
		p.opt("value", t.Value)
		p.str("keyOfScaledMod", t.KeyOfScaledMod)
		return valueKindGemProperty, p.done(), true
	case ExplodeRef:
		p.str("type", t.Type)
		p.num("chance", t.Chance)
		p.num("amount", t.Amount)
		p.str("keyOfScaledMod", t.KeyOfScaledMod)
		return valueKindExplode, p.done(), true
	case JewelFn:
		p.str("type", t.Type)
		p.str("radius", t.Radius)
		if t.Func != nil {
			p = append(p, Param{"func", JewelFnRef{ID: t.ID, Fn: t.Func}})
		}
		return valueKindJewelFn, p.done(), true
	case ConqueredBy:
		p.num("id", t.Seed)
		if t.Conqueror != nil {
			p = append(p, Param{"conqueror", *t.Conqueror})
		}
		return valueKindConqueredBy, p.done(), true
	case Pairs:
		return valueKindPairs, []Param{{"pairs", t}}, true
	case SelfDamage:
		p.opt("baseDamage", t.BaseDamage)
		p.opt("dmgMult", t.DmgMult)
		p.str("damageType", t.DamageType)
		return valueKindSelfDamage, p.done(), true
	case AscendancyNodeRef:
		p.str("name", t.Name)
		p.str("side", t.Side)
		return valueKindAscendancyNode, p.done(), true
	case LinkedSupportRef:
		p.str("targetSlotName", t.TargetSlotName)
		return valueKindLinkedSupport, p.done(), true
	}
	return "", nil, false
}

// ValueFromParams is ValueParams' inverse.
func ValueFromParams(kind string, params []Param) (Value, bool) {
	r := paramReader{m: make(map[string]ParamValue, len(params))}
	for _, p := range params {
		r.m[p.Name] = p.Value
	}
	var v Value
	switch kind {
	case valueKindMod:
		v = ModRef{Mod: r.mod("mod"), OnlyAllies: r.flag("onlyAllies"), FromAllies: r.flag("fromAllies"), MinionType: r.str("type")}
	case valueKindPropertyMod:
		v = PropertyModRef{Mod: r.mod("value")}
	case valueKindSkill:
		v = SkillRef{SkillID: r.str("skillId"), Level: r.opt("level"), TriggerChance: r.opt("triggerChance"), Triggered: r.flag("triggered"),
			NoSupports: r.flag("noSupports"), ApplyToPlayer: r.flag("applyToPlayer"), Source: r.str("source"), MinionList: r.strs("minionList")}
	case valueKindData:
		d := DataRef{Key: r.str("key"), Merge: r.str("merge")}
		if inner, ok := r.take("value"); ok {
			d.Value, ok = inner.(Value)
			if !ok {
				r.bad = true
			}
		}
		v = d
	case valueKindGemProperty:
		v = GemPropertyRef{Keyword: r.str("keyword"), KeywordList: r.strs("keywordList"), Key: r.str("key"), Value: r.opt("value"), KeyOfScaledMod: r.str("keyOfScaledMod")}
	case valueKindExplode:
		v = ExplodeRef{Type: r.str("type"), Chance: r.num("chance"), Amount: r.num("amount"), KeyOfScaledMod: r.str("keyOfScaledMod")}
	case valueKindJewelFn:
		j := JewelFn{Type: r.str("type"), Radius: r.str("radius")}
		if fn, ok := r.take("func"); ok {
			if ref, isRef := fn.(JewelFnRef); isRef {
				j.Func, j.ID = ref.Fn, ref.ID
			}
		}
		v = j
	case valueKindConqueredBy:
		c := ConqueredBy{Seed: r.num("id")}
		if cq, ok := r.take("conqueror"); ok {
			if t, isConq := cq.(Conqueror); isConq {
				c.Conqueror = &t
			} else {
				r.bad = true
			}
		}
		v = c
	case valueKindPairs:
		v = r.pairs("pairs")
	case valueKindSelfDamage:
		v = SelfDamage{BaseDamage: r.opt("baseDamage"), DmgMult: r.opt("dmgMult"), DamageType: r.str("damageType")}
	case valueKindAscendancyNode:
		v = AscendancyNodeRef{Name: r.str("name"), Side: r.str("side")}
	case valueKindLinkedSupport:
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
