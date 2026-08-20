// Port of .archive/src/Export/Scripts/tattooPassives.lua, including its
// private stringify() serializer.

package export

import (
	"regexp"
	"sort"
	"strings"
)

func init() {
	Scripts = append(Scripts, Script{Name: "tattooPassives", Outs: []string{"Data/TattooPassives.lua"}, Run: scriptTattooPassives})
}

var reDotDdsEnd = regexp.MustCompile(`\.dds$`)

// stringifyTattoo ports the script's stringify(): sorted keys, "\n\t"-led
// entries, trailing ", " after every value, nested tables re-indented by one
// tab, no string escaping.
func stringifyTattoo(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return luaStr(t)
	case int:
		return luaStr(t)
	case int64:
		return luaStr(t)
	case float64:
		return luaNum(t)
	case luaTable:
		var b strings.Builder
		b.WriteString("{")
		keys := make([]any, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(a, b int) bool {
			if ka, ok := keys[a].(string); ok {
				return ka < keys[b].(string)
			}
			return keyNum(keys[a]) < keyNum(keys[b])
		})
		for _, k := range keys {
			b.WriteString("\n\t")
			if ks, ok := k.(string); ok {
				b.WriteString("[\"" + ks + "\"] = ")
			} else {
				b.WriteString("[" + luaNum(keyNum(k)) + "] = ")
			}
			val := t[k]
			if s, ok := val.(string); ok {
				b.WriteString("\"" + s + "\", ")
			} else if sub, ok := val.(luaTable); ok {
				b.WriteString(strings.ReplaceAll(stringifyTattoo(sub)+", ", "\n", "\n\t"))
			} else {
				b.WriteString(stringifyTattoo(val) + ", ")
			}
		}
		b.WriteString("\n}")
		return b.String()
	}
	panic("stringifyTattoo: unsupported type")
}

// statValTable converts a describeStats-mutated statVal into the Lua table
// the script serializes (min/max/fmt plus the mutation flags).
func statValTable(v *statVal) luaTable {
	t := luaTable{"min": v.min, "max": v.max}
	if v.fmt != "" {
		t["fmt"] = v.fmt
	}
	if v.minZ {
		t["minZ"] = true
	}
	if v.maxZ {
		t["maxZ"] = true
	}
	return t
}

// listRows iterates a Key-list cell the way Lua ipairs does: stopping at the
// first nil (null ref).
func listRows(cell any) []*Row {
	var out []*Row
	for _, v := range cell.([]any) {
		r, ok := v.(*Row)
		if !ok {
			break
		}
		out = append(out, r)
	}
	return out
}

