// sourceupdate is the one-shot league/source update: it regenerates every
// generatable data/raw artifact and verifies the result, so no step can be
// forgotten.
//
//  1. All export artifacts from the extracted GGPK (export.WriteArtifacts —
//     every script document, conquertables.json, the foulborn map).
//  2. tree_<version>.json from GGG's published tree JSON
//     (github.com/grindinggear/skilltree-export, fetched by release tag, or
//     -treejson for a local copy).
//  3. modcache.jsonl from the Go parser over the regenerated data
//     (internal/modcachegen), run as a subprocess so the fresh artifacts
//     are the ones compiled in (data/raw is embedded).
//  4. Reports what it cannot regenerate: the .archive reference itself
//     (shipped LUT bins etc. — replace manually on a version bump; every
//     differential compares against it).
//  5. Runs the full test suite plus the MP_EXPORT export differential
//     (-skiptests to skip). After a data change, failures here are the
//     drift report: fix or update the reference side deliberately.
//
// Usage: go run ./cmd/sourceupdate [-treetag 3.29.1]
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/export"
	"github.com/MissingL-tter/missingPassives/internal/modcachegen"
)

func fail(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

// treeVersionFromTag derives tree_<version>.json's version from the GGG
// release tag's major.minor: 3.29.1 -> 3_29.
func treeVersionFromTag(tag string) string {
	m := regexp.MustCompile(`^(\d+)\.(\d+)`).FindStringSubmatch(tag)
	if m == nil {
		fail("cannot derive tree version from tag", tag)
	}
	return m[1] + "_" + m[2]
}

func main() {
	src := flag.String("src", filepath.Join(".archive", "src", "Export", "ggpk"), "extracted GGPK root (holds Data/ and Metadata/)")
	out := flag.String("out", filepath.Join("data", "raw"), "artifact output directory")
	treeTag := flag.String("treetag", "3.29.1", "grindinggear/skilltree-export release tag to fetch data.json from")
	treeJSON := flag.String("treejson", "", "local GGG tree data.json (skips the fetch)")
	skipTests := flag.Bool("skiptests", false, "skip the verification test runs")
	modcacheOnly := flag.Bool("modcache-only", false, "regenerate modcache.jsonl only (the main run invokes this as a subprocess so the freshly written artifacts are the ones compiled in)")
	flag.Parse()

	if *modcacheOnly {
		data.Load(data.RawSources())
		out := filepath.Join(*out, "modcache.jsonl")
		if err := os.WriteFile(out, modcachegen.Build(treeVersionFromTag(*treeTag)), 0o644); err != nil {
			fail(err)
		}
		fmt.Println("->", filepath.Base(out))
		return
	}

	treeVersion := treeVersionFromTag(*treeTag)

	// 1. GGPK export artifacts.
	if _, err := os.Stat(filepath.Join(*src, "Data", "stats.datc64")); err != nil {
		fail("extracted GGPK not present:", err)
	}
	if err := export.WriteEnumFiles(filepath.Join(*src, "Data")); err != nil {
		fail("writing enum tables:", err)
	}
	dats, err := export.LoadDats(filepath.Join(*src, "Data"))
	if err != nil {
		fail("loading dats:", err)
	}
	ctx := &export.Ctx{Dats: dats, SrcDir: *src}
	written, err := export.WriteArtifacts(ctx, *out, nil, data.NodeGraphIDs())
	for _, name := range written {
		fmt.Println("->", name)
	}
	if err != nil {
		fail(err)
	}

	// 2. The tree document from GGG's published JSON.
	var blob []byte
	if *treeJSON != "" {
		if blob, err = os.ReadFile(*treeJSON); err != nil {
			fail(err)
		}
	} else {
		url := "https://raw.githubusercontent.com/grindinggear/skilltree-export/" + *treeTag + "/data.json"
		fmt.Println("fetching", url)
		client := &http.Client{Timeout: 2 * time.Minute}
		resp, err := client.Get(url)
		if err != nil {
			fail("fetching tree JSON:", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fail("fetching tree JSON:", resp.Status, "— check the tag exists (-treetag), or pass -treejson")
		}
		if blob, err = io.ReadAll(resp.Body); err != nil {
			fail("fetching tree JSON:", err)
		}
	}
	doc, err := export.BuildTreeDoc(blob)
	if err != nil {
		fail("tree:", err)
	}
	treeOut := filepath.Join(*out, "tree_"+treeVersion+".json")
	if err := os.WriteFile(treeOut, doc, 0o644); err != nil {
		fail(err)
	}
	fmt.Println("->", filepath.Base(treeOut))

	// 3. modcache.jsonl, from the Go parser over the regenerated data.
	// Runs as a subprocess: the data/raw documents are compiled in via
	// embed, so a fresh build must pick up what steps 1-2 just wrote.
	run := func(env []string, args ...string) {
		fmt.Println("\n$", args)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		cmd.Env = append(os.Environ(), env...)
		if err := cmd.Run(); err != nil {
			fail("step failed:", err)
		}
	}
	run(nil, "go", "run", "./cmd/sourceupdate", "-modcache-only", "-out", *out, "-treetag", *treeTag)

	// 4. What this cannot regenerate.
	fmt.Println()
	fmt.Println("NOT regenerated (manual):")
	fmt.Println("  - .archive reference (shipped LUT bins, reference Lua/builds): replace on a version bump;")
	fmt.Println("    all differentials compare against it, so stale reference = failing tests below")

	// 5. Verify.
	if *skipTests {
		fmt.Println("\ntests skipped (-skiptests); run:")
		fmt.Println("  go test ./... -count=1")
		fmt.Println("  MP_EXPORT=1 go test ./test/ -run TestExportAgainstReference -count=1")
		return
	}
	run(nil, "go", "test", "./...", "-count=1")
	run([]string{"MP_EXPORT=1"}, "go", "test", "./test/", "-run", "TestExportAgainstReference", "-count=1")
	fmt.Println("\nsource update complete, all verification green")
}
