// pobexport regenerates the game-data tables (Data/*.lua) from an extracted
// GGPK directory, standing in for the Lua Export application.
//
// Usage: pobexport -src <extracted-ggpk-root> -out <data-dir> [script ...]
// With no script names, every ported script runs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MissingL-tter/missingPassives/export"
)

func main() {
	src := flag.String("src", "", "extracted GGPK root (holds Data/ and Metadata/)")
	out := flag.String("out", "", "output directory for the generated Data/*.lua")
	tpl := flag.String("tpl", ".archive/src", "source tree holding hand-maintained templates (Export/Uniques, Export/Skills)")
	flag.Parse()
	if *src == "" || *out == "" {
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
	ctx := &export.Ctx{Dats: dats, SrcDir: *src, OutDir: *out, TplDir: *tpl}

	want := map[string]bool{}
	for _, name := range flag.Args() {
		want[name] = true
	}
	for _, s := range export.Scripts {
		if len(want) > 0 && !want[s.Name] {
			continue
		}
		if err := s.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", s.Name, err)
			os.Exit(1)
		}
		fmt.Println(s.Name, "->", s.Outs)
	}
}
