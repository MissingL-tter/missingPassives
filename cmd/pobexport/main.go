// pobexport regenerates the game data from an extracted GGPK directory as
// structured JSON documents (one per script), standing in for the Lua Export
// application. The reference's Data/*.lua text is reproduced only inside the
// differential test (test/luarender).
//
// Usage: pobexport -src <extracted-ggpk-root> -out <data-dir> [script ...]
// With no script names, every ported script runs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/export"
)

func main() {
	src := flag.String("src", "", "extracted GGPK root (holds Data/ and Metadata/)")
	out := flag.String("out", "data/raw", "output directory for the generated <script>.json documents")
	flag.Parse()
	if *src == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := export.WriteEnumFiles(filepath.Join(*src, "Data")); err != nil {
		fmt.Fprintln(os.Stderr, "writing enum tables:", err)
		os.Exit(1)
	}
	dats, err := export.LoadDats(filepath.Join(*src, "Data"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading dats:", err)
		os.Exit(1)
	}
	ctx := &export.Ctx{Dats: dats, SrcDir: *src}

	want := map[string]bool{}
	for _, name := range flag.Args() {
		want[name] = true
	}
	written, err := export.WriteArtifacts(ctx, *out, want, data.NodeGraphIDs())
	for _, name := range written {
		fmt.Println("->", name)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
