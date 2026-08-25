// data.minions and data.spectres, from the minions document. The document's
// mod lines (skillStatMap-derived constructors and template extras) are
// evaluated by modexpr into structured mods.

package data

import (
	"strconv"

	"github.com/MissingL-tter/missingPassives/gamedata"
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
	Hostile                      any      `lua:"hostile"`
	SkillList                    []string `lua:"skillList"`
	ModList                      []any    `lua:"modList"` // *modparser.Mod (or *D for flag-slot typos)
}

func loadMinionDef(m gamedata.MinionDef) *Minion {
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
	if m.Hostile != "" {
		switch m.Hostile {
		case "true":
			out.Hostile = true
		case "false":
			out.Hostile = false
		default:
			n, err := strconv.ParseFloat(m.Hostile, 64)
			if err != nil {
				panic("data: unhandled hostile value " + m.Hostile)
			}
			out.Hostile = n
		}
	}
	out.ModList = []any{}
	for _, line := range m.ModList {
		if mod, ok := evalModLine(line); ok {
			out.ModList = append(out.ModList, mod)
		}
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
			ModList:         []any{},
		},
	}
}

func loadMinions(src gamedata.Minions) {
	load := func(defs []gamedata.MinionDef) map[string]*Minion {
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
		for _, m := range minion.ModList {
			switch mod := m.(type) {
			case *modparser.Mod:
				mod.Source = "Minion:" + minion.Name
				mod.SourceSet = true
			case *modparser.D:
				if mod.KV == nil {
					mod.KV = map[string]any{}
				}
				mod.KV["source"] = "Minion:" + minion.Name
			}
		}
	}
}

// ModCanon converts a structured skill/minion mod into the plain table
// shape the archive canon uses (registered as a luacanon adapter by the
// game-data test).
func ModCanon(m *modparser.Mod) map[string]any {
	out := map[string]any{
		"name":         m.Name,
		"type":         m.Type,
		"flags":        m.Flags,
		"keywordFlags": m.KeywordFlags,
	}
	if m.Value != nil {
		out["value"] = m.Value
	}
	if m.SourceSet {
		out["source"] = m.Source
	}
	if m.SourceSlot != "" {
		out["sourceSlot"] = m.SourceSlot
	}
	// ReplaceMod/ConvertMod stamp these on the mod table, so the archive
	// canon carries them; without them the calc differential is blind to
	// that bookkeeping. Matches modparser's writeCanonMod.
	if m.Replaced {
		out["replaced"] = true
	}
	if m.Converted {
		out["converted"] = true
	}
	for i, tag := range m.Tags {
		if tag != nil {
			out[strconv.Itoa(i+1)] = tag
		}
	}
	return out
}
