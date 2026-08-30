// Port of Classes/PassiveTree.lua's logic half: load the tree data, type
// and link the nodes, parse their stats into mod lists, and precompute the
// geometry (positions, per-socket radius sets). The view half — sprites,
// assets, connectors, overlays — stays behind.
package tree

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// Stats is the parsed-modifier state ProcessStats maintains on anything
// carrying stat lines (tree nodes, mastery effects, alternate passives).
type Stats struct {
	Sd      []string // stat lines (multiline entries split in place)
	Mods    []*NodeMod
	ModKey  string
	ModList []*modparser.Mod
	Unknown bool // some line the parser had no idea about
	Extra   bool // some line only partially understood
}

// NodeMod is one node.mods entry: the parse result for one stat line.
type NodeMod struct {
	List  []*modparser.Mod
	Nil   bool // Lua list == nil (parser did not recognise the line)
	Extra string
}

// MasteryEffectRef is one entry of a mastery node's masteryEffects array.
type MasteryEffectRef struct {
	Effect       int64
	Stats        []string
	ReminderText []string
}

// MasteryEffect is one tree-wide mastery effect (tree.masteryEffects).
type MasteryEffect struct {
	ID int64
	Stats
}

// Node is one passive tree node, raw fields plus processed state.
// Alternate (conquered-replacement) passives are also Nodes; they carry a string id
// (IDStr) and no geometry.
type Node struct {
	ID    int64
	IDStr string // mod source id: decimal of ID, or the alt passive's string id
	Name  string // dn (= name in the 3_10+ format)
	// NameStr is node.name as distinct from dn: tree nodes carry both (the
	// same string); alternate passives have only dn.
	NameStr    *string
	Icon       string
	GroupID    int64
	Orbit      int64
	OrbitIndex int64
	Out        []int64
	In         []int64

	Keystone        bool // ks
	Notable         bool // not
	Mastery         bool // m
	JewelSocket     bool
	AscendancyStart bool
	AscendancyName  string
	ClassStartIndex *int64

	MasteryEffects       []*MasteryEffectRef
	PassivePointsGranted float64
	// Alternate-pool metadata (nil on real tree nodes): the roll ranges the
	// conquering substitution reads.
	SortedStats []string
	StatDescs   []*ConqueredStatDesc
	// Tattoo override pool fields (false/empty on real tree nodes).
	IsTattooFlag      bool
	OverrideTypeOf    OverrideKind
	ActiveEffectImage string // the saved-Overrides match key

	IsProxy                bool
	IsBlighted             bool
	IsJustIcon             bool
	IsMultipleChoice       bool
	IsMultipleChoiceOption bool
	ExpansionJewel         *ExpansionJewel // set on cluster jewel sockets

	GrantedStrength     float64
	GrantedDexterity    float64
	GrantedIntelligence float64

	// Processed state.
	Type        NodeKind
	LinkedIDs   []int64
	Group       *Group
	Angle       float64
	X, Y        float64
	HasPosition bool
	StartArt    string
	KeystoneMod *modparser.Mod
	Stats

	// Per-socket precomputation (sockets and keystones only), indexed by
	// jewel radius (1-based, as the reference indexes data.jewelRadius).
	NodesInRadius      []map[int64]*Node
	AttributesInRadius []map[string]float64
	CharmSocket        bool
}

// Group is one node group (position cluster) on the tree.
type Group struct {
	ID              int64
	X, Y            float64
	NodeIDs         []int64
	Orbits          map[int64]bool // oo
	IsProxy         bool
	AscendancyName  string
	AscendancyStart bool
}

// AscendClass is one ascendancy entry of a class.
type AscendClass struct {
	ID          string
	InternalID  string
	Name        string
	StartNodeID int64
}

// Class is one character class.
type Class struct {
	Name        string
	BaseStr     float64
	BaseDex     float64
	BaseInt     float64
	Classes     map[int64]*AscendClass // ascendancies, [0] = None
	StartNodeID int64
}

