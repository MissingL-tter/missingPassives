// The generated game data, embedded: data/raw holds the exporter's output
// (cmd/pobexport, run explicitly whenever the GGPK is updated — the port of
// PoB's offline Export step and its committed Data/ tables). Embedding it
// makes a built binary self-contained: no GGPK, no repo checkout, no files
// beside the executable.
//
// Only the documents something reads are compiled in: Load's sources, the
// mod cache, the foulborn map, and the tree-side documents (tree, conquer
// tables, conquered passives, tattoos). statdesc.json, skillgemlist.json
// and mapuniquetofoulborn.json stay on disk as exporter outputs only.
package data

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed raw/miscdata.json raw/costs.json raw/bossdata.json
//go:embed raw/modscalability.json raw/essence.json raw/pantheons.json
//go:embed raw/crucible.json raw/masters.json raw/flavourtext.json
//go:embed raw/enchant.json raw/mods.json raw/cluster.json raw/bases.json
//go:embed raw/umodstotext.json raw/minions.json raw/skills.json
//go:embed raw/modcache.jsonl raw/modfoulbornmap.jsonc
//go:embed raw/tree_3_29.json raw/conquertables.json
//go:embed raw/conqueredpassives.json raw/tattoopassives.json
var rawFS embed.FS

// readFn resolves a raw file name (with extension) to its bytes.
type readFn func(name string) ([]byte, error)

func embeddedRead(name string) ([]byte, error) {
	return fs.ReadFile(rawFS, "raw/"+name)
}

func rawDocFrom(read readFn, name string, out any) error {
	b, err := read(name + ".json")
	if err != nil {
		return fmt.Errorf("data: raw document missing (run pobexport): %w", err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("data: raw/%s.json: %w", name, err)
	}
	return nil
}

func rawSourcesFrom(read readFn) (Sources, error) {
	var src Sources
	for _, doc := range []struct {
		name string
		out  any
	}{
		{"miscdata", &src.Misc}, {"costs", &src.Costs}, {"bossdata", &src.Boss},
		{"modscalability", &src.ModScalability}, {"essence", &src.Essences},
		{"pantheons", &src.Pantheons}, {"crucible", &src.Crucible},
		{"masters", &src.Masters}, {"flavourtext", &src.FlavourText},
		{"enchant", &src.Enchants}, {"mods", &src.Mods}, {"cluster", &src.Cluster},
		{"bases", &src.Bases}, {"umodstotext", &src.Uniques},
		{"minions", &src.MinionsDoc}, {"skills", &src.Skills},
	} {
		if err := rawDocFrom(read, doc.name, doc.out); err != nil {
			return Sources{}, err
		}
	}
	mc, err := read("modcache.jsonl")
	if err != nil {
		return Sources{}, fmt.Errorf("data: raw/modcache.jsonl missing (run cmd/sourceupdate -modcache-only): %w", err)
	}
	src.ModCacheJSONL = mc
	fb, err := read("modfoulbornmap.jsonc")
	if err != nil {
		return Sources{}, fmt.Errorf("data: raw/modfoulbornmap.jsonc missing (run pobexport): %w", err)
	}
	src.FoulbornMapJSONC = fb
	return src, nil
}

// RawSources builds Load's input from the embedded raw documents. The one
// field left empty is StatMapCopyFixture, which only the game-data
// differential supplies.
func RawSources() (Sources, error) { return rawSourcesFrom(embeddedRead) }

// RawSourcesFromDir builds Load's input from the same document set read
// from dir instead of the embedded copies — cmd/sourceupdate feeds the
// artifacts it has just written without a rebuild.
func RawSourcesFromDir(dir string) (Sources, error) {
	return rawSourcesFrom(func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dir, name))
	})
}

// RawDoc decodes one embedded document for consumers beyond Load (tree
// data, tattoos, conquer tables). name is the script name without
// extension. The documents are compiled in, so a missing or undecodable
// one is a build defect: it panics.
func RawDoc(name string, out any) {
	if err := rawDocFrom(embeddedRead, name, out); err != nil {
		panic(err)
	}
}
