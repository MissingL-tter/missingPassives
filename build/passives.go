package build

import (
	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/tree"
)

// Passives resolves the passives an item names rather than references by
// id, against the spec's tree. It implements calc.PassiveLookup.
//
// The reference falls back to the application's latest loaded tree when
// the spec's own tree does not know the name; one tree version ships
// here, so Latest defaults to the spec's.
type Passives struct {
	Spec   *tree.Spec
	Latest *tree.Tree
}

func (p Passives) latest() *tree.Tree {
	if p.Latest != nil {
		return p.Latest
	}
	return p.Spec.Tree
}

// GrantedPassive resolves an anointed passive: the spec tree's notables,
// then its ascendancy nodes, preferring the spec's own instance of the
// node (which carries conquered-jewel rewrites), then the latest tree's
// ascendancy names.
func (p Passives) GrantedPassive(name string) *calc.NodeInput {
	node := p.Spec.Tree.NotableMap[name]
	if node == nil {
		node = p.Spec.Tree.AscendancyMap[name]
	}
	if node != nil {
		if specNode := p.Spec.Nodes[node.ID]; specNode != nil {
			return SpecNodeInput(specNode)
		}
		return TreeNodeInput(node)
	}
	if node := p.latest().AscendancyMap[name]; node != nil {
		return TreeNodeInput(node)
	}
	return nil
}

// GrantedAscendancyNode resolves a Forbidden Flame/Flesh node name against
// the spec tree's ascendancy nodes, then the latest tree's. Unlike an
// anoint, this one always takes the bare tree node: the reference does not
// look for a spec instance here.
func (p Passives) GrantedAscendancyNode(name string) *calc.NodeInput {
	node := p.Spec.Tree.AscendancyMap[name]
	if node == nil {
		node = p.latest().AscendancyMap[name]
	}
	if node == nil {
		return nil
	}
	return TreeNodeInput(node)
}
