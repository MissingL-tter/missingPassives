package schema

// BasesData holds the item base database plus the generated rare templates.
// Types is keyed by template file ("axe", ..., "graft"); each entry is one
// event per emitting directive (#base -> 0 or 1 bases, #baseMatch -> n), in
// template order. Rares is one event per #setBestBase/#setBase directive
// (nil when the directive found no base).
type BasesData struct {
	Types map[string][][]ItemBase `json:"types"`
	Rares []*RareItem             `json:"rares"`
	// RareBlobs is the complete ordered rare-template list as the generated
	// file carries it: the directive outputs interleaved with the
	// hand-written blobs the template holds as passthrough.
	RareBlobs [][]string `json:"rareBlobs"`
}

// RareItem is one generated crafted-item template ([[...]] block).
type RareItem struct {
	Lines []string `json:"lines"`
}

type ItemBase struct {
	DisplayName      string   `json:"displayName"`
	Type             string   `json:"type"`
	SubType          string   `json:"subType,omitempty"`
	Hidden           bool     `json:"hidden,omitempty"`
	SocketLimit      *float64 `json:"socketLimit,omitempty"`
	Tags             []string `json:"tags"` // sorted
	InfluenceBaseTag string   `json:"influenceBaseTag,omitempty"`

	Implicit         []string `json:"implicit,omitempty"` // described lines
	ImplicitModTypes []string `json:"implicitModTypes"`   // one ModTags string per line
	ImplicitIds      []string `json:"implicitIds,omitempty"`

	Enchant          []string `json:"enchant,omitempty"`
	EnchantIds       []string `json:"enchantIds,omitempty"`
	EnchantModTypes  []string `json:"enchantModTypes,omitempty"`
	CannotBeAnointed bool     `json:"cannotBeAnointed,omitempty"`

	Weapon   *WeaponBase   `json:"weapon,omitempty"`
	Armour   *ArmourBase   `json:"armour,omitempty"`
	Flask    *FlaskBase    `json:"flask,omitempty"`
	Tincture *TinctureBase `json:"tincture,omitempty"`

	ReqLevel *int64 `json:"reqLevel,omitempty"`
	ReqStr   *int64 `json:"reqStr,omitempty"`
	ReqDex   *int64 `json:"reqDex,omitempty"`
	ReqInt   *int64 `json:"reqInt,omitempty"`

	FlavourText []string `json:"flavourText,omitempty"`
}

type WeaponBase struct {
	PhysicalMin    int64   `json:"physicalMin"`
	PhysicalMax    int64   `json:"physicalMax"`
	CritChanceBase float64 `json:"critChanceBase"`
	AttackRateBase float64 `json:"attackRateBase"`
	Range          int64   `json:"range"`
}

// ArmourBase mirrors the emitted armour table; each pair is present only
// when its minimum is positive. MovementPenalty carries the emitted
// (negated) value.
type ArmourBase struct {
	BlockChance     *int64 `json:"blockChance,omitempty"`
	ArmourMin       *int64 `json:"armourMin,omitempty"`
	ArmourMax       *int64 `json:"armourMax,omitempty"`
	EvasionMin      *int64 `json:"evasionMin,omitempty"`
	EvasionMax      *int64 `json:"evasionMax,omitempty"`
	EnergyShieldMin *int64 `json:"energyShieldMin,omitempty"`
	EnergyShieldMax *int64 `json:"energyShieldMax,omitempty"`
	MovementPenalty *int64 `json:"movementPenalty,omitempty"`
	WardMin         *int64 `json:"wardMin,omitempty"`
	WardMax         *int64 `json:"wardMax,omitempty"`
}

type FlaskBase struct {
	Life        *int64   `json:"life,omitempty"`
	Mana        *int64   `json:"mana,omitempty"`
	Duration    float64  `json:"duration"`
	ChargesUsed int64    `json:"chargesUsed"`
	ChargesMax  int64    `json:"chargesMax"`
	Buff        []string `json:"buff,omitempty"` // described buff lines
	HasBuff     bool     `json:"hasBuff,omitempty"`
}

type TinctureBase struct {
	ManaBurn float64 `json:"manaBurn"`
	Cooldown float64 `json:"cooldown"`
}
