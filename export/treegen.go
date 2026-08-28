// treegen converts GGG's published passive-tree JSON
// (github.com/grindinggear/skilltree-export, per-version tags) into the
// conventional-JSON tree document the tree package consumes
// (data/raw/tree_<v>.json), replacing the retired luajit dump of PoB's
// TreeData/<v>/tree.lua.
//
// It reproduces the SEMANTICS of PoB's whole ingestion pipeline:
//
//  1. fix_ascendancy_positions.py (upstream repo root, 3.29.1 state):
//     GGG keyword-tag stripping on node stats/reminderText, ascendancy
//     group repositioning to fixed board slots, the extra legacy notables,
//     and dropping extraImages/sprites/imageZoomLevels (sprites are view
//     data; the sprites.json split is not emitted here).
//  2. Common.lua jsonToLua + Lua load: a REGEX pipeline over the fixed
//     JSON text. Structurally that means numeric-looking object keys
//     become Lua number keys and arrays become 1-based tables; textually
//     it swaps every '['/']' for '{'/'}' INSIDE string values too, and
//     its `{(%w+)}` -> `{[0]=%1}` quirk can only fire inside strings
//     (the python serializer's indent puts real single-element arrays on
//     multiple lines, so the pattern never matches structure).
//  3. conventional JSON out; the Lua-table container shape lives only in
//     the tests that compare against the archive.
//
// The canon-format equivalence to the retired luajit dump was proven
// byte-for-byte when this port landed; test/treegen_test.go pins the
// current output to the committed artifact and the GGG source.
package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type point struct{ x, y float64 }

// nodeGroups: fix_ascendancy_positions.py NODE_GROUPS.
var nodeGroups = map[string]point{
	"Juggernaut": {-10400, 5200}, "Berserker": {-10400, 3700}, "Chieftain": {-10400, 2200},
	"Raider": {10200, 5200}, "Deadeye": {10200, 2200}, "Pathfinder": {10200, 3700},
	"Occultist": {-1500, -9850}, "Elementalist": {0, -9850}, "Necromancer": {1500, -9850},
	"Slayer": {1500, 9800}, "Gladiator": {-1500, 9800}, "Champion": {0, 9800},
	"Inquisitor": {-10400, -2200}, "Hierophant": {-10400, -3700}, "Guardian": {-10400, -5200},
	"Assassin": {10200, -5200}, "Trickster": {10200, -3700}, "Saboteur": {10200, -2200},
	"Ascendant": {-7800, 7200}, "Reliquarian": {-7800, 8900}, "Luminary": {-7800, 10600},
	"Warden": {8250, 8350}, "Primalist": {7200, 9400}, "Warlock": {9300, 7300},
	"Aul": {-6750, 12000}, "Breachlord": {-5250, 12000}, "Catarina": {-3750, 12000},
	"Trialmaster": {-2250, 12000}, "Delirious": {-750, 12000}, "Farrul": {750, 12000},
	"Lycia": {2250, 12000}, "KingInTheMists": {3750, 12000}, "Olroth": {5250, 12000},
	"Oshabi": {6750, 12000}, "Necromantic": {9750, 12000},
	"Abyssal": {-750, 13600}, "Brinerot": {750, 13600},
}

type extraNode struct {
	name    string
	icon    string
	notable bool
	skill   float64
	offset  point
}

// extraNodes: EXTRA_NODES, keyed by ascendancy (iteration order does not
// affect the output — every write is keyed).
var extraNodes = map[string][]extraNode{
	"Necromancer": {{"Nine Lives", "Art/2DArt/SkillIcons/passives/Ascendants/Int.png", true, 27602, point{-1500, -1000}}},
	"Guardian":    {{"Searing Purity", "Art/2DArt/SkillIcons/passives/Ascendants/StrInt.png", true, 57568, point{-1000, 1500}}},
	"Berserker":   {{"Indomitable Resolve", "Art/2DArt/SkillIcons/passives/Ascendants/Str.png", true, 52435, point{-1000, 0}}},
	"Ascendant":   {{"Unleashed Potential", "Art/2DArt/SkillIcons/passives/Ascendants/SkillPoint.png", false, 19355, point{-1000, 1000}}},
	"Champion":    {{"Fatal Flourish", "Art/2DArt/SkillIcons/passives/Ascendants/StrDex.png", true, 42469, point{0, 1000}}},
	"Raider":      {{"Fury of Nature", "Art/2DArt/SkillIcons/passives/Ascendants/Dex.png", true, 18054, point{1000, -1500}}},
	"Saboteur":    {{"Harness the Void", "Art/2DArt/SkillIcons/passives/Ascendants/DexInt.png", true, 57331, point{1000, -1500}}},
}

