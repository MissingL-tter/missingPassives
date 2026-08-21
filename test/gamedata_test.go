package test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/luacanon"
	"github.com/MissingL-tter/missingPassives/modparser"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// diffWindow returns s around the first byte where s and other diverge.
func diffWindow(s, other string) string {
	i := 0
	for i < len(s) && i < len(other) && s[i] == other[i] {
		i++
	}
	start := i - 80
	if start < 0 {
		start = 0
	}
	end := i + 120
	if end > len(s) {
		end = len(s)
	}
	return "[@" + strconv.Itoa(i) + "] ..." + s[start:end] + "..."
}

// Structured mods inside the data tree canonicalise via their plain-table
// shadow.
var _ = func() bool {
	luacanon.RegisterAdapter(func(v any) (any, bool) {
		switch t := v.(type) {
		case *modparser.Mod:
			return data.ModCanon(t), true
		case *modparser.D:
			return data.DCanon(t), true
		case *data.GrantedEffect:
			return data.GrantedEffectCanon(t), true
		case *data.StatMapEntry:
			return data.StatMapEntryCanon(t), true
		case *data.SkillLevel:
			return data.SkillLevelCanon(t), true
		case data.UnportedFn:
			return luacanon.Fn{}, true
		case *data.Gem:
			return data.GemCanon(t), true
		}
		return nil, false
	})
	return true
}()

