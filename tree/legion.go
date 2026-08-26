// The timeless-jewel passive pool (Data/TimelessJewelData/LegionPassives),
// loaded into the tree the way the constructor's legion section does:
// abyss notables renamed, nodes typed and stat-parsed, legion keystones
// added to keystoneMap when the tree has none of that name.
package tree

import (
	"regexp"

	"github.com/MissingL-tter/missingPassives/data"
)

// LegionAddition is one legion.additions entry (a stat addition a timeless
// jewel can roll onto a node).
type LegionAddition struct {
	ID          string           `json:"id"`
	Name        string           `json:"dn"`
	Sd          []string         `json:"sd"`
	StatDescs   []map[string]any `json:"stats"`
	SortedStats []string         `json:"sortedStats"`
}

type legionNodeDoc struct {
	ID          string           `json:"id"`
	Icon        string           `json:"icon"`
	Ks          bool             `json:"ks"`
	Not         bool             `json:"not"`
	M           bool             `json:"m"`
	Name        string           `json:"dn"`
	OrbitIndex  int64            `json:"oidx"`
	Sd          []string         `json:"sd"`
	StatDescs   []map[string]any `json:"stats"`
	SortedStats []string         `json:"sortedStats"`
}

// Legion holds the loaded pool.
type Legion struct {
	Nodes     map[string]*Node
	Additions map[string]*LegionAddition
}

var (
	abyssIDRe = regexp.MustCompile(`^abyss_.+_notable_\d+$`)
	abyssDNRe = regexp.MustCompile(`^Notable \d+$`)
)

func (t *Tree) loadLegion() {
	var doc struct {
		Nodes     []*legionNodeDoc  `json:"nodes"`
		Additions []*LegionAddition `json:"additions"`
	}
	data.RawDoc("legionPassives", &doc)
	t.Legion = &Legion{Nodes: map[string]*Node{}, Additions: map[string]*LegionAddition{}}

	// The game numbers abyss notables; use the manual name for their first
	// stat, or the full first stat line when no name exists yet.
	for _, addition := range doc.Additions {
		if abyssIDRe.MatchString(addition.ID) && abyssDNRe.MatchString(addition.Name) {
			if len(addition.SortedStats) > 0 {
				if name, ok := data.AbyssNotableNames[addition.SortedStats[0]].(string); ok {
					addition.Name = name
				} else if len(addition.Sd) > 0 {
					addition.Name = addition.Sd[0]
				}
			} else if len(addition.Sd) > 0 {
				addition.Name = addition.Sd[0]
			}
		}
		t.Legion.Additions[addition.ID] = addition
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
		switch {
		case nd.M:
			node.Type = "Mastery"
		case nd.Ks:
			node.Type = "Keystone"
			// Don't override good tree data with legacy keystones.
			if t.KeystoneMap[node.Name] == nil {
				t.KeystoneMap[node.Name] = node
			}
		case nd.Not:
			node.Type = "Notable"
		default:
			node.Type = "Normal"
		}
		processStats(&node.Stats, node.IDStr, 0)
		t.Legion.Nodes[nd.ID] = node
	}
}
