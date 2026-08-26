package test

// Stage-2 spec differential: build each corpus spec natively (XML nodes +
// mastery selections + socketed jewels, over the natively parsed items)
// and byte-compare the allocated-node projections, the mastery/notable/
// keystone counts and the radius-jewel node data against the calc
// fixtures. Builds needing the unported stages (cluster subgraphs,
// timeless conquering, tattoo overrides) are skipped and counted.

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/internal/luacanon"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/tree"
)

type xmlSpecSocket struct {
	NodeID int64 `xml:"nodeId,attr"`
	ItemID int   `xml:"itemId,attr"`
}

type xmlSpec struct {
	ClassID                string `xml:"classId,attr"`
	AscendClassID          string `xml:"ascendClassId,attr"`
	SecondaryAscendClassID string `xml:"secondaryAscendClassId,attr"`
	TreeVersion            string `xml:"treeVersion,attr"`
	Nodes                  string `xml:"nodes,attr"`
	MasteryEffects         string `xml:"masteryEffects,attr"`
	Sockets                struct {
		Sockets []xmlSpecSocket `xml:"Socket"`
	} `xml:"Sockets"`
	ClusterHashFormatVersion string `xml:"clusterHashFormatVersion,attr"`
	Overrides                struct {
		Overrides []struct {
			NodeID            int64  `xml:"nodeId,attr"`
			Dn                string `xml:"dn,attr"`
			Icon              string `xml:"icon,attr"`
			ActiveEffectImage string `xml:"activeEffectImage,attr"`
		} `xml:"Override"`
	} `xml:"Overrides"`
}

type xmlBuildTree struct {
	Tree struct {
		ActiveSpec int       `xml:"activeSpec,attr"`
		Specs      []xmlSpec `xml:"Spec"`
	} `xml:"Tree"`
}

// specNodeFixture mirrors dump_calc.lua's nodeFixture projection.
type specNodeFixture struct {
	ID                   float64          `lua:"id"`
	Type                 string           `lua:"type"`
	Name                 *string          `lua:"name"`
	Dn                   *string          `lua:"dn"`
	IsTattoo             *bool            `lua:"isTattoo"`
	OverrideType         *string          `lua:"overrideType"`
	ConqueredBy          *bool            `lua:"conqueredBy"`
	DistanceToClassStart *float64         `lua:"distanceToClassStart"`
	ModList              []*modparser.Mod `lua:"modList"`
	KeystoneMod          *modparser.Mod   `lua:"keystoneMod"`
}

func specNodeFixtureOf(n *tree.SpecNode) *specNodeFixture {
	f := &specNodeFixture{
		ID:                   float64(n.ID()),
		Type:                 n.Type(),
		Name:                 n.EffectiveName(),
		Dn:                   strPtr(n.Dn),
		DistanceToClassStart: n.DistanceToClassStart,
		ModList:              n.Stats.ModList,
		KeystoneMod:          n.KeystoneMod,
	}
	if n.IsTattoo {
		f.IsTattoo = truePtr(true)
	}
	if n.OverrideType != "" {
		f.OverrideType = strPtr(n.OverrideType)
	}
	if n.ConqueredBy != nil {
		f.ConqueredBy = truePtr(true)
	}
	return f
}

// refNodeFixture re-projects the decoded NodeInput for comparison.
type refNodeProjection struct {
	in *calc.NodeInput
}

func loadCorpusSpec(t *testing.T, xmlPath string, items map[int]*item.Item) (*tree.Spec, bool) {
	t.Helper()
	blob, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatalf("read %s: %v", xmlPath, err)
	}
	var doc xmlBuildTree
	if err := xml.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("decode %s: %v", xmlPath, err)
	}
	idx := doc.Tree.ActiveSpec
	if idx < 1 || idx > len(doc.Tree.Specs) {
		idx = 1
	}
	if len(doc.Tree.Specs) == 0 {
		t.Fatalf("%s: no Spec element", xmlPath)
	}
	x := doc.Tree.Specs[idx-1]
	if x.TreeVersion != "3_29" {
		t.Fatalf("%s: unexpected treeVersion %q", xmlPath, x.TreeVersion)
	}
	attr64 := func(v string) int64 {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0 // tostring(nil) and absent attributes read as 0
		}
		return n
	}
	// The reference's version default: an explicit attribute wins; a spec
	// with a nodes attribute but no version attribute is legacy (1).
	version := 2
	if n, err := strconv.Atoi(x.ClusterHashFormatVersion); err == nil {
		version = n
	} else if x.Nodes != "" || strings.Contains(string(blob), `nodes="`) {
		version = 1
	}
	saved := &tree.SavedSpec{
		ClassID:                  attr64(x.ClassID),
		AscendClassID:            attr64(x.AscendClassID),
		SecondaryAscendClassID:   attr64(x.SecondaryAscendClassID),
		Nodes:                    x.Nodes,
		MasteryEffects:           x.MasteryEffects,
		Sockets:                  map[int64]int{},
		ClusterHashFormatVersion: version,
	}
	for _, o := range x.Overrides.Overrides {
		saved.Overrides = append(saved.Overrides, tree.SavedOverride{
			NodeID: o.NodeID, Dn: o.Dn, Icon: o.Icon, ActiveEffectImage: o.ActiveEffectImage,
		})
	}
	for _, socket := range x.Sockets.Sockets {
		saved.Sockets[socket.NodeID] = socket.ItemID
	}
	// Stage eligibility: timeless jewels wait on the algorithm stage.
	for _, itemID := range saved.Sockets {
		if it := items[itemID]; it != nil {
			if it.JewelData != nil && it.JewelData["conqueredBy"] != nil {
				return nil, false
			}
		}
	}
	tr := loadTree329(t)
	spec := tree.NewSpec(tr, items)
	spec.LoadSaved(saved)
	spec.PostLoad()
	return spec, true
}

