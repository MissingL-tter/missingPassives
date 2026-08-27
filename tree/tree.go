// Port of Classes/PassiveTree.lua's logic half: load the tree data, type
// and link the nodes, parse their stats into mod lists, and precompute the
// geometry (positions, per-socket radius sets). The view half — sprites,
// assets, connectors, overlays — stays behind.
package tree

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// Stats is the parsed-modifier state ProcessStats maintains on anything
// carrying stat lines (tree nodes, mastery effects, legion nodes).
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
// Legion (timeless jewel) passives are also Nodes; they carry a string id
// (IDStr) and no geometry.
type Node struct {
	ID    int64
	IDStr string // mod source id: decimal of ID, or the legion id
	Name  string // dn (= name in the 3_10+ format)
	// NameStr is node.name as distinct from dn: tree nodes carry both (the
	// same string); legion passives have only dn.
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
	// Legion pool metadata (nil on real tree nodes): the roll ranges the
	// timeless substitution reads.
	SortedStats []string
	StatDescs   []*LegionStatDesc
	// Tattoo override pool fields (false/empty on real tree nodes).
	IsTattooFlag           bool
	OverrideTypeStr        string
	IsProxy                bool
	IsBlighted             bool
	IsMultipleChoiceOption bool

	GrantedStrength     float64
	GrantedDexterity    float64
	GrantedIntelligence float64

	// Raw keeps the node object as decoded, for fields later stages read
	// (expansionJewel, recipe, ...).
	Raw map[string]any

	// Processed state.
	Type        string
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

	Legion *Legion
	Tattoo *Tattoo

	MinX, MinY, MaxX, MaxY float64
	Size                   float64
}

// legacyAlternateAscendancies are hidden as no longer obtainable.
var legacyAlternateAscendancies = map[string]bool{
	"Warden": true, "Warlock": true, "Primalist": true,
}

var classArt = map[int64]string{
	0: "centerscion", 1: "centermarauder", 2: "centerranger",
	3: "centerwitch", 4: "centerduelist", 5: "centertemplar", 6: "centershadow",
}