// AscendNameEntry is one ascendNameMap value.
type AscendNameEntry struct {
	ClassID       int64
	IsAlternate   bool
	AscendClassID int64
	AscendClass   *AscendClass
	Class         *Class
}

// Tree is the processed passive tree for one version.
type Tree struct {
	Version string

	Classes                map[int64]*Class
	ClassNameMap           map[string]int64
	AscendNameMap          map[string]*AscendNameEntry
	InternalAscendNameMap  map[string]*AscendNameEntry
	SecondaryAscendNameMap map[string]*AscendNameEntry
	ClassNotables          map[string][]string

	Nodes  map[int64]*Node
	Groups map[int64]*Group

	KeystoneMap    map[string]*Node
	NotableMap     map[string]*Node
	AscendancyMap  map[string]*Node
	ClusterNodeMap map[string]*Node
	Sockets        map[int64]*Node
	MasteryEffects map[int64]*MasteryEffect

	SkillsPerOrbit     []int64
	OrbitRadii         []float64
	OrbitAnglesByOrbit [][]float64

	ConqueredPassives *ConqueredPassives
	Tattoo            *Tattoo

	MinX, MinY, MaxX, MaxY float64
	Size                   float64

	// The decoded document, kept for the abyss generator's world (it needs
	// the legacy alternate-ascendancy nodes Load drops).
	doc       *schema.PassiveTree
	abyssOnce sync.Once
	abyss     *abyssWorld
}

// legacyAlternateAscendancies are hidden as no longer obtainable.
var legacyAlternateAscendancies = map[string]bool{
	"Warden": true, "Warlock": true, "Primalist": true,
}

var classArt = map[int64]string{
	0: "centerscion", 1: "centermarauder", 2: "centerranger",
	3: "centerwitch", 4: "centerduelist", 5: "centertemplar", 6: "centershadow",
}

// loadDoc decodes data/raw/tree_<version>.json.
func loadDoc(version string) *schema.PassiveTree {
	var doc schema.PassiveTree
	data.RawDoc("tree_"+version, &doc)
	return &doc
}

