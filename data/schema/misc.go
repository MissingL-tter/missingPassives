package schema

// Misc holds the miscellaneous balance tables of Data/Misc.lua.
type Misc struct {
	// Per-level monster tables from DefaultMonsterStats, level 1..100.
	MonsterEvasion             []float64 `json:"monsterEvasion"`
	MonsterAccuracy            []float64 `json:"monsterAccuracy"`
	MonsterLife                []float64 `json:"monsterLife"`
	MonsterLife2               []float64 `json:"monsterLife2"`
	MonsterLife3               []float64 `json:"monsterLife3"`
	MonsterAllyLife            []float64 `json:"monsterAllyLife"`
	MonsterDamage              []float64 `json:"monsterDamage"`
	MonsterAllyDamage          []float64 `json:"monsterAllyDamage"`
	MonsterArmour              []float64 `json:"monsterArmour"`
	MonsterAilmentThreshold    []float64 `json:"monsterAilmentThreshold"`
	MonsterPhysConversionMulti []float64 `json:"monsterPhysConversionMulti"`

	GameConstants      []IdValue `json:"gameConstants"`      // GameConstants row order
	CharacterConstants []KV      `json:"characterConstants"` // Character.ot, file order
	MonsterConstants   []KV      `json:"monsterConstants"`   // Monster.ot, file order

	TotemLifeMult           []IntMult   `json:"totemLifeMult"`          // first-seen SkillTotem order
	MonsterVarietyLifeMult  []NameMult  `json:"monsterVarietyLifeMult"` // MonsterVarieties row order
	MapLevelLifeMult        []LevelMult `json:"mapLevelLifeMult"`
	MapLevelBossLifeMult    []LevelMult `json:"mapLevelBossLifeMult"`
	MapLevelBossAilmentMult []LevelMult `json:"mapLevelBossAilmentMult"`
	GoldRespecPrices        []int64     `json:"goldRespecPrices"`
}

type IdValue struct {
	Id    string  `json:"id"`
	Value float64 `json:"value"`
}

// KV is one .ot constant, parsed to a number at the export edge (the
// reference shipped the raw text after the '=' and re-evaluated it with
// tonumber at load — lua-residue.md T4).
type KV struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

type IntMult struct {
	Id   int64   `json:"id"`
	Mult float64 `json:"mult"`
}

type NameMult struct {
	Name string  `json:"name"`
	Mult float64 `json:"mult"`
}

type LevelMult struct {
	Level int64   `json:"level"`
	Mult  float64 `json:"mult"`
}

// CurrencyNames maps currency base item type ids to item names.
type CurrencyNames map[string]string

// MiscData is the miscdata script's document: Misc.lua + CurrencyNames.lua.
type MiscData struct {
	Misc          Misc          `json:"misc"`
	CurrencyNames CurrencyNames `json:"currencyNames"`
}

func (MiscData) isDocument() {}

// FlavourTexts holds unique item flavour text, in the exporter's emission
// order (forced names first, then sorted visual-identity ids).
type FlavourTexts []FlavourText

func (FlavourTexts) isDocument() {}

type FlavourText struct {
	Id   string   `json:"id"`
	Name string   `json:"name"`
	Text []string `json:"text"`
}

// FoulbornMap maps unique item names to their Foulborn-transformed mod lines.
type FoulbornMap map[string][]string

func (FoulbornMap) isDocument() {}

// SkillGemList is the gem list grouped by attribute colour (the
// Export/Skills/SkillGems.txt template helper).
type SkillGemList struct {
	Groups []SkillGemGroup `json:"groups"`
}

func (SkillGemList) isDocument() {}

type SkillGemGroup struct {
	Type    string          `json:"type"`
	Active  []SkillGemEntry `json:"active"`
	Support []SkillGemEntry `json:"support"`
}

type SkillGemEntry struct {
	Name    string   `json:"name"`
	Effects []string `json:"effects"` // granted effect ids (1 or 2)
}
