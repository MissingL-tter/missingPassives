package build

import (
	"regexp"
	"sort"
	"strconv"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/skills"
	"github.com/MissingL-tter/missingPassives/tree"
)

// baseSlots is ItemsTab.lua's slot order; the slot table is built from it
// and the abyssal sockets each entry can host.
var baseSlots = []string{
	"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots",
	"Amulet", "Ring 1", "Ring 2", "Ring 3", "Belt", "Graft 1", "Graft 2",
	"Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5",
}

// abyssalHosts are the slots that carry abyssal socket sub-slots.
var abyssalHosts = map[string]bool{
	"Weapon 1": true, "Weapon 2": true, "Helmet": true,
	"Gloves": true, "Body Armour": true, "Boots": true, "Belt": true,
}

const abyssalSocketsPerSlot = 6

var (
	reTrailingNum = regexp.MustCompile(`[0-9]+$`)
	reAnyNum      = regexp.MustCompile(`[0-9]+`)
)

// slotNumOf is the control's `tonumber(name:match("%d+$") or
// name:match("%d+"))`: trailing digits win, otherwise the first run.
func slotNumOf(name string) *float64 {
	digits := reTrailingNum.FindString(name)
	if digits == "" {
		digits = reAnyNum.FindString(name)
	}
	if digits == "" {
		return nil
	}
	n, err := strconv.ParseFloat(digits, 64)
	if err != nil {
		return nil
	}
	return &n
}

// isWeaponSlot reports slotName:match("Weapon") - the base slots that get a
// weapon-swap twin.
func isWeaponSlot(name string) bool { return len(name) >= 6 && name[:6] == "Weapon" }

// copyMods deep-copies a mod list. The tree is loaded once and shared
// across every build assembled from it, while the calc stamps sources
// into the mods it is handed.
func copyMods(mods []*modparser.Mod) []*modparser.Mod {
	if mods == nil {
		return nil
	}
	out := make([]*modparser.Mod, len(mods))
	for i, m := range mods {
		if m != nil {
			out[i] = m.Clone()
		}
	}
	return out
}

func strPtr(s string) *string { return &s }
func numPtr(v float64) *float64 {
	return &v
}
func truePtr() *bool { t := true; return &t }

// SpecNodeInput projects an allocated (or in-radius) spec node into the
// shape the calc's node-list stage reads.
func SpecNodeInput(n *tree.SpecNode) *calc.NodeInput {
	in := &calc.NodeInput{
		ID:                   float64(n.ID()),
		Type:                 n.Type().String(),
		Name:                 n.EffectiveName(),
		DN:                   strPtr(n.Dn),
		DistanceToClassStart: n.DistanceToClassStart,
		ModList:              copyMods(n.Stats.ModList),
	}
	if n.KeystoneMod != nil {
		in.KeystoneMod = n.KeystoneMod.Clone()
	}
	if n.IsTattoo {
		in.IsTattoo = truePtr()
	}
	if n.OverrideType != "" {
		in.OverrideType = strPtr(string(n.OverrideType))
	}
	if n.ConqueredBy != nil {
		in.ConqueredBy = truePtr()
	}
	return in
}

// TreeNodeInput projects a bare tree node, for a name-resolved node the
// spec has no instance of (an ascendancy node of another class).
func TreeNodeInput(n *tree.Node) *calc.NodeInput {
	return &calc.NodeInput{
		ID:          float64(n.ID),
		Type:        n.Type.String(),
		Name:        n.NameStr,
		DN:          strPtr(n.Name),
		ModList:     copyMods(n.ModList),
		KeystoneMod: n.KeystoneMod,
	}
}

// SpecInput projects a loaded spec, plus the node data its socketed radius
// jewels reach, into the calc's spec input.
func SpecInput(spec *tree.Spec, items map[int]*item.Item) *calc.SpecInput {
	out := &calc.SpecInput{
		AllocNodes:                map[int]*calc.NodeInput{},
		KeystoneMap:               map[string][]*modparser.Mod{},
		RadiusNodeData:            map[int]*calc.NodeInput{},
		AllocatedNotableCount:     spec.AllocatedNotableCount,
		AllocatedKeystoneCount:    spec.AllocatedKeystoneCount,
		AllocatedMasteryCount:     spec.AllocatedMasteryCount,
		AllocatedMasteryTypeCount: spec.AllocatedMasteryTypeCount,
		AllocatedMasteryTypes:     map[string]float64{},
		AllocatedTattooTypes:      map[string]float64{},
		Passives:                  Passives{Spec: spec},
	}
	for id, node := range spec.AllocNodes {
		out.AllocNodes[int(id)] = SpecNodeInput(node)
	}
	for name, ksNode := range spec.Tree.KeystoneMap {
		out.KeystoneMap[name] = copyMods(ksNode.ModList)
	}
	for socketID, itemID := range spec.Jewels {
		it := items[itemID]
		if it == nil || it.JewelRadiusIndex == nil {
			continue
		}
		socket := spec.Nodes[socketID]
		if socket == nil || socket.T.NodesInRadius == nil {
			continue
		}
		for id := range socket.T.NodesInRadius[*it.JewelRadiusIndex-1] {
			if spec.AllocNodes[id] == nil && spec.Nodes[id] != nil {
				out.RadiusNodeData[int(id)] = SpecNodeInput(spec.Nodes[id])
			}
		}
	}
	for k, v := range spec.AllocatedMasteryTypes {
		out.AllocatedMasteryTypes[k] = v
	}
	for k, v := range spec.AllocatedTattooTypes {
		out.AllocatedTattooTypes[string(k)] = v
	}
	return out
}

