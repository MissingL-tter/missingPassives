// Renders schema.Minions as Data/Spectres.lua and Data/Minions.lua
// (Scripts/minions.lua's outputs), replaying the templates and emitting one
// definition per #emit / #spectre directive.

package luarender

import (
	"fmt"
	"sort"

	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// tableToString reproduces minions.lua's tableToString over the decoded
// structured values: sorted keys, nested tables inlined with a dotted
// prefix.
// #EVAL: archive parity — the comma logic skips nested-table entries, so a
// nested table directly abuts its neighbour without a separator.
func tableToString(v any, pre string) string {
	type kv struct {
		name string
		num  float64
		str  bool
		val  any
	}
	var entries []kv
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			entries = append(entries, kv{name: k, str: true, val: e})
		}
	case []any:
		for i, e := range t {
			entries = append(entries, kv{name: luaNum(float64(i + 1)), num: float64(i + 1), val: e})
		}
	default:
		panic(fmt.Sprintf("tableToString: unexpected value %T", v))
	}
	sort.Slice(entries, func(a, b int) bool {
		if entries[a].str != entries[b].str {
			panic("tableToString: mixed key types")
		}
		if entries[a].str {
			return entries[a].name < entries[b].name
		}
		return entries[a].num < entries[b].num
	})
	s := "{ "
	for i, e := range entries {
		switch sub := e.val.(type) {
		case map[string]any:
			s += tableToString(sub, pre+e.name+".")
		case []any:
			s += tableToString(sub, pre+e.name+".")
		default:
			if i > 0 {
				s += ", "
			}
			quote := ""
			if _, ok := e.val.(string); ok {
				quote = "\""
			}
			s += pre + e.name + " = " + quote + luaAnyString(e.val) + quote
		}
	}
	return s + " }"
}

// luaAnyString is tostring() over the value kinds tableToString meets.
func luaAnyString(v any) string {
	switch t := v.(type) {
	case float64:
		return luaNum(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	}
	panic(fmt.Sprintf("luaAnyString: unexpected value %T", v))
}

// modEntryText reprints one generated minion mod as the reference file's
// mod(...) constructor line (or the unmapped-stat comment).
func modEntryText(e schema.ModEntry) string {
	modStats := " [" + e.Stat + " = " + luaNum(*e.StatValue) + "]"
	if len(e.Mods) == 0 {
		return "-- " + e.Entry + modStats
	}
	mod := modparser.DecodeMods(e.Mods)[0]
	valueStr := ""
	switch v := luacanon.ValueTable(mod.Value).(type) {
	case map[string]any:
		valueStr = tableToString(v, "")
	case []any:
		valueStr = tableToString(v, "")
	default:
		valueStr = luaAnyString(v)
	}
	line := "mod(\"" + mod.Name + "\", \"" + mod.Type.String() + "\", " + valueStr +
		", " + luaNum(float64(mod.Flags)) + ", " + luaNum(float64(mod.KeywordFlags))
	for _, tag := range mod.Tags {
		line += ", " + tableToString(luacanon.TagTable(tag), "")
	}
	return line + "), -- " + e.Entry + modStats
}

func init() { register("minions", renderMinions) }

func renderMinions(d schema.Minions, tpl Templates) (map[string]string, error) {
	files := map[string]string{}
	for _, tf := range []struct {
		name string
		defs []schema.MinionDef
	}{{"Spectres", d.Spectres}, {"Minions", d.Minions}} {
		next := 0
		var extraTexts []string
		emit := func(_ string, b *B) {
			if next >= len(tf.defs) {
				panic(fmt.Sprintf("minions: template %s has more emits than definitions", tf.name))
			}
			m := tf.defs[next]
			next++
			// Consume this monster's queued hand-written mods (discarded on
			// skip, as the build discards them at the next #monster).
			extras := extraTexts
			extraTexts = nil
			if m.Skip {
				return
			}
			b.W("minions[\"", m.Key, "\"] = {\n")
			b.W("\tname = \"", m.Name, "\",\n")
			b.W("\tmonsterTags = { ")
			for _, tag := range m.MonsterTags {
				b.W("\"", tag, "\", ")
			}
			b.W("},\n")
			if m.BaseDamageIgnoresAttackSpeed {
				b.W("\tbaseDamageIgnoresAttackSpeed = true,\n")
			}
			b.W("\tlife = ", m.Life, ",\n")
			for _, ls := range m.LifeScaling {
				b.W("\tlifeScaling = \"", ls, "\",\n")
			}
			if m.EnergyShield != nil {
				b.W("\tenergyShield = ", *m.EnergyShield, ",\n")
			}
			if m.Armour != nil {
				b.W("\tarmour = ", *m.Armour, ",\n")
			}
			if m.Evasion != nil {
				b.W("\tevasion = ", *m.Evasion, ",\n")
			}
			b.W("\tfireResist = ", m.FireResist, ",\n")
			b.W("\tcoldResist = ", m.ColdResist, ",\n")
			b.W("\tlightningResist = ", m.LightningResist, ",\n")
			b.W("\tchaosResist = ", m.ChaosResist, ",\n")
			b.W("\tdamage = ", m.Damage, ",\n")
			b.W("\tdamageSpread = ", m.DamageSpread, ",\n")
			b.W("\tattackTime = ", m.AttackTime, ",\n")
			b.W("\tattackRange = ", m.AttackRange, ",\n")
			b.W("\taccuracy = ", m.Accuracy, ",\n")
			for _, f := range m.DamageFixups {
				b.W("\tdamageFixup = ", f, ",\n")
			}
			if m.WeaponType1 != nil {
				b.W("\tweaponType1 = \"", *m.WeaponType1, "\",\n")
			}
			if m.WeaponType2 != nil {
				b.W("\tweaponType2 = \"", *m.WeaponType2, "\",\n")
			}
			if m.Limit != "" {
				b.W("\tlimit = \"", m.Limit, "\",\n")
			}
			if m.Hostile {
				b.W("\thostile = true,\n")
			}
			b.W("\tskillList = {\n")
			for _, skill := range m.SkillList {
				b.W("\t\t\"", skill, "\",\n")
			}
			b.W("\t},\n")
			b.W("\tmodList = {\n")
			for _, e := range m.ModList {
				if e.Extra {
					// Hand-written template mods print verbatim from the
					// archive template's #mod directives, in order.
					if len(extras) == 0 {
						panic("minions: more extra mods in the document than #mod directives in the template")
					}
					b.W("\t\t", extras[0], ",\n")
					extras = extras[1:]
					continue
				}
				b.W("\t\t", modEntryText(e), "\n")
			}
			b.W("\t},\n")
			b.W("}\n")
		}
		directives := map[string]func(args string, b *B){
			"emit":    emit,
			"spectre": emit,
			"mod":     func(args string, _ *B) { extraTexts = append(extraTexts, args) },
		}
		var b B
		if err := processTemplate(tpl, tf.name, "Minions/", &b, directives); err != nil {
			return nil, err
		}
		files["Data/"+tf.name+".lua"] = b.String()
	}
	return files, nil
}
