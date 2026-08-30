// CalcOffence.lua L2547-2950: the second pass loop's opening — exerted
// attacks from warcries, the Pact uptime scaling, and the Ruthless Blow /
// Fist of War multipliers.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// warcryInfo is the per-cry data the exert loop reads back.
type warcryStats struct {
	duration, cooldown, castTime float64
}

func (env *Env) warcryStatsFor(c *offenceCtx, value *ActiveSkill) warcryStats {
	cooldown, _ := env.calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
	return warcryStats{
		duration: env.calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, c.enemyDB),
		cooldown: cooldown,
		castTime: env.calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, c.actor),
	}
}

// warcryStoredUses is `value.skillData.storedUses + Sum(AdditionalCooldownUses)`.
func warcryStoredUses(value *ActiveSkill) float64 {
	return value.SkillData.N("storedUses") +
		value.SkillModList.Sum(modparser.Base, value.SkillCfg, "AdditionalCooldownUses")
}

// offenceExerts ports L2552-2905 for one pass.
func (env *Env) offenceExerts(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, activeSkill, actor := c.skillFlags, c.activeSkill, c.actor
	enemyDB, cfg := c.enemyDB, pass.cfg
	globalOutput := c.output

	// Exerted Attack members
	exertedDoubleDamage := env.ModDB.Sum(modparser.Base, cfg, "ExertDoubleDamageChance")
	exertingWarcryCount := env.ModDB.Sum(modparser.Base, nil, "Multiplier:ExertingWarcryCount")
	globalOutput.SetN("OffensiveWarcryEffect", 1.0)
	globalOutput.SetN("MaxOffensiveWarcryEffect", 1.0)
	globalOutput.SetN("TheoreticalOffensiveWarcryEffect", 1.0)
	globalOutput.SetN("TheoreticalMaxOffensiveWarcryEffect", 1.0)
	globalOutput.SetN("RallyingHitEffect", 1.0)
	globalOutput.SetN("AilmentWarcryEffect", 1.0)
	globalOutput.SetN("GlobalWarcryUptimeRatio", 0.0)

	if !env.ModeBuffs {
		return
	}

	addGlobalUptime := func(v float64) {
		globalOutput.SetN("GlobalWarcryUptimeRatio", globalOutput.N("GlobalWarcryUptimeRatio")+v)
	}
	// baseUptime is the ratio every cry computes the same way.
	uptimeFor := func(exerts float64, w warcryStats, value *ActiveSkill) float64 {
		baseUptimeRatio := math.Min((exerts/globalOutput.N("Speed"))/(w.cooldown+w.castTime), 1) * 100
		return math.Min(100, baseUptimeRatio*warcryStoredUses(value))
	}

	// Iterate over all the active skills to account for exerted attacks
	// provided by warcries
	if !activeSkill.SkillTypes[modparser.SkillTypeNeverExertable] && !activeSkill.SkillTypes[modparser.SkillTypeTriggered] &&
		!activeSkill.SkillTypes[modparser.SkillTypeChannel] && !activeSkill.SkillTypes[modparser.SkillTypeOtherThingUsesSkill] &&
		!activeSkill.SkillTypes[modparser.SkillTypeRetaliation] && !activeSkill.SkillTypes[modparser.SkillTypeSummonsTotem] {
		for _, value := range actor.skills {
			name := value.ActiveEffect.GrantedEffect.Name
			switch {
			case name == "Ancestral Cry" && activeSkill.SkillTypes[modparser.SkillTypeMeleeSingleTarget] &&
				!globalOutput.Flag("AncestralCryCalculated") && !value.SkillFlags["disable"]:
				w := env.warcryStatsFor(c, value)
				globalOutput.SetN("AncestralCryDuration", w.duration)
				globalOutput.SetN("AncestralCryCooldown", w.cooldown)
				globalOutput.SetN("AncestralCryCastTime", w.castTime)
				globalOutput.SetN("AncestralExertsCount", env.ModDB.Sum(modparser.Base, nil, "NumAncestralExerts"))
				globalOutput.SetN("AncestralUpTimeRatio", uptimeFor(globalOutput.N("AncestralExertsCount"), w, value))
				addGlobalUptime(globalOutput.N("AncestralUpTimeRatio"))
				globalOutput.SetFlag("AncestralCryCalculated", true)
			case name == "Infernal Cry" && !globalOutput.Flag("InfernalCryCalculated") && !value.SkillFlags["disable"]:
				w := env.warcryStatsFor(c, value)
				globalOutput.SetN("InfernalCryDuration", w.duration)
				globalOutput.SetN("InfernalCryCooldown", w.cooldown)
				globalOutput.SetN("InfernalCryCastTime", w.castTime)
				if activeSkill.SkillTypes[modparser.SkillTypeMelee] {
					globalOutput.SetN("InfernalExertsCount", env.ModDB.Sum(modparser.Base, nil, "NumInfernalExerts"))
					globalOutput.SetN("InfernalUpTimeRatio", uptimeFor(globalOutput.N("InfernalExertsCount"), w, value))
					addGlobalUptime(globalOutput.N("InfernalUpTimeRatio"))
				}
				globalOutput.SetFlag("InfernalCryCalculated", true)
			case name == "Intimidating Cry" && activeSkill.SkillTypes[modparser.SkillTypeMelee] &&
				!globalOutput.Flag("IntimidatingCryCalculated") && !value.SkillFlags["disable"]:
				globalOutput.SetFlag("CreateWarcryOffensiveCalcSection", true)
				w := env.warcryStatsFor(c, value)
				globalOutput.SetN("IntimidatingCryDuration", w.duration)
				globalOutput.SetN("IntimidatingCryCooldown", w.cooldown)
				globalOutput.SetN("IntimidatingCryCastTime", w.castTime)
				globalOutput.SetN("IntimidatingExertsCount", env.ModDB.Sum(modparser.Base, nil, "NumIntimidatingExerts"))
				globalOutput.SetN("IntimidatingUpTimeRatio", uptimeFor(globalOutput.N("IntimidatingExertsCount"), w, value))
				addGlobalUptime(globalOutput.N("IntimidatingUpTimeRatio"))
				selfDD := 0.0
				if env.ModeEffective {
					selfDD = enemyDB.Sum(modparser.Base, cfg, "SelfDoubleDamageChance")
				}
				ddChance := math.Min(skillModList.Sum(modparser.Base, cfg, "DoubleDamageChance")+selfDD+exertedDoubleDamage, 100)
				globalOutput.SetN("IntimidatingAvgDmg", 2*(1-ddChance/100))
				globalOutput.SetN("IntimidatingHitEffect", 1+globalOutput.N("IntimidatingAvgDmg")*globalOutput.N("IntimidatingUpTimeRatio")/100)
				globalOutput.SetN("IntimidatingMaxHitEffect", 1+globalOutput.N("IntimidatingAvgDmg"))
				globalOutput.SetN("TheoreticalOffensiveWarcryEffect", globalOutput.N("TheoreticalOffensiveWarcryEffect")*globalOutput.N("IntimidatingHitEffect"))
				globalOutput.SetN("TheoreticalMaxOffensiveWarcryEffect", globalOutput.N("TheoreticalMaxOffensiveWarcryEffect")*globalOutput.N("IntimidatingMaxHitEffect"))
				globalOutput.SetFlag("IntimidatingCryCalculated", true)
			case name == "Rallying Cry" && activeSkill.SkillTypes[modparser.SkillTypeMelee] &&
				!globalOutput.Flag("RallyingCryCalculated") && !value.SkillFlags["disable"]:
				globalOutput.SetFlag("CreateWarcryOffensiveCalcSection", true)
				w := env.warcryStatsFor(c, value)
				globalOutput.SetN("RallyingCryDuration", w.duration)
				globalOutput.SetN("RallyingCryCooldown", w.cooldown)
				globalOutput.SetN("RallyingCryCastTime", w.castTime)
				globalOutput.SetN("RallyingExertsCount", env.ModDB.Sum(modparser.Base, nil, "NumRallyingExerts"))
				globalOutput.SetN("RallyingUpTimeRatio", uptimeFor(globalOutput.N("RallyingExertsCount"), w, value))
				addGlobalUptime(globalOutput.N("RallyingUpTimeRatio"))
				globalOutput.SetN("RallyingAvgDmg", math.Min(env.ModDB.Sum(modparser.Base, cfg, "Multiplier:NearbyAlly"), 5)*
					(env.ModDB.Sum(modparser.Base, nil, "RallyingExertMoreDamagePerAlly")/100))
				globalOutput.SetN("RallyingHitEffect", 1+globalOutput.N("RallyingAvgDmg")*globalOutput.N("RallyingUpTimeRatio")/100)
				globalOutput.SetN("RallyingMaxHitEffect", 1+globalOutput.N("RallyingAvgDmg"))
				globalOutput.SetN("OffensiveWarcryEffect", globalOutput.N("OffensiveWarcryEffect")*globalOutput.N("RallyingHitEffect"))
				globalOutput.SetN("MaxOffensiveWarcryEffect", globalOutput.N("MaxOffensiveWarcryEffect")*globalOutput.N("RallyingMaxHitEffect"))
				globalOutput.SetN("TheoreticalOffensiveWarcryEffect", globalOutput.N("TheoreticalOffensiveWarcryEffect")*globalOutput.N("RallyingHitEffect"))
				globalOutput.SetN("TheoreticalMaxOffensiveWarcryEffect", globalOutput.N("TheoreticalMaxOffensiveWarcryEffect")*globalOutput.N("RallyingMaxHitEffect"))
				globalOutput.SetFlag("RallyingCryCalculated", true)
			case name == "Seismic Cry" && activeSkill.SkillTypes[modparser.SkillTypeSlam] &&
				!globalOutput.Flag("SeismicCryCalculated") && !value.SkillFlags["disable"]:
				globalOutput.SetFlag("CreateWarcryOffensiveCalcSection", true)
				w := env.warcryStatsFor(c, value)
				globalOutput.SetN("SeismicCryDuration", w.duration)
				globalOutput.SetN("SeismicCryCooldown", w.cooldown)
				globalOutput.SetN("SeismicCryCastTime", w.castTime)
				globalOutput.SetN("SeismicExertsCount", env.ModDB.Sum(modparser.Base, nil, "NumSeismicExerts"))
				globalOutput.SetN("SeismicUpTimeRatio", uptimeFor(globalOutput.N("SeismicExertsCount"), w, value))
				addGlobalUptime(globalOutput.N("SeismicUpTimeRatio"))
				// account for AoE increase
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					skillModList.AddMod(newModS("AreaOfEffect", modparser.More, modparser.Num(env.ModDB.Sum(modparser.Base, nil, "SeismicMoreAoE")), "Max Seismic Exert AoE"))
				} else {
					skillModList.AddMod(newModS("AreaOfEffect", modparser.More, modparser.Num(math.Floor(env.ModDB.Sum(modparser.Base, nil, "SeismicMoreAoE")/100*globalOutput.N("SeismicUpTimeRatio"))), "Avg Seismic Exert AoE"))
				}
				env.calcAreaOfEffect(c)
				globalOutput.SetFlag("SeismicCryCalculated", true)
			case name == "Battlemage's Cry" && !globalOutput.Flag("BattleMageCryCalculated") && !value.SkillFlags["disable"]:
				w := env.warcryStatsFor(c, value)
				globalOutput.SetN("BattleMageCryDuration", w.duration)
				globalOutput.SetN("BattleMageCryCooldown", w.cooldown)
				globalOutput.SetN("BattleMageCryCastTime", w.castTime)
				if activeSkill.SkillTypes[modparser.SkillTypeMelee] {
					globalOutput.SetN("BattleCryExertsCount", env.ModDB.Sum(modparser.Base, nil, "NumBattlemageExerts"))
					globalOutput.SetN("BattlemageUpTimeRatio", uptimeFor(globalOutput.N("BattleCryExertsCount"), w, value))
					addGlobalUptime(globalOutput.N("BattlemageUpTimeRatio"))
				}
				globalOutput.SetFlag("BattleMageCryCalculated", true)
			}
		}

		if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			globalOutput.Set("AilmentWarcryEffect", globalOutput.Get("MaxOffensiveWarcryEffect"))
			skillData.SetFlag("showAverage", true)
			skillFlags["showAverage"] = true
			skillFlags["notAverage"] = false
		} else {
			globalOutput.Set("AilmentWarcryEffect", globalOutput.Get("OffensiveWarcryEffect"))
		}

		// Calculate Exerted Attack Uptime
		// 1) they don't pay attention and therefore we calculate exerted
		// attack uptime as just the maximum uptime of any enabled warcries
		// that exert attacks
		warcryList := []string{"AncestralUpTimeRatio", "InfernalUpTimeRatio", "IntimidatingUpTimeRatio",
			"RallyingUpTimeRatio", "SeismicUpTimeRatio", "BattlemageUpTimeRatio"}
		for _, cryTimeRatio := range warcryList {
			globalOutput.SetN("ExertedAttackUptimeRatio", math.Max(globalOutput.N("ExertedAttackUptimeRatio"), globalOutput.N(cryTimeRatio)))
		}
		if globalOutput.N("ExertedAttackUptimeRatio") > 0 && !globalOutput.Flag("ExertedAttackUptimeRatioCalculated") {
			incExertedAttacks := skillModList.Sum(modparser.Inc, cfg, "ExertIncrease")
			moreExertedAttacks := skillModList.Sum(modparser.More, cfg, "ExertIncrease")
			moreExertedAttackDamage := skillModList.Sum(modparser.More, cfg, "ExertAttackIncrease")
			overexertionExertedDamage := skillModList.Sum(modparser.More, cfg, "OverexertionExertAverageIncrease")
			echoesOfCreationExertedDamage := skillModList.Sum(modparser.More, cfg, "EchoesExertAverageIncrease")
			if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
				skillModList.AddMod(newModS("Damage", modparser.Inc, modparser.Num(incExertedAttacks), "Exerted Attacks"))
				skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(moreExertedAttacks), "Exerted Attacks"))
				skillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(moreExertedAttackDamage), "Exerted Attack Damage", modparser.FlagAttack, modparser.KeywordNone))
				skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(overexertionExertedDamage*exertingWarcryCount), "Max Autoexertion Support"))
				skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(echoesOfCreationExertedDamage*exertingWarcryCount), "Max Echoes of Creation"))
			} else {
				uptime := globalOutput.N("ExertedAttackUptimeRatio")
				globalUptime := globalOutput.N("GlobalWarcryUptimeRatio")
				skillModList.AddMod(newModS("Damage", modparser.Inc, modparser.Num(incExertedAttacks*uptime/100), "Uptime Scaled Exerted Attacks"))
				skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(moreExertedAttacks*uptime/100), "Uptime Scaled Exerted Attacks"))
				skillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(moreExertedAttackDamage*uptime/100), "Uptime Scaled Exerted Attack Damage", modparser.FlagAttack, modparser.KeywordNone))
				skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(overexertionExertedDamage*globalUptime/100), "Uptime Scaled Autoexertion Support"))
				skillModList.AddMod(newModS("Damage", modparser.More, modparser.Num(echoesOfCreationExertedDamage*globalUptime/100), "Uptime Scaled Echoes of Creation"))
			}
			avg := Mod(skillModList, skillCfg, "ExertIncrease")
			avg = avg * Mod(skillModList, skillCfg, "ExertAttackIncrease", "OverexertionExertAverageIncrease", "EchoesExertAverageIncrease")
			globalOutput.SetN("ExertedAttackAvgDmg", avg)
			globalOutput.SetN("ExertedAttackHitEffect", avg*globalOutput.N("ExertedAttackUptimeRatio")/100)
			globalOutput.SetN("ExertedAttackMaxHitEffect", avg)
			globalOutput.SetFlag("ExertedAttackUptimeRatioCalculated", true)
		}
	}

	// Each Pact has different eligible spell types, but shares the same
	// uptime calculation.
	if activeSkill.SkillTypes[modparser.SkillTypeSpell] && !activeSkill.SkillTypes[modparser.SkillTypeBrand] {
		for _, value := range actor.skills {
			pactName := value.ActiveEffect.GrantedEffect.Name
			if !strings.HasPrefix(pactName, "Pact of ") {
				continue
			}
			pactKey := pactName[len("Pact of "):]
			if pactKey == "" {
				continue
			}
			if pactKey == "K'Tash" {
				pactKey = "Ktash"
			}
			isVaal := activeSkill.SkillTypes[modparser.SkillTypeVaal]
			pactApplies := pactKey == "Beidat" && !isVaal && !activeSkill.SkillTypes[modparser.SkillTypeChannel] &&
				(skillFlags["projectile"] || activeSkill.SkillTypes[modparser.SkillTypeCascadable] || (skillFlags["chaining"] && !skillFlags["projectile"])) ||
				pactKey == "Ghorr" && !isVaal && activeSkill.SkillTypes[modparser.SkillTypeDamageOverTime] ||
				pactKey == "Ktash" && isVaal && (activeSkill.SkillTypes[modparser.SkillTypeDamage] || activeSkill.SkillTypes[modparser.SkillTypeDamageOverTime]) ||
				pactKey == "Lycia" && !isVaal && activeSkill.SkillTypes[modparser.SkillTypeChannel] && skillFlags["selfCast"]
			calculated := "PactOf" + pactKey + "Calculated"
			if !pactApplies || globalOutput.Flag(calculated) || value.SkillFlags["disable"] {
				continue
			}
			cooldown, _ := env.calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
			pactCastTime := 0.0
			if ct := value.ActiveEffect.GrantedEffect.CastTime; ct != nil {
				pactCastTime = *ct
			}
			castRate := 1 / pactCastTime * Mod(value.SkillModList, value.SkillCfg, "Speed") * env.actionSpeedMod(actor)
			castTime := 1 / math.Min(castRate, data.Misc.ServerTickRate)
			count := value.SkillModList.Sum(modparser.Base, value.SkillCfg, pactKey+"EmpoweredSpells")
			storedUses := warcryStoredUses(value)
			uptime := 100.0
			if globalOutput.N("Speed") != 0 {
				uptime = math.Min((count/globalOutput.N("Speed"))/(cooldown+castTime), 1) * 100
			}
			uptime = math.Min(100, uptime*storedUses)
			effect := uptime
			if skillModList.Flag(nil, "Condition:PactMaxHit") {
				effect = 100
			}
			effectMult := effect / 100

			globalOutput.SetFlag("CreatePactOffensiveCalcSection", true)
			globalOutput.SetN("PactOf"+pactKey+"Cooldown", cooldown)
			globalOutput.SetN("PactOf"+pactKey+"CastTime", castTime)
			globalOutput.SetN(pactKey+"EmpoweredCount", count)
			globalOutput.SetN(pactKey+"UpTimeRatio", uptime)

			if list := value.SkillModList.List(value.SkillCfg, pactKey+"PactDamage"); len(list) > 0 {
				if mod := modRefOf(list[0]); mod != nil {
					skillModList.AddMod(modparser.NewModFull(mod.Name, mod.Type, modparser.Num(valueNum(mod.Value)*effectMult), "Uptime Scaled "+pactName, true, mod.Flags, mod.KeywordFlags, mod.Tags...))
				}
			}

			switch pactKey {
			case "Beidat":
				// Coverage bonuses are averaged with the same uptime as the
				// damage bonus.
				if skillFlags["projectile"] && !skillModList.Flag(skillCfg, "NoAdditionalProjectiles") && !skillModList.Flag(skillCfg, "SingleProjectile") {
					globalOutput.SetN("BeidatAdditionalProjectiles", value.SkillModList.Sum(modparser.Base, value.SkillCfg, "BeidatAdditionalProjectiles")*effectMult)
					globalOutput.SetN("ProjectileCount", globalOutput.N("ProjectileCount")+globalOutput.N("BeidatAdditionalProjectiles"))
				}
				if skillFlags["chaining"] && !skillFlags["projectile"] && !skillModList.Flag(skillCfg, "CannotChain") && !skillModList.Flag(skillCfg, "NoAdditionalChains") {
					globalOutput.SetN("BeidatAdditionalBeamChains", value.SkillModList.Sum(modparser.Base, value.SkillCfg, "BeidatAdditionalBeamChains")*effectMult)
					globalOutput.SetN("ChainMax", globalOutput.N("ChainMax")+globalOutput.N("BeidatAdditionalBeamChains"))
					globalOutput.Set("ChainMaxString", globalOutput.Get("ChainMax"))
					globalOutput.SetN("ChainRemaining", math.Max(0, globalOutput.N("ChainMax")-globalOutput.N("Chain")))
				}
				if activeSkill.SkillTypes[modparser.SkillTypeCascadable] {
					globalOutput.SetN("BeidatAdditionalCascades", value.SkillModList.Sum(modparser.Base, value.SkillCfg, "BeidatAdditionalCascades")*effectMult)
				}
			case "Ktash":
				globalOutput.SetN("KtashSoulRefundChance", value.SkillModList.Sum(modparser.Base, value.SkillCfg, "KtashPactSoulRefundChance")*effectMult)
				if globalOutput.Has("SoulGainPreventionDuration") {
					duration := globalOutput.N("SoulGainPreventionDuration")
					durationMod := 1 + value.SkillModList.Sum(modparser.Base, value.SkillCfg, "KtashPactSoulGainPrevention")*effectMult/100
					globalOutput.SetN("SoulGainPreventionDuration", math.Max(math.Ceil(duration*durationMod*data.Misc.ServerTickRate), 1)/data.Misc.ServerTickRate)
				}
			}
			globalOutput.SetFlag(calculated, true)
		}
	}
}