// SkillsTabInput projects a loaded skills tab.
func SkillsTabInput(tab *skills.Tab) *calc.SkillsTabInput {
	out := &calc.SkillsTabInput{ImbuedSupportBySlot: map[string]string{}}
	for _, group := range tab.SocketGroupList {
		g := &calc.SocketGroupInput{SocketGroup: group}
		for _, gem := range group.GemList {
			g.GemList = append(g.GemList, &calc.SocketGemInput{Gem: gem})
		}
		out.SocketGroups = append(out.SocketGroups, g)
	}
	for slot, ge := range tab.ImbuedSupportBySlot {
		out.ImbuedSupportBySlot[slot] = ge.Id
	}
	return out
}

// itemSet is one decoded <ItemSet>: per-slot selection and activation.
type itemSet struct {
	id                 int
	useSecondWeaponSet bool
	selItemID          map[string]int
	active             map[string]bool
}

// slotTable builds the ordered slot list ItemsTab constructs: the base
// slots, each weapon's swap twin, six abyssal sockets under every slot
// that hosts them, and one socket slot per jewel node in the tree.
func slotTable(spec *tree.Spec) []*calc.SlotInput {
	var slots []*calc.SlotInput
	add := func(name, label string, weaponSet *float64, parent *string) *calc.SlotInput {
		s := &calc.SlotInput{
			SlotName:       name,
			Label:          label,
			SlotNum:        slotNumOf(name),
			WeaponSet:      weaponSet,
			ParentSlotName: parent,
		}
		slots = append(slots, s)
		return s
	}
	for _, slotName := range baseSlots {
		main := add(slotName, slotName, nil, nil)
		if isWeaponSlot(slotName) {
			main.WeaponSet = numPtr(1)
			swapName := slotName + " Swap"
			add(swapName, slotName, numPtr(2), nil)
			for i := 1; i <= abyssalSocketsPerSlot; i++ {
				add(swapName+" Abyssal Socket "+strconv.Itoa(i),
					"Abyssal #"+strconv.Itoa(i), numPtr(2), strPtr(swapName))
			}
		}
		if abyssalHosts[slotName] {
			for i := 1; i <= abyssalSocketsPerSlot; i++ {
				var ws *float64
				if isWeaponSlot(slotName) {
					ws = numPtr(1)
				}
				add(slotName+" Abyssal Socket "+strconv.Itoa(i),
					"Abyssal #"+strconv.Itoa(i), ws, strPtr(slotName))
			}
		}
	}
	var socketIDs []int64
	for id, node := range spec.Tree.Nodes {
		if node.Type == tree.NodeSocket {
			socketIDs = append(socketIDs, id)
		}
	}
	sort.Slice(socketIDs, func(i, j int) bool { return socketIDs[i] < socketIDs[j] })
	// Every socket slot is labelled "Socket": the numbering the tab
	// shows is applied by its own layout pass, not on load.
	for _, id := range socketIDs {
		s := add("Jewel "+strconv.FormatInt(id, 10), "Socket", nil, nil)
		s.NodeID = numPtr(float64(id))
	}
	return slots
}

// fillSlots stamps the equipped item, activation and radius data onto the
// slot table: non-socket slots take the active item set's selection, jewel
// sockets take the spec's.
func fillSlots(slots []*calc.SlotInput, spec *tree.Spec, items map[int]*item.Item, active *itemSet) {
	for _, s := range slots {
		if s.NodeID != nil {
			if id := spec.Jewels[int64(*s.NodeID)]; id != 0 {
				s.ItemID = numPtr(float64(id))
			}
		} else if active != nil {
			if id := active.selItemID[s.SlotName]; id != 0 {
				s.ItemID = numPtr(float64(id))
			}
			if active.active[s.SlotName] {
				s.Active = truePtr()
			}
		}
		if s.NodeID == nil || s.ItemID == nil {
			continue
		}
		it := items[int(*s.ItemID)]
		if it == nil || it.JewelRadiusIndex == nil {
			continue
		}
		socket := spec.Nodes[int64(*s.NodeID)]
		if socket == nil {
			continue
		}
		idx := *it.JewelRadiusIndex - 1
		s.RadiusNodes = map[int]string{}
		if socket.T.NodesInRadius != nil {
			for id, radNode := range socket.T.NodesInRadius[idx] {
				s.RadiusNodes[int(id)] = radNode.Type.String()
			}
		}
		if socket.T.AttributesInRadius != nil {
			s.RadiusAttributes = map[string]float64{}
			for k, v := range socket.T.AttributesInRadius[idx] {
				s.RadiusAttributes[k] = v
			}
		}
	}
}
