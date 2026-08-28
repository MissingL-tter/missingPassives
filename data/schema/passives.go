package schema

// PassiveStat is one stat on an exported passive node, as mutated by the
// stat describer.
type PassiveStat struct {
	Id        string   `json:"id"`
	Min       float64  `json:"min"`
	Max       float64  `json:"max"`
	Fmt       string   `json:"fmt,omitempty"`
	MinZ      bool     `json:"minZ,omitempty"`
	MaxZ      bool     `json:"maxZ,omitempty"`
	Index     *int     `json:"index,omitempty"`
	StatOrder *float64 `json:"statOrder,omitempty"`
}

// LegionPassives holds the timeless jewel alternate tree data.
type LegionPassives struct {
	Nodes     []LegionNode        `json:"nodes"`     // AlternatePassiveSkills row order
	Additions []ConqueredAddition `json:"additions"` // AlternatePassiveAdditions row order
}

type LegionNode struct {
	Id          string        `json:"id"`
	Icon        string        `json:"icon"`
	Ks          bool          `json:"ks"`
	Not         bool          `json:"not"`
	Dn          string        `json:"dn"`
	Oidx        float64       `json:"oidx"` // #EVAL: for non-keystones this is a LuaJIT-PRNG layout offset
	Sd          []string      `json:"sd"`
	Stats       []PassiveStat `json:"stats"` // sorted by stat id
	SortedStats []string      `json:"sortedStats"`
}

type ConqueredAddition struct {
	Id          string        `json:"id"`
	Dn          string        `json:"dn"`
	Sd          []string      `json:"sd"`
	Stats       []PassiveStat `json:"stats"`
	SortedStats []string      `json:"sortedStats"`
}

// TattooPassives holds the tattoo override nodes, in PassiveSkillOverrides
// row order (the Lua keys them by display name, later rows overwriting).
type TattooPassives struct {
	Nodes []TattooNode `json:"nodes"`
}

type TattooNode struct {
	Name              string        `json:"name"` // display name; the output key
	Id                string        `json:"id"`   // override row id
	OverrideType      string        `json:"overrideType"`
	Ks                bool          `json:"ks"`
	Not               bool          `json:"not"`
	M                 bool          `json:"m"`
	TargetType        string        `json:"targetType"`
	TargetValue       string        `json:"targetValue"`
	ReminderText      *string       `json:"reminderText,omitempty"`
	MinimumConnected  int64         `json:"minimumConnected"`
	MaximumConnected  int64         `json:"maximumConnected"` // 100 when the row has no cap
	ActiveEffectImage string        `json:"activeEffectImage"`
	Legacy            *bool         `json:"legacy,omitempty"`
	Icon              string        `json:"icon"`
	Sd                []string      `json:"sd"`
	Stats             []PassiveStat `json:"stats"` // sorted by stat id
}
