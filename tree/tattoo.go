// The tattoo override pool, read straight from the export document
// (data/raw/tattoopassives.json — the same document the export differential
// proves renders byte-identical to the committed Data/TattooPassives.lua).
// The document is a row list; the runtime pool keys by display name with
// later rows overwriting, exactly as the Lua file's keyed constructor did.
package tree

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
)

// Tattoo holds the override nodes keyed by display name (the Overrides
// loader can register extra aliases when a saved name was renamed).
type Tattoo struct {
	Nodes map[string]*Node
}

func (t *Tree) loadTattoo() {
	var doc schema.TattooPassives
	data.RawDoc("tattoopassives", &doc)
	t.Tattoo = &Tattoo{Nodes: map[string]*Node{}}
	for i := range doc.Nodes {
		nd := &doc.Nodes[i]
		node := &Node{
			IDStr:             nd.Id,
			Name:              nd.Name,
			Icon:              nd.Icon,
			Keystone:          nd.Ks,
			Notable:           nd.Not,
			Mastery:           nd.M,
			IsTattooFlag:      true,
			OverrideTypeOf:    OverrideKind(nd.OverrideType),
			ActiveEffectImage: nd.ActiveEffectImage,
		}
		// The alternate masteries (runegrafts) carry their own name; it
		// shadows the tattooed node's original name.
		if node.OverrideTypeOf == AlternateMastery {
			ownName := "Runegraft Mastery"
			node.NameStr = &ownName
		}
		node.Sd = append([]string{}, nd.Sd...)
		switch {
		case node.Mastery:
			node.Type = NodeMastery
		case node.Keystone:
			node.Type = NodeKeystone
		case node.Notable:
			node.Type = NodeNotable
		default:
			node.Type = NodeNormal
		}
		t.Tattoo.Nodes[nd.Name] = node // later rows overwrite, like the Lua keys
	}
	// The reference processes stats over the keyed table — after the
	// duplicate-name collapse — so shadowed rows are never parsed.
	for _, node := range t.Tattoo.Nodes {
		processStats(&node.Stats, node.IDStr, 0)
	}
}
