// BuildAllDependsAndPaths (PassiveSpec.lua L1098) and its helpers: the
// dependency/pruning analysis over the allocation graph, radius-jewel
// rules, mastery effect application, and the path/distance rebuilds.
package tree

import (
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// collectAbyssConquests ports the abyss pre-pass: read each socketed
// abyss-conquering jewel's affected nodes before the nodes are reset
// (Zorath needs the currently allocated path from its socket).
func (s *Spec) collectAbyssConquests(jewelIDs []int64) map[int64]*Conquest {
	conquests := map[int64]*Conquest{}
	for _, socketID := range jewelIDs {
		it := s.jewel(s.Jewels[socketID])
		cq := jewelConquest(it)
		if cq == nil || s.AllocNodes[socketID] == nil || jewelData(it).LimitDisabled || !isAbyss(cq.Conqueror) {
			continue
		}
		var path map[int64]bool
		if cq.Conqueror == modparser.ConquerorAbyssSpecial {
			path = s.getShortestPathToClassStart(socketID)
		}
		for nodeID, modification := range s.Tree.AbyssPassive(int64(cq.Seed), socketID, int(cq.Conqueror), path) {
			conquests[nodeID] = &Conquest{Seed: cq.Seed, Conqueror: cq.Conqueror, ConqID: cq.ConqID, Abyss: modification}
		}
	}
	return conquests
}

// BuildAllDependsAndPaths ports the same-named method.
func (s *Spec) BuildAllDependsAndPaths() {
	var visited []*SpecNode

	jewelIDs := sortedNodeIDs(s.Jewels)
	abyssConquests := s.collectAbyssConquests(jewelIDs)

	// First pass: reset every node to its tree state and apply radius-jewel
	// flags.
	nodeIDs := sortedNodeIDs(s.Nodes)
	for _, id := range nodeIDs {
		node := s.Nodes[id]
		node.Depends = node.Depends[:0]
		node.IntuitiveLeapLikesAffecting = nil
		node.ConqueredBy = nil
		if treeNode := s.Tree.Nodes[id]; treeNode != nil {
			s.replaceNode(node, treeNode)
		}
		if cq := abyssConquests[id]; cq != nil {
			node.ConqueredBy = cq
		}
		if node.Type() != NodeClassStart && node.Type() != NodeSocket && node.AscendancyName() == "" {
			for _, socketID := range jewelIDs {
				itemID := s.Jewels[socketID]
				it := s.jewel(itemID)
				if it == nil || it.JewelRadiusIndex == nil || s.AllocNodes[socketID] == nil ||
					it.JewelData == nil || it.JewelData.LimitDisabled {
					continue
				}
				jd := it.JewelData
				radiusIndex := *it.JewelRadiusIndex
				socketNode := s.Nodes[socketID]
				if socketNode != nil && socketNode.T.NodesInRadius != nil &&
					socketNode.T.NodesInRadius[radiusIndex-1][node.ID()] != nil {
					if itemID != 0 {
						if jd.IntuitiveLeapLike &&
							!(jd.IntuitiveLeapKeystoneOnly && node.Type() != NodeKeystone) {
							node.IntuitiveLeapLikesAffecting = append(node.IntuitiveLeapLikesAffecting, s.Nodes[socketID])
						}
						if cq := jewelConquest(it); cq != nil && !isAbyss(cq.Conqueror) {
							node.ConqueredBy = cq
						}
					}
				}
				if jd.ImpossibleEscapeKeystone != "" {
					keystones := jd.ImpossibleEscapeKeystones
					for _, keyName := range sortedStringKeys(s.Tree.KeystoneMap) {
						keyNode := s.Tree.KeystoneMap[keyName]
						if keystones[keyName] && keyNode.NodesInRadius != nil &&
							keyNode.NodesInRadius[radiusIndex-1][node.ID()] != nil {
							node.IntuitiveLeapLikesAffecting = append(node.IntuitiveLeapLikesAffecting, s.Nodes[socketID])
						}
					}
				}
			}
		}
		if node.Alloc {
			node.Depends = append(node.Depends, node)
		}
	}

	// Second pass: tattoo overrides, then timeless conquering.
	for _, id := range nodeIDs {
		node := s.Nodes[id]
		if ov := s.HashOverrides[node.ID()]; ov != nil {
			s.replaceNode(node, ov)
		}
		s.applyConquered(node)
	}

	// Third pass: mastery effects and the allocation counts.
	s.AllocatedMasteryCount = 0
	s.AllocatedNotableCount = 0
	s.AllocatedKeystoneCount = 0
	s.AllocatedMasteryTypes = map[string]float64{}
	s.AllocatedMasteryTypeCount = 0
	s.AllocatedTattooTypes = map[OverrideKind]float64{}
	for _, id := range nodeIDs {
		node := s.Nodes[id]
		if node.Type() == NodeMastery && s.MasterySelections[id] != 0 {
			effect := s.Tree.MasteryEffects[s.MasterySelections[id]]
			if effect != nil && s.AllocNodes[id] != nil {
				if ov := s.HashOverrides[id]; ov != nil {
					s.replaceNode(node, ov)
				} else {
					node.Stats.Sd = effect.Sd
					node.Stats.cloneSd()
					node.src, node.srcME = nil, effect
				}
				node.AllMasteryOptions = false
				node.ReminderText = []string{"Tip: Right click to select a different effect"}
				processStats(&node.Stats, strconv.FormatInt(id, 10), 0)
				s.AllocatedMasteryCount++
				name := node.nameForCounts()
				if s.AllocatedMasteryTypes[name] == 0 {
					s.AllocatedMasteryTypeCount++
				}
				s.AllocatedMasteryTypes[name]++
			} else {
				node.Alloc = false
				delete(s.AllocNodes, id)
				delete(s.MasterySelections, id)
			}
		} else if node.Type() == NodeMastery {
			s.addMasteryEffectOptionsToNode(node)
		} else if node.Type() == NodeNotable && node.Alloc {
			s.AllocatedNotableCount++
		} else if node.Type() == NodeKeystone && node.Alloc {
			s.AllocatedKeystoneCount++
		}
		if node.IsTattoo && node.Alloc && node.OverrideType != "" {
			s.AllocatedTattooTypes[node.OverrideType]++
		}
	}

	// Fourth pass: dependencies and orphan pruning.
	potentialDeps := map[int64][]*SpecNode{}
	var potentialOrder []int64
	intuitiveLeaps := map[int64][]*SpecNode{}
	var leapOrder []int64
	for _, id := range sortedNodeIDs(s.AllocNodes) {
		node := s.AllocNodes[id]
		if node == nil || !node.Alloc {
			continue // pruned by an earlier iteration
		}
		node.Visited = true
		node.ConnectedToStart = false
		anyStartFound := node.Type().IsStart()
		for _, other := range node.Linked {
			if !other.Alloc || dependsContains(node.Depends, other) {
				continue
			}
			if other.Type().IsStart() {
				anyStartFound = true
				node.ConnectedToStart = true
			} else if s.findStartFromNode(other, &visited, false) {
				anyStartFound = true
				node.ConnectedToStart = true
				for _, n := range visited {
					n.Visited = false
				}
				visited = visited[:0]
			} else {
				depIDs := map[int64]bool{}
				for _, n := range visited {
					if len(n.IntuitiveLeapLikesAffecting) == 0 {
						depIDs[n.ID()] = true
					}
				}
				for _, n := range visited {
					if n.Type() == NodeMastery {
						otherPath := false
						allocatedLinkCount := 0
						for _, linkedNode := range n.Linked {
							if linkedNode.Alloc {
								allocatedLinkCount++
							}
						}
						if allocatedLinkCount > 1 {
							for _, linkedNode := range n.Linked {
								if linkedNode.Alloc && !depIDs[linkedNode.ID()] {
									otherPath = true
								}
							}
						}
						if !otherPath {
							node.Depends = append(node.Depends, n)
						}
					} else {
						if len(n.IntuitiveLeapLikesAffecting) > 0 {
							if potentialDeps[n.ID()] == nil {
								potentialOrder = append(potentialOrder, n.ID())
							}
							potentialDeps[n.ID()] = append(potentialDeps[n.ID()], node)
						} else {
							node.Depends = append(node.Depends, n)
						}
						if intuitiveLeaps[node.ID()] == nil {
							leapOrder = append(leapOrder, node.ID())
							intuitiveLeaps[node.ID()] = s.nodesInIntuitiveLeapLikeRadius(n)
						} else {
							intuitiveLeaps[node.ID()] = append(intuitiveLeaps[node.ID()], s.nodesInIntuitiveLeapLikeRadius(n)...)
						}
					}
					n.Visited = false
				}
				visited = visited[:0]
			}
		}
		node.Visited = false
		if !anyStartFound {
			for _, depNode := range node.Depends {
				prune := true
				for _, socketID := range jewelIDs {
					itemID := s.Jewels[socketID]
					if s.AllocNodes[socketID] == nil || itemID == 0 {
						continue
					}
					it := s.jewel(itemID)
					if it == nil {
						continue
					}
					leapCovers := jewelData(it).IntuitiveLeapLike && it.JewelRadiusIndex != nil &&
						s.Nodes[socketID] != nil && s.Nodes[socketID].T.NodesInRadius != nil &&
						s.Nodes[socketID].T.NodesInRadius[*it.JewelRadiusIndex-1][depNode.ID()] != nil
					escapeCovers := false
					if keystones := jewelData(it).ImpossibleEscapeKeystones; keystones != nil && it.JewelRadiusIndex != nil {
						escapeCovers = s.nodeInKeystoneRadius(keystones, depNode.ID(), *it.JewelRadiusIndex)
					}
					if leapCovers || escapeCovers {
						prune = false
						if intuitiveLeaps[socketID] == nil {
							leapOrder = append(leapOrder, socketID)
						}
						intuitiveLeaps[socketID] = append(intuitiveLeaps[socketID], depNode)
						break
					}
				}
				if prune {
					s.deallocSingleNode(depNode)
				}
			}
		}
	}

	// Resolve dependencies for nodes affected by intuitive-leap-like
	// jewels.
	for _, id := range potentialOrder {
		deps := potentialDeps[id]
		potentialNode := s.Nodes[id]
		seen := map[int64]bool{}
		for _, node := range deps {
			allDep := true
			for _, provider := range potentialNode.IntuitiveLeapLikesAffecting {
				if !dependsContains(node.Depends, provider) {
					allDep = false
				}
			}
			if allDep && !seen[node.ID()] {
				node.Depends = append(node.Depends, potentialNode)
				seen[node.ID()] = true
			}
		}
	}
	for _, id := range leapOrder {
		deps := intuitiveLeaps[id]
		node := s.Nodes[id]
		seen := map[int64]bool{}
		for _, dep := range deps {
			if dep.ConnectedToStart {
				continue
			}
			allDep := true
			for _, intuitiveDep := range dep.IntuitiveLeapLikesAffecting {
				if !dependsContains(node.Depends, intuitiveDep) {
					allDep = false
					break
				}
			}
			if allDep && !seen[dep.ID()] {
				node.Depends = append(node.Depends, dep)
				seen[dep.ID()] = true
			}
		}
	}

	// Rebuild the paths and socket distances.
	for _, id := range nodeIDs {
		node := s.Nodes[id]
		if node.Alloc && len(node.IntuitiveLeapLikesAffecting) == 0 {
			node.PathDist = 0
		} else {
			node.PathDist = 1000
		}
		node.Path = nil
		node.HasPath = false
		if node.T.JewelSocket || node.T.ExpansionJewel != nil {
			zero := 0.0
			node.DistanceToClassStart = &zero
		}
	}
	for _, id := range sortedNodeIDs(s.AllocNodes) {
		node := s.AllocNodes[id]
		if len(node.IntuitiveLeapLikesAffecting) == 0 || node.ConnectedToStart {
			s.buildPathFromNode(node)
			if node.T.JewelSocket || node.T.ExpansionJewel != nil {
				s.setNodeDistanceToClassStart(node)
			}
		}
	}

	s.buildSplitPersonalityPath()
}

func (n *SpecNode) nameForCounts() string {
	if name := n.EffectiveName(); name != nil {
		return *name
	}
	return n.Dn
}

func (s *Spec) deallocSingleNode(node *SpecNode) {
	node.Alloc = false
	delete(s.AllocNodes, node.ID())
	if node.Type() == NodeMastery {
		s.addMasteryEffectOptionsToNode(node)
		delete(s.MasterySelections, node.ID())
	}
}

// addMasteryEffectOptionsToNode ports the same-named method: unallocated
// masteries show every effect option.
func (s *Spec) addMasteryEffectOptionsToNode(node *SpecNode) {
	node.Stats.Sd = []string{}
	node.src, node.srcME = nil, nil
	if len(node.T.MasteryEffects) > 0 {
		for _, effectRef := range node.T.MasteryEffects {
			effect := s.Tree.MasteryEffects[effectRef.Effect]
			startIndex := len(node.Stats.Sd) // 0 on the first effect = full reset
			node.Stats.Sd = append(node.Stats.Sd, effect.Sd...)
			processStats(&node.Stats, strconv.FormatInt(node.ID(), 10), startIndex)
		}
	} else {
		processStats(&node.Stats, strconv.FormatInt(node.ID(), 10), 0)
	}
	node.AllMasteryOptions = true
}

// findStartFromNode ports FindStartFromNode.
func (s *Spec) findStartFromNode(node *SpecNode, visited *[]*SpecNode, noAscend bool) bool {
	node.Visited = true
	*visited = append(*visited, node)
	for _, other := range node.Linked {
		startIndex := len(*visited)
		if other.Alloc &&
			(other.Type().IsStart() ||
				(!other.Visited && node.Type() != NodeMastery && s.findStartFromNode(other, visited, noAscend))) {
			if node.AscendancyName() != "" && other.AscendancyName() == "" {
				// Pathing out of Ascendant: un-visit the outside nodes.
				for i := startIndex; i < len(*visited); i++ {
					(*visited)[i].Visited = false
				}
				*visited = (*visited)[:startIndex]
			} else if !noAscend || other.Type() != NodeAscendClassStart {
				return true
			}
		}
	}
	return false
}

// buildPathFromNode ports BuildPathFromNode.
func (s *Spec) buildPathFromNode(root *SpecNode) {
	root.PathDist = 0
	root.Path = []*SpecNode{}
	root.HasPath = true
	queue := []*SpecNode{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		curDist := node.PathDist
		for index, other := range node.Linked {
			// Cluster rebuilds can replace node objects; normalize stale
			// links to the canonical object.
			canonical := s.Nodes[other.ID()]
			if canonical == nil {
				continue
			}
			if canonical != other {
				node.Linked[index] = canonical
				other = canonical
			}
			otherPathDist := other.PathDist
			if node.Type() != NodeMastery && !other.Type().IsStart() &&
				otherPathDist > curDist &&
				(node.AscendancyName() == other.AscendancyName() || (curDist == 0 && other.AscendancyName() == "")) {
				other.PathDist = curDist
				if !other.Alloc {
					other.PathDist++
				}
				other.Path = append([]*SpecNode{other}, node.Path...)
				other.HasPath = true
				queue = append(queue, other)
			}
		}
	}
}

// setNodeDistanceToClassStart ports SetNodeDistanceToClassStart.
func (s *Spec) setNodeDistanceToClassStart(root *SpecNode) {
	zero := 0.0
	root.DistanceToClassStart = &zero
	if !root.Alloc || !root.ConnectedToStart {
		return
	}
	targetNodeID := s.CurClass.StartNodeID
	dist := map[int64]int{root.ID(): 0}
	queue := []*SpecNode{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		curDist := dist[node.ID()] + 1
		for _, other := range node.Linked {
			if other.ID() == targetNodeID {
				d := float64(curDist - 1)
				root.DistanceToClassStart = &d
				return
			}
			if other.Alloc && node.Type() != NodeMastery && !other.Type().IsStart() {
				if _, seen := dist[other.ID()]; !seen {
					dist[other.ID()] = curDist
					queue = append(queue, other)
				}
			}
		}
	}
}

// getShortestPathToClassStart ports GetShortestPathToClassStart: the set
// of node ids on the shortest allocated path, or nil.
func (s *Spec) getShortestPathToClassStart(rootID int64) map[int64]bool {
	root := s.Nodes[rootID]
	if root == nil || !root.Alloc {
		return nil
	}
	targetNodeID := s.CurClass.StartNodeID
	parent := map[int64]*SpecNode{}
	queue := []*SpecNode{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, other := range node.Linked {
			if other.ID() == targetNodeID {
				path := map[int64]bool{root.ID(): true, other.ID(): true}
				cur := node
				for cur != nil {
					path[cur.ID()] = true
					cur = parent[cur.ID()]
				}
				return path
			}
			if other.Alloc && node.Type() != NodeMastery && !other.Type().IsStart() &&
				parent[other.ID()] == nil && other.ID() != root.ID() {
				parent[other.ID()] = node
				queue = append(queue, other)
			}
		}
	}
	return nil
}

// nodesInIntuitiveLeapLikeRadius ports the same-named method.
func (s *Spec) nodesInIntuitiveLeapLikeRadius(node *SpecNode) []*SpecNode {
	var result []*SpecNode
	itemID := s.Jewels[node.ID()]
	if itemID <= 0 {
		return result
	}
	it := s.jewel(itemID)
	if it == nil {
		return result
	}
	if it.JewelRadiusIndex == nil {
		return result
	}
	radiusIndex := *it.JewelRadiusIndex
	if jewelData(it).IntuitiveLeapLike {
		socketNode := s.Nodes[node.ID()]
		if socketNode != nil && socketNode.T.NodesInRadius != nil {
			for _, affectedID := range sortedNodeIDs(socketNode.T.NodesInRadius[radiusIndex-1]) {
				if affected := s.Nodes[affectedID]; affected != nil && affected.Alloc {
					result = append(result, affected)
				}
			}
		}
	}
	if jewelData(it).ImpossibleEscapeKeystone != "" {
		keystones := jewelData(it).ImpossibleEscapeKeystones
		for _, keyName := range sortedStringKeys(keystones) {
			keyNode := s.Tree.KeystoneMap[keyName]
			if keyNode != nil && keyNode.NodesInRadius != nil {
				for _, affectedID := range sortedNodeIDs(keyNode.NodesInRadius[radiusIndex-1]) {
					if affected := s.Nodes[affectedID]; affected != nil && affected.Alloc {
						result = append(result, affected)
					}
				}
			}
		}
	}
	return result
}

// nodeInKeystoneRadius ports NodeInKeystoneRadius (lowercased names).
func (s *Spec) nodeInKeystoneRadius(keystoneNames map[string]bool, nodeID int64, radiusIndex int) bool {
	for _, node := range s.Nodes {
		// node.name through the metatable: a conquered keystone's shadow name
		// is nil (alternate nodes have only dn) and falls through to the tree's.
		name := node.EffectiveName()
		if node.Type() == NodeKeystone && name != nil && keystoneNames[strings.ToLower(*name)] {
			if node.T.NodesInRadius != nil && node.T.NodesInRadius[radiusIndex-1][nodeID] != nil {
				return true
			}
		}
	}
	return false
}

func (s *Spec) buildSplitPersonalityPath() {
	splitPersonalityPath := map[int64]bool{}
	for _, socketID := range sortedNodeIDs(s.Jewels) {
		it := s.jewel(s.Jewels[socketID])
		if it != nil && jewelData(it).JewelIncEffectFromClassStart != 0 {
			if path := s.getShortestPathToClassStart(socketID); path != nil {
				for id := range path {
					splitPersonalityPath[id] = true
				}
			}
		}
	}
	s.SplitPersonalityPath = splitPersonalityPath
}

func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
