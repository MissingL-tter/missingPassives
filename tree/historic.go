// Historic jewel node generation — what a conquered passive becomes, for
// both families, computed in place of PoB's shipped LUTs (proven
// cell-for-cell against them; nothing from the bins ships):
//
//   - Timeless jewels (types 1-6): TimelessPassive reproduces
//     Modules/DataLegionLookUpTableHelper.lua's readLUT contract (global
//     ids + rolls) via the reverse-engineered generation algorithm.
//     Differential: test/timeless_test.go, every cell of all 6 bins.
//   - Abyss jewels (types 7-11): AbyssPassive reproduces
//     readAbyssJewelLUT (socket walks; Zorath path + ascendancy picks) via
//     the open C# DatafileGenerator's algorithm. Differential:
//     tree/abyss_test.go, every record of all 5 bins.
package tree

import (
	"regexp"
	"sort"
	"sync"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
)

// --- pool tables (data/raw/conquerTables.json) ---

type altVersion struct {
	Key           int  `json:"k"`
	SmallAttr     bool `json:"smallAttr"` // small attribute passives replaced
	SmallNorm     bool `json:"smallNorm"` // small normal passives replaced
	MinAdd        int  `json:"minAdd"`
	MaxAdd        int  `json:"maxAdd"`
	NotableWeight int  `json:"notableWeight"` // notable replacement spawn weight
}

type altSkill struct {
	ID     string `json:"id"`
	TV     int    `json:"tv"`
	Types  []int  `json:"types"`
	Stats  int    `json:"stats"` // len(StatsKeys)
	Min    [4]int `json:"min"`
	Max    [4]int `json:"max"`
	W      uint32 `json:"w"`
	RMin   int    `json:"rmin"`
	RMax   int    `json:"rmax"`
	global int    // 337 + row index
}

type altAddition struct {
	ID     string `json:"id"`
	TV     int    `json:"tv"`
	Types  []int  `json:"types"`
	Stats  int    `json:"stats"`
	Min    [2]int `json:"min"`
	Max    [2]int `json:"max"`
	W      uint32 `json:"w"`
	global int    // row index
}

type conquerTables struct {
	Versions  []*altVersion  `json:"versions"`
	Skills    []*altSkill    `json:"skills"`
	Additions []*altAddition `json:"additions"`
	Passives  []struct {
		G int64 `json:"g"`
		T int   `json:"t"` // 1 SmallAttribute, 2 SmallNormal, 3 Notable
	} `json:"passives"`

	versionByKey   map[int]*altVersion
	skillPools     map[[2]int][]*altSkill // (passiveType, treeVersion)
	addPools       map[[2]int][]*altAddition
	typeByGraph    map[int64]int
	firstSkillByTV map[int]*altSkill // first row per tree version (keystone lookup)
}

var (
	conquerOnce  sync.Once
	conquerCache *conquerTables
)

func conquerData() *conquerTables {
	conquerOnce.Do(func() {
		t := &conquerTables{}
		data.RawDoc("conquertables", t)
		t.versionByKey = map[int]*altVersion{}
		for _, v := range t.Versions {
			t.versionByKey[v.Key] = v
		}
		t.skillPools = map[[2]int][]*altSkill{}
		t.firstSkillByTV = map[int]*altSkill{}
		for i, s := range t.Skills {
			s.global = 337 + i
			for _, pt := range s.Types {
				key := [2]int{pt, s.TV}
				t.skillPools[key] = append(t.skillPools[key], s)
			}
			if t.firstSkillByTV[s.TV] == nil {
				t.firstSkillByTV[s.TV] = s
			}
		}
		t.addPools = map[[2]int][]*altAddition{}
		for i, a := range t.Additions {
			a.global = i
			for _, pt := range a.Types {
				key := [2]int{pt, a.TV}
				t.addPools[key] = append(t.addPools[key], a)
			}
		}
		t.typeByGraph = map[int64]int{}
		for _, p := range t.Passives {
			t.typeByGraph[p.G] = p.T
		}
		conquerCache = t
	})
	return conquerCache
}

