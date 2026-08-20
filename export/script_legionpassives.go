// Port of .archive/src/Export/Scripts/legionPassives.lua.

package export

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

func init() {
	Scripts = append(Scripts, Script{Name: "legionPassives", Outs: []string{"Data/TimelessJewelData/LegionPassives.lua"}, Run: scriptLegionPassives})
}

var (
	reLeadWordSpace = regexp.MustCompile(`^[0-9A-Za-z]* `)
	reWordCap       = regexp.MustCompile(`([a-z])([0-9A-Za-z]*)`)
)

// legionDatErrors lists errors in the ggpk dat files, as in the Lua.
var legionDatErrors = map[string]struct {
	matchField, matchValue, replaceId string
}{
	"templar_notable_minimum_frenzy_charge":    {"Name", "Powerful Faith", "templar_notable_minimum_power_charge"},
	"templar_notable_minimum_power_charge":     {"Name", "Frenzied Faith", "templar_notable_minimum_frenzy_charge"},
	"karui_notable_add_physical_taken_as_fire": {"Id", "karui_notable_add_physical_taken_as_fire", "karui_notable_add_rage_on_melee_hit"},
	"karui_notable_add_faster_burn":            {"Id", "karui_notable_add_faster_burn", "karui_notable_add_faster_ignite"},
	"maraketh_notable_add_ailment_avoid":       {"Id", "maraketh_notable_add_ailment_avoid", "maraketh_notable_add_stun_avoid"},
	"maraketh_notable_add_flask_effect":        {"Id", "maraketh_notable_add_flask_effect", "maraketh_notable_add_alchemists_genius"},
}

func fixLegionDatErrors(row map[string]any) {
	fix, ok := legionDatErrors[luaStr(row["Id"])]
	if !ok {
		return
	}
	if luaStr(row[fix.matchField]) != fix.matchValue {
		return
	}
	row["Id"] = fix.replaceId
}

func intListContains(cell any, val int64) bool {
	for _, v := range cell.([]any) {
		if n, ok := v.(int64); ok && n == val {
			return true
		}
	}
	return false
}

