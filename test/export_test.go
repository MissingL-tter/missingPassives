package test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/export"
	"github.com/MissingL-tter/missingPassives/internal/luarender"
)

// tplFS serves the hand-maintained template files to luarender from the
// archive tree.
type tplFS struct{ root string }

func (t tplFS) Read(rel string) (string, error) {
	b, err := os.ReadFile(filepath.Join(t.root, filepath.FromSlash(rel)))
	return string(b), err
}

// The export differential test: builds every script's schema document over
// the extracted GGPK, round-trips it through JSON, renders it back to Lua
// with internal/luarender, and byte-compares each rendered file against the
// checked-in copy under .archive/src (which the archive Lua exporter produced
// from the same game version). Fails on any disagreement.
//
// Requires the extracted GGPK at .archive/src/Export/ggpk (see that
// directory's README for bun_extract_file usage); skips when absent.
func TestExportAgainstReference(t *testing.T) {
	// The export differential re-verifies the generator against the GGPK —
	// ~97s, and only meaningful when the exporter or the game files change.
	// Everything else runs from the committed data/raw, so this stays off
	// unless asked for: MP_EXPORT=1 go test ./test/ -run TestExportAgainstReference
	if os.Getenv("MP_EXPORT") == "" {
		t.Skip("export differential skipped by default; set MP_EXPORT=1 to run")
	}
	srcDir := filepath.Join("..", ".archive", "src", "Export", "ggpk")
	refDir := filepath.Join("..", ".archive", "src")
	if _, err := os.Stat(filepath.Join(srcDir, "Data", "stats.datc64")); err != nil {
		t.Skipf("extracted GGPK not present at %s: %v", srcDir, err)
	}

	if err := export.WriteEnumFiles(filepath.Join(srcDir, "Data")); err != nil {
		t.Fatalf("writing enum tables: %v", err)
	}
	dats, err := export.LoadDats(filepath.Join(srcDir, "Data"))
	if err != nil {
		t.Fatalf("loading dats: %v", err)
	}
	ctx := &export.Ctx{Dats: dats, SrcDir: srcDir, TplDir: refDir}

	var checked, disagree int
	rawByName := map[string]json.RawMessage{}
	for _, s := range export.Scripts {
		data, err := s.Build(ctx)
		if err != nil {
			t.Fatalf("%s: build: %v", s.Name, err)
		}
		raw, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("%s: marshal: %v", s.Name, err)
		}
		rawByName[s.Name] = raw
		render, ok := luarender.Renderers[s.Name]
		if !ok {
			t.Fatalf("%s: no renderer registered", s.Name)
		}
		files, err := render(raw, tplFS{refDir})
		if err != nil {
			t.Fatalf("%s: render: %v", s.Name, err)
		}
		rels := make([]string, 0, len(files))
		for rel := range files {
			rels = append(rels, rel)
		}
		sort.Strings(rels)
		for _, rel := range rels {
			checked++
			want, err := os.ReadFile(filepath.Join(refDir, filepath.FromSlash(rel)))
			if err != nil {
				disagree++
				t.Errorf("%s: reference %s unreadable: %v", s.Name, rel, err)
				continue
			}
			if !bytes.Equal([]byte(files[rel]), want) {
				disagree++
				t.Errorf("%s: %s differs from the archive (%d vs %d bytes)", s.Name, rel, len(files[rel]), len(want))
			}
		}
	}
	t.Logf("export vs archive: %d files checked, %d disagreements", checked, disagree)
	if checked != 123 {
		t.Errorf("expected 123 files, checked %d", checked)
	}

	// conquertables.json has no reference Lua file to render against — its
	// contract is the committed artifact (whose content the historic
	// differentials prove against the shipped LUT bins).
	ctDoc, err := export.BuildConquerTables(ctx, data.NodeGraphIDs())
	if err != nil {
		t.Fatalf("conquerTables: %v", err)
	}
	wantCT, err := os.ReadFile(filepath.Join("..", "data", "raw", "conquertables.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ctDoc, wantCT) {
		t.Errorf("conquertables.json does not match BuildConquerTables over the GGPK (%d vs %d bytes)", len(ctDoc), len(wantCT))
	}
	if disagree > 0 {
		t.Fatalf("%d disagreements with the archive", disagree)
	}

	// Negative control: corrupt one field of a built document, re-render and
	// require the archive comparison to notice. Guards against the test
	// degenerating into one that cannot fail.
	corruptions := []struct {
		script string
		mutate func(t *testing.T, raw json.RawMessage) any
	}{
		{"costs", func(t *testing.T, raw json.RawMessage) any {
			var d schema.Costs
			mustUnmarshal(t, raw, &d)
			d[0].Resource += "X"
			return d
		}},
		{"mods", func(t *testing.T, raw json.RawMessage) any {
			var d schema.ModsData
			mustUnmarshal(t, raw, &d)
			d.Pools["ModExplicit"][0].Lines[0] += "!"
			return d
		}},
		{"skills", func(t *testing.T, raw json.RawMessage) any {
			var d schema.SkillsData
			mustUnmarshal(t, raw, &d)
			f := d.Files["act_str"]
			f.Skills[0].Name += "X"
			d.Files["act_str"] = f
			return d
		}},
		{"uModsToText", func(t *testing.T, raw json.RawMessage) any {
			var d schema.Uniques
			mustUnmarshal(t, raw, &d)
			f := d["axe"]
			f.Sections[0].Items[0][0] += "!"
			d["axe"] = f
			return d
		}},
	}
	for _, c := range corruptions {
		t.Run("corruption detected in "+c.script, func(t *testing.T) {
			raw, err := json.Marshal(c.mutate(t, rawByName[c.script]))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			files, err := luarender.Renderers[c.script](raw, tplFS{refDir})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			differs := 0
			for rel, got := range files {
				want, err := os.ReadFile(filepath.Join(refDir, filepath.FromSlash(rel)))
				if err != nil {
					t.Fatalf("reference %s unreadable: %v", rel, err)
				}
				if !bytes.Equal([]byte(got), want) {
					differs++
				}
			}
			if differs == 0 {
				t.Fatalf("corrupted %s document still byte-matches the archive — the comparison cannot fail", c.script)
			}
		})
	}
}

func mustUnmarshal(t *testing.T, raw json.RawMessage, v any) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
