// Port of .archive/src/Export/Scripts/minions.lua.

package export

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "minions", Build: buildMinions})
}

// tableToString ports minions.lua's tableToString: sorted keys, nested
// tables inlined with a dotted prefix.
// #EVAL: archive parity — the comma logic skips nested-table entries, so a
// nested table directly abuts its neighbour without a separator.
func tableToString(tbl luaTable, pre string) string {
	s := "{ "
	keys := make([]any, 0, len(tbl))
	for k := range tbl {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if ka, ok := keys[a].(string); ok {
			if kb, ok2 := keys[b].(string); ok2 {
				return ka < kb
			}
			panic("tableToString: mixed key types")
		}
		return keyNum(keys[a]) < keyNum(keys[b])
	})
	for i, k := range keys {
		var name string
		if ks, ok := k.(string); ok {
			name = ks
		} else {
			name = luaNum(keyNum(k))
		}
		v := tbl[k]
		if sub, ok := v.(luaTable); ok {
			s += tableToString(sub, pre+name+".")
		} else {
			if i > 0 {
				s += ", "
			}
			quote := ""
			if _, ok := v.(string); ok {
				quote = "\""
			}
			s += pre + name + " = " + quote + luaStrAny(v) + quote
		}
	}
	return s + " }"
}

// luaStrAny is tostring() over the value kinds tableToString meets.
func luaStrAny(v any) string {
	switch t := v.(type) {
	case float64:
		return luaNum(t)
	default:
		return luaStr(v)
	}
}

// otMod is a stat entry parsed from an .ot file: it stands in for a Mods row
// in the emit loop (Id, Stat1 = {Id}, Stat1Value = {value}).
type otMod struct {
	id    string
	value float64
}

var (
	reExtends = regexp.MustCompile(`extends "(.+)"`)
	reKeyVal  = regexp.MustCompile(`^(.*?)=(.+)$`)
)

var otWs = strings.NewReplacer(" ", "", "\t", "", "\v", "", "\f", "", "\r", "")

// getOTStats ports minions.lua's getOTStats: collects Stats-block entries
// from an .ot file and its superclasses.
func (x *Ctx) getOTStats(otFile string, modList []any) []any {
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
		return modList
	}
	inWantedBlock := false
	for _, line := range reLine.FindAllString(text, -1) {
		if m := reExtends.FindStringSubmatch(line); m != nil && m[1] != "Metadata/Monsters/Monster" && m[1] != "nothing" {
			modList = x.getOTStats(m[1], modList)
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
					panic("getOTStats: non-numeric stat value " + m[2])
				}
				modList = append(modList, &otMod{id: m[1], value: v})
			}
		}
	}
	return modList
}

// WalkTemplate reads an in-repo template document (export/templates/) and
// calls the handler for each #directive line (the build-side half of
// processTemplateFile).
func (x *Ctx) WalkTemplate(name, inDir string, directives map[string]func(args string)) error {
	doc, err := readTemplate(inDir, name)
	if err != nil {
		return err
	}
	for _, line := range doc.Directives {
		if m := reDirective.FindStringSubmatch(line); m != nil {
			if fn := directives[m[1]]; fn != nil {
				fn(m[2])
			}
		}
	}
	return nil
}

