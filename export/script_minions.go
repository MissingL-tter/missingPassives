// Port of .archive/src/Export/Scripts/minions.lua.

package export

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/modparser"
)

func init() {
	Scripts = append(Scripts, Script{Name: "minions", Build: buildMinions})
}

// statEntry is one source of minion stats in the emit loop: a Mods row's
// six Stat<i>/Stat<i>Value slots, or an .ot Stats-block entry occupying
// slot 1 only (minions.lua's modStatX reads either the same way).
type statEntry struct {
	ID    string
	Stats [6]struct {
		ID  string
		Val float64
	}
}

func statEntryFromMod(mod *Row) statEntry {
	e := statEntry{ID: mod.Str("Id")}
	for i := range e.Stats {
		if sr := mod.Ref(fmt.Sprintf("Stat%d", i+1)); sr != nil {
			e.Stats[i].ID = sr.Str("Id")
			e.Stats[i].Val = float64(mod.Ivl(fmt.Sprintf("Stat%dValue", i+1))[0])
		}
	}
	return e
}

func statEntryFromOT(id string, val float64) statEntry {
	e := statEntry{ID: id}
	e.Stats[0].ID, e.Stats[0].Val = id, val
	return e
}

var (
	reExtends = regexp.MustCompile(`extends "(.+)"`)
	reKeyVal  = regexp.MustCompile(`^(.*?)=(.+)$`)
)

var otWs = strings.NewReplacer(" ", "", "\t", "", "\v", "", "\f", "", "\r", "")

// getOTStats ports minions.lua's getOTStats: appends the Stats-block entries
// of an .ot file, superclasses first.
func (x *Ctx) getOTStats(otFile string, modList []statEntry) ([]statEntry, error) {
	file := otFile + ".ot"
	var text string
	if cached, ok := x.otCache[file]; ok {
		text = cached
	} else if raw := x.GetFile(file); raw != "" {
		text = convertUTF16to8([]byte(raw), 0)
		if x.otCache == nil {
			x.otCache = map[string]string{}
		}
		x.otCache[file] = text
	} else {
		// The Lua prints "Invalid OT File location".
		return modList, nil
	}
	inWantedBlock := false
	for _, line := range reLine.FindAllString(text, -1) {
		if m := reExtends.FindStringSubmatch(line); m != nil && m[1] != "Metadata/Monsters/Monster" && m[1] != "nothing" {
			var err error
			if modList, err = x.getOTStats(m[1], modList); err != nil {
				return nil, err
			}
		}
		if strings.HasPrefix(line, "Stats") {
			inWantedBlock = true
		} else if inWantedBlock && strings.HasPrefix(line, "}") {
			inWantedBlock = false
		} else if inWantedBlock && strings.Contains(line, "=") && !strings.Contains(line, "//") {
			stripped := otWs.Replace(line)
			if m := reKeyVal.FindStringSubmatch(stripped); m != nil {
				v, err := strconv.ParseFloat(m[2], 64)
				if err != nil {
					return nil, fmt.Errorf("%s: non-numeric stat value %q", file, m[2])
				}
				modList = append(modList, statEntryFromOT(m[1], v))
			}
		}
	}
	return modList, nil
}