// --- the game's TinyMT32-variant RNG ---

type conquerRNG struct {
	state [4]uint32
}

func manipAlpha(v uint32) uint32 { return (v ^ (v >> 27)) * 0x19660D }
func manipBravo(v uint32) uint32 { return (v ^ (v >> 27)) * 0x5D588B65 }

func (g *conquerRNG) reset(graphID, seed uint32) {
	g.state = [4]uint32{0x40336050, 0xCFA3723C, 0x3CAC5F6F, 0x3793FDFF}
	s := &g.state
	index := uint32(1)
	for _, sd := range [2]uint32{graphID, seed} {
		r := manipAlpha(s[index%4] ^ s[(index+1)%4] ^ s[(index+3)%4])
		s[(index+1)%4] += r
		r += sd + index
		s[(index+2)%4] += r
		s[index%4] = r
		index = (index + 1) % 4
	}
	for i := 0; i < 5; i++ {
		r := manipAlpha(s[index%4] ^ s[(index+1)%4] ^ s[(index+3)%4])
		s[(index+1)%4] += r
		r += index
		s[(index+2)%4] += r
		s[index%4] = r
		index = (index + 1) % 4
	}
	for i := 0; i < 4; i++ {
		r := manipBravo(s[index%4] + s[(index+1)%4] + s[(index+3)%4])
		s[(index+1)%4] ^= r
		r -= index
		s[(index+2)%4] ^= r
		s[index%4] = r
		index = (index + 1) % 4
	}
	for i := 0; i < 8; i++ {
		g.nextState()
	}
}

func (g *conquerRNG) nextState() {
	s := &g.state
	a := s[3]
	b := (s[0] & 0x7FFFFFFF) ^ s[1] ^ s[2]
	a ^= a << 1
	b ^= (b >> 1) ^ a
	s[0] = s[1]
	s[1] = s[2]
	s[2] = a ^ (b << 10)
	s[3] = b
	if b&1 != 0 {
		s[1] ^= 0x8F7011EE
		s[2] ^= 0xFC78FF1F
	}
}

func (g *conquerRNG) genUint() uint32 {
	g.nextState()
	b := g.state[0] + (g.state[2] >> 8)
	a := g.state[3] ^ b
	if b&1 != 0 {
		a ^= 0x3793FDFF
	}
	return a
}

func (g *conquerRNG) genSingle(exclusiveMax uint32) uint32 {
	return g.genUint() % exclusiveMax
}

func (g *conquerRNG) generate(lo, hi uint32) uint32 {
	a := lo + 0x80000000
	b := hi + 0x80000000
	return g.genSingle(b-a+1) + a + 0x80000000
}

func (g *conquerRNG) genSigned(lo, hi int32) int32 {
	return int32(g.generate(uint32(lo), uint32(hi)))
}

// --- the generation algorithm (per cell) ---

type timelessAddRoll struct {
	add   *altAddition
	rolls []int32
}

type timelessCell struct {
	skill *altSkill // nil when the node is only augmented
	rolls []int32
	adds  []timelessAddRoll
}

