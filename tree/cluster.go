// Cluster jewel subgraphs: PassiveSpec.lua L1651-1885 and L1885-2273.
// A socketed cluster jewel fabricates nodes (ids 0x10000+) arranged on a
// ring hanging off the proxy group, wires them into the spec, and recurses
// into any smaller sockets it grants.
package tree

import (
	"sort"
	"strconv"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// SubGraph is one generated cluster graph.
type SubGraph struct {
	Nodes        []*SpecNode
	Group        *Group
	ParentSocket *SpecNode
	EntranceNode *SpecNode
}

// getSocketedJewel ports GetSocketedJewel (with the legacy reverse map).
func (s *Spec) getSocketedJewel(nodeID int64) *item.Item {
	itemID := s.Jewels[nodeID]
	if itemID == 0 && s.legacyClusterNodeMapReverse != nil {
		if legacyID, ok := s.legacyClusterNodeMapReverse[nodeID]; ok {
			itemID = s.Jewels[legacyID]
		}
	}
	it := s.jewel(itemID)
	if it == nil || it.JewelData == nil {
		return nil
	}
	return it
}

// --- Legacy (v1) cluster hash conversion ---

func (s *Spec) beginLegacyClusterHashConversion() bool {
	version := s.ClusterHashFormatVersion
	if version == 0 {
		version = 2
	}
	needs := version < 2
	if needs {
		s.legacyClusterNodeMap = map[int64]int64{}
		s.legacyClusterNodeMapReverse = map[int64]int64{}
	} else {
		s.legacyClusterNodeMap = nil
		s.legacyClusterNodeMapReverse = nil
	}
	return needs
}

func (s *Spec) registerLegacyClusterNodeMap(legacyNodeID, currentNodeID int64) {
	if s.legacyClusterNodeMap == nil || legacyNodeID == 0 || currentNodeID == 0 {
		return
	}
	s.legacyClusterNodeMap[legacyNodeID] = currentNodeID
	s.legacyClusterNodeMapReverse[currentNodeID] = legacyNodeID
}

func (s *Spec) getMappedClusterNodeID(nodeID int64) int64 {
	if mapped, ok := s.legacyClusterNodeMap[nodeID]; ok && s.Nodes[mapped] != nil {
		return mapped
	}
	return nodeID
}

func (s *Spec) applyLegacyClusterNodeRemap() {
	if s.legacyClusterNodeMap == nil {
		return
	}
	var converted []int64
	seen := map[int64]bool{}
	for _, nodeID := range s.AllocSubgraphNodes {
		nodeID = s.getMappedClusterNodeID(nodeID)
		if !seen[nodeID] {
			seen[nodeID] = true
			converted = append(converted, nodeID)
		}
	}
	s.AllocSubgraphNodes = converted
	for _, legacyNodeID := range sortedNodeIDs(s.legacyClusterNodeMap) {
		currentNodeID := s.legacyClusterNodeMap[legacyNodeID]
		if legacyNodeID != currentNodeID && s.AllocNodes[legacyNodeID] != nil && s.Nodes[currentNodeID] != nil {
			s.AllocNodes[legacyNodeID].Alloc = false
			delete(s.AllocNodes, legacyNodeID)
			if !seen[currentNodeID] {
				seen[currentNodeID] = true
				s.AllocSubgraphNodes = append(s.AllocSubgraphNodes, currentNodeID)
			}
		}
	}
	convertedJewels := map[int64]int{}
	for nodeID, itemID := range s.Jewels {
		convertedJewels[s.getMappedClusterNodeID(nodeID)] = itemID
	}
	s.Jewels = convertedJewels
}

func (s *Spec) endLegacyClusterHashConversion() {
	s.ClusterHashFormatVersion = 2
	s.legacyClusterNodeMap = nil
	s.legacyClusterNodeMapReverse = nil
}

// BuildClusterJewelGraphs ports the same-named method (no imported
// subgraph data: that path only exists for character imports).
func (s *Spec) BuildClusterJewelGraphs() {
	needsLegacy := s.beginLegacyClusterHashConversion()

	for _, id := range sortedNodeIDs(s.SubGraphs) {
		subGraph := s.SubGraphs[id]
		for _, node := range subGraph.Nodes {
			delete(s.Nodes, node.ID())
			if s.AllocNodes[node.ID()] != nil {
				delete(s.AllocNodes, node.ID())
				s.AllocSubgraphNodes = append(s.AllocSubgraphNodes, node.ID())
			}
		}
		for i, linked := range subGraph.ParentSocket.Linked {
			if linked == subGraph.EntranceNode {
				subGraph.ParentSocket.Linked = append(subGraph.ParentSocket.Linked[:i], subGraph.ParentSocket.Linked[i+1:]...)
				break
			}
		}
	}
	s.SubGraphs = map[int64]*SubGraph{}

	for _, nodeID := range sortedNodeIDs(s.Tree.Sockets) {
		node := s.Tree.Nodes[nodeID]
		if node == nil {
			continue
		}
		if node.ExpansionJewel == nil || node.ExpansionJewel.Size != 2 {
			continue
		}
		jewel := s.getSocketedJewel(nodeID)
		if jewel != nil && jewelData(jewel).ClusterJewelValid {
			s.buildSubgraph(jewel, s.Nodes[nodeID], 0, nil)
		}
	}

	if needsLegacy {
		s.applyLegacyClusterNodeRemap()
	}

	for _, nodeID := range s.AllocSubgraphNodes {
		node := s.Nodes[nodeID]
		if node != nil {
			node.Alloc = true
			if s.AllocNodes[nodeID] == nil {
				s.AllocNodes[nodeID] = node
				found := false
				for _, id := range s.AllocExtendedNodes {
					if id == nodeID {
						found = true
						break
					}
				}
				if !found {
					s.AllocExtendedNodes = append(s.AllocExtendedNodes, nodeID)
				}
			}
		}
	}
	s.AllocSubgraphNodes = nil

	s.BuildAllDependsAndPaths()
	s.endLegacyClusterHashConversion()
}

// findClusterSocket ports FindClusterSocket.
func (s *Spec) findClusterSocket(group *Group, index int64) *Node {
	for _, nodeID := range group.NodeIDs {
		node := s.Tree.Nodes[nodeID]
		if node == nil {
			continue
		}
		if ej := node.ExpansionJewel; ej != nil && ej.Index == index {
			return node
		}
	}
	return nil
}

// buildLegacyProxyGroup ports BuildLegacyProxyGroup.
func (s *Spec) buildLegacyProxyGroup(proxyGroup *Group, expansionJewelSize, clusterSizeIndex int64) *Group {
	legacyGroup := proxyGroup
	groupSize := expansionJewelSize
	for guard := 0; clusterSizeIndex < groupSize && guard < 4; guard++ {
		socket := s.findClusterSocket(legacyGroup, 1)
		if socket == nil {
			socket = s.findClusterSocket(legacyGroup, 0)
		}
		if socket == nil {
			break
		}
		legacyProxyNode := s.Tree.Nodes[socket.ExpansionJewel.Proxy]
		if legacyProxyNode == nil || legacyProxyNode.Group == nil {
			break
		}
		legacyGroup = legacyProxyNode.Group
		groupSize = socket.ExpansionJewel.Size
	}
	return legacyGroup
}

var (
	orbit12to16 = []int64{0, 1, 3, 4, 5, 7, 8, 9, 11, 12, 13, 15}
	orbit16to12 = []int64{0, 1, 1, 2, 3, 4, 4, 5, 6, 7, 7, 8, 9, 10, 10, 11}
	orbit6to16  = []int64{0, 3, 5, 8, 11, 13}
	orbit16to6  = []int64{0, 0, 0, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 5, 5, 5}
)

// translateClusterOrbitIndex ports TranslateClusterOrbitIndex.
func translateClusterOrbitIndex(srcOidx, srcNodesPerOrbit, destNodesPerOrbit int64) int64 {
	switch {
	case srcNodesPerOrbit == destNodesPerOrbit:
		return srcOidx
	case srcNodesPerOrbit == 12 && destNodesPerOrbit == 16:
		return orbit12to16[srcOidx]
	case srcNodesPerOrbit == 16 && destNodesPerOrbit == 12:
		return orbit16to12[srcOidx]
	case srcNodesPerOrbit == 6 && destNodesPerOrbit == 16:
		return orbit6to16[srcOidx]
	case srcNodesPerOrbit == 16 && destNodesPerOrbit == 6:
		return orbit16to6[srcOidx]
	}
	return srcOidx * destNodesPerOrbit / srcNodesPerOrbit
}

// buildLegacyClusterOrbitMappings ports BuildLegacyClusterOrbitMappings.
func (s *Spec) buildLegacyClusterOrbitMappings(indicies map[int64]*Node, proxyNode *Node, clusterTotalIndicies, skillsPerOrbit int64) {
	if s.legacyClusterNodeMap == nil {
		return
	}
	legacySkillsPerOrbit := s.Tree.SkillsPerOrbit[proxyNode.Orbit]
	legacyProxyOidx := translateClusterOrbitIndex(proxyNode.OrbitIndex, legacySkillsPerOrbit, clusterTotalIndicies)
	legacyNodeIDsByOidx := map[int64]int64{}
	currentNodeIDsByOidx := map[int64]int64{}
	for _, nodeIndex := range sortedNodeIDs(indicies) {
		node := indicies[nodeIndex]
		legacyOidx := translateClusterOrbitIndex((nodeIndex+legacyProxyOidx)%clusterTotalIndicies, clusterTotalIndicies, legacySkillsPerOrbit)
		legacyNodeIDsByOidx[legacyOidx] = node.ID

		currentRel := translateClusterOrbitIndex(node.OrbitIndex, skillsPerOrbit, clusterTotalIndicies)
		currentInLegacy := translateClusterOrbitIndex(currentRel, clusterTotalIndicies, legacySkillsPerOrbit)
		currentNodeIDsByOidx[currentInLegacy] = node.ID
	}
	for _, oidx := range sortedNodeIDs(legacyNodeIDsByOidx) {
		legacyNodeID := legacyNodeIDsByOidx[oidx]
		currentNodeID := currentNodeIDsByOidx[oidx]
		if currentNodeID != 0 && legacyNodeID != currentNodeID {
			s.registerLegacyClusterNodeMap(legacyNodeID, currentNodeID)
		}
	}
}

// registerSubgraphNode wraps a synthesized raw node into the spec.
func (s *Spec) registerSubgraphNode(subGraph *SubGraph, raw *Node) *SpecNode {
	node := &SpecNode{T: raw}
	node.resetToSource(raw)
	subGraph.Nodes = append(subGraph.Nodes, node)
	return node
}

func linkSpecNodes(a, b *SpecNode) {
	a.Linked = append(a.Linked, b)
	b.Linked = append(b.Linked, a)
}

// buildSubgraph ports BuildSubgraph (imported character data excluded: the
// saved-XML path has none, so addToAllocatedSubgraphNodes is always
// false).
func (s *Spec) buildSubgraph(jewel *item.Item, parentSocket *SpecNode, id int64, upSize *int64) {
	parentEJ := parentSocket.T.ExpansionJewel
	if parentEJ == nil {
		panic("tree: buildSubgraph on non-expansion socket " + parentSocket.T.IDStr)
	}
	parentIndex, parentProxy, parentSize := parentEJ.Index, parentEJ.Proxy, parentEJ.Size
	clusterJewel := jewel.ClusterJewel
	if clusterJewel == nil {
		panic("tree: buildSubgraph without cluster jewel data at socket " + parentSocket.T.IDStr)
	}

	subGraph := &SubGraph{
		Group:        &Group{Orbits: map[int64]bool{}},
		ParentSocket: parentSocket,
	}

	// Subgraph id bit layout: 0-3 node index, 4-5 group size, 6-8 large
	// index, 9-10 medium index, 16 signal bit.
	if id == 0 {
		id = 0x10000
	}
	if parentSize == 2 {
		id += parentIndex << 6
	} else if parentSize == 1 {
		id += parentIndex << 9
	}
	sizeIndex := int64(clusterJewel.SizeIndex)
	nodeID := id + sizeIndex<<4

	s.SubGraphs[nodeID] = subGraph

	proxyNode := s.Tree.Nodes[parentProxy]
	if proxyNode == nil {
		panic("tree: proxy node " + strconv.FormatInt(parentProxy, 10) + " not found for socket " + parentSocket.T.IDStr)
	}
	proxyGroup := proxyNode.Group
	subGraph.Group.X = proxyGroup.X
	subGraph.Group.Y = proxyGroup.Y

	jdata := jewel.JewelData

	if keystone := jdata.ClusterJewelKeystone; keystone != "" {
		keystoneNode := s.Tree.ClusterNodeMap[keystone]
		if keystoneNode == nil {
			panic("tree: cluster keystone " + keystone + " not found (socket " + parentSocket.T.IDStr + ")")
		}
		raw := &Node{
			Type:       NodeKeystone,
			ID:         nodeID,
			IDStr:      strconv.FormatInt(nodeID, 10),
			Name:       keystoneNode.Name,
			Icon:       keystoneNode.Icon,
			Group:      subGraph.Group,
			Orbit:      0,
			OrbitIndex: 1,
		}
		raw.Sd = keystoneNode.Sd
		s.Tree.ProcessNode(raw)
		node := s.registerSubgraphNode(subGraph, raw)
		linkSpecNodes(node, parentSocket)
		subGraph.EntranceNode = node
		s.Nodes[raw.ID] = node
		return
	}

	var legacyProxyGroup *Group
	if s.legacyClusterNodeMap != nil {
		legacyProxyGroup = s.buildLegacyProxyGroup(proxyGroup, parentSize, sizeIndex)
	}

	nodeOrbit := sizeIndex + 1
	subGraph.Group.Orbits[nodeOrbit] = true

	// Notables, sorted by the cluster sort order.
	var notableList []*Node
	sortOrder := data.ClusterJewels.NotableSortOrder
	for _, name := range jdata.ClusterJewelNotables {
		baseNode := s.Tree.ClusterNodeMap[name]
		if baseNode == nil {
			// Old-tree notables that no longer exist: drop the subgraph.
			delete(s.SubGraphs, nodeID)
			return
		}
		if _, ok := sortOrder[baseNode.Name]; !ok {
			panic("tree: cluster notable " + name + " has no sort order (socket " + parentSocket.T.IDStr + ")")
		}
		notableList = append(notableList, baseNode)
	}
	sort.SliceStable(notableList, func(i, j int) bool {
		return sortOrder[notableList[i].Name] < sortOrder[notableList[j].Name]
	})

	skill, hasSkill := clusterJewel.Skills[jdata.ClusterJewelSkill]
	if !hasSkill {
		skill = data.ClusterSkillData{
			Name: "Nothingness",
			Icon: "Art/2DArt/SkillIcons/passives/MasteryBlank.png",
		}
	}
	socketCount := jdata.ClusterJewelSocketCount
	if jdata.ClusterJewelSocketCountOverride != 0 {
		socketCount = jdata.ClusterJewelSocketCountOverride
	}
	notableCount := len(notableList)
	nothingness := jdata.ClusterJewelNothingnessCount
	nodeCount := jdata.ClusterJewelNodeCount
	if nodeCount == 0 {
		nodeCount = socketCount + notableCount + nothingness
	}
	smallCount := nodeCount - socketCount - notableCount

	if skill.MasteryIcon != nil {
		subGraph.Group.Orbits[0] = true
		raw := &Node{
			Type:       NodeMastery,
			ID:         nodeID + 12,
			IDStr:      strconv.FormatInt(nodeID+12, 10),
			Name:       "Nothingness",
			Icon:       *skill.MasteryIcon,
			Group:      subGraph.Group,
			Orbit:      0,
			OrbitIndex: 0,
		}
		raw.Sd = []string{}
		s.registerSubgraphNode(subGraph, raw)
	}

	indicies := map[int64]*Node{}

	makeJewel := func(nodeIndex, jewelIndex int64) {
		socket := s.findClusterSocket(proxyGroup, jewelIndex)
		if socket == nil {
			panic("tree: cluster socket index " + strconv.FormatInt(jewelIndex, 10) + " not found in group " + strconv.FormatInt(proxyGroup.ID, 10))
		}
		raw := &Node{
			Type:           NodeSocket,
			ID:             socket.ID,
			IDStr:          socket.IDStr,
			Name:           socket.Name,
			Icon:           socket.Icon,
			Group:          subGraph.Group,
			Orbit:          nodeOrbit,
			OrbitIndex:     nodeIndex,
			ExpansionJewel: socket.ExpansionJewel,
		}
		raw.Sd = []string{}
		s.registerSubgraphNode(subGraph, raw)
		indicies[nodeIndex] = raw
		if legacyProxyGroup != nil {
			if legacySocket := s.findClusterSocket(legacyProxyGroup, jewelIndex); legacySocket != nil {
				s.registerLegacyClusterNodeMap(legacySocket.ID, raw.ID)
			}
		}
	}

	// First pass: sockets.
	if clusterJewel.Size == "Large" && socketCount == 1 {
		makeJewel(6, 1)
	} else {
		if socketCount > len(clusterJewel.SocketIndicies) {
			panic("tree: " + strconv.Itoa(socketCount) + " cluster sockets exceed the " + clusterJewel.Size + " template (socket " + parentSocket.T.IDStr + ")")
		}
		getJewels := []int64{0, 2, 1}
		for i := 0; i < socketCount; i++ {
			makeJewel(int64(clusterJewel.SocketIndicies[i]), getJewels[i])
		}
	}

	// Second pass: notables.
	var notableIndexList []int64
	for _, idxF := range clusterJewel.NotableIndicies {
		nodeIndex := int64(idxF)
		if len(notableIndexList) == notableCount {
			break
		}
		if clusterJewel.Size == "Medium" {
			if socketCount == 0 && notableCount == 2 {
				if nodeIndex == 6 {
					nodeIndex = 4
				} else if nodeIndex == 10 {
					nodeIndex = 8
				}
			} else if nodeCount == 4 {
				if nodeIndex == 10 {
					nodeIndex = 9
				} else if nodeIndex == 2 {
					nodeIndex = 3
				}
			}
		}
		if indicies[nodeIndex] == nil {
			notableIndexList = append(notableIndexList, nodeIndex)
		}
	}
	sort.Slice(notableIndexList, func(i, j int) bool { return notableIndexList[i] < notableIndexList[j] })

	for index, baseNode := range notableList {
		if index >= len(notableIndexList) {
			break
		}
		nodeIndex := notableIndexList[index]
		raw := &Node{
			Type:       NodeNotable,
			ID:         nodeID + nodeIndex,
			IDStr:      strconv.FormatInt(nodeID+nodeIndex, 10),
			Name:       baseNode.Name,
			Icon:       baseNode.Icon,
			Group:      subGraph.Group,
			Orbit:      nodeOrbit,
			OrbitIndex: nodeIndex,
		}
		raw.Sd = baseNode.Sd
		s.registerSubgraphNode(subGraph, raw)
		indicies[nodeIndex] = raw
	}

	// Third pass: small fill.
	var smallIndexList []int64
	for _, idxF := range clusterJewel.SmallIndicies {
		nodeIndex := int64(idxF)
		if len(smallIndexList) == smallCount {
			break
		}
		if clusterJewel.Size == "Medium" {
			if nodeCount == 5 && nodeIndex == 4 {
				nodeIndex = 3
			} else if nodeCount == 4 {
				if nodeIndex == 8 {
					nodeIndex = 9
				} else if nodeIndex == 4 {
					nodeIndex = 3
				}
			}
		}
		if indicies[nodeIndex] == nil {
			smallIndexList = append(smallIndexList, nodeIndex)
		}
	}
	for index := 0; index < smallCount; index++ {
		if index >= len(smallIndexList) {
			break
		}
		nodeIndex := smallIndexList[index]
		raw := &Node{
			Type:       NodeNormal,
			ID:         nodeID + nodeIndex,
			IDStr:      strconv.FormatInt(nodeID+nodeIndex, 10),
			Name:       skill.Name,
			Icon:       skill.Icon,
			Group:      subGraph.Group,
			Orbit:      nodeOrbit,
			OrbitIndex: nodeIndex,
		}
		raw.Sd = append([]string{}, skill.Stats...)
		raw.Sd = append(raw.Sd, jdata.ClusterJewelAddedMods...)
		s.registerSubgraphNode(subGraph, raw)
		indicies[nodeIndex] = raw
	}

	if indicies[0] == nil {
		panic("tree: subgraph " + strconv.FormatInt(nodeID, 10) + " has no entrance node")
	}

	// Convert template index space into tree orbit index space.
	skillsPerOrbit := s.Tree.SkillsPerOrbit[sizeIndex+1]
	startOidx := int64(data.ClusterJewels.OrbitOffsets[proxyNode.ID][int(sizeIndex)])
	totalIndicies := int64(clusterJewel.TotalIndicies)
	for _, raw := range indicies {
		corrected := (raw.OrbitIndex + startOidx) % totalIndicies
		raw.OrbitIndex = translateClusterOrbitIndex(corrected, totalIndicies, skillsPerOrbit)
	}
	s.buildLegacyClusterOrbitMappings(indicies, proxyNode, totalIndicies, skillsPerOrbit)

	// Process: positions, mods, cluster jewel effect.
	incEffect := jdata.ClusterJewelIncEffect
	for _, node := range subGraph.Nodes {
		s.Tree.ProcessNode(node.T)
		node.resetToSource(node.T)
		if incEffect != 0 && node.T.Type == NodeNormal {
			node.Stats.ModList = append(node.Stats.ModList, modparser.NewMod("PassiveSkillEffect", modparser.Inc, modparser.Num(incEffect)))
		}
	}

	// Connectors: template-index ring order (the map keys stay in template
	// space; only node.oidx was rewritten), closing the loop on non-small
	// clusters.
	var firstNode, lastNode *SpecNode
	specByRaw := map[*Node]*SpecNode{}
	for _, node := range subGraph.Nodes {
		specByRaw[node.T] = node
	}
	for i := int64(0); i < totalIndicies; i++ {
		raw := indicies[i]
		if raw == nil {
			continue
		}
		thisNode := specByRaw[raw]
		if firstNode == nil {
			firstNode = thisNode
		}
		if lastNode != nil {
			linkSpecNodes(thisNode, lastNode)
		}
		lastNode = thisNode
	}
	if firstNode != lastNode && clusterJewel.Size != "Small" {
		linkSpecNodes(firstNode, lastNode)
	}
	entrance := specByRaw[indicies[0]]
	subGraph.EntranceNode = entrance
	linkSpecNodes(entrance, parentSocket)

	// Register the synthetic nodes and recurse into smaller sockets.
	for _, node := range subGraph.Nodes {
		s.Nodes[node.ID()] = node
		if node.T.Type == NodeSocket {
			if socketJewel := s.getSocketedJewel(node.ID()); socketJewel != nil && jewelData(socketJewel).ClusterJewelValid {
				s.buildSubgraph(socketJewel, node, id, upSize)
			}
		}
	}
}
