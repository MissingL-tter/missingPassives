// The per-kind data an item carries after BuildModList (the reference's
// weaponData/armourData/flaskData/tinctureData/jewelData tables), granted
// skills, requirements and affix ranges. Fixed fields are the keys the
// reference always computes; WeaponData.Extra keeps whatever data-driven
// LIST modifiers (WeaponData {key, value}) add beyond them.
package item

import (
	"fmt"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// DamageRange is one damage type's weapon roll and the DPS it yields.
type DamageRange struct{ Min, Max, DPS float64 }

// WeaponData is one weapon slot's computed stats (weaponData[slotNum]).
// The Opt fields exist only on item-built data: minion and unarmed weapon
// tables never carry them, and CritChance is present-but-zero unarmed.
type WeaponData struct {
	Type, SubType, Name string
	AttackRate, Range   float64
	AttackSpeedInc      util.Opt[float64]
	RangeBonus          util.Opt[float64]
	CritChance          util.Opt[float64]
	TotalDPS            util.Opt[float64]
	ElementalDPS        float64

	Physical, Lightning, Cold, Fire, Chaos DamageRange

	CountsAsAll1H        bool
	CountsAsDualWielding bool
	AddedUsing           AddedUsingFlags

	Extra map[string]modparser.Value
}

// AddedUsingFlags is the AddedUsing<Type> block CalcPerform.lua L156-164
// writes onto a counts-as-all-one-handed weapon: per one-handed weapon
// type, whether the Using<Type> condition was absent before this weapon
// added it. Written records that the block exists, because the reference
// reads the entries through a presence test (CalcMods.lua only rejects a
// negated Using<Type> tag when the key is there), and an absent key is not
// a false one.
type AddedUsingFlags struct {
	Written                        bool
	Axe, Sword, Dagger, Mace, Claw bool
}

// Added is weaponData["AddedUsing"..weaponType]: the flag and whether the
// block holds it at all.
func (f AddedUsingFlags) Added(weaponType string) (added, present bool) {
	if !f.Written {
		return false, false
	}
	switch weaponType {
	case "Axe":
		return f.Axe, true
	case "Sword":
		return f.Sword, true
	case "Dagger":
		return f.Dagger, true
	case "Mace":
		return f.Mace, true
	case "Claw":
		return f.Claw, true
	}
	return false, false
}

// Damage returns the range for a damage type name, nil for an unknown one.
func (w *WeaponData) Damage(dmgType string) *DamageRange {
	switch dmgType {
	case "Physical":
		return &w.Physical
	case "Lightning":
		return &w.Lightning
	case "Cold":
		return &w.Cold
	case "Fire":
		return &w.Fire
	case "Chaos":
		return &w.Chaos
	}
	return nil
}

// Clone is copyTable(weaponData): the map is copied, not shared.
func (w *WeaponData) Clone() *WeaponData {
	cp := *w
	cp.Extra = cloneMap(w.Extra)
	return &cp
}

func cloneMap[V any](m map[string]V) map[string]V {
	if m == nil {
		return nil
	}
	out := make(map[string]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Set applies a WeaponData {key, value} entry; a nil value clears the key.
func (w *WeaponData) Set(key string, v modparser.Value) {
	if dmgType, side, ok := damageKey(key); ok {
		r := w.Damage(dmgType)
		switch side {
		case "Min":
			r.Min = num(v)
		case "Max":
			r.Max = num(v)
		case "DPS":
			r.DPS = num(v)
		}
		return
	}
	switch key {
	case "type":
		w.Type = str(v)
	case "subType":
		w.SubType = str(v)
	case "name":
		w.Name = str(v)
	case "AttackRate":
		w.AttackRate = num(v)
	case "range":
		w.Range = num(v)
	case "AttackSpeedInc":
		w.AttackSpeedInc = optNum(v)
	case "rangeBonus":
		w.RangeBonus = optNum(v)
	case "CritChance":
		w.CritChance = optNum(v)
	case "TotalDPS":
		w.TotalDPS = optNum(v)
	case "ElementalDPS":
		w.ElementalDPS = num(v)
	case "countsAsAll1H":
		w.CountsAsAll1H = modparser.Truthy(v)
	case "countsAsDualWielding":
		w.CountsAsDualWielding = modparser.Truthy(v)
	default:
		w.Extra = setExtra(w.Extra, key, v)
	}
}

// damageKey splits "<Type>Min"/"<Type>Max"/"<Type>DPS".
func damageKey(key string) (dmgType, side string, ok bool) {
	for _, t := range dmgTypeList {
		if rest, found := strings.CutPrefix(key, t); found {
			switch rest {
			case "Min", "Max", "DPS":
				return t, rest, true
			}
		}
	}
	return "", "", false
}

// DefenceStat is one armour stat with its base-roll percentile.
type DefenceStat struct{ Value, BasePercentile util.Opt[float64] }

// ArmourData is armour-piece state: parsed from the item text before
// BuildModList recomputes it, so every stat is optional.
type ArmourData struct {
	Armour, Evasion, EnergyShield, Ward DefenceStat
	BlockChance                         util.Opt[float64]
}

// Defence returns the stat by reference name ("Armour", "Evasion",
// "EnergyShield", "Ward"), nil otherwise.
func (a *ArmourData) Defence(name string) *DefenceStat {
	switch name {
	case "Armour":
		return &a.Armour
	case "Evasion":
		return &a.Evasion
	case "EnergyShield":
		return &a.EnergyShield
	case "Ward":
		return &a.Ward
	}
	return nil
}

// Set applies an ArmourData {key, value} entry; a nil value clears the key.
// Any other key is dropped: the reference stores it on armourData (Item.lua
// L719 writes every "BasePercentile"-containing spec name) and never reads
// it back — the percentile readers and the raw-line writer both name the
// four defence types.
func (a *ArmourData) Set(key string, v modparser.Value) {
	if stat := a.Defence(strings.TrimSuffix(key, "BasePercentile")); stat != nil {
		if strings.HasSuffix(key, "BasePercentile") {
			stat.BasePercentile = optNum(v)
		} else {
			stat.Value = optNum(v)
		}
		return
	}
	if key == "BlockChance" {
		a.BlockChance = optNum(v)
	}
}

// FlaskRecovery is one recovery pool's flask numbers (life*/mana* keys).
type FlaskRecovery struct {
	Base, Instant, Gradual, Total float64
	Additional                    util.Opt[float64] // life only
	EffectNotRemoved              bool
}

// FlaskData is a flask's computed state. InstantPerc/InstantLowLifePerc
// and the pools exist only on recovery flasks.
type FlaskData struct {
	Duration, ChargesMax, ChargesUsed, GainMod, EffectInc float64
	InstantPerc, InstantLowLifePerc                       util.Opt[float64]
	Life, Mana                                            *FlaskRecovery
}

// Pool returns the "Life"/"Mana" recovery pool, nil when absent.
func (f *FlaskData) Pool(name string) *FlaskRecovery {
	switch name {
	case "Life":
		return f.Life
	case "Mana":
		return f.Mana
	}
	return nil
}

// Set applies a FlaskData {key, value} entry; a nil value clears the key.
// Any other key is dropped: no FlaskData LIST modifier exists to produce
// one, and the reference would only store it unread.
func (f *FlaskData) Set(key string, v modparser.Value) {
	switch key {
	case "duration":
		f.Duration = num(v)
	case "chargesMax":
		f.ChargesMax = num(v)
	case "chargesUsed":
		f.ChargesUsed = num(v)
	case "gainMod":
		f.GainMod = num(v)
	case "effectInc":
		f.EffectInc = num(v)
	case "instantPerc":
		f.InstantPerc = optNum(v)
	case "instantLowLifePerc":
		f.InstantLowLifePerc = optNum(v)
	default:
		for _, pool := range []struct {
			prefix string
			slot   **FlaskRecovery
		}{{"life", &f.Life}, {"mana", &f.Mana}} {
			rest, ok := strings.CutPrefix(key, pool.prefix)
			if !ok {
				continue
			}
			if *pool.slot == nil {
				*pool.slot = &FlaskRecovery{}
			}
			rec := *pool.slot
			switch rest {
			case "Base":
				rec.Base = num(v)
			case "Instant":
				rec.Instant = num(v)
			case "Gradual":
				rec.Gradual = num(v)
			case "Total":
				rec.Total = num(v)
			case "Additional":
				rec.Additional = optNum(v)
			case "EffectNotRemoved":
				rec.EffectNotRemoved = modparser.Truthy(v)
			}
			return
		}
	}
}

// TinctureData is a tincture's computed state.
type TinctureData struct {
	ManaBurn, CooldownInc, Cooldown, EffectInc float64
}

// Set applies a TinctureData {key, value} entry. Any other key is dropped:
// no TinctureData LIST modifier exists to produce one, and the reference
// would only store it unread.
func (t *TinctureData) Set(key string, v modparser.Value) {
	switch key {
	case "manaBurn":
		t.ManaBurn = num(v)
	case "cooldownInc":
		t.CooldownInc = num(v)
	case "cooldown":
		t.Cooldown = num(v)
	case "effectInc":
		t.EffectInc = num(v)
	}
}

// JewelData is a jewel's parsed behaviour: radius/conquering/leap flags
// from the mod lines, the cluster-jewel layout, and the radius functions.
// Counts are 0 when absent (the reference never stores a zero count).
type JewelData struct {
	RadiusIndex int
	ConqueredBy *modparser.ConqueredBy
	FuncList    []modparser.JewelFn

	IntuitiveLeapLike, IntuitiveLeapKeystoneOnly bool
	ImpossibleEscapeKeystone                     string // the keystone named on the jewel
	ImpossibleEscapeKeystones                    map[string]bool
	JewelIncEffectFromClassStart                 float64
	CorruptedMagicJewelIncEffect                 float64
	CorruptedRareJewelIncEffect                  float64
	// LimitDisabled is set by the items tab when the jewel exceeds its
	// limit (never on the parse path).
	LimitDisabled bool

	ClusterJewelKeystone, ClusterJewelSkill string
	ClusterJewelNodeCount                   int
	ClusterJewelSocketCount                 int
	ClusterJewelSocketCountOverride         int
	ClusterJewelNothingnessCount            int
	ClusterJewelIncEffect                   float64
	ClusterJewelSmallsAreNothingness        bool
	ClusterJewelNotables                    []string
	ClusterJewelAddedMods                   []string
	// ClusterJewelValid: the jewel describes a buildable subgraph (a
	// keystone, a skill/nothingness layout with a node count, or a
	// socket-count override with a nothingness count).
	ClusterJewelValid bool
}

// Set applies a JewelData {key, value} entry; a nil value clears the key.
// Any other key is dropped: the JewelData LIST modifiers all name a field
// above, and the reference would only store an off-list key unread.
func (j *JewelData) Set(key string, v modparser.Value) {
	switch key {
	case "radiusIndex":
		j.RadiusIndex = int(num(v))
	case "conqueredBy":
		if cq, ok := v.(modparser.ConqueredBy); ok {
			j.ConqueredBy = &cq
		} else {
			j.ConqueredBy = nil
		}
	case "intuitiveLeapLike":
		j.IntuitiveLeapLike = modparser.Truthy(v)
	case "intuitiveLeapKeystoneOnly":
		j.IntuitiveLeapKeystoneOnly = modparser.Truthy(v)
	case "impossibleEscapeKeystone":
		j.ImpossibleEscapeKeystone = str(v)
	case "jewelIncEffectFromClassStart":
		j.JewelIncEffectFromClassStart = num(v)
	case "corruptedMagicJewelIncEffect":
		j.CorruptedMagicJewelIncEffect = num(v)
	case "corruptedRareJewelIncEffect":
		j.CorruptedRareJewelIncEffect = num(v)
	case "limitDisabled":
		j.LimitDisabled = modparser.Truthy(v)
	case "clusterJewelKeystone":
		j.ClusterJewelKeystone = str(v)
	case "clusterJewelSkill":
		j.ClusterJewelSkill = str(v)
	case "clusterJewelNodeCount":
		j.ClusterJewelNodeCount = int(num(v))
	case "clusterJewelSocketCount":
		j.ClusterJewelSocketCount = int(num(v))
	case "clusterJewelSocketCountOverride":
		j.ClusterJewelSocketCountOverride = int(num(v))
	case "clusterJewelNothingnessCount":
		j.ClusterJewelNothingnessCount = int(num(v))
	case "clusterJewelIncEffect":
		j.ClusterJewelIncEffect = num(v)
	case "clusterJewelSmallsAreNothingness":
		j.ClusterJewelSmallsAreNothingness = modparser.Truthy(v)
	case "clusterJewelValid":
		j.ClusterJewelValid = modparser.Truthy(v)
	}
}

// GrantedSkill is one grantedSkills entry (an ExtraSkill modifier).
type GrantedSkill struct {
	SkillID, Source      string
	Level, TriggerChance util.Opt[float64]
	NoSupports           bool
	Triggered            bool
}

// Requirements is the item's attribute/level requirement table:
// Str/Dex/Int are always present once parsed, Level when known, the *Mod
// values once BuildModList applies requirement modifiers.
type Requirements struct {
	Str, Dex, Int, Level, StrMod, DexMod, IntMod util.Opt[float64]
}

// Attribute returns the requirement by its lower-case key ("str", "dex",
// "int", "level"), nil otherwise.
func (r *Requirements) Attribute(key string) *util.Opt[float64] {
	switch key {
	case "str":
		return &r.Str
	case "dex":
		return &r.Dex
	case "int":
		return &r.Int
	case "level":
		return &r.Level
	}
	return nil
}

// AffixRange is a crafted affix's roll: a single fraction or one per
// value.
type AffixRange struct {
	Single util.Opt[float64]
	Multi  []float64
}

// num, optNum and str read a Set value's scalar. A nil value is the Lua
// nil assignment (clears the key: zero / unset / ""); any other kind is a
// shape no producer emits, and panics rather than silently writing a zero
// (lua-residue.md T2).
func num(v modparser.Value) float64 {
	switch t := v.(type) {
	case nil:
		return 0
	case modparser.Num:
		return float64(t)
	}
	panic(fmt.Sprintf("item: %T where the reference stores a number", v))
}

func optNum(v modparser.Value) util.Opt[float64] {
	switch t := v.(type) {
	case nil:
		return util.Opt[float64]{}
	case modparser.Num:
		return util.Some(float64(t))
	}
	panic(fmt.Sprintf("item: %T where the reference stores a number", v))
}

func str(v modparser.Value) string {
	switch t := v.(type) {
	case nil:
		return ""
	case modparser.Str:
		return string(t)
	}
	panic(fmt.Sprintf("item: %T where the reference stores text", v))
}

func setExtra(m map[string]modparser.Value, key string, v modparser.Value) map[string]modparser.Value {
	if v == nil {
		delete(m, key)
		return m
	}
	if m == nil {
		m = map[string]modparser.Value{}
	}
	m[key] = v
	return m
}
