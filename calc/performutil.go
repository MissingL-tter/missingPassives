// CalcPerform.lua L30-612: the per-actor helper functions of the perform
// stage (mergeBuff, life/mana, attributes/conditions, reservations, curse
// priority, enemy modifier application). Breakdown output (CALCS-mode UI)
// is skipped throughout.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// performActor bundles the per-actor state perform manipulates (the Lua
// actor tables: env.player / env.minion / env.enemy).
type performActor struct {
	ms     *modstore.Actor
	db     *modstore.DB
	output modstore.Output
	// output.MainHand / output.OffHand: the per-weapon pass tables an
	// attack's offence leaves behind (nil until then)
	mainHand, offHand modstore.Output

	mainSkill *ActiveSkill
	skills    []*ActiveSkill
	minion    *Minion // set when this actor is the main skill's minion

	enemy  *performActor
	parent *performActor

	reservedLifeBase, reservedLifePercent float64
	reservedManaBase, reservedManaPercent float64
	uncancellableLife, uncancellableMana  float64
	strDmgBonus                           float64
	appliedEnemyModifiers                 map[*modparser.Mod]bool

	// offence-stage weapon reach (actor.weaponRange1/2, set in the
	// skill-type-stats section)
	weaponRange1, weaponRange2 float64

	// EHP-stage shift tables (actor.damageShiftTable in the reference)
	damageShiftTable         map[string]map[string]float64
	damageOverTimeShiftTable map[string]map[string]float64
}

// takeOutput is `env.player.output = newEnv.player.output`: the actor
// adopts another actor's output table, weapon-pass tables included.
func (pa *performActor) takeOutput(from *performActor) {
	pa.output = from.output
	pa.mainHand, pa.offHand = from.mainHand, from.offHand
}

// floorDec is Common.lua's floor(val, dec) — note the 0.0001 it adds before
// flooring, which is there to stop float error from dropping a value a whole
// unit at the requested precision (a More() product of 1.331 can land at
// 1.3309999999999997, which would floor to 1.3309 without it).
func floorDec(v float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(v*p+0.0001) / p
}

// mergeBuff ports CalcPerform's mergeBuff: merge an instance of a buff,
// taking the highest value of each modifier.
func mergeBuff(src []*modparser.Mod, destTable map[string]*modstore.List, destKey string) {
	if destTable[destKey] == nil {
		destTable[destKey] = modstore.NewList(nil)
	}
	dest := destTable[destKey]
	for _, mod := range src {
		match := false
		if mod.Type != modparser.List {
			for index, destMod := range dest.Mods {
				if modparser.CompareModParams(mod, destMod) {
					if dv, ok := destMod.Value.(modparser.Num); ok && valueNum(mod.Value) > float64(dv) {
						dest.Mods[index] = mod
					}
					match = true
					break
				}
			}
		}
		if !match {
			dest.Mods = append(dest.Mods, mod)
		}
	}
}

// doActorLifeMana ports CalcPerform's doActorLifeMana.
func (env *Env) doActorLifeMana(actor *performActor) {
	modDB := actor.db
	output := actor.output
	condList := modDB.Conditions

	lowLifePerc := modDB.Sum(modparser.Base, nil, "LowLifePercentage")
	if lowLifePerc > 0 {
		output.SetN("LowLifePercentage", 100.0*lowLifePerc)
	} else {
		output.SetN("LowLifePercentage", 100.0*data.Misc.LowPoolThreshold)
	}
	fullLifePerc := modDB.Sum(modparser.Base, nil, "FullLifePercentage")
	if fullLifePerc > 0 {
		output.SetN("FullLifePercentage", 100.0*fullLifePerc)
	} else {
		output.SetN("FullLifePercentage", 100.0)
	}

	ci := modDB.Flag(nil, "ChaosInoculation")
	// Lua Flag() yields nil when unset, so the key only exists when true
	if ci {
		output.SetFlag("ChaosInoculation", true)
	}
	// Life/mana pools
	if ci {
		output.SetN("Life", 1.0)
		condList.Set("FullLife", true)
	} else {
		base := modDB.Sum(modparser.Base, nil, "Life")
		inc := modDB.Sum(modparser.Inc, nil, "Life")
		more := modDB.More(nil, "Life")
		override, _ := modDB.Override(nil, "Life")
		conv := modDB.Sum(modparser.Base, nil, "LifeConvertToEnergyShield")
		if modparser.Truthy(override) {
			output.SetN("Life", valueNum(override))
		} else {
			output.SetN("Life", math.Max(util.RoundHalfUp(base*(1+inc/100)*more*(1-conv/100), 0), 1))
		}
	}
	manaConv := modDB.Sum(modparser.Base, nil, "ManaConvertToArmour")
	output.SetN("Mana", util.RoundHalfUp(Val(modDB, "Mana", nil)*(1-manaConv/100), 0))
	output.SetN("LowestOfMaximumLifeAndMaximumMana", math.Min(output.N("Life"), output.N("Mana")))
}