// offenceRuthless ports L2907-2990: the Ruthless Blow and Fist of War
// multipliers for one pass.
func (env *Env) offenceRuthless(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output

	output.SetN("RuthlessBlowHitEffect", 1.0)
	output.SetN("RuthlessBlowAilmentEffect", 1.0)
	output.SetN("FistOfWarDamageEffect", 1.0)
	if !env.ModeCombat {
		return
	}
	ruthlessEffect := "AVERAGE"
	if v := env.ConfigInput.RuthlessSupportMode; v != "" {
		ruthlessEffect = v
	}
	// Calculate Ruthless Blow chance/multipliers + Fist of War multipliers
	output.SetN("RuthlessBlowMaxCount", skillModList.Sum(modparser.Base, cfg, "RuthlessBlowMaxCount"))
	maxCount := output.N("RuthlessBlowMaxCount")
	usedByMirage := skillCfg.SkillCond != nil && skillCfg.SkillCond["usedByMirage"]
	if maxCount > 0 && (!usedByMirage || skillData.N("mirageUses") > maxCount) {
		switch ruthlessEffect {
		case "AVERAGE":
			output.SetN("RuthlessBlowChance", util.RoundHalfUp(100/maxCount, 0))
		case "MAX":
			output.SetN("RuthlessBlowChance", 100.0)
			// `dpsMultiplier / (output.RuthlessBlowMaxCount or 1)`: the `or 1`
			// is unreachable (maxCount > 0 gates this branch and the value is
			// a number, so it is never nil), and so is this guard.
			denom := maxCount
			if denom == 0 {
				denom = 1
			}
			skillData.SetN("dpsMultiplier", skillData.N("dpsMultiplier")/denom)
		}
	} else {
		output.SetN("RuthlessBlowChance", 0.0)
	}
	output.SetN("RuthlessBlowHitMultiplier", 1+skillModList.Sum(modparser.Base, cfg, "RuthlessBlowHitMultiplier")/100)
	output.SetN("RuthlessBlowAilmentMultiplier", 1+skillModList.Sum(modparser.Base, cfg, "RuthlessBlowAilmentMultiplier")/100)
	chance := output.N("RuthlessBlowChance")
	output.SetN("RuthlessBlowHitEffect", 1-chance/100+chance/100*output.N("RuthlessBlowHitMultiplier"))
	output.SetN("RuthlessBlowAilmentEffect", 1-chance/100+chance/100*output.N("RuthlessBlowAilmentMultiplier"))

	globalOutput.SetN("FistOfWarCooldown", skillModList.Sum(modparser.Base, cfg, "FistOfWarCooldown"))
	// If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill & NOT
	// used by mirage or other
	if globalOutput.N("FistOfWarCooldown") != 0 && activeSkill.SkillTypes[modparser.SkillTypeSlam] &&
		!activeSkill.SkillTypes[modparser.SkillTypeVaal] && !activeSkill.SkillTypes[modparser.SkillTypeOtherThingUsesSkill] {
		env.offenceFistOfWar(c, pass)
	}
}

var _ = modstore.Cfg{}
