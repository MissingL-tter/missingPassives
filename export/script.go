// Scaffolding shared by the ported Export/Scripts/*.lua: the execution
// context (Main.lua's dat()/getFile() environment) and the Lua string/number
// formatting the builders bake into their schema documents.

package export

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Ctx is what a ported script runs against.
type Ctx struct {
	Dats   *DatSet
	SrcDir string // extracted GGPK root (holds Data/, Metadata/)
	TplDir string // the real src/ tree, for reading hand-maintained templates

	txtCache         map[string]string
	otCache          map[string]string
	sd               *statDescState
	modItemExclusive map[string]*modEntry
	modFoulborn      map[string]*modEntry
	modsDoc          any // cached mods schema document
	flavourEntries   []flavourEntry
}

// Script is one ported export script: Build produces its schema document
// (serialised as <name>.json). The Lua files the reference produced are
// reproduced only by internal/luarender in the differential test.
type Script struct {
	Name string // the Lua script's basename, e.g. "costs"
	// OutName overrides the raw artifact's basename when the honest Go-side
	// name differs from the reference script's (see
	// .claude/documentation/lua-go-map.md); empty = Name.
	OutName string
	Build   func(*Ctx) (any, error)
}

// Scripts lists every ported script.
var Scripts []Script

// Dat is Main.lua's dat().
func (x *Ctx) Dat(name string) *DatFile {
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

// luaNum formats a float64 the way LuaJIT's tostring/write does (%.14g, with
// inf/nan spelled Lua-style).
func luaNum(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if math.IsNaN(f) {
		return "nan"
	}
	s := strconv.FormatFloat(f, 'g', 14, 64)
	// Go writes exponents as 1e+05 like C's %g; keep that. Trim Go's lack of
	// difference aside, the formats agree for %.14g.
	return s
}

// luaStr mirrors tostring() for the value kinds scripts hit: nil, booleans,
// numbers and strings.
func luaStr(v any) string {
	switch t := v.(type) {
	case nil:
		return "nil"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return luaNum(t)
	}
	panic(fmt.Sprintf("luaStr: unsupported type %T", v))
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

var reDirective = regexp.MustCompile(`#([A-Za-z]+) ?(.*)`)
