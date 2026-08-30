package schema

import "encoding/json"

// Minions holds the minion/spectre definitions, one list per template file,
// in template emit order.
type Minions struct {
	Spectres []MinionDef `json:"spectres"`
	Minions  []MinionDef `json:"minions"`
}

func (Minions) isDocument() {}

type MinionDef struct {
	// Skip marks an emit whose monster variety id was not found; it emits
	// nothing but keeps the template's emit sequence aligned.
	Skip bool `json:"skip,omitempty"`

	Key                          string     `json:"key"` // the minions[...] table key
	Name                         string     `json:"name"`
	MonsterTags                  []string   `json:"monsterTags"`
	BaseDamageIgnoresAttackSpeed bool       `json:"baseDamageIgnoresAttackSpeed,omitempty"`
	Life                         float64    `json:"life"`
	LifeScaling                  []string   `json:"lifeScaling,omitempty"` // AltLife1/AltLife2 lines
	EnergyShield                 *float64   `json:"energyShield,omitempty"`
	Armour                       *float64   `json:"armour,omitempty"`
	Evasion                      *float64   `json:"evasion,omitempty"`
	FireResist                   int64      `json:"fireResist"`
	ColdResist                   int64      `json:"coldResist"`
	LightningResist              int64      `json:"lightningResist"`
	ChaosResist                  int64      `json:"chaosResist"`
	Damage                       float64    `json:"damage"`
	DamageSpread                 float64    `json:"damageSpread"`
	AttackTime                   float64    `json:"attackTime"`
	AttackRange                  int64      `json:"attackRange"`
	Accuracy                     float64    `json:"accuracy"`
	DamageFixups                 []float64  `json:"damageFixups,omitempty"`
	WeaponType1                  *string    `json:"weaponType1,omitempty"`
	WeaponType2                  *string    `json:"weaponType2,omitempty"`
	Limit                        string     `json:"limit,omitempty"`
	Hostile                      string     `json:"hostile,omitempty"` // raw template text
	SkillList                    []string   `json:"skillList"`
	ModList                      []ModEntry `json:"modList"`
}

// ModEntry is one minion mod: structured mods (modparser codec) plus the
// dat provenance the render test's comment text needs. An entry with no
// mods is an unmapped stat (rendered as a comment); Extra marks the
// hand-written template mods (rendered from the archive template).
type ModEntry struct {
	Mods      json.RawMessage `json:"mods,omitempty"`
	Entry     string          `json:"entry,omitempty"`
	Stat      string          `json:"stat,omitempty"`
	StatValue *float64        `json:"statValue,omitempty"`
	Extra     bool            `json:"extra,omitempty"`
}
