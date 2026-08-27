// Timeless jewel node generation: a port of the reverse-engineered
// algorithm behind the game's alternate-tree rolls (the same algorithm
// that generated PoB's shipped Data/TimelessJewelData LUT bins — see the
// exhaustive differential in test/timeless_test.go). readTimelessLUT
// reproduces Modules/DataLegionLookUpTableHelper.lua's readLUT contract
// (global ids + rolls) by computing the cell instead of reading a table.
package tree

import (
	"strconv"
	"sync"

	"github.com/MissingL-tter/missingPassives/data"
)

// --- pool tables (data/raw/timelessTables.json) ---

type timelessVersion struct {
	Key           int  `json:"k"`
	SmallAttr     bool `json:"smallAttr"` // small attribute passives replaced
	SmallNorm     bool `json:"smallNorm"` // small normal passives replaced
	MinAdd        int  `json:"minAdd"`
	MaxAdd        int  `json:"maxAdd"`
	NotableWeight int  `json:"notableWeight"` // notable replacement spawn weight
}

type timelessSkill struct {
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

type timelessAddition struct {
	ID     string `json:"id"`
	TV     int    `json:"tv"`
	Types  []int  `json:"types"`
	Stats  int    `json:"stats"`
	Min    [2]int `json:"min"`
	Max    [2]int `json:"max"`
	W      uint32 `json:"w"`
	global int    // row index
}

type timelessTables struct {
	Versions  []*timelessVersion  `json:"versions"`
	Skills    []*timelessSkill    `json:"skills"`
	Additions []*timelessAddition `json:"additions"`
	Passives  []struct {
		G int64 `json:"g"`
		T int   `json:"t"` // 1 SmallAttribute, 2 SmallNormal, 3 Notable
	} `json:"passives"`

	versionByKey   map[int]*timelessVersion
	skillPools     map[[2]int][]*timelessSkill // (passiveType, treeVersion)
	addPools       map[[2]int][]*timelessAddition
	typeByGraph    map[int64]int
	firstSkillByTV map[int]*timelessSkill // first row per tree version (keystone lookup)
}

var (
	timelessOnce sync.Once
	timeless     *timelessTables
)

func timelessData() *timelessTables {
	timelessOnce.Do(func() {
		t := &timelessTables{}
		data.RawDoc("timelessTables", t)
		t.versionByKey = map[int]*timelessVersion{}
		for _, v := range t.Versions {
			t.versionByKey[v.Key] = v
		}
		t.skillPools = map[[2]int][]*timelessSkill{}
		t.firstSkillByTV = map[int]*timelessSkill{}
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
		t.addPools = map[[2]int][]*timelessAddition{}
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
		timeless = t
	})
	return timeless
}

// --- the game's TinyMT32-variant RNG ---

type timelessRNG struct {
	state [4]uint32
}

func manipAlpha(v uint32) uint32 { return (v ^ (v >> 27)) * 0x19660D }
func manipBravo(v uint32) uint32 { return (v ^ (v >> 27)) * 0x5D588B65 }

func (g *timelessRNG) reset(graphID, seed uint32) {
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

func (g *timelessRNG) nextState() {
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

func (g *timelessRNG) genUint() uint32 {
	g.nextState()
	b := g.state[0] + (g.state[2] >> 8)
	a := g.state[3] ^ b
	if b&1 != 0 {
		a ^= 0x3793FDFF
	}
	return a
}

func (g *timelessRNG) genSingle(exclusiveMax uint32) uint32 {
	return g.genUint() % exclusiveMax
}

func (g *timelessRNG) generate(lo, hi uint32) uint32 {
	a := lo + 0x80000000
	b := hi + 0x80000000
	return g.genSingle(b-a+1) + a + 0x80000000
}

func (g *timelessRNG) genSigned(lo, hi int32) int32 {
	return int32(g.generate(uint32(lo), uint32(hi)))
}

// --- the generation algorithm (per cell) ---

type timelessAddRoll struct {
	add   *timelessAddition
	rolls []int32
}

type timelessCell struct {
	skill *timelessSkill // nil when the node is only augmented
	rolls []int32
	adds  []timelessAddRoll
}

// computeTimelessCell rolls one (node, seed) cell. graphID is the tree
// node id, seed the EFFECTIVE seed (Elegant Hubris already divided by 20),
// jewelType 1-6. Returns nil for a node outside the alterable set.
func computeTimelessCell(graphID int64, seed uint32, jewelType int) *timelessCell {
	t := timelessData()
	pt, ok := t.typeByGraph[graphID]
	if !ok {
		return nil
	}
	tv := t.versionByKey[jewelType]
	if tv == nil {
		return nil
	}
	var rng timelessRNG
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
	var rolled *timelessSkill
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

func rollTimelessAdditions(rng *timelessRNG, t *timelessTables, pt, jewelType, minAdd, maxAdd int) []timelessAddRoll {
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
		var rolled *timelessAddition
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

// ReadTimelessLUT reproduces data.readLUT's returned table (already
// converted to global ids): for Glorious Vanity the full record — the
// replacement id followed by its stat rolls, or the addition ids followed
// by their rolls — and for the other types the single replacement or
// addition id. The reference reads notable rows only for types 2-6; the
// caller preserves that gate through the same nodeIDList index checks.
func ReadTimelessLUT(seed int64, nodeID int64, jewelType int) []int {
	nl := data.NodeIDList
	if nl == nil {
		return nil
	}
	entry := nodeIDListEntry(nl, nodeID)
	if entry == nil {
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
	if !nodeIDListIndexIsNotable(nl, nodeID) {
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

func nodeIDListEntry(nl map[string]any, nodeID int64) map[string]any {
	v, _ := nl[strconv.FormatInt(nodeID, 10)].(map[string]any)
	return v
}

func nodeIDListIndexIsNotable(nl map[string]any, nodeID int64) bool {
	entry := nodeIDListEntry(nl, nodeID)
	if entry == nil {
		return false
	}
	index, _ := entry["index"].(float64)
	sizeNotable, _ := nl["sizeNotable"].(float64)
	// The reference tests index <= sizeNotable, but row sizeNotable does not
	// exist in the fixed-stride files; its read produces an empty result. The
	// strict inequality reproduces the observable behavior.
	return index < sizeNotable
}
