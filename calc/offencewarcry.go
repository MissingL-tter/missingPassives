// CalcOffence.lua L2547-2950: the second pass loop's opening — exerted
// attacks from warcries, the Pact uptime scaling, and the Ruthless Blow /
// Fist of War multipliers.
package calc

import (
	"math"
	"strings"

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
	return anyNum(value.SkillData["storedUses"]) +
		value.SkillModList.Sum("BASE", value.SkillCfg, "AdditionalCooldownUses")
}

// offenceExerts ports L2552-2905 for one pass.
func (env *Env) offenceExerts(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, activeSkill, actor := c.skillFlags, c.activeSkill, c.actor
	enemyDB, cfg := c.enemyDB, pass.cfg
	globalOutput := c.output
	d := env.Data

	// Exerted Attack members
	exertedDoubleDamage := env.ModDB.Sum("BASE", cfg, "ExertDoubleDamageChance")
	exertingWarcryCount := env.ModDB.Sum("BASE", nil, "Multiplier:ExertingWarcryCount")
	globalOutput["OffensiveWarcryEffect"] = 1.0
	globalOutput["MaxOffensiveWarcryEffect"] = 1.0
	globalOutput["TheoreticalOffensiveWarcryEffect"] = 1.0
	globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = 1.0
	globalOutput["RallyingHitEffect"] = 1.0
	globalOutput["AilmentWarcryEffect"] = 1.0
	globalOutput["GlobalWarcryUptimeRatio"] = 0.0

	if !env.ModeBuffs {
		return
	}

	addGlobalUptime := func(v float64) {
		globalOutput["GlobalWarcryUptimeRatio"] = outNum(globalOutput, "GlobalWarcryUptimeRatio") + v
	}
	// baseUptime is the ratio every cry computes the same way.
	uptimeFor := func(exerts float64, w warcryStats, value *ActiveSkill) float64 {
		baseUptimeRatio := math.Min((exerts/outNum(globalOutput, "Speed"))/(w.cooldown+w.castTime), 1) * 100
		return math.Min(100, baseUptimeRatio*warcryStoredUses(value))
	}

	// Iterate over all the active skills to account for exerted attacks
	// provided by warcries
	if !activeSkill.SkillTypes[modparser.SkillType.NeverExertable] && !activeSkill.SkillTypes[modparser.SkillType.Triggered] &&
		!activeSkill.SkillTypes[modparser.SkillType.Channel] && !activeSkill.SkillTypes[modparser.SkillType.OtherThingUsesSkill] &&
		!activeSkill.SkillTypes[modparser.SkillType.Retaliation] && !activeSkill.SkillTypes[modparser.SkillType.SummonsTotem] {
		for _, value := range actor.skills {
			name := value.ActiveEffect.GrantedEffect.Name
			switch {
			case name == "Ancestral Cry" && activeSkill.SkillTypes[modparser.SkillType.MeleeSingleTarget] &&
				!truthy(globalOutput["AncestralCryCalculated"]) && !value.SkillFlags["disable"]:
				w := env.warcryStatsFor(c, value)
				globalOutput["AncestralCryDuration"] = w.duration
				globalOutput["AncestralCryCooldown"] = w.cooldown
				globalOutput["AncestralCryCastTime"] = w.castTime
				globalOutput["AncestralExertsCount"] = env.ModDB.Sum("BASE", nil, "NumAncestralExerts")
				globalOutput["AncestralUpTimeRatio"] = uptimeFor(outNum(globalOutput, "AncestralExertsCount"), w, value)
				addGlobalUptime(outNum(globalOutput, "AncestralUpTimeRatio"))
				globalOutput["AncestralCryCalculated"] = true
			case name == "Infernal Cry" && !truthy(globalOutput["InfernalCryCalculated"]) && !value.SkillFlags["disable"]:
				w := env.warcryStatsFor(c, value)
				globalOutput["InfernalCryDuration"] = w.duration
				globalOutput["InfernalCryCooldown"] = w.cooldown
				globalOutput["InfernalCryCastTime"] = w.castTime
				if activeSkill.SkillTypes[modparser.SkillType.Melee] {
					globalOutput["InfernalExertsCount"] = env.ModDB.Sum("BASE", nil, "NumInfernalExerts")
					globalOutput["InfernalUpTimeRatio"] = uptimeFor(outNum(globalOutput, "InfernalExertsCount"), w, value)
					addGlobalUptime(outNum(globalOutput, "InfernalUpTimeRatio"))
				}
				globalOutput["InfernalCryCalculated"] = true
			case name == "Intimidating Cry" && activeSkill.SkillTypes[modparser.SkillType.Melee] &&
				!truthy(globalOutput["IntimidatingCryCalculated"]) && !value.SkillFlags["disable"]:
				globalOutput["CreateWarcryOffensiveCalcSection"] = true
				w := env.warcryStatsFor(c, value)
				globalOutput["IntimidatingCryDuration"] = w.duration
				globalOutput["IntimidatingCryCooldown"] = w.cooldown
				globalOutput["IntimidatingCryCastTime"] = w.castTime
				globalOutput["IntimidatingExertsCount"] = env.ModDB.Sum("BASE", nil, "NumIntimidatingExerts")
				globalOutput["IntimidatingUpTimeRatio"] = uptimeFor(outNum(globalOutput, "IntimidatingExertsCount"), w, value)
				addGlobalUptime(outNum(globalOutput, "IntimidatingUpTimeRatio"))
				selfDD := 0.0
				if env.ModeEffective {
					selfDD = enemyDB.Sum("BASE", cfg, "SelfDoubleDamageChance")
				}
				ddChance := math.Min(skillModList.Sum("BASE", cfg, "DoubleDamageChance")+selfDD+exertedDoubleDamage, 100)
				globalOutput["IntimidatingAvgDmg"] = 2 * (1 - ddChance/100)
				globalOutput["IntimidatingHitEffect"] = 1 + outNum(globalOutput, "IntimidatingAvgDmg")*outNum(globalOutput, "IntimidatingUpTimeRatio")/100
				globalOutput["IntimidatingMaxHitEffect"] = 1 + outNum(globalOutput, "IntimidatingAvgDmg")
				globalOutput["TheoreticalOffensiveWarcryEffect"] = outNum(globalOutput, "TheoreticalOffensiveWarcryEffect") * outNum(globalOutput, "IntimidatingHitEffect")
				globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = outNum(globalOutput, "TheoreticalMaxOffensiveWarcryEffect") * outNum(globalOutput, "IntimidatingMaxHitEffect")
				globalOutput["IntimidatingCryCalculated"] = true
			case name == "Rallying Cry" && activeSkill.SkillTypes[modparser.SkillType.Melee] &&
				!truthy(globalOutput["RallyingCryCalculated"]) && !value.SkillFlags["disable"]:
				globalOutput["CreateWarcryOffensiveCalcSection"] = true
				w := env.warcryStatsFor(c, value)
				globalOutput["RallyingCryDuration"] = w.duration
				globalOutput["RallyingCryCooldown"] = w.cooldown
				globalOutput["RallyingCryCastTime"] = w.castTime
				globalOutput["RallyingExertsCount"] = env.ModDB.Sum("BASE", nil, "NumRallyingExerts")
				globalOutput["RallyingUpTimeRatio"] = uptimeFor(outNum(globalOutput, "RallyingExertsCount"), w, value)
				addGlobalUptime(outNum(globalOutput, "RallyingUpTimeRatio"))
				globalOutput["RallyingAvgDmg"] = math.Min(env.ModDB.Sum("BASE", cfg, "Multiplier:NearbyAlly"), 5) *
					(env.ModDB.Sum("BASE", nil, "RallyingExertMoreDamagePerAlly") / 100)
				globalOutput["RallyingHitEffect"] = 1 + outNum(globalOutput, "RallyingAvgDmg")*outNum(globalOutput, "RallyingUpTimeRatio")/100
				globalOutput["RallyingMaxHitEffect"] = 1 + outNum(globalOutput, "RallyingAvgDmg")
				globalOutput["OffensiveWarcryEffect"] = outNum(globalOutput, "OffensiveWarcryEffect") * outNum(globalOutput, "RallyingHitEffect")
				globalOutput["MaxOffensiveWarcryEffect"] = outNum(globalOutput, "MaxOffensiveWarcryEffect") * outNum(globalOutput, "RallyingMaxHitEffect")
				globalOutput["TheoreticalOffensiveWarcryEffect"] = outNum(globalOutput, "TheoreticalOffensiveWarcryEffect") * outNum(globalOutput, "RallyingHitEffect")
				globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = outNum(globalOutput, "TheoreticalMaxOffensiveWarcryEffect") * outNum(globalOutput, "RallyingMaxHitEffect")
				globalOutput["RallyingCryCalculated"] = true
			case name == "Seismic Cry" && activeSkill.SkillTypes[modparser.SkillType.Slam] &&
				!truthy(globalOutput["SeismicCryCalculated"]) && !value.SkillFlags["disable"]:
				globalOutput["CreateWarcryOffensiveCalcSection"] = true
				w := env.warcryStatsFor(c, value)
				globalOutput["SeismicCryDuration"] = w.duration
				globalOutput["SeismicCryCooldown"] = w.cooldown
				globalOutput["SeismicCryCastTime"] = w.castTime
				globalOutput["SeismicExertsCount"] = env.ModDB.Sum("BASE", nil, "NumSeismicExerts")
				globalOutput["SeismicUpTimeRatio"] = uptimeFor(outNum(globalOutput, "SeismicExertsCount"), w, value)
				addGlobalUptime(outNum(globalOutput, "SeismicUpTimeRatio"))
				// account for AoE increase
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					skillModList.AddMod(newMod("AreaOfEffect", "MORE", env.ModDB.Sum("BASE", nil, "SeismicMoreAoE"), "Max Seismic Exert AoE"))
				} else {
					skillModList.AddMod(newMod("AreaOfEffect", "MORE",
						math.Floor(env.ModDB.Sum("BASE", nil, "SeismicMoreAoE")/100*outNum(globalOutput, "SeismicUpTimeRatio")), "Avg Seismic Exert AoE"))
				}
				env.calcAreaOfEffect(c)
				globalOutput["SeismicCryCalculated"] = true
			case name == "Battlemage's Cry" && !truthy(globalOutput["BattleMageCryCalculated"]) && !value.SkillFlags["disable"]:
				w := env.warcryStatsFor(c, value)
				globalOutput["BattleMageCryDuration"] = w.duration
				globalOutput["BattleMageCryCooldown"] = w.cooldown
				globalOutput["BattleMageCryCastTime"] = w.castTime
				if activeSkill.SkillTypes[modparser.SkillType.Melee] {
					globalOutput["BattleCryExertsCount"] = env.ModDB.Sum("BASE", nil, "NumBattlemageExerts")
					globalOutput["BattlemageUpTimeRatio"] = uptimeFor(outNum(globalOutput, "BattleCryExertsCount"), w, value)
					addGlobalUptime(outNum(globalOutput, "BattlemageUpTimeRatio"))
				}
				globalOutput["BattleMageCryCalculated"] = true
			}
		}

		if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			globalOutput["AilmentWarcryEffect"] = globalOutput["MaxOffensiveWarcryEffect"]
			skillData["showAverage"] = true
			skillFlags["showAverage"] = true
			skillFlags["notAverage"] = false
		} else {
			globalOutput["AilmentWarcryEffect"] = globalOutput["OffensiveWarcryEffect"]
		}

		// Calculate Exerted Attack Uptime
		// 1) they don't pay attention and therefore we calculate exerted
		// attack uptime as just the maximum uptime of any enabled warcries
		// that exert attacks
		warcryList := []string{"AncestralUpTimeRatio", "InfernalUpTimeRatio", "IntimidatingUpTimeRatio",
			"RallyingUpTimeRatio", "SeismicUpTimeRatio", "BattlemageUpTimeRatio"}
		for _, cryTimeRatio := range warcryList {
			globalOutput["ExertedAttackUptimeRatio"] = math.Max(outNum(globalOutput, "ExertedAttackUptimeRatio"), outNum(globalOutput, cryTimeRatio))
		}
		if outNum(globalOutput, "ExertedAttackUptimeRatio") > 0 && !truthy(globalOutput["ExertedAttackUptimeRatioCalculated"]) {
			incExertedAttacks := skillModList.Sum("INC", cfg, "ExertIncrease")
			moreExertedAttacks := skillModList.Sum("MORE", cfg, "ExertIncrease")
			moreExertedAttackDamage := skillModList.Sum("MORE", cfg, "ExertAttackIncrease")
			overexertionExertedDamage := skillModList.Sum("MORE", cfg, "OverexertionExertAverageIncrease")
			echoesOfCreationExertedDamage := skillModList.Sum("MORE", cfg, "EchoesExertAverageIncrease")
			if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
				skillModList.AddMod(newMod("Damage", "INC", incExertedAttacks, "Exerted Attacks"))
				skillModList.AddMod(newMod("Damage", "MORE", moreExertedAttacks, "Exerted Attacks"))
				skillModList.AddMod(newMod("Damage", "MORE", moreExertedAttackDamage, "Exerted Attack Damage", modparser.ModFlag.Attack))
				skillModList.AddMod(newMod("Damage", "MORE", overexertionExertedDamage*exertingWarcryCount, "Max Autoexertion Support"))
				skillModList.AddMod(newMod("Damage", "MORE", echoesOfCreationExertedDamage*exertingWarcryCount, "Max Echoes of Creation"))
			} else {
				uptime := outNum(globalOutput, "ExertedAttackUptimeRatio")
				globalUptime := outNum(globalOutput, "GlobalWarcryUptimeRatio")
				skillModList.AddMod(newMod("Damage", "INC", incExertedAttacks*uptime/100, "Uptime Scaled Exerted Attacks"))
				skillModList.AddMod(newMod("Damage", "MORE", moreExertedAttacks*uptime/100, "Uptime Scaled Exerted Attacks"))
				skillModList.AddMod(newMod("Damage", "MORE", moreExertedAttackDamage*uptime/100, "Uptime Scaled Exerted Attack Damage", modparser.ModFlag.Attack))
				skillModList.AddMod(newMod("Damage", "MORE", overexertionExertedDamage*globalUptime/100, "Uptime Scaled Autoexertion Support"))
				skillModList.AddMod(newMod("Damage", "MORE", echoesOfCreationExertedDamage*globalUptime/100, "Uptime Scaled Echoes of Creation"))
			}
			avg := Mod(skillModList, skillCfg, "ExertIncrease")
			avg = avg * Mod(skillModList, skillCfg, "ExertAttackIncrease", "OverexertionExertAverageIncrease", "EchoesExertAverageIncrease")
			globalOutput["ExertedAttackAvgDmg"] = avg
			globalOutput["ExertedAttackHitEffect"] = avg * outNum(globalOutput, "ExertedAttackUptimeRatio") / 100
			globalOutput["ExertedAttackMaxHitEffect"] = avg
			globalOutput["ExertedAttackUptimeRatioCalculated"] = true
		}
	}

	// Each Pact has different eligible spell types, but shares the same
	// uptime calculation.
	if activeSkill.SkillTypes[modparser.SkillType.Spell] && !activeSkill.SkillTypes[modparser.SkillType.Brand] {
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
			isVaal := activeSkill.SkillTypes[modparser.SkillType.Vaal]
			pactApplies := pactKey == "Beidat" && !isVaal && !activeSkill.SkillTypes[modparser.SkillType.Channel] &&
				(skillFlags["projectile"] || activeSkill.SkillTypes[modparser.SkillType.Cascadable] || (skillFlags["chaining"] && !skillFlags["projectile"])) ||
				pactKey == "Ghorr" && !isVaal && activeSkill.SkillTypes[modparser.SkillType.DamageOverTime] ||
				pactKey == "Ktash" && isVaal && (activeSkill.SkillTypes[modparser.SkillType.Damage] || activeSkill.SkillTypes[modparser.SkillType.DamageOverTime]) ||
				pactKey == "Lycia" && !isVaal && activeSkill.SkillTypes[modparser.SkillType.Channel] && skillFlags["selfCast"]
			calculated := "PactOf" + pactKey + "Calculated"
			if !pactApplies || truthy(globalOutput[calculated]) || value.SkillFlags["disable"] {
				continue
			}
			cooldown, _ := env.calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
			pactCastTime := 0.0
			if ct := value.ActiveEffect.GrantedEffect.CastTime; ct != nil {
				pactCastTime = *ct
			}
			castRate := 1 / pactCastTime * Mod(value.SkillModList, value.SkillCfg, "Speed") * env.actionSpeedMod(actor)
			castTime := 1 / math.Min(castRate, d.Misc.ServerTickRate)
			count := value.SkillModList.Sum("BASE", value.SkillCfg, pactKey+"EmpoweredSpells")
			storedUses := warcryStoredUses(value)
			uptime := 100.0
			if outNum(globalOutput, "Speed") != 0 {
				uptime = math.Min((count/outNum(globalOutput, "Speed"))/(cooldown+castTime), 1) * 100
			}
			uptime = math.Min(100, uptime*storedUses)
			effect := uptime
			if skillModList.Flag(nil, "Condition:PactMaxHit") {
				effect = 100
			}
			effectMult := effect / 100

			globalOutput["CreatePactOffensiveCalcSection"] = true
			globalOutput["PactOf"+pactKey+"Cooldown"] = cooldown
			globalOutput["PactOf"+pactKey+"CastTime"] = castTime
			globalOutput[pactKey+"EmpoweredCount"] = count
			globalOutput[pactKey+"UpTimeRatio"] = uptime

			if list := value.SkillModList.List(value.SkillCfg, pactKey+"PactDamage"); len(list) > 0 {
				tag, _ := list[0].(modparser.Tag)
				if mod, _ := tag["mod"].(*modparser.Mod); mod != nil {
					skillModList.AddMod(newMod(mod.Name, mod.Type, anyNum(mod.Value)*effectMult,
						modArgs("Uptime Scaled "+pactName, mod.Flags, mod.KeywordFlags, mod.Tags)...))
				}
			}

			switch pactKey {
			case "Beidat":
				// Coverage bonuses are averaged with the same uptime as the
				// damage bonus.
				if skillFlags["projectile"] && !skillModList.Flag(skillCfg, "NoAdditionalProjectiles") && !skillModList.Flag(skillCfg, "SingleProjectile") {
					globalOutput["BeidatAdditionalProjectiles"] = value.SkillModList.Sum("BASE", value.SkillCfg, "BeidatAdditionalProjectiles") * effectMult
					globalOutput["ProjectileCount"] = outNum(globalOutput, "ProjectileCount") + outNum(globalOutput, "BeidatAdditionalProjectiles")
				}
				if skillFlags["chaining"] && !skillFlags["projectile"] && !skillModList.Flag(skillCfg, "CannotChain") && !skillModList.Flag(skillCfg, "NoAdditionalChains") {
					globalOutput["BeidatAdditionalBeamChains"] = value.SkillModList.Sum("BASE", value.SkillCfg, "BeidatAdditionalBeamChains") * effectMult
					globalOutput["ChainMax"] = outNum(globalOutput, "ChainMax") + outNum(globalOutput, "BeidatAdditionalBeamChains")
					globalOutput["ChainMaxString"] = globalOutput["ChainMax"]
					globalOutput["ChainRemaining"] = math.Max(0, outNum(globalOutput, "ChainMax")-outNum(globalOutput, "Chain"))
				}
				if activeSkill.SkillTypes[modparser.SkillType.Cascadable] {
					globalOutput["BeidatAdditionalCascades"] = value.SkillModList.Sum("BASE", value.SkillCfg, "BeidatAdditionalCascades") * effectMult
				}
			case "Ktash":
				globalOutput["KtashSoulRefundChance"] = value.SkillModList.Sum("BASE", value.SkillCfg, "KtashPactSoulRefundChance") * effectMult
				if truthy(globalOutput["SoulGainPreventionDuration"]) {
					duration := outNum(globalOutput, "SoulGainPreventionDuration")
					durationMod := 1 + value.SkillModList.Sum("BASE", value.SkillCfg, "KtashPactSoulGainPrevention")*effectMult/100
					globalOutput["SoulGainPreventionDuration"] = math.Max(math.Ceil(duration*durationMod*d.Misc.ServerTickRate), 1) / d.Misc.ServerTickRate
				}
			}
			globalOutput[calculated] = true
		}
	}
}

