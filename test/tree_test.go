package test

// The tree differential: tools/dump_tree_archive.lua dumps a freshly built
// PassiveTree (before any calc can stamp item sources onto shared node mod
// lists), and the Go tree.Load must reproduce every node's processed state
// byte-for-byte — positions, links, parsed mod lists, the name lookup
// maps, mastery effects, and each socket's per-radius node sets.

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/MissingL-tter/missingPassives/internal/luacanon"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/tree"
)

var (
	treeOnce   sync.Once
	treeCached *tree.Tree
)

func loadTree329(t *testing.T) *tree.Tree {
	t.Helper()
	loadGameData(t)
	treeOnce.Do(func() {
		treeCached = tree.Load("3_29")
	})
	return treeCached
}

type treeModState struct {
	Extra *string  `lua:"extra"`
	Count *float64 `lua:"count"`
}

type treeNodeState struct {
	ID                   any              `lua:"id"`
	Type                 string           `lua:"type"`
	Dn                   *string          `lua:"dn"`
	AscendancyName       *string          `lua:"ascendancyName"`
	Group                *float64         `lua:"group"`
	X                    *float64         `lua:"x"`
	Y                    *float64         `lua:"y"`
	LinkedID             []float64        `lua:"linkedId"`
	PassivePointsGranted *float64         `lua:"passivePointsGranted"`
	ModKey               *string          `lua:"modKey"`
	Mods                 []treeModState   `lua:"mods"`
	ModList              []*modparser.Mod `lua:"modList"`
	KeystoneMod          *modparser.Mod   `lua:"keystoneMod"`
	Unknown              *bool            `lua:"unknown"`
	Extra                *bool            `lua:"extra"`
	Sd                   []string         `lua:"sd"`
	IsProxy              *bool            `lua:"isProxy"`
	IsBlighted           *bool            `lua:"isBlighted"`
}

func strPtr(s string) *string { return &s }

func truePtr(b bool) *bool {
	if !b {
		return nil
	}
	return &b
}

func nodeStateOf(n *tree.Node, legion bool) *treeNodeState {
	st := &treeNodeState{
		Type:        n.Type,
		Dn:          strPtr(n.Name),
		ModKey:      strPtr(n.ModKey),
		ModList:     n.ModList,
		Unknown:     truePtr(n.Stats.Unknown),
		Extra:       truePtr(n.Stats.Extra),
		Sd:          n.Sd,
		KeystoneMod: n.KeystoneMod,
	}
	if legion {
		st.ID = n.IDStr
	} else {
		st.ID = float64(n.ID)
		if n.AscendancyName != "" {
			st.AscendancyName = strPtr(n.AscendancyName)
		}
		if n.GroupID != 0 {
			g := float64(n.GroupID)
			st.Group = &g
		}
		if n.HasPosition {
			x, y := n.X, n.Y
			st.X, st.Y = &x, &y
		}
		linked := make([]float64, len(n.LinkedIDs))
		for i, id := range n.LinkedIDs {
			linked[i] = float64(id)
		}
		st.LinkedID = linked
		p := n.PassivePointsGranted
		st.PassivePointsGranted = &p
		st.IsProxy = truePtr(n.IsProxy)
		st.IsBlighted = truePtr(n.IsBlighted)
		mods := make([]treeModState, len(n.Mods))
		for i, m := range n.Mods {
			if m == nil {
				continue
			}
			if m.Extra != "" {
				mods[i].Extra = strPtr(m.Extra)
			}
			if !m.Nil {
				c := float64(len(m.List))
				mods[i].Count = &c
			}
		}
		st.Mods = mods
	}
	if legion {
		// Legion records carry only the parse state.
		st.Mods = nil
	}
	return st
}

// legionNodeState mirrors the dump's slimmer legion record.
type legionNodeState struct {
	ID      string           `lua:"id"`
	Type    string           `lua:"type"`
	Dn      *string          `lua:"dn"`
	ModKey  *string          `lua:"modKey"`
	ModList []*modparser.Mod `lua:"modList"`
	Unknown *bool            `lua:"unknown"`
	Extra   *bool            `lua:"extra"`
	Sd      []string         `lua:"sd"`
}

// funcAddrRe masks function addresses inside modKey strings: they are
// run-dependent in the reference itself (tostring(function)), so no two
// dumps agree on them either.
var funcAddrRe = regexp.MustCompile(`func=(function: )?0x[0-9a-fA-F]+`)

func maskFuncAddrs(s string) string {
	return funcAddrRe.ReplaceAllString(s, "func=#")
}

// nodeRecordsEquivalent is the fallback for a byte mismatch: everything
// must still match exactly except x/y, which may drift by libm's one-ulp
// sin/cos differences (observed only where the true coordinate is ~0 and
// the value is float noise around 1e-14).
func nodeRecordsEquivalent(got, want string) bool {
	var g, w map[string]any
	if json.Unmarshal([]byte(got), &g) != nil || json.Unmarshal([]byte(want), &w) != nil {
		return false
	}
	if len(g) != len(w) {
		return false
	}
	for k, gv := range g {
		wv, ok := w[k]
		if !ok {
			return false
		}
		if k == "x" || k == "y" {
			gn, gok := gv.(float64)
			wn, wok := wv.(float64)
			if !gok || !wok {
				return false
			}
			d := gn - wn
			if d < -1e-8 || d > 1e-8 {
				return false
			}
			continue
		}
		if luacanon.Encode(gv) != luacanon.Encode(wv) {
			return false
		}
	}
	return true
}

