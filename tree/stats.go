// processStats: PassiveTreeClass:ProcessStats (L769) — parse a node's stat
// lines into its mod list, with the reference's multiline splitting and
// line-combining behavior.
package tree

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

func parse3(line string) ([]*modparser.Mod, string, bool) {
	mods, extra := modparser.Parse(line)
	if mods == nil {
		return nil, extra, false
	}
	out := make([]*modparser.Mod, 0, len(mods))
	for _, mv := range mods {
		m, ok := mv.(*modparser.Mod)
		if !ok {
			panic("tree: non-Mod entry parsing stat line: " + line)
		}
		out = append(out, m)
	}
	return out, extra, true
}

// processStats parses s.Sd[startIndex:] and appends to the mod state.
// startIndex 0 resets (the reference's startIndex == 1); subgraph nodes
// re-enter with the old length to process appended lines only.
func processStats(s *Stats, srcID string, startIndex int) {
	if startIndex == 0 {
		s.ModKey = ""
		s.Mods = nil
		s.ModList = []*modparser.Mod{}
	}
	if s.Sd == nil {
		return
	}
	setMod := func(i int, m *NodeMod) {
		for len(s.Mods) <= i {
			s.Mods = append(s.Mods, nil)
		}
		s.Mods[i] = m
	}
	i := startIndex
	for i < len(s.Sd) {
		if strings.Contains(s.Sd[i], "\n") {
			parts := []string{}
			for _, part := range strings.Split(s.Sd[i], "\n") {
				if part != "" {
					parts = append(parts, part)
				}
			}
			s.Sd = append(s.Sd[:i], append(parts, s.Sd[i+1:]...)...)
		}
		line := s.Sd[i]
		list, extra, parsed := parse3(line)
		if !parsed || extra != "" {
			// Try combining with one or more following lines.
			endI := i + 1
			for endI < len(s.Sd) {
				comb := line
				for ci := i + 1; ci <= endI; ci++ {
					comb += " " + s.Sd[ci]
				}
				list, extra, parsed = parse3(comb)
				if parsed && extra == "" {
					// Dummy entries for the combined-in lines.
					for ci := i + 1; ci <= endI; ci++ {
						setMod(ci, &NodeMod{List: []*modparser.Mod{}})
					}
					break
				}
				endI++
			}
		}
		if !parsed {
			s.Unknown = true
		} else if extra != "" {
			s.Extra = true
		} else {
			for _, mod := range list {
				s.ModKey += "[" + modparser.FormatMod(mod) + "]"
			}
		}
		setMod(i, &NodeMod{List: list, Nil: !parsed, Extra: extra})
		i++
		for i < len(s.Mods) && s.Mods[i] != nil {
			i++
		}
	}
	for mi := startIndex; mi < len(s.Mods); mi++ {
		m := s.Mods[mi]
		if m == nil || m.Nil || m.Extra != "" {
			continue
		}
		for _, mod := range m.List {
			s.ModList = append(s.ModList, modparser.SetSource(mod, "Tree:"+srcID))
		}
	}
}

// processNodeStats runs processStats for one tree node, adding the
// keystone marker mod the reference attaches (source "Tree<id>", no colon
// — the reference's own concatenation).
func (t *Tree) processNodeStats(node *Node) {
	processStats(&node.Stats, node.IDStr, 0)
	if node.Type == "Keystone" {
		node.KeystoneMod = modparser.NewMod("Keystone", "LIST", node.Name, "Tree"+node.IDStr)
	}
}
