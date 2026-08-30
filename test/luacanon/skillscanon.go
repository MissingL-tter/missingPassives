package luacanon

// Socket groups and gems render as the flat scalar tables dump_calc.lua's
// skillsTab fixture held (the reference's gemInstance/socketGroup keys),
// and decode back from them. Absent fields are absent keys; a plain string
// "" or number 0 that the reference never stores is absent too.

import (
	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/skills"
)

// Extras holds fixture keys the typed model has no field for (view-only
// colour codes and display labels), keyed by the decoded struct, so the
// fixture echo re-emits them. Filled by the *FromTable decoders.
var Extras = map[any]map[string]any{}

func init() {
	RegisterAdapter(func(v any) (any, bool) {
		switch t := v.(type) {
		case *skills.Gem:
			return withExtras(t, GemInstanceTable(t)), true
		case *skills.SocketGroup:
			return withExtras(t, SocketGroupTable(t)), true
		case *calc.SocketGemInput:
			m := map[string]any{"kv": withExtras(t.Gem, GemInstanceTable(t.Gem))}
			if t.GemDataID != nil {
				m["gemDataId"] = *t.GemDataID
			}
			if t.GrantedEffectID != nil {
				m["grantedEffectId"] = *t.GrantedEffectID
			}
			if t.ExplodeSourceItemID != nil {
				m["explodeSourceItemId"] = *t.ExplodeSourceItemID
			}
			if t.ExplodeSourceNodeID != nil {
				m["explodeSourceNodeId"] = *t.ExplodeSourceNodeID
			}
			return m, true
		case *calc.SocketGroupInput:
			return map[string]any{
				"kv":      withExtras(t.SocketGroup, SocketGroupTable(t.SocketGroup)),
				"gemList": t.GemList,
			}, true
		}
		return nil, false
	})
}

func withExtras(key any, t map[string]any) map[string]any {
	for k, v := range Extras[key] {
		t[k] = v
	}
	return t
}

func (t table) optBool(key string, v util.Opt[bool]) {
	if v.Set {
		t[key] = v.V
	}
}

// GemInstanceTable is one gem instance as the reference table.
func GemInstanceTable(g *skills.Gem) map[string]any {
	t := table{"nameSpec": g.NameSpec, "level": g.Level, "quality": g.Quality, "enabled": g.Enabled}
	t.str("gemId", g.GemID)
	t.str("skillId", g.SkillID)
	t.str("errMsg", g.ErrMsg)
	t.opt("count", g.Count)
	t.optBool("enableGlobal1", g.EnableGlobal1)
	t.optBool("enableGlobal2", g.EnableGlobal2)
	t.optBool("matchesSocket", g.MatchesSocket)
	t.flag("new", g.New)
	t.flag("triggered", g.Triggered)
	t.flag("noSupports", g.NoSupports)
	t.optBool("fromItem", g.FromItem)
	t.opt("reqLevel", g.ReqLevel)
	t.opt("reqStr", g.ReqStr)
	t.opt("reqDex", g.ReqDex)
	t.opt("reqInt", g.ReqInt)
	t.num("naturalMaxLevel", g.NaturalMaxLevel)
	t.opt("triggerChance", g.TriggerChance)
	t.opt("skillPart", g.SkillPart)
	t.opt("skillPartCalcs", g.SkillPartCalcs)
	t.opt("skillStageCount", g.SkillStageCount)
	t.opt("skillStageCountCalcs", g.SkillStageCountCalcs)
	t.opt("skillMineCount", g.SkillMineCount)
	t.opt("skillMineCountCalcs", g.SkillMineCountCalcs)
	t.str("skillMinion", g.SkillMinion)
	t.str("skillMinionCalcs", g.SkillMinionCalcs)
	t.opt("skillMinionSkill", g.SkillMinionSkill)
	t.opt("skillMinionSkillCalcs", g.SkillMinionSkillCalcs)
	t.opt("skillMinionItemSet", g.SkillMinionItemSet)
	t.opt("skillMinionItemSetCalcs", g.SkillMinionItemSetCalcs)
	return t
}

// SocketGroupTable is one socket group's scalars as the reference table.
func SocketGroupTable(g *skills.SocketGroup) map[string]any {
	t := table{"enabled": g.Enabled, "label": g.Label}
	t.optBool("includeInFullDPS", g.IncludeInFullDPS)
	t.num("groupCount", g.GroupCount)
	t.str("slot", g.Slot)
	t.str("source", g.Source)
	t.str("imbuedSupport", g.ImbuedSupport)
	t.opt("mainActiveSkill", g.MainActiveSkill)
	t.opt("mainActiveSkillCalcs", g.MainActiveSkillCalcs)
	t.flag("noSupports", g.NoSupports)
	t.optBool("slotEnabled", g.SlotEnabled)
	return t
}

