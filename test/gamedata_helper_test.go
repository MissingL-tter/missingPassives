package test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/export"
	"github.com/MissingL-tter/missingPassives/gamedata"
)

// readStatMapCopiesHelper parses the skills.statMapKeys fixture record from
// the game-data dump (the canon string is itself JSON with arrays as
// {"1": ..} objects). Panics on malformed fixtures (runs under sync.Once).
func readStatMapCopiesHelper(dumpPath string) map[string][]string {
	f, err := os.Open(dumpPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<28)
	for sc.Scan() {
		var rec struct {
			K string `json:"k"`
			C string `json:"c"`
		}
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			panic(err)
		}
		if rec.K != "skills.statMapKeys" {
			continue
		}
		var raw map[string]map[string]string
		if err := json.Unmarshal([]byte(rec.C), &raw); err != nil {
			panic("statMapKeys fixture: " + err.Error())
		}
		out := map[string][]string{}
		for id, keys := range raw {
			list := make([]string, len(keys))
			for idx, key := range keys {
				n, err := strconv.Atoi(idx)
				if err != nil || n < 1 || n > len(keys) {
					panic("statMapKeys fixture: bad index " + idx)
				}
				list[n-1] = key
			}
			out[id] = list
		}
		return out
	}
	return map[string][]string{}
}

var (
	gameDataOnce sync.Once
	gameDataSet  *data.Data
	gameDataSkip string
	gameDataFail string
	// gameDataGlobalEffect is each skill's hasGlobalEffect as data.Load left
	// it; see restoreGameDataLoadState.
	gameDataGlobalEffect map[string]bool
)

// restoreGameDataLoadState undoes the granted-effect mutation a calc run
// leaves on the shared data set, so the game-data comparison sees the
// post-load state whatever ran before it.
func restoreGameDataLoadState(d *data.Data) {
	for id, ge := range d.Skills {
		ge.HasGlobalEffect = gameDataGlobalEffect[id]
	}
}

// loadGameData assembles the runtime data set once per test binary: the
// gamedata documents built over the extracted GGPK, loaded via data.Load.
// Skips when the GGPK extraction or the game-data archive dump (source of
// the statMap replay fixture) is missing.
func loadGameData(t *testing.T) *data.Data {
	t.Helper()
	gameDataOnce.Do(func() {
		dumpPath := filepath.Join("testdata", "gamedata_archive.jsonl")
		if _, err := os.Stat(dumpPath); err != nil {
			gameDataSkip = "archive dump not present: " + err.Error()
			return
		}
		srcDir := filepath.Join("..", ".archive", "src", "Export", "ggpk")
		refDir := filepath.Join("..", ".archive", "src")
		if _, err := os.Stat(filepath.Join(srcDir, "Data", "stats.datc64")); err != nil {
			gameDataSkip = "extracted GGPK not present: " + err.Error()
			return
		}
		if err := export.WriteEnumFiles(filepath.Join(srcDir, "Data")); err != nil {
			gameDataFail = "writing enum tables: " + err.Error()
			return
		}
		dats, err := export.LoadDats(filepath.Join(srcDir, "Data"))
		if err != nil {
			gameDataFail = "loading dats: " + err.Error()
			return
		}
		ctx := &export.Ctx{Dats: dats, SrcDir: srcDir, TplDir: refDir}

		docs := map[string]any{}
		for _, s := range export.Scripts {
			switch s.Name {
			case "miscdata", "costs", "bossData", "modScalability",
				"essence", "pantheons", "crucible", "masters", "flavourText", "enchant",
				"mods", "cluster", "bases", "uModsToText", "minions", "skills":
				doc, err := s.Build(ctx)
				if err != nil {
					gameDataFail = s.Name + ": build: " + err.Error()
					return
				}
				docs[s.Name] = doc
			}
		}
		foulborn, err := os.ReadFile(filepath.Join(refDir, "Data", "ModFoulbornMap.jsonc"))
		if err != nil {
			gameDataFail = "ModFoulbornMap: " + err.Error()
			return
		}
		gameDataSet = data.Load(data.Sources{
			Misc:             docs["miscdata"].(gamedata.MiscData),
			Costs:            docs["costs"].(gamedata.Costs),
			Boss:             docs["bossData"].(gamedata.BossData),
			ModScalability:   docs["modScalability"].(gamedata.ModScalability),
			Essences:         docs["essence"].(gamedata.Essences),
			Pantheons:        docs["pantheons"].(gamedata.Pantheons),
			Crucible:         docs["crucible"].(gamedata.CrucibleNodes),
			Masters:          docs["masters"].(gamedata.MasterCrafts),
			FlavourText:      docs["flavourText"].(gamedata.FlavourTexts),
			Enchants:         docs["enchant"].(gamedata.Enchants),
			Mods:             docs["mods"].(gamedata.ModsData),
			Cluster:          docs["cluster"].(gamedata.ClusterJewels),
			Bases:            docs["bases"].(gamedata.BasesData),
			Uniques:          docs["uModsToText"].(gamedata.Uniques),
			MinionsDoc:       docs["minions"].(gamedata.Minions),
			Skills:           docs["skills"].(gamedata.SkillsData),
			StatMapCopies:    readStatMapCopiesHelper(dumpPath),
			FoulbornMapJSONC: foulborn,
		})
		// A calc run stamps hasGlobalEffect onto granted effects through
		// the lazy statMap copies, faithfully -- the reference mutates its
		// own global skill tables the same way. The game-data canon is the
		// post-load state, and this binary shares one data set across
		// tests, so record what to put back before comparing.
		gameDataGlobalEffect = map[string]bool{}
		for id, ge := range gameDataSet.Skills {
			gameDataGlobalEffect[id] = ge.HasGlobalEffect
		}
	})
	if gameDataSkip != "" {
		t.Skip(gameDataSkip)
	}
	if gameDataFail != "" {
		t.Fatal(gameDataFail)
	}
	return gameDataSet
}
