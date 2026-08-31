// treegen is the CLI over export.BuildTreeDoc: it converts GGG's published
// passive-tree JSON (github.com/grindinggear/skilltree-export, per-version
// tags) into the conventional-JSON tree document the tree package consumes
// (data/raw/tree_<v>.json), replacing the retired luajit dump of PoB's
// TreeData/<v>/tree.lua.
//
// export/treegen.go carries the authoritative description of what is and
// is not reproduced — in particular that PoB's Common.lua jsonToLua regex
// stage is deliberately NOT reproduced, because this port has no Lua load
// step and the mangling never applied to 3.29 data. Read that comment, not
// this one, before changing the conversion.
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
