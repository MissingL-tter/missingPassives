package test

// Tattoo artifact drift guard: the runtime tattoo pool reads
// data/raw/tattoopassives.json directly, so this renders the EMBEDDED
// document back to Lua-file form and byte-compares it against the
// archive's committed Data/TattooPassives.lua. If the archive updates,
// this fails instead of silently drifting. (The gated export differential
// proves the same for a freshly built document; this guards the embedded
// copy without needing the GGPK.)

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/internal/luarender"
)

func TestTattooArtifactMatchesArchive(t *testing.T) {
	refPath := filepath.Join("..", ".archive", "src", "Data", "TattooPassives.lua")
	want, err := os.ReadFile(refPath)
	if err != nil {
		t.Skipf("archive not present: %v", err)
	}
	var doc schema.TattooPassives
	data.RawDoc("tattoopassives", &doc)
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	files, err := luarender.Renderers["tattooPassives"](raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := files["Data/TattooPassives.lua"]
	if !ok {
		t.Fatalf("renderer produced %d files, none named Data/TattooPassives.lua", len(files))
	}
	if !bytes.Equal([]byte(got), want) {
		t.Fatalf("embedded tattoopassives.json no longer renders to the archive's TattooPassives.lua (%d vs %d bytes) — regenerate with pobexport", len(got), len(want))
	}
}
