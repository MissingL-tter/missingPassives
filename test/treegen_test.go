package test

// Tree artifact provenance guard: data/raw/tree_3_29.json must be exactly
// what export.BuildTreeDoc produces from GGG's published 3.29.1 tree JSON
// (github.com/grindinggear/skilltree-export, tag 3.29.1 — the gzipped copy
// in testdata). This replaces the retired luajit dump of PoB's tree.lua:
// the whole PoB ingestion pipeline (fix_ascendancy_positions.py +
// jsonToLua + canon) is reproduced in Go and pinned here byte-for-byte.

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/MissingL-tter/missingPassives/export"
)

func TestTreeArtifactMatchesGGGSource(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "ggg_tree_3_29_1.json.gz"))
	if err != nil {
		t.Skipf("GGG tree source not present: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	src, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	got, err := export.BuildTreeDoc(src)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "data", "raw", "tree_3_29.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("tree_3_29.json is not BuildTreeDoc(GGG 3.29.1) (%d vs %d bytes) — regenerate with cmd/treegen", len(got), len(want))
	}
}
