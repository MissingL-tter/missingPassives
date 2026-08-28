package test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MissingL-tter/missingPassives/internal/modcachegen"
)

// The mod cache regeneration guard: the Go generator (fresh parses over the
// tree + item DB walk) must reproduce the committed artifact byte-for-byte.
func TestModCacheGeneration(t *testing.T) {
	loadData(t)
	got := modcachegen.Build("3_29")
	want, err := os.ReadFile(filepath.Join("..", "data", "raw", "modcache.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		i := 0
		for i < len(got) && i < len(want) && got[i] == want[i] {
			i++
		}
		lo := max(0, i-120)
		t.Fatalf("regenerated modcache.jsonl diverges from the committed artifact at byte %d (%d vs %d bytes)\n got ...%q\nwant ...%q",
			i, len(got), len(want), string(got[lo:min(len(got), i+120)]), string(want[lo:min(len(want), i+120)]))
	}
}
