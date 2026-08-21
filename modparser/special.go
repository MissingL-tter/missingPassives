package modparser

// specialModList (transformed entries) — ModParser.lua:2120-5880. The
// closures needing real statements live in special_hand.go.
var specialModListData = map[string]any{
	// Explode mods
	`non*?aura curses you inflict are not removed from dying enemies`: d(),
	`enemies near corpses affected by your curses are blinded`:        []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Blinded")}, Tag{"type": "MultiplierThreshold", "var": "NearbyCorpse", "threshold": 1}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})},
	// Keystones
	`([0-9]+) rage regenerated for every ([0-9]+) mana regeneration per second`: fn(func(c caps) any {
		return []any{mod("RageRegen", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "ManaRegen", "div": c.n(2)}), flag("Condition:CanGainRage")}
	}),
	`([0-9a-zA-Z]+) recovery from regeneration is not applied`: fn(func(c caps) any { return []any{flag("UnaffectedBy" + firstToUpper(c.s(1)) + "Regen")} }),
	`([0-9]+)% less damage taken for every ([0-9]+)% life recovery per second from leech`: fn(func(c caps) any {
		return []any{mod("DamageTaken", "MORE", -c.n(1), Tag{"type": "PerStat", "stat": "MaxLifeLeechRatePercent", "div": c.n(2)})}
	}),
	`([0-9a-zA-Z]+) recovery from non-instant leech is not applied`: fn(func(c caps) any { return []any{flag("UnaffectedByNonInstant" + firstToUpper(c.s(1)) + "Leech")} }),
	`([0-9]+)% additional physical damage reduction for every ([0-9]+)% life recovery per second from leech`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageReduction", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "MaxLifeLeechRatePercent", "div": c.n(2)}, Tag{"type": "Condition", "var": "Leeching"})}
	}),
	`modifiers to chance to suppress spell damage instead apply to chance to dodge spell hits at 50% of their value`: []any{flag("ConvertSpellSuppressionToSpellDodge"), mod("SpellSuppressionChance", "OVERRIDE", 0, "Acrobatics")},
	`chance to evade hits is based off of ([0-9]+)% of your ward instead of your evasion rating`: fn(func(c caps) any {
		return []any{flag("EvadeChanceBasedOnWard"), mod("EvadeChanceBasedOnWardPercent", "OVERRIDE", c.n(1), "Black Scythe Training")}
	}),
	`physical damage reduction from hits is based off of ([0-9]+)% of your ward instead of your armour`: fn(func(c caps) any {
		return []any{flag("PhysicalReductionBasedOnWard"), mod("PhysicalReductionBasedOnWardPercent", "OVERRIDE", c.n(1), "Black Scythe Training")}
	}),
	`maximum chance to dodge spell hits is ([0-9]+)%`:             fn(func(c caps) any { return []any{mod("SpellDodgeChanceMax", "OVERRIDE", c.n(1), "Acrobatics")} }),
	`dexterity provides no bonus to evasion rating`:               []any{flag("NoDexBonusToEvasion")},
	`dexterity provides no inherent bonus to evasion rating`:      []any{flag("NoDexBonusToEvasion")},
	`strength's damage bonus applies to all spell damage as well`: []any{flag("IronWill")},
	`your hits can't be evaded`:                                   []any{flag("CannotBeEvaded")},
	`your melee hits can't be evaded while wielding a sword`:      []any{flag("CannotBeEvaded", nil, ModFlag.Melee|ModFlag.Hit, Tag{"type": "Condition", "var": "UsingSword"})},
	`minion hits can't be evaded`:                                 []any{mod("MinionModifier", "LIST", Tag{"mod": flag("CannotBeEvaded")})},
	`never deal critical strikes`:                                 []any{flag("NeverCrit"), flag("Condition:NeverCrit")},
	`minions never deal critical strikes`:                         []any{mod("MinionModifier", "LIST", Tag{"mod": flag("NeverCrit")}), mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:NeverCrit")})},
	`never deal critical strikes with spells`:                     []any{flag("NeverCrit", nil, ModFlag.Spell), flag("Condition:NeverCrit", nil, ModFlag.Spell)},
	`never deal critical strikes with attacks`:                    []any{flag("NeverCrit", nil, ModFlag.Attack), flag("Condition:NeverCrit", nil, ModFlag.Attack)},
	`cannot deal critical strikes`:                                []any{flag("NeverCrit"), flag("Condition:NeverCrit")},
	`cannot deal critical strikes with spells`:                    []any{flag("NeverCrit", nil, ModFlag.Spell), flag("Condition:NeverCrit", nil, ModFlag.Spell)},
	`cannot deal critical strikes with attacks`:                   []any{flag("NeverCrit", nil, ModFlag.Attack), flag("Condition:NeverCrit", nil, ModFlag.Attack)},
	`no critical strike multiplier`:                               []any{flag("NoCritMultiplier")},
	`ailments never count as being from critical strikes`:         []any{flag("AilmentsAreNeverFromCrit")},
	`the increase to physical damage from strength applies to projectile attacks as well as melee attacks`:                        []any{flag("IronGrip")},
	`strength's damage bonus applies to projectile attack damage as well as melee damage`:                                         []any{flag("IronGrip")},
	`converts all evasion rating to armour\. dexterity provides no bonus to evasion rating`:                                       []any{flag("NoDexBonusToEvasion"), flag("IronReflexes")},
	`30% chance to dodge attack hits\. 50% less armour, 30% less energy shield, 30% less chance to block spell and attack damage`: []any{mod("AttackDodgeChance", "BASE", 30), mod("Armour", "MORE", -50), mod("EnergyShield", "MORE", -30), mod("BlockChance", "MORE", -30), mod("SpellBlockChance", "MORE", -30)},
	`([0-9]+)% increased blind effect`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("BlindEffect", "INC", c.n(1))})}
	}),
	`([0-9]+)% increased effect of blind from melee weapons`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("BlindEffect", "INC", c.n(1))})}
	}),
	`\+([0-9]+)% chance to block spell damage for each ([0-9]+)% overcapped chance to block attack damage`: fn(func(c caps) any {
		return []any{mod("SpellBlockChance", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "BlockChanceOverCap", "div": c.n(2)})}
	}),
	`maximum life becomes 1, immune to chaos damage`:           []any{flag("ChaosInoculation"), mod("ChaosDamageTaken", "MORE", -100)},
	`life regeneration is applied to energy shield instead`:    []any{flag("ZealotsOath")},
	`life leeched per second is doubled`:                       []any{mod("LifeLeechRate", "MORE", 100)},
	`life regeneration has no effect`:                          []any{flag("NoLifeRegen")},
	`energy shield recharge instead applies to life`:           []any{flag("EnergyShieldRechargeAppliesToLife")},
	`blade vortex and blade blast deal no non-physical damage`: []any{flag("DealNoLightning", Tag{"type": "SkillName", "skillNameList": []any{"Blade Vortex", "Blade Blast"}, "includeTransfigured": true}), flag("DealNoCold", Tag{"type": "SkillName", "skillNameList": []any{"Blade Vortex", "Blade Blast"}, "includeTransfigured": true}), flag("DealNoFire", Tag{"type": "SkillName", "skillNameList": []any{"Blade Vortex", "Blade Blast"}, "includeTransfigured": true}), flag("DealNoChaos", Tag{"type": "SkillName", "skillNameList": []any{"Blade Vortex", "Blade Blast"}, "includeTransfigured": true})},
	`([0-9]+)% of physical, cold and lightning damage converted to fire damage`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageConvertToFire", "BASE", c.n(1)), mod("LightningDamageConvertToFire", "BASE", c.n(1)), mod("ColdDamageConvertToFire", "BASE", c.n(1))}
	}),
	`all elemental damage converted to chaos damage`:           []any{mod("ColdDamageConvertToChaos", "BASE", 100), mod("FireDamageConvertToChaos", "BASE", 100), mod("LightningDamageConvertToChaos", "BASE", 100)},
	`removes all mana\. spend life instead of mana for skills`: []any{mod("Mana", "MORE", -100), flag("CostLifeInsteadOfMana")},
	`removes all mana`:                                          []any{mod("Mana", "MORE", -100)},
	`removes all energy shield`:                                 []any{mod("EnergyShield", "MORE", -100)},
	`skills cost life instead of mana`:                          []any{flag("CostLifeInsteadOfMana")},
	`skills reserve life instead of mana`:                       []any{flag("BloodMagicReserved")},
	`your skills that throw mines reserve life instead of mana`: []any{flag("BloodMagicReserved", nil, 0, KeywordFlag.Mine)},
	`curse skills cost life instead of mana`:                    []any{flag("CostLifeInsteadOfMana", nil, 0, KeywordFlag.Curse)},
	`curse aura skills reserve life instead of mana`:            []any{flag("BloodMagicReserved", nil, 0, KeywordFlag.Curse, Tag{"type": "SkillType", "skillType": SkillType.Aura})},
	`your travel skills critically strike once every 3 uses`:    []any{flag("Every3UseCrit", Tag{"type": "SkillType", "skillType": SkillType.Travel})},
	`non-aura skills cost no mana or life while focus?sed`:      []any{mod("ManaCost", "MORE", -100, Tag{"type": "Condition", "var": "Focused"}, Tag{"type": "SkillType", "skillType": SkillType.Aura, "neg": true}), mod("LifeCost", "MORE", -100, Tag{"type": "Condition", "var": "Focused"}, Tag{"type": "SkillType", "skillType": SkillType.Aura, "neg": true})},
	`spend life instead of mana for effects of skills`:          d(),
	`skills cost \+([0-9]+) rage`:                               fn(func(c caps) any { return []any{mod("RageCostBase", "BASE", c.n(1))} }),
	`warcries cost \+([0-9]+)% of life`: fn(func(c caps) any {
		return []any{mod("LifeCostBase", "BASE", 1, nil, 0, KeywordFlag.Warcry, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1), "floor": true})}
	}),
	`vaal skills used during effect have ([0-9]+)% reduced soul gain prevention duration`: fn(func(c caps) any {
		return []any{mod("SoulGainPreventionDuration", "INC", -c.n(1), Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "SkillType", "skillType": SkillType.Vaal})}
	}),
	`vaal volcanic fissure and vaal molten strike have ([0-9]+)% reduced soul gain prevention duration`: fn(func(c caps) any {
		return []any{mod("SoulGainPreventionDuration", "INC", -c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Volcanic Fissure", "Molten Strike"}, "includeTransfigured": true}, Tag{"type": "SkillType", "skillType": SkillType.Vaal})}
	}),
	`vaal skills can store \+([0-9]+) uses?`: fn(func(c caps) any {
		return []any{mod("AdditionalUses", "BASE", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Vaal})}
	}),
	`vaal attack skills cost rage instead of requiring souls to use`:           []any{flag("CostRageInsteadOfSouls", nil, ModFlag.Attack, Tag{"type": "SkillType", "skillType": SkillType.Vaal})},
	`vaal attack skills you use yourself cost rage instead of requiring souls`: []any{flag("CostRageInsteadOfSouls", nil, ModFlag.Attack, Tag{"type": "SkillType", "skillType": SkillType.Vaal})},
	`you cannot gain rage during soul gain prevention`:                         []any{mod("RageRegen", "MORE", -100, Tag{"type": "Condition", "var": "SoulGainPrevention"})},
	`hits that deal elemental damage remove exposure to those elements and inflict exposure to other elements exposure inflicted this way applies (-[0-9]+)% to resistances`: fn(func(c caps) any {
		return []any{flag("ElementalEquilibrium"), mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", c.n(1), Tag{"type": "Condition", "varList": []any{"HitByColdDamage", "HitByLightningDamage"}})}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", c.n(1), Tag{"type": "Condition", "varList": []any{"HitByFireDamage", "HitByLightningDamage"}})}), mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", c.n(1), Tag{"type": "Condition", "varList": []any{"HitByFireDamage", "HitByColdDamage"}})})}
	}),
	`projectile attack hits deal up to 30% more damage to targets at the start of their movement, dealing less damage to targets as the projectile travels farther`: []any{flag("PointBlank")},
	`leech energy shield instead of life`: []any{flag("GhostReaver")},
	`minions explode when reduced to low life, dealing 33% of their maximum life as fire damage to surrounding enemies`:                                []any{mod("ExtraMinionSkill", "LIST", Tag{"skillId": "MinionInstability"})},
	`minions explode when reduced to low life, dealing 33% of their life as fire damage to surrounding enemies`:                                        []any{mod("ExtraMinionSkill", "LIST", Tag{"skillId": "MinionInstability"})},
	`all bonuses from an equipped shield apply to your minions instead of you`:                                                                         d(),
	`spend energy shield before mana for skill m?a?n?a? ?costs`:                                                                                        d(),
	`you have perfect agony if you've dealt a critical strike recently`:                                                                                []any{mod("Keystone", "LIST", "Perfect Agony", Tag{"type": "Condition", "var": "CritRecently"})},
	`energy shield protects mana instead of life`:                                                                                                      []any{flag("EnergyShieldProtectsMana")},
	`modifiers to critical strike multiplier also apply to damage over time multiplier for ailments from critical strikes at ([0-9]+)% of their value`: fn(func(c caps) any { return []any{mod("CritMultiplierAppliesToDegen", "BASE", c.n(1))} }),
	`damage over time multiplier for ailments is equal to critical strike multiplier`:                                                                  []any{flag("DotMultiplierIsCritMultiplier")},
	`\+([0-9]+)% to cold damage over time multiplier for each ([0-9]+)% overcapped cold resistance`: fn(func(c caps) any {
		return []any{mod("ColdDotMultiplier", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "ColdResistOverCap", "div": c.s(2)})}
	}),
	`your bleeding does not deal extra damage while the enemy is moving`:                          []any{flag("Condition:NoExtraBleedDamageToMovingEnemy")},
	`your bleeding does not deal extra damage while the enemy is moving and cannot be aggravated`: []any{flag("Condition:NoExtraBleedDamageToMovingEnemy"), flag("Condition:CannotAggravate")},
	`you and enemies in your presence count as moving while affected by elemental ailments`:       []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Moving", Tag{"type": "Condition", "varList": []any{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}})}), flag("Condition:Moving", Tag{"type": "Condition", "varList": []any{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}})},
	`you can inflict bleeding on an enemy up to ([0-9]+) times?`: fn(func(c caps) any {
		return []any{mod("BleedStacksMax", "OVERRIDE", c.n(1)), flag("Condition:HaveCrimsonDance")}
	}),
	`your minions spread caustic ground on death, dealing ([0-9]+)% of their maximum life as chaos damage per second`: fn(func(c caps) any {
		return []any{mod("ExtraMinionSkill", "LIST", Tag{"skillId": "SiegebreakerCausticGround"}), mod("MinionModifier", "LIST", Tag{"mod": mod("Multiplier:SiegebreakerCausticGroundPercent", "BASE", c.n(1))})}
	}),
	`your minions spread burning ground on death, dealing ([0-9]+)% of their maximum life as fire damage per second`: fn(func(c caps) any {
		return []any{mod("ExtraMinionSkill", "LIST", Tag{"skillId": "ReplicaSiegebreakerBurningGround"}), mod("MinionModifier", "LIST", Tag{"mod": mod("Multiplier:SiegebreakerBurningGroundPercent", "BASE", c.n(1))})}
	}),
	`you can have an additional brand attached to an enemy`: []any{mod("BrandsAttachedLimit", "BASE", 1)},
	`gain ([0-9]+) grasping vines each second while stationary`: fn(func(c caps) any {
		return []any{mod("Multiplier:GraspingVinesCount", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "StationarySeconds", "limit": 10, "limitTotal": true}, Tag{"type": "Condition", "var": "Stationary"})}
	}),
	`all damage inflicts poison against enemies affected by at least ([0-9]+) grasping vines`: fn(func(c caps) any {
		return []any{mod("PoisonChance", "BASE", 100, Tag{"type": "MultiplierThreshold", "var": "GraspingVinesAffectingEnemy", "threshold": c.n(1)}), flag("FireCanPoison", Tag{"type": "MultiplierThreshold", "var": "GraspingVinesAffectingEnemy", "threshold": c.n(1)}), flag("ColdCanPoison", Tag{"type": "MultiplierThreshold", "var": "GraspingVinesAffectingEnemy", "threshold": c.n(1)}), flag("LightningCanPoison", Tag{"type": "MultiplierThreshold", "var": "GraspingVinesAffectingEnemy", "threshold": c.n(1)})}
	}),
	`attack projectiles always inflict bleeding and maim, and knock back enemies`: []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Attack|ModFlag.Projectile), mod("EnemyKnockbackChance", "BASE", 100, nil, ModFlag.Attack|ModFlag.Projectile)},
	`projectiles cannot pierce, fork or chain`:                                    []any{flag("CannotPierce", nil, ModFlag.Projectile), flag("CannotChain", nil, ModFlag.Projectile), flag("CannotFork", nil, ModFlag.Projectile)},
	`projectiles cannot continue after colliding with targets`:                    []any{flag("CannotPierce", nil, ModFlag.Projectile), flag("CannotChain", nil, ModFlag.Projectile), flag("CannotFork", nil, ModFlag.Projectile), flag("CannotSplit", nil, ModFlag.Projectile)},
	`critical strikes inflict scorch, brittle and sapped`:                         []any{flag("CritAlwaysAltAilments")},
	`chance to block attack damage is doubled`:                                    []any{mod("BlockChance", "MORE", 100)},
	`chance to block spell damage is doubled`:                                     []any{mod("SpellBlockChance", "MORE", 100)},
	`you take ([0-9]+)% of damage from blocked hits`:                              fn(func(c caps) any { return []any{mod("BlockEffect", "BASE", c.n(1))} }),
	`ignore attribute requirements`:                                               []any{flag("IgnoreAttributeRequirements")},
	`ignore attribute requirements of gems socketed in blue sockets`:              []any{mod("GemProperty", "LIST", Tag{"keyword": "all", "key": "req", "value": 0}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "socketColor": "B"})},
	`ignore attribute requirements of socketed gems`:                              []any{mod("GemProperty", "LIST", Tag{"keyword": "all", "key": "req", "value": 0}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`gain no inherent bonuses from attributes`:                                    []any{flag("NoAttributeBonuses")},
	`gain no inherent bonuses from strength`:                                      []any{flag("NoStrengthAttributeBonuses")},
	`gain no inherent bonuses from dexterity`:                                     []any{flag("NoDexterityAttributeBonuses")},
	`gain no inherent bonuses from intelligence`:                                  []any{flag("NoIntelligenceAttributeBonuses")},

	`physical damage taken bypasses energy shield`:                                               []any{mod("PhysicalEnergyShieldBypass", "BASE", 100)},
	`auras from your skills do not affect allies`:                                                []any{flag("SelfAuraSkillsCannotAffectAllies")},
	`auras from your skills have ([0-9]+)% more effect on you`:                                   fn(func(c caps) any { return []any{mod("SkillAuraEffectOnSelf", "MORE", c.n(1))} }),
	`auras from your skills have ([0-9]+)% increased effect on you`:                              fn(func(c caps) any { return []any{mod("SkillAuraEffectOnSelf", "INC", c.n(1))} }),
	`increases and reductions to mana regeneration rate instead apply to rage regeneration rate`: []any{flag("ManaRegenToRageRegen")},
	`increases and reductions to maximum energy shield instead apply to ward`:                    []any{flag("EnergyShieldToWard")},
	`([0-9]+)% of damage taken bypasses ward`:                                                    fn(func(c caps) any { return []any{mod("WardBypass", "BASE", c.n(1))} }),
	`ward has a ([0-9]+)% chance to not break`:                                                   fn(func(c caps) any { return []any{mod("WardAvoidBreakChance", "BASE", c.n(1))} }),
	`damage taken bypasses unbroken ward if the hit deals less damage than ([0-9]+)% of ward`:    fn(func(c caps) any { return []any{mod("WardBypassBelowPercent", "BASE", c.n(1))} }),
	`maximum energy shield is ([0-9]+)`:                                                          fn(func(c caps) any { return []any{mod("EnergyShield", "OVERRIDE", c.n(1))} }),
	`while not on full life, sacrifice ([0-9.]+)% of mana per second to recover that much life`: fn(func(c caps) any {
		return []any{mod("ManaDegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "FullLife", "neg": true}), mod("LifeRecovery", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "Condition", "var": "FullLife", "neg": true})}
	}),
	`([0-9]+)% increased maximum energy shield`: fn(func(c caps) any { return []any{mod("EnergyShield", "INC", c.n(1), Tag{"type": "Global"})} }),
	`you are blind`: []any{flag("Condition:Blinded", Tag{"type": "Condition", "var": "CannotBeBlinded", "neg": true})},
	`armour applies to fire, cold and lightning damage taken from hits instead of physical damage`:                                          []any{mod("ArmourAppliesToFireDamageTaken", "BASE", 100), mod("ArmourAppliesToColdDamageTaken", "BASE", 100), mod("ArmourAppliesToLightningDamageTaken", "BASE", 100), flag("ArmourDoesNotApplyToPhysicalDamageTaken")},
	`([0-9]+)% of armour also applies to chaos damage taken from hits`:                                                                      fn(func(c caps) any { return []any{mod("ArmourAppliesToChaosDamageTaken", "BASE", c.n(1))} }),
	`armour also applies to chaos damage taken from hits`:                                                                                   fn(func(c caps) any { return []any{mod("ArmourAppliesToChaosDamageTaken", "BASE", 100)} }),
	`maximum damage reduction for any damage type is ([0-9]+)%`:                                                                             fn(func(c caps) any { return []any{mod("DamageReductionMax", "OVERRIDE", c.n(1))} }),
	`gain additional elemental damage reduction equal to half your chaos resistance`:                                                        []any{mod("ElementalDamageReduction", "BASE", 1, Tag{"type": "PerStat", "stat": "ChaosResist", "div": 2})},
	`([0-9]+)% of maximum mana is converted to twice that much armour`:                                                                      fn(func(c caps) any { return []any{mod("ManaConvertToArmour", "BASE", c.n(1))} }),
	`life recovery from flasks also applies to energy shield`:                                                                               []any{flag("LifeFlaskAppliesToEnergyShield")},
	`increase to cast speed from arcane surge also applies to movement speed`:                                                               []any{flag("ArcaneSurgeCastSpeedToMovementSpeed")},
	`arcane surge also grants ([0-9]+)% increased life regeneration rate to you`:                                                            fn(func(c caps) any { return []any{mod("ArcaneSurgeAlsoLifeRegen", "BASE", c.n(1))} }),
	`increases and reductions to effect of flasks applied to you also applies to effect of arcane surge on you at ([0-9]+)% of their value`: fn(func(c caps) any { return []any{mod("FlaskEffectToArcaneSurgeEffect", "BASE", c.n(1))} }),
	`non-instant mana recovery from flasks is also recovered as life`:                                                                       []any{flag("ManaFlaskAppliesToLife")},
	`life leech effects recover energy shield instead while on full life`:                                                                   []any{flag("ImmortalAmbition", Tag{"type": "Condition", "var": "FullLife"}, Tag{"type": "Condition", "var": "LeechingLife"})},
	`shepherd of souls`: []any{flag("ShepherdOfSouls")},
	`you have shepherd of souls if at least ([0-9]+) corrupted items are equipped`: fn(func(c caps) any {
		return []any{flag("ShepherdOfSouls", Tag{"type": "MultiplierThreshold", "var": "CorruptedItem", "threshold": c.n(1)})}
	}),
	`you have everlasting sacrifice if at least ([0-9]+) corrupted items are equipped`: fn(func(c caps) any {
		return []any{flag("Condition:EverlastingSacrifice", Tag{"type": "MultiplierThreshold", "var": "CorruptedItem", "threshold": c.n(1)})}
	}),
	`adds ([0-9]+) to ([0-9]+) attack physical damage to melee skills per ([0-9]+) dexterity while you are unencumbered`: fn(func(c caps) any {
		return []any{mod("PhysicalMin", "BASE", c.n(1), nil, ModFlag.Melee, KeywordFlag.Attack, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(3)}, Tag{"type": "Condition", "var": "Unencumbered"}), mod("PhysicalMax", "BASE", c.n(2), nil, ModFlag.Melee, KeywordFlag.Attack, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(3)}, Tag{"type": "Condition", "var": "Unencumbered"})}
	}),
	`adds ([0-9]+) to ([0-9]+) fire damage to attacks for every ([0-9]+)% your light radius is above base value`: fn(func(c caps) any {
		return []any{mod("FireMin", "BASE", c.n(1), nil, ModFlag.Attack, Tag{"type": "PerStat", "stat": "LightRadiusInc", "div": c.n(3)}), mod("FireMax", "BASE", c.n(2), nil, ModFlag.Attack, Tag{"type": "PerStat", "stat": "LightRadiusInc", "div": c.n(3)})}
	}),
	`([0-9]+)% more attack damage if accuracy rating is higher than maximum life`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), "Damage", ModFlag.Attack, Tag{"type": "Condition", "var": "MainHandAccRatingHigherThanMaxLife"}, Tag{"type": "Condition", "var": "MainHandAttack"}), mod("Damage", "MORE", c.n(1), "Damage", ModFlag.Attack, Tag{"type": "Condition", "var": "OffHandAccRatingHigherThanMaxLife"}, Tag{"type": "Condition", "var": "OffHandAttack"})}
	}),
	`your hexes have infinite duration`: []any{mod("Duration", "BASE", m_huge, Tag{"type": "SkillType", "skillType": SkillType.AppliesCurse})},
	// Legacy support
	// Masteries
	`hits have ([0-9]+)% chance to treat enemy monster elemental resistance values as inverted`: fn(func(c caps) any { return []any{mod("HitsInvertEleResChance", "CHANCE", c.n(1)/100, nil)} }),
	`off hand accuracy is equal to main hand accuracy while wielding a sword`:                   []any{flag("Condition:OffHandAccuracyIsMainHandAccuracy", Tag{"type": "Condition", "var": "UsingSword"})},
	`([0-9]+)% increased accuracy rating at close range`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "INC", c.n(1), Tag{"type": "Condition", "var": "AtCloseRange"})}
	}),
	`([0-9]+)% more accuracy rating at close range`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "MORE", c.n(1), Tag{"type": "Condition", "var": "AtCloseRange"})}
	}),
	`([0-9]+)% increased accuracy rating against unique enemies`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})}
	}),
	`([0-9]+)% more accuracy rating against unique enemies`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "MORE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})}
	}),
	`defend with ([0-9]+)% of armour while not on low energy shield`: fn(func(c caps) any {
		return []any{mod("ArmourDefense", "MAX", c.n(1)-100, "Armour and Energy Shield Mastery", Tag{"type": "Condition", "var": "LowEnergyShield", "neg": true})}
	}),
	`([0-9]+)% increased armour and energy shield from equipped body armour if equipped helmet, gloves and boots all have armour and energy shield`: fn(func(c caps) any {
		return []any{mod("Body ArmourESAndArmour", "INC", c.n(1), Tag{"type": "StatThreshold", "stat": "ArmourOnGloves", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "EnergyShieldOnGloves", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "ArmourOnHelmet", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "EnergyShieldOnHelmet", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "ArmourOnBoots", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "EnergyShieldOnBoots", "threshold": 1})}
	}),
	`brands have ([0-9]+)% increased area of effect if ([0-9]+)% of attached duration expired`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Condition", "var": "BrandLastHalf"}, Tag{"type": "SkillType", "skillType": SkillType.Brand})}
	}),
	`corrupted blood cannot be inflicted on you`: []any{flag("CorruptedBloodImmune")},
	`you cannot be hindered`:                     []any{flag("HinderImmune")},
	`you cannot be maimed`:                       []any{flag("MaimImmune")},
	`you cannot be impaled`:                      []any{flag("ImpaleImmune")},
	// Exerted Attacks
	`exerted attacks deal ([0-9]+)% increased damage`:             fn(func(c caps) any { return []any{mod("ExertIncrease", "INC", c.n(1), nil, ModFlag.Attack, 0)} }),
	`exerted attacks have ([0-9]+)% chance to deal double damage`: fn(func(c caps) any { return []any{mod("ExertDoubleDamageChance", "BASE", c.n(1), nil, ModFlag.Attack, 0)} }),
	// Ascendant
	`grants ([0-9]+) passive skill points?`:                     fn(func(c caps) any { return []any{mod("ExtraPoints", "BASE", c.n(1))} }),
	`can allocate passives from the [a-zA-Z]+'s starting point`: d(),
	`projectiles gain damage as they travel farther, dealing up to ([0-9]+)% increased damage with hits to targets`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Hit|ModFlag.Projectile, Tag{"type": "DistanceRamp", "ramp": []any{[]any{35, 0}, []any{70, 1}}})}
	}),
	`([0-9]+)% chance to gain elusive on kill`:                        []any{flag("Condition:CanBeElusive")},
	`immun[ei]t?y? to elemental ailments while on consecrated ground`: []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "OnConsecratedGround"})},
	// Assassin
	`poison you inflict with critical strikes deals ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Poison, Tag{"type": "Condition", "var": "CriticalStrike"})}
	}),
	`([0-9]+)% chance to gain elusive on critical strike`: []any{flag("Condition:CanBeElusive")},
	`([0-9]+)% more damage while there is at most one rare or unique enemy nearby`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, Tag{"type": "Condition", "var": "AtMostOneNearbyRareOrUniqueEnemy"})}
	}),
	`([0-9]+)% reduced damage taken while there are at least two rare or unique enemies nearby`: fn(func(c caps) any {
		return []any{mod("DamageTaken", "INC", -c.n(1), nil, 0, Tag{"type": "MultiplierThreshold", "var": "NearbyRareOrUniqueEnemies", "threshold": 2})}
	}),
	`([0-9]+)% less damage taken while there are at least two rare or unique enemies nearby`: fn(func(c caps) any {
		return []any{mod("DamageTaken", "MORE", -c.n(1), nil, 0, Tag{"type": "MultiplierThreshold", "var": "NearbyRareOrUniqueEnemies", "threshold": 2})}
	}),
	`you take no extra damage from critical strikes while elusive`: []any{mod("ReduceCritExtraDamage", "BASE", 100, Tag{"type": "Condition", "var": "Elusive"})},
	`mark skills cost no mana`:                                     []any{mod("ManaCost", "MORE", -100, nil, 0, 0, Tag{"type": "SkillType", "skillType": SkillType.Mark})},
	// Berserker
	`gain [0-9]+ rage when you kill an enemy`:                                       []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage when you use a warcry`:                                        []any{flag("Condition:CanGainRage")},
	`you and nearby party members gain [0-9]+ rage when you warcry`:                 []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with attacks, no more than once every [0-9.]+ seconds`: []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on attack hit`:                                                []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with axes or swords`:                                   []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on melee hit`:                                                 []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on melee weapon hit`:                                          []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with axes`:                                             []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage when hit by an enemy`:                                         []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with retaliation skills`:                               []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage when you use a life flask`:                                    []any{flag("Condition:CanGainRage")},
	`while a unique enemy is in your presence, gain [0-9]+ rage on hit with attacks, no more than once every [0-9.]+ seconds`:        []any{flag("Condition:CanGainRage", Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})},
	`while a pinnacle atlas boss is in your presence, gain [0-9]+ rage on hit with attacks, no more than once every [0-9.]+ seconds`: []any{flag("Condition:CanGainRage", Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"})},
	`maximum rage is halved`:                        []any{mod("MaximumRage", "MORE", -50)},
	`inherent effects from having rage are tripled`: []any{mod("RageEffect", "MORE", 200)},
	`inherent effects from having rage are doubled`: []any{mod("RageEffect", "MORE", 100)},
	`cannot be stunned while you have at least ([0-9]+) rage`: fn(func(c caps) any {
		return []any{flag("StunImmune", Tag{"type": "MultiplierThreshold", "var": "Rage", "threshold": c.n(1)})}
	}),
	`([0-9]+)% less damage taken per ([0-9]+) rage, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("DamageTaken", "MORE", -c.n(1), Tag{"type": "Multiplier", "var": "Rage", "div": c.s(2), "limit": -c.n(3), "limitNegTotal": true})}
	}),
	`lose ([0-9.]+)% of life per second per rage while you are not losing rage`: fn(func(c caps) any {
		return []any{mod("LifeDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "RageEffect"})}
	}),
	`if you've warcried recently, you and nearby allies have ([0-9]+)% increased attack speed`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("Speed", "INC", c.n(1), nil, ModFlag.Attack)}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})}
	}),
	`gain ([0-9]+)% increased armour per ([0-9]+) power for 8 seconds when you warcry, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Armour", "INC", c.n(1), Tag{"type": "Multiplier", "var": "WarcryPower", "div": c.n(2), "globalLimit": c.n(3), "globalLimitKey": "WarningCall"}, Tag{"type": "Condition", "var": "UsedWarcryInPast8Seconds"})}
	}),
	`warcries grant ([0-9]+) rage per ([0-9]+) power if you have less than ([0-9]+) rage`:    []any{flag("Condition:CanGainRage")},
	`warcries grant ([0-9]+) rage per ([0-9]+) enemy power, up to ([0-9]+)`:                  []any{flag("Condition:CanGainRage")},
	`exerted attacks deal ([0-9]+)% more attack damage if a warcry sacrificed rage recently`: fn(func(c caps) any { return []any{mod("ExertAttackIncrease", "MORE", c.n(1), nil, ModFlag.Attack, 0)} }),
	`deal ([0-9]+)% less damage`:           fn(func(c caps) any { return []any{mod("Damage", "MORE", -c.n(1))} }),
	`warcries exert twice as many attacks`: []any{mod("ExtraExertedAttacks", "MORE", 100)},
	// Champion
	`cannot be stunned while you have fortify`:                 []any{flag("StunImmune", Tag{"type": "Condition", "var": "Fortified"})},
	`cannot be stunned while fortified`:                        []any{flag("StunImmune", Tag{"type": "Condition", "var": "Fortified"})},
	`you cannot be stunned while at maximum endurance charges`: []any{flag("StunImmune", Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMax"})},
	`fortify`:                             []any{flag("Condition:Fortified")},
	`you have your maximum fortification`: []any{flag("Condition:Fortified"), flag("Condition:HaveMaxFortification")},
	`you have ([0-9]+) fortification`:     fn(func(c caps) any { return []any{mod("MinimumFortification", "BASE", c.n(1))} }),
	`nearby allies count as having fortification equal to yours`: []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("YourFortifyEqualToParent", "FLAG", true, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})})},
	`enemies taunted by you cannot evade attacks`:                []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("CannotEvade", Tag{"type": "Condition", "var": "Taunted"})})},
	`if you've impaled an enemy recently, you and nearby allies have \+([0-9]+) to armour`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("Armour", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "ImpaledRecently"})}
	}),
	`your hits permanently intimidate enemies that are on full life`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated", Tag{"type": "Condition", "var": "ChampionIntimidate"})})},
	`you and allies affected by your placed banners regenerate ([0-9.]+)% of life per second for each stage`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByPlacedBanner"}, Tag{"type": "Multiplier", "var": "BannerValour"})})}
	}),
	`you and allies near your banner regenerate ([0-9.]+)% of life per second for each valour consumed for that banner`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByPlacedBanner"}, Tag{"type": "Multiplier", "var": "BannerValour"})})}
	}),
	// Chieftain
	`enemies near your totems take ([0-9]+)% increased physical and fire damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("PhysicalDamageTaken", "INC", c.n(1))}), mod("EnemyModifier", "LIST", Tag{"mod": mod("FireDamageTaken", "INC", c.n(1))})}
	}),
	`every ([0-9]+) seconds, gain ([0-9]+)% of physical damage as extra fire damage for ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsFire", "BASE", c.n(2), Tag{"type": "Condition", "var": "NgamahuFlamesAdvance"})}
	}),
	`([0-9]+)% more damage for each endurance charge lost recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), Tag{"type": "Multiplier", "var": "EnduranceChargesLostRecently", "limit": c.n(2), "limitTotal": true})}
	}),
	`nearby enemy monsters have no fire resistance against damage over time while you are stationary`: []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireResist", "OVERRIDE", 0, nil, ModFlag.Dot, Tag{"type": "ActorCondition", "actor": "player", "var": "Stationary"})})},
	`([0-9]+)% more damage if you've lost an endurance charge in the past 8 seconds`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), Tag{"type": "Condition", "var": "LostEnduranceChargeInPast8Sec"})}
	}),
	`trigger level ([0-9]+) (.+) when you attack with a non-vaal slam or strike skill near an enemy`: fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	// Deadeye
	`projectiles pierce all nearby targets`: []any{flag("PierceAllTargets")},
	`gain \+([0-9]+) life when you hit a bleeding enemy`: fn(func(c caps) any {
		return []any{mod("LifeOnHit", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"})}
	}),
	`([0-9]+)% increased blink arrow and mirror arrow cooldown recovery speed`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "INC", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Blink Arrow", "Mirror Arrow"}, "includeTransfigured": true})}
	}),
	`critical strikes which inflict bleeding also inflict rupture`:         []any{flag("Condition:CanInflictRupture", Tag{"type": "Condition", "neg": true, "var": "NeverCrit"})},
	`gain [0-9]+ gale force when you use a skill`:                          []any{flag("Condition:CanGainGaleForce")},
	`if you've used a skill recently, you and nearby allies have tailwind`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Condition:Tailwind")}, Tag{"type": "Condition", "var": "UsedSkillRecently"})},
	`you and nearby allies have tailwind`:                                  []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Condition:Tailwind")})},
	`projectiles deal ([0-9]+)% more damage for each remaining chain`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, ModFlag.Projectile, Tag{"type": "PerStat", "stat": "ChainRemaining"})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits and ailments for each remaining chain`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "PerStat", "stat": "ChainRemaining"}, Tag{"type": "SkillType", "skillType": SkillType.Projectile})}
	}),
	`projectiles deal ([0-9]+)% increased damage for each remaining chain`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Projectile, Tag{"type": "PerStat", "stat": "ChainRemaining"})}
	}),
	`far shot`: []any{flag("FarShot")},
	`projectiles gain damage as they travel farther, dealing up to ([0-9]+)% more damage with hits and ailments`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "DistanceRamp", "ramp": []any{[]any{35, 0}, []any{70, 1}}})}
	}),
	`([0-9]+)% increased mirage archer duration`:                 fn(func(c caps) any { return []any{mod("MirageArcherDuration", "INC", c.n(1))} }),
	`([\-+][0-9]+) to maximum number of summoned mirage archers`: fn(func(c caps) any { return []any{mod("MirageArcherMaxCount", "BASE", c.n(1))} }),
	`([\-+][0-9]+) to maximum number of sacred wisps`:            fn(func(c caps) any { return []any{mod("SacredWispsMaxCount", "BASE", c.n(1))} }),
	// Elementalist
	`gain ([0-9]+)% increased area of effect for [0-9]+ seconds`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Condition", "var": "PendulumOfDestructionAreaOfEffect"})}
	}),
	`gain ([0-9]+)% increased elemental damage for [0-9]+ seconds`: fn(func(c caps) any {
		return []any{mod("ElementalDamage", "INC", c.n(1), Tag{"type": "Condition", "var": "PendulumOfDestructionElementalDamage"})}
	}),
	`for each element you've been hit by damage of recently, ([0-9]+)% increased damage of that element`: fn(func(c caps) any {
		return []any{mod("FireDamage", "INC", c.n(1), Tag{"type": "Condition", "var": "HitByFireDamageRecently"}), mod("ColdDamage", "INC", c.n(1), Tag{"type": "Condition", "var": "HitByColdDamageRecently"}), mod("LightningDamage", "INC", c.n(1), Tag{"type": "Condition", "var": "HitByLightningDamageRecently"})}
	}),
	`for each element you've been hit by damage of recently, ([0-9]+)% reduced damage taken of that element`: fn(func(c caps) any {
		return []any{mod("FireDamageTaken", "INC", -c.n(1), Tag{"type": "Condition", "var": "HitByFireDamageRecently"}), mod("ColdDamageTaken", "INC", -c.n(1), Tag{"type": "Condition", "var": "HitByColdDamageRecently"}), mod("LightningDamageTaken", "INC", -c.n(1), Tag{"type": "Condition", "var": "HitByLightningDamageRecently"})}
	}),
	`gain convergence when you hit a unique enemy, no more than once every [0-9]+ seconds`: []any{flag("Condition:CanGainConvergence")},
	`([0-9]+)% increased area of effect while you don't have convergence`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Condition", "neg": true, "var": "Convergence"})}
	}),
	`exposure you inflict applies an extra (-?[0-9]+)% to the affected resistance`:        fn(func(c caps) any { return []any{mod("ExtraExposure", "BASE", c.n(1))} }),
	`cannot take reflected elemental damage`:                                              []any{mod("ElementalReflectedDamageTaken", "MORE", -100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`every [0-9]+ seconds:`:                                                               d(),
	`gain chilling conflux for [0-9] seconds`:                                             []any{flag("PhysicalCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), flag("LightningCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), flag("FireCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), flag("ChaosCanChill", Tag{"type": "Condition", "var": "ChillingConflux"})},
	`gain shocking conflux for [0-9] seconds`:                                             []any{mod("EnemyShockChance", "BASE", 100, Tag{"type": "Condition", "var": "ShockingConflux"}), flag("PhysicalCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), flag("ColdCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), flag("FireCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), flag("ChaosCanShock", Tag{"type": "Condition", "var": "ShockingConflux"})},
	`gain igniting conflux for [0-9] seconds`:                                             []any{mod("EnemyIgniteChance", "BASE", 100, Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("PhysicalCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("LightningCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("ColdCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("ChaosCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"})},
	`([0-9]+)% chance to gain elemental conflux for [0-9] seconds when you kill an enemy`: []any{flag("PhysicalCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), flag("LightningCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), flag("FireCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), flag("ChaosCanChill", Tag{"type": "Condition", "var": "ChillingConflux"}), mod("EnemyShockChance", "BASE", 100, Tag{"type": "Condition", "var": "ShockingConflux"}), flag("PhysicalCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), flag("ColdCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), flag("FireCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), flag("ChaosCanShock", Tag{"type": "Condition", "var": "ShockingConflux"}), mod("EnemyIgniteChance", "BASE", 100, Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("PhysicalCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("LightningCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("ColdCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"}), flag("ChaosCanIgnite", Tag{"type": "Condition", "var": "IgnitingConflux"})},
	`you have elemental conflux if the stars are aligned`:                                 []any{flag("PhysicalCanChill", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("LightningCanChill", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("FireCanChill", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("ChaosCanChill", Tag{"type": "Condition", "var": "StarsAreAligned"}), mod("EnemyShockChance", "BASE", 100, Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("PhysicalCanShock", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("ColdCanShock", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("FireCanShock", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("ChaosCanShock", Tag{"type": "Condition", "var": "StarsAreAligned"}), mod("EnemyIgniteChance", "BASE", 100, Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("PhysicalCanIgnite", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("LightningCanIgnite", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("ColdCanIgnite", Tag{"type": "Condition", "var": "StarsAreAligned"}), flag("ChaosCanIgnite", Tag{"type": "Condition", "var": "StarsAreAligned"})},
	`gain chilling, shocking and igniting conflux for [0-9] seconds`:                      d(),
	`you have igniting, chilling and shocking conflux while affected by glorious madness`: []any{flag("PhysicalCanChill", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("LightningCanChill", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("FireCanChill", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("ChaosCanChill", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), mod("EnemyIgniteChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("PhysicalCanIgnite", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("LightningCanIgnite", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("ColdCanIgnite", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("ChaosCanIgnite", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), mod("EnemyShockChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("PhysicalCanShock", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("ColdCanShock", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("FireCanShock", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("ChaosCanShock", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"})},
	`all damage from critical strikes can apply lightning ailments during effect`:         []any{flag("PhysicalCanShock", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ColdCanShock", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("FireCanShock", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ChaosCanShock", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("PhysicalCanSap", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ColdCanSap", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("FireCanSap", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ChaosCanSap", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"})},
	`all damage from critical strikes can apply cold ailments during effect`:              []any{flag("PhysicalCanChill", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("LightningCanChill", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("FireCanChill", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ChaosCanChill", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("PhysicalCanFreeze", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("LightningCanFreeze", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("FireCanFreeze", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ChaosCanFreeze", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("PhysicalCanBrittle", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("LightningCanBrittle", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("FireCanBrittle", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"}), flag("ChaosCanBrittle", Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"})},
	`immun[ei]t?y? to elemental ailments while affected by glorious madness`:              []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"})},
	`immun[ei]t?y? to elemental ailments while focus?sed`:                                 []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "Focused"})},
	`summoned golems are immune to elemental damage`:                                      []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Elemancer")}, Tag{"type": "SkillType", "skillType": SkillType.Golem}), mod("MinionModifier", "LIST", Tag{"mod": mod("ElementalDamageTaken", "MORE", -100)}, Tag{"type": "SkillType", "skillType": SkillType.Golem})},
	`([0-9]+)% increased golem damage per summoned golem`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1))}, Tag{"type": "SkillType", "skillType": SkillType.Golem}, Tag{"type": "PerStat", "stat": "ActiveGolemLimit"})}
	}),
	`shocks from your hits always increase damage taken by at least ([0-9]+)%`: fn(func(c caps) any { return []any{mod("ShockMinimum", "BASE", c.n(1))} }),
	`chills from your hits always reduce action speed by at least ([0-9]+)%`:   fn(func(c caps) any { return []any{mod("ChillBase", "BASE", c.n(1))} }),
	`([0-9]+)% more damage with ignites you inflict with hits for which the highest damage type is fire`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Ignite, Tag{"type": "Condition", "var": "FireIsHighestDamageType"}), flag("ChecksHighestDamage")}
	}),
	`([0-9]+)% more effect of cold ailments you inflict with hits for which the highest damage type is cold`: fn(func(c caps) any {
		return []any{mod("EnemyChillEffect", "MORE", c.n(1), Tag{"type": "Condition", "var": "ColdIsHighestDamageType"}), mod("EnemyBrittleEffect", "MORE", c.n(1), Tag{"type": "Condition", "var": "ColdIsHighestDamageType"}), flag("ChecksHighestDamage")}
	}),
	`([0-9]+)% more effect of lightning ailments you inflict with hits if the highest damage type is lightning`: fn(func(c caps) any {
		return []any{mod("EnemyShockEffect", "MORE", c.n(1), Tag{"type": "Condition", "var": "LightningIsHighestDamageType"}), mod("EnemySapEffect", "MORE", c.n(1), Tag{"type": "Condition", "var": "LightningIsHighestDamageType"}), flag("ChecksHighestDamage")}
	}),
	`your chills can reduce action speed by up to a maximum of ([0-9]+)%`: fn(func(c caps) any { return []any{mod("ChillMax", "OVERRIDE", c.n(1))} }),
	`your hits always ignite`:         []any{mod("EnemyIgniteChance", "BASE", 100)},
	`hits always ignite`:              []any{mod("EnemyIgniteChance", "BASE", 100)},
	`your hits always shock`:          []any{mod("EnemyShockChance", "BASE", 100)},
	`hits always shock`:               []any{mod("EnemyShockChance", "BASE", 100)},
	`always freeze, shock and ignite`: []any{mod("EnemyFreezeChance", "BASE", 100), mod("EnemyShockChance", "BASE", 100), mod("EnemyIgniteChance", "BASE", 100)},
	`all damage with hits can ignite`: []any{flag("PhysicalCanIgnite"), flag("ColdCanIgnite"), flag("LightningCanIgnite"), flag("ChaosCanIgnite")},
	`all damage can ignite`:           []any{flag("PhysicalCanIgnite"), flag("ColdCanIgnite"), flag("LightningCanIgnite"), flag("ChaosCanIgnite")},
	`all damage with hits can chill`:  []any{flag("PhysicalCanChill"), flag("FireCanChill"), flag("LightningCanChill"), flag("ChaosCanChill")},
	`all damage with hits can shock`:  []any{flag("PhysicalCanShock"), flag("FireCanShock"), flag("ColdCanShock"), flag("ChaosCanShock")},
	`all damage can shock`:            []any{flag("PhysicalCanShock"), flag("FireCanShock"), flag("ColdCanShock"), flag("ChaosCanShock")},
	`other aegis skills are disabled`: []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Aegis}), flag("EnableSkill", Tag{"type": "SkillName", "skillId": "Primal Aegis"})},
	`primal aegis can take ([0-9]+) elemental damage per allocated notable passive skill`: fn(func(c caps) any {
		return []any{mod("ElementalAegisValue", "MAX", c.n(1), 0, 0, Tag{"type": "Multiplier", "var": "AllocatedNotable"}, Tag{"type": "GlobalEffect", "effectType": "Buff", "unscalable": true})}
	}),
	`enemies chilled by your hits lessen their damage dealt by half of chill effect`: []any{flag("ChillEffectLessDamageDealt")},
	// Gladiator
	`chance to block spell damage is equal to chance to block attack damage`:                                    []any{flag("SpellBlockChanceIsBlockChance")},
	`maximum chance to block spell damage is equal to maximum chance to block attack damage`:                    []any{flag("SpellBlockChanceMaxIsBlockChanceMax")},
	`attack damage is lucky if you[' ]h?a?ve blocked in the past ([0-9]+) seconds`:                              []any{flag("LuckyHits", nil, ModFlag.Attack, Tag{"type": "Condition", "var": "BlockedRecently"})},
	`attack damage while dual wielding is lucky if you[' ]h?a?ve blocked in the past ([0-9]+) seconds`:          []any{flag("LuckyHits", nil, ModFlag.Attack, Tag{"type": "Condition", "var": "BlockedRecently"}, Tag{"type": "Condition", "var": "DualWielding"})},
	`hits ignore enemy monster physical damage reduction if you[' ]h?a?ve blocked in the past ([0-9]+) seconds`: []any{flag("IgnoreEnemyPhysicalDamageReduction", Tag{"type": "Condition", "var": "BlockedRecently"})},
	`([0-9]+)% more attack and movement speed per challenger charge`: fn(func(c caps) any {
		return []any{mod("Speed", "MORE", c.n(1), nil, ModFlag.Attack, 0, Tag{"type": "Multiplier", "var": "ChallengerCharge"}), mod("MovementSpeed", "MORE", c.n(1), Tag{"type": "Multiplier", "var": "ChallengerCharge"})}
	}),
	`gain ([0-9]+)% chance to block from equipped shield instead of the shield's value`: fn(func(c caps) any {
		return []any{mod("ReplaceShieldBlock", "OVERRIDE", c.n(1), Tag{"type": "Condition", "var": "UsingShield"})}
	}),
	`deal ([0-9]+)% more damage with hits and ailments to rare and unique enemies for each second they've ever been in your presence, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "Multiplier", "var": "EnemyPresenceSeconds", "actor": "enemy", "limit": c.n(2)}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})}
	}),
	`deal ([0-9]+)% more damage with hits and ailments to rare and unique enemies for every ([0-9]+) seconds they've ever been in your presence, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "Multiplier", "var": "EnemyPresenceSeconds", "actor": "enemy", "limit": c.n(3), "div": c.s(2)}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})}
	}),
	`([0-9]+)% more damage with hits and ailments against enemies that are on low life while you are wielding an axe`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "LowLife"}, Tag{"type": "Condition", "var": "UsingAxe"})}
	}),
	`retaliation skills have ([0-9]+)% increased speed`: fn(func(c caps) any {
		return []any{mod("Speed", "INC", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Retaliation}), mod("WarcrySpeed", "INC", c.n(1), nil, 0, KeywordFlag.Warcry, Tag{"type": "SkillType", "skillType": SkillType.Retaliation})}
	}),
	// Guardian
	`grants armour equal to ([0-9]+)% of your reserved life to you and nearby allies`: fn(func(c caps) any {
		return []any{mod("GrantReservedLifeAsAura", "LIST", Tag{"mod": mod("Armour", "BASE", c.n(1)/100)})}
	}),
	`grants armour equal to ([0-9]+)% of your reserved mana to you and nearby allies`: fn(func(c caps) any {
		return []any{mod("GrantReservedManaAsAura", "LIST", Tag{"mod": mod("Armour", "BASE", c.n(1)/100)})}
	}),
	`grants maximum energy shield equal to ([0-9]+)% of your reserved mana to you and nearby allies`: fn(func(c caps) any {
		return []any{mod("GrantReservedManaAsAura", "LIST", Tag{"mod": mod("EnergyShield", "BASE", c.n(1)/100)})}
	}),
	`warcries cost no mana`: []any{mod("ManaCost", "MORE", -100, nil, 0, KeywordFlag.Warcry)},
	`\+([0-9]+)% chance to block attack damage for [0-9] seconds? every [0-9] seconds`: fn(func(c caps) any {
		return []any{mod("BlockChance", "BASE", c.n(1), Tag{"type": "Condition", "var": "BastionOfHopeActive"})}
	}),
	`if you've blocked in the past [0-9]+ seconds, you and nearby allies cannot be stunned`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("StunImmune")}, Tag{"type": "Condition", "var": "BlockedRecently"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`if you've attacked recently, you and nearby allies have \+([0-9]+)% chance to block attack damage`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("BlockChance", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "AttackedRecently"})}
	}),
	`if you've cast a spell recently, you and nearby allies have \+([0-9]+)% chance to block spell damage`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("SpellBlockChance", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "CastSpellRecently"})}
	}),
	`while there is at least one nearby ally, you and nearby allies deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("Damage", "MORE", c.n(1))}, Tag{"type": "MultiplierThreshold", "var": "NearbyAlly", "threshold": 1})}
	}),
	`while there are at least five nearby allies, you and nearby allies have onslaught`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Onslaught")}, Tag{"type": "MultiplierThreshold", "var": "NearbyAlly", "threshold": 5})},
	`linked targets and allies in your link beams have \+([0-9]+)% to all maximum elemental resistances`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("ElementalResistMax", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByLink", "neg": true})}, Tag{"type": "MultiplierThreshold", "var": "LinkedTargets", "threshold": 1}), mod("ExtraLinkEffect", "LIST", Tag{"mod": mod("ElementalResistMax", "BASE", c.n(1), Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})})}
	}),
	`enemies in your link beams cannot apply elemental ailments`:                              []any{flag("ElementalAilmentImmune", Tag{"type": "ActorCondition", "actor": "enemy", "var": "BetweenYouAndLinkedTarget"})},
	`([0-9]+)% of damage from hits is taken from your sentinel of radiance's life before you`: fn(func(c caps) any { return []any{mod("takenFromRadianceSentinelBeforeYou", "BASE", c.n(1))} }),
	`you can inflict \+([0-9]+) hallowing flame on enemies`:                                   fn(func(c caps) any { return []any{mod("Multiplier:HallowingFlameMax", "BASE", c.n(1))} }),
	`gain ([0-9]+)% of ([a-zA-Z]+) damage as extra ([a-zA-Z]+) damage for each of your hallowing flames that have been removed by an allied hit recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(2))+"DamageGainAs"+firstToUpper(c.s(3)), "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "HallowingFlame"}, Tag{"type": "Multiplier", "var": "HallowingFlameStacksRemovedByAlly", "limit": c.n(4) / c.n(1)})}
	}),
	`([0-9]+)% increased magnitude of hallowing flame you inflict`: fn(func(c caps) any { return []any{mod("HallowingFlameMagnitude", "INC", c.n(1))} }),
	// Hierophant
	`you and your totems regenerate ([0-9.]+)% of life per second for each summoned totem`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "TotemsSummoned"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "TotemsSummoned"}, 0, KeywordFlag.Totem)}
	}),
	`enemies take ([0-9]+)% increased damage for each of your brands attached to them`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "Multiplier", "var": "BrandsAttached"})})}
	}),
	`immun[ei]t?y? to elemental ailments while you have arcane surge`: []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "AffectedByArcaneSurge"})},
	`brands have ([0-9]+)% more activation frequency if ([0-9]+)% of attached duration expired`: fn(func(c caps) any {
		return []any{mod("BrandActivationFrequency", "MORE", c.n(1), Tag{"type": "Condition", "var": "BrandLastQuarter"})}
	}),
	`arcane surge a?l?s?o? ?grants ([0-9]+)% more spell damage to you`: fn(func(c caps) any { return []any{mod("ArcaneSurgeDamage", "MAX", c.n(1))} }),
	// Inquisitor
	`critical strikes ignore enemy monster elemental resistances`: []any{flag("IgnoreElementalResistances", Tag{"type": "Condition", "var": "CriticalStrike"})},
	`non-critical strikes penetrate ([0-9]+)% of enemy elemental resistances`: fn(func(c caps) any {
		return []any{mod("ElementalPenetration", "BASE", c.n(1), Tag{"type": "Condition", "var": "CriticalStrike", "neg": true})}
	}),
	`consecrated ground you create applies ([0-9]+)% increased damage taken to enemies`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTakenConsecratedGround", "INC", c.n(1), Tag{"type": "Condition", "var": "OnConsecratedGround"})})}
	}),
	`you have consecrated ground around you while stationary`:                                    []any{flag("Condition:OnConsecratedGround", Tag{"type": "Condition", "var": "Stationary"})},
	`consecrated ground you create grants immun[ei]t?y? to elemental ailments to you and allies`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "OnConsecratedGround"})})},
	`gain fanaticism for ([0-9]+) seconds on reaching maximum fanatic charges`:                   []any{flag("Condition:CanGainFanaticism")},
	`([0-9]+)% increased critical strike chance per point of strength or intelligence, whichever is lower`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), Tag{"type": "PerStat", "stat": "Str"}, Tag{"type": "Condition", "var": "IntHigherThanStr"}), mod("CritChance", "INC", c.n(1), Tag{"type": "PerStat", "stat": "Int"}, Tag{"type": "Condition", "neg": true, "var": "IntHigherThanStr"})}
	}),
	`consecrated ground you create causes life regeneration to also recover energy shield for you and allies`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("LifeRegenerationRecoversEnergyShield", Tag{"type": "Condition", "var": "OnConsecratedGround"})})},
	`([0-9]+)% more attack damage for each non-instant spell you've cast in the past 8 seconds, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, ModFlag.Attack, Tag{"type": "Multiplier", "var": "CastLast8Seconds", "limit": c.s(2), "limitTotal": true})}
	}),
	// Juggernaut
	`action speed cannot be modified to below base value`:   []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`movement speed cannot be modified to below base value`: []any{flag("MovementSpeedCannotBeBelowBase")},
	`you cannot be slowed to below base speed`:              []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`cannot be slowed to below base speed`:                  []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`gain accuracy rating equal to your strength`:           []any{mod("Accuracy", "BASE", 1, Tag{"type": "PerStat", "stat": "Str"})},
	`gain accuracy rating equal to twice your strength`:     []any{mod("Accuracy", "BASE", 2, Tag{"type": "PerStat", "stat": "Str"})},
	// Necromancer
	`your offering skills also affect you`: []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SkillData", "LIST", Tag{"key": "buffNotPlayer", "value": false})}, Tag{"type": "SkillName", "skillNameList": []any{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})},
	`your offerings have ([0-9]+)% reduced effect on you`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("BuffEffectOnPlayer", "INC", -c.n(1))}, Tag{"type": "SkillName", "skillNameList": []any{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`your offerings have ([0-9]+)% increased effect on you`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("BuffEffectOnPlayer", "INC", c.n(1))}, Tag{"type": "SkillName", "skillNameList": []any{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`if you've consumed a corpse recently, you and your minions have ([0-9]+)% increased area of effect`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Condition", "var": "ConsumedCorpseRecently"}), mod("MinionModifier", "LIST", Tag{"mod": mod("AreaOfEffect", "INC", c.n(1))}, Tag{"type": "Condition", "var": "ConsumedCorpseRecently"})}
	}),
	`with at least one nearby corpse, you and nearby allies deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("Damage", "MORE", c.n(1))}, Tag{"type": "MultiplierThreshold", "var": "NearbyCorpse", "threshold": 1})}
	}),
	`with at least one nearby corpse, nearby enemies deal ([0-9]+)% reduced damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("Damage", "INC", -c.n(1))}, Tag{"type": "MultiplierThreshold", "var": "NearbyCorpse", "threshold": 1})}
	}),
	`for each nearby corpse, you and nearby allies regenerate ([0-9.]+)% of energy shield per second, up to ([0-9.]+)% per second`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("EnergyShieldRegenPercent", "BASE", c.n(1))}, Tag{"type": "Multiplier", "var": "NearbyCorpse", "limit": c.n(2), "limitTotal": true})}
	}),
	`for each nearby corpse, you and nearby allies regenerate ([0-9]+) mana per second, up to ([0-9]+) per second`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("ManaRegen", "BASE", c.n(1))}, Tag{"type": "Multiplier", "var": "NearbyCorpse", "limit": c.n(2), "limitTotal": true})}
	}),
	`enemies near corpses you spawned recently are chilled and shocked`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Chilled")}, Tag{"type": "Condition", "var": "SpawnedCorpseRecently"}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Shocked")}, Tag{"type": "Condition", "var": "SpawnedCorpseRecently"}), mod("ChillBase", "BASE", nonDamagingAilmentDefault["Chill"], Tag{"type": "Condition", "var": "SpawnedCorpseRecently"}), mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "Condition", "var": "SpawnedCorpseRecently"})},
	`regenerate ([0-9]+)% of energy shield over 2 seconds when you consume a corpse`: fn(func(c caps) any {
		return []any{mod("EnergyShieldRegenPercent", "BASE", c.n(1)/2, Tag{"type": "Condition", "var": "ConsumedCorpseInPast2Sec"})}
	}),
	`regenerate ([0-9]+)% of mana over 2 seconds when you consume a corpse`: fn(func(c caps) any {
		return []any{mod("ManaRegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1) / 2}, Tag{"type": "Condition", "var": "ConsumedCorpseInPast2Sec"})}
	}),
	`corpses you spawn have ([0-9]+)% increased maximum life`: fn(func(c caps) any { return []any{mod("CorpseLife", "INC", c.n(1))} }),
	`corpses you spawn have ([0-9]+)% reduced maximum life`:   fn(func(c caps) any { return []any{mod("CorpseLife", "INC", -c.n(1))} }),
	`minions gain added physical damage equal to ([0-9]+)% of maximum energy shield on your equipped helmet`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("PhysicalMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShieldOnHelmet", "actor": "parent", "percent": c.n(1)})}), mod("MinionModifier", "LIST", Tag{"mod": mod("PhysicalMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShieldOnHelmet", "actor": "parent", "percent": c.n(1)})})}
	}),
	// Occultist
	`when you kill an enemy, for each curse on that enemy, gain ([0-9]+)% of non-chaos damage as extra chaos damage for 4 seconds`: fn(func(c caps) any {
		return []any{mod("NonChaosDamageGainAsChaos", "BASE", c.n(1), Tag{"type": "Condition", "var": "KilledRecently"}, Tag{"type": "Multiplier", "var": "CurseOnEnemy"})}
	}),
	`cannot be stunned while you have energy shield`:                        []any{flag("StunImmune", Tag{"type": "Condition", "var": "HaveEnergyShield"})},
	`every second, inflict withered on nearby enemies for ([0-9]+) seconds`: []any{flag("Condition:CanWither")},
	`nearby hindered enemies deal ([0-9]+)% reduced damage over time`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageOverTime", "INC", -c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Hindered"})}
	}),
	`nearby chilled enemies deal ([0-9]+)% reduced damage with hits`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("Damage", "INC", -c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"})}
	}),
	`gain spirit infusion every ?([0-9]?)\.?([0-9]?) seconds? while channelling a spell`: fn(func(c caps) any { return []any{flag("Condition:CanGainSpiritInfusion")} }),
	// Pathfinder
	`always poison on hit while using a flask`: []any{mod("PoisonChance", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"})},
	`poisons you inflict during any flask effect have ([0-9]+)% chance to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Poison, Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`immun[ei]t?y? to elemental ailments during any flask effect`:                                                                             []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grant bonuses to non-channelling skills you use by consuming ([0-9]+) charges from a flask of each of the following types, if possible:`: d(),
	`if diamond flask charges are consumed, ([0-9]+)% increased critical strike chance`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Triggered, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "Condition", "var": "HaveDiamondFlask"})}
	}),
	`if bismuth flask charges are consumed, penetrate ([0-9]+)% elemental resistances`: fn(func(c caps) any {
		return []any{mod("ElementalPenetration", "BASE", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Triggered, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "Condition", "var": "HaveBismuthFlask"})}
	}),
	`if amethyst flask charges are consumed, ([0-9]+)% of physical damage as extra chaos damage`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsChaos", "BASE", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Triggered, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "Condition", "var": "HaveAmethystFlask"})}
	}),
	// Raider
	`nearby enemies have ([0-9]+)% less accuracy rating while you have phasing`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("Accuracy", "MORE", -c.n(1))}, Tag{"type": "Condition", "var": "Phasing"})}
	}),
	`immun[ei]t?y? to elemental ailments while phasing`: []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "Phasing"})},
	`nearby enemies have fire, cold and lightning exposure while you have phasing, applying -([0-9]+)% to those resistances`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -c.n(1))}, Tag{"type": "Condition", "var": "Phasing"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -c.n(1))}, Tag{"type": "Condition", "var": "Phasing"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -c.n(1))}, Tag{"type": "Condition", "var": "Phasing"})}
	}),
	`nearby enemies have fire, cold and lightning exposure while you have phasing`: []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Phasing"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Phasing"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Phasing"})},
	// Saboteur
	`hits have ([0-9]+)% chance to deal ([0-9]+)% more area damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", (c.n(1) * c.n(2) / 100), nil, ModFlag.Area|ModFlag.Hit)}
	}),
	`immun[ei]t?y? to ignite and shock`: []any{flag("IgniteImmune"), flag("ShockImmune")},
	`you gain ([0-9]+)% increased damage for each trap`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), Tag{"type": "PerStat", "stat": "ActiveTrapLimit"})}
	}),
	`you gain ([0-9]+)% increased area of effect for each mine`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "PerStat", "stat": "ActiveMineLimit"})}
	}),
	`triggers level ([0-9]+) summon triggerbots when allocated`: []any{flag("HaveTriggerBots")},
	// Slayer
	`deal up to ([0-9]+)% more melee damage to enemies, based on proximity`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, ModFlag.Attack|ModFlag.Melee, Tag{"type": "MeleeProximity", "ramp": []any{1, 0}})}
	}),
	`cannot be stunned while leeching`:                                               []any{flag("StunImmune", Tag{"type": "Condition", "var": "Leeching"})},
	`you are immune to bleeding while leeching`:                                      []any{flag("BleedImmune", Tag{"type": "Condition", "var": "Leeching"})},
	`life leech effects are not removed at full life`:                                []any{flag("CanLeechLifeOnFullLife")},
	`life leech effects are not removed when unreserved life is filled`:              []any{flag("CanLeechLifeOnFullLife")},
	`energy shield leech effects from attacks are not removed at full energy shield`: []any{flag("CanLeechEnergyShieldOnFullEnergyShield")},
	`cannot take reflected physical damage`:                                          []any{mod("PhysicalReflectedDamageTaken", "MORE", -100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`gain ([0-9]+)% increased movement speed for 20 seconds when you kill an enemy`: fn(func(c caps) any {
		return []any{mod("MovementSpeed", "INC", c.n(1), Tag{"type": "Condition", "var": "KilledRecently"})}
	}),
	`gain ([0-9]+)% increased attack speed for 20 seconds when you kill a rare or unique enemy`: fn(func(c caps) any {
		return []any{mod("Speed", "INC", c.n(1), nil, ModFlag.Attack, 0, Tag{"type": "Condition", "var": "KilledUniqueEnemy"})}
	}),
	`kill enemies that have ([0-9]+)% or lower life when hit by your skills`: fn(func(c caps) any { return []any{mod("CullPercent", "MAX", c.n(1))} }),
	`you are unaffected by bleeding while leeching`:                          []any{mod("SelfBleedEffect", "MORE", -100, Tag{"type": "Condition", "var": "Leeching"})},
	// Trickster
	`([0-9]+)% chance to gain ([0-9]+)% of non-chaos damage with hits as extra chaos damage`: fn(func(c caps) any { return []any{mod("NonChaosDamageGainAsChaos", "BASE", c.n(1)/100*c.n(2))} }),
	`movement skills cost no mana`: []any{mod("ManaCost", "MORE", -100, nil, 0, KeywordFlag.Movement)},
	`cannot be stunned while you have ghost shrouds`: fn(func(c caps) any {
		return []any{flag("StunImmune", Tag{"type": "MultiplierThreshold", "var": "GhostShroud", "threshold": 1})}
	}),
	`your action speed is at least ([0-9]+)% of base value`: fn(func(c caps) any { return []any{mod("MinimumActionSpeed", "MAX", c.n(1))} }),
	`nearby enemy monsters' action speed is at most ([0-9]+)% of base value`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("MaximumActionSpeedReduction", "MAX", 100-c.n(1))})}
	}),
	`prevent \+([0-9]+)% of suppressed spell damage while on full energy shield`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionEffect", "BASE", c.n(1), Tag{"type": "Condition", "var": "FullEnergyShield"})}
	}),
	`energy shield leech effects are not removed when energy shield is filled`: []any{flag("CanLeechEnergyShieldOnFullEnergyShield")},
	`take ([0-9]+)% less damage from hits for ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("DamageTakenWhenHit", "MORE", -c.n(1), Tag{"type": "Condition", "var": "HeartstopperHIT"}), mod("DamageTakenWhenHit", "MORE", -c.n(1)*c.n(2)/10, Tag{"type": "Condition", "var": "HeartstopperAVERAGE"})}
	}),
	`take ([0-9]+)% less damage over time for ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("DamageTakenOverTime", "MORE", -c.n(1), Tag{"type": "Condition", "var": "HeartstopperDOT"}), mod("DamageTakenOverTime", "MORE", -c.n(1)*c.n(2)/10, Tag{"type": "Condition", "var": "HeartstopperAVERAGE"})}
	}),
	// Warden
	`prevent \+([0-9]+)% of suppressed spell damage per bark below maximum`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionEffect", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "MissingBarkskinStacks"})}
	}),
	`hits that would ignite instead scorch`:                                          []any{flag("IgniteCanScorch"), flag("CannotIgnite")},
	`you can inflict an additional scorch on each enemy`:                             []any{flag("ScorchCanStack"), mod("ScorchStacksMax", "BASE", 1)},
	`maximum effect of shock is ([0-9]+)% increased damage taken`:                    fn(func(c caps) any { return []any{mod("ShockMax", "OVERRIDE", c.n(1))} }),
	`you can apply up to ([0-9]+) shocks to each enemy`:                              fn(func(c caps) any { return []any{flag("ShockCanStack"), mod("ShockStacksMax", "OVERRIDE", c.n(1))} }),
	`hits that fail to freeze due to insufficient freeze duration inflict hoarfrost`: []any{flag("HitsCanInflictHoarfrost")},
	`your hits always inflict freeze, shock and ignite while unbound`:                []any{mod("EnemyFreezeChance", "BASE", 100, Tag{"type": "Condition", "var": "Unbound"}), mod("EnemyShockChance", "BASE", 100, Tag{"type": "Condition", "var": "Unbound"}), mod("EnemyIgniteChance", "BASE", 100, Tag{"type": "Condition", "var": "Unbound"})},
	`([0-9]+)% more elemental damage while unbound`: fn(func(c caps) any {
		return []any{mod("ElementalDamage", "MORE", c.n(1), Tag{"type": "Condition", "var": "Unbound"})}
	}),
	// Warden (Affliction)
	`defences from equipped body armour are doubled if it has no socketed gems`: []any{mod("Defences", "MORE", 100, Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInBody Armour", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingBody Armour"}, Tag{"type": "SlotName", "slotName": "Body Armour"}, Tag{"type": "Multiplier", "var": "OathOfTheMajiDoubled", "globalLimit": 100, "globalLimitKey": "OathOfTheMajiLimit"}), mod("Multiplier:OathOfTheMajiDoubled", "OVERRIDE", 1, Tag{"type": "SlotName", "slotName": "Body Armour"})},
	`([+\-][0-9]+)% to all elemental resistances if you have an equipped helmet with no socketed gems`: fn(func(c caps) any {
		return []any{mod("ElementalResist", "BASE", c.n(1), Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInHelmet", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingHelmet"})}
	}),
	`([0-9]+)% increased maximum life if you have equipped gloves with no socketed gems`: fn(func(c caps) any {
		return []any{mod("Life", "INC", c.n(1), Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInGloves", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingGloves"})}
	}),
	`([0-9]+)% increased movement speed if you have equipped boots with no socketed gems`: fn(func(c caps) any {
		return []any{mod("MovementSpeed", "INC", c.n(1), Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInBoots", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingBoots"})}
	}),
	// Warlock
	`spells you cast yourself gain added physical damage equal to ([0-9]+)% of life cost, if life cost is not higher than the maximum you could spend`: fn(func(c caps) any {
		return []any{mod("PhysicalMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "LifeCost", "percent": c.n(1)}, Tag{"type": "StatThreshold", "stat": "LifeUnreserved", "thresholdStat": "LifeCost", "thresholdPercent": c.n(1)}), mod("PhysicalMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "LifeCost", "percent": c.n(1)}, Tag{"type": "StatThreshold", "stat": "LifeUnreserved", "thresholdStat": "LifeCost", "thresholdPercent": c.n(1)})}
	}),
	`gain maximum life instead of maximum energy shield from equipped armour items`: []any{flag("ConvertArmourESToLife")},
	// Ritualist Bloodline
	`unaffected by bleeding`:      []any{mod("SelfBleedEffect", "MORE", -100)},
	`unaffected by poison`:        []any{mod("SelfPoisonEffect", "MORE", -100)},
	`can't use amulets`:           []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Amulet"})},
	`can't use belts`:             []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Belt"})},
	`\+1 ring slot`:               []any{flag("AdditionalRingSlot")},
	`utility flasks are disabled`: []any{flag("UtilityFlasksDoNotApplyToPlayer")},
	// Aul Bloodline
	`action speed cannot be modified to below base value if you have equipped boots with no socketed gems`:    []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInBoots", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingBoots"})},
	`cannot be stunned if you have an equipped helmet with no socketed gems`:                                  []any{flag("StunImmune", Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInHelmet", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingHelmet"})},
	`elemental ailments cannot be inflicted on you if you have an equipped body armour with no socketed gems`: []any{flag("ElementalAilmentImmune", Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInBody Armour", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingBody Armour"})},
	`take no extra damage from critical strikes if you have equipped gloves with no socketed gems`:            []any{mod("ReduceCritExtraDamage", "BASE", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "MultiplierThreshold", "var": "SocketedGemsInGloves", "threshold": 0, "upper": true}, Tag{"type": "Condition", "var": "UsingGloves"})},
	// Delirious Bloodline
	`while affected by glorious madness, inflict mania on nearby enemies every second`: []any{flag("Condition:CanInflictMania", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:AfflictedByMania")}, Tag{"type": "Condition", "var": "AffectedByGloriousMadness"})},
	// Lycia Bloodline
	`herald skills have ([0-9]+)% more buff effect for every ([0-9]+)% of maximum mana they reserve`: fn(func(c caps) any {
		return []any{mod("BuffEffect", "MORE", c.n(1), Tag{"type": "PerStat", "stat": "ManaReservedPercent", "div": c.n(2)}, Tag{"type": "SkillType", "skillType": SkillType.Herald})}
	}),
	`herald skills and minions from herald skills deal ([0-9]+)% more damage for every ([0-9]+)% of maximum life those skills reserve`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "MORE", c.n(1), Tag{"type": "PerStat", "stat": "LifeReservedPercent", "div": c.n(2), "actor": "parent"})}, Tag{"type": "SkillType", "skillType": SkillType.Herald}), mod("Damage", "MORE", c.n(1), Tag{"type": "PerStat", "stat": "LifeReservedPercent", "div": c.n(2)}, Tag{"type": "SkillType", "skillType": SkillType.Herald})}
	}),
	// Oshabi Bloodline
	`unsealed spells gain ([0-9]+)% more damage each time their effects reoccur`: fn(func(c caps) any { return []any{mod("MaxSealDamage", "MORE", c.n(1))} }),
	`skills gain added chaos damage equal to ([0-9]+)% of life cost, if life cost is not higher than the maximum you could spend`: fn(func(c caps) any {
		return []any{mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "LifeCost", "percent": c.n(1)}, Tag{"type": "StatThreshold", "stat": "LifeUnreserved", "thresholdStat": "LifeCost", "thresholdPercent": c.n(1)}), mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "LifeCost", "percent": c.n(1)}, Tag{"type": "StatThreshold", "stat": "LifeUnreserved", "thresholdStat": "LifeCost", "thresholdPercent": c.n(1)})}
	}),
	`lose all rage on reaching maximum rage and gain wild savagery for 1 second per 10 rage lost this way`: []any{flag("WildSavagery")},
	// Velka Bloodline
	`inflict barnacles on nearby enemies every second`:  []any{flag("CanInflictBarnacles")},
	`drop brine ground while moving, lasting 4 seconds`: []any{flag("CanCreateBrineGround")},
	// Item local modifiers
	`has no sockets`: []any{flag("NoSockets")},
	`gems socketed always have the quality bonus from socket colour`: []any{flag("SocketAlwaysMatches")},
	`cannot have non-abyssal sockets`:                                []any{flag("NoSockets")},
	`socketed [a-zA-Z]+ abyssal jewels will be consumed`:             d(),
	`one modifier from consumed jewels will be retained`:             d(),
	`reflects your o[tp][hp][eo][rs]i?t?e? ring`:                     d(),
	`cannot gain intangibility`:                                      d(),
	`has ([0-9]+) sockets?`:                                          fn(func(c caps) any { return []any{mod("SocketCount", "BASE", c.n(1))} }),
	`has ([0-9]+) abyssal sockets?`:                                  fn(func(c caps) any { return []any{mod("AbyssalSocketCount", "BASE", c.n(1))} }),
	`no physical damage`:                                             []any{mod("WeaponData", "LIST", Tag{"key": "PhysicalMin"}), mod("WeaponData", "LIST", Tag{"key": "PhysicalMax"}), mod("WeaponData", "LIST", Tag{"key": "PhysicalDPS"})},
	`has ([0-9]+)% increased elemental damage`:                       fn(func(c caps) any { return []any{mod("LocalElementalDamage", "INC", c.n(1))} }),
	`all attacks with this weapon are critical strikes`:              []any{mod("WeaponData", "LIST", Tag{"key": "CritChance", "value": 100})},
	`this weapon's critical strike chance is ([0-9]+)%`:              fn(func(c caps) any { return []any{mod("WeaponData", "LIST", Tag{"key": "CritChance", "value": c.n(1)})} }),
	`counts as dual wielding`:                                        []any{mod("WeaponData", "LIST", Tag{"key": "countsAsDualWielding", "value": true})},
	`counts as all one handed melee weapon types`:                    []any{mod("WeaponData", "LIST", Tag{"key": "countsAsAll1H", "value": true})},
	`no block chance`:                                                []any{mod("ArmourData", "LIST", Tag{"key": "BlockChance", "value": 0})},
	`no chance to block`:                                             []any{mod("ArmourData", "LIST", Tag{"key": "BlockChance", "value": 0})},
	`has no energy shield`:                                           []any{mod("ArmourData", "LIST", Tag{"key": "EnergyShield", "value": 0})},
	`hits can't be evaded`:                                           []any{flag("CannotBeEvaded", Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`causes bleeding on hit`:                                         []any{mod("BleedChance", "BASE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`poisonous hit`:                                                  []any{mod("PoisonChance", "BASE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`attacks with this weapon deal double damage`:                    []any{mod("DoubleDamageChance", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})},
	`hits with this weapon gain ([0-9]+)% of physical damage as extra cold or lightning damage`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsColdOrLightning", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`hits with this weapon shock enemies as though dealing ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("ShockAsThoughDealing", "MORE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`hits with this weapon freeze enemies as though dealing ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("FreezeAsThoughDealing", "MORE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`ignites inflicted with this weapon deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Ignite, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`hits with this weapon always ignite, freeze, and shock`:         []any{mod("EnemyIgniteChance", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), mod("EnemyFreezeChance", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), mod("EnemyShockChance", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})},
	`attacks with this weapon deal double damage to chilled enemies`: []any{mod("DoubleDamageChance", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"})},
	`life leech from hits with this weapon applies instantly`:        []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`life leech from hits with this weapon is instant`:               []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`mana leech from hits with this weapon is instant`:               []any{mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`life leech from melee damage is instant`:                        []any{mod("InstantLifeLeech", "BASE", 100, nil, ModFlag.Melee)},
	`gain life from leech instantly from hits with this weapon`:      []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})},
	`([0-9]+)% of leech from hits with this weapon is instant per enemy power`: fn(func(c caps) any {
		return []any{mod("InstantLifeLeech", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "Multiplier", "var": "EnemyPower"}), mod("InstantManaLeech", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "Multiplier", "var": "EnemyPower"}), mod("InstantEnergyShieldLeech", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "Multiplier", "var": "EnemyPower"})}
	}),
	`instant recovery`:                                                                                       []any{mod("FlaskInstantRecovery", "BASE", 100)},
	`([0-9]+)% of recovery applied instantly`:                                                                fn(func(c caps) any { return []any{mod("FlaskInstantRecovery", "BASE", c.n(1))} }),
	`instant recovery when on low life`:                                                                      []any{mod("FlaskLowLifeInstantRecovery", "BASE", 100), mod("Dummy", "DUMMY", 1, "", Tag{"type": "Condition", "var": "LowLife"})},
	`life flasks used while on low life apply recovery instantly`:                                            []any{mod("LifeFlaskInstantRecovery", "BASE", 100, Tag{"type": "Condition", "var": "LowLife"})},
	`mana flasks used while on low mana apply recovery instantly`:                                            []any{mod("ManaFlaskInstantRecovery", "BASE", 100, Tag{"type": "Condition", "var": "LowMana"})},
	`has no attribute requirements`:                                                                          []any{flag("NoAttributeRequirements")},
	`trigger a socketed spell when you attack with this weapon`:                                              []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you attack with this weapon, with a ([0-9.]+) second cooldown`:            []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you use a skill`:                                                          []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnSkillUse", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you use a skill, with a ([0-9]+) second cooldown`:                         []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnSkillUse", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you use a skill, with a ([0-9]+) second cooldown and ([0-9]+)% more cost`: []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnSkillUse", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger socketed spells when you focus`:                                                                 []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellFromHelmet", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger socketed spells when you focus, with a ([0-9.]+) second cooldown`:                               []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellFromHelmet", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you attack with a bow`:                                                    []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnBowAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you attack with a bow, with a ([0-9.]+) second cooldown`:                  []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnBowAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed bow skill when you attack with a bow`:                                                []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerBowSkillOnBowAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed bow skill when you attack with a bow, with a ([0-9.]+) second cooldown`:              []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerBowSkillOnBowAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed bow skill when you cast a spell while wielding a bow`:                                []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerBowSkillOnBowAttack", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`([0-9]+)% chance to trigger socketed spell on kill, with a ([0-9.]+) second cooldown`:                   []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnKill", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`([0-9]+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least ([0-9]+) mana to use a skill`: fn(func(c caps) any {
		return []any{mod("KitavaTriggerChance", "BASE", c.n(1), "Kitava's Thirst"), mod("KitavaRequiredManaCost", "BASE", c.n(2), "Kitava's Thirst"), mod("ExtraSupport", "LIST", Tag{"skillId": "SupportCastOnManaSpent", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`([0-9]+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least ([0-9]+) mana on an upfront cost to use or trigger a skill, with a ([0-9.]+) second cooldown`: fn(func(c caps) any {
		return []any{mod("KitavaTriggerChance", "BASE", c.n(1), "Kitava's Thirst"), mod("KitavaRequiredManaCost", "BASE", c.n(2), "Kitava's Thirst"), mod("ExtraSupport", "LIST", Tag{"skillId": "SupportCastOnManaSpent", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`([0-9]+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least ([0-9]+) life on an upfront cost to use or trigger a skill, with a ([0-9.]+) second cooldown`: fn(func(c caps) any {
		return []any{mod("FoulbornKitavaTriggerChance", "BASE", c.n(1), "Kitava's Thirst"), mod("FoulbornKitavaRequiredLifeCost", "BASE", c.n(2), "Kitava's Thirst"), mod("ExtraSupport", "LIST", Tag{"skillId": "SupportCastOnLifeSpent", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`trigger a socketed fire spell on hit, with a ([0-9.]+) second cooldown`: []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerFireSpellOnHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	// Socketed gem modifiers
	`([+\-][0-9]+) to level of socketed gems`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "all", "key": "level", "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`([+\-][0-9]+) to level of socketed skill gems per socketed gem`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "grants_active_skill", "key": "level", "value": c.n(1), "keyOfScaledMod": "value"}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "Multiplier", "var": "SocketedGemsIn{SlotName}"})}
	}),
	`([+\-][0-9]+)% to quality of all skill gems`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "grants_active_skill", "key": "quality", "value": c.n(1), "keyOfScaledMod": "value"})}
	}),
	`([+\-][0-9]+) to level of all elemental skill gems if the stars are aligned`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keywordList": []any{"elemental", "grants_active_skill"}, "key": "level", "value": c.n(1), "keyOfScaledMod": "value"}, Tag{"type": "Condition", "var": "StarsAreAligned"})}
	}),
	`([+\-][0-9]+) to level of all elemental support gems if the stars are aligned`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keywordList": []any{"elemental", "support"}, "key": "level", "value": c.n(1), "keyOfScaledMod": "value"}, Tag{"type": "Condition", "var": "StarsAreAligned"})}
	}),
	`([+\-][0-9]+) to level of socketed active skill gems per ([0-9]+) player levels`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "grants_active_skill", "key": "level", "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "Multiplier", "var": "Level", "div": c.n(2)})}
	}),
	`([+\-][0-9]+) to level of all ([a-zA-Z]+) skill gems if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": c.s(2), "key": "level", "value": c.n(1), "keyOfScaledMod": "value"}, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(4)) + "Item", "threshold": c.n(3)})}
	}),
	`([+\-][0-9]+) to level of all ([a-zA-Z]+) skill gems if a?t? ?l?e?a?s?t? ?([0-9]+) ([a-zA-Z]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": c.s(2), "key": "level", "value": c.n(1), "keyOfScaledMod": "value"}, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(4)) + firstToUpper(c.s(5)) + "Item", "threshold": c.n(3)})}
	}),
	`([+\-][0-9]+) to level of all ?([a-zA-Z\- ]*) support gems if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": c.s(2), "key": "level", "value": c.n(1), "keyOfScaledMod": "value"}, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(4)) + "Item", "threshold": c.n(3)})}
	}),
	`([+\-][0-9]+) to level of socketed skill gems per ([0-9]+) player levels`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "grants_active_skill", "key": "level", "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "Multiplier", "var": "Level", "div": c.n(2)})}
	}),
	`([+\-][0-9]+) to level of socketed gems while there is a single gem socketed in this item`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "all", "key": "level", "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "MultiplierThreshold", "var": "SocketedGemsIn{SlotName}", "threshold": 1, "equals": true})}
	}),
	`socketed gems fire an additional projectile`: []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", 1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed gems fire ([0-9]+) additional projectiles`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`socketed gems reserve no mana`:     []any{mod("ManaReserved", "MORE", -100, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed gems have no reservation`: []any{mod("Reserved", "MORE", -100, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed skill gems get a ([0-9]+)% mana multiplier`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SupportManaMultiplier", "MORE", c.n(1)-100)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`socketed skill gems get a ([0-9]+)% cost & reservation multiplier`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SupportManaMultiplier", "MORE", c.n(1)-100)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`socketed gems have blood magic`:                      []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportBloodMagicUniquePrismGuardian", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed gems cost and reserve life instead of mana`: []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportBloodMagicUniquePrismGuardian", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed gems have elemental equilibrium`:            []any{mod("Keystone", "LIST", "Elemental Equilibrium")},
	`socketed gems have secrets of suffering`:             []any{flag("CannotIgnite", Tag{"type": "SocketedIn", "slotName": "{SlotName}"}), flag("CannotChill", Tag{"type": "SocketedIn", "slotName": "{SlotName}"}), flag("CannotFreeze", Tag{"type": "SocketedIn", "slotName": "{SlotName}"}), flag("CannotShock", Tag{"type": "SocketedIn", "slotName": "{SlotName}"}), flag("CritAlwaysAltAilments", Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed skills deal double damage`:                  []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("DoubleDamageChance", "BASE", 100)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed gems gain ([0-9]+)% of physical damage as extra lightning damage`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("PhysicalDamageGainAsLightning", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`socketed red gems get ([0-9]+)% physical damage as extra fire damage`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("PhysicalDamageGainAsFire", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "strength"})}
	}),
	`socketed non-channelling bow skills are triggered by snipe`: d(),
	`grants level ([0-9]+) snipe skill`: fn(func(c caps) any {
		return []any{mod("ExtraSkill", "LIST", Tag{"skillId": "Snipe", "level": c.n(1)}), mod("ExtraSupport", "LIST", Tag{"skillId": "ChannelledSnipeSupport", "level": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`socketed triggered bow skills deal ([0-9]+)% less damage`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("Damage", "MORE", -c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "bow"}, Tag{"type": "SkillType", "skillType": SkillType.Triggerable})}
	}),
	`socketed vaal skills require ([0-9]+)% less souls per use`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SoulCost", "MORE", -c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "SkillType", "skillType": SkillType.Vaal})}
	}),
	`hits from socketed vaal skills ignore enemy monster resistances`:               []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("IgnoreElementalResistances")}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "SkillType", "skillType": SkillType.Vaal}), mod("ExtraSkillMod", "LIST", Tag{"mod": flag("IgnoreChaosResistance")}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "SkillType", "skillType": SkillType.Vaal})},
	`hits from socketed vaal skills ignore enemy monster physical damage reduction`: []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("IgnoreEnemyPhysicalDamageReduction")}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "SkillType", "skillType": SkillType.Vaal})},
	`socketed vaal skills grant elusive when used`:                                  []any{flag("Condition:CanBeElusive")},
	`damage with hits from socketed vaal skills is lucky`:                           []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("LuckyHits")}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "SkillType", "skillType": SkillType.Vaal})},
	// Global gem modifiers
	`gems socketed in red sockets have [+\-]([0-9]+) to level`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "all", "key": "level", "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "socketColor": "R"})}
	}),
	`gems socketed in green sockets have [+\-]([0-9]+)% to quality`: fn(func(c caps) any {
		return []any{mod("GemProperty", "LIST", Tag{"keyword": "all", "key": "quality", "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "socketColor": "G"})}
	}),
	`\+([0-9]+)% to fire resistance when socketed with a red gem`: fn(func(c caps) any {
		return []any{mod("SocketProperty", "LIST", Tag{"value": mod("FireResist", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "strength", "sockets": []any{1}})}
	}),
	`\+([0-9]+)% to cold resistance when socketed with a green gem`: fn(func(c caps) any {
		return []any{mod("SocketProperty", "LIST", Tag{"value": mod("ColdResist", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "dexterity", "sockets": []any{1}})}
	}),
	`\+([0-9]+)% to lightning resistance when socketed with a blue gem`: fn(func(c caps) any {
		return []any{mod("SocketProperty", "LIST", Tag{"value": mod("LightningResist", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "intelligence", "sockets": []any{1}})}
	}),
	// Doomsower, Lion Sword
	`attack skills gain ([0-9]+)% of physical damage as extra fire damage per socketed red gem`: fn(func(c caps) any {
		return []any{mod("SocketProperty", "LIST", Tag{"value": mod("PhysicalDamageGainAsFire", "BASE", c.n(1), nil, ModFlag.Attack)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "strength", "sockets": []any{1, 2, 3, 4, 5, 6}})}
	}),
	`([0-9]+)% of damage taken recouped as life per socketed red gem`: fn(func(c caps) any {
		return []any{mod("SocketProperty", "LIST", Tag{"value": mod("LifeRecoup", "BASE", c.n(1))}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "keyword": "strength", "sockets": []any{1, 2, 3, 4, 5, 6}})}
	}),
	`you have vaal pact while all socketed gems are red`:         []any{mod("GroupProperty", "LIST", Tag{"value": mod("Keystone", "LIST", "Vaal Pact")}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "socketColor": "R", "sockets": "all"})},
	`you have immortal ambition while all socketed gems are red`: []any{mod("GroupProperty", "LIST", Tag{"value": mod("Keystone", "LIST", "Immortal Ambition")}, Tag{"type": "SocketedIn", "slotName": "{SlotName}", "socketColor": "R", "sockets": "all"})},
	// Mahuxotl's Machination Steel Kite Shield
	`everlasting sacrifice`: []any{flag("Condition:EverlastingSacrifice")},
	// Self hit dmg
	`take ([0-9]+) (.+) damage when you ignite an enemy`: fn(func(c caps) any {
		return []any{mod("EyeOfInnocenceSelfDamage", "LIST", Tag{"baseDamage": c.n(1), "damageType": c.s(2)})}
	}),
	`([0-9]+) (.+) damage taken on minion death`: fn(func(c caps) any {
		return []any{mod("HeartboundLoopSelfDamage", "LIST", Tag{"baseDamage": c.n(1), "damageType": c.s(2)})}
	}),
	`take ([0-9]+) (.+) damage when herald of thunder hits an enemy`: fn(func(c caps) any {
		return []any{mod("StormSecretSelfDamage", "LIST", Tag{"baseDamage": c.n(1), "damageType": c.s(2)})}
	}),
	`take ([0-9]+) (.+) damage when you use a skill`: fn(func(c caps) any {
		return []any{mod("EnmitysEmbraceSelfDamage", "LIST", Tag{"baseDamage": c.n(1), "damageType": c.s(2)})}
	}),
	`your skills deal you ([0-9]+)% of mana cost as (.+) damage`: fn(func(c caps) any {
		return []any{mod("ScoldsBridleSelfDamage", "LIST", Tag{"dmgMult": c.n(1), "damageType": c.s(2)})}
	}),
	`your skills deal you ([0-9]+)% of mana spent on upfront skill mana costs as (.+) damage`: fn(func(c caps) any {
		return []any{mod("ScoldsBridleSelfDamage", "LIST", Tag{"dmgMult": c.n(1), "damageType": c.s(2)})}
	}),
	`when you attack, take ([0-9]+)% of life as (.+) damage for each warcry exerting the attack`: fn(func(c caps) any {
		return []any{mod("EchoesOfCreationSelfDamage", "LIST", Tag{"dmgMult": c.n(1), "damageType": c.s(2)})}
	}),
	// Extra skill/support
	`grants ([^0-9]+)`:           fn(func(c caps) any { return grantedExtraSkill(c.s(1), 1) }),
	`grants level ([0-9]+) (.+)`: fn(func(c caps) any { return grantedExtraSkill(c.s(2), c.n(1)) }),
	`grants level ([0-9]+) (.+), which will be used by shaper memory`:                                               fn(func(c caps) any { return grantedExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when equipped`:                                                    fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) on [a-zA-Z]+`:                                                     fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`use level ([0-9]+) (.+) on [a-zA-Z]+`:                                                                          fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you attack`:                                                  fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you deal a critical strike`:                                  fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when hit`:                                                         fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you kill an enemy`:                                           fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you use a skill`:                                             fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`(.+) can trigger level ([0-9]+) (.+)`:                                                                          fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.n(2), Tag{"sourceSkill": c.s(1)}) }),
	`trigger level ([0-9]+) (.+) when you use a skill while you have a spirit charge`:                               fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit an enemy while cursed`:                                                fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a bleeding enemy`:                                                     fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you attack with this weapon`:                                                  fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a rare or unique enemy`:                                               fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a rare or unique enemy and have no mark`:                              fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a frozen enemy`:                                                       fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you kill a frozen enemy`:                                                      fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you attack with a bow`:                                                        fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you block`:                                                                    fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when animated guardian kills an enemy`:                                             fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you lose cat's stealth`:                                                       fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when your trap is triggered`:                                                       fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on hit with this weapon`:                                                           fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit`:                                                                      fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit while cursed`:                                                         fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit with this weapon`:                                                     fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) every [0-9.]+ seconds while phasing`:                                               fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you gain avian's might or avian's flight`:                                     fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit if you have at least ([0-9]+) strength`:                               fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on critical strike with cleave or reave`:                                           fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1), Tag{"onCrit": true}) }),
	`trigger level ([0-9]+) (.+) on melee critical strike`:                                                          fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1), Tag{"onCrit": true}) }),
	`trigger level ([0-9]+) (.+) on critical strike against marked unique enemy`:                                    fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1), Tag{"onCrit": true}) }),
	`trigger level ([0-9]+) (.+) on critical strike`:                                                                fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1), Tag{"onCrit": true}) }),
	`trigger level ([0-9]+) (.+) when you take a critical strike from a unique enemy`:                               fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you suppress spell damage from a unique enemy`:                                fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you block damage from a unique enemy`:                                         fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on taking a savage hit from a unique enemy`:                                        fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when a totem dies while a unique enemy is in your presence`:                        fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you use a travel skill while a unique enemy is in your presence`:              fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you reach maximum rage while a unique enemy is in your presence`:              fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you reach low life while a unique enemy is in your presence`:                  fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when energy shield recharge starts while a unique enemy is in your presence`:       fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when your ward breaks`:                                                             fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) once every second`:                                                                 fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`triggers level ([0-9]+) (.+)`:                                                                                  fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on attack critical strike against a rare or unique enemy and y?o?u? ?have no mark`: fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1), Tag{"onCrit": true}) }),
	`triggers level ([0-9]+) (.+) when equipped`:                                                                    fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`triggers level ([0-9]+) (.+) when allocated`:                                                                   fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`([0-9]+)% chance to attack with level ([0-9]+) (.+) on melee hit`:                                              fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when animated weapon kills an enemy`:                           fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) on melee hit`:                                                  fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) [ow][nh]e?n? ?y?o?u? kill ?a?n? ?e?n?e?m?y?`:                   fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when you use a socketed skill`:                                 fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when you gain avian's might or avian's flight`:                 fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) on critical strike with this weapon`: fn(func(c caps) any {
		return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1), "onCrit": true})
	}),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when you or a nearby ally kill an enemy, or hit a rare or unique enemy`: fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to trigger (.+) when you kill an enemy`:                                                                fn(func(c caps) any { return triggerExtraSkill(c.s(2), 1, Tag{"triggerChance": c.n(1)}) }),
	`([0-9]+)% chance to [ct][ar][si][tg]g?e?r? level ([0-9]+) (.+) on [a-zA-Z]+`:                                            fn(func(c caps) any { return triggerExtraSkill(c.s(3), c.s(2), Tag{"triggerChance": c.n(1)}) }),
	`attack with level ([0-9]+) (.+) when you kill a bleeding enemy`:                                                         fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`triggers? level ([0-9]+) (.+) when you kill a bleeding enemy`:                                                           fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`curse enemies with ([^0-9]+) on [a-zA-Z]+`:                                                                              fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[0-9]+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with ([^0-9]+) on [a-zA-Z]+`: fn(func(c caps) any {
		return []any{mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(1)), "level": 1, "noSupports": true, "triggered": true})}
	}),
	`curse enemies with level ([0-9]+) ([^0-9]+) on [a-zA-Z]+, which can apply to hexproof enemies`: fn(func(c caps) any {
		return triggerExtraSkill(c.s(2), c.n(1), Tag{"noSupports": true, "ignoreHexproof": true})
	}),
	`curse enemies with level ([0-9]+) (.+) on [a-zA-Z]+`:                      fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1), Tag{"noSupports": true}) }),
	`[ct][ar][si][tg]g?e?r?s? (.+) on [a-zA-Z]+`:                               fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[at][tr][ti][ag][cg][ke]r? (.+) on [a-zA-Z]+`:                             fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[at][tr][ti][ag][cg][ke]r? with (.+) on [a-zA-Z]+`:                        fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[ct][ar][si][tg]g?e?r?s? (.+) when hit`:                                   fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[at][tr][ti][ag][cg][ke]r? (.+) when hit`:                                 fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[at][tr][ti][ag][cg][ke]r? with (.+) when hit`:                            fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[ct][ar][si][tg]g?e?r?s? (.+) when your skills or minions kill`:           fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[at][tr][ti][ag][cg][ke]r? (.+) when you take a critical strike`:          fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`[at][tr][ti][ag][cg][ke]r? with (.+) when you take a critical strike`:     fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`trigger commandment of inferno on critical strike`:                        []any{mod("ExtraSkill", "LIST", Tag{"skillId": "UniqueEnchantmentOfInfernoOnCrit", "level": 1, "noSupports": true, "triggered": true}), mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SkillData", "LIST", Tag{"key": "triggerOnCrit", "value": true})}, Tag{"type": "SkillId", "skillId": "UniqueEnchantmentOfInfernoOnCrit"})},
	`trigger (.+) on critical strike`:                                          fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true, "onCrit": true}) }),
	`triggers? (.+) when you take a critical strike`:                           fn(func(c caps) any { return triggerExtraSkill(c.s(1), 1, Tag{"noSupports": true}) }),
	`socketed [a-zA-Z+]* ?gems a?r?e? ?supported by level ([0-9]+) (.+)`:       fn(func(c caps) any { return extraSupport(c.s(2), c.n(1)) }),
	`socketed [a-zA-Z+]* ?spells a?r?e? ?supported by level ([0-9]+) (.+)`:     fn(func(c caps) any { return extraSupport(c.s(2), c.n(1)) }),
	`skills from equipped (.+) are supported by level ([0-9]+) (.+)`:           fn(func(c caps) any { return extraSupport(c.s(3), c.s(2), c.s(1)) }),
	`skills socketed in your (.+) are supported by level ([0-9]+) (.+)`:        fn(func(c caps) any { return extraSupport(c.s(3), c.s(2), c.s(1)) }),
	`socketed hex curse skills are triggered by doedre's effigy when summoned`: []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportCursePillarTriggerCurses", "level": 20}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`socketed projectile spells have \+([0-9.]+) seconds to cooldown`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "BASE", c.n(1), Tag{"type": "SocketedIn", "slotName": "{SlotName}"}, Tag{"type": "SkillType", "skillType": SkillType.Projectile}, Tag{"type": "SkillType", "skillType": SkillType.Spell})}
	}),
	`trigger level ([0-9]+) (.+) every ([0-9]+) seconds`: fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+), (.+) or (.+) every ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(2)), "level": c.n(1), "triggered": true}), mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(3)), "level": c.n(1), "triggered": true}), mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(4)), "level": c.n(1), "triggered": true})}
	}),
	`offering skills triggered this way also affect you`:                                                    []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SkillData", "LIST", Tag{"key": "buffNotPlayer", "value": false})}, Tag{"type": "SkillName", "skillNameList": []any{"Bone Offering", "Flesh Offering", "Spirit Offering"}})},
	`trigger level ([0-9]+) (.+) after spending a total of ([0-9]+) mana`:                                   fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`consumes a void charge to trigger level ([0-9]+) (.+) when you fire arrows`:                            fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`consumes a void charge to trigger level ([0-9]+) (.+) when you fire arrows with a non-triggered skill`: fn(func(c caps) any { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`your hits treat cold resistance as ([0-9]+)% higher than actual value`:                                 fn(func(c caps) any { return []any{mod("ColdPenetration", "BASE", -c.n(1), nil, 0, KeywordFlag.Hit)} }),
	// Conversion
	`increases and reductions to minion damage also affects? you`: []any{flag("MinionDamageAppliesToPlayer")},
	`increases and reductions to minion damage also affects? you at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("MinionDamageAppliesToPlayer"), mod("ImprovedMinionDamageAppliesToPlayer", "MAX", c.n(1))}
	}),
	`increases and reductions to minion damage also affect dominating blow and absolution at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("MinionDamageAppliesToPlayer", Tag{"type": "SkillName", "skillNameList": []any{"Dominating Blow", "Absolution"}, "includeTransfigured": true}), mod("ImprovedMinionDamageAppliesToPlayer", "MAX", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Dominating Blow", "Absolution"}, "includeTransfigured": true})}
	}),
	`increases and reductions to minion attack speed also affects? you`: []any{flag("MinionAttackSpeedAppliesToPlayer")},
	`increases and reductions to minion cast speed also affects? you`:   []any{flag("MinionCastSpeedAppliesToPlayer")},
	`increases and reductions to minion maximum life also apply to you at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("MinionLifeAppliesToPlayer"), mod("ImprovedMinionLifeAppliesToPlayer", "MAX", c.n(1))}
	}),
	`increases and reductions to cast speed apply to attack speed at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("CastSpeedAppliesToAttacks"), mod("ImprovedCastSpeedAppliesToAttacks", "MAX", c.n(1))}
	}),
	`increases and reductions to cast speed apply to attack speed`:                    fn(func(c caps) any { return []any{flag("CastSpeedAppliesToAttacks")} }),
	`increases and reductions to spell damage also apply to attacks`:                  []any{flag("SpellDamageAppliesToAttacks")},
	`increases and reductions to your evasion rating also apply to your spell damage`: []any{flag("EvasionAppliesToSpellDamage")},
	`arcane might`: []any{flag("SpellDamageAppliesToAttacks")},
	`([0-9]+)% arcane might`: fn(func(c caps) any {
		return []any{flag("SpellDamageAppliesToAttacks"), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", c.n(1))}
	}),
	`attacks have ([0-9]+)% arcane might`: fn(func(c caps) any {
		return []any{flag("SpellDamageAppliesToAttacks"), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", c.n(1))}
	}),
	`attacks have ([0-9]+)% arcane might while wielding a wand`: fn(func(c caps) any {
		return []any{flag("SpellDamageAppliesToAttacks", Tag{"type": "Condition", "var": "UsingWand"}), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", c.n(1), Tag{"type": "Condition", "var": "UsingWand"})}
	}),
	`retaliation skills have ([0-9]+)% arcane might`: fn(func(c caps) any {
		return []any{flag("SpellDamageAppliesToAttacks", Tag{"type": "SkillType", "skillType": SkillType.Retaliation}), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Retaliation})}
	}),
	`attacks have arcane might while wielding a wand`: []any{flag("SpellDamageAppliesToAttacks", Tag{"type": "Condition", "var": "UsingWand"})},
	`increases and reductions to spell damage also apply to attacks at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("SpellDamageAppliesToAttacks"), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", c.n(1))}
	}),
	`increases and reductions to spell damage also apply to attack damage with retaliation skills at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("SpellDamageAppliesToAttacks", Tag{"type": "SkillType", "skillType": SkillType.Retaliation}), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Retaliation})}
	}),
	`increases and reductions to spell damage also apply to attacks while wielding a wand`: []any{flag("SpellDamageAppliesToAttacks", Tag{"type": "Condition", "var": "UsingWand"})},
	`increases and reductions to maximum mana also apply to shock effect at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("ManaAppliesToShockEffect"), mod("ImprovedManaAppliesToShockEffect", "MAX", c.n(1))}
	}),
	`increases and reductions to ([a-zA-Z]+) damage also apply to effect of auras from ([a-zA-Z]+) skills at ([0-9]+)% of their value, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{flag(firstToUpper(c.s(1)) + "DamageAppliesTo" + (firstToUpper(c.s(2)) + "AuraEffect")), mod(("Improved"+firstToUpper(c.s(1)))+"DamageAppliesTo"+(firstToUpper(c.s(2))+"AuraEffect"), "BASE", c.n(3)), mod(firstToUpper(c.s(1))+"DamageAppliesTo"+(firstToUpper(c.s(2))+"AuraEffectLimit"), "MAX", c.n(4))}
	}),
	`modifiers to claw damage also apply to unarmed`:                                                          []any{flag("ClawDamageAppliesToUnarmed")},
	`modifiers to claw damage also apply to unarmed attack damage`:                                            []any{flag("ClawDamageAppliesToUnarmed")},
	`modifiers to claw damage also apply to unarmed attack damage with melee skills`:                          []any{flag("ClawDamageAppliesToUnarmed")},
	`modifiers to claw attack speed also apply to unarmed`:                                                    []any{flag("ClawAttackSpeedAppliesToUnarmed")},
	`modifiers to claw attack speed also apply to unarmed attack speed`:                                       []any{flag("ClawAttackSpeedAppliesToUnarmed")},
	`modifiers to claw attack speed also apply to unarmed attack speed with melee skills`:                     []any{flag("ClawAttackSpeedAppliesToUnarmed")},
	`modifiers to claw critical strike chance also apply to unarmed`:                                          []any{flag("ClawCritChanceAppliesToUnarmed")},
	`modifiers to claw critical strike chance also apply to unarmed attack critical strike chance`:            []any{flag("ClawCritChanceAppliesToUnarmed")},
	`modifiers to claw critical strike chance also apply to unarmed critical strike chance with melee skills`: []any{flag("ClawCritChanceAppliesToUnarmed")},
	`increases and reductions to light radius also apply to accuracy`:                                         []any{flag("LightRadiusAppliesToAccuracy")},
	`increases and reductions to light radius also apply to area of effect at 50% of their value`:             []any{flag("LightRadiusAppliesToAreaOfEffect")},
	`increases and reductions to light radius also apply to damage`:                                           []any{flag("LightRadiusAppliesToDamage")},
	`increases and reductions to cast speed also apply to trap throwing speed`:                                []any{flag("CastSpeedAppliesToTrapThrowingSpeed")},
	`increases and reductions to armour also apply to energy shield recharge rate at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{flag("ArmourAppliesToEnergyShieldRecharge"), mod("ImprovedArmourAppliesToEnergyShieldRecharge", "MAX", c.n(1))}
	}),
	`increases and reductions to projectile speed also apply to damage with bows`: []any{flag("ProjectileSpeedAppliesToBowDamage")},
	`modifiers to maximum ([a-zA-Z]+) resistance also apply to maximum ([a-zA-Z]+) and ([a-zA-Z]+) resistances`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(1))+"MaxResConvertTo"+firstToUpper(c.s(2)), "BASE", 100), mod(firstToUpper(c.s(1))+"MaxResConvertTo"+firstToUpper(c.s(3)), "BASE", 100)}
	}),
	`modifiers to ([a-zA-Z]+) resistance also apply to ([a-zA-Z]+) and ([a-zA-Z]+) resistances at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(1))+"ResConvertTo"+firstToUpper(c.s(2)), "BASE", c.n(4)), mod(firstToUpper(c.s(1))+"ResConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(4))}
	}),
	`gain ([0-9]+)% of bow physical damage as extra damage of each element`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsLightning", "BASE", c.n(1), nil, ModFlag.Bow), mod("PhysicalDamageGainAsCold", "BASE", c.n(1), nil, ModFlag.Bow), mod("PhysicalDamageGainAsFire", "BASE", c.n(1), nil, ModFlag.Bow)}
	}),
	`gain ([0-9]+)% of weapon physical damage as extra damage of each element`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsLightning", "BASE", c.n(1), nil, ModFlag.Weapon), mod("PhysicalDamageGainAsCold", "BASE", c.n(1), nil, ModFlag.Weapon), mod("PhysicalDamageGainAsFire", "BASE", c.n(1), nil, ModFlag.Weapon)}
	}),
	`gain ([0-9]+)% of physical damage as extra damage of each element per spirit charge`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsLightning", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "SpiritCharge"}), mod("PhysicalDamageGainAsCold", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "SpiritCharge"}), mod("PhysicalDamageGainAsFire", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "SpiritCharge"})}
	}),
	`gain ([0-9]+)% of physical damage as extra damage of each element if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsLightning", "BASE", c.n(1), Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)}), mod("PhysicalDamageGainAsCold", "BASE", c.n(1), Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)}), mod("PhysicalDamageGainAsFire", "BASE", c.n(1), Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)})}
	}),
	`gain ([0-9]+)% of weapon physical damage as extra damage of an? r?a?n?d?o?m? ?element`:              fn(func(c caps) any { return []any{mod("PhysicalDamageGainAsRandom", "BASE", c.n(1), nil, ModFlag.Weapon)} }),
	`gain ([0-9]+)% of physical damage as extra damage of a random element`:                              fn(func(c caps) any { return []any{mod("PhysicalDamageGainAsRandom", "BASE", c.n(1))} }),
	`([0-9]+)% chance for hits to deal ([0-9]+)% of physical damage as extra damage of a random element`: fn(func(c caps) any { return []any{mod("PhysicalDamageGainAsRandom", "BASE", (c.n(1) * c.n(2) / 100))} }),
	`gain ([0-9]+)% of physical damage as extra damage of a random element while you are ignited`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsRandom", "BASE", c.n(1), Tag{"type": "Condition", "var": "Ignited"})}
	}),
	`([0-9]+)% of physical damage from hits with this weapon is converted to a random element`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageConvertToRandom", "BASE", c.n(1), Tag{"type": "Condition", "var": "{Hand}Attack"})}
	}),
	`([0-9]+)% of physical damage converted to a random element`: fn(func(c caps) any { return []any{mod("PhysicalDamageConvertToRandom", "BASE", c.n(1))} }),
	`nearby enemies convert ([0-9]+)% of their ([a-zA-Z]+) damage to ([a-zA-Z]+)`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1))})}
	}),
	`enemies ignited by you have ([0-9]+)% of ([a-zA-Z]+) damage they deal converted to ([a-zA-Z]+)`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1), Tag{"type": "Condition", "var": "Ignited"})})}
	}),
	`enemies shocked by you have ([0-9]+)% of ([a-zA-Z]+) damage they deal converted to ([a-zA-Z]+)`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1), Tag{"type": "Condition", "var": "Shocked"})})}
	}),
	`enemies poisoned by you have ([0-9]+)% of ([a-zA-Z]+) damage they deal converted to ([a-zA-Z]+)`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1), Tag{"type": "Condition", "var": "Poisoned"})})}
	}),
	`shield crush and spectral shield throw do not gain added physical damage based on armour or evasion on shield`: []any{flag("Condition:ShieldThrowCrushNoArmourEvasion", Tag{"type": "SkillName", "skillNameList": []any{"Spectral Shield Throw", "Shield Crush"}, "includeTransfigured": true})},
	`shield crush and spectral shield throw gains ([0-9]+) to ([0-9]+) added lightning damage per ([0-9]+) energy shield on shield`: fn(func(c caps) any {
		return []any{mod("LightningMin", "BASE", c.s(1), 0, 0, Tag{"type": "Condition", "var": "OffHandAttack"}, Tag{"type": "PerStat", "stat": "EnergyShieldOnWeapon 2", "div": c.s(3)}, Tag{"type": "SkillName", "skillNameList": []any{"Spectral Shield Throw", "Shield Crush"}, "includeTransfigured": true}), mod("LightningMax", "BASE", c.s(2), 0, 0, Tag{"type": "Condition", "var": "OffHandAttack"}, Tag{"type": "PerStat", "stat": "EnergyShieldOnWeapon 2", "div": c.s(3)}, Tag{"type": "SkillName", "skillNameList": []any{"Spectral Shield Throw", "Shield Crush"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% of shield crush and spectral shield throw physical damage converted to lightning damage`: fn(func(c caps) any {
		return []any{mod("SkillPhysicalDamageConvertToLightning", "BASE", c.n(1), 0, 0, Tag{"type": "SkillName", "skillNameList": []any{"Spectral Shield Throw", "Shield Crush"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% of exsanguinate and reap physical damage converted to fire damage`: fn(func(c caps) any {
		return []any{mod("SkillPhysicalDamageConvertToFire", "BASE", c.n(1), 0, 0, Tag{"type": "SkillName", "skillNameList": []any{"Exsanguinate", "Reap"}, "includeTransfigured": true})}
	}),
	`-([0-9]+)% of toxic rain physical damage converted to chaos damage`: fn(func(c caps) any {
		return []any{mod("SkillPhysicalDamageConvertToChaos", "BASE", -c.n(1), 0, 0, Tag{"type": "SkillName", "skillName": "Toxic Rain", "includeTransfigured": true})}
	}),
	`cobra lash and venom gyre have -([0-9]+)% of physical damage converted to chaos damage`: fn(func(c caps) any {
		return []any{mod("SkillPhysicalDamageConvertToChaos", "BASE", -c.n(1), 0, 0, Tag{"type": "SkillName", "skillNameList": []any{"Cobra Lash", "Venom Gyre"}})}
	}),
	`([0-9]+)% of consecrated path and purifying flame fire damage converted to chaos damage`: fn(func(c caps) any {
		return []any{mod("SkillFireDamageConvertToChaos", "BASE", c.n(1), 0, 0, Tag{"type": "SkillName", "skillNameList": []any{"Consecrated Path", "Purifying Flame"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% of manabond and stormbind lightning damage converted to cold damage`: fn(func(c caps) any {
		return []any{mod("SkillLightningDamageConvertToCold", "BASE", c.n(1), 0, 0, Tag{"type": "SkillName", "skillNameList": []any{"Manabond", "Stormbind"}, "includeTransfigured": true})}
	}),
	`exsanguinate debuffs deal fire damage per second instead of physical damage per second`: []any{flag("Condition:ExsanguinateDebuffIsFireDamage", Tag{"type": "SkillName", "skillName": "Exsanguinate", "includeTransfigured": true})},
	`reap debuffs deal fire damage per second instead of physical damage per second`:         []any{flag("Condition:ReapDebuffIsFireDamage", Tag{"type": "SkillName", "skillName": "Reap"})},
	// Crit
	`your critical strike chance is lucky`:                                   []any{flag("CritChanceLucky")},
	`your critical strike chance is lucky while on low life`:                 []any{flag("CritChanceLucky", Tag{"type": "Condition", "var": "LowLife"})},
	`your critical strike chance is lucky while focus?sed`:                   []any{flag("CritChanceLucky", Tag{"type": "Condition", "var": "Focused"})},
	`your critical strikes do not deal extra damage`:                         []any{flag("NoCritMultiplier")},
	`critical strikes do not deal extra damage`:                              []any{flag("NoCritMultiplier")},
	`critical strikes with this weapon do not deal extra damage`:             []any{flag("NoCritMultiplier", Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`minion critical strikes do not deal extra damage`:                       []any{mod("MinionModifier", "LIST", Tag{"mod": flag("NoCritMultiplier")})},
	`lightning damage with non-critical strikes is lucky`:                    []any{flag("LightningNoCritLucky")},
	`your damage with critical strikes is lucky`:                             []any{flag("CritLucky")},
	`spell critical strike chance bifurcates`:                                []any{flag("BifurcateCrit", nil, ModFlag.Spell)},
	`critical strikes deal no damage`:                                        []any{mod("Damage", "MORE", -100, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`critical strike chance is increased by uncapped lightning resistance`:   []any{flag("CritChanceIncreasedByUncappedLightningRes")},
	`critical strike chance is increased by lightning resistance`:            []any{flag("CritChanceIncreasedByLightningRes")},
	`critical strike chance is increased by overcapped lightning resistance`: []any{flag("CritChanceIncreasedByOvercappedLightningRes")},
	`barrage and frenzy have ([0-9]+)% increased critical strike chance per endurance charge`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), Tag{"type": "Multiplier", "var": "EnduranceCharge"}, Tag{"type": "SkillName", "skillNameList": []any{"Barrage", "Frenzy"}, "includeTransfigured": true})}
	}),
	`non-critical strikes deal ([0-9]+)% damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", -100+c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "CriticalStrike", "neg": true})}
	}),
	`non-critical strikes deal no damage`: []any{mod("Damage", "MORE", -100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "CriticalStrike", "neg": true})},
	`non-critical strikes deal ([0-9]+)% less damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", -c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "CriticalStrike", "neg": true})}
	}),
	`spell skills always deal critical strikes on final repeat`:        []any{flag("SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat", nil, ModFlag.Spell)},
	`spell skills cannot deal critical strikes except on final repeat`: []any{flag("SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat", nil, ModFlag.Spell), flag("", Tag{"type": "Condition", "var": "alwaysFinalRepeat"})},
	`critical strikes penetrate ([0-9]+)% of enemy elemental resistances while affected by zealotry`: fn(func(c caps) any {
		return []any{mod("ElementalPenetration", "BASE", c.n(1), Tag{"type": "Condition", "var": "CriticalStrike"}, Tag{"type": "Condition", "var": "AffectedByZealotry"})}
	}),
	`attack critical strikes ignore enemy monster elemental resistances`: []any{flag("IgnoreElementalResistances", Tag{"type": "Condition", "var": "CriticalStrike"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})},
	`treats enemy monster chaos resistance values as inverted`:           []any{mod("HitsInvertChaosResChance", "CHANCE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"})},
	`([+\-][0-9]+)% to critical strike multiplier if you've shattered an enemy recently`: fn(func(c caps) any {
		return []any{mod("CritMultiplier", "BASE", c.n(1), Tag{"type": "Condition", "var": "ShatteredEnemyRecently"})}
	}),
	`([0-9]+)% chance to gain a flask charge when you deal a critical strike`:             fn(func(c caps) any { return []any{mod("FlaskChargeOnCritChance", "BASE", c.n(1))} }),
	`gain a flask charge when you deal a critical strike`:                                 []any{mod("FlaskChargeOnCritChance", "BASE", 100)},
	`gain a flask charge when you deal a critical strike while affected by precision`:     []any{mod("FlaskChargeOnCritChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByPrecision"})},
	`gain a flask charge when you deal a critical strike while at maximum frenzy charges`: []any{mod("FlaskChargeOnCritChance", "BASE", 100, Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"})},
	`on non-channelling attack, set a life flask with greater than [0-9]+% of maximum charges remaining to ([0-9]+)% for each charge removed this way, that attack gains \+([0-9]+)% to damage over time multiplier`: fn(func(c caps) any {
		return []any{mod("DotMultiplier", "BASE", c.n(2), Tag{"type": "PercentStat", "stat": "LifeFlaskCharges", "percent": 100 - c.n(1), "floor": true}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`enemies poisoned by you cannot deal critical strikes`:                        []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("NeverCrit", Tag{"type": "Condition", "var": "Poisoned"})}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:NeverCrit", Tag{"type": "Condition", "var": "Poisoned"})})},
	`marked enemy cannot deal critical strikes`:                                   []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("NeverCrit", Tag{"type": "Condition", "var": "Marked"})}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:NeverCrit", Tag{"type": "Condition", "var": "Marked"})})},
	`marked enemy cannot evade attacks`:                                           []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("CannotEvade", Tag{"type": "Condition", "var": "Marked"})})},
	`hits against you cannot be critical strikes if you've been stunned recently`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("NeverCrit")}, Tag{"type": "Condition", "var": "StunnedRecently"}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:NeverCrit")}, Tag{"type": "Condition", "var": "StunnedRecently"})},
	`nearby enemies cannot deal critical strikes`:                                 []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("NeverCrit")}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:NeverCrit")})},
	`hits against you are always critical strikes`:                                []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("AlwaysCrit")}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:AlwaysCrit")})},
	`your hits are always critical strikes`:                                       []any{mod("CritChance", "OVERRIDE", 100)},
	`all hits are critical strikes while holding a fishing rod`:                   []any{mod("CritChance", "OVERRIDE", 100, Tag{"type": "Condition", "var": "UsingFishing"})},
	`all hits with your next non-channelling attack within ([0-9]+) seconds of taking a critical strike will be critical strikes`: []any{mod("CritChance", "OVERRIDE", 100, Tag{"type": "Condition", "var": "BeenCritRecently"}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Attack})},
	`hits have ([0-9]+)% increased critical strike chance against you`:                                                            fn(func(c caps) any { return []any{mod("EnemyCritChance", "INC", c.n(1))} }),
	`stuns from critical strikes have ([0-9]+)% increased duration`:                                                               fn(func(c caps) any { return []any{mod("EnemyStunDurationOnCrit", "INC", c.n(1))} }),
	// Generic Ailments
	`enemies take ([0-9]+)% increased damage for each type of ailment you have inflicted on them`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Scorched"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Brittle"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Sapped"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"})}
	}),
	`([0-9]+)% chance to deal double damage against enemies for each type of ailment you have inflicted on them`: fn(func(c caps) any {
		return []any{mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Scorched"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Brittle"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Sapped"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}), mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"})}
	}),
	// Elemental Ailments
	`([0-9]+)% increased elemental damage with hits and ailments for each type of elemental ailment on enemy`: fn(func(c caps) any {
		return []any{mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"}), mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}), mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"}), mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}), mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Scorched"}), mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Brittle"}), mod("ElementalDamage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Sapped"})}
	}),
	`your shocks can increase damage taken by up to a maximum of ([0-9]+)%`: fn(func(c caps) any { return []any{mod("ShockMax", "OVERRIDE", c.n(1))} }),
	`\+([0-9]+)% to maximum effect of shock`:                                fn(func(c caps) any { return []any{mod("ShockMax", "BASE", c.n(1))} }),
	`your ([0-9a-zA-Z]+) damage can ([0-9a-zA-Z]+)`:                         fn(func(c caps) any { return []any{flag(firstToUpper(c.s(1)) + "Can" + firstToUpper(c.s(2)))} }),
	`your ([0-9a-zA-Z]+) damage cannot ([0-9a-zA-Z]+)`:                      fn(func(c caps) any { return []any{flag(firstToUpper(c.s(1)) + "Cannot" + firstToUpper(c.s(2)))} }),
	`your elemental damage can shock`:                                       []any{flag("ColdCanShock"), flag("FireCanShock")},
	`all y?o?u?r? ?damage can freeze`:                                       []any{flag("PhysicalCanFreeze"), flag("LightningCanFreeze"), flag("FireCanFreeze"), flag("ChaosCanFreeze")},
	`all damage with maces and sceptres inflicts chill`:                     []any{flag("PhysicalCanChill", Tag{"type": "Condition", "var": "UsingMace"}), flag("LightningCanChill", Tag{"type": "Condition", "var": "UsingMace"}), flag("FireCanChill", Tag{"type": "Condition", "var": "UsingMace"}), flag("ChaosCanChill", Tag{"type": "Condition", "var": "UsingMace"})},
	`all damage from lightning strike and frost blades hits can ignite`:     []any{flag("PhysicalCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Strike", "Frost Blades"}, "includeTransfigured": true}), flag("ColdCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Strike", "Frost Blades"}, "includeTransfigured": true}), flag("LightningCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Strike", "Frost Blades"}, "includeTransfigured": true}), flag("ChaosCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Strike", "Frost Blades"}, "includeTransfigured": true})},
	`all damage from lightning arrow and ice shot hits can ignite`:          []any{flag("PhysicalCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Arrow", "Ice Shot"}, "includeTransfigured": true}), flag("ColdCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Arrow", "Ice Shot"}, "includeTransfigured": true}), flag("LightningCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Arrow", "Ice Shot"}, "includeTransfigured": true}), flag("ChaosCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Lightning Arrow", "Ice Shot"}, "includeTransfigured": true})},
	`all damage from shock nova and storm call hits can ignite`:             []any{flag("PhysicalCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Shock Nova", "Storm Call"}, "includeTransfigured": true}), flag("ColdCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Shock Nova", "Storm Call"}, "includeTransfigured": true}), flag("LightningCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Shock Nova", "Storm Call"}, "includeTransfigured": true}), flag("ChaosCanIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Shock Nova", "Storm Call"}, "includeTransfigured": true})},
	`your fire damage can shock but not ignite`:                             []any{flag("FireCanShock"), flag("FireCannotIgnite")},
	`your cold damage can ignite but not freeze or chill`:                   []any{flag("ColdCanIgnite"), flag("ColdCannotFreeze"), flag("ColdCannotChill")},
	`your lightning damage can freeze but not shock`:                        []any{flag("LightningCanFreeze"), flag("LightningCannotShock")},
	`your physical damage can ignite during effect`:                         []any{flag("PhysicalCanIgnite")},
	`chaos damage can ignite, chill and shock`:                              []any{flag("ChaosCanIgnite"), flag("ChaosCanChill"), flag("ChaosCanShock")},
	`you always ignite while burning`:                                       []any{mod("EnemyIgniteChance", "BASE", 100, Tag{"type": "Condition", "var": "Burning"})},
	`critical strikes do not a?l?w?a?y?s?i?n?h?e?r?e?n?t?l?y? freeze`:       []any{flag("CritsDontAlwaysFreeze")},
	`cannot inflict elemental ailments`:                                     []any{flag("CannotIgnite"), flag("CannotChill"), flag("CannotFreeze"), flag("CannotShock"), flag("CannotScorch"), flag("CannotBrittle"), flag("CannotSap")},
	`non-critical strikes cannot inflict ailments`:                          []any{flag("AilmentsOnlyFromCrit")},
	`flameblast and incinerate cannot inflict elemental ailments`:           []any{flag("CannotIgnite", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true}), flag("CannotChill", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true}), flag("CannotFreeze", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true}), flag("CannotShock", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true}), flag("CannotScorch", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true}), flag("CannotBrittle", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true}), flag("CannotSap", Tag{"type": "SkillName", "skillNameList": []any{"Flameblast", "Incinerate"}, "includeTransfigured": true})},
	`you can inflict up to ([0-9]+) ignites on an enemy`:                    fn(func(c caps) any { return []any{flag("IgniteCanStack"), mod("IgniteStacks", "OVERRIDE", c.n(1))} }),
	`you can inflict an additional ignite on [ea][an]c?h? enemy`:            []any{flag("IgniteCanStack"), mod("IgniteStacks", "BASE", 1)},
	`enemies chilled by you take ([0-9]+)% increased burning damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireDamageTakenOverTime", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"})}
	}),
	`damaging ailments deal damage ([0-9]+)% faster`: fn(func(c caps) any {
		return []any{mod("IgniteBurnFaster", "INC", c.n(1)), mod("BleedFaster", "INC", c.n(1)), mod("PoisonFaster", "INC", c.n(1))}
	}),
	`damaging ailments you inflict deal damage ([0-9]+)% faster while affected by malevolence`: fn(func(c caps) any {
		return []any{mod("IgniteBurnFaster", "INC", c.n(1), Tag{"type": "Condition", "var": "AffectedByMalevolence"}), mod("BleedFaster", "INC", c.n(1), Tag{"type": "Condition", "var": "AffectedByMalevolence"}), mod("PoisonFaster", "INC", c.n(1), Tag{"type": "Condition", "var": "AffectedByMalevolence"})}
	}),
	`([0-9]+)% increased damage with damaging ailments you inflict while you are affected by the same ailment`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Bleed, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}, Tag{"type": "Condition", "var": "Bleeding"}), mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Ignite, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"}, Tag{"type": "Condition", "var": "Ignited"}), mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Poison, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"}, Tag{"type": "Condition", "var": "Poisoned"})}
	}),
	`ignited enemies burn ([0-9]+)% faster`: fn(func(c caps) any { return []any{mod("IgniteBurnFaster", "INC", c.n(1))} }),
	// Overrides the base duration of a damaging ailment (the only ailments with a fixed base duration
	// consumed by the calcs; freeze duration is derived from damage and chill/shock durations are not modelled)
	`ignited enemies burn ([0-9]+)% slower`:                         fn(func(c caps) any { return []any{mod("IgniteBurnSlower", "INC", c.n(1))} }),
	`enemies ignited by an attack burn ([0-9]+)% faster`:            fn(func(c caps) any { return []any{mod("IgniteBurnFaster", "INC", c.n(1), nil, ModFlag.Attack)} }),
	`ignites you inflict with attacks deal damage ([0-9]+)% faster`: fn(func(c caps) any { return []any{mod("IgniteBurnFaster", "INC", c.n(1), nil, ModFlag.Attack)} }),
	`ignites you inflict deal damage ([0-9]+)% faster`:              fn(func(c caps) any { return []any{mod("IgniteBurnFaster", "INC", c.n(1))} }),
	`([0-9]+)% chance for ignites inflicted with lightning strike or frost blades to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Ignite, Tag{"type": "SkillName", "skillNameList": []any{"Lightning Strike", "Frost Blades"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% chance for ignites inflicted with lightning arrow or ice shot to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Ignite, Tag{"type": "SkillName", "skillNameList": []any{"Lightning Arrow", "Ice Shot"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% chance for ignites inflicted with shock nova or storm call to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Ignite, Tag{"type": "SkillName", "skillNameList": []any{"Shock Nova", "Storm Call"}, "includeTransfigured": true})}
	}),
	`enemies ignited by you during f?l?a?s?k? ?effect take ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"})}
	}),
	`enemies ignited by you take chaos damage instead of fire damage from ignite`: []any{flag("IgniteToChaos")},
	`enemies chilled by your hits are shocked`:                                    []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "ChilledByYourHits"}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Shocked", Tag{"type": "Condition", "var": "ChilledByYourHits"})})},
	`cannot inflict ignite`:                                                       []any{flag("CannotIgnite")},
	`cannot inflict freeze or chill`:                                              []any{flag("CannotFreeze"), flag("CannotChill")},
	`cannot inflict shock`:                                                        []any{flag("CannotShock")},
	`cannot ignite, chill, freeze or shock`:                                       []any{flag("CannotIgnite"), flag("CannotChill"), flag("CannotFreeze"), flag("CannotShock")},
	`shock enemies as though dealing ([0-9]+)% more damage`:                       fn(func(c caps) any { return []any{mod("ShockAsThoughDealing", "MORE", c.n(1))} }),
	`chill enemies as though dealing ([0-9]+)% more damage`:                       fn(func(c caps) any { return []any{mod("ChillAsThoughDealing", "MORE", c.n(1))} }),
	`inflict non-damaging ailments as though dealing ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("ShockAsThoughDealing", "MORE", c.n(1)), mod("ChillAsThoughDealing", "MORE", c.n(1)), mod("FreezeAsThoughDealing", "MORE", c.n(1)), mod("ScorchAsThoughDealing", "MORE", c.n(1)), mod("BrittleAsThoughDealing", "MORE", c.n(1)), mod("SapAsThoughDealing", "MORE", c.n(1))}
	}),
	`non-damaging elemental ailments you inflict have ([0-9]+)% more effect`: fn(func(c caps) any {
		return []any{mod("EnemyShockEffect", "MORE", c.n(1)), mod("EnemyChillEffect", "MORE", c.n(1)), mod("EnemyFreezeEffect", "MORE", c.n(1)), mod("EnemyScorchEffect", "MORE", c.n(1)), mod("EnemyBrittleEffect", "MORE", c.n(1)), mod("EnemySapEffect", "MORE", c.n(1))}
	}),
	`immun[ei]t?y? to elemental ailments while on consecrated ground if you have at least ([0-9]+) devotion`: fn(func(c caps) any {
		return []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "OnConsecratedGround"}, Tag{"type": "StatThreshold", "stat": "Devotion", "threshold": c.n(1)})}
	}),
	`freeze enemies as though dealing ([0-9]+)% more damage`: fn(func(c caps) any { return []any{mod("FreezeAsThoughDealing", "MORE", c.n(1))} }),
	`freeze chilled enemies as though dealing ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("FreezeAsThoughDealing", "MORE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"})}
	}),
	`manabond and stormbind freeze enemies as though dealing ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("FreezeAsThoughDealing", "MORE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Manabond", "Stormbind"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% chance to inflict brittle on enemies when you block their damage`:                                    fn(func(c caps) any { return []any{mod("EnemyBrittleChance", "BASE", c.n(1))} }),
	`([0-9]+)% chance to inflict sap on enemies when you block their damage`:                                        fn(func(c caps) any { return []any{mod("EnemySapChance", "BASE", c.n(1))} }),
	`([0-9]+)% chance to inflict scorch on enemies when you block their damage`:                                     fn(func(c caps) any { return []any{mod("EnemyScorchChance", "BASE", c.n(1))} }),
	`scorch enemies in close range when you block`:                                                                  []any{mod("EnemyScorchChance", "BASE", 100)},
	`([0-9]+)% chance to shock attackers for ([0-9]+) seconds on block`:                                             []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"])},
	`shock attackers for ([0-9]+) seconds on block`:                                                                 []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "Condition", "var": "BlockedRecently"}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Shocked")}, Tag{"type": "Condition", "var": "BlockedRecently"})},
	`shock nearby enemies for ([0-9]+) seconds when you focus`:                                                      []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "Condition", "var": "Focused"}), mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Shocked")}, Tag{"type": "Condition", "var": "Focused"})},
	`shock yourself for ([0-9]+) seconds when you focus`:                                                            []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "Condition", "var": "Focused"}), flag("Condition:Shocked", Tag{"type": "Condition", "var": "Focused"})},
	`drops shocked ground while moving, lasting ([0-9]+) seconds`:                                                   []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnShockedGround"})},
	`drops scorched ground while moving, lasting ([0-9]+) seconds`:                                                  []any{mod("ScorchBase", "BASE", nonDamagingAilmentDefault["Scorch"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnScorchedGround"})},
	`drops brittle ground while moving, lasting ([0-9]+) seconds`:                                                   []any{mod("BrittleBase", "BASE", nonDamagingAilmentDefault["Brittle"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnBrittleGround"})},
	`drops sapped ground while moving, lasting ([0-9]+) seconds`:                                                    []any{mod("SapBase", "BASE", nonDamagingAilmentDefault["Sap"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnSappedGround"})},
	`while a unique enemy is in your presence, drops shocked ground while moving, lasting ([0-9]+) seconds`:         []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnShockedGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})},
	`while a unique enemy is in your presence, drops scorched ground while moving, lasting ([0-9]+) seconds`:        []any{mod("ScorchBase", "BASE", nonDamagingAilmentDefault["Scorch"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnScorchedGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})},
	`while a unique enemy is in your presence, drops brittle ground while moving, lasting ([0-9]+) seconds`:         []any{mod("BrittleBase", "BASE", nonDamagingAilmentDefault["Brittle"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnBrittleGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})},
	`while a unique enemy is in your presence, drops sapped ground while moving, lasting ([0-9]+) seconds`:          []any{mod("SapBase", "BASE", nonDamagingAilmentDefault["Sap"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnSappedGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})},
	`while a pinnacle atlas boss is in your presence, drops shocked ground while moving, lasting ([0-9]+) seconds`:  []any{mod("ShockBase", "BASE", nonDamagingAilmentDefault["Shock"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnShockedGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"})},
	`while a pinnacle atlas boss is in your presence, drops scorched ground while moving, lasting ([0-9]+) seconds`: []any{mod("ScorchBase", "BASE", nonDamagingAilmentDefault["Scorch"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnScorchedGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"})},
	`while a pinnacle atlas boss is in your presence, drops brittle ground while moving, lasting ([0-9]+) seconds`:  []any{mod("BrittleBase", "BASE", nonDamagingAilmentDefault["Brittle"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnBrittleGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"})},
	`while a pinnacle atlas boss is in your presence, drops sapped ground while moving, lasting ([0-9]+) seconds`:   []any{mod("SapBase", "BASE", nonDamagingAilmentDefault["Sap"], Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnSappedGround"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"})},
	`\+([0-9]+)% chance to ignite, freeze, shock, and poison cursed enemies`: fn(func(c caps) any {
		return []any{mod("EnemyIgniteChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}), mod("EnemyFreezeChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}), mod("EnemyShockChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}), mod("PoisonChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`you have scorching conflux, brittle conflux and sapping conflux while your two highest attributes are equal`: []any{mod("EnemyScorchChance", "BASE", 100, Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), mod("EnemyBrittleChance", "BASE", 100, Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), mod("EnemySapChance", "BASE", 100, Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("PhysicalCanScorch", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("LightningCanScorch", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("ColdCanScorch", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("ChaosCanScorch", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("PhysicalCanBrittle", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("LightningCanBrittle", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("FireCanBrittle", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("ChaosCanBrittle", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("PhysicalCanSap", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("ColdCanSap", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("FireCanSap", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"}), flag("ChaosCanSap", Tag{"type": "Condition", "var": "TwoHighestAttributesEqual"})},
	`all damage from cold snap and creeping frost can sap`:                                                        []any{flag("PhysicalCanSap", Tag{"type": "SkillName", "skillNameList": []any{"Cold Snap", "Creeping Frost"}, "includeTransfigured": true}), flag("ColdCanSap", Tag{"type": "SkillName", "skillNameList": []any{"Cold Snap", "Creeping Frost"}, "includeTransfigured": true}), flag("FireCanSap", Tag{"type": "SkillName", "skillNameList": []any{"Cold Snap", "Creeping Frost"}, "includeTransfigured": true}), flag("ChaosCanSap", Tag{"type": "SkillName", "skillNameList": []any{"Cold Snap", "Creeping Frost"}, "includeTransfigured": true})},
	`always inflict scorch, brittle and sapped with elemental hit and wild strike hits`:                           []any{mod("EnemyScorchChance", "BASE", 100, Tag{"type": "SkillName", "skillNameList": []any{"Elemental Hit", "Wild Strike"}, "includeTransfigured": true}), mod("EnemyBrittleChance", "BASE", 100, Tag{"type": "SkillName", "skillNameList": []any{"Elemental Hit", "Wild Strike"}, "includeTransfigured": true}), mod("EnemySapChance", "BASE", 100, Tag{"type": "SkillName", "skillNameList": []any{"Elemental Hit", "Wild Strike"}, "includeTransfigured": true})},
	`hits with prismatic skills always ?i?n?f?l?i?c?t? ([0-9a-zA-Z]+)`: fn(func(c caps) any {
		return []any{mod("Enemy"+firstToUpper(c.s(1))+"Chance", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "SkillType", "skillType": SkillType.RandomElement})}
	}),
	`critical strikes do not [ia][np][fp]l[iy]c?t? non-damaging ailments`: []any{flag("CritsDontAlwaysChill"), flag("CritsDontAlwaysFreeze"), flag("CritsDontAlwaysShock")},
	`critical strikes do not inherently ignite`:                           []any{flag("CritsDontAlwaysIgnite")},
	`always scorch while affected by anger`:                               []any{mod("EnemyScorchChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByAnger"})},
	`always inflict brittle while affected by hatred`:                     []any{mod("EnemyBrittleChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByHatred"})},
	`always sap while affected by wrath`:                                  []any{mod("EnemySapChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByWrath"})},
	`([0-9]+)% chance to sap enemies in chilling areas`: fn(func(c caps) any {
		return []any{mod("EnemySapChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "InChillingArea"})}
	}),
	`([0-9]+)% chance for cold snap and creeping frost to sap enemies in chilling areas`: fn(func(c caps) any {
		return []any{mod("EnemySapChance", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Cold Snap", "Creeping Frost"}, "includeTransfigured": true}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "InChillingArea"})}
	}),
	`drops burning ground while moving, dealing ([0-9]+) fire damage per second for [0-9]+ seconds`: fn(func(c caps) any { return []any{mod("DropsBurningGround", "BASE", c.n(1))} }),
	`take ([0-9]+) fire damage per second while flame-touched`: fn(func(c caps) any {
		return []any{mod("FireDegen", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByApproachingFlames"})}
	}),
	`gain adrenaline when you become flame-touched`:                       []any{flag("Condition:Adrenaline", Tag{"type": "Condition", "var": "AffectedByApproachingFlames"})},
	`lose adrenaline when you cease to be flame-touched`:                  d(),
	`modifiers to ignite duration on you apply to all elemental ailments`: []any{flag("IgniteDurationAppliesToElementalAilments")},
	`([0-9]+)% increased duration of ailments of types you haven't inflicted recently`: fn(func(c caps) any {
		return []any{mod("EnemyFreezeDuration", "INC", c.n(1), Tag{"type": "Condition", "var": "FrozenEnemyRecently", "neg": true}), mod("EnemyChillDuration", "INC", c.n(1), Tag{"type": "Condition", "var": "FrozenEnemyRecently", "neg": true}), mod("EnemyIgniteDuration", "INC", c.n(1), Tag{"type": "Condition", "var": "IgnitedEnemyRecently", "neg": true}), mod("EnemyShockDuration", "INC", c.n(1), Tag{"type": "Condition", "var": "ShockedEnemyRecently", "neg": true}), mod("EnemyBleedDuration", "INC", c.n(1), Tag{"type": "Condition", "var": "CausedBleedingRecently", "neg": true}), mod("EnemyPoisonDuration", "INC", c.n(1), Tag{"type": "Condition", "var": "PoisonedEnemyRecently", "neg": true})}
	}),
	`chance to avoid being shocked applies to all elemental ailments`:            []any{flag("ShockAvoidAppliesToElementalAilments")},
	`modifiers to chance to avoid being shocked apply to all elemental ailments`: []any{flag("ShockAvoidAppliesToElementalAilments")},
	`enemies permanently take ([0-9]+)% increased damage for each second they've ever been frozen by you, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "Condition", "var": "FrozenByYou"}, Tag{"type": "Multiplier", "var": "FrozenByYouSeconds", "limit": c.n(2) / c.n(1)})})}
	}),
	`enemies permanently take ([0-9]+)% increased damage for each second they've ever been chilled by you, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "Condition", "var": "ChilledByYou"}, Tag{"type": "Multiplier", "var": "ChilledByYouSeconds", "limit": c.n(2) / c.n(1)})})}
	}),
	`modifiers to chance to suppress spell damage also apply to chance to avoid elemental ailments at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionAppliesToAilmentAvoidancePercent", "BASE", c.n(1)), flag("SpellSuppressionAppliesToAilmentAvoidance")}
	}),
	`modifiers to chance to suppress spell damage also apply to chance to defend with ([0-9]+)% of armour at ([0-9]+)% of their value`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour", "MAX", c.n(1)), mod("SpellSuppressionAppliesToChanceToDefendWithArmourPercent", "MAX", c.n(2)), flag("SpellSuppressionAppliesToChanceToDefendWithArmour")}
	}),
	`enemies chilled by your hits have damage taken increased by chill effect`:                []any{flag("ChillEffectIncDamageTaken")},
	`enemies chilled by your hits have cold damage taken increased by chill effect`:           []any{flag("ChillEffectIncColdDamageTaken")},
	`enemies in your chilling areas have cold damage taken increased by chill effect`:         []any{flag("ChillingAreaIncColdDamageTaken", Tag{"type": "ActorCondition", "actor": "enemy", "var": "InChillingArea"})},
	`left ring slot: your chilling skitterbot's aura applies socketed h?e?x? ?curse instead`:  []any{flag("SkitterbotsCannotChill", Tag{"type": "SlotNumber", "num": 1})},
	`right ring slot: your shocking skitterbot's aura applies socketed h?e?x? ?curse instead`: []any{flag("SkitterbotsCannotShock", Tag{"type": "SlotNumber", "num": 2})},
	`summon skitterbots also summons a scorching skitterbot`:                                  []any{flag("ScorchingSkitterbot")},
	`summoned skitterbots' auras affect you as well as enemies`:                               []any{flag("SkitterbotAffectPlayer")},
	`([0-9]+)% increased effect of non-damaging ailments inflicted by summoned skitterbots`:   fn(func(c caps) any { return []any{mod("SkitterbotAilmentEffect", "INC", c.n(1))} }),
	// Bleed
	`melee attacks cause bleeding`:                       []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Melee)},
	`attacks cause bleeding when hitting cursed enemies`: []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Attack, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})},
	`melee critical strikes cause bleeding`:              []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Melee, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`causes bleeding on melee critical strike`:           []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Melee, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`melee critical strikes have ([0-9]+)% chance to cause bleeding`: fn(func(c caps) any {
		return []any{mod("BleedChance", "BASE", c.n(1), nil, ModFlag.Melee, Tag{"type": "Condition", "var": "CriticalStrike"})}
	}),
	`attacks always inflict bleeding while you have cat's stealth`:        []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Attack, Tag{"type": "Condition", "var": "AffectedByCat'sStealth"})},
	`you have crimson dance while you have cat's stealth`:                 []any{mod("Keystone", "LIST", "Crimson Dance", Tag{"type": "Condition", "var": "AffectedByCat'sStealth"})},
	`you have crimson dance if you have dealt a critical strike recently`: []any{mod("Keystone", "LIST", "Crimson Dance", Tag{"type": "Condition", "var": "CritRecently"})},
	`bleeding you inflict deals damage ([0-9]+)% faster`:                  fn(func(c caps) any { return []any{mod("BleedFaster", "INC", c.n(1))} }),
	`bleeding you inflict on non-bleeding enemies deals ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Bleed, Tag{"type": "Condition", "var": "SingleBleed"})}
	}),
	`([0-9]+)% chance for bleeding inflicted with this weapon to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Bleed, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`([0-9]+)% chance for bleeding inflicted with cobra lash or venom gyre to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Bleed, Tag{"type": "SkillName", "skillNameList": []any{"Cobra Lash", "Venom Gyre"}})}
	}),
	`bleeding you inflict deals damage ([0-9]+)% faster per frenzy charge`: fn(func(c caps) any {
		return []any{mod("BleedFaster", "INC", c.n(1), Tag{"type": "Multiplier", "var": "FrenzyCharge"})}
	}),
	`rain of arrows and toxic rain deal ([0-9]+)% more damage with bleeding`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(1), nil, 0, KeywordFlag.Bleed, Tag{"type": "SkillName", "skillNameList": []any{"Rain of Arrows", "Toxic Rain"}, "includeTransfigured": true})}
	}),
	// Impale and Bleed
	`([0-9]+)% increased effect of impales inflicted by hits that also inflict bleeding`: fn(func(c caps) any { return []any{mod("ImpaleEffectOnBleed", "INC", c.n(1), nil, 0, KeywordFlag.Hit)} }),
	`([0-9]+)% chance for blade vortex and blade blast to impale enemies on hit`: fn(func(c caps) any {
		return []any{mod("ImpaleChance", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Blade Vortex", "Blade Blast"}, "includeTransfigured": true})}
	}),
	`critical strikes with spells inflict impale`:                                                      []any{mod("ImpaleChance", "BASE", 100, nil, ModFlag.Spell, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`([0-9]+)% chance on hitting an enemy for all impales on that enemy to last for an additional hit`: fn(func(c caps) any { return []any{mod("ImpaleAdditionalDurationChance", "BASE", c.n(1))} }),
	`projectiles gain impale effect as they travel farther, causing impales they inflict to have up to ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("ImpaleEffect", "INC", c.n(1), nil, ModFlag.Projectile, Tag{"type": "DistanceRamp", "ramp": []any{[]any{35, 0}, []any{70, 1}}})}
	}),
	// Poison and Bleed
	`([0-9]+)% increased damage with bleeding inflicted on poisoned enemies`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Bleed, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"})}
	}),
	// Poison
	`y?o?u?r? ?fire damage can poison`:                                      []any{flag("FireCanPoison")},
	`y?o?u?r? ?cold damage can poison`:                                      []any{flag("ColdCanPoison")},
	`y?o?u?r? ?lightning damage can poison`:                                 []any{flag("LightningCanPoison")},
	`all damage from hits can poison`:                                       []any{flag("FireCanPoison"), flag("ColdCanPoison"), flag("LightningCanPoison")},
	`all damage can poison`:                                                 []any{flag("FireCanPoison"), flag("ColdCanPoison"), flag("LightningCanPoison")},
	`all damage with triggered spells can poison`:                           []any{flag("FireCanPoison", nil, 0, KeywordFlag.Spell, Tag{"type": "SkillType", "skillType": SkillType.Triggered}), flag("ColdCanPoison", nil, 0, KeywordFlag.Spell, Tag{"type": "SkillType", "skillType": SkillType.Triggered}), flag("LightningCanPoison", nil, 0, KeywordFlag.Spell, Tag{"type": "SkillType", "skillType": SkillType.Triggered})},
	`all damage from hits with this weapon can poison`:                      []any{flag("FireCanPoison", Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), flag("ColdCanPoison", Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), flag("LightningCanPoison", Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})},
	`all damage inflicts poison while affected by glorious madness`:         []any{mod("PoisonChance", "BASE", 100, Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("FireCanPoison", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("ColdCanPoison", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}), flag("LightningCanPoison", Tag{"type": "Condition", "var": "AffectedByGloriousMadness"})},
	`all damage from blast rain and artillery ballista hits can poison`:     []any{flag("FireCanPoison", Tag{"type": "SkillName", "skillNameList": []any{"Blast Rain", "Artillery Ballista"}}), flag("ColdCanPoison", Tag{"type": "SkillName", "skillNameList": []any{"Blast Rain", "Artillery Ballista"}}), flag("LightningCanPoison", Tag{"type": "SkillName", "skillNameList": []any{"Blast Rain", "Artillery Ballista"}})},
	`all damage from hits with freezing pulse and eye of winter can poison`: []any{flag("FireCanPoison", Tag{"type": "SkillName", "skillNameList": []any{"Freezing Pulse", "Eye of Winter"}, "includeTransfigured": true}), flag("ColdCanPoison", Tag{"type": "SkillName", "skillNameList": []any{"Freezing Pulse", "Eye of Winter"}, "includeTransfigured": true}), flag("LightningCanPoison", Tag{"type": "SkillName", "skillNameList": []any{"Freezing Pulse", "Eye of Winter"}, "includeTransfigured": true})},
	`your chaos damage poisons enemies`:                                     []any{mod("ChaosPoisonChance", "BASE", 100)},
	`your chaos damage has ([0-9]+)% chance to poison enemies`:              fn(func(c caps) any { return []any{mod("ChaosPoisonChance", "BASE", c.n(1))} }),
	`melee attacks poison on hit`:                                           []any{mod("PoisonChance", "BASE", 100, nil, ModFlag.Melee)},
	`triggered spells poison on hit`:                                        []any{mod("PoisonChance", "BASE", 100, nil, 0, KeywordFlag.Spell, Tag{"type": "SkillType", "skillType": SkillType.Triggered})},
	`melee critical strikes have ([0-9]+)% chance to poison the enemy`: fn(func(c caps) any {
		return []any{mod("PoisonChance", "BASE", c.n(1), nil, ModFlag.Melee, Tag{"type": "Condition", "var": "CriticalStrike"})}
	}),
	`critical strikes with daggers have a ([0-9]+)% chance to poison the enemy`: fn(func(c caps) any {
		return []any{mod("PoisonChance", "BASE", c.n(1), nil, ModFlag.Dagger, Tag{"type": "Condition", "var": "CriticalStrike"})}
	}),
	`critical strikes with daggers poison the enemy`:                 []any{mod("PoisonChance", "BASE", 100, nil, ModFlag.Dagger, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`poison cursed enemies on hit`:                                   []any{mod("PoisonChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})},
	`always poison on hit against cursed enemies`:                    []any{mod("PoisonChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})},
	`wh[ie][ln]e? at maximum frenzy charges, attacks poison enemies`: []any{mod("PoisonChance", "BASE", 100, nil, ModFlag.Attack, Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"})},
	`traps and mines have a ([0-9]+)% chance to poison on hit`: fn(func(c caps) any {
		return []any{mod("PoisonChance", "BASE", c.n(1), nil, 0, KeywordFlag.Trap|KeywordFlag.Mine)}
	}),
	`poisons you inflict deal damage ([0-9]+)% faster`: fn(func(c caps) any { return []any{mod("PoisonFaster", "INC", c.n(1))} }),
	`([0-9]+)% chance for poisons inflicted with this weapon to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Poison, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`([0-9]+)% chance for poisons inflicted with blast rain or artillery balls?i?s?ta to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Poison, Tag{"type": "SkillName", "skillNameList": []any{"Blast Rain", "Artillery Ballista"}})}
	}),
	`([0-9]+)% chance for poisons inflicted with freezing pulse and eye of winter to deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", c.n(2)*c.n(1)/100, nil, 0, KeywordFlag.Poison, Tag{"type": "SkillName", "skillNameList": []any{"Freezing Pulse", "Eye of Winter"}, "includeTransfigured": true})}
	}),

	`poisons you inflict on non-poisoned enemies deal ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Poison, Tag{"type": "Condition", "var": "NonPoisonedOnly"})}
	}),
	`poisons inflicted by sunder or ground slam on non-poisoned enemies deal ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Poison, Tag{"type": "Condition", "var": "NonPoisonedOnly"}, Tag{"type": "SkillName", "skillNameList": []any{"Sunder", "Ground Slam"}, "includeTransfigured": true})}
	}),
	`poisons on you expire ([0-9]+)% slower`:                                                      fn(func(c caps) any { return []any{mod("SelfPoisonDebuffExpirationRate", "BASE", -c.n(1))} }),
	`([0-9]+)% chance to inflict an additional poison on the same target when you inflict poison`: fn(func(c caps) any { return []any{mod("AdditionalPoisonChance", "BASE", c.n(1))} }),
	`inflict ([0-9]+) additional poisons? on the same target when you inflict poisons? with this weapon`: fn(func(c caps) any {
		return []any{mod("AdditionalPoisonStacks", "BASE", c.n(1), Tag{"type": "Condition", "var": "{Hand}Attack"})}
	}),
	`cannot poison enemies with at least ([0-9]+) poisons? on them`:                         fn(func(c caps) any { return []any{mod("PoisonStackLimit", "MIN", c.n(1))} }),
	`cannot inflict multiple poisons in the same hit`:                                       []any{flag("CannotMultiplePoison")},
	`wither on hit with this weapon against enemies with at least ([0-9]+) poisons on them`: []any{flag("Condition:CanWither")},
	// Suppression
	`y?o?u?r? ?chance to suppress spell damage is lucky`:   []any{flag("SpellSuppressionChanceIsLucky")},
	`y?o?u?r? ?chance to suppress spell damage is unlucky`: []any{flag("SpellSuppressionChanceIsUnlucky")},
	`prevent \+([0-9]+)% of suppressed spell damage`:       fn(func(c caps) any { return []any{mod("SpellSuppressionEffect", "BASE", c.n(1))} }),
	`prevent \+([0-9]+)% of suppressed spell damage per hit suppressed recently`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionEffect", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "HitsSuppressedRecently"})}
	}),
	`prevent \+([0-9]+)% of suppressed spell damage if you have not suppressed spell damage recently`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionEffect", "BASE", c.n(1), Tag{"type": "Condition", "var": "SuppressedRecently", "neg": true})}
	}),
	`inflict fire, cold and lightning exposure on enemies when you suppress their spell damage`: []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "SuppressedRecently"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "SuppressedRecently"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "SuppressedRecently"})},
	`critical strike chance is increased by chance to suppress spell damage`:                    []any{flag("CritChanceIncreasedBySpellSuppressChance")},
	`you take ([0-9]+)% reduced extra damage from suppressed critical strikes`:                  fn(func(c caps) any { return []any{mod("ReduceSuppressedCritExtraDamage", "BASE", c.n(1))} }),
	`\+([0-9]+)% chance to suppress spell damage if your e?q?u?i?p?p?e?d? ?boots, helmet and gloves have evasion`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionChance", "BASE", c.n(1), Tag{"type": "StatThreshold", "stat": "EvasionOnBoots", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "EvasionOnHelmet", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "EvasionOnGloves", "threshold": 1})}
	}),
	`evasion rating is doubled against projectile attacks`: []any{mod("ProjectileEvasion", "MORE", 100)},
	`evasion rating is doubled against melee attacks`:      []any{mod("MeleeEvasion", "MORE", 100)},
	`\+([0-9]+)% chance to suppress spell damage for each dagger you're wielding`: fn(func(c caps) any {
		return []any{mod("SpellSuppressionChance", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsingDagger"}), mod("SpellSuppressionChance", "BASE", c.n(1), Tag{"type": "Condition", "var": "DualWieldingDaggers"})}
	}),
	// Buffs/debuffs
	`phasing`:                              []any{flag("Condition:Phasing")},
	`onslaught`:                            []any{flag("Condition:Onslaught")},
	`rampage`:                              []any{flag("Condition:Rampage")},
	`soul eater`:                           []any{flag("Condition:CanHaveSoulEater")},
	`unholy might`:                         []any{flag("Condition:UnholyMight"), flag("Condition:CanWither")},
	`chaotic might`:                        []any{flag("Condition:ChaoticMight")},
	`elusive`:                              []any{flag("Condition:CanBeElusive")},
	`adrenaline`:                           []any{flag("Condition:Adrenaline")},
	`arcane surge`:                         []any{flag("Condition:ArcaneSurge")},
	`your aura buffs do not affect allies`: []any{flag("SelfAurasCannotAffectAllies")},
	`your curses have ([0-9]+)% increased effect if ([0-9]+)% of curse duration expired`: fn(func(c caps) any {
		return []any{mod("CurseEffect", "INC", c.n(1), Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "CurseExpired", "threshold": c.n(2)}, Tag{"type": "SkillType", "skillType": SkillType.Hex})}
	}),
	`non-aura hexes expire upon reaching ([0-9]+)% of base effect non-aura hexes gain ([0-9]+)% increased effect per second`: fn(func(c caps) any {
		return []any{mod("CurseEffect", "INC", c.n(2), Tag{"type": "Multiplier", "actor": "enemy", "var": "CurseDurationExpired", "limit": c.n(1), "limitTotal": true}, Tag{"type": "SkillType", "skillType": SkillType.Aura, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Hex})}
	}),
	`enemies cursed by you have malediction if ([0-9]+)% of curse duration expired`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMalediction", Tag{"type": "MultiplierThreshold", "var": "CurseExpired", "threshold": c.n(1)}, Tag{"type": "ActorCondition", "var": "Cursed"})})}
	}),
	`enemies cursed by you are hindered if ([0-9]+)% of curse duration expired`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Hindered", Tag{"type": "MultiplierThreshold", "var": "CurseExpired", "threshold": c.n(1)}, Tag{"type": "ActorCondition", "var": "Cursed"})})}
	}),
	`excommunicate enemies on melee hit for ([0-9]+) seconds`:        []any{flag("Condition:CanExcommunicate")},
	`auras from your skills can only affect you`:                     []any{flag("SelfAurasOnlyAffectYou")},
	`auras from your skills which affect allies also affect enemies`: []any{flag("AurasAffectEnemies")},
	`aura buffs from skills have ([0-9]+)% increased effect on you for each herald affecting you`: fn(func(c caps) any {
		return []any{mod("SkillAuraEffectOnSelf", "INC", c.n(1), Tag{"type": "Multiplier", "var": "Herald"})}
	}),
	`aura buffs from skills have ([0-9]+)% increased effect on you for each herald affecting you, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("SkillAuraEffectOnSelf", "INC", c.n(1), Tag{"type": "Multiplier", "var": "Herald", "globalLimit": c.n(2), "globalLimitKey": "PurposefulHarbinger"})}
	}),
	`auras from your skills have ([0-9]+)% increased effect on you for each herald affecting you, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("SkillAuraEffectOnSelf", "INC", c.n(1), Tag{"type": "Multiplier", "var": "Herald", "globalLimit": c.n(2), "globalLimitKey": "PurposefulHarbinger"})}
	}),
	`([0-9]+)% increased area of effect per power charge, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Multiplier", "var": "PowerCharge", "globalLimit": c.n(2), "globalLimitKey": "VastPower"})}
	}),
	`([0-9]+)% increased area of effect per second you've been stationary, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Multiplier", "var": "StationarySeconds", "globalLimit": c.n(2), "globalLimitKey": "ExpansiveMight", "limitTotal": true})}
	}),
	`([0-9]+)% increased chaos damage per ([0-9]+) maximum mana, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("ChaosDamage", "INC", c.n(1), Tag{"type": "PerStat", "stat": "Mana", "div": c.n(2), "globalLimit": c.n(3), "globalLimitKey": "DarkIdeation"})}
	}),
	`minions have \+([0-9]+)% to damage over time multiplier per ghastly eye jewel affecting you, up to a maximum of \+([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("DotMultiplier", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "GhastlyEyeJewel", "actor": "parent", "globalLimit": c.n(2), "globalLimitKey": "AmanamuGaze"})})}
	}),
	`([0-9]+)% increased effect of arcane surge on you per hypnotic eye jewel affecting you, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("ArcaneSurgeEffect", "INC", c.n(1), Tag{"type": "Multiplier", "var": "HypnoticEyeJewel", "globalLimit": c.n(2), "globalLimitKey": "KurgalGaze"})}
	}),
	`([0-9]+)% increased main hand critical strike chance per murderous eye jewel affecting you, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), Tag{"type": "Multiplier", "var": "MurderousEyeJewel", "globalLimit": c.n(2), "globalLimitKey": "TecrodGazeMainHand"}, Tag{"type": "Condition", "var": "MainHandAttack"})}
	}),
	`\+([0-9]+)% to off hand critical strike multiplier per murderous eye jewel affecting you, up to a maximum of \+([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("CritMultiplier", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "MurderousEyeJewel", "globalLimit": c.n(2), "globalLimitKey": "TecrodGazeOffHand"}, Tag{"type": "Condition", "var": "OffHandAttack"})}
	}),
	`nearby allies' damage with hits is lucky`:                                           []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": flag("LuckyHits")})},
	`your damage with hits is lucky`:                                                     []any{flag("LuckyHits")},
	`your damage with hits is lucky while on low life`:                                   []any{flag("LuckyHits", Tag{"type": "Condition", "var": "LowLife"})},
	`damage with hits is unlucky`:                                                        []any{flag("UnluckyHits")},
	`chaos damage with hits is lucky`:                                                    []any{flag("ChaosLuckyHits")},
	`lightning damage with hits is lucky if you[' ]h?a?ve blocked spell damage recently`: []any{flag("LightningLuckHits", Tag{"type": "Condition", "var": "BlockedSpellRecently"})},
	`cold damage with hits is lucky if you[' ]h?a?ve suppressed spell damage recently`:   []any{flag("ColdLuckyHits", Tag{"type": "Condition", "var": "SuppressedRecently"})},
	`fire damage with hits is lucky if you[' ]h?a?ve blocked an attack recently`:         []any{flag("FireLuckyHits", Tag{"type": "Condition", "var": "BlockedAttackRecently"})},
	`elemental damage with hits is lucky while you are shocked`:                          []any{flag("ElementalLuckHits", Tag{"type": "Condition", "var": "Shocked"})},
	`your lucky or unlucky effects are instead unexciting`:                               []any{flag("Unexciting")},
	`allies' aura buffs do not affect you`:                                               []any{flag("AlliesAurasCannotAffectSelf")},
	`([0-9]+)% increased effect of non-curse auras from your skills on enemies`: fn(func(c caps) any {
		return []any{mod("DebuffEffect", "INC", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.AppliesCurse, "neg": true}), mod("AuraEffect", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Death Aura"})}
	}),
	`enemies can have 1 additional curse`:                              []any{mod("EnemyCurseLimit", "BASE", 1)},
	`you can apply an additional curse`:                                []any{mod("EnemyCurseLimit", "BASE", 1)},
	`you can apply an additional curse during effect`:                  []any{mod("EnemyCurseLimit", "BASE", 1)},
	`you can apply an additional curse while affected by malevolence`:  []any{mod("EnemyCurseLimit", "BASE", 1, Tag{"type": "Condition", "var": "AffectedByMalevolence"})},
	`you can apply an additional curse while at maximum power charges`: []any{mod("EnemyCurseLimit", "BASE", 1, Tag{"type": "StatThreshold", "stat": "PowerCharges", "thresholdStat": "PowerChargesMax"})},
	`you can apply an additional curse if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("EnemyCurseLimit", "BASE", 1, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "Item", "threshold": c.n(1)})}
	}),
	`you can apply one fewer curse`: []any{mod("EnemyCurseLimit", "BASE", -1)},
	`curses on enemies in your chilling areas have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("CurseEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "InChillingArea"})}
	}),
	`hexes you inflict have their effect increased by twice their doom instead`: []any{mod("DoomEffect", "MORE", 100)},
	`nearby enemies have an additional ([0-9]+)% chance to receive a critical strike`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("SelfExtraCritChance", "BASE", c.n(1))})}
	}),
	`nearby enemies have (-[0-9]+)% to all resistances`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("ElementalResist", "BASE", c.n(1))}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ChaosResist", "BASE", c.n(1))})}
	}),
	`enemies ignited or chilled by you have (-[0-9]+)% to elemental resistances`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("ElementalResist", "BASE", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Ignited", "Chilled"}})}
	}),
	`reserves ([0-9]+)% of nearby enemy monsters' life`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("LifeReservationPercent", "BASE", c.n(1))})}
	}),
	`nearby enemy monsters have at least ([0-9]+)% of life reserved`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("LifeReservationPercent", "BASE", c.n(1))})}
	}),
	`your hits inflict decay, dealing ([0-9]+) chaos damage per second for [0-9]+ seconds`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "decay", "value": c.n(1), "merge": "MAX"})}
	}),
	`inflict decay on enemies you curse with hex or mark skills, dealing ([0-9]+) chaos damage per second for [0-9]+ seconds`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "decay", "value": c.n(1), "merge": "MAX"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`inflict decay on enemies you curse with hex skills, dealing ([0-9]+) chaos damage per second for [0-9]+ seconds`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "decay", "value": c.n(1), "merge": "MAX"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`temporal chains has ([0-9]+)% reduced effect on you`: fn(func(c caps) any {
		return []any{mod("CurseEffectOnSelf", "INC", -c.n(1), Tag{"type": "SkillName", "skillName": "Temporal Chains"})}
	}),
	`unaffected by temporal chains`:        []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Temporal Chains"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`targets are unaffected by your hexes`: []any{mod("CurseEffect", "MORE", -100, Tag{"type": "SkillType", "skillType": SkillType.Hex})},
	`([+\-][0-9.]+) seconds to cat's stealth duration`: fn(func(c caps) any {
		return []any{mod("PrimaryDuration", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Aspect of the Cat"})}
	}),
	`([+\-][0-9.]+) seconds to cat's agility duration`: fn(func(c caps) any {
		return []any{mod("SecondaryDuration", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Aspect of the Cat"})}
	}),
	`([+\-][0-9.]+) seconds to avian's might duration`: fn(func(c caps) any {
		return []any{mod("PrimaryDuration", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Aspect of the Avian"})}
	}),
	`([+\-][0-9.]+) seconds to avian's flight duration`: fn(func(c caps) any {
		return []any{mod("SecondaryDuration", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Aspect of the Avian"})}
	}),
	`aspect of the spider can inflict spider's web on enemies an additional time`:       []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("Multiplier:SpiderWebApplyStackMax", "BASE", 1)}, Tag{"type": "SkillName", "skillName": "Aspect of the Spider"})},
	`aspect of the avian also grants avian's might and avian's flight to nearby allies`: []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("BuffAppliesToAllies")}, Tag{"type": "SkillName", "skillName": "Aspect of the Avian"})},
	`marked enemy takes ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"})}
	}),
	`marked enemy has ([0-9]+)% reduced accuracy rating`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("Accuracy", "INC", -c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"})}
	}),
	`you are cursed with level ([0-9]+) ([^0-9]+)`: fn(func(c caps) any {
		return []any{mod("ExtraCurse", "LIST", Tag{"skillId": gemIdOrNil(c.s(2)), "level": c.n(1), "applyToPlayer": true})}
	}),
	`you are cursed with ([^0-9]+)`: fn(func(c caps) any {
		return []any{mod("ExtraCurse", "LIST", Tag{"skillId": gemIdOrNil(c.s(1)), "level": 1, "applyToPlayer": true})}
	}),
	`you count as on low life while you are cursed with vulnerability`:  []any{flag("Condition:LowLife", Tag{"type": "Condition", "var": "AffectedByVulnerability"})},
	`you count as on full life while you are cursed with vulnerability`: []any{flag("Condition:FullLife", Tag{"type": "Condition", "var": "AffectedByVulnerability"})},
	`if you consumed a corpse recently, you and nearby allies regenerate ([0-9]+)% of life per second`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("LifeRegenPercent", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "ConsumedCorpseRecently"})}
	}),
	`if you have blocked recently, you and nearby allies regenerate ([0-9]+)% of life per second`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("LifeRegenPercent", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "BlockedRecently"})}
	}),
	`you are at maximum chance to block attack damage if you have not blocked recently`: []any{flag("MaxBlockIfNotBlockedRecently", Tag{"type": "Condition", "var": "BlockedRecently", "neg": true})},
	`you are at maximum chance to block spell damage if you have not blocked recently`:  []any{flag("MaxSpellBlockIfNotBlockedRecently", Tag{"type": "Condition", "var": "BlockedRecently", "neg": true})},
	`\+([0-9]+)% chance to block attack damage if you have not blocked recently`: fn(func(c caps) any {
		return []any{mod("BlockChance", "BASE", c.n(1), Tag{"type": "Condition", "var": "BlockedRecently", "neg": true})}
	}),
	`\+([0-9]+)% chance to block spell damage if you have not blocked recently`: fn(func(c caps) any {
		return []any{mod("SpellBlockChance", "BASE", c.n(1), Tag{"type": "Condition", "var": "BlockedRecently", "neg": true})}
	}),
	`y?o?u?r? ?chance to block is lucky`:                                                  []any{flag("BlockChanceIsLucky"), flag("ProjectileBlockChanceIsLucky"), flag("SpellBlockChanceIsLucky"), flag("SpellProjectileBlockChanceIsLucky")},
	`y?o?u?r? ?chance to block attack damage is lucky`:                                    []any{flag("BlockChanceIsLucky"), flag("ProjectileBlockChanceIsLucky")},
	`y?o?u?r? ?chance to block attack damage is unlucky`:                                  []any{flag("BlockChanceIsUnlucky"), flag("ProjectileBlockChanceIsUnlucky")},
	`y?o?u?r? ?chance to block is unlucky`:                                                []any{flag("BlockChanceIsUnlucky"), flag("ProjectileBlockChanceIsUnlucky"), flag("SpellBlockChanceIsUnlucky"), flag("SpellProjectileBlockChanceIsUnlucky")},
	`y?o?u?r? ?chance to block spell damage is lucky`:                                     []any{flag("SpellBlockChanceIsLucky"), flag("SpellProjectileBlockChanceIsLucky")},
	`y?o?u?r? ?chance to block spell damage is unlucky`:                                   []any{flag("SpellBlockChanceIsUnlucky"), flag("SpellProjectileBlockChanceIsUnlucky")},
	`your lucky or unlucky effects use the best or worst from three rolls instead of two`: []any{flag("ExtremeLuck")},
	`chance to block attack or spell damage is lucky if you've blocked recently`:          []any{flag("BlockChanceIsLucky", Tag{"type": "Condition", "var": "BlockedRecently"}), flag("ProjectileBlockChanceIsLucky", Tag{"type": "Condition", "var": "BlockedRecently"}), flag("SpellBlockChanceIsLucky", Tag{"type": "Condition", "var": "BlockedRecently"}), flag("SpellProjectileBlockChanceIsLucky", Tag{"type": "Condition", "var": "BlockedRecently"})},
	`([0-9.]+)% of evasion rating is regenerated as life per second while focus?sed`: fn(func(c caps) any {
		return []any{mod("LifeRegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Evasion", "percent": c.n(1)}, Tag{"type": "Condition", "var": "Focused"})}
	}),
	`nearby allies have ([0-9]+)% increased defences per ([0-9]+) strength you have`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("Defences", "INC", c.n(1))}, Tag{"type": "PerStat", "stat": "Str", "div": c.n(2)})}
	}),
	`nearby allies have \+([0-9]+)% to critical strike multiplier per ([0-9]+) dexterity you have`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("CritMultiplier", "BASE", c.n(1))}, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(2)})}
	}),
	`nearby allies have ([0-9]+)% increased cast speed per ([0-9]+) intelligence you have`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("Speed", "INC", c.n(1), nil, ModFlag.Cast)}, Tag{"type": "PerStat", "stat": "Int", "div": c.n(2)})}
	}),
	`quicksilver flasks you use also apply to nearby allies`:                  []any{flag("QuickSilverAppliesToAllies")},
	`you gain divinity for [0-9]+ seconds on reaching maximum divine charges`: []any{mod("ElementalDamage", "MORE", 75, Tag{"type": "Condition", "var": "Divinity"}), mod("ElementalDamageTaken", "MORE", -25, Tag{"type": "Condition", "var": "Divinity"})},
	`your nearby party members maximum endurance charges is equal to yours`:   []any{flag("PartyMemberMaximumEnduranceChargesEqualToYours")},
	`your maximum endurance charges is equal to your maximum frenzy charges`:  []any{flag("MaximumEnduranceChargesIsMaximumFrenzyCharges")},
	`your maximum frenzy charges is equal to your maximum power charges`:      []any{flag("MaximumFrenzyChargesIsMaximumPowerCharges")},
	`your curse limit is equal to your maximum power charges`:                 []any{flag("CurseLimitIsMaximumPowerCharges")},
	`consecrated ground you create grants ([0-9]+)% increased accuracy rating to you and allies`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("ConsecratedGroundAlsoAccuracy", "INC", c.n(1), Tag{"type": "Condition", "var": "OnConsecratedGround"})})}
	}),
	`consecrated ground created during effect applies ([0-9]+)% increased damage taken to enemies`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTakenConsecratedGround", "INC", c.n(1), Tag{"type": "Condition", "var": "OnConsecratedGround"})}, Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`consecrated ground you create while affected by zealotry causes enemies to take ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTakenConsecratedGround", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnConsecratedGround"}, Tag{"type": "Condition", "var": "AffectedByZealotry"})}
	}),
	`if you've warcried recently, you and nearby allies have ([0-9]+)% increased attack, cast and movement speed`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("Speed", "INC", c.n(1))}, Tag{"type": "Condition", "var": "UsedWarcryRecently"}), mod("ExtraAura", "LIST", Tag{"mod": mod("MovementSpeed", "INC", c.n(1))}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})}
	}),
	`([0-9]+)% increased movement speed while on full life`: fn(func(c caps) any {
		return []any{mod("MovementSpeed", "INC", c.n(1), Tag{"type": "Condition", "var": "FullLife"})}
	}),
	`when you warcry, you and nearby allies gain onslaught for 4 seconds`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Onslaught")}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})},
	`warcries grant arcane surge to you and allies, with ([0-9]+)% increased effect per ([0-9]+) power, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Condition:ArcaneSurge")}, Tag{"type": "Condition", "var": "UsedWarcryRecently"}), mod("ArcaneSurgeEffect", "INC", c.n(1), Tag{"type": "PerStat", "stat": "WarcryPower", "div": c.n(2), "globalLimit": c.n(3), "globalLimitKey": "Brinerot Flag"}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})}
	}),
	`gain arcane surge after spending a total of ([0-9]+) mana`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Condition:ArcaneSurge")}, Tag{"type": "MultiplierThreshold", "var": "ManaSpentRecently", "threshold": c.n(1)})}
	}),
	`gain arcane surge after spending a total of ([0-9]+) life`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Condition:ArcaneSurge")}, Tag{"type": "MultiplierThreshold", "var": "LifeSpentRecently", "threshold": c.n(1)})}
	}),
	`gain onslaught for ([0-9]+) seconds on hit while at maximum frenzy charges`: []any{flag("Onslaught", Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"}, Tag{"type": "Condition", "var": "HitRecently"})},
	`enemies in your chilling areas take ([0-9]+)% increased lightning damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningDamageTaken", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "InChillingArea"})}
	}),
	`warcries count as having ([0-9]+) additional nearby enemies`: fn(func(c caps) any { return []any{mod("Multiplier:WarcryNearbyEnemies", "BASE", c.n(1))} }),
	`enemies taunted by your warcries take ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "Condition", "var": "Taunted"})}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})}
	}),
	`warcries have minimum of ([0-9]+) power`:                                                                              []any{flag("CryWolfMinimumPower")},
	`warcries have infinite power`:                                                                                         []any{flag("WarcryInfinitePower")},
	`your warcries do not grant buffs or charges to you`:                                                                   []any{flag("CannotGainWarcryBuffs")},
	`([0-9]+)% chance to inflict corrosion on hit with attacks`:                                                            []any{flag("Condition:CanCorrode")},
	`([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit`:                                                     []any{flag("Condition:CanWither")},
	`melee weapon hits inflict ([0-9]+) withered debuffs for ([0-9]+) seconds`:                                             []any{flag("Condition:CanWither")},
	`([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit with this weapon`:                                    []any{flag("Condition:CanWither")},
	`([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit against cursed enemies`:                              []any{flag("Condition:CanWither", Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})},
	`([0-9]+)% chance to inflict withered for two seconds on hit if there are ([0-9]+) or fewer withered debuffs on enemy`: []any{flag("Condition:CanWither")},
	`inflict withered for ([0-9]+) seconds on hit with this weapon`:                                                        []any{flag("Condition:CanWither")},
	`chaos skills inflict up to ([0-9]+) withered debuffs on hit for ([0-9]+) seconds`:                                     []any{flag("Condition:CanWither")},
	`minions have ([0-9]+)% chance to inflict withered on hit`:                                                             []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:CanWither")})},
	`enemies take ([0-9]+)% increased elemental damage from your hits for each withered you have inflicted on them`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("ElementalDamageTaken", "INC", c.n(1), Tag{"type": "Multiplier", "var": "WitheredStack", "limit": 15})})}
	}),
	`your hits cannot penetrate or ignore elemental resistances`:                             []any{flag("CannotElePenIgnore")},
	`nearby enemies have malediction`:                                                        []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMalediction")})},
	`gain shaper's presence for 10 seconds when you kill a rare or unique enemy`:             []any{mod("ExtraAura", "LIST", Tag{"mod": flag("HasShapersPresence")}, Tag{"type": "Condition", "var": "KilledUniqueEnemy"})},
	`gain maddening presence for 10 seconds when you kill a rare or unique enemy`:            []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMaddeningPresence")}, Tag{"type": "Condition", "var": "KilledUniqueEnemy"})},
	`elemental damage you deal with hits is resisted by lowest elemental resistance instead`: []any{flag("ElementalDamageUsesLowestResistance")},
	`you take ([0-9]+) chaos damage per second for 3 seconds on kill`: fn(func(c caps) any {
		return []any{mod("ChaosDegen", "BASE", c.n(1), Tag{"type": "Condition", "var": "KilledLast3Seconds"})}
	}),
	`regenerate ([0-9]+) life per second for each ([0-9]+)% uncapped fire resistance`: fn(func(c caps) any {
		return []any{mod("LifeRegen", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "FireResistTotal", "div": 1 / c.n(2)})}
	}),
	`regenerate ([0-9]+) life over 1 second for each spell you cast`: fn(func(c caps) any {
		return []any{mod("LifeRegen", "BASE", c.n(1), Tag{"type": "Condition", "var": "CastLast1Seconds"})}
	}),
	`and nearby allies regenerate ([0-9]+) life per second`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"mod": mod("LifeRegen", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "KilledPoisonedLast2Seconds"})}
	}),
	`([0-9]+)% increased life regeneration rate`: fn(func(c caps) any { return []any{mod("LifeRegen", "INC", c.n(1))} }),
	`every ([0-9]+) seconds, regenerate life equal to ([0-9]+)% of armour and evasion rating over ([0-9]+) second`: fn(func(c caps) any {
		return []any{mod("LifeRegen", "BASE", 1, Tag{"type": "Condition", "var": "LifeRegenBurstFull"}, Tag{"type": "PercentStat", "stat": "Armour", "percent": c.s(2)}), mod("LifeRegen", "BASE", 1/c.n(1)*c.n(3), Tag{"type": "Condition", "var": "LifeRegenBurstAvg"}, Tag{"type": "PercentStat", "stat": "Armour", "percent": c.s(2)}), mod("LifeRegen", "BASE", 1, Tag{"type": "Condition", "var": "LifeRegenBurstFull"}, Tag{"type": "PercentStat", "stat": "Evasion", "percent": c.s(2)}), mod("LifeRegen", "BASE", 1/c.n(1)*c.n(3), Tag{"type": "Condition", "var": "LifeRegenBurstAvg"}, Tag{"type": "PercentStat", "stat": "Evasion", "percent": c.s(2)})}
	}),
	`every ([0-9]+) seconds, regenerate energy shield equal to ([0-9]+)% of evasion rating over ([0-9]+) second`: fn(func(c caps) any {
		return []any{mod("EnergyShieldRegen", "BASE", 1, Tag{"type": "Condition", "var": "LifeRegenBurstFull"}, Tag{"type": "PercentStat", "stat": "Evasion", "percent": c.s(2)}), mod("EnergyShieldRegen", "BASE", 1/c.n(1)*c.n(3), Tag{"type": "Condition", "var": "LifeRegenBurstAvg"}, Tag{"type": "PercentStat", "stat": "Evasion", "percent": c.s(2)})}
	}),
	`regenerate ([0-9]+)% of life per second for each different ailment affecting you`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Bleeding"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Ignited"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Scorched"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Chilled"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Frozen"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Brittle"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Shocked"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Sapped"}), mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Poisoned"})}
	}),
	`fire skills have a ([0-9]+)% chance to apply fire exposure on hit`:                  fn(func(c caps) any { return []any{mod("FireExposureChance", "BASE", c.n(1))} }),
	`cold skills have a ([0-9]+)% chance to apply cold exposure on hit`:                  fn(func(c caps) any { return []any{mod("ColdExposureChance", "BASE", c.n(1))} }),
	`lightning skills have a ([0-9]+)% chance to apply lightning exposure on hit`:        fn(func(c caps) any { return []any{mod("LightningExposureChance", "BASE", c.n(1))} }),
	`([0-9]+)% chance to inflict cold exposure on hit with cold damage`:                  fn(func(c caps) any { return []any{mod("ColdExposureChance", "BASE", c.n(1))} }),
	`socketed skills apply fire, cold and lightning exposure on hit`:                     []any{mod("FireExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"}), mod("ColdExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"}), mod("LightningExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"})},
	`inflict fire, cold, and lightning exposure on hit`:                                  []any{mod("FireExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"}), mod("ColdExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"}), mod("LightningExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"})},
	`inflict fire exposure on hit`:                                                       []any{mod("FireExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"})},
	`inflict cold exposure on hit`:                                                       []any{mod("ColdExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"})},
	`inflict lightning exposure on hit`:                                                  []any{mod("LightningExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"})},
	`nearby enemies have fire exposure`:                                                  []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"})},
	`nearby enemies have cold exposure`:                                                  []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"})},
	`nearby enemies have lightning exposure`:                                             []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"})},
	`nearby enemies have fire exposure while at maximum rage`:                            []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "HaveMaximumRage"})},
	`nearby enemies have fire exposure while you are affected by herald of ash`:          []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "AffectedByHeraldofAsh"})},
	`nearby enemies have cold exposure while you are affected by herald of ice`:          []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "AffectedByHeraldofIce"})},
	`nearby enemies have lightning exposure while you are affected by herald of thunder`: []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "AffectedByHeraldofThunder"})},
	`inflict fire, cold and lightning exposure on nearby enemies when used`:              []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "UsingFlask"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "UsingFlask"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -10)}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "Condition", "var": "UsingFlask"})},
	`enemies near your linked targets have fire, cold and lightning exposure`:            []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireExposure", "BASE", -10, Tag{"type": "Condition", "var": "NearLinkedTarget"})}, Tag{"type": "Condition", "var": "Effective"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdExposure", "BASE", -10, Tag{"type": "Condition", "var": "NearLinkedTarget"})}, Tag{"type": "Condition", "var": "Effective"}), mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningExposure", "BASE", -10, Tag{"type": "Condition", "var": "NearLinkedTarget"})}, Tag{"type": "Condition", "var": "Effective"})},
	`inflict ([0-9a-zA-Z]+) exposure on hit, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(1))+"ExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"}), mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(3))+"Exposure", "BASE", -c.n(2))}, Tag{"type": "Condition", "var": "Effective"})}
	}),
	`while a unique enemy is in your presence, inflict ([0-9a-zA-Z]+) exposure on hit, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(1))+"ExposureChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"}, Tag{"type": "Condition", "var": "Effective"}), mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(3))+"Exposure", "BASE", -c.n(2), Tag{"type": "Condition", "var": "RareOrUnique"})}, Tag{"type": "Condition", "var": "Effective"})}
	}),
	`while a pinnacle atlas boss is in your presence, inflict ([0-9a-zA-Z]+) exposure on hit, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(1))+"ExposureChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"}, Tag{"type": "Condition", "var": "Effective"}), mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(3))+"Exposure", "BASE", -c.n(2), Tag{"type": "Condition", "var": "PinnacleBoss"})}, Tag{"type": "Condition", "var": "Effective"})}
	}),
	`inflict fire exposure on hit against enemies with ([0-9]+) cinderflame, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: fn(func(c caps) any {
		return []any{mod("FireExposureChance", "BASE", 100, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "MultiplierThreshold", "var": "CinderflameStacks", "threshold": c.n(1)}), mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(3))+"Exposure", "BASE", -c.n(2))}, Tag{"type": "Condition", "var": "Effective"}, Tag{"type": "MultiplierThreshold", "var": "CinderflameStacks", "threshold": c.n(1)})}
	}),
	`fire exposure you inflict applies an extra (-?[0-9]+)% to fire resistance`:           fn(func(c caps) any { return []any{mod("ExtraFireExposure", "BASE", c.n(1))} }),
	`cold exposure you inflict applies an extra (-?[0-9]+)% to cold resistance`:           fn(func(c caps) any { return []any{mod("ExtraColdExposure", "BASE", c.n(1))} }),
	`lightning exposure you inflict applies an extra (-?[0-9]+)% to lightning resistance`: fn(func(c caps) any { return []any{mod("ExtraLightningExposure", "BASE", c.n(1))} }),
	`exposure you inflict applies at least (-[0-9]+)% to the affected resistance`:         fn(func(c caps) any { return []any{mod("ExposureMin", "OVERRIDE", c.n(1))} }),
	`modifiers to minimum endurance charges instead apply to minimum brutal charges`:      []any{flag("MinimumEnduranceChargesEqualsMinimumBrutalCharges")},
	`modifiers to minimum frenzy charges instead apply to minimum affliction charges`:     []any{flag("MinimumFrenzyChargesEqualsMinimumAfflictionCharges")},
	`modifiers to minimum power charges instead apply to minimum absorption charges`:      []any{flag("MinimumPowerChargesEqualsMinimumAbsorptionCharges")},
	`maximum brutal charges is equal to maximum endurance charges`:                        []any{flag("MaximumEnduranceChargesEqualsMaximumBrutalCharges")},
	`maximum affliction charges is equal to maximum frenzy charges`:                       []any{flag("MaximumFrenzyChargesEqualsMaximumAfflictionCharges")},
	`maximum absorption charges is equal to maximum power charges`:                        []any{flag("MaximumPowerChargesEqualsMaximumAbsorptionCharges")},
	`([0-9]+)% chance to gain a brine charge instead of an endurance charge`: fn(func(c caps) any {
		return []any{flag("CanGainBrineCharges"), mod("BrineChargeGainChance", "BASE", c.n(1))}
	}),
	`gain brutal charges instead of endurance charges`:  []any{flag("EnduranceChargesConvertToBrutalCharges")},
	`gain affliction charges instead of frenzy charges`: []any{flag("FrenzyChargesConvertToAfflictionCharges")},
	`gain absorption charges instead of power charges`:  []any{flag("PowerChargesConvertToAbsorptionCharges")},
	`regenerate ([0-9]+)% life over one second when hit while sane`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "Insane", "neg": true}, Tag{"type": "Condition", "var": "BeenHitRecently"})}
	}),
	`you count as on low ([a-zA-Z]+) while at ([0-9]+)% of maximum ([a-zA-Z]+) or below`:  fn(func(c caps) any { return []any{mod("Low"+firstToUpper(c.s(1))+"Percentage", "BASE", c.n(2)/100.0)} }),
	`you count as on full ([a-zA-Z]+) while at ([0-9]+)% of maximum ([a-zA-Z]+) or above`: fn(func(c caps) any { return []any{mod("Full"+firstToUpper(c.s(1))+"Percentage", "BASE", c.n(2)/100.0)} }),
	`([0-9]+)% more maximum life if you have at least ([0-9]+) life masteries allocated`: fn(func(c caps) any {
		return []any{mod("Life", "MORE", c.n(1), Tag{"type": "MultiplierThreshold", "var": "AllocatedLifeMastery", "threshold": c.n(2)})}
	}),
	`left ring slot: cover enemies in ash for ([0-9]+) seconds when you ignite them`:    []any{mod("CoveredInAshEffect", "BASE", 20, Tag{"type": "SlotNumber", "num": 1}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"})},
	`right ring slot: cover enemies in frost for ([0-9]+) seconds when you freeze them`: []any{mod("CoveredInFrostEffect", "BASE", 20, Tag{"type": "SlotNumber", "num": 2}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"})},
	`nearby enemies are covered in ash`:                                                 []any{mod("CoveredInAshEffect", "BASE", 20)},
	`nearby enemies are covered in ash if you haven't moved in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("CoveredInAshEffect", "BASE", 20, Tag{"type": "MultiplierThreshold", "var": "StationarySeconds", "threshold": c.n(1)}, Tag{"type": "Condition", "var": "Stationary"})}
	}),
	`your warcries cover enemies in ash for ([0-9]+) seconds`:                                            []any{mod("CoveredInAshEffect", "BASE", 20, Tag{"type": "Condition", "var": "UsedWarcryRecently"})},
	`enemies near targets you shatter have ([0-9]+)% chance to be covered in frost for ([0-9]+) seconds`: []any{mod("CoveredInFrostEffect", "BASE", 20, Tag{"type": "Condition", "var": "ShatteredEnemyRecently"})},
	`([a-zA-Z \t\n\v\f\r]+) has ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("BuffEffect", "INC", c.s(2), Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))})}
	}),
	`debuffs on you expire ([0-9]+)% faster`: fn(func(c caps) any { return []any{mod("SelfDebuffExpirationRate", "BASE", c.n(1))} }),
	`debuffs on you expire ([0-9]+)% slower`: fn(func(c caps) any { return []any{mod("SelfDebuffExpirationRate", "BASE", -c.n(1))} }),
	`debuffs on you expire ([0-9]+)% faster while affected by haste`: fn(func(c caps) any {
		return []any{mod("SelfDebuffExpirationRate", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByHaste"})}
	}),
	`warcries debilitate enemies for ([0-9]+) seconds?`:                                                                                          []any{mod("DebilitateChance", "BASE", 100)},
	`debilitate enemies for ([0-9]+) seconds? when you suppress their spell damage`:                                                              []any{mod("DebilitateChance", "BASE", 100)},
	`debilitate nearby enemies for ([0-9]+) seconds? when f?l?a?s?k? ?effect ends`:                                                               []any{mod("DebilitateChance", "BASE", 100)},
	`counterattacks have a ([0-9]+)% chance to debilitate on hit for ([0-9]+) seconds?`:                                                          fn(func(c caps) any { return []any{mod("DebilitateChance", "BASE", c.n(1))} }),
	`retaliation skills debilitate enemies for ([0-9]+) seconds on hit`:                                                                          []any{mod("DebilitateChance", "BASE", 100)},
	`eat a soul when you hit a unique enemy, no more than once every second`:                                                                     []any{flag("Condition:CanHaveSoulEater")},
	`eat a soul when you hit a rare or unique enemy, no more than once every [0-9.]+ seconds`:                                                    []any{flag("Condition:CanHaveSoulEater")},
	`([0-9]+)% chance to gain soul eater for ([0-9]+) seconds on killing blow against rare and unique enemies with double strike or dual strike`: []any{flag("Condition:CanHaveSoulEater")},
	`when ([0-9]+)% of your hex's duration expires on an enemy, eat ([0-9]+) soul per enemy power`:                                               []any{flag("Condition:CanHaveSoulEater")},
	`eat ([0-9]+) souls when you kill a rare or unique enemy with this weapon`:                                                                   []any{flag("Condition:CanHaveSoulEater")},
	`maximum ([0-9]+) eaten souls`:                   fn(func(c caps) any { return []any{mod("SoulEaterMax", "OVERRIDE", c.n(1))} }),
	`([+\-][0-9]+) to maximum number of eaten souls`: fn(func(c caps) any { return []any{mod("SoulEaterMax", "BASE", c.n(1))} }),
	`([0-9]+)% increased attack and cast speed if you've killed recently`: fn(func(c caps) any {
		return []any{mod("Speed", "INC", c.n(1), nil, ModFlag.Cast, Tag{"type": "Condition", "var": "KilledRecently"}), mod("Speed", "INC", c.n(1), nil, ModFlag.Attack, Tag{"type": "Condition", "var": "KilledRecently"})}
	}),
	`gain adrenaline for 1 second when you change stance`:                                        []any{flag("Condition:Adrenaline", Tag{"type": "Condition", "var": "StanceChangeLastSecond"})},
	`with a searching eye jewel socketed, maim enemies for ([0-9]) seconds on hit with attacks`:  []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Maimed", nil, ModFlag.Attack)}, Tag{"type": "Condition", "var": "HaveSearchingEyeJewelIn{SlotName}"})},
	`with a searching eye jewel socketed, blind enemies for ([0-9]) seconds on hit with attacks`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Blinded", nil, ModFlag.Attack)}, Tag{"type": "Condition", "var": "HaveSearchingEyeJewelIn{SlotName}"})},
	`enemies maimed by you take ([0-9]+)% increased damage over time`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTakenOverTime", "INC", c.n(1))}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Maimed"})}
	}),
	`([0-9]+)% increased defences while you have at least four linked targets`: fn(func(c caps) any {
		return []any{mod("Defences", "INC", c.n(1), Tag{"type": "MultiplierThreshold", "var": "LinkedTargets", "threshold": 4})}
	}),
	`your movement speed is equal to the highest movement speed among linked players`: []any{flag("MovementSpeedEqualHighestLinkedPlayers", Tag{"type": "MultiplierThreshold", "var": "LinkedTargets", "threshold": 1})},
	`([0-9]+)% increased movement speed while you have at least two linked targets`: fn(func(c caps) any {
		return []any{mod("MovementSpeed", "INC", c.n(1), Tag{"type": "MultiplierThreshold", "var": "LinkedTargets", "threshold": 2})}
	}),
	`link skills have ([0-9]+)% increased buff effect if you have linked to a target recently`: fn(func(c caps) any {
		return []any{mod("BuffEffect", "INC", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Link}, Tag{"type": "Condition", "var": "LinkedRecently"})}
	}),
	`link skills can target damageable minions`: []any{flag("Condition:CanLinkToMinions", Tag{"type": "Condition", "var": "HaveDamageableMinion"})},
	`your linked minions take ([0-9]+)% less damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("DamageTaken", "MORE", -c.n(1), Tag{"type": "Condition", "var": "AffectedByLink"})})}
	}),
	`curses are inflicted on you instead of linked targets`:                       []any{mod("ExtraLinkEffect", "LIST", Tag{"mod": flag("CurseImmune")})},
	`elemental ailments are inflicted on you instead of linked targets`:           []any{mod("ExtraLinkEffect", "LIST", Tag{"mod": flag("ElementalAilmentImmune")})},
	`non-unique utility flasks you use apply to linked targets`:                   []any{mod("ExtraLinkEffect", "LIST", Tag{"mod": mod("ParentNonUniqueFlasksAppliedToYou", "FLAG", true, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})})},
	`non-curse auras from your skills only apply to you and linked targets`:       []any{flag("SelfAurasAffectYouAndLinkedTarget", Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.AppliesCurse, "neg": true})},
	`linked targets always count as in range of non-curse auras from your skills`: d(),
	`gain unholy might on block for ([0-9]) seconds`:                              []any{flag("Condition:UnholyMight", Tag{"type": "Condition", "var": "BlockedRecently"}), flag("Condition:CanWither", Tag{"type": "Condition", "var": "BlockedRecently"})},
	`your warcries inflict hallowing flame`:                                       []any{flag("Condition:CanInflictHallowingFlame", Tag{"type": "Condition", "var": "UsedWarcryRecently"})},
	`attacks with this weapon inflict hallowing flame on hit`:                     []any{flag("Condition:CanInflictHallowingFlame")},
	`inflict hallowing flame on hit while on consecrated ground`:                  []any{flag("Condition:CanInflictHallowingFlame", Tag{"type": "Condition", "var": "OnConsecratedGround"})},
	`inflict hallowing flame on melee hit`:                                        []any{flag("Condition:CanInflictHallowingFlame")},
	// Traps, Mines
	`traps and mines deal ([0-9]+)-([0-9]+) additional physical damage`: fn(func(c caps) any {
		return []any{mod("PhysicalMin", "BASE", c.n(1), nil, 0, KeywordFlag.Trap|KeywordFlag.Mine), mod("PhysicalMax", "BASE", c.n(2), nil, 0, KeywordFlag.Trap|KeywordFlag.Mine)}
	}),
	`traps and mines deal ([0-9]+) to ([0-9]+) additional physical damage`: fn(func(c caps) any {
		return []any{mod("PhysicalMin", "BASE", c.n(1), nil, 0, KeywordFlag.Trap|KeywordFlag.Mine), mod("PhysicalMax", "BASE", c.n(2), nil, 0, KeywordFlag.Trap|KeywordFlag.Mine)}
	}),
	`each mine applies ([0-9]+)% increased damage taken to enemies near it, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "Multiplier", "var": "ActiveMineCount", "limit": c.n(2) / c.n(1)})})}
	}),
	`each mine applies ([0-9]+)% reduced damage dealt to enemies near it, up to ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("Damage", "INC", -c.n(1), Tag{"type": "Multiplier", "var": "ActiveMineCount", "limit": c.n(2) / c.n(1)})})}
	}),
	`stormblast, icicle and pyroclast mine have ([0-9]+)% increased aura effect`: fn(func(c caps) any {
		return []any{mod("AuraEffect", "INC", c.n(1), nil, 0, KeywordFlag.Mine, Tag{"type": "SkillName", "skillNameList": []any{"Stormblast Mine", "Icicle Mine", "Pyroclast Mine"}, "includeTransfigured": true})}
	}),
	`stormblast, icicle and pyroclast mine deal no damage`:              []any{flag("DealNoDamage", Tag{"type": "SkillName", "skillNameList": []any{"Stormblast Mine", "Icicle Mine", "Pyroclast Mine"}, "includeTransfigured": true})},
	`can have up to ([0-9]+) additional traps? placed at a time`:        fn(func(c caps) any { return []any{mod("ActiveTrapLimit", "BASE", c.n(1))} }),
	`can have ([0-9]+) fewer traps placed at a time`:                    fn(func(c caps) any { return []any{mod("ActiveTrapLimit", "BASE", -c.n(1))} }),
	`can have up to ([0-9]+) additional remote mines? placed at a time`: fn(func(c caps) any { return []any{mod("ActiveMineLimit", "BASE", c.n(1))} }),
	// Additional trap & mine throw
	`throw an additional trap`:                                   []any{mod("TrapThrowCount", "BASE", 1)},
	`([0-9]+)% chance to throw up to ([0-9]+) additional traps?`: fn(func(c caps) any { return []any{mod("TrapThrowCount", "BASE", c.n(2)*c.n(1)/100.0)} }),
	`throw an additional mine`:                                   []any{mod("MineThrowCount", "BASE", 1)},
	`([0-9]+)% chance to throw up to ([0-9]+) additional mines?`: fn(func(c caps) any { return []any{mod("MineThrowCount", "BASE", c.n(2)*c.n(1)/100.0)} }),
	`([0-9]+)% chance to throw up to ([0-9]+) additional traps? or mines?`: fn(func(c caps) any {
		return []any{mod("TrapThrowCount", "BASE", c.n(2)*c.n(1)/100.0), mod("MineThrowCount", "BASE", c.n(2)*c.n(1)/100.0)}
	}),
	`skills which throw traps throw up to ([0-9]+) additional traps?`: fn(func(c caps) any { return []any{mod("TrapThrowCount", "BASE", c.n(1))} }),
	// Totems
	`can have up to ([0-9]+) additional totems? summoned at a time`:         fn(func(c caps) any { return []any{mod("ActiveTotemLimit", "BASE", c.n(1))} }),
	`attack skills can have ([0-9]+) additional totems? summoned at a time`: fn(func(c caps) any { return []any{mod("ActiveTotemLimit", "BASE", c.n(1), nil, 0, KeywordFlag.Attack)} }),
	`can [hs][au][vm][em]o?n? 1 additional siege ballista totem per ([0-9]+) dexterity`: fn(func(c caps) any {
		return []any{mod("ActiveBallistaLimit", "BASE", 1, Tag{"type": "SkillName", "skillName": "Siege Ballista", "includeTransfigured": true}, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(1)})}
	}),
	`totems fire ([0-9]+) additional projectiles`: fn(func(c caps) any { return []any{mod("ProjectileCount", "BASE", c.n(1), nil, 0, KeywordFlag.Totem)} }),
	`([0-9.]+)% of damage dealt by y?o?u?r? ?totems is leeched to you as life`: fn(func(c caps) any {
		return []any{mod("DamageLifeLeechToPlayer", "BASE", c.n(1), nil, 0, KeywordFlag.Totem)}
	}),
	`([0-9.]+)% of damage dealt by y?o?u?r? ?mines is leeched to you as life`: fn(func(c caps) any {
		return []any{mod("DamageLifeLeechToPlayer", "BASE", c.n(1), nil, 0, KeywordFlag.Mine)}
	}),
	`you can cast an additional brand`:        []any{mod("ActiveBrandLimit", "BASE", 1)},
	`you can cast ([0-9]+) additional brands`: fn(func(c caps) any { return []any{mod("ActiveBrandLimit", "BASE", c.n(1))} }),
	`([0-9]+)% increased damage while you are wielding a bow and have a totem`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), Tag{"type": "Condition", "var": "HaveTotem"}, Tag{"type": "Condition", "var": "UsingBow"})}
	}),
	`each totem applies ([0-9]+)% increased damage taken to enemies near it`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "Multiplier", "var": "TotemsSummoned"})})}
	}),
	`totems gain \+([0-9]+)% to ([0-9a-zA-Z]+) resistance`:                                   fn(func(c caps) any { return []any{mod("Totem"+firstToUpper(c.s(2))+"Resist", "BASE", c.n(1))} }),
	`totems gain \+([0-9]+)% to all elemental resistances`:                                   fn(func(c caps) any { return []any{mod("TotemElementalResist", "BASE", c.n(1))} }),
	`rejuvenation totem also grants mana regeneration equal to 15% of its life regeneration`: []any{flag("Condition:RejuvenationTotemManaRegen")},
	// Minions
	`your strength is added to your minions`:                                []any{mod("StrengthAddedToMinions", "BASE", 100)},
	`half of your strength is added to your minions`:                        []any{mod("StrengthAddedToMinions", "BASE", 50)},
	`minions gain added resistances equal to ([0-9]+)% of your resistances`: fn(func(c caps) any { return []any{mod("ResistanceAddedToMinions", "BASE", c.n(1))} }),
	`minions' accuracy rating is equal to yours`:                            []any{flag("MinionAccuracyEqualsAccuracy")},
	`minions poison enemies on hit`:                                         []any{mod("MinionModifier", "LIST", Tag{"mod": mod("PoisonChance", "BASE", 100)})},
	`minions have ([0-9]+)% chance to poison enemies on hit`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("PoisonChance", "BASE", c.n(1))})}
	}),
	`([0-9]+)% increased minion damage if you have hit recently`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1))}, Tag{"type": "Condition", "var": "HitRecently"})}
	}),
	`([0-9]+)% increased minion damage if you've used a minion skill recently`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1))}, Tag{"type": "Condition", "var": "UsedMinionSkillRecently"})}
	}),
	`minions deal ([0-9]+)% increased damage if you have warcried recently`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "UsedWarcryRecently"})})}
	}),
	`minions deal ([0-9]+)% increased damage if you've used a minion skill recently`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1))}, Tag{"type": "Condition", "var": "UsedMinionSkillRecently"})}
	}),
	`minions deal ([0-9]+)% more damage while they are on low life`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "MORE", c.n(1), Tag{"type": "Condition", "var": "LowLife"})})}
	}),
	`minions have ([0-9]+)% increased attack and cast speed if you or your minions have killed recently`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Speed", "INC", c.n(1))}, Tag{"type": "Condition", "varList": []any{"KilledRecently", "MinionsKilledRecently"}})}
	}),
	`([0-9]+)% increased minion attack speed per ([0-9]+) dexterity`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Speed", "INC", c.n(1), nil, ModFlag.Attack)}, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(2)})}
	}),
	`([0-9]+)% increased minion movement speed per ([0-9]+) dexterity`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("MovementSpeed", "INC", c.n(1))}, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(2)})}
	}),
	`minions deal ([0-9]+)% increased damage per ([0-9]+) dexterity`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1))}, Tag{"type": "PerStat", "stat": "Dex", "div": c.n(2)})}
	}),
	`minions have ([0-9]+)% chance to deal double damage while they are on full life`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "Condition", "var": "FullLife"})})}
	}),
	`minions have ([0-9]+)% chance to deal double damage per fortification on you`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "FortificationStacks", "actor": "parent"})})}
	}),
	`([0-9]+)% increased golem damage for each type of golem you have summoned`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "HavePhysicalGolem"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem}), mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveLightningGolem"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem}), mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveColdGolem"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem}), mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveFireGolem"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem}), mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveChaosGolem"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem}), mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveCarrionGolem"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem})}
	}),
	`can summon up to ([0-9]) additional golems? at a time`: fn(func(c caps) any { return []any{mod("ActiveGolemLimit", "BASE", c.n(1))} }),
	`\+([0-9]) to maximum number of sentinels of purity`:    fn(func(c caps) any { return []any{mod("ActiveSentinelOfPurityLimit", "BASE", c.n(1))} }),
	`if you have 3 primordial jewels, can summon up to ([0-9]) additional golems? at a time`: fn(func(c caps) any {
		return []any{mod("ActiveGolemLimit", "BASE", c.n(1), Tag{"type": "MultiplierThreshold", "var": "PrimordialItem", "threshold": 3})}
	}),
	`golems regenerate ([0-9])% of their maximum life per second`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("LifeRegenPercent", "BASE", c.n(1))}, Tag{"type": "SkillType", "skillType": SkillType.Golem})}
	}),
	`summoned golems regenerate ([0-9])% of their life per second`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("LifeRegenPercent", "BASE", c.n(1))}, Tag{"type": "SkillType", "skillType": SkillType.Golem})}
	}),
	`summoned carrion golems impale on hit if you have the same number of them as summoned chaos golems`: []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ImpaleChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "parent", "var": "CarrionEqualChaosGolem"}, Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveChaosGolem"})}, Tag{"type": "SkillName", "skillName": "Summon Carrion Golem", "includeTransfigured": true})},
	`summoned chaos golems impale on hit if you have the same number of them as summoned stone golems`:   []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ImpaleChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "parent", "var": "ChaosEqualStoneGolem"}, Tag{"type": "ActorCondition", "actor": "parent", "var": "HavePhysicalGolem"})}, Tag{"type": "SkillName", "skillName": "Summon Chaos Golem", "includeTransfigured": true})},
	`summoned stone golems impale on hit if you have the same number of them as summoned carrion golems`: []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ImpaleChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "parent", "var": "StoneEqualCarrionGolem"}, Tag{"type": "ActorCondition", "actor": "parent", "var": "HaveCarrionGolem"})}, Tag{"type": "SkillName", "skillName": "Summon Stone Golem", "includeTransfigured": true})},
	`maximum life of summoned elemental golems is doubled`:                                               []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Life", "MORE", 100)}, Tag{"type": "SkillType", "skillType": SkillType.Golem}, Tag{"type": "SkillType", "skillTypeList": []any{SkillType.Lightning, SkillType.Cold, SkillType.Fire}})},
	`golems summoned in the past 8 seconds deal ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "SummonedGolemInPast8Sec"})}, Tag{"type": "SkillType", "skillType": SkillType.Golem})}
	}),
	`raised zombies and spectres gain adrenaline for 8 seconds when raised`: []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:Adrenaline")}, Tag{"type": "Condition", "var": "SummonedSpectreInPast8Sec"}, Tag{"type": "SkillName", "skillName": "Raise Spectre", "includeTransfigured": true}), mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:Adrenaline")}, Tag{"type": "Condition", "var": "SummonedZombieInPast8Sec"}, Tag{"type": "SkillName", "skillName": "Raise Zombie", "includeTransfigured": true})},
	`raised spectres fire ([0-9]+) additional projectiles`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Raise Spectre", "includeTransfigured": true})}
	}),
	`gain onslaught for 10 seconds when you cast socketed golem skill`: fn(func(c caps) any {
		return []any{flag("Condition:Onslaught", Tag{"type": "Condition", "var": "SummonedGolemInPast10Sec"})}
	}),
	`s?u?m?m?o?n?e?d? ?raging spirits' hits always ignite`: []any{mod("MinionModifier", "LIST", Tag{"mod": mod("EnemyIgniteChance", "BASE", 100)}, Tag{"type": "SkillName", "skillName": "Summon Raging Spirit", "includeTransfigured": true})},
	`raised zombies have avatar of fire`:                   []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Keystone", "LIST", "Avatar of Fire")}, Tag{"type": "SkillName", "skillName": "Raise Zombie", "includeTransfigured": true})},
	`raised zombies take ([0-9.]+)% of their maximum life per second as fire damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("FireDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}, Tag{"type": "SkillName", "skillName": "Raise Zombie", "includeTransfigured": true})}
	}),
	`maximum number of summoned raging spirits is ([0-9]+)`:                fn(func(c caps) any { return []any{mod("ActiveRagingSpiritLimit", "OVERRIDE", c.n(1))} }),
	`maximum number of summoned phantasms is ([0-9]+)`:                     fn(func(c caps) any { return []any{mod("ActivePhantasmLimit", "OVERRIDE", c.n(1))} }),
	`summoned raging spirits have diamond shrine and massive shrine buffs`: []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:DiamondShrine")}, Tag{"type": "SkillName", "skillName": "Summon Raging Spirit", "includeTransfigured": true}), mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:MassiveShrine")}, Tag{"type": "SkillName", "skillName": "Summon Raging Spirit", "includeTransfigured": true})},
	`summoned phantasms have diamond shrine and massive shrine buffs`:      []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:DiamondShrine")}, Tag{"type": "SkillName", "skillName": "Summon Phantasm"}), mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:MassiveShrine")}, Tag{"type": "SkillName", "skillName": "Summon Phantasm"})},
	`minions deal no non-([a-zA-Z]+) damage`:                               fn(func(c caps) any { return dealNoNonDamageType(c.s(1), true) }),
	`minions convert ([0-9]+)% of (.+) damage to (.+) damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1))})}
	}),
	`summoned skeletons have avatar of fire`: []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Keystone", "LIST", "Avatar of Fire")}, Tag{"type": "SkillName", "skillName": "Summon Skeletons", "includeTransfigured": true})},
	`summoned skeletons take ([0-9.]+)% of their maximum life per second as fire damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("FireDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}, Tag{"type": "SkillName", "skillName": "Summon Skeletons", "includeTransfigured": true})}
	}),
	`summoned skeletons have ([0-9]+)% chance to wither enemies for ([0-9]+) seconds on hit`: []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("Condition:CanWither")}, Tag{"type": "SkillName", "skillName": "Summon Skeletons", "includeTransfigured": true})},
	`summoned skeletons have ([0-9]+)% of physical damage converted to chaos damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("PhysicalDamageConvertToChaos", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Skeletons", "includeTransfigured": true})}
	}),
	`summoned skeletons gain added chaos damage equal to ([0-9]+)% of maximum energy shield on your equipped shield`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShieldOnWeapon 2", "actor": "parent", "percent": c.n(1)})}), mod("MinionModifier", "LIST", Tag{"mod": mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShieldOnWeapon 2", "actor": "parent", "percent": c.n(1)})})}
	}),
	`skeletons gain added chaos damage equal to ([0-9]+)% of maximum energy shield on your equipped shield`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShieldOnWeapon 2", "actor": "parent", "percent": c.n(1)})}), mod("MinionModifier", "LIST", Tag{"mod": mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShieldOnWeapon 2", "actor": "parent", "percent": c.n(1)})})}
	}),
	`minions convert ([0-9]+)% of (.+) damage to (.+) damage per (.+) socket`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1))}, Tag{"type": "Multiplier", "var": firstToUpper(c.s(4)) + "SocketIn{SlotName}"})}
	}),
	`minions have a ([0-9]+)% chance to impale on hit with attacks`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ImpaleChance", "BASE", c.n(1))})}
	}),
	`minions have ([0-9]+)% chance to impale on attack hit per socketed ghastly eye jewel`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ImpaleChance", "BASE", c.n(1))}, Tag{"type": "Multiplier", "var": "GhastlyEyeJewelIn{SlotName}"})}
	}),
	`minions from herald skills deal ([0-9]+)% more damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "MORE", c.n(1))}, Tag{"type": "SkillType", "skillType": SkillType.Herald})}
	}),
	`minions have ([0-9]+)% increased movement speed for each herald affecting you`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("MovementSpeed", "INC", c.n(1), Tag{"type": "Multiplier", "var": "Herald", "actor": "parent"})})}
	}),
	`minions deal ([0-9]+)% increased damage while you are affected by a herald`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "AffectedByHerald"})})}
	}),
	`minions have ([0-9]+)% increased attack and cast speed while you are affected by a herald`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Speed", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "parent", "var": "AffectedByHerald"})})}
	}),
	`minions have unholy might`: []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:UnholyMight")}), mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:CanWither")})},
	`summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?deal triple damage with this weapon if you've hit with this weapon recently`: []any{mod("MinionModifier", "LIST", Tag{"mod": mod("TripleDamageChance", "BASE", 100, Tag{"type": "ActorCondition", "actor": "parent", "var": "HitRecentlyWithWeapon"})}, Tag{"type": "SkillName", "skillName": "Summon Skeletons"}, Tag{"type": "ItemCondition", "itemSlot": "Weapon 1", "nameCond": "The Iron Mass, Gladius"})},
	`summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?wield a? ?c?o?p?y? ?o?f? ?this weapon while in your main hand`:               d(),
	`each summoned phantasm grants you phantasmal might`:                                                                               []any{flag("Condition:PhantasmalMight")},
	`minions have ([0-9]+)% increased critical strike chance per maximum power charge you have`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("CritChance", "INC", c.n(1), Tag{"type": "Multiplier", "actor": "parent", "var": "PowerChargeMax"})})}
	}),
	`minions' base attack critical strike chance is equal to the critical strike chance of your main hand weapon`: []any{mod("MinionModifier", "LIST", Tag{"mod": flag("AttackCritIsEqualToParentMainHand", nil, ModFlag.Attack)})},
	`non-spectre minions' base attack time is equal to the attack time of your main hand weapon`:                  []any{flag("NonSpectreMinionsUseParentMainHandAttackTime")},
	`minions can hear the whispers for 5 seconds after they deal a critical strike`:                               []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("Speed", "INC", 50, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"})}), mod("ExtraSkillMod", "LIST", Tag{"mod": mod("Damage", "INC", 50, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"})}), mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ChaosDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": 20}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "Condition", "neg": true, "var": "NeverCrit"})})},
	`chaos damage t?a?k?e?n? ?does not bypass minions' energy shield`:                                             []any{mod("MinionModifier", "LIST", Tag{"mod": flag("ChaosNotBypassEnergyShield")})},
	`while minions have energy shield, their hits ignore monster elemental resistances`:                           []any{mod("MinionModifier", "LIST", Tag{"mod": flag("IgnoreElementalResistances", Tag{"type": "StatThreshold", "stat": "EnergyShield", "threshold": 1})})},
	`summoned arbalists' projectiles pierce ([0-9]+) additional targets`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("PierceCount", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"})}
	}),
	`summoned arbalists' projectiles fork`: []any{mod("MinionModifier", "LIST", Tag{"mod": flag("ForkOnce")}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"}), mod("MinionModifier", "LIST", Tag{"mod": mod("ForkCountMax", "BASE", 1)}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"})},
	`summoned arbalists' projectiles chain \+([0-9]+) times`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ChainCountMax", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"})}
	}),
	`summoned arbalists have ([0-9]+)% chance to inflict (.+) exposure on hit`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(2))+"ExposureChance", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Summon Arbalists"})}
	}),
	`summoned arbalists convert ([0-9]+)% of (.+) damage to (.+) damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"})}
	}),
	`summoned arbalists have ([0-9]+)% chance to freeze, shock, and ignite`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("EnemyFreezeChance", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"}), mod("MinionModifier", "LIST", Tag{"mod": mod("EnemyShockChance", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"}), mod("MinionModifier", "LIST", Tag{"mod": mod("EnemyIgniteChance", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Summon Arbalists"})}
	}),
	`skeleton warriors are permanent minions and follow you`:  []any{flag("RaisedSkeletonPermanentDuration", Tag{"type": "SkillName", "skillName": "Summon Skeletons"})},
	`summoned skeleton warriors are permanent and follow you`: []any{flag("RaisedSkeletonPermanentDuration", Tag{"type": "SkillName", "skillName": "Summon Skeletons"})},
	`your blink and mirror arrow clones use your gloves`:      []any{flag("BlinkAndMirrorUseGloves")},
	// Projectiles
	`skills chain \+([0-9]) times`:                                    fn(func(c caps) any { return []any{mod("ChainCountMax", "BASE", c.n(1))} }),
	`projectiles chain an additional time`:                            []any{mod("ChainCountMax", "BASE", 1, nil, ModFlag.Projectile)},
	`projectiles chain \+([0-9]) times`:                               fn(func(c caps) any { return []any{mod("ChainCountMax", "BASE", c.n(1), nil, ModFlag.Projectile)} }),
	`arrows chain \+([0-9]) times`:                                    fn(func(c caps) any { return []any{mod("ChainCountMax", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow)} }),
	`skills chain an additional time while at maximum frenzy charges`: []any{mod("ChainCountMax", "BASE", 1, Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"})},
	`attacks chain an additional time when in main hand`:              []any{mod("ChainCountMax", "BASE", 1, nil, ModFlag.Attack, Tag{"type": "SlotNumber", "num": 1})},
	`attacks fire an additional projectile when in off hand`:          []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Attack, Tag{"type": "SlotNumber", "num": 2})},
	`projectiles chain \+([0-9]) times while you have phasing`: fn(func(c caps) any {
		return []any{mod("ChainCountMax", "BASE", c.n(1), nil, ModFlag.Projectile, Tag{"type": "Condition", "var": "Phasing"})}
	}),
	`projectiles can chain from any number of additional targets in close range`: []any{mod("ChainCountMax", "BASE", m_huge, nil, ModFlag.Projectile, Tag{"type": "Condition", "var": "AtCloseRange"})},
	`projectiles split towards \+([0-9]) targets`:                                fn(func(c caps) any { return []any{mod("SplitCount", "BASE", c.n(1))} }),
	`adds an additional arrow`:                                                   []any{mod("ProjectileCount", "BASE", 1, nil, 0, KeywordFlag.Arrow)},
	`([0-9]+) additional arrows`:                                                 fn(func(c caps) any { return []any{mod("ProjectileCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow)} }),
	`bow attacks fire an additional arrow`:                                       []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Bow)},
	`bow attacks fire ([0-9]+) additional arrows`:                                fn(func(c caps) any { return []any{mod("ProjectileCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow)} }),
	`bow attacks fire ([0-9]+) additional arrows if you haven't cast dash recently`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow, Tag{"type": "Condition", "var": "CastDashRecently", "neg": true})}
	}),
	`wand attacks fire an additional projectile`: []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Wand)},
	`skills fire an additional projectile`:       []any{mod("ProjectileCount", "BASE", 1)},
	`skills fire an additional projectile if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "BASE", 1, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "Item", "threshold": c.n(1)})}
	}),
	`spells [hf][ai][vr]e an additional projectile`:         []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Spell)},
	`spells [hf][ai][vr]e ([0-9]+) additional projectiles`:  fn(func(c caps) any { return []any{mod("ProjectileCount", "BASE", c.n(1), nil, ModFlag.Spell)} }),
	`attacks fire an additional projectile`:                 []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Attack)},
	`attacks [hf][ai][vr]e ([0-9]+) additional projectiles`: fn(func(c caps) any { return []any{mod("ProjectileCount", "BASE", c.n(1), nil, ModFlag.Attack)} }),
	`attacks [hf][ai][vr]e ([0-9]+) additional projectiles? when in off hand`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "BASE", c.n(1), nil, ModFlag.Attack, Tag{"type": "SlotNumber", "num": 2})}
	}),
	`fire at most 1 projectile`:                              []any{flag("SingleProjectile")},
	`attacks have an additional projectile when in off hand`: []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Attack, Tag{"type": "SlotNumber", "num": 2})},
	`caustic arrow and scourge arrow fire ([0-9]+)% more projectiles`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "MORE", c.n(1), nil, Tag{"type": "SkillName", "skillNameList": []any{"Caustic Arrow", "Scourge Arrow"}, "includeTransfigured": true})}
	}),
	`essence drain and soulrend fire ([0-9]+) additional projectiles`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "BASE", c.n(1), nil, Tag{"type": "SkillName", "skillNameList": []any{"Essence Drain", "Soulrend"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% reduced essence drain and soulrend projectile speed`: fn(func(c caps) any {
		return []any{mod("ProjectileSpeed", "INC", -c.n(1), nil, Tag{"type": "SkillName", "skillNameList": []any{"Essence Drain", "Soulrend"}, "includeTransfigured": true})}
	}),
	`tornado shot fires an additional secondary projectile`: []any{mod("tornadoShotSecondaryProjectiles", "BASE", 1, nil, Tag{"type": "SkillName", "skillNameList": []any{"Tornado Shot"}})},
	`tornado shot fires 2 additional secondary projectiles`: []any{mod("tornadoShotSecondaryProjectiles", "BASE", 2, nil, Tag{"type": "SkillName", "skillNameList": []any{"Tornado Shot"}})},
	`projectiles pierce an additional target`:               []any{mod("PierceCount", "BASE", 1)},
	`projectiles pierce ([0-9]+) targets?`:                  fn(func(c caps) any { return []any{mod("PierceCount", "BASE", c.n(1))} }),
	`projectiles pierce ([0-9]+) additional targets?`:       fn(func(c caps) any { return []any{mod("PierceCount", "BASE", c.n(1))} }),
	`projectiles pierce ([0-9]+) additional targets while you have phasing`: fn(func(c caps) any {
		return []any{mod("PierceCount", "BASE", c.n(1), Tag{"type": "Condition", "var": "Phasing"})}
	}),
	`projectiles pierce ([0-9]+) additional targets if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("PierceCount", "BASE", c.s(1), Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)})}
	}),
	`projectiles pierce all targets while you have phasing`: []any{flag("PierceAllTargets", Tag{"type": "Condition", "var": "Phasing"})},
	`projectiles pierce all burning enemies`:                []any{flag("PierceAllTargets", Tag{"type": "ActorCondition", "actor": "enemy", "var": "Burning"})},
	`arrows pierce an additional target`:                    []any{mod("PierceCount", "BASE", 1, nil, 0, KeywordFlag.Arrow)},
	`arrows pierce ([0-9]+) additional targets`:             fn(func(c caps) any { return []any{mod("PierceCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow)} }),
	`arrows pierce one target`:                              []any{mod("PierceCount", "BASE", 1, nil, 0, KeywordFlag.Arrow)},
	`arrows pierce ([0-9]+) targets?`:                       fn(func(c caps) any { return []any{mod("PierceCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow)} }),
	`always pierce with arrows`:                             []any{flag("PierceAllTargets", nil, 0, KeywordFlag.Arrow)},
	`arrows always pierce`:                                  []any{flag("PierceAllTargets", nil, 0, KeywordFlag.Arrow)},
	`arrows pierce all targets`:                             []any{flag("PierceAllTargets", nil, 0, KeywordFlag.Arrow)},
	`arrows that pierce cause bleeding`:                     []any{mod("BleedChance", "BASE", 100, nil, ModFlag.Projectile, KeywordFlag.Arrow, Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1})},
	`arrows that pierce have ([0-9]+)% chance to [ic][na][fu][ls][ie]c?t? bleeding`: fn(func(c caps) any {
		return []any{mod("BleedChance", "BASE", c.n(1), nil, ModFlag.Projectile, KeywordFlag.Arrow, Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1})}
	}),
	`arrows that pierce deal ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Projectile, KeywordFlag.Arrow, Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1})}
	}),
	`projectiles gain ([0-9]+)% of non-chaos damage as extra chaos damage per chain`: fn(func(c caps) any {
		return []any{mod("NonChaosDamageGainAsChaos", "BASE", c.n(1), nil, ModFlag.Projectile, Tag{"type": "PerStat", "stat": "Chain"})}
	}),
	`projectiles that have chained gain ([0-9]+)% of non-chaos damage as extra chaos damage`: fn(func(c caps) any {
		return []any{mod("NonChaosDamageGainAsChaos", "BASE", c.n(1), nil, ModFlag.Projectile, Tag{"type": "StatThreshold", "stat": "Chain", "threshold": 1})}
	}),
	`left ring slot: projectiles from spells cannot chain`:     []any{flag("CannotChain", nil, ModFlag.Spell|ModFlag.Projectile, Tag{"type": "SlotNumber", "num": 1})},
	`left ring slot: projectiles from spells fork`:             []any{flag("ForkOnce", nil, ModFlag.Spell|ModFlag.Projectile, Tag{"type": "SlotNumber", "num": 1}), mod("ForkCountMax", "BASE", 1, nil, ModFlag.Spell|ModFlag.Projectile, Tag{"type": "SlotNumber", "num": 1})},
	`right ring slot: projectiles from spells chain \+1 times`: []any{mod("ChainCountMax", "BASE", 1, nil, ModFlag.Spell|ModFlag.Projectile, Tag{"type": "SlotNumber", "num": 2})},
	`right ring slot: projectiles from spells cannot fork`:     []any{flag("CannotFork", nil, ModFlag.Spell|ModFlag.Projectile, Tag{"type": "SlotNumber", "num": 2})},
	`projectiles from spells cannot pierce`:                    []any{flag("CannotPierce", nil, ModFlag.Spell)},
	// The chance to return is ignored; the number of returning Projectiles that hit is a config option
	`projectiles return to you`:                          []any{flag("ProjectilesReturn")},
	`projectiles have ([0-9]+)% chance to return to you`: []any{flag("ProjectilesReturn")},
	`projectiles fork`:                                   []any{flag("ForkOnce", nil, ModFlag.Projectile), mod("ForkCountMax", "BASE", 1, nil, ModFlag.Projectile)},
	`projectiles from attacks fork`:                      []any{flag("ForkOnce", nil, ModFlag.Projectile, Tag{"type": "SkillType", "skillType": SkillType.RangedAttack}), mod("ForkCountMax", "BASE", 1, nil, ModFlag.Projectile, Tag{"type": "SkillType", "skillType": SkillType.RangedAttack})},
	`projectiles from attacks fork an additional time`:   []any{flag("ForkTwice", nil, ModFlag.Projectile, Tag{"type": "SkillType", "skillType": SkillType.RangedAttack}), mod("ForkCountMax", "BASE", 1, nil, ModFlag.Projectile, Tag{"type": "SkillType", "skillType": SkillType.RangedAttack})},
	`projectiles from attacks can fork ([0-9]+) additional times?`: fn(func(c caps) any {
		return []any{flag("ForkTwice", nil, ModFlag.Projectile, Tag{"type": "SkillType", "skillType": SkillType.RangedAttack}), mod("ForkCountMax", "BASE", c.n(1), nil, ModFlag.Projectile, Tag{"type": "SkillType", "skillType": SkillType.RangedAttack})}
	}),
	`([0-9]+)% increased critical strike chance with arrows that fork`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), nil, 0, KeywordFlag.Arrow, Tag{"type": "StatThreshold", "stat": "ForkRemaining", "threshold": 1}, Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 0, "upper": true})}
	}),
	`arrows that pierce have \+([0-9]+)% to critical strike multiplier`: fn(func(c caps) any {
		return []any{mod("CritMultiplier", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow, Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1})}
	}),
	`arrows pierce all targets after forking`:                                                             []any{flag("PierceAllTargets", nil, 0, KeywordFlag.Arrow, Tag{"type": "StatThreshold", "stat": "ForkedCount", "threshold": 1})},
	`modifiers to number of projectiles instead apply to the number of targets projectiles split towards`: []any{flag("NoAdditionalProjectiles"), flag("AdditionalProjectilesAddSplitsInstead")},
	`modifiers to number of projectiles do not apply to fireball and rolling magma`:                       []any{flag("NoAdditionalProjectiles", Tag{"type": "SkillName", "skillNameList": []any{"Fireball", "Rolling Magma"}})},
	`attack skills fire an additional projectile while wielding a claw or dagger`:                         []any{mod("ProjectileCount", "BASE", 1, nil, ModFlag.Attack, Tag{"type": "ModFlagOr", "modFlags": ModFlag.Claw | ModFlag.Dagger})},
	`skills fire ([0-9]+) additional projectiles for 4 seconds after you consume a total of 12 steel shards`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "BASE", c.n(1), Tag{"type": "Condition", "var": "Consumed12SteelShardsRecently"})}
	}),
	`bow attacks sacrifice a random damageable minion to fire ([0-9]+) additional arrows?`: fn(func(c caps) any {
		return []any{mod("ProjectileCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow, Tag{"type": "Condition", "var": "SacrificeMinionOnAttack"}, Tag{"type": "Condition", "var": "HaveDamageableMinion"}, Tag{"type": "SkillType", "skillType": SkillType.Triggered, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.SummonsTotem, "neg": true})}
	}),
	`non-projectile chaining lightning skills chain \+([0-9]+) times`: fn(func(c caps) any {
		return []any{mod("ChainCountMax", "BASE", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Projectile, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Chains}, Tag{"type": "SkillType", "skillType": SkillType.Lightning})}
	}),
	`arrows gain damage as they travel farther, dealing up to ([0-9]+)% increased damage with hits to targets`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Hit, KeywordFlag.Arrow, Tag{"type": "DistanceRamp", "ramp": []any{[]any{35, 0}, []any{70, 1}}})}
	}),
	`arrows gain critical strike chance as they travel farther, up to ([0-9]+)% increased critical strike chance`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), nil, 0, KeywordFlag.Arrow, Tag{"type": "DistanceRamp", "ramp": []any{[]any{35, 0}, []any{70, 1}}})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits to targets at the start of their movement, reducing to ([0-9]+)% as they travel farther`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Hit|ModFlag.Projectile, Tag{"type": "DistanceRamp", "ramp": []any{[]any{35, 1}, []any{70, 0}}})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits and ailments for each time they have chained`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "PerStat", "stat": "Chain"}, Tag{"type": "SkillType", "skillType": SkillType.Projectile})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits and ailments for each enemy pierced`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, 0, KeywordFlag.Hit|KeywordFlag.Ailment, Tag{"type": "PerStat", "stat": "PiercedCount"}, Tag{"type": "SkillType", "skillType": SkillType.Projectile})}
	}),
	`([0-9]+)% increased bonuses gained from equipped (.+)`: fn(func(c caps) any { return []any{mod("EffectOfBonusesFrom"+firstToUpper(c.s(2)), "INC", c.n(1))} }),
	// Strike Skills
	`non-vaal strike skills target ([0-9]+) additional nearby enem[yi]e?s?`: fn(func(c caps) any {
		return []any{mod("AdditionalStrikeTarget", "BASE", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.MeleeSingleTarget}, Tag{"type": "SkillType", "skillType": SkillType.Vaal, "neg": true})}
	}),
	// Leech/Gain on Hit/Kill
	`cannot leech life`:                                                                       []any{flag("CannotLeechLife")},
	`cannot leech mana`:                                                                       []any{flag("CannotLeechMana")},
	`cannot leech when on low life`:                                                           []any{flag("CannotLeechLife", Tag{"type": "Condition", "var": "LowLife"}), flag("CannotLeechMana", Tag{"type": "Condition", "var": "LowLife"})},
	`cannot leech life from critical strikes`:                                                 []any{flag("CannotLeechLife", Tag{"type": "Condition", "var": "CriticalStrike"})},
	`leech applies instantly on critical strike`:                                              []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"}), mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"}), mod("InstantEnergyShieldLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`gain life and mana from leech instantly on critical strike`:                              []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"}), mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"}), mod("InstantEnergyShieldLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`leech applies instantly during f?l?a?s?k? ?effect`:                                       []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"}), mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"}), mod("InstantEnergyShieldLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"})},
	`gain life and mana from leech instantly during f?l?a?s?k? ?effect`:                       []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"}), mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"})},
	`life and mana leech are instant during f?l?a?s?k? ?effect`:                               []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"}), mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"})},
	`life and mana leech from critical strikes are instant`:                                   []any{mod("InstantLifeLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"}), mod("InstantManaLeech", "BASE", 100, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`with 5 corrupted items equipped: life leech recovers based on your chaos damage instead`: []any{flag("LifeLeechBasedOnChaosDamage", Tag{"type": "MultiplierThreshold", "var": "CorruptedItem", "threshold": 5})},
	`you have vaal pact if you've dealt a critical strike recently`:                           []any{mod("Keystone", "LIST", "Vaal Pact", Tag{"type": "Condition", "var": "CritRecently"})},
	`you have vaal pact while at maximum endurance charges`:                                   []any{mod("Keystone", "LIST", "Vaal Pact", Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMax"})},
	`you have vaal pact while focus?sed`:                                                      []any{mod("Keystone", "LIST", "Vaal Pact", Tag{"type": "Condition", "var": "Focused"})},
	`gain ([0-9]+) energy shield for each enemy you hit which is affected by a spider's web`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnHit", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "Spider's WebStack", "threshold": 1})}
	}),
	`([0-9]+)% chance to gain ([0-9]+) life on hit with attacks`: fn(func(c caps) any {
		return []any{mod("LifeOnHit", "BASE", c.n(2)*c.n(1)/100, nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "Condition", "var": "AverageResourceGain"}), mod("LifeOnHit", "BASE", c.n(2), nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "Condition", "var": "MaxResourceGain"})}
	}),
	`([0-9]+) life gained for each cursed enemy hit by your attacks`: fn(func(c caps) any {
		return []any{mod("LifeOnHit", "BASE", c.n(1), nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`gain ([0-9]+) life per cursed enemy hit with attacks`: fn(func(c caps) any {
		return []any{mod("LifeOnHit", "BASE", c.n(1), nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`gain ([0-9]+) life for each ignited enemy hit with attacks`: fn(func(c caps) any {
		return []any{mod("LifeOnHit", "BASE", c.n(1), nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"})}
	}),
	`([0-9]+) mana gained for each cursed enemy hit by your attacks`: fn(func(c caps) any {
		return []any{mod("ManaOnHit", "BASE", c.n(1), nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`gain ([0-9]+) mana per cursed enemy hit with attacks`: fn(func(c caps) any {
		return []any{mod("ManaOnHit", "BASE", c.n(1), nil, ModFlag.Attack|ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`gain ([0-9]+) life per blinded enemy hit with this weapon`: fn(func(c caps) any {
		return []any{mod("LifeOnHit", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Blinded"}, Tag{"type": "Condition", "var": "{Hand}Attack"})}
	}),
	`recover ([0-9]+)% of life on kill`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}
	}),
	`recover ([0-9]+)% of life on kill for each different type of mastery you have allocated`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "AllocatedMasteryType"})}
	}),
	`recover ([0-9]+)% of life on killing a poisoned enemy`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "PoisonStack", "threshold": 1})}
	}),
	`recover ([0-9]+)% of life on killing a chilled enemy`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"})}
	}),
	`recover ([0-9]+)% of life when you kill a cursed enemy`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`recover ([0-9]+)% of life per withered debuff on each enemy you kill`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "WitheredStack", "actor": "enemy", "limit": 15})}
	}),
	`minions recover ([0-9]+)% of life on killing a poisoned enemy`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "PoisonStack", "threshold": 1})})}
	}),
	`minions recover ([0-9]+)% of their life when they block`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("LifeOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})})}
	}),
	`recover ([0-9]+)% of mana when you kill a cursed enemy`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`recover ([0-9]+)% of energy shield when you kill a cursed enemy`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`recover ([0-9]+)% of life on kill if you've spent life recently`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "MultiplierThreshold", "var": "LifeSpentRecently", "threshold": 1})}
	}),
	`([0-9]+)% chance to recover all life when you kill an enemy`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Condition", "var": "AverageResourceGain"}), mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": 100}, Tag{"type": "Condition", "var": "MaxResourceGain"})}
	}),
	`lose ([0-9]+)% of life on kill`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", -1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}
	}),
	`\+([0-9]+) life gained on killing ignited enemies`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"})}
	}),
	`gain ([0-9]+) life per ignited enemy killed`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"})}
	}),
	`recover ([0-9]+)% of mana on kill`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)})}
	}),
	`recover ([0-9]+)% of mana on kill while you have a tincture active`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "Condition", "var": "UsingTincture"})}
	}),
	`recover ([0-9]+)% of mana on kill for each different type of mastery you have allocated`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "AllocatedMasteryType"})}
	}),
	`lose ([0-9]+)% of mana on kill`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", -1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)})}
	}),
	`\+([0-9]+) mana gained on killing a frozen enemy`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"})}
	}),
	`recover ([0-9]+)% of energy shield on kill`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)})}
	}),
	`recover ([0-9]+)% of energy shield on kill for each different type of mastery you have allocated`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "AllocatedMasteryType"})}
	}),
	`lose ([0-9]+)% of energy shield on kill`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", -1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)})}
	}),
	`\+([0-9]+) energy shield gained on killing a shocked enemy`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"})}
	}),
	`\+([0-9]+) energy shield gained on kill per level`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "Level"})}
	}),
	// Defences
	`chaos damage t?a?k?e?n? ?does not bypass energy shield`:                                   []any{flag("ChaosNotBypassEnergyShield")},
	`([0-9]+)% of chaos damage t?a?k?e?n? ?does not bypass energy shield`:                      fn(func(c caps) any { return []any{mod("ChaosEnergyShieldBypass", "BASE", -c.n(1))} }),
	`chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life`:             []any{flag("ChaosNotBypassEnergyShield", Tag{"type": "Condition", "varList": []any{"LowLife"}, "neg": true})},
	`chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life or low mana`: []any{flag("ChaosNotBypassEnergyShield", Tag{"type": "Condition", "varList": []any{"LowLife", "LowMana"}, "neg": true})},
	`chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low mana`:             []any{flag("ChaosNotBypassEnergyShield", Tag{"type": "Condition", "varList": []any{"LowMana"}, "neg": true})},
	`chaos damage is taken from mana before life`:                                              []any{mod("ChaosDamageTakenFromManaBeforeLife", "BASE", 100)},
	`([0-9]+)% of physical damage is taken from mana before life`:                              fn(func(c caps) any { return []any{mod("PhysicalDamageTakenFromManaBeforeLife", "BASE", c.n(1))} }),
	`minions take ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1))})}
	}),
	`minions take ([0-9]+)% reduced damage`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", -c.n(1))})}
	}),
	`you and your minions take ([0-9]+)% reduced reflected ([0-9a-zA-Z]+) damage`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(2))+"ReflectedDamageTaken", "INC", -c.n(1), Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}), mod("MinionModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(2))+"ReflectedDamageTaken", "INC", -c.n(1), Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})})}
	}),
	`([0-9]+)% reduced reflected damage taken during effect`: fn(func(c caps) any {
		return []any{mod("ReflectedDamageTaken", "INC", -c.n(1), Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})}
	}),
	`you and your minions take ([0-9]+)% reduced reflected damage`: fn(func(c caps) any {
		return []any{mod("ReflectedDamageTaken", "INC", -c.n(1), Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}), mod("MinionModifier", "LIST", Tag{"mod": mod("ReflectedDamageTaken", "INC", -c.n(1), Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})})}
	}),
	`damage cannot be reflected`: []any{mod("ReflectedDamageTaken", "MORE", -100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`immune to reflected damage`: []any{mod("ReflectedDamageTaken", "MORE", -100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`cannot take reflected ([a-zA-Z]+) damage if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod(firstToUpper(c.s(1))+"ReflectedDamageTaken", "MORE", -100, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)})}
	}),
	`you have mind over matter while at maximum power charges`:            []any{mod("Keystone", "LIST", "Mind Over Matter", Tag{"type": "StatThreshold", "stat": "PowerCharges", "thresholdStat": "PowerChargesMax"})},
	`cannot evade enemy attacks`:                                          []any{flag("CannotEvade")},
	`attacks cannot hit you`:                                              []any{flag("AlwaysEvade")},
	`attacks against you always hit`:                                      []any{flag("CannotEvade")},
	`cannot block`:                                                        []any{flag("CannotBlockAttacks"), flag("CannotBlockSpells")},
	`cannot block while you have no energy shield`:                        []any{flag("CannotBlockAttacks", Tag{"type": "Condition", "var": "HaveEnergyShield", "neg": true}), flag("CannotBlockSpells", Tag{"type": "Condition", "var": "HaveEnergyShield", "neg": true})},
	`cannot block attacks`:                                                []any{flag("CannotBlockAttacks")},
	`cannot block attack damage`:                                          []any{flag("CannotBlockAttacks")},
	`cannot block spells`:                                                 []any{flag("CannotBlockSpells")},
	`cannot block spell damage`:                                           []any{flag("CannotBlockSpells")},
	`monsters cannot block your attacks`:                                  []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("CannotBlockAttacks")})},
	`damage t?a?k?e?n? from blocked hits cannot bypass energy shield`:     []any{flag("BlockedDamageDoesntBypassES", Tag{"type": "Condition", "var": "EVBypass", "neg": true})},
	`damage t?a?k?e?n? from unblocked hits always bypasses energy shield`: []any{flag("UnblockedDamageDoesBypassES", Tag{"type": "Condition", "var": "EVBypass", "neg": true})},
	`recover ([0-9]+) life when you block`:                                fn(func(c caps) any { return []any{mod("LifeOnBlock", "BASE", c.n(1))} }),
	`recover ([0-9]+) energy shield when you block spell damage`:          fn(func(c caps) any { return []any{mod("EnergyShieldOnSpellBlock", "BASE", c.n(1))} }),
	`recover ([0-9]+) energy shield when you suppress spell damage`:       fn(func(c caps) any { return []any{mod("EnergyShieldOnSuppress", "BASE", c.n(1))} }),
	`recover ([0-9]+) life when you suppress spell damage`:                fn(func(c caps) any { return []any{mod("LifeOnSuppress", "BASE", c.n(1))} }),
	`recover ([0-9]+)% of life when you block`: fn(func(c caps) any {
		return []any{mod("LifeOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}
	}),
	`recover ([0-9]+)% of life when you block attack damage while wielding a staff`: fn(func(c caps) any {
		return []any{mod("LifeOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Condition", "var": "UsingStaff"})}
	}),
	`recover ([0-9]+)% of your maximum mana when you block`: fn(func(c caps) any {
		return []any{mod("ManaOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)})}
	}),
	`recover ([0-9]+)% of energy shield when you block`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)})}
	}),
	`recover ([0-9]+)% of energy shield when you block spell damage while wielding a staff`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnSpellBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Condition", "var": "UsingStaff"})}
	}),
	`replenishes energy shield by ([0-9]+)% of armour when you block`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Armour", "percent": c.n(1)})}
	}),
	`recover energy shield equal to ([0-9]+)% of armour when you block`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Armour", "percent": c.n(1)})}
	}),
	`recover energy shield equal to ([0-9]+)% of evasion rating when you block`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnBlock", "BASE", 1, Tag{"type": "PercentStat", "stat": "Evasion", "percent": c.n(1)})}
	}),
	`([0-9]+)% of damage taken while affected by clarity recouped as mana`: fn(func(c caps) any {
		return []any{mod("ManaRecoup", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByClarity"})}
	}),
	`([0-9]+)% of damage taken while frozen recouped as life`: fn(func(c caps) any {
		return []any{mod("LifeRecoup", "BASE", c.n(1), Tag{"type": "Condition", "var": "Frozen"})}
	}),
	`recoup effects instead occur over 3 seconds`:                            []any{flag("3SecondRecoup")},
	`life recoup effects instead occur over 3 seconds`:                       []any{flag("3SecondLifeRecoup")},
	`recoup energy shield instead of life`:                                   []any{flag("EnergyShieldRecoupInsteadOfLife")},
	`damage taken recouped as ([a-zA-Z]+) is also recouped as energy shield`: fn(func(c caps) any { return []any{flag("Add" + firstToUpper(c.s(1)) + "RecoupToEnergyShieldRecoup")} }),
	`([0-9.]+)% of physical damage prevented from hits in the past ([0-9]+) seconds is regenerated as life per second`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageMitigatedLifePseudoRecoup", "BASE", c.n(1)*c.n(2)), mod("PhysicalDamageMitigatedLifePseudoRecoupDuration", "BASE", c.s(2))}
	}),
	`([0-9.]+)% of physical damage prevented from hits recently is regenerated as energy shield per second`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", "BASE", c.n(1)*4)}
	}),
	`([0-9.]+)% of physical damage prevented recently is regenerated as energy shield per second if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", "BASE", c.n(1)*4, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)})}
	}),
	`cannot leech or regenerate mana`:                                 []any{flag("NoManaRegen"), flag("CannotLeechMana")},
	`y?o?u? ?cannot recharge energy shield`:                           []any{flag("NoEnergyShieldRecharge")},
	`cannot recharge or regenerate energy shield`:                     []any{flag("NoEnergyShieldRecharge"), flag("NoEnergyShieldRegen")},
	`left ring slot: you cannot recharge or regenerate energy shield`: []any{flag("NoEnergyShieldRecharge", Tag{"type": "SlotNumber", "num": 1}), flag("NoEnergyShieldRegen", Tag{"type": "SlotNumber", "num": 1})},
	`cannot gain energy shield`:                                       []any{flag("CannotGainEnergyShield")},
	`cannot gain life`:                                                []any{flag("CannotGainLife")},
	`cannot gain mana`:                                                []any{flag("CannotGainMana")},
	`cannot recover life other than from leech`:                       []any{flag("CannotRecoverLifeOutsideLeech"), flag("NoLifeRegen")},
	`cannot gain energy shield during f?l?a?s?k? ?effect`:             []any{flag("CannotGainEnergyShield", Tag{"type": "Condition", "var": "UsingFlask"})},
	`cannot gain life during f?l?a?s?k? ?effect`:                      []any{flag("CannotGainLife", Tag{"type": "Condition", "var": "UsingFlask"})},
	`cannot gain mana during f?l?a?s?k? ?effect`:                      []any{flag("CannotGainMana", Tag{"type": "Condition", "var": "UsingFlask"})},
	`life that would be lost by taking damage is instead reserved`:    []any{flag("DamageInsteadReservesLife")},
	`you have no armour or energy shield`:                             []any{mod("Armour", "MORE", -100), mod("EnergyShield", "MORE", -100)},
	`you have no armour or maximum energy shield`:                     []any{mod("Armour", "MORE", -100), mod("EnergyShield", "MORE", -100)},
	`defences are zero`:                                               []any{mod("Armour", "MORE", -100), mod("EnergyShield", "MORE", -100), mod("Evasion", "MORE", -100), mod("Ward", "MORE", -100)},
	`you have no intelligence`:                                        []any{mod("Int", "MORE", -100)},
	`you have no dexterity`:                                           []any{mod("Dex", "MORE", -100)},
	`you have no strength`:                                            []any{mod("Str", "MORE", -100)},
	`physical damage reduction is zero`:                               []any{mod("PhysicalDamageReduction", "OVERRIDE", 0), flag("ArmourDoesNotApplyToPhysicalDamageTaken")},
	`elemental resistances are zero`:                                  []any{mod("FireResist", "OVERRIDE", 0), mod("ColdResist", "OVERRIDE", 0), mod("LightningResist", "OVERRIDE", 0)},
	`chaos resistance is zero`:                                        []any{mod("ChaosResist", "OVERRIDE", 0)},
	`nearby enemies' chaos resistance is ([0-9]+)`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("ChaosResist", "OVERRIDE", c.n(1))})}
	}),
	`your maximum resistances are ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("FireResistMax", "OVERRIDE", c.n(1)), mod("ColdResistMax", "OVERRIDE", c.n(1)), mod("LightningResistMax", "OVERRIDE", c.n(1)), mod("ChaosResistMax", "OVERRIDE", c.n(1))}
	}),
	`fire resistance is ([0-9]+)%`:      fn(func(c caps) any { return []any{mod("FireResist", "OVERRIDE", c.n(1))} }),
	`cold resistance is ([0-9]+)%`:      fn(func(c caps) any { return []any{mod("ColdResist", "OVERRIDE", c.n(1))} }),
	`lightning resistance is ([0-9]+)%`: fn(func(c caps) any { return []any{mod("LightningResist", "OVERRIDE", c.n(1))} }),
	`elemental resistances are capped by your highest maximum elemental resistance instead`: []any{flag("ElementalResistMaxIsHighestResistMax")},
	`nearby enemies have ([0-9]+)% increased fire and cold resistances`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("FireResist", "INC", c.n(1))}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdResist", "INC", c.n(1))})}
	}),
	`nearby enemies are blinded while physical aegis is not depleted`:    []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Blinded")}, Tag{"type": "Condition", "var": "PhysicalAegisDepleted", "neg": true})},
	`maximum energy shield is increased by chance to block spell damage`: []any{flag("EnergyShieldIncreasedByChanceToBlockSpellDamage")},
	`maximum energy shield is increased by chaos resistance`:             []any{flag("EnergyShieldIncreasedByChaosResistance")},
	`armour is increased by uncapped fire resistance`:                    []any{flag("ArmourIncreasedByUncappedFireRes")},
	`armour is increased by overcapped fire resistance`:                  []any{flag("ArmourIncreasedByOvercappedFireRes")},
	`minion life is increased by t?h?e?i?r? ?overcapped fire resistance`: []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Life", "INC", 1, Tag{"type": "PerStat", "stat": "FireResistOverCap", "div": 1})})},
	`totem life is increased by t?h?e?i?r? ?overcapped fire resistance`:  []any{mod("TotemLife", "INC", 1, Tag{"type": "PerStat", "stat": "TotemFireResistOverCap", "div": 1})},
	`evasion rating is increased by uncapped cold resistance`:            []any{flag("EvasionRatingIncreasedByUncappedColdRes")},
	`evasion rating is increased by overcapped cold resistance`:          []any{flag("EvasionRatingIncreasedByOvercappedColdRes")},
	`reflects ([0-9]+) physical damage to melee attackers`:               d(),
	`ignore all movement penalties from armour`:                          []any{flag("Condition:IgnoreMovementPenalties")},
	`gain armour equal to your reserved mana`:                            []any{mod("Armour", "BASE", 1, Tag{"type": "PerStat", "stat": "ManaReserved", "div": 1})},
	`gain ward instead of ([0-9]+)% of armour and evasion rating from equipped body armour`: fn(func(c caps) any {
		return []any{flag("ConvertBodyArmourArmourEvasionToWard"), mod("BodyArmourArmourEvasionToWardPercent", "BASE", c.n(1))}
	}),
	`([0-9]+)% increased armour per ([0-9]+) reserved mana`: fn(func(c caps) any {
		return []any{mod("Armour", "INC", c.n(1), Tag{"type": "PerStat", "stat": "ManaReserved", "div": c.n(2)})}
	}),
	`cannot be stunned`:                                  []any{flag("StunImmune")},
	`cannot be stunned while bleeding`:                   []any{flag("StunImmune", Tag{"type": "Condition", "var": "Bleeding"})},
	`cannot be stunned when on low life`:                 []any{flag("StunImmune", Tag{"type": "Condition", "var": "LowLife"})},
	`cannot be stunned if you haven't been hit recently`: []any{flag("StunImmune", Tag{"type": "Condition", "var": "BeenHitRecently", "neg": true})},
	`cannot be stunned if you have at least ([0-9]+) crab barriers`: fn(func(c caps) any {
		return []any{flag("StunImmune", Tag{"type": "StatThreshold", "stat": "CrabBarriers", "threshold": c.n(1)})}
	}),
	`cannot be stunned if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{flag("StunImmune", Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "Item", "threshold": c.n(1)})}
	}),
	`cannot be blinded`:                             []any{flag("Condition:CannotBeBlinded"), flag("BlindImmune")},
	`cannot be blinded while affected by precision`: []any{flag("Condition:CannotBeBlinded", Tag{"type": "Condition", "var": "AffectedByPrecision"}), flag("BlindImmune", Tag{"type": "Condition", "var": "AffectedByPrecision"})},
	`cannot be knocked back`:                        []any{flag("KnockbackImmune")},
	`cannot be shocked`:                             []any{flag("ShockImmune")},
	`immun[ei]t?y? to shock`:                        []any{flag("ShockImmune")},
	`cannot be frozen`:                              []any{flag("FreezeImmune")},
	`cannot be frozen if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{flag("FreezeImmune", Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "Item", "threshold": c.n(1)})}
	}),
	`immun[ei]t?y? to freeze`:                                                                               []any{flag("FreezeImmune")},
	`cannot be chilled`:                                                                                     []any{flag("ChillImmune")},
	`immun[ei]t?y? to chill`:                                                                                []any{flag("ChillImmune")},
	`cannot be ignited`:                                                                                     []any{flag("IgniteImmune")},
	`immun[ei]t?y? to ignite`:                                                                               []any{flag("IgniteImmune")},
	`immune to ignite while affected by purity of fire`:                                                     []any{flag("IgniteImmune", Tag{"type": "Condition", "var": "AffectedByPurityofFire"})},
	`immune to freeze while affected by purity of ice`:                                                      []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "AffectedByPurityofIce"})},
	`immune to shock while affected by purity of lightning`:                                                 []any{flag("ShockImmune", Tag{"type": "Condition", "var": "AffectedByPurityofLightning"})},
	`critical strikes against you do not inherently inflict elemental ailments`:                             []any{flag("CritsOnYouDontAlwaysApplyElementalAilments")},
	`cannot be ignited while at maximum endurance charges`:                                                  []any{flag("IgniteImmune", Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMax"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`grants immunity to ignite for ([0-9]+) seconds if used while ignited`:                                  []any{flag("IgniteImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants immunity to bleeding for ([0-9]+) seconds if used while bleeding`:                               []any{flag("BleedImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants immunity to corrupted blood for ([0-9]+) seconds if used while affected by corrupted blood`:     []any{flag("CorruptedBloodImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants immunity to poison for ([0-9]+) seconds if used while poisoned`:                                 []any{flag("PoisonImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants immunity to freeze for ([0-9]+) seconds if used while frozen`:                                   []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants immunity to chill for ([0-9]+) seconds if used while chilled`:                                   []any{flag("ChillImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants immunity to shock for ([0-9]+) seconds if used while shocked`:                                   []any{flag("ShockImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`cannot be chilled while at maximum frenzy charges`:                                                     []any{flag("ChillImmune", Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"})},
	`cannot be shocked while at maximum power charges`:                                                      []any{flag("ShockImmune", Tag{"type": "StatThreshold", "stat": "PowerCharges", "thresholdStat": "PowerChargesMax"})},
	`you cannot be shocked while at maximum endurance charges`:                                              []any{flag("ShockImmune", Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMax"})},
	`you cannot be shocked while chilled`:                                                                   []any{flag("ShockImmune", Tag{"type": "Condition", "var": "Chilled"})},
	`you cannot be shocked while frozen`:                                                                    []any{flag("ShockImmune", Tag{"type": "Condition", "var": "Frozen"})},
	`cannot be shocked while chilled`:                                                                       []any{flag("ShockImmune", Tag{"type": "Condition", "var": "Chilled"})},
	`cannot be shocked if intelligence is higher than strength`:                                             []any{flag("ShockImmune", Tag{"type": "Condition", "var": "IntHigherThanStr"})},
	`cannot be frozen if dexterity is higher than intelligence`:                                             []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "DexHigherThanInt"})},
	`cannot be frozen if energy shield recharge has started recently`:                                       []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "EnergyShieldRechargeRecently"})},
	`cannot be ignited if strength is higher than dexterity`:                                                []any{flag("IgniteImmune", Tag{"type": "Condition", "var": "StrHigherThanDex"})},
	`cannot be chilled while burning`:                                                                       []any{flag("ChillImmune", Tag{"type": "Condition", "var": "Burning"})},
	`cannot be chilled while you have onslaught`:                                                            []any{flag("ChillImmune", Tag{"type": "Condition", "var": "Onslaught"})},
	`cannot be chilled during onslaught`:                                                                    []any{flag("ChillImmune", Tag{"type": "Condition", "var": "Onslaught"})},
	`cannot be frozen or chilled if you've used a fire skill recently`:                                      []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "UsedFireSkillRecently"}), flag("ChillImmune", Tag{"type": "Condition", "var": "UsedFireSkillRecently"})},
	`cannot be inflicted with bleeding`:                                                                     []any{flag("BleedImmune")},
	`bleeding cannot be inflicted on you`:                                                                   []any{flag("BleedImmune")},
	`you are immune to bleeding`:                                                                            []any{flag("BleedImmune")},
	`immune to bleeding if equipped helmet has higher armour than evasion rating`:                           []any{flag("BleedImmune", Tag{"type": "Condition", "var": "HelmetArmourHigherThanEvasion"})},
	`immune to poison if equipped helmet has higher evasion rating than armour`:                             []any{flag("PoisonImmune", Tag{"type": "Condition", "var": "HelmetEvasionHigherThanArmour"})},
	`immun[ei]t?y? to bleeding and corrupted blood during f?l?a?s?k? ?effect`:                               []any{flag("BleedImmune", Tag{"type": "Condition", "var": "UsingFlask"}), flag("CorruptedBloodImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to poison`:                                                                               []any{flag("PoisonImmune")},
	`cannot be poisoned`:                                                                                    []any{flag("PoisonImmune")},
	`cannot be poisoned while bleeding`:                                                                     []any{flag("PoisonImmune", Tag{"type": "Condition", "var": "Bleeding"})},
	`immun[ei]t?y? to poison during f?l?a?s?k? ?effect`:                                                     []any{flag("PoisonImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to shock during f?l?a?s?k? ?effect`:                                                      []any{flag("ShockImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to freeze and chill during f?l?a?s?k? ?effect`:                                           []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "UsingFlask"}), flag("ChillImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to freeze and chill while ignited`:                                                       []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "Ignited"}), flag("ChillImmune", Tag{"type": "Condition", "var": "Ignited"})},
	`immun[ei]t?y? to ignite during f?l?a?s?k? ?effect`:                                                     []any{flag("IgniteImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to bleeding during f?l?a?s?k? ?effect`:                                                   []any{flag("BleedImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to curses during f?l?a?s?k? ?effect`:                                                     []any{flag("CurseImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	`immun[ei]t?y? to curses while channelling`:                                                             []any{flag("CurseImmune", Tag{"type": "Condition", "var": "Channelling"})},
	`when you kill an enemy affected by a non-aura hex, become immune to curses for remaining hex duration`: []any{flag("Condition:CanBeCurseImmune")},
	`when you kill an enemy cursed with a non-aura hex, become immune to curses for remaining hex duration`: []any{flag("Condition:CanBeCurseImmune")},
	`immun[ei]t?y? to freeze, chill, curses and stuns during f?l?a?s?k? ?effect`:                            []any{flag("FreezeImmune", Tag{"type": "Condition", "var": "UsingFlask"}), flag("ChillImmune", Tag{"type": "Condition", "var": "UsingFlask"}), flag("CurseImmune", Tag{"type": "Condition", "var": "UsingFlask"}), flag("StunImmune", Tag{"type": "Condition", "var": "UsingFlask"})},
	// This mod doesn't work the way it should. It prevents self-chill among other issues.
	// Since we don't currently really do anything with enemy ailment infliction, this should probably be removed
	// ["cursed enemies cannot inflict elemental ailments on you"] = {
	// mod("AvoidElementalAilments", "BASE", 100, { type = "ActorCondition", actor = "enemy", var = "Cursed" }, { type = "GlobalEffect", effectType = "Global", unscalable = true }),
	// },
	`enemies inflict elemental ailments on you instead of nearby allies`: []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": flag("ElementalAilmentImmune")})},
	`unaffected by curses`:                                                  []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`you are immune to curses`:                                              []any{flag("CurseImmune")},
	`unaffected by curses while affected by zealotry`:                       []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "Condition", "var": "AffectedByZealotry"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by vulnerability while affected by determination`:           []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Vulnerability"}, Tag{"type": "Condition", "var": "AffectedByDetermination"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by enfeeble while affected by grace`:                        []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Enfeeble"}, Tag{"type": "Condition", "var": "AffectedByGrace"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by temporal chains while affected by haste`:                 []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Temporal Chains"}, Tag{"type": "Condition", "var": "AffectedByHaste"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by elemental weakness while affected by purity of elements`: []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Elemental Weakness"}, Tag{"type": "Condition", "var": "AffectedByPurityofElements"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by flammability while affected by purity of fire`:           []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Flammability"}, Tag{"type": "Condition", "var": "AffectedByPurityofFire"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by frostbite while affected by purity of ice`:               []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Frostbite"}, Tag{"type": "Condition", "var": "AffectedByPurityofIce"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by conductivity while affected by purity of lightning`:      []any{mod("CurseEffectOnSelf", "MORE", -100, Tag{"type": "SkillName", "skillName": "Conductivity"}, Tag{"type": "Condition", "var": "AffectedByPurityofLightning"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`immun[ei]t?y? to curses while you have at least ([0-9]+) rage`: fn(func(c caps) any {
		return []any{flag("CurseImmune", Tag{"type": "MultiplierThreshold", "var": "Rage", "threshold": c.n(1)})}
	}),
	`you cannot be cursed with silence`:                    []any{flag("SilenceImmune")},
	`unaffected by ignite`:                                 []any{mod("SelfIgniteEffect", "MORE", -100)},
	`unaffected by chill`:                                  []any{mod("SelfChillEffect", "MORE", -100)},
	`unaffected by chill while leeching mana`:              []any{mod("SelfChillEffect", "MORE", -100, Tag{"type": "Condition", "var": "LeechingMana"})},
	`unaffected by chill while channelling`:                []any{mod("SelfChillEffect", "MORE", -100, Tag{"type": "Condition", "var": "Channelling"})},
	`unaffected by freeze`:                                 []any{mod("SelfFreezeEffect", "MORE", -100)},
	`unaffected by shock`:                                  []any{mod("SelfShockEffect", "MORE", -100)},
	`unaffected by shock while leeching energy shield`:     []any{mod("SelfShockEffect", "MORE", -100, Tag{"type": "Condition", "var": "LeechingEnergyShield"})},
	`unaffected by shock while channelling`:                []any{mod("SelfShockEffect", "MORE", -100, Tag{"type": "Condition", "var": "Channelling"})},
	`unaffected by scorch`:                                 []any{mod("SelfScorchEffect", "MORE", -100)},
	`unaffected by brittle`:                                []any{mod("SelfBrittleEffect", "MORE", -100)},
	`unaffected by sap`:                                    []any{mod("SelfSapEffect", "MORE", -100)},
	`unaffected by damaging ailments`:                      []any{mod("SelfBleedEffect", "MORE", -100), mod("SelfIgniteEffect", "MORE", -100), mod("SelfPoisonEffect", "MORE", -100)},
	`unaffected by bleeding while affected by malevolence`: []any{mod("SelfBleedEffect", "MORE", -100, Tag{"type": "Condition", "var": "AffectedByMalevolence"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by poison while affected by malevolence`:   []any{mod("SelfPoisonEffect", "MORE", -100, Tag{"type": "Condition", "var": "AffectedByMalevolence"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`unaffected by (.+) if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("Self"+firstToUpper(c.s(1))+"Effect", "MORE", -100, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(3)) + "Item", "threshold": c.n(2)})}
	}),
	`the effect of chill on you is reversed`:                             []any{flag("SelfChillEffectIsReversed"), mod("Dummy", "DUMMY", 1, "", Tag{"type": "Condition", "var": "Chilled"})},
	`your movement speed is ([0-9]+)% of its base value`:                 fn(func(c caps) any { return []any{mod("MovementSpeed", "OVERRIDE", c.n(1)/100)} }),
	`action speed cannot be modified to below ([0-9]+)% base value`:      fn(func(c caps) any { return []any{mod("MinimumActionSpeed", "MAX", c.n(1))} }),
	`action speed cannot be modified to below base value while ignited`:  []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "Condition", "var": "Ignited"})},
	`nearby allies' action speed cannot be modified to below base value`: []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})})},
	`armour also applies to lightning damage taken from hits`:            []any{mod("ArmourAppliesToLightningDamageTaken", "BASE", 100)},
	`lightning resistance does not affect lightning damage taken`:        []any{flag("SelfIgnoreLightningResistance")},
	`([0-9]+)% increased maximum life and reduced fire resistance`:       fn(func(c caps) any { return []any{mod("Life", "INC", c.n(1)), mod("FireResist", "INC", -c.n(1))} }),
	`([0-9]+)% increased maximum mana and reduced cold resistance`:       fn(func(c caps) any { return []any{mod("Mana", "INC", c.n(1)), mod("ColdResist", "INC", -c.n(1))} }),
	`([0-9]+)% increased global maximum energy shield and reduced lightning resistance`: fn(func(c caps) any {
		return []any{mod("EnergyShield", "INC", c.n(1), Tag{"type": "Global"}), mod("LightningResist", "INC", -c.n(1))}
	}),
	`phasing while on low life`:                                []any{flag("Condition:Phasing", Tag{"type": "Condition", "var": "LowLife"})},
	`cannot be ignited while on low life`:                      []any{flag("IgniteImmune", Tag{"type": "Condition", "var": "LowLife"})},
	`ward does not break during f?l?a?s?k? ?effect`:            []any{flag("Condition:WardNotBreak", Tag{"type": "Condition", "var": "UsingFlask"}), flag("Condition:UnbrokenWard", Tag{"type": "Condition", "var": "UsingFlask"})},
	`stun threshold is based on energy shield instead of life`: []any{flag("StunThresholdBasedOnEnergyShieldInsteadOfLife"), mod("StunThresholdEnergyShieldPercent", "BASE", 100)},
	`stun threshold is based on ([0-9]+)% of your energy shield instead of life`: fn(func(c caps) any {
		return []any{flag("StunThresholdBasedOnEnergyShieldInsteadOfLife"), mod("StunThresholdEnergyShieldPercent", "BASE", c.n(1))}
	}),
	`stun threshold is based on ([0-9]+)% of your mana instead of life`: fn(func(c caps) any {
		return []any{flag("StunThresholdBasedOnManaInsteadOfLife"), mod("StunThresholdManaPercent", "BASE", c.n(1))}
	}),
	`([0-9]+)% of your energy shield is added to your stun threshold`: fn(func(c caps) any {
		return []any{flag("AddESToStunThreshold"), mod("ESToStunThresholdPercent", "BASE", c.n(1))}
	}),
	`([0-9]+)% of damage from hits is taken from your spectres' life before you`: fn(func(c caps) any { return []any{mod("takenFromSpectresBeforeYou", "BASE", c.n(1))} }),
	`([0-9]+)% of damage from hits is taken from your nearest totem's life before you`: fn(func(c caps) any {
		return []any{mod("takenFromTotemsBeforeYou", "BASE", c.n(1), Tag{"type": "Condition", "var": "HaveTotem"})}
	}),
	`([0-9]+)% of damage from hits is taken from void spawns' life before you per void spawn`: fn(func(c caps) any {
		return []any{mod("takenFromVoidSpawnBeforeYou", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "ActiveVoidSpawnLimit"})}
	}),
	`([a-zA-Z]+) resistance cannot be penetrated`: fn(func(c caps) any { return []any{flag("EnemyCannotPen" + firstToUpper(c.s(1)) + "Resistance")} }),
	// Knockback
	`cannot knock enemies back`:                                     []any{flag("CannotKnockback")},
	`knocks back enemies if you get a critical strike with a staff`: []any{mod("EnemyKnockbackChance", "BASE", 100, nil, ModFlag.Staff, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`knocks back enemies if you get a critical strike with a bow`:   []any{mod("EnemyKnockbackChance", "BASE", 100, nil, ModFlag.Bow, Tag{"type": "Condition", "var": "CriticalStrike"})},
	`bow knockback at close range`:                                  []any{mod("EnemyKnockbackChance", "BASE", 100, nil, ModFlag.Bow, Tag{"type": "Condition", "var": "AtCloseRange"})},
	`adds knockback during f?l?a?s?k? ?effect`:                      []any{mod("EnemyKnockbackChance", "BASE", 100, Tag{"type": "Condition", "var": "UsingFlask"})},
	`adds knockback to melee attacks during f?l?a?s?k? ?effect`:     []any{mod("EnemyKnockbackChance", "BASE", 100, nil, ModFlag.Melee, Tag{"type": "Condition", "var": "UsingFlask"})},
	`knockback direction is reversed`:                               []any{mod("EnemyKnockbackDistance", "MORE", -200)},
	// Culling
	`culling strike`:                                                     []any{mod("CullPercent", "MAX", 10, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`culling strike with melee weapons`:                                  []any{mod("CullPercent", "MAX", 10, nil, ModFlag.WeaponMelee, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`melee weapon attacks have culling strike`:                           []any{mod("CullPercent", "MAX", 10, nil, ModFlag.Attack|ModFlag.WeaponMelee, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`culling strike during f?l?a?s?k? ?effect`:                           []any{mod("CullPercent", "MAX", 10, Tag{"type": "Condition", "var": "UsingFlask"}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`hits with this weapon have culling strike against bleeding enemies`: []any{mod("CullPercent", "MAX", 10, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"})},
	`you have culling strike against cursed enemies`:                     []any{mod("CullPercent", "MAX", 10, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})},
	`critical strikes have culling strike`:                               []any{mod("CriticalCullPercent", "MAX", 10)},
	`your critical strikes have culling strike`:                          []any{mod("CriticalCullPercent", "MAX", 10)},
	`your spells have culling strike`:                                    []any{mod("CullPercent", "MAX", 10, nil, ModFlag.Spell)},
	`bow attacks have culling strike`:                                    []any{mod("CullPercent", "MAX", 10, nil, ModFlag.Attack|ModFlag.Bow)},
	`culling strike against burning enemies`:                             []any{mod("CullPercent", "MAX", 10, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Burning"})},
	`culling strike against frozen enemies`:                              []any{mod("CullPercent", "MAX", 10, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"})},
	`culling strike against marked enemy`:                                []any{mod("CullPercent", "MAX", 10, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"})},
	`nearby allies have culling strike`:                                  []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("CullPercent", "MAX", 10)})},
	`hits that stun enemies have culling strike`:                         []any{flag("Condition:maceMasteryStunCullSpecced")},
	// Intimidate
	`permanently intimidate enemies on block`:                                                         []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "Condition", "var": "BlockedRecently"})},
	`with a murderous eye jewel socketed, intimidate enemies for ([0-9]) seconds on hit with attacks`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "Condition", "var": "HaveMurderousEyeJewelIn{SlotName}"})},
	`enemies taunted by your warcries are intimidated`:                                                []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated", Tag{"type": "Condition", "var": "Taunted"})}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})},
	`intimidate enemies for ([0-9]+) seconds on block while holding a shield`:                         []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "Condition", "var": "BlockedRecently"}, Tag{"type": "Condition", "var": "UsingShield"})},
	`intimidate enemies for ([0-9]+) seconds on hit with attacks while at maximum endurance charges`:  []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMax"}, Tag{"type": "Condition", "var": "HitRecently"})},
	`your hits intimidate enemies for ([0-9]+) seconds while you are using pride`:                     []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "Condition", "var": "AffectedByPride"})},
	// Flasks
	`flasks do not apply to you`:                               []any{flag("FlasksDoNotApplyToPlayer")},
	`flasks apply to your zombies and spectres`:                []any{flag("FlasksApplyToMinion", Tag{"type": "SkillName", "skillNameList": []any{"Raise Zombie", "Raise Spectre"}, "includeTransfigured": true})},
	`flasks apply to your raised zombies and spectres`:         []any{flag("FlasksApplyToMinion", Tag{"type": "SkillName", "skillNameList": []any{"Raise Zombie", "Raise Spectre"}, "includeTransfigured": true})},
	`flasks you use apply to your raised zombies and spectres`: []any{flag("FlasksApplyToMinion", Tag{"type": "SkillName", "skillNameList": []any{"Raise Zombie", "Raise Spectre"}, "includeTransfigured": true})},
	`your minions use your flasks when summoned`:               []any{flag("FlasksApplyToMinion")},
	`recover an additional ([0-9]+)% of flask's life recovery amount over 10 seconds if used while not on full life`: fn(func(c caps) any { return []any{mod("FlaskAdditionalLifeRecovery", "BASE", c.n(1))} }),
	`creates a smoke cloud on use`:                     d(),
	`creates chilled ground on use`:                    d(),
	`creates consecrated ground on use`:                d(),
	`removes bleeding on use`:                          d(),
	`removes burning on use`:                           d(),
	`removes all burning when used`:                    d(),
	`removes curses on use`:                            d(),
	`removes freeze and chill on use`:                  d(),
	`removes poison on use`:                            d(),
	`removes shock on use`:                             d(),
	`g?a?i?n? ?unholy might during f?l?a?s?k? ?effect`: []any{flag("Condition:UnholyMight", Tag{"type": "Condition", "var": "UsingFlask"}), flag("Condition:CanWither", Tag{"type": "Condition", "var": "UsingFlask"})},
	`zealot's oath during f?l?a?s?k? ?effect`:          []any{flag("ZealotsOath", Tag{"type": "Condition", "var": "UsingFlask"})},
	`shocks nearby enemies during f?l?a?s?k? ?effect, causing ([0-9]+)% increased damage taken`: fn(func(c caps) any {
		return []any{mod("ShockOverride", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`during f?l?a?s?k? ?effect, ([0-9]+)% reduced damage taken of each element for which your uncapped elemental resistance is lowest`: fn(func(c caps) any {
		return []any{mod("LightningDamageTaken", "INC", -c.n(1), Tag{"type": "StatThreshold", "stat": "LightningResistTotal", "thresholdStat": "ColdResistTotal", "upper": true}, Tag{"type": "StatThreshold", "stat": "LightningResistTotal", "thresholdStat": "FireResistTotal", "upper": true}), mod("ColdDamageTaken", "INC", -c.n(1), Tag{"type": "StatThreshold", "stat": "ColdResistTotal", "thresholdStat": "LightningResistTotal", "upper": true}, Tag{"type": "StatThreshold", "stat": "ColdResistTotal", "thresholdStat": "FireResistTotal", "upper": true}), mod("FireDamageTaken", "INC", -c.n(1), Tag{"type": "StatThreshold", "stat": "FireResistTotal", "thresholdStat": "LightningResistTotal", "upper": true}, Tag{"type": "StatThreshold", "stat": "FireResistTotal", "thresholdStat": "ColdResistTotal", "upper": true})}
	}),
	`during f?l?a?s?k? ?effect, damage penetrates ([0-9]+)% o?f? ?resistance of each element for which your uncapped elemental resistance is highest`: fn(func(c caps) any {
		return []any{mod("LightningPenetration", "BASE", c.n(1), Tag{"type": "StatThreshold", "stat": "LightningResistTotal", "thresholdStat": "ColdResistTotal"}, Tag{"type": "StatThreshold", "stat": "LightningResistTotal", "thresholdStat": "FireResistTotal"}), mod("ColdPenetration", "BASE", c.n(1), Tag{"type": "StatThreshold", "stat": "ColdResistTotal", "thresholdStat": "LightningResistTotal"}, Tag{"type": "StatThreshold", "stat": "ColdResistTotal", "thresholdStat": "FireResistTotal"}), mod("FirePenetration", "BASE", c.n(1), Tag{"type": "StatThreshold", "stat": "FireResistTotal", "thresholdStat": "LightningResistTotal"}, Tag{"type": "StatThreshold", "stat": "FireResistTotal", "thresholdStat": "ColdResistTotal"})}
	}),
	`damage penetrates fire resistance equal to your overcapped fire resistance`: []any{flag("FirePenIncreasedByUncappedFireRes")},
	`damage penetrates fire resistance equal to your overcapped fire resistance, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("FirePenetration", "BASE", 1, Tag{"type": "PerStat", "stat": "FireResistOverCap", "limit": c.n(1), "limitTotal": true})}
	}),
	`recover ([0-9]+)% of life when you kill an enemy during f?l?a?s?k? ?effect`: fn(func(c caps) any {
		return []any{mod("LifeOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`recover ([0-9]+)% of mana when you kill an enemy during f?l?a?s?k? ?effect`: fn(func(c caps) any {
		return []any{mod("ManaOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`recover ([0-9]+)% of energy shield when you kill an enemy during f?l?a?s?k? ?effect`: fn(func(c caps) any {
		return []any{mod("EnergyShieldOnKill", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`([0-9]+)% of maximum life taken as chaos damage per second`: fn(func(c caps) any {
		return []any{mod("ChaosDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}
	}),
	`your critical strikes do not deal extra damage during f?l?a?s?k? ?effect`: []any{flag("NoCritMultiplier", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants perfect agony during f?l?a?s?k? ?effect`:                           []any{mod("Keystone", "LIST", "Perfect Agony", Tag{"type": "Condition", "var": "UsingFlask"})},
	`grants eldritch battery during f?l?a?s?k? ?effect`:                        []any{mod("Keystone", "LIST", "Eldritch Battery", Tag{"type": "Condition", "var": "UsingFlask"})},
	`eldritch battery during f?l?a?s?k? ?effect`:                               []any{mod("Keystone", "LIST", "Eldritch Battery", Tag{"type": "Condition", "var": "UsingFlask"})},
	`chaos damage t?a?k?e?n? ?does not bypass energy shield during effect`:     []any{flag("ChaosNotBypassEnergyShield")},
	`when hit during effect, ([0-9]+)% of life loss from damage taken occurs over 4 seconds instead`: fn(func(c caps) any {
		return []any{mod("LifeLossPrevented", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`y?o?u?r? ?skills [ch][oa][sv][te] no mana c?o?s?t? ?during f?l?a?s?k? ?effect`:      []any{mod("ManaCost", "MORE", -100, Tag{"type": "Condition", "var": "UsingFlask"})},
	`life recovery from flasks also applies to energy shield during f?l?a?s?k? ?effect`:  []any{flag("LifeFlaskAppliesToEnergyShield", Tag{"type": "Condition", "var": "UsingFlask"})},
	`profane ground you create inflicts malediction on enemies`:                          []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMalediction", Tag{"type": "Condition", "var": "OnProfaneGround"})})},
	`profane ground you create also affects you and your allies, granting chaotic might`: []any{mod("ExtraAura", "LIST", Tag{"mod": flag("Condition:ChaoticMight", Tag{"type": "Condition", "var": "OnProfaneGround"})})},
	`raised beast spectres have farrul's farric presence`:                                []any{mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("Accuracy", "INC", 80)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"}), mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("CritChance", "INC", 120)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"}), mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("ReduceCritExtraDamage", "BASE", 100)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"})},
	`raised beast spectres have farrul's fertile presence`:                               []any{mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("Damage", "INC", 100)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"}), mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("LifeRegenPercent", "BASE", 3)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"}), mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": flag("StunImmune")}, Tag{"type": "Condition", "var": "HaveBeastSpectre"})},
	`raised beast spectres have farrul's wild presence`:                                  []any{mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("Speed", "INC", 20)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"}), mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("MovementSpeed", "INC", 20)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"}), mod("ExtraAura", "LIST", Tag{"fromAllies": true, "mod": mod("MinimumActionSpeed", "MAX", 100)}, Tag{"type": "Condition", "var": "HaveBeastSpectre"})},
	`gain alchemist's genius when you use a flask`:                                       []any{flag("Condition:CanHaveAlchemistGenius")},
	`([0-9]+)% chance to gain alchemist's genius when you use a flask`:                   []any{flag("Condition:CanHaveAlchemistGenius")},
	`([0-9]+)% less flask charges gained from kills`:                                     fn(func(c caps) any { return []any{mod("FlaskChargesGained", "MORE", -c.n(1), "from Kills")} }),
	`flasks gain ([0-9]+) charges? every ([0-9]+) seconds`:                               fn(func(c caps) any { return []any{mod("FlaskChargesGenerated", "BASE", c.n(1)/c.n(2))} }),
	`flasks gain a charge every ([0-9]+) seconds`:                                        fn(func(c caps) any { return []any{mod("FlaskChargesGenerated", "BASE", 1/c.n(1))} }),
	`while a unique enemy is in your presence, flasks gain a charge every ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("FlaskChargesGenerated", "BASE", 1/c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})}
	}),
	`while a pinnacle atlas boss is in your presence, flasks gain a charge every ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("FlaskChargesGenerated", "BASE", 1/c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"})}
	}),
	`utility flasks gain ([0-9]+) charges? every ([0-9]+) seconds`: fn(func(c caps) any { return []any{mod("UtilityFlaskChargesGenerated", "BASE", c.n(1)/c.n(2))} }),
	`iron flasks gain ([0-9]+) charges? when your ward breaks`:     fn(func(c caps) any { return []any{mod("IronFlaskChargesGeneratedOnWardBreak", "BASE", c.n(1))} }),
	`life flasks gain ([0-9]+) charges? every ([0-9]+) seconds`:    fn(func(c caps) any { return []any{mod("LifeFlaskChargesGenerated", "BASE", c.n(1)/c.n(2))} }),
	`while on low life, life flasks gain ([0-9]+) charges? every ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("LifeFlaskChargesGenerated", "BASE", c.n(1)/c.n(2), Tag{"type": "Condition", "var": "LowLife"})}
	}),
	`mana flasks gain ([0-9]+) charges? every ([0-9]+) seconds`:                 fn(func(c caps) any { return []any{mod("ManaFlaskChargesGenerated", "BASE", c.n(1)/c.n(2))} }),
	`flasks gain ([0-9]+) charges? per empty flask slot every ([0-9]+) seconds`: fn(func(c caps) any { return []any{mod("FlaskChargesGeneratedPerEmptyFlask", "BASE", c.n(1)/c.n(2))} }),
	`flasks gain ([0-9]+) charges? per second if you've hit a unique enemy recently`: fn(func(c caps) any {
		return []any{mod("FlaskChargesGenerated", "BASE", c.n(1), Tag{"type": "Condition", "var": "HitRecently"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"})}
	}),
	`effect is not removed when unreserved mana is filled`:              []any{flag("ManaFlaskEffectNotRemoved")},
	`life flask effects are not removed when unreserved life is filled`: []any{flag("LifeFlaskEffectNotRemoved")},
	`mana flask effects are not removed when unreserved mana is filled`: []any{flag("ManaFlaskEffectNotRemoved")},
	// Jewels
	`passives? ?s?k?i?l?l?s? in radius of ([a-zA-Z \t\n\v\f\r']+) can be allocated without being connected to your tree`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "impossibleEscapeKeystone", "value": c.s(1)}), mod("ImpossibleEscapeKeystones", "LIST", Tag{"key": c.s(1), "value": true})}
	}),
	`passives? ?s?k?i?l?l?s? in radius can be allocated without being connected to your tree`: []any{mod("JewelData", "LIST", Tag{"key": "intuitiveLeapLike", "value": true})},
	`keystone passive skills in radius can be allocated without being connected to your tree`: []any{mod("JewelData", "LIST", Tag{"key": "intuitiveLeapLike", "value": true}), mod("JewelData", "LIST", Tag{"key": "intuitiveLeapKeystoneOnly", "value": true})},
	`affects passives in small ring`:                    []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 6})},
	`affects passives in medium ring`:                   []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 7})},
	`affects passives in large ring`:                    []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 8})},
	`affects passives in very large ring`:               []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 9})},
	`affects passives in massive ring`:                  []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 10})},
	`only affects passives in small ring`:               []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 6})},
	`only affects passives in medium ring`:              []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 7})},
	`only affects passives in large ring`:               []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 8})},
	`only affects passives in very large ring`:          []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 9})},
	`only affects passives in massive ring`:             []any{mod("JewelData", "LIST", Tag{"key": "radiusIndex", "value": 10})},
	`primordial`:                                        []any{mod("Multiplier:PrimordialItem", "BASE", 1)},
	`spectres have a base duration of ([0-9]+) seconds`: []any{mod("SkillData", "LIST", Tag{"key": "duration", "value": 6}, Tag{"type": "SkillName", "skillName": "Raise Spectre", "includeTransfigured": true})},
	`flasks applied to you have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("FlaskEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`flasks applied to you have ([0-9]+)% increased effect per level`: fn(func(c caps) any {
		return []any{mod("FlaskEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "player"}, Tag{"type": "Multiplier", "var": "Level"})}
	}),
	`equipped magic flasks have ([0-9]+)% increased effect on you if no flasks are adjacent to them`: fn(func(c caps) any { return []any{mod("MagicFlaskNoAdjacentEffect", "INC", c.n(1))} }),
	`while a unique enemy is in your presence, flasks applied to you have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("FlaskEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"}, Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`while a pinnacle atlas boss is in your presence, flasks applied to you have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("FlaskEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "PinnacleBoss"}, Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`magic utility flasks applied to you have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("MagicUtilityFlaskEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`flasks applied to you have ([0-9]+)% reduced effect`: fn(func(c caps) any {
		return []any{mod("FlaskEffect", "INC", -c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`tinctures applied to you have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("TinctureEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`tinctures applied to you have ([0-9]+)% increased effect if you've used a life flask recently`: fn(func(c caps) any {
		return []any{mod("TinctureEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "player"}, Tag{"type": "Condition", "var": "UsingLifeFlask"})}
	}),
	`tinctures applied to you have ([0-9]+)% increased effect while affected by no flasks`: fn(func(c caps) any {
		return []any{mod("TinctureEffect", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "player"}, Tag{"type": "Condition", "var": "UsingFlask", "neg": true})}
	}),
	`tinctures have ([0-9]+)% increased effect while at or above ([0-9]+) stacks of mana burn`: fn(func(c caps) any {
		return []any{mod("TinctureEffect", "INC", c.n(1), Tag{"type": "MultiplierThreshold", "varList": []any{"ManaBurnStacks", "WeepingWoundsStacks"}, "threshold": c.n(2)})}
	}),
	`tinctures applied to you have ([0-9]+)% reduced mana burn rate`: fn(func(c caps) any {
		return []any{mod("TinctureManaBurnRate", "INC", -c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`tinctures applied to you have ([0-9]+)% less mana burn rate`: fn(func(c caps) any {
		return []any{mod("TinctureManaBurnRate", "MORE", -c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`the first ([0-9]+) mana burn applied to you have no effect`: fn(func(c caps) any {
		return []any{mod("EffectiveManaBurnStacks", "BASE", -c.n(1), Tag{"type": "ActorCondition", "actor": "player"})}
	}),
	`tinctures deactivate when you have ([0-9]+) or more mana burn`: fn(func(c caps) any { return []any{mod("MaxManaBurnStacks", "BASE", c.n(1))} }),
	`tinctures inflict weeping wounds instead of mana burn`:         []any{flag("Condition:WeepingWoundsInsteadOfManaBurn")},
	`tincture effects also apply to ranged weapons`:                 []any{flag("TinctureRangedWeapons")},
	`you can have an additional tincture active`:                    []any{mod("TinctureLimit", "BASE", 1)},
	`([0-9]+)% increased tincture cooldown recovery rate`:           fn(func(c caps) any { return []any{mod("TinctureCooldownRecovery", "INC", c.n(1))} }),
	`adds ([0-9]+) passive skills`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelNodeCount", "value": c.n(1)})}
	}),
	`1 added passive skill is a jewel socket`: []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelSocketCount", "value": 1})},
	`([0-9]+) added passive skills are jewel sockets`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelSocketCount", "value": c.n(1)})}
	}),
	`adds ([0-9]+) jewel socket passive skills`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelSocketCountOverride", "value": c.n(1)})}
	}),
	`adds ([0-9]+) small passive skills? which grants? nothing`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelNothingnessCount", "value": c.n(1)})}
	}),
	`added small passive skills grant nothing`: []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelSmallsAreNothingness", "value": true})},
	`added small passive skills have ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelIncEffect", "value": c.n(1)})}
	}),
	`this jewel's socket has ([0-9]+)% increased effect per allocated passive skill between it and your class' starting location`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "jewelIncEffectFromClassStart", "value": c.n(1)})}
	}),
	`([0-9]+)% increased effect of jewel socket passive skills containing corrupted (m?r?ag?r?i?e?c?) jewels, if not from cluster jewels`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "corrupted" + firstToUpper(c.s(2)) + "JewelIncEffect", "value": c.n(1)})}
	}),
	`([0-9]+)% increased effect of jewel socket passive skills containing corrupted (m?r?ag?r?i?e?c?) jewels`: fn(func(c caps) any {
		return []any{mod("JewelData", "LIST", Tag{"key": "corrupted" + firstToUpper(c.s(2)) + "JewelIncEffect", "value": c.n(1)})}
	}),
	// Misc
	`can't use chest armour`:        []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Body Armour"})},
	`can't use helmets?`:            []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Helmet"})},
	`can't use other rings`:         []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Ring 2"}, Tag{"type": "SlotNumber", "num": 1}), mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Ring 1"}, Tag{"type": "SlotNumber", "num": 2})},
	`uses both hand slots`:          []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Weapon 2"}, Tag{"type": "SlotNumber", "num": 1}), mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Weapon 1"}, Tag{"type": "SlotNumber", "num": 2})},
	`can't use flask in fifth slot`: []any{mod("CanNotUseItem", "Flag", 1, Tag{"type": "DisablesItem", "slotName": "Flask 5", "excludeItemType": "Tincture"})},
	`boneshatter has ([0-9]+)% chance to grant \+1 trauma`: fn(func(c caps) any {
		return []any{mod("ExtraTrauma", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Boneshatter", "includeTransfigured": true})}
	}),
	`your minimum frenzy, endurance and power charges are equal to your maximum while you are stationary`: []any{flag("MinimumFrenzyChargesIsMaximumFrenzyCharges", Tag{"type": "Condition", "var": "Stationary"}), flag("MinimumEnduranceChargesIsMaximumEnduranceCharges", Tag{"type": "Condition", "var": "Stationary"}), flag("MinimumPowerChargesIsMaximumPowerCharges", Tag{"type": "Condition", "var": "Stationary"})},
	`minimum power charges equal to maximum while stationary`:                                             []any{flag("MinimumPowerChargesIsMaximumPowerCharges", Tag{"type": "Condition", "var": "Stationary"})},
	`minimum frenzy charges equal to maximum while stationary`:                                            []any{flag("MinimumFrenzyChargesIsMaximumFrenzyCharges", Tag{"type": "Condition", "var": "Stationary"})},
	`minimum endurance charges equal to maximum while stationary`:                                         []any{flag("MinimumEnduranceChargesIsMaximumEnduranceCharges", Tag{"type": "Condition", "var": "Stationary"})},
	`count as having maximum number of power charges`:                                                     []any{flag("HaveMaximumPowerCharges")},
	`count as having maximum number of frenzy charges`:                                                    []any{flag("HaveMaximumFrenzyCharges")},
	`count as having maximum number of endurance charges`:                                                 []any{flag("HaveMaximumEnduranceCharges")},
	`leftmost ([0-9]+) magic utility flasks constantly apply their flask effects to you`:                  fn(func(c caps) any { return []any{mod("LeftActiveMagicUtilityFlasks", "BASE", c.n(1))} }),
	`rightmost ([0-9]+) magic utility flasks constantly apply their flask effects to you`:                 fn(func(c caps) any { return []any{mod("RightActiveMagicUtilityFlasks", "BASE", c.n(1))} }),
	`marauder: melee skills have ([0-9]+)% increased area of effect`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Condition", "var": "ConnectedToMarauderStart"}, Tag{"type": "SkillType", "skillType": SkillType.Melee})}
	}),
	`intelligence provides no bonus to energy shield`:                                       []any{flag("NoIntBonusToES")},
	`intelligence provides no inherent bonus to energy shield`:                              []any{flag("NoIntBonusToES")},
	`gain accuracy rating equal to your intelligence`:                                       []any{mod("Accuracy", "BASE", 1, Tag{"type": "PerStat", "stat": "Int"})},
	`intelligence is added to accuracy rating with wands`:                                   []any{mod("Accuracy", "BASE", 1, nil, ModFlag.Wand, Tag{"type": "PerStat", "stat": "Int"})},
	`dexterity's accuracy bonus instead grants \+([0-9]+) to accuracy rating per dexterity`: fn(func(c caps) any { return []any{mod("DexAccBonusOverride", "OVERRIDE", c.n(1))} }),
	`([0-9]+)% increased accuracy rating against marked enemy`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "INC", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"})}
	}),
	`([0-9]+)% more accuracy rating against marked enemy`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "MORE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"})}
	}),
	`\+([0-9]+) to accuracy against bleeding enemies`: fn(func(c caps) any {
		return []any{mod("AccuracyVsEnemy", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"})}
	}),
	`cannot recover energy shield to above armour`:                         []any{flag("ArmourESRecoveryCap")},
	`cannot recover energy shield to above evasion rating`:                 []any{flag("EvasionESRecoveryCap")},
	`warcries exert ([0-9]+) additional attacks?`:                          fn(func(c caps) any { return []any{mod("ExtraExertedAttacks", "BASE", c.n(1))} }),
	`battlemage's cry exerts ([0-9]+) additional attack`:                   fn(func(c caps) any { return []any{mod("BattlemageExertedAttacks", "BASE", c.n(1))} }),
	`rallying cry exerts ([0-9]+) additional attack`:                       fn(func(c caps) any { return []any{mod("RallyingExertedAttacks", "BASE", c.n(1))} }),
	`warcries have ([0-9]+)% chance to exert ([0-9]+) additional attacks?`: fn(func(c caps) any { return []any{mod("ExtraExertedAttacks", "BASE", (c.n(1) * c.n(2) / 100))} }),
	`skills deal ([0-9]+)% more damage for each warcry exerting them`:      fn(func(c caps) any { return []any{mod("EchoesExertAverageIncrease", "MORE", c.n(1), nil)} }),
	`iron will`:                      []any{flag("IronWill")},
	`iron reflexes while stationary`: []any{mod("Keystone", "LIST", "Iron Reflexes", Tag{"type": "Condition", "var": "Stationary"})},
	`you have iron reflexes while at maximum frenzy charges`:    []any{mod("Keystone", "LIST", "Iron Reflexes", Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"})},
	`you have zealot's oath if you haven't been hit recently`:   []any{mod("Keystone", "LIST", "Zealot's Oath", Tag{"type": "Condition", "var": "BeenHitRecently", "neg": true})},
	`deal no physical damage`:                                   []any{flag("DealNoPhysical")},
	`deal no cold damage`:                                       []any{flag("DealNoCold")},
	`deal no fire damage`:                                       []any{flag("DealNoFire")},
	`deal no lightning damage`:                                  []any{flag("DealNoLightning")},
	`deal no elemental damage`:                                  []any{flag("DealNoLightning"), flag("DealNoCold"), flag("DealNoFire")},
	`deal no chaos damage`:                                      []any{flag("DealNoChaos")},
	`deal no damage`:                                            []any{flag("DealNoDamage")},
	`you can't deal damage with skills yourself`:                []any{flag("DealNoDamage", Tag{"type": "SkillType", "skillTypeList": []any{SkillType.SummonsTotem, SkillType.RemoteMined, SkillType.Trapped}, "neg": true}, Tag{"type": "Condition", "var": "usedByMirage", "neg": true})},
	`deal no non-elemental damage`:                              []any{flag("DealNoPhysical"), flag("DealNoChaos")},
	`deal no non-([a-zA-Z]+) damage`:                            fn(func(c caps) any { return dealNoNonDamageType(c.s(1)) }),
	`cannot deal non-([a-zA-Z]+) damage`:                        fn(func(c caps) any { return dealNoNonDamageType(c.s(1)) }),
	`deal no physical or elemental damage`:                      []any{flag("DealNoPhysical"), flag("DealNoCold"), flag("DealNoFire"), flag("DealNoLightning")},
	`deal no damage when not on low life`:                       []any{flag("DealNoDamage", Tag{"type": "Condition", "var": "LowLife", "neg": true})},
	`spell skills deal no damage`:                               []any{flag("DealNoDamage", Tag{"type": "SkillType", "skillType": SkillType.Spell})},
	`attacks have blood magic`:                                  []any{flag("CostLifeInsteadOfMana", nil, ModFlag.Attack)},
	`attacks cost life instead of mana`:                         []any{flag("CostLifeInsteadOfMana", nil, ModFlag.Attack)},
	`attack skills cost life instead of ([0-9]+)% of mana cost`: fn(func(c caps) any { return []any{mod("HybridManaAndLifeCost_Life", "BASE", c.n(1), nil, ModFlag.Attack)} }),
	`skills cost energy shield instead of mana or life`:         []any{flag("CostESInsteadOfManaOrLife")},
	`spells have an additional life cost equal to ([0-9]+)% of your maximum life`: fn(func(c caps) any {
		return []any{mod("LifeCostBase", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1), "floor": true}, Tag{"type": "SkillType", "skillType": SkillType.Spell})}
	}),
	`spells have added spell damage equal to ([0-9]+)% of physical damage of your equipped two handed weapon`: fn(func(c caps) any {
		return []any{flag("WeaponPhysAppliesToSpells"), mod("WeaponPhysAppliesToSpellsPercent", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsingTwoHandedWeapon"})}
	}),
	`spells cost \+([0-9]+)% of life`: fn(func(c caps) any {
		return []any{mod("LifeCostBase", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1), "floor": true}, Tag{"type": "SkillType", "skillType": SkillType.Spell})}
	}),
	`trigger a socketed elemental spell on block, with a ([0-9.]+) second cooldown`:                                 []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerElementalSpellOnBlock", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when you block, with a ([0-9.]+) second cooldown`:                                     []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueVirulenceSpellsCastOnBlock", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`([0-9]+)% chance to cast a? ?socketed lightning spells? on hit`:                                                []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueMjolnerLightningSpellsCastOnHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`cast a socketed lightning spell on hit`:                                                                        []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueMjolnerLightningSpellsCastOnHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed lightning spell on hit`:                                                                     []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueMjolnerLightningSpellsCastOnHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed lightning spell on hit, with a ([0-9.]+) second cooldown`:                                   []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueMjolnerLightningSpellsCastOnHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell when a hit from this weapon freezes a target, with a ([0-9.]+) second cooldown`:       []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnBowAttackFreezeHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`trigger a socketed spell on unarmed melee critical strike, with a ([0-9.]+) second cooldown`:                   []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportTriggerSpellOnUnarmedMeleeCriticalHit", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike`:                                   []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike, with a ([0-9.]+) second cooldown`: []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`you cannot cast socketed hex curse skills inflict socketed hexes on enemies that trigger your traps`:           []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportCurseOnTrapTriggered", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`your curses can apply to hexproof enemies`:                                                                     []any{flag("CursesIgnoreHexproof")},
	`your hexes can affect hexproof enemies`:                                                                        []any{flag("CursesIgnoreHexproof")},
	`([a-zA-Z \t\n\v\f\r]+) can affect hexproof enemies`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "ignoreHexproof", "value": true}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))})}
	}),
	`hexes from socketed skills can apply ([0-9]) additional curses`: fn(func(c caps) any {
		return []any{mod("SocketedCursesHexLimitValue", "BASE", c.n(1)), flag("SocketedCursesAdditionalLimit", Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	// This is being changed from ignoreHexLimit to SocketedCursesAdditionalLimit due to patch 3.16.0, which states that legacy versions "will be affected by this Curse Limit change,
	// though they will only have 20% less Curse Effect of Curses triggered with Summon Doedre’s Effigy."
	// Legacy versions will still show that "Hexes from Socketed Skills ignore Curse limit", but will instead have an internal limit of 5 to match the current functionality.
	`hexes from socketed skills ignore curse limit`: fn(func(c caps) any {
		return []any{mod("SocketedCursesHexLimitValue", "BASE", 5), flag("SocketedCursesAdditionalLimit", Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`reserves ([0-9]+)% of life`:                  fn(func(c caps) any { return []any{mod("ExtraLifeReserved", "BASE", c.n(1))} }),
	`([0-9]+)% of cold damage taken as lightning`: fn(func(c caps) any { return []any{mod("ColdDamageTakenAsLightning", "BASE", c.n(1))} }),
	`([0-9]+)% of fire damage taken as lightning`: fn(func(c caps) any { return []any{mod("FireDamageTakenAsLightning", "BASE", c.n(1))} }),
	`([0-9]+)% of fire and lightning damage from hits taken as cold damage during effect`: fn(func(c caps) any {
		return []any{mod("FireDamageFromHitsTakenAsCold", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsingFlask"}), mod("LightningDamageFromHitsTakenAsCold", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsingFlask"})}
	}),
	`items and gems have ([0-9]+)% reduced attribute requirements`:   fn(func(c caps) any { return []any{mod("GlobalAttributeRequirements", "INC", -c.n(1))} }),
	`items and gems have ([0-9]+)% increased attribute requirements`: fn(func(c caps) any { return []any{mod("GlobalAttributeRequirements", "INC", c.n(1))} }),
	`mana reservation of herald skills is always ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "ManaReservationPercentForced", "value": c.n(1)}, Tag{"type": "SkillType", "skillType": SkillType.Herald})}
	}),
	`([a-zA-Z \t\n\v\f\r]+) reserves no mana`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "manaReservationFlat", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationFlat", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "manaReservationPercent", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationPercent", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true})}
	}),
	`([a-zA-Z \t\n\v\f\r]+) has no reservation`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "manaReservationFlat", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationFlat", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "manaReservationPercent", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationPercent", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true})}
	}),
	`([a-zA-Z \t\n\v\f\r]+) has no reservation if cast as an aura`: fn(func(c caps) any {
		return []any{mod("SkillData", "LIST", Tag{"key": "manaReservationFlat", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationFlat", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "manaReservationPercent", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationPercent", "value": 0}, Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))}, Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true})}
	}),
	`banner skills reserve no mana`:     []any{mod("SkillData", "LIST", Tag{"key": "manaReservationPercent", "value": 0}, Tag{"type": "SkillType", "skillType": SkillType.Banner}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationPercent", "value": 0}, Tag{"type": "SkillType", "skillType": SkillType.Banner}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true})},
	`banner skills have no reservation`: []any{mod("SkillData", "LIST", Tag{"key": "manaReservationPercent", "value": 0}, Tag{"type": "SkillType", "skillType": SkillType.Banner}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true}), mod("SkillData", "LIST", Tag{"key": "lifeReservationPercent", "value": 0}, Tag{"type": "SkillType", "skillType": SkillType.Banner}, Tag{"type": "SkillType", "skillType": SkillType.Blessing, "neg": true})},
	`placed banners also grant ([0-9]+)% increased attack damage to you and allies`: fn(func(c caps) any {
		return []any{mod("ExtraAuraEffect", "LIST", Tag{"mod": mod("Damage", "INC", c.n(1), nil, ModFlag.Attack)}, Tag{"type": "Condition", "var": "BannerPlanted"}, Tag{"type": "SkillType", "skillType": SkillType.Banner})}
	}),
	`banners also cause enemies to take ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("ExtraAuraDebuffEffect", "LIST", Tag{"mod": mod("DamageTaken", "INC", c.n(1), Tag{"type": "GlobalEffect", "effectType": "AuraDebuff", "unscalable": true})}, Tag{"type": "Condition", "var": "BannerPlanted"}, Tag{"type": "SkillType", "skillType": SkillType.Banner})}
	}),
	`dread banner grants an additional \+([0-9]+) to maximum fortification when placing the banner`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("MaximumFortification", "BASE", c.n(1), Tag{"type": "GlobalEffect", "effectType": "Buff"})}, Tag{"type": "Condition", "var": "BannerPlanted"}, Tag{"type": "SkillName", "skillName": "Dread Banner"})}
	}),
	`your aura skills are disabled`:     []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Aura})},
	`your blessing skills are disabled`: []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Blessing})},
	`your spells are disabled`:          []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Spell}), flag("ForceEnableCurseApplication")},
	`your warcries are disabled`:        []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Warcry})},
	`your travel skills are disabled`:   []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Travel})},
	`aura skills other than ([a-zA-Z \t\n\v\f\r]+) are disabled`: fn(func(c caps) any {
		return []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Aura}, Tag{"type": "SkillType", "skillType": SkillType.RemoteMined, "neg": true}), flag("EnableSkill", Tag{"type": "SkillName", "skillName": c.s(1)})}
	}),
	`travel skills other than ([a-zA-Z \t\n\v\f\r]+) are disabled`: fn(func(c caps) any {
		return []any{flag("DisableSkill", Tag{"type": "SkillType", "skillType": SkillType.Travel}), flag("EnableSkill", Tag{"type": "SkillId", "skillId": gemIdOrNil(c.s(1))})}
	}),
	`strength's damage bonus instead grants ([0-9]+)% increased melee physical damage per ([0-9]+) strength`: fn(func(c caps) any { return []any{mod("StrDmgBonusRatioOverride", "BASE", c.n(1)/c.n(2))} }),
	`while in her embrace, take ([0-9.]+)% of your total maximum life and energy shield as fire damage per second per level`: fn(func(c caps) any {
		return []any{mod("FireDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "Level"}, Tag{"type": "Condition", "var": "HerEmbrace"}), mod("FireDegen", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Multiplier", "var": "Level"}, Tag{"type": "Condition", "var": "HerEmbrace"})}
	}),
	`gain her embrace for [0-9]+ seconds when you ignite an enemy`: []any{flag("Condition:CanGainHerEmbrace")},
	`when you cast a spell, sacrifice all mana to gain added maximum lightning damage equal to ([0-9]+)% of sacrificed mana for 4 seconds`: fn(func(c caps) any {
		return []any{flag("Condition:HaveManaStorm"), mod("LightningMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaUnreserved", "percent": c.n(1)}, Tag{"type": "Condition", "var": "SacrificeManaForLightning"})}
	}),
	`attacks with this weapon have added maximum lightning damage equal to ([0-9]+)% of your energy shield`: fn(func(c caps) any {
		return []any{mod("LightningMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`attacks with this weapon have added maximum lightning damage equal to ([0-9]+)% of your maximum energy shield`: fn(func(c caps) any {
		return []any{mod("LightningMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`attacks with this weapon have added fire damage equal to ([0-9]+)% of player's maximum life`: fn(func(c caps) any {
		return []any{mod("FireMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1), "actor": "parent"}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), mod("FireMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1), "actor": "parent"}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`adds ([0-9]+)% of your maximum energy shield as cold damage to attacks with this weapon`: fn(func(c caps) any {
		return []any{mod("ColdMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), mod("ColdMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1)}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`attacks with this weapon have added maximum lightning damage equal to ([0-9]+)% of player'?s? maximum energy shield`: fn(func(c caps) any {
		return []any{mod("LightningMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "EnergyShield", "percent": c.n(1), "actor": "parent"}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`adds ([0-9]+)% of your maximum mana as fire damage to attacks with this weapon`: fn(func(c caps) any {
		return []any{mod("FireMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}), mod("FireMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`gain added chaos damage equal to ([0-9]+)% of ward`: fn(func(c caps) any {
		return []any{mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "Ward", "percent": c.n(1)}), mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "Ward", "percent": c.n(1)})}
	}),
	`while you have unbroken ward, your next non-channelling attack you use yourself breaks your ward to gain added cold damage equal to ([0-9]+)% of ward`: fn(func(c caps) any {
		return []any{mod("ColdMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "Ward", "percent": c.n(1)}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "Condition", "var": "WardNotBreak", "neg": true}, Tag{"type": "Condition", "var": "UnbrokenWard"}), mod("ColdMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "Ward", "percent": c.n(1)}, Tag{"type": "SkillType", "skillType": SkillType.Channel, "neg": true}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "Condition", "var": "WardNotBreak", "neg": true}, Tag{"type": "Condition", "var": "UnbrokenWard"})}
	}),
	`spells deal added chaos damage equal to ([0-9]+)% of your maximum life`: fn(func(c caps) any {
		return []any{mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "SkillType", "skillType": SkillType.Spell}), mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)}, Tag{"type": "SkillType", "skillType": SkillType.Spell})}
	}),
	`every 16 seconds you gain iron reflexes for 8 seconds`:                                              []any{flag("Condition:HaveArborix")},
	`every 16 seconds you gain elemental overload for 8 seconds`:                                         []any{flag("Condition:HaveAugyre")},
	`every 8 seconds, gain avatar of fire for 4 seconds`:                                                 []any{flag("Condition:HaveVulconus")},
	`when hit, gain a random movement speed modifier from 40% reduced to 100% increased until hit again`: []any{flag("Condition:HaveGamblesprint")},
	`trigger socketed curse spell when you cast a curse spell, with a ([0-9.]+) second cooldown`:         []any{mod("ExtraSupport", "LIST", Tag{"skillId": "SupportUniqueCastCurseOnCurse", "level": 1}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})},
	`modifiers to attributes instead apply to omniscience`:                                               []any{flag("Omniscience")},
	`attribute requirements can be satisfied by ([0-9]+)% of omniscience`: fn(func(c caps) any {
		return []any{mod("OmniAttributeRequirements", "INC", c.n(1)), flag("OmniscienceRequirements")}
	}),
	`you have far shot while you do not have iron reflexes`:                []any{flag("FarShot", Tag{"neg": true, "type": "Condition", "var": "HaveIronReflexes"})},
	`you have resolute technique while you do not have elemental overload`: []any{mod("Keystone", "LIST", "Resolute Technique", Tag{"neg": true, "type": "Condition", "var": "HaveElementalOverload"})},
	`hits ignore enemy monster fire resistance while you are ignited`:      []any{flag("IgnoreFireResistance", Tag{"type": "Condition", "var": "Ignited"})},
	`your hits can't be evaded by blinded enemies`:                         []any{flag("CannotBeEvaded", Tag{"type": "ActorCondition", "actor": "enemy", "var": "Blinded"})},
	`blind does not affect your chance to hit`:                             []any{flag("UnaffectedByBlind")},
	`unaffected by blind`: []any{flag("UnaffectedByBlind")},
	`enemies blinded by you while you are blinded have malediction`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMalediction", Tag{"type": "Condition", "var": "Blinded"})}, Tag{"type": "Condition", "var": "Blinded"}, Tag{"type": "Condition", "var": "CannotBeBlinded", "neg": true})},
	`enemies blinded by you have malediction`:                       []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMalediction", Tag{"type": "Condition", "var": "Blinded"})})},
	`enemies ignited by you during effect have malediction`:         []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("HasMalediction", Tag{"type": "Condition", "var": "Ignited"})})},
	`skills which throw traps have blood magic`:                     []any{flag("CostLifeInsteadOfMana", Tag{"type": "SkillType", "skillType": SkillType.Trapped})},
	`skills which throw traps cost life instead of mana`:            []any{flag("CostLifeInsteadOfMana", Tag{"type": "SkillType", "skillType": SkillType.Trapped})},
	`strength provides no bonus to maximum life`:                    []any{flag("NoStrBonusToLife")},
	`strength provides no inherent bonus to maximum life`:           []any{flag("NoStrBonusToLife")},
	`intelligence provides no bonus to maximum mana`:                []any{flag("NoIntBonusToMana")},
	`intelligence provides no inherent bonus to maximum mana`:       []any{flag("NoIntBonusToMana")},
	`with a ghastly eye jewel socketed, minions have \+([0-9]+) to accuracy rating`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("Accuracy", "BASE", c.n(1))}, Tag{"type": "Condition", "var": "HaveGhastlyEyeJewelIn{SlotName}"})}
	}),
	`with a ghastly eye jewel socketed, minions have ([0-9]+)% chance to gain unholy might on hit with spells`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": flag("Condition:UnholyMight", Tag{"type": "Condition", "var": "HitSpellRecently"})}, Tag{"type": "Condition", "var": "HaveGhastlyEyeJewelIn{SlotName}"})}
	}),
	`with a hypnotic eye jewel socketed, gain arcane surge on hit with spells`: fn(func(c caps) any {
		return []any{flag("Condition:ArcaneSurge", Tag{"type": "Condition", "var": "HitSpellRecently"}, Tag{"type": "Condition", "var": "HaveHypnoticEyeJewelIn{SlotName}"})}
	}),
	`hits ignore enemy monster chaos resistance if all equipped items are shaper items`: []any{flag("IgnoreChaosResistance", Tag{"type": "MultiplierThreshold", "var": "NonShaperItem", "upper": true, "threshold": 0})},
	`hits ignore enemy monster chaos resistance if all equipped items are elder items`:  []any{flag("IgnoreChaosResistance", Tag{"type": "MultiplierThreshold", "var": "NonElderItem", "upper": true, "threshold": 0})},
	`your hits ignore enemy monster ([a-zA-Z]+) resistances? if all equipped rings are ([a-zA-Z]+) rings`: fn(func(c caps) any {
		return []any{flag("Ignore"+firstToUpper(c.s(1))+"Resistance", Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "RingEquipped", "threshold": 2})}
	}),
	`the stars are aligned if you have 6 influence types among other equipped items`:         []any{flag("Condition:StarsAreAligned", Tag{"type": "MultiplierThreshold", "var": "ShaperItem", "threshold": 2}, Tag{"type": "MultiplierThreshold", "var": "ElderItem", "threshold": 2}, Tag{"type": "MultiplierThreshold", "var": "WarlordItem", "threshold": 2}, Tag{"type": "MultiplierThreshold", "var": "HunterItem", "threshold": 2}, Tag{"type": "MultiplierThreshold", "var": "CrusaderItem", "threshold": 2}, Tag{"type": "MultiplierThreshold", "var": "RedeemerItem", "threshold": 2})},
	`gain [0-9]+ rage on critical hit with attacks, no more than once every [0-9.]+ seconds`: []any{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on critical hit with attacks`:                                          []any{flag("Condition:CanGainRage")},
	`warcry skills' cooldown time is ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "OVERRIDE", c.n(1), nil, 0, KeywordFlag.Warcry)}
	}),
	`non-instant warcries you use yourself have no cooldown`: fn(func(c caps) any {
		// #EVAL: archive parity — the reference writes SkillType.Totem here,
		// which does not exist in Global.lua, so the list carries nil.
		return []any{mod("CooldownRecovery", "OVERRIDE", 0, nil, 0, KeywordFlag.Warcry, Tag{"type": "SkillType", "skillTypeList": []any{SkillType.Instant, nil, SkillType.Triggered}, "neg": true})}
	}),
	`non-instant warcries ignore their cooldown when used`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "OVERRIDE", 0, nil, 0, KeywordFlag.Warcry, Tag{"type": "SkillType", "skillType": SkillType.Instant, "neg": true})}
	}),
	`warcry skills have (\+[0-9]+) seconds to cooldown`:   fn(func(c caps) any { return []any{mod("CooldownRecovery", "BASE", c.n(1), nil, 0, KeywordFlag.Warcry)} }),
	`([0-9]+)% increased total power counted by warcries`: fn(func(c caps) any { return []any{mod("WarcryPower", "INC", c.n(1))} }),
	`warcries have a minimum of ([0-9]+) power`:           fn(func(c caps) any { return []any{mod("MinimumWarcryPower", "BASE", c.n(1))} }),
	`stance skills have (\+[0-9]+) seconds to cooldown`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "BASE", c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Stance})}
	}),
	`using warcries is instant`: []any{flag("InstantWarcry")},
	`attacks with axes or swords grant ([0-9]+) rage on hit, no more than once every second`:                        []any{flag("Condition:CanGainRage", Tag{"type": "Condition", "varList": []any{"UsingAxe", "UsingSword"}})},
	`when you lose temporal chains you gain maximum rage`:                                                           []any{flag("Condition:CanGainRage")},
	`with a murderous eye jewel socketed, melee attacks grant ([0-9]+) rage on hit, no more than once every second`: []any{flag("Condition:CanGainRage", Tag{"type": "Condition", "var": "HaveMurderousEyeJewelIn{SlotName}"})},
	`gain [0-9]+ rage after spending a total of [0-9]+ mana`:                                                        []any{flag("Condition:CanGainRage")},
	`rage grants cast speed instead of attack speed`:                                                                []any{flag("Condition:RageCastSpeed")},
	`rage grants spell damage instead of attack damage`:                                                             []any{flag("Condition:RageSpellDamage")},
	`inherent loss of rage is ([0-9]+)% slower`:                                                                     fn(func(c caps) any { return []any{mod("InherentRageLoss", "INC", -c.n(1))} }),
	`inherent loss of rage is ([0-9]+)% faster`:                                                                     fn(func(c caps) any { return []any{mod("InherentRageLoss", "INC", c.n(1))} }),
	`inherent rage loss starts ([0-9]+) seconds? later`:                                                             fn(func(c caps) any { return []any{mod("InherentRageLossDelay", "BASE", c.n(1))} }),
	`your critical strike multiplier is ([0-9]+)%`:                                                                  fn(func(c caps) any { return []any{mod("CritMultiplier", "OVERRIDE", c.n(1))} }),
	`base critical strike chance for attacks with weapons is ([0-9.]+)%`:                                            fn(func(c caps) any { return []any{mod("WeaponBaseCritChance", "OVERRIDE", c.n(1))} }),
	`base critical strike chance of spells is the critical strike chance of y?o?u?r? ?main hand weapon`:             []any{flag("BaseCritFromMainHand", nil, ModFlag.Spell)},
	`base spell critical strike chance of spells is equal to that of main hand weapon`:                              []any{flag("BaseCritFromMainHand", nil, ModFlag.Spell)},
	`critical strike chance is ([0-9]+)% for hits with this weapon`: fn(func(c caps) any {
		return []any{mod("CritChance", "OVERRIDE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`hits with this weapon have \+([0-9]+)% to critical strike multiplier per enemy power`: fn(func(c caps) any {
		return []any{mod("CritMultiplier", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}, Tag{"type": "Multiplier", "var": "EnemyPower"})}
	}),
	`maximum critical strike chance is ([0-9]+)%`: fn(func(c caps) any { return []any{mod("CritChanceCap", "OVERRIDE", c.n(1))} }),
	`allocates (.+) if you have the matching modifiers? on forbidden (.+)`: fn(func(c caps) any {
		return []any{mod("GrantedAscendancyNode", "LIST", Tag{"side": c.s(2), "name": c.s(1)})}
	}),
	`allocates (.+)`:          fn(func(c caps) any { return []any{mod("GrantedPassive", "LIST", c.s(1))} }),
	`battlemage`:              []any{flag("Battlemage"), mod("MainHandWeaponDamageAppliesToSpells", "MAX", 100)},
	`transfiguration of body`: []any{flag("TransfigurationOfBody")},
	`transfiguration of mind`: []any{flag("TransfigurationOfMind")},
	`transfiguration of soul`: []any{flag("TransfigurationOfSoul")},
	`offering skills have ([0-9]+)% increased duration`: fn(func(c caps) any {
		return []any{mod("Duration", "INC", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`offering skills have ([0-9]+)% reduced duration`: fn(func(c caps) any {
		return []any{mod("Duration", "INC", -c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`enemies have -([0-9]+)% to total physical damage reduction against your hits`:              fn(func(c caps) any { return []any{mod("EnemyPhysicalDamageReduction", "BASE", -c.n(1))} }),
	`enemies you impale have -([0-9]+)% to total physical damage reduction against impale hits`: fn(func(c caps) any { return []any{mod("EnemyImpalePhysicalDamageReduction", "BASE", -c.n(1))} }),
	`hits with this weapon overwhelm ([0-9]+)% physical damage reduction`: fn(func(c caps) any {
		return []any{mod("EnemyPhysicalDamageReduction", "BASE", -c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`overwhelm ([0-9]+)% physical damage reduction`:                        fn(func(c caps) any { return []any{mod("EnemyPhysicalDamageReduction", "BASE", -c.n(1))} }),
	`hits have ([0-9]+)% chance to ignore enemy physical damage reduction`: fn(func(c caps) any { return []any{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", c.n(1))} }),
	`hits with this weapon have ([0-9]+)% chance to ignore enemy physical damage reduction`: fn(func(c caps) any {
		return []any{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", c.n(1), nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`hits with this weapon ignore enemy physical damage reduction`: fn(func(c caps) any {
		return []any{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", 100, nil, ModFlag.Hit, Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack})}
	}),
	`hits against you overwhelm ([0-9]+)% of physical damage reduction`:                            fn(func(c caps) any { return []any{mod("EnemyPhysicalOverwhelm", "BASE", c.n(1))} }),
	`impale damage dealt to enemies impaled by you overwhelms ([0-9]+)% physical damage reduction`: fn(func(c caps) any { return []any{mod("EnemyImpalePhysicalDamageReduction", "BASE", -c.n(1))} }),
	`impale damage dealt to enemies impaled by you ignores enemy physical damage reduction`:        []any{flag("IgnoreEnemyImpalePhysicalDamageReduction")},
	`you are crushed`:                                     []any{flag("Condition:Crushed")},
	`nearby enemies are crushed`:                          []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Crushed")})},
	`crush enemies on hit with maces and sceptres`:        []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Crushed")}, Tag{"type": "Condition", "var": "UsingMace"})},
	`you have fungal ground around you while stationary`:  []any{mod("ExtraAura", "LIST", Tag{"mod": mod("ChaosResist", "BASE", 25)}, Tag{"type": "Condition", "varList": []any{"OnFungalGround", "Stationary"}}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ChaosResist", "BASE", -10)}, Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"OnFungalGround", "Stationary"}}), mod("EnemyModifier", "LIST", Tag{"mod": mod("ElementalResist", "BASE", -10)}, Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"OnFungalGround", "Stationary"}})},
	`create fungal ground instead of consecrated ground`:  []any{flag("Condition:CreateFungalGround")},
	`create profane ground instead of consecrated ground`: []any{flag("Condition:CreateProfaneGround")},
	`([0-9]+)% chance to create profane ground on critical strike if intelligence is your highest attribute`:                []any{flag("Condition:CreateProfaneGround", Tag{"type": "Condition", "var": "IntHighestAttribute"})},
	`consecrated path and purifying flame create profane ground instead of consecrated ground`:                              []any{flag("Condition:CreateProfaneGround")},
	`you gain added cold damage instead of added damage of other types if dexterity exceeds both other attributes`:          []any{flag("AllAddedDamageAsCold", Tag{"type": "Condition", "var": "DexSingleHighestAttribute"})},
	`you gain added lightn?ing damage instead of added damage of other types if intelligence exceeds both other attributes`: []any{flag("AllAddedDamageAsLightning", Tag{"type": "Condition", "var": "IntSingleHighestAttribute"})},
	`elemental hit's added damage cannot be replaced this way`:                                                              d(),
	`you have consecrated ground around you while stationary if strength is your highest attribute`:                         []any{flag("Condition:OnConsecratedGround", Tag{"type": "Condition", "var": "StrHighestAttribute"}, Tag{"type": "Condition", "var": "Stationary"})},
	`consecrated ground around you while stationary if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{flag("Condition:OnConsecratedGround", Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "Item", "threshold": c.n(1)})}
	}),
	`you count as dual wielding while you are unencumbered`:                              []any{flag("Condition:DualWielding", Tag{"type": "Condition", "var": "Unencumbered"})},
	`dual wielding does not inherently grant chance to block attack damage`:              []any{flag("Condition:NoInherentBlock")},
	`inherent attack speed bonus from dual wielding is doubled while wielding two claws`: []any{flag("Condition:DoubledInherentDualWieldingSpeed", Tag{"type": "Condition", "var": "DualWieldingClaws"})},
	`inherent bonuses from dual wielding are doubled`:                                    []any{flag("Condition:DoubledInherentDualWieldingSpeed"), flag("Condition:DoubledInherentDualWieldingBlock")},
	`([0-9]+)% reduced enemy chance to block sword attacks`:                              fn(func(c caps) any { return []any{mod("reduceEnemyBlock", "BASE", c.n(1), nil, ModFlag.Sword)} }),
	`you do not inherently take less damage for having fortification`:                    []any{flag("Condition:NoFortificationMitigation")},
	`skills supported by intensify have \+([0-9]) to maximum intensity`:                  fn(func(c caps) any { return []any{mod("Multiplier:IntensityLimit", "BASE", c.n(1))} }),
	`spells which can gain intensity have \+([0-9]) to maximum intensity`:                fn(func(c caps) any { return []any{mod("Multiplier:IntensityLimit", "BASE", c.n(1))} }),
	`final repeat of spells has ([0-9]+)% increased area of effect`: fn(func(c caps) any {
		return []any{mod("RepeatFinalAreaOfEffect", "INC", c.n(1), nil, ModFlag.Spell, 0, Tag{"type": "Condition", "var": "CastOnFrostbolt", "neg": true}, Tag{"type": "Condition", "varList": []any{"averageRepeat", "alwaysFinalRepeat"}})}
	}),
	`hexes you inflict have ([+\-][0-9]+) to maximum doom`: fn(func(c caps) any { return []any{mod("MaxDoom", "BASE", c.n(1))} }),
	`while stationary, gain ([0-9]+)% increased area of effect every second, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Multiplier", "var": "StationarySeconds", "globalLimit": c.n(2), "globalLimitKey": "ExpansiveMight"}, Tag{"type": "Condition", "var": "Stationary"})}
	}),
	`fireball and rolling magma have ([0-9]+)% more area of effect`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "MORE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Fireball", "Rolling Magma"}})}
	}),
	`attack skills have added lightning damage equal to ([0-9]+)% of maximum mana`: fn(func(c caps) any {
		return []any{mod("LightningMin", "BASE", 1, nil, ModFlag.Attack, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)}), mod("LightningMax", "BASE", 1, nil, ModFlag.Attack, Tag{"type": "PercentStat", "stat": "Mana", "percent": c.n(1)})}
	}),
	`arc and crackling lance gains added cold damage equal to ([0-9]+)% of mana cost, if mana cost is not higher than the maximum you could spend`: fn(func(c caps) any {
		return []any{mod("ColdMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaCost", "percent": c.n(1)}, Tag{"type": "SkillName", "skillNameList": []any{"Arc", "Crackling Lance"}, "includeTransfigured": true}), mod("ColdMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaCost", "percent": c.n(1)}, Tag{"type": "SkillName", "skillNameList": []any{"Arc", "Crackling Lance"}, "includeTransfigured": true})}
	}),
	`forbidden rite and dark pact gains added chaos damage equal to ([0-9]+)% of mana cost, if mana cost is not higher than the maximum you could spend`: fn(func(c caps) any {
		return []any{mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaCost", "percent": c.n(1)}, Tag{"type": "SkillName", "skillNameList": []any{"Forbidden Rite", "Dark Bargain"}, "includeTransfigured": true}), mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaCost", "percent": c.n(1)}, Tag{"type": "SkillName", "skillNameList": []any{"Forbidden Rite", "Dark Bargain"}, "includeTransfigured": true})}
	}),
	`skills gain added chaos damage equal to ([0-9]+)% of mana cost, if mana cost is not higher than the maximum you could spend`: fn(func(c caps) any {
		return []any{mod("ChaosMin", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaCost", "percent": c.n(1)}), mod("ChaosMax", "BASE", 1, Tag{"type": "PercentStat", "stat": "ManaCost", "percent": c.n(1)})}
	}),
	`herald of thunder's storms hit enemies with ([0-9]+)% increased frequency`:                       fn(func(c caps) any { return []any{mod("HeraldStormFrequency", "INC", c.n(1))} }),
	`storms hit enemies with ([0-9]+)% increased frequency`:                                           fn(func(c caps) any { return []any{mod("HeraldStormFrequency", "INC", c.n(1))} }),
	`your critical strikes have a ([0-9]+)% chance to deal double damage`:                             fn(func(c caps) any { return []any{mod("DoubleDamageChanceOnCrit", "BASE", c.n(1))} }),
	`elemental skills deal triple damage`:                                                             []any{mod("TripleDamageChance", "BASE", 100, Tag{"type": "SkillType", "skillTypeList": []any{SkillType.Cold, SkillType.Fire, SkillType.Lightning}})},
	`deal triple damage with elemental skills`:                                                        []any{mod("TripleDamageChance", "BASE", 100, Tag{"type": "SkillType", "skillTypeList": []any{SkillType.Cold, SkillType.Fire, SkillType.Lightning}})},
	`skills supported by unleash have \+([0-9]) to maximum number of seals`:                           fn(func(c caps) any { return []any{mod("SealCount", "BASE", c.n(1))} }),
	`left ring slot: skills supported by unleash have \+([0-9]) to maximum number of seals`:           fn(func(c caps) any { return []any{mod("SealCount", "BASE", c.n(1), Tag{"type": "SlotNumber", "num": 1})} }),
	`skills supported by unleash have ([0-9]+)% increased seal gain frequency`:                        fn(func(c caps) any { return []any{mod("SealGainFrequency", "INC", c.n(1))} }),
	`([0-9]+)% increased critical strike chance with spells which remove the maximum number of seals`: fn(func(c caps) any { return []any{mod("MaxSealCrit", "INC", c.n(1))} }),
	`gain elusive on critical strike`:                                                                 []any{flag("Condition:CanBeElusive")},
	`gain a random shrine buff every ([0-9]+) seconds`:                                                []any{flag("Condition:CanHaveRegularShrines")},
	`gain a random shrine buff for ([0-9]+) seconds when you kill a rare or unique enemy`:             []any{flag("Condition:CanHaveRegularShrines")},
	`([0-9]+)% chance to gain elusive when you block while dual wielding`:                             []any{flag("Condition:CanBeElusive", Tag{"type": "Condition", "var": "DualWielding"})},
	`elusive on you reduces in effect ([0-9]+)% slower`:                                               fn(func(c caps) any { return []any{mod("ElusiveEffectLossSlower", "INC", c.n(1))} }),
	`elusive's effect on you is increased instead for the first ([0-9]+) seconds`:                     fn(func(c caps) any { return []any{mod("ElusiveEffectIncreaseDuration", "BASE", c.n(1))} }),
	`elusive is removed from you at ([0-9]+)% effect`:                                                 fn(func(c caps) any { return []any{mod("ElusiveEffectMinThreshold", "OVERRIDE", c.n(1))} }),
	`nearby enemies have ([a-zA-Z]+) resistance equal to yours`:                                       fn(func(c caps) any { return []any{flag("Enemy" + firstToUpper(c.s(1)) + "ResistEqualToYours")} }),
	`for each nearby corpse, regenerate ([0-9.]+)% life per second, up to ([0-9.]+)%`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "NearbyCorpse", "limit": c.n(2), "limitTotal": true})}
	}),
	`gain sacrificial zeal when you use a skill, dealing you [0-9]+% of the skill's mana cost as physical damage per second`: []any{flag("SacrificialZeal")},
	`skills gain a base life cost equal to ([0-9]+)% of base mana cost`:                                                      fn(func(c caps) any { return []any{mod("ManaCostAsLifeCost", "BASE", c.n(1))} }),
	`skills gain a base energy shield cost equal to ([0-9]+)% of base mana cost`:                                             fn(func(c caps) any { return []any{mod("ManaCostAsEnergyShieldCost", "BASE", c.n(1))} }),
	`skills cost life instead of ([0-9]+)% of mana cost`:                                                                     fn(func(c caps) any { return []any{mod("HybridManaAndLifeCost_Life", "BASE", c.n(1))} }),
	`([0-9]+)% increased cost of arc and crackling lance`: fn(func(c caps) any {
		return []any{mod("Cost", "INC", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Arc", "Crackling Lance"}, "includeTransfigured": true})}
	}),
	`hits overwhelm ([0-9]+)% of physical damage reduction while you have sacrificial zeal`: fn(func(c caps) any {
		return []any{mod("EnemyPhysicalDamageReduction", "BASE", -c.n(1), nil, Tag{"type": "Condition", "var": "SacrificialZeal"})}
	}),
	`([0-9]+)% chance for hits to ignore enemy physical damage reduction while you have sacrificial zeal`: fn(func(c caps) any {
		return []any{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", c.n(1), nil, Tag{"type": "Condition", "var": "SacrificialZeal"})}
	}),
	`hits have ([0-9]+)% chance to ignore enemy physical damage reduction while you have sacrificial zeal`: fn(func(c caps) any {
		return []any{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", c.n(1), nil, Tag{"type": "Condition", "var": "SacrificialZeal"})}
	}),
	`minions attacks overwhelm ([0-9]+)% physical damage reduction`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("EnemyPhysicalDamageReduction", "BASE", -c.n(1), Tag{"type": "SkillType", "skillType": SkillType.Attack})})}
	}),
	`minions hits have ([0-9]+)% chance to ignore enemy physical damage reduction`: fn(func(c caps) any {
		return []any{mod("MinionModifier", "LIST", Tag{"mod": mod("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", c.n(1))})}
	}),
	`focus has ([0-9]+)% increased cooldown recovery rate`: fn(func(c caps) any {
		return []any{mod("FocusCooldownRecovery", "INC", c.n(1), Tag{"type": "Condition", "var": "Focused"})}
	}),
	`focus has ([0-9]+)% reduced cooldown recovery rate`: fn(func(c caps) any {
		return []any{mod("FocusCooldownRecovery", "INC", -c.n(1), Tag{"type": "Condition", "var": "Focused"})}
	}),
	`([0-9]+)% more frozen legion and general's cry cooldown recovery rate`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "MORE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Frozen Legion", "General's Cry"}, "includeTransfigured": true})}
	}),
	`flamethrower, seismic and lightning spire trap have ([0-9]+)% increased cooldown recovery rate`: fn(func(c caps) any {
		return []any{mod("CooldownRecovery", "INC", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Flamethrower Trap", "Seismic Trap", "Lightning Spire Trap"}, "includeTransfigured": true})}
	}),
	`flamethrower, seismic and lightning spire trap have -([0-9]+) cooldown uses?`: fn(func(c caps) any {
		return []any{mod("AdditionalCooldownUses", "BASE", -c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Flamethrower Trap", "Seismic Trap", "Lightning Spire Trap"}, "includeTransfigured": true})}
	}),
	`right ring slot: shockwave has \+([0-9]+) to cooldown uses?`: fn(func(c caps) any {
		return []any{mod("AdditionalCooldownUses", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Shockwave"}, Tag{"type": "SlotNumber", "num": 2})}
	}),
	`flameblast starts with ([0-9]+) additional stages`: fn(func(c caps) any {
		return []any{mod("Multiplier:FlameblastMinimumStage", "BASE", c.n(1), 0, 0, Tag{"type": "GlobalEffect", "effectType": "Buff", "unscalable": true})}
	}),
	`incinerate starts with ([0-9]+) additional stages`: fn(func(c caps) any {
		return []any{mod("Multiplier:IncinerateMinimumStage", "BASE", c.n(1), 0, 0, Tag{"type": "GlobalEffect", "effectType": "Buff", "unscalable": true})}
	}),
	`\+([0-9.]+) seconds to flameblast and incinerate cooldown`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SkillData", "LIST", Tag{"key": "cooldown", "value": 0})}, Tag{"type": "SkillName", "skillNameList": []any{"Incinerate", "Flameblast"}, "includeTransfigured": true}), mod("CooldownRecovery", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Incinerate", "Flameblast"}, "includeTransfigured": true})}
	}),
	`([0-9]+)% chance to deal double damage with attacks if attack time is longer than 1 second`: fn(func(c caps) any {
		return []any{mod("DoubleDamageChance", "BASE", c.n(1), 0, 0, Tag{"type": "Condition", "var": "OneSecondAttackTime"})}
	}),
	`elusive also grants \+([0-9]+)% to critical strike multiplier for skills supported by nightblade`: fn(func(c caps) any { return []any{mod("NightbladeElusiveCritMultiplier", "BASE", c.n(1))} }),
	`skills supported by nightblade have ([0-9]+)% increased effect of elusive`:                        fn(func(c caps) any { return []any{mod("NightbladeSupportedElusiveEffect", "INC", c.n(1))} }),
	`nearby enemies are scorched`: []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Scorched")}), mod("ScorchBase", "BASE", 10)},
	`hits have ([0-9]+)% chance to ignore enemy monster physical damage reduction`: fn(func(c caps) any { return []any{mod("PartialIgnoreEnemyPhysicalDamageReduction", "BASE", c.n(1))} }),
	`attacks you use yourself have ([0-9]+)% more attack speed`: fn(func(c caps) any {
		return []any{mod("Speed", "MORE", c.n(1), nil, ModFlag.Attack, 0, Tag{"type": "SkillType", "neg": true, "skillTypeList": []any{SkillType.SummonsTotem, SkillType.RemoteMined, SkillType.Trapped, SkillType.Triggered}}, Tag{"type": "Condition", "neg": true, "var": "usedByMirage"})}
	}),
	`attacks you use yourself repeat an additional time`:        []any{mod("RepeatCount", "BASE", 1, nil, ModFlag.Attack, 0, Tag{"type": "SkillType", "neg": true, "skillTypeList": []any{SkillType.SummonsTotem, SkillType.RemoteMined, SkillType.Trapped, SkillType.Triggered}}, Tag{"type": "Condition", "neg": true, "var": "usedByMirage"}, Tag{"type": "Condition", "varList": []any{"averageRepeat", "alwaysFinalRepeat"}})},
	`final repeat of attack skills deals ([0-9]+)% more damage`: fn(func(c caps) any { return []any{mod("RepeatFinalDamage", "MORE", c.n(1), nil, 0, KeywordFlag.Attack)} }),
	`non-travel attack skills repeat an additional time`:        []any{mod("RepeatCount", "BASE", 1, nil, 0, KeywordFlag.Attack, Tag{"type": "SkillType", "skillType": SkillType.Travel, "neg": true}, Tag{"type": "Condition", "varList": []any{"averageRepeat", "alwaysFinalRepeat"}})},
	`viper strike and pestilent strike deal ([0-9]+)% increased attack damage per frenzy charge`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Attack, Tag{"type": "Multiplier", "var": "FrenzyCharge"}, Tag{"type": "SkillName", "skillNameList": []any{"Viper Strike", "Pestilent Strike"}, "includeTransfigured": true})}
	}),
	`shield charge and chain hook have ([0-9]+)% increased attack speed per ([0-9]+) rampage kills`: fn(func(c caps) any {
		return []any{mod("Speed", "INC", c.n(1), nil, ModFlag.Attack, Tag{"type": "Multiplier", "var": "Rampage", "div": c.s(2), "limit": 1000 / c.n(2), "limitTotal": true}, Tag{"type": "SkillName", "skillNameList": []any{"Shield Charge", "Chain Hook"}, "includeTransfigured": true})}
	}),
	`tectonic slam and infernal blow deal ([0-9]+)% increased attack damage per ([0-9]+) armour`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), nil, ModFlag.Attack, Tag{"type": "PerStat", "stat": "Armour", "div": c.s(2)}, Tag{"type": "SkillName", "skillNameList": []any{"Tectonic Slam", "Infernal Blow"}, "includeTransfigured": true})}
	}),
	`frozen sweep deals ([0-9]+)% less damage`: fn(func(c caps) any {
		return []any{mod("Damage", "MORE", -c.n(1), Tag{"type": "SkillName", "skillName": "Frozen Sweep", "includeTransfigured": true})}
	}),
	`ice trap and lightning trap damage penetrates ([0-9]+)% of enemy elemental resistances`: fn(func(c caps) any {
		return []any{mod("LightningPenetration", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Ice Trap", "Lightning Trap"}, "includeTransfigured": true}), mod("ColdPenetration", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Ice Trap", "Lightning Trap"}, "includeTransfigured": true}), mod("FirePenetration", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Ice Trap", "Lightning Trap"}, "includeTransfigured": true})}
	}),
	`volatile dead and cremation penetrate ([0-9]+)% fire resistance per ([0-9]+) dexterity`: fn(func(c caps) any {
		return []any{mod("FirePenetration", "BASE", c.n(1), Tag{"type": "PerStat", "stat": "Dex", "div": c.s(2)}, Tag{"type": "SkillName", "skillNameList": []any{"Volatile Dead", "Cremation"}, "includeTransfigured": true})}
	}),
	`regenerate ([0-9]+) mana per second while any enemy is in your righteous fire or scorching ray`: fn(func(c caps) any {
		return []any{mod("ManaRegen", "BASE", c.n(1), Tag{"type": "Condition", "var": "InRFOrScorchingRay"})}
	}),
	`\+([0-9]+)% to wave of conviction damage over time multiplier per ([0-9.]+) seconds of duration expired`:               fn(func(c caps) any { return []any{mod("WaveOfConvictionDurationDotMulti", "INC", c.n(1))} }),
	`when an enemy hit deals elemental damage to you, their resistance to those elements becomes zero for ([0-9]+) seconds`: []any{flag("Condition:HaveTrickstersSmile")},
	// Conditional Player Quantity / Rarity
	`([0-9]+)% increased quantity of items dropped by slain normal enemies`: fn(func(c caps) any { return []any{mod("LootQuantityNormalEnemies", "INC", c.n(1))} }),
	`([0-9]+)% increased rarity of items dropped by slain magic enemies`:    fn(func(c caps) any { return []any{mod("LootRarityMagicEnemies", "INC", c.n(1))} }),
	// Pantheon: Soul of Tukohama support
	`while stationary, gain ([0-9.]+)% of life regenerated per second every second, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "StationarySeconds", "limit": c.n(2), "limitTotal": true}, Tag{"type": "Condition", "var": "Stationary"})}
	}),
	// Pantheon: Soul of Ryslatha support
	`life flasks gain ([0-9]+) charges? every ([0-9]+) seconds if you haven't used a life flask recently`: fn(func(c caps) any {
		return []any{mod("LifeFlaskChargesGenerated", "BASE", c.n(1)/c.n(2), Tag{"type": "Condition", "var": "UsingLifeFlask", "neg": true})}
	}),
	// Skill-specific enchantment modifiers
	`([0-9]+)% increased decoy totem life`: fn(func(c caps) any {
		return []any{mod("TotemLife", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Decoy Totem"})}
	}),
	`([0-9]+)% increased ice spear critical strike chance in second form`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Ice Spear", "includeTransfigured": true}, Tag{"type": "SkillPart", "skillPartList": []any{2, 4}})}
	}),
	`shock nova ring deals ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Shock Nova", "includeTransfigured": true}, Tag{"type": "SkillPart", "skillPart": 1})}
	}),
	`enemies affected by bear trap take ([0-9]+)% increased damage from trap or mine hits`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("EnemyModifier", "LIST", Tag{"mod": mod("TrapMineDamageTaken", "INC", c.n(1), Tag{"type": "GlobalEffect", "effectType": "Debuff"})})}, Tag{"type": "SkillName", "skillName": "Bear Trap", "includeTransfigured": true})}
	}),
	`blade vortex has \+([0-9]+)% to critical strike multiplier for each blade`: fn(func(c caps) any {
		return []any{mod("CritMultiplier", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "BladeVortexBlade"}, Tag{"type": "SkillName", "skillName": "Blade Vortex", "includeTransfigured": true})}
	}),
	`burning arrow has ([0-9]+)% increased debuff effect`: fn(func(c caps) any {
		return []any{mod("DebuffEffect", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Burning Arrow"})}
	}),
	`double strike has a ([0-9]+)% chance to deal double damage to bleeding enemies`: fn(func(c caps) any {
		return []any{mod("DoubleDamageChance", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}, Tag{"type": "SkillName", "skillName": "Double Strike", "includeTransfigured": true})}
	}),
	`frost bomb has ([0-9]+)% increased debuff duration`: fn(func(c caps) any {
		return []any{mod("SecondaryDuration", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Frost Bomb"})}
	}),
	`incinerate has \+([0-9]+) to maximum stages`: fn(func(c caps) any {
		return []any{mod("Multiplier:IncinerateMaxStages", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Incinerate"})}
	}),
	`perforate creates \+([0-9]+) spikes?`: fn(func(c caps) any { return []any{mod("Multiplier:PerforateMaxSpikes", "BASE", c.n(1))} }),
	`scourge arrow has ([0-9]+)% chance to poison per stage`: fn(func(c caps) any {
		return []any{mod("PoisonChance", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Scourge Arrow", "includeTransfigured": true}, Tag{"type": "Multiplier", "var": "ScourgeArrowStage"})}
	}),
	`winter orb has \+([0-9]+) maximum stages`: fn(func(c caps) any { return []any{mod("Multiplier:WinterOrbMaxStages", "BASE", c.n(1))} }),
	`summoned holy relics have ([0-9]+)% increased buff effect`: fn(func(c caps) any {
		return []any{mod("BuffEffect", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Summon Holy Relic", "includeTransfigured": true})}
	}),
	`\+([0-9]+) to maximum virulence`: fn(func(c caps) any { return []any{mod("Multiplier:VirulenceStacksMax", "BASE", c.n(1))} }),
	`winter orb has ([0-9]+)% increased area of effect per stage`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Winter Orb"}, Tag{"type": "Multiplier", "var": "WinterOrbStage"})}
	}),
	`wintertide brand has \+([0-9]+) to maximum stages`: fn(func(c caps) any {
		return []any{mod("Multiplier:WintertideBrandMaxStages", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Wintertide Brand"})}
	}),
	`wave of conviction's exposure applies (-[0-9]+)% elemental resistance`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "purge_expose_resist_%_matching_highest_element_damage", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Wave of Conviction"})}
	}),
	`wave of conviction's exposure applies an extra (-[0-9]+)% to elemental resistance`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "purge_expose_resist_%_matching_highest_element_damage", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Wave of Conviction"})}
	}),
	`arcane cloak spends an additional ([0-9]+)% of current mana`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "arcane_cloak_consume_%_of_mana", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Arcane Cloak"})}
	}),
	`arcane cloak grants life regeneration equal to ([0-9]+)% of mana spent per second`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("LifeRegen", "BASE", c.n(1)/100, 0, 0, Tag{"type": "Multiplier", "var": "ArcaneCloakConsumedMana"}, Tag{"type": "GlobalEffect", "effectType": "Buff"})}, Tag{"type": "SkillName", "skillName": "Arcane Cloak"})}
	}),
	`caustic arrow has ([0-9]+)% chance to inflict withered on hit for ([0-9]+) seconds base duration`: []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("Condition:CanWither")}, Tag{"type": "SkillName", "skillName": "Caustic Arrow", "includeTransfigured": true})},
	`venom gyre has a ([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit`:                []any{mod("ExtraSkillMod", "LIST", Tag{"mod": flag("Condition:CanWither")}, Tag{"type": "SkillName", "skillName": "Venom Gyre"})},
	`sigil of power's buff also grants ([0-9]+)% increased critical strike chance per stage`: fn(func(c caps) any {
		return []any{mod("CritChance", "INC", c.n(1), 0, 0, Tag{"type": "Multiplier", "var": "SigilOfPowerStage", "limit": 4}, Tag{"type": "GlobalEffect", "effectType": "Buff", "effectName": "Sigil of Power"})}
	}),
	`cobra lash chains ([0-9]+) additional times`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ChainCountMax", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Cobra Lash"})}
	}),
	`general's cry has ([+\-][0-9]) to maximum number of mirage warriors`: fn(func(c caps) any { return []any{mod("GeneralsCryDoubleMaxCount", "BASE", c.n(1))} }),
	`([+\-][0-9]) to maximum blade flurry stages`: fn(func(c caps) any {
		return []any{mod("Multiplier:BladeFlurryMaxStages", "BASE", c.n(1)), mod("Multiplier:BladeFlurryofIncisionMaxStages", "BASE", c.n(1))}
	}),
	`steelskin buff can take ([0-9]+)% increased amount of damage`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "steelskin_damage_limit_+%", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Steelskin"})}
	}),
	`hydrosphere has ([0-9]+)% increased pulse frequency`: fn(func(c caps) any { return []any{mod("HydroSphereFrequency", "INC", c.n(1))} }),
	`void sphere has ([0-9]+)% increased pulse frequency`: fn(func(c caps) any { return []any{mod("VoidSphereFrequency", "INC", c.n(1))} }),
	`shield crush central wave has ([0-9]+)% more area of effect`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "MORE", c.n(1), Tag{"type": "SkillName", "skillName": "Shield Crush", "includeTransfigured": true}, Tag{"type": "SkillPart", "skillPart": 2})}
	}),
	`storm rain has ([0-9]+)% increased beam frequency`:                                 fn(func(c caps) any { return []any{mod("StormRainBeamFrequency", "INC", c.n(1))} }),
	`voltaxic burst deals ([0-9]+)% increased damage per ([0-9.]+) seconds of duration`: fn(func(c caps) any { return []any{mod("VoltaxicDurationIncDamage", "INC", c.n(1))} }),
	`earthquake deals ([0-9]+)% increased damage per ([0-9.]+) seconds duration`:        fn(func(c caps) any { return []any{mod("EarthquakeDurationIncDamage", "INC", c.n(1))} }),
	`consecrated ground from holy flame totem applies ([0-9]+)% increased damage taken to enemies`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod("DamageTakenConsecratedGround", "INC", c.n(1), Tag{"type": "Condition", "var": "OnConsecratedGround"})})}
	}),
	`consecrated ground from purifying flame applies ([0-9]+)% increased damage taken to enemies`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "consecrated_ground_enemy_damage_taken_+%", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Purifying Flame", "includeTransfigured": true})}
	}),
	`enemies drenched by hydrosphere have cold and lightning exposure, applying (-[0-9]+)% to resistances`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "water_sphere_cold_lightning_exposure_%", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Hydrosphere"})}
	}),
	`frost shield has \+([0-9]+) to maximum life per stage`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "frost_globe_health_per_stage", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Frost Shield"})}
	}),
	`flame wall grants ([0-9]+) to ([0-9]+) added fire damage to projectiles`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "flame_wall_minimum_added_fire_damage", "value": c.s(1)}, Tag{"type": "SkillName", "skillName": "Flame Wall"}), mod("ExtraSkillStat", "LIST", Tag{"key": "flame_wall_maximum_added_fire_damage", "value": c.s(2)}, Tag{"type": "SkillName", "skillName": "Flame Wall"})}
	}),
	`plague bearer buff grants \+([0-9]+)% to poison damage over time multiplier while infecting`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "corrosive_shroud_poison_dot_multiplier_+_while_aura_active", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Plague Bearer"})}
	}),
	`([0-9]+)% increased lightning trap lightning ailment effect`: fn(func(c caps) any {
		return []any{mod("ExtraSkillStat", "LIST", Tag{"key": "shock_effect_+%", "value": c.n(1)}, Tag{"type": "SkillName", "skillName": "Lightning Trap", "includeTransfigured": true})}
	}),
	`wild strike's beam chains an additional ([0-9]+) times`: fn(func(c caps) any {
		return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ChainCountMax", "BASE", c.n(1))}, Tag{"type": "SkillName", "skillName": "Wild Strike", "includeTransfigured": true}, Tag{"type": "SkillPart", "skillPart": 4})}
	}),
	`energy blades have ([0-9]+)% increased attack speed`: fn(func(c caps) any { return []any{mod("EnergyBladeAttackSpeed", "INC", c.n(1))} }),
	`ensnaring arrow has ([0-9]+)% increased debuff effect`: fn(func(c caps) any {
		return []any{mod("DebuffEffect", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Ensnaring Arrow"})}
	}),
	`unearth spawns corpses with ([+\-][0-9]) level`: fn(func(c caps) any {
		return []any{mod("CorpseLevel", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Unearth"})}
	}),
	`seismic trap releases an additional wave`:        []any{mod("MaximumWaves", "BASE", 1, Tag{"type": "SkillName", "skillName": "Seismic Trap", "includeTransfigured": true})},
	`lightning spire trap strikes an additional area`: []any{mod("MaximumWaves", "BASE", 1, Tag{"type": "SkillName", "skillName": "Lightning Spire Trap", "includeTransfigured": true})},
	`explosive trap causes ([0-9]+) additional smaller explosions`: fn(func(c caps) any {
		return []any{mod("SmallExplosions", "BASE", c.n(1), Tag{"type": "SkillName", "skillNameList": []any{"Explosive Trap", "Explosive Trap of Swells"}})}
	}),
	`frozen sweep deals ([0-9]+)% increased damage`: fn(func(c caps) any {
		return []any{mod("Damage", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Frozen Sweep", "includeTransfigured": true})}
	}),
	`([0-9]+)% increased attack speed with snipe`: fn(func(c caps) any {
		return []any{mod("Speed", "INC", c.n(1), nil, ModFlag.Attack, Tag{"type": "SkillName", "skillName": "Snipe"})}
	}),
	`\+([0-9]+) to maximum snipe stages`: fn(func(c caps) any {
		return []any{mod("Multiplier:SnipeStagesMax", "BASE", c.n(1), 0, 0, Tag{"type": "GlobalEffect", "effectType": "Buff", "unscalable": true})}
	}),
	`chain hook has \+([0-9.]+) metres? to radius per ([0-9]+) rage`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "BASE", c.n(1)*10, Tag{"type": "PerStat", "stat": "Rage", "div": c.n(2)}, Tag{"type": "SkillName", "skillName": "Chain Hook", "includeTransfigured": true})}
	}),
	`\+([0-9.]+) metres? to discharge radius`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "BASE", c.n(1)*10, Tag{"type": "SkillName", "skillName": "Discharge", "includeTransfigured": true})}
	}),
	// Alternate Quality
	`quality does not increase physical damage`:                 []any{mod("AlternateQualityWeapon", "BASE", 1)},
	`([0-9]+)% increased critical strike chance per 4% quality`: fn(func(c caps) any { return []any{mod("AlternateQualityLocalCritChancePer4Quality", "INC", c.n(1))} }),
	`grants ([0-9]+)% increased accuracy per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("Accuracy", "INC", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`([0-9]+)% increased attack speed per 8% quality`:     fn(func(c caps) any { return []any{mod("AlternateQualityLocalAttackSpeedPer8Quality", "INC", c.n(1))} }),
	`\+([0-9]+) weapon range per 10% quality`:             fn(func(c caps) any { return []any{mod("AlternateQualityLocalWeaponRangePer10Quality", "BASE", c.n(1))} }),
	`\+([0-9.]+) metres? to weapon range per 10% quality`: fn(func(c caps) any { return []any{mod("AlternateQualityLocalWeaponRangePer10Quality", "BASE", c.n(1)*10)} }),
	`grants ([0-9]+)% increased elemental damage per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("ElementalDamage", "INC", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants ([0-9]+)% increased area of effect per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("AreaOfEffect", "INC", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`quality does not increase defences`: []any{mod("AlternateQualityArmour", "BASE", 1)},
	`grants \+([0-9]+) to maximum life per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("Life", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+) to maximum mana per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("Mana", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+) to strength per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("Str", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+) to dexterity per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("Dex", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+) to intelligence per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("Int", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+)% to fire resistance per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("FireResist", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+)% to cold resistance per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("ColdResist", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`grants \+([0-9]+)% to lightning resistance per ([0-9]+)% quality`: fn(func(c caps) any {
		return []any{mod("LightningResist", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "QualityOn{SlotName}", "div": c.n(2)})}
	}),
	`\+([0-9]+)% to quality`: fn(func(c caps) any { return []any{mod("Quality", "BASE", c.n(1))} }),
	`infernal blow debuff deals an additional ([0-9]+)% of damage per charge`: fn(func(c caps) any {
		return []any{mod("DebuffEffect", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Infernal Blow", "includeTransfigured": true})}
	}),
	// Legion modifiers
	`passives in radius are conquered by the ([^0-9]+)`: d(),
	`passives affected are conquered by the abyssal`:    d(),
	`historic`: d(),
	// Tattoos
	`\+([0-9]+) to maximum life per allocated journey tattoo of the body`: fn(func(c caps) any {
		return []any{mod("Life", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "JourneyTattooBody"}), mod("Multiplier:JourneyTattooBody", "BASE", 1)}
	}),
	`\+([0-9]+) to maximum energy shield per allocated journey tattoo of the soul`: fn(func(c caps) any {
		return []any{mod("EnergyShield", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "JourneyTattooSoul"}), mod("Multiplier:JourneyTattooSoul", "BASE", 1)}
	}),
	`\+([0-9]+) to maximum mana per allocated journey tattoo of the mind`: fn(func(c caps) any {
		return []any{mod("Mana", "BASE", c.n(1), Tag{"type": "Multiplier", "var": "JourneyTattooMind"}), mod("Multiplier:JourneyTattooMind", "BASE", 1)}
	}),
	// Display-only modifiers
	`extra gore`: d(),
	`prefixes:`:  d(),
	`suffixes:`:  d(),
	`while your passive skill tree connects to a class' starting location, you gain:`:          d(),
	`socketed lightning spells [hd][ae][va][el] ([0-9]+)% increased spell damage if triggered`: d(),
	`manifeste?d? dancing dervishe?s? disables both weapon slots`:                              d(),
	`manifeste?d? dancing dervishe?s? dies? when rampage ends`:                                 d(),
	`survival`: d(),
	`you can have two different banners at the same time`:                d(),
	`[+\-]([0-9]+) prefix modifiers? allowed`:                            d(),
	`[+\-]([0-9]+) suffix modifiers? allowed`:                            d(),
	`can have a second enchantment modifier`:                             d(),
	`can have ([0-9]+) additional enchantment modifiers`:                 d(),
	`this item can be anointed by cassia`:                                d(),
	`can be anointed`:                                                    d(),
	`implicit modifiers cannot be changed`:                               d(),
	`has a crucible passive skill tree`:                                  d(),
	`has elder, shaper and all conqueror influences`:                     d(),
	`has a two handed sword crucible passive skill tree`:                 d(),
	`has a crucible passive skill tree with only support passive skills`: d(),
	`crucible passive skill tree is removed if this modifier is removed`: d(),
	`all sockets are white`:                                              d(),
	`cannot roll ([a-zA-Z]+) modifiers`:                                  d(),
	`cannot roll modifiers of non-([a-zA-Z]+) damage types`:              d(),
	`every ([0-9]+) seconds, regenerate ([0-9]+)% of life over one second`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(2), Tag{"type": "Condition", "var": "LifeRegenBurstFull"}), mod("LifeRegenPercent", "BASE", c.n(2)/c.n(1), Tag{"type": "Condition", "var": "LifeRegenBurstAvg"})}
	}),
	`every ([0-9]+) seconds, regenerate ([0-9]+)% of life over one second if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return []any{mod("LifeRegenPercent", "BASE", c.n(2), Tag{"type": "Condition", "var": "LifeRegenBurstFull"}, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(4)) + "Item", "threshold": c.n(3)}), mod("LifeRegenPercent", "BASE", c.n(2)/c.n(1), Tag{"type": "Condition", "var": "LifeRegenBurstAvg"}, Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(4)) + "Item", "threshold": c.n(3)})}
	}),
	`take no extra damage from critical strikes`:                                            []any{mod("ReduceCritExtraDamage", "BASE", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})},
	`take no extra damage from critical strikes if you have a magic ring in left slot`:      []any{mod("ReduceCritExtraDamage", "BASE", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "Condition", "var": "MagicItemInRing 1"})},
	`take no extra damage from critical strikes if energy shield recharge started recently`: []any{mod("ReduceCritExtraDamage", "BASE", 100, Tag{"type": "Condition", "var": "EnergyShieldRechargeRecently"})},
	`you take ([0-9]+)% reduced extra damage from critical strikes while affected by determination`: fn(func(c caps) any {
		return []any{mod("ReduceCritExtraDamage", "BASE", c.n(1), Tag{"type": "Condition", "var": "AffectedByDetermination"})}
	}),
	`you take ([0-9]+)% reduced extra damage from critical strikes`:   fn(func(c caps) any { return []any{mod("ReduceCritExtraDamage", "BASE", c.n(1))} }),
	`you take ([0-9]+)% increased extra damage from critical strikes`: fn(func(c caps) any { return []any{mod("ReduceCritExtraDamage", "BASE", -c.n(1))} }),
	`you take ([0-9]+)% reduced extra damage from critical strikes while you have no power charges`: fn(func(c caps) any {
		return []any{mod("ReduceCritExtraDamage", "BASE", c.n(1), Tag{"type": "StatThreshold", "stat": "PowerCharges", "threshold": 0, "upper": true})}
	}),
	`you take ([0-9]+)% reduced extra damage from critical strikes by poisoned enemies`: fn(func(c caps) any {
		return []any{mod("ReduceCritExtraDamage", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"})}
	}),
	`you take ([0-9]+)% reduced extra damage from critical strikes by cursed enemies`: fn(func(c caps) any {
		return []any{mod("ReduceCritExtraDamage", "BASE", c.n(1), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})}
	}),
	`nearby allies have ([0-9]+)% chance to block attack damage per ([0-9]+) strength you have`: fn(func(c caps) any {
		return []any{mod("ExtraAura", "LIST", Tag{"onlyAllies": true, "mod": mod("BlockChance", "BASE", c.n(1))}, Tag{"type": "PerStat", "stat": "Str", "div": c.n(2)})}
	}),
	`physical skills have ([0-9]+)% increased duration per ([0-9]+) intelligence`: fn(func(c caps) any {
		return []any{mod("Duration", "INC", c.n(1), nil, nil, KeywordFlag.Physical, Tag{"type": "PerStat", "stat": "Int", "div": c.n(2)})}
	}),
	`y?o?u?r? ?maximum energy shield is equal to ([0-9]+)% of y?o?u?r? ?maximum life`: fn(func(c caps) any {
		return []any{mod("EnergyShield", "OVERRIDE", 1, Tag{"type": "PercentStat", "stat": "Life", "percent": c.n(1)})}
	}),
	`immun[ei]t?y? to elemental ailments while bleeding`: []any{flag("ElementalAilmentImmune", Tag{"type": "Condition", "var": "Bleeding"})},
	`mana is increased by ([0-9]+)% of overcapped lightning resistance`: fn(func(c caps) any {
		return []any{flag("ManaIncreasedByOvercappedLightningRes"), mod("Mana", "INC", c.n(1)/100, Tag{"type": "PerStat", "stat": "LightningResistOverCap"})}
	}),
	// handled in item parsing
	`[0-9]+% [ir][ne][cd][ru][ec][ae][sd]e?d? ?[a-zA-Z \t\n\v\f\r]* modifier magnitudes`: d(),
	`[0-9]+% [ir][ne][cd][ru][ec][ae][sd]e?d? effect of [sp][ur][fe]fixes`:               d(),
	`[a-zA-Z \t\n\v\f\r]* modifier magnitudes are doubled`:                               d(),
}