// tableReader pulls typed keys out of a fixture table, tracking the ones
// consumed so the leftovers can be kept as Extras.
type tableReader struct {
	m    map[string]any
	used map[string]bool
}

func (r *tableReader) str(key string) string {
	r.used[key] = true
	s, _ := r.m[key].(string)
	return s
}
func (r *tableReader) num(key string) float64 {
	r.used[key] = true
	n, _ := r.m[key].(float64)
	return n
}
func (r *tableReader) flag(key string) bool {
	r.used[key] = true
	b, _ := r.m[key].(bool)
	return b
}
func (r *tableReader) optNum(key string) util.Opt[float64] {
	r.used[key] = true
	if n, ok := r.m[key].(float64); ok {
		return util.Some(n)
	}
	return util.Opt[float64]{}
}
func (r *tableReader) optBool(key string) util.Opt[bool] {
	r.used[key] = true
	if b, ok := r.m[key].(bool); ok {
		return util.Some(b)
	}
	return util.Opt[bool]{}
}
func (r *tableReader) leftovers() map[string]any {
	var out map[string]any
	for k, v := range r.m {
		if !r.used[k] {
			if out == nil {
				out = map[string]any{}
			}
			out[k] = v
		}
	}
	return out
}

// GemInstanceFromTable rebuilds a gem instance from its fixture table; keys without a field
// are kept in Extras.
func GemInstanceFromTable(m map[string]any) *skills.Gem {
	r := &tableReader{m: m, used: map[string]bool{}}
	g := &skills.Gem{
		NameSpec:                r.str("nameSpec"),
		GemID:                   r.str("gemId"),
		SkillID:                 r.str("skillId"),
		ErrMsg:                  r.str("errMsg"),
		Level:                   r.num("level"),
		Quality:                 r.num("quality"),
		Count:                   r.optNum("count"),
		Enabled:                 r.flag("enabled"),
		EnableGlobal1:           r.optBool("enableGlobal1"),
		EnableGlobal2:           r.optBool("enableGlobal2"),
		MatchesSocket:           r.optBool("matchesSocket"),
		New:                     r.flag("new"),
		Triggered:               r.flag("triggered"),
		NoSupports:              r.flag("noSupports"),
		FromItem:                r.optBool("fromItem"),
		ReqLevel:                r.optNum("reqLevel"),
		ReqStr:                  r.optNum("reqStr"),
		ReqDex:                  r.optNum("reqDex"),
		ReqInt:                  r.optNum("reqInt"),
		NaturalMaxLevel:         r.num("naturalMaxLevel"),
		TriggerChance:           r.optNum("triggerChance"),
		SkillPart:               r.optNum("skillPart"),
		SkillPartCalcs:          r.optNum("skillPartCalcs"),
		SkillStageCount:         r.optNum("skillStageCount"),
		SkillStageCountCalcs:    r.optNum("skillStageCountCalcs"),
		SkillMineCount:          r.optNum("skillMineCount"),
		SkillMineCountCalcs:     r.optNum("skillMineCountCalcs"),
		SkillMinion:             r.str("skillMinion"),
		SkillMinionCalcs:        r.str("skillMinionCalcs"),
		SkillMinionSkill:        r.optNum("skillMinionSkill"),
		SkillMinionSkillCalcs:   r.optNum("skillMinionSkillCalcs"),
		SkillMinionItemSet:      r.optNum("skillMinionItemSet"),
		SkillMinionItemSetCalcs: r.optNum("skillMinionItemSetCalcs"),
	}
	if extra := r.leftovers(); extra != nil {
		Extras[g] = extra
	}
	return g
}

// SocketGroupFromTable rebuilds a group's scalars from its fixture table
// (the gem list is the caller's); keys without a field are kept in Extras.
func SocketGroupFromTable(m map[string]any) *skills.SocketGroup {
	r := &tableReader{m: m, used: map[string]bool{}}
	g := &skills.SocketGroup{
		Enabled:              r.flag("enabled"),
		IncludeInFullDPS:     r.optBool("includeInFullDPS"),
		GroupCount:           r.num("groupCount"),
		Label:                r.str("label"),
		Slot:                 r.str("slot"),
		Source:               r.str("source"),
		ImbuedSupport:        r.str("imbuedSupport"),
		MainActiveSkill:      r.optNum("mainActiveSkill"),
		MainActiveSkillCalcs: r.optNum("mainActiveSkillCalcs"),
		NoSupports:           r.flag("noSupports"),
		SlotEnabled:          r.optBool("slotEnabled"),
	}
	if extra := r.leftovers(); extra != nil {
		Extras[g] = extra
	}
	return g
}
