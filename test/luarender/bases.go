// Renders schema.BasesData as the Data/Bases/<type>.lua files and
// Data/Rares.lua (Scripts/bases.lua's outputs).

package luarender

import (
	"fmt"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() { register("bases", renderBases) }

var basesItemTypes = []string{
	"axe", "bow", "claw", "dagger", "fishing", "mace", "staff", "sword", "wand",
	"helmet", "body", "gloves", "boots", "shield", "quiver",
	"amulet", "ring", "belt", "jewel", "flask", "tincture", "graft",
}

// inlineStrings is Utils.stringifyInline over an array of strings.
func inlineStrings(list []string) string {
	var b strings.Builder
	b.WriteString("{ ")
	for _, s := range list {
		b.WriteString(luaQ(s))
		b.WriteString(", ")
	}
	b.WriteString("}")
	return b.String()
}

func renderItemBase(b *B, ib schema.ItemBase) {
	b.W("itemBases[\"", ib.DisplayName, "\"] = {\n")
	b.W("\ttype = \"", ib.Type, "\",\n")
	if ib.SubType != "" {
		b.W("\tsubType = \"", ib.SubType, "\",\n")
	}
	if ib.Hidden {
		b.W("\thidden = true,\n")
	}
	if ib.SocketLimit != nil {
		b.W("\tsocketLimit = ", *ib.SocketLimit, ",\n")
	}
	b.W("\ttags = { ")
	for _, tag := range ib.Tags {
		b.W(tag, " = true, ")
	}
	b.W("},\n")
	if ib.InfluenceBaseTag != "" {
		b.W("\tinfluenceTags = { ")
		for i, suffix := range []string{"shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie", "cleansing", "tangle"} {
			if i != 0 {
				b.W(", ")
			}
			b.W(suffix, " = \"", ib.InfluenceBaseTag, "_", suffix, "\"")
		}
		b.W(" },\n")
	}
	if len(ib.Implicit) > 0 {
		b.W("\timplicit = \"", strings.Join(ib.Implicit, "\\n"), "\",\n")
	}
	b.W("\timplicitModTypes = { ")
	for _, t := range ib.ImplicitModTypes {
		b.W("{ ", joinModTags(t), " }, ")
	}
	b.W("},\n")
	if len(ib.ImplicitIds) > 0 {
		b.W("\timplicitIds = ", inlineStrings(ib.ImplicitIds), ",\n")
	}
	if len(ib.Enchant) > 0 {
		b.W("\tenchant = \"", strings.Join(ib.Enchant, "\\n"), "\",\n")
		if len(ib.EnchantIds) > 0 {
			b.W("\tenchantIds = ", inlineStrings(ib.EnchantIds), ",\n")
		}
		b.W("\tenchantModTypes = { ")
		for _, t := range ib.EnchantModTypes {
			b.W("{ ", joinModTags(t), " }, ")
		}
		b.W("},\n")
	}
	if ib.CannotBeAnointed {
		b.W("\tcannotBeAnointed = true,\n")
	}
	if w := ib.Weapon; w != nil {
		b.W("\tweapon = { ")
		b.W("PhysicalMin = ", w.PhysicalMin, ", PhysicalMax = ", w.PhysicalMax, ", ")
		b.W("CritChanceBase = ", w.CritChanceBase, ", ")
		b.W("AttackRateBase = ", w.AttackRateBase, ", ")
		b.W("Range = ", w.Range, ", ")
		b.W("},\n")
	}
	if a := ib.Armour; a != nil {
		b.W("\tarmour = { ")
		if a.BlockChance != nil {
			b.W("BlockChance = ", *a.BlockChance, ", ")
		}
		pair := func(name string, mn, mx *int64) {
			if mn != nil {
				b.W(name, "BaseMin = ", *mn, ", ")
				b.W(name, "BaseMax = ", *mx, ", ")
			}
		}
		pair("Armour", a.ArmourMin, a.ArmourMax)
		pair("Evasion", a.EvasionMin, a.EvasionMax)
		pair("EnergyShield", a.EnergyShieldMin, a.EnergyShieldMax)
		if a.MovementPenalty != nil {
			b.W("MovementPenalty = ", *a.MovementPenalty, ", ")
		}
		pair("Ward", a.WardMin, a.WardMax)
		b.W("},\n")
	}
	if f := ib.Flask; f != nil {
		b.W("\tflask = { ")
		if f.Life != nil {
			b.W("life = ", *f.Life, ", ")
		}
		if f.Mana != nil {
			b.W("mana = ", *f.Mana, ", ")
		}
		b.W("duration = ", f.Duration, ", ")
		b.W("chargesUsed = ", f.ChargesUsed, ", ")
		b.W("chargesMax = ", f.ChargesMax, ", ")
		if f.HasBuff {
			b.W("buff = { \"", strings.Join(f.Buff, "\", \""), "\" }, ")
		}
		b.W("},\n")
	}
	if t := ib.Tincture; t != nil {
		b.W("\ttincture = { manaBurn = ", t.ManaBurn, ", cooldown = ", t.Cooldown, " },\n")
	}
	b.W("\treq = { ")
	if ib.ReqLevel != nil {
		b.W("level = ", *ib.ReqLevel, ", ")
	}
	for _, attr := range []struct {
		name string
		v    *int64
	}{{"str", ib.ReqStr}, {"dex", ib.ReqDex}, {"int", ib.ReqInt}} {
		if attr.v != nil {
			b.W(attr.name, " = ", *attr.v, ", ")
		}
	}
	b.W("},\n")
	if len(ib.FlavourText) > 0 {
		b.W("\tflavourText = {\n")
		for _, line := range ib.FlavourText {
			b.W("\t\t\"", luaEsc(line), "\",\n")
		}
		b.W("\t},\n")
	}
	b.W("}\n")
}

func renderBases(d schema.BasesData, tpl Templates) (map[string]string, error) {
	files := map[string]string{}
	for _, name := range basesItemTypes {
		events := d.Types[name]
		next := 0
		emit := func(_ string, b *B) {
			if next >= len(events) {
				panic(fmt.Sprintf("bases: template %s has more emitting directives than events", name))
			}
			for _, ib := range events[next] {
				renderItemBase(b, ib)
			}
			next++
		}
		var b B
		err := processTemplate(tpl, name, "Bases/", &b, map[string]func(string, *B){
			"base": emit, "baseMatch": emit,
		})
		if err != nil {
			return nil, err
		}
		files["Data/Bases/"+name+".lua"] = b.String()
	}

	next := 0
	emitRare := func(_ string, b *B) {
		if next >= len(d.Rares) {
			panic("bases: Rares template has more directives than events")
		}
		r := d.Rares[next]
		next++
		if r == nil {
			return
		}
		b.W("[[\n")
		for _, line := range r.Lines {
			b.W(line, "\n")
		}
		b.W("]],")
	}
	var rb B
	err := processTemplate(tpl, "Rares", "Bases/", &rb, map[string]func(string, *B){
		"setBestBase": emitRare, "setBase": emitRare,
	})
	if err != nil {
		return nil, err
	}
	files["Data/Rares.lua"] = rb.String()
	return files, nil
}
