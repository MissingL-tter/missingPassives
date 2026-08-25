// Port of calcs.perform (CalcPerform.lua L1252-3718): the perform BODY,
// stopping where the reference hands off to defence/offence (which the
// dump stubs out for the checkpoint). Corpus-unreachable specials panic.
//
// String-keyed map iterations mirror the dump's sorted-pairs injection
// (numeric asc, then string asc); table-keyed sets iterate in a fixed
// order where the reference's raw order is immaterial.
package calc

import (
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

func sortedListKeysOf(m map[string]*modstore.List) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Perform ports calcs.perform with the defence/offence handoff excluded --
// the shape the archive dump stubs out and every staged checkpoint compares
// against. PerformFull is the one that runs the handoff for real.
func (env *Env) Perform() {
	env.performBody()
	env.performGemLevel()
	env.cacheData()
}

// PerformFull is calcs.perform end to end, in the reference's own order:
// body, then the handoff per actor, then the gem level/quality block, then
// cacheData. skipEHP is the reference's second argument, which
// buildActiveSkill passes true for.
func (env *Env) PerformFull(skipEHP bool) {
	env.performBody()

	// Defence/offence calculations
	env.defence(env.playerPA)
	if !skipEHP {
		env.buildDefenceEstimations(env.playerPA)
	}
	env.RunTriggersPlayer()
	env.RunOffencePlayer()

	if env.Minion != nil {
		env.defence(env.minionPA)
		if !skipEHP {
			env.buildDefenceEstimations(env.minionPA)
		}
		env.RunTriggersMinion()
		env.RunOffenceMinion()
	}

	env.performGemLevel()
	env.cacheData()
}

func (env *Env) performBody() {
	modDB := env.ModDB
	enemyDB := env.EnemyDB
	d := env.Data

	// Merge keystone modifiers
	env.Keystone.KeystonesAdded = map[string]bool{}
	modstore.MergeKeystones(&env.Keystone, modDB)

	// Build minion skills
	for _, activeSkill := range env.PlayerActiveSkills {
		newList := modstore.NewList(activeSkill.BaseSkillModList)
		activeSkill.SkillModList = newList
		if activeSkill.Minion != nil {
			minion := activeSkill.Minion
			minion.DB = modstore.NewDB(nil)
			minion.Ms = &modstore.Actor{
				DB:       minion.DB,
				Level:    minion.Level,
				Output:   map[string]any{},
				ItemList: map[string]modstore.Item{},
			}
			// minion actor wiring: parent/enemy set at initMinionModDB time
			// in spirit; the store-facing links are enough for eval
			minion.DB.Actor = minion.Ms
			if minion.Hostile {
				minion.Ms.Enemy = env.Player
			} else {
				minion.Ms.Enemy = env.Enemy
				minion.Ms.ParentActor = env.Player
				minion.Ms.Player = env.Player
			}
			minion.Ms.WeaponData1 = minion.WeaponData1
			minion.Ms.WeaponData2 = minion.WeaponData2
			for k, v := range minion.ItemList {
				minion.Ms.ItemList[k] = v
			}
			env.createMinionSkills(activeSkill)
			activeSkill.SkillPartName = activeSkill.Minion.MainSkill.ActiveEffect.GrantedEffect.Name
		}
	}

	playerOutput := map[string]any{}
	env.Player.Output = playerOutput
	env.Enemy.Output = map[string]any{}
	output := playerOutput

	env.playerPA = &performActor{
		ms: env.Player, db: modDB, output: playerOutput,
		mainSkill: env.PlayerMainSkill, skills: env.PlayerActiveSkills,
	}
	env.enemyPA = &performActor{ms: env.Enemy, db: enemyDB, output: env.Enemy.Output}
	env.playerPA.enemy = env.enemyPA
	env.enemyPA.enemy = env.playerPA

	// party members: nil for ladder replays (partyTab.actor empty)

	// Calculator passes reuse the environment after the shield is removed
	if env.AegisModList != nil {
		if w2 := env.Player.ItemList["Weapon 2"]; w2 != nil {
			env.aegisItem = w2
		}
	}
	env.Minion = env.PlayerMainSkill.Minion
	if env.Minion != nil {
		minionOutput := map[string]any{}
		output["Minion"] = minionOutput
		env.initMinionModDB(env.PlayerMainSkill, minionOutput)
		env.minionPA = &performActor{
			ms: env.Minion.Ms, db: env.Minion.DB, output: minionOutput,
			mainSkill: env.Minion.MainSkill, skills: env.Minion.ActiveSkillList,
			minion: env.Minion,
		}
		env.Minion.Output = minionOutput
		env.Minion.Ms.Output = minionOutput
		if env.Minion.Hostile {
			env.minionPA.enemy = env.playerPA
		} else {
			env.minionPA.enemy = env.enemyPA
			env.minionPA.parent = env.playerPA
		}
	} else {
		env.minionPA = nil
	}
	if env.AegisModList != nil {
		delete(env.Player.ItemList, "Weapon 2")
	}
	if modDB.Flag(nil, "AlchemistsGenius") {
		effectMod := 1 + modDB.Sum("INC", nil, "BuffEffectOnSelf")/100
		modDB.AddMod(newMod("FlaskEffect", "INC", math.Floor(10*effectMod), "Alchemist's Genius"))
		modDB.AddMod(newMod("FlaskChargesGained", "INC", math.Floor(20*effectMod), "Alchemist's Genius"))
	}

	hasGuaranteedBonechill := false

	// Banners
	if modDB.Flag(nil, "Condition:BannerPlanted") {
		max := modDB.Sum("BASE", nil, "MaximumValour")
		stacks := modDB.Sum("BASE", nil, "Multiplier:ValourStacks")
		modDB.AddMod(newMod("Multiplier:BannerValour", "BASE", math.Min(stacks, max), "Base"))
	}

	if modDB.Flag(nil, "CryWolfMinimumPower") && modDB.Sum("BASE", nil, "WarcryPower") < 10 {
		modDB.AddMod(newMod("WarcryPower", "OVERRIDE", 10.0, "Minimum Warcry Power from CryWolf"))
	}
	if modDB.Flag(nil, "WarcryInfinitePower") {
		modDB.AddMod(newMod("WarcryPower", "OVERRIDE", 999999.0, "Warcries have infinite power"))
	}
	output["WarcryPower"] = overrideOr(modDB, "WarcryPower", modDB.Sum("BASE", nil, "WarcryPower"))
	modDB.Multipliers["WarcryPower"] = outNum(output, "WarcryPower")

	applyEnemyModifiers(env.playerPA, true)
	if env.minionPA != nil {
		applyEnemyModifiers(env.minionPA, true)
	}
	applyEnemyModifiers(env.enemyPA, true)
	minionCounts := map[string]*struct {
		total, nonVaal, permanent float64
		hasNonVaal, hasPermanent  bool
	}{}

	for _, activeSkill := range env.PlayerActiveSkills {
		if activeSkill.SkillTypes[modparser.SkillType.Brand] {
			attachLimit := math.Min(activeSkill.SkillModList.Sum("BASE", activeSkill.SkillCfg, "BrandsAttachedLimit"),
				activeSkill.SkillModList.Sum("BASE", activeSkill.SkillCfg, "ActiveBrandLimit"))
			configured := modDB.Sum("BASE", nil, "Multiplier:ConfigBrandsAttachedToEnemy")
			attached := attachLimit
			if configured > 0 {
				attached = math.Min(configured, attachLimit)
			}
			activeSkill.SkillData["attachedBrandCount"] = attached
			activeBrands := modDB.Sum("BASE", nil, "Multiplier:ConfigActiveBrands")
			modDB.Multipliers["ActiveBrand"] = math.Max(math.Min(activeBrands, activeSkill.SkillModList.Sum("BASE", activeSkill.SkillCfg, "ActiveBrandLimit")), modDB.Multipliers["ActiveBrand"])
			modDB.Multipliers["BrandsAttachedToEnemy"] = math.Max(attached, modDB.Multipliers["BrandsAttachedToEnemy"])
			enemyDB.Multipliers["BrandsAttached"] = math.Max(attached, enemyDB.Multipliers["BrandsAttached"])
		}
		if activeSkill.SkillFlags["totem"] {
			limit := env.PlayerMainSkill.SkillModList.Sum("BASE", env.PlayerMainSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
			output["ActiveTotemLimit"] = math.Max(limit, outNum(output, "ActiveTotemLimit"))
			output["TotemsSummoned"] = overrideOr(modDB, "TotemsSummoned", outNum(output, "ActiveTotemLimit"))
			enemyDB.Multipliers["TotemsSummoned"] = math.Max(outNum(output, "TotemsSummoned"), enemyDB.Multipliers["TotemsSummoned"])
		}
		// #EVAL the reference's trailing `and Sum(...,"MaxDoom")` is a bare
		// number, so it is always truthy and gates nothing
		if activeSkill.SkillFlags["hex"] && activeSkill.SkillFlags["curse"] &&
			!activeSkill.SkillTypes[modparser.SkillType.TotemCastsWhenNotDetached] {
			hexDoom := modDB.Sum("BASE", nil, "Multiplier:HexDoomStack")
			maxDoom := activeSkill.SkillModList.Sum("BASE", nil, "MaxDoom")
			doomEffect := activeSkill.SkillModList.More(nil, "DoomEffect")
			output["HexDoomLimit"] = math.Max(maxDoom, outNum(output, "HexDoomLimit"))
			activeSkill.SkillModList.AddMod(newMod("CurseEffect", "INC", math.Min(hexDoom, maxDoom)*doomEffect, "Doom"))
			modDB.Multipliers["HexDoom"] = math.Min(math.Max(hexDoom, modDB.Multipliers["HexDoom"]), outNum(output, "HexDoomLimit"))
		}
		geName := activeSkill.ActiveEffect.GrantedEffect.Name
		if geName == "Vaal Lightning Trap" || geName == "Shock Ground" {
			effect := activeSkill.SkillModList.Sum("BASE", nil, "ShockedGroundEffect") * (1 + activeSkill.SkillModList.Sum("INC", nil, "EnemyShockEffect")/100)
			modDB.AddMod(newMod("ShockOverride", "BASE", effect, "Shocked Ground", modparser.Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnShockedGround"}))
		}
		if truthy(activeSkill.SkillData["supportBonechill"]) &&
			(activeSkill.SkillTypes[modparser.SkillType.ChillingArea] || activeSkill.SkillTypes[modparser.SkillType.NonHitChill] || !activeSkill.SkillModList.Flag(nil, "CannotChill")) {
			output["HasBonechill"] = true
		}
		if geName == "Summon Skitterbots" {
			skitterbotAilmentEffect := activeSkill.SkillModList.Sum("INC", nil, "SkitterbotAilmentEffect")
			skillSrcCfg := &modstore.Cfg{Source: "Skill"}
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotShock") {
				effect := *d.NonDamagingAilment["Shock"].Default * (1 + (activeSkill.SkillModList.Sum("INC", skillSrcCfg, "EnemyShockEffect")+skitterbotAilmentEffect)/100)
				modDB.AddMod(newMod("ShockOverride", "BASE", effect, geName))
				enemyDB.AddMod(newMod("Condition:Shocked", "FLAG", true, geName))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					modDB.AddMod(newMod("Shock", "FLAG", true, geName))
					modDB.AddMod(newMod("SelfShockOverride", "BASE", effect, geName))
				}
			}
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotChill") {
				effect := *d.NonDamagingAilment["Chill"].Default * (1 + (activeSkill.SkillModList.Sum("INC", skillSrcCfg, "EnemyChillEffect")+skitterbotAilmentEffect)/100)
				modDB.AddMod(newMod("ChillOverride", "BASE", effect, geName))
				enemyDB.AddMod(newMod("Condition:Chilled", "FLAG", true, geName))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					modDB.AddMod(newMod("Chill", "FLAG", true, geName))
					modDB.AddMod(newMod("SelfChillOverride", "BASE", effect, geName))
				}
				if truthy(activeSkill.SkillData["supportBonechill"]) {
					hasGuaranteedBonechill = true
					modDB.AddMod(newMod("SkitterbotBonechill", "FLAG", true, geName))
				}
			}
			if activeSkill.SkillModList.Flag(nil, "ScorchingSkitterbot") {
				effect := *d.NonDamagingAilment["Scorch"].Default * (1 + (activeSkill.SkillModList.Sum("INC", skillSrcCfg, "EnemyScorchEffect")+skitterbotAilmentEffect)/100)
				modDB.AddMod(newMod("ScorchOverride", "BASE", effect, geName))
				enemyDB.AddMod(newMod("Condition:Scorched", "FLAG", true, geName))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					modDB.AddMod(newMod("Scorch", "FLAG", true, geName))
					modDB.AddMod(newMod("SelfScorchOverride", "BASE", effect, geName))
				}
			}
		} else if activeSkill.SkillTypes[modparser.SkillType.ChillingArea] ||
			(activeSkill.SkillTypes[modparser.SkillType.NonHitChill] && !activeSkill.SkillModList.Flag(nil, "CannotChill")) {
			effect := *d.NonDamagingAilment["Chill"].Default * Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "EnemyChillEffect")
			modDB.AddMod(newMod("ChillOverride", "BASE", effect, geName))
			enemyDB.AddMod(newMod("Condition:Chilled", "FLAG", true, geName))
			if truthy(activeSkill.SkillData["supportBonechill"]) {
				hasGuaranteedBonechill = true
			}
		}
		// Count active minions
		if !activeSkill.SkillFlags["disable"] && len(activeSkill.MinionList) > 0 {
			ge := activeSkill.ActiveEffect.GrantedEffect
			for _, minionType := range activeSkill.MinionList {
				minionData := d.Minions[minionType]
				if minionData != nil && !truthy(minionData.Hostile) {
					key := minionData.Limit
					if key == "" {
						key = ge.Id + ":" + minionType
					}
					count := 1.0
					if minionData.Limit != "" {
						if ov := modDB.Override(nil, minionData.Limit); truthy(ov) {
							count = math.Floor(anyNum(ov))
						} else {
							count = math.Floor(Val(activeSkill.SkillModList, minionData.Limit, nil) * activeSkill.SkillModList.More(activeSkill.SkillCfg, "ActiveMinionLimit"))
						}
						output[minionData.Limit] = math.Max(count, outNum(output, minionData.Limit))
					}
					counts := minionCounts[key]
					if counts == nil {
						counts = &struct {
							total, nonVaal, permanent float64
							hasNonVaal, hasPermanent  bool
						}{}
						minionCounts[key] = counts
					}
					counts.total = math.Max(count, counts.total)
					if !activeSkill.SkillTypes[modparser.SkillType.Vaal] {
						counts.nonVaal = math.Max(count, counts.nonVaal)
						counts.hasNonVaal = true
					}
					if activeSkill.SkillFlags["permanentMinion"] {
						counts.permanent = math.Max(count, counts.permanent)
						counts.hasPermanent = true
					}
				}
			}
		}
		if activeSkill.SkillTypes[modparser.SkillType.CreatesMinion] && !activeSkill.SkillTypes[modparser.SkillType.MinionsAreUndamagable] {
			modDB.AddMod(newMod("Condition:HaveDamageableMinion", "FLAG", true))
		}
		if env.ModeBuffs && activeSkill.SkillFlags["warcry"] {
			if geName == "Rallying Cry" && !activeSkill.SkillModList.Flag(nil, "CannotShareWarcryBuffs") && !modDB.Flag(nil, "RallyingActive") {
				modDB.AddMod(newMod("RallyingExertMoreDamagePerAlly", "BASE", activeSkill.SkillModList.Sum("BASE", env.PlayerMainSkill.SkillCfg, "RallyingCryExertDamageBonus")))
				modDB.AddMod(newMod("RallyingActive", "FLAG", true))
			} else if geName == "Seismic Cry" && !modDB.Flag(nil, "SeismicActive") {
				modDB.AddMod(newMod("SeismicMoreAoE", "BASE", activeSkill.SkillModList.Sum("BASE", env.PlayerMainSkill.SkillCfg, "SeismicAoEMoreMultiplier")))
				modDB.AddMod(newMod("SeismicActive", "FLAG", true))
			}
		}
		if truthy(activeSkill.SkillData["triggeredOnDeath"]) && !activeSkill.SkillFlags["minion"] {
			activeSkill.SkillData["triggered"] = true
			for _, typ := range []string{"INC", "MORE"} {
				for _, value := range activeSkill.SkillModList.Tabulate(typ, env.PlayerMainSkill.SkillCfg, "TriggeredDamage") {
					args := []any{value.Mod.Source, value.Mod.Flags, value.Mod.KeywordFlags}
					args = append(args, value.Mod.Tags...)
					activeSkill.SkillModList.AddMod(newMod("Damage", typ, value.Mod.Value, args...))
				}
			}
			activeSkill.SkillData["triggerTime"] = 60.0 * 1000
		}
		// (The Saviour infoMessage is UI-only)
	}

	summonedMinions := 0.0
	countKeys := make([]string, 0, len(minionCounts))
	for k := range minionCounts {
		countKeys = append(countKeys, k)
	}
	sort.Strings(countKeys)
	for _, k := range countKeys {
		counts := minionCounts[k]
		summonedMinions += counts.total
		modDB.AddMod(newMod("Multiplier:SummonedMinion", "BASE", counts.total, "Config", modparser.Tag{"type": "Condition", "var": "Combat"}))
		if counts.hasNonVaal {
			modDB.AddMod(newMod("Multiplier:NonVaalSummonedMinion", "BASE", counts.nonVaal, "Config", modparser.Tag{"type": "Condition", "var": "Combat"}))
		}
		if counts.hasPermanent {
			modDB.AddMod(newMod("Multiplier:PermanentMinion", "BASE", counts.permanent, "Config", modparser.Tag{"type": "Condition", "var": "Combat"}))
		}
	}

	// Companionship only works while exactly one minion is summoned
	summonedForOnly := summonedMinions
	summonedMinionOverride := anyNum(env.ConfigInput["multiplierSummonedMinion"])
	if summonedMinionOverride != 0 {
		summonedForOnly = summonedMinionOverride
	}
	modDB.Conditions["OnlyMinion"] = summonedForOnly == 1

	// Special Rarity / Quantity Calc for Bisco's
	lootQuantityNormalEnemies := modDB.Sum("INC", nil, "LootQuantityNormalEnemies")
	if lootQuantityNormalEnemies > 0 {
		output["LootQuantityNormalEnemies"] = lootQuantityNormalEnemies + modDB.Sum("INC", nil, "LootQuantity")
	} else {
		output["LootQuantityNormalEnemies"] = 0.0
	}
	lootRarityMagicEnemies := modDB.Sum("INC", nil, "LootRarityMagicEnemies")
	if lootRarityMagicEnemies > 0 {
		output["LootRarityMagicEnemies"] = lootRarityMagicEnemies + modDB.Sum("INC", nil, "LootRarity")
	} else {
		output["LootRarityMagicEnemies"] = 0.0
	}

	// (breakdown module: CALCS-mode only)

	if modDB.Flag(nil, "ConvertArmourESToLife") {
		for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
			item, _ := env.Player.ItemList[slot].(*Item)
			if item != nil && item.In.ArmourData != nil {
				energyShieldBase := anyNum(item.In.ArmourData["EnergyShield"])
				if energyShieldBase > 0 {
					modDB.AddMod(newMod("Life", "BASE", energyShieldBase, slot+" ES to Life Conversion"))
				}
			}
		}
	}

	for _, element := range []string{"Lightning", "Fire", "Cold", "Chaos", "Physical"} {
		if modDB.Flag(nil, element+"DamageAppliesTo"+element+"AuraEffect") {
			// Damage to Aura Effect conversion from Breach rings
			multiplier := modDB.Sum("BASE", nil, "Improved"+element+"DamageAppliesTo"+element+"AuraEffect")
			if multiplier == 0 {
				multiplier = 100
			}
			multiplier = multiplier / 100
			limit, hasLimit := modDB.Max(nil, element+"DamageAppliesTo"+element+"AuraEffectLimit")
			totalConverted := 0.0
			for _, value := range modDB.Tabulate("INC", &modstore.Cfg{}, element+"Damage") {
				mod := value.Mod
				modifiers := GetConvertedModTags(mod, multiplier, true)
				converted := math.Floor(anyNum(mod.Value) * multiplier)
				if hasLimit && converted > 0 {
					remaining := limit - totalConverted
					if remaining <= 0 {
						converted = 0
					} else {
						converted = math.Min(converted, remaining)
						totalConverted += converted
					}
				}
				if converted != 0 {
					kwFlag := keywordFlagByName[element]
					args := []any{mod.Source, mod.Flags, kwFlag}
					args = append(args, modifiers...)
					modDB.AddMod(newMod("AuraEffect", mod.Type, converted, args...))
				}
			}
		}
	}

	if modDB.Flag(nil, "MinionLifeAppliesToPlayer") {
		// Minion Life conversion from Rigwald's Hunt
		multiplier := 100.0
		if v, ok := modDB.Max(nil, "ImprovedMinionLifeAppliesToPlayer"); ok {
			multiplier = v
		}
		multiplier = multiplier / 100
		for _, v := range modDB.List(nil, "MinionModifier") {
			tag, _ := v.(modparser.Tag)
			mod, _ := tag["mod"].(*modparser.Mod)
			if mod != nil && mod.Name == "Life" && mod.Type == "INC" {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				args := []any{mod.Source, mod.Flags, mod.KeywordFlags}
				args = append(args, modifiers...)
				modDB.AddMod(newMod("Life", "INC", math.Floor(anyNum(mod.Value)*multiplier), args...))
			}
		}
	}

	// Special handling of Mageblood
	maxLeftActiveMagicUtilityCount := modDB.Sum("BASE", nil, "LeftActiveMagicUtilityFlasks")
	maxRightActiveMagicUtilityCount := modDB.Sum("BASE", nil, "RightActiveMagicUtilityFlasks")
	if maxLeftActiveMagicUtilityCount > 0 || maxRightActiveMagicUtilityCount > 0 {
		var magicUtilityFlasks []*Item
		for _, slot := range env.Build.ItemsTab.Slots {
			if slot.ItemID == nil {
				continue
			}
			item := env.ItemPool[int(*slot.ItemID)]
			if item != nil && item.In.Type == "Flask" && item.In.Rarity == "MAGIC" {
				bn := deref(item.In.BaseName)
				if !strings.Contains(bn, "Life Flask") && !strings.Contains(bn, "Mana Flask") && !strings.Contains(bn, "Hybrid Flask") {
					magicUtilityFlasks = append(magicUtilityFlasks, item)
				}
			}
		}
		if maxLeftActiveMagicUtilityCount > 0 {
			for i := 0; i < int(math.Min(maxLeftActiveMagicUtilityCount, float64(len(magicUtilityFlasks)))); i++ {
				env.Flasks[magicUtilityFlasks[i]] = true
			}
		}
		if maxRightActiveMagicUtilityCount > 0 {
			for i := len(magicUtilityFlasks) - 1; i >= int(math.Max(float64(len(magicUtilityFlasks))-maxRightActiveMagicUtilityCount, 1))-1; i-- {
				env.Flasks[magicUtilityFlasks[i]] = true
			}
		}
	}

	nonUniqueFlasksApplyToMinion := env.Minion != nil && env.Minion.DB.Flag(nil, "ParentNonUniqueFlasksAppliedToYou")

	if env.ModeCombat {
		// 2 steps to account for effects affecting life recovery from flasks
		env.mergeFlasks(env.Flasks, false, true, nonUniqueFlasksApplyToMinion)
		// Merge keystones again to catch any added by flasks
		modstore.MergeKeystones(&env.Keystone, modDB)
	}

	// Calculate attributes and life/mana pools
	env.doActorAttribsConditions(env.playerPA)
	env.doActorLifeMana(env.playerPA)
	if env.Minion != nil {
		if env.Minion.Hostile {
			for _, value := range modDB.TabulateAll(nil, "EnemyModifier") {
				tag, _ := value.Value.(modparser.Tag)
				mod, _ := tag["mod"].(*modparser.Mod)
				if mod != nil {
					cp := modparser.CopyMod(mod)
					src := mod.Source
					if !mod.SourceSet {
						src = value.Mod.Source
					}
					env.Minion.DB.AddMod(modparser.SetSource(cp, src))
				}
			}
		}
		if !env.Minion.Hostile {
			addMinionModifiers(env.PlayerMainSkill.SkillModList, env.PlayerMainSkill.SkillCfg, env.Minion)
			for _, v := range env.Minion.DB.List(nil, "Keystone") {
				name, _ := v.(string)
				if mods, ok := env.Build.Spec.KeystoneMap[name]; ok {
					env.Minion.DB.AddList(mods)
				}
			}
		}
		env.doActorAttribsConditions(env.minionPA)
	}

	// Calculate skill life and mana reservations
	pp := env.playerPA
	pp.reservedLifeBase = 0
	pp.reservedLifePercent = modDB.Sum("BASE", nil, "ExtraLifeReserved")
	pp.reservedManaBase = 0
	pp.reservedManaPercent = 0
	pp.uncancellableLife = modDB.Sum("BASE", nil, "ExtraLifeReserved")
	pp.uncancellableMana = modDB.Sum("BASE", nil, "ExtraManaReserved")
	for _, activeSkill := range env.PlayerActiveSkills {
		if (activeSkill.SkillTypes[modparser.SkillType.HasReservation] || truthy(activeSkill.SkillData["triggeredByAutoexertion"])) &&
			!activeSkill.SkillTypes[modparser.SkillType.ReservationBecomesCost] {
			skillModList := activeSkill.SkillModList
			skillCfg := activeSkill.SkillCfg
			mult := floorDec(skillModList.More(skillCfg, "SupportManaMultiplier"), 4)
			gel := activeSkill.ActiveEffect.GrantedEffectLevel
			type poolVals struct {
				baseFlat, basePercent float64
			}
			// skillDataOr mirrors Lua's `skillData.key or fallback`: a PRESENT
			// zero is used (0 is truthy in Lua); only nil/false fall through.
			skillDataOr := func(sd map[string]any, key string, gel *data.SkillLevel) float64 {
				if v, ok := sd[key]; ok && v != nil && v != false {
					return anyNum(v)
				}
				if v, ok := lvlExtra(gel, key); ok {
					return v
				}
				return 0
			}
			pool := map[string]*poolVals{"Mana": {}, "Life": {}}
			pool["Mana"].baseFlat = skillDataOr(activeSkill.SkillData, "manaReservationFlat", gel)
			if skillModList.Flag(skillCfg, "ManaCostGainAsReservation") && gel != nil && gel.Cost != nil {
				pool["Mana"].baseFlat = skillModList.Sum("BASE", skillCfg, "ManaCostBase") + gel.Cost["Mana"]
			}
			pool["Mana"].basePercent = skillDataOr(activeSkill.SkillData, "manaReservationPercent", gel)
			pool["Life"].baseFlat = skillDataOr(activeSkill.SkillData, "lifeReservationFlat", gel)
			if skillModList.Flag(skillCfg, "LifeCostGainAsReservation") && gel != nil && gel.Cost != nil {
				pool["Life"].baseFlat = skillModList.Sum("BASE", skillCfg, "LifeCostBase") + gel.Cost["Life"]
			}
			pool["Life"].basePercent = skillDataOr(activeSkill.SkillData, "lifeReservationPercent", gel)
			if skillModList.Flag(skillCfg, "BloodMagicReserved") {
				pool["Life"].baseFlat += pool["Mana"].baseFlat
				pool["Mana"].baseFlat = 0
				setKV(activeSkill.SkillData, "LifeReservationFlatForced", activeSkill.SkillData["ManaReservationFlatForced"])
				delete(activeSkill.SkillData, "ManaReservationFlatForced")
				pool["Life"].basePercent += pool["Mana"].basePercent
				pool["Mana"].basePercent = 0
				setKV(activeSkill.SkillData, "LifeReservationPercentForced", activeSkill.SkillData["ManaReservationPercentForced"])
				delete(activeSkill.SkillData, "ManaReservationPercentForced")
			}
			for _, name := range []string{"Life", "Mana"} { // sorted pairs
				values := pool[name]
				more := skillModList.More(skillCfg, name+"Reserved", "Reserved")
				inc := skillModList.Sum("INC", skillCfg, name+"Reserved", "Reserved")
				efficiency := math.Max(skillModList.Sum("INC", skillCfg, name+"ReservationEfficiency", "ReservationEfficiency"), -100)
				efficiencyMore := skillModList.More(skillCfg, name+"ReservationEfficiency", "ReservationEfficiency")
				if name == "Mana" {
					env.Player.ManaEfficiency = efficiency
				}
				var reservedFlat float64
				// Lua truthiness: a present 0 forces the reservation to 0
				if v, ok := activeSkill.SkillData[name+"ReservationFlatForced"]; ok && v != nil && v != false {
					reservedFlat = anyNum(v)
				} else {
					baseFlatVal := math.Floor(values.baseFlat * mult)
					if more > 0 && inc > -100 && baseFlatVal != 0 {
						reservedFlat = math.Max(roundDec(baseFlatVal*(100+inc)/100*more/(1+efficiency/100)/efficiencyMore, 0), 0)
					}
				}
				var reservedPercent float64
				if v, ok := activeSkill.SkillData[name+"ReservationPercentForced"]; ok && v != nil && v != false {
					reservedPercent = anyNum(v)
				} else {
					basePercentVal := values.basePercent * mult
					if more > 0 && inc > -100 && basePercentVal != 0 {
						reservedPercent = math.Max(roundDec(basePercentVal*(100+inc)/100*more/(1+efficiency/100)/efficiencyMore, 2), 0)
					}
				}
				if activeSkill.ActiveMineCount != nil {
					reservedFlat = reservedFlat * *activeSkill.ActiveMineCount
					reservedPercent = reservedPercent * *activeSkill.ActiveMineCount
				}
				if activeSkill.SkillCfg.SkillName == "Blood Sacrament" && activeSkill.ActiveStageCount != nil {
					reservedFlat = reservedFlat * (*activeSkill.ActiveStageCount + 1)
					reservedPercent = reservedPercent * (*activeSkill.ActiveStageCount + 1)
				}
				if reservedFlat != 0 {
					activeSkill.SkillData[name+"ReservedBase"] = reservedFlat
					if name == "Life" {
						pp.reservedLifeBase += reservedFlat
					} else {
						pp.reservedManaBase += reservedFlat
					}
				}
				if reservedPercent != 0 {
					activeSkill.SkillData[name+"ReservedPercent"] = reservedPercent
					activeSkill.SkillData[name+"ReservedBase"] = reservedFlat + math.Ceil(outNum(output, name)*reservedPercent/100)
					if name == "Life" {
						pp.reservedLifePercent += reservedPercent
					} else {
						pp.reservedManaPercent += reservedPercent
					}
				}
				if skillModList.Flag(skillCfg, "HasUncancellableReservation") {
					if name == "Life" {
						pp.uncancellableLife += reservedPercent
					} else {
						pp.uncancellableMana += reservedPercent
					}
				}
			}
		}
	}

	// Set the life/mana reservations
	env.doActorLifeManaReservation(env.playerPA, !modDB.Flag(nil, "ManaIncreasedByOvercappedLightningRes"))

	if env.ModeCombat {
		env.mergeTinctures(env.Tinctures)
	}

	// Process attribute requirements
	{
		reqMult := Mod(modDB, nil, "GlobalAttributeRequirements")
		omniRequirements := 0.0
		hasOmniReq := modDB.Flag(nil, "OmniscienceRequirements")
		if hasOmniReq {
			omniRequirements = Mod(modDB, nil, "OmniAttributeRequirements")
		}
		ignoreAttrReq := modDB.Flag(nil, "IgnoreAttributeRequirements")
		attrTable := []string{"Str", "Dex", "Int"}
		if hasOmniReq {
			attrTable = []string{"Omni", "Str", "Dex", "Int"}
		}
		for _, attr := range attrTable {
			breakdownAttr := attr
			if hasOmniReq {
				breakdownAttr = "Omni"
			}
			outVal := 0.0
			reqOf := func(r ItemRequirement) float64 {
				switch attr {
				case "Str", "Omni":
					if attr == "Omni" {
						return 0 // Omni entries don't exist on requirement sources
					}
					return r.Str
				case "Dex":
					return r.Dex
				case "Int":
					return r.Int
				}
				return 0
			}
			for _, reqSource := range env.RequirementsTableItems {
				val := reqOf(reqSource)
				if val > 0 {
					req := math.Floor(val * reqMult)
					if hasOmniReq {
						omniReqMult := 1 / (omniRequirements - 1)
						attributereq := math.Floor(val * reqMult)
						req = math.Floor(attributereq * omniReqMult)
					}
					if req > outVal {
						outVal = req
					}
				}
			}
			gemReqOf := func(r GemRequirement) float64 {
				switch attr {
				case "Str":
					return r.Str
				case "Dex":
					return r.Dex
				case "Int":
					return r.Int
				}
				return 0
			}
			for _, reqSource := range env.RequirementsTableGems {
				val := gemReqOf(reqSource)
				if val > 0 {
					req := math.Floor(val * reqMult)
					if hasOmniReq {
						omniReqMult := 1 / (omniRequirements - 1)
						attributereq := math.Floor(val * reqMult)
						req = math.Floor(attributereq * omniReqMult)
					}
					if req > outVal {
						outVal = req
					}
				}
			}
			if ignoreAttrReq {
				outVal = 0
			}
			output["Req"+attr+"String"] = 0.0
			if outVal > outNum(output, "Req"+breakdownAttr) {
				output["Req"+breakdownAttr+"String"] = outVal
				output["Req"+breakdownAttr] = outVal
			}
		}
	}

	// Count active heralds and self-affecting auras
	if env.ModeBuffs {
		heraldList := map[string]bool{}
		auraList := map[string]bool{}
		for _, activeSkill := range env.PlayerActiveSkills {
			if activeSkill.SkillTypes[modparser.SkillType.Herald] && !heraldList[activeSkill.SkillCfg.SkillName] {
				heraldList[activeSkill.SkillCfg.SkillName] = true
				modDB.Multipliers["Herald"] = modDB.Multipliers["Herald"] + 1
				modDB.Conditions["AffectedByHerald"] = true
			} else if activeSkill.SkillTypes[modparser.SkillType.Aura] && !activeSkill.SkillTypes[modparser.SkillType.AuraAffectsEnemies] &&
				!truthy(activeSkill.SkillData["auraCannotAffectSelf"]) && !auraList[activeSkill.SkillCfg.SkillName] {
				auraList[activeSkill.SkillCfg.SkillName] = true
				modDB.Multipliers["AuraAffectingSelf"] = modDB.Multipliers["AuraAffectingSelf"] + 1
			}
		}
	}

	if modDB.Flag(nil, "ManaAppliesToShockEffect") {
		// Maximum Mana conversion from Lightning Mastery
		multiplier := 100.0
		if v, ok := modDB.Max(nil, "ImprovedManaAppliesToShockEffect"); ok {
			multiplier = v
		}
		multiplier = multiplier / 100
		for _, value := range modDB.Tabulate("INC", nil, "Mana") {
			mod := value.Mod
			modifiers := GetConvertedModTags(mod, multiplier, false)
			args := []any{mod.Source, mod.Flags, mod.KeywordFlags}
			args = append(args, modifiers...)
			modDB.AddMod(newMod("EnemyShockEffect", "INC", math.Floor(anyNum(mod.Value)*multiplier), args...))
		}
	}

	// Calculate charges early
	env.doActorCharges(env.playerPA)

	env.performBuffs(hasGuaranteedBonechill, nonUniqueFlasksApplyToMinion)
}