// computeTimelessCell rolls one (node, seed) cell. graphID is the tree
// node id, seed the EFFECTIVE seed (Elegant Hubris already divided by 20),
// jewelType 1-6. Returns nil for a node outside the alterable set.
func computeTimelessCell(graphID int64, seed uint32, jewelType int) *timelessCell {
	t := conquerData()
	pt, ok := t.typeByGraph[graphID]
	if !ok {
		return nil
	}
	tv := t.versionByKey[jewelType]
	if tv == nil {
		return nil
	}
	var rng conquerRNG
	gid := uint32(graphID)

	// IsPassiveSkillReplaced
	var replaced bool
	switch {
	case pt == 3 && tv.NotableWeight >= 100:
		replaced = true
	case pt == 3 && tv.NotableWeight == 0:
		replaced = false
	case pt == 3:
		rng.reset(gid, seed)
		replaced = rng.generate(0, 100) < uint32(tv.NotableWeight)
	case pt == 1:
		replaced = tv.SmallAttr
	default:
		replaced = tv.SmallNorm
	}

	cell := &timelessCell{}
	if !replaced {
		// AugmentPassiveSkill
		rng.reset(gid, seed)
		if pt == 3 {
			rng.generate(0, 100)
		}
		cell.adds = rollTimelessAdditions(&rng, t, pt, jewelType, tv.MinAdd, tv.MaxAdd)
		return cell
	}

	// ReplacePassiveSkill (keystones never reach here: not in the passive set)
	pool := t.skillPools[[2]int{pt, jewelType}]
	rng.reset(gid, seed)
	if pt == 3 {
		rng.generate(0, 100)
	}
	var rolled *altSkill
	curWeight := uint32(0)
	for _, s := range pool {
		curWeight += s.W
		if rng.genSingle(curWeight) < s.W {
			rolled = s
		}
	}
	cell.skill = rolled
	elements := rolled.Stats
	if elements > 4 {
		elements = 4
	}
	for i := 0; i < elements; i++ {
		v := int32(rolled.Min[i])
		if rolled.Max[i] > rolled.Min[i] {
			v = rng.genSigned(int32(rolled.Min[i]), int32(rolled.Max[i]))
		}
		cell.rolls = append(cell.rolls, v)
	}
	if rolled.RMin == 0 && rolled.RMax == 0 {
		return cell
	}
	cell.adds = rollTimelessAdditions(&rng, t, pt, jewelType,
		tv.MinAdd+rolled.RMin, tv.MaxAdd+rolled.RMax)
	return cell
}

func rollTimelessAdditions(rng *conquerRNG, t *conquerTables, pt, jewelType, minAdd, maxAdd int) []timelessAddRoll {
	count := minAdd
	if maxAdd > minAdd {
		count = int(rng.generate(uint32(minAdd), uint32(maxAdd)))
	}
	pool := t.addPools[[2]int{pt, jewelType}]
	total := uint32(0)
	for _, a := range pool {
		total += a.W
	}
	out := make([]timelessAddRoll, 0, count)
	for n := 0; n < count; n++ {
		var rolled *altAddition
		for rolled == nil {
			roll := rng.genSingle(total)
			for _, a := range pool {
				if a.W > roll {
					rolled = a
					break
				}
				roll -= a.W
			}
		}
		elements := rolled.Stats
		if elements > 2 {
			elements = 2
		}
		var rolls []int32
		for j := 0; j < elements; j++ {
			v := int32(rolled.Min[j])
			if rolled.Max[j] > rolled.Min[j] {
				v = rng.genSigned(int32(rolled.Min[j]), int32(rolled.Max[j]))
			}
			rolls = append(rolls, v)
		}
		out = append(out, timelessAddRoll{add: rolled, rolls: rolls})
	}
	return out
}

