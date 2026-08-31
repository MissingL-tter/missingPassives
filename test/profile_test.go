package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MissingL-tter/missingPassives/calc"
)

// TestProfileRecalc times a full recalculation of one build — the work the
// app will do per edit. Off by default; run with
//
//	MP_PROFILE=cocuser go test ./test/ -run TestProfileRecalc -count=1 -v
//	MP_PROFILE=cocuser go test ./test/ -run TestProfileRecalc -count=1 -cpuprofile cpu.out
//
// Cold = empty GlobalCache (BuildOutput performs every active skill, the
// build-load case). Warm = cache carried over from the previous recalc (the
// per-edit case: only the main skill's perform re-runs).
func TestProfileRecalc(t *testing.T) {
	build := os.Getenv("MP_PROFILE")
	if build == "" {
		t.Skip("profiling harness; set MP_PROFILE=<dump key> to run")
	}
	loadData(t)

	path := filepath.Join("testdata", "calc_"+build+".jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no dump for %q: %v", build, err)
	}
	variant := build + ".full"
	var fixture, grantedNodes, grantedAsc string
	forEachCalcRecord(t, path, func(k, c string) {
		switch k {
		case variant + ".fixture":
			fixture = c
		case variant + ".grantedPassiveNodes":
			grantedNodes = c
		case variant + ".grantedAscendancyNodes":
			grantedAsc = c
		}
	})
	var m map[string]any
	if err := json.Unmarshal([]byte(fixture), &m); err != nil {
		t.Fatal(err)
	}
	replay := &calc.ReplayInput{}
	passives := &fixturePassives{
		passive:    decodeGrantedPassiveNodes(grantedNodes),
		ascendancy: decodeGrantedPassiveNodes(grantedAsc),
	}

	const iters = 30

	// Cold: fresh cache each time (build-load recalc).
	// One live build input, reused across recalcs — the app's shape (PoB
	// mutates one build object in place per edit).
	in := decodeCalcFixture(m)
	in.Spec.Passives = passives
	coldStart := time.Now()
	var lastEnv *calc.Env
	for i := 0; i < iters; i++ {
		lastEnv = calc.BuildOutput(in, "MAIN", replay)
	}
	cold := time.Since(coldStart) / iters

	// Warm: reuse the previous recalc's cache (per-edit recalc — the cache
	// fill loop finds every entry present and only the main perform runs).
	warmReplay := *replay
	warmReplay.GlobalCache = lastEnv.GlobalCache
	warmStart := time.Now()
	for i := 0; i < iters; i++ {
		calc.BuildOutput(in, "MAIN", &warmReplay)
	}
	warm := time.Since(warmStart) / iters

	skills := len(lastEnv.PlayerActiveSkills)
	fmt.Printf("PROFILE %s: cold(full cache rebuild) %v/recalc, warm(per-edit) %v/recalc, %d active skills, %d cache entries\n",
		build, cold.Round(time.Microsecond), warm.Round(time.Microsecond), skills, len(lastEnv.GlobalCache))
}
