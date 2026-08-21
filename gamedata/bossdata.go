package gamedata

// BossData is the bossData script's document (Data/BossSkills.lua and
// Data/Bosses.lua).
type BossData struct {
	Skills     []BossSkill    `json:"skills"`     // one per #skill directive
	SkillLists [][]string     `json:"skillLists"` // one per #skillList directive
	Bosses     []*BossMonster `json:"bosses"`     // one per #boss (nil = unknown type)
}

type BossSkill struct {
	Key        string `json:"key"` // "<boss> <skill>"
	DamageType string `json:"damageType"`
	// DamageMultipliers in Physical/Lightning/Cold/Fire/Chaos order.
	DamageMultipliers    []BossDamageMult `json:"damageMultipliers"`
	UberDamageMultiplier *float64         `json:"uberDamageMultiplier,omitempty"`

	HasPen     bool       `json:"hasPen,omitempty"`
	Pens       []PenEntry `json:"pens,omitempty"`
	HasUberPen bool       `json:"hasUberPen,omitempty"`
	UberPens   []PenEntry `json:"uberPens,omitempty"`

	Speed       float64  `json:"speed"`               // emitted when != 700
	UberSpeed   *float64 `json:"uberSpeed,omitempty"` // emitted when set and != 700
	CritChance  int64    `json:"critChance"`          // emitted when != 5
	EarlierUber bool     `json:"earlierUber,omitempty"`

	// Additional stats; the counts gate emission and reproduce the
	// reference's base/uber count bookkeeping. Values are pre-rendered
	// (number text or "flag" literal).
	HasAdditional bool              `json:"hasAdditional,omitempty"`
	BaseCount     int64             `json:"baseCount,omitempty"`
	UberCount     int64             `json:"uberCount,omitempty"`
	BaseVals      map[string]string `json:"baseVals,omitempty"`
	UberVals      map[string]string `json:"uberVals,omitempty"`
}

type BossDamageMult struct {
	Type   string  `json:"type"`
	Min    float64 `json:"min"`
	Spread float64 `json:"spread"` // (max-min)/100
}

// PenEntry is one penetration line; Text is the pre-rendered value
// (a number or the literal `""`).
type PenEntry struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type BossMonster struct {
	DisplayName string `json:"displayName"`
	ArmourMult  int64  `json:"armourMult"`
	EvasionMult int64  `json:"evasionMult"`
	IsUber      bool   `json:"isUber"`
}
