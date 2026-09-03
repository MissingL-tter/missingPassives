package config

import (
	"math"
	"sort"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// The bodies the mechanical conversion could not express: one with a
// stored-value compatibility branch, the map-affix dropdowns (which
// dispatch into the map-modifier data's own appliers), and the boss-skill
// preset, which rewrites most of the enemy stat block.

// applyConditionStationary ports ConfigOptions.lua L158. Older saves
// stored this condition as a boolean; those read as one second or none.
func applyConditionStationary(v Value, t *Tab) {
	var n float64
	if b, ok := v.(Bool); ok {
		if b {
			n = 1
		}
	} else {
		n, _ = NumOf(v)
	}
	n = math.Max(0, n)
	t.mod("Multiplier:StationarySeconds", modparser.Base, modparser.Num(n))
	if n > 0 {
		t.mod("Condition:Stationary", modparser.Flag, modparser.Bool(true))
	}
}

// applyMapAffix ports mapAffixDropDownFunction (ConfigOptions.lua L118):
// each of the eight map prefix/suffix dropdowns hands its selection to the
// affix's own applier in the map-modifier data.
//
// The check-type call passes `var`, an undeclared global, so those
// appliers receive nil. None of them reads it.
func applyMapAffix(v Value, t *Tab) {
	name, _ := StrOf(v)
	if name == "NONE" {
		return
	}
	affix := data.MapMods.AffixData[name]
	if affix == nil || affix.Apply == data.MapModApplyNone {
		return
	}
	apply := mapAffixApplies[name]
	if apply == nil {
		return
	}
	effect := 1 + t.InputNum("multiplierMapModEffect")/100
	// The tier dropdown lists red/yellow/white, and the appliers index
	// their value tables low to high, so the selection inverts.
	tier := 4 - t.SelIndex("multiplierMapModTier")
	if t.SelIndex("multiplierMapModTier") == 0 {
		tier = 3
	}
	apply(mapAffixArgs{Tier: tier, Effect: effect, RollRange: 100, Values: affix.Values}, t)
}

// mapAffixArgs is what one map-modifier applier is called with: the
// selected tier, the configured effect multiplier, the roll range (always
// the maximum here) and the affix's own value table.
type mapAffixArgs struct {
	Tier      int
	Effect    float64
	RollRange float64
	Values    []data.MapModValue
}

// applyPresetBossSkills ports ConfigOptions.lua L2251: a named boss skill
// fills in the enemy's damage, penetration, speed and crit placeholders,
// and locks the damage-type selector while it is set.
func applyPresetBossSkills(v Value, t *Tab) {
	name, _ := StrOf(v)
	if name == "None" {
		// Releasing the lock restores the averaged damage type.
		if !t.Enabled("enemyDamageType") {
			t.Input["enemyDamageType"] = Str("Average")
		}
		t.SetEnabled("enemyDamageType", true)
		return
	}
	boss, ok := data.BossSkills[name]
	if !ok {
		return
	}
	preset, _ := StrOf(t.Input["enemyIsBoss"])
	isUber := preset == "Uber" || (boss.EarlierUber && preset == "Pinnacle")

	for _, dmg := range []Var{"enemyPhysicalDamage", "enemyLightningDamage",
		"enemyColdDamage", "enemyFireDamage", "enemyChaosDamage"} {
		t.ClearPlaceholder(dmg)
	}

	rollRange := t.InputNum("enemyDamageRollRange")
	if !Truthy(t.Input["enemyDamageRollRange"]) {
		rollRange = t.PlaceholderNum("enemyDamageRollRange")
	}
	rollRange = math.Min(math.Max(rollRange, 0), 100)

	base := monsterDamage(t.EnemyLevel)
	for _, damageType := range sortedKeys(boss.DamageMultipliers) {
		mult := boss.DamageMultipliers[damageType]
		if len(mult) < 2 {
			continue
		}
		value := base * (mult[0] + rollRange*mult[1])
		if isUber && boss.UberDamageMultiplier != nil {
			value *= *boss.UberDamageMultiplier
		}
		t.SetPlaceholder(Var("enemy"+damageType+"Damage"), round(value))
	}

	for _, pen := range []Var{"enemyPhysicalOverwhelm", "enemyLightningPen",
		"enemyColdPen", "enemyFirePen"} {
		t.ClearPlaceholder(pen)
	}
	for _, penType := range sortedKeys(boss.DamagePenetrations) {
		value := boss.DamagePenetrations[penType]
		if isUber {
			if uber, ok := boss.UberDamagePenetrations[penType]; ok && uber.Set {
				value = uber
			}
		}
		if value.Set {
			t.SetPlaceholder(Var("enemy"+penType), value.V)
		}
	}

	if boss.DamageType != "" {
		t.SelByValue("enemyDamageType", Str(boss.DamageType))
		t.Input["enemyDamageType"] = Str(boss.DamageType)
	}
	t.SetEnabled("enemyDamageType", false)

	switch {
	case isUber && boss.UberSpeed != nil:
		t.SetPlaceholder("enemySpeed", *boss.UberSpeed)
	case boss.Speed != nil:
		t.SetPlaceholder("enemySpeed", *boss.Speed)
	}
	if boss.CritChance != nil {
		t.SetPlaceholder("enemyCritChance", *boss.CritChance)
	}

	t.mod("BossSkillActive", modparser.Flag, modparser.Bool(true))

	if name == "Atziri Flameblast" && isUber {
		t.enemyModSrc("Damage", modparser.Inc, modparser.Num(60), "Alluring Abyss Map Mod")
	}
	if boss.AdditionalStats != nil {
		stats := boss.AdditionalStats.Base
		if isUber && boss.AdditionalStats.Uber != nil {
			stats = boss.AdditionalStats.Uber
		}
		for _, k := range sortedKeys(stats) {
			stat := stats[k]
			if stat.Flag {
				t.enemyModSrc(k, modparser.Flag, modparser.Bool(true), "BossSkillAdditionalData")
			} else {
				t.enemyModSrc(k, modparser.Base, modparser.Num(stat.Value), "BossSkillAdditionalData")
			}
		}
	}
}

// sortedKeys walks a map in ascending key order. The reference iterates
// with pairs(), and its dumps were taken with pairs() replaced by a sorted
// walk, so sorted order is what the archive recorded.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Five map affixes carry an apply body the reference left empty - three
// of them keep their previous modifier commented out above nothing, and
// one asks "SHOULD THIS BE SUPPORTED?". Registering them keeps the
// dispatch total and records that doing nothing is the reference's
// behaviour, not a gap here.
func init() {
	for _, name := range []string{"Punishing", "Mirrored", "of Balance", "of Transience", "of Blinding"} {
		if mapAffixApplies[name] != nil {
			panic("config: " + name + " has an applier; the reference's is empty")
		}
		mapAffixApplies[name] = func(mapAffixArgs, *Tab) {}
	}
}
