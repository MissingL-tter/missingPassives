// CalcOffence.lua L5864-6010: the self-hit damage sources (Heartbound Loop,
// Storm's Secret, Eye of Innocence, Echoes of Creation, Scold's Bridle,
// Trauma, Enmity's Embrace), plus the CalcDefence helper they call.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// capDamageType ports `string.gsub(" "..value.damageType, "%W%l", string.upper):sub(2)`:
// uppercase every lowercase letter that follows a non-word character, then
// drop the space that was prepended.
func capDamageType(s string) string {
	b := []byte(" " + s)
	isWord := func(c byte) bool {
		return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	for i := 0; i+1 < len(b); {
		if !isWord(b[i]) && b[i+1] >= 'a' && b[i+1] <= 'z' {
			b[i+1] -= 32
			i += 2
		} else {
			i++
		}
	}
	return string(b[1:])
}

// applyDmgTakenConversion ports calcs.applyDmgTakenConversion
// (CalcDefence.lua L83): how much of a self-inflicted hit actually lands.
func (env *Env) applyDmgTakenConversion(activeSkill *ActiveSkill, output modstore.Output, sourceType string, baseDmg float64) float64 {
	skillModList := activeSkill.SkillModList
	totalDamageTaken := 0.0
	shiftTable := map[string]float64{}
	totalTakenAs := 0.0
	for _, damageType := range dmgTypeList {
		names := []string{sourceType + "DamageTakenAs" + damageType, sourceType + "DamageFromHitsTakenAs" + damageType}
		if isElementalRes[sourceType] {
			names = append(names, "ElementalDamageTakenAs"+damageType, "ElementalDamageFromHitsTakenAs"+damageType)
		}
		shiftTable[damageType] = skillModList.Sum(modparser.Base, nil, names...)
		totalTakenAs += shiftTable[damageType]
	}
	for _, damageType := range dmgTypeList {
		var damageTakenAs float64
		if damageType == sourceType {
			damageTakenAs = math.Max(1-totalTakenAs/100, 0)
		} else {
			damageTakenAs = shiftTable[damageType] / 100
		}
		if damageTakenAs == 0 {
			continue
		}
		damage := baseDmg * damageTakenAs

		baseTakenInc := skillModList.Sum(modparser.Inc, nil, "DamageTaken", damageType+"DamageTaken", "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
		baseTakenMore := skillModList.More(nil, "DamageTaken", damageType+"DamageTaken", "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
		if damageType == "Lightning" || damageType == "Cold" || damageType == "Fire" {
			baseTakenInc += skillModList.Sum(modparser.Inc, nil, "ElementalDamageTaken", "ElementalDamageTakenWhenHit")
			baseTakenMore *= skillModList.More(nil, "ElementalDamageTaken", "ElementalDamageTakenWhenHit")
		}
		damageTakenMods := math.Max((1+baseTakenInc/100)*baseTakenMore, 0)
		reduction := 0.0
		if !skillModList.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
			if v := output["Base"+damageType+"DamageReductionWhenHit"]; v.Truthy() {
				reduction = v.Num()
			} else {
				reduction = output.N("Base" + damageType + "DamageReduction")
			}
		}
		resist := 0.0
		if !skillModList.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
			if v := output[damageType+"ResistWhenHit"]; v.Truthy() {
				resist = v.Num()
			} else {
				resist = output.N(damageType + "Resist")
			}
		}
		armourReduct := 0.0
		resMult := 1 - resist/100

		percentOfArmourApplies := 0.0
		if !skillModList.Flag(nil, "ArmourDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfArmourApplies = skillModList.Sum(modparser.Base, nil, "ArmourAppliesTo"+damageType+"DamageTaken")
		}
		percentOfArmourApplies = math.Min(percentOfArmourApplies, 100)
		physicalReductionBasedOnWard := damageType == "Physical" && skillModList.Flag(nil, "PhysicalReductionBasedOnWard")
		if percentOfArmourApplies > 0 || physicalReductionBasedOnWard {
			effArmour := (output.N("Armour") * percentOfArmourApplies / 100) * (1 + output.N("ArmourDefense"))
			if physicalReductionBasedOnWard {
				multiplier := overrideNum(skillModList, nil, "PhysicalReductionBasedOnWardPercent") / 100
				effArmour = output.N("Ward") * multiplier
			}
			effDamage := damage * resMult
			if effArmour != 0 && damage*resMult != 0 {
				armourReduct = util.RoundHalfUp(effArmour/(effArmour+effDamage*5)*100, 0)
			}
			armourReduct = math.Min(output.N("DamageReductionMax"), armourReduct)
		}
		reductMult := (1 - math.Max(math.Min(output.N("DamageReductionMax"), armourReduct+reduction), 0)/100) * damageTakenMods
		totalDamageTaken += damage * resMult * reductMult
	}
	return totalDamageTaken
}

// selfDamageMod reads the {baseDamage, damageType} list entries the unique
// self-damage mods carry, summing baseDamage across them.
func selfDamageList(activeSkill *ActiveSkill, name string) (dmgType string, dmgVal float64, ok bool) {
	for _, value := range activeSkill.SkillModList.List(nil, name) {
		tag, isSelfDamage := value.(modparser.SelfDamage)
		if !isSelfDamage {
			panic("calc: non-SelfDamage value in " + name + " list (the Lua errors)")
		}
		dmgVal += tag.BaseDamage.Or(0)
		dmgType = capDamageType(tag.DamageType)
		ok = true
	}
	return
}

// selfDamageFirst is the same read but taking only the first entry.
func selfDamageFirst(activeSkill *ActiveSkill, name, key string) (dmgType string, val float64, ok bool) {
	for _, value := range activeSkill.SkillModList.List(nil, name) {
		tag, isSelfDamage := value.(modparser.SelfDamage)
		if !isSelfDamage {
			panic("calc: non-SelfDamage value in " + name + " list (the Lua errors)")
		}
		if key == "dmgMult" {
			val = tag.DmgMult.Or(0)
		} else {
			val = tag.BaseDamage.Or(0)
		}
		dmgType = capDamageType(tag.DamageType)
		return dmgType, val, true
	}
	return "", 0, false
}

// offenceSelfHit ports L5864-6010.
func (env *Env) offenceSelfHit(c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillFlags := c.skillFlags

	add := func(v float64) {
		output.SetN("SelfHitDamage", output.N("SelfHitDamage")+v)
	}

	// The reference iterates the handler table with pairs(); each handler
	// only adds into SelfHitDamage, so the order is immaterial.
	if activeSkill.ActiveEffect.GrantedEffect.Name == "Summon Skeletons" { // Heartbound Loop
		if dmgType, dmgVal, ok := selfDamageList(activeSkill, "HeartboundLoopSelfDamage"); ok && dmgType != "" {
			add(env.applyDmgTakenConversion(activeSkill, output, dmgType, dmgVal) * output.N("SummonedMinionsPerCast"))
		}
	}
	if activeSkill.ActiveEffect.GrantedEffect.Name == "Herald of Thunder" { // Storm's Secret
		if dmgType, dmgVal, ok := selfDamageList(activeSkill, "StormSecretSelfDamage"); ok && dmgType != "" {
			add(env.applyDmgTakenConversion(activeSkill, output, dmgType, dmgVal))
		}
	}
	if dmgType, dmgVal, ok := selfDamageFirst(activeSkill, "EyeOfInnocenceSelfDamage", "baseDamage"); ok && skillFlags["ignite"] && dmgType != "" {
		add(env.applyDmgTakenConversion(activeSkill, output, dmgType, dmgVal))
	}
	if dmgType, dmgMult, ok := selfDamageFirst(activeSkill, "EchoesOfCreationSelfDamage", "dmgMult"); ok && dmgType != "" {
		averageWarcryCount := output.N("GlobalWarcryUptimeRatio") / 100
		if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			averageWarcryCount = env.ModDB.Sum(modparser.Base, nil, "Multiplier:ExertingWarcryCount")
		}
		add(env.applyDmgTakenConversion(activeSkill, output, dmgType, output.N("Life")*dmgMult/100*averageWarcryCount))
	}
	if dmgType, dmgMult, ok := selfDamageFirst(activeSkill, "ScoldsBridleSelfDamage", "dmgMult"); ok && output.Flag("ManaHasCost") && dmgType != "" {
		add(env.applyDmgTakenConversion(activeSkill, output, dmgType, output.N("ManaCost")*dmgMult/100))
	}
	{ // Trauma
		currentTraumaStacks := math.Max(activeSkill.SkillModList.Sum(modparser.Base, nil, "Multiplier:TraumaStacks"), 1)
		damagePerTrauma := activeSkill.SkillModList.Sum(modparser.Base, nil, "TraumaSelfDamageTakenLife")
		if activeSkill.BaseSkillModList.Flag(nil, "HasTrauma") {
			add(env.applyDmgTakenConversion(activeSkill, output, "Physical", damagePerTrauma*currentTraumaStacks))
		}
	}
	if dmgType, dmgVal, ok := selfDamageList(activeSkill, "EnmitysEmbraceSelfDamage"); ok && dmgType != "" {
		add(env.applyDmgTakenConversion(activeSkill, output, dmgType, dmgVal))
	}

	// The reference's Forbidden Rite block iterates
	// `ipairs({["FRDamageTaken"] = "Forbidden Rite"})` — a table with only a
	// string key, so ipairs yields nothing and the block never runs.
}
