// Scaffolding shared by the ported Export/Scripts/*.lua: the execution
// context (Main.lua's dat()/getFile() environment) and the Lua string/number
// formatting the builders bake into their schema documents.

package export

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

// Ctx is what a ported script runs against.
type Ctx struct {
	Dats   *DatSet
	SrcDir string // extracted GGPK root (holds Data/, Metadata/)

	txtCache         map[string]string
	otCache          map[string]string
	sd               *statDescState
	modItemExclusive map[string]*modEntry
	modFoulborn      map[string]*modEntry
	modsDoc          *schema.ModsData // cached mods document
	flavourEntries   []flavourEntry
}

// Script is one ported export script: Build produces its schema document
// (serialised as <name>.json). The Lua files the reference produced are
// reproduced only by test/luarender in the differential test.
type Script struct {
	Name string // the Lua script's basename, e.g. "costs"
	// OutName overrides the raw artifact's basename when the honest Go-side
	// name differs from the reference script's (see
	// .claude/documentation/lua-go-map.md); empty = Name.
	OutName string
	Build   func(*Ctx) (schema.Document, error)
}

// Scripts lists every ported script.
var Scripts []Script

// Dat is Main.lua's dat(); a table missing from the extract is an error.
func (x *Ctx) Dat(name string) (*DatFile, error) {
	return x.Dats.Dat(name)
}

// GetFile is Main.lua's getFile(): the raw bytes of a file in the extracted
// GGPK, cached, or "" when absent.
func (x *Ctx) GetFile(name string) string {
	name = strings.ToLower(name)
	if x.txtCache == nil {
		x.txtCache = map[string]string{}
	}
	if s, ok := x.txtCache[name]; ok {
		return s
	}
	b, err := os.ReadFile(filepath.Join(x.SrcDir, filepath.FromSlash(name)))
	if err != nil {
		return ""
	}
	x.txtCache[name] = string(b)
	return string(b)
}

// modEntry is one exported mod's data as written by mods.lua, kept in memory
// for the scripts that reload the generated files (uModsToText,
// mapUniqueToFoulborn).
type modEntry struct {
	lines  []string
	orders []float64
	tags   []string
}

// EnsureMods runs the mods build if it hasn't populated the caches yet
// (the Lua equivalent: dofile("Scripts/mods.lua") when table.containsId is
// missing).
func (x *Ctx) EnsureMods() error {
	if x.modItemExclusive != nil {
		return nil
	}
	_, err := buildMods(x)
	return err
}

// flavourEntry is one FlavourText.lua entry (id and name), kept for
// mapUniqueToFoulborn.
type flavourEntry struct {
	id, name string
}