// TimelessPassive reproduces data.readLUT's returned table (already
// converted to global ids): for Glorious Vanity the full record — the
// replacement id followed by its stat rolls, or the addition ids followed
// by their rolls — and for the other types the single replacement or
// addition id. The reference reads notable rows only for types 2-6; the
// caller preserves that gate through the same nodeIDList index checks.
func TimelessPassive(seed int64, nodeID int64, jewelType int) []int {
	if data.NodeIDList == nil {
		return nil
	}
	entry, known := data.NodeIDList[nodeID]
	if !known {
		// reference: ConPrintf ERROR, returns { }
		return []int{}
	}
	if jewelType == 5 {
		seed = seed / 20
	}
	cell := computeTimelessCell(nodeID, uint32(seed), jewelType)
	if cell == nil {
		return []int{}
	}
	if jewelType == 1 {
		var out []int
		if cell.skill != nil && len(cell.adds) == 0 {
			out = append(out, cell.skill.global)
			for _, r := range cell.rolls {
				out = append(out, int(r))
			}
		} else {
			for _, ar := range cell.adds {
				out = append(out, ar.add.global)
			}
			for _, ar := range cell.adds {
				for _, r := range ar.rolls {
					out = append(out, int(r))
				}
			}
		}
		return out
	}
	// Types 2-6: notables only (the reference indexes within sizeNotable).
	// The reference tests index <= sizeNotable, but row sizeNotable does not
	// exist in the fixed-stride files; its read produces an empty result. The
	// strict inequality reproduces the observable behavior.
	if int(entry.Index) >= data.NodeIDListSizeNotable {
		return []int{}
	}
	if cell.skill != nil {
		return []int{cell.skill.global}
	}
	if len(cell.adds) == 1 {
		return []int{cell.adds[0].add.global}
	}
	return []int{}
}

// ---------------------------------------------------------------------------
// Abyss historic jewels
// Abyss timeless jewel generation: a port of the open C# DatafileGenerator
// (LocalIdentity/TimelessJewelData, Generator branch) that produced PoB's
// shipped Abyss*.zip LUTs. Types 7-10 select nodes by a weighted random
// walk over the tree graph from the socket; type 11 (Zorath) modifies the
// allocated path to the class start and picks one notable per ascendancy.
// AbyssPassive reproduces Modules/DataAbyssJewelLookUpTableHelper.lua's
// readAbyssJewelLUT contract by computing records instead of reading them.
//
// Unlike the timeless-jewel generator, this one draws with rejection sampling
// (uniform modulo over the accepted range) — proven per-bit against the
// bins by the differential; the timeless bins predate that change and keep
// plain modulo.

const abyssDefaultSize = 60

type abyssNode struct {
	gid   int64
	ins   []int64
	outs  []int64
	ptype int // 0 none, 1 smallAttr, 2 smallNorm, 3 notable, 4 keystone, 5 ascNotable
	walkW uint32

	notable       bool
	mastery       bool
	charstart     bool
	transformable bool
	specialCand   bool // ABYN node-record candidate (transformable or eligible asc notable)
	ascEligible   bool // IsEligibleAscendancyNotable
	ascName       string
	connected     bool // nodes PoB keeps that the generator's tree lacks are orphans
}

type abyssWorld struct {
	nodes   map[int64]*abyssNode
	sockets []*abyssNode // base jewel sockets, ordered by graph id

	zorathNodes map[int64]bool // gids with an ABYN block
	ascOrder    []string       // supported ascendancies, ordinal order
	ascStart    map[string]*abyssNode
	ascClass    map[string]*abyssNode
	ascCosts    map[string]map[int64]int
}

var abyssAttrRe = regexp.MustCompile(`^\+\d+ to (Strength|Dexterity|Intelligence)$`)

// abyssData is the tree's generator world, built on first use.
func (t *Tree) abyssData() *abyssWorld {
	t.abyssOnce.Do(func() { t.abyss = buildAbyssWorld(t.doc) })
	return t.abyss
}

