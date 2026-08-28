// A faithful simulation of LuaJIT 2.1's table implementation (lj_tab.c) for
// number keys, so pairs() iteration order over number-keyed tables can be
// reproduced byte-for-byte (mods.lua writes tradeHashes via pairs()).
//
// #EVAL: archive parity only — the tradeHashes ordering is a hash-table
// artifact, not meaningful data; once the format is Go-owned, sort the keys
// and delete this file.
//
// Only what that use needs is implemented: t[k]=v inserts of non-nil values
// with number keys into a table created empty, then a full traversal.

package luarender

import (
	"math"
	"math/bits"
)

const (
	ljMaxHBits = 26
	ljMaxABits = 28
	ljMaxASize = (1 << (ljMaxABits - 1)) + 1
)

type ljValue struct {
	key float64
	val any
	set bool // val is occupied (non-nil in Lua terms)
}

type ljNode struct {
	ljValue
	hasKey bool // key slot is non-nil
	next   int  // index into nodes, -1 = none
}

// ljTab mirrors GCtab: array part (0-based slots for keys 0..asize-1) and a
// hash part with Brent-style collision handling.
type ljTab struct {
	array   []ljValue // asize = len(array)
	nodes   []ljNode  // hmask = len(nodes)-1; nil = no hash part
	freetop int       // index one past the highest free node scan position
}

func ljRol(x uint32, n uint) uint32 { return bits.RotateLeft32(x, int(n)) }

func ljFls(x uint32) uint32 { return uint32(31 - bits.LeadingZeros32(x)) }

// hashrot from lj_tab.h (x86/x64 variant).
func hashrot(lo, hi uint32) uint32 {
	lo ^= hi
	hi = ljRol(hi, 14)
	lo -= hi
	hi = ljRol(hi, 5)
	hi ^= lo
	hi -= ljRol(lo, 13)
	return hi
}

// hashnum: hashrot(u32.lo, u32.hi << 1) of the double's bits.
func (t *ljTab) hashnum(k float64) int {
	u := math.Float64bits(k)
	lo := uint32(u)
	hi := uint32(u>>32) << 1
	return int(hashrot(lo, hi) & uint32(len(t.nodes)-1))
}

// intKey reports whether k is an exact int32 (lj_num2int_check).
func intKey(k float64) (int32, bool) {
	i := int32(int64(k)) // lj_num2int truncates
	if float64(i) == k && !(i == 0 && math.Signbit(k)) {
		return i, true
	}
	return 0, false
}

// Set is lj_tab_set/lj_tab_setint for a non-nil value.
func (t *ljTab) Set(k float64, v any) {
	if i, ok := intKey(k); ok && uint32(i) < uint32(len(t.array)) {
		t.array[i] = ljValue{key: k, val: v, set: true}
		return
	}
	if t.nodes != nil {
		n := t.hashnum(k)
		for n >= 0 {
			if t.nodes[n].hasKey && t.nodes[n].key == k {
				t.nodes[n].val = v
				t.nodes[n].set = true
				return
			}
			n = t.nodes[n].next
		}
	}
	t.newKey(k, v)
}

// newKey is lj_tab_newkey.
func (t *ljTab) newKey(k float64, v any) {
	var n int
	if t.nodes != nil {
		n = t.hashnum(k)
	}
	if t.nodes == nil || t.nodes[n].set {
		// Find a free node from freetop downwards.
		freenode := t.freetop
		for {
			if freenode == 0 {
				t.rehash(k)
				t.Set(k, v)
				return
			}
			freenode--
			if !t.nodes[freenode].hasKey {
				break
			}
		}
		t.freetop = freenode
		collide := t.hashnum(t.nodes[n].key)
		if collide != n {
			// Colliding node is not in its main position: relocate it.
			for t.nodes[collide].next != n {
				collide = t.nodes[collide].next
			}
			t.nodes[collide].next = freenode
			t.nodes[freenode] = t.nodes[n]
			t.nodes[n].next = -1
			t.nodes[n].set = false
			// Rechain resurrected keys with colliding hashes.
			for t.nodes[freenode].next >= 0 {
				nn := t.nodes[freenode].next
				if t.nodes[nn].set && t.hashnum(t.nodes[nn].key) == n {
					t.nodes[freenode].next = t.nodes[nn].next
					t.nodes[nn].next = t.nodes[n].next
					t.nodes[n].next = nn
					for {
						nn2 := t.nodes[freenode].next
						if nn2 < 0 {
							break
						}
						if t.nodes[nn2].set {
							mn := t.hashnum(t.nodes[nn2].key)
							if mn != freenode && mn != nn2 {
								t.nodes[freenode].next = t.nodes[nn2].next
								t.nodes[nn2].next = t.nodes[mn].next
								t.nodes[mn].next = nn2
							} else {
								freenode = nn2
							}
						} else {
							freenode = nn2
						}
					}
					break
				}
				freenode = nn
			}
		} else {
			t.nodes[freenode].next = t.nodes[n].next
			t.nodes[n].next = freenode
			n = freenode
		}
	}
	t.nodes[n].key = k
	t.nodes[n].hasKey = true
	t.nodes[n].val = v
	t.nodes[n].set = true
}

