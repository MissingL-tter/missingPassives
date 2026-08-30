package test

import (
	"bufio"
	"encoding/json"
	goflag "flag"
	"os"
	"testing"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// The differential test: every modifier line the application has ever parsed
// (the key set of src/Data/ModCache.lua), replayed through this port and
// compared byte for byte against what ModParser.lua produced for it.
//
// Regenerate the archive dump from src/ with: luajit ../../tools/dump_parse.lua
//
// This test FAILS unless every line agrees. That is the port's definition of
// done.

var (
	maxDiffs = goflag.Int("diffs", 10, "number of disagreeing lines to print")
	diffGrep = goflag.String("diffgrep", "", "print only disagreements whose line contains this substring")
)

type archiveRecord struct {
	Line  string  `json:"line"`
	Mods  *string `json:"-"`
	Extra *string `json:"extra"`
}

// The mods field arrives as arbitrary JSON; it is recaptured as the raw
// canonical text so no reserialisation can perturb the comparison.
func (r *archiveRecord) UnmarshalJSON(b []byte) error {
	var probe struct {
		Line  string          `json:"line"`
		Mods  json.RawMessage `json:"mods"`
		Extra *string         `json:"extra"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	r.Line = probe.Line
	r.Extra = probe.Extra
	if string(probe.Mods) != "null" {
		s := string(probe.Mods)
		r.Mods = &s
	}
	return nil
}

func loadArchive(t *testing.T) []archiveRecord {
	t.Helper()
	f, err := os.Open("testdata/parse_archive.jsonl")
	if err != nil {
		t.Fatalf("archive dump not generated (run luajit ../../tools/dump_parse.lua from .archive/src/): %v", err)
	}
	defer f.Close()
	var out []archiveRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var rec archiveRecord
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("decoding archive dump: %v", err)
		}
		out = append(out, rec)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("reading archive dump: %v", err)
	}
	return out
}

func TestAgainstReference(t *testing.T) {
	// The reference dump for this differential parsed fresh (its tool wipes
	// the preloaded ModCache); run the parser in the same mode.
	modparser.SetModCache(nil)

	records := loadArchive(t)
	if len(records) == 0 {
		t.Fatal("empty archive dump")
	}

	agree, disagree, shown := 0, 0, 0
	for _, rec := range records {
		mods, extra, recognised := modparser.Parse(rec.Line)

		gotMods := "null"
		if recognised {
			gotMods = luacanon.CanonMods(mods)
		}
		wantMods := "null"
		if rec.Mods != nil {
			// The reference's parser kept a handful of numeric tag fields as
			// the raw capture text; the port parses them (luacanon).
			wantMods = luacanon.NormalizeArchiveMods(*rec.Mods)
		}
		gotExtra := extra
		wantExtra := ""
		if rec.Extra != nil {
			wantExtra = *rec.Extra
		}

		if gotMods == wantMods && gotExtra == wantExtra {
			agree++
			continue
		}
		disagree++
		if *diffGrep != "" && !contains(rec.Line, *diffGrep) {
			continue
		}
		if shown < *maxDiffs {
			shown++
			t.Errorf("line:  %s\n  want mods:  %s\n  got  mods:  %s\n  want extra: %q\n  got  extra: %q",
				rec.Line, clip(wantMods), clip(gotMods), wantExtra, gotExtra)
		}
	}

	t.Logf("corpus %d lines: %d agree, %d disagree", len(records), agree, disagree)
	if disagree > 0 {
		t.Fatalf("%d of %d lines disagree with the archive", disagree, len(records))
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func clip(s string) string {
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}
