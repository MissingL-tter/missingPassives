// The alternate passive pool — every conquering jewel family's replacement
// nodes and stat additions (the reference ships it as
// Data/TimelessJewelData/LegionPassives and calls it tree.legion, though it
// holds the abyss families too). Loaded the way the constructor's legion
// section does: abyss notables renamed, nodes typed and stat-parsed,
// keystones added to keystoneMap when the tree has none of that name.
package tree

import (
	"regexp"

	"github.com/MissingL-tter/missingPassives/data"
)

// ConqueredStatDesc is one entry of an alternate node's/addition's stats table:
// the roll range and format the timeless substitution uses.
type ConqueredStatDesc struct {
	ID    string  `json:"id"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Fmt   string  `json:"fmt"`
	Index int     `json:"index"` // 1-based position in the LUT roll list
}

// ConqueredAddition is one pool additions entry (a stat addition a conquering
// jewel can roll onto a node).
type ConqueredAddition struct {
	ID          string               `json:"id"`
	Name        string               `json:"dn"`
	Sd          []string             `json:"sd"`
	StatDescs   []*ConqueredStatDesc `json:"stats"`
	SortedStats []string             `json:"sortedStats"`
}

type conqueredNodeDoc struct {
	ID          string               `json:"id"`
	Icon        string               `json:"icon"`
	Ks          bool                 `json:"ks"`
	Not         bool                 `json:"not"`
	M           bool                 `json:"m"`
	Name        string               `json:"dn"`
	OrbitIndex  int64                `json:"oidx"`
	Sd          []string             `json:"sd"`
	StatDescs   []*ConqueredStatDesc `json:"stats"`
	SortedStats []string             `json:"sortedStats"`
}

// ConqueredPassives holds the loaded pool. The ordered slices preserve the file's
// array order — the reference indexes legion.nodes/additions numerically
// (its name for the pool)
// (conqueredNodes[77], conqueredAdditions[globalId + 1]).
type ConqueredPassives struct {
	Nodes            map[string]*Node
	Additions        map[string]*ConqueredAddition
	NodesOrdered     []*Node
	AdditionsOrdered []*ConqueredAddition
}

var (
	abyssIDRe = regexp.MustCompile(`^abyss_.+_notable_\d+$`)
	abyssDNRe = regexp.MustCompile(`^Notable \d+$`)
)

func (t *Tree) loadConqueredPassives() {
	var doc struct {
		Nodes     []*conqueredNodeDoc  `json:"nodes"`
		Additions []*ConqueredAddition `json:"additions"`
	}
	data.RawDoc("conqueredpassives", &doc)
	t.ConqueredPassives = &ConqueredPassives{Nodes: map[string]*Node{}, Additions: map[string]*ConqueredAddition{}}

	// The game numbers abyss notables; use the manual name for their first
	// stat, or the full first stat line when no name exists yet.
	for _, addition := range doc.Additions {
		if abyssIDRe.MatchString(addition.ID) && abyssDNRe.MatchString(addition.Name) {
			if len(addition.SortedStats) > 0 {
				if name, ok := data.AbyssNotableNames[addition.SortedStats[0]]; ok {
					addition.Name = name
				} else if len(addition.Sd) > 0 {
					addition.Name = addition.Sd[0]
				}
			} else if len(addition.Sd) > 0 {
				addition.Name = addition.Sd[0]
			}
		}
		t.ConqueredPassives.Additions[addition.ID] = addition
		t.ConqueredPassives.AdditionsOrdered = append(t.ConqueredPassives.AdditionsOrdered, addition)
	}

	for _, nd := range doc.Nodes {
		node := &Node{
			IDStr:      nd.ID,
			Name:       nd.Name,
			Icon:       nd.Icon,
			OrbitIndex: nd.OrbitIndex,
			Keystone:   nd.Ks,
			Notable:    nd.Not,
			Mastery:    nd.M,
		}
		node.Sd = append([]string{}, nd.Sd...)
		node.SortedStats = nd.SortedStats
		node.StatDescs = nd.StatDescs
		switch {
		case nd.M:
			node.Type = NodeMastery
		case nd.Ks:
			node.Type = NodeKeystone
			// Don't override good tree data with legacy keystones.
			if t.KeystoneMap[node.Name] == nil {
				t.KeystoneMap[node.Name] = node
			}
		case nd.Not:
			node.Type = NodeNotable
		default:
			node.Type = NodeNormal
		}
		t.processNodeStats(node)
		t.ConqueredPassives.Nodes[nd.ID] = node
		t.ConqueredPassives.NodesOrdered = append(t.ConqueredPassives.NodesOrdered, node)
	}
}