// buildAbyssWorld builds the generator's view of the tree document. It
// reads the document rather than Tree.Nodes because the generator's tree
// still holds the legacy alternate-ascendancy nodes Load drops (their
// ascendancies get Zorath selections too).
func buildAbyssWorld(doc *schema.PassiveTree) *abyssWorld {
	w := &abyssWorld{nodes: map[int64]*abyssNode{}}

	for _, nd := range doc.Nodes {
		gid := nd.Skill
		if gid == 0 {
			continue
		}
		if _, dup := w.nodes[gid]; dup {
			continue
		}
		n := &abyssNode{gid: gid}
		n.ins = idsOf(nd.In)
		n.outs = idsOf(nd.Out)
		n.connected = len(n.ins)+len(n.outs) > 0
		n.notable = nd.IsNotable
		n.mastery = nd.IsMastery
		n.charstart = nd.ClassStartIndex != nil
		n.ascName = nd.AscendancyName
		keystone := nd.IsKeystone
		socket := nd.IsJewelSocket
		proxy := nd.IsProxy
		blight := nd.IsBlighted
		justIcon := nd.IsJustIcon
		cluster := nd.Orbit == nil
		attr := len(nd.Stats) == 1 && abyssAttrRe.MatchString(nd.Stats[0])

		switch {
		case n.ascName != "" && n.notable:
			n.ptype = 5
		case socket:
			n.ptype = 0
		case keystone:
			n.ptype = 4
		case n.notable:
			n.ptype = 3
		case attr:
			n.ptype = 1
		default:
			n.ptype = 2
		}
		switch {
		case cluster:
			n.walkW = 5
		case n.notable:
			n.walkW = 25
		case attr:
			n.walkW = 1
		default:
			n.walkW = 5
		}
		n.transformable = !(cluster || n.ascName != "" || proxy || n.mastery || socket || blight || n.charstart || justIcon)
		n.specialCand = n.transformable ||
			(n.ascName != "" && n.notable && !proxy && !n.mastery && !justIcon)
		n.ascEligible = !(proxy || n.mastery || socket || blight || justIcon ||
			nd.IsAscendancyStart || nd.IsMultipleChoice || nd.IsMultipleChoiceOption)

		isClusterExpSocket := nd.ExpansionJewel != nil && nd.ExpansionJewel.Parent != nil
		if socket && !isClusterExpSocket && n.ascName == "" {
			w.sockets = append(w.sockets, n)
		}
		w.nodes[gid] = n
	}
	sort.Slice(w.sockets, func(i, j int) bool { return w.sockets[i].gid < w.sockets[j].gid })

	// Zorath's node blocks: special candidates the generator's tree contains
	// (orphaned legacy nodes excluded) that any type-11 pool can modify.
	t := conquerData()
	w.zorathNodes = map[int64]bool{}
	for gid, n := range w.nodes {
		if n.specialCand && n.connected && abyssCanModify(t, n, 11) {
			w.zorathNodes[gid] = true
		}
	}

	// Ascendancy machinery: start node per ascendancy (lowest graph id with
	// a class-start connection; validated exhaustively by the differential),
	// its class start, and BFS costs from the ascendancy start.
	w.ascStart = map[string]*abyssNode{}
	w.ascClass = map[string]*abyssNode{}
	w.ascCosts = map[string]map[int64]int{}
	var byGid []*abyssNode
	for _, n := range w.nodes {
		byGid = append(byGid, n)
	}
	sort.Slice(byGid, func(i, j int) bool { return byGid[i].gid < byGid[j].gid })
	for _, n := range byGid {
		if n.ascName == "" || w.ascStart[n.ascName] != nil {
			continue
		}
		if cls := w.charStartConnection(n); cls != nil {
			w.ascStart[n.ascName] = n
			w.ascClass[n.ascName] = cls
		}
	}
	for name, start := range w.ascStart {
		costs := map[int64]int{start.gid: 0}
		queue := []*abyssNode{start}
		for len(queue) > 0 {
			sel := queue[0]
			queue = queue[1:]
			for _, gid := range append(append([]int64{}, sel.ins...), sel.outs...) {
				if _, seen := costs[gid]; seen {
					continue
				}
				c := w.nodes[gid]
				if c == nil || c.ascName != name {
					continue
				}
				costs[gid] = costs[sel.gid] + 1
				queue = append(queue, c)
			}
		}
		w.ascCosts[name] = costs
		w.ascOrder = append(w.ascOrder, name)
	}
	sort.Strings(w.ascOrder)
	return w
}

func (w *abyssWorld) charStartConnection(n *abyssNode) *abyssNode {
	for _, gid := range append(append([]int64{}, n.ins...), n.outs...) {
		if c := w.nodes[gid]; c != nil && c.charstart {
			return c
		}
	}
	return nil
}