// The game-data differential test: builds the gamedata documents over the
// extracted GGPK, assembles the runtime data set with data.Load, and
// compares each ported subtree canonically against the archive dump
// (tools/dump_gamedata.lua run over the loaded Lua application). Fails on
// any disagreement.
func TestGameDataAgainstReference(t *testing.T) {
	dumpPath := filepath.Join("testdata", "gamedata_archive.jsonl")
	d := loadGameData(t)

	checks := map[string]func() any{
		"monsterEvasionTable":             func() any { return d.MonsterEvasionTable },
		"monsterAccuracyTable":            func() any { return d.MonsterAccuracyTable },
		"monsterLifeTable":                func() any { return d.MonsterLifeTable },
		"monsterLifeTable2":               func() any { return d.MonsterLifeTable2 },
		"monsterLifeTable3":               func() any { return d.MonsterLifeTable3 },
		"monsterAllyLifeTable":            func() any { return d.MonsterAllyLifeTable },
		"monsterDamageTable":              func() any { return d.MonsterDamageTable },
		"monsterAllyDamageTable":          func() any { return d.MonsterAllyDamageTable },
		"monsterArmourTable":              func() any { return d.MonsterArmourTable },
		"monsterAilmentThresholdTable":    func() any { return d.MonsterAilmentThresholdTable },
		"monsterPhysConversionMultiTable": func() any { return d.MonsterPhysConversionMultiTable },
		"gameConstants":                   func() any { return d.GameConstants },
		"characterConstants":              func() any { return d.CharacterConstants },
		"monsterConstants":                func() any { return d.MonsterConstants },
		"totemLifeMult":                   func() any { return d.TotemLifeMult },
		"monsterVarietyLifeMult":          func() any { return d.MonsterVarietyLifeMult },
		"mapLevelLifeMult":                func() any { return d.MapLevelLifeMult },
		"mapLevelBossLifeMult":            func() any { return d.MapLevelBossLifeMult },
		"mapLevelBossAilmentMult":         func() any { return d.MapLevelBossAilmentMult },
		"goldRespecPrices":                func() any { return d.GoldRespecPrices },
		"misc":                            func() any { return d.Misc },
		"powerStatList": func() any {
			m := map[string]any{"GetFromOutput": luacanon.Fn{}}
			for i, e := range d.PowerStatList {
				m[strconv.Itoa(i+1)] = e
			}
			return m
		},
		"skillColorMap":                  func() any { return d.SkillColorMap },
		"monsterExperienceLevelMap":      func() any { return d.MonsterExperienceLevelMap },
		"cursePriority":                  func() any { return d.CursePriority },
		"keystones":                      func() any { return d.Keystones },
		"ailmentTypeList":                func() any { return d.AilmentTypeList },
		"elementalAilmentTypeList":       func() any { return d.ElementalAilmentTypeList },
		"nonDamagingAilmentTypeList":     func() any { return d.NonDamagingAilmentTypeList },
		"nonElementalAilmentTypeList":    func() any { return d.NonElementalAilmentTypeList },
		"nonDamagingAilment":             func() any { return d.NonDamagingAilment },
		"defaultHighPrecision":           func() any { return d.DefaultHighPrecision },
		"highPrecisionMods":              func() any { return d.HighPrecisionMods },
		"modScalability":                 func() any { return d.ModScalability },
		"weaponTypeInfo":                 func() any { return d.WeaponTypeInfo },
		"unarmedWeaponData":              func() any { return d.UnarmedWeaponData },
		"jewelRadii":                     func() any { return d.JewelRadii },
		"jewelRadius":                    func() any { return d.JewelRadius },
		"maxJewelRadius":                 func() any { return d.MaxJewelRadius },
		"enchantmentSource":              func() any { return d.EnchantmentSource },
		"timelessJewelTypes":             func() any { return d.TimelessJewelTypes },
		"timelessJewelSeedMin":           func() any { return d.TimelessJewelSeedMin },
		"timelessJewelSeedMax":           func() any { return d.TimelessJewelSeedMax },
		"timelessJewelAdditions":         func() any { return d.TimelessJewelAdditions },
		"itemTagSpecial":                 func() any { return d.ItemTagSpecial },
		"itemTagSpecialExclusionPattern": func() any { return d.ItemTagSpecialExclusionPattern },
		"casterTagCrucibleUniques":       func() any { return d.CasterTagCrucibleUniques },
		"minionTagCrucibleUniques":       func() any { return d.MinionTagCrucibleUniques },
		"costs":                          func() any { return d.Costs },
		"mapMods":                        func() any { return d.MapMods },
		"nodeIDList":                     func() any { return d.NodeIDList },
		"abyssNotableNames":              func() any { return d.AbyssNotableNames },
		"timelessJewelTradeIDs":          func() any { return d.TimelessJewelTradeIDs },
		"timelessJewelLUTs":              func() any { return d.TimelessJewelLUTs },
		// function-valued members: ported by the stat-describer and
		// timeless-jewel-data modules
		"describeStats":              func() any { return luacanon.Fn{} },
		"readLUT":                    func() any { return luacanon.Fn{} },
		"repairLUTs":                 func() any { return luacanon.Fn{} },
		"readAbyssJewelLUT":          func() any { return luacanon.Fn{} },
		"resolveAbyssJewelComponent": func() any { return luacanon.Fn{} },
		"getAbyssJewelComponentRoll": func() any { return luacanon.Fn{} },
		"bosses":                         func() any { return d.Bosses },
		"bossStats":                      func() any { return d.BossStats },
		"enemyIsBossTooltip":             func() any { return d.EnemyIsBossTooltip },
		"essences":                       func() any { return d.Essences },
		"pantheons":                      func() any { return d.Pantheons },
		"crucible":                       func() any { return d.Crucible },
		"masterMods":                     func() any { return d.MasterMods },
		"flavourText":                    func() any { return d.FlavourText },
		"enchantments": func() any {
			m := map[string]any{"Helmet": d.HelmetEnchants}
			for k, v := range d.Enchantments {
				m[k] = v
			}
			return m
		},
		"beastCraft":     func() any { return d.BeastCraft },
		"necropolisMods": func() any { return d.NecropolisMods },
		"uniqueMods":     func() any { return d.UniqueMods },
		"veiledMods":     func() any { return d.VeiledMods },
		"clusterJewels":  func() any { return d.ClusterJewels },
		"clusterJewelInfoForNotable": func() any { return d.ClusterJewelInfoForNotable },
		"bossSkills":       func() any { return d.BossSkills },
		"bossSkillsList":   func() any { return d.BossSkillsList },
		"foulbornMap":      func() any { return d.FoulbornMap },
		"itemBases":        func() any { return d.ItemBases },
		"itemBaseLists":    func() any { return d.ItemBaseLists },
		"itemBaseTypeList": func() any { return d.ItemBaseTypeList },
		"rares":            func() any { return d.Rares },
		"rareLikeUniques":  func() any { return d.RareLikeUniques },
		"minions":          func() any { return d.Minions },
		"spectres":         func() any { return d.Spectres },
		"skills":           func() any { return d.Skills },
		"skillStatMap":     func() any { return d.SkillStatMap },
		"gems":             func() any { return d.Gems },
		"gemForSkill":      func() any { return d.GemForSkillCanon() },
		"gemForBaseName":   func() any { return d.GemForBaseName },
		"gemsByGameId": func() any {
			out := map[string]map[string]string{}
			for gameId, m := range d.GemsByGameId {
				out[gameId] = map[string]string{}
				for variantId, gem := range m {
					out[gameId][variantId] = gem.Id
				}
			}
			return out
		},
		"gemGrantedEffectIdForVaalGemId": func() any { return d.GemGrantedEffectIdForVaalGemId },
		"gemVaalGemIdForBaseGemId":       func() any { return d.GemVaalGemIdForBaseGemId },
		"skills.statMapKeys": func() any {
			out := map[string][]string{}
			for id, keys := range readStatMapCopiesHelper(dumpPath) {
				out[id] = keys
			}
			return out
		},
	}
	for typ := range d.Uniques {
		typ := typ
		checks["uniques."+typ] = func() any { return d.Uniques[typ] }
	}
	_ = checks["uniques.generated"] // built by data.buildGeneratedUniques
	for key := range d.ItemMods {
		key := key
		checks["itemMods."+key] = func() any { return d.ItemMods[key] }
	}

	f, err := os.Open(dumpPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<28)
	var checked, disagree int
	for sc.Scan() {
		var rec struct {
			K string `json:"k"`
			C string `json:"c"`
			H string `json:"h"`
		}
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("bad dump line: %v", err)
		}
		check, ok := checks[rec.K]
		if !ok {
			disagree++
			t.Errorf("%s: dumped but not ported/checked", rec.K)
			continue
		}
		checked++
		got := luacanon.Encode(check())
		if rec.H != "" {
			if msHash(got) != rec.H {
				disagree++
				t.Errorf("%s differs from the archive (canon hash mismatch, %d bytes)", rec.K, len(got))
			}
			continue
		}
		if got != rec.C {
			disagree++
			t.Errorf("%s differs from the archive\n  got:  %s\n  want: %s", rec.K, diffWindow(got, rec.C), diffWindow(rec.C, got))
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("game-data vs archive: %d subtrees checked, %d disagreements", checked, disagree)
	if disagree > 0 {
		t.Fatalf("%d disagreements with the archive", disagree)
	}
}
