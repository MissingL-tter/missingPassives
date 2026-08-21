package test

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// The ModTools differential test: for every corpus line that parses to
// modifiers, the archive's formatting, tag round-trip, source-mod round-trip
// and comparison results are replayed against this port's. Fails on any
// disagreement.
//
// Regenerate from .archive/src/ with: luajit ../../tools/dump_modtools.lua

type modtoolsRecord struct {
	Line     string   `json:"line"`
	Fmt      []string `json:"fmt"`
	Params   []string `json:"params"`
	Tags     []string `json:"tags"`
	SrcFmt   []string `json:"srcfmt"`
	PTags    []string `json:"ptags"`
	PSrc     []string `json:"psrc"`
	SelfCmp  []bool   `json:"selfcmp"`
	CrossCmp *bool    `json:"crosscmp"`
}

func TestModToolsAgainstReference(t *testing.T) {
	f, err := os.Open("testdata/modtools_archive.jsonl")
	if err != nil {
		t.Fatalf("modtools archive dump not generated (run luajit ../../tools/dump_modtools.lua from .archive/src/): %v", err)
	}
	defer f.Close()

	var checked, disagree, shown int
	var prevFirst *modparser.Mod

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var rec modtoolsRecord
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("decoding modtools archive dump: %v", err)
		}

		mods, _ := modparser.Parse(rec.Line)
		if len(mods) != len(rec.Fmt) {
			t.Fatalf("%s: parsed %d mods, archive dump has %d", rec.Line, len(mods), len(rec.Fmt))
		}

		fail := func(what string, i int, want, got string) {
			disagree++
			if shown < 10 {
				shown++
				t.Errorf("%s [mod %d] %s:\n  want %s\n  got  %s", rec.Line, i+1, what, want, got)
			}
		}

		for i, m := range mods {
			mm := m.(*modparser.Mod)
			checked++

			if got := modparser.FormatMod(mm); got != rec.Fmt[i] {
				fail("formatMod", i, rec.Fmt[i], got)
			}
			if got := modparser.FormatModParams(mm); got != rec.Params[i] {
				fail("formatModParams", i, rec.Params[i], got)
			}
			gotTags := modparser.FormatTags(modparser.ModTags(mm))
			if gotTags != rec.Tags[i] {
				fail("formatTags", i, rec.Tags[i], gotTags)
			}

			cp := modparser.CopyMod(mm)
			modparser.SetSource(cp, "GoPort")
			if got := modparser.FormatSourceMod(cp); got != rec.SrcFmt[i] {
				fail("formatSourceMod", i, rec.SrcFmt[i], got)
			}

			if got := modparser.Canon(modparser.ParseTags(rec.Tags[i])); got != rec.PTags[i] {
				fail("parseTags", i, rec.PTags[i], got)
			}
			gotPSrc := "null"
			if pm := modparser.ParseFormattedSourceMod(rec.SrcFmt[i]); pm != nil {
				gotPSrc = modparser.Canon(pm)
			}
			if gotPSrc != rec.PSrc[i] {
				fail("parseFormattedSourceMod", i, rec.PSrc[i], gotPSrc)
			}

			if got := modparser.CompareModParams(mm, modparser.CopyMod(mm)); got != rec.SelfCmp[i] {
				fail("compareModParams(self)", i, boolStr(rec.SelfCmp[i]), boolStr(got))
			}
		}

		if rec.CrossCmp != nil && prevFirst != nil {
			if got := modparser.CompareModParams(mods[0].(*modparser.Mod), prevFirst); got != *rec.CrossCmp {
				fail("compareModParams(cross)", 0, boolStr(*rec.CrossCmp), boolStr(got))
			}
		}
		prevFirst = mods[0].(*modparser.Mod)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	t.Logf("modtools vs archive: %d mods checked across 7 behaviours, %d disagreements", checked, disagree)
	if disagree > 0 {
		t.Fatalf("%d disagreements with the archive", disagree)
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