func abyssCanModify(t *conquerTables, n *abyssNode, jewelType int) bool {
	return len(t.skillPools[[2]int{n.ptype, jewelType}]) > 0 ||
		len(t.addPools[[2]int{n.ptype, jewelType}]) > 0
}

// --- rejection-sampling RNG variants (the abyss generator's Generate) ---

func (g *conquerRNG) genSingleRejected(exclusiveMax uint32) uint32 {
	if exclusiveMax <= 1 {
		return 0 // no state consumed
	}
	limit := uint64(1<<32) / uint64(exclusiveMax) * uint64(exclusiveMax)
	for {
		v := g.genUint()
		if uint64(v) < limit {
			return v % exclusiveMax
		}
	}
}

func (g *conquerRNG) generateRejected(lo, hi uint32) uint32 {
	a := lo + 0x80000000
	b := hi + 0x80000000
	return g.genSingleRejected(b-a+1) + a + 0x80000000
}

// abyssAscRNG is the generator's SingleSeedRandomNumberGenerator (standard
// TinyMT32 single-seed init).
type abyssAscRNG struct {
	state [4]uint32
}

func newAbyssAscRNG(seed uint32) *abyssAscRNG {
	g := &abyssAscRNG{state: [4]uint32{seed, 0x8F7011EE, 0xFC78FF1F, 0x3793FDFF}}
	// state[1..4] in the C# maps to state[0..3] here (its slot 0 is a counter).
	for i := uint32(1); i < 8; i++ {
		cur := i & 3
		prev := (i - 1) & 3
		g.state[cur] ^= i + 0x6C078965*(g.state[prev]^(g.state[prev]>>30))
	}
	for i := 0; i < 8; i++ {
		g.nextState()
	}
	return g
}

func (g *abyssAscRNG) nextState() {
	s := &g.state
	a := s[3]
	b := (s[0] & 0x7FFFFFFF) ^ s[1] ^ s[2]
	a ^= a << 1
	b ^= (b >> 1) ^ a
	s[0] = s[1]
	s[1] = s[2]
	s[2] = a ^ (b << 10)
	s[3] = b
	if b&1 != 0 {
		s[1] ^= 0x8F7011EE
		s[2] ^= 0xFC78FF1F
	}
}

func (g *abyssAscRNG) generate(exclusiveMax uint32) uint32 {
	if exclusiveMax <= 1 {
		return 0
	}
	limit := uint64(1<<32) / uint64(exclusiveMax) * uint64(exclusiveMax)
	for {
		g.nextState()
		s := &g.state
		b := s[0] + (s[2] >> 8)
		a := s[3] ^ b
		if b&1 != 0 {
			a ^= 0x3793FDFF
		}
		if uint64(a) < limit {
			return a % exclusiveMax
		}
	}
}

// --- the walk (AbyssTreeManager.SelectAffectedPassives) ---

type abyssWeighted struct {
	node   *abyssNode
	weight uint32
}

func (w *abyssWorld) walk(socketGid int64, seed uint32, abyssSize int) []*abyssNode {
	var rng conquerRNG
	rng.reset(uint32(socketGid), seed)
	frontier := []abyssWeighted{{w.nodes[socketGid], 1}}
	seen := map[int64]bool{}
	var affected []*abyssNode
	total := uint32(1)
	for i := 0; i < abyssSize && len(frontier) > 0; i++ {
		roll := rng.genSingleRejected(total)
		idx := 0
		for j := range frontier {
			if frontier[j].weight > roll {
				idx = j
				break
			}
			roll -= frontier[j].weight
		}
		sel := frontier[idx]
		frontier = append(frontier[:idx], frontier[idx+1:]...)
		total -= sel.weight
		for _, conns := range [2][]int64{sel.node.ins, sel.node.outs} {
			for _, gid := range conns {
				if seen[gid] {
					continue
				}
				seen[gid] = true
				c := w.nodes[gid]
				if c == nil || c.mastery || c.charstart {
					continue
				}
				frontier = append(frontier, abyssWeighted{c, c.walkW})
				total += c.walkW
			}
		}
		if sel.node.transformable {
			affected = append(affected, sel.node)
		}
	}
	return affected
}

