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
	"github.com/MissingL-tter/missingPassives/internal/util"
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
				Output:   modstore.Output{},
				Resolver: gemIds{},
				ItemList: map[string]modstore.Item{},
				// the reference's actor.minionData is the minion table
				// itself; carry the fields its readers touch
				MinionData: &modstore.MinionData{
					MonsterTags: minion.MinionData.MonsterTags,
					DamageFixup: minion.MinionData.DamageFixup,
				},
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
			minion.Ms.WeaponData1 = weaponRef(minion.WeaponData1)
			minion.Ms.WeaponData2 = weaponRef(minion.WeaponData2)
			for k, v := range minion.ItemList {
				minion.Ms.ItemList[k] = v
			}
			env.createMinionSkills(activeSkill)
			activeSkill.SkillPartName = activeSkill.Minion.MainSkill.ActiveEffect.GrantedEffect.Name
		}
	}

	playerOutput := modstore.Output{}
	env.Player.Output = playerOutput
	enemyOutput := modstore.Output{}
	env.Enemy.Output = enemyOutput
	output := playerOutput

	env.playerPA = &performActor{
		ms: env.Player, db: modDB, output: playerOutput,
		mainSkill: env.PlayerMainSkill, skills: env.PlayerActiveSkills,
	}
	env.enemyPA = &performActor{ms: env.Enemy, db: enemyDB, output: enemyOutput}
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
		// the reference also hangs this table off output.Minion; nothing
		// ported reads it there
		minionOutput := modstore.Output{}
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
		effectMod := 1 + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")/100
		modDB.AddMod(newModS("FlaskEffect", modparser.Inc, modparser.Num(math.Floor(10*effectMod)), "Alchemist's Genius"))
		modDB.AddMod(newModS("FlaskChargesGained", modparser.Inc, modparser.Num(math.Floor(20*effectMod)), "Alchemist's Genius"))
	}

	hasGuaranteedBonechill := false

	// Banners
	if modDB.Flag(nil, "Condition:BannerPlanted") {
		max := modDB.Sum(modparser.Base, nil, "MaximumValour")
		stacks := modDB.Sum(modparser.Base, nil, "Multiplier:ValourStacks")
		modDB.AddMod(newModS("Multiplier:BannerValour", modparser.Base, modparser.Num(math.Min(stacks, max)), "Base"))
	}

	if modDB.Flag(nil, "CryWolfMinimumPower") && modDB.Sum(modparser.Base, nil, "WarcryPower") < 10 {
		modDB.AddMod(newModS("WarcryPower", modparser.Override, modparser.Num(10.0), "Minimum Warcry Power from CryWolf"))
	}
	if modDB.Flag(nil, "WarcryInfinitePower") {
		modDB.AddMod(newModS("WarcryPower", modparser.Override, modparser.Num(999999.0), "Warcries have infinite power"))
	}
	output.SetN("WarcryPower", overrideOr(modDB, "WarcryPower", modDB.Sum(modparser.Base, nil, "WarcryPower")))
	modDB.Multipliers["WarcryPower"] = output.N("WarcryPower")

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
		if activeSkill.SkillTypes[modparser.SkillTypeBrand] {
			attachLimit := math.Min(activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "BrandsAttachedLimit"),
				activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "ActiveBrandLimit"))
			configured := modDB.Sum(modparser.Base, nil, "Multiplier:ConfigBrandsAttachedToEnemy")
			attached := attachLimit
			if configured > 0 {
				attached = math.Min(configured, attachLimit)
			}
			activeSkill.SkillData.SetN("attachedBrandCount", attached)
			activeBrands := modDB.Sum(modparser.Base, nil, "Multiplier:ConfigActiveBrands")
			modDB.Multipliers["ActiveBrand"] = math.Max(math.Min(activeBrands, activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "ActiveBrandLimit")), modDB.Multipliers["ActiveBrand"])
			modDB.Multipliers["BrandsAttachedToEnemy"] = math.Max(attached, modDB.Multipliers["BrandsAttachedToEnemy"])
			enemyDB.Multipliers["BrandsAttached"] = math.Max(attached, enemyDB.Multipliers["BrandsAttached"])
		}
		if activeSkill.SkillFlags["totem"] {
			limit := env.PlayerMainSkill.SkillModList.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
			output.SetN("ActiveTotemLimit", math.Max(limit, output.N("ActiveTotemLimit")))
			output.SetN("TotemsSummoned", overrideOr(modDB, "TotemsSummoned", output.N("ActiveTotemLimit")))
			enemyDB.Multipliers["TotemsSummoned"] = math.Max(output.N("TotemsSummoned"), enemyDB.Multipliers["TotemsSummoned"])
		}
		// #EVAL the reference's trailing `and Sum(...,"MaxDoom")` is a bare
		// number, so it is always truthy and gates nothing
		if activeSkill.SkillFlags["hex"] && activeSkill.SkillFlags["curse"] &&
			!activeSkill.SkillTypes[modparser.SkillTypeTotemCastsWhenNotDetached] {
			hexDoom := modDB.Sum(modparser.Base, nil, "Multiplier:HexDoomStack")
			maxDoom := activeSkill.SkillModList.Sum(modparser.Base, nil, "MaxDoom")
			doomEffect := activeSkill.SkillModList.More(nil, "DoomEffect")
			output.SetN("HexDoomLimit", math.Max(maxDoom, output.N("HexDoomLimit")))
			activeSkill.SkillModList.AddMod(newModS("CurseEffect", modparser.Inc, modparser.Num(math.Min(hexDoom, maxDoom)*doomEffect), "Doom"))
			modDB.Multipliers["HexDoom"] = math.Min(math.Max(hexDoom, modDB.Multipliers["HexDoom"]), output.N("HexDoomLimit"))
		}
		geName := activeSkill.ActiveEffect.GrantedEffect.Name
		if geName == "Vaal Lightning Trap" || geName == "Shock Ground" {
			effect := activeSkill.SkillModList.Sum(modparser.Base, nil, "ShockedGroundEffect") * (1 + activeSkill.SkillModList.Sum(modparser.Inc, nil, "EnemyShockEffect")/100)
			modDB.AddMod(newModS("ShockOverride", modparser.Base, modparser.Num(effect), "Shocked Ground", &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "OnShockedGround"}))
		}
		if activeSkill.SkillData.Flag("supportBonechill") &&
			(activeSkill.SkillTypes[modparser.SkillTypeChillingArea] || activeSkill.SkillTypes[modparser.SkillTypeNonHitChill] || !activeSkill.SkillModList.Flag(nil, "CannotChill")) {
			output.SetFlag("HasBonechill", true)
		}
		if geName == "Summon Skitterbots" {
			skitterbotAilmentEffect := activeSkill.SkillModList.Sum(modparser.Inc, nil, "SkitterbotAilmentEffect")
			skillSrcCfg := &modstore.Cfg{Source: "Skill"}
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotShock") {
				effect := *data.NonDamagingAilment["Shock"].Default * (1 + (activeSkill.SkillModList.Sum(modparser.Inc, skillSrcCfg, "EnemyShockEffect")+skitterbotAilmentEffect)/100)
				modDB.AddMod(newModS("ShockOverride", modparser.Base, modparser.Num(effect), geName))
				enemyDB.AddMod(newModS("Condition:Shocked", modparser.Flag, modparser.Bool(true), geName))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					modDB.AddMod(newModS("Shock", modparser.Flag, modparser.Bool(true), geName))
					modDB.AddMod(newModS("SelfShockOverride", modparser.Base, modparser.Num(effect), geName))
				}
			}
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotChill") {
				effect := *data.NonDamagingAilment["Chill"].Default * (1 + (activeSkill.SkillModList.Sum(modparser.Inc, skillSrcCfg, "EnemyChillEffect")+skitterbotAilmentEffect)/100)
				modDB.AddMod(newModS("ChillOverride", modparser.Base, modparser.Num(effect), geName))
				enemyDB.AddMod(newModS("Condition:Chilled", modparser.Flag, modparser.Bool(true), geName))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					modDB.AddMod(newModS("Chill", modparser.Flag, modparser.Bool(true), geName))
					modDB.AddMod(newModS("SelfChillOverride", modparser.Base, modparser.Num(effect), geName))
				}
				if activeSkill.SkillData.Flag("supportBonechill") {
					hasGuaranteedBonechill = true
					modDB.AddMod(newModS("SkitterbotBonechill", modparser.Flag, modparser.Bool(true), geName))
				}
			}
			if activeSkill.SkillModList.Flag(nil, "ScorchingSkitterbot") {
				effect := *data.NonDamagingAilment["Scorch"].Default * (1 + (activeSkill.SkillModList.Sum(modparser.Inc, skillSrcCfg, "EnemyScorchEffect")+skitterbotAilmentEffect)/100)
				modDB.AddMod(newModS("ScorchOverride", modparser.Base, modparser.Num(effect), geName))
				enemyDB.AddMod(newModS("Condition:Scorched", modparser.Flag, modparser.Bool(true), geName))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					modDB.AddMod(newModS("Scorch", modparser.Flag, modparser.Bool(true), geName))
					modDB.AddMod(newModS("SelfScorchOverride", modparser.Base, modparser.Num(effect), geName))
				}
			}
		} else if activeSkill.SkillTypes[modparser.SkillTypeChillingArea] ||
			(activeSkill.SkillTypes[modparser.SkillTypeNonHitChill] && !activeSkill.SkillModList.Flag(nil, "CannotChill")) {
			effect := *data.NonDamagingAilment["Chill"].Default * Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "EnemyChillEffect")
			modDB.AddMod(newModS("ChillOverride", modparser.Base, modparser.Num(effect), geName))
			enemyDB.AddMod(newModS("Condition:Chilled", modparser.Flag, modparser.Bool(true), geName))
			if activeSkill.SkillData.Flag("supportBonechill") {
				hasGuaranteedBonechill = true
			}
		}
		// Count active minions
		if !activeSkill.SkillFlags["disable"] && len(activeSkill.MinionList) > 0 {
			ge := activeSkill.ActiveEffect.GrantedEffect
			for _, minionType := range activeSkill.MinionList {
				minionData := data.Minions[minionType]
				if minionData != nil && !minionData.Hostile {
					key := minionData.Limit
					if key == "" {
						key = ge.Id + ":" + minionType
					}
					count := 1.0
					if minionData.Limit != "" {
						if ov, ok := modDB.Override(nil, minionData.Limit); ok {
							count = math.Floor(valueNum(ov))
						} else {
							count = math.Floor(Val(activeSkill.SkillModList, minionData.Limit, nil) * activeSkill.SkillModList.More(activeSkill.SkillCfg, "ActiveMinionLimit"))
						}
						output.SetN(minionData.Limit, math.Max(count, output.N(minionData.Limit)))
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
					if !activeSkill.SkillTypes[modparser.SkillTypeVaal] {
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
		if activeSkill.SkillTypes[modparser.SkillTypeCreatesMinion] && !activeSkill.SkillTypes[modparser.SkillTypeMinionsAreUndamagable] {
			modDB.AddMod(newMod("Condition:HaveDamageableMinion", modparser.Flag, modparser.Bool(true)))
		}
		if env.ModeBuffs && activeSkill.SkillFlags["warcry"] {
			if geName == "Rallying Cry" && !activeSkill.SkillModList.Flag(nil, "CannotShareWarcryBuffs") && !modDB.Flag(nil, "RallyingActive") {
				modDB.AddMod(newMod("RallyingExertMoreDamagePerAlly", modparser.Base, modparser.Num(activeSkill.SkillModList.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, "RallyingCryExertDamageBonus"))))
				modDB.AddMod(newMod("RallyingActive", modparser.Flag, modparser.Bool(true)))
			} else if geName == "Seismic Cry" && !modDB.Flag(nil, "SeismicActive") {
				modDB.AddMod(newMod("SeismicMoreAoE", modparser.Base, modparser.Num(activeSkill.SkillModList.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, "SeismicAoEMoreMultiplier"))))
				modDB.AddMod(newMod("SeismicActive", modparser.Flag, modparser.Bool(true)))
			}
		}
		if activeSkill.SkillData.Flag("triggeredOnDeath") && !activeSkill.SkillFlags["minion"] {
			activeSkill.SkillData.SetFlag("triggered", true)
			for _, typ := range []modparser.ModType{modparser.Inc, modparser.More} {
				for _, value := range activeSkill.SkillModList.Tabulate(typ, env.PlayerMainSkill.SkillCfg, "TriggeredDamage") {
					m := value.Mod
					activeSkill.SkillModList.AddMod(modparser.NewModFull("Damage", typ, m.Value, m.Source, m.SourceSet, m.Flags, m.KeywordFlags, m.Tags...))
				}
			}
			activeSkill.SkillData.SetN("triggerTime", 60.0*1000)
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
		modDB.AddMod(newModS("Multiplier:SummonedMinion", modparser.Base, modparser.Num(counts.total), "Config", &modparser.CondTag{Var: "Combat"}))
		if counts.hasNonVaal {
			modDB.AddMod(newModS("Multiplier:NonVaalSummonedMinion", modparser.Base, modparser.Num(counts.nonVaal), "Config", &modparser.CondTag{Var: "Combat"}))
		}
		if counts.hasPermanent {
			modDB.AddMod(newModS("Multiplier:PermanentMinion", modparser.Base, modparser.Num(counts.permanent), "Config", &modparser.CondTag{Var: "Combat"}))
		}
	}

	// Companionship only works while exactly one minion is summoned
	summonedForOnly := summonedMinions
	summonedMinionOverride := env.ConfigInput.MultiplierSummonedMinion
	if summonedMinionOverride != 0 {
		summonedForOnly = summonedMinionOverride
	}
	modDB.Conditions.Set("OnlyMinion", summonedForOnly == 1)

	// Special Rarity / Quantity Calc for Bisco's
	lootQuantityNormalEnemies := modDB.Sum(modparser.Inc, nil, "LootQuantityNormalEnemies")
	if lootQuantityNormalEnemies > 0 {
		output.SetN("LootQuantityNormalEnemies", lootQuantityNormalEnemies+modDB.Sum(modparser.Inc, nil, "LootQuantity"))
	} else {
		output.SetN("LootQuantityNormalEnemies", 0.0)
	}
	lootRarityMagicEnemies := modDB.Sum(modparser.Inc, nil, "LootRarityMagicEnemies")
	if lootRarityMagicEnemies > 0 {
		output.SetN("LootRarityMagicEnemies", lootRarityMagicEnemies+modDB.Sum(modparser.Inc, nil, "LootRarity"))
	} else {
		output.SetN("LootRarityMagicEnemies", 0.0)
	}

	// (breakdown module: CALCS-mode only)

	if modDB.Flag(nil, "ConvertArmourESToLife") {
		for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
			item, _ := env.Player.ItemList[slot].(*Item)
			if item != nil && item.In.ArmourData != nil {
				energyShieldBase := armourDataOf(item, "EnergyShield")
				if energyShieldBase > 0 {
					modDB.AddMod(newModS("Life", modparser.Base, modparser.Num(energyShieldBase), slot+" ES to Life Conversion"))
				}
			}
		}
	}

	for _, element := range []string{"Lightning", "Fire", "Cold", "Chaos", "Physical"} {
		if modDB.Flag(nil, element+"DamageAppliesTo"+element+"AuraEffect") {
			// Damage to Aura Effect conversion from Breach rings
			multiplier := modDB.Sum(modparser.Base, nil, "Improved"+element+"DamageAppliesTo"+element+"AuraEffect")
			if multiplier == 0 {
				multiplier = 100
			}
			multiplier = multiplier / 100
			limit, hasLimit := modDB.Max(nil, element+"DamageAppliesTo"+element+"AuraEffectLimit")
			totalConverted := 0.0
			for _, value := range modDB.Tabulate(modparser.Inc, &modstore.Cfg{}, element+"Damage") {
				mod := value.Mod
				modifiers := GetConvertedModTags(mod, multiplier, true)
				converted := math.Floor(valueNum(mod.Value) * multiplier)
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
					kwFlag := modparser.KeywordFlagByName[element]
					modDB.AddMod(modparser.NewModFull("AuraEffect", mod.Type, modparser.Num(converted), mod.Source, mod.SourceSet, mod.Flags, kwFlag, modifiers...))
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
			mod := modRefOf(v)
			if mod != nil && mod.Name == "Life" && mod.Type == modparser.Inc {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				modDB.AddMod(modparser.NewModFull("Life", modparser.Inc, modparser.Num(math.Floor(valueNum(mod.Value)*multiplier)), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, modifiers...))
			}
		}
	}

	// Special handling of Mageblood
	maxLeftActiveMagicUtilityCount := modDB.Sum(modparser.Base, nil, "LeftActiveMagicUtilityFlasks")
	maxRightActiveMagicUtilityCount := modDB.Sum(modparser.Base, nil, "RightActiveMagicUtilityFlasks")
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
				mod := modRefOf(value.Value)
				if mod != nil {
					cp := cloneMod(mod)
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
				name := str(v)
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
	pp.reservedLifePercent = modDB.Sum(modparser.Base, nil, "ExtraLifeReserved")
	pp.reservedManaBase = 0
	pp.reservedManaPercent = 0
	pp.uncancellableLife = modDB.Sum(modparser.Base, nil, "ExtraLifeReserved")
	pp.uncancellableMana = modDB.Sum(modparser.Base, nil, "ExtraManaReserved")
	for _, activeSkill := range env.PlayerActiveSkills {
		if (activeSkill.SkillTypes[modparser.SkillTypeHasReservation] || activeSkill.SkillData.Flag("triggeredByAutoexertion")) &&
			!activeSkill.SkillTypes[modparser.SkillTypeReservationBecomesCost] {
			skillModList := activeSkill.SkillModList
			skillCfg := activeSkill.SkillCfg
			mult := floorDec(skillModList.More(skillCfg, "SupportManaMultiplier"), 4)
			gel := activeSkill.ActiveEffect.GrantedEffectLevel
			type poolVals struct {
				baseFlat, basePercent float64
			}
			// skillDataOr mirrors Lua's `skillData.key or fallback`: a PRESENT
			// zero is used (0 is truthy in Lua); only nil/false fall through.
			skillDataOr := func(sd *SkillData, key string, gel *data.SkillLevel) float64 {
				if v := sd.Get(key); v.Truthy() {
					return v.Num()
				}
				if v, ok := lvlExtra(gel, key); ok {
					return v
				}
				return 0
			}
			pool := map[string]*poolVals{"Mana": {}, "Life": {}}
			pool["Mana"].baseFlat = skillDataOr(activeSkill.SkillData, "manaReservationFlat", gel)
			if skillModList.Flag(skillCfg, "ManaCostGainAsReservation") && gel != nil && gel.Cost != nil {
				pool["Mana"].baseFlat = skillModList.Sum(modparser.Base, skillCfg, "ManaCostBase") + gel.Cost["Mana"]
			}
			pool["Mana"].basePercent = skillDataOr(activeSkill.SkillData, "manaReservationPercent", gel)
			pool["Life"].baseFlat = skillDataOr(activeSkill.SkillData, "lifeReservationFlat", gel)
			if skillModList.Flag(skillCfg, "LifeCostGainAsReservation") && gel != nil && gel.Cost != nil {
				pool["Life"].baseFlat = skillModList.Sum(modparser.Base, skillCfg, "LifeCostBase") + gel.Cost["Life"]
			}
			pool["Life"].basePercent = skillDataOr(activeSkill.SkillData, "lifeReservationPercent", gel)
			if skillModList.Flag(skillCfg, "BloodMagicReserved") {
				pool["Life"].baseFlat += pool["Mana"].baseFlat
				pool["Mana"].baseFlat = 0
				if sd := activeSkill.SkillData; sd.Has("ManaReservationFlatForced") {
					sd.Set("LifeReservationFlatForced", sd.Get("ManaReservationFlatForced"))
				}
				activeSkill.SkillData.Del("ManaReservationFlatForced")
				pool["Life"].basePercent += pool["Mana"].basePercent
				pool["Mana"].basePercent = 0
				if sd := activeSkill.SkillData; sd.Has("ManaReservationPercentForced") {
					sd.Set("LifeReservationPercentForced", sd.Get("ManaReservationPercentForced"))
				}
				activeSkill.SkillData.Del("ManaReservationPercentForced")
			}
			for _, name := range []string{"Life", "Mana"} { // sorted pairs
				values := pool[name]
				more := skillModList.More(skillCfg, name+"Reserved", "Reserved")
				inc := skillModList.Sum(modparser.Inc, skillCfg, name+"Reserved", "Reserved")
				efficiency := math.Max(skillModList.Sum(modparser.Inc, skillCfg, name+"ReservationEfficiency", "ReservationEfficiency"), -100)
				efficiencyMore := skillModList.More(skillCfg, name+"ReservationEfficiency", "ReservationEfficiency")
				if name == "Mana" {
					env.Player.ManaEfficiency = efficiency
				}
				var reservedFlat float64
				// Lua truthiness: a present 0 forces the reservation to 0
				if v := activeSkill.SkillData.Get(name + "ReservationFlatForced"); v.Truthy() {
					reservedFlat = v.Num()
				} else {
					baseFlatVal := math.Floor(values.baseFlat * mult)
					if more > 0 && inc > -100 && baseFlatVal != 0 {
						reservedFlat = math.Max(util.RoundHalfUp(baseFlatVal*(100+inc)/100*more/(1+efficiency/100)/efficiencyMore, 0), 0)
					}
				}
				var reservedPercent float64
				if v := activeSkill.SkillData.Get(name + "ReservationPercentForced"); v.Truthy() {
					reservedPercent = v.Num()
				} else {
					basePercentVal := values.basePercent * mult
					if more > 0 && inc > -100 && basePercentVal != 0 {
						reservedPercent = math.Max(util.RoundHalfUp(basePercentVal*(100+inc)/100*more/(1+efficiency/100)/efficiencyMore, 2), 0)
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
					activeSkill.SkillData.SetN(name+"ReservedBase", reservedFlat)
					if name == "Life" {
						pp.reservedLifeBase += reservedFlat
					} else {
						pp.reservedManaBase += reservedFlat
					}
				}
				if reservedPercent != 0 {
					activeSkill.SkillData.SetN(name+"ReservedPercent", reservedPercent)
					activeSkill.SkillData.SetN(name+"ReservedBase", reservedFlat+math.Ceil(output.N(name)*reservedPercent/100))
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
			output.SetN("Req"+attr+"String", 0.0)
			if outVal > output.N("Req"+breakdownAttr) {
				output.SetN("Req"+breakdownAttr+"String", outVal)
				output.SetN("Req"+breakdownAttr, outVal)
			}
		}
	}

	// Count active heralds and self-affecting auras
	if env.ModeBuffs {
		heraldList := map[string]bool{}
		auraList := map[string]bool{}
		for _, activeSkill := range env.PlayerActiveSkills {
			if activeSkill.SkillTypes[modparser.SkillTypeHerald] && !heraldList[activeSkill.SkillCfg.SkillName] {
				heraldList[activeSkill.SkillCfg.SkillName] = true
				modDB.Multipliers["Herald"] = modDB.Multipliers["Herald"] + 1
				modDB.Conditions.Set("AffectedByHerald", true)
			} else if activeSkill.SkillTypes[modparser.SkillTypeAura] && !activeSkill.SkillTypes[modparser.SkillTypeAuraAffectsEnemies] &&
				!activeSkill.SkillData.Flag("auraCannotAffectSelf") && !auraList[activeSkill.SkillCfg.SkillName] {
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
		for _, value := range modDB.Tabulate(modparser.Inc, nil, "Mana") {
			mod := value.Mod
			modifiers := GetConvertedModTags(mod, multiplier, false)
			modDB.AddMod(modparser.NewModFull("EnemyShockEffect", modparser.Inc, modparser.Num(math.Floor(valueNum(mod.Value)*multiplier)), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, modifiers...))
		}
	}

	// Calculate charges early
	env.doActorCharges(env.playerPA)

	env.performBuffs(hasGuaranteedBonechill, nonUniqueFlasksApplyToMinion)
}