// matchAfterSpace is baseName:match("%s(%S+)"): the word after the first
// whitespace.
func matchAfterSpace(s string) string {
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return ""
	}
	rest := s[idx+1:]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		rest = rest[1:]
	}
	end := strings.IndexAny(rest, " \t")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// doActorAttribsConditions ports CalcPerform's doActorAttribsConditions.
func (env *Env) doActorAttribsConditions(actor *performActor) {
	modDB := actor.db
	output := actor.output
	condList := modDB.Conditions
	itemList := actor.ms.ItemList
	weaponData1 := weaponOf(actor.ms.WeaponData1)
	weaponData2 := weaponOf(actor.ms.WeaponData2)

	// Set conditions
	if (itemList["Weapon 2"] != nil && itemList["Weapon 2"].ItemType() == "Shield") ||
		(actor == env.playerPA && env.AegisModList != nil) {
		condList.Set("UsingShield", true)
	} else if itemList["Weapon 2"] == nil {
		condList.Set("OffHandIsEmpty", true)
	}
	// An actor without weapon data (a mirage) still runs this with an
	// empty table: the reference sets "Using" and UsingTwoHandedWeapon.
	applyWeaponConds := func(weaponData *item.WeaponData) {
		info := data.WeaponTypeInfo[weaponType(weaponData)]
		condList.Set("Using"+info.Flag, true)
		if weaponData != nil && weaponData.CountsAsAll1H {
			if weaponData.AddedUsing == nil {
				weaponData.AddedUsing = map[string]bool{}
			}
			weaponData.AddedUsing["Axe"] = !condList.Get("UsingAxe")
			condList.Set("UsingAxe", true)
			// Varunastra is a sword
			weaponData.AddedUsing["Sword"] = strings.Contains(weaponData.Name, "Varunastra") || !condList.Get("UsingSword")
			condList.Set("UsingSword", true)
			weaponData.AddedUsing["Dagger"] = !condList.Get("UsingDagger")
			condList.Set("UsingDagger", true)
			weaponData.AddedUsing["Mace"] = !condList.Get("UsingMace")
			condList.Set("UsingMace", true)
			weaponData.AddedUsing["Claw"] = !condList.Get("UsingClaw")
			condList.Set("UsingClaw", true)
			condList.Set("WieldingDifferentWeaponTypes", true)
		}
		if info.Melee {
			condList.Set("UsingMeleeWeapon", true)
		}
		if info.OneHand {
			condList.Set("UsingOneHandedWeapon", true)
		} else {
			condList.Set("UsingTwoHandedWeapon", true)
		}
	}
	if weaponType(weaponData1) == "None" {
		condList.Set("Unarmed", true)
		if itemList["Weapon 2"] == nil && itemList["Gloves"] == nil {
			condList.Set("Unencumbered", true)
		}
	} else {
		applyWeaponConds(weaponData1)
	}
	for _, slotName := range []string{"Helmet", "Body Armour", "Gloves", "Boots"} {
		if itemList[slotName] != nil {
			condList.Set("Using"+slotName, true)
		}
	}
	for _, graftSlot := range []string{"Graft 1", "Graft 2"} {
		if it, _ := itemList[graftSlot].(*Item); it != nil && it.In.BaseName != nil {
			condList.Set("Using"+matchAfterSpace(*it.In.BaseName), true)
		}
	}
	if weaponType(weaponData2) != "" {
		applyWeaponConds(weaponData2)
	}
	if weaponType(weaponData1) != "" && weaponType(weaponData2) != "" {
		condList.Set("DualWielding", true)
		if (weaponData1.Type == "Claw" || weaponData1.CountsAsAll1H) &&
			(weaponData2.Type == "Claw" || weaponData2.CountsAsAll1H) {
			condList.Set("DualWieldingClaws", true)
		}
		if (weaponData1.Type == "Dagger" || weaponData1.CountsAsAll1H) &&
			(weaponData2.Type == "Dagger" || weaponData2.CountsAsAll1H) {
			condList.Set("DualWieldingDaggers", true)
		}
		getWeaponType := func(weaponData *item.WeaponData) string {
			// GGG treats thrusting 1H swords as 1H swords
			if weaponData.Type == "One Handed Sword" && weaponData.SubType == "Thrusting" {
				return "One Handed Sword"
			}
			info := data.WeaponTypeInfo[weaponData.Type]
			if info.Label != nil {
				return *info.Label
			}
			if weaponData.SubType != "" {
				return weaponData.SubType
			}
			return weaponData.Type
		}
		if getWeaponType(weaponData1) != getWeaponType(weaponData2) {
			info1 := data.WeaponTypeInfo[weaponData1.Type]
			info2 := data.WeaponTypeInfo[weaponData2.Type]
			if info1.OneHand && info2.OneHand {
				condList.Set("WieldingDifferentWeaponTypes", true)
			}
		}
	}
	mainSkill := actor.mainSkill
	if env.ModeCombat {
		if !mainSkill.SkillData.Flag("triggered") && !mainSkill.SkillFlags["trap"] && !mainSkill.SkillFlags["mine"] && !mainSkill.SkillFlags["totem"] {
			if mainSkill.SkillFlags["attack"] {
				condList.Set("AttackedRecently", true)
			} else if mainSkill.SkillFlags["spell"] {
				condList.Set("CastSpellRecently", true)
			}
			if mainSkill.SkillTypes[modparser.SkillTypeMovement] {
				condList.Set("UsedMovementSkillRecently", true)
			}
			if mainSkill.SkillFlags["minion"] && !mainSkill.SkillFlags["permanentMinion"] {
				condList.Set("UsedMinionSkillRecently", true)
			}
			if mainSkill.SkillTypes[modparser.SkillTypeVaal] {
				condList.Set("UsedVaalSkillRecently", true)
			}
			if mainSkill.SkillTypes[modparser.SkillTypeChannel] {
				condList.Set("Channelling", true)
			}
		}
		if mainSkill.SkillFlags["hit"] && !mainSkill.SkillFlags["trap"] && !mainSkill.SkillFlags["mine"] && !mainSkill.SkillFlags["totem"] {
			condList.Set("HitRecently", true)
			if mainSkill.SkillFlags["spell"] {
				condList.Set("HitSpellRecently", true)
			}
		}
		if mainSkill.SkillFlags["totem"] {
			condList.Set("HaveTotem", true)
			condList.Set("SummonedTotemRecently", true)
			if mainSkill.SkillFlags["hit"] {
				condList.Set("TotemsHitRecently", true)
				if mainSkill.SkillFlags["spell"] {
					condList.Set("TotemsSpellHitRecently", true)
				}
			}
		}
		if mainSkill.SkillFlags["mine"] {
			condList.Set("DetonatedMinesRecently", true)
		}
		if mainSkill.SkillFlags["trap"] {
			condList.Set("TriggeredTrapsRecently", true)
		}
		playerMainCfg := env.PlayerMainSkill.SkillCfg
		if modDB.Sum(modparser.Base, nil, "EnemyScorchChance") > 0 ||
			modDB.Flag(nil, "CritAlwaysAltAilments") && !modDB.Flag(playerMainCfg, "NeverCrit") ||
			modDB.Flag(nil, "IgniteCanScorch") {
			condList.Set("CanInflictScorch", true)
		}
		if modDB.Sum(modparser.Base, nil, "EnemyBrittleChance") > 0 ||
			modDB.Flag(nil, "CritAlwaysAltAilments") && !modDB.Flag(playerMainCfg, "NeverCrit") {
			condList.Set("CanInflictBrittle", true)
		}
		if modDB.Sum(modparser.Base, nil, "EnemySapChance") > 0 ||
			modDB.Flag(nil, "CritAlwaysAltAilments") && !modDB.Flag(playerMainCfg, "NeverCrit") {
			condList.Set("CanInflictSap", true)
		}
		// Shrine buffs: before life pool for massive shrine
		shrineEffectMod := 1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf", "ShrineBuffEffect")/100
		flr := func(v float64) float64 { return math.Floor(v) }
		if modDB.Flag(nil, "AccelerationShrine") {
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(flr(50*shrineEffectMod)), "Acceleration Shrine"))
			modDB.AddMod(newModS("ProjectileSpeed", modparser.Inc, modparser.Num(flr(80*shrineEffectMod)), "Acceleration Shrine"))
		}
		if modDB.Flag(nil, "BrutalShrine") {
			modDB.AddMod(newModS("Damage", modparser.Inc, modparser.Num(flr(50*shrineEffectMod)), "Brutal Shrine"))
			modDB.AddMod(newModS("EnemyStunDuration", modparser.Inc, modparser.Num(flr(30*shrineEffectMod)), "Brutal Shrine"))
			modDB.AddMod(newModS("EnemyKnockbackChance", modparser.Inc, modparser.Num(100.0), "Brutal Shrine"))
		}
		if modDB.Flag(nil, "DiamondShrine") {
			modDB.AddMod(newModS("CritChance", modparser.Override, modparser.Num(100.0), "Diamond Shrine"))
		}
		if modDB.Flag(nil, "DivineShrine") {
			modDB.AddMod(newModS("DamageTaken", modparser.More, modparser.Num(-100.0), "Divine Shrine"))
		}
		if modDB.Flag(nil, "EchoingShrine") {
			modDB.AddMod(newModSF("Speed", modparser.More, modparser.Num(flr(100*shrineEffectMod)), "Echoing Shrine", modparser.FlagAttack, modparser.KeywordNone))
			modDB.AddMod(newModSF("Speed", modparser.More, modparser.Num(flr(100*shrineEffectMod)), "Echoing Shrine", modparser.FlagCast, modparser.KeywordNone))
			modDB.AddMod(newModS("RepeatCount", modparser.Base, modparser.Num(flr(1*shrineEffectMod)), "Echoing Shrine"))
		}
		if modDB.Flag(nil, "GloomShrine") {
			modDB.AddMod(newModS("NonChaosDamageGainAsChaos", modparser.Base, modparser.Num(flr(10*shrineEffectMod)), "Gloom Shrine"))
		}
		if modDB.Flag(nil, "GreaterFreezingShrine") {
			modDB.AddMod(newModS("PhysicalDamageGainAsCold", modparser.Base, modparser.Num(flr(30*shrineEffectMod)), "Greater Freezing Shrine"))
		}
		if modDB.Flag(nil, "GreaterShockingShrine") {
			modDB.AddMod(newModS("PhysicalDamageGainAsLightning", modparser.Base, modparser.Num(flr(30*shrineEffectMod)), "Greater Shocking Shrine"))
		}
		if modDB.Flag(nil, "GreaterSkeletalShrine") {
			modDB.AddMod(newModS("PhysicalDamageGainAsChaos", modparser.Base, modparser.Num(flr(30*shrineEffectMod)), "Greater Skeletal Shrine"))
		}
		if modDB.Flag(nil, "ImpenetrableShrine") {
			modDB.AddMod(newModS("Armour", modparser.Inc, modparser.Num(flr(100*shrineEffectMod)), "Impenetrable Shrine"))
			modDB.AddMod(newModS("Evasion", modparser.Inc, modparser.Num(flr(100*shrineEffectMod)), "Impenetrable Shrine"))
			modDB.AddMod(newModS("EnergyShield", modparser.Inc, modparser.Num(flr(100*shrineEffectMod)), "Impenetrable Shrine"))
		}
		if modDB.Flag(nil, "MassiveShrine") {
			modDB.AddMod(newModS("Life", modparser.Inc, modparser.Num(flr(40*shrineEffectMod)), "Massive Shrine"))
			modDB.AddMod(newModS("AreaOfEffect", modparser.Inc, modparser.Num(flr(40*shrineEffectMod)), "Massive Shrine"))
		}
		if modDB.Flag(nil, "ReplenishingShrine") {
			modDB.AddMod(newModS("ManaRegen", modparser.Inc, modparser.Num(200*shrineEffectMod), "Replenishing Shrine"))
			modDB.AddMod(newModS("LifeRegenPercent", modparser.Base, modparser.Num(6.7*shrineEffectMod), "Replenishing Shrine"))
		}
		if modDB.Flag(nil, "ResistanceShrine") {
			modDB.AddMod(newModS("ElementalResist", modparser.Base, modparser.Num(flr(50*shrineEffectMod)), "Resistance Shrine"))
			modDB.AddMod(newModS("ElementalResistMax", modparser.Base, modparser.Num(flr(10*shrineEffectMod)), "Resistance Shrine"))
			modDB.AddMod(newModS("ChaosResistMax", modparser.Base, modparser.Num(flr(10*shrineEffectMod)), "Resistance Shrine"))
		}
		if modDB.Flag(nil, "ResonatingShrine") {
			modDB.AddMod(newModS("PowerChargesMax", modparser.Base, modparser.Num(flr(1*shrineEffectMod)), "Resonating Shrine"))
			modDB.AddMod(newModS("FrenzyChargesMax", modparser.Base, modparser.Num(flr(1*shrineEffectMod)), "Resonating Shrine"))
			modDB.AddMod(newModS("EnduranceChargesMax", modparser.Base, modparser.Num(flr(1*shrineEffectMod)), "Resonating Shrine"))
		}
		if modDB.Flag(nil, "LesserAccelerationShrine") && !modDB.Flag(nil, "AccelerationShrine") {
			modDB.AddMod(newModS("ActionSpeed", modparser.Inc, modparser.Num(flr(10*shrineEffectMod)), "Lesser Acceleration Shrine"))
			modDB.AddMod(newModS("ProjectileSpeed", modparser.Inc, modparser.Num(flr(30*shrineEffectMod)), "Lesser Acceleration Shrine"))
		}
		if modDB.Flag(nil, "LesserBrutalShrine") {
			modDB.AddMod(newModS("Damage", modparser.Inc, modparser.Num(flr(20*shrineEffectMod)), "Lesser Brutal Shrine"))
			modDB.AddMod(newModS("EnemyStunDuration", modparser.Inc, modparser.Num(flr(20*shrineEffectMod)), "Lesser Brutal Shrine"))
			modDB.AddMod(newModS("EnemyKnockbackChance", modparser.Inc, modparser.Num(100.0), "Lesser Brutal Shrine"))
		}
		if modDB.Flag(nil, "LesserImpenetrableShrine") {
			modDB.AddMod(newModS("Armour", modparser.Inc, modparser.Num(flr(50*shrineEffectMod)), "Lesser Impenetrable Shrine"))
			modDB.AddMod(newModS("Evasion", modparser.Inc, modparser.Num(flr(50*shrineEffectMod)), "Lesser Impenetrable Shrine"))
			modDB.AddMod(newModS("EnergyShield", modparser.Inc, modparser.Num(flr(50*shrineEffectMod)), "Lesser Impenetrable Shrine"))
		}
		if modDB.Flag(nil, "LesserMassiveShrine") {
			modDB.AddMod(newModS("Life", modparser.Inc, modparser.Num(flr(20*shrineEffectMod)), "Lesser Massive Shrine"))
			modDB.AddMod(newModS("AreaOfEffect", modparser.Inc, modparser.Num(flr(20*shrineEffectMod)), "Lesser Massive Shrine"))
		}
		if modDB.Flag(nil, "LesserReplenishingShrine") {
			modDB.AddMod(newModS("ManaRegen", modparser.Inc, modparser.Num(100*shrineEffectMod), "Lesser Replenishing Shrine"))
			modDB.AddMod(newModS("LifeRegenPercent", modparser.Base, modparser.Num(3.3*shrineEffectMod), "Lesser Replenishing Shrine"))
		}
		if modDB.Flag(nil, "LesserResistanceShrine") {
			modDB.AddMod(newModS("ElementalResist", modparser.Base, modparser.Num(flr(25*shrineEffectMod)), "Lesser Resistance Shrine"))
			modDB.AddMod(newModS("ElementalResistMax", modparser.Base, modparser.Num(flr(2*shrineEffectMod)), "Lesser Resistance Shrine"))
			modDB.AddMod(newModS("ChaosResistMax", modparser.Base, modparser.Num(flr(2*shrineEffectMod)), "Lesser Resistance Shrine"))
		}
	}
	if env.ModeEffective {
		pm := env.PlayerMainSkill
		for _, el := range []string{"Fire", "Cold", "Lightning"} {
			if pm.SkillModList.Sum(modparser.Base, pm.SkillCfg, el+"ExposureChance") > 0 || modDB.Sum(modparser.Base, nil, el+"ExposureChance") > 0 {
				condList.Set("CanApply"+el+"Exposure", true)
			}
		}
	}

	setAttribConds := func() {
		stats := []float64{output.N("Str"), output.N("Dex"), output.N("Int")}
		sort.Float64s(stats)
		output.SetN("LowestAttribute", stats[0])
		condList.Set("TwoHighestAttributesEqual", stats[1] == stats[2])

		oStr, oDex, oInt := output.N("Str"), output.N("Dex"), output.N("Int")
		condList.Set("DexHigherThanInt", oDex > oInt)
		condList.Set("StrHigherThanInt", oStr > oInt)
		condList.Set("IntHigherThanDex", oInt > oDex)
		condList.Set("StrHigherThanDex", oStr > oDex)
		condList.Set("IntHigherThanStr", oInt > oStr)
		condList.Set("DexHigherThanStr", oDex > oStr)

		condList.Set("StrHighestAttribute", oStr >= oDex && oStr >= oInt)
		condList.Set("IntHighestAttribute", oInt >= oStr && oInt >= oDex)
		condList.Set("DexHighestAttribute", oDex >= oStr && oDex >= oInt)
		condList.Set("IntSingleHighestAttribute", oInt > oStr && oInt > oDex)
		condList.Set("DexSingleHighestAttribute", oDex > oStr && oDex > oInt)
	}

	calculateAttributes := func() {
		for pass := 1; pass <= 2; pass++ {
			for _, stat := range []string{"Str", "Dex", "Int"} {
				output.SetN(stat, math.Max(util.RoundHalfUp(Val(modDB, stat, nil), 0), 0))
			}
			setAttribConds()
		}
	}

	calculateOmniscience := func() {
		classStats := env.Build.ClassStats
		baseOf := map[string]float64{"Str": classStats.BaseStr, "Dex": classStats.BaseDex, "Int": classStats.BaseInt}
		for pass := 1; pass <= 2; pass++ {
			if pass != 1 {
				for _, stat := range []string{"Str", "Dex", "Int"} {
					base := baseOf[stat]
					output.SetN(stat, math.Min(util.RoundHalfUp(Val(modDB, stat, nil), 0), base))
					modDB.AddMod(newModS("Omni", modparser.Base, modparser.Num(modDB.Sum(modparser.Base, nil, stat)-base), stat+" conversion Omniscience"))
					modDB.AddMod(newModS("Omni", modparser.Inc, modparser.Num(modDB.Sum(modparser.Inc, nil, stat)), "Omniscience"))
					modDB.AddMod(newModS("Omni", modparser.More, modparser.Num(modDB.Sum(modparser.More, nil, stat)), "Omniscience"))
				}
			}
			if pass != 2 {
				// Subtract out double and triple dips
				reduction := map[modparser.ModType]float64{}
				for _, typ := range []modparser.ModType{modparser.Base, modparser.Inc, modparser.More} {
					conv := map[string]float64{}
					for _, stat := range []string{"StrDex", "StrInt", "DexInt", "All"} {
						conv[stat] = modDB.Sum(typ, nil, stat)
					}
					reduction[typ] = conv["StrDex"] + conv["StrInt"] + conv["DexInt"] + 2*conv["All"]
				}
				modDB.AddMod(newModS("Omni", modparser.Base, modparser.Num(-reduction[modparser.Base]), "Reduction from Double/Triple Dipped attributes to Omniscience"))
				modDB.AddMod(newModS("Omni", modparser.Inc, modparser.Num(-reduction[modparser.Inc]), "Reduction from Double/Triple Dipped attributes to Omniscience"))
				modDB.AddMod(newModS("Omni", modparser.More, modparser.Num(-reduction[modparser.More]), "Reduction from Double/Triple Dipped attributes to Omniscience"))
			}
			for _, stat := range []string{"Str", "Dex", "Int"} {
				output.SetN(stat, baseOf[stat])
			}
			output.SetN("Omni", math.Max(util.RoundHalfUp(Val(modDB, "Omni", nil), 0), 0))
			setAttribConds()
		}
	}

	if modDB.Flag(nil, "Omniscience") {
		calculateOmniscience()
	} else {
		calculateAttributes()
	}

	// Calculate total attributes
	output.SetN("TotalAttr", output.N("Str")+output.N("Dex")+output.N("Int"))

	// Special case for Devotion
	output.SetN("Devotion", modDB.Sum(modparser.Base, nil, "Devotion"))

	// Add attribute bonuses
	if !modDB.Flag(nil, "NoAttributeBonuses") {
		if !modDB.Flag(nil, "NoStrengthAttributeBonuses") {
			if !modDB.Flag(nil, "NoStrBonusToLife") {
				modDB.AddMod(newModS("Life", modparser.Base, modparser.Num(math.Floor(output.N("Str")/2)), "Strength"))
			}
			strDmgBonusRatioOverride := modDB.Sum(modparser.Base, nil, "StrDmgBonusRatioOverride")
			if strDmgBonusRatioOverride > 0 {
				actor.strDmgBonus = math.Floor((output.N("Str") + modDB.Sum(modparser.Base, nil, "DexIntToMeleeBonus")) * strDmgBonusRatioOverride)
			} else {
				actor.strDmgBonus = math.Floor((output.N("Str") + modDB.Sum(modparser.Base, nil, "DexIntToMeleeBonus")) / 5)
			}
			modDB.AddMod(newModSF("PhysicalDamage", modparser.Inc, modparser.Num(actor.strDmgBonus), "Strength", modparser.FlagMelee, modparser.KeywordNone))
		}
		if !modDB.Flag(nil, "NoDexterityAttributeBonuses") {
			accPerDex := data.Misc.AccuracyPerDexBase
			if ov, ok := modDB.Override(nil, "DexAccBonusOverride"); ok {
				accPerDex = valueNum(ov)
			}
			modDB.AddMod(newModS("Accuracy", modparser.Base, modparser.Num(output.N("Dex")*accPerDex), "Dexterity"))
			if !modDB.Flag(nil, "NoDexBonusToEvasion") {
				modDB.AddMod(newModS("Evasion", modparser.Inc, modparser.Num(math.Floor(output.N("Dex")/5)), "Dexterity"))
			}
		}
		if !modDB.Flag(nil, "NoIntelligenceAttributeBonuses") {
			if !modDB.Flag(nil, "NoIntBonusToMana") {
				modDB.AddMod(newModS("Mana", modparser.Base, modparser.Num(math.Floor(output.N("Int")/2)), "Intelligence"))
			}
			if !modDB.Flag(nil, "NoIntBonusToES") {
				modDB.AddMod(newModS("EnergyShield", modparser.Inc, modparser.Num(math.Floor(output.N("Int")/10)), "Intelligence"))
			}
		}
	}

	env.doActorLifeMana(actor)
}