// --- per-node modification (WriteModification's content) ---

// AbyssComponent is one modification component the way readAbyssJewelLUT
// returns it; ID is the global id (already converted), Rolls the stat
// rolls in stat order.
type AbyssComponent struct {
	Kind  AbyssComponentKind
	ID    int
	Rolls []int32
}

func abyssModification(t *conquerTables, n *abyssNode, jewelType int, seed uint32) []AbyssComponent {
	tv := t.versionByKey[jewelType]
	var rng conquerRNG
	gid := uint32(n.gid)

	var replaced bool
	switch {
	case n.ptype == 5 || n.ptype == 4:
		replaced = true
	case n.notable:
		if tv.NotableWeight >= 100 {
			replaced = true
		} else {
			rng.reset(gid, seed)
			replaced = rng.generateRejected(0, 100) < uint32(tv.NotableWeight)
		}
	case n.ptype == 1:
		replaced = tv.SmallAttr
	default:
		replaced = tv.SmallNorm
	}

	if !replaced {
		rng.reset(gid, seed)
		if n.ptype == 3 {
			rng.generateRejected(0, 100)
		}
		return abyssRollAdditions(t, &rng, n.ptype, jewelType, tv.MinAdd, tv.MaxAdd, nil)
	}

	if n.ptype == 4 {
		// keystone: the tree version's first alternate skill (its keystone row)
		ks := t.firstSkillByTV[jewelType]
		if ks == nil || !containsInt(ks.Types, 4) {
			return nil
		}
		return []AbyssComponent{{Kind: ComponentReplace, ID: ks.global, Rolls: []int32{int32(ks.Min[0])}}}
	}

	rng.reset(gid, seed)
	if n.ptype == 3 {
		rng.generateRejected(0, 100)
	}
	pool := t.skillPools[[2]int{n.ptype, jewelType}]
	var rolled *altSkill
	cur := uint32(0)
	for _, s := range pool {
		cur += s.W
		if rng.genSingleRejected(cur) < s.W {
			rolled = s
		}
	}
	elements := rolled.Stats
	if elements > 4 {
		elements = 4
	}
	rolls := make([]int32, 0, elements)
	for i := 0; i < elements; i++ {
		v := int32(rolled.Min[i])
		if rolled.Max[i] > rolled.Min[i] {
			v = int32(rolled.Min[i]) + int32(rng.genSingleRejected(uint32(rolled.Max[i]-rolled.Min[i]+1)))
		}
		rolls = append(rolls, v)
	}
	out := []AbyssComponent{{Kind: ComponentReplace, ID: rolled.global, Rolls: rolls}}
	if rolled.RMin == 0 && rolled.RMax == 0 {
		return out
	}
	return abyssRollAdditions(t, &rng, n.ptype, jewelType,
		tv.MinAdd+rolled.RMin, tv.MaxAdd+rolled.RMax, out)
}

