// data.minions and data.spectres, from the minions document. The document's
// mod lines (skillStatMap-derived constructors and template extras) are
// evaluated by modexpr into structured mods.

package data

import (
	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// Minion is one data.minions entry.
type Minion struct {
	Name                         string   `lua:"name"`
	MonsterTags                  []string `lua:"monsterTags"`
	BaseDamageIgnoresAttackSpeed bool     `lua:"baseDamageIgnoresAttackSpeed,omitempty"`
	Life                         float64  `lua:"life"`
	LifeScaling                  string   `lua:"lifeScaling,omitempty"`
	EnergyShield                 *float64 `lua:"energyShield"`
	Armour                       *float64 `lua:"armour"`
	Evasion                      *float64 `lua:"evasion"`
	FireResist                   float64  `lua:"fireResist"`
	ColdResist                   float64  `lua:"coldResist"`
	LightningResist              float64  `lua:"lightningResist"`
	ChaosResist                  float64  `lua:"chaosResist"`
	Damage                       float64  `lua:"damage"`
	DamageSpread                 float64  `lua:"damageSpread"`
	AttackTime                   float64  `lua:"attackTime"`
	AttackRange                  float64  `lua:"attackRange"`
	Accuracy                     float64  `lua:"accuracy"`
	DamageFixup                  *float64 `lua:"damageFixup"`
	WeaponType1                  *string  `lua:"weaponType1"`
	WeaponType2                  *string  `lua:"weaponType2"`
	Limit                        string   `lua:"limit,omitempty"`
	// Hostile marks an enemy minion: it picks the monster damage and life
	// tables over the ally ones, and keeps the player's auras off it.
	Hostile   bool             `lua:"hostile,omitempty"`
	SkillList []string         `lua:"skillList"`
	ModList   []*modparser.Mod `lua:"modList"`
}

func loadMinionDef(m schema.MinionDef) *Minion {
	out := &Minion{
		Name:                         m.Name,
		MonsterTags:                  emptyIfNil(m.MonsterTags),
		BaseDamageIgnoresAttackSpeed: m.BaseDamageIgnoresAttackSpeed,
		Life:                         m.Life,
		EnergyShield:                 m.EnergyShield,
		Armour:                       m.Armour,
		Evasion:                      m.Evasion,
		FireResist:                   float64(m.FireResist),
		ColdResist:                   float64(m.ColdResist),
		LightningResist:              float64(m.LightningResist),
		ChaosResist:                  float64(m.ChaosResist),
		Damage:                       m.Damage,
		DamageSpread:                 m.DamageSpread,
		AttackTime:                   m.AttackTime,
		AttackRange:                  float64(m.AttackRange),
		Accuracy:                     m.Accuracy,
		WeaponType1:                  m.WeaponType1,
		WeaponType2:                  m.WeaponType2,
		Limit:                        m.Limit,
		SkillList:                    emptyIfNil(m.SkillList),
	}
	// Duplicate constructor keys keep the FIRST value under LuaJIT
	// (#EVAL: see Misc.EnergyShieldRechargeBase).
	if len(m.LifeScaling) > 0 {
		out.LifeScaling = m.LifeScaling[0]
	}
	if len(m.DamageFixups) > 0 {
		out.DamageFixup = &m.DamageFixups[0]
	}
	out.Hostile = m.Hostile
	out.ModList = []*modparser.Mod{}
	for _, entry := range m.ModList {
		if len(entry.Mods) == 0 {
			continue // unmapped stat: a comment in the reference file
		}
		out.ModList = append(out.ModList, modparser.DecodeMods(entry.Mods)...)
	}
	return out
}

// handMinionsTable ports the minion blocks the Minions template hand-writes
// as passthrough (currently just the combined relic guardian).
func handMinionsTable() map[string]*Minion {
	return map[string]*Minion{
		"GuardianRelicAll": {
			Name:            "All Relics",
			Life:            4,
			EnergyShield:    num(0.6),
			FireResist:      40,
			ColdResist:      40,
			LightningResist: 40,
			ChaosResist:     20,
			Damage:          1,
			DamageSpread:    0,
			AttackTime:      1,
			AttackRange:     6,
			Accuracy:        1,
			SkillList:       []string{"RelicTeleport", "Anger", "Hatred", "Wrath"},
			ModList:         []*modparser.Mod{},
		},
	}
}

func loadMinions(src schema.Minions) {
	load := func(defs []schema.MinionDef) map[string]*Minion {
		out := map[string]*Minion{}
		for _, m := range defs {
			if m.Skip {
				continue
			}
			out[m.Key] = loadMinionDef(m)
		}
		return out
	}
	// Data.lua loads Data/Minions into data.minions and Data/Spectres into
	// data.spectres, then merges spectres into minions with the spectre
	// limit applied.
	Minions = load(src.Minions)
	for name, m := range handMinionsTable() {
		Minions[name] = m
	}
	Spectres = load(src.Spectres)
	for name, spectre := range Spectres {
		spectre.Limit = "ActiveSpectreLimit"
		Minions[name] = spectre
	}
	for _, minion := range Minions {
		for _, mod := range minion.ModList {
			mod.Source = "Minion:" + minion.Name
			mod.SourceSet = true
		}
	}
}