func TestTreeAgainstReference(t *testing.T) {
	tr := loadTree329(t)
	path := filepath.Join("testdata", "tree_archive.jsonl")
	records := map[string]string{}
	forEachCalcRecord(t, path, func(k, c string) { records[k] = c })
	if len(records) < 3000 {
		t.Fatalf("tree archive looks truncated: %d records", len(records))
	}

	nodes, legion, bad := 0, 0, 0
	report := func(name, got, want string) {
		bad++
		if bad <= 5 {
			t.Errorf("%s diverged\n%s", name, diffWindow(got, want))
		}
	}
	seen := map[string]bool{}
	keys := make([]string, 0, len(records))
	for k := range records {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		want := records[k]
		switch {
		case strings.HasPrefix(k, "node."):
			id, _ := strconv.ParseInt(strings.TrimPrefix(k, "node."), 10, 64)
			node := tr.Nodes[id]
			if node == nil {
				t.Fatalf("%s: node missing from loaded tree", k)
			}
			seen[k] = true
			if got := maskFuncAddrs(luacanon.Encode(nodeStateOf(node, false))); got != maskFuncAddrs(want) && !nodeRecordsEquivalent(got, maskFuncAddrs(want)) {
				report(k, got, maskFuncAddrs(want))
			}
			nodes++
		case strings.HasPrefix(k, "legion."):
			id := strings.TrimPrefix(k, "legion.")
			node := tr.Legion.Nodes[id]
			if node == nil {
				t.Fatalf("%s: legion node missing", k)
			}
			if got := luacanon.Encode(&legionNodeState{
				ID: node.IDStr, Type: node.Type, Dn: strPtr(node.Name),
				ModKey: strPtr(node.ModKey), ModList: node.ModList,
				Unknown: truePtr(node.Stats.Unknown), Extra: truePtr(node.Stats.Extra), Sd: node.Sd,
			}); maskFuncAddrs(got) != maskFuncAddrs(want) {
				report(k, maskFuncAddrs(got), maskFuncAddrs(want))
			}
			legion++
		case k == "masteryEffects":
			effects := map[string]any{}
			for id, effect := range tr.MasteryEffects {
				effects[strconv.FormatInt(id, 10)] = map[string]any{
					"sd": effect.Sd, "modKey": effect.ModKey, "modList": effect.ModList,
				}
			}
			if got := luacanon.Encode(effects); got != want {
				report(k, got, want)
			}
		case k == "keystoneMap" || k == "notableMap" || k == "ascendancyMap" || k == "clusterNodeMap":
			var m map[string]*tree.Node
			switch k {
			case "keystoneMap":
				m = tr.KeystoneMap
			case "notableMap":
				m = tr.NotableMap
			case "ascendancyMap":
				m = tr.AscendancyMap
			case "clusterNodeMap":
				m = tr.ClusterNodeMap
			}
			var refIDs map[string]any
			if err := json.Unmarshal([]byte(want), &refIDs); err != nil {
				t.Fatal(err)
			}
			for name, refID := range refIDs {
				node := m[name]
				if node == nil {
					t.Errorf("%s: %q missing from the loaded tree", k, name)
					continue
				}
				var myID any = node.IDStr
				if node.ID != 0 {
					myID = float64(node.ID)
				}
				if myID == refID {
					continue
				}
				// Duplicate-named nodes: the reference's pairs() overwrite
				// order is hash luck. Accept when the two picks are
				// byte-identical apart from the id itself.
				refNum, isNum := refID.(float64)
				if isNum && node.ID != 0 {
					st := nodeStateOf(node, false)
					st.ID = refNum
					refIDStr := strconv.FormatInt(int64(refNum), 10)
					got := luacanon.Encode(st)
					// The mod sources embed the node id; rewrite them too.
					got = strings.ReplaceAll(got, "Tree:"+strconv.FormatInt(node.ID, 10), "Tree:"+refIDStr)
					if maskFuncAddrs(got) == maskFuncAddrs(records["node."+refIDStr]) {
						continue
					}
				}
				t.Errorf("%s %q: id %v vs reference %v (and contents differ)", k, name, myID, refID)
			}
			for name := range m {
				if _, ok := refIDs[name]; !ok {
					t.Errorf("%s: loaded tree has extra entry %q", k, name)
				}
			}
		case k == "sockets":
			sockets := map[string]any{}
			for id, socket := range tr.Sockets {
				entry := map[string]any{}
				if socket.CharmSocket {
					entry["charmSocket"] = true
				}
				if socket.NodesInRadius != nil {
					radii := make([]any, len(socket.NodesInRadius))
					for i, set := range socket.NodesInRadius {
						ids := map[string]any{}
						for nodeID := range set {
							ids[strconv.FormatInt(nodeID, 10)] = true
						}
						radii[i] = ids
					}
					entry["nodesInRadius"] = radii
				}
				sockets[strconv.FormatInt(id, 10)] = entry
			}
			if got := luacanon.Encode(sockets); got != want {
				report(k, got, want)
			}
		case k == "meta":
			var meta struct {
				NodeCount float64 `json:"nodeCount"`
			}
			var outer map[string]any
			if err := json.Unmarshal([]byte(want), &outer); err != nil {
				t.Fatal(err)
			}
			meta.NodeCount = outer["nodeCount"].(float64)
			if int(meta.NodeCount) != len(tr.Nodes) {
				t.Errorf("meta: node count %d vs reference %d", len(tr.Nodes), int(meta.NodeCount))
			}
		}
	}
	for id := range tr.Nodes {
		if !seen["node."+strconv.FormatInt(id, 10)] {
			t.Errorf("loaded tree has node %d the reference lacks", id)
		}
	}
	if bad > 5 {
		t.Errorf("suppressed diffs: %d total", bad)
	}
	t.Logf("tree archive: %d nodes, %d legion passives byte-identical", nodes, legion)
}
