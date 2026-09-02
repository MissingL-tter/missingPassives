package luacanon

// The item's typed sub-records (weaponData, armourData, flaskData,
// tinctureData, jewelData, grantedSkills, requirements) render as the flat
// scalar tables dump_build.lua's itemFixture held, and decode back from
// them. Keys are the reference's; absent fields are absent keys;
// weaponData's Extra entries merge in (scalars only, as the fixture's
// scalar projection kept).

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
)

func init() {
	RegisterAdapter(func(v any) (any, bool) {
		switch t := v.(type) {
		case *item.WeaponData:
			return WeaponDataTable(t), true
		case *item.ArmourData:
			return ArmourDataTable(t), true
		case *item.FlaskData:
			return FlaskDataTable(t), true
		case *item.TinctureData:
			return TinctureDataTable(t), true
		case *item.JewelData:
			return JewelDataTable(t), true
		case item.GrantedSkill:
			return GrantedSkillTable(t), true
		case *item.Requirements:
			return RequirementsTable(t), true
		case item.Requirements:
			return RequirementsTable(&t), true
		}
		return nil, false
	})
}

// table collects a record's present keys.
type table map[string]any

func (t table) num(key string, v float64) {
	if v != 0 {
		t[key] = v
	}
}
func (t table) opt(key string, v util.Opt[float64]) {
	if v.Set {
		t[key] = v.V
	}
}
func (t table) str(key string, v string) {
	if v != "" {
		t[key] = v
	}
}
func (t table) flag(key string, v bool) {
	if v {
		t[key] = v
	}
}

// extra merges the scalar Extra entries.
func (t table) extra(m map[string]modparser.Value) {
	for k, v := range m {
		switch s := v.(type) {
		case modparser.Num:
			t[k] = float64(s)
		case modparser.Bool:
			t[k] = bool(s)
		case modparser.Str:
			t[k] = string(s)
		}
	}
}

// WeaponDataTable is one weaponData side as the reference table.
func WeaponDataTable(w *item.WeaponData) map[string]any {
	t := table{}
	t.str("type", w.Type)
	t.str("subType", w.SubType)
	t.str("name", w.Name)
	t.num("AttackRate", w.AttackRate)
	t.num("range", w.Range)
	t.opt("AttackSpeedInc", w.AttackSpeedInc)
	t.opt("rangeBonus", w.RangeBonus)
	t.opt("CritChance", w.CritChance)
	t.opt("TotalDPS", w.TotalDPS)
	t.num("ElementalDPS", w.ElementalDPS)
	for _, dmgType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		r := w.Damage(dmgType)
		t.num(dmgType+"Min", r.Min)
		t.num(dmgType+"Max", r.Max)
		t.num(dmgType+"DPS", r.DPS)
	}
	t.flag("countsAsAll1H", w.CountsAsAll1H)
	t.flag("countsAsDualWielding", w.CountsAsDualWielding)
	if a := w.AddedUsing; a.Written {
		t["AddedUsingAxe"] = a.Axe
		t["AddedUsingSword"] = a.Sword
		t["AddedUsingDagger"] = a.Dagger
		t["AddedUsingMace"] = a.Mace
		t["AddedUsingClaw"] = a.Claw
	}
	t.extra(w.Extra)
	return t
}

// ArmourDataTable is armourData as the reference table.
func ArmourDataTable(a *item.ArmourData) map[string]any {
	t := table{}
	for _, name := range []string{"Armour", "Evasion", "EnergyShield", "Ward"} {
		stat := a.Defence(name)
		t.opt(name, stat.Value)
		t.opt(name+"BasePercentile", stat.BasePercentile)
	}
	t.opt("BlockChance", a.BlockChance)
	return t
}

// FlaskDataTable is flaskData as the reference table.
func FlaskDataTable(f *item.FlaskData) map[string]any {
	t := table{
		"duration": f.Duration, "chargesMax": f.ChargesMax, "chargesUsed": f.ChargesUsed,
		"gainMod": f.GainMod, "effectInc": f.EffectInc,
	}
	t.opt("instantPerc", f.InstantPerc)
	t.opt("instantLowLifePerc", f.InstantLowLifePerc)
	for _, pool := range []struct {
		prefix string
		rec    *item.FlaskRecovery
	}{{"life", f.Life}, {"mana", f.Mana}} {
		if pool.rec == nil {
			continue
		}
		t[pool.prefix+"Base"] = pool.rec.Base
		t[pool.prefix+"Instant"] = pool.rec.Instant
		t[pool.prefix+"Gradual"] = pool.rec.Gradual
		t[pool.prefix+"Total"] = pool.rec.Total
		t.opt(pool.prefix+"Additional", pool.rec.Additional)
		t[pool.prefix+"EffectNotRemoved"] = pool.rec.EffectNotRemoved
	}
	return t
}

// TinctureDataTable is tinctureData as the reference table.
func TinctureDataTable(d *item.TinctureData) map[string]any {
	t := table{"manaBurn": d.ManaBurn, "cooldownInc": d.CooldownInc, "cooldown": d.Cooldown, "effectInc": d.EffectInc}
	return t
}