// doActorLifeManaReservation ports CalcPerform's doActorLifeManaReservation.
func (env *Env) doActorLifeManaReservation(actor *performActor, addAura bool) {
	modDB := actor.db
	output := actor.output
	condList := modDB.Conditions

	for _, pool := range []string{"Life", "Mana"} {
		max := output.N(pool)
		reserved := 0.0
		if max > 0 {
			lowPerc := modDB.Sum(modparser.Base, nil, "Low"+pool+"Percentage")
			resBase, resPercent, uncancellable := actor.reservedLifeBase, actor.reservedLifePercent, actor.uncancellableLife
			if pool == "Mana" {
				resBase, resPercent, uncancellable = actor.reservedManaBase, actor.reservedManaPercent, actor.uncancellableMana
			}
			reserved = resBase + math.Ceil(max*resPercent/100)
			output.SetN(pool+"Reserved", math.Min(reserved, max))
			output.SetN(pool+"ReservedPercent", math.Min(reserved/max*100, 100))
			output.SetN(pool+"Unreserved", max-reserved)
			output.SetN(pool+"UnreservedPercent", (max-reserved)/max*100)
			output.SetN(pool+"UncancellableReservation", math.Min(uncancellable, 0))
			output.SetN(pool+"CancellableReservation", 100-uncancellable)
			threshold := data.Misc.LowPoolThreshold
			if lowPerc > 0 {
				threshold = lowPerc
			}
			if (max-reserved)/max <= threshold {
				condList.Set("Low"+pool, true)
			}
		}
		if addAura {
			for _, v := range modDB.List(nil, "GrantReserved"+pool+"AsAura") {
				mod := modRefOf(v)
				if mod == nil {
					continue
				}
				auraMod := cloneMod(mod)
				auraMod.Value = modparser.Num(math.Floor(valueNum(auraMod.Value) * math.Min(reserved, output.N(pool))))
				modDB.AddMod(newMod("ExtraAura", modparser.List, modparser.ModRef{Mod: auraMod}))
			}
		}
	}
}

