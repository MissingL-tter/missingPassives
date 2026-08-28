// Port of Classes/PassiveSpec.lua's load half: the per-build allocation
// state. Spec nodes shadow tree nodes (the reference uses __index
// metatables; here explicit copies reset by replaceNode), and
// BuildAllDependsAndPaths reproduces the dependency/pruning analysis, the
// radius-jewel rules (intuitive-leap-like, Impossible Escape), mastery
// effect application, tattoo overrides, jewel conquering (conquer.go,
// all families) and the path/distance rebuilds. Cluster jewel subgraphs
// live in cluster.go.
package tree

import (
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// SpecNode is one node of the spec's local tree copy.
type SpecNode struct {
	T *Node // the underlying tree (or override/alternate) node

	Alloc                       bool
	Linked                      []*SpecNode
	Depends                     []*SpecNode
	IntuitiveLeapLikesAffecting []*SpecNode
	ConqueredBy                 any
	Visited                     bool
	ConnectedToStart            bool
	PathDist                    int
	Path                        []*SpecNode
	HasPath                     bool
	DistanceToClassStart        *float64

	// Shadowable display/mod state (replaceNode resets these to a source
	// node's).
	Name              *string
	Dn                string
	Stats             Stats
	KeystoneMod       *modparser.Mod
	IsTattoo          bool
	OverrideType      string
	ReminderText      []string
	AllMasteryOptions bool

	// TimelessAdditions records the might/legacy-of-the-vaal additions
	// merge (Glorious Vanity): the raw inserted addition-id sequence and
	// how many mods each distinct addition prepended, in processing order.
	// The merge runs in first-seen order while the reference's pairs()
	// order is LuaJIT hash-slot order; the differential uses this record
	// to permute the mod list into the reference's order for comparison.
	TimelessAdditions *TimelessAdditionsRecord

	// sdIdentity tracks which source object the current sd came from, for
	// replaceNode's identity short-circuit (the reference compares table
	// identity).
	sdIdentity any
}

// TimelessAdditionsRecord — see SpecNode.TimelessAdditions.
type TimelessAdditionsRecord struct {
	Inserted []int
	Blocks   []TimelessAdditionBlock
}

// TimelessAdditionBlock is one distinct addition's contribution.
type TimelessAdditionBlock struct {
	ID       int
	ModCount int
}

func (n *SpecNode) ID() int64              { return n.T.ID }
func (n *SpecNode) Type() string           { return n.T.Type }
func (n *SpecNode) AscendancyName() string { return n.T.AscendancyName }

// Spec is one build's passive spec.
type Spec struct {
	Tree  *Tree
	Items map[int]*item.Item // the build's items, for socketed jewels

	Nodes      map[int64]*SpecNode
	AllocNodes map[int64]*SpecNode

	AllocSubgraphNodes       []int64
	AllocExtendedNodes       []int64
	Jewels                   map[int64]int
	MasterySelections        map[int64]int64
	HashOverrides            map[int64]*Node
	ClusterHashFormatVersion int
	SubGraphs                map[int64]*SubGraph

	legacyClusterNodeMap        map[int64]int64
	legacyClusterNodeMapReverse map[int64]int64

	CurClassID                  int64
	CurClass                    *Class
	CurClassName                string
	CurAscendClassID            int64
	CurAscendClass              *AscendClass
	CurAscendClassName          string
	CurAscendClassBaseName      string
	CurSecondaryAscendClassID   int64
	CurSecondaryAscendClass     *AscendClass
	CurSecondaryAscendClassName string
	classSelected               bool
	ascendSelected              bool
	secondarySelected           bool

	AllocatedNotableCount     float64
	AllocatedKeystoneCount    float64
	AllocatedMasteryCount     float64
	AllocatedMasteryTypeCount float64
	AllocatedMasteryTypes     map[string]float64
	AllocatedTattooTypes      map[string]float64

	SplitPersonalityPath map[int64]bool
}

// SavedOverride is one <Override> child (a tattooed node).
type SavedOverride struct {
	NodeID            int64
	Dn                string
	Icon              string
	ActiveEffectImage string
}

// SavedSpec is the decoded <Spec> element.
type SavedSpec struct {
	ClassID                int64
	AscendClassID          int64
	SecondaryAscendClassID int64
	Nodes                  string // the "nodes" attribute (comma list)
	MasteryEffects         string // "{node,effect},..."
	Sockets                map[int64]int
	Overrides              []SavedOverride
	// ClusterHashFormatVersion: 2 is current; a nodes attribute without
	// the version attribute is legacy (1).
	ClusterHashFormatVersion int
}

// NewSpec ports PassiveSpecClass:Init for one tree.
func NewSpec(t *Tree, items map[int]*item.Item) *Spec {
	s := &Spec{
		Tree:                  t,
		Items:                 items,
		Nodes:                 map[int64]*SpecNode{},
		AllocNodes:            map[int64]*SpecNode{},
		Jewels:                map[int64]int{},
		MasterySelections:     map[int64]int64{},
		HashOverrides:         map[int64]*Node{},
		SubGraphs:             map[int64]*SubGraph{},
		AllocatedMasteryTypes: map[string]float64{},
		AllocatedTattooTypes:  map[string]float64{},
		SplitPersonalityPath:  map[int64]bool{},
	}
	for id, treeNode := range t.Nodes {
		if treeNode.Group == nil || treeNode.IsProxy || treeNode.Group.IsProxy {
			continue
		}
		if ej, ok := treeNode.Raw["expansionJewel"].(map[string]any); ok {
			if _, hasParent := ej["parent"]; hasParent {
				continue
			}
		}
		node := &SpecNode{T: treeNode}
		node.resetToSource(treeNode)
		s.Nodes[id] = node
	}
	for _, node := range s.Nodes {
		for _, otherID := range node.T.LinkedIDs {
			if other := s.Nodes[otherID]; other != nil {
				node.Linked = append(node.Linked, other)
			}
		}
	}
	return s
}

// resetToSource is replaceNode's copy half: point the spec node's display
// and mod state at src.
func (n *SpecNode) resetToSource(src *Node) {
	n.Dn = src.Name
	// Name shadows the underlying node's; a source without one (tattoo,
	// alternate) leaves the shadow empty and EffectiveName falls through to
	// the spec node's own tree name — the reference's metatable behavior.
	n.Name = src.NameStr
	n.Stats = src.Stats
	n.Stats.ModList = append([]*modparser.Mod{}, src.ModList...)
	n.KeystoneMod = src.KeystoneMod
	n.IsTattoo = src.IsTattooFlag
	n.OverrideType = src.OverrideTypeStr
	n.ReminderText = nil
	n.TimelessAdditions = nil
	n.sdIdentity = src
}

// EffectiveName is node.name through the reference's metatable: the shadow
// when set, else the underlying node's own name (nil for synthesized
// cluster nodes, which have only dn).
func (n *SpecNode) EffectiveName() *string {
	if n.Name != nil {
		return n.Name
	}
	return n.T.NameStr
}

// replaceNode ports PassiveSpecClass:ReplaceNode.
func (s *Spec) replaceNode(old *SpecNode, src *Node) {
	if old.sdIdentity == any(src) {
		return
	}
	old.resetToSource(src)
}

// LoadSaved ports the Spec-element half of PassiveSpecClass:Load plus
// ImportFromNodeList.
func (s *Spec) LoadSaved(saved *SavedSpec) {
	s.ClusterHashFormatVersion = saved.ClusterHashFormatVersion
	// Tattoo overrides: resolve each by name, with the reference's
	// substitution fallback (match by images when the name was renamed,
	// registering the alias in the shared pool).
	for _, o := range saved.Overrides {
		if s.Tree.Tattoo.Nodes[o.Dn] == nil {
			for _, name := range sortedStringKeys(s.Tree.Tattoo.Nodes) {
				dataNode := s.Tree.Tattoo.Nodes[name]
				if str(dataNode.Raw["activeEffectImage"]) == o.ActiveEffectImage && dataNode.Icon == o.Icon {
					s.Tree.Tattoo.Nodes[o.Dn] = dataNode
				}
			}
		}
		if src := s.Tree.Tattoo.Nodes[o.Dn]; src != nil {
			cp := *src // copyTable(..., true): shallow, sharing sd/modList
			cp.ID = o.NodeID
			cp.IDStr = strconv.FormatInt(o.NodeID, 10)
			s.HashOverrides[o.NodeID] = &cp
		}
	}
	for nodeID, itemID := range saved.Sockets {
		if itemID > 0 {
			s.Jewels[nodeID] = itemID
		}
	}
	var hashList []int64
	for _, part := range strings.FieldsFunc(saved.Nodes, func(r rune) bool { return r < '0' || r > '9' }) {
		n, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			hashList = append(hashList, n)
		}
	}
	masteryEffects := map[int64]int64{}
	rest := saved.MasteryEffects
	for {
		open := strings.IndexByte(rest, '{')
		if open < 0 {
			break
		}
		close := strings.IndexByte(rest[open:], '}')
		if close < 0 {
			break
		}
		pair := rest[open+1 : open+close]
		rest = rest[open+close:]
		comma := strings.IndexByte(pair, ',')
		if comma > 0 {
			mastery, err1 := strconv.ParseInt(pair[:comma], 10, 64)
			effect, err2 := strconv.ParseInt(pair[comma+1:], 10, 64)
			if err1 == nil && err2 == nil {
				masteryEffects[mastery] = effect
			}
		}
	}
	s.importFromNodeList(saved.ClassID, saved.AscendClassID, saved.SecondaryAscendClassID, hashList, masteryEffects)
}

