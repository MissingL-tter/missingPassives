package schema

// ClusterJewels holds the cluster jewel data.
type ClusterJewels struct {
	Jewels           []ClusterJewelSize `json:"jewels"` // PassiveTreeExpansionJewels row order
	NotableSortOrder []NameOrder        `json:"notableSortOrder"`
	Keystones        []string           `json:"keystones"`
	OrbitOffsets     []OrbitOffset      `json:"orbitOffsets"`
}

func (ClusterJewels) isDocument() {}

type ClusterJewelSize struct {
	Name            string         `json:"name"`
	Size            string         `json:"size"`
	SizeIndex       int            `json:"sizeIndex"`
	MinNodes        int64          `json:"minNodes"`
	MaxNodes        int64          `json:"maxNodes"`
	SmallIndicies   []int64        `json:"smallIndicies"`
	NotableIndicies []int64        `json:"notableIndicies"`
	SocketIndicies  []int64        `json:"socketIndicies"`
	TotalIndicies   int64          `json:"totalIndicies"`
	Skills          []ClusterSkill `json:"skills"`
}

type ClusterSkill struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"` // "(Legacy)" suffix already applied
	Icon        string   `json:"icon"`
	MasteryIcon *string  `json:"masteryIcon"` // nil when the skill has no mastery
	Tag         string   `json:"tag"`
	Stats       []string `json:"stats"` // described lines; enchant text derives from these
}

type NameOrder struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type OrbitOffset struct {
	NodeId int64   `json:"nodeId"`
	Starts []int64 `json:"starts"` // per cluster size 0..2
}