// treeNodeFixtureOf projects a bare tree node the way nodeFixture does
// when no spec node exists (granted ascendancy nodes from other classes).
func treeNodeFixtureOf(n *tree.Node) *specNodeFixture {
	return &specNodeFixture{
		ID:          float64(n.ID),
		Type:        n.Type,
		Name:        n.NameStr,
		Dn:          strPtr(n.Name),
		ModList:     n.ModList,
		KeystoneMod: n.KeystoneMod,
	}
}

func TestSpecAgainstReference(t *testing.T) {
	loadGameData(t)
	manifest := readManifest(t)
	dumpPaths, err := filepath.Glob(filepath.Join("testdata", "calc_*.jsonl"))
	if err != nil || len(dumpPaths) == 0 {
		t.Skipf("archive dumps not present")
	}
	sort.Strings(dumpPaths)
	only := os.Getenv("MP_ONLY_SPEC")
	comparedBuilds, skipped, nodesCompared := 0, 0, 0
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
		var fixture, grantedPassives, grantedAsc string
		forEachCalcRecord(t, path, func(k, c string) {
			if strings.HasSuffix(k, ".full.fixture") || (fixture == "" && strings.HasSuffix(k, ".fixture")) {
				fixture = c
			}
			if strings.HasSuffix(k, ".full.grantedPassiveNodes") || (grantedPassives == "" && strings.HasSuffix(k, ".grantedPassiveNodes")) {
				grantedPassives = c
			}
			if strings.HasSuffix(k, ".full.grantedAscendancyNodes") || (grantedAsc == "" && strings.HasSuffix(k, ".grantedAscendancyNodes")) {
				grantedAsc = c
			}
		})
		var m map[string]any
		if err := json.Unmarshal([]byte(fixture), &m); err != nil {
			t.Fatal(err)
		}
		ref := decodeCalcFixture(m)
		items := loadCorpusItems(t, xmlPath)
		spec, ok := loadCorpusSpec(t, xmlPath, items)
		if !ok {
			skipped++
			continue
		}
		// Allocated nodes.
		var refIDs []int
		for id := range ref.Spec.AllocNodes {
			refIDs = append(refIDs, id)
		}
		sort.Ints(refIDs)
		var gotIDs []int
		for id := range spec.AllocNodes {
			gotIDs = append(gotIDs, int(id))
		}
		sort.Ints(gotIDs)
		if len(refIDs) != len(gotIDs) {
			t.Errorf("%s: allocNodes %d vs reference %d", buildKey, len(gotIDs), len(refIDs))
			continue
		}
		mismatch := false
		for i := range refIDs {
			if refIDs[i] != gotIDs[i] {
				t.Errorf("%s: allocNodes id sets differ at %d: %d vs %d", buildKey, i, gotIDs[i], refIDs[i])
				mismatch = true
				break
			}
		}
		if mismatch {
			continue
		}
		for _, id := range refIDs {
			want := luacanon.EncodeExact(ref.Spec.AllocNodes[id])
			got := luacanon.EncodeExact(specNodeFixtureOf(spec.AllocNodes[int64(id)]))
			if got != want {
				t.Errorf("%s allocNode %d: diverged\n%s", buildKey, id, diffWindow(got, want))
			}
			nodesCompared++
		}
		// Counts.
		if spec.AllocatedNotableCount != ref.Spec.AllocatedNotableCount ||
			spec.AllocatedKeystoneCount != ref.Spec.AllocatedKeystoneCount ||
			spec.AllocatedMasteryCount != ref.Spec.AllocatedMasteryCount ||
			spec.AllocatedMasteryTypeCount != ref.Spec.AllocatedMasteryTypeCount {
			t.Errorf("%s: counts diverged: notable %v/%v keystone %v/%v mastery %v/%v types %v/%v", buildKey,
				spec.AllocatedNotableCount, ref.Spec.AllocatedNotableCount,
				spec.AllocatedKeystoneCount, ref.Spec.AllocatedKeystoneCount,
				spec.AllocatedMasteryCount, ref.Spec.AllocatedMasteryCount,
				spec.AllocatedMasteryTypeCount, ref.Spec.AllocatedMasteryTypeCount)
		}
		gotTypes := map[string]float64{}
		for k, v := range spec.AllocatedMasteryTypes {
			gotTypes[k] = v
		}
		if luacanon.Encode(gotTypes) != luacanon.Encode(ref.Spec.AllocatedMasteryTypes) {
			t.Errorf("%s: mastery types diverged", buildKey)
		}
		// Radius node data: unallocated nodes inside a socketed radius
		// jewel's ring.
		gotRadius := map[string]*specNodeFixture{}
		for socketID, itemID := range spec.Jewels {
			it := items[itemID]
			if it == nil || it.JewelRadiusIndex == nil {
				continue
			}
			socketNode := spec.Tree.Nodes[socketID]
			if socketNode == nil || socketNode.NodesInRadius == nil {
				continue
			}
			for id := range socketNode.NodesInRadius[*it.JewelRadiusIndex-1] {
				if spec.AllocNodes[id] == nil && spec.Nodes[id] != nil {
					gotRadius[strconv.FormatInt(id, 10)] = specNodeFixtureOf(spec.Nodes[id])
				}
			}
		}
		for id, refNode := range ref.Spec.RadiusNodeData {
			gotNode := gotRadius[strconv.Itoa(id)]
			if gotNode == nil {
				t.Errorf("%s: radius node %d missing", buildKey, id)
				continue
			}
			if got, want := luacanon.EncodeExact(gotNode), luacanon.EncodeExact(refNode); got != want {
				t.Errorf("%s radius node %d diverged\n%s", buildKey, id, diffWindow(got, want))
			}
			nodesCompared++
		}
		for id := range gotRadius {
			n, _ := strconv.Atoi(id)
			if ref.Spec.RadiusNodeData[n] == nil {
				t.Errorf("%s: extra radius node %s", buildKey, id)
			}
		}
		// Granted passives (anoints): resolve each dumped name through the
		// tree maps the way CalcSetup does and compare the projections.
		var gp map[string]any
		if err := json.Unmarshal([]byte(grantedPassives), &gp); err != nil {
			t.Fatal(err)
		}
		for name, refVal := range gp {
			node := spec.Tree.NotableMap[name]
			if node == nil {
				node = spec.Tree.AscendancyMap[name]
			}
			if node == nil {
				t.Errorf("%s: granted passive %q unresolved", buildKey, name)
				continue
			}
			var got string
			if specNode := spec.Nodes[node.ID]; specNode != nil {
				got = luacanon.EncodeExact(specNodeFixtureOf(specNode))
			} else {
				got = luacanon.EncodeExact(treeNodeFixtureOf(node))
			}
			want := luacanon.EncodeExact(decodeCalcNode(refVal.(map[string]any)))
			if got != want {
				t.Errorf("%s granted passive %q diverged\n%s", buildKey, name, diffWindow(got, want))
			}
			nodesCompared++
		}
		var ga map[string]any
		if err := json.Unmarshal([]byte(grantedAsc), &ga); err != nil {
			t.Fatal(err)
		}
		for name, refVal := range ga {
			node := spec.Tree.AscendancyMap[name]
			if node == nil {
				t.Errorf("%s: granted ascendancy node %q unresolved", buildKey, name)
				continue
			}
			got := luacanon.EncodeExact(treeNodeFixtureOf(node))
			want := luacanon.EncodeExact(decodeCalcNode(refVal.(map[string]any)))
			if got != want {
				t.Errorf("%s granted ascendancy %q diverged\n%s", buildKey, name, diffWindow(got, want))
			}
			nodesCompared++
		}
		comparedBuilds++
	}
	if only == "" && comparedBuilds < 5 {
		t.Fatalf("expected a healthy eligible set, compared %d (skipped %d)", comparedBuilds, skipped)
	}
	t.Logf("spec differential: %d nodes byte-identical across %d builds (%d deferred to later stages)",
		nodesCompared, comparedBuilds, skipped)
}