func abyssRollAdditions(t *conquerTables, rng *conquerRNG, pt, jewelType, minAdd, maxAdd int, out []AbyssComponent) []AbyssComponent {
	count := minAdd
	if maxAdd > minAdd {
		count = int(rng.generateRejected(uint32(minAdd), uint32(maxAdd)))
	}
	pool := t.addPools[[2]int{pt, jewelType}]
	total := uint32(0)
	for _, a := range pool {
		total += a.W
	}
	for k := 0; k < count; k++ {
		var rolled *altAddition
		for rolled == nil {
			roll := rng.genSingleRejected(total)
			for _, a := range pool {
				if a.W > roll {
					rolled = a
					break
				}
				roll -= a.W
			}
		}
		elements := rolled.Stats
		if elements > 2 {
			elements = 2
		}
		rolls := make([]int32, 0, elements)
		for j := 0; j < elements; j++ {
			v := int32(rolled.Min[j])
			if rolled.Max[j] > rolled.Min[j] {
				v = int32(rolled.Min[j]) + int32(rng.genSingleRejected(uint32(rolled.Max[j]-rolled.Min[j]+1)))
			}
			rolls = append(rolls, v)
		}
		out = append(out, AbyssComponent{Kind: ComponentAdd, ID: rolled.global, Rolls: rolls})
	}
	return out
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// --- ascendancy selection (AbyssAscendancyManager.SelectAffectedNotables) ---

func (w *abyssWorld) selectAscendancyNotables(name string, seed uint32, requested int) []*abyssNode {
	start := w.ascStart[name]
	if start == nil {
		return nil
	}
	costs := w.ascCosts[name]
	rng := newAbyssAscRNG(seed)
	frontier := []*abyssNode{w.ascClass[name], start}
	seen := map[int64]bool{w.ascClass[name].gid: true, start.gid: true}
	var affected []*abyssNode
	for len(frontier) > 0 && len(affected) < requested {
		idx := 0
		if len(frontier) > 1 {
			idx = int(rng.generate(uint32(len(frontier))))
		}
		sel := frontier[idx]
		frontier = append(frontier[:idx], frontier[idx+1:]...)
		for _, conns := range [2][]int64{sel.ins, sel.outs} {
			for _, gid := range conns {
				if seen[gid] {
					continue
				}
				seen[gid] = true
				c := w.nodes[gid]
				if c == nil || c.ascName != name {
					continue
				}
				if c.notable && c.ascEligible {
					if cost, ok := costs[gid]; ok && cost < 4 {
						affected = append(affected, c)
					}
				}
				frontier = append(frontier, c)
			}
		}
	}
	return affected
}

// --- the readAbyssJewelLUT replacement ---

// AbyssPassive reproduces readAbyssJewelLUT: for types 7-10 the
// affected nodes around the socket, for type 11 (with a path) the
// modifications along the path plus one selected notable per ascendancy.
func (t *Tree) AbyssPassive(seed int64, socketID int64, jewelType int, path map[int64]bool) map[int64][]AbyssComponent {
	if seed < 100 || seed > 8000 {
		return map[int64][]AbyssComponent{}
	}
	w := t.abyssData()
	tables := conquerData()
	s := uint32(seed)

	if jewelType >= 7 && jewelType <= 10 {
		sock := w.nodes[socketID]
		if sock == nil || !socketInBase(w, socketID) {
			return map[int64][]AbyssComponent{}
		}
		affected := map[int64][]AbyssComponent{}
		for _, n := range w.walk(socketID, s, abyssDefaultSize) {
			if abyssCanModify(tables, n, jewelType) {
				affected[n.gid] = abyssModification(tables, n, jewelType, s)
			}
		}
		return affected
	}
	if jewelType != 11 || path == nil {
		return map[int64][]AbyssComponent{}
	}

	affected := map[int64][]AbyssComponent{}
	for _, gid := range sortedNodeIDs(path) {
		if !w.zorathNodes[gid] {
			continue
		}
		if mod := abyssModification(tables, w.nodes[gid], 11, s); len(mod) > 0 {
			affected[gid] = mod
		}
	}
	for _, name := range w.ascOrder {
		for _, n := range w.selectAscendancyNotables(name, s, 1) {
			var mod []AbyssComponent
			if w.zorathNodes[n.gid] {
				mod = abyssModification(tables, n, 11, s)
			}
			affected[n.gid] = mod
		}
	}
	return affected
}

func socketInBase(w *abyssWorld, socketID int64) bool {
	for _, s := range w.sockets {
		if s.gid == socketID {
			return true
		}
	}
	return false
}