func scriptTattooPassives(x *Ctx) error {
	x.LoadStatFile("stat_descriptions.txt")
	x.LoadStatFile("passive_skill_stat_descriptions.txt")

	stats := x.Dat("Stats")
	overridesDat := x.Dat("PassiveSkillOverrides")
	tattoosDat := x.Dat("PassiveSkillTattoos")
	clientStrings := x.Dat("ClientStrings")
	baseItemTypes := x.Dat("BaseItemTypes")
	currencyExchange := x.Dat("CurrencyExchange")

	// sortSd sorts a node's sd lines by their description order.
	sortSd := func(sd []string, descOrders map[string]float64) {
		sort.Slice(sd, func(a, b int) bool { return descOrders[sd[a]] < descOrders[sd[b]] })
	}

	parsePassiveStats := func(row *Row, nodeStats luaTable, sd *[]string) {
		descOrders := map[string]float64{}
		for idx, statKey := range listRows(row.Get("Stats")) {
			statId := luaStr(stats.GetRowByIndex(statKey.Index).Get("Id"))
			rangeV := float64(row.Get("Stat" + luaStr(idx+1)).(int64))
			sv := &statVal{min: rangeV, max: rangeV}
			lines := x.DescribeStats(map[string]*statVal{statId: sv})
			entry := statValTable(sv)
			entry["index"] = idx + 1
			if len(lines.Orders) > 0 {
				entry["statOrder"] = lines.Orders[0]
			}
			nodeStats[statId] = entry
			for i, line := range lines.Lines {
				*sd = append(*sd, line)
				descOrders[line] = lines.Orders[i]
			}
		}
		sortSd(*sd, descOrders)
	}

	parseStats := func(rowMap map[string]any, nodeStats luaTable, sd *[]string) {
		descOrders := map[string]float64{}
		statMap := map[string]*statVal{}
		values := rowMap["StatValues"].([]any)
		for idx, statKey := range listRows(rowMap["StatsKeys"]) {
			statId := luaStr(stats.GetRowByIndex(statKey.Index).Get("Id"))
			v := float64(values[idx].(int64))
			statMap[statId] = &statVal{min: v, max: v}
		}
		lines := x.DescribeStats(statMap)
		for id, sv := range statMap {
			nodeStats[id] = statValTable(sv)
		}
		for i, line := range lines.Lines {
			*sd = append(*sd, line)
			descOrders[line] = lines.Orders[i]
		}
		sortSd(*sd, descOrders)
	}

	dumpRow := func(d *DatFile, i int) map[string]any {
		m := map[string]any{}
		for j, col := range d.spec {
			m[col.Name] = d.readCell(i, j)
		}
		return m
	}

	data := luaTable{}
	nodes := luaTable{}
	data["nodes"] = nodes

	tattooDatRows := map[string]map[string]any{}
	for i := 1; i <= tattoosDat.RowCount; i++ {
		m := dumpRow(tattoosDat, i)
		tattooDatRows[luaStr(m["Override"].(*Row).Get("Id"))] = m
	}

	for i := 1; i <= overridesDat.RowCount; i++ {
		rowMap := dumpRow(overridesDat, i)
		id := luaStr(rowMap["Id"])

		tattooDatRow := tattooDatRows[id]
		if tattooDatRow == nil {
			tattooDatRow = tattooDatRows["DisplayRandomKeystone"]
		}
		node := luaTable{}
		node["id"] = id

		var sd []string
		nodeStats := luaTable{}
		node["stats"] = nodeStats
		node["isTattoo"] = true
		overrideType := luaStr(rowMap["OverrideType"].(*Row).Get("Id"))
		node["overrideType"] = overrideType
		node["ks"] = false
		nodeTarget := tattooDatRow["NodeTarget"].(*Row)
		targetType := luaStr(nodeTarget.Get("Type"))
		node["not"] = targetType == "Notable"
		node["m"] = overrideType == "AlternateMastery"
		node["targetType"] = targetType
		node["targetValue"] = luaStr(nodeTarget.Get("Value"))

		minConn := rowMap["MinimumConnected"].(int64)
		maxConn := rowMap["MaximumConnected"].(int64)
		if minConn > 0 {
			text := luaStr(clientStrings.GetRow("Id", "PassiveSkillTattooAdjacentRequirementLower").Get("Text"))
			node["reminderText"] = luaTable{1: strings.ReplaceAll(text, "{}", luaStr(minConn))}
		}
		node["MinimumConnected"] = minConn
		if maxConn > 0 {
			text := luaStr(clientStrings.GetRow("Id", "PassiveSkillTattooAdjacentRequirementUpper").Get("Text"))
			node["reminderText"] = luaTable{1: strings.ReplaceAll(text, "{}", luaStr(maxConn))}
		}
		if maxConn > 0 {
			node["MaximumConnected"] = maxConn
		} else {
			node["MaximumConnected"] = int64(100)
		}

		var limitText string
		var haveLimit bool
		if limit, ok := rowMap["Limit"].(*Row); ok {
			limitText = strings.ReplaceAll(
				luaStr(clientStrings.GetRow("Id", "PassiveSkillTattooLimitReminder").Get("Text")),
				"{0}", luaStr(limit.Get("Description")))
			haveLimit = true
		}

		node["activeEffectImage"] = luaStr(rowMap["Background"]) + ".png"

		// After this switch the Lua rebinds datFileRow for keystones; these
		// carry the post-switch reads.
		var finalName, finalIcon, finalId string
		if overrideType == "KeystoneTattoo" {
			node["ks"] = true
			ps := rowMap["PassiveSkill"].(*Row)
			parsePassiveStats(ps, nodeStats, &sd)
			finalName = luaStr(ps.Get("Name"))
			finalIcon = luaStr(ps.Get("Icon"))
			finalId = luaStr(ps.Get("Id"))
		} else {
			if overrideType == "AlternateMastery" {
				node["name"] = "Runegraft Mastery"
			}
			parseStats(rowMap, nodeStats, &sd)
			finalName = luaStr(rowMap["Name"])
			finalIcon = luaStr(rowMap["Icon"])
			finalId = id
		}

		node["dn"] = finalName
		if finalName != "" && node["ks"] == false {
			if bit := baseItemTypes.GetRow("Name", finalName); bit != nil {
				if ce := currencyExchange.GetRow("BaseItemType", bit); ce != nil {
					node["legacy"] = !ce.Get("EnabledInLeague").(bool)
				}
			}
		}

		node["icon"] = reDotDdsEnd.ReplaceAllString(finalIcon, ".png")
		if haveLimit {
			sd = append(sd, limitText)
		}
		sdTable := luaTable{}
		for i, line := range sd {
			sdTable[i+1] = line
		}
		node["sd"] = sdTable

		if finalId != "DisplayRandomKeystone" && !strings.Contains(finalName, "DNT") && !strings.Contains(finalName, "of the Test") {
			nodes[finalName] = node
		}
	}

	data["groups"] = luaTable{
		float64(1e9): luaTable{
			"x":  float64(-6500),
			"y":  float64(-6500),
			"oo": luaTable{},
			"n":  luaTable{},
		},
	}

	out := x.Out("Data/TattooPassives.lua")
	out.W("-- This file is automatically generated, do not edit!\n-- Item data (c) Grinding Gear Games\n\n")
	out.W("return " + stringifyTattoo(data))
	return out.Close()
}