// extraNodeGroupIDs: EXTRA_NODE_IDS (GroupID only; the script uses nothing
// else on this path).
var extraNodeGroupIDs = map[string]float64{
	"Nine Lives": 44472, "Searing Purity": 50933, "Harness the Void": 37841,
	"Fury of Nature": 56600, "Fatal Flourish": 63033, "Indomitable Resolve": 25519,
	"Unleashed Potential": 60495,
}

// extraNodeStats: EXTRA_NODES_STATS.
var extraNodeStats = map[string]struct {
	stats    []string
	reminder []string
}{
	"Nine Lives":          {[]string{"25% of Damage taken Recouped as Life, Mana and Energy Shield", "Recoup Effects instead occur over 3 seconds"}, []string{"(Only Damage from Hits can be Recouped, over 4 seconds following the Hit)"}},
	"Searing Purity":      {[]string{"45% of Chaos Damage taken as Fire Damage", "45% of Chaos Damage taken as Lightning Damage"}, []string{}},
	"Harness the Void":    {[]string{"27% chance to gain 25% of Non-Chaos Damage with Hits as Extra Chaos Damage", "13% chance to gain 50% of Non-Chaos Damage with Hits as Extra Chaos Damage", "7% chance to gain 100% of Non-Chaos Damage with Hits as Extra Chaos Damage"}, []string{}},
	"Fury of Nature":      {[]string{"Non-Damaging Elemental Ailments you inflict spread to nearby enemies within 2 metres", "Non-Damaging Elemental Ailments you inflict have 100% more Effect"}, []string{"(Elemental Ailments are Ignited, Scorched, Chilled, Frozen, Brittled, Shocked, and Sapped)"}},
	"Fatal Flourish":      {[]string{"Final Repeat of Attack Skills deals 60% more Damage", "Non-Travel Attack Skills Repeat an additional Time"}, []string{}},
	"Indomitable Resolve": {[]string{"Deal 10% less Damage", "Take 25% less Damage"}, []string{}},
	"Unleashed Potential": {[]string{"400% increased Endurance, Frenzy and Power Charge Duration", "25% chance to gain a Power, Frenzy or Endurance Charge on Kill", "+1 to Maximum Endurance Charges", "+1 to Maximum Frenzy Charges", "+1 to Maximum Power Charges"}, []string{}},
}

// escapeGGGString comes from statdesc.go (the same Common.lua function the
// python fix script mirrors).
var (
	reZeroQuirk = regexp.MustCompile(`\{([A-Za-z0-9]+)\}`) // Lua %w: letters+digits
	reNumKey    = regexp.MustCompile(`^\d[\d.]*$`)
)

func obj(v any) map[string]any { return v.(map[string]any) }

