// Renders gamedata.Minions as Data/Spectres.lua and Data/Minions.lua
// (Scripts/minions.lua's outputs), replaying the templates and emitting one
// definition per #emit / #spectre directive.

package luarender

import (
	"fmt"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() { register("minions", renderMinions) }

func renderMinions(d gamedata.Minions, tpl Templates) (map[string]string, error) {
	files := map[string]string{}
	for _, tf := range []struct {
		name string
		defs []gamedata.MinionDef
	}{{"Spectres", d.Spectres}, {"Minions", d.Minions}} {
		next := 0
		emit := func(_ string, b *B) {
			if next >= len(tf.defs) {
				panic(fmt.Sprintf("minions: template %s has more emits than definitions", tf.name))
			}
			m := tf.defs[next]
			next++
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
			if m.Hostile != "" {
				b.W("\thostile = ", m.Hostile, ",\n")
			}
			b.W("\tskillList = {\n")
			for _, skill := range m.SkillList {
				b.W("\t\t\"", skill, "\",\n")
			}
			b.W("\t},\n")
			b.W("\tmodList = {\n")
			for _, line := range m.ModList {
				b.W("\t\t", line, "\n")
			}
			b.W("\t},\n")
			b.W("}\n")
		}
		directives := map[string]func(args string, b *B){
			"emit":    emit,
			"spectre": emit,
		}
		var b B
		if err := processTemplate(tpl, tf.name, "Minions/", &b, directives); err != nil {
			return nil, err
		}
		files["Data/"+tf.name+".lua"] = b.String()
	}
	return files, nil
}
