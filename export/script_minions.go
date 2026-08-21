// Port of .archive/src/Export/Scripts/minions.lua.

package export

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func init() {
	Scripts = append(Scripts, Script{Name: "minions", Outs: []string{"Data/Spectres.lua", "Data/Minions.lua"}, Run: scriptMinions})
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

func scriptMinions(x *Ctx) error {
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

	directives := map[string]func(args string, out *OutFile){}
	directives["monster"] = func(args string, out *OutFile) {
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
	directives["limit"] = func(args string, out *OutFile) { state.limit = args }
	directives["hostile"] = func(args string, out *OutFile) { state.hostile = args }
	directives["mod"] = func(args string, out *OutFile) { state.extraModList = append(state.extraModList, args) }
	directives["skill"] = func(args string, out *OutFile) { state.extraSkillList = append(state.extraSkillList, args) }
	directives["emit"] = func(args string, out *OutFile) {
		mv := x.Dat("MonsterVarieties").GetRow("Id", state.varietyId)
		if mv == nil {
			return // the Lua prints "Invalid Variety"
		}
		typ := mv.Get("Type").(*Row)
		out.W("minions[\"", state.name, "\"] = {\n")
		out.W("\tname = \"", luaStr(mv.Get("Name")), "\",\n")
		out.W("\tmonsterTags = { ")
		for _, tag := range listRows(mv.Get("Tags")) {
			out.W("\"", luaStr(tag.Get("Id")), "\", ")
		}
		out.W("},\n")
		if typ.Get("BaseDamageIgnoresAttackSpeed").(bool) {
			out.W("\tbaseDamageIgnoresAttackSpeed = true,\n")
		}
		out.W("\tlife = ", luaNum(float64(mv.Get("LifeMultiplier").(int64))/100), ",\n")
		if typ.Get("AltLife1").(bool) {
			out.W("\tlifeScaling = \"AltLife1\",\n")
		}
		if typ.Get("AltLife2").(bool) {
			out.W("\tlifeScaling = \"AltLife2\",\n")
		}
		if es := typ.Get("EnergyShield").(int64); es != 0 {
			out.W("\tenergyShield = ", luaNum(0.4*float64(es)/100), ",\n")
		}
		if ar := typ.Get("Armour").(int64); ar != 0 {
			out.W("\tarmour = ", luaNum(float64(ar)/100), ",\n")
		}
		if ev := typ.Get("Evasion").(int64); ev != 0 {
			out.W("\tevasion = ", luaNum(float64(ev)/100), ",\n")
		}
		res := typ.Get("Resistances").(*Row)
		out.W("\tfireResist = ", res.Get("FireMerciless").(int64), ",\n")
		out.W("\tcoldResist = ", res.Get("ColdMerciless").(int64), ",\n")
		out.W("\tlightningResist = ", res.Get("LightningMerciless").(int64), ",\n")
		out.W("\tchaosResist = ", res.Get("ChaosMerciless").(int64), ",\n")
		out.W("\tdamage = ", luaNum(float64(mv.Get("DamageMultiplier").(int64))/100), ",\n")
		out.W("\tdamageSpread = ", luaNum(float64(typ.Get("DamageSpread").(int64))/100), ",\n")
		out.W("\tattackTime = ", luaNum(float64(mv.Get("AttackDuration").(int64))/1000), ",\n")
		out.W("\tattackRange = ", mv.Get("MaximumAttackRange").(int64), ",\n")
		out.W("\taccuracy = ", luaNum(float64(typ.Get("Accuracy").(int64))/100), ",\n")
		for _, mod := range listRows(mv.Get("Mods")) {
			switch luaStr(mod.Get("Id")) {
			case "MonsterSpeedAndDamageFixupSmall":
				out.W("\tdamageFixup = 0.11,\n")
			case "MonsterSpeedAndDamageFixupLarge":
				out.W("\tdamageFixup = 0.22,\n")
			case "MonsterSpeedAndDamageFixupComplete":
				out.W("\tdamageFixup = 0.33,\n")
			}
		}
		if mh, ok := mv.Get("MainHandItemClass").(*Row); ok {
			if mapped, found := itemClassMap[luaStr(mh.Get("Id"))]; found {
				out.W("\tweaponType1 = \"", mapped, "\",\n")
			}
		}
		if oh, ok := mv.Get("OffHandItemClass").(*Row); ok {
			if mapped, found := itemClassMap[luaStr(oh.Get("Id"))]; found {
				out.W("\tweaponType2 = \"", mapped, "\",\n")
			}
		}
		if state.limit != "" {
			out.W("\tlimit = \"", state.limit, "\",\n")
		}
		if state.hostile != "" {
			out.W("\thostile = ", state.hostile, ",\n")
		}
		out.W("\tskillList = {\n")
		for _, ge := range listRows(mv.Get("GrantedEffects")) {
			out.W("\t\t\"", luaStr(ge.Get("Id")), "\",\n")
		}
		for _, skill := range state.extraSkillList {
			out.W("\t\t\"", skill, "\",\n")
		}
		out.W("\t},\n")

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
		out.W("\tmodList = {\n")
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
					out.W("\t\t-- ", entryId, modStats, "\n")
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
						if d, ok := mapping["div"].(float64); ok {
							div = d
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
				out.W("\t\tmod(\"", luaStr(newMod["name"]), "\", \"", luaStr(newMod["type"]), "\", ", valueStr, ", ", flags, ", ", kwFlags)
				for j := 1; ; j++ {
					extra, ok := newMod[j].(luaTable)
					if !ok {
						break
					}
					out.W(", ", tableToString(extra, ""))
				}
				out.W("), -- ", entryId, modStats, "\n")
			}
		}
		for _, mod := range state.extraModList {
			out.W("\t\t", mod, ",\n")
		}
		out.W("\t},\n")
		out.W("}\n")
	}
	directives["spectre"] = func(args string, out *OutFile) {
		directives["monster"](args, out)
		directives["emit"]("", out)
	}

	for _, name := range []string{"Spectres", "Minions"} {
		out, err := x.ProcessTemplateFile(name, "Minions/", "../Data/", directives)
		if err != nil {
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}