// importFromNodeList ports ImportFromNodeList (className resolution and
// tree-version switching excluded: the saved-XML path supplies ids and one
// version).
func (s *Spec) importFromNodeList(classID, ascendClassID, secondaryAscendClassID int64, hashList []int64, masteryEffects map[int64]int64) {
	s.resetNodes()
	s.SelectClass(classID)
	s.SelectAscendClass(ascendClassID)
	s.SelectSecondaryAscendClass(secondaryAscendClassID)
	for k := range s.MasterySelections {
		delete(s.MasterySelections, k)
	}
	for mastery, effect := range masteryEffects {
		if effect < 65536 {
			s.MasterySelections[mastery] = effect
		}
	}
	for _, id := range sortedNodeIDs(s.HashOverrides) {
		if node := s.Nodes[id]; node != nil {
			s.replaceNode(node, s.HashOverrides[id])
		}
	}
	for _, id := range hashList {
		node := s.Nodes[id]
		if node != nil {
			if node.Type() != "Mastery" || s.MasterySelections[id] != 0 {
				node.Alloc = true
				s.AllocNodes[id] = node
			}
		} else {
			s.AllocSubgraphNodes = append(s.AllocSubgraphNodes, id)
		}
	}
	for _, id := range s.AllocExtendedNodes {
		if node := s.Nodes[id]; node != nil {
			node.Alloc = true
			s.AllocNodes[id] = node
		}
	}
	s.BuildAllDependsAndPaths()
}

