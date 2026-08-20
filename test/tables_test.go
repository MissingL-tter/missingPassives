package test

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/MissingL-tter/missingPassives/internal/luapat"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// The table-level differential test: every entry of every pattern table,
// compared canonically against the reference's own tables. The parse test only
// verifies entries some corpus line reaches; this one verifies all of them —
// data entries byte for byte, closure entries as agreeing that both sides hold
// a function there.
//
// Regenerate from .archive/src/ with: luajit ../../tools/dump_tables.lua

type tableRecord struct {
	Table string `json:"table"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func TestTablesAgainstReference(t *testing.T) {
	f, err := os.Open("testdata/tables_oracle.jsonl")
	if err != nil {
		t.Fatalf("table oracle not generated (run luajit ../../tools/dump_tables.lua from .archive/src/): %v", err)
	}
	defer f.Close()

	tables := modparser.Tables()
	seen := map[string]map[string]bool{}
	for name := range tables {
		seen[name] = map[string]bool{}
	}

	var total, mismatched, missing, shown int
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var rec tableRecord
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("decoding table oracle: %v", err)
		}
		total++
		goTable, ok := tables[rec.Table]
		if !ok {
			t.Fatalf("no Go table named %q", rec.Table)
		}

		// The reference keys are Lua patterns; the Go tables are keyed by their
		// regex conversion — except plain-substring tables and the jewel tables'
		// exact-text keys, which are identical on both sides.
		goKey := rec.Key
		if _, present := goTable[goKey]; !present {
			if converted, err := luapat.Convert(rec.Key); err == nil {
				goKey = converted
			}
		}
		seen[rec.Table][goKey] = true

		goVal, ok := goTable[goKey]
		if !ok {
			missing++
			if shown < 20 {
				shown++
				t.Errorf("%s[%q]: missing from the Go table (as %q)", rec.Table, rec.Key, goKey)
			}
			continue
		}
		if accepted, ok := referenceNondeterminism[rec.Table+"|"+goKey]; ok && accepted[rec.Value] {
			continue
		}
		if got := modparser.Canon(goVal); got != rec.Value {
			mismatched++
			if shown < 20 {
				shown++
				t.Errorf("%s[%q]:\n  want %s\n  got  %s", rec.Table, rec.Key, clip(rec.Value), clip(got))
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	// Entries the Go side has that the reference does not.
	var extra int
	for name, goTable := range tables {
		for k := range goTable {
			if !seen[name][k] {
				extra++
				if shown < 20 {
					shown++
					t.Errorf("%s[%q]: not present in the reference", name, k)
				}
			}
		}
	}

	t.Logf("table oracle: %d entries, %d mismatched, %d missing, %d extra", total, mismatched, missing, extra)
	if mismatched+missing+extra > 0 {
		t.Fatalf("table comparison failed: %d mismatched, %d missing, %d extra of %d", mismatched, missing, extra, total)
	}
}

// referenceNondeterminism lists entries where the reference itself gives
// different values run to run, so the fresh-dump comparison accepts any of
// them. The only known case: two cluster jewel sizes share this enchant text
// with different skill ids, and Lua's pairs() order decides which survives —
// three consecutive runs of the reference's own loop produced both answers.
// The Go side deterministically keeps affliction_curse_effect_small (the
// lexicographically greatest, see tools/gen_vocab.lua).
var referenceNondeterminism = map[string]map[string]bool{
	"clusterJewelSkills|added small passive skills grant: 2% increased effect of your curses": {
		`{"1":{"flags":0,"keywordFlags":0,"name":"JewelData","type":"LIST","value":{"key":"clusterJewelSkill","value":"affliction_curse_effect"}}}`:       true,
		`{"1":{"flags":0,"keywordFlags":0,"name":"JewelData","type":"LIST","value":{"key":"clusterJewelSkill","value":"affliction_curse_effect_small"}}}`: true,
	},
}
