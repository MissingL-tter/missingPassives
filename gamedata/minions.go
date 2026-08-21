package gamedata

// Minions holds the minion/spectre definitions, one list per template file,
// in template emit order.
type Minions struct {
	Spectres []MinionDef `json:"spectres"`
	Minions  []MinionDef `json:"minions"`
}

type MinionDef struct {
	// Skip marks an emit whose monster variety id was not found; it emits
	// nothing but keeps the template's emit sequence aligned.
	Skip bool `json:"skip,omitempty"`

	Key                          string    `json:"key"` // the minions[...] table key
	Name                         string    `json:"name"`
	MonsterTags                  []string  `json:"monsterTags"`
	BaseDamageIgnoresAttackSpeed bool      `json:"baseDamageIgnoresAttackSpeed,omitempty"`
	Life                         float64   `json:"life"`
	LifeScaling                  []string  `json:"lifeScaling,omitempty"` // AltLife1/AltLife2 lines
	EnergyShield                 *float64  `json:"energyShield,omitempty"`
	Armour                       *float64  `json:"armour,omitempty"`
	Evasion                      *float64  `json:"evasion,omitempty"`
	FireResist                   int64     `json:"fireResist"`
	ColdResist                   int64     `json:"coldResist"`
	LightningResist              int64     `json:"lightningResist"`
	ChaosResist                  int64     `json:"chaosResist"`
	Damage                       float64   `json:"damage"`
	DamageSpread                 float64   `json:"damageSpread"`
	AttackTime                   float64   `json:"attackTime"`
	AttackRange                  int64     `json:"attackRange"`
	Accuracy                     float64   `json:"accuracy"`
	DamageFixups                 []float64 `json:"damageFixups,omitempty"`
	WeaponType1                  *string   `json:"weaponType1,omitempty"`
	WeaponType2                  *string   `json:"weaponType2,omitempty"`
	Limit                        string    `json:"limit,omitempty"`
	Hostile                      string    `json:"hostile,omitempty"` // raw template text
	SkillList                    []string  `json:"skillList"`
	// ModList entries are the emitted mod-constructor / comment lines. A
	// structured model for these lands with the minion module.
	ModList []string `json:"modList"`
}
