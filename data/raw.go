// The generated game data, embedded: data/raw holds the exporter's output
// (cmd/pobexport, run explicitly whenever the GGPK is updated — the port of
// PoB's offline Export step and its committed Data/ tables). Embedding it
// makes a built binary self-contained: no GGPK, no repo checkout, no files
// beside the executable.
package data

import (
	"embed"
	"encoding/json"
)

//go:embed raw
var rawFS embed.FS

func rawDoc(name string, out any) {
	b, err := rawFS.ReadFile("raw/" + name + ".json")
	if err != nil {
		panic("data: raw document missing (run pobexport): " + err.Error())
	}
	if err := json.Unmarshal(b, out); err != nil {
		panic("data: raw/" + name + ".json: " + err.Error())
	}
}

// RawSources builds Load's input from the embedded raw documents. The one
// field left empty is StatMapCopies, the archive-dump replay fixture the
// caller supplies.
func RawSources() Sources {
	var src Sources
	rawDoc("miscdata", &src.Misc)
	rawDoc("costs", &src.Costs)
	rawDoc("bossData", &src.Boss)
	rawDoc("modScalability", &src.ModScalability)
	rawDoc("essence", &src.Essences)
	rawDoc("pantheons", &src.Pantheons)
	rawDoc("crucible", &src.Crucible)
	rawDoc("masters", &src.Masters)
	rawDoc("flavourText", &src.FlavourText)
	rawDoc("enchant", &src.Enchants)
	rawDoc("mods", &src.Mods)
	rawDoc("cluster", &src.Cluster)
	rawDoc("bases", &src.Bases)
	rawDoc("uModsToText", &src.Uniques)
	rawDoc("minions", &src.MinionsDoc)
	rawDoc("skills", &src.Skills)
	fb, err := rawFS.ReadFile("raw/ModFoulbornMap.jsonc")
	if err != nil {
		panic("data: raw/ModFoulbornMap.jsonc missing (run pobexport): " + err.Error())
	}
	src.FoulbornMapJSONC = fb
	return src
}

// RawDoc exposes one embedded document for consumers beyond Load (future
// modules: statdesc, tree data). name is the script name without extension.
func RawDoc(name string, out any) {
	rawDoc(name, out)
}
