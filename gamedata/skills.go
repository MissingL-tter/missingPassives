package gamedata

// SkillsData is the skills script's document: per-template skill
// definitions plus the gem list (Data/Gems.lua).
type SkillsData struct {
	Files map[string]SkillFile `json:"files"` // keyed by template ("act_str", ...)
	Gems  []GemDef             `json:"gems"`  // SkillGems row order
}

// SkillFile pairs by position: Skills has one entry per #skill directive,
// Tails one per #mods directive (snapshotted at #mods time, which preserves
// the reference's stale-state and aliased-interpolation semantics).
type SkillFile struct {
	Skills []SkillHeader `json:"skills"`
	Tails  []SkillTail   `json:"tails"`
}

type SkillHeader struct {
	GrantedId string `json:"grantedId"` // the skills[...] key
	// Invalid marks an unknown granted effect: only the opening line was
	// emitted.
	Invalid bool `json:"invalid,omitempty"`

	Name         string   `json:"name"`
	Hidden       bool     `json:"hidden,omitempty"`
	BaseTypeName *string  `json:"baseTypeName,omitempty"`
	Description  *string  `json:"description,omitempty"` // final escaped text
	HasFlavour   bool     `json:"hasFlavour,omitempty"`
	FlavourText  []string `json:"flavourText,omitempty"`

	Color                    int64    `json:"color"`
	BaseEffectiveness        *float64 `json:"baseEffectiveness,omitempty"`
	IncrementalEffectiveness *float64 `json:"incrementalEffectiveness,omitempty"`

	Support bool `json:"support,omitempty"`
	// Support-only fields ("SkillType.X" strings).
	RequireSkillTypes []string `json:"requireSkillTypes,omitempty"`
	AddSkillTypes     []string `json:"addSkillTypes,omitempty"`
	ExcludeSkillTypes []string `json:"excludeSkillTypes,omitempty"`
	IsTrigger         bool     `json:"isTrigger,omitempty"`
	SupportGemsOnly   bool     `json:"supportGemsOnly,omitempty"`
	IgnoreMinionTypes bool     `json:"ignoreMinionTypes,omitempty"`
	PlusVersionOf     *string  `json:"plusVersionOf,omitempty"`

	// Active-only fields.
	SkillTypes        []string `json:"skillTypes,omitempty"`
	MinionSkillTypes  []string `json:"minionSkillTypes,omitempty"`
	SkillTotemId      *int64   `json:"skillTotemId,omitempty"`
	CastTime          *float64 `json:"castTime,omitempty"`
	CannotBeSupported bool     `json:"cannotBeSupported,omitempty"`

	WeaponTypes          []string `json:"weaponTypes,omitempty"` // sorted
	StatDescriptionScope string   `json:"statDescriptionScope"`
}

type SkillTail struct {
	// ModsArgs is the raw #mods directive argument (the noBaseFlags/noStats/
	// ... gates).
	ModsArgs      string          `json:"modsArgs,omitempty"`
	Support       bool            `json:"support,omitempty"`
	BaseFlags     []string        `json:"baseFlags,omitempty"`
	BaseMods      []string        `json:"baseMods,omitempty"` // raw template text
	QualityStats  []StatValue     `json:"qualityStats,omitempty"`
	ConstantStats []StatValue     `json:"constantStats,omitempty"`
	Stats         []string        `json:"stats"`
	NotMinionStat []string        `json:"notMinionStat,omitempty"`
	Levels        []SkillLevel    `json:"levels"`
}

type StatValue struct {
	Id    string  `json:"id"`
	Value float64 `json:"value"`
}

type SkillLevel struct {
	Level  int64              `json:"level"`
	Values []float64          `json:"values"`
	Extra  map[string]float64 `json:"extra,omitempty"`  // rendered sorted by key
	Interp []string           `json:"interp,omitempty"` // statInterpolation, pre-rendered
	Cost   map[string]int64   `json:"cost,omitempty"`   // rendered sorted by key
}

type GemDef struct {
	VariantId                string   `json:"variantId"` // key = "Metadata/Items/Gems/SkillGem" + VariantId
	Name                     string   `json:"name"`
	BaseTypeName             *string  `json:"baseTypeName,omitempty"`
	GameId                   string   `json:"gameId"`
	GrantedEffectId          string   `json:"grantedEffectId"`
	SecondaryGrantedEffectId *string  `json:"secondaryGrantedEffectId,omitempty"`
	SecondaryEffectName      *string  `json:"secondaryEffectName,omitempty"`
	VaalGem                  bool     `json:"vaalGem,omitempty"`
	Tags                     []string `json:"tags"`
	TagString                string   `json:"tagString"`
	ReqStr                   int64    `json:"reqStr"`
	ReqDex                   int64    `json:"reqDex"`
	ReqInt                   int64    `json:"reqInt"`
	NaturalMaxLevel          int      `json:"naturalMaxLevel"`
}