// Load builds the tree for one version from the raw document
// (data/raw/tree_<version>.json). The parser must already be loaded
// (data.Load), since node stats parse through it.
func Load(version string) (*Tree, error) {
	doc := loadDoc(version)
	t := &Tree{Version: version, doc: doc}

	t.MinX, t.MinY = doc.MinX, doc.MinY
	t.MaxX, t.MaxY = doc.MaxX, doc.MaxY
	t.Size = math.Min(t.MaxX-t.MinX, t.MaxY-t.MinY) * 1.1

	// Classes arrive 1-based; migrate to the old 0-based form.
	t.Classes = map[int64]*Class{}
	for i, cm := range doc.Classes {
		class := &Class{
			Name:    cm.Name,
			BaseStr: cm.BaseStr,
			BaseDex: cm.BaseDex,
			BaseInt: cm.BaseInt,
			Classes: map[int64]*AscendClass{0: {Name: "None"}},
		}
		for j, am := range cm.Ascendancies {
			class.Classes[int64(j+1)] = &AscendClass{
				ID:         am.Id,
				InternalID: am.InternalId,
				Name:       am.Name,
			}
		}
		t.Classes[int64(i)] = class
	}

	t.ClassNameMap = map[string]int64{}
	t.AscendNameMap = map[string]*AscendNameEntry{}
	t.InternalAscendNameMap = map[string]*AscendNameEntry{}
	t.ClassNotables = map[string][]string{}
	for classID, class := range t.Classes {
		t.ClassNameMap[class.Name] = classID
		for ascendClassID, ascendClass := range class.Classes {
			entry := &AscendNameEntry{
				ClassID:       classID,
				Class:         class,
				AscendClassID: ascendClassID,
				AscendClass:   ascendClass,
			}
			if ascendClass.ID != "" {
				t.AscendNameMap[ascendClass.ID] = entry
			}
			t.AscendNameMap[ascendClass.Name] = entry
			if ascendClass.InternalID != "" {
				t.InternalAscendNameMap[ascendClass.InternalID] = entry
			}
		}
	}

	// Alternate ascendancies: filter the legacy ones, index the rest.
	altAsc := map[int64]*AscendClass{}
	altNodeNames := map[string]bool{}
	for i, am := range doc.AlternateAscendancies {
		asc := &AscendClass{ID: am.Id, Name: am.Name}
		if legacyAlternateAscendancies[asc.ID] {
			altNodeNames[asc.ID] = true
			continue
		}
		altAsc[int64(i+1)] = asc
	}
	if len(altAsc) > 0 {
		t.SecondaryAscendNameMap = map[string]*AscendNameEntry{}
		altClass := &Class{Name: "alternate_ascendancies", Classes: altAsc}
		for ascendClassID, asc := range altAsc {
			entry := &AscendNameEntry{
				IsAlternate:   true,
				Class:         altClass,
				AscendClassID: ascendClassID,
				AscendClass:   asc,
			}
			t.AscendNameMap[asc.ID] = entry
			t.SecondaryAscendNameMap[asc.ID] = entry
		}
	}

	t.SkillsPerOrbit = doc.Constants.SkillsPerOrbit
	t.OrbitRadii = doc.Constants.OrbitRadii
	t.OrbitAnglesByOrbit = make([][]float64, len(t.SkillsPerOrbit))
	for orbit, skillsInOrbit := range t.SkillsPerOrbit {
		t.OrbitAnglesByOrbit[orbit] = calcOrbitAngles(skillsInOrbit)
	}

	// Groups (migrated: n = nodes, oo = orbit set).
	t.Groups = map[int64]*Group{}
	for gid, gm := range doc.Groups {
		id, err := strconv.ParseInt(gid, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("tree %s: non-numeric group id %q", version, gid)
		}
		group := &Group{
			ID:      id,
			X:       gm.X,
			Y:       gm.Y,
			IsProxy: gm.IsProxy,
			Orbits:  map[int64]bool{},
			NodeIDs: idsOf(gm.Nodes),
		}
		for _, orbit := range gm.Orbits {
			group.Orbits[orbit] = true
		}
		t.Groups[id] = group
	}

	// Nodes: decode, drop root, drop legacy alternate-ascendancy nodes
	// (and their groups).
	t.Nodes = map[int64]*Node{}
	for key, nd := range doc.Nodes {
		if key == "root" {
			continue
		}
		node := decodeNode(&nd)
		if node.AscendancyName != "" && altNodeNames[node.AscendancyName] {
			delete(t.Groups, node.GroupID)
			continue
		}
		t.Nodes[node.ID] = node
	}

	// Type the nodes and build the lookup maps, then process each. The
	// reference iterates pairs(self.nodes); every effect here is
	// order-independent except the notableMap group-preference rule, which
	// is replicated below, so sorted order is safe.
	t.KeystoneMap = map[string]*Node{}
	t.NotableMap = map[string]*Node{}
	t.AscendancyMap = map[string]*Node{}
	t.ClusterNodeMap = map[string]*Node{}
	t.Sockets = map[int64]*Node{}
	t.MasteryEffects = map[int64]*MasteryEffect{}
	nodeIDs := sortedNodeIDs(t.Nodes)
	for _, id := range nodeIDs {
		node := t.Nodes[id]
		if err := t.typeNode(node); err != nil {
			return nil, err
		}
		group := t.Groups[node.GroupID]
		if group != nil {
			node.Group = group
			group.AscendancyName = node.AscendancyName
			if node.AscendancyStart {
				group.AscendancyStart = true
			}
		} else if node.Type == NodeNotable || node.Type == NodeKeystone {
			t.ClusterNodeMap[node.Name] = node
		}
		t.ProcessNode(node)
	}

	// linkedId from out + in.
	for _, id := range nodeIDs {
		node := t.Nodes[id]
		node.LinkedIDs = append(node.LinkedIDs, node.Out...)
		node.LinkedIDs = append(node.LinkedIDs, node.In...)
	}

	t.buildRadiusSets()

	// Nodes adjacent to a class start get a connection condition flag.
	for _, class := range t.Classes {
		startNode := t.Nodes[class.StartNodeID]
		if startNode == nil {
			continue
		}
		for _, nodeID := range startNode.LinkedIDs {
			node := t.Nodes[nodeID]
			if node != nil && node.Type == NodeNormal {
				node.ModList = append(node.ModList, modparser.NewModFull(
					"Condition:ConnectedTo"+class.Name+"Start", modparser.Flag, modparser.Bool(true),
					"Tree:"+strconv.FormatInt(nodeID, 10), true, 0, 0))
			}
		}
	}

	t.loadConqueredPassives()
	t.loadTattoo()

	// Late load the Generated data so we can take advantage of a tree
	// existing (the reference's buildTreeDependentUniques call).
	var nativeKeystones []string
	seen := map[*Node]bool{}
	for _, name := range sortedStringKeys(t.KeystoneMap) {
		node := t.KeystoneMap[name]
		if node.Keystone && !node.IsBlighted && node.HasPosition && !seen[node] {
			seen[node] = true
			nativeKeystones = append(nativeKeystones, node.Name)
		}
	}
	data.BuildTreeDependentUniques(t.ClassNotables, nativeKeystones)
	return t, nil
}

