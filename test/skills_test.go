package test

// Skills-tab differential: load each corpus build's <Skills> element
// natively (package skills) and byte-compare every socket group's and
// gem's reference table (the typed fields rendered by luacanon) against the
// calc fixture's skillsTab dump. Runtime keys the calc stamps after load
// (and view-only keys the port skips) are masked on the fixture side, each
// with its owner noted.

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/skills"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

type xmlSkillsDoc struct {
	Build struct {
		Level string `xml:"level,attr"`
	} `xml:"Build"`
	Skills skills.XMLSkills `xml:"Skills"`
}

// gemMaskedKeys: fixture gem keys the native load does not produce.
var gemMaskedKeys = map[string]string{
	"color":     "view-only (colour code)",
	"triggered": "calc runtime mark (persisted by the app's load-time calc)",
}

// groupMaskedKeys: fixture group keys the native load does not produce.
var groupMaskedKeys = map[string]string{
	"displayLabel": "view-only",
	"slotEnabled":  "calc runtime mark (items stage)",
}

// grantedGroupKeys: what LoadSkill itself owns on a granted group.
var grantedGroupKeys = []string{"enabled", "includeInFullDPS", "groupCount", "label", "slot", "source", "mainActiveSkill", "mainActiveSkillCalcs"}

func pickKeys(bag map[string]any, keys []string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := bag[k]; ok {
			out[k] = v
		}
	}
	return out
}

func maskBag(bag map[string]any, mask map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range bag {
		if _, drop := mask[k]; !drop {
			out[k] = v
		}
	}
	return out
}

// groupTable/gemTable: the typed scalars plus the fixture's view-only keys
// (masked above), as the reference table.
func groupTable(g *skills.SocketGroup) map[string]any {
	t := luacanon.SocketGroupTable(g)
	for k, v := range luacanon.Extras[g] {
		t[k] = v
	}
	return t
}

func gemTable(g *skills.Gem) map[string]any {
	t := luacanon.GemInstanceTable(g)
	for k, v := range luacanon.Extras[g] {
		t[k] = v
	}
	return t
}