// determineCursePriority ports CalcPerform's determineCursePriority.
func (env *Env) determineCursePriority(curseName string, activeSkill *ActiveSkill) float64 {
	source := ""
	slot := ""
	socket := 1
	if activeSkill != nil && activeSkill.SocketGroup != nil {
		source = activeSkill.SocketGroup.Source
		slot = activeSkill.SocketGroup.Slot
		for k, v := range activeSkill.SocketGroup.GemList {
			if v.GemData != nil && v.GemData.Name == curseName {
				// limit of 8 to avoid collision with CurseFromEquipment
				socket = k + 1
				if socket > 8 {
					socket = 8
				}
				break
			}
		}
	}
	basePriority := float64(data.CursePriority[curseName])
	socketPriority := float64(socket) * float64(data.CursePriority["SocketPriorityBase"])
	// slot:gsub(" (Swap)", "") — the parens are a Lua capture: strips " Swap"
	slotPriority := float64(data.CursePriority[strings.ReplaceAll(slot, " Swap", "")])
	sourcePriority := 0.0
	if activeSkill != nil && activeSkill.SkillTypes[modparser.SkillTypeAura] {
		sourcePriority = float64(data.CursePriority["CurseFromAura"])
	} else if source != "" {
		sourcePriority = float64(data.CursePriority["CurseFromEquipment"])
	}
	if source != "" && (slotPriority == float64(data.CursePriority["Ring 2"]) || slotPriority == float64(data.CursePriority["Ring 3"])) {
		slotPriority = float64(data.CursePriority["Ring 1"])
	}
	return basePriority + socketPriority + slotPriority + sourcePriority
}

// applyEnemyModifiers ports CalcPerform's applyEnemyModifiers.
func applyEnemyModifiers(actor *performActor, clearCache bool) {
	if clearCache || actor.appliedEnemyModifiers == nil {
		actor.appliedEnemyModifiers = map[*modparser.Mod]bool{}
	}
	cache := actor.appliedEnemyModifiers
	enemyDB := actor.enemy.db
	for _, value := range actor.db.TabulateAll(nil, "EnemyModifier") {
		mod := modRefOf(value.Value)
		if mod != nil && !cache[mod] {
			source := mod.Source
			if !mod.SourceSet {
				source = value.Mod.Source
			}
			enemyDB.AddMod(modparser.SetSource(mod, source))
			cache[mod] = true
		}
	}
}
