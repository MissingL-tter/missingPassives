// Port of .archive/src/Export/Scripts/legionPassives.lua. The artifact is
// named conqueredPassives.json: despite the reference name, the pool holds the
// ALTERNATE passives of every conquering jewel family — timeless (legion +
// Heroic Tragedy) AND abyss.

package export

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "legionPassives", OutName: "conqueredPassives", Build: buildLegionPassives})
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

// legionId is the row's Id with legionDatErrors applied.
func legionId(row *Row) string {
	id := row.Str("Id")
	fix, ok := legionDatErrors[id]
	if !ok || row.Str(fix.matchField) != fix.matchValue {
		return id
	}
	return fix.replaceId
}

func intListContains(list []int64, val int64) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

func buildLegionPassives(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("passive_skill_stat_descriptions.txt")

	altSkills, err := x.Dat("AlternatePassiveSkills")
	if err != nil {
		return nil, err
	}
	conqueredAdditions, err := x.Dat("AlternatePassiveAdditions")
	if err != nil {
		return nil, err
	}

	// parseStats yields the node's sd, stats and sortedStats.
	parseStats := func(row *Row) (sd []string, statsOut []schema.PassiveStat, sorted []string, err error) {
		type entry struct {
			sv        *statVal
			index     int
			statOrder float64
			hasOrder  bool
		}
		entries := map[string]*entry{}
		statMap := map[string]*statVal{}
		for idx, statKey := range row.Refs("StatsKeys") {
			statId := statKey.Str("Id")
			rangeV := row.Ivl(fmt.Sprintf("Stat%d", idx+1))
			sv := &statVal{min: float64(rangeV[0]), max: float64(rangeV[1])}
			// describeStats changes values while formatting them, so use a
			// copy when only finding the order.
			orderProbe, err := x.DescribeStats(map[string]*statVal{statId: {min: sv.min, max: sv.max}})
			if err != nil {
				return nil, nil, nil, err
			}
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
		lines, err := x.DescribeStats(statMap)
		if err != nil {
			return nil, nil, nil, err
		}
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
		return sd, statsOut, sortIds, nil
	}

	var doc schema.LegionPassives
	for row := range altSkills.Rows() {
		node := schema.LegionNode{
			Id:   legionId(row),
			Icon: row.Str("DDSIcon"),
			Dn:   row.Str("Name"),
		}
		passiveType := row.Ints("PassiveType")
		node.Ks = intListContains(passiveType, 4)
		node.Not = intListContains(passiveType, 3)

		node.Sd, node.Stats, node.SortedStats, err = parseStats(row)
		if err != nil {
			return nil, err
		}

		if node.Id == "vaal_keystone_2_v2" { // Immortal Ambition needs to be manually added
			node.Sd = []string{
				"Energy Shield starts at zero",
				"Cannot Recharge or Regenerate Energy Shield",
				"Lose 5% of Energy Shield per second",
				"Life Leech effects are not removed when Unreserved Life is Filled",
				"Life Leech effects Recover Energy Shield instead while on Full Life",
			}
		}

		doc.Nodes = append(doc.Nodes, node)
	}

	for row := range conqueredAdditions.Rows() {
		id := legionId(row)
		add := schema.ConqueredAddition{Id: id}
		// Additions have no name, so construct one from the id.
		dn := strings.ReplaceAll(id, "_", " ")
		dn = reLeadWordSpace.ReplaceAllString(dn, "")
		dn = reLeadWordSpace.ReplaceAllString(dn, "")
		dn = reWordCap.ReplaceAllStringFunc(dn, func(m string) string {
			return strings.ToUpper(m[:1]) + m[1:]
		})
		add.Dn = dn

		add.Sd, add.Stats, add.SortedStats, err = parseStats(row)
		if err != nil {
			return nil, err
		}
		doc.Additions = append(doc.Additions, add)
	}
	return doc, nil
}