// fixTree ports fix_ascendancy_positions (minus the sprites.json split;
// sprites are deleted below either way).
func fixTree(data map[string]any) error {
	nodes := obj(data["nodes"])
	for _, nv := range nodes {
		node := obj(nv)
		for _, field := range []string{"stats", "reminderText"} {
			if lines, ok := node[field].([]any); ok {
				for i, l := range lines {
					lines[i] = escapeGGGString(l.(string))
				}
			}
		}
	}
	groups := obj(data["groups"])
	type ascGroup struct {
		name  string
		group map[string]any
	}
	var ascGroups []ascGroup
	for _, gv := range groups {
		group := obj(gv)
		groupNodes := group["nodes"].([]any)
		if len(groupNodes) == 0 {
			continue
		}
		first := obj(nodes[groupNodes[0].(string)])
		if asc, ok := first["ascendancyName"].(string); ok {
			ascGroups = append(ascGroups, ascGroup{asc, group})
		}
	}
	start := map[string]point{}
	for _, ag := range ascGroups {
		for _, nid := range ag.group["nodes"].([]any) {
			if _, isStart := obj(nodes[nid.(string)])["isAscendancyStart"]; isStart {
				start[ag.name] = point{ag.group["x"].(float64), ag.group["y"].(float64)}
			}
		}
	}
	for _, ag := range ascGroups {
		target, ok := nodeGroups[ag.name]
		if !ok {
			return fmt.Errorf("no board slot for ascendancy %q", ag.name)
		}
		// offset first, then add — float addition is order-sensitive and
		// the python computes Point2D(target - start) before +=.
		offX := target.x - start[ag.name].x
		offY := target.y - start[ag.name].y
		ag.group["x"] = ag.group["x"].(float64) + offX
		ag.group["y"] = ag.group["y"].(float64) + offY
	}
	for asc, list := range extraNodes {
		for _, en := range list {
			groupID := extraNodeGroupIDs[en.name]
			gid := strconv.FormatInt(int64(groupID), 10)
			if _, taken := groups[gid]; taken {
				return fmt.Errorf("extra node %q: group id %s already taken", en.name, gid)
			}
			groups[gid] = map[string]any{
				"x":      nodeGroups[asc].x + en.offset.x,
				"y":      nodeGroups[asc].y + en.offset.y,
				"orbits": []any{float64(0)},
				"nodes":  []any{strconv.FormatInt(int64(en.skill), 10)},
			}
			node := map[string]any{
				"name":           en.name,
				"icon":           en.icon,
				"skill":          en.skill,
				"group":          groupID,
				"ascendancyName": asc,
				"orbit":          float64(0),
				"orbitIndex":     float64(0),
				"out":            []any{},
				"in":             []any{},
				"stats":          []any{},
				"reminderText":   []any{},
			}
			if en.notable {
				node["isNotable"] = true
			}
			if st, ok := extraNodeStats[en.name]; ok {
				stats := make([]any, len(st.stats))
				for i, s := range st.stats {
					stats[i] = s
				}
				reminder := make([]any, len(st.reminder))
				for i, s := range st.reminder {
					reminder[i] = s
				}
				node["stats"] = stats
				node["reminderText"] = reminder
			}
			nodes[strconv.FormatInt(int64(en.skill), 10)] = node
		}
	}
	delete(data, "extraImages")
	delete(data, "sprites")
	delete(data, "imageZoomLevels")
	return nil
}

// luaString applies jsonToLua's text-level effect on a string value: the
// global bracket swap, then the {alnum} -> {[0]=alnum} injection (which,
// inside a string, is just text).
func luaString(s string) string {
	s = strings.ReplaceAll(s, "[", "{")
	s = strings.ReplaceAll(s, "]", "}")
	return reZeroQuirk.ReplaceAllString(s, "{[0]=$1}")
}

// toLoaded applies the jsonToLua pipeline's SEMANTIC effects to the fixed
// JSON value, leaving the container format conventional (arrays stay
// arrays): numeric-looking object keys collapse the way Lua number keys
// do (1.0 and 1 are one key, formatted per tostring), and every string
// undergoes the jsonToLua text mangling. The Lua-table container shape is
// reproduced only in tests (tree differential via luacanon).
func toLoaded(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			key := k
			if reNumKey.MatchString(k) {
				n, err := strconv.ParseFloat(k, 64)
				if err != nil {
					panic("numeric-looking key failed to parse: " + k)
				}
				key = numKey(n)
			} else {
				key = luaString(k)
			}
			out[key] = toLoaded(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = toLoaded(val)
		}
		return out
	case string:
		return luaString(t)
	case nil:
		panic("JSON null has no Lua-load equivalent; the tree data never carries one")
	default:
		return v
	}
}

// numKey is Lua tostring() for a number key (canon.lua sorts tostring'd
// keys; %.14g with an integer fast path).
func numKey(v float64) string {
	if v == float64(int64(v)) && v < 1e15 && v > -1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

// BuildTreeDoc runs the whole pipeline: GGG data.json bytes in,
// conventional-JSON tree document bytes out (with trailing newline).
func BuildTreeDoc(gggJSON []byte) ([]byte, error) {
	var data map[string]any
	if err := json.Unmarshal(gggJSON, &data); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if err := fixTree(data); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(toLoaded(data)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