// Load builds the tree for one version from the raw document
// (data/raw/tree_<version>.json). The parser must already be loaded
// (data.Load), since node stats parse through it.
func Load(version string) *Tree {
	var raw map[string]any
	data.RawDoc("tree_"+version, &raw)
	t := &Tree{Version: version}

	t.MinX, t.MinY = num(raw["min_x"]), num(raw["min_y"])
	t.MaxX, t.MaxY = num(raw["max_x"]), num(raw["max_y"])
	t.Size = math.Min(t.MaxX-t.MinX, t.MaxY-t.MinY) * 1.1

	// Classes arrive 1-based; migrate to the old 0-based form.
	t.Classes = map[int64]*Class{}
	for i, cv := range canonArray(raw["classes"]) {
		cm := cv.(map[string]any)
		class := &Class{
			Name:    str(cm["name"]),
			BaseStr: num(cm["base_str"]),
			BaseDex: num(cm["base_dex"]),
			BaseInt: num(cm["base_int"]),
			Classes: map[int64]*AscendClass{0: {Name: "None"}},
		}
		for j, av := range canonArray(cm["ascendancies"]) {
			am := av.(map[string]any)
			class.Classes[int64(j+1)] = &AscendClass{
				ID:         str(am["id"]),
				InternalID: str(am["internalId"]),
				Name:       str(am["name"]),
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
	for i, av := range canonArray(raw["alternate_ascendancies"]) {
		am := av.(map[string]any)
		asc := &AscendClass{ID: str(am["id"]), Name: str(am["name"])}
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

	constants := raw["constants"].(map[string]any)
	for _, v := range canonArray(constants["skillsPerOrbit"]) {
		t.SkillsPerOrbit = append(t.SkillsPerOrbit, int64(num(v)))
	}
	for _, v := range canonArray(constants["orbitRadii"]) {
		t.OrbitRadii = append(t.OrbitRadii, num(v))
	}
	t.OrbitAnglesByOrbit = make([][]float64, len(t.SkillsPerOrbit))
	for orbit, skillsInOrbit := range t.SkillsPerOrbit {
		t.OrbitAnglesByOrbit[orbit] = calcOrbitAngles(skillsInOrbit)
	}

	// Groups (migrated: n = nodes, oo = orbit set).
	t.Groups = map[int64]*Group{}
	for gid, gv := range raw["groups"].(map[string]any) {
		gm := gv.(map[string]any)
		id, err := strconv.ParseInt(gid, 10, 64)
		if err != nil {
			panic("tree: non-numeric group id " + gid)
		}
		group := &Group{
			ID:      id,
			X:       num(gm["x"]),
			Y:       num(gm["y"]),
			IsProxy: boolean(gm["isProxy"]),
			Orbits:  map[int64]bool{},
		}
		group.NodeIDs = idList(gm["nodes"])
		for _, ov := range canonArray(gm["orbits"]) {
			group.Orbits[int64(num(ov))] = true
		}
		t.Groups[id] = group
	}

	// Nodes: decode, drop root, drop legacy alternate-ascendancy nodes
	// (and their groups).
	t.Nodes = map[int64]*Node{}
	for key, nv := range raw["nodes"].(map[string]any) {
		if key == "root" {
			continue
		}
		nm := nv.(map[string]any)
		node := decodeNode(nm)
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
	nodeIDs := make([]int64, 0, len(t.Nodes))
	for id := range t.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })
	for _, id := range nodeIDs {
		node := t.Nodes[id]
		t.typeNode(node)
		group := t.Groups[node.GroupID]
		if group != nil {
			node.Group = group
			group.AscendancyName = node.AscendancyName
			if node.AscendancyStart {
				group.AscendancyStart = true
			}
		} else if node.Type == "Notable" || node.Type == "Keystone" {
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
			if node != nil && node.Type == "Normal" {
				node.ModList = append(node.ModList, modparser.NewMod(
					"Condition:ConnectedTo"+class.Name+"Start", "FLAG", true,
					"Tree:"+strconv.FormatInt(nodeID, 10)))
			}
		}
	}

	t.loadLegion()
	t.loadTattoo()
	return t
}

func decodeNode(nm map[string]any) *Node {
	node := &Node{
		ID:                     int64(num(nm["skill"])),
		Name:                   str(nm["name"]),
		Icon:                   str(nm["icon"]),
		GroupID:                int64(num(nm["group"])),
		Orbit:                  int64(num(nm["orbit"])),
		OrbitIndex:             int64(num(nm["orbitIndex"])),
		Out:                    idList(nm["out"]),
		In:                     idList(nm["in"]),
		Keystone:               boolean(nm["isKeystone"]),
		Notable:                boolean(nm["isNotable"]),
		Mastery:                boolean(nm["isMastery"]),
		JewelSocket:            boolean(nm["isJewelSocket"]),
		AscendancyStart:        boolean(nm["isAscendancyStart"]),
		AscendancyName:         str(nm["ascendancyName"]),
		PassivePointsGranted:   num(nm["grantedPassivePoints"]),
		IsProxy:                boolean(nm["isProxy"]),
		IsBlighted:             boolean(nm["isBlighted"]),
		IsMultipleChoiceOption: boolean(nm["isMultipleChoiceOption"]),
		GrantedStrength:        num(nm["grantedStrength"]),
		GrantedDexterity:       num(nm["grantedDexterity"]),
		GrantedIntelligence:    num(nm["grantedIntelligence"]),
		Raw:                    nm,
	}
	node.IDStr = strconv.FormatInt(node.ID, 10)
	node.NameStr = &node.Name
	node.Sd = strList(nm["stats"])
	if csi, ok := nm["classStartIndex"].(float64); ok {
		v := int64(csi)
		node.ClassStartIndex = &v
	}
	for _, ev := range canonArray(nm["masteryEffects"]) {
		em := ev.(map[string]any)
		node.MasteryEffects = append(node.MasteryEffects, &MasteryEffectRef{
			Effect:       int64(num(em["effect"])),
			Stats:        strList(em["stats"]),
			ReminderText: strList(em["reminderText"]),
		})
	}
	return node
}

// typeNode ports the constructor's node-type chain.
func (t *Tree) typeNode(node *Node) {
	switch {
	case node.ClassStartIndex != nil:
		node.Type = "ClassStart"
		class := t.Classes[*node.ClassStartIndex]
		class.StartNodeID = node.ID
		node.StartArt = classArt[*node.ClassStartIndex]
	case node.AscendancyStart:
		node.Type = "AscendClassStart"
		t.AscendNameMap[node.AscendancyName].AscendClass.StartNodeID = node.ID
	case node.Mastery:
		node.Type = "Mastery"
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
		node.Type = "Socket"
		t.Sockets[node.ID] = node
	case node.Keystone:
		node.Type = "Keystone"
		t.KeystoneMap[node.Name] = node
		t.KeystoneMap[luaLower(node.Name)] = node
	case node.Notable:
		node.Type = "Notable"
		if node.AscendancyName == "" {
			// Duplicate-named off-tree nodes lose to on-tree (grouped)
			// ones; cluster notables have no group and still register.
			if t.NotableMap[luaLower(node.Name)] == nil || node.GroupID != 0 {
				t.NotableMap[luaLower(node.Name)] = node
			}
		} else {
			t.AscendancyMap[luaLower(node.Name)] = node
			className := t.AscendNameMap[node.AscendancyName].Class.Name
			if className != "Scion" {
				t.ClassNotables[className] = append(t.ClassNotables[className], node.Name)
			} else if t.ClassNotables[className] == nil {
				t.ClassNotables[className] = []string{}
			}
		}
	default:
		node.Type = "Normal"
		isAscendantSpecial := node.AscendancyName == "Ascendant" && !node.IsMultipleChoiceOption &&
			!strings.Contains(node.Name, "Dexterity") && !strings.Contains(node.Name, "Intelligence") &&
			!strings.Contains(node.Name, "Strength") && !strings.Contains(node.Name, "Passive")
		if (isAscendantSpecial || (node.IsMultipleChoiceOption && node.AscendancyName != "")) &&
			node.AscendancyName != "Reliquarian" && node.AscendancyName != "Luminary" {
			className := t.AscendNameMap[node.AscendancyName].Class.Name
			t.AscendancyMap[luaLower(node.Name)] = node
			t.ClassNotables[className] = append(t.ClassNotables[className], node.Name)
		}
	}
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

func luaLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