// offenceRuthless ports L2907-2990: the Ruthless Blow and Fist of War
// multipliers for one pass.
func (env *Env) offenceRuthless(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output

	output["RuthlessBlowHitEffect"] = 1.0
	output["RuthlessBlowAilmentEffect"] = 1.0
	output["FistOfWarDamageEffect"] = 1.0
	if !env.ModeCombat {
		return
	}
	ruthlessEffect := "AVERAGE"
	if v := str(env.ConfigInput["ruthlessSupportMode"]); v != "" {
		ruthlessEffect = v
	}
	// Calculate Ruthless Blow chance/multipliers + Fist of War multipliers
	output["RuthlessBlowMaxCount"] = skillModList.Sum("BASE", cfg, "RuthlessBlowMaxCount")
	maxCount := outNum(output, "RuthlessBlowMaxCount")
	usedByMirage := skillCfg.SkillCond != nil && skillCfg.SkillCond["usedByMirage"]
	if maxCount > 0 && (!usedByMirage || anyNum(skillData["mirageUses"]) > maxCount) {
		switch ruthlessEffect {
		case "AVERAGE":
			output["RuthlessBlowChance"] = roundDec(100/maxCount, 0)
		case "MAX":
			output["RuthlessBlowChance"] = 100.0
			denom := maxCount
			if denom == 0 {
				denom = 1
			}
			skillData["dpsMultiplier"] = anyNum(skillData["dpsMultiplier"]) / denom
		}
	} else {
		output["RuthlessBlowChance"] = 0.0
	}
	output["RuthlessBlowHitMultiplier"] = 1 + skillModList.Sum("BASE", cfg, "RuthlessBlowHitMultiplier")/100
	output["RuthlessBlowAilmentMultiplier"] = 1 + skillModList.Sum("BASE", cfg, "RuthlessBlowAilmentMultiplier")/100
	chance := outNum(output, "RuthlessBlowChance")
	output["RuthlessBlowHitEffect"] = 1 - chance/100 + chance/100*outNum(output, "RuthlessBlowHitMultiplier")
	output["RuthlessBlowAilmentEffect"] = 1 - chance/100 + chance/100*outNum(output, "RuthlessBlowAilmentMultiplier")

	globalOutput["FistOfWarCooldown"] = skillModList.Sum("BASE", cfg, "FistOfWarCooldown")
	// If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill & NOT
	// used by mirage or other
	if outNum(globalOutput, "FistOfWarCooldown") != 0 && activeSkill.SkillTypes[modparser.SkillType.Slam] &&
		!activeSkill.SkillTypes[modparser.SkillType.Vaal] && !activeSkill.SkillTypes[modparser.SkillType.OtherThingUsesSkill] {
		env.offenceFistOfWar(c, pass)
	}
}

var _ = modstore.Cfg{}