// countint from lj_tab.c.
func countint(k float64, bins *[ljMaxABits]uint32) uint32 {
	if i, ok := intKey(k); ok && uint32(i) < ljMaxASize {
		b := 0
		if i > 2 {
			b = int(ljFls(uint32(i - 1)))
		}
		bins[b]++
		return 1
	}
	return 0
}

func (t *ljTab) rehash(ek float64) {
	var bins [ljMaxABits]uint32
	// countarray
	var asize uint32
	i := 0
	for b := 0; b < ljMaxABits; b++ {
		top := 2 << b
		if top >= len(t.array) {
			top = len(t.array) - 1
			if i > top {
				break
			}
		}
		var n uint32
		for ; i <= top; i++ {
			if t.array[i].set {
				n++
			}
		}
		bins[b] += n
		asize += n
	}
	total := 1 + asize
	// counthash
	for _, nd := range t.nodes {
		if nd.set {
			asize += countint(nd.key, &bins)
			total++
		}
	}
	asize += countint(ek, &bins)
	// bestasize
	na := func() uint32 {
		var na, sz uint32
		nn := asize
		var sum uint32
		for b := 0; 2*nn > (1<<b) && sum != nn; b++ {
			if bins[b] > 0 {
				sum += bins[b]
				if 2*sum > (1 << b) {
					sz = (2 << b) + 1
					na = sum
				}
			}
		}
		asize = sz
		return na
	}()
	total -= na
	// hsize2hbits
	hbits := 0
	if total > 0 {
		if total == 1 {
			hbits = 1
		} else {
			hbits = 1 + int(ljFls(total-1))
		}
	}
	t.resize(int(asize), hbits)
}

// resize is lj_tab_resize.
func (t *ljTab) resize(asize, hbits int) {
	oldNodes := t.nodes
	oldArray := t.array
	if asize > len(t.array) {
		newArr := make([]ljValue, asize)
		copy(newArr, t.array)
		t.array = newArr
	}
	if hbits > 0 {
		hsize := 1 << hbits
		t.nodes = make([]ljNode, hsize)
		for i := range t.nodes {
			t.nodes[i].next = -1
		}
		t.freetop = hsize
	} else {
		t.nodes = nil
		t.freetop = 0
	}
	if asize < len(oldArray) {
		t.array = t.array[:asize]
		for i := asize; i < len(oldArray); i++ {
			if oldArray[i].set {
				t.Set(float64(i), oldArray[i].val)
			}
		}
	}
	for i := range oldNodes {
		if oldNodes[i].set {
			t.Set(oldNodes[i].key, oldNodes[i].val)
		}
	}
}

// Pairs traverses like lj_tab_next from a nil key: array part in index
// order, then hash nodes in slot order.
func (t *ljTab) Pairs(yield func(k float64, v any)) {
	for i := range t.array {
		if t.array[i].set {
			yield(float64(i), t.array[i].val)
		}
	}
	for i := range t.nodes {
		if t.nodes[i].set {
			yield(t.nodes[i].key, t.nodes[i].val)
		}
	}
}

// NewLJTab returns an empty table (test hook).
func NewLJTab() *ljTab { return &ljTab{} }