func buildMinions(x *Ctx) (schema.Document, error) {
	itemClassMap := map[string]string{
		"Claw":                     "Claw",
		"Dagger":                   "Dagger",
		"Wand":                     "Wand",
		"One Hand Sword":           "One Handed Sword",
		"Thrusting One Hand Sword": "One Handed Sword",
		"One Hand Axe":             "One Handed Axe",
		"One Hand Mace":            "One Handed Mace",
		"Bow":                      "Bow",
		"Fishing Rod":              "Fishing Rod",
		"Staff":                    "Staff",
		"Two Hand Sword":           "Two Handed Sword",
		"Two Hand Axe":             "Two Handed Axe",
		"Two Hand Mace":            "Two Handed Mace",
		"Shield":                   "Shield",
		"Sceptre":                  "One Handed Mace",
		"Unarmed":                  "None",
	}

	type minionState struct {
		varietyId, name, limit, hostile string
		extraSkillList                  []string
		extraModList                    []json.RawMessage
	}
	state := &minionState{}
	var defs *[]schema.MinionDef

	monsterVarieties, err := x.Dat("MonsterVarieties")
	if err != nil {
		return nil, err
	}

	// monster opens a definition; the name defaults to the variety id.
	monster := func(variety, name string, skills []string) {
		*state = minionState{varietyId: variety, name: name, extraSkillList: skills}
		if name == "" {
			state.name = variety
		}
	}
	emit := func() error {
		mv := monsterVarieties.GetRow("Id", state.varietyId)
		if mv == nil {
			// The Lua prints "Invalid Variety"; keep the emit sequence aligned.
			*defs = append(*defs, schema.MinionDef{Skip: true})
			return nil
		}
		typ := mv.Ref("Type")
		d := schema.MinionDef{
			Key:  state.name,
			Name: mv.Str("Name"),
		}
		for _, tag := range mv.Refs("Tags") {
			d.MonsterTags = append(d.MonsterTags, tag.Str("Id"))
		}
		d.BaseDamageIgnoresAttackSpeed = typ.Bool("BaseDamageIgnoresAttackSpeed")
		d.Life = float64(mv.Int("LifeMultiplier")) / 100
		if typ.Bool("AltLife1") {
			d.LifeScaling = append(d.LifeScaling, "AltLife1")
		}
		if typ.Bool("AltLife2") {
			d.LifeScaling = append(d.LifeScaling, "AltLife2")
		}
		if es := typ.Int("EnergyShield"); es != 0 {
			v := 0.4 * float64(es) / 100
			d.EnergyShield = &v
		}
		if ar := typ.Int("Armour"); ar != 0 {
			v := float64(ar) / 100
			d.Armour = &v
		}
		if ev := typ.Int("Evasion"); ev != 0 {
			v := float64(ev) / 100
			d.Evasion = &v
		}
		res := typ.Ref("Resistances")
		d.FireResist = res.Int("FireMerciless")
		d.ColdResist = res.Int("ColdMerciless")
		d.LightningResist = res.Int("LightningMerciless")
		d.ChaosResist = res.Int("ChaosMerciless")
		d.Damage = float64(mv.Int("DamageMultiplier")) / 100
		d.DamageSpread = float64(typ.Int("DamageSpread")) / 100
		d.AttackTime = float64(mv.Int("AttackDuration")) / 1000
		d.AttackRange = mv.Int("MaximumAttackRange")
		d.Accuracy = float64(typ.Int("Accuracy")) / 100
		for _, mod := range mv.Refs("Mods") {
			switch mod.Str("Id") {
			case "MonsterSpeedAndDamageFixupSmall":
				d.DamageFixups = append(d.DamageFixups, 0.11)
			case "MonsterSpeedAndDamageFixupLarge":
				d.DamageFixups = append(d.DamageFixups, 0.22)
			case "MonsterSpeedAndDamageFixupComplete":
				d.DamageFixups = append(d.DamageFixups, 0.33)
			}
		}
		if mh := mv.Ref("MainHandItemClass"); mh != nil {
			if mapped, found := itemClassMap[mh.Str("Id")]; found {
				d.WeaponType1 = &mapped
			}
		}
		if oh := mv.Ref("OffHandItemClass"); oh != nil {
			if mapped, found := itemClassMap[oh.Str("Id")]; found {
				d.WeaponType2 = &mapped
			}
		}
		d.Limit = state.limit
		d.Hostile = state.hostile
		for _, ge := range mv.Refs("GrantedEffects") {
			d.SkillList = append(d.SkillList, ge.Str("Id"))
		}
		d.SkillList = append(d.SkillList, state.extraSkillList...)

		var modList []statEntry
		for _, mod := range mv.Refs("Mods") {
			modList = append(modList, statEntryFromMod(mod))
		}
		for _, mod := range mv.Refs("SpecialMods") {
			modList = append(modList, statEntryFromMod(mod))
		}
		if objType := mv.Str("ObjectType"); objType != "" && objType != "Metadata/Monsters/Monster" {
			if modList, err = x.getOTStats(objType, modList); err != nil {
				return err
			}
		}
		for _, entry := range modList {
			for _, stat := range entry.Stats {
				statId := stat.ID
				if statId == "" {
					continue
				}
				statVal := stat.Val
				sv := statVal
				me := schema.ModEntry{Entry: entry.ID, Stat: statId, StatValue: &sv}
				mapping, found := data.StatMapTable()[statId]
				if !found {
					d.ModList = append(d.ModList, me) // unmapped: a comment in the reference file
					continue
				}
				// The reference's generator takes the mapping's FIRST mod and
				// fills its value: a table value stands; otherwise the
				// mapping's fixed value, else the stat value scaled by
				// mult/div (booleans are overwritten too — faithfully).
				first := mapping.Mods[0].Mod.Clone()
				if scalarValue(first.Value) {
					if mapping.Value.Set {
						first.Value = modparser.Num(mapping.Value.V)
					} else {
						first.Value = modparser.Num(statVal * mapping.Mult.Or(1) / mapping.Div.Or(1))
					}
				}
				me.Mods = modparser.EncodeMods([]*modparser.Mod{first})
				d.ModList = append(d.ModList, me)
			}
		}
		for _, mods := range state.extraModList {
			d.ModList = append(d.ModList, schema.ModEntry{Mods: mods, Extra: true})
		}
		*defs = append(*defs, d)
		return nil
	}

	var doc schema.Minions
	for _, tf := range []struct {
		name string
		list *[]schema.MinionDef
	}{{"Spectres", &doc.Spectres}, {"Minions", &doc.Minions}} {
		defs = tf.list
		tpl, err := readTemplate("Minions/", tf.name, minionDirectives)
		if err != nil {
			return nil, err
		}
		for _, d := range tpl.Directives {
			switch d := d.(type) {
			case *monsterDirective:
				monster(d.Variety, d.Name, d.Skills)
			case *spectreDirective:
				monster(d.Variety, d.Name, d.Skills)
				err = emit()
			case *limitDirective:
				state.limit = d.Name
			case *hostileDirective:
				if d.Value {
					state.hostile = "true"
				}
			case *extraSkillDirective:
				state.extraSkillList = append(state.extraSkillList, d.Name)
			case *modDirective:
				state.extraModList = append(state.extraModList, d.Mods)
			case *emitDirective:
				err = emit()
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return doc, nil
}

// scalarValue reports a value the generator overwrites (a record stands).
func scalarValue(v modparser.Value) bool {
	switch v.(type) {
	case nil, modparser.Num, modparser.Bool, modparser.Str:
		return true
	}
	return false
}
