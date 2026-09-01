package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestApplyCoverage checks the option table against the reference's own:
// every option the application gives an apply function has one here, and
// this port invents none. The list was extracted once from the booted
// reference, alongside the option table itself.
func TestApplyCoverage(t *testing.T) {
	blob, err := os.ReadFile(filepath.Join("testdata", "reference_applies.json"))
	if err != nil {
		t.Fatalf("reference list missing: %v", err)
	}
	var want []string
	if err := json.Unmarshal(blob, &want); err != nil {
		t.Fatal(err)
	}
	inWant := make(map[string]bool, len(want))
	for _, v := range want {
		inWant[v] = true
	}
	missing := 0
	for _, v := range want {
		opt := byVar[Var(v)]
		if opt == nil {
			t.Errorf("%s: not in the option table", v)
			continue
		}
		if opt.Apply == nil {
			t.Errorf("%s: the reference applies modifiers here, this port does not", v)
			missing++
		}
	}
	for i := range Options {
		if Options[i].Apply != nil && !inWant[string(Options[i].Var)] {
			t.Errorf("%s: this port applies modifiers the reference does not", Options[i].Var)
		}
	}
	t.Logf("apply coverage: %d of %d options", len(want)-missing, len(want))
}
