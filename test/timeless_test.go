package test

// The timeless-jewel differential: run the ported generation algorithm
// over every (node, seed) cell of every jewel type and compare against the
// shipped LUT bins in .archive/src/Data/TimelessJewelData/ — the bins are
// PoB's behavior contract; nothing from them ships in the binary. The
// compressed .zip(.partN) files are the committed source of truth (the
// .bin files beside them are PoB's own inflate cache).

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/tree"
)

func inflateJewelFile(t *testing.T, name string) []byte {
	t.Helper()
	dir := filepath.Join("..", ".archive", "src", "Data", "TimelessJewelData")
	var compressed []byte
	if b, err := os.ReadFile(filepath.Join(dir, name+".zip")); err == nil {
		compressed = b
	} else {
		for part := 0; ; part++ {
			b, err := os.ReadFile(filepath.Join(dir, name+".zip.part"+strconv.Itoa(part)))
			if err != nil {
				break
			}
			compressed = append(compressed, b...)
		}
	}
	if len(compressed) == 0 {
		t.Skipf("timeless bins not present (%s)", name)
	}
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("%s inflate: %v", name, err)
	}
	return out
}

// timelessNodeIndex decodes data.NodeIDList: graph id -> (index, size).
func timelessNodeIndex(t *testing.T) (nodes map[int64][2]int, sizeNotable int) {
	t.Helper()
	nodes = map[int64][2]int{}
	for k, v := range data.NodeIDList {
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			continue
		}
		nodes[id] = [2]int{int(entry["index"].(float64)), int(entry["size"].(float64))}
	}
	sizeNotable = int(data.NodeIDList["sizeNotable"].(float64))
	if len(nodes) != 1937 || sizeNotable != 454 {
		t.Fatalf("nodeIDList decode: %d nodes, sizeNotable %d", len(nodes), sizeNotable)
	}
	return nodes, sizeNotable
}

// timelessLocalToGlobal decodes NodeIDList.localIdToGlobalId for one jewel
// type (a Lua array: index 0 lives in the D's KV, 1.. in Arr, sparse high
// locals in KV).
func timelessLocalToGlobal(t *testing.T, jewelType int) func(int) int {
	t.Helper()
	arr, ok := data.NodeIDList["localIdToGlobalId"].([]any)
	if !ok || jewelType > len(arr) {
		t.Fatalf("localIdToGlobalId missing for type %d", jewelType)
	}
	d, ok := arr[jewelType-1].(*modparser.D)
	if !ok {
		t.Fatalf("localIdToGlobalId[%d]: unexpected shape", jewelType)
	}
	return func(local int) int {
		if local >= 1 && local <= len(d.Arr) {
			if g, ok := d.Arr[local-1].(float64); ok {
				return int(g)
			}
		}
		if g, ok := d.KV[strconv.Itoa(local)].(float64); ok {
			return int(g)
		}
		return local // reference: unmapped ids pass through
	}
}

func TestTimelessAgainstBins(t *testing.T) {
	loadData(t)
	nodes, sizeNotable := timelessNodeIndex(t)

	type jewelSpec struct {
		jt      int
		name    string
		seedMin int64 // effective seed space (Elegant Hubris already /20)
		seedMax int64
	}
	specs := []jewelSpec{
		{1, "GloriousVanity", 100, 8000},
		{2, "LethalPride", 10000, 18000},
		{3, "BrutalRestraint", 500, 8000},
		{4, "MilitantFaith", 2000, 10000},
		{5, "ElegantHubris", 100, 8000},
		{6, "HeroicTragedy", 100, 8000},
	}

	type nodeEntry struct {
		id    int64
		index int
		size  int
	}
	var ordered []nodeEntry
	for id, is := range nodes {
		ordered = append(ordered, nodeEntry{id, is[0], is[1]})
	}
	// index order = file order
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].index < ordered[j].index })

	cellsCompared := 0
	for _, spec := range specs {
		bin := inflateJewelFile(t, spec.name)
		l2g := timelessLocalToGlobal(t, spec.jt)
		seedSize := int(spec.seedMax - spec.seedMin + 1)

		workNodes := ordered
		if spec.jt != 1 {
			workNodes = nil
			for _, n := range ordered {
				if n.index <= sizeNotable-1 {
					workNodes = append(workNodes, n)
				}
			}
			if want := len(workNodes) * seedSize; want != len(bin) {
				t.Fatalf("%s: file size %d, expected %d", spec.name, len(bin), want)
			}
		}

		// Glorious Vanity: sizes table then concatenated variable records.
		var gvStart map[int]int
		if spec.jt == 1 {
			gvStart = map[int]int{}
			off := len(ordered) * seedSize
			for _, n := range workNodes {
				gvStart[n.index] = off
				off += n.size
			}
			if off != len(bin) {
				t.Fatalf("%s: record walk ends at %d, file size %d", spec.name, off, len(bin))
			}
		}

		var mu sync.Mutex
		mismatches := 0
		cells := 0
		report := func(nodeID int64, seed int64, got, want []int) {
			mu.Lock()
			defer mu.Unlock()
			mismatches++
			if mismatches <= 10 {
				t.Errorf("%s node %d seed %d: got %v want %v", spec.name, nodeID, seed, got, want)
			}
		}

		sem := make(chan struct{}, runtime.NumCPU())
		var wg sync.WaitGroup
		for _, n := range workNodes {
			wg.Add(1)
			sem <- struct{}{}
			go func(n nodeEntry) {
				defer func() { <-sem; wg.Done() }()
				local := 0
				for s := 0; s < seedSize; s++ {
					seed := spec.seedMin + int64(s)
					callSeed := seed
					if spec.jt == 5 {
						callSeed = seed * 20 // the caller passes the raw jewel seed
					}
					got := tree.TimelessPassive(callSeed, n.id, spec.jt)
					var want []int
					if spec.jt == 1 {
						recLen := int(bin[n.index*seedSize+s])
						rec := bin[gvStart[n.index]+local : gvStart[n.index]+local+recLen]
						local += recLen
						want = make([]int, recLen)
						for i, b := range rec {
							want[i] = int(b)
						}
						// readLUT's conversion positions
						if recLen == 2 || recLen == 3 {
							want[0] = l2g(want[0])
						} else if recLen == 6 || recLen == 8 {
							for i := 0; i < recLen/2; i++ {
								want[i] = l2g(want[i])
							}
						}
					} else {
						want = []int{l2g(int(bin[n.index*seedSize+s]))}
					}
					if !intSliceEq(got, want) {
						report(n.id, seed, got, want)
					}
				}
				mu.Lock()
				cells += seedSize
				mu.Unlock()
			}(n)
		}
		wg.Wait()
		if mismatches > 0 {
			t.Fatalf("%s: %d/%d cells diverged", spec.name, mismatches, cells)
		}
		cellsCompared += cells
	}
	t.Logf("timeless differential: %d cells byte-identical across %d jewel types", cellsCompared, len(specs))
}

func intSliceEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
