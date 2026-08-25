// pobexport regenerates the game data from an extracted GGPK directory as
// structured JSON documents (one per script), standing in for the Lua Export
// application. The reference's Data/*.lua text is reproduced only inside the
// differential test (internal/luarender).
//
// Usage: pobexport -src <extracted-ggpk-root> -out <data-dir> [script ...]
// With no script names, every ported script runs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MissingL-tter/missingPassives/export"
)

func main() {
	src := flag.String("src", "", "extracted GGPK root (holds Data/ and Metadata/)")
	out := flag.String("out", "data/raw", "output directory for the generated <script>.json documents")
	tpl := flag.String("tpl", ".archive/src", "source tree holding hand-maintained templates (Export/Uniques, Export/Skills)")
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
	ctx := &export.Ctx{Dats: dats, SrcDir: *src, TplDir: *tpl}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	want := map[string]bool{}
	for _, name := range flag.Args() {
		want[name] = true
	}
	for _, s := range export.Scripts {
		if len(want) > 0 && !want[s.Name] {
			continue
		}
		data, err := s.Build(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", s.Name, err)
			os.Exit(1)
		}
		raw, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", s.Name, err)
			os.Exit(1)
		}
		dest := filepath.Join(*out, s.Name+".json")
		if err := os.WriteFile(dest, append(raw, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", s.Name, err)
			os.Exit(1)
		}
		fmt.Println(s.Name, "->", s.Name+".json")
	}

	// The hand-maintained foulborn map rides along so raw/ is the complete
	// data artifact.
	if len(want) == 0 {
		fb, err := os.ReadFile(filepath.Join(*tpl, "Data", "ModFoulbornMap.jsonc"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "ModFoulbornMap:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(*out, "ModFoulbornMap.jsonc"), fb, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "ModFoulbornMap:", err)
			os.Exit(1)
		}
		fmt.Println("ModFoulbornMap.jsonc copied")
	}
}
