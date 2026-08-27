// Abyss timeless jewel generation: a port of the open C# DatafileGenerator
// (LocalIdentity/TimelessJewelData, Generator branch) that produced PoB's
// shipped Abyss*.zip LUTs. Types 7-10 select nodes by a weighted random
// walk over the tree graph from the socket; type 11 (Zorath) modifies the
// allocated path to the class start and picks one notable per ascendancy.
// ReadAbyssJewelLUT reproduces Modules/DataAbyssJewelLookUpTableHelper.lua's
// readAbyssJewelLUT contract by computing records instead of reading them.
//
// Unlike the legion generator, this one draws with rejection sampling
// (uniform modulo over the accepted range) — proven per-bit against the
// bins by the differential; the legion bins predate that change and keep
// plain modulo.
package tree

import (
	"regexp"
	"sort"
	"sync"

	"github.com/MissingL-tter/missingPassives/data"
)

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

var (
	abyssOnce  sync.Once
	abyssState *abyssWorld
)

var abyssAttrRe = regexp.MustCompile(`^\+\d+ to (Strength|Dexterity|Intelligence)$`)

func abyssData() *abyssWorld {
	abyssOnce.Do(func() { abyssState = buildAbyssWorld() })
	return abyssState
}

func buildAbyssWorld() *abyssWorld {
	var doc struct {
		Nodes map[string]map[string]any `json:"nodes"`
	}
	data.RawDoc("tree_3_29", &doc)
	w := &abyssWorld{nodes: map[int64]*abyssNode{}}

	for _, raw := range doc.Nodes {
		gid := int64(num(raw["skill"]))
		if gid == 0 {
			continue
		}
		if _, dup := w.nodes[gid]; dup {
			continue
		}
		n := &abyssNode{gid: gid}
		n.ins = idList(raw["in"])
		n.outs = idList(raw["out"])
		n.connected = len(n.ins)+len(n.outs) > 0
		n.notable = boolean(raw["isNotable"])
		n.mastery = boolean(raw["isMastery"])
		n.charstart = raw["classStartIndex"] != nil
		n.ascName = str(raw["ascendancyName"])
		keystone := boolean(raw["isKeystone"])
		socket := boolean(raw["isJewelSocket"])
		proxy := boolean(raw["isProxy"])
		blight := boolean(raw["isBlighted"])
		justIcon := boolean(raw["isJustIcon"])
		cluster := raw["orbit"] == nil
		stats := strList(raw["stats"])
		attr := len(stats) == 1 && abyssAttrRe.MatchString(stats[0])

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
			boolean(raw["isAscendancyStart"]) || boolean(raw["isMultipleChoice"]) || boolean(raw["isMultipleChoiceOption"]))

		isClusterExpSocket := false
		if ej, ok := raw["expansionJewel"].(map[string]any); ok {
			_, isClusterExpSocket = ej["parent"]
		}
		if socket && !isClusterExpSocket && n.ascName == "" {
			w.sockets = append(w.sockets, n)
		}
		w.nodes[gid] = n
	}
	sort.Slice(w.sockets, func(i, j int) bool { return w.sockets[i].gid < w.sockets[j].gid })

	// Zorath's node blocks: special candidates the generator's tree contains
	// (orphaned legacy nodes excluded) that any type-11 pool can modify.
	t := timelessData()
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

func abyssCanModify(t *timelessTables, n *abyssNode, jewelType int) bool {
	return len(t.skillPools[[2]int{n.ptype, jewelType}]) > 0 ||
		len(t.addPools[[2]int{n.ptype, jewelType}]) > 0
}

// --- rejection-sampling RNG variants (the abyss generator's Generate) ---

func (g *timelessRNG) genSingleRejected(exclusiveMax uint32) uint32 {
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

func (g *timelessRNG) generateRejected(lo, hi uint32) uint32 {
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
	var rng timelessRNG
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
// returns it: type 1 replaces the node, type 2 adds stats; ID is the
// global id (already converted), Rolls the stat rolls in stat order.
type AbyssComponent struct {
	Type  int
	ID    int
	Rolls []int32
}

func abyssModification(t *timelessTables, n *abyssNode, jewelType int, seed uint32) []AbyssComponent {
	tv := t.versionByKey[jewelType]
	var rng timelessRNG
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
		return []AbyssComponent{{Type: 1, ID: ks.global, Rolls: []int32{int32(ks.Min[0])}}}
	}

	rng.reset(gid, seed)
	if n.ptype == 3 {
		rng.generateRejected(0, 100)
	}
	pool := t.skillPools[[2]int{n.ptype, jewelType}]
	var rolled *timelessSkill
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
	out := []AbyssComponent{{Type: 1, ID: rolled.global, Rolls: rolls}}
	if rolled.RMin == 0 && rolled.RMax == 0 {
		return out
	}
	return abyssRollAdditions(t, &rng, n.ptype, jewelType,
		tv.MinAdd+rolled.RMin, tv.MaxAdd+rolled.RMax, out)
}

func abyssRollAdditions(t *timelessTables, rng *timelessRNG, pt, jewelType, minAdd, maxAdd int, out []AbyssComponent) []AbyssComponent {
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
		var rolled *timelessAddition
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
		out = append(out, AbyssComponent{Type: 2, ID: rolled.global, Rolls: rolls})
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

// ReadAbyssJewelLUT reproduces readAbyssJewelLUT: for types 7-10 the
// affected nodes around the socket, for type 11 (with a path) the
// modifications along the path plus one selected notable per ascendancy.
func ReadAbyssJewelLUT(seed int64, socketID int64, jewelType int, path map[int64]bool) map[int64][]AbyssComponent {
	if seed < 100 || seed > 8000 {
		return map[int64][]AbyssComponent{}
	}
	w := abyssData()
	t := timelessData()
	s := uint32(seed)

	if jewelType >= 7 && jewelType <= 10 {
		sock := w.nodes[socketID]
		if sock == nil || !socketInBase(w, socketID) {
			return map[int64][]AbyssComponent{}
		}
		affected := map[int64][]AbyssComponent{}
		for _, n := range w.walk(socketID, s, abyssDefaultSize) {
			if abyssCanModify(t, n, jewelType) {
				affected[n.gid] = abyssModification(t, n, jewelType, s)
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
		if mod := abyssModification(t, w.nodes[gid], 11, s); len(mod) > 0 {
			affected[gid] = mod
		}
	}
	for _, name := range w.ascOrder {
		for _, n := range w.selectAscendancyNotables(name, s, 1) {
			var mod []AbyssComponent
			if w.zorathNodes[n.gid] {
				mod = abyssModification(t, n, 11, s)
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