// PostLoad ports PassiveSpecClass:PostLoad.
func (s *Spec) PostLoad() {
	s.BuildClusterJewelGraphs()
}

func (s *Spec) resetNodes() {
	for id, node := range s.Nodes {
		if node.Type() != "ClassStart" && node.Type() != "AscendClassStart" {
			node.Alloc = false
			delete(s.AllocNodes, id)
		}
	}
	for k := range s.MasterySelections {
		delete(s.MasterySelections, k)
	}
}

// SelectClass ports SelectClass (minus the recursive SelectAscendClass(0)
// tail's rebuild; importFromNodeList immediately selects the real one, and
// the reference's intermediate rebuild has no lasting effect on state the
// fixture sees — but run it anyway for faithfulness).
func (s *Spec) SelectClass(classID int64) {
	if s.classSelected {
		oldStartNodeID := s.CurClass.StartNodeID
		if node := s.Nodes[oldStartNodeID]; node != nil {
			node.Alloc = false
			delete(s.AllocNodes, oldStartNodeID)
		}
	}
	s.resetAscendClass()
	s.classSelected = true
	s.CurClassID = classID
	class := s.Tree.Classes[classID]
	if class == nil {
		panic("tree: unknown class id " + strconv.FormatInt(classID, 10))
	}
	s.CurClass = class
	s.CurClassName = class.Name
	startNode := s.Nodes[class.StartNodeID]
	startNode.Alloc = true
	s.AllocNodes[startNode.ID()] = startNode
	s.SelectAscendClass(0)
}