// JewelDataTable is jewelData's scalar projection (the fixture drops the
// record/list entries: conqueredBy, funcList, the cluster name lists).
func JewelDataTable(j *item.JewelData) map[string]any {
	t := table{}
	t.num("radiusIndex", float64(j.RadiusIndex))
	t.flag("intuitiveLeapLike", j.IntuitiveLeapLike)
	t.flag("intuitiveLeapKeystoneOnly", j.IntuitiveLeapKeystoneOnly)
	t.str("impossibleEscapeKeystone", j.ImpossibleEscapeKeystone)
	t.num("jewelIncEffectFromClassStart", j.JewelIncEffectFromClassStart)
	t.num("corruptedMagicJewelIncEffect", j.CorruptedMagicJewelIncEffect)
	t.num("corruptedRareJewelIncEffect", j.CorruptedRareJewelIncEffect)
	t.flag("limitDisabled", j.LimitDisabled)
	t.str("clusterJewelKeystone", j.ClusterJewelKeystone)
	t.str("clusterJewelSkill", j.ClusterJewelSkill)
	t.num("clusterJewelNodeCount", float64(j.ClusterJewelNodeCount))
	t.num("clusterJewelSocketCount", float64(j.ClusterJewelSocketCount))
	t.num("clusterJewelSocketCountOverride", float64(j.ClusterJewelSocketCountOverride))
	t.num("clusterJewelNothingnessCount", float64(j.ClusterJewelNothingnessCount))
	t.num("clusterJewelIncEffect", j.ClusterJewelIncEffect)
	t.flag("clusterJewelSmallsAreNothingness", j.ClusterJewelSmallsAreNothingness)
	// The reference stores the or-chain's operand: the keystone name, the
	// node count, or the nothingness count.
	if j.ClusterJewelValid {
		switch {
		case j.ClusterJewelKeystone != "":
			t["clusterJewelValid"] = j.ClusterJewelKeystone
		case (j.ClusterJewelSkill != "" || j.ClusterJewelSmallsAreNothingness) && j.ClusterJewelNodeCount != 0:
			t["clusterJewelValid"] = float64(j.ClusterJewelNodeCount)
		default:
			t["clusterJewelValid"] = float64(j.ClusterJewelNothingnessCount)
		}
	}
	return t
}

// GrantedSkillTable is one grantedSkills entry as the reference table.
func GrantedSkillTable(g item.GrantedSkill) map[string]any {
	t := table{"source": g.Source}
	t.str("skillId", g.SkillID)
	t.opt("level", g.Level)
	t.flag("noSupports", g.NoSupports)
	t.flag("triggered", g.Triggered)
	t.opt("triggerChance", g.TriggerChance)
	return t
}

// RequirementsTable is item.requirements as the reference table.
func RequirementsTable(r *item.Requirements) map[string]any {
	t := table{}
	t.opt("str", r.Str)
	t.opt("dex", r.Dex)
	t.opt("int", r.Int)
	t.opt("level", r.Level)
	t.opt("strMod", r.StrMod)
	t.opt("dexMod", r.DexMod)
	t.opt("intMod", r.IntMod)
	return t
}

// dataSetter is the {key, value} application every item data record has.
type dataSetter interface {
	Set(key string, v modparser.Value)
}

func setAll(dst dataSetter, m map[string]any) {
	for k, v := range m {
		dst.Set(k, ValueFromTable(v))
	}
}

// WeaponDataFromTable rebuilds one weaponData side from its fixture table.
func WeaponDataFromTable(m map[string]any) *item.WeaponData {
	w := &item.WeaponData{}
	setAll(w, m)
	return w
}

// ArmourDataFromTable rebuilds armourData from its fixture table.
func ArmourDataFromTable(m map[string]any) *item.ArmourData {
	a := &item.ArmourData{}
	setAll(a, m)
	return a
}

// FlaskDataFromTable rebuilds flaskData from its fixture table.
func FlaskDataFromTable(m map[string]any) *item.FlaskData {
	f := &item.FlaskData{}
	setAll(f, m)
	return f
}

// TinctureDataFromTable rebuilds tinctureData from its fixture table.
func TinctureDataFromTable(m map[string]any) *item.TinctureData {
	d := &item.TinctureData{}
	setAll(d, m)
	return d
}

// JewelDataFromTable rebuilds jewelData's scalars from its fixture table.
func JewelDataFromTable(m map[string]any) *item.JewelData {
	j := &item.JewelData{}
	setAll(j, m)
	return j
}

// GrantedSkillFromTable rebuilds one grantedSkills entry.
func GrantedSkillFromTable(m map[string]any) item.GrantedSkill {
	g := item.GrantedSkill{}
	for k, v := range m {
		switch k {
		case "skillId":
			g.SkillID = v.(string)
		case "source":
			g.Source = v.(string)
		case "level":
			g.Level = util.Some(v.(float64))
		case "triggerChance":
			g.TriggerChance = util.Some(v.(float64))
		case "noSupports":
			g.NoSupports = v.(bool)
		case "triggered":
			g.Triggered = v.(bool)
		default:
			panic("luacanon: unknown grantedSkills key " + k)
		}
	}
	return g
}

// RequirementsFromTable rebuilds item.requirements.
func RequirementsFromTable(m map[string]any) *item.Requirements {
	r := &item.Requirements{}
	for k, v := range m {
		n := util.Some(v.(float64))
		if strings.HasSuffix(k, "Mod") {
			switch k {
			case "strMod":
				r.StrMod = n
			case "dexMod":
				r.DexMod = n
			case "intMod":
				r.IntMod = n
			}
			continue
		}
		if p := r.Attribute(k); p != nil {
			*p = n
			continue
		}
		panic("luacanon: unknown requirements key " + k)
	}
	return r
}
