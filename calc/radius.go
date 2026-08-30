// Radius jewels: env.radiusJewelList construction (CalcSetup items loop
// L773-830) and the adapters that let modparser's ported jewel functions
// (jewels.go, keyed by exact mod line) run over calc's node and mod-list
// types. Functions are re-derived from the item's mod lines through
// modparser.Parse — the same parse that built them in the reference — and
// checked against the dump's recorded funcList types.
package calc

import (
	"fmt"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// RadiusJewel is one env.radiusJewelList entry.
type RadiusJewel struct {
	Nodes      map[int]string // node id -> node type (rad.nodes membership)
	Fn         modparser.JewelNodeFn
	Type       string
	Item       *Item
	NodeID     int
	Attributes map[string]float64
	Data       *modparser.JewelFuncTag
}

// jewelNodeRef adapts *NodeInput to modparser.JewelNodeRef.
type jewelNodeRef struct{ n *NodeInput }

func (r jewelNodeRef) ConqueredBy() bool { return r.n.ConqueredBy != nil && *r.n.ConqueredBy }
func (r jewelNodeRef) Type() string      { return r.n.Type }
func (r jewelNodeRef) IsTattoo() bool    { return r.n.IsTattoo != nil && *r.n.IsTattoo }
func (r jewelNodeRef) ModList() []*modparser.Mod {
	return r.n.ModList
}

// listWriter adapts *modstore.List to modparser.JewelStoreWriter.
type listWriter struct{ l *modstore.List }

func (w listWriter) Sum(typ modparser.ModType, names ...string) float64 {
	return w.l.Sum(typ, nil, names...)
}
func (w listWriter) AddMod(m *modparser.Mod)       { w.l.AddMod(m) }
func (w listWriter) MergeMod(m *modparser.Mod)     { w.l.MergeMod(m, false) }
func (w listWriter) AddList(list []*modparser.Mod) { w.l.AddList(list) }
func (w listWriter) Mods() []*modparser.Mod        { return w.l.Mods }

// defaultRadiusFunc is the fallback for radius jewels without a funcList:
// tally all attributes in radius (CalcSetup L775-782).
func defaultRadiusFunc(node modparser.JewelNodeRef, out modparser.JewelStoreWriter, data *modparser.JewelFuncTag) {
	if node != nil {
		for _, stat := range []string{"Str", "Dex", "Int"} {
			data.AddStat(stat, out.Sum(modparser.Base, stat))
		}
	}
}

// deriveFuncList rebuilds item.jewelData.funcList by parsing the item's
// active mod lines (implicit-family lines first, then explicits — the
// item parse order) and collecting the JewelFunc modifier values. The
// result is asserted against the dump's recorded types.
func deriveFuncList(item *Item) []struct {
	Typ string
	Fn  modparser.JewelNodeFn
} {
	var out []struct {
		Typ string
		Fn  modparser.JewelNodeFn
	}
	lines := append(append([]string{}, item.In.OtherLines...), item.In.ExplicitLines...)
	for _, line := range lines {
		// Item.lua parses continuation lines joined by a SPACE but stores
		// them joined by \n (Item.lua L1171-1176)
		mods, _, _ := modparser.Parse(strings.ReplaceAll(line, "\n", " "))
		for _, mod := range mods {
			if mod.Name != "JewelFunc" {
				continue
			}
			fn, ok := mod.Value.(modparser.JewelFn)
			if !ok || fn.Func == nil {
				panic("calc: JewelFunc value without a function for line " + line)
			}
			out = append(out, struct {
				Typ string
				Fn  modparser.JewelNodeFn
			}{Typ: fn.Type, Fn: fn.Func})
		}
	}
	// assert against the reference's recorded funcList types
	if len(out) != len(item.In.FuncTypes) {
		panic(fmt.Sprintf("calc: derived %d jewel funcs, reference had %d for %s",
			len(out), len(item.In.FuncTypes), item.In.Name))
	}
	for i, e := range out {
		if e.Typ != item.In.FuncTypes[i] {
			panic(fmt.Sprintf("calc: jewel func %d type %q != reference %q for %s",
				i, e.Typ, item.In.FuncTypes[i], item.In.Name))
		}
	}
	return out
}

// addRadiusJewel ports the radius-jewel registration (items loop L773-830,
// minus the override.extraJewelFuncs re-entry which stays unported).
func (env *Env) addRadiusJewel(slot *SlotInput, item *Item) {
	type entry = struct {
		Typ string
		Fn  modparser.JewelNodeFn
	}
	var funcList []entry
	if item.In.FuncTypes != nil {
		funcList = deriveFuncList(item)
	} else {
		funcList = []entry{{Typ: "Self", Fn: defaultRadiusFunc}}
	}
	for _, fn := range funcList {
		env.RadiusJewelList = append(env.RadiusJewelList, &RadiusJewel{
			Nodes:      slot.RadiusNodes,
			Fn:         fn.Fn,
			Type:       fn.Typ,
			Item:       item,
			NodeID:     int(*slot.NodeID),
			Attributes: slot.RadiusAttributes,
			Data:       &modparser.JewelFuncTag{},
		})
		if fn.Typ != "Self" && slot.RadiusNodes != nil {
			// Add nearby unallocated nodes to the extra node list
			for nodeId := range slot.RadiusNodes {
				if env.AllocNodes[nodeId] == nil {
					node := env.Build.Spec.RadiusNodeData[nodeId]
					if node == nil {
						panic(fmt.Sprintf("calc: in-radius node %d missing from radiusNodeData", nodeId))
					}
					env.ExtraRadiusNodeList[nodeId] = node
				}
			}
		}
	}
}