func TestSkillsTabAgainstReference(t *testing.T) {
	loadData(t)
	manifest := readManifest(t)
	dumpPaths, err := filepath.Glob(filepath.Join("testdata", "calc_*.jsonl"))
	if err != nil || len(dumpPaths) == 0 {
		t.Skipf("archive dumps not present")
	}
	sort.Strings(dumpPaths)
	only := os.Getenv("MP_ONLY_SKILLS")
	groupsCompared, gemsCompared, builds := 0, 0, 0
	for _, path := range dumpPaths {
		buildKey := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "calc_"), ".jsonl")
		if only != "" && buildKey != only {
			continue
		}
		xmlRel := manifest[buildKey]
		if xmlRel == "" {
			continue
		}
		xmlPath := filepath.Clean(filepath.Join("..", ".archive", "src", xmlRel))
		var fixture string
		forEachCalcRecord(t, path, func(k, c string) {
			if strings.HasSuffix(k, ".full.fixture") || (fixture == "" && strings.HasSuffix(k, ".fixture")) {
				fixture = c
			}
		})
		var m map[string]any
		if err := json.Unmarshal([]byte(fixture), &m); err != nil {
			t.Fatal(err)
		}
		ref := decodeCalcFixture(m)
		if ref.SkillsTab == nil {
			continue
		}

		blob, err := os.ReadFile(xmlPath)
		if err != nil {
			t.Fatal(err)
		}
		var doc xmlSkillsDoc
		if err := xml.Unmarshal(blob, &doc); err != nil {
			t.Fatal(err)
		}
		tab := skills.Load(&doc.Skills, ref.CharacterLevel)

		// UpdateSocketGroups needs slot -> selected item.
		items := loadCorpusItems(t, xmlPath)
		slotSel := map[string]*item.Item{}
		for _, slot := range ref.ItemsTab.Slots {
			if slot.ItemID != nil {
				slotSel[slot.SlotName] = items[int(*slot.ItemID)]
			}
		}
		tab.UpdateSocketGroups(func(slotName string) *item.Item { return slotSel[slotName] })

		refGroups := ref.SkillsTab.SocketGroups
		if len(tab.SocketGroupList) > len(refGroups) {
			t.Errorf("%s: %d socket groups vs reference %d", buildKey, len(tab.SocketGroupList), len(refGroups))
			continue
		}
		// Fixture-only trailing groups: granted-skill groups the calc
		// CREATED during the dump's own run (authored shells whose XML was
		// never re-saved). Our calc's match-or-create makes them too; the
		// load half only requires that every extra is a source group.
		for i := len(tab.SocketGroupList); i < len(refGroups); i++ {
			if !refGroups[i].Granted() {
				t.Errorf("%s: fixture-only group %d is not a granted group", buildKey, i+1)
			}
		}
		refGroups = refGroups[:len(tab.SocketGroupList)]
		for i, group := range tab.SocketGroupList {
			refGroup := refGroups[i]
			if group.Granted() {
				// Item/tree-granted groups: the calc's granted-skill update
				// rewrites their gems after load (that update is ported in
				// calc); the load half owns only the XML-loaded group keys.
				got := luacanon.EncodeExact(pickKeys(groupTable(group), grantedGroupKeys))
				want := luacanon.EncodeExact(pickKeys(groupTable(refGroup.SocketGroup), grantedGroupKeys))
				if !luacanon.SameCanon(got, want) {
					t.Errorf("%s granted group %d diverged\n%s", buildKey, i+1, diffWindow(got, want))
				}
				groupsCompared++
				continue
			}
			got := luacanon.EncodeExact(maskBag(groupTable(group), groupMaskedKeys))
			want := luacanon.EncodeExact(maskBag(groupTable(refGroup.SocketGroup), groupMaskedKeys))
			if !luacanon.SameCanon(got, want) {
				t.Errorf("%s group %d diverged\n%s", buildKey, i+1, diffWindow(got, want))
			}
			groupsCompared++
			if len(group.GemList) != len(refGroup.GemList) {
				t.Errorf("%s group %d: %d gems vs reference %d", buildKey, i+1, len(group.GemList), len(refGroup.GemList))
				continue
			}
			for j, gem := range group.GemList {
				refGem := refGroup.GemList[j]
				gotShadow := gemShadow(gemTable(gem), gemDataID(gem), grantedEffectID(gem))
				wantShadow := gemShadow(gemTable(refGem.Gem), refGem.GemDataID, refGem.GrantedEffectID)
				dropCalcStamped(gotShadow.KV, wantShadow.KV)
				gotGem := luacanon.EncodeExact(gotShadow)
				wantGem := luacanon.EncodeExact(wantShadow)
				if gotGem != wantGem {
					t.Errorf("%s group %d gem %d (%s) diverged\n%s", buildKey, i+1, j+1, gem.NameSpec, diffWindow(gotGem, wantGem))
				}
				gemsCompared++
			}
		}
		// Imbued support map.
		gotImbued := map[string]string{}
		for slot, ge := range tab.ImbuedSupportBySlot {
			gotImbued[slot] = ge.Id
		}
		wantImbued := map[string]string{}
		for slot, id := range ref.SkillsTab.ImbuedSupportBySlot {
			wantImbued[slot] = id
		}
		if luacanon.Encode(gotImbued) != luacanon.Encode(wantImbued) {
			t.Errorf("%s: imbuedSupportBySlot diverged: %v vs %v", buildKey, gotImbued, wantImbued)
		}
		builds++
	}
	if only == "" && builds < 5 {
		t.Fatalf("expected a healthy build set, compared %d", builds)
	}
	t.Logf("skills tab differential: %d groups / %d gems byte-identical across %d builds", groupsCompared, gemsCompared, builds)
}

type gemShadowT struct {
	KV              map[string]any `lua:"kv"`
	GemDataID       *string        `lua:"gemDataId"`
	GrantedEffectID *string        `lua:"grantedEffectId"`
}

// calcStampedGemKeys: keys the calc writes back onto the gem instance when
// the skill has parts/stages (CalcActiveSkill L248-266); when the native
// load produced no value the fixture's calc-stamped one is accepted.
var calcStampedGemKeys = []string{"skillPart", "skillPartCalcs", "skillStageCount", "skillStageCountCalcs", "skillMineCount", "skillMineCountCalcs", "skillMinion", "skillMinionCalcs", "skillMinionSkill", "skillMinionSkillCalcs", "skillMinionItemSet", "skillMinionItemSetCalcs"}

func gemShadow(kv map[string]any, gemDataID, grantedEffectID *string) *gemShadowT {
	return &gemShadowT{KV: maskBag(kv, gemMaskedKeys), GemDataID: gemDataID, GrantedEffectID: grantedEffectID}
}

func dropCalcStamped(native, fixture map[string]any) {
	for _, k := range calcStampedGemKeys {
		if _, ok := native[k]; !ok {
			delete(fixture, k)
		}
	}
}

func gemDataID(g *skills.Gem) *string {
	if g.GemData == nil {
		return nil
	}
	return &g.GemData.Id
}

func grantedEffectID(g *skills.Gem) *string {
	if g.GrantedEffect == nil {
		return nil
	}
	return &g.GrantedEffect.Id
}

var _ = calc.BuildInput{}