// cloneStrings copies a document string list; absent lists become empty
// ones (every stat list is a table in the reference).
func cloneStrings(v []string) []string {
	out := make([]string, len(v))
	copy(out, v)
	return out
}

// idsOf converts a document id list.
func idsOf(ids []schema.NodeID) []int64 {
	if ids == nil {
		return nil
	}
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = int64(id)
	}
	return out
}

func decodeNode(nd *schema.TreeNode) *Node {
	node := &Node{
		ID:                     nd.Skill,
		Name:                   nd.Name,
		Icon:                   nd.Icon,
		GroupID:                nd.Group,
		OrbitIndex:             nd.OrbitIndex,
		Out:                    idsOf(nd.Out),
		In:                     idsOf(nd.In),
		Keystone:               nd.IsKeystone,
		Notable:                nd.IsNotable,
		Mastery:                nd.IsMastery,
		JewelSocket:            nd.IsJewelSocket,
		AscendancyStart:        nd.IsAscendancyStart,
		AscendancyName:         nd.AscendancyName,
		ClassStartIndex:        nd.ClassStartIndex,
		PassivePointsGranted:   nd.GrantedPassivePoints,
		IsProxy:                nd.IsProxy,
		IsBlighted:             nd.IsBlighted,
		IsJustIcon:             nd.IsJustIcon,
		IsMultipleChoice:       nd.IsMultipleChoice,
		IsMultipleChoiceOption: nd.IsMultipleChoiceOption,
		ActiveEffectImage:      nd.ActiveEffectImage,
		GrantedStrength:        nd.GrantedStrength,
		GrantedDexterity:       nd.GrantedDexterity,
		GrantedIntelligence:    nd.GrantedIntelligence,
	}
	if nd.Orbit != nil {
		node.Orbit = *nd.Orbit
	}
	if ej := nd.ExpansionJewel; ej != nil {
		node.ExpansionJewel = &ExpansionJewel{Size: ej.Size, Index: ej.Index, Proxy: int64(ej.Proxy)}
		if ej.Parent != nil {
			parent := int64(*ej.Parent)
			node.ExpansionJewel.Parent = &parent
		}
	}
	node.IDStr = strconv.FormatInt(node.ID, 10)
	node.NameStr = &node.Name
	node.Sd = cloneStrings(nd.Stats)
	for _, em := range nd.MasteryEffects {
		node.MasteryEffects = append(node.MasteryEffects, &MasteryEffectRef{
			Effect:       em.Effect,
			Stats:        cloneStrings(em.Stats),
			ReminderText: cloneStrings(em.ReminderText),
		})
	}
	return node
}

