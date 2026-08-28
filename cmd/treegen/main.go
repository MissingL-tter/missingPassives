// treegen converts GGG's published passive-tree JSON
// (github.com/grindinggear/skilltree-export, per-version tags) into the
// canon tree document the tree package consumes (data/raw/tree_<v>.json),
// replacing the retired luajit dump of PoB's TreeData/<v>/tree.lua.
//
// It reproduces PoB's whole ingestion pipeline byte-for-byte:
//
//  1. fix_ascendancy_positions.py (upstream repo root, 3.29.1 state):
//     GGG keyword-tag stripping on node stats/reminderText, ascendancy
//     group repositioning to fixed board slots, the extra legacy notables,
//     and dropping extraImages/sprites/imageZoomLevels (sprites are view
//     data; the sprites.json split is not emitted here).
//  2. Common.lua jsonToLua + Lua load: a REGEX pipeline over the fixed
//     JSON text. Structurally that means numeric-looking object keys
//     become Lua number keys and arrays become 1-based tables; textually
//     it swaps every '['/']' for '{'/'}' INSIDE string values too, and
//     its `{(%w+)}` -> `{[0]=%1}` quirk can only fire inside strings
//     (the python serializer's indent puts real single-element arrays on
//     multiple lines, so the pattern never matches structure).
//  3. conventional JSON out (the Lua-table shape lives only in tests).
//
// The canon-format equivalence to the retired luajit dump was proven when
// the port landed; test/treegen_test.go pins the current output to the
// committed artifact and the GGG source.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MissingL-tter/missingPassives/export"
)

func main() {
	in := flag.String("in", "", "GGG data.json (grindinggear/skilltree-export, matching version tag)")
	out := flag.String("out", "", "output path (default data/raw/tree_<version>.json)")
	version := flag.String("version", "3_29", "tree version")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: treegen -in data.json [-version 3_29] [-out path]")
		os.Exit(2)
	}
	if *out == "" {
		*out = "data/raw/tree_" + *version + ".json"
	}
	blob, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	doc, err := export.BuildTreeDoc(blob)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, doc, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("tree %s -> %s\n", *version, *out)
}