func (s *Spec) resetAscendClass() {
	if s.ascendSelected {
		ascendClass := s.CurClass.Classes[s.CurAscendClassID]
		if ascendClass == nil {
			ascendClass = s.CurClass.Classes[0]
		}
		if ascendClass.StartNodeID != 0 {
			if node := s.Nodes[ascendClass.StartNodeID]; node != nil {
				node.Alloc = false
				delete(s.AllocNodes, ascendClass.StartNodeID)
			}
		}
	}
}

func (s *Spec) SelectAscendClass(ascendClassID int64) {
	s.resetAscendClass()
	s.ascendSelected = true
	s.CurAscendClassID = ascendClassID
	ascendClass := s.CurClass.Classes[ascendClassID]
	if ascendClass == nil {
		ascendClass = s.CurClass.Classes[0]
	}
	s.CurAscendClass = ascendClass
	s.CurAscendClassName = ascendClass.Name
	s.CurAscendClassBaseName = ascendClass.ID
	if ascendClass.StartNodeID != 0 {
		startNode := s.Nodes[ascendClass.StartNodeID]
		startNode.Alloc = true
		s.AllocNodes[startNode.ID()] = startNode
	}
	s.BuildAllDependsAndPaths()
}

func (s *Spec) SelectSecondaryAscendClass(ascendClassID int64) {
	if s.Tree.SecondaryAscendNameMap == nil {
		return
	}
	if s.secondarySelected {
		if asc := s.secondaryAscendClass(s.CurSecondaryAscendClassID); asc != nil && asc.StartNodeID != 0 {
			if node := s.Nodes[asc.StartNodeID]; node != nil {
				node.Alloc = false
				delete(s.AllocNodes, asc.StartNodeID)
			}
		}
	}
	s.secondarySelected = true
	s.CurSecondaryAscendClassID = ascendClassID
	if ascendClassID == 0 {
		s.CurSecondaryAscendClass = nil
		s.CurSecondaryAscendClassName = "None"
	} else if asc := s.secondaryAscendClass(ascendClassID); asc != nil {
		s.CurSecondaryAscendClass = asc
		s.CurSecondaryAscendClassName = asc.Name
		if asc.StartNodeID != 0 {
			startNode := s.Nodes[asc.StartNodeID]
			startNode.Alloc = true
			s.AllocNodes[startNode.ID()] = startNode
		}
	}
	s.BuildAllDependsAndPaths()
}

// secondaryAscendClass resolves an alternate-ascendancy id the way the
// reference indexes tree.alternate_ascendancies.
func (s *Spec) secondaryAscendClass(id int64) *AscendClass {
	for _, entry := range s.Tree.SecondaryAscendNameMap {
		if entry.AscendClassID == id {
			return entry.AscendClass
		}
	}
	return nil
}

func (s *Spec) jewel(itemID int) *item.Item {
	if itemID == 0 {
		return nil
	}
	return s.Items[itemID]
}

func jd(it *item.Item, key string) any {
	if it == nil || it.JewelData == nil {
		return nil
	}
	return it.JewelData[key]
}

func jdTrue(it *item.Item, key string) bool { return truthyVal(jd(it, key)) }

func truthyVal(v any) bool { return v != nil && v != false }

func sortedNodeIDs[V any](m map[int64]V) []int64 {
	ids := make([]int64, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func dependsContains(list []*SpecNode, node *SpecNode) bool {
	for _, n := range list {
		if n == node {
			return true
		}
	}
	return false
}