// typeNode ports the constructor's node-type chain.
func (t *Tree) typeNode(node *Node) error {
	switch {
	case node.ClassStartIndex != nil:
		node.Type = NodeClassStart
		class := t.Classes[*node.ClassStartIndex]
		if class == nil {
			return fmt.Errorf("tree: node %d starts unknown class %d", node.ID, *node.ClassStartIndex)
		}
		class.StartNodeID = node.ID
		node.StartArt = classArt[*node.ClassStartIndex]
	case node.AscendancyStart:
		node.Type = NodeAscendClassStart
		entry := t.AscendNameMap[node.AscendancyName]
		if entry == nil {
			return fmt.Errorf("tree: node %d starts unknown ascendancy %q", node.ID, node.AscendancyName)
		}
		entry.AscendClass.StartNodeID = node.ID
	case node.Mastery:
		node.Type = NodeMastery
		for _, effect := range node.MasteryEffects {
			if t.MasteryEffects[effect.Effect] == nil {
				me := &MasteryEffect{ID: effect.Effect}
				me.Sd = effect.Stats
				t.MasteryEffects[effect.Effect] = me
				processStats(&me.Stats, strconv.FormatInt(me.ID, 10), 0)
			} else {
				// Share the multiline-split stats from the first pass.
				effect.Stats = t.MasteryEffects[effect.Effect].Sd
			}
		}
	case node.JewelSocket:
		node.Type = NodeSocket
		t.Sockets[node.ID] = node
	case node.Keystone:
		node.Type = NodeKeystone
		t.KeystoneMap[node.Name] = node
		t.KeystoneMap[strings.ToLower(node.Name)] = node
	case node.Notable:
		node.Type = NodeNotable
		if node.AscendancyName == "" {
			// Duplicate-named off-tree nodes lose to on-tree (grouped)
			// ones; cluster notables have no group and still register.
			if t.NotableMap[strings.ToLower(node.Name)] == nil || node.GroupID != 0 {
				t.NotableMap[strings.ToLower(node.Name)] = node
			}
		} else {
			t.AscendancyMap[strings.ToLower(node.Name)] = node
			className, err := t.ascendancyClassName(node)
			if err != nil {
				return err
			}
			if className != "Scion" {
				t.ClassNotables[className] = append(t.ClassNotables[className], node.Name)
			} else if t.ClassNotables[className] == nil {
				t.ClassNotables[className] = []string{}
			}
		}
	default:
		node.Type = NodeNormal
		isAscendantSpecial := node.AscendancyName == "Ascendant" && !node.IsMultipleChoiceOption &&
			!strings.Contains(node.Name, "Dexterity") && !strings.Contains(node.Name, "Intelligence") &&
			!strings.Contains(node.Name, "Strength") && !strings.Contains(node.Name, "Passive")
		if (isAscendantSpecial || (node.IsMultipleChoiceOption && node.AscendancyName != "")) &&
			node.AscendancyName != "Reliquarian" && node.AscendancyName != "Luminary" {
			className, err := t.ascendancyClassName(node)
			if err != nil {
				return err
			}
			t.AscendancyMap[strings.ToLower(node.Name)] = node
			t.ClassNotables[className] = append(t.ClassNotables[className], node.Name)
		}
	}
	return nil
}

// ascendancyClassName is the class owning the node's ascendancy.
func (t *Tree) ascendancyClassName(node *Node) (string, error) {
	entry := t.AscendNameMap[node.AscendancyName]
	if entry == nil {
		return "", fmt.Errorf("tree: node %d in unknown ascendancy %q", node.ID, node.AscendancyName)
	}
	return entry.Class.Name, nil
}

