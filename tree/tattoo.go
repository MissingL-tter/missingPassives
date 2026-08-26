// The tattoo override pool: the committed Data/TattooPassives.lua (dumped
// to data/raw/tattooOverrides.json — NOT the GGPK export pipeline's
// tattooPassives.json, which differs in shape), loaded and stat-parsed the
// way the PassiveTree constructor's tattoo loop does.
package tree

import (
	"github.com/MissingL-tter/missingPassives/data"
)

// Tattoo holds the override nodes keyed by display name (the Overrides
// loader can register extra aliases when a saved name was renamed).
type Tattoo struct {
	Nodes map[string]*Node
}

func (t *Tree) loadTattoo() {
	var raw map[string]any
	data.RawDoc("tattooOverrides", &raw)
	t.Tattoo = &Tattoo{Nodes: map[string]*Node{}}
	nodes, _ := raw["nodes"].(map[string]any)
	for name, nv := range nodes {
		nm := nv.(map[string]any)
		node := &Node{
			IDStr:           str(nm["id"]),
			Name:            str(nm["dn"]),
			Icon:            str(nm["icon"]),
			Keystone:        boolean(nm["ks"]),
			Notable:         boolean(nm["not"]),
			Mastery:         boolean(nm["m"]),
			IsTattooFlag:    boolean(nm["isTattoo"]),
			OverrideTypeStr: str(nm["overrideType"]),
			Raw:             nm,
		}
		// A few overrides (the alternate masteries) carry their own name
		// field; it shadows the tattooed node's original name.
		if ownName, ok := nm["name"].(string); ok {
			node.NameStr = &ownName
		}
		node.Sd = strList(nm["sd"])
		switch {
		case node.Mastery:
			node.Type = "Mastery"
		case node.Keystone:
			node.Type = "Keystone"
		case node.Notable:
			node.Type = "Notable"
		default:
			node.Type = "Normal"
		}
		processStats(&node.Stats, node.IDStr, 0)
		t.Tattoo.Nodes[name] = node
	}
}
