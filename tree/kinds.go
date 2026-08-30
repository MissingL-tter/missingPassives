// Closed vocabularies of the tree: node kinds, tattoo override kinds, abyss
// modification components, the conquering-jewel families and the fixed
// slots of the alternate passive pool.
package tree

import "github.com/MissingL-tter/missingPassives/modparser"

// NodeKind is node.type: the constructor's classification chain.
type NodeKind uint8

const (
	NodeNormal NodeKind = iota + 1
	NodeClassStart
	NodeAscendClassStart
	NodeMastery
	NodeSocket
	NodeKeystone
	NodeNotable
)

var nodeKindNames = [...]string{"", "Normal", "ClassStart", "AscendClassStart", "Mastery", "Socket", "Keystone", "Notable"}

// String is the reference's node.type text.
func (k NodeKind) String() string { return nodeKindNames[k] }

// IsStart is ClassStart or AscendClassStart: the anchors the dependency
// search walks toward.
func (k NodeKind) IsStart() bool { return k == NodeClassStart || k == NodeAscendClassStart }

// OverrideKind is a tattoo row's overrideType (open: the export document's
// text, used as a counting key).
type OverrideKind string

// AlternateMastery is the runegraft override: its rows carry their own
// display name.
const AlternateMastery OverrideKind = "AlternateMastery"

// AbyssComponentKind tells what one abyss modification component does.
type AbyssComponentKind uint8

const (
	ComponentReplace AbyssComponentKind = 1 // the node becomes the pool node
	ComponentAdd     AbyssComponentKind = 2 // the pool addition's stats join the node
)

// Conquest is a spec node's conqueredBy record: the jewel that conquers it.
// Timeless jewels carry the seed; abyss jewels carry the computed
// modification components.
type Conquest struct {
	Seed      float64
	Conqueror modparser.ConquerorKind // 0 when the jewel's conqueror name was unknown
	ConqID    string                  // the conqueror's id text: the "<family>_keystone_<id>" match tail
	Abyss     []AbyssComponent
}

// conquestOf builds the record for a jewel's conqueredBy.
func conquestOf(cq *modparser.ConqueredBy) *Conquest {
	c := &Conquest{Seed: cq.Seed}
	if cq.Conqueror != nil {
		c.Conqueror = cq.Conqueror.Kind
		c.ConqID = cq.Conqueror.IDText()
	}
	return c
}

// isAbyss is the abyss family half of ConquerorKind.
func isAbyss(k modparser.ConquerorKind) bool { return k >= modparser.ConquerorAbyssMurderous }

// lutType is the family's LUT number (data.TimelessJewelSeedMin index,
// conquer table version key). An unknown conqueror reads the eternal
// tables (the reference's `or 5`).
func lutType(k modparser.ConquerorKind) int {
	if k == 0 {
		return int(modparser.ConquerorEternal)
	}
	return int(k)
}

// Fixed 1-based slots of the alternate passive pool (conqueredNode1).
const (
	poolMightOfTheVaal      = 77
	poolLegacyOfTheVaal     = 78
	poolTemplarDevotionNode = 91
	poolEternalSmallBlank   = 110
)

// poolNodeBase is the global id of the first pool node: LUT ids at or above
// it are replacements, below it additions.
const poolNodeBase = 337

// ExpansionJewel marks a cluster jewel socket node.
type ExpansionJewel struct {
	Size   int64  // 0 small, 1 medium, 2 large
	Index  int64  // slot in the parent socket
	Proxy  int64  // the ring's proxy node
	Parent *int64 // enclosing socket; nil on the outer large sockets
}
