// luaPairsIntKeys reproduces the shipped LuaJIT's pairs() iteration order
// over a table built by inserting small non-negative integer keys one at a
// time (the GV might/legacy additions merge builds exactly such a table).
// TEST-SIDE ONLY: production merges those additions in first-seen order
// and records the merge (SpecNode.TimelessAdditions); the differential
// permutes into this order before byte-comparing. It emulates lj_tab.c:
// array-part sizing, the number hash, collision handling with
// main-position eviction, and rehash-on-full — verified exhaustively
// against the repo's luajit for the domain (keys 0..37, sequences up to
// length 4) by TestLuaPairsAgainstLuaJIT.
package test

import "math"

import "math/bits"

type luaTabNode struct {
	used bool
	key  int
	next int // -1 = none
}

type luaTab struct {
	array []bool // array part: key k present when array[k]
	akeys []int
	node  []luaTabNode
	hmask int
	free  int // freetop scan position (one past last free candidate)
}

func luaHashRot(lo, hi uint32) uint32 {
	lo ^= hi
	hi = bits.RotateLeft32(hi, 14)
	lo -= hi
	hi = bits.RotateLeft32(hi, 5)
	hi ^= lo
	hi -= bits.RotateLeft32(lo, 13)
	return hi
}

// luaHashNum hashes the double representation of k the way hashnum does.
func luaHashNum(k int) uint32 {
	b := math.Float64bits(float64(k))
	lo := uint32(b)
	hi := uint32(b>>32) << 1
	return luaHashRot(lo, hi)
}

func newLuaTab(asize, hbits int) *luaTab {
	t := &luaTab{}
	if asize > 0 {
		t.array = make([]bool, asize)
	}
	hsize := 0
	if hbits > 0 {
		hsize = 1 << hbits
	}
	if hsize > 0 {
		t.node = make([]luaTabNode, hsize)
		for i := range t.node {
			t.node[i].next = -1
		}
		t.hmask = hsize - 1
		t.free = hsize
	}
	return t
}

func (t *luaTab) mainPos(k int) int {
	if t.node == nil {
		return -1
	}
	return int(luaHashNum(k) & uint32(t.hmask))
}

func (t *luaTab) has(k int) bool {
	if k >= 0 && k < len(t.array) {
		return t.array[k]
	}
	if t.node == nil {
		return false
	}
	for i := t.mainPos(k); i != -1; i = t.node[i].next {
		if t.node[i].used && t.node[i].key == k {
			return true
		}
	}
	return false
}

func (t *luaTab) set(k int) {
	if t.has(k) {
		return
	}
	if k >= 0 && k < len(t.array) {
		t.array[k] = true
		t.akeys = append(t.akeys, k)
		return
	}
	t.newKey(k)
}

func (t *luaTab) newKey(k int) {
	if t.node == nil {
		t.rehash(k)
		t.set(k)
		return
	}
	n := t.mainPos(k)
	if t.node[n].used {
		// find a free node scanning down from freetop
		free := -1
		for f := t.free; f > 0; f-- {
			if !t.node[f-1].used {
				free = f - 1
				t.free = f - 1
				break
			}
		}
		if free == -1 {
			t.rehash(k)
			t.set(k)
			return
		}
		collideMain := t.mainPos(t.node[n].key)
		if collideMain != n {
			// colliding node is not in its main position: move it out
			prev := collideMain
			for t.node[prev].next != n {
				prev = t.node[prev].next
			}
			t.node[prev].next = free
			t.node[free] = t.node[n]
			t.node[n] = luaTabNode{used: true, key: k, next: -1}
			return
		}
		// new key goes into the free node, chained after n
		t.node[free] = luaTabNode{used: true, key: k, next: t.node[n].next}
		t.node[n].next = free
		return
	}
	t.node[n] = luaTabNode{used: true, key: k, next: -1}
}

// rehash grows the table for pending key k, re-inserting existing keys the
// way lj_tab_resize does (array part ascending, then hash nodes in slot
// order).
func (t *luaTab) rehash(k int) {
	keys := t.keys()
	total := len(keys) + 1

	// bestasize over the int keys including the new one
	var bins [32]uint32
	na := uint32(0)
	countInt := func(key int) {
		if key >= 0 {
			b := 0
			if key > 2 { // bins[lj_fls(k-1)]
				b = 31 - bits.LeadingZeros32(uint32(key-1))
			}
			bins[b]++
			na++
		}
	}
	for _, key := range keys {
		countInt(key)
	}
	countInt(k)

	asize := 0
	{
		nn := na
		sum := uint32(0)
		naBest := uint32(0)
		for b := 0; 2*nn > (uint32(1) << b); b++ {
			if bins[b] > 0 {
				sum += bins[b]
				if 2*sum > (uint32(1) << b) {
					asize = (2 << b) + 1
					naBest = sum
				}
			}
			if sum == nn {
				break
			}
		}
		na = naBest
	}

	hnum := total - int(na)
	hbits := 0
	if hnum > 0 {
		hbits = 1
		for (1 << hbits) < hnum {
			hbits++
		}
	}

	old := *t
	*t = *newLuaTab(asize, hbits)
	// reinsert: array part ascending, then hash slots in order
	for key := 0; key < len(old.array); key++ {
		if old.array[key] {
			t.set(key)
		}
	}
	for i := range old.node {
		if old.node[i].used {
			t.set(old.node[i].key)
		}
	}
}

func (t *luaTab) keys() []int {
	var out []int
	for k := 0; k < len(t.array); k++ {
		if t.array[k] {
			out = append(out, k)
		}
	}
	for i := range t.node {
		if t.node[i].used {
			out = append(out, t.node[i].key)
		}
	}
	return out
}

// luaPairsIntKeys inserts keys in first-seen order and returns pairs()
// iteration order: array part ascending, then hash nodes by slot index.
func luaPairsIntKeys(insertOrder []int) []int {
	t := newLuaTab(0, 0)
	for _, k := range insertOrder {
		t.set(k)
	}
	return t.keys()
}
