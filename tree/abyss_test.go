package tree

// The abyss differential: recompute EVERY record of the five shipped
// Abyss*.zip LUTs (socket walks, per-node modifications, Zorath's node
// blocks and ascendancy selections) and byte-compare against the inflated
// bins in .archive/src/Data/TimelessJewelData/. Skips when the archive is
// not present.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
)

func inflateAbyssBin(t *testing.T, name string) []byte {
	t.Helper()
	dir := filepath.Join("..", ".archive", "src", "Data", "TimelessJewelData")
	var compressed []byte
	for part := 0; ; part++ {
		b, err := os.ReadFile(filepath.Join(dir, name+".zip.part"+strconv.Itoa(part)))
		if err != nil {
			break
		}
		compressed = append(compressed, b...)
	}
	if len(compressed) == 0 {
		t.Skipf("abyss bins not present (%s)", name)
	}
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// abyssGlobalToLocal rebuilds the generator's per-type local id mapping
// (identity for ids that fit a byte, overflow ids packed into free slots
// from 0 upward, both in ascending global order).
func abyssGlobalToLocal(jewelType int) map[int]byte {
	tt := conquerData()
	var globals []int
	for _, a := range tt.Additions {
		if a.TV == jewelType {
			globals = append(globals, a.global)
		}
	}
	for _, s := range tt.Skills {
		if s.TV == jewelType {
			globals = append(globals, s.global)
		}
	}
	sort.Ints(globals)
	g2l := map[int]byte{}
	used := map[byte]bool{}
	var overflow []int
	for _, g := range globals {
		if g <= 255 && !used[byte(g)] {
			g2l[g] = byte(g)
			used[byte(g)] = true
			continue
		}
		overflow = append(overflow, g)
	}
	next := byte(0)
	for _, g := range overflow {
		for used[next] {
			next++
		}
		g2l[g] = next
		used[next] = true
		next++
	}
	return g2l
}

func encodeAbyssModification(buf *bytes.Buffer, comps []AbyssComponent, g2l map[int]byte) {
	buf.WriteByte(byte(len(comps)))
	for _, c := range comps {
		buf.WriteByte(byte(c.Type))
		buf.WriteByte(g2l[c.ID])
		buf.WriteByte(byte(len(c.Rolls)))
		for _, r := range c.Rolls {
			var tmp [2]byte
			binary.LittleEndian.PutUint16(tmp[:], uint16(r))
			buf.Write(tmp[:])
		}
	}
}

func abyssHeader(t *testing.T, data []byte, wantMagic string, jewelType int) (seedMin, seedMax, seedInc, off int) {
	t.Helper()
	if string(data[:4]) != wantMagic || data[4] != 1 || int(data[5]) != jewelType {
		t.Fatalf("bad header: %q v%d type %d", data[:4], data[4], data[5])
	}
	seedMin = int(binary.LittleEndian.Uint16(data[6:]))
	seedMax = int(binary.LittleEndian.Uint16(data[8:]))
	seedInc = int(binary.LittleEndian.Uint16(data[10:]))
	return seedMin, seedMax, seedInc, 12
}

func skipAbyssModification(data []byte, o int) int {
	cc := int(data[o])
	o++
	for c := 0; c < cc; c++ {
		sc := int(data[o+2])
		o += 3 + sc*2
	}
	return o
}

func TestAbyssSocketBinsAgainstAlgorithm(t *testing.T) {
	w := abyssData()
	tt := conquerData()
	names := map[int]string{7: "AbyssTecrod", 8: "AbyssUlaman", 9: "AbyssKurgal", 10: "AbyssAmanamu"}
	recordsCompared := 0
	for jt := 7; jt <= 10; jt++ {
		data := inflateAbyssBin(t, names[jt])
		seedMin, seedMax, seedInc, off := abyssHeader(t, data, "ABYS", jt)
		seedCount := (seedMax-seedMin)/seedInc + 1
		socketCount := int(data[off])
		if int(data[off+1]) != abyssDefaultSize {
			t.Fatalf("%s: abyssSize %d", names[jt], data[off+1])
		}
		off += 2
		socketIDs := make([]int64, socketCount)
		for i := 0; i < socketCount; i++ {
			socketIDs[i] = int64(binary.LittleEndian.Uint16(data[off+2*i:]))
		}
		off += 2 * socketCount
		if len(socketIDs) != len(w.sockets) {
			t.Fatalf("%s: %d sockets vs %d expected", names[jt], len(socketIDs), len(w.sockets))
		}
		for i, s := range w.sockets {
			if socketIDs[i] != s.gid {
				t.Fatalf("%s: socket[%d] = %d, expected %d", names[jt], i, socketIDs[i], s.gid)
			}
		}
		g2l := abyssGlobalToLocal(jt)

		// Slice per-socket blocks by walking the records once.
		blockStart := make([]int, socketCount+1)
		o := off
		for si := 0; si < socketCount; si++ {
			blockStart[si] = o
			for s := 0; s < seedCount; s++ {
				n := int(data[o])
				o++
				for k := 0; k < n; k++ {
					o = skipAbyssModification(data, o+2)
				}
			}
		}
		blockStart[socketCount] = o
		if o != len(data) {
			t.Fatalf("%s: record walk ends at %d, file size %d", names[jt], o, len(data))
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, runtime.NumCPU())
		var mu sync.Mutex
		var failed []string
		for si := 0; si < socketCount; si++ {
			wg.Add(1)
			sem <- struct{}{}
			go func(si int) {
				defer func() { <-sem; wg.Done() }()
				var buf bytes.Buffer
				for s := 0; s < seedCount; s++ {
					seed := uint32(seedMin + s*seedInc)
					affected := w.walk(socketIDs[si], seed, abyssDefaultSize)
					var kept []*abyssNode
					for _, n := range affected {
						if abyssCanModify(tt, n, jt) {
							kept = append(kept, n)
						}
					}
					buf.WriteByte(byte(len(kept)))
					for _, n := range kept {
						var tmp [2]byte
						binary.LittleEndian.PutUint16(tmp[:], uint16(n.gid))
						buf.Write(tmp[:])
						encodeAbyssModification(&buf, abyssModification(tt, n, jt, seed), g2l)
					}
				}
				if !bytes.Equal(buf.Bytes(), data[blockStart[si]:blockStart[si+1]]) {
					mu.Lock()
					failed = append(failed, names[jt]+" socket "+strconv.FormatInt(socketIDs[si], 10))
					mu.Unlock()
				}
			}(si)
		}
		wg.Wait()
		if len(failed) > 0 {
			t.Fatalf("diverged blocks: %v", failed)
		}
		recordsCompared += socketCount * seedCount
	}
	t.Logf("abyss socket differential: %d records byte-identical across 4 jewel types", recordsCompared)
}

func TestAbyssZorathBinAgainstAlgorithm(t *testing.T) {
	w := abyssData()
	tt := conquerData()
	data := inflateAbyssBin(t, "AbyssZorath")
	seedMin, seedMax, seedInc, off := abyssHeader(t, data, "ABYN", 11)
	seedCount := (seedMax-seedMin)/seedInc + 1
	nodeCount := int(binary.LittleEndian.Uint16(data[off:]))
	off += 2
	nodeIDs := make([]int64, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodeIDs[i] = int64(binary.LittleEndian.Uint16(data[off+2*i:]))
	}
	off += 2 * nodeCount

	var wantNodes []int64
	for gid := range w.zorathNodes {
		wantNodes = append(wantNodes, gid)
	}
	sort.Slice(wantNodes, func(i, j int) bool { return wantNodes[i] < wantNodes[j] })
	if len(wantNodes) != nodeCount {
		t.Fatalf("node list: %d vs %d expected", nodeCount, len(wantNodes))
	}
	for i := range wantNodes {
		if nodeIDs[i] != wantNodes[i] {
			t.Fatalf("node list[%d] = %d, expected %d", i, nodeIDs[i], wantNodes[i])
		}
	}
	g2l := abyssGlobalToLocal(11)

	// Node blocks.
	blockStart := make([]int, nodeCount+1)
	o := off
	for ni := 0; ni < nodeCount; ni++ {
		blockStart[ni] = o
		for s := 0; s < seedCount; s++ {
			o = skipAbyssModification(data, o)
		}
	}
	blockStart[nodeCount] = o

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	var mu sync.Mutex
	var failed []string
	for ni := 0; ni < nodeCount; ni++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(ni int) {
			defer func() { <-sem; wg.Done() }()
			var buf bytes.Buffer
			node := w.nodes[nodeIDs[ni]]
			for s := 0; s < seedCount; s++ {
				encodeAbyssModification(&buf, abyssModification(tt, node, 11, uint32(seedMin+s*seedInc)), g2l)
			}
			if !bytes.Equal(buf.Bytes(), data[blockStart[ni]:blockStart[ni+1]]) {
				mu.Lock()
				failed = append(failed, strconv.FormatInt(nodeIDs[ni], 10))
				mu.Unlock()
			}
		}(ni)
	}
	wg.Wait()
	if len(failed) > 0 {
		t.Fatalf("diverged node blocks: %v", failed[:min(len(failed), 10)])
	}

	// Ascendancy sections.
	if string(data[o:o+4]) != "ASCS" {
		t.Fatalf("missing ASCS at %d", o)
	}
	o += 4
	ascCount := int(binary.LittleEndian.Uint16(data[o:]))
	o += 2
	if ascCount != len(w.ascOrder) {
		t.Fatalf("ascendancies: %d vs %d expected", ascCount, len(w.ascOrder))
	}
	selectionsCompared := 0
	for a := 0; a < ascCount; a++ {
		nameLen := int(data[o])
		name := string(data[o+1 : o+1+nameLen])
		o += 1 + nameLen
		if name != w.ascOrder[a] {
			t.Fatalf("ascendancy[%d] = %q, expected %q", a, name, w.ascOrder[a])
		}
		for s := 0; s < seedCount; s++ {
			cnt := int(data[o])
			o++
			var want []int64
			for k := 0; k < cnt; k++ {
				want = append(want, int64(binary.LittleEndian.Uint16(data[o:])))
				o += 2
			}
			got := w.selectAscendancyNotables(name, uint32(seedMin+s*seedInc), 1)
			ok := len(got) == cnt
			if ok {
				for k := range got {
					if got[k].gid != want[k] {
						ok = false
						break
					}
				}
			}
			if !ok {
				t.Fatalf("%s seed %d: selection diverged", name, seedMin+s*seedInc)
			}
			selectionsCompared++
		}
	}
	if o != len(data) {
		t.Fatalf("file walk ends at %d, size %d", o, len(data))
	}
	t.Logf("zorath differential: %d node blocks and %d ascendancy selections byte-identical",
		nodeCount, selectionsCompared)
}
