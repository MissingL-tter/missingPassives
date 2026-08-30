package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// NodeID is a passive node id as the tree document carries it: link lists
// and group node lists hold decimal strings, `skill` and expansionJewel
// proxies hold numbers.
type NodeID int64

func (id *NodeID) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("node id %q is not numeric", s)
		}
		*id = NodeID(n)
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*id = NodeID(n)
	return nil
}

// PassiveTree is data/raw/tree_<version>.json (the GGG tree data with the
// exporter's fixups), the fields the runtime tree reads.
type PassiveTree struct {
	MinX                  float64              `json:"min_x"`
	MinY                  float64              `json:"min_y"`
	MaxX                  float64              `json:"max_x"`
	MaxY                  float64              `json:"max_y"`
	Classes               []TreeClass          `json:"classes"`
	AlternateAscendancies []TreeAscendancy     `json:"alternate_ascendancies"`
	Constants             TreeConstants        `json:"constants"`
	Groups                map[string]TreeGroup `json:"groups"`
	Nodes                 map[string]TreeNode  `json:"nodes"` // keyed by skill id; "root" is the class-start hub
}

func (PassiveTree) isDocument() {}

type TreeClass struct {
	Name         string           `json:"name"`
	BaseStr      float64          `json:"base_str"`
	BaseDex      float64          `json:"base_dex"`
	BaseInt      float64          `json:"base_int"`
	Ascendancies []TreeAscendancy `json:"ascendancies"`
}

type TreeAscendancy struct {
	Id         string `json:"id"`
	InternalId string `json:"internalId"`
	Name       string `json:"name"`
}

type TreeConstants struct {
	SkillsPerOrbit []int64   `json:"skillsPerOrbit"`
	OrbitRadii     []float64 `json:"orbitRadii"`
}

type TreeGroup struct {
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	IsProxy bool     `json:"isProxy"`
	Nodes   []NodeID `json:"nodes"`
	Orbits  []int64  `json:"orbits"`
}

type TreeNode struct {
	Skill                  int64               `json:"skill"`
	Name                   string              `json:"name"`
	Icon                   string              `json:"icon"`
	Group                  int64               `json:"group"`
	Orbit                  *int64              `json:"orbit"` // absent on cluster-jewel template nodes
	OrbitIndex             int64               `json:"orbitIndex"`
	Out                    []NodeID            `json:"out"`
	In                     []NodeID            `json:"in"`
	IsKeystone             bool                `json:"isKeystone"`
	IsNotable              bool                `json:"isNotable"`
	IsMastery              bool                `json:"isMastery"`
	IsJewelSocket          bool                `json:"isJewelSocket"`
	IsAscendancyStart      bool                `json:"isAscendancyStart"`
	IsProxy                bool                `json:"isProxy"`
	IsBlighted             bool                `json:"isBlighted"`
	IsJustIcon             bool                `json:"isJustIcon"`
	IsMultipleChoice       bool                `json:"isMultipleChoice"`
	IsMultipleChoiceOption bool                `json:"isMultipleChoiceOption"`
	AscendancyName         string              `json:"ascendancyName"`
	ClassStartIndex        *int64              `json:"classStartIndex"`
	GrantedPassivePoints   float64             `json:"grantedPassivePoints"`
	GrantedStrength        float64             `json:"grantedStrength"`
	GrantedDexterity       float64             `json:"grantedDexterity"`
	GrantedIntelligence    float64             `json:"grantedIntelligence"`
	Stats                  []string            `json:"stats"`
	MasteryEffects         []TreeMasteryEffect `json:"masteryEffects"`
	ExpansionJewel         *TreeExpansionJewel `json:"expansionJewel"`
	ActiveEffectImage      string              `json:"activeEffectImage"`
}

type TreeMasteryEffect struct {
	Effect       int64    `json:"effect"`
	Stats        []string `json:"stats"`
	ReminderText []string `json:"reminderText"`
}

// TreeExpansionJewel marks a cluster jewel socket: Size 0-2 (small,
// medium, large), Index its slot in the parent, Proxy the ring's proxy
// node, Parent the enclosing socket (absent on the outer large sockets).
type TreeExpansionJewel struct {
	Size   int64   `json:"size"`
	Index  int64   `json:"index"`
	Proxy  NodeID  `json:"proxy"`
	Parent *NodeID `json:"parent"`
}
