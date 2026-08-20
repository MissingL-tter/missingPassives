package test

import (
	"bufio"
	"encoding/json"
	goflag "flag"
	"os"
	"testing"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// The differential test: every modifier line the application has ever parsed
// (the key set of src/Data/ModCache.lua), replayed through this port and
// compared byte for byte against what ModParser.lua produced for it.
//
// Regenerate the oracle from src/ with: luajit ../../tools/dump_oracle.lua
//
// This test FAILS unless every line agrees. That is the port's definition of
// done.

var (
	maxDiffs = goflag.Int("diffs", 10, "number of disagreeing lines to print")
	diffGrep = goflag.String("diffgrep", "", "print only disagreements whose line contains this substring")
)

type oracleRecord struct {
	Line  string  `json:"line"`
	Mods  *string `json:"-"`
	Extra *string `json:"extra"`
}

// The mods field arrives as arbitrary JSON; it is recaptured as the raw
// canonical text so no reserialisation can perturb the comparison.
func (r *oracleRecord) UnmarshalJSON(b []byte) error {
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

func loadOracle(t *testing.T) []oracleRecord {
	t.Helper()
	f, err := os.Open("testdata/oracle.jsonl")
	if err != nil {
		t.Fatalf("oracle not generated (run luajit ../../tools/dump_oracle.lua from .archive/src/): %v", err)
	}
	defer f.Close()
	var out []oracleRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var rec oracleRecord
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("decoding oracle: %v", err)
		}
		out = append(out, rec)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("reading oracle: %v", err)
	}
	return out
}

func TestAgainstReference(t *testing.T) {
	records := loadOracle(t)
	if len(records) == 0 {
		t.Fatal("empty oracle")
	}

	agree, disagree, shown := 0, 0, 0
	for _, rec := range records {
		mods, extra := modparser.Parse(rec.Line)

		gotMods := "null"
		if mods != nil {
			gotMods = modparser.Canon(mods)
		}
		wantMods := "null"
		if rec.Mods != nil {
			wantMods = *rec.Mods
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
		t.Fatalf("%d of %d lines disagree with the reference", disagree, len(records))
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
