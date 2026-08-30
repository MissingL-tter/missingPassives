// Port of .archive/src/Export/Scripts/tattooPassives.lua.

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

func buildTattooPassives(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("stat_descriptions.txt")
	x.LoadStatFile("passive_skill_stat_descriptions.txt")

	var overridesDat, tattoosDat, clientStrings, baseItemTypes, currencyExchange *DatFile
	for name, dst := range map[string]**DatFile{
		"PassiveSkillOverrides": &overridesDat,
		"PassiveSkillTattoos":   &tattoosDat,
		"ClientStrings":         &clientStrings,
		"BaseItemTypes":         &baseItemTypes,
		"CurrencyExchange":      &currencyExchange,
	} {
		var err error
		if *dst, err = x.Dat(name); err != nil {
			return nil, err
		}
	}

	// sortSd sorts a node's sd lines by their description order.
	sortSd := func(sd []string, descOrders map[string]float64) {
		sort.Slice(sd, func(a, b int) bool { return descOrders[sd[a]] < descOrders[sd[b]] })
	}

	parsePassiveStats := func(row *Row) (nodeStats []schema.PassiveStat, sd []string, err error) {
		descOrders := map[string]float64{}
		for idx, statKey := range row.Refs("Stats") {
			statId := statKey.Str("Id")
			rangeV := float64(row.Int(fmt.Sprintf("Stat%d", idx+1)))
			sv := &statVal{min: rangeV, max: rangeV}
			lines, err := x.DescribeStats(map[string]*statVal{statId: sv})
			if err != nil {
				return nil, nil, err
			}
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
		return nodeStats, sd, nil
	}

	parseStats := func(row *Row) (nodeStats []schema.PassiveStat, sd []string, err error) {
		descOrders := map[string]float64{}
		statMap := map[string]*statVal{}
		values := row.Ints("StatValues")
		for idx, statKey := range row.Refs("StatsKeys") {
			v := float64(values[idx])
			statMap[statKey.Str("Id")] = &statVal{min: v, max: v}
		}
		lines, err := x.DescribeStats(statMap)
		if err != nil {
			return nil, nil, err
		}
		for id, sv := range statMap {
			nodeStats = append(nodeStats, passiveStat(id, sv))
		}
		sortStatsById(nodeStats)
		for i, line := range lines.Lines {
			sd = append(sd, line)
			descOrders[line] = lines.Orders[i]
		}
		sortSd(sd, descOrders)
		return nodeStats, sd, nil
	}

	var doc schema.TattooPassives

	tattooDatRows := map[string]*Row{}
	for row := range tattoosDat.Rows() {
		tattooDatRows[row.Ref("Override").Str("Id")] = row
	}

	for row := range overridesDat.Rows() {
		id := row.Str("Id")

		tattooDatRow := tattooDatRows[id]
		if tattooDatRow == nil {
			tattooDatRow = tattooDatRows["DisplayRandomKeystone"]
		}
		node := schema.TattooNode{Id: id}

		overrideType := row.Ref("OverrideType").Str("Id")
		node.OverrideType = overrideType
		nodeTarget := tattooDatRow.Ref("NodeTarget")
		targetType := nodeTarget.Str("Type")
		node.Not = targetType == "Notable"
		node.M = overrideType == "AlternateMastery"
		node.TargetType = targetType
		node.TargetValue = nodeTarget.Str("Value")

		minConn := row.Int("MinimumConnected")
		maxConn := row.Int("MaximumConnected")
		if minConn > 0 {
			text := clientStrings.RowByStr("Id", "PassiveSkillTattooAdjacentRequirementLower").Str("Text")
			reminder := strings.ReplaceAll(text, "{}", strconv.FormatInt(minConn, 10))
			node.ReminderText = &reminder
		}
		node.MinimumConnected = minConn
		if maxConn > 0 {
			text := clientStrings.RowByStr("Id", "PassiveSkillTattooAdjacentRequirementUpper").Str("Text")
			reminder := strings.ReplaceAll(text, "{}", strconv.FormatInt(maxConn, 10))
			node.ReminderText = &reminder
		}
		if maxConn > 0 {
			node.MaximumConnected = maxConn
		} else {
			node.MaximumConnected = 100
		}

		var limitText string
		var haveLimit bool
		if limit := row.Ref("Limit"); limit != nil {
			limitText = strings.ReplaceAll(
				clientStrings.RowByStr("Id", "PassiveSkillTattooLimitReminder").Str("Text"),
				"{0}", limit.Str("Description"))
			haveLimit = true
		}

		node.ActiveEffectImage = row.Str("Background") + ".png"

		// After this switch the Lua rebinds datFileRow for keystones; these
		// carry the post-switch reads.
		var finalName, finalIcon, finalId string
		var sd []string
		var err error
		if overrideType == "KeystoneTattoo" {
			node.Ks = true
			ps := row.Ref("PassiveSkill")
			node.Stats, sd, err = parsePassiveStats(ps)
			finalName = ps.Str("Name")
			finalIcon = ps.Str("Icon")
			finalId = ps.Str("Id")
		} else {
			node.Stats, sd, err = parseStats(row)
			finalName = row.Str("Name")
			finalIcon = row.Str("Icon")
			finalId = id
		}
		if err != nil {
			return nil, err
		}

		node.Name = finalName
		if finalName != "" && !node.Ks {
			if bit := baseItemTypes.RowByStr("Name", finalName); bit != nil {
				if ce := currencyExchange.RowByRef("BaseItemType", bit); ce != nil {
					legacy := !ce.Bool("EnabledInLeague")
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