// ProcessNode ports PassiveTreeClass:ProcessNode's logic half: position
// from group and orbit, then stat parsing. Shared with subgraph nodes.
func (t *Tree) ProcessNode(node *Node) {
	if node.Group != nil {
		node.Angle = t.OrbitAnglesByOrbit[node.Orbit][node.OrbitIndex]
		orbitRadius := t.OrbitRadii[node.Orbit]
		node.X = node.Group.X + math.Sin(node.Angle)*orbitRadius
		node.Y = node.Group.Y - math.Cos(node.Angle)*orbitRadius
		node.HasPosition = true
	}
	t.processNodeStats(node)
}

// buildRadiusSets precomputes nodesInRadius/attributesInRadius for sockets
// and nodesInRadius for keystones.
func (t *Tree) buildRadiusSets() {
	radii := data.JewelRadius
	inRadius := func(center *Node, excludeSockets bool) []map[int64]*Node {
		sets := make([]map[int64]*Node, len(radii))
		for i := range sets {
			sets[i] = map[int64]*Node{}
		}
		minX, maxX := center.X-data.MaxJewelRadius, center.X+data.MaxJewelRadius
		minY, maxY := center.Y-data.MaxJewelRadius, center.Y+data.MaxJewelRadius
		for _, node := range t.Nodes {
			if !node.HasPosition || node.X < minX || node.X > maxX || node.Y < minY || node.Y > maxY {
				continue
			}
			if node == center || node.IsBlighted || node.Group == nil || node.IsProxy ||
				node.Group.IsProxy || node.Mastery {
				continue
			}
			if excludeSockets && node.JewelSocket {
				continue
			}
			vX, vY := node.X-center.X, node.Y-center.Y
			distSquared := vX*vX + vY*vY
			for i, radiusInfo := range radii {
				if distSquared <= *radiusInfo.OuterSquared && *radiusInfo.InnerSquared <= distSquared {
					sets[i][node.ID] = node
				}
			}
		}
		return sets
	}
	for _, socket := range t.Sockets {
		if socket.Name == "Charm Socket" {
			socket.CharmSocket = true
			continue
		}
		socket.NodesInRadius = inRadius(socket, false)
		socket.AttributesInRadius = make([]map[string]float64, len(radii))
		for i := range socket.AttributesInRadius {
			socket.AttributesInRadius[i] = map[string]float64{}
		}
	}
	for _, keystone := range t.KeystoneMap {
		if keystone.NodesInRadius != nil || !keystone.HasPosition {
			if keystone.NodesInRadius == nil {
				keystone.NodesInRadius = make([]map[int64]*Node, len(radii))
				for i := range keystone.NodesInRadius {
					keystone.NodesInRadius[i] = map[int64]*Node{}
				}
			}
			continue
		}
		// The reference's keystone pass excludes node.isSocket — a field
		// nothing sets on tree nodes — so it filters nothing there either.
		keystone.NodesInRadius = inRadius(keystone, false)
	}
}

func calcOrbitAngles(nodesInOrbit int64) []float64 {
	var orbitAngles []float64
	switch nodesInOrbit {
	case 16:
		orbitAngles = []float64{0, 30, 45, 60, 90, 120, 135, 150, 180, 210, 225, 240, 270, 300, 315, 330}
	case 40:
		orbitAngles = []float64{0, 10, 20, 30, 40, 45, 50, 60, 70, 80, 90, 100, 110, 120, 130, 135, 140, 150, 160, 170, 180, 190, 200, 210, 220, 225, 230, 240, 250, 260, 270, 280, 290, 300, 310, 315, 320, 330, 340, 350}
	default:
		orbitAngles = make([]float64, nodesInOrbit+1)
		for i := int64(0); i <= nodesInOrbit; i++ {
			orbitAngles[i] = 360 * float64(i) / float64(nodesInOrbit)
		}
	}
	for i, degrees := range orbitAngles {
		orbitAngles[i] = degrees * math.Pi / 180
	}
	return orbitAngles
}