func scriptLegionPassives(x *Ctx) error {
	x.LoadStatFile("passive_skill_stat_descriptions.txt")

	stats := x.Dat("Stats")
	altSkills := x.Dat("AlternatePassiveSkills")
	altAdditions := x.Dat("AlternatePassiveAdditions")

	dumpRow := func(d *DatFile, i int) map[string]any {
		m := map[string]any{}
		for j, col := range d.spec {
			m[col.Name] = d.readCell(i, j)
		}
		return m
	}

	// parseStats fills the node's stats/sd/sortedStats.
	parseStats := func(rowMap map[string]any, node luaTable) {
		type entry struct {
			sv        *statVal
			index     int
			statOrder float64
			hasOrder  bool
		}
		entries := map[string]*entry{}
		statMap := map[string]*statVal{}
		for idx, statKey := range listRows(rowMap["StatsKeys"]) {
			statId := luaStr(stats.GetRowByIndex(statKey.Index).Get("Id"))
			rangeV := rowMap["Stat"+luaStr(idx+1)].(Interval)
			sv := &statVal{min: float64(rangeV[0]), max: float64(rangeV[1])}
			// describeStats changes values while formatting them, so use a
			// copy when only finding the order.
			orderProbe := x.DescribeStats(map[string]*statVal{statId: {min: sv.min, max: sv.max}})
			e := &entry{sv: sv, index: idx + 1}
			if len(orderProbe.Orders) > 0 {
				e.statOrder = orderProbe.Orders[0]
				e.hasOrder = true
			}
			entries[statId] = e
			statMap[statId] = sv
		}
		// A description can combine several stats, such as minimum and
		// maximum added damage, so describe the complete set together.
		lines := x.DescribeStats(statMap)
		sd := luaTable{}
		for i, line := range lines.Lines {
			sd[i+1] = line
		}
		node["sd"] = sd
		nodeStats := luaTable{}
		for id, e := range entries {
			t := statValTable(e.sv)
			t["index"] = e.index
			if e.hasOrder {
				t["statOrder"] = e.statOrder
			}
			nodeStats[id] = t
		}
		node["stats"] = nodeStats
		ids := make([]string, 0, len(entries))
		for id := range entries {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		sort.Slice(ids, func(a, b int) bool {
			ea, eb := entries[ids[a]], entries[ids[b]]
			oa, ob := math.Inf(1), math.Inf(1)
			if ea.hasOrder {
				oa = ea.statOrder
			}
			if eb.hasOrder {
				ob = eb.statOrder
			}
			return oa < ob || (oa == ob && ea.index < eb.index)
		})
		sortedStats := luaTable{}
		for i, id := range ids {
			sortedStats[i+1] = id
		}
		node["sortedStats"] = sortedStats
	}

	data := luaTable{}
	nodes := luaTable{}
	groups := luaTable{}
	additions := luaTable{}
	data["nodes"] = nodes
	data["groups"] = groups
	data["additions"] = additions
	ksCount := int64(-1)
	prng := newLuaPRNG()

	for i := 1; i <= altSkills.RowCount; i++ {
		rowMap := dumpRow(altSkills, i)
		fixLegionDatErrors(rowMap)
		node := luaTable{}
		node["id"] = luaStr(rowMap["Id"])
		node["icon"] = luaStr(rowMap["DDSIcon"])
		ks := intListContains(rowMap["PassiveType"], 4)
		node["ks"] = ks
		if ks {
			ksCount++
		}
		node["not"] = intListContains(rowMap["PassiveType"], 3)
		node["dn"] = luaStr(rowMap["Name"])
		node["m"] = false
		node["isJewelSocket"] = false
		node["isMultipleChoice"] = false
		node["isMultipleChoiceOption"] = false
		node["passivePointsGranted"] = 0
		node["spc"] = luaTable{}

		parseStats(rowMap, node)

		if node["id"] == "vaal_keystone_2_v2" { // Immortal Ambition needs to be manually added
			node["sd"] = luaTable{
				1: "Energy Shield starts at zero",
				2: "Cannot Recharge or Regenerate Energy Shield",
				3: "Lose 5% of Energy Shield per second",
				4: "Life Leech effects are not removed when Unreserved Life is Filled",
				5: "Life Leech effects Recover Energy Shield instead while on Full Life",
			}
		}

		node["g"] = float64(1e9)
		if ks {
			node["o"] = 4
			node["oidx"] = ksCount * 3
		} else {
			node["o"] = 3
			node["oidx"] = math.Floor(prng.random() * 1e5)
		}
		node["sa"] = 0
		node["da"] = 0
		node["ia"] = 0
		node["out"] = luaTable{}
		node["in"] = luaTable{}

		nodes[i] = node
	}

	groupN := luaTable{}
	for i := 1; i <= altSkills.RowCount; i++ {
		groupN[i] = i
	}
	groups[float64(1e9)] = luaTable{
		"x":  float64(-6500),
		"y":  float64(-6500),
		"oo": luaTable{},
		"n":  groupN,
	}

	for i := 1; i <= altAdditions.RowCount; i++ {
		rowMap := dumpRow(altAdditions, i)
		fixLegionDatErrors(rowMap)
		add := luaTable{}
		add["id"] = luaStr(rowMap["Id"])
		// Additions have no name, so construct one from the id.
		dn := strings.ReplaceAll(luaStr(rowMap["Id"]), "_", " ")
		dn = reLeadWordSpace.ReplaceAllString(dn, "")
		dn = reLeadWordSpace.ReplaceAllString(dn, "")
		dn = reWordCap.ReplaceAllStringFunc(dn, func(m string) string {
			return strings.ToUpper(m[:1]) + m[1:]
		})
		add["dn"] = dn

		parseStats(rowMap, add)
		additions[i] = add
	}

	out := x.Out("Data/TimelessJewelData/LegionPassives.lua")
	out.W("-- This file is automatically generated, do not edit!\n-- Item data (c) Grinding Gear Games\n\n")
	out.W("return " + stringifyTattoo(data))
	return out.Close()
}
