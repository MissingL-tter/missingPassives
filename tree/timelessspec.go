// The timeless-conquering half of BuildAllDependsAndPaths (PassiveSpec.lua
// L1191-1376): replacing or augmenting conquered nodes from computed LUT
// records, plus NodeAdditionOrReplacementFromString and
// ReconnectNodeToClassStart.
package tree

import (
	"math"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// timelessJewelTypeByConqueror — PassiveSpec.lua L1099.
var timelessJewelTypeByConqueror = map[string]int{
	"vaal":            1,
	"karui":           2,
	"maraketh":        3,
	"templar":         4,
	"eternal":         5,
	"kalguur":         6,
	"abyss_murderous": 7,
	"abyss_searching": 8,
	"abyss_hypnotic":  9,
	"abyss_ghastly":   10,
	"abyss_special":   11,
}

func conqueredByMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func conquerorOf(cq any) map[string]any {
	m := conqueredByMap(cq)
	if m == nil {
		return nil
	}
	return conqueredByMap(m["conqueror"])
}

func conquerorType(cq any) string {
	if c := conquerorOf(cq); c != nil {
		if s, ok := c["type"].(string); ok {
			return s
		}
	}
	return ""
}

// conquerorIDStr is the conqueror id as the reference concatenates it
// (number via tostring, or the "2_v2" string forms).
func conquerorIDStr(cq any) string {
	c := conquerorOf(cq)
	if c == nil {
		return ""
	}
	if s, ok := c["id"].(string); ok {
		return s
	}
	if n, ok := anyNum(c["id"]); ok {
		return luaNumStr(n)
	}
	return ""
}

func conqueredSeed(cq any) float64 {
	m := conqueredByMap(cq)
	if m == nil {
		return 0
	}
	n, _ := anyNum(m["id"])
	return n
}

func luaNumStr(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

// luaRound is Common.lua round(val, dec).
func luaRound(val float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(val*p+0.5) / p
}

// timelessReplaceStat is BuildAllDependsAndPaths' replaceHelperFunc.
func timelessReplaceStat(stat string, desc *LegionStatDesc, value float64) string {
	if desc.Fmt == "g" {
		if strings.Contains(desc.ID, "per_minute") {
			value = luaRound(value/60, 1)
		} else if strings.Contains(desc.ID, "permyriad") {
			value = value / 100
		} else if strings.Contains(desc.ID, "_ms") {
			value = value / 1000
		}
	}
	if desc.Min != desc.Max {
		return strings.ReplaceAll(stat, "("+luaNumStr(desc.Min)+"-"+luaNumStr(desc.Max)+")", luaNumStr(value))
	} else if desc.Min != value {
		// only true for might/legacy of the vaal, which can combine stats
		return strings.ReplaceAll(stat, luaNumStr(desc.Min), luaNumStr(value))
	}
	return stat
}

// nodeAdditionOrReplacementFromString ports the same-named method: parse sd
// (splitting embedded newlines) and either replace the node's stat state or
// merge it in (new mods first in the mod list, appended in sd/mods order).
func (s *Spec) nodeAdditionOrReplacementFromString(node *SpecNode, sd string, replacement bool) {
	var add Stats
	add.Sd = []string{sd}
	processStats(&add, strconv.FormatInt(node.ID(), 10), 0)
	if replacement {
		node.Stats.Sd = add.Sd
		node.Stats.Mods = add.Mods
		node.Stats.ModKey = add.ModKey
		node.Stats.ModList = add.ModList
	} else {
		node.Stats.Sd = append(append([]string{}, node.Stats.Sd...), add.Sd...)
		node.Stats.Mods = append(append([]*NodeMod{}, node.Stats.Mods...), add.Mods...)
		node.Stats.ModKey = node.Stats.ModKey + add.ModKey
		node.Stats.ModList = append(append([]*modparser.Mod{}, add.ModList...), node.Stats.ModList...)
	}
	node.sdIdentity = nil
}

// reconnectNodeToClassStart ports ReconnectNodeToClassStart (Pure Talent).
func (s *Spec) reconnectNodeToClassStart(node *SpecNode) {
	for _, linkedID := range node.T.LinkedIDs {
		for _, classID := range sortedNodeIDs(s.Tree.Classes) {
			class := s.Tree.Classes[classID]
			if linkedID == class.StartNodeID && node.Type() == "Normal" {
				node.Stats.ModList = append(node.Stats.ModList,
					modparser.NewMod("Condition:ConnectedTo"+class.Name+"Start", "FLAG", true,
						"Tree:"+strconv.FormatInt(linkedID, 10)))
			}
		}
	}
}

// applyConquered ports the "If node is conquered, replace it or add mods"
// branch of BuildAllDependsAndPaths' second node loop.
func (s *Spec) applyConquered(node *SpecNode) {
	cq := node.ConqueredBy
	if cq == nil || node.Type() == "Socket" {
		return
	}
	legionNodes := s.Tree.Legion.NodesOrdered
	legionAdditions := s.Tree.Legion.AdditionsOrdered

	jewelType, ok := timelessJewelTypeByConqueror[conquerorType(cq)]
	if !ok {
		jewelType = 5
	}
	if jewelType >= 7 {
		modification, _ := conqueredByMap(cq)["modification"].([]AbyssComponent)
		for _, comp := range modification {
			var sd []string
			var descs []*LegionStatDesc
			replaces := false
			switch comp.Type {
			case 1:
				if legionNode := legionNode1(s.Tree.Legion.NodesOrdered, comp.ID+1-337); legionNode != nil {
					s.replaceNode(node, legionNode)
					sd = legionNode.Sd
					descs = legionNode.StatDescs
					replaces = true
				}
			case 2:
				if comp.ID >= 0 && comp.ID < len(legionAdditions) {
					addition := legionAdditions[comp.ID]
					sd = addition.Sd
					descs = addition.StatDescs
				}
			}
			if sd == nil && !replaces {
				continue // "Unhandled Abyss component ID"
			}
			for statIndex, statLine := range sd {
				for _, desc := range descs {
					if desc.Index-1 < len(comp.Rolls) {
						statLine = timelessReplaceStat(statLine, desc, float64(comp.Rolls[desc.Index-1]))
					}
				}
				prefix := " \n"
				if replaces {
					prefix = ""
				}
				s.nodeAdditionOrReplacementFromString(node, prefix+statLine, replaces && statIndex == 0)
			}
		}
		s.reconnectNodeToClassStart(node)
		return
	}
	rawSeed := conqueredSeed(cq)
	seed := rawSeed
	if jewelType == 5 {
		seed = seed / 20
	}
	seedInRange := seed >= data.TimelessJewelSeedMin[jewelType] && seed <= data.TimelessJewelSeedMax[jewelType]

	switch {
	case node.Type() == "Notable":
		var rec []int
		if seedInRange {
			rec = ReadTimelessLUT(int64(rawSeed), node.ID(), jewelType)
		}
		if len(rec) == 0 {
			// "Missing LUT" — no node change; reconnect still runs.
		} else if jewelType == 1 {
			headerSize := len(rec)
			switch {
			case headerSize == 2 || headerSize == 3:
				legionNode := legionNode1(legionNodes, rec[0]+1-337)
				s.replaceNode(node, legionNode)
				sd := legionNode.Sd // snapshot: the node's sd is rebuilt below
				for i, repStat := range sd {
					desc := legionStatDescOf(legionNode, legionNode.SortedStats[i])
					repStat = timelessReplaceStat(repStat, desc, float64(rec[desc.Index]))
					s.nodeAdditionOrReplacementFromString(node, repStat, i == 0) // wipe mods on first run
				}
			case headerSize == 6 || headerSize == 8:
				bias := 0
				for i, val := range rec {
					if i >= headerSize/2 {
						break
					}
					if val <= 21 {
						bias++
					} else {
						bias--
					}
				}
				if bias >= 0 {
					s.replaceNode(node, legionNode1(legionNodes, 77)) // might of the vaal
				} else {
					s.replaceNode(node, legionNode1(legionNodes, 78)) // legacy of the vaal
				}
				additions := map[int]float64{}
				var inserted []int
				var order []int
				for i := 0; i < headerSize/2; i++ {
					val := rec[i]
					roll := float64(rec[i+headerSize/2])
					inserted = append(inserted, val)
					if _, seen := additions[val]; !seen {
						additions[val] = roll
						order = append(order, val)
					} else {
						additions[val] += roll
					}
				}
				// The reference iterates the additions table with pairs() —
				// LuaJIT hash-slot order. This port merges in first-seen
				// order instead and records the merge so the differential
				// can permute into the reference's order (the difference is
				// display-only: which rolled line sits where on the node).
				node.TimelessAdditions = &TimelessAdditionsRecord{Inserted: inserted}
				for _, addID := range order {
					val := additions[addID]
					addition := legionAdditions[addID] // legionAdditions[add + 1]
					before := len(node.Stats.ModList)
					for _, addStat := range addition.Sd {
						for _, desc := range addition.StatDescs { // should only be 1 big
							addStat = timelessReplaceStat(addStat, desc, val)
						}
						s.nodeAdditionOrReplacementFromString(node, addStat, false)
					}
					node.TimelessAdditions.Blocks = append(node.TimelessAdditions.Blocks,
						TimelessAdditionBlock{ID: addID, ModCount: len(node.Stats.ModList) - before})
				}
			default:
				// "Unhandled Glorious Vanity headerSize" — no change
			}
		} else {
			for _, g := range rec {
				if g >= 337 { // replace
					if legionNode := legionNode1(legionNodes, g+1-337); legionNode != nil {
						s.replaceNode(node, legionNode)
					}
				} else { // add
					addition := legionAdditions[g]
					for _, addStat := range addition.Sd {
						s.nodeAdditionOrReplacementFromString(node, " \n"+addStat, false)
					}
				}
			}
		}

	case node.Type() == "Keystone":
		matchStr := conquerorType(cq) + "_keystone_" + conquerorIDStr(cq)
		for _, legionNode := range legionNodes {
			if legionNode.IDStr == matchStr {
				s.replaceNode(node, legionNode)
				break
			}
		}

	case node.Type() == "Normal":
		isAttr := node.Dn == "Dexterity" || node.Dn == "Intelligence" || node.Dn == "Strength"
		switch conquerorType(cq) {
		case "vaal":
			var rec []int
			if seedInRange {
				rec = ReadTimelessLUT(int64(rawSeed), node.ID(), jewelType)
			}
			if len(rec) != 0 {
				legionNode := legionNode1(legionNodes, rec[0]+1-337)
				s.replaceNode(node, legionNode)
				sd := node.Stats.Sd // the reference iterates node.sd (now the legion sd)
				for i, repStat := range sd {
					desc := legionStatDescOf(legionNode, legionNode.SortedStats[i])
					repStat = timelessReplaceStat(repStat, desc, float64(rec[1]))
					s.nodeAdditionOrReplacementFromString(node, repStat, true)
				}
			}
		case "karui":
			str := "4"
			if isAttr || node.IsTattoo {
				str = "2"
			}
			s.nodeAdditionOrReplacementFromString(node, " \n+"+str+" to Strength", false)
		case "maraketh":
			dex := "4"
			if isAttr || node.IsTattoo {
				dex = "2"
			}
			s.nodeAdditionOrReplacementFromString(node, " \n+"+dex+" to Dexterity", false)
		case "kalguur":
			ward := "2"
			if isAttr || node.IsTattoo {
				ward = "1"
			}
			s.nodeAdditionOrReplacementFromString(node, " \n"+ward+"% increased Ward", false)
		case "templar":
			if isAttr || node.IsTattoo {
				s.replaceNode(node, legionNode1(legionNodes, 91)) // templar_devotion_node
			} else {
				s.nodeAdditionOrReplacementFromString(node, " \n+5 to Devotion", false)
			}
		case "eternal":
			s.replaceNode(node, legionNode1(legionNodes, 110)) // eternal_small_blank
		}
	}
	s.reconnectNodeToClassStart(node)
}

// legionNode1 indexes the ordered legion pool 1-based (legionNodes[i]).
func legionNode1(nodes []*Node, index1 int) *Node {
	if index1 >= 1 && index1 <= len(nodes) {
		return nodes[index1-1]
	}
	return nil
}

func legionStatDescOf(node *Node, statKey string) *LegionStatDesc {
	for _, desc := range node.StatDescs {
		if desc.ID == statKey {
			return desc
		}
	}
	panic("tree: legion node " + node.IDStr + " missing stat desc " + statKey)
}
