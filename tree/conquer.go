// The conquering half of BuildAllDependsAndPaths (PassiveSpec.lua
// L1191-1376): replacing or augmenting conquered nodes — every historic
// jewel family, timeless and abyss — from computed records, plus
// NodeAdditionOrReplacementFromString and ReconnectNodeToClassStart.
package tree

import (
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// conquerReplaceStat is BuildAllDependsAndPaths' replaceHelperFunc.
func conquerReplaceStat(stat string, desc *ConqueredStatDesc, value float64) string {
	if desc.Fmt == "g" {
		if strings.Contains(desc.ID, "per_minute") {
			value = util.RoundHalfUp(value/60, 1)
		} else if strings.Contains(desc.ID, "permyriad") {
			value = value / 100
		} else if strings.Contains(desc.ID, "_ms") {
			value = value / 1000
		}
	}
	if desc.Min != desc.Max {
		return strings.ReplaceAll(stat, "("+util.FormatIntOrG14(desc.Min)+"-"+util.FormatIntOrG14(desc.Max)+")", util.FormatIntOrG14(value))
	} else if desc.Min != value {
		// only true for might/legacy of the vaal, which can combine stats
		return strings.ReplaceAll(stat, util.FormatIntOrG14(desc.Min), util.FormatIntOrG14(value))
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
	// The stats are the node's own now: no source shares them.
	node.src, node.srcME = nil, nil
}

// reconnectNodeToClassStart ports ReconnectNodeToClassStart (Pure Talent).
func (s *Spec) reconnectNodeToClassStart(node *SpecNode) {
	for _, linkedID := range node.T.LinkedIDs {
		for _, classID := range sortedNodeIDs(s.Tree.Classes) {
			class := s.Tree.Classes[classID]
			if linkedID == class.StartNodeID && node.Type() == NodeNormal {
				node.Stats.ModList = append(node.Stats.ModList,
					modparser.NewModFull("Condition:ConnectedTo"+class.Name+"Start", modparser.Flag, modparser.Bool(true),
						"Tree:"+strconv.FormatInt(linkedID, 10), true, 0, 0))
			}
		}
	}
}

// applyConquered ports the "If node is conquered, replace it or add mods"
// branch of BuildAllDependsAndPaths' second node loop.
func (s *Spec) applyConquered(node *SpecNode) {
	cq := node.ConqueredBy
	if cq == nil || node.Type() == NodeSocket {
		return
	}
	if isAbyss(cq.Conqueror) {
		s.applyAbyssConquest(node, cq)
		s.reconnectNodeToClassStart(node)
		return
	}
	conqueredNodes := s.Tree.ConqueredPassives.NodesOrdered
	conqueredAdditions := s.Tree.ConqueredPassives.AdditionsOrdered
	jewelType := lutType(cq.Conqueror)
	rawSeed := cq.Seed
	seed := rawSeed
	if jewelType == int(modparser.ConquerorEternal) {
		seed = seed / 20
	}
	seedInRange := seed >= data.TimelessJewelSeedMin[jewelType] && seed <= data.TimelessJewelSeedMax[jewelType]

	switch node.Type() {
	case NodeNotable:
		var rec []int
		if seedInRange {
			rec = TimelessPassive(int64(rawSeed), node.ID(), jewelType)
		}
		if len(rec) == 0 {
			// "Missing LUT" — no node change; reconnect still runs.
		} else if cq.Conqueror == modparser.ConquerorVaal {
			s.applyGloriousVanity(node, rec)
		} else {
			for _, g := range rec {
				if g >= poolNodeBase { // replace
					if conqueredNode := conqueredNode1(conqueredNodes, g+1-poolNodeBase); conqueredNode != nil {
						s.replaceNode(node, conqueredNode)
					}
				} else { // add
					addition := conqueredAdditions[g]
					for _, addStat := range addition.Sd {
						s.nodeAdditionOrReplacementFromString(node, " \n"+addStat, false)
					}
				}
			}
		}

	case NodeKeystone:
		matchStr := cq.Conqueror.String() + "_keystone_" + cq.ConqID
		for _, conqueredNode := range conqueredNodes {
			if conqueredNode.IDStr == matchStr {
				s.replaceNode(node, conqueredNode)
				break
			}
		}

	case NodeNormal:
		isAttr := node.Dn == "Dexterity" || node.Dn == "Intelligence" || node.Dn == "Strength"
		small := isAttr || node.IsTattoo // attribute and tattooed smalls take the lesser bonus
		switch cq.Conqueror {
		case modparser.ConquerorVaal:
			var rec []int
			if seedInRange {
				rec = TimelessPassive(int64(rawSeed), node.ID(), jewelType)
			}
			if len(rec) != 0 {
				conqueredNode := conqueredNode1(conqueredNodes, rec[0]+1-poolNodeBase)
				s.replaceNode(node, conqueredNode)
				sd := node.Stats.Sd // the reference iterates node.sd (now the alt node's sd)
				for i, repStat := range sd {
					desc := conqueredStatDescOf(conqueredNode, conqueredNode.SortedStats[i])
					repStat = conquerReplaceStat(repStat, desc, float64(rec[1]))
					s.nodeAdditionOrReplacementFromString(node, repStat, true)
				}
			}
		case modparser.ConquerorKarui:
			s.nodeAdditionOrReplacementFromString(node, " \n+"+pick(small, "2", "4")+" to Strength", false)
		case modparser.ConquerorMaraketh:
			s.nodeAdditionOrReplacementFromString(node, " \n+"+pick(small, "2", "4")+" to Dexterity", false)
		case modparser.ConquerorKalguur:
			s.nodeAdditionOrReplacementFromString(node, " \n"+pick(small, "1", "2")+"% increased Ward", false)
		case modparser.ConquerorTemplar:
			if small {
				s.replaceNode(node, conqueredNode1(conqueredNodes, poolTemplarDevotionNode))
			} else {
				s.nodeAdditionOrReplacementFromString(node, " \n+5 to Devotion", false)
			}
		case modparser.ConquerorEternal:
			s.replaceNode(node, conqueredNode1(conqueredNodes, poolEternalSmallBlank))
		}
	}
	s.reconnectNodeToClassStart(node)
}

func pick(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}

// applyAbyssConquest applies the computed abyss components: a replacement
// (rolled into the pool node's stats) and/or stat additions.
func (s *Spec) applyAbyssConquest(node *SpecNode, cq *Conquest) {
	conqueredAdditions := s.Tree.ConqueredPassives.AdditionsOrdered
	for _, comp := range cq.Abyss {
		var sd []string
		var descs []*ConqueredStatDesc
		replaces := false
		switch comp.Kind {
		case ComponentReplace:
			if conqueredNode := conqueredNode1(s.Tree.ConqueredPassives.NodesOrdered, comp.ID+1-poolNodeBase); conqueredNode != nil {
				s.replaceNode(node, conqueredNode)
				sd = conqueredNode.Sd
				descs = conqueredNode.StatDescs
				replaces = true
			}
		case ComponentAdd:
			if comp.ID >= 0 && comp.ID < len(conqueredAdditions) {
				addition := conqueredAdditions[comp.ID]
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
					statLine = conquerReplaceStat(statLine, desc, float64(comp.Rolls[desc.Index-1]))
				}
			}
			prefix := " \n"
			if replaces {
				prefix = ""
			}
			s.nodeAdditionOrReplacementFromString(node, prefix+statLine, replaces && statIndex == 0)
		}
	}
}

// Glorious Vanity LUT record layouts (TimelessPassive for the vaal family).
const (
	vaalReplaceRecord1Roll  = 2 // replacement id + one roll
	vaalReplaceRecord2Rolls = 3 // replacement id + two rolls
	vaalAdditionsRecord3    = 6 // three addition ids + three rolls
	vaalAdditionsRecord4    = 8 // four addition ids + four rolls
)

// applyGloriousVanity ports the vaal notable branch: a rolled replacement,
// or the might/legacy-of-the-vaal blank plus merged additions.
func (s *Spec) applyGloriousVanity(node *SpecNode, rec []int) {
	conqueredNodes := s.Tree.ConqueredPassives.NodesOrdered
	conqueredAdditions := s.Tree.ConqueredPassives.AdditionsOrdered
	headerSize := len(rec)
	switch headerSize {
	case vaalReplaceRecord1Roll, vaalReplaceRecord2Rolls:
		conqueredNode := conqueredNode1(conqueredNodes, rec[0]+1-poolNodeBase)
		s.replaceNode(node, conqueredNode)
		sd := conqueredNode.Sd // snapshot: the node's sd is rebuilt below
		for i, repStat := range sd {
			desc := conqueredStatDescOf(conqueredNode, conqueredNode.SortedStats[i])
			repStat = conquerReplaceStat(repStat, desc, float64(rec[desc.Index]))
			s.nodeAdditionOrReplacementFromString(node, repStat, i == 0) // wipe mods on first run
		}
	case vaalAdditionsRecord3, vaalAdditionsRecord4:
		bias := 0
		for _, val := range rec[:headerSize/2] {
			if val <= 21 {
				bias++
			} else {
				bias--
			}
		}
		if bias >= 0 {
			s.replaceNode(node, conqueredNode1(conqueredNodes, poolMightOfTheVaal))
		} else {
			s.replaceNode(node, conqueredNode1(conqueredNodes, poolLegacyOfTheVaal))
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
			addition := conqueredAdditions[addID] // conqueredAdditions[add + 1]
			before := len(node.Stats.ModList)
			for _, addStat := range addition.Sd {
				for _, desc := range addition.StatDescs { // should only be 1 big
					addStat = conquerReplaceStat(addStat, desc, val)
				}
				s.nodeAdditionOrReplacementFromString(node, addStat, false)
			}
			node.TimelessAdditions.Blocks = append(node.TimelessAdditions.Blocks,
				TimelessAdditionBlock{ID: addID, ModCount: len(node.Stats.ModList) - before})
		}
	default:
		// "Unhandled Glorious Vanity headerSize" — no change
	}
}

// conqueredNode1 indexes the ordered alternate pool 1-based (the reference's
// legionNodes[i]).
func conqueredNode1(nodes []*Node, index1 int) *Node {
	if index1 >= 1 && index1 <= len(nodes) {
		return nodes[index1-1]
	}
	return nil
}

// conqueredStatDescOf finds the pool node's roll descriptor for a stat id;
// a missing one is a pool-document defect (the Lua errors indexing nil).
func conqueredStatDescOf(node *Node, statKey string) *ConqueredStatDesc {
	for _, desc := range node.StatDescs {
		if desc.ID == statKey {
			return desc
		}
	}
	panic("tree: alt node " + node.IDStr + " missing stat desc " + statKey)
}