func buildMinions(x *Ctx) (any, error) {
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
		extraModList, extraSkillList    []string
	}
	state := &minionState{}
	var defs *[]schema.MinionDef

	directives := map[string]func(args string){}
	directives["monster"] = func(args string) {
		*state = minionState{}
		for _, arg := range strings.Fields(args) {
			if state.varietyId == "" {
				state.varietyId = arg
			} else if state.name == "" {
				if arg == "#" {
					state.name = state.varietyId
				} else {
					state.name = arg
				}
			} else {
				state.extraSkillList = append(state.extraSkillList, arg)
			}
		}
		if state.varietyId == "" {
			state.varietyId = args
		}
		if state.name == "" {
			state.name = args
		}
	}
	directives["limit"] = func(args string) { state.limit = args }
	directives["hostile"] = func(args string) { state.hostile = args }
	directives["mod"] = func(args string) { state.extraModList = append(state.extraModList, args) }
	directives["skill"] = func(args string) { state.extraSkillList = append(state.extraSkillList, args) }
	directives["emit"] = func(string) {
		mv := x.Dat("MonsterVarieties").GetRow("Id", state.varietyId)
		if mv == nil {
			// The Lua prints "Invalid Variety"; keep the emit sequence aligned.
			*defs = append(*defs, schema.MinionDef{Skip: true})
			return
		}
		typ := mv.Get("Type").(*Row)
		d := schema.MinionDef{
			Key:  state.name,
			Name: luaStr(mv.Get("Name")),
		}
		for _, tag := range listRows(mv.Get("Tags")) {
			d.MonsterTags = append(d.MonsterTags, luaStr(tag.Get("Id")))
		}
		d.BaseDamageIgnoresAttackSpeed = typ.Get("BaseDamageIgnoresAttackSpeed").(bool)
		d.Life = float64(mv.Get("LifeMultiplier").(int64)) / 100
		if typ.Get("AltLife1").(bool) {
			d.LifeScaling = append(d.LifeScaling, "AltLife1")
		}
		if typ.Get("AltLife2").(bool) {
			d.LifeScaling = append(d.LifeScaling, "AltLife2")
		}
		if es := typ.Get("EnergyShield").(int64); es != 0 {
			v := 0.4 * float64(es) / 100
			d.EnergyShield = &v
		}
		if ar := typ.Get("Armour").(int64); ar != 0 {
			v := float64(ar) / 100
			d.Armour = &v
		}
		if ev := typ.Get("Evasion").(int64); ev != 0 {
			v := float64(ev) / 100
			d.Evasion = &v
		}
		res := typ.Get("Resistances").(*Row)
		d.FireResist = res.Get("FireMerciless").(int64)
		d.ColdResist = res.Get("ColdMerciless").(int64)
		d.LightningResist = res.Get("LightningMerciless").(int64)
		d.ChaosResist = res.Get("ChaosMerciless").(int64)
		d.Damage = float64(mv.Get("DamageMultiplier").(int64)) / 100
		d.DamageSpread = float64(typ.Get("DamageSpread").(int64)) / 100
		d.AttackTime = float64(mv.Get("AttackDuration").(int64)) / 1000
		d.AttackRange = mv.Get("MaximumAttackRange").(int64)
		d.Accuracy = float64(typ.Get("Accuracy").(int64)) / 100
		for _, mod := range listRows(mv.Get("Mods")) {
			switch luaStr(mod.Get("Id")) {
			case "MonsterSpeedAndDamageFixupSmall":
				d.DamageFixups = append(d.DamageFixups, 0.11)
			case "MonsterSpeedAndDamageFixupLarge":
				d.DamageFixups = append(d.DamageFixups, 0.22)
			case "MonsterSpeedAndDamageFixupComplete":
				d.DamageFixups = append(d.DamageFixups, 0.33)
			}
		}
		if mh, ok := mv.Get("MainHandItemClass").(*Row); ok {
			if mapped, found := itemClassMap[luaStr(mh.Get("Id"))]; found {
				d.WeaponType1 = &mapped
			}
		}
		if oh, ok := mv.Get("OffHandItemClass").(*Row); ok {
			if mapped, found := itemClassMap[luaStr(oh.Get("Id"))]; found {
				d.WeaponType2 = &mapped
			}
		}
		d.Limit = state.limit
		d.Hostile = state.hostile
		for _, ge := range listRows(mv.Get("GrantedEffects")) {
			d.SkillList = append(d.SkillList, luaStr(ge.Get("Id")))
		}
		d.SkillList = append(d.SkillList, state.extraSkillList...)

		var modList []any
		for _, mod := range listRows(mv.Get("Mods")) {
			modList = append(modList, mod)
		}
		for _, mod := range listRows(mv.Get("SpecialMods")) {
			modList = append(modList, mod)
		}
		if objType := luaStr(mv.Get("ObjectType")); objType != "" && objType != "Metadata/Monsters/Monster" {
			modList = x.getOTStats(objType, modList)
		}
		for _, entry := range modList {
			// modStatX reads Stat<i> / Stat<i>Value off either a Mods row or
			// an .ot stat entry.
			var statIds []string
			var statVals []float64
			var entryId string
			switch e := entry.(type) {
			case *Row:
				entryId = luaStr(e.Get("Id"))
				for i := 1; i <= 6; i++ {
					if sr, ok := e.Get(fmt.Sprintf("Stat%d", i)).(*Row); ok {
						statIds = append(statIds, luaStr(sr.Get("Id")))
						statVals = append(statVals, float64(e.Get(fmt.Sprintf("Stat%dValue", i)).(Interval)[0]))
					} else {
						statIds = append(statIds, "")
						statVals = append(statVals, 0)
					}
				}
			case *otMod:
				entryId = e.id
				statIds = []string{e.id}
				statVals = []float64{e.value}
			}
			for i, statId := range statIds {
				if statId == "" {
					continue
				}
				statVal := statVals[i]
				modStats := " [" + statId + " = " + luaNum(statVal) + "]"
				mapping, found := skillStatMap[statId]
				if !found {
					d.ModList = append(d.ModList, "-- "+entryId+modStats)
					continue
				}
				newMod := mapping[1].(luaTable)
				var valueStr string
				nv, hasNV := newMod["value"]
				if hasNV {
					if _, isBool := nv.(bool); !isBool {
						valueStr = tableToString(nv.(luaTable), "")
					}
				}
				if valueStr == "" {
					if ev, hasEV := mapping["value"]; hasEV {
						valueStr = luaStrAny(ev)
					} else {
						mult := 1.0
						if m, ok := mapping["mult"].(float64); ok {
							mult = m
						}
						div := 1.0
						if dv, ok := mapping["div"].(float64); ok {
							div = dv
						}
						valueStr = luaNum(statVal * mult / div)
					}
				}
				flags := "0"
				if f, ok := newMod["flags"].(float64); ok {
					flags = luaNum(f)
				}
				kwFlags := "0"
				if f, ok := newMod["keywordFlags"].(float64); ok {
					kwFlags = luaNum(f)
				}
				line := "mod(\"" + luaStr(newMod["name"]) + "\", \"" + luaStr(newMod["type"]) + "\", " + valueStr + ", " + flags + ", " + kwFlags
				for j := 1; ; j++ {
					extra, ok := newMod[j].(luaTable)
					if !ok {
						break
					}
					line += ", " + tableToString(extra, "")
				}
				line += "), -- " + entryId + modStats
				d.ModList = append(d.ModList, line)
			}
		}
		for _, mod := range state.extraModList {
			d.ModList = append(d.ModList, mod+",")
		}
		*defs = append(*defs, d)
	}
	directives["spectre"] = func(args string) {
		directives["monster"](args)
		directives["emit"]("")
	}

	var doc schema.Minions
	for _, tf := range []struct {
		name string
		list *[]schema.MinionDef
	}{{"Spectres", &doc.Spectres}, {"Minions", &doc.Minions}} {
		defs = tf.list
		if err := x.WalkTemplate(tf.name, "Minions/", directives); err != nil {
			return nil, err
		}
	}
	return doc, nil
}
