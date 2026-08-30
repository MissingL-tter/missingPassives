// data.minions and data.spectres, from the minions document. The document's
// mod lines (skillStatMap-derived constructors and template extras) are
// evaluated by modexpr into structured mods.

package data

import (
	"fmt"
	"strconv"

	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/internal/util"
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
	// Hostile marks an enemy minion; HostileScale is the template's numeric
	// form of the key (no shipped minion uses it; canon renders it as the
	// number).
	Hostile      bool `lua:"hostile,omitempty"`
	HostileScale util.Opt[float64]
	SkillList    []string         `lua:"skillList"`
	ModList      []*modparser.Mod `lua:"modList"`
}

func loadMinionDef(m schema.MinionDef) (*Minion, error) {
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
	switch m.Hostile {
	case "", "false":
	case "true":
		out.Hostile = true
	default:
		n, err := strconv.ParseFloat(m.Hostile, 64)
		if err != nil {
			return nil, fmt.Errorf("data: minion %s: bad hostile value %q", m.Key, m.Hostile)
		}
		out.Hostile = true // any number is truthy
		out.HostileScale = util.Some(n)
	}
	out.ModList = []*modparser.Mod{}
	for _, entry := range m.ModList {
		if len(entry.Mods) == 0 {
			continue // unmapped stat: a comment in the reference file
		}
		out.ModList = append(out.ModList, modparser.DecodeMods(entry.Mods)...)
	}
	return out, nil
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

func loadMinions(src schema.Minions) error {
	load := func(defs []schema.MinionDef) (map[string]*Minion, error) {
		out := map[string]*Minion{}
		for _, m := range defs {
			if m.Skip {
				continue
			}
			def, err := loadMinionDef(m)
			if err != nil {
				return nil, err
			}
			out[m.Key] = def
		}
		return out, nil
	}
	// Data.lua loads Data/Minions into data.minions and Data/Spectres into
	// data.spectres, then merges spectres into minions with the spectre
	// limit applied.
	var err error
	if Minions, err = load(src.Minions); err != nil {
		return err
	}
	for name, m := range handMinionsTable() {
		Minions[name] = m
	}
	if Spectres, err = load(src.Spectres); err != nil {
		return err
	}
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
	return nil
}
