// Port of .archive/src/Export/Scripts/tattooPassives.lua.

package export

import (
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "tattooPassives", Build: buildTattooPassives})
}

var reDotDdsEnd = regexp.MustCompile(`\.dds$`)

// passiveStat converts a describeStats-mutated statVal into its gamedata
// form.
func passiveStat(id string, v *statVal) schema.PassiveStat {
	return schema.PassiveStat{Id: id, Min: v.min, Max: v.max, Fmt: v.fmt, MinZ: v.minZ, MaxZ: v.maxZ}
}

// sortStatsById sorts stably by stat id, preserving insertion order for
// duplicates (the Lua's table overwrite keeps the last).
func sortStatsById(stats []schema.PassiveStat) {
	sort.SliceStable(stats, func(a, b int) bool { return stats[a].Id < stats[b].Id })
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

func buildTattooPassives(x *Ctx) (any, error) {
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

	parsePassiveStats := func(row *Row) (nodeStats []schema.PassiveStat, sd []string) {
		descOrders := map[string]float64{}
		for idx, statKey := range listRows(row.Get("Stats")) {
			statId := luaStr(stats.GetRowByIndex(statKey.Index).Get("Id"))
			rangeV := float64(row.Get("Stat" + luaStr(idx+1)).(int64))
			sv := &statVal{min: rangeV, max: rangeV}
			lines := x.DescribeStats(map[string]*statVal{statId: sv})
			entry := passiveStat(statId, sv)
			index := idx + 1
			entry.Index = &index
			if len(lines.Orders) > 0 {
				order := lines.Orders[0]
				entry.StatOrder = &order
			}
			nodeStats = append(nodeStats, entry)
			for i, line := range lines.Lines {
				sd = append(sd, line)
				descOrders[line] = lines.Orders[i]
			}
		}
		sortSd(sd, descOrders)
		sortStatsById(nodeStats)
		return nodeStats, sd
	}

	parseStats := func(rowMap map[string]any) (nodeStats []schema.PassiveStat, sd []string) {
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
			nodeStats = append(nodeStats, passiveStat(id, sv))
		}
		sortStatsById(nodeStats)
		for i, line := range lines.Lines {
			sd = append(sd, line)
			descOrders[line] = lines.Orders[i]
		}
		sortSd(sd, descOrders)
		return nodeStats, sd
	}

	dumpRow := func(d *DatFile, i int) map[string]any {
		m := map[string]any{}
		for j, col := range d.spec {
			m[col.Name] = d.readCell(i, j)
		}
		return m
	}

	var doc schema.TattooPassives

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
		node := schema.TattooNode{Id: id}

		overrideType := luaStr(rowMap["OverrideType"].(*Row).Get("Id"))
		node.OverrideType = overrideType
		nodeTarget := tattooDatRow["NodeTarget"].(*Row)
		targetType := luaStr(nodeTarget.Get("Type"))
		node.Not = targetType == "Notable"
		node.M = overrideType == "AlternateMastery"
		node.TargetType = targetType
		node.TargetValue = luaStr(nodeTarget.Get("Value"))

		minConn := rowMap["MinimumConnected"].(int64)
		maxConn := rowMap["MaximumConnected"].(int64)
		if minConn > 0 {
			text := luaStr(clientStrings.GetRow("Id", "PassiveSkillTattooAdjacentRequirementLower").Get("Text"))
			reminder := strings.ReplaceAll(text, "{}", luaStr(minConn))
			node.ReminderText = &reminder
		}
		node.MinimumConnected = minConn
		if maxConn > 0 {
			text := luaStr(clientStrings.GetRow("Id", "PassiveSkillTattooAdjacentRequirementUpper").Get("Text"))
			reminder := strings.ReplaceAll(text, "{}", luaStr(maxConn))
			node.ReminderText = &reminder
		}
		if maxConn > 0 {
			node.MaximumConnected = maxConn
		} else {
			node.MaximumConnected = 100
		}

		var limitText string
		var haveLimit bool
		if limit, ok := rowMap["Limit"].(*Row); ok {
			limitText = strings.ReplaceAll(
				luaStr(clientStrings.GetRow("Id", "PassiveSkillTattooLimitReminder").Get("Text")),
				"{0}", luaStr(limit.Get("Description")))
			haveLimit = true
		}

		node.ActiveEffectImage = luaStr(rowMap["Background"]) + ".png"

		// After this switch the Lua rebinds datFileRow for keystones; these
		// carry the post-switch reads.
		var finalName, finalIcon, finalId string
		var sd []string
		if overrideType == "KeystoneTattoo" {
			node.Ks = true
			ps := rowMap["PassiveSkill"].(*Row)
			node.Stats, sd = parsePassiveStats(ps)
			finalName = luaStr(ps.Get("Name"))
			finalIcon = luaStr(ps.Get("Icon"))
			finalId = luaStr(ps.Get("Id"))
		} else {
			node.Stats, sd = parseStats(rowMap)
			finalName = luaStr(rowMap["Name"])
			finalIcon = luaStr(rowMap["Icon"])
			finalId = id
		}

		node.Name = finalName
		if finalName != "" && !node.Ks {
			if bit := baseItemTypes.GetRow("Name", finalName); bit != nil {
				if ce := currencyExchange.GetRow("BaseItemType", bit); ce != nil {
					legacy := !ce.Get("EnabledInLeague").(bool)
					node.Legacy = &legacy
				}
			}
		}

		node.Icon = reDotDdsEnd.ReplaceAllString(finalIcon, ".png")
		if haveLimit {
			sd = append(sd, limitText)
		}
		node.Sd = sd

		if finalId != "DisplayRandomKeystone" && !strings.Contains(finalName, "DNT") && !strings.Contains(finalName, "of the Test") {
			doc.Nodes = append(doc.Nodes, node)
		}
	}
	return doc, nil
}
