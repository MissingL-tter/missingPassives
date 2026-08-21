// Port of .archive/src/Export/Scripts/legionPassives.lua.

package export

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() {
	Scripts = append(Scripts, Script{Name: "legionPassives", Build: buildLegionPassives})
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

func buildLegionPassives(x *Ctx) (any, error) {
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

	// parseStats yields the node's sd, stats and sortedStats.
	parseStats := func(rowMap map[string]any) (sd []string, statsOut []gamedata.PassiveStat, sorted []string) {
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
		sd = lines.Lines
		ids := make([]string, 0, len(entries))
		for id := range entries {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			e := entries[id]
			ps := passiveStat(id, e.sv)
			index := e.index
			ps.Index = &index
			if e.hasOrder {
				order := e.statOrder
				ps.StatOrder = &order
			}
			statsOut = append(statsOut, ps)
		}
		sortIds := append([]string(nil), ids...)
		sort.Slice(sortIds, func(a, b int) bool {
			ea, eb := entries[sortIds[a]], entries[sortIds[b]]
			oa, ob := math.Inf(1), math.Inf(1)
			if ea.hasOrder {
				oa = ea.statOrder
			}
			if eb.hasOrder {
				ob = eb.statOrder
			}
			return oa < ob || (oa == ob && ea.index < eb.index)
		})
		return sd, statsOut, sortIds
	}

	var doc gamedata.LegionPassives
	ksCount := int64(-1)
	prng := newLuaPRNG()

	for i := 1; i <= altSkills.RowCount; i++ {
		rowMap := dumpRow(altSkills, i)
		fixLegionDatErrors(rowMap)
		node := gamedata.LegionNode{
			Id:   luaStr(rowMap["Id"]),
			Icon: luaStr(rowMap["DDSIcon"]),
			Dn:   luaStr(rowMap["Name"]),
		}
		node.Ks = intListContains(rowMap["PassiveType"], 4)
		if node.Ks {
			ksCount++
		}
		node.Not = intListContains(rowMap["PassiveType"], 3)

		node.Sd, node.Stats, node.SortedStats = parseStats(rowMap)

		if node.Id == "vaal_keystone_2_v2" { // Immortal Ambition needs to be manually added
			node.Sd = []string{
				"Energy Shield starts at zero",
				"Cannot Recharge or Regenerate Energy Shield",
				"Lose 5% of Energy Shield per second",
				"Life Leech effects are not removed when Unreserved Life is Filled",
				"Life Leech effects Recover Energy Shield instead while on Full Life",
			}
		}

		if node.Ks {
			node.Oidx = float64(ksCount * 3)
		} else {
			// #EVAL: archive parity — a LuaJIT-PRNG layout offset baked into
			// the data; deserves a deterministic layout once Go-owned.
			node.Oidx = math.Floor(prng.random() * 1e5)
		}
		doc.Nodes = append(doc.Nodes, node)
	}

	for i := 1; i <= altAdditions.RowCount; i++ {
		rowMap := dumpRow(altAdditions, i)
		fixLegionDatErrors(rowMap)
		add := gamedata.LegionAddition{Id: luaStr(rowMap["Id"])}
		// Additions have no name, so construct one from the id.
		dn := strings.ReplaceAll(luaStr(rowMap["Id"]), "_", " ")
		dn = reLeadWordSpace.ReplaceAllString(dn, "")
		dn = reLeadWordSpace.ReplaceAllString(dn, "")
		dn = reWordCap.ReplaceAllStringFunc(dn, func(m string) string {
			return strings.ToUpper(m[:1]) + m[1:]
		})
		add.Dn = dn

		add.Sd, add.Stats, add.SortedStats = parseStats(rowMap)
		doc.Additions = append(doc.Additions, add)
	}
	return doc, nil
}
