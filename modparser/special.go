package modparser

// specialModList (transformed entries) — ModParser.lua:2120-5880. The
// closures needing real statements live in special_hand.go.
var specialModListData = map[string]modsValue{
	// Explode mods
	`non*?aura curses you inflict are not removed from dying enemies`: modList{},
	`enemies near corpses affected by your curses are blinded`:        modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Blinded")}, &MultiplierTag{IsThreshold: true, Var: "NearbyCorpse", Threshold: opt(1)}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})},
	// Keystones
	`([0-9]+) rage regenerated for every ([0-9]+) mana regeneration per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("RageRegen", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ManaRegen", Div: opt(c.n(2))}), flag("Condition:CanGainRage")}
	}),
	`([0-9a-zA-Z]+) recovery from regeneration is not applied`: modFn(func(c caps) []*Mod { return []*Mod{flag("UnaffectedBy" + firstToUpper(c.s(1)) + "Regen")} }),
	`([0-9]+)% less damage taken for every ([0-9]+)% life recovery per second from leech`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DamageTaken", More, Num(-c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "MaxLifeLeechRatePercent", Div: opt(c.n(2))})}
	}),
	`([0-9a-zA-Z]+) recovery from non-instant leech is not applied`: modFn(func(c caps) []*Mod { return []*Mod{flag("UnaffectedByNonInstant" + firstToUpper(c.s(1)) + "Leech")} }),
	`([0-9]+)% additional physical damage reduction for every ([0-9]+)% life recovery per second from leech`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageReduction", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "MaxLifeLeechRatePercent", Div: opt(c.n(2))}, &CondTag{Var: "Leeching"})}
	}),
	`modifiers to chance to suppress spell damage instead apply to chance to dodge spell hits at 50% of their value`: modList{flag("ConvertSpellSuppressionToSpellDodge"), mods("SpellSuppressionChance", Override, Num(0), "Acrobatics")},
	`chance to evade hits is based off of ([0-9]+)% of your ward instead of your evasion rating`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("EvadeChanceBasedOnWard"), mods("EvadeChanceBasedOnWardPercent", Override, Num(c.n(1)), "Black Scythe Training")}
	}),
	`physical damage reduction from hits is based off of ([0-9]+)% of your ward instead of your armour`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("PhysicalReductionBasedOnWard"), mods("PhysicalReductionBasedOnWardPercent", Override, Num(c.n(1)), "Black Scythe Training")}
	}),
	`maximum chance to dodge spell hits is ([0-9]+)%`:             modFn(func(c caps) []*Mod { return []*Mod{mods("SpellDodgeChanceMax", Override, Num(c.n(1)), "Acrobatics")} }),
	`dexterity provides no bonus to evasion rating`:               modList{flag("NoDexBonusToEvasion")},
	`dexterity provides no inherent bonus to evasion rating`:      modList{flag("NoDexBonusToEvasion")},
	`strength's damage bonus applies to all spell damage as well`: modList{flag("IronWill")},
	`your hits can't be evaded`:                                   modList{flag("CannotBeEvaded")},
	`your melee hits can't be evaded while wielding a sword`:      modList{flagf("CannotBeEvaded", FlagMelee|FlagHit, KeywordNone, &CondTag{Var: "UsingSword"})},
	`minion hits can't be evaded`:                                 modList{mod("MinionModifier", List, ModRef{Mod: flag("CannotBeEvaded")})},
	`never deal critical strikes`:                                 modList{flag("NeverCrit"), flag("Condition:NeverCrit")},
	`minions never deal critical strikes`:                         modList{mod("MinionModifier", List, ModRef{Mod: flag("NeverCrit")}), mod("MinionModifier", List, ModRef{Mod: flag("Condition:NeverCrit")})},
	`never deal critical strikes with spells`:                     modList{flagf("NeverCrit", FlagSpell, KeywordNone), flagf("Condition:NeverCrit", FlagSpell, KeywordNone)},
	`never deal critical strikes with attacks`:                    modList{flagf("NeverCrit", FlagAttack, KeywordNone), flagf("Condition:NeverCrit", FlagAttack, KeywordNone)},
	`cannot deal critical strikes`:                                modList{flag("NeverCrit"), flag("Condition:NeverCrit")},
	`cannot deal critical strikes with spells`:                    modList{flagf("NeverCrit", FlagSpell, KeywordNone), flagf("Condition:NeverCrit", FlagSpell, KeywordNone)},
	`cannot deal critical strikes with attacks`:                   modList{flagf("NeverCrit", FlagAttack, KeywordNone), flagf("Condition:NeverCrit", FlagAttack, KeywordNone)},
	`no critical strike multiplier`:                               modList{flag("NoCritMultiplier")},
	`ailments never count as being from critical strikes`:         modList{flag("AilmentsAreNeverFromCrit")},
	`the increase to physical damage from strength applies to projectile attacks as well as melee attacks`:                        modList{flag("IronGrip")},
	`strength's damage bonus applies to projectile attack damage as well as melee damage`:                                         modList{flag("IronGrip")},
	`converts all evasion rating to armour\. dexterity provides no bonus to evasion rating`:                                       modList{flag("NoDexBonusToEvasion"), flag("IronReflexes")},
	`30% chance to dodge attack hits\. 50% less armour, 30% less energy shield, 30% less chance to block spell and attack damage`: modList{mod("AttackDodgeChance", Base, Num(30)), mod("Armour", More, Num(-50)), mod("EnergyShield", More, Num(-30)), mod("BlockChance", More, Num(-30)), mod("SpellBlockChance", More, Num(-30))},
	`([0-9]+)% increased blind effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("BlindEffect", Inc, Num(c.n(1)))})}
	}),
	`([0-9]+)% increased effect of blind from melee weapons`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("BlindEffect", Inc, Num(c.n(1)))})}
	}),
	`\+([0-9]+)% chance to block spell damage for each ([0-9]+)% overcapped chance to block attack damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellBlockChance", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "BlockChanceOverCap", Div: opt(c.n(2))})}
	}),
	`maximum life becomes 1, immune to chaos damage`:           modList{flag("ChaosInoculation"), mod("ChaosDamageTaken", More, Num(-100))},
	`life regeneration is applied to energy shield instead`:    modList{flag("ZealotsOath")},
	`life leeched per second is doubled`:                       modList{mod("LifeLeechRate", More, Num(100))},
	`life regeneration has no effect`:                          modList{flag("NoLifeRegen")},
	`energy shield recharge instead applies to life`:           modList{flag("EnergyShieldRechargeAppliesToLife")},
	`blade vortex and blade blast deal no non-physical damage`: modList{flag("DealNoLightning", &SkillNameTag{SkillNameList: []string{"Blade Vortex", "Blade Blast"}, IncludeTransfigured: true}), flag("DealNoCold", &SkillNameTag{SkillNameList: []string{"Blade Vortex", "Blade Blast"}, IncludeTransfigured: true}), flag("DealNoFire", &SkillNameTag{SkillNameList: []string{"Blade Vortex", "Blade Blast"}, IncludeTransfigured: true}), flag("DealNoChaos", &SkillNameTag{SkillNameList: []string{"Blade Vortex", "Blade Blast"}, IncludeTransfigured: true})},
	`([0-9]+)% of physical, cold and lightning damage converted to fire damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageConvertToFire", Base, Num(c.n(1))), mod("LightningDamageConvertToFire", Base, Num(c.n(1))), mod("ColdDamageConvertToFire", Base, Num(c.n(1)))}
	}),
	`all elemental damage converted to chaos damage`:           modList{mod("ColdDamageConvertToChaos", Base, Num(100)), mod("FireDamageConvertToChaos", Base, Num(100)), mod("LightningDamageConvertToChaos", Base, Num(100))},
	`removes all mana\. spend life instead of mana for skills`: modList{mod("Mana", More, Num(-100)), flag("CostLifeInsteadOfMana")},
	`removes all mana`:                                          modList{mod("Mana", More, Num(-100))},
	`removes all energy shield`:                                 modList{mod("EnergyShield", More, Num(-100))},
	`skills cost life instead of mana`:                          modList{flag("CostLifeInsteadOfMana")},
	`skills reserve life instead of mana`:                       modList{flag("BloodMagicReserved")},
	`your skills that throw mines reserve life instead of mana`: modList{flagf("BloodMagicReserved", FlagNone, KeywordMine)},
	`curse skills cost life instead of mana`:                    modList{flagf("CostLifeInsteadOfMana", FlagNone, KeywordCurse)},
	`curse aura skills reserve life instead of mana`:            modList{flagf("BloodMagicReserved", FlagNone, KeywordCurse, &SkillTypeTag{SkillType: SkillTypeAura})},
	`your travel skills critically strike once every 3 uses`:    modList{flag("Every3UseCrit", &SkillTypeTag{SkillType: SkillTypeTravel})},
	`non-aura skills cost no mana or life while focus?sed`:      modList{mod("ManaCost", More, Num(-100), &CondTag{Var: "Focused"}, &SkillTypeTag{SkillType: SkillTypeAura, Neg: true}), mod("LifeCost", More, Num(-100), &CondTag{Var: "Focused"}, &SkillTypeTag{SkillType: SkillTypeAura, Neg: true})},
	`spend life instead of mana for effects of skills`:          modList{},
	`skills cost \+([0-9]+) rage`:                               modFn(func(c caps) []*Mod { return []*Mod{mod("RageCostBase", Base, Num(c.n(1)))} }),
	`warcries cost \+([0-9]+)% of life`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeCostBase", Base, Num(1), FlagNone, KeywordWarcry, &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1)), Floor: true})}
	}),
	`vaal skills used during effect have ([0-9]+)% reduced soul gain prevention duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SoulGainPreventionDuration", Inc, Num(-c.n(1)), &CondTag{Var: "UsingFlask"}, &SkillTypeTag{SkillType: SkillTypeVaal})}
	}),
	`vaal volcanic fissure and vaal molten strike have ([0-9]+)% reduced soul gain prevention duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SoulGainPreventionDuration", Inc, Num(-c.n(1)), &SkillNameTag{SkillNameList: []string{"Volcanic Fissure", "Molten Strike"}, IncludeTransfigured: true}, &SkillTypeTag{SkillType: SkillTypeVaal})}
	}),
	`vaal skills can store \+([0-9]+) uses?`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AdditionalUses", Base, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeVaal})}
	}),
	`vaal attack skills cost rage instead of requiring souls to use`:           modList{flagf("CostRageInsteadOfSouls", FlagAttack, KeywordNone, &SkillTypeTag{SkillType: SkillTypeVaal})},
	`vaal attack skills you use yourself cost rage instead of requiring souls`: modList{flagf("CostRageInsteadOfSouls", FlagAttack, KeywordNone, &SkillTypeTag{SkillType: SkillTypeVaal})},
	`you cannot gain rage during soul gain prevention`:                         modList{mod("RageRegen", More, Num(-100), &CondTag{Var: "SoulGainPrevention"})},
	`hits that deal elemental damage remove exposure to those elements and inflict exposure to other elements exposure inflicted this way applies (-[0-9]+)% to resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ElementalEquilibrium"), mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(c.n(1)), &CondTag{VarList: []string{"HitByColdDamage", "HitByLightningDamage"}})}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(c.n(1)), &CondTag{VarList: []string{"HitByFireDamage", "HitByLightningDamage"}})}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(c.n(1)), &CondTag{VarList: []string{"HitByFireDamage", "HitByColdDamage"}})})}
	}),
	`projectile attack hits deal up to 30% more damage to targets at the start of their movement, dealing less damage to targets as the projectile travels farther`: modList{flag("PointBlank")},
	`leech energy shield instead of life`: modList{flag("GhostReaver")},
	`minions explode when reduced to low life, dealing 33% of their maximum life as fire damage to surrounding enemies`:                                modList{mod("ExtraMinionSkill", List, SkillRef{SkillID: "MinionInstability"})},
	`minions explode when reduced to low life, dealing 33% of their life as fire damage to surrounding enemies`:                                        modList{mod("ExtraMinionSkill", List, SkillRef{SkillID: "MinionInstability"})},
	`all bonuses from an equipped shield apply to your minions instead of you`:                                                                         modList{},
	`spend energy shield before mana for skill m?a?n?a? ?costs`:                                                                                        modList{},
	`you have perfect agony if you've dealt a critical strike recently`:                                                                                modList{mod("Keystone", List, Str("Perfect Agony"), &CondTag{Var: "CritRecently"})},
	`energy shield protects mana instead of life`:                                                                                                      modList{flag("EnergyShieldProtectsMana")},
	`modifiers to critical strike multiplier also apply to damage over time multiplier for ailments from critical strikes at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod { return []*Mod{mod("CritMultiplierAppliesToDegen", Base, Num(c.n(1)))} }),
	`damage over time multiplier for ailments is equal to critical strike multiplier`:                                                                  modList{flag("DotMultiplierIsCritMultiplier")},
	`\+([0-9]+)% to cold damage over time multiplier for each ([0-9]+)% overcapped cold resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ColdDotMultiplier", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ColdResistOverCap", Div: opt(c.n(2))})}
	}),
	`your bleeding does not deal extra damage while the enemy is moving`:                          modList{flag("Condition:NoExtraBleedDamageToMovingEnemy")},
	`your bleeding does not deal extra damage while the enemy is moving and cannot be aggravated`: modList{flag("Condition:NoExtraBleedDamageToMovingEnemy"), flag("Condition:CannotAggravate")},
	`you and enemies in your presence count as moving while affected by elemental ailments`:       modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Moving", &CondTag{VarList: []string{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}})}), flag("Condition:Moving", &CondTag{VarList: []string{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}})},
	`you can inflict bleeding on an enemy up to ([0-9]+) times?`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BleedStacksMax", Override, Num(c.n(1))), flag("Condition:HaveCrimsonDance")}
	}),
	`your minions spread caustic ground on death, dealing ([0-9]+)% of their maximum life as chaos damage per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraMinionSkill", List, SkillRef{SkillID: "SiegebreakerCausticGround"}), mod("MinionModifier", List, ModRef{Mod: mod("Multiplier:SiegebreakerCausticGroundPercent", Base, Num(c.n(1)))})}
	}),
	`your minions spread burning ground on death, dealing ([0-9]+)% of their maximum life as fire damage per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraMinionSkill", List, SkillRef{SkillID: "ReplicaSiegebreakerBurningGround"}), mod("MinionModifier", List, ModRef{Mod: mod("Multiplier:SiegebreakerBurningGroundPercent", Base, Num(c.n(1)))})}
	}),
	`you can have an additional brand attached to an enemy`: modList{mod("BrandsAttachedLimit", Base, Num(1))},
	`gain ([0-9]+) grasping vines each second while stationary`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Multiplier:GraspingVinesCount", Base, Num(c.n(1)), &MultiplierTag{Var: "StationarySeconds", Limit: opt(10), LimitTotal: true}, &CondTag{Var: "Stationary"})}
	}),
	`all damage inflicts poison against enemies affected by at least ([0-9]+) grasping vines`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PoisonChance", Base, Num(100), &MultiplierTag{IsThreshold: true, Var: "GraspingVinesAffectingEnemy", Threshold: opt(c.n(1))}), flag("FireCanPoison", &MultiplierTag{IsThreshold: true, Var: "GraspingVinesAffectingEnemy", Threshold: opt(c.n(1))}), flag("ColdCanPoison", &MultiplierTag{IsThreshold: true, Var: "GraspingVinesAffectingEnemy", Threshold: opt(c.n(1))}), flag("LightningCanPoison", &MultiplierTag{IsThreshold: true, Var: "GraspingVinesAffectingEnemy", Threshold: opt(c.n(1))})}
	}),
	`attack projectiles always inflict bleeding and maim, and knock back enemies`: modList{modf("BleedChance", Base, Num(100), FlagAttack|FlagProjectile, KeywordNone), modf("EnemyKnockbackChance", Base, Num(100), FlagAttack|FlagProjectile, KeywordNone)},
	`projectiles cannot pierce, fork or chain`:                                    modList{flagf("CannotPierce", FlagProjectile, KeywordNone), flagf("CannotChain", FlagProjectile, KeywordNone), flagf("CannotFork", FlagProjectile, KeywordNone)},
	`projectiles cannot continue after colliding with targets`:                    modList{flagf("CannotPierce", FlagProjectile, KeywordNone), flagf("CannotChain", FlagProjectile, KeywordNone), flagf("CannotFork", FlagProjectile, KeywordNone), flagf("CannotSplit", FlagProjectile, KeywordNone)},
	`critical strikes inflict scorch, brittle and sapped`:                         modList{flag("CritAlwaysAltAilments")},
	`chance to block attack damage is doubled`:                                    modList{mod("BlockChance", More, Num(100))},
	`chance to block spell damage is doubled`:                                     modList{mod("SpellBlockChance", More, Num(100))},
	`you take ([0-9]+)% of damage from blocked hits`:                              modFn(func(c caps) []*Mod { return []*Mod{mod("BlockEffect", Base, Num(c.n(1)))} }),
	`ignore attribute requirements`:                                               modList{flag("IgnoreAttributeRequirements")},
	`ignore attribute requirements of gems socketed in blue sockets`:              modList{mod("GemProperty", List, GemPropertyRef{Keyword: "all", Key: "req", Value: opt(0)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", SocketColor: "B"})},
	`ignore attribute requirements of socketed gems`:                              modList{mod("GemProperty", List, GemPropertyRef{Keyword: "all", Key: "req", Value: opt(0)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`gain no inherent bonuses from attributes`:                                    modList{flag("NoAttributeBonuses")},
	`gain no inherent bonuses from strength`:                                      modList{flag("NoStrengthAttributeBonuses")},
	`gain no inherent bonuses from dexterity`:                                     modList{flag("NoDexterityAttributeBonuses")},
	`gain no inherent bonuses from intelligence`:                                  modList{flag("NoIntelligenceAttributeBonuses")},

	`physical damage taken bypasses energy shield`:                                               modList{mod("PhysicalEnergyShieldBypass", Base, Num(100))},
	`auras from your skills do not affect allies`:                                                modList{flag("SelfAuraSkillsCannotAffectAllies")},
	`auras from your skills have ([0-9]+)% more effect on you`:                                   modFn(func(c caps) []*Mod { return []*Mod{mod("SkillAuraEffectOnSelf", More, Num(c.n(1)))} }),
	`auras from your skills have ([0-9]+)% increased effect on you`:                              modFn(func(c caps) []*Mod { return []*Mod{mod("SkillAuraEffectOnSelf", Inc, Num(c.n(1)))} }),
	`increases and reductions to mana regeneration rate instead apply to rage regeneration rate`: modList{flag("ManaRegenToRageRegen")},
	`increases and reductions to maximum energy shield instead apply to ward`:                    modList{flag("EnergyShieldToWard")},
	`([0-9]+)% of damage taken bypasses ward`:                                                    modFn(func(c caps) []*Mod { return []*Mod{mod("WardBypass", Base, Num(c.n(1)))} }),
	`ward has a ([0-9]+)% chance to not break`:                                                   modFn(func(c caps) []*Mod { return []*Mod{mod("WardAvoidBreakChance", Base, Num(c.n(1)))} }),
	`damage taken bypasses unbroken ward if the hit deals less damage than ([0-9]+)% of ward`:    modFn(func(c caps) []*Mod { return []*Mod{mod("WardBypassBelowPercent", Base, Num(c.n(1)))} }),
	`maximum energy shield is ([0-9]+)`:                                                          modFn(func(c caps) []*Mod { return []*Mod{mod("EnergyShield", Override, Num(c.n(1)))} }),
	`while not on full life, sacrifice ([0-9.]+)% of mana per second to recover that much life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaDegenPercent", Base, Num(c.n(1)), &CondTag{Var: "FullLife", Neg: true}), mod("LifeRecovery", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &CondTag{Var: "FullLife", Neg: true})}
	}),
	`([0-9]+)% increased maximum energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShield", Inc, Num(c.n(1)), &MarkerTag{Marker: TagGlobal})}
	}),
	`you are blind`: modList{flag("Condition:Blinded", &CondTag{Var: "CannotBeBlinded", Neg: true})},
	`armour applies to fire, cold and lightning damage taken from hits instead of physical damage`:                                          modList{mod("ArmourAppliesToFireDamageTaken", Base, Num(100)), mod("ArmourAppliesToColdDamageTaken", Base, Num(100)), mod("ArmourAppliesToLightningDamageTaken", Base, Num(100)), flag("ArmourDoesNotApplyToPhysicalDamageTaken")},
	`([0-9]+)% of armour also applies to chaos damage taken from hits`:                                                                      modFn(func(c caps) []*Mod { return []*Mod{mod("ArmourAppliesToChaosDamageTaken", Base, Num(c.n(1)))} }),
	`armour also applies to chaos damage taken from hits`:                                                                                   modFn(func(c caps) []*Mod { return []*Mod{mod("ArmourAppliesToChaosDamageTaken", Base, Num(100))} }),
	`maximum damage reduction for any damage type is ([0-9]+)%`:                                                                             modFn(func(c caps) []*Mod { return []*Mod{mod("DamageReductionMax", Override, Num(c.n(1)))} }),
	`gain additional elemental damage reduction equal to half your chaos resistance`:                                                        modList{mod("ElementalDamageReduction", Base, Num(1), &StatTag{StatKind: TagPerStat, Stat: "ChaosResist", Div: opt(2)})},
	`([0-9]+)% of maximum mana is converted to twice that much armour`:                                                                      modFn(func(c caps) []*Mod { return []*Mod{mod("ManaConvertToArmour", Base, Num(c.n(1)))} }),
	`life recovery from flasks also applies to energy shield`:                                                                               modList{flag("LifeFlaskAppliesToEnergyShield")},
	`increase to cast speed from arcane surge also applies to movement speed`:                                                               modList{flag("ArcaneSurgeCastSpeedToMovementSpeed")},
	`arcane surge also grants ([0-9]+)% increased life regeneration rate to you`:                                                            modFn(func(c caps) []*Mod { return []*Mod{mod("ArcaneSurgeAlsoLifeRegen", Base, Num(c.n(1)))} }),
	`increases and reductions to effect of flasks applied to you also applies to effect of arcane surge on you at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod { return []*Mod{mod("FlaskEffectToArcaneSurgeEffect", Base, Num(c.n(1)))} }),
	`non-instant mana recovery from flasks is also recovered as life`:                                                                       modList{flag("ManaFlaskAppliesToLife")},
	`life leech effects recover energy shield instead while on full life`:                                                                   modList{flag("ImmortalAmbition", &CondTag{Var: "FullLife"}, &CondTag{Var: "LeechingLife"})},
	`shepherd of souls`: modList{flag("ShepherdOfSouls")},
	`you have shepherd of souls if at least ([0-9]+) corrupted items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ShepherdOfSouls", &MultiplierTag{IsThreshold: true, Var: "CorruptedItem", Threshold: opt(c.n(1))})}
	}),
	`you have everlasting sacrifice if at least ([0-9]+) corrupted items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Condition:EverlastingSacrifice", &MultiplierTag{IsThreshold: true, Var: "CorruptedItem", Threshold: opt(c.n(1))})}
	}),
	`adds ([0-9]+) to ([0-9]+) attack physical damage to melee skills per ([0-9]+) dexterity while you are unencumbered`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalMin", Base, Num(c.n(1)), FlagMelee, KeywordAttack, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(3))}, &CondTag{Var: "Unencumbered"}), modf("PhysicalMax", Base, Num(c.n(2)), FlagMelee, KeywordAttack, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(3))}, &CondTag{Var: "Unencumbered"})}
	}),
	`adds ([0-9]+) to ([0-9]+) fire damage to attacks for every ([0-9]+)% your light radius is above base value`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("FireMin", Base, Num(c.n(1)), FlagAttack, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "LightRadiusInc", Div: opt(c.n(3))}), modf("FireMax", Base, Num(c.n(2)), FlagAttack, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "LightRadiusInc", Div: opt(c.n(3))})}
	}),
	`([0-9]+)% more attack damage if accuracy rating is higher than maximum life`: modFn(func(c caps) []*Mod {
		return []*Mod{modsf("Damage", More, Num(c.n(1)), "Damage", FlagAttack, KeywordNone, &CondTag{Var: "MainHandAccRatingHigherThanMaxLife"}, &CondTag{Var: "MainHandAttack"}), modsf("Damage", More, Num(c.n(1)), "Damage", FlagAttack, KeywordNone, &CondTag{Var: "OffHandAccRatingHigherThanMaxLife"}, &CondTag{Var: "OffHandAttack"})}
	}),
	`your hexes have infinite duration`: modList{mod("Duration", Base, Num(m_huge), &SkillTypeTag{SkillType: SkillTypeAppliesCurse})},
	// Legacy support
	// Masteries
	`hits have ([0-9]+)% chance to treat enemy monster elemental resistance values as inverted`: modFn(func(c caps) []*Mod { return []*Mod{mod("HitsInvertEleResChance", Chance, Num(c.n(1)/100))} }),
	`off hand accuracy is equal to main hand accuracy while wielding a sword`:                   modList{flag("Condition:OffHandAccuracyIsMainHandAccuracy", &CondTag{Var: "UsingSword"})},
	`([0-9]+)% increased accuracy rating at close range`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", Inc, Num(c.n(1)), &CondTag{Var: "AtCloseRange"})}
	}),
	`([0-9]+)% more accuracy rating at close range`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", More, Num(c.n(1)), &CondTag{Var: "AtCloseRange"})}
	}),
	`([0-9]+)% increased accuracy rating against unique enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})}
	}),
	`([0-9]+)% more accuracy rating against unique enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", More, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})}
	}),
	`defend with ([0-9]+)% of armour while not on low energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mods("ArmourDefense", Max, Num(c.n(1)-100), "Armour and Energy Shield Mastery", &CondTag{Var: "LowEnergyShield", Neg: true})}
	}),
	`([0-9]+)% increased armour and energy shield from equipped body armour if equipped helmet, gloves and boots all have armour and energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Body ArmourESAndArmour", Inc, Num(c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnGloves", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EnergyShieldOnGloves", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnHelmet", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EnergyShieldOnHelmet", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnBoots", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EnergyShieldOnBoots", Threshold: opt(1)})}
	}),
	`brands have ([0-9]+)% increased area of effect if ([0-9]+)% of attached duration expired`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &CondTag{Var: "BrandLastHalf"}, &SkillTypeTag{SkillType: SkillTypeBrand})}
	}),
	`corrupted blood cannot be inflicted on you`: modList{flag("CorruptedBloodImmune")},
	`you cannot be hindered`:                     modList{flag("HinderImmune")},
	`you cannot be maimed`:                       modList{flag("MaimImmune")},
	`you cannot be impaled`:                      modList{flag("ImpaleImmune")},
	// Exerted Attacks
	`exerted attacks deal ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod { return []*Mod{modf("ExertIncrease", Inc, Num(c.n(1)), FlagAttack, KeywordNone)} }),
	`exerted attacks have ([0-9]+)% chance to deal double damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ExertDoubleDamageChance", Base, Num(c.n(1)), FlagAttack, KeywordNone)}
	}),
	// Ascendant
	`grants ([0-9]+) passive skill points?`:                     modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraPoints", Base, Num(c.n(1)))} }),
	`can allocate passives from the [a-zA-Z]+'s starting point`: modList{},
	`projectiles gain damage as they travel farther, dealing up to ([0-9]+)% increased damage with hits to targets`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagHit|FlagProjectile, KeywordNone, &DistanceRampTag{Ramp: Pairs{{35, 0}, {70, 1}}})}
	}),
	`([0-9]+)% chance to gain elusive on kill`:                        modList{flag("Condition:CanBeElusive")},
	`immun[ei]t?y? to elemental ailments while on consecrated ground`: modList{flag("ElementalAilmentImmune", &CondTag{Var: "OnConsecratedGround"})},
	// Assassin
	`poison you inflict with critical strikes deals ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordPoison, &CondTag{Var: "CriticalStrike"})}
	}),
	`([0-9]+)% chance to gain elusive on critical strike`: modList{flag("Condition:CanBeElusive")},
	`([0-9]+)% more damage while there is at most one rare or unique enemy nearby`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordNone, &CondTag{Var: "AtMostOneNearbyRareOrUniqueEnemy"})}
	}),
	`([0-9]+)% reduced damage taken while there are at least two rare or unique enemies nearby`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("DamageTaken", Inc, Num(-c.n(1)), FlagNone, KeywordNone, &MultiplierTag{IsThreshold: true, Var: "NearbyRareOrUniqueEnemies", Threshold: opt(2)})}
	}),
	`([0-9]+)% less damage taken while there are at least two rare or unique enemies nearby`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("DamageTaken", More, Num(-c.n(1)), FlagNone, KeywordNone, &MultiplierTag{IsThreshold: true, Var: "NearbyRareOrUniqueEnemies", Threshold: opt(2)})}
	}),
	`you take no extra damage from critical strikes while elusive`: modList{mod("ReduceCritExtraDamage", Base, Num(100), &CondTag{Var: "Elusive"})},
	`mark skills cost no mana`:                                     modList{modf("ManaCost", More, Num(-100), FlagNone, KeywordNone, &SkillTypeTag{SkillType: SkillTypeMark})},
	// Berserker
	`gain [0-9]+ rage when you kill an enemy`:                                       modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage when you use a warcry`:                                        modList{flag("Condition:CanGainRage")},
	`you and nearby party members gain [0-9]+ rage when you warcry`:                 modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with attacks, no more than once every [0-9.]+ seconds`: modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on attack hit`:                                                modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with axes or swords`:                                   modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on melee hit`:                                                 modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on melee weapon hit`:                                          modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with axes`:                                             modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage when hit by an enemy`:                                         modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on hit with retaliation skills`:                               modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage when you use a life flask`:                                    modList{flag("Condition:CanGainRage")},
	`while a unique enemy is in your presence, gain [0-9]+ rage on hit with attacks, no more than once every [0-9.]+ seconds`:        modList{flag("Condition:CanGainRage", &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})},
	`while a pinnacle atlas boss is in your presence, gain [0-9]+ rage on hit with attacks, no more than once every [0-9.]+ seconds`: modList{flag("Condition:CanGainRage", &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"})},
	`maximum rage is halved`:                        modList{mod("MaximumRage", More, Num(-50))},
	`inherent effects from having rage are tripled`: modList{mod("RageEffect", More, Num(200))},
	`inherent effects from having rage are doubled`: modList{mod("RageEffect", More, Num(100))},
	`cannot be stunned while you have at least ([0-9]+) rage`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("StunImmune", &MultiplierTag{IsThreshold: true, Var: "Rage", Threshold: opt(c.n(1))})}
	}),
	`([0-9]+)% less damage taken per ([0-9]+) rage, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DamageTaken", More, Num(-c.n(1)), &MultiplierTag{Var: "Rage", Div: opt(c.n(2)), Limit: opt(-c.n(3)), LimitNegTotal: true})}
	}),
	`lose ([0-9.]+)% of life per second per rage while you are not losing rage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{Var: "RageEffect"})}
	}),
	`if you've warcried recently, you and nearby allies have ([0-9]+)% increased attack speed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: modf("Speed", Inc, Num(c.n(1)), FlagAttack, KeywordNone)}, &CondTag{Var: "UsedWarcryRecently"})}
	}),
	`gain ([0-9]+)% increased armour per ([0-9]+) power for 8 seconds when you warcry, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Armour", Inc, Num(c.n(1)), &MultiplierTag{Var: "WarcryPower", Div: opt(c.n(2)), GlobalLimit: opt(c.n(3)), GlobalLimitKey: "WarningCall"}, &CondTag{Var: "UsedWarcryInPast8Seconds"})}
	}),
	`warcries grant ([0-9]+) rage per ([0-9]+) power if you have less than ([0-9]+) rage`: modList{flag("Condition:CanGainRage")},
	`warcries grant ([0-9]+) rage per ([0-9]+) enemy power, up to ([0-9]+)`:               modList{flag("Condition:CanGainRage")},
	`exerted attacks deal ([0-9]+)% more attack damage if a warcry sacrificed rage recently`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ExertAttackIncrease", More, Num(c.n(1)), FlagAttack, KeywordNone)}
	}),
	`deal ([0-9]+)% less damage`:           modFn(func(c caps) []*Mod { return []*Mod{mod("Damage", More, Num(-c.n(1)))} }),
	`warcries exert twice as many attacks`: modList{mod("ExtraExertedAttacks", More, Num(100))},
	// Champion
	`cannot be stunned while you have fortify`:                 modList{flag("StunImmune", &CondTag{Var: "Fortified"})},
	`cannot be stunned while fortified`:                        modList{flag("StunImmune", &CondTag{Var: "Fortified"})},
	`you cannot be stunned while at maximum endurance charges`: modList{flag("StunImmune", &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMax"})},
	`fortify`:                             modList{flag("Condition:Fortified")},
	`you have your maximum fortification`: modList{flag("Condition:Fortified"), flag("Condition:HaveMaxFortification")},
	`you have ([0-9]+) fortification`:     modFn(func(c caps) []*Mod { return []*Mod{mod("MinimumFortification", Base, Num(c.n(1)))} }),
	`nearby allies count as having fortification equal to yours`: modList{mod("ExtraAura", List, ModRef{Mod: mod("YourFortifyEqualToParent", Flag, Bool(true), &GlobalEffectTag{EffectType: "Global", Unscalable: true}), OnlyAllies: true})},
	`enemies taunted by you cannot evade attacks`:                modList{mod("EnemyModifier", List, ModRef{Mod: flag("CannotEvade", &CondTag{Var: "Taunted"})})},
	`if you've impaled an enemy recently, you and nearby allies have \+([0-9]+) to armour`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("Armour", Base, Num(c.n(1)))}, &CondTag{Var: "ImpaledRecently"})}
	}),
	`your hits permanently intimidate enemies that are on full life`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated", &CondTag{Var: "ChampionIntimidate"})})},
	`you and allies affected by your placed banners regenerate ([0-9.]+)% of life per second for each stage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "AffectedByPlacedBanner"}, &MultiplierTag{Var: "BannerValour"})})}
	}),
	`you and allies near your banner regenerate ([0-9.]+)% of life per second for each valour consumed for that banner`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "AffectedByPlacedBanner"}, &MultiplierTag{Var: "BannerValour"})})}
	}),
	// Chieftain
	`enemies near your totems take ([0-9]+)% increased physical and fire damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("PhysicalDamageTaken", Inc, Num(c.n(1)))}), mod("EnemyModifier", List, ModRef{Mod: mod("FireDamageTaken", Inc, Num(c.n(1)))})}
	}),
	`every ([0-9]+) seconds, gain ([0-9]+)% of physical damage as extra fire damage for ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsFire", Base, Num(c.n(2)), &CondTag{Var: "NgamahuFlamesAdvance"})}
	}),
	`([0-9]+)% more damage for each endurance charge lost recently, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", More, Num(c.n(1)), &MultiplierTag{Var: "EnduranceChargesLostRecently", Limit: opt(c.n(2)), LimitTotal: true})}
	}),
	`nearby enemy monsters have no fire resistance against damage over time while you are stationary`: modList{mod("EnemyModifier", List, ModRef{Mod: modf("FireResist", Override, Num(0), FlagDot, KeywordNone, &CondTag{IsActor: true, Actor: "player", Var: "Stationary"})})},
	`([0-9]+)% more damage if you've lost an endurance charge in the past 8 seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", More, Num(c.n(1)), &CondTag{Var: "LostEnduranceChargeInPast8Sec"})}
	}),
	`trigger level ([0-9]+) (.+) when you attack with a non-vaal slam or strike skill near an enemy`: modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	// Deadeye
	`projectiles pierce all nearby targets`: modList{flag("PierceAllTargets")},
	`gain \+([0-9]+) life when you hit a bleeding enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeOnHit", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"})}
	}),
	`([0-9]+)% increased blink arrow and mirror arrow cooldown recovery speed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CooldownRecovery", Inc, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Blink Arrow", "Mirror Arrow"}, IncludeTransfigured: true})}
	}),
	`critical strikes which inflict bleeding also inflict rupture`:         modList{flag("Condition:CanInflictRupture", &CondTag{Neg: true, Var: "NeverCrit"})},
	`gain [0-9]+ gale force when you use a skill`:                          modList{flag("Condition:CanGainGaleForce")},
	`if you've used a skill recently, you and nearby allies have tailwind`: modList{mod("ExtraAura", List, ModRef{Mod: flag("Condition:Tailwind")}, &CondTag{Var: "UsedSkillRecently"})},
	`you and nearby allies have tailwind`:                                  modList{mod("ExtraAura", List, ModRef{Mod: flag("Condition:Tailwind")})},
	`projectiles deal ([0-9]+)% more damage for each remaining chain`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagProjectile, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "ChainRemaining"})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits and ailments for each remaining chain`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &StatTag{StatKind: TagPerStat, Stat: "ChainRemaining"}, &SkillTypeTag{SkillType: SkillTypeProjectile})}
	}),
	`projectiles deal ([0-9]+)% increased damage for each remaining chain`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagProjectile, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "ChainRemaining"})}
	}),
	`far shot`: modList{flag("FarShot")},
	`projectiles gain damage as they travel farther, dealing up to ([0-9]+)% more damage with hits and ailments`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &DistanceRampTag{Ramp: Pairs{{35, 0}, {70, 1}}})}
	}),
	`([0-9]+)% increased mirage archer duration`:                 modFn(func(c caps) []*Mod { return []*Mod{mod("MirageArcherDuration", Inc, Num(c.n(1)))} }),
	`([\-+][0-9]+) to maximum number of summoned mirage archers`: modFn(func(c caps) []*Mod { return []*Mod{mod("MirageArcherMaxCount", Base, Num(c.n(1)))} }),
	`([\-+][0-9]+) to maximum number of sacred wisps`:            modFn(func(c caps) []*Mod { return []*Mod{mod("SacredWispsMaxCount", Base, Num(c.n(1)))} }),
	// Elementalist
	`gain ([0-9]+)% increased area of effect for [0-9]+ seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &CondTag{Var: "PendulumOfDestructionAreaOfEffect"})}
	}),
	`gain ([0-9]+)% increased elemental damage for [0-9]+ seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalDamage", Inc, Num(c.n(1)), &CondTag{Var: "PendulumOfDestructionElementalDamage"})}
	}),
	`for each element you've been hit by damage of recently, ([0-9]+)% increased damage of that element`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireDamage", Inc, Num(c.n(1)), &CondTag{Var: "HitByFireDamageRecently"}), mod("ColdDamage", Inc, Num(c.n(1)), &CondTag{Var: "HitByColdDamageRecently"}), mod("LightningDamage", Inc, Num(c.n(1)), &CondTag{Var: "HitByLightningDamageRecently"})}
	}),
	`for each element you've been hit by damage of recently, ([0-9]+)% reduced damage taken of that element`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireDamageTaken", Inc, Num(-c.n(1)), &CondTag{Var: "HitByFireDamageRecently"}), mod("ColdDamageTaken", Inc, Num(-c.n(1)), &CondTag{Var: "HitByColdDamageRecently"}), mod("LightningDamageTaken", Inc, Num(-c.n(1)), &CondTag{Var: "HitByLightningDamageRecently"})}
	}),
	`gain convergence when you hit a unique enemy, no more than once every [0-9]+ seconds`: modList{flag("Condition:CanGainConvergence")},
	`([0-9]+)% increased area of effect while you don't have convergence`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &CondTag{Neg: true, Var: "Convergence"})}
	}),
	`exposure you inflict applies an extra (-?[0-9]+)% to the affected resistance`:        modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraExposure", Base, Num(c.n(1)))} }),
	`cannot take reflected elemental damage`:                                              modList{mod("ElementalReflectedDamageTaken", More, Num(-100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`every [0-9]+ seconds:`:                                                               modList{},
	`gain chilling conflux for [0-9] seconds`:                                             modList{flag("PhysicalCanChill", &CondTag{Var: "ChillingConflux"}), flag("LightningCanChill", &CondTag{Var: "ChillingConflux"}), flag("FireCanChill", &CondTag{Var: "ChillingConflux"}), flag("ChaosCanChill", &CondTag{Var: "ChillingConflux"})},
	`gain shocking conflux for [0-9] seconds`:                                             modList{mod("EnemyShockChance", Base, Num(100), &CondTag{Var: "ShockingConflux"}), flag("PhysicalCanShock", &CondTag{Var: "ShockingConflux"}), flag("ColdCanShock", &CondTag{Var: "ShockingConflux"}), flag("FireCanShock", &CondTag{Var: "ShockingConflux"}), flag("ChaosCanShock", &CondTag{Var: "ShockingConflux"})},
	`gain igniting conflux for [0-9] seconds`:                                             modList{mod("EnemyIgniteChance", Base, Num(100), &CondTag{Var: "IgnitingConflux"}), flag("PhysicalCanIgnite", &CondTag{Var: "IgnitingConflux"}), flag("LightningCanIgnite", &CondTag{Var: "IgnitingConflux"}), flag("ColdCanIgnite", &CondTag{Var: "IgnitingConflux"}), flag("ChaosCanIgnite", &CondTag{Var: "IgnitingConflux"})},
	`([0-9]+)% chance to gain elemental conflux for [0-9] seconds when you kill an enemy`: modList{flag("PhysicalCanChill", &CondTag{Var: "ChillingConflux"}), flag("LightningCanChill", &CondTag{Var: "ChillingConflux"}), flag("FireCanChill", &CondTag{Var: "ChillingConflux"}), flag("ChaosCanChill", &CondTag{Var: "ChillingConflux"}), mod("EnemyShockChance", Base, Num(100), &CondTag{Var: "ShockingConflux"}), flag("PhysicalCanShock", &CondTag{Var: "ShockingConflux"}), flag("ColdCanShock", &CondTag{Var: "ShockingConflux"}), flag("FireCanShock", &CondTag{Var: "ShockingConflux"}), flag("ChaosCanShock", &CondTag{Var: "ShockingConflux"}), mod("EnemyIgniteChance", Base, Num(100), &CondTag{Var: "IgnitingConflux"}), flag("PhysicalCanIgnite", &CondTag{Var: "IgnitingConflux"}), flag("LightningCanIgnite", &CondTag{Var: "IgnitingConflux"}), flag("ColdCanIgnite", &CondTag{Var: "IgnitingConflux"}), flag("ChaosCanIgnite", &CondTag{Var: "IgnitingConflux"})},
	`you have elemental conflux if the stars are aligned`:                                 modList{flag("PhysicalCanChill", &CondTag{Var: "StarsAreAligned"}), flag("LightningCanChill", &CondTag{Var: "StarsAreAligned"}), flag("FireCanChill", &CondTag{Var: "StarsAreAligned"}), flag("ChaosCanChill", &CondTag{Var: "StarsAreAligned"}), mod("EnemyShockChance", Base, Num(100), &CondTag{Var: "StarsAreAligned"}), flag("PhysicalCanShock", &CondTag{Var: "StarsAreAligned"}), flag("ColdCanShock", &CondTag{Var: "StarsAreAligned"}), flag("FireCanShock", &CondTag{Var: "StarsAreAligned"}), flag("ChaosCanShock", &CondTag{Var: "StarsAreAligned"}), mod("EnemyIgniteChance", Base, Num(100), &CondTag{Var: "StarsAreAligned"}), flag("PhysicalCanIgnite", &CondTag{Var: "StarsAreAligned"}), flag("LightningCanIgnite", &CondTag{Var: "StarsAreAligned"}), flag("ColdCanIgnite", &CondTag{Var: "StarsAreAligned"}), flag("ChaosCanIgnite", &CondTag{Var: "StarsAreAligned"})},
	`gain chilling, shocking and igniting conflux for [0-9] seconds`:                      modList{},
	`you have igniting, chilling and shocking conflux while affected by glorious madness`: modList{flag("PhysicalCanChill", &CondTag{Var: "AffectedByGloriousMadness"}), flag("LightningCanChill", &CondTag{Var: "AffectedByGloriousMadness"}), flag("FireCanChill", &CondTag{Var: "AffectedByGloriousMadness"}), flag("ChaosCanChill", &CondTag{Var: "AffectedByGloriousMadness"}), mod("EnemyIgniteChance", Base, Num(100), &CondTag{Var: "AffectedByGloriousMadness"}), flag("PhysicalCanIgnite", &CondTag{Var: "AffectedByGloriousMadness"}), flag("LightningCanIgnite", &CondTag{Var: "AffectedByGloriousMadness"}), flag("ColdCanIgnite", &CondTag{Var: "AffectedByGloriousMadness"}), flag("ChaosCanIgnite", &CondTag{Var: "AffectedByGloriousMadness"}), mod("EnemyShockChance", Base, Num(100), &CondTag{Var: "AffectedByGloriousMadness"}), flag("PhysicalCanShock", &CondTag{Var: "AffectedByGloriousMadness"}), flag("ColdCanShock", &CondTag{Var: "AffectedByGloriousMadness"}), flag("FireCanShock", &CondTag{Var: "AffectedByGloriousMadness"}), flag("ChaosCanShock", &CondTag{Var: "AffectedByGloriousMadness"})},
	`all damage from critical strikes can apply lightning ailments during effect`:         modList{flag("PhysicalCanShock", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ColdCanShock", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("FireCanShock", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ChaosCanShock", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("PhysicalCanSap", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ColdCanSap", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("FireCanSap", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ChaosCanSap", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"})},
	`all damage from critical strikes can apply cold ailments during effect`:              modList{flag("PhysicalCanChill", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("LightningCanChill", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("FireCanChill", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ChaosCanChill", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("PhysicalCanFreeze", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("LightningCanFreeze", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("FireCanFreeze", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ChaosCanFreeze", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("PhysicalCanBrittle", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("LightningCanBrittle", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("FireCanBrittle", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"}), flag("ChaosCanBrittle", &CondTag{Var: "UsingFlask"}, &CondTag{Neg: true, Var: "NeverCrit"})},
	`immun[ei]t?y? to elemental ailments while affected by glorious madness`:              modList{flag("ElementalAilmentImmune", &CondTag{Var: "AffectedByGloriousMadness"})},
	`immun[ei]t?y? to elemental ailments while focus?sed`:                                 modList{flag("ElementalAilmentImmune", &CondTag{Var: "Focused"})},
	`summoned golems are immune to elemental damage`:                                      modList{mod("MinionModifier", List, ModRef{Mod: flag("Elemancer")}, &SkillTypeTag{SkillType: SkillTypeGolem}), mod("MinionModifier", List, ModRef{Mod: mod("ElementalDamageTaken", More, Num(-100))}, &SkillTypeTag{SkillType: SkillTypeGolem})},
	`([0-9]+)% increased golem damage per summoned golem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)))}, &SkillTypeTag{SkillType: SkillTypeGolem}, &StatTag{StatKind: TagPerStat, Stat: "ActiveGolemLimit"})}
	}),
	`shocks from your hits always increase damage taken by at least ([0-9]+)%`: modFn(func(c caps) []*Mod { return []*Mod{mod("ShockMinimum", Base, Num(c.n(1)))} }),
	`chills from your hits always reduce action speed by at least ([0-9]+)%`:   modFn(func(c caps) []*Mod { return []*Mod{mod("ChillBase", Base, Num(c.n(1)))} }),
	`([0-9]+)% more damage with ignites you inflict with hits for which the highest damage type is fire`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordIgnite, &CondTag{Var: "FireIsHighestDamageType"}), flag("ChecksHighestDamage")}
	}),
	`([0-9]+)% more effect of cold ailments you inflict with hits for which the highest damage type is cold`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyChillEffect", More, Num(c.n(1)), &CondTag{Var: "ColdIsHighestDamageType"}), mod("EnemyBrittleEffect", More, Num(c.n(1)), &CondTag{Var: "ColdIsHighestDamageType"}), flag("ChecksHighestDamage")}
	}),
	`([0-9]+)% more effect of lightning ailments you inflict with hits if the highest damage type is lightning`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyShockEffect", More, Num(c.n(1)), &CondTag{Var: "LightningIsHighestDamageType"}), mod("EnemySapEffect", More, Num(c.n(1)), &CondTag{Var: "LightningIsHighestDamageType"}), flag("ChecksHighestDamage")}
	}),
	`your chills can reduce action speed by up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod { return []*Mod{mod("ChillMax", Override, Num(c.n(1)))} }),
	`your hits always ignite`:         modList{mod("EnemyIgniteChance", Base, Num(100))},
	`hits always ignite`:              modList{mod("EnemyIgniteChance", Base, Num(100))},
	`your hits always shock`:          modList{mod("EnemyShockChance", Base, Num(100))},
	`hits always shock`:               modList{mod("EnemyShockChance", Base, Num(100))},
	`always freeze, shock and ignite`: modList{mod("EnemyFreezeChance", Base, Num(100)), mod("EnemyShockChance", Base, Num(100)), mod("EnemyIgniteChance", Base, Num(100))},
	`all damage with hits can ignite`: modList{flag("PhysicalCanIgnite"), flag("ColdCanIgnite"), flag("LightningCanIgnite"), flag("ChaosCanIgnite")},
	`all damage can ignite`:           modList{flag("PhysicalCanIgnite"), flag("ColdCanIgnite"), flag("LightningCanIgnite"), flag("ChaosCanIgnite")},
	`all damage with hits can chill`:  modList{flag("PhysicalCanChill"), flag("FireCanChill"), flag("LightningCanChill"), flag("ChaosCanChill")},
	`all damage with hits can shock`:  modList{flag("PhysicalCanShock"), flag("FireCanShock"), flag("ColdCanShock"), flag("ChaosCanShock")},
	`all damage can shock`:            modList{flag("PhysicalCanShock"), flag("FireCanShock"), flag("ColdCanShock"), flag("ChaosCanShock")},
	`other aegis skills are disabled`: modList{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeAegis}), flag("EnableSkill", &SkillNameTag{SkillID: "Primal Aegis"})},
	`primal aegis can take ([0-9]+) elemental damage per allocated notable passive skill`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ElementalAegisValue", Max, Num(c.n(1)), FlagNone, KeywordNone, &MultiplierTag{Var: "AllocatedNotable"}, &GlobalEffectTag{EffectType: "Buff", Unscalable: true})}
	}),
	`enemies chilled by your hits lessen their damage dealt by half of chill effect`: modList{flag("ChillEffectLessDamageDealt")},
	// Gladiator
	`chance to block spell damage is equal to chance to block attack damage`:                                    modList{flag("SpellBlockChanceIsBlockChance")},
	`maximum chance to block spell damage is equal to maximum chance to block attack damage`:                    modList{flag("SpellBlockChanceMaxIsBlockChanceMax")},
	`attack damage is lucky if you[' ]h?a?ve blocked in the past ([0-9]+) seconds`:                              modList{flagf("LuckyHits", FlagAttack, KeywordNone, &CondTag{Var: "BlockedRecently"})},
	`attack damage while dual wielding is lucky if you[' ]h?a?ve blocked in the past ([0-9]+) seconds`:          modList{flagf("LuckyHits", FlagAttack, KeywordNone, &CondTag{Var: "BlockedRecently"}, &CondTag{Var: "DualWielding"})},
	`hits ignore enemy monster physical damage reduction if you[' ]h?a?ve blocked in the past ([0-9]+) seconds`: modList{flag("IgnoreEnemyPhysicalDamageReduction", &CondTag{Var: "BlockedRecently"})},
	`([0-9]+)% more attack and movement speed per challenger charge`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Speed", More, Num(c.n(1)), FlagAttack, KeywordNone, &MultiplierTag{Var: "ChallengerCharge"}), mod("MovementSpeed", More, Num(c.n(1)), &MultiplierTag{Var: "ChallengerCharge"})}
	}),
	`gain ([0-9]+)% chance to block from equipped shield instead of the shield's value`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReplaceShieldBlock", Override, Num(c.n(1)), &CondTag{Var: "UsingShield"})}
	}),
	`deal ([0-9]+)% more damage with hits and ailments to rare and unique enemies for each second they've ever been in your presence, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &MultiplierTag{Var: "EnemyPresenceSeconds", Actor: "enemy", Limit: opt(c.n(2))}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})}
	}),
	`deal ([0-9]+)% more damage with hits and ailments to rare and unique enemies for every ([0-9]+) seconds they've ever been in your presence, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &MultiplierTag{Var: "EnemyPresenceSeconds", Actor: "enemy", Limit: opt(c.n(3)), Div: opt(c.n(2))}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})}
	}),
	`([0-9]+)% more damage with hits and ailments against enemies that are on low life while you are wielding an axe`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "LowLife"}, &CondTag{Var: "UsingAxe"})}
	}),
	`retaliation skills have ([0-9]+)% increased speed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Speed", Inc, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeRetaliation}), modf("WarcrySpeed", Inc, Num(c.n(1)), FlagNone, KeywordWarcry, &SkillTypeTag{SkillType: SkillTypeRetaliation})}
	}),
	// Guardian
	`grants armour equal to ([0-9]+)% of your reserved life to you and nearby allies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GrantReservedLifeAsAura", List, ModRef{Mod: mod("Armour", Base, Num(c.n(1)/100))})}
	}),
	`grants armour equal to ([0-9]+)% of your reserved mana to you and nearby allies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GrantReservedManaAsAura", List, ModRef{Mod: mod("Armour", Base, Num(c.n(1)/100))})}
	}),
	`grants maximum energy shield equal to ([0-9]+)% of your reserved mana to you and nearby allies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GrantReservedManaAsAura", List, ModRef{Mod: mod("EnergyShield", Base, Num(c.n(1)/100))})}
	}),
	`warcries cost no mana`: modList{modf("ManaCost", More, Num(-100), FlagNone, KeywordWarcry)},
	`\+([0-9]+)% chance to block attack damage for [0-9] seconds? every [0-9] seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BlockChance", Base, Num(c.n(1)), &CondTag{Var: "BastionOfHopeActive"})}
	}),
	`if you've blocked in the past [0-9]+ seconds, you and nearby allies cannot be stunned`: modList{mod("ExtraAura", List, ModRef{Mod: flag("StunImmune")}, &CondTag{Var: "BlockedRecently"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`if you've attacked recently, you and nearby allies have \+([0-9]+)% chance to block attack damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("BlockChance", Base, Num(c.n(1)))}, &CondTag{Var: "AttackedRecently"})}
	}),
	`if you've cast a spell recently, you and nearby allies have \+([0-9]+)% chance to block spell damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("SpellBlockChance", Base, Num(c.n(1)))}, &CondTag{Var: "CastSpellRecently"})}
	}),
	`while there is at least one nearby ally, you and nearby allies deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("Damage", More, Num(c.n(1)))}, &MultiplierTag{IsThreshold: true, Var: "NearbyAlly", Threshold: opt(1)})}
	}),
	`while there are at least five nearby allies, you and nearby allies have onslaught`: modList{mod("ExtraAura", List, ModRef{Mod: flag("Onslaught")}, &MultiplierTag{IsThreshold: true, Var: "NearbyAlly", Threshold: opt(5)})},
	`linked targets and allies in your link beams have \+([0-9]+)% to all maximum elemental resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("ElementalResistMax", Base, Num(c.n(1)), &CondTag{Var: "AffectedByLink", Neg: true}), OnlyAllies: true}, &MultiplierTag{IsThreshold: true, Var: "LinkedTargets", Threshold: opt(1)}), mod("ExtraLinkEffect", List, ModRef{Mod: mod("ElementalResistMax", Base, Num(c.n(1)), &GlobalEffectTag{EffectType: "Global", Unscalable: true})})}
	}),
	`enemies in your link beams cannot apply elemental ailments`:                              modList{flag("ElementalAilmentImmune", &CondTag{IsActor: true, Actor: "enemy", Var: "BetweenYouAndLinkedTarget"})},
	`([0-9]+)% of damage from hits is taken from your sentinel of radiance's life before you`: modFn(func(c caps) []*Mod { return []*Mod{mod("takenFromRadianceSentinelBeforeYou", Base, Num(c.n(1)))} }),
	`you can inflict \+([0-9]+) hallowing flame on enemies`:                                   modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:HallowingFlameMax", Base, Num(c.n(1)))} }),
	`gain ([0-9]+)% of ([a-zA-Z]+) damage as extra ([a-zA-Z]+) damage for each of your hallowing flames that have been removed by an allied hit recently, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(2))+"DamageGainAs"+firstToUpper(c.s(3)), Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "HallowingFlame"}, &MultiplierTag{Var: "HallowingFlameStacksRemovedByAlly", Limit: opt(c.n(4) / c.n(1))})}
	}),
	`([0-9]+)% increased magnitude of hallowing flame you inflict`: modFn(func(c caps) []*Mod { return []*Mod{mod("HallowingFlameMagnitude", Inc, Num(c.n(1)))} }),
	// Hierophant
	`you and your totems regenerate ([0-9.]+)% of life per second for each summoned totem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "TotemsSummoned"}), modf("LifeRegenPercent", Base, Num(c.n(1)), FlagNone, KeywordTotem)}
	}),
	`enemies take ([0-9]+)% increased damage for each of your brands attached to them`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &MultiplierTag{Var: "BrandsAttached"})})}
	}),
	`immun[ei]t?y? to elemental ailments while you have arcane surge`: modList{flag("ElementalAilmentImmune", &CondTag{Var: "AffectedByArcaneSurge"})},
	`brands have ([0-9]+)% more activation frequency if ([0-9]+)% of attached duration expired`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BrandActivationFrequency", More, Num(c.n(1)), &CondTag{Var: "BrandLastQuarter"})}
	}),
	`arcane surge a?l?s?o? ?grants ([0-9]+)% more spell damage to you`: modFn(func(c caps) []*Mod { return []*Mod{mod("ArcaneSurgeDamage", Max, Num(c.n(1)))} }),
	// Inquisitor
	`critical strikes ignore enemy monster elemental resistances`: modList{flag("IgnoreElementalResistances", &CondTag{Var: "CriticalStrike"})},
	`non-critical strikes penetrate ([0-9]+)% of enemy elemental resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalPenetration", Base, Num(c.n(1)), &CondTag{Var: "CriticalStrike", Neg: true})}
	}),
	`consecrated ground you create applies ([0-9]+)% increased damage taken to enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTakenConsecratedGround", Inc, Num(c.n(1)), &CondTag{Var: "OnConsecratedGround"})})}
	}),
	`you have consecrated ground around you while stationary`:                                    modList{flag("Condition:OnConsecratedGround", &CondTag{Var: "Stationary"})},
	`consecrated ground you create grants immun[ei]t?y? to elemental ailments to you and allies`: modList{mod("ExtraAura", List, ModRef{Mod: flag("ElementalAilmentImmune", &CondTag{Var: "OnConsecratedGround"})})},
	`gain fanaticism for ([0-9]+) seconds on reaching maximum fanatic charges`:                   modList{flag("Condition:CanGainFanaticism")},
	`([0-9]+)% increased critical strike chance per point of strength or intelligence, whichever is lower`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritChance", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "Str"}, &CondTag{Var: "IntHigherThanStr"}), mod("CritChance", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "Int"}, &CondTag{Neg: true, Var: "IntHigherThanStr"})}
	}),
	`consecrated ground you create causes life regeneration to also recover energy shield for you and allies`: modList{mod("ExtraAura", List, ModRef{Mod: flag("LifeRegenerationRecoversEnergyShield", &CondTag{Var: "OnConsecratedGround"})})},
	`([0-9]+)% more attack damage for each non-instant spell you've cast in the past 8 seconds, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagAttack, KeywordNone, &MultiplierTag{Var: "CastLast8Seconds", Limit: opt(c.n(2)), LimitTotal: true})}
	}),
	// Juggernaut
	`action speed cannot be modified to below base value`:   modList{mod("MinimumActionSpeed", Max, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`movement speed cannot be modified to below base value`: modList{flag("MovementSpeedCannotBeBelowBase")},
	`you cannot be slowed to below base speed`:              modList{mod("MinimumActionSpeed", Max, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`cannot be slowed to below base speed`:                  modList{mod("MinimumActionSpeed", Max, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`gain accuracy rating equal to your strength`:           modList{mod("Accuracy", Base, Num(1), &StatTag{StatKind: TagPerStat, Stat: "Str"})},
	`gain accuracy rating equal to twice your strength`:     modList{mod("Accuracy", Base, Num(2), &StatTag{StatKind: TagPerStat, Stat: "Str"})},
	// Necromancer
	`your offering skills also affect you`: modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("SkillData", List, DataRef{Key: "buffNotPlayer", Value: Bool(false)})}, &SkillNameTag{SkillNameList: []string{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})},
	`your offerings have ([0-9]+)% reduced effect on you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("BuffEffectOnPlayer", Inc, Num(-c.n(1)))}, &SkillNameTag{SkillNameList: []string{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`your offerings have ([0-9]+)% increased effect on you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("BuffEffectOnPlayer", Inc, Num(c.n(1)))}, &SkillNameTag{SkillNameList: []string{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`if you've consumed a corpse recently, you and your minions have ([0-9]+)% increased area of effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &CondTag{Var: "ConsumedCorpseRecently"}), mod("MinionModifier", List, ModRef{Mod: mod("AreaOfEffect", Inc, Num(c.n(1)))}, &CondTag{Var: "ConsumedCorpseRecently"})}
	}),
	`with at least one nearby corpse, you and nearby allies deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("Damage", More, Num(c.n(1)))}, &MultiplierTag{IsThreshold: true, Var: "NearbyCorpse", Threshold: opt(1)})}
	}),
	`with at least one nearby corpse, nearby enemies deal ([0-9]+)% reduced damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("Damage", Inc, Num(-c.n(1)))}, &MultiplierTag{IsThreshold: true, Var: "NearbyCorpse", Threshold: opt(1)})}
	}),
	`for each nearby corpse, you and nearby allies regenerate ([0-9.]+)% of energy shield per second, up to ([0-9.]+)% per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("EnergyShieldRegenPercent", Base, Num(c.n(1)))}, &MultiplierTag{Var: "NearbyCorpse", Limit: opt(c.n(2)), LimitTotal: true})}
	}),
	`for each nearby corpse, you and nearby allies regenerate ([0-9]+) mana per second, up to ([0-9]+) per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("ManaRegen", Base, Num(c.n(1)))}, &MultiplierTag{Var: "NearbyCorpse", Limit: opt(c.n(2)), LimitTotal: true})}
	}),
	`enemies near corpses you spawned recently are chilled and shocked`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Chilled")}, &CondTag{Var: "SpawnedCorpseRecently"}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Shocked")}, &CondTag{Var: "SpawnedCorpseRecently"}), mod("ChillBase", Base, Num(nonDamagingAilmentDefault["Chill"]), &CondTag{Var: "SpawnedCorpseRecently"}), mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{Var: "SpawnedCorpseRecently"})},
	`regenerate ([0-9]+)% of energy shield over 2 seconds when you consume a corpse`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldRegenPercent", Base, Num(c.n(1)/2), &CondTag{Var: "ConsumedCorpseInPast2Sec"})}
	}),
	`regenerate ([0-9]+)% of mana over 2 seconds when you consume a corpse`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaRegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1) / 2)}, &CondTag{Var: "ConsumedCorpseInPast2Sec"})}
	}),
	`corpses you spawn have ([0-9]+)% increased maximum life`: modFn(func(c caps) []*Mod { return []*Mod{mod("CorpseLife", Inc, Num(c.n(1)))} }),
	`corpses you spawn have ([0-9]+)% reduced maximum life`:   modFn(func(c caps) []*Mod { return []*Mod{mod("CorpseLife", Inc, Num(-c.n(1)))} }),
	`minions gain added physical damage equal to ([0-9]+)% of maximum energy shield on your equipped helmet`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("PhysicalMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShieldOnHelmet", Actor: "parent", Percent: opt(c.n(1))})}), mod("MinionModifier", List, ModRef{Mod: mod("PhysicalMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShieldOnHelmet", Actor: "parent", Percent: opt(c.n(1))})})}
	}),
	// Occultist
	`when you kill an enemy, for each curse on that enemy, gain ([0-9]+)% of non-chaos damage as extra chaos damage for 4 seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("NonChaosDamageGainAsChaos", Base, Num(c.n(1)), &CondTag{Var: "KilledRecently"}, &MultiplierTag{Var: "CurseOnEnemy"})}
	}),
	`cannot be stunned while you have energy shield`:                        modList{flag("StunImmune", &CondTag{Var: "HaveEnergyShield"})},
	`every second, inflict withered on nearby enemies for ([0-9]+) seconds`: modList{flag("Condition:CanWither")},
	`nearby hindered enemies deal ([0-9]+)% reduced damage over time`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageOverTime", Inc, Num(-c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Hindered"})}
	}),
	`nearby chilled enemies deal ([0-9]+)% reduced damage with hits`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("Damage", Inc, Num(-c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"})}
	}),
	`gain spirit infusion every ?([0-9]?)\.?([0-9]?) seconds? while channelling a spell`: modFn(func(c caps) []*Mod { return []*Mod{flag("Condition:CanGainSpiritInfusion")} }),
	// Pathfinder
	`always poison on hit while using a flask`: modList{mod("PoisonChance", Base, Num(100), &CondTag{Var: "UsingFlask"})},
	`poisons you inflict during any flask effect have ([0-9]+)% chance to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordPoison, &CondTag{Var: "UsingFlask"})}
	}),
	`immun[ei]t?y? to elemental ailments during any flask effect`:                                                                             modList{flag("ElementalAilmentImmune", &CondTag{Var: "UsingFlask"})},
	`grant bonuses to non-channelling skills you use by consuming ([0-9]+) charges from a flask of each of the following types, if possible:`: modList{},
	`if diamond flask charges are consumed, ([0-9]+)% increased critical strike chance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritChance", Inc, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeTriggered, Neg: true}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &CondTag{Var: "HaveDiamondFlask"})}
	}),
	`if bismuth flask charges are consumed, penetrate ([0-9]+)% elemental resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalPenetration", Base, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeTriggered, Neg: true}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &CondTag{Var: "HaveBismuthFlask"})}
	}),
	`if amethyst flask charges are consumed, ([0-9]+)% of physical damage as extra chaos damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsChaos", Base, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeTriggered, Neg: true}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &CondTag{Var: "HaveAmethystFlask"})}
	}),
	// Raider
	`nearby enemies have ([0-9]+)% less accuracy rating while you have phasing`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("Accuracy", More, Num(-c.n(1)))}, &CondTag{Var: "Phasing"})}
	}),
	`immun[ei]t?y? to elemental ailments while phasing`: modList{flag("ElementalAilmentImmune", &CondTag{Var: "Phasing"})},
	`nearby enemies have fire, cold and lightning exposure while you have phasing, applying -([0-9]+)% to those resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-c.n(1)))}, &CondTag{Var: "Phasing"}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-c.n(1)))}, &CondTag{Var: "Phasing"}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-c.n(1)))}, &CondTag{Var: "Phasing"})}
	}),
	`nearby enemies have fire, cold and lightning exposure while you have phasing`: modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10))}, &CondTag{Var: "Phasing"}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-10))}, &CondTag{Var: "Phasing"}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-10))}, &CondTag{Var: "Phasing"})},
	// Saboteur
	`hits have ([0-9]+)% chance to deal ([0-9]+)% more area damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num((c.n(1) * c.n(2) / 100)), FlagArea|FlagHit, KeywordNone)}
	}),
	`immun[ei]t?y? to ignite and shock`: modList{flag("IgniteImmune"), flag("ShockImmune")},
	`you gain ([0-9]+)% increased damage for each trap`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ActiveTrapLimit"})}
	}),
	`you gain ([0-9]+)% increased area of effect for each mine`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ActiveMineLimit"})}
	}),
	`triggers level ([0-9]+) summon triggerbots when allocated`: modList{flag("HaveTriggerBots")},
	// Slayer
	`deal up to ([0-9]+)% more melee damage to enemies, based on proximity`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagAttack|FlagMelee, KeywordNone, &MeleeProximityTag{Ramp: []float64{1, 0}})}
	}),
	`cannot be stunned while leeching`:                                               modList{flag("StunImmune", &CondTag{Var: "Leeching"})},
	`you are immune to bleeding while leeching`:                                      modList{flag("BleedImmune", &CondTag{Var: "Leeching"})},
	`life leech effects are not removed at full life`:                                modList{flag("CanLeechLifeOnFullLife")},
	`life leech effects are not removed when unreserved life is filled`:              modList{flag("CanLeechLifeOnFullLife")},
	`energy shield leech effects from attacks are not removed at full energy shield`: modList{flag("CanLeechEnergyShieldOnFullEnergyShield")},
	`cannot take reflected physical damage`:                                          modList{mod("PhysicalReflectedDamageTaken", More, Num(-100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`gain ([0-9]+)% increased movement speed for 20 seconds when you kill an enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MovementSpeed", Inc, Num(c.n(1)), &CondTag{Var: "KilledRecently"})}
	}),
	`gain ([0-9]+)% increased attack speed for 20 seconds when you kill a rare or unique enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Speed", Inc, Num(c.n(1)), FlagAttack, KeywordNone, &CondTag{Var: "KilledUniqueEnemy"})}
	}),
	`kill enemies that have ([0-9]+)% or lower life when hit by your skills`: modFn(func(c caps) []*Mod { return []*Mod{mod("CullPercent", Max, Num(c.n(1)))} }),
	`you are unaffected by bleeding while leeching`:                          modList{mod("SelfBleedEffect", More, Num(-100), &CondTag{Var: "Leeching"})},
	// Trickster
	`([0-9]+)% chance to gain ([0-9]+)% of non-chaos damage with hits as extra chaos damage`: modFn(func(c caps) []*Mod { return []*Mod{mod("NonChaosDamageGainAsChaos", Base, Num(c.n(1)/100*c.n(2)))} }),
	`movement skills cost no mana`: modList{modf("ManaCost", More, Num(-100), FlagNone, KeywordMovement)},
	`cannot be stunned while you have ghost shrouds`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("StunImmune", &MultiplierTag{IsThreshold: true, Var: "GhostShroud", Threshold: opt(1)})}
	}),
	`your action speed is at least ([0-9]+)% of base value`: modFn(func(c caps) []*Mod { return []*Mod{mod("MinimumActionSpeed", Max, Num(c.n(1)))} }),
	`nearby enemy monsters' action speed is at most ([0-9]+)% of base value`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("MaximumActionSpeedReduction", Max, Num(100-c.n(1)))})}
	}),
	`prevent \+([0-9]+)% of suppressed spell damage while on full energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionEffect", Base, Num(c.n(1)), &CondTag{Var: "FullEnergyShield"})}
	}),
	`energy shield leech effects are not removed when energy shield is filled`: modList{flag("CanLeechEnergyShieldOnFullEnergyShield")},
	`take ([0-9]+)% less damage from hits for ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DamageTakenWhenHit", More, Num(-c.n(1)), &CondTag{Var: "HeartstopperHIT"}), mod("DamageTakenWhenHit", More, Num(-c.n(1)*c.n(2)/10), &CondTag{Var: "HeartstopperAVERAGE"})}
	}),
	`take ([0-9]+)% less damage over time for ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DamageTakenOverTime", More, Num(-c.n(1)), &CondTag{Var: "HeartstopperDOT"}), mod("DamageTakenOverTime", More, Num(-c.n(1)*c.n(2)/10), &CondTag{Var: "HeartstopperAVERAGE"})}
	}),
	// Warden
	`prevent \+([0-9]+)% of suppressed spell damage per bark below maximum`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionEffect", Base, Num(c.n(1)), &MultiplierTag{Var: "MissingBarkskinStacks"})}
	}),
	`hits that would ignite instead scorch`:                       modList{flag("IgniteCanScorch"), flag("CannotIgnite")},
	`you can inflict an additional scorch on each enemy`:          modList{flag("ScorchCanStack"), mod("ScorchStacksMax", Base, Num(1))},
	`maximum effect of shock is ([0-9]+)% increased damage taken`: modFn(func(c caps) []*Mod { return []*Mod{mod("ShockMax", Override, Num(c.n(1)))} }),
	`you can apply up to ([0-9]+) shocks to each enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ShockCanStack"), mod("ShockStacksMax", Override, Num(c.n(1)))}
	}),
	`hits that fail to freeze due to insufficient freeze duration inflict hoarfrost`: modList{flag("HitsCanInflictHoarfrost")},
	`your hits always inflict freeze, shock and ignite while unbound`:                modList{mod("EnemyFreezeChance", Base, Num(100), &CondTag{Var: "Unbound"}), mod("EnemyShockChance", Base, Num(100), &CondTag{Var: "Unbound"}), mod("EnemyIgniteChance", Base, Num(100), &CondTag{Var: "Unbound"})},
	`([0-9]+)% more elemental damage while unbound`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalDamage", More, Num(c.n(1)), &CondTag{Var: "Unbound"})}
	}),
	// Warden (Affliction)
	`defences from equipped body armour are doubled if it has no socketed gems`: modList{mod("Defences", More, Num(100), &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInBody Armour", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingBody Armour"}, &SlotTag{SlotKind: TagSlotName, SlotName: "Body Armour"}, &MultiplierTag{Var: "OathOfTheMajiDoubled", GlobalLimit: opt(100), GlobalLimitKey: "OathOfTheMajiLimit"}), mod("Multiplier:OathOfTheMajiDoubled", Override, Num(1), &SlotTag{SlotKind: TagSlotName, SlotName: "Body Armour"})},
	`([+\-][0-9]+)% to all elemental resistances if you have an equipped helmet with no socketed gems`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalResist", Base, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInHelmet", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingHelmet"})}
	}),
	`([0-9]+)% increased maximum life if you have equipped gloves with no socketed gems`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Life", Inc, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInGloves", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingGloves"})}
	}),
	`([0-9]+)% increased movement speed if you have equipped boots with no socketed gems`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MovementSpeed", Inc, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInBoots", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingBoots"})}
	}),
	// Warlock
	`spells you cast yourself gain added physical damage equal to ([0-9]+)% of life cost, if life cost is not higher than the maximum you could spend`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "LifeCost", Percent: opt(c.n(1))}, &StatTag{StatKind: TagStatThreshold, Stat: "LifeUnreserved", ThresholdStat: "LifeCost", ThresholdPercent: opt(c.n(1))}), mod("PhysicalMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "LifeCost", Percent: opt(c.n(1))}, &StatTag{StatKind: TagStatThreshold, Stat: "LifeUnreserved", ThresholdStat: "LifeCost", ThresholdPercent: opt(c.n(1))})}
	}),
	`gain maximum life instead of maximum energy shield from equipped armour items`: modList{flag("ConvertArmourESToLife")},
	// Ritualist Bloodline
	`unaffected by bleeding`:      modList{mod("SelfBleedEffect", More, Num(-100))},
	`unaffected by poison`:        modList{mod("SelfPoisonEffect", More, Num(-100))},
	`can't use amulets`:           modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Amulet"})},
	`can't use belts`:             modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Belt"})},
	`\+1 ring slot`:               modList{flag("AdditionalRingSlot")},
	`utility flasks are disabled`: modList{flag("UtilityFlasksDoNotApplyToPlayer")},
	// Aul Bloodline
	`action speed cannot be modified to below base value if you have equipped boots with no socketed gems`:    modList{mod("MinimumActionSpeed", Max, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInBoots", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingBoots"})},
	`cannot be stunned if you have an equipped helmet with no socketed gems`:                                  modList{flag("StunImmune", &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInHelmet", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingHelmet"})},
	`elemental ailments cannot be inflicted on you if you have an equipped body armour with no socketed gems`: modList{flag("ElementalAilmentImmune", &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInBody Armour", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingBody Armour"})},
	`take no extra damage from critical strikes if you have equipped gloves with no socketed gems`:            modList{mod("ReduceCritExtraDamage", Base, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &MultiplierTag{IsThreshold: true, Var: "SocketedGemsInGloves", Threshold: opt(0), Upper: true}, &CondTag{Var: "UsingGloves"})},
	// Delirious Bloodline
	`while affected by glorious madness, inflict mania on nearby enemies every second`: modList{flag("Condition:CanInflictMania", &CondTag{Var: "AffectedByGloriousMadness"}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:AfflictedByMania")}, &CondTag{Var: "AffectedByGloriousMadness"})},
	// Lycia Bloodline
	`herald skills have ([0-9]+)% more buff effect for every ([0-9]+)% of maximum mana they reserve`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BuffEffect", More, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ManaReservedPercent", Div: opt(c.n(2))}, &SkillTypeTag{SkillType: SkillTypeHerald})}
	}),
	`herald skills and minions from herald skills deal ([0-9]+)% more damage for every ([0-9]+)% of maximum life those skills reserve`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", More, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "LifeReservedPercent", Div: opt(c.n(2)), Actor: "parent"})}, &SkillTypeTag{SkillType: SkillTypeHerald}), mod("Damage", More, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "LifeReservedPercent", Div: opt(c.n(2))}, &SkillTypeTag{SkillType: SkillTypeHerald})}
	}),
	// Oshabi Bloodline
	`unsealed spells gain ([0-9]+)% more damage each time their effects reoccur`: modFn(func(c caps) []*Mod { return []*Mod{mod("MaxSealDamage", More, Num(c.n(1)))} }),
	`skills gain added chaos damage equal to ([0-9]+)% of life cost, if life cost is not higher than the maximum you could spend`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "LifeCost", Percent: opt(c.n(1))}, &StatTag{StatKind: TagStatThreshold, Stat: "LifeUnreserved", ThresholdStat: "LifeCost", ThresholdPercent: opt(c.n(1))}), mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "LifeCost", Percent: opt(c.n(1))}, &StatTag{StatKind: TagStatThreshold, Stat: "LifeUnreserved", ThresholdStat: "LifeCost", ThresholdPercent: opt(c.n(1))})}
	}),
	`lose all rage on reaching maximum rage and gain wild savagery for 1 second per 10 rage lost this way`: modList{flag("WildSavagery")},
	// Velka Bloodline
	`inflict barnacles on nearby enemies every second`:  modList{flag("CanInflictBarnacles")},
	`drop brine ground while moving, lasting 4 seconds`: modList{flag("CanCreateBrineGround")},
	// Item local modifiers
	`has no sockets`: modList{flag("NoSockets")},
	`gems socketed always have the quality bonus from socket colour`: modList{flag("SocketAlwaysMatches")},
	`cannot have non-abyssal sockets`:                                modList{flag("NoSockets")},
	`socketed [a-zA-Z]+ abyssal jewels will be consumed`:             modList{},
	`one modifier from consumed jewels will be retained`:             modList{},
	`reflects your o[tp][hp][eo][rs]i?t?e? ring`:                     modList{},
	`cannot gain intangibility`:                                      modList{},
	`has ([0-9]+) sockets?`:                                          modFn(func(c caps) []*Mod { return []*Mod{mod("SocketCount", Base, Num(c.n(1)))} }),
	`has ([0-9]+) abyssal sockets?`:                                  modFn(func(c caps) []*Mod { return []*Mod{mod("AbyssalSocketCount", Base, Num(c.n(1)))} }),
	`no physical damage`:                                             modList{mod("WeaponData", List, DataRef{Key: "PhysicalMin"}), mod("WeaponData", List, DataRef{Key: "PhysicalMax"}), mod("WeaponData", List, DataRef{Key: "PhysicalDPS"})},
	`has ([0-9]+)% increased elemental damage`:                       modFn(func(c caps) []*Mod { return []*Mod{mod("LocalElementalDamage", Inc, Num(c.n(1)))} }),
	`all attacks with this weapon are critical strikes`:              modList{mod("WeaponData", List, DataRef{Key: "CritChance", Value: Num(100)})},
	`this weapon's critical strike chance is ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("WeaponData", List, DataRef{Key: "CritChance", Value: Num(c.n(1))})}
	}),
	`counts as dual wielding`:                     modList{mod("WeaponData", List, DataRef{Key: "countsAsDualWielding", Value: Bool(true)})},
	`counts as all one handed melee weapon types`: modList{mod("WeaponData", List, DataRef{Key: "countsAsAll1H", Value: Bool(true)})},
	`no block chance`:                             modList{mod("ArmourData", List, DataRef{Key: "BlockChance", Value: Num(0)})},
	`no chance to block`:                          modList{mod("ArmourData", List, DataRef{Key: "BlockChance", Value: Num(0)})},
	`has no energy shield`:                        modList{mod("ArmourData", List, DataRef{Key: "EnergyShield", Value: Num(0)})},
	`hits can't be evaded`:                        modList{flag("CannotBeEvaded", &CondTag{Var: "{Hand}Attack"})},
	`causes bleeding on hit`:                      modList{mod("BleedChance", Base, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`poisonous hit`:                               modList{mod("PoisonChance", Base, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`attacks with this weapon deal double damage`: modList{modf("DoubleDamageChance", Base, Num(100), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})},
	`hits with this weapon gain ([0-9]+)% of physical damage as extra cold or lightning damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalDamageGainAsColdOrLightning", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`hits with this weapon shock enemies as though dealing ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ShockAsThoughDealing", More, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`hits with this weapon freeze enemies as though dealing ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("FreezeAsThoughDealing", More, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`ignites inflicted with this weapon deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordIgnite, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`hits with this weapon always ignite, freeze, and shock`:         modList{modf("EnemyIgniteChance", Base, Num(100), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), modf("EnemyFreezeChance", Base, Num(100), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), modf("EnemyShockChance", Base, Num(100), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})},
	`attacks with this weapon deal double damage to chilled enemies`: modList{modf("DoubleDamageChance", Base, Num(100), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}, &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"})},
	`life leech from hits with this weapon applies instantly`:        modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`life leech from hits with this weapon is instant`:               modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`mana leech from hits with this weapon is instant`:               modList{mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`life leech from melee damage is instant`:                        modList{modf("InstantLifeLeech", Base, Num(100), FlagMelee, KeywordNone)},
	`gain life from leech instantly from hits with this weapon`:      modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})},
	`([0-9]+)% of leech from hits with this weapon is instant per enemy power`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("InstantLifeLeech", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}, &MultiplierTag{Var: "EnemyPower"}), modf("InstantManaLeech", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}, &MultiplierTag{Var: "EnemyPower"}), modf("InstantEnergyShieldLeech", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}, &MultiplierTag{Var: "EnemyPower"})}
	}),
	`instant recovery`:                                                                                       modList{mod("FlaskInstantRecovery", Base, Num(100))},
	`([0-9]+)% of recovery applied instantly`:                                                                modFn(func(c caps) []*Mod { return []*Mod{mod("FlaskInstantRecovery", Base, Num(c.n(1)))} }),
	`instant recovery when on low life`:                                                                      modList{mod("FlaskLowLifeInstantRecovery", Base, Num(100)), mods("Dummy", Dummy, Num(1), "", &CondTag{Var: "LowLife"})},
	`life flasks used while on low life apply recovery instantly`:                                            modList{mod("LifeFlaskInstantRecovery", Base, Num(100), &CondTag{Var: "LowLife"})},
	`mana flasks used while on low mana apply recovery instantly`:                                            modList{mod("ManaFlaskInstantRecovery", Base, Num(100), &CondTag{Var: "LowMana"})},
	`has no attribute requirements`:                                                                          modList{flag("NoAttributeRequirements")},
	`trigger a socketed spell when you attack with this weapon`:                                              modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you attack with this weapon, with a ([0-9.]+) second cooldown`:            modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you use a skill`:                                                          modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnSkillUse", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you use a skill, with a ([0-9]+) second cooldown`:                         modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnSkillUse", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you use a skill, with a ([0-9]+) second cooldown and ([0-9]+)% more cost`: modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnSkillUse", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger socketed spells when you focus`:                                                                 modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellFromHelmet", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger socketed spells when you focus, with a ([0-9.]+) second cooldown`:                               modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellFromHelmet", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you attack with a bow`:                                                    modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnBowAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you attack with a bow, with a ([0-9.]+) second cooldown`:                  modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnBowAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed bow skill when you attack with a bow`:                                                modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed bow skill when you attack with a bow, with a ([0-9.]+) second cooldown`:              modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed bow skill when you cast a spell while wielding a bow`:                                modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`([0-9]+)% chance to trigger socketed spell on kill, with a ([0-9.]+) second cooldown`:                   modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnKill", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`([0-9]+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least ([0-9]+) mana to use a skill`: modFn(func(c caps) []*Mod {
		return []*Mod{mods("KitavaTriggerChance", Base, Num(c.n(1)), "Kitava's Thirst"), mods("KitavaRequiredManaCost", Base, Num(c.n(2)), "Kitava's Thirst"), mod("ExtraSupport", List, SkillRef{SkillID: "SupportCastOnManaSpent", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`([0-9]+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least ([0-9]+) mana on an upfront cost to use or trigger a skill, with a ([0-9.]+) second cooldown`: modFn(func(c caps) []*Mod {
		return []*Mod{mods("KitavaTriggerChance", Base, Num(c.n(1)), "Kitava's Thirst"), mods("KitavaRequiredManaCost", Base, Num(c.n(2)), "Kitava's Thirst"), mod("ExtraSupport", List, SkillRef{SkillID: "SupportCastOnManaSpent", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`([0-9]+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least ([0-9]+) life on an upfront cost to use or trigger a skill, with a ([0-9.]+) second cooldown`: modFn(func(c caps) []*Mod {
		return []*Mod{mods("FoulbornKitavaTriggerChance", Base, Num(c.n(1)), "Kitava's Thirst"), mods("FoulbornKitavaRequiredLifeCost", Base, Num(c.n(2)), "Kitava's Thirst"), mod("ExtraSupport", List, SkillRef{SkillID: "SupportCastOnLifeSpent", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`trigger a socketed fire spell on hit, with a ([0-9.]+) second cooldown`: modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerFireSpellOnHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	// Socketed gem modifiers
	`([+\-][0-9]+) to level of socketed gems`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "all", Key: "level", Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`([+\-][0-9]+) to level of socketed skill gems per socketed gem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "grants_active_skill", Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &MultiplierTag{Var: "SocketedGemsIn{SlotName}"})}
	}),
	`([+\-][0-9]+)% to quality of all skill gems`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "grants_active_skill", Key: "quality", Value: opt(c.n(1)), KeyOfScaledMod: "value"})}
	}),
	`([+\-][0-9]+) to level of all elemental skill gems if the stars are aligned`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{KeywordList: []string{"elemental", "grants_active_skill"}, Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"}, &CondTag{Var: "StarsAreAligned"})}
	}),
	`([+\-][0-9]+) to level of all elemental support gems if the stars are aligned`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{KeywordList: []string{"elemental", "support"}, Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"}, &CondTag{Var: "StarsAreAligned"})}
	}),
	`([+\-][0-9]+) to level of socketed active skill gems per ([0-9]+) player levels`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "grants_active_skill", Key: "level", Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &MultiplierTag{Var: "Level", Div: opt(c.n(2))})}
	}),
	`([+\-][0-9]+) to level of all ([a-zA-Z]+) skill gems if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: c.s(2), Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"}, &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(4)) + "Item", Threshold: opt(c.n(3))})}
	}),
	`([+\-][0-9]+) to level of all ([a-zA-Z]+) skill gems if a?t? ?l?e?a?s?t? ?([0-9]+) ([a-zA-Z]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: c.s(2), Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"}, &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(4)) + firstToUpper(c.s(5)) + "Item", Threshold: opt(c.n(3))})}
	}),
	`([+\-][0-9]+) to level of all ?([a-zA-Z\- ]*) support gems if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: c.s(2), Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"}, &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(4)) + "Item", Threshold: opt(c.n(3))})}
	}),
	`([+\-][0-9]+) to level of socketed skill gems per ([0-9]+) player levels`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "grants_active_skill", Key: "level", Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &MultiplierTag{Var: "Level", Div: opt(c.n(2))})}
	}),
	`([+\-][0-9]+) to level of socketed gems while there is a single gem socketed in this item`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "all", Key: "level", Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &MultiplierTag{IsThreshold: true, Var: "SocketedGemsIn{SlotName}", Threshold: opt(1), Equals: true})}
	}),
	`socketed gems fire an additional projectile`: modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("ProjectileCount", Base, Num(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed gems fire ([0-9]+) additional projectiles`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ProjectileCount", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`socketed gems reserve no mana`:     modList{mod("ManaReserved", More, Num(-100), &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed gems have no reservation`: modList{mod("Reserved", More, Num(-100), &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed skill gems get a ([0-9]+)% mana multiplier`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("SupportManaMultiplier", More, Num(c.n(1)-100))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`socketed skill gems get a ([0-9]+)% cost & reservation multiplier`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("SupportManaMultiplier", More, Num(c.n(1)-100))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`socketed gems have blood magic`:                      modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportBloodMagicUniquePrismGuardian", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed gems cost and reserve life instead of mana`: modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportBloodMagicUniquePrismGuardian", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed gems have elemental equilibrium`:            modList{mod("Keystone", List, Str("Elemental Equilibrium"))},
	`socketed gems have secrets of suffering`:             modList{flag("CannotIgnite", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}), flag("CannotChill", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}), flag("CannotFreeze", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}), flag("CannotShock", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}), flag("CritAlwaysAltAilments", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed skills deal double damage`:                  modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("DoubleDamageChance", Base, Num(100))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed gems gain ([0-9]+)% of physical damage as extra lightning damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("PhysicalDamageGainAsLightning", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`socketed red gems get ([0-9]+)% physical damage as extra fire damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("PhysicalDamageGainAsFire", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "strength"})}
	}),
	`socketed non-channelling bow skills are triggered by snipe`: modList{},
	`grants level ([0-9]+) snipe skill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkill", List, SkillRef{SkillID: "Snipe", Level: opt(c.n(1))}), mod("ExtraSupport", List, SkillRef{SkillID: "ChannelledSnipeSupport", Level: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`socketed triggered bow skills deal ([0-9]+)% less damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("Damage", More, Num(-c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "bow"}, &SkillTypeTag{SkillType: SkillTypeTriggerable})}
	}),
	`socketed vaal skills require ([0-9]+)% less souls per use`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("SoulCost", More, Num(-c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &SkillTypeTag{SkillType: SkillTypeVaal})}
	}),
	`hits from socketed vaal skills ignore enemy monster resistances`:               modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("IgnoreElementalResistances")}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &SkillTypeTag{SkillType: SkillTypeVaal}), mod("ExtraSkillMod", List, ModRef{Mod: flag("IgnoreChaosResistance")}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &SkillTypeTag{SkillType: SkillTypeVaal})},
	`hits from socketed vaal skills ignore enemy monster physical damage reduction`: modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("IgnoreEnemyPhysicalDamageReduction")}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &SkillTypeTag{SkillType: SkillTypeVaal})},
	`socketed vaal skills grant elusive when used`:                                  modList{flag("Condition:CanBeElusive")},
	`damage with hits from socketed vaal skills is lucky`:                           modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("LuckyHits")}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &SkillTypeTag{SkillType: SkillTypeVaal})},
	// Global gem modifiers
	`gems socketed in red sockets have [+\-]([0-9]+) to level`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "all", Key: "level", Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", SocketColor: "R"})}
	}),
	`gems socketed in green sockets have [+\-]([0-9]+)% to quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: "all", Key: "quality", Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", SocketColor: "G"})}
	}),
	`\+([0-9]+)% to fire resistance when socketed with a red gem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketProperty", List, PropertyModRef{Mod: mod("FireResist", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "strength", Sockets: []float64{1}})}
	}),
	`\+([0-9]+)% to cold resistance when socketed with a green gem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketProperty", List, PropertyModRef{Mod: mod("ColdResist", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "dexterity", Sockets: []float64{1}})}
	}),
	`\+([0-9]+)% to lightning resistance when socketed with a blue gem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketProperty", List, PropertyModRef{Mod: mod("LightningResist", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "intelligence", Sockets: []float64{1}})}
	}),
	// Doomsower, Lion Sword
	`attack skills gain ([0-9]+)% of physical damage as extra fire damage per socketed red gem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketProperty", List, PropertyModRef{Mod: modf("PhysicalDamageGainAsFire", Base, Num(c.n(1)), FlagAttack, KeywordNone)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "strength", Sockets: []float64{1, 2, 3, 4, 5, 6}})}
	}),
	`([0-9]+)% of damage taken recouped as life per socketed red gem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketProperty", List, PropertyModRef{Mod: mod("LifeRecoup", Base, Num(c.n(1)))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Keyword: "strength", Sockets: []float64{1, 2, 3, 4, 5, 6}})}
	}),
	`you have vaal pact while all socketed gems are red`:         modList{mod("GroupProperty", List, PropertyModRef{Mod: mod("Keystone", List, Str("Vaal Pact"))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", SocketColor: "R", SocketsAll: true})},
	`you have immortal ambition while all socketed gems are red`: modList{mod("GroupProperty", List, PropertyModRef{Mod: mod("Keystone", List, Str("Immortal Ambition"))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", SocketColor: "R", SocketsAll: true})},
	// Mahuxotl's Machination Steel Kite Shield
	`everlasting sacrifice`: modList{flag("Condition:EverlastingSacrifice")},
	// Self hit dmg
	`take ([0-9]+) (.+) damage when you ignite an enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EyeOfInnocenceSelfDamage", List, SelfDamage{BaseDamage: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	`([0-9]+) (.+) damage taken on minion death`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("HeartboundLoopSelfDamage", List, SelfDamage{BaseDamage: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	`take ([0-9]+) (.+) damage when herald of thunder hits an enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("StormSecretSelfDamage", List, SelfDamage{BaseDamage: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	`take ([0-9]+) (.+) damage when you use a skill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnmitysEmbraceSelfDamage", List, SelfDamage{BaseDamage: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	`your skills deal you ([0-9]+)% of mana cost as (.+) damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ScoldsBridleSelfDamage", List, SelfDamage{DmgMult: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	`your skills deal you ([0-9]+)% of mana spent on upfront skill mana costs as (.+) damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ScoldsBridleSelfDamage", List, SelfDamage{DmgMult: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	`when you attack, take ([0-9]+)% of life as (.+) damage for each warcry exerting the attack`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EchoesOfCreationSelfDamage", List, SelfDamage{DmgMult: opt(c.n(1)), DamageType: c.s(2)})}
	}),
	// Extra skill/support
	`grants ([^0-9]+)`:           modFn(func(c caps) []*Mod { return grantedExtraSkill(c.s(1), 1) }),
	`grants level ([0-9]+) (.+)`: modFn(func(c caps) []*Mod { return grantedExtraSkill(c.s(2), c.n(1)) }),
	`grants level ([0-9]+) (.+), which will be used by shaper memory`:                                               modFn(func(c caps) []*Mod { return grantedExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when equipped`:                                                    modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) on [a-zA-Z]+`:                                                     modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`use level ([0-9]+) (.+) on [a-zA-Z]+`:                                                                          modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you attack`:                                                  modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you deal a critical strike`:                                  modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when hit`:                                                         modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you kill an enemy`:                                           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`[ct][ar][si][tg]g?e?r?s? level ([0-9]+) (.+) when you use a skill`:                                             modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`(.+) can trigger level ([0-9]+) (.+)`:                                                                          modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{SourceSkill: c.s(1)}) }),
	`trigger level ([0-9]+) (.+) when you use a skill while you have a spirit charge`:                               modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit an enemy while cursed`:                                                modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a bleeding enemy`:                                                     modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you attack with this weapon`:                                                  modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a rare or unique enemy`:                                               modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a rare or unique enemy and have no mark`:                              modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you hit a frozen enemy`:                                                       modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you kill a frozen enemy`:                                                      modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you attack with a bow`:                                                        modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you block`:                                                                    modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when animated guardian kills an enemy`:                                             modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you lose cat's stealth`:                                                       modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when your trap is triggered`:                                                       modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on hit with this weapon`:                                                           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit`:                                                                      modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit while cursed`:                                                         modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit with this weapon`:                                                     modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) every [0-9.]+ seconds while phasing`:                                               modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you gain avian's might or avian's flight`:                                     modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on melee hit if you have at least ([0-9]+) strength`:                               modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on critical strike with cleave or reave`:                                           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{OnCrit: true}) }),
	`trigger level ([0-9]+) (.+) on melee critical strike`:                                                          modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{OnCrit: true}) }),
	`trigger level ([0-9]+) (.+) on critical strike against marked unique enemy`:                                    modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{OnCrit: true}) }),
	`trigger level ([0-9]+) (.+) on critical strike`:                                                                modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{OnCrit: true}) }),
	`trigger level ([0-9]+) (.+) when you take a critical strike from a unique enemy`:                               modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you suppress spell damage from a unique enemy`:                                modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you block damage from a unique enemy`:                                         modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on taking a savage hit from a unique enemy`:                                        modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when a totem dies while a unique enemy is in your presence`:                        modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you use a travel skill while a unique enemy is in your presence`:              modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you reach maximum rage while a unique enemy is in your presence`:              modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when you reach low life while a unique enemy is in your presence`:                  modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when energy shield recharge starts while a unique enemy is in your presence`:       modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) when your ward breaks`:                                                             modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) once every second`:                                                                 modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`triggers level ([0-9]+) (.+)`:                                                                                  modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+) on attack critical strike against a rare or unique enemy and y?o?u? ?have no mark`: modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{OnCrit: true}) }),
	`triggers level ([0-9]+) (.+) when equipped`:                                                                    modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`triggers level ([0-9]+) (.+) when allocated`:                                                                   modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`([0-9]+)% chance to attack with level ([0-9]+) (.+) on melee hit`:                                              modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when animated weapon kills an enemy`:                           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) on melee hit`:                                                  modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) [ow][nh]e?n? ?y?o?u? kill ?a?n? ?e?n?e?m?y?`:                   modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when you use a socketed skill`:                                 modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when you gain avian's might or avian's flight`:                 modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) on critical strike with this weapon`: modFn(func(c caps) []*Mod {
		return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1)), OnCrit: true})
	}),
	`([0-9]+)% chance to trigger level ([0-9]+) (.+) when you or a nearby ally kill an enemy, or hit a rare or unique enemy`: modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to trigger (.+) when you kill an enemy`:                                                                modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), 1, triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`([0-9]+)% chance to [ct][ar][si][tg]g?e?r? level ([0-9]+) (.+) on [a-zA-Z]+`:                                            modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(3), c.n(2), triggerOpts{TriggerChance: opt(c.n(1))}) }),
	`attack with level ([0-9]+) (.+) when you kill a bleeding enemy`:                                                         modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`triggers? level ([0-9]+) (.+) when you kill a bleeding enemy`:                                                           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`curse enemies with ([^0-9]+) on [a-zA-Z]+`:                                                                              modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[0-9]+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with ([^0-9]+) on [a-zA-Z]+`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(1)), Level: opt(1), NoSupports: true, Triggered: true})}
	}),
	`curse enemies with level ([0-9]+) ([^0-9]+) on [a-zA-Z]+, which can apply to hexproof enemies`: modFn(func(c caps) []*Mod {
		return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{NoSupports: true, IgnoreHexproof: true})
	}),
	`curse enemies with level ([0-9]+) (.+) on [a-zA-Z]+`:                      modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1), triggerOpts{NoSupports: true}) }),
	`[ct][ar][si][tg]g?e?r?s? (.+) on [a-zA-Z]+`:                               modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[at][tr][ti][ag][cg][ke]r? (.+) on [a-zA-Z]+`:                             modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[at][tr][ti][ag][cg][ke]r? with (.+) on [a-zA-Z]+`:                        modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[ct][ar][si][tg]g?e?r?s? (.+) when hit`:                                   modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[at][tr][ti][ag][cg][ke]r? (.+) when hit`:                                 modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[at][tr][ti][ag][cg][ke]r? with (.+) when hit`:                            modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[ct][ar][si][tg]g?e?r?s? (.+) when your skills or minions kill`:           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[at][tr][ti][ag][cg][ke]r? (.+) when you take a critical strike`:          modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`[at][tr][ti][ag][cg][ke]r? with (.+) when you take a critical strike`:     modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`trigger commandment of inferno on critical strike`:                        modList{mod("ExtraSkill", List, SkillRef{SkillID: "UniqueEnchantmentOfInfernoOnCrit", Level: opt(1), NoSupports: true, Triggered: true}), mod("ExtraSkillMod", List, ModRef{Mod: mod("SkillData", List, DataRef{Key: "triggerOnCrit", Value: Bool(true)})}, &SkillIDTag{SkillID: "UniqueEnchantmentOfInfernoOnCrit"})},
	`trigger (.+) on critical strike`:                                          modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true, OnCrit: true}) }),
	`triggers? (.+) when you take a critical strike`:                           modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(1), 1, triggerOpts{NoSupports: true}) }),
	`socketed [a-zA-Z+]* ?gems a?r?e? ?supported by level ([0-9]+) (.+)`:       modFn(func(c caps) []*Mod { return extraSupport(c.s(2), c.n(1)) }),
	`socketed [a-zA-Z+]* ?spells a?r?e? ?supported by level ([0-9]+) (.+)`:     modFn(func(c caps) []*Mod { return extraSupport(c.s(2), c.n(1)) }),
	`skills from equipped (.+) are supported by level ([0-9]+) (.+)`:           modFn(func(c caps) []*Mod { return extraSupport(c.s(3), c.n(2), c.s(1)) }),
	`skills socketed in your (.+) are supported by level ([0-9]+) (.+)`:        modFn(func(c caps) []*Mod { return extraSupport(c.s(3), c.n(2), c.s(1)) }),
	`socketed hex curse skills are triggered by doedre's effigy when summoned`: modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportCursePillarTriggerCurses", Level: opt(20)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`socketed projectile spells have \+([0-9.]+) seconds to cooldown`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CooldownRecovery", Base, Num(c.n(1)), &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"}, &SkillTypeTag{SkillType: SkillTypeProjectile}, &SkillTypeTag{SkillType: SkillTypeSpell})}
	}),
	`trigger level ([0-9]+) (.+) every ([0-9]+) seconds`: modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`trigger level ([0-9]+) (.+), (.+) or (.+) every ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(2)), Level: opt(c.n(1)), Triggered: true}), mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(3)), Level: opt(c.n(1)), Triggered: true}), mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(4)), Level: opt(c.n(1)), Triggered: true})}
	}),
	`offering skills triggered this way also affect you`:                                                    modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("SkillData", List, DataRef{Key: "buffNotPlayer", Value: Bool(false)})}, &SkillNameTag{SkillNameList: []string{"Bone Offering", "Flesh Offering", "Spirit Offering"}})},
	`trigger level ([0-9]+) (.+) after spending a total of ([0-9]+) mana`:                                   modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`consumes a void charge to trigger level ([0-9]+) (.+) when you fire arrows`:                            modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`consumes a void charge to trigger level ([0-9]+) (.+) when you fire arrows with a non-triggered skill`: modFn(func(c caps) []*Mod { return triggerExtraSkill(c.s(2), c.n(1)) }),
	`your hits treat cold resistance as ([0-9]+)% higher than actual value`:                                 modFn(func(c caps) []*Mod { return []*Mod{modf("ColdPenetration", Base, Num(-c.n(1)), FlagNone, KeywordHit)} }),
	// Conversion
	`increases and reductions to minion damage also affects? you`: modList{flag("MinionDamageAppliesToPlayer")},
	`increases and reductions to minion damage also affects? you at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("MinionDamageAppliesToPlayer"), mod("ImprovedMinionDamageAppliesToPlayer", Max, Num(c.n(1)))}
	}),
	`increases and reductions to minion damage also affect dominating blow and absolution at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("MinionDamageAppliesToPlayer", &SkillNameTag{SkillNameList: []string{"Dominating Blow", "Absolution"}, IncludeTransfigured: true}), mod("ImprovedMinionDamageAppliesToPlayer", Max, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Dominating Blow", "Absolution"}, IncludeTransfigured: true})}
	}),
	`increases and reductions to minion attack speed also affects? you`: modList{flag("MinionAttackSpeedAppliesToPlayer")},
	`increases and reductions to minion cast speed also affects? you`:   modList{flag("MinionCastSpeedAppliesToPlayer")},
	`increases and reductions to minion maximum life also apply to you at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("MinionLifeAppliesToPlayer"), mod("ImprovedMinionLifeAppliesToPlayer", Max, Num(c.n(1)))}
	}),
	`increases and reductions to cast speed apply to attack speed at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("CastSpeedAppliesToAttacks"), mod("ImprovedCastSpeedAppliesToAttacks", Max, Num(c.n(1)))}
	}),
	`increases and reductions to cast speed apply to attack speed`:                    modFn(func(c caps) []*Mod { return []*Mod{flag("CastSpeedAppliesToAttacks")} }),
	`increases and reductions to spell damage also apply to attacks`:                  modList{flag("SpellDamageAppliesToAttacks")},
	`increases and reductions to your evasion rating also apply to your spell damage`: modList{flag("EvasionAppliesToSpellDamage")},
	`arcane might`: modList{flag("SpellDamageAppliesToAttacks")},
	`([0-9]+)% arcane might`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("SpellDamageAppliesToAttacks"), mod("ImprovedSpellDamageAppliesToAttacks", Max, Num(c.n(1)))}
	}),
	`attacks have ([0-9]+)% arcane might`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("SpellDamageAppliesToAttacks"), mod("ImprovedSpellDamageAppliesToAttacks", Max, Num(c.n(1)))}
	}),
	`attacks have ([0-9]+)% arcane might while wielding a wand`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("SpellDamageAppliesToAttacks", &CondTag{Var: "UsingWand"}), mod("ImprovedSpellDamageAppliesToAttacks", Max, Num(c.n(1)), &CondTag{Var: "UsingWand"})}
	}),
	`retaliation skills have ([0-9]+)% arcane might`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("SpellDamageAppliesToAttacks", &SkillTypeTag{SkillType: SkillTypeRetaliation}), mod("ImprovedSpellDamageAppliesToAttacks", Max, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeRetaliation})}
	}),
	`attacks have arcane might while wielding a wand`: modList{flag("SpellDamageAppliesToAttacks", &CondTag{Var: "UsingWand"})},
	`increases and reductions to spell damage also apply to attacks at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("SpellDamageAppliesToAttacks"), mod("ImprovedSpellDamageAppliesToAttacks", Max, Num(c.n(1)))}
	}),
	`increases and reductions to spell damage also apply to attack damage with retaliation skills at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("SpellDamageAppliesToAttacks", &SkillTypeTag{SkillType: SkillTypeRetaliation}), mod("ImprovedSpellDamageAppliesToAttacks", Max, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeRetaliation})}
	}),
	`increases and reductions to spell damage also apply to attacks while wielding a wand`: modList{flag("SpellDamageAppliesToAttacks", &CondTag{Var: "UsingWand"})},
	`increases and reductions to maximum mana also apply to shock effect at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ManaAppliesToShockEffect"), mod("ImprovedManaAppliesToShockEffect", Max, Num(c.n(1)))}
	}),
	`increases and reductions to ([a-zA-Z]+) damage also apply to effect of auras from ([a-zA-Z]+) skills at ([0-9]+)% of their value, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{flag(firstToUpper(c.s(1)) + "DamageAppliesTo" + (firstToUpper(c.s(2)) + "AuraEffect")), mod(("Improved"+firstToUpper(c.s(1)))+"DamageAppliesTo"+(firstToUpper(c.s(2))+"AuraEffect"), Base, Num(c.n(3))), mod(firstToUpper(c.s(1))+"DamageAppliesTo"+(firstToUpper(c.s(2))+"AuraEffectLimit"), Max, Num(c.n(4)))}
	}),
	`modifiers to claw damage also apply to unarmed`:                                                          modList{flag("ClawDamageAppliesToUnarmed")},
	`modifiers to claw damage also apply to unarmed attack damage`:                                            modList{flag("ClawDamageAppliesToUnarmed")},
	`modifiers to claw damage also apply to unarmed attack damage with melee skills`:                          modList{flag("ClawDamageAppliesToUnarmed")},
	`modifiers to claw attack speed also apply to unarmed`:                                                    modList{flag("ClawAttackSpeedAppliesToUnarmed")},
	`modifiers to claw attack speed also apply to unarmed attack speed`:                                       modList{flag("ClawAttackSpeedAppliesToUnarmed")},
	`modifiers to claw attack speed also apply to unarmed attack speed with melee skills`:                     modList{flag("ClawAttackSpeedAppliesToUnarmed")},
	`modifiers to claw critical strike chance also apply to unarmed`:                                          modList{flag("ClawCritChanceAppliesToUnarmed")},
	`modifiers to claw critical strike chance also apply to unarmed attack critical strike chance`:            modList{flag("ClawCritChanceAppliesToUnarmed")},
	`modifiers to claw critical strike chance also apply to unarmed critical strike chance with melee skills`: modList{flag("ClawCritChanceAppliesToUnarmed")},
	`increases and reductions to light radius also apply to accuracy`:                                         modList{flag("LightRadiusAppliesToAccuracy")},
	`increases and reductions to light radius also apply to area of effect at 50% of their value`:             modList{flag("LightRadiusAppliesToAreaOfEffect")},
	`increases and reductions to light radius also apply to damage`:                                           modList{flag("LightRadiusAppliesToDamage")},
	`increases and reductions to cast speed also apply to trap throwing speed`:                                modList{flag("CastSpeedAppliesToTrapThrowingSpeed")},
	`increases and reductions to armour also apply to energy shield recharge rate at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ArmourAppliesToEnergyShieldRecharge"), mod("ImprovedArmourAppliesToEnergyShieldRecharge", Max, Num(c.n(1)))}
	}),
	`increases and reductions to projectile speed also apply to damage with bows`: modList{flag("ProjectileSpeedAppliesToBowDamage")},
	`modifiers to maximum ([a-zA-Z]+) resistance also apply to maximum ([a-zA-Z]+) and ([a-zA-Z]+) resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(1))+"MaxResConvertTo"+firstToUpper(c.s(2)), Base, Num(100)), mod(firstToUpper(c.s(1))+"MaxResConvertTo"+firstToUpper(c.s(3)), Base, Num(100))}
	}),
	`modifiers to ([a-zA-Z]+) resistance also apply to ([a-zA-Z]+) and ([a-zA-Z]+) resistances at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(1))+"ResConvertTo"+firstToUpper(c.s(2)), Base, Num(c.n(4))), mod(firstToUpper(c.s(1))+"ResConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(4)))}
	}),
	`gain ([0-9]+)% of bow physical damage as extra damage of each element`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalDamageGainAsLightning", Base, Num(c.n(1)), FlagBow, KeywordNone), modf("PhysicalDamageGainAsCold", Base, Num(c.n(1)), FlagBow, KeywordNone), modf("PhysicalDamageGainAsFire", Base, Num(c.n(1)), FlagBow, KeywordNone)}
	}),
	`gain ([0-9]+)% of weapon physical damage as extra damage of each element`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalDamageGainAsLightning", Base, Num(c.n(1)), FlagWeapon, KeywordNone), modf("PhysicalDamageGainAsCold", Base, Num(c.n(1)), FlagWeapon, KeywordNone), modf("PhysicalDamageGainAsFire", Base, Num(c.n(1)), FlagWeapon, KeywordNone)}
	}),
	`gain ([0-9]+)% of physical damage as extra damage of each element per spirit charge`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsLightning", Base, Num(c.n(1)), &MultiplierTag{Var: "SpiritCharge"}), mod("PhysicalDamageGainAsCold", Base, Num(c.n(1)), &MultiplierTag{Var: "SpiritCharge"}), mod("PhysicalDamageGainAsFire", Base, Num(c.n(1)), &MultiplierTag{Var: "SpiritCharge"})}
	}),
	`gain ([0-9]+)% of physical damage as extra damage of each element if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsLightning", Base, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))}), mod("PhysicalDamageGainAsCold", Base, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))}), mod("PhysicalDamageGainAsFire", Base, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))})}
	}),
	`gain ([0-9]+)% of weapon physical damage as extra damage of an? r?a?n?d?o?m? ?element`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalDamageGainAsRandom", Base, Num(c.n(1)), FlagWeapon, KeywordNone)}
	}),
	`gain ([0-9]+)% of physical damage as extra damage of a random element`: modFn(func(c caps) []*Mod { return []*Mod{mod("PhysicalDamageGainAsRandom", Base, Num(c.n(1)))} }),
	`([0-9]+)% chance for hits to deal ([0-9]+)% of physical damage as extra damage of a random element`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsRandom", Base, Num((c.n(1) * c.n(2) / 100)))}
	}),
	`gain ([0-9]+)% of physical damage as extra damage of a random element while you are ignited`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsRandom", Base, Num(c.n(1)), &CondTag{Var: "Ignited"})}
	}),
	`([0-9]+)% of physical damage from hits with this weapon is converted to a random element`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageConvertToRandom", Base, Num(c.n(1)), &CondTag{Var: "{Hand}Attack"})}
	}),
	`([0-9]+)% of physical damage converted to a random element`: modFn(func(c caps) []*Mod { return []*Mod{mod("PhysicalDamageConvertToRandom", Base, Num(c.n(1)))} }),
	`nearby enemies convert ([0-9]+)% of their ([a-zA-Z]+) damage to ([a-zA-Z]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)))})}
	}),
	`enemies ignited by you have ([0-9]+)% of ([a-zA-Z]+) damage they deal converted to ([a-zA-Z]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)), &CondTag{Var: "Ignited"})})}
	}),
	`enemies shocked by you have ([0-9]+)% of ([a-zA-Z]+) damage they deal converted to ([a-zA-Z]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)), &CondTag{Var: "Shocked"})})}
	}),
	`enemies poisoned by you have ([0-9]+)% of ([a-zA-Z]+) damage they deal converted to ([a-zA-Z]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)), &CondTag{Var: "Poisoned"})})}
	}),
	`shield crush and spectral shield throw do not gain added physical damage based on armour or evasion on shield`: modList{flag("Condition:ShieldThrowCrushNoArmourEvasion", &SkillNameTag{SkillNameList: []string{"Spectral Shield Throw", "Shield Crush"}, IncludeTransfigured: true})},
	`shield crush and spectral shield throw gains ([0-9]+) to ([0-9]+) added lightning damage per ([0-9]+) energy shield on shield`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LightningMin", Base, c.v(1), FlagNone, KeywordNone, &CondTag{Var: "OffHandAttack"}, &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnWeapon 2", Div: opt(c.n(3))}, &SkillNameTag{SkillNameList: []string{"Spectral Shield Throw", "Shield Crush"}, IncludeTransfigured: true}), modf("LightningMax", Base, c.v(2), FlagNone, KeywordNone, &CondTag{Var: "OffHandAttack"}, &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnWeapon 2", Div: opt(c.n(3))}, &SkillNameTag{SkillNameList: []string{"Spectral Shield Throw", "Shield Crush"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% of shield crush and spectral shield throw physical damage converted to lightning damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("SkillPhysicalDamageConvertToLightning", Base, Num(c.n(1)), FlagNone, KeywordNone, &SkillNameTag{SkillNameList: []string{"Spectral Shield Throw", "Shield Crush"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% of exsanguinate and reap physical damage converted to fire damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("SkillPhysicalDamageConvertToFire", Base, Num(c.n(1)), FlagNone, KeywordNone, &SkillNameTag{SkillNameList: []string{"Exsanguinate", "Reap"}, IncludeTransfigured: true})}
	}),
	`-([0-9]+)% of toxic rain physical damage converted to chaos damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("SkillPhysicalDamageConvertToChaos", Base, Num(-c.n(1)), FlagNone, KeywordNone, &SkillNameTag{SkillName: "Toxic Rain", IncludeTransfigured: true})}
	}),
	`cobra lash and venom gyre have -([0-9]+)% of physical damage converted to chaos damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("SkillPhysicalDamageConvertToChaos", Base, Num(-c.n(1)), FlagNone, KeywordNone, &SkillNameTag{SkillNameList: []string{"Cobra Lash", "Venom Gyre"}})}
	}),
	`([0-9]+)% of consecrated path and purifying flame fire damage converted to chaos damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("SkillFireDamageConvertToChaos", Base, Num(c.n(1)), FlagNone, KeywordNone, &SkillNameTag{SkillNameList: []string{"Consecrated Path", "Purifying Flame"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% of manabond and stormbind lightning damage converted to cold damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("SkillLightningDamageConvertToCold", Base, Num(c.n(1)), FlagNone, KeywordNone, &SkillNameTag{SkillNameList: []string{"Manabond", "Stormbind"}, IncludeTransfigured: true})}
	}),
	`exsanguinate debuffs deal fire damage per second instead of physical damage per second`: modList{flag("Condition:ExsanguinateDebuffIsFireDamage", &SkillNameTag{SkillName: "Exsanguinate", IncludeTransfigured: true})},
	`reap debuffs deal fire damage per second instead of physical damage per second`:         modList{flag("Condition:ReapDebuffIsFireDamage", &SkillNameTag{SkillName: "Reap"})},
	// Crit
	`your critical strike chance is lucky`:                                   modList{flag("CritChanceLucky")},
	`your critical strike chance is lucky while on low life`:                 modList{flag("CritChanceLucky", &CondTag{Var: "LowLife"})},
	`your critical strike chance is lucky while focus?sed`:                   modList{flag("CritChanceLucky", &CondTag{Var: "Focused"})},
	`your critical strikes do not deal extra damage`:                         modList{flag("NoCritMultiplier")},
	`critical strikes do not deal extra damage`:                              modList{flag("NoCritMultiplier")},
	`critical strikes with this weapon do not deal extra damage`:             modList{flag("NoCritMultiplier", &CondTag{Var: "{Hand}Attack"})},
	`minion critical strikes do not deal extra damage`:                       modList{mod("MinionModifier", List, ModRef{Mod: flag("NoCritMultiplier")})},
	`lightning damage with non-critical strikes is lucky`:                    modList{flag("LightningNoCritLucky")},
	`your damage with critical strikes is lucky`:                             modList{flag("CritLucky")},
	`spell critical strike chance bifurcates`:                                modList{flagf("BifurcateCrit", FlagSpell, KeywordNone)},
	`critical strikes deal no damage`:                                        modList{mod("Damage", More, Num(-100), &CondTag{Var: "CriticalStrike"})},
	`critical strike chance is increased by uncapped lightning resistance`:   modList{flag("CritChanceIncreasedByUncappedLightningRes")},
	`critical strike chance is increased by lightning resistance`:            modList{flag("CritChanceIncreasedByLightningRes")},
	`critical strike chance is increased by overcapped lightning resistance`: modList{flag("CritChanceIncreasedByOvercappedLightningRes")},
	`barrage and frenzy have ([0-9]+)% increased critical strike chance per endurance charge`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritChance", Inc, Num(c.n(1)), &MultiplierTag{Var: "EnduranceCharge"}, &SkillNameTag{SkillNameList: []string{"Barrage", "Frenzy"}, IncludeTransfigured: true})}
	}),
	`non-critical strikes deal ([0-9]+)% damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(-100+c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "CriticalStrike", Neg: true})}
	}),
	`non-critical strikes deal no damage`: modList{modf("Damage", More, Num(-100), FlagHit, KeywordNone, &CondTag{Var: "CriticalStrike", Neg: true})},
	`non-critical strikes deal ([0-9]+)% less damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(-c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "CriticalStrike", Neg: true})}
	}),
	`spell skills always deal critical strikes on final repeat`:        modList{flagf("SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat", FlagSpell, KeywordNone)},
	`spell skills cannot deal critical strikes except on final repeat`: modList{flagf("SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat", FlagSpell, KeywordNone), flag("", &CondTag{Var: "alwaysFinalRepeat"})},
	`critical strikes penetrate ([0-9]+)% of enemy elemental resistances while affected by zealotry`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalPenetration", Base, Num(c.n(1)), &CondTag{Var: "CriticalStrike"}, &CondTag{Var: "AffectedByZealotry"})}
	}),
	`attack critical strikes ignore enemy monster elemental resistances`: modList{flag("IgnoreElementalResistances", &CondTag{Var: "CriticalStrike"}, &SkillTypeTag{SkillType: SkillTypeAttack})},
	`treats enemy monster chaos resistance values as inverted`:           modList{mod("HitsInvertChaosResChance", Chance, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`([+\-][0-9]+)% to critical strike multiplier if you've shattered an enemy recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritMultiplier", Base, Num(c.n(1)), &CondTag{Var: "ShatteredEnemyRecently"})}
	}),
	`([0-9]+)% chance to gain a flask charge when you deal a critical strike`:             modFn(func(c caps) []*Mod { return []*Mod{mod("FlaskChargeOnCritChance", Base, Num(c.n(1)))} }),
	`gain a flask charge when you deal a critical strike`:                                 modList{mod("FlaskChargeOnCritChance", Base, Num(100))},
	`gain a flask charge when you deal a critical strike while affected by precision`:     modList{mod("FlaskChargeOnCritChance", Base, Num(100), &CondTag{Var: "AffectedByPrecision"})},
	`gain a flask charge when you deal a critical strike while at maximum frenzy charges`: modList{mod("FlaskChargeOnCritChance", Base, Num(100), &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"})},
	`on non-channelling attack, set a life flask with greater than [0-9]+% of maximum charges remaining to ([0-9]+)% for each charge removed this way, that attack gains \+([0-9]+)% to damage over time multiplier`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DotMultiplier", Base, Num(c.n(2)), &StatTag{StatKind: TagPercentStat, Stat: "LifeFlaskCharges", Percent: opt(100 - c.n(1)), Floor: true}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`enemies poisoned by you cannot deal critical strikes`:                        modList{mod("EnemyModifier", List, ModRef{Mod: flag("NeverCrit", &CondTag{Var: "Poisoned"})}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:NeverCrit", &CondTag{Var: "Poisoned"})})},
	`marked enemy cannot deal critical strikes`:                                   modList{mod("EnemyModifier", List, ModRef{Mod: flag("NeverCrit", &CondTag{Var: "Marked"})}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:NeverCrit", &CondTag{Var: "Marked"})})},
	`marked enemy cannot evade attacks`:                                           modList{mod("EnemyModifier", List, ModRef{Mod: flag("CannotEvade", &CondTag{Var: "Marked"})})},
	`hits against you cannot be critical strikes if you've been stunned recently`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("NeverCrit")}, &CondTag{Var: "StunnedRecently"}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:NeverCrit")}, &CondTag{Var: "StunnedRecently"})},
	`nearby enemies cannot deal critical strikes`:                                 modList{mod("EnemyModifier", List, ModRef{Mod: flag("NeverCrit")}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:NeverCrit")})},
	`hits against you are always critical strikes`:                                modList{mod("EnemyModifier", List, ModRef{Mod: flag("AlwaysCrit")}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:AlwaysCrit")})},
	`your hits are always critical strikes`:                                       modList{mod("CritChance", Override, Num(100))},
	`all hits are critical strikes while holding a fishing rod`:                   modList{mod("CritChance", Override, Num(100), &CondTag{Var: "UsingFishing"})},
	`all hits with your next non-channelling attack within ([0-9]+) seconds of taking a critical strike will be critical strikes`: modList{mod("CritChance", Override, Num(100), &CondTag{Var: "BeenCritRecently"}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &SkillTypeTag{SkillType: SkillTypeAttack})},
	`hits have ([0-9]+)% increased critical strike chance against you`:                                                            modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyCritChance", Inc, Num(c.n(1)))} }),
	`stuns from critical strikes have ([0-9]+)% increased duration`:                                                               modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyStunDurationOnCrit", Inc, Num(c.n(1)))} }),
	// Generic Ailments
	`enemies take ([0-9]+)% increased damage for each type of ailment you have inflicted on them`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Scorched"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Brittle"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Sapped"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}), mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"})}
	}),
	`([0-9]+)% chance to deal double damage against enemies for each type of ailment you have inflicted on them`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Scorched"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Brittle"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Sapped"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}), mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"})}
	}),
	// Elemental Ailments
	`([0-9]+)% increased elemental damage with hits and ailments for each type of elemental ailment on enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"}), modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}), modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"}), modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}), modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Scorched"}), modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Brittle"}), modf("ElementalDamage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &CondTag{IsActor: true, Actor: "enemy", Var: "Sapped"})}
	}),
	`your shocks can increase damage taken by up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod { return []*Mod{mod("ShockMax", Override, Num(c.n(1)))} }),
	`\+([0-9]+)% to maximum effect of shock`:                                modFn(func(c caps) []*Mod { return []*Mod{mod("ShockMax", Base, Num(c.n(1)))} }),
	`your ([0-9a-zA-Z]+) damage can ([0-9a-zA-Z]+)`:                         modFn(func(c caps) []*Mod { return []*Mod{flag(firstToUpper(c.s(1)) + "Can" + firstToUpper(c.s(2)))} }),
	`your ([0-9a-zA-Z]+) damage cannot ([0-9a-zA-Z]+)`:                      modFn(func(c caps) []*Mod { return []*Mod{flag(firstToUpper(c.s(1)) + "Cannot" + firstToUpper(c.s(2)))} }),
	`your elemental damage can shock`:                                       modList{flag("ColdCanShock"), flag("FireCanShock")},
	`all y?o?u?r? ?damage can freeze`:                                       modList{flag("PhysicalCanFreeze"), flag("LightningCanFreeze"), flag("FireCanFreeze"), flag("ChaosCanFreeze")},
	`all damage with maces and sceptres inflicts chill`:                     modList{flag("PhysicalCanChill", &CondTag{Var: "UsingMace"}), flag("LightningCanChill", &CondTag{Var: "UsingMace"}), flag("FireCanChill", &CondTag{Var: "UsingMace"}), flag("ChaosCanChill", &CondTag{Var: "UsingMace"})},
	`all damage from lightning strike and frost blades hits can ignite`:     modList{flag("PhysicalCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Strike", "Frost Blades"}, IncludeTransfigured: true}), flag("ColdCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Strike", "Frost Blades"}, IncludeTransfigured: true}), flag("LightningCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Strike", "Frost Blades"}, IncludeTransfigured: true}), flag("ChaosCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Strike", "Frost Blades"}, IncludeTransfigured: true})},
	`all damage from lightning arrow and ice shot hits can ignite`:          modList{flag("PhysicalCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Arrow", "Ice Shot"}, IncludeTransfigured: true}), flag("ColdCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Arrow", "Ice Shot"}, IncludeTransfigured: true}), flag("LightningCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Arrow", "Ice Shot"}, IncludeTransfigured: true}), flag("ChaosCanIgnite", &SkillNameTag{SkillNameList: []string{"Lightning Arrow", "Ice Shot"}, IncludeTransfigured: true})},
	`all damage from shock nova and storm call hits can ignite`:             modList{flag("PhysicalCanIgnite", &SkillNameTag{SkillNameList: []string{"Shock Nova", "Storm Call"}, IncludeTransfigured: true}), flag("ColdCanIgnite", &SkillNameTag{SkillNameList: []string{"Shock Nova", "Storm Call"}, IncludeTransfigured: true}), flag("LightningCanIgnite", &SkillNameTag{SkillNameList: []string{"Shock Nova", "Storm Call"}, IncludeTransfigured: true}), flag("ChaosCanIgnite", &SkillNameTag{SkillNameList: []string{"Shock Nova", "Storm Call"}, IncludeTransfigured: true})},
	`your fire damage can shock but not ignite`:                             modList{flag("FireCanShock"), flag("FireCannotIgnite")},
	`your cold damage can ignite but not freeze or chill`:                   modList{flag("ColdCanIgnite"), flag("ColdCannotFreeze"), flag("ColdCannotChill")},
	`your lightning damage can freeze but not shock`:                        modList{flag("LightningCanFreeze"), flag("LightningCannotShock")},
	`your physical damage can ignite during effect`:                         modList{flag("PhysicalCanIgnite")},
	`chaos damage can ignite, chill and shock`:                              modList{flag("ChaosCanIgnite"), flag("ChaosCanChill"), flag("ChaosCanShock")},
	`you always ignite while burning`:                                       modList{mod("EnemyIgniteChance", Base, Num(100), &CondTag{Var: "Burning"})},
	`critical strikes do not a?l?w?a?y?s?i?n?h?e?r?e?n?t?l?y? freeze`:       modList{flag("CritsDontAlwaysFreeze")},
	`cannot inflict elemental ailments`:                                     modList{flag("CannotIgnite"), flag("CannotChill"), flag("CannotFreeze"), flag("CannotShock"), flag("CannotScorch"), flag("CannotBrittle"), flag("CannotSap")},
	`non-critical strikes cannot inflict ailments`:                          modList{flag("AilmentsOnlyFromCrit")},
	`flameblast and incinerate cannot inflict elemental ailments`:           modList{flag("CannotIgnite", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true}), flag("CannotChill", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true}), flag("CannotFreeze", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true}), flag("CannotShock", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true}), flag("CannotScorch", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true}), flag("CannotBrittle", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true}), flag("CannotSap", &SkillNameTag{SkillNameList: []string{"Flameblast", "Incinerate"}, IncludeTransfigured: true})},
	`you can inflict up to ([0-9]+) ignites on an enemy`:                    modFn(func(c caps) []*Mod { return []*Mod{flag("IgniteCanStack"), mod("IgniteStacks", Override, Num(c.n(1)))} }),
	`you can inflict an additional ignite on [ea][an]c?h? enemy`:            modList{flag("IgniteCanStack"), mod("IgniteStacks", Base, Num(1))},
	`enemies chilled by you take ([0-9]+)% increased burning damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("FireDamageTakenOverTime", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"})}
	}),
	`damaging ailments deal damage ([0-9]+)% faster`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("IgniteBurnFaster", Inc, Num(c.n(1))), mod("BleedFaster", Inc, Num(c.n(1))), mod("PoisonFaster", Inc, Num(c.n(1)))}
	}),
	`damaging ailments you inflict deal damage ([0-9]+)% faster while affected by malevolence`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("IgniteBurnFaster", Inc, Num(c.n(1)), &CondTag{Var: "AffectedByMalevolence"}), mod("BleedFaster", Inc, Num(c.n(1)), &CondTag{Var: "AffectedByMalevolence"}), mod("PoisonFaster", Inc, Num(c.n(1)), &CondTag{Var: "AffectedByMalevolence"})}
	}),
	`([0-9]+)% increased damage with damaging ailments you inflict while you are affected by the same ailment`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordBleed, &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}, &CondTag{Var: "Bleeding"}), modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordIgnite, &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"}, &CondTag{Var: "Ignited"}), modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordPoison, &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"}, &CondTag{Var: "Poisoned"})}
	}),
	`ignited enemies burn ([0-9]+)% faster`: modFn(func(c caps) []*Mod { return []*Mod{mod("IgniteBurnFaster", Inc, Num(c.n(1)))} }),
	// Overrides the base duration of a damaging ailment (the only ailments with a fixed base duration
	// consumed by the calcs; freeze duration is derived from damage and chill/shock durations are not modelled)
	`ignited enemies burn ([0-9]+)% slower`: modFn(func(c caps) []*Mod { return []*Mod{mod("IgniteBurnSlower", Inc, Num(c.n(1)))} }),
	`enemies ignited by an attack burn ([0-9]+)% faster`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("IgniteBurnFaster", Inc, Num(c.n(1)), FlagAttack, KeywordNone)}
	}),
	`ignites you inflict with attacks deal damage ([0-9]+)% faster`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("IgniteBurnFaster", Inc, Num(c.n(1)), FlagAttack, KeywordNone)}
	}),
	`ignites you inflict deal damage ([0-9]+)% faster`: modFn(func(c caps) []*Mod { return []*Mod{mod("IgniteBurnFaster", Inc, Num(c.n(1)))} }),
	`([0-9]+)% chance for ignites inflicted with lightning strike or frost blades to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordIgnite, &SkillNameTag{SkillNameList: []string{"Lightning Strike", "Frost Blades"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% chance for ignites inflicted with lightning arrow or ice shot to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordIgnite, &SkillNameTag{SkillNameList: []string{"Lightning Arrow", "Ice Shot"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% chance for ignites inflicted with shock nova or storm call to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordIgnite, &SkillNameTag{SkillNameList: []string{"Shock Nova", "Storm Call"}, IncludeTransfigured: true})}
	}),
	`enemies ignited by you during f?l?a?s?k? ?effect take ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"})}
	}),
	`enemies ignited by you take chaos damage instead of fire damage from ignite`: modList{flag("IgniteToChaos")},
	`enemies chilled by your hits are shocked`:                                    modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{IsActor: true, Actor: "enemy", Var: "ChilledByYourHits"}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Shocked", &CondTag{Var: "ChilledByYourHits"})})},
	`cannot inflict ignite`:                                                       modList{flag("CannotIgnite")},
	`cannot inflict freeze or chill`:                                              modList{flag("CannotFreeze"), flag("CannotChill")},
	`cannot inflict shock`:                                                        modList{flag("CannotShock")},
	`cannot ignite, chill, freeze or shock`:                                       modList{flag("CannotIgnite"), flag("CannotChill"), flag("CannotFreeze"), flag("CannotShock")},
	`shock enemies as though dealing ([0-9]+)% more damage`:                       modFn(func(c caps) []*Mod { return []*Mod{mod("ShockAsThoughDealing", More, Num(c.n(1)))} }),
	`chill enemies as though dealing ([0-9]+)% more damage`:                       modFn(func(c caps) []*Mod { return []*Mod{mod("ChillAsThoughDealing", More, Num(c.n(1)))} }),
	`inflict non-damaging ailments as though dealing ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ShockAsThoughDealing", More, Num(c.n(1))), mod("ChillAsThoughDealing", More, Num(c.n(1))), mod("FreezeAsThoughDealing", More, Num(c.n(1))), mod("ScorchAsThoughDealing", More, Num(c.n(1))), mod("BrittleAsThoughDealing", More, Num(c.n(1))), mod("SapAsThoughDealing", More, Num(c.n(1)))}
	}),
	`non-damaging elemental ailments you inflict have ([0-9]+)% more effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyShockEffect", More, Num(c.n(1))), mod("EnemyChillEffect", More, Num(c.n(1))), mod("EnemyFreezeEffect", More, Num(c.n(1))), mod("EnemyScorchEffect", More, Num(c.n(1))), mod("EnemyBrittleEffect", More, Num(c.n(1))), mod("EnemySapEffect", More, Num(c.n(1)))}
	}),
	`immun[ei]t?y? to elemental ailments while on consecrated ground if you have at least ([0-9]+) devotion`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ElementalAilmentImmune", &CondTag{Var: "OnConsecratedGround"}, &StatTag{StatKind: TagStatThreshold, Stat: "Devotion", Threshold: opt(c.n(1))})}
	}),
	`freeze enemies as though dealing ([0-9]+)% more damage`: modFn(func(c caps) []*Mod { return []*Mod{mod("FreezeAsThoughDealing", More, Num(c.n(1)))} }),
	`freeze chilled enemies as though dealing ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FreezeAsThoughDealing", More, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"})}
	}),
	`manabond and stormbind freeze enemies as though dealing ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FreezeAsThoughDealing", More, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Manabond", "Stormbind"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% chance to inflict brittle on enemies when you block their damage`:                                    modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyBrittleChance", Base, Num(c.n(1)))} }),
	`([0-9]+)% chance to inflict sap on enemies when you block their damage`:                                        modFn(func(c caps) []*Mod { return []*Mod{mod("EnemySapChance", Base, Num(c.n(1)))} }),
	`([0-9]+)% chance to inflict scorch on enemies when you block their damage`:                                     modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyScorchChance", Base, Num(c.n(1)))} }),
	`scorch enemies in close range when you block`:                                                                  modList{mod("EnemyScorchChance", Base, Num(100))},
	`([0-9]+)% chance to shock attackers for ([0-9]+) seconds on block`:                                             modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]))},
	`shock attackers for ([0-9]+) seconds on block`:                                                                 modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{Var: "BlockedRecently"}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Shocked")}, &CondTag{Var: "BlockedRecently"})},
	`shock nearby enemies for ([0-9]+) seconds when you focus`:                                                      modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{Var: "Focused"}), mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Shocked")}, &CondTag{Var: "Focused"})},
	`shock yourself for ([0-9]+) seconds when you focus`:                                                            modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{Var: "Focused"}), flag("Condition:Shocked", &CondTag{Var: "Focused"})},
	`drops shocked ground while moving, lasting ([0-9]+) seconds`:                                                   modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnShockedGround"})},
	`drops scorched ground while moving, lasting ([0-9]+) seconds`:                                                  modList{mod("ScorchBase", Base, Num(nonDamagingAilmentDefault["Scorch"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnScorchedGround"})},
	`drops brittle ground while moving, lasting ([0-9]+) seconds`:                                                   modList{mod("BrittleBase", Base, Num(nonDamagingAilmentDefault["Brittle"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnBrittleGround"})},
	`drops sapped ground while moving, lasting ([0-9]+) seconds`:                                                    modList{mod("SapBase", Base, Num(nonDamagingAilmentDefault["Sap"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnSappedGround"})},
	`while a unique enemy is in your presence, drops shocked ground while moving, lasting ([0-9]+) seconds`:         modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnShockedGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})},
	`while a unique enemy is in your presence, drops scorched ground while moving, lasting ([0-9]+) seconds`:        modList{mod("ScorchBase", Base, Num(nonDamagingAilmentDefault["Scorch"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnScorchedGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})},
	`while a unique enemy is in your presence, drops brittle ground while moving, lasting ([0-9]+) seconds`:         modList{mod("BrittleBase", Base, Num(nonDamagingAilmentDefault["Brittle"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnBrittleGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})},
	`while a unique enemy is in your presence, drops sapped ground while moving, lasting ([0-9]+) seconds`:          modList{mod("SapBase", Base, Num(nonDamagingAilmentDefault["Sap"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnSappedGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})},
	`while a pinnacle atlas boss is in your presence, drops shocked ground while moving, lasting ([0-9]+) seconds`:  modList{mod("ShockBase", Base, Num(nonDamagingAilmentDefault["Shock"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnShockedGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"})},
	`while a pinnacle atlas boss is in your presence, drops scorched ground while moving, lasting ([0-9]+) seconds`: modList{mod("ScorchBase", Base, Num(nonDamagingAilmentDefault["Scorch"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnScorchedGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"})},
	`while a pinnacle atlas boss is in your presence, drops brittle ground while moving, lasting ([0-9]+) seconds`:  modList{mod("BrittleBase", Base, Num(nonDamagingAilmentDefault["Brittle"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnBrittleGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"})},
	`while a pinnacle atlas boss is in your presence, drops sapped ground while moving, lasting ([0-9]+) seconds`:   modList{mod("SapBase", Base, Num(nonDamagingAilmentDefault["Sap"]), &CondTag{IsActor: true, Actor: "enemy", Var: "OnSappedGround"}, &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"})},
	`\+([0-9]+)% chance to ignite, freeze, shock, and poison cursed enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyIgniteChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}), mod("EnemyFreezeChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}), mod("EnemyShockChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}), mod("PoisonChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`you have scorching conflux, brittle conflux and sapping conflux while your two highest attributes are equal`: modList{mod("EnemyScorchChance", Base, Num(100), &CondTag{Var: "TwoHighestAttributesEqual"}), mod("EnemyBrittleChance", Base, Num(100), &CondTag{Var: "TwoHighestAttributesEqual"}), mod("EnemySapChance", Base, Num(100), &CondTag{Var: "TwoHighestAttributesEqual"}), flag("PhysicalCanScorch", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("LightningCanScorch", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("ColdCanScorch", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("ChaosCanScorch", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("PhysicalCanBrittle", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("LightningCanBrittle", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("FireCanBrittle", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("ChaosCanBrittle", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("PhysicalCanSap", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("ColdCanSap", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("FireCanSap", &CondTag{Var: "TwoHighestAttributesEqual"}), flag("ChaosCanSap", &CondTag{Var: "TwoHighestAttributesEqual"})},
	`all damage from cold snap and creeping frost can sap`:                                                        modList{flag("PhysicalCanSap", &SkillNameTag{SkillNameList: []string{"Cold Snap", "Creeping Frost"}, IncludeTransfigured: true}), flag("ColdCanSap", &SkillNameTag{SkillNameList: []string{"Cold Snap", "Creeping Frost"}, IncludeTransfigured: true}), flag("FireCanSap", &SkillNameTag{SkillNameList: []string{"Cold Snap", "Creeping Frost"}, IncludeTransfigured: true}), flag("ChaosCanSap", &SkillNameTag{SkillNameList: []string{"Cold Snap", "Creeping Frost"}, IncludeTransfigured: true})},
	`always inflict scorch, brittle and sapped with elemental hit and wild strike hits`:                           modList{mod("EnemyScorchChance", Base, Num(100), &SkillNameTag{SkillNameList: []string{"Elemental Hit", "Wild Strike"}, IncludeTransfigured: true}), mod("EnemyBrittleChance", Base, Num(100), &SkillNameTag{SkillNameList: []string{"Elemental Hit", "Wild Strike"}, IncludeTransfigured: true}), mod("EnemySapChance", Base, Num(100), &SkillNameTag{SkillNameList: []string{"Elemental Hit", "Wild Strike"}, IncludeTransfigured: true})},
	`hits with prismatic skills always ?i?n?f?l?i?c?t? ([0-9a-zA-Z]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Enemy"+firstToUpper(c.s(1))+"Chance", Base, Num(100), FlagHit, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRandomElement})}
	}),
	`critical strikes do not [ia][np][fp]l[iy]c?t? non-damaging ailments`: modList{flag("CritsDontAlwaysChill"), flag("CritsDontAlwaysFreeze"), flag("CritsDontAlwaysShock")},
	`critical strikes do not inherently ignite`:                           modList{flag("CritsDontAlwaysIgnite")},
	`always scorch while affected by anger`:                               modList{mod("EnemyScorchChance", Base, Num(100), &CondTag{Var: "AffectedByAnger"})},
	`always inflict brittle while affected by hatred`:                     modList{mod("EnemyBrittleChance", Base, Num(100), &CondTag{Var: "AffectedByHatred"})},
	`always sap while affected by wrath`:                                  modList{mod("EnemySapChance", Base, Num(100), &CondTag{Var: "AffectedByWrath"})},
	`([0-9]+)% chance to sap enemies in chilling areas`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemySapChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "InChillingArea"})}
	}),
	`([0-9]+)% chance for cold snap and creeping frost to sap enemies in chilling areas`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemySapChance", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Cold Snap", "Creeping Frost"}, IncludeTransfigured: true}, &CondTag{IsActor: true, Actor: "enemy", Var: "InChillingArea"})}
	}),
	`drops burning ground while moving, dealing ([0-9]+) fire damage per second for [0-9]+ seconds`: modFn(func(c caps) []*Mod { return []*Mod{mod("DropsBurningGround", Base, Num(c.n(1)))} }),
	`take ([0-9]+) fire damage per second while flame-touched`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireDegen", Base, Num(c.n(1)), &CondTag{Var: "AffectedByApproachingFlames"})}
	}),
	`gain adrenaline when you become flame-touched`:                       modList{flag("Condition:Adrenaline", &CondTag{Var: "AffectedByApproachingFlames"})},
	`lose adrenaline when you cease to be flame-touched`:                  modList{},
	`modifiers to ignite duration on you apply to all elemental ailments`: modList{flag("IgniteDurationAppliesToElementalAilments")},
	`([0-9]+)% increased duration of ailments of types you haven't inflicted recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyFreezeDuration", Inc, Num(c.n(1)), &CondTag{Var: "FrozenEnemyRecently", Neg: true}), mod("EnemyChillDuration", Inc, Num(c.n(1)), &CondTag{Var: "FrozenEnemyRecently", Neg: true}), mod("EnemyIgniteDuration", Inc, Num(c.n(1)), &CondTag{Var: "IgnitedEnemyRecently", Neg: true}), mod("EnemyShockDuration", Inc, Num(c.n(1)), &CondTag{Var: "ShockedEnemyRecently", Neg: true}), mod("EnemyBleedDuration", Inc, Num(c.n(1)), &CondTag{Var: "CausedBleedingRecently", Neg: true}), mod("EnemyPoisonDuration", Inc, Num(c.n(1)), &CondTag{Var: "PoisonedEnemyRecently", Neg: true})}
	}),
	`chance to avoid being shocked applies to all elemental ailments`:            modList{flag("ShockAvoidAppliesToElementalAilments")},
	`modifiers to chance to avoid being shocked apply to all elemental ailments`: modList{flag("ShockAvoidAppliesToElementalAilments")},
	`enemies permanently take ([0-9]+)% increased damage for each second they've ever been frozen by you, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &CondTag{Var: "FrozenByYou"}, &MultiplierTag{Var: "FrozenByYouSeconds", Limit: opt(c.n(2) / c.n(1))})})}
	}),
	`enemies permanently take ([0-9]+)% increased damage for each second they've ever been chilled by you, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &CondTag{Var: "ChilledByYou"}, &MultiplierTag{Var: "ChilledByYouSeconds", Limit: opt(c.n(2) / c.n(1))})})}
	}),
	`modifiers to chance to suppress spell damage also apply to chance to avoid elemental ailments at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionAppliesToAilmentAvoidancePercent", Base, Num(c.n(1))), flag("SpellSuppressionAppliesToAilmentAvoidance")}
	}),
	`modifiers to chance to suppress spell damage also apply to chance to defend with ([0-9]+)% of armour at ([0-9]+)% of their value`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour", Max, Num(c.n(1))), mod("SpellSuppressionAppliesToChanceToDefendWithArmourPercent", Max, Num(c.n(2))), flag("SpellSuppressionAppliesToChanceToDefendWithArmour")}
	}),
	`enemies chilled by your hits have damage taken increased by chill effect`:                modList{flag("ChillEffectIncDamageTaken")},
	`enemies chilled by your hits have cold damage taken increased by chill effect`:           modList{flag("ChillEffectIncColdDamageTaken")},
	`enemies in your chilling areas have cold damage taken increased by chill effect`:         modList{flag("ChillingAreaIncColdDamageTaken", &CondTag{IsActor: true, Actor: "enemy", Var: "InChillingArea"})},
	`left ring slot: your chilling skitterbot's aura applies socketed h?e?x? ?curse instead`:  modList{flag("SkitterbotsCannotChill", &SlotTag{SlotKind: TagSlotNumber, Num: 1})},
	`right ring slot: your shocking skitterbot's aura applies socketed h?e?x? ?curse instead`: modList{flag("SkitterbotsCannotShock", &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`summon skitterbots also summons a scorching skitterbot`:                                  modList{flag("ScorchingSkitterbot")},
	`summoned skitterbots' auras affect you as well as enemies`:                               modList{flag("SkitterbotAffectPlayer")},
	`([0-9]+)% increased effect of non-damaging ailments inflicted by summoned skitterbots`:   modFn(func(c caps) []*Mod { return []*Mod{mod("SkitterbotAilmentEffect", Inc, Num(c.n(1)))} }),
	// Bleed
	`melee attacks cause bleeding`:                       modList{modf("BleedChance", Base, Num(100), FlagMelee, KeywordNone)},
	`attacks cause bleeding when hitting cursed enemies`: modList{modf("BleedChance", Base, Num(100), FlagAttack, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})},
	`melee critical strikes cause bleeding`:              modList{modf("BleedChance", Base, Num(100), FlagMelee, KeywordNone, &CondTag{Var: "CriticalStrike"})},
	`causes bleeding on melee critical strike`:           modList{modf("BleedChance", Base, Num(100), FlagMelee, KeywordNone, &CondTag{Var: "CriticalStrike"})},
	`melee critical strikes have ([0-9]+)% chance to cause bleeding`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("BleedChance", Base, Num(c.n(1)), FlagMelee, KeywordNone, &CondTag{Var: "CriticalStrike"})}
	}),
	`attacks always inflict bleeding while you have cat's stealth`:        modList{modf("BleedChance", Base, Num(100), FlagAttack, KeywordNone, &CondTag{Var: "AffectedByCat'sStealth"})},
	`you have crimson dance while you have cat's stealth`:                 modList{mod("Keystone", List, Str("Crimson Dance"), &CondTag{Var: "AffectedByCat'sStealth"})},
	`you have crimson dance if you have dealt a critical strike recently`: modList{mod("Keystone", List, Str("Crimson Dance"), &CondTag{Var: "CritRecently"})},
	`bleeding you inflict deals damage ([0-9]+)% faster`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("BleedFaster", Inc, Num(c.n(1)))} }),
	`bleeding you inflict on non-bleeding enemies deals ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordBleed, &CondTag{Var: "SingleBleed"})}
	}),
	`([0-9]+)% chance for bleeding inflicted with this weapon to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordBleed, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`([0-9]+)% chance for bleeding inflicted with cobra lash or venom gyre to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordBleed, &SkillNameTag{SkillNameList: []string{"Cobra Lash", "Venom Gyre"}})}
	}),
	`bleeding you inflict deals damage ([0-9]+)% faster per frenzy charge`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BleedFaster", Inc, Num(c.n(1)), &MultiplierTag{Var: "FrenzyCharge"})}
	}),
	`rain of arrows and toxic rain deal ([0-9]+)% more damage with bleeding`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(1)), FlagNone, KeywordBleed, &SkillNameTag{SkillNameList: []string{"Rain of Arrows", "Toxic Rain"}, IncludeTransfigured: true})}
	}),
	// Impale and Bleed
	`([0-9]+)% increased effect of impales inflicted by hits that also inflict bleeding`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ImpaleEffectOnBleed", Inc, Num(c.n(1)), FlagNone, KeywordHit)}
	}),
	`([0-9]+)% chance for blade vortex and blade blast to impale enemies on hit`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ImpaleChance", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Blade Vortex", "Blade Blast"}, IncludeTransfigured: true})}
	}),
	`critical strikes with spells inflict impale`:                                                      modList{modf("ImpaleChance", Base, Num(100), FlagSpell, KeywordNone, &CondTag{Var: "CriticalStrike"})},
	`([0-9]+)% chance on hitting an enemy for all impales on that enemy to last for an additional hit`: modFn(func(c caps) []*Mod { return []*Mod{mod("ImpaleAdditionalDurationChance", Base, Num(c.n(1)))} }),
	`projectiles gain impale effect as they travel farther, causing impales they inflict to have up to ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ImpaleEffect", Inc, Num(c.n(1)), FlagProjectile, KeywordNone, &DistanceRampTag{Ramp: Pairs{{35, 0}, {70, 1}}})}
	}),
	// Poison and Bleed
	`([0-9]+)% increased damage with bleeding inflicted on poisoned enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordBleed, &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"})}
	}),
	// Poison
	`y?o?u?r? ?fire damage can poison`:                                      modList{flag("FireCanPoison")},
	`y?o?u?r? ?cold damage can poison`:                                      modList{flag("ColdCanPoison")},
	`y?o?u?r? ?lightning damage can poison`:                                 modList{flag("LightningCanPoison")},
	`all damage from hits can poison`:                                       modList{flag("FireCanPoison"), flag("ColdCanPoison"), flag("LightningCanPoison")},
	`all damage can poison`:                                                 modList{flag("FireCanPoison"), flag("ColdCanPoison"), flag("LightningCanPoison")},
	`all damage with triggered spells can poison`:                           modList{flagf("FireCanPoison", FlagNone, KeywordSpell, &SkillTypeTag{SkillType: SkillTypeTriggered}), flagf("ColdCanPoison", FlagNone, KeywordSpell, &SkillTypeTag{SkillType: SkillTypeTriggered}), flagf("LightningCanPoison", FlagNone, KeywordSpell, &SkillTypeTag{SkillType: SkillTypeTriggered})},
	`all damage from hits with this weapon can poison`:                      modList{flag("FireCanPoison", &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), flag("ColdCanPoison", &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), flag("LightningCanPoison", &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})},
	`all damage inflicts poison while affected by glorious madness`:         modList{mod("PoisonChance", Base, Num(100), &CondTag{Var: "AffectedByGloriousMadness"}), flag("FireCanPoison", &CondTag{Var: "AffectedByGloriousMadness"}), flag("ColdCanPoison", &CondTag{Var: "AffectedByGloriousMadness"}), flag("LightningCanPoison", &CondTag{Var: "AffectedByGloriousMadness"})},
	`all damage from blast rain and artillery ballista hits can poison`:     modList{flag("FireCanPoison", &SkillNameTag{SkillNameList: []string{"Blast Rain", "Artillery Ballista"}}), flag("ColdCanPoison", &SkillNameTag{SkillNameList: []string{"Blast Rain", "Artillery Ballista"}}), flag("LightningCanPoison", &SkillNameTag{SkillNameList: []string{"Blast Rain", "Artillery Ballista"}})},
	`all damage from hits with freezing pulse and eye of winter can poison`: modList{flag("FireCanPoison", &SkillNameTag{SkillNameList: []string{"Freezing Pulse", "Eye of Winter"}, IncludeTransfigured: true}), flag("ColdCanPoison", &SkillNameTag{SkillNameList: []string{"Freezing Pulse", "Eye of Winter"}, IncludeTransfigured: true}), flag("LightningCanPoison", &SkillNameTag{SkillNameList: []string{"Freezing Pulse", "Eye of Winter"}, IncludeTransfigured: true})},
	`your chaos damage poisons enemies`:                                     modList{mod("ChaosPoisonChance", Base, Num(100))},
	`your chaos damage has ([0-9]+)% chance to poison enemies`:              modFn(func(c caps) []*Mod { return []*Mod{mod("ChaosPoisonChance", Base, Num(c.n(1)))} }),
	`melee attacks poison on hit`:                                           modList{modf("PoisonChance", Base, Num(100), FlagMelee, KeywordNone)},
	`triggered spells poison on hit`:                                        modList{modf("PoisonChance", Base, Num(100), FlagNone, KeywordSpell, &SkillTypeTag{SkillType: SkillTypeTriggered})},
	`melee critical strikes have ([0-9]+)% chance to poison the enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PoisonChance", Base, Num(c.n(1)), FlagMelee, KeywordNone, &CondTag{Var: "CriticalStrike"})}
	}),
	`critical strikes with daggers have a ([0-9]+)% chance to poison the enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PoisonChance", Base, Num(c.n(1)), FlagDagger, KeywordNone, &CondTag{Var: "CriticalStrike"})}
	}),
	`critical strikes with daggers poison the enemy`:                 modList{modf("PoisonChance", Base, Num(100), FlagDagger, KeywordNone, &CondTag{Var: "CriticalStrike"})},
	`poison cursed enemies on hit`:                                   modList{mod("PoisonChance", Base, Num(100), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})},
	`always poison on hit against cursed enemies`:                    modList{mod("PoisonChance", Base, Num(100), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})},
	`wh[ie][ln]e? at maximum frenzy charges, attacks poison enemies`: modList{modf("PoisonChance", Base, Num(100), FlagAttack, KeywordNone, &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"})},
	`traps and mines have a ([0-9]+)% chance to poison on hit`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PoisonChance", Base, Num(c.n(1)), FlagNone, KeywordTrap|KeywordMine)}
	}),
	`poisons you inflict deal damage ([0-9]+)% faster`: modFn(func(c caps) []*Mod { return []*Mod{mod("PoisonFaster", Inc, Num(c.n(1)))} }),
	`([0-9]+)% chance for poisons inflicted with this weapon to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordPoison, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`([0-9]+)% chance for poisons inflicted with blast rain or artillery balls?i?s?ta to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordPoison, &SkillNameTag{SkillNameList: []string{"Blast Rain", "Artillery Ballista"}})}
	}),
	`([0-9]+)% chance for poisons inflicted with freezing pulse and eye of winter to deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", More, Num(c.n(2)*c.n(1)/100), FlagNone, KeywordPoison, &SkillNameTag{SkillNameList: []string{"Freezing Pulse", "Eye of Winter"}, IncludeTransfigured: true})}
	}),

	`poisons you inflict on non-poisoned enemies deal ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordPoison, &CondTag{Var: "NonPoisonedOnly"})}
	}),
	`poisons inflicted by sunder or ground slam on non-poisoned enemies deal ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordPoison, &CondTag{Var: "NonPoisonedOnly"}, &SkillNameTag{SkillNameList: []string{"Sunder", "Ground Slam"}, IncludeTransfigured: true})}
	}),
	`poisons on you expire ([0-9]+)% slower`:                                                      modFn(func(c caps) []*Mod { return []*Mod{mod("SelfPoisonDebuffExpirationRate", Base, Num(-c.n(1)))} }),
	`([0-9]+)% chance to inflict an additional poison on the same target when you inflict poison`: modFn(func(c caps) []*Mod { return []*Mod{mod("AdditionalPoisonChance", Base, Num(c.n(1)))} }),
	`inflict ([0-9]+) additional poisons? on the same target when you inflict poisons? with this weapon`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AdditionalPoisonStacks", Base, Num(c.n(1)), &CondTag{Var: "{Hand}Attack"})}
	}),
	`cannot poison enemies with at least ([0-9]+) poisons? on them`:                         modFn(func(c caps) []*Mod { return []*Mod{mod("PoisonStackLimit", Min, Num(c.n(1)))} }),
	`cannot inflict multiple poisons in the same hit`:                                       modList{flag("CannotMultiplePoison")},
	`wither on hit with this weapon against enemies with at least ([0-9]+) poisons on them`: modList{flag("Condition:CanWither")},
	// Suppression
	`y?o?u?r? ?chance to suppress spell damage is lucky`:   modList{flag("SpellSuppressionChanceIsLucky")},
	`y?o?u?r? ?chance to suppress spell damage is unlucky`: modList{flag("SpellSuppressionChanceIsUnlucky")},
	`prevent \+([0-9]+)% of suppressed spell damage`:       modFn(func(c caps) []*Mod { return []*Mod{mod("SpellSuppressionEffect", Base, Num(c.n(1)))} }),
	`prevent \+([0-9]+)% of suppressed spell damage per hit suppressed recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionEffect", Base, Num(c.n(1)), &MultiplierTag{Var: "HitsSuppressedRecently"})}
	}),
	`prevent \+([0-9]+)% of suppressed spell damage if you have not suppressed spell damage recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionEffect", Base, Num(c.n(1)), &CondTag{Var: "SuppressedRecently", Neg: true})}
	}),
	`inflict fire, cold and lightning exposure on enemies when you suppress their spell damage`: modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "SuppressedRecently"}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "SuppressedRecently"}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "SuppressedRecently"})},
	`critical strike chance is increased by chance to suppress spell damage`:                    modList{flag("CritChanceIncreasedBySpellSuppressChance")},
	`you take ([0-9]+)% reduced extra damage from suppressed critical strikes`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("ReduceSuppressedCritExtraDamage", Base, Num(c.n(1)))} }),
	`\+([0-9]+)% chance to suppress spell damage if your e?q?u?i?p?p?e?d? ?boots, helmet and gloves have evasion`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionChance", Base, Num(c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnBoots", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnHelmet", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnGloves", Threshold: opt(1)})}
	}),
	`evasion rating is doubled against projectile attacks`: modList{mod("ProjectileEvasion", More, Num(100))},
	`evasion rating is doubled against melee attacks`:      modList{mod("MeleeEvasion", More, Num(100))},
	`\+([0-9]+)% chance to suppress spell damage for each dagger you're wielding`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellSuppressionChance", Base, Num(c.n(1)), &CondTag{Var: "UsingDagger"}), mod("SpellSuppressionChance", Base, Num(c.n(1)), &CondTag{Var: "DualWieldingDaggers"})}
	}),
	// Buffs/debuffs
	`phasing`:                              modList{flag("Condition:Phasing")},
	`onslaught`:                            modList{flag("Condition:Onslaught")},
	`rampage`:                              modList{flag("Condition:Rampage")},
	`soul eater`:                           modList{flag("Condition:CanHaveSoulEater")},
	`unholy might`:                         modList{flag("Condition:UnholyMight"), flag("Condition:CanWither")},
	`chaotic might`:                        modList{flag("Condition:ChaoticMight")},
	`elusive`:                              modList{flag("Condition:CanBeElusive")},
	`adrenaline`:                           modList{flag("Condition:Adrenaline")},
	`arcane surge`:                         modList{flag("Condition:ArcaneSurge")},
	`your aura buffs do not affect allies`: modList{flag("SelfAurasCannotAffectAllies")},
	`your curses have ([0-9]+)% increased effect if ([0-9]+)% of curse duration expired`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CurseEffect", Inc, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "CurseExpired", Threshold: opt(c.n(2))}, &SkillTypeTag{SkillType: SkillTypeHex})}
	}),
	`non-aura hexes expire upon reaching ([0-9]+)% of base effect non-aura hexes gain ([0-9]+)% increased effect per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CurseEffect", Inc, Num(c.n(2)), &MultiplierTag{Actor: "enemy", Var: "CurseDurationExpired", Limit: opt(c.n(1)), LimitTotal: true}, &SkillTypeTag{SkillType: SkillTypeAura, Neg: true}, &SkillTypeTag{SkillType: SkillTypeHex})}
	}),
	`enemies cursed by you have malediction if ([0-9]+)% of curse duration expired`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: flag("HasMalediction", &MultiplierTag{IsThreshold: true, Var: "CurseExpired", Threshold: opt(c.n(1))}, &CondTag{IsActor: true, Var: "Cursed"})})}
	}),
	`enemies cursed by you are hindered if ([0-9]+)% of curse duration expired`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Hindered", &MultiplierTag{IsThreshold: true, Var: "CurseExpired", Threshold: opt(c.n(1))}, &CondTag{IsActor: true, Var: "Cursed"})})}
	}),
	`excommunicate enemies on melee hit for ([0-9]+) seconds`:        modList{flag("Condition:CanExcommunicate")},
	`auras from your skills can only affect you`:                     modList{flag("SelfAurasOnlyAffectYou")},
	`auras from your skills which affect allies also affect enemies`: modList{flag("AurasAffectEnemies")},
	`aura buffs from skills have ([0-9]+)% increased effect on you for each herald affecting you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillAuraEffectOnSelf", Inc, Num(c.n(1)), &MultiplierTag{Var: "Herald"})}
	}),
	`aura buffs from skills have ([0-9]+)% increased effect on you for each herald affecting you, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillAuraEffectOnSelf", Inc, Num(c.n(1)), &MultiplierTag{Var: "Herald", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "PurposefulHarbinger"})}
	}),
	`auras from your skills have ([0-9]+)% increased effect on you for each herald affecting you, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillAuraEffectOnSelf", Inc, Num(c.n(1)), &MultiplierTag{Var: "Herald", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "PurposefulHarbinger"})}
	}),
	`([0-9]+)% increased area of effect per power charge, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &MultiplierTag{Var: "PowerCharge", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "VastPower"})}
	}),
	`([0-9]+)% increased area of effect per second you've been stationary, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &MultiplierTag{Var: "StationarySeconds", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "ExpansiveMight", LimitTotal: true})}
	}),
	`([0-9]+)% increased chaos damage per ([0-9]+) maximum mana, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosDamage", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "Mana", Div: opt(c.n(2)), GlobalLimit: opt(c.n(3)), GlobalLimitKey: "DarkIdeation"})}
	}),
	`minions have \+([0-9]+)% to damage over time multiplier per ghastly eye jewel affecting you, up to a maximum of \+([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("DotMultiplier", Base, Num(c.n(1)), &MultiplierTag{Var: "GhastlyEyeJewel", Actor: "parent", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "AmanamuGaze"})})}
	}),
	`([0-9]+)% increased effect of arcane surge on you per hypnotic eye jewel affecting you, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ArcaneSurgeEffect", Inc, Num(c.n(1)), &MultiplierTag{Var: "HypnoticEyeJewel", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "KurgalGaze"})}
	}),
	`([0-9]+)% increased main hand critical strike chance per murderous eye jewel affecting you, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritChance", Inc, Num(c.n(1)), &MultiplierTag{Var: "MurderousEyeJewel", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "TecrodGazeMainHand"}, &CondTag{Var: "MainHandAttack"})}
	}),
	`\+([0-9]+)% to off hand critical strike multiplier per murderous eye jewel affecting you, up to a maximum of \+([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritMultiplier", Base, Num(c.n(1)), &MultiplierTag{Var: "MurderousEyeJewel", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "TecrodGazeOffHand"}, &CondTag{Var: "OffHandAttack"})}
	}),
	`nearby allies' damage with hits is lucky`:                                           modList{mod("ExtraAura", List, ModRef{Mod: flag("LuckyHits"), OnlyAllies: true})},
	`your damage with hits is lucky`:                                                     modList{flag("LuckyHits")},
	`your damage with hits is lucky while on low life`:                                   modList{flag("LuckyHits", &CondTag{Var: "LowLife"})},
	`damage with hits is unlucky`:                                                        modList{flag("UnluckyHits")},
	`chaos damage with hits is lucky`:                                                    modList{flag("ChaosLuckyHits")},
	`lightning damage with hits is lucky if you[' ]h?a?ve blocked spell damage recently`: modList{flag("LightningLuckHits", &CondTag{Var: "BlockedSpellRecently"})},
	`cold damage with hits is lucky if you[' ]h?a?ve suppressed spell damage recently`:   modList{flag("ColdLuckyHits", &CondTag{Var: "SuppressedRecently"})},
	`fire damage with hits is lucky if you[' ]h?a?ve blocked an attack recently`:         modList{flag("FireLuckyHits", &CondTag{Var: "BlockedAttackRecently"})},
	`elemental damage with hits is lucky while you are shocked`:                          modList{flag("ElementalLuckHits", &CondTag{Var: "Shocked"})},
	`your lucky or unlucky effects are instead unexciting`:                               modList{flag("Unexciting")},
	`allies' aura buffs do not affect you`:                                               modList{flag("AlliesAurasCannotAffectSelf")},
	`([0-9]+)% increased effect of non-curse auras from your skills on enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DebuffEffect", Inc, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeAppliesCurse, Neg: true}), mod("AuraEffect", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Death Aura"})}
	}),
	`enemies can have 1 additional curse`:                              modList{mod("EnemyCurseLimit", Base, Num(1))},
	`you can apply an additional curse`:                                modList{mod("EnemyCurseLimit", Base, Num(1))},
	`you can apply an additional curse during effect`:                  modList{mod("EnemyCurseLimit", Base, Num(1))},
	`you can apply an additional curse while affected by malevolence`:  modList{mod("EnemyCurseLimit", Base, Num(1), &CondTag{Var: "AffectedByMalevolence"})},
	`you can apply an additional curse while at maximum power charges`: modList{mod("EnemyCurseLimit", Base, Num(1), &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", ThresholdStat: "PowerChargesMax"})},
	`you can apply an additional curse if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyCurseLimit", Base, Num(1), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "Item", Threshold: opt(c.n(1))})}
	}),
	`you can apply one fewer curse`: modList{mod("EnemyCurseLimit", Base, Num(-1))},
	`curses on enemies in your chilling areas have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CurseEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "InChillingArea"})}
	}),
	`hexes you inflict have their effect increased by twice their doom instead`: modList{mod("DoomEffect", More, Num(100))},
	`nearby enemies have an additional ([0-9]+)% chance to receive a critical strike`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("SelfExtraCritChance", Base, Num(c.n(1)))})}
	}),
	`nearby enemies have (-[0-9]+)% to all resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("ElementalResist", Base, Num(c.n(1)))}), mod("EnemyModifier", List, ModRef{Mod: mod("ChaosResist", Base, Num(c.n(1)))})}
	}),
	`enemies ignited or chilled by you have (-[0-9]+)% to elemental resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("ElementalResist", Base, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Ignited", "Chilled"}})}
	}),
	`reserves ([0-9]+)% of nearby enemy monsters' life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("LifeReservationPercent", Base, Num(c.n(1)))})}
	}),
	`nearby enemy monsters have at least ([0-9]+)% of life reserved`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("LifeReservationPercent", Base, Num(c.n(1)))})}
	}),
	`your hits inflict decay, dealing ([0-9]+) chaos damage per second for [0-9]+ seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "decay", Value: Num(c.n(1)), Merge: "MAX"})}
	}),
	`inflict decay on enemies you curse with hex or mark skills, dealing ([0-9]+) chaos damage per second for [0-9]+ seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "decay", Value: Num(c.n(1)), Merge: "MAX"}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`inflict decay on enemies you curse with hex skills, dealing ([0-9]+) chaos damage per second for [0-9]+ seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "decay", Value: Num(c.n(1)), Merge: "MAX"}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`temporal chains has ([0-9]+)% reduced effect on you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CurseEffectOnSelf", Inc, Num(-c.n(1)), &SkillNameTag{SkillName: "Temporal Chains"})}
	}),
	`unaffected by temporal chains`:        modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Temporal Chains"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`targets are unaffected by your hexes`: modList{mod("CurseEffect", More, Num(-100), &SkillTypeTag{SkillType: SkillTypeHex})},
	`([+\-][0-9.]+) seconds to cat's stealth duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PrimaryDuration", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Aspect of the Cat"})}
	}),
	`([+\-][0-9.]+) seconds to cat's agility duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SecondaryDuration", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Aspect of the Cat"})}
	}),
	`([+\-][0-9.]+) seconds to avian's might duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PrimaryDuration", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Aspect of the Avian"})}
	}),
	`([+\-][0-9.]+) seconds to avian's flight duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SecondaryDuration", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Aspect of the Avian"})}
	}),
	`aspect of the spider can inflict spider's web on enemies an additional time`:       modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("Multiplier:SpiderWebApplyStackMax", Base, Num(1))}, &SkillNameTag{SkillName: "Aspect of the Spider"})},
	`aspect of the avian also grants avian's might and avian's flight to nearby allies`: modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("BuffAppliesToAllies")}, &SkillNameTag{SkillName: "Aspect of the Avian"})},
	`marked enemy takes ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"})}
	}),
	`marked enemy has ([0-9]+)% reduced accuracy rating`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("Accuracy", Inc, Num(-c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"})}
	}),
	`you are cursed with level ([0-9]+) ([^0-9]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraCurse", List, SkillRef{SkillID: gemIdOrNil(c.s(2)), Level: opt(c.n(1)), ApplyToPlayer: true})}
	}),
	`you are cursed with ([^0-9]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraCurse", List, SkillRef{SkillID: gemIdOrNil(c.s(1)), Level: opt(1), ApplyToPlayer: true})}
	}),
	`you count as on low life while you are cursed with vulnerability`:  modList{flag("Condition:LowLife", &CondTag{Var: "AffectedByVulnerability"})},
	`you count as on full life while you are cursed with vulnerability`: modList{flag("Condition:FullLife", &CondTag{Var: "AffectedByVulnerability"})},
	`if you consumed a corpse recently, you and nearby allies regenerate ([0-9]+)% of life per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(c.n(1)))}, &CondTag{Var: "ConsumedCorpseRecently"})}
	}),
	`if you have blocked recently, you and nearby allies regenerate ([0-9]+)% of life per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(c.n(1)))}, &CondTag{Var: "BlockedRecently"})}
	}),
	`you are at maximum chance to block attack damage if you have not blocked recently`: modList{flag("MaxBlockIfNotBlockedRecently", &CondTag{Var: "BlockedRecently", Neg: true})},
	`you are at maximum chance to block spell damage if you have not blocked recently`:  modList{flag("MaxSpellBlockIfNotBlockedRecently", &CondTag{Var: "BlockedRecently", Neg: true})},
	`\+([0-9]+)% chance to block attack damage if you have not blocked recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BlockChance", Base, Num(c.n(1)), &CondTag{Var: "BlockedRecently", Neg: true})}
	}),
	`\+([0-9]+)% chance to block spell damage if you have not blocked recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SpellBlockChance", Base, Num(c.n(1)), &CondTag{Var: "BlockedRecently", Neg: true})}
	}),
	`y?o?u?r? ?chance to block is lucky`:                                                  modList{flag("BlockChanceIsLucky"), flag("ProjectileBlockChanceIsLucky"), flag("SpellBlockChanceIsLucky"), flag("SpellProjectileBlockChanceIsLucky")},
	`y?o?u?r? ?chance to block attack damage is lucky`:                                    modList{flag("BlockChanceIsLucky"), flag("ProjectileBlockChanceIsLucky")},
	`y?o?u?r? ?chance to block attack damage is unlucky`:                                  modList{flag("BlockChanceIsUnlucky"), flag("ProjectileBlockChanceIsUnlucky")},
	`y?o?u?r? ?chance to block is unlucky`:                                                modList{flag("BlockChanceIsUnlucky"), flag("ProjectileBlockChanceIsUnlucky"), flag("SpellBlockChanceIsUnlucky"), flag("SpellProjectileBlockChanceIsUnlucky")},
	`y?o?u?r? ?chance to block spell damage is lucky`:                                     modList{flag("SpellBlockChanceIsLucky"), flag("SpellProjectileBlockChanceIsLucky")},
	`y?o?u?r? ?chance to block spell damage is unlucky`:                                   modList{flag("SpellBlockChanceIsUnlucky"), flag("SpellProjectileBlockChanceIsUnlucky")},
	`your lucky or unlucky effects use the best or worst from three rolls instead of two`: modList{flag("ExtremeLuck")},
	`chance to block attack or spell damage is lucky if you've blocked recently`:          modList{flag("BlockChanceIsLucky", &CondTag{Var: "BlockedRecently"}), flag("ProjectileBlockChanceIsLucky", &CondTag{Var: "BlockedRecently"}), flag("SpellBlockChanceIsLucky", &CondTag{Var: "BlockedRecently"}), flag("SpellProjectileBlockChanceIsLucky", &CondTag{Var: "BlockedRecently"})},
	`([0-9.]+)% of evasion rating is regenerated as life per second while focus?sed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Evasion", Percent: opt(c.n(1))}, &CondTag{Var: "Focused"})}
	}),
	`nearby allies have ([0-9]+)% increased defences per ([0-9]+) strength you have`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("Defences", Inc, Num(c.n(1))), OnlyAllies: true}, &StatTag{StatKind: TagPerStat, Stat: "Str", Div: opt(c.n(2))})}
	}),
	`nearby allies have \+([0-9]+)% to critical strike multiplier per ([0-9]+) dexterity you have`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("CritMultiplier", Base, Num(c.n(1))), OnlyAllies: true}, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(2))})}
	}),
	`nearby allies have ([0-9]+)% increased cast speed per ([0-9]+) intelligence you have`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: modf("Speed", Inc, Num(c.n(1)), FlagCast, KeywordNone), OnlyAllies: true}, &StatTag{StatKind: TagPerStat, Stat: "Int", Div: opt(c.n(2))})}
	}),
	`quicksilver flasks you use also apply to nearby allies`:                  modList{flag("QuickSilverAppliesToAllies")},
	`you gain divinity for [0-9]+ seconds on reaching maximum divine charges`: modList{mod("ElementalDamage", More, Num(75), &CondTag{Var: "Divinity"}), mod("ElementalDamageTaken", More, Num(-25), &CondTag{Var: "Divinity"})},
	`your nearby party members maximum endurance charges is equal to yours`:   modList{flag("PartyMemberMaximumEnduranceChargesEqualToYours")},
	`your maximum endurance charges is equal to your maximum frenzy charges`:  modList{flag("MaximumEnduranceChargesIsMaximumFrenzyCharges")},
	`your maximum frenzy charges is equal to your maximum power charges`:      modList{flag("MaximumFrenzyChargesIsMaximumPowerCharges")},
	`your curse limit is equal to your maximum power charges`:                 modList{flag("CurseLimitIsMaximumPowerCharges")},
	`consecrated ground you create grants ([0-9]+)% increased accuracy rating to you and allies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("ConsecratedGroundAlsoAccuracy", Inc, Num(c.n(1)), &CondTag{Var: "OnConsecratedGround"})})}
	}),
	`consecrated ground created during effect applies ([0-9]+)% increased damage taken to enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTakenConsecratedGround", Inc, Num(c.n(1)), &CondTag{Var: "OnConsecratedGround"})}, &CondTag{Var: "UsingFlask"})}
	}),
	`consecrated ground you create while affected by zealotry causes enemies to take ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTakenConsecratedGround", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "OnConsecratedGround"}, &CondTag{Var: "AffectedByZealotry"})}
	}),
	`if you've warcried recently, you and nearby allies have ([0-9]+)% increased attack, cast and movement speed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("Speed", Inc, Num(c.n(1)))}, &CondTag{Var: "UsedWarcryRecently"}), mod("ExtraAura", List, ModRef{Mod: mod("MovementSpeed", Inc, Num(c.n(1)))}, &CondTag{Var: "UsedWarcryRecently"})}
	}),
	`([0-9]+)% increased movement speed while on full life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MovementSpeed", Inc, Num(c.n(1)), &CondTag{Var: "FullLife"})}
	}),
	`when you warcry, you and nearby allies gain onslaught for 4 seconds`: modList{mod("ExtraAura", List, ModRef{Mod: flag("Onslaught")}, &CondTag{Var: "UsedWarcryRecently"})},
	`warcries grant arcane surge to you and allies, with ([0-9]+)% increased effect per ([0-9]+) power, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: flag("Condition:ArcaneSurge")}, &CondTag{Var: "UsedWarcryRecently"}), mod("ArcaneSurgeEffect", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "WarcryPower", Div: opt(c.n(2)), GlobalLimit: opt(c.n(3)), GlobalLimitKey: "Brinerot Flag"}, &CondTag{Var: "UsedWarcryRecently"})}
	}),
	`gain arcane surge after spending a total of ([0-9]+) mana`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: flag("Condition:ArcaneSurge")}, &MultiplierTag{IsThreshold: true, Var: "ManaSpentRecently", Threshold: opt(c.n(1))})}
	}),
	`gain arcane surge after spending a total of ([0-9]+) life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: flag("Condition:ArcaneSurge")}, &MultiplierTag{IsThreshold: true, Var: "LifeSpentRecently", Threshold: opt(c.n(1))})}
	}),
	`gain onslaught for ([0-9]+) seconds on hit while at maximum frenzy charges`: modList{flag("Onslaught", &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"}, &CondTag{Var: "HitRecently"})},
	`enemies in your chilling areas take ([0-9]+)% increased lightning damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("LightningDamageTaken", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "InChillingArea"})}
	}),
	`warcries count as having ([0-9]+) additional nearby enemies`: modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:WarcryNearbyEnemies", Base, Num(c.n(1)))} }),
	`enemies taunted by your warcries take ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &CondTag{Var: "Taunted"})}, &CondTag{Var: "UsedWarcryRecently"})}
	}),
	`warcries have minimum of ([0-9]+) power`:                                                                              modList{flag("CryWolfMinimumPower")},
	`warcries have infinite power`:                                                                                         modList{flag("WarcryInfinitePower")},
	`your warcries do not grant buffs or charges to you`:                                                                   modList{flag("CannotGainWarcryBuffs")},
	`([0-9]+)% chance to inflict corrosion on hit with attacks`:                                                            modList{flag("Condition:CanCorrode")},
	`([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit`:                                                     modList{flag("Condition:CanWither")},
	`melee weapon hits inflict ([0-9]+) withered debuffs for ([0-9]+) seconds`:                                             modList{flag("Condition:CanWither")},
	`([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit with this weapon`:                                    modList{flag("Condition:CanWither")},
	`([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit against cursed enemies`:                              modList{flag("Condition:CanWither", &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})},
	`([0-9]+)% chance to inflict withered for two seconds on hit if there are ([0-9]+) or fewer withered debuffs on enemy`: modList{flag("Condition:CanWither")},
	`inflict withered for ([0-9]+) seconds on hit with this weapon`:                                                        modList{flag("Condition:CanWither")},
	`chaos skills inflict up to ([0-9]+) withered debuffs on hit for ([0-9]+) seconds`:                                     modList{flag("Condition:CanWither")},
	`minions have ([0-9]+)% chance to inflict withered on hit`:                                                             modList{mod("MinionModifier", List, ModRef{Mod: flag("Condition:CanWither")})},
	`enemies take ([0-9]+)% increased elemental damage from your hits for each withered you have inflicted on them`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("ElementalDamageTaken", Inc, Num(c.n(1)), &MultiplierTag{Var: "WitheredStack", Limit: opt(15)})})}
	}),
	`your hits cannot penetrate or ignore elemental resistances`:                             modList{flag("CannotElePenIgnore")},
	`nearby enemies have malediction`:                                                        modList{mod("EnemyModifier", List, ModRef{Mod: flag("HasMalediction")})},
	`gain shaper's presence for 10 seconds when you kill a rare or unique enemy`:             modList{mod("ExtraAura", List, ModRef{Mod: flag("HasShapersPresence")}, &CondTag{Var: "KilledUniqueEnemy"})},
	`gain maddening presence for 10 seconds when you kill a rare or unique enemy`:            modList{mod("EnemyModifier", List, ModRef{Mod: flag("HasMaddeningPresence")}, &CondTag{Var: "KilledUniqueEnemy"})},
	`elemental damage you deal with hits is resisted by lowest elemental resistance instead`: modList{flag("ElementalDamageUsesLowestResistance")},
	`you take ([0-9]+) chaos damage per second for 3 seconds on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosDegen", Base, Num(c.n(1)), &CondTag{Var: "KilledLast3Seconds"})}
	}),
	`regenerate ([0-9]+) life per second for each ([0-9]+)% uncapped fire resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegen", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "FireResistTotal", Div: opt(1 / c.n(2))})}
	}),
	`regenerate ([0-9]+) life over 1 second for each spell you cast`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegen", Base, Num(c.n(1)), &CondTag{Var: "CastLast1Seconds"})}
	}),
	`and nearby allies regenerate ([0-9]+) life per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("LifeRegen", Base, Num(c.n(1)))}, &CondTag{Var: "KilledPoisonedLast2Seconds"})}
	}),
	`([0-9]+)% increased life regeneration rate`: modFn(func(c caps) []*Mod { return []*Mod{mod("LifeRegen", Inc, Num(c.n(1)))} }),
	`every ([0-9]+) seconds, regenerate life equal to ([0-9]+)% of armour and evasion rating over ([0-9]+) second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegen", Base, Num(1), &CondTag{Var: "LifeRegenBurstFull"}, &StatTag{StatKind: TagPercentStat, Stat: "Armour", Percent: opt(c.n(2))}), mod("LifeRegen", Base, Num(1/c.n(1)*c.n(3)), &CondTag{Var: "LifeRegenBurstAvg"}, &StatTag{StatKind: TagPercentStat, Stat: "Armour", Percent: opt(c.n(2))}), mod("LifeRegen", Base, Num(1), &CondTag{Var: "LifeRegenBurstFull"}, &StatTag{StatKind: TagPercentStat, Stat: "Evasion", Percent: opt(c.n(2))}), mod("LifeRegen", Base, Num(1/c.n(1)*c.n(3)), &CondTag{Var: "LifeRegenBurstAvg"}, &StatTag{StatKind: TagPercentStat, Stat: "Evasion", Percent: opt(c.n(2))})}
	}),
	`every ([0-9]+) seconds, regenerate energy shield equal to ([0-9]+)% of evasion rating over ([0-9]+) second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldRegen", Base, Num(1), &CondTag{Var: "LifeRegenBurstFull"}, &StatTag{StatKind: TagPercentStat, Stat: "Evasion", Percent: opt(c.n(2))}), mod("EnergyShieldRegen", Base, Num(1/c.n(1)*c.n(3)), &CondTag{Var: "LifeRegenBurstAvg"}, &StatTag{StatKind: TagPercentStat, Stat: "Evasion", Percent: opt(c.n(2))})}
	}),
	`regenerate ([0-9]+)% of life per second for each different ailment affecting you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Bleeding"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Ignited"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Scorched"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Chilled"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Frozen"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Brittle"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Shocked"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Sapped"}), mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Poisoned"})}
	}),
	`fire skills have a ([0-9]+)% chance to apply fire exposure on hit`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("FireExposureChance", Base, Num(c.n(1)))} }),
	`cold skills have a ([0-9]+)% chance to apply cold exposure on hit`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("ColdExposureChance", Base, Num(c.n(1)))} }),
	`lightning skills have a ([0-9]+)% chance to apply lightning exposure on hit`:        modFn(func(c caps) []*Mod { return []*Mod{mod("LightningExposureChance", Base, Num(c.n(1)))} }),
	`([0-9]+)% chance to inflict cold exposure on hit with cold damage`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("ColdExposureChance", Base, Num(c.n(1)))} }),
	`socketed skills apply fire, cold and lightning exposure on hit`:                     modList{mod("FireExposureChance", Base, Num(100), &CondTag{Var: "Effective"}), mod("ColdExposureChance", Base, Num(100), &CondTag{Var: "Effective"}), mod("LightningExposureChance", Base, Num(100), &CondTag{Var: "Effective"})},
	`inflict fire, cold, and lightning exposure on hit`:                                  modList{mod("FireExposureChance", Base, Num(100), &CondTag{Var: "Effective"}), mod("ColdExposureChance", Base, Num(100), &CondTag{Var: "Effective"}), mod("LightningExposureChance", Base, Num(100), &CondTag{Var: "Effective"})},
	`inflict fire exposure on hit`:                                                       modList{mod("FireExposureChance", Base, Num(100), &CondTag{Var: "Effective"})},
	`inflict cold exposure on hit`:                                                       modList{mod("ColdExposureChance", Base, Num(100), &CondTag{Var: "Effective"})},
	`inflict lightning exposure on hit`:                                                  modList{mod("LightningExposureChance", Base, Num(100), &CondTag{Var: "Effective"})},
	`nearby enemies have fire exposure`:                                                  modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10))}, &CondTag{Var: "Effective"})},
	`nearby enemies have cold exposure`:                                                  modList{mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-10))}, &CondTag{Var: "Effective"})},
	`nearby enemies have lightning exposure`:                                             modList{mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-10))}, &CondTag{Var: "Effective"})},
	`nearby enemies have fire exposure while at maximum rage`:                            modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "HaveMaximumRage"})},
	`nearby enemies have fire exposure while you are affected by herald of ash`:          modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "AffectedByHeraldofAsh"})},
	`nearby enemies have cold exposure while you are affected by herald of ice`:          modList{mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "AffectedByHeraldofIce"})},
	`nearby enemies have lightning exposure while you are affected by herald of thunder`: modList{mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "AffectedByHeraldofThunder"})},
	`inflict fire, cold and lightning exposure on nearby enemies when used`:              modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "UsingFlask"}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "UsingFlask"}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-10))}, &CondTag{Var: "Effective"}, &CondTag{Var: "UsingFlask"})},
	`enemies near your linked targets have fire, cold and lightning exposure`:            modList{mod("EnemyModifier", List, ModRef{Mod: mod("FireExposure", Base, Num(-10), &CondTag{Var: "NearLinkedTarget"})}, &CondTag{Var: "Effective"}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdExposure", Base, Num(-10), &CondTag{Var: "NearLinkedTarget"})}, &CondTag{Var: "Effective"}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningExposure", Base, Num(-10), &CondTag{Var: "NearLinkedTarget"})}, &CondTag{Var: "Effective"})},
	`inflict ([0-9a-zA-Z]+) exposure on hit, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(1))+"ExposureChance", Base, Num(100), &CondTag{Var: "Effective"}), mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(3))+"Exposure", Base, Num(-c.n(2)))}, &CondTag{Var: "Effective"})}
	}),
	`while a unique enemy is in your presence, inflict ([0-9a-zA-Z]+) exposure on hit, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(1))+"ExposureChance", Base, Num(100), &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"}, &CondTag{Var: "Effective"}), mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(3))+"Exposure", Base, Num(-c.n(2)), &CondTag{Var: "RareOrUnique"})}, &CondTag{Var: "Effective"})}
	}),
	`while a pinnacle atlas boss is in your presence, inflict ([0-9a-zA-Z]+) exposure on hit, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(1))+"ExposureChance", Base, Num(100), &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"}, &CondTag{Var: "Effective"}), mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(3))+"Exposure", Base, Num(-c.n(2)), &CondTag{Var: "PinnacleBoss"})}, &CondTag{Var: "Effective"})}
	}),
	`inflict fire exposure on hit against enemies with ([0-9]+) cinderflame, applying -([0-9]+)% to ([0-9a-zA-Z]+) resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireExposureChance", Base, Num(100), &CondTag{Var: "Effective"}, &MultiplierTag{IsThreshold: true, Var: "CinderflameStacks", Threshold: opt(c.n(1))}), mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(3))+"Exposure", Base, Num(-c.n(2)))}, &CondTag{Var: "Effective"}, &MultiplierTag{IsThreshold: true, Var: "CinderflameStacks", Threshold: opt(c.n(1))})}
	}),
	`fire exposure you inflict applies an extra (-?[0-9]+)% to fire resistance`:           modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraFireExposure", Base, Num(c.n(1)))} }),
	`cold exposure you inflict applies an extra (-?[0-9]+)% to cold resistance`:           modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraColdExposure", Base, Num(c.n(1)))} }),
	`lightning exposure you inflict applies an extra (-?[0-9]+)% to lightning resistance`: modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraLightningExposure", Base, Num(c.n(1)))} }),
	`exposure you inflict applies at least (-[0-9]+)% to the affected resistance`:         modFn(func(c caps) []*Mod { return []*Mod{mod("ExposureMin", Override, Num(c.n(1)))} }),
	`modifiers to minimum endurance charges instead apply to minimum brutal charges`:      modList{flag("MinimumEnduranceChargesEqualsMinimumBrutalCharges")},
	`modifiers to minimum frenzy charges instead apply to minimum affliction charges`:     modList{flag("MinimumFrenzyChargesEqualsMinimumAfflictionCharges")},
	`modifiers to minimum power charges instead apply to minimum absorption charges`:      modList{flag("MinimumPowerChargesEqualsMinimumAbsorptionCharges")},
	`maximum brutal charges is equal to maximum endurance charges`:                        modList{flag("MaximumEnduranceChargesEqualsMaximumBrutalCharges")},
	`maximum affliction charges is equal to maximum frenzy charges`:                       modList{flag("MaximumFrenzyChargesEqualsMaximumAfflictionCharges")},
	`maximum absorption charges is equal to maximum power charges`:                        modList{flag("MaximumPowerChargesEqualsMaximumAbsorptionCharges")},
	`([0-9]+)% chance to gain a brine charge instead of an endurance charge`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("CanGainBrineCharges"), mod("BrineChargeGainChance", Base, Num(c.n(1)))}
	}),
	`gain brutal charges instead of endurance charges`:  modList{flag("EnduranceChargesConvertToBrutalCharges")},
	`gain affliction charges instead of frenzy charges`: modList{flag("FrenzyChargesConvertToAfflictionCharges")},
	`gain absorption charges instead of power charges`:  modList{flag("PowerChargesConvertToAbsorptionCharges")},
	`regenerate ([0-9]+)% life over one second when hit while sane`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(1)), &CondTag{Var: "Insane", Neg: true}, &CondTag{Var: "BeenHitRecently"})}
	}),
	`you count as on low ([a-zA-Z]+) while at ([0-9]+)% of maximum ([a-zA-Z]+) or below`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Low"+firstToUpper(c.s(1))+"Percentage", Base, Num(c.n(2)/100.0))}
	}),
	`you count as on full ([a-zA-Z]+) while at ([0-9]+)% of maximum ([a-zA-Z]+) or above`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Full"+firstToUpper(c.s(1))+"Percentage", Base, Num(c.n(2)/100.0))}
	}),
	`([0-9]+)% more maximum life if you have at least ([0-9]+) life masteries allocated`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Life", More, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "AllocatedLifeMastery", Threshold: opt(c.n(2))})}
	}),
	`left ring slot: cover enemies in ash for ([0-9]+) seconds when you ignite them`:    modList{mod("CoveredInAshEffect", Base, Num(20), &SlotTag{SlotKind: TagSlotNumber, Num: 1}, &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"})},
	`right ring slot: cover enemies in frost for ([0-9]+) seconds when you freeze them`: modList{mod("CoveredInFrostEffect", Base, Num(20), &SlotTag{SlotKind: TagSlotNumber, Num: 2}, &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"})},
	`nearby enemies are covered in ash`:                                                 modList{mod("CoveredInAshEffect", Base, Num(20))},
	`nearby enemies are covered in ash if you haven't moved in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CoveredInAshEffect", Base, Num(20), &MultiplierTag{IsThreshold: true, Var: "StationarySeconds", Threshold: opt(c.n(1))}, &CondTag{Var: "Stationary"})}
	}),
	`your warcries cover enemies in ash for ([0-9]+) seconds`:                                            modList{mod("CoveredInAshEffect", Base, Num(20), &CondTag{Var: "UsedWarcryRecently"})},
	`enemies near targets you shatter have ([0-9]+)% chance to be covered in frost for ([0-9]+) seconds`: modList{mod("CoveredInFrostEffect", Base, Num(20), &CondTag{Var: "ShatteredEnemyRecently"})},
	`([a-zA-Z \t\n\v\f\r]+) has ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BuffEffect", Inc, c.v(2), &SkillIDTag{SkillID: gemIdOrNil(c.s(1))})}
	}),
	`debuffs on you expire ([0-9]+)% faster`: modFn(func(c caps) []*Mod { return []*Mod{mod("SelfDebuffExpirationRate", Base, Num(c.n(1)))} }),
	`debuffs on you expire ([0-9]+)% slower`: modFn(func(c caps) []*Mod { return []*Mod{mod("SelfDebuffExpirationRate", Base, Num(-c.n(1)))} }),
	`debuffs on you expire ([0-9]+)% faster while affected by haste`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SelfDebuffExpirationRate", Base, Num(c.n(1)), &CondTag{Var: "AffectedByHaste"})}
	}),
	`warcries debilitate enemies for ([0-9]+) seconds?`:                                                                                          modList{mod("DebilitateChance", Base, Num(100))},
	`debilitate enemies for ([0-9]+) seconds? when you suppress their spell damage`:                                                              modList{mod("DebilitateChance", Base, Num(100))},
	`debilitate nearby enemies for ([0-9]+) seconds? when f?l?a?s?k? ?effect ends`:                                                               modList{mod("DebilitateChance", Base, Num(100))},
	`counterattacks have a ([0-9]+)% chance to debilitate on hit for ([0-9]+) seconds?`:                                                          modFn(func(c caps) []*Mod { return []*Mod{mod("DebilitateChance", Base, Num(c.n(1)))} }),
	`retaliation skills debilitate enemies for ([0-9]+) seconds on hit`:                                                                          modList{mod("DebilitateChance", Base, Num(100))},
	`eat a soul when you hit a unique enemy, no more than once every second`:                                                                     modList{flag("Condition:CanHaveSoulEater")},
	`eat a soul when you hit a rare or unique enemy, no more than once every [0-9.]+ seconds`:                                                    modList{flag("Condition:CanHaveSoulEater")},
	`([0-9]+)% chance to gain soul eater for ([0-9]+) seconds on killing blow against rare and unique enemies with double strike or dual strike`: modList{flag("Condition:CanHaveSoulEater")},
	`when ([0-9]+)% of your hex's duration expires on an enemy, eat ([0-9]+) soul per enemy power`:                                               modList{flag("Condition:CanHaveSoulEater")},
	`eat ([0-9]+) souls when you kill a rare or unique enemy with this weapon`:                                                                   modList{flag("Condition:CanHaveSoulEater")},
	`maximum ([0-9]+) eaten souls`:                   modFn(func(c caps) []*Mod { return []*Mod{mod("SoulEaterMax", Override, Num(c.n(1)))} }),
	`([+\-][0-9]+) to maximum number of eaten souls`: modFn(func(c caps) []*Mod { return []*Mod{mod("SoulEaterMax", Base, Num(c.n(1)))} }),
	`([0-9]+)% increased attack and cast speed if you've killed recently`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Speed", Inc, Num(c.n(1)), FlagCast, KeywordNone, &CondTag{Var: "KilledRecently"}), modf("Speed", Inc, Num(c.n(1)), FlagAttack, KeywordNone, &CondTag{Var: "KilledRecently"})}
	}),
	`gain adrenaline for 1 second when you change stance`:                                        modList{flag("Condition:Adrenaline", &CondTag{Var: "StanceChangeLastSecond"})},
	`with a searching eye jewel socketed, maim enemies for ([0-9]) seconds on hit with attacks`:  modList{mod("EnemyModifier", List, ModRef{Mod: flagf("Condition:Maimed", FlagAttack, KeywordNone)}, &CondTag{Var: "HaveSearchingEyeJewelIn{SlotName}"})},
	`with a searching eye jewel socketed, blind enemies for ([0-9]) seconds on hit with attacks`: modList{mod("EnemyModifier", List, ModRef{Mod: flagf("Condition:Blinded", FlagAttack, KeywordNone)}, &CondTag{Var: "HaveSearchingEyeJewelIn{SlotName}"})},
	`enemies maimed by you take ([0-9]+)% increased damage over time`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTakenOverTime", Inc, Num(c.n(1)))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Maimed"})}
	}),
	`([0-9]+)% increased defences while you have at least four linked targets`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Defences", Inc, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "LinkedTargets", Threshold: opt(4)})}
	}),
	`your movement speed is equal to the highest movement speed among linked players`: modList{flag("MovementSpeedEqualHighestLinkedPlayers", &MultiplierTag{IsThreshold: true, Var: "LinkedTargets", Threshold: opt(1)})},
	`([0-9]+)% increased movement speed while you have at least two linked targets`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MovementSpeed", Inc, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "LinkedTargets", Threshold: opt(2)})}
	}),
	`link skills have ([0-9]+)% increased buff effect if you have linked to a target recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BuffEffect", Inc, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeLink}, &CondTag{Var: "LinkedRecently"})}
	}),
	`link skills can target damageable minions`: modList{flag("Condition:CanLinkToMinions", &CondTag{Var: "HaveDamageableMinion"})},
	`your linked minions take ([0-9]+)% less damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("DamageTaken", More, Num(-c.n(1)), &CondTag{Var: "AffectedByLink"})})}
	}),
	`curses are inflicted on you instead of linked targets`:                       modList{mod("ExtraLinkEffect", List, ModRef{Mod: flag("CurseImmune")})},
	`elemental ailments are inflicted on you instead of linked targets`:           modList{mod("ExtraLinkEffect", List, ModRef{Mod: flag("ElementalAilmentImmune")})},
	`non-unique utility flasks you use apply to linked targets`:                   modList{mod("ExtraLinkEffect", List, ModRef{Mod: mod("ParentNonUniqueFlasksAppliedToYou", Flag, Bool(true), &GlobalEffectTag{EffectType: "Global", Unscalable: true})})},
	`non-curse auras from your skills only apply to you and linked targets`:       modList{flag("SelfAurasAffectYouAndLinkedTarget", &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeAppliesCurse, Neg: true})},
	`linked targets always count as in range of non-curse auras from your skills`: modList{},
	`gain unholy might on block for ([0-9]) seconds`:                              modList{flag("Condition:UnholyMight", &CondTag{Var: "BlockedRecently"}), flag("Condition:CanWither", &CondTag{Var: "BlockedRecently"})},
	`your warcries inflict hallowing flame`:                                       modList{flag("Condition:CanInflictHallowingFlame", &CondTag{Var: "UsedWarcryRecently"})},
	`attacks with this weapon inflict hallowing flame on hit`:                     modList{flag("Condition:CanInflictHallowingFlame")},
	`inflict hallowing flame on hit while on consecrated ground`:                  modList{flag("Condition:CanInflictHallowingFlame", &CondTag{Var: "OnConsecratedGround"})},
	`inflict hallowing flame on melee hit`:                                        modList{flag("Condition:CanInflictHallowingFlame")},
	// Traps, Mines
	`traps and mines deal ([0-9]+)-([0-9]+) additional physical damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalMin", Base, Num(c.n(1)), FlagNone, KeywordTrap|KeywordMine), modf("PhysicalMax", Base, Num(c.n(2)), FlagNone, KeywordTrap|KeywordMine)}
	}),
	`traps and mines deal ([0-9]+) to ([0-9]+) additional physical damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("PhysicalMin", Base, Num(c.n(1)), FlagNone, KeywordTrap|KeywordMine), modf("PhysicalMax", Base, Num(c.n(2)), FlagNone, KeywordTrap|KeywordMine)}
	}),
	`each mine applies ([0-9]+)% increased damage taken to enemies near it, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &MultiplierTag{Var: "ActiveMineCount", Limit: opt(c.n(2) / c.n(1))})})}
	}),
	`each mine applies ([0-9]+)% reduced damage dealt to enemies near it, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("Damage", Inc, Num(-c.n(1)), &MultiplierTag{Var: "ActiveMineCount", Limit: opt(c.n(2) / c.n(1))})})}
	}),
	`stormblast, icicle and pyroclast mine have ([0-9]+)% increased aura effect`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("AuraEffect", Inc, Num(c.n(1)), FlagNone, KeywordMine, &SkillNameTag{SkillNameList: []string{"Stormblast Mine", "Icicle Mine", "Pyroclast Mine"}, IncludeTransfigured: true})}
	}),
	`stormblast, icicle and pyroclast mine deal no damage`:              modList{flag("DealNoDamage", &SkillNameTag{SkillNameList: []string{"Stormblast Mine", "Icicle Mine", "Pyroclast Mine"}, IncludeTransfigured: true})},
	`can have up to ([0-9]+) additional traps? placed at a time`:        modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveTrapLimit", Base, Num(c.n(1)))} }),
	`can have ([0-9]+) fewer traps placed at a time`:                    modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveTrapLimit", Base, Num(-c.n(1)))} }),
	`can have up to ([0-9]+) additional remote mines? placed at a time`: modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveMineLimit", Base, Num(c.n(1)))} }),
	// Additional trap & mine throw
	`throw an additional trap`:                                   modList{mod("TrapThrowCount", Base, Num(1))},
	`([0-9]+)% chance to throw up to ([0-9]+) additional traps?`: modFn(func(c caps) []*Mod { return []*Mod{mod("TrapThrowCount", Base, Num(c.n(2)*c.n(1)/100.0))} }),
	`throw an additional mine`:                                   modList{mod("MineThrowCount", Base, Num(1))},
	`([0-9]+)% chance to throw up to ([0-9]+) additional mines?`: modFn(func(c caps) []*Mod { return []*Mod{mod("MineThrowCount", Base, Num(c.n(2)*c.n(1)/100.0))} }),
	`([0-9]+)% chance to throw up to ([0-9]+) additional traps? or mines?`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TrapThrowCount", Base, Num(c.n(2)*c.n(1)/100.0)), mod("MineThrowCount", Base, Num(c.n(2)*c.n(1)/100.0))}
	}),
	`skills which throw traps throw up to ([0-9]+) additional traps?`: modFn(func(c caps) []*Mod { return []*Mod{mod("TrapThrowCount", Base, Num(c.n(1)))} }),
	// Totems
	`can have up to ([0-9]+) additional totems? summoned at a time`: modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveTotemLimit", Base, Num(c.n(1)))} }),
	`attack skills can have ([0-9]+) additional totems? summoned at a time`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ActiveTotemLimit", Base, Num(c.n(1)), FlagNone, KeywordAttack)}
	}),
	`can [hs][au][vm][em]o?n? 1 additional siege ballista totem per ([0-9]+) dexterity`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ActiveBallistaLimit", Base, Num(1), &SkillNameTag{SkillName: "Siege Ballista", IncludeTransfigured: true}, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(1))})}
	}),
	`totems fire ([0-9]+) additional projectiles`: modFn(func(c caps) []*Mod { return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagNone, KeywordTotem)} }),
	`([0-9.]+)% of damage dealt by y?o?u?r? ?totems is leeched to you as life`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("DamageLifeLeechToPlayer", Base, Num(c.n(1)), FlagNone, KeywordTotem)}
	}),
	`([0-9.]+)% of damage dealt by y?o?u?r? ?mines is leeched to you as life`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("DamageLifeLeechToPlayer", Base, Num(c.n(1)), FlagNone, KeywordMine)}
	}),
	`you can cast an additional brand`:        modList{mod("ActiveBrandLimit", Base, Num(1))},
	`you can cast ([0-9]+) additional brands`: modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveBrandLimit", Base, Num(c.n(1)))} }),
	`([0-9]+)% increased damage while you are wielding a bow and have a totem`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", Inc, Num(c.n(1)), &CondTag{Var: "HaveTotem"}, &CondTag{Var: "UsingBow"})}
	}),
	`each totem applies ([0-9]+)% increased damage taken to enemies near it`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &MultiplierTag{Var: "TotemsSummoned"})})}
	}),
	`totems gain \+([0-9]+)% to ([0-9a-zA-Z]+) resistance`:                                   modFn(func(c caps) []*Mod { return []*Mod{mod("Totem"+firstToUpper(c.s(2))+"Resist", Base, Num(c.n(1)))} }),
	`totems gain \+([0-9]+)% to all elemental resistances`:                                   modFn(func(c caps) []*Mod { return []*Mod{mod("TotemElementalResist", Base, Num(c.n(1)))} }),
	`rejuvenation totem also grants mana regeneration equal to 15% of its life regeneration`: modList{flag("Condition:RejuvenationTotemManaRegen")},
	// Minions
	`your strength is added to your minions`:                                modList{mod("StrengthAddedToMinions", Base, Num(100))},
	`half of your strength is added to your minions`:                        modList{mod("StrengthAddedToMinions", Base, Num(50))},
	`minions gain added resistances equal to ([0-9]+)% of your resistances`: modFn(func(c caps) []*Mod { return []*Mod{mod("ResistanceAddedToMinions", Base, Num(c.n(1)))} }),
	`minions' accuracy rating is equal to yours`:                            modList{flag("MinionAccuracyEqualsAccuracy")},
	`minions poison enemies on hit`:                                         modList{mod("MinionModifier", List, ModRef{Mod: mod("PoisonChance", Base, Num(100))})},
	`minions have ([0-9]+)% chance to poison enemies on hit`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("PoisonChance", Base, Num(c.n(1)))})}
	}),
	`([0-9]+)% increased minion damage if you have hit recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)))}, &CondTag{Var: "HitRecently"})}
	}),
	`([0-9]+)% increased minion damage if you've used a minion skill recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)))}, &CondTag{Var: "UsedMinionSkillRecently"})}
	}),
	`minions deal ([0-9]+)% increased damage if you have warcried recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "UsedWarcryRecently"})})}
	}),
	`minions deal ([0-9]+)% increased damage if you've used a minion skill recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)))}, &CondTag{Var: "UsedMinionSkillRecently"})}
	}),
	`minions deal ([0-9]+)% more damage while they are on low life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", More, Num(c.n(1)), &CondTag{Var: "LowLife"})})}
	}),
	`minions have ([0-9]+)% increased attack and cast speed if you or your minions have killed recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Speed", Inc, Num(c.n(1)))}, &CondTag{VarList: []string{"KilledRecently", "MinionsKilledRecently"}})}
	}),
	`([0-9]+)% increased minion attack speed per ([0-9]+) dexterity`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: modf("Speed", Inc, Num(c.n(1)), FlagAttack, KeywordNone)}, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(2))})}
	}),
	`([0-9]+)% increased minion movement speed per ([0-9]+) dexterity`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("MovementSpeed", Inc, Num(c.n(1)))}, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(2))})}
	}),
	`minions deal ([0-9]+)% increased damage per ([0-9]+) dexterity`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)))}, &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(2))})}
	}),
	`minions have ([0-9]+)% chance to deal double damage while they are on full life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{Var: "FullLife"})})}
	}),
	`minions have ([0-9]+)% chance to deal double damage per fortification on you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("DoubleDamageChance", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "FortificationStacks", Actor: "parent"})})}
	}),
	`([0-9]+)% increased golem damage for each type of golem you have summoned`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "HavePhysicalGolem"})}, &SkillTypeTag{SkillType: SkillTypeGolem}), mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "HaveLightningGolem"})}, &SkillTypeTag{SkillType: SkillTypeGolem}), mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "HaveColdGolem"})}, &SkillTypeTag{SkillType: SkillTypeGolem}), mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "HaveFireGolem"})}, &SkillTypeTag{SkillType: SkillTypeGolem}), mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "HaveChaosGolem"})}, &SkillTypeTag{SkillType: SkillTypeGolem}), mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "HaveCarrionGolem"})}, &SkillTypeTag{SkillType: SkillTypeGolem})}
	}),
	`can summon up to ([0-9]) additional golems? at a time`: modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveGolemLimit", Base, Num(c.n(1)))} }),
	`\+([0-9]) to maximum number of sentinels of purity`:    modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveSentinelOfPurityLimit", Base, Num(c.n(1)))} }),
	`if you have 3 primordial jewels, can summon up to ([0-9]) additional golems? at a time`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ActiveGolemLimit", Base, Num(c.n(1)), &MultiplierTag{IsThreshold: true, Var: "PrimordialItem", Threshold: opt(3)})}
	}),
	`golems regenerate ([0-9])% of their maximum life per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(c.n(1)))}, &SkillTypeTag{SkillType: SkillTypeGolem})}
	}),
	`summoned golems regenerate ([0-9])% of their life per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(c.n(1)))}, &SkillTypeTag{SkillType: SkillTypeGolem})}
	}),
	`summoned carrion golems impale on hit if you have the same number of them as summoned chaos golems`: modList{mod("MinionModifier", List, ModRef{Mod: mod("ImpaleChance", Base, Num(100), &CondTag{IsActor: true, Actor: "parent", Var: "CarrionEqualChaosGolem"}, &CondTag{IsActor: true, Actor: "parent", Var: "HaveChaosGolem"})}, &SkillNameTag{SkillName: "Summon Carrion Golem", IncludeTransfigured: true})},
	`summoned chaos golems impale on hit if you have the same number of them as summoned stone golems`:   modList{mod("MinionModifier", List, ModRef{Mod: mod("ImpaleChance", Base, Num(100), &CondTag{IsActor: true, Actor: "parent", Var: "ChaosEqualStoneGolem"}, &CondTag{IsActor: true, Actor: "parent", Var: "HavePhysicalGolem"})}, &SkillNameTag{SkillName: "Summon Chaos Golem", IncludeTransfigured: true})},
	`summoned stone golems impale on hit if you have the same number of them as summoned carrion golems`: modList{mod("MinionModifier", List, ModRef{Mod: mod("ImpaleChance", Base, Num(100), &CondTag{IsActor: true, Actor: "parent", Var: "StoneEqualCarrionGolem"}, &CondTag{IsActor: true, Actor: "parent", Var: "HaveCarrionGolem"})}, &SkillNameTag{SkillName: "Summon Stone Golem", IncludeTransfigured: true})},
	`maximum life of summoned elemental golems is doubled`:                                               modList{mod("MinionModifier", List, ModRef{Mod: mod("Life", More, Num(100))}, &SkillTypeTag{SkillType: SkillTypeGolem}, &SkillTypeTag{SkillTypeList: []SkillTypeID{SkillTypeLightning, SkillTypeCold, SkillTypeFire}})},
	`golems summoned in the past 8 seconds deal ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "SummonedGolemInPast8Sec"})}, &SkillTypeTag{SkillType: SkillTypeGolem})}
	}),
	`raised zombies and spectres gain adrenaline for 8 seconds when raised`: modList{mod("MinionModifier", List, ModRef{Mod: flag("Condition:Adrenaline")}, &CondTag{Var: "SummonedSpectreInPast8Sec"}, &SkillNameTag{SkillName: "Raise Spectre", IncludeTransfigured: true}), mod("MinionModifier", List, ModRef{Mod: flag("Condition:Adrenaline")}, &CondTag{Var: "SummonedZombieInPast8Sec"}, &SkillNameTag{SkillName: "Raise Zombie", IncludeTransfigured: true})},
	`raised spectres fire ([0-9]+) additional projectiles`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ProjectileCount", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Raise Spectre", IncludeTransfigured: true})}
	}),
	`gain onslaught for 10 seconds when you cast socketed golem skill`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Condition:Onslaught", &CondTag{Var: "SummonedGolemInPast10Sec"})}
	}),
	`s?u?m?m?o?n?e?d? ?raging spirits' hits always ignite`: modList{mod("MinionModifier", List, ModRef{Mod: mod("EnemyIgniteChance", Base, Num(100))}, &SkillNameTag{SkillName: "Summon Raging Spirit", IncludeTransfigured: true})},
	`raised zombies have avatar of fire`:                   modList{mod("MinionModifier", List, ModRef{Mod: mod("Keystone", List, Str("Avatar of Fire"))}, &SkillNameTag{SkillName: "Raise Zombie", IncludeTransfigured: true})},
	`raised zombies take ([0-9.]+)% of their maximum life per second as fire damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("FireDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}, &SkillNameTag{SkillName: "Raise Zombie", IncludeTransfigured: true})}
	}),
	`maximum number of summoned raging spirits is ([0-9]+)`:                modFn(func(c caps) []*Mod { return []*Mod{mod("ActiveRagingSpiritLimit", Override, Num(c.n(1)))} }),
	`maximum number of summoned phantasms is ([0-9]+)`:                     modFn(func(c caps) []*Mod { return []*Mod{mod("ActivePhantasmLimit", Override, Num(c.n(1)))} }),
	`summoned raging spirits have diamond shrine and massive shrine buffs`: modList{mod("MinionModifier", List, ModRef{Mod: flag("Condition:DiamondShrine")}, &SkillNameTag{SkillName: "Summon Raging Spirit", IncludeTransfigured: true}), mod("MinionModifier", List, ModRef{Mod: flag("Condition:MassiveShrine")}, &SkillNameTag{SkillName: "Summon Raging Spirit", IncludeTransfigured: true})},
	`summoned phantasms have diamond shrine and massive shrine buffs`:      modList{mod("MinionModifier", List, ModRef{Mod: flag("Condition:DiamondShrine")}, &SkillNameTag{SkillName: "Summon Phantasm"}), mod("MinionModifier", List, ModRef{Mod: flag("Condition:MassiveShrine")}, &SkillNameTag{SkillName: "Summon Phantasm"})},
	`minions deal no non-([a-zA-Z]+) damage`:                               modFn(func(c caps) []*Mod { return dealNoNonDamageType(c.s(1), true) }),
	`minions convert ([0-9]+)% of (.+) damage to (.+) damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)))})}
	}),
	`summoned skeletons have avatar of fire`: modList{mod("MinionModifier", List, ModRef{Mod: mod("Keystone", List, Str("Avatar of Fire"))}, &SkillNameTag{SkillName: "Summon Skeletons", IncludeTransfigured: true})},
	`summoned skeletons take ([0-9.]+)% of their maximum life per second as fire damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("FireDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}, &SkillNameTag{SkillName: "Summon Skeletons", IncludeTransfigured: true})}
	}),
	`summoned skeletons have ([0-9]+)% chance to wither enemies for ([0-9]+) seconds on hit`: modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("Condition:CanWither")}, &SkillNameTag{SkillName: "Summon Skeletons", IncludeTransfigured: true})},
	`summoned skeletons have ([0-9]+)% of physical damage converted to chaos damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("PhysicalDamageConvertToChaos", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Skeletons", IncludeTransfigured: true})}
	}),
	`summoned skeletons gain added chaos damage equal to ([0-9]+)% of maximum energy shield on your equipped shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShieldOnWeapon 2", Actor: "parent", Percent: opt(c.n(1))})}), mod("MinionModifier", List, ModRef{Mod: mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShieldOnWeapon 2", Actor: "parent", Percent: opt(c.n(1))})})}
	}),
	`skeletons gain added chaos damage equal to ([0-9]+)% of maximum energy shield on your equipped shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShieldOnWeapon 2", Actor: "parent", Percent: opt(c.n(1))})}), mod("MinionModifier", List, ModRef{Mod: mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShieldOnWeapon 2", Actor: "parent", Percent: opt(c.n(1))})})}
	}),
	`minions convert ([0-9]+)% of (.+) damage to (.+) damage per (.+) socket`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)))}, &MultiplierTag{Var: firstToUpper(c.s(4)) + "SocketIn{SlotName}"})}
	}),
	`minions have a ([0-9]+)% chance to impale on hit with attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ImpaleChance", Base, Num(c.n(1)))})}
	}),
	`minions have ([0-9]+)% chance to impale on attack hit per socketed ghastly eye jewel`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ImpaleChance", Base, Num(c.n(1)))}, &MultiplierTag{Var: "GhastlyEyeJewelIn{SlotName}"})}
	}),
	`minions from herald skills deal ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", More, Num(c.n(1)))}, &SkillTypeTag{SkillType: SkillTypeHerald})}
	}),
	`minions have ([0-9]+)% increased movement speed for each herald affecting you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("MovementSpeed", Inc, Num(c.n(1)), &MultiplierTag{Var: "Herald", Actor: "parent"})})}
	}),
	`minions deal ([0-9]+)% increased damage while you are affected by a herald`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "AffectedByHerald"})})}
	}),
	`minions have ([0-9]+)% increased attack and cast speed while you are affected by a herald`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Speed", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "parent", Var: "AffectedByHerald"})})}
	}),
	`minions have unholy might`: modList{mod("MinionModifier", List, ModRef{Mod: flag("Condition:UnholyMight")}), mod("MinionModifier", List, ModRef{Mod: flag("Condition:CanWither")})},
	`summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?deal triple damage with this weapon if you've hit with this weapon recently`: modList{mod("MinionModifier", List, ModRef{Mod: mod("TripleDamageChance", Base, Num(100), &CondTag{IsActor: true, Actor: "parent", Var: "HitRecentlyWithWeapon"})}, &SkillNameTag{SkillName: "Summon Skeletons"}, &ItemCondTag{ItemSlot: "Weapon 1", NameCond: "The Iron Mass, Gladius"})},
	`summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?wield a? ?c?o?p?y? ?o?f? ?this weapon while in your main hand`:               modList{},
	`each summoned phantasm grants you phantasmal might`:                                                                               modList{flag("Condition:PhantasmalMight")},
	`minions have ([0-9]+)% increased critical strike chance per maximum power charge you have`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("CritChance", Inc, Num(c.n(1)), &MultiplierTag{Actor: "parent", Var: "PowerChargeMax"})})}
	}),
	`minions' base attack critical strike chance is equal to the critical strike chance of your main hand weapon`: modList{mod("MinionModifier", List, ModRef{Mod: flagf("AttackCritIsEqualToParentMainHand", FlagAttack, KeywordNone)})},
	`non-spectre minions' base attack time is equal to the attack time of your main hand weapon`:                  modList{flag("NonSpectreMinionsUseParentMainHandAttackTime")},
	`minions can hear the whispers for 5 seconds after they deal a critical strike`:                               modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("Speed", Inc, Num(50), &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &CondTag{Neg: true, Var: "NeverCrit"})}), mod("ExtraSkillMod", List, ModRef{Mod: mod("Damage", Inc, Num(50), &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &CondTag{Neg: true, Var: "NeverCrit"})}), mod("ExtraSkillMod", List, ModRef{Mod: mod("ChaosDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(20)}, &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &CondTag{Neg: true, Var: "NeverCrit"})})},
	`chaos damage t?a?k?e?n? ?does not bypass minions' energy shield`:                                             modList{mod("MinionModifier", List, ModRef{Mod: flag("ChaosNotBypassEnergyShield")})},
	`while minions have energy shield, their hits ignore monster elemental resistances`:                           modList{mod("MinionModifier", List, ModRef{Mod: flag("IgnoreElementalResistances", &StatTag{StatKind: TagStatThreshold, Stat: "EnergyShield", Threshold: opt(1)})})},
	`summoned arbalists' projectiles pierce ([0-9]+) additional targets`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("PierceCount", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Arbalists"})}
	}),
	`summoned arbalists' projectiles fork`: modList{mod("MinionModifier", List, ModRef{Mod: flag("ForkOnce")}, &SkillNameTag{SkillName: "Summon Arbalists"}), mod("MinionModifier", List, ModRef{Mod: mod("ForkCountMax", Base, Num(1))}, &SkillNameTag{SkillName: "Summon Arbalists"})},
	`summoned arbalists' projectiles chain \+([0-9]+) times`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ChainCountMax", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Arbalists"})}
	}),
	`summoned arbalists have ([0-9]+)% chance to inflict (.+) exposure on hit`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(2))+"ExposureChance", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Summon Arbalists"})}
	}),
	`summoned arbalists convert ([0-9]+)% of (.+) damage to (.+) damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Arbalists"})}
	}),
	`summoned arbalists have ([0-9]+)% chance to freeze, shock, and ignite`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("EnemyFreezeChance", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Arbalists"}), mod("MinionModifier", List, ModRef{Mod: mod("EnemyShockChance", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Arbalists"}), mod("MinionModifier", List, ModRef{Mod: mod("EnemyIgniteChance", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Summon Arbalists"})}
	}),
	`skeleton warriors are permanent minions and follow you`:  modList{flag("RaisedSkeletonPermanentDuration", &SkillNameTag{SkillName: "Summon Skeletons"})},
	`summoned skeleton warriors are permanent and follow you`: modList{flag("RaisedSkeletonPermanentDuration", &SkillNameTag{SkillName: "Summon Skeletons"})},
	`your blink and mirror arrow clones use your gloves`:      modList{flag("BlinkAndMirrorUseGloves")},
	// Projectiles
	`skills chain \+([0-9]) times`:         modFn(func(c caps) []*Mod { return []*Mod{mod("ChainCountMax", Base, Num(c.n(1)))} }),
	`projectiles chain an additional time`: modList{modf("ChainCountMax", Base, Num(1), FlagProjectile, KeywordNone)},
	`projectiles chain \+([0-9]) times`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ChainCountMax", Base, Num(c.n(1)), FlagProjectile, KeywordNone)}
	}),
	`arrows chain \+([0-9]) times`:                                    modFn(func(c caps) []*Mod { return []*Mod{modf("ChainCountMax", Base, Num(c.n(1)), FlagNone, KeywordArrow)} }),
	`skills chain an additional time while at maximum frenzy charges`: modList{mod("ChainCountMax", Base, Num(1), &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"})},
	`attacks chain an additional time when in main hand`:              modList{modf("ChainCountMax", Base, Num(1), FlagAttack, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 1})},
	`attacks fire an additional projectile when in off hand`:          modList{modf("ProjectileCount", Base, Num(1), FlagAttack, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`projectiles chain \+([0-9]) times while you have phasing`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ChainCountMax", Base, Num(c.n(1)), FlagProjectile, KeywordNone, &CondTag{Var: "Phasing"})}
	}),
	`projectiles can chain from any number of additional targets in close range`: modList{modf("ChainCountMax", Base, Num(m_huge), FlagProjectile, KeywordNone, &CondTag{Var: "AtCloseRange"})},
	`projectiles split towards \+([0-9]) targets`:                                modFn(func(c caps) []*Mod { return []*Mod{mod("SplitCount", Base, Num(c.n(1)))} }),
	`adds an additional arrow`:                                                   modList{modf("ProjectileCount", Base, Num(1), FlagNone, KeywordArrow)},
	`([0-9]+) additional arrows`:                                                 modFn(func(c caps) []*Mod { return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagNone, KeywordArrow)} }),
	`bow attacks fire an additional arrow`:                                       modList{modf("ProjectileCount", Base, Num(1), FlagBow, KeywordNone)},
	`bow attacks fire ([0-9]+) additional arrows`:                                modFn(func(c caps) []*Mod { return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagNone, KeywordArrow)} }),
	`bow attacks fire ([0-9]+) additional arrows if you haven't cast dash recently`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagNone, KeywordArrow, &CondTag{Var: "CastDashRecently", Neg: true})}
	}),
	`wand attacks fire an additional projectile`: modList{modf("ProjectileCount", Base, Num(1), FlagWand, KeywordNone)},
	`skills fire an additional projectile`:       modList{mod("ProjectileCount", Base, Num(1))},
	`skills fire an additional projectile if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ProjectileCount", Base, Num(1), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "Item", Threshold: opt(c.n(1))})}
	}),
	`spells [hf][ai][vr]e an additional projectile`:        modList{modf("ProjectileCount", Base, Num(1), FlagSpell, KeywordNone)},
	`spells [hf][ai][vr]e ([0-9]+) additional projectiles`: modFn(func(c caps) []*Mod { return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagSpell, KeywordNone)} }),
	`attacks fire an additional projectile`:                modList{modf("ProjectileCount", Base, Num(1), FlagAttack, KeywordNone)},
	`attacks [hf][ai][vr]e ([0-9]+) additional projectiles`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagAttack, KeywordNone)}
	}),
	`attacks [hf][ai][vr]e ([0-9]+) additional projectiles? when in off hand`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagAttack, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 2})}
	}),
	`fire at most 1 projectile`:                              modList{flag("SingleProjectile")},
	`attacks have an additional projectile when in off hand`: modList{modf("ProjectileCount", Base, Num(1), FlagAttack, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`caustic arrow and scourge arrow fire ([0-9]+)% more projectiles`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ProjectileCount", More, Num(c.n(1)), nil, &SkillNameTag{SkillNameList: []string{"Caustic Arrow", "Scourge Arrow"}, IncludeTransfigured: true})}
	}),
	`essence drain and soulrend fire ([0-9]+) additional projectiles`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ProjectileCount", Base, Num(c.n(1)), nil, &SkillNameTag{SkillNameList: []string{"Essence Drain", "Soulrend"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% reduced essence drain and soulrend projectile speed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ProjectileSpeed", Inc, Num(-c.n(1)), nil, &SkillNameTag{SkillNameList: []string{"Essence Drain", "Soulrend"}, IncludeTransfigured: true})}
	}),
	`tornado shot fires an additional secondary projectile`: modList{mod("tornadoShotSecondaryProjectiles", Base, Num(1), nil, &SkillNameTag{SkillNameList: []string{"Tornado Shot"}})},
	`tornado shot fires 2 additional secondary projectiles`: modList{mod("tornadoShotSecondaryProjectiles", Base, Num(2), nil, &SkillNameTag{SkillNameList: []string{"Tornado Shot"}})},
	`projectiles pierce an additional target`:               modList{mod("PierceCount", Base, Num(1))},
	`projectiles pierce ([0-9]+) targets?`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("PierceCount", Base, Num(c.n(1)))} }),
	`projectiles pierce ([0-9]+) additional targets?`:       modFn(func(c caps) []*Mod { return []*Mod{mod("PierceCount", Base, Num(c.n(1)))} }),
	`projectiles pierce ([0-9]+) additional targets while you have phasing`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PierceCount", Base, Num(c.n(1)), &CondTag{Var: "Phasing"})}
	}),
	`projectiles pierce ([0-9]+) additional targets if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PierceCount", Base, c.v(1), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))})}
	}),
	`projectiles pierce all targets while you have phasing`: modList{flag("PierceAllTargets", &CondTag{Var: "Phasing"})},
	`projectiles pierce all burning enemies`:                modList{flag("PierceAllTargets", &CondTag{IsActor: true, Actor: "enemy", Var: "Burning"})},
	`arrows pierce an additional target`:                    modList{modf("PierceCount", Base, Num(1), FlagNone, KeywordArrow)},
	`arrows pierce ([0-9]+) additional targets`:             modFn(func(c caps) []*Mod { return []*Mod{modf("PierceCount", Base, Num(c.n(1)), FlagNone, KeywordArrow)} }),
	`arrows pierce one target`:                              modList{modf("PierceCount", Base, Num(1), FlagNone, KeywordArrow)},
	`arrows pierce ([0-9]+) targets?`:                       modFn(func(c caps) []*Mod { return []*Mod{modf("PierceCount", Base, Num(c.n(1)), FlagNone, KeywordArrow)} }),
	`always pierce with arrows`:                             modList{flagf("PierceAllTargets", FlagNone, KeywordArrow)},
	`arrows always pierce`:                                  modList{flagf("PierceAllTargets", FlagNone, KeywordArrow)},
	`arrows pierce all targets`:                             modList{flagf("PierceAllTargets", FlagNone, KeywordArrow)},
	`arrows that pierce cause bleeding`:                     modList{modf("BleedChance", Base, Num(100), FlagProjectile, KeywordArrow, &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)})},
	`arrows that pierce have ([0-9]+)% chance to [ic][na][fu][ls][ie]c?t? bleeding`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("BleedChance", Base, Num(c.n(1)), FlagProjectile, KeywordArrow, &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)})}
	}),
	`arrows that pierce deal ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagProjectile, KeywordArrow, &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)})}
	}),
	`projectiles gain ([0-9]+)% of non-chaos damage as extra chaos damage per chain`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("NonChaosDamageGainAsChaos", Base, Num(c.n(1)), FlagProjectile, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "Chain"})}
	}),
	`projectiles that have chained gain ([0-9]+)% of non-chaos damage as extra chaos damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("NonChaosDamageGainAsChaos", Base, Num(c.n(1)), FlagProjectile, KeywordNone, &StatTag{StatKind: TagStatThreshold, Stat: "Chain", Threshold: opt(1)})}
	}),
	`left ring slot: projectiles from spells cannot chain`:     modList{flagf("CannotChain", FlagSpell|FlagProjectile, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 1})},
	`left ring slot: projectiles from spells fork`:             modList{flagf("ForkOnce", FlagSpell|FlagProjectile, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 1}), modf("ForkCountMax", Base, Num(1), FlagSpell|FlagProjectile, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 1})},
	`right ring slot: projectiles from spells chain \+1 times`: modList{modf("ChainCountMax", Base, Num(1), FlagSpell|FlagProjectile, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`right ring slot: projectiles from spells cannot fork`:     modList{flagf("CannotFork", FlagSpell|FlagProjectile, KeywordNone, &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`projectiles from spells cannot pierce`:                    modList{flagf("CannotPierce", FlagSpell, KeywordNone)},
	// The chance to return is ignored; the number of returning Projectiles that hit is a config option
	`projectiles return to you`:                          modList{flag("ProjectilesReturn")},
	`projectiles have ([0-9]+)% chance to return to you`: modList{flag("ProjectilesReturn")},
	`projectiles fork`:                                   modList{flagf("ForkOnce", FlagProjectile, KeywordNone), modf("ForkCountMax", Base, Num(1), FlagProjectile, KeywordNone)},
	`projectiles from attacks fork`:                      modList{flagf("ForkOnce", FlagProjectile, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRangedAttack}), modf("ForkCountMax", Base, Num(1), FlagProjectile, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRangedAttack})},
	`projectiles from attacks fork an additional time`:   modList{flagf("ForkTwice", FlagProjectile, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRangedAttack}), modf("ForkCountMax", Base, Num(1), FlagProjectile, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRangedAttack})},
	`projectiles from attacks can fork ([0-9]+) additional times?`: modFn(func(c caps) []*Mod {
		return []*Mod{flagf("ForkTwice", FlagProjectile, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRangedAttack}), modf("ForkCountMax", Base, Num(c.n(1)), FlagProjectile, KeywordNone, &SkillTypeTag{SkillType: SkillTypeRangedAttack})}
	}),
	`([0-9]+)% increased critical strike chance with arrows that fork`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CritChance", Inc, Num(c.n(1)), FlagNone, KeywordArrow, &StatTag{StatKind: TagStatThreshold, Stat: "ForkRemaining", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(0), Upper: true})}
	}),
	`arrows that pierce have \+([0-9]+)% to critical strike multiplier`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CritMultiplier", Base, Num(c.n(1)), FlagNone, KeywordArrow, &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)})}
	}),
	`arrows pierce all targets after forking`:                                                             modList{flagf("PierceAllTargets", FlagNone, KeywordArrow, &StatTag{StatKind: TagStatThreshold, Stat: "ForkedCount", Threshold: opt(1)})},
	`modifiers to number of projectiles instead apply to the number of targets projectiles split towards`: modList{flag("NoAdditionalProjectiles"), flag("AdditionalProjectilesAddSplitsInstead")},
	`modifiers to number of projectiles do not apply to fireball and rolling magma`:                       modList{flag("NoAdditionalProjectiles", &SkillNameTag{SkillNameList: []string{"Fireball", "Rolling Magma"}})},
	`attack skills fire an additional projectile while wielding a claw or dagger`:                         modList{modf("ProjectileCount", Base, Num(1), FlagAttack, KeywordNone, &ModFlagOrTag{ModFlags: FlagClaw | FlagDagger})},
	`skills fire ([0-9]+) additional projectiles for 4 seconds after you consume a total of 12 steel shards`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ProjectileCount", Base, Num(c.n(1)), &CondTag{Var: "Consumed12SteelShardsRecently"})}
	}),
	`bow attacks sacrifice a random damageable minion to fire ([0-9]+) additional arrows?`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ProjectileCount", Base, Num(c.n(1)), FlagNone, KeywordArrow, &CondTag{Var: "SacrificeMinionOnAttack"}, &CondTag{Var: "HaveDamageableMinion"}, &SkillTypeTag{SkillType: SkillTypeTriggered, Neg: true}, &SkillTypeTag{SkillType: SkillTypeSummonsTotem, Neg: true})}
	}),
	`non-projectile chaining lightning skills chain \+([0-9]+) times`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChainCountMax", Base, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeProjectile, Neg: true}, &SkillTypeTag{SkillType: SkillTypeChains}, &SkillTypeTag{SkillType: SkillTypeLightning})}
	}),
	`arrows gain damage as they travel farther, dealing up to ([0-9]+)% increased damage with hits to targets`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagHit, KeywordArrow, &DistanceRampTag{Ramp: Pairs{{35, 0}, {70, 1}}})}
	}),
	`arrows gain critical strike chance as they travel farther, up to ([0-9]+)% increased critical strike chance`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CritChance", Inc, Num(c.n(1)), FlagNone, KeywordArrow, &DistanceRampTag{Ramp: Pairs{{35, 0}, {70, 1}}})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits to targets at the start of their movement, reducing to ([0-9]+)% as they travel farther`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagHit|FlagProjectile, KeywordNone, &DistanceRampTag{Ramp: Pairs{{35, 1}, {70, 0}}})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits and ailments for each time they have chained`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &StatTag{StatKind: TagPerStat, Stat: "Chain"}, &SkillTypeTag{SkillType: SkillTypeProjectile})}
	}),
	`projectiles deal ([0-9]+)% increased damage with hits and ailments for each enemy pierced`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagNone, KeywordHit|KeywordAilment, &StatTag{StatKind: TagPerStat, Stat: "PiercedCount"}, &SkillTypeTag{SkillType: SkillTypeProjectile})}
	}),
	`([0-9]+)% increased bonuses gained from equipped (.+)`: modFn(func(c caps) []*Mod { return []*Mod{mod("EffectOfBonusesFrom"+firstToUpper(c.s(2)), Inc, Num(c.n(1)))} }),
	// Strike Skills
	`non-vaal strike skills target ([0-9]+) additional nearby enem[yi]e?s?`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AdditionalStrikeTarget", Base, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeMeleeSingleTarget}, &SkillTypeTag{SkillType: SkillTypeVaal, Neg: true})}
	}),
	// Leech/Gain on Hit/Kill
	`cannot leech life`:                                                                       modList{flag("CannotLeechLife")},
	`cannot leech mana`:                                                                       modList{flag("CannotLeechMana")},
	`cannot leech when on low life`:                                                           modList{flag("CannotLeechLife", &CondTag{Var: "LowLife"}), flag("CannotLeechMana", &CondTag{Var: "LowLife"})},
	`cannot leech life from critical strikes`:                                                 modList{flag("CannotLeechLife", &CondTag{Var: "CriticalStrike"})},
	`leech applies instantly on critical strike`:                                              modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"}), mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"}), mod("InstantEnergyShieldLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"})},
	`gain life and mana from leech instantly on critical strike`:                              modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"}), mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"}), mod("InstantEnergyShieldLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"})},
	`leech applies instantly during f?l?a?s?k? ?effect`:                                       modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "UsingFlask"}), mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "UsingFlask"}), mod("InstantEnergyShieldLeech", Base, Num(100), &CondTag{Var: "UsingFlask"})},
	`gain life and mana from leech instantly during f?l?a?s?k? ?effect`:                       modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "UsingFlask"}), mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "UsingFlask"})},
	`life and mana leech are instant during f?l?a?s?k? ?effect`:                               modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "UsingFlask"}), mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "UsingFlask"})},
	`life and mana leech from critical strikes are instant`:                                   modList{mod("InstantLifeLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"}), mod("InstantManaLeech", Base, Num(100), &CondTag{Var: "CriticalStrike"})},
	`with 5 corrupted items equipped: life leech recovers based on your chaos damage instead`: modList{flag("LifeLeechBasedOnChaosDamage", &MultiplierTag{IsThreshold: true, Var: "CorruptedItem", Threshold: opt(5)})},
	`you have vaal pact if you've dealt a critical strike recently`:                           modList{mod("Keystone", List, Str("Vaal Pact"), &CondTag{Var: "CritRecently"})},
	`you have vaal pact while at maximum endurance charges`:                                   modList{mod("Keystone", List, Str("Vaal Pact"), &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMax"})},
	`you have vaal pact while focus?sed`:                                                      modList{mod("Keystone", List, Str("Vaal Pact"), &CondTag{Var: "Focused"})},
	`gain ([0-9]+) energy shield for each enemy you hit which is affected by a spider's web`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("EnergyShieldOnHit", Base, Num(c.n(1)), FlagHit, KeywordNone, &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "Spider's WebStack", Threshold: opt(1)})}
	}),
	`([0-9]+)% chance to gain ([0-9]+) life on hit with attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeOnHit", Base, Num(c.n(2)*c.n(1)/100), FlagAttack|FlagHit, KeywordNone, &CondTag{Var: "AverageResourceGain"}), modf("LifeOnHit", Base, Num(c.n(2)), FlagAttack|FlagHit, KeywordNone, &CondTag{Var: "MaxResourceGain"})}
	}),
	`([0-9]+) life gained for each cursed enemy hit by your attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeOnHit", Base, Num(c.n(1)), FlagAttack|FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`gain ([0-9]+) life per cursed enemy hit with attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeOnHit", Base, Num(c.n(1)), FlagAttack|FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`gain ([0-9]+) life for each ignited enemy hit with attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeOnHit", Base, Num(c.n(1)), FlagAttack|FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"})}
	}),
	`([0-9]+) mana gained for each cursed enemy hit by your attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ManaOnHit", Base, Num(c.n(1)), FlagAttack|FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`gain ([0-9]+) mana per cursed enemy hit with attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ManaOnHit", Base, Num(c.n(1)), FlagAttack|FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`gain ([0-9]+) life per blinded enemy hit with this weapon`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LifeOnHit", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{IsActor: true, Actor: "enemy", Var: "Blinded"}, &CondTag{Var: "{Hand}Attack"})}
	}),
	`recover ([0-9]+)% of life on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}
	}),
	`recover ([0-9]+)% of life on kill for each different type of mastery you have allocated`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{Var: "AllocatedMasteryType"})}
	}),
	`recover ([0-9]+)% of life on killing a poisoned enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "PoisonStack", Threshold: opt(1)})}
	}),
	`recover ([0-9]+)% of life on killing a chilled enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"})}
	}),
	`recover ([0-9]+)% of life when you kill a cursed enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`recover ([0-9]+)% of life per withered debuff on each enemy you kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{Var: "WitheredStack", Actor: "enemy", Limit: opt(15)})}
	}),
	`minions recover ([0-9]+)% of life on killing a poisoned enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "PoisonStack", Threshold: opt(1)})})}
	}),
	`minions recover ([0-9]+)% of their life when they block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("LifeOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})})}
	}),
	`recover ([0-9]+)% of mana when you kill a cursed enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`recover ([0-9]+)% of energy shield when you kill a cursed enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`recover ([0-9]+)% of life on kill if you've spent life recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{IsThreshold: true, Var: "LifeSpentRecently", Threshold: opt(1)})}
	}),
	`([0-9]+)% chance to recover all life when you kill an enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &CondTag{Var: "AverageResourceGain"}), mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(100)}, &CondTag{Var: "MaxResourceGain"})}
	}),
	`lose ([0-9]+)% of life on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(-1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}
	}),
	`\+([0-9]+) life gained on killing ignited enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"})}
	}),
	`gain ([0-9]+) life per ignited enemy killed`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"})}
	}),
	`recover ([0-9]+)% of mana on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))})}
	}),
	`recover ([0-9]+)% of mana on kill while you have a tincture active`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &CondTag{Var: "UsingTincture"})}
	}),
	`recover ([0-9]+)% of mana on kill for each different type of mastery you have allocated`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &MultiplierTag{Var: "AllocatedMasteryType"})}
	}),
	`lose ([0-9]+)% of mana on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(-1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))})}
	}),
	`\+([0-9]+) mana gained on killing a frozen enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"})}
	}),
	`recover ([0-9]+)% of energy shield on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))})}
	}),
	`recover ([0-9]+)% of energy shield on kill for each different type of mastery you have allocated`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &MultiplierTag{Var: "AllocatedMasteryType"})}
	}),
	`lose ([0-9]+)% of energy shield on kill`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(-1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))})}
	}),
	`\+([0-9]+) energy shield gained on killing a shocked enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"})}
	}),
	`\+([0-9]+) energy shield gained on kill per level`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(c.n(1)), &MultiplierTag{Var: "Level"})}
	}),
	// Defences
	`chaos damage t?a?k?e?n? ?does not bypass energy shield`:                                   modList{flag("ChaosNotBypassEnergyShield")},
	`([0-9]+)% of chaos damage t?a?k?e?n? ?does not bypass energy shield`:                      modFn(func(c caps) []*Mod { return []*Mod{mod("ChaosEnergyShieldBypass", Base, Num(-c.n(1)))} }),
	`chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life`:             modList{flag("ChaosNotBypassEnergyShield", &CondTag{VarList: []string{"LowLife"}, Neg: true})},
	`chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life or low mana`: modList{flag("ChaosNotBypassEnergyShield", &CondTag{VarList: []string{"LowLife", "LowMana"}, Neg: true})},
	`chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low mana`:             modList{flag("ChaosNotBypassEnergyShield", &CondTag{VarList: []string{"LowMana"}, Neg: true})},
	`chaos damage is taken from mana before life`:                                              modList{mod("ChaosDamageTakenFromManaBeforeLife", Base, Num(100))},
	`([0-9]+)% of physical damage is taken from mana before life`:                              modFn(func(c caps) []*Mod { return []*Mod{mod("PhysicalDamageTakenFromManaBeforeLife", Base, Num(c.n(1)))} }),
	`minions take ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)))})}
	}),
	`minions take ([0-9]+)% reduced damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("DamageTaken", Inc, Num(-c.n(1)))})}
	}),
	`you and your minions take ([0-9]+)% reduced reflected ([0-9a-zA-Z]+) damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(2))+"ReflectedDamageTaken", Inc, Num(-c.n(1)), &GlobalEffectTag{EffectType: "Global", Unscalable: true}), mod("MinionModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"ReflectedDamageTaken", Inc, Num(-c.n(1)), &GlobalEffectTag{EffectType: "Global", Unscalable: true})})}
	}),
	`([0-9]+)% reduced reflected damage taken during effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReflectedDamageTaken", Inc, Num(-c.n(1)), &GlobalEffectTag{EffectType: "Global", Unscalable: true})}
	}),
	`you and your minions take ([0-9]+)% reduced reflected damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReflectedDamageTaken", Inc, Num(-c.n(1)), &GlobalEffectTag{EffectType: "Global", Unscalable: true}), mod("MinionModifier", List, ModRef{Mod: mod("ReflectedDamageTaken", Inc, Num(-c.n(1)), &GlobalEffectTag{EffectType: "Global", Unscalable: true})})}
	}),
	`damage cannot be reflected`: modList{mod("ReflectedDamageTaken", More, Num(-100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`immune to reflected damage`: modList{mod("ReflectedDamageTaken", More, Num(-100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`cannot take reflected ([a-zA-Z]+) damage if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod(firstToUpper(c.s(1))+"ReflectedDamageTaken", More, Num(-100), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))})}
	}),
	`you have mind over matter while at maximum power charges`:            modList{mod("Keystone", List, Str("Mind Over Matter"), &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", ThresholdStat: "PowerChargesMax"})},
	`cannot evade enemy attacks`:                                          modList{flag("CannotEvade")},
	`attacks cannot hit you`:                                              modList{flag("AlwaysEvade")},
	`attacks against you always hit`:                                      modList{flag("CannotEvade")},
	`cannot block`:                                                        modList{flag("CannotBlockAttacks"), flag("CannotBlockSpells")},
	`cannot block while you have no energy shield`:                        modList{flag("CannotBlockAttacks", &CondTag{Var: "HaveEnergyShield", Neg: true}), flag("CannotBlockSpells", &CondTag{Var: "HaveEnergyShield", Neg: true})},
	`cannot block attacks`:                                                modList{flag("CannotBlockAttacks")},
	`cannot block attack damage`:                                          modList{flag("CannotBlockAttacks")},
	`cannot block spells`:                                                 modList{flag("CannotBlockSpells")},
	`cannot block spell damage`:                                           modList{flag("CannotBlockSpells")},
	`monsters cannot block your attacks`:                                  modList{mod("EnemyModifier", List, ModRef{Mod: flag("CannotBlockAttacks")})},
	`damage t?a?k?e?n? from blocked hits cannot bypass energy shield`:     modList{flag("BlockedDamageDoesntBypassES", &CondTag{Var: "EVBypass", Neg: true})},
	`damage t?a?k?e?n? from unblocked hits always bypasses energy shield`: modList{flag("UnblockedDamageDoesBypassES", &CondTag{Var: "EVBypass", Neg: true})},
	`recover ([0-9]+) life when you block`:                                modFn(func(c caps) []*Mod { return []*Mod{mod("LifeOnBlock", Base, Num(c.n(1)))} }),
	`recover ([0-9]+) energy shield when you block spell damage`:          modFn(func(c caps) []*Mod { return []*Mod{mod("EnergyShieldOnSpellBlock", Base, Num(c.n(1)))} }),
	`recover ([0-9]+) energy shield when you suppress spell damage`:       modFn(func(c caps) []*Mod { return []*Mod{mod("EnergyShieldOnSuppress", Base, Num(c.n(1)))} }),
	`recover ([0-9]+) life when you suppress spell damage`:                modFn(func(c caps) []*Mod { return []*Mod{mod("LifeOnSuppress", Base, Num(c.n(1)))} }),
	`recover ([0-9]+)% of life when you block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}
	}),
	`recover ([0-9]+)% of life when you block attack damage while wielding a staff`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &CondTag{Var: "UsingStaff"})}
	}),
	`recover ([0-9]+)% of your maximum mana when you block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))})}
	}),
	`recover ([0-9]+)% of energy shield when you block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))})}
	}),
	`recover ([0-9]+)% of energy shield when you block spell damage while wielding a staff`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnSpellBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{Var: "UsingStaff"})}
	}),
	`replenishes energy shield by ([0-9]+)% of armour when you block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Armour", Percent: opt(c.n(1))})}
	}),
	`recover energy shield equal to ([0-9]+)% of armour when you block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Armour", Percent: opt(c.n(1))})}
	}),
	`recover energy shield equal to ([0-9]+)% of evasion rating when you block`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnBlock", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Evasion", Percent: opt(c.n(1))})}
	}),
	`([0-9]+)% of damage taken while affected by clarity recouped as mana`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaRecoup", Base, Num(c.n(1)), &CondTag{Var: "AffectedByClarity"})}
	}),
	`([0-9]+)% of damage taken while frozen recouped as life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRecoup", Base, Num(c.n(1)), &CondTag{Var: "Frozen"})}
	}),
	`recoup effects instead occur over 3 seconds`:                            modList{flag("3SecondRecoup")},
	`life recoup effects instead occur over 3 seconds`:                       modList{flag("3SecondLifeRecoup")},
	`recoup energy shield instead of life`:                                   modList{flag("EnergyShieldRecoupInsteadOfLife")},
	`damage taken recouped as ([a-zA-Z]+) is also recouped as energy shield`: modFn(func(c caps) []*Mod { return []*Mod{flag("Add" + firstToUpper(c.s(1)) + "RecoupToEnergyShieldRecoup")} }),
	`([0-9.]+)% of physical damage prevented from hits in the past ([0-9]+) seconds is regenerated as life per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageMitigatedLifePseudoRecoup", Base, Num(c.n(1)*c.n(2))), mod("PhysicalDamageMitigatedLifePseudoRecoupDuration", Base, c.v(2))}
	}),
	`([0-9.]+)% of physical damage prevented from hits recently is regenerated as energy shield per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", Base, Num(c.n(1)*4))}
	}),
	`([0-9.]+)% of physical damage prevented recently is regenerated as energy shield per second if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", Base, Num(c.n(1)*4), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))})}
	}),
	`cannot leech or regenerate mana`:                                 modList{flag("NoManaRegen"), flag("CannotLeechMana")},
	`y?o?u? ?cannot recharge energy shield`:                           modList{flag("NoEnergyShieldRecharge")},
	`cannot recharge or regenerate energy shield`:                     modList{flag("NoEnergyShieldRecharge"), flag("NoEnergyShieldRegen")},
	`left ring slot: you cannot recharge or regenerate energy shield`: modList{flag("NoEnergyShieldRecharge", &SlotTag{SlotKind: TagSlotNumber, Num: 1}), flag("NoEnergyShieldRegen", &SlotTag{SlotKind: TagSlotNumber, Num: 1})},
	`cannot gain energy shield`:                                       modList{flag("CannotGainEnergyShield")},
	`cannot gain life`:                                                modList{flag("CannotGainLife")},
	`cannot gain mana`:                                                modList{flag("CannotGainMana")},
	`cannot recover life other than from leech`:                       modList{flag("CannotRecoverLifeOutsideLeech"), flag("NoLifeRegen")},
	`cannot gain energy shield during f?l?a?s?k? ?effect`:             modList{flag("CannotGainEnergyShield", &CondTag{Var: "UsingFlask"})},
	`cannot gain life during f?l?a?s?k? ?effect`:                      modList{flag("CannotGainLife", &CondTag{Var: "UsingFlask"})},
	`cannot gain mana during f?l?a?s?k? ?effect`:                      modList{flag("CannotGainMana", &CondTag{Var: "UsingFlask"})},
	`life that would be lost by taking damage is instead reserved`:    modList{flag("DamageInsteadReservesLife")},
	`you have no armour or energy shield`:                             modList{mod("Armour", More, Num(-100)), mod("EnergyShield", More, Num(-100))},
	`you have no armour or maximum energy shield`:                     modList{mod("Armour", More, Num(-100)), mod("EnergyShield", More, Num(-100))},
	`defences are zero`:                                               modList{mod("Armour", More, Num(-100)), mod("EnergyShield", More, Num(-100)), mod("Evasion", More, Num(-100)), mod("Ward", More, Num(-100))},
	`you have no intelligence`:                                        modList{mod("Int", More, Num(-100))},
	`you have no dexterity`:                                           modList{mod("Dex", More, Num(-100))},
	`you have no strength`:                                            modList{mod("Str", More, Num(-100))},
	`physical damage reduction is zero`:                               modList{mod("PhysicalDamageReduction", Override, Num(0)), flag("ArmourDoesNotApplyToPhysicalDamageTaken")},
	`elemental resistances are zero`:                                  modList{mod("FireResist", Override, Num(0)), mod("ColdResist", Override, Num(0)), mod("LightningResist", Override, Num(0))},
	`chaos resistance is zero`:                                        modList{mod("ChaosResist", Override, Num(0))},
	`nearby enemies' chaos resistance is ([0-9]+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("ChaosResist", Override, Num(c.n(1)))})}
	}),
	`your maximum resistances are ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireResistMax", Override, Num(c.n(1))), mod("ColdResistMax", Override, Num(c.n(1))), mod("LightningResistMax", Override, Num(c.n(1))), mod("ChaosResistMax", Override, Num(c.n(1)))}
	}),
	`fire resistance is ([0-9]+)%`:      modFn(func(c caps) []*Mod { return []*Mod{mod("FireResist", Override, Num(c.n(1)))} }),
	`cold resistance is ([0-9]+)%`:      modFn(func(c caps) []*Mod { return []*Mod{mod("ColdResist", Override, Num(c.n(1)))} }),
	`lightning resistance is ([0-9]+)%`: modFn(func(c caps) []*Mod { return []*Mod{mod("LightningResist", Override, Num(c.n(1)))} }),
	`elemental resistances are capped by your highest maximum elemental resistance instead`: modList{flag("ElementalResistMaxIsHighestResistMax")},
	`nearby enemies have ([0-9]+)% increased fire and cold resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("FireResist", Inc, Num(c.n(1)))}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdResist", Inc, Num(c.n(1)))})}
	}),
	`nearby enemies are blinded while physical aegis is not depleted`:    modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Blinded")}, &CondTag{Var: "PhysicalAegisDepleted", Neg: true})},
	`maximum energy shield is increased by chance to block spell damage`: modList{flag("EnergyShieldIncreasedByChanceToBlockSpellDamage")},
	`maximum energy shield is increased by chaos resistance`:             modList{flag("EnergyShieldIncreasedByChaosResistance")},
	`armour is increased by uncapped fire resistance`:                    modList{flag("ArmourIncreasedByUncappedFireRes")},
	`armour is increased by overcapped fire resistance`:                  modList{flag("ArmourIncreasedByOvercappedFireRes")},
	`minion life is increased by t?h?e?i?r? ?overcapped fire resistance`: modList{mod("MinionModifier", List, ModRef{Mod: mod("Life", Inc, Num(1), &StatTag{StatKind: TagPerStat, Stat: "FireResistOverCap", Div: opt(1)})})},
	`totem life is increased by t?h?e?i?r? ?overcapped fire resistance`:  modList{mod("TotemLife", Inc, Num(1), &StatTag{StatKind: TagPerStat, Stat: "TotemFireResistOverCap", Div: opt(1)})},
	`evasion rating is increased by uncapped cold resistance`:            modList{flag("EvasionRatingIncreasedByUncappedColdRes")},
	`evasion rating is increased by overcapped cold resistance`:          modList{flag("EvasionRatingIncreasedByOvercappedColdRes")},
	`reflects ([0-9]+) physical damage to melee attackers`:               modList{},
	`ignore all movement penalties from armour`:                          modList{flag("Condition:IgnoreMovementPenalties")},
	`gain armour equal to your reserved mana`:                            modList{mod("Armour", Base, Num(1), &StatTag{StatKind: TagPerStat, Stat: "ManaReserved", Div: opt(1)})},
	`gain ward instead of ([0-9]+)% of armour and evasion rating from equipped body armour`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ConvertBodyArmourArmourEvasionToWard"), mod("BodyArmourArmourEvasionToWardPercent", Base, Num(c.n(1)))}
	}),
	`([0-9]+)% increased armour per ([0-9]+) reserved mana`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Armour", Inc, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ManaReserved", Div: opt(c.n(2))})}
	}),
	`cannot be stunned`:                                  modList{flag("StunImmune")},
	`cannot be stunned while bleeding`:                   modList{flag("StunImmune", &CondTag{Var: "Bleeding"})},
	`cannot be stunned when on low life`:                 modList{flag("StunImmune", &CondTag{Var: "LowLife"})},
	`cannot be stunned if you haven't been hit recently`: modList{flag("StunImmune", &CondTag{Var: "BeenHitRecently", Neg: true})},
	`cannot be stunned if you have at least ([0-9]+) crab barriers`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("StunImmune", &StatTag{StatKind: TagStatThreshold, Stat: "CrabBarriers", Threshold: opt(c.n(1))})}
	}),
	`cannot be stunned if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("StunImmune", &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "Item", Threshold: opt(c.n(1))})}
	}),
	`cannot be blinded`:                             modList{flag("Condition:CannotBeBlinded"), flag("BlindImmune")},
	`cannot be blinded while affected by precision`: modList{flag("Condition:CannotBeBlinded", &CondTag{Var: "AffectedByPrecision"}), flag("BlindImmune", &CondTag{Var: "AffectedByPrecision"})},
	`cannot be knocked back`:                        modList{flag("KnockbackImmune")},
	`cannot be shocked`:                             modList{flag("ShockImmune")},
	`immun[ei]t?y? to shock`:                        modList{flag("ShockImmune")},
	`cannot be frozen`:                              modList{flag("FreezeImmune")},
	`cannot be frozen if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("FreezeImmune", &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "Item", Threshold: opt(c.n(1))})}
	}),
	`immun[ei]t?y? to freeze`:                                                                               modList{flag("FreezeImmune")},
	`cannot be chilled`:                                                                                     modList{flag("ChillImmune")},
	`immun[ei]t?y? to chill`:                                                                                modList{flag("ChillImmune")},
	`cannot be ignited`:                                                                                     modList{flag("IgniteImmune")},
	`immun[ei]t?y? to ignite`:                                                                               modList{flag("IgniteImmune")},
	`immune to ignite while affected by purity of fire`:                                                     modList{flag("IgniteImmune", &CondTag{Var: "AffectedByPurityofFire"})},
	`immune to freeze while affected by purity of ice`:                                                      modList{flag("FreezeImmune", &CondTag{Var: "AffectedByPurityofIce"})},
	`immune to shock while affected by purity of lightning`:                                                 modList{flag("ShockImmune", &CondTag{Var: "AffectedByPurityofLightning"})},
	`critical strikes against you do not inherently inflict elemental ailments`:                             modList{flag("CritsOnYouDontAlwaysApplyElementalAilments")},
	`cannot be ignited while at maximum endurance charges`:                                                  modList{flag("IgniteImmune", &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMax"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`grants immunity to ignite for ([0-9]+) seconds if used while ignited`:                                  modList{flag("IgniteImmune", &CondTag{Var: "UsingFlask"})},
	`grants immunity to bleeding for ([0-9]+) seconds if used while bleeding`:                               modList{flag("BleedImmune", &CondTag{Var: "UsingFlask"})},
	`grants immunity to corrupted blood for ([0-9]+) seconds if used while affected by corrupted blood`:     modList{flag("CorruptedBloodImmune", &CondTag{Var: "UsingFlask"})},
	`grants immunity to poison for ([0-9]+) seconds if used while poisoned`:                                 modList{flag("PoisonImmune", &CondTag{Var: "UsingFlask"})},
	`grants immunity to freeze for ([0-9]+) seconds if used while frozen`:                                   modList{flag("FreezeImmune", &CondTag{Var: "UsingFlask"})},
	`grants immunity to chill for ([0-9]+) seconds if used while chilled`:                                   modList{flag("ChillImmune", &CondTag{Var: "UsingFlask"})},
	`grants immunity to shock for ([0-9]+) seconds if used while shocked`:                                   modList{flag("ShockImmune", &CondTag{Var: "UsingFlask"})},
	`cannot be chilled while at maximum frenzy charges`:                                                     modList{flag("ChillImmune", &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"})},
	`cannot be shocked while at maximum power charges`:                                                      modList{flag("ShockImmune", &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", ThresholdStat: "PowerChargesMax"})},
	`you cannot be shocked while at maximum endurance charges`:                                              modList{flag("ShockImmune", &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMax"})},
	`you cannot be shocked while chilled`:                                                                   modList{flag("ShockImmune", &CondTag{Var: "Chilled"})},
	`you cannot be shocked while frozen`:                                                                    modList{flag("ShockImmune", &CondTag{Var: "Frozen"})},
	`cannot be shocked while chilled`:                                                                       modList{flag("ShockImmune", &CondTag{Var: "Chilled"})},
	`cannot be shocked if intelligence is higher than strength`:                                             modList{flag("ShockImmune", &CondTag{Var: "IntHigherThanStr"})},
	`cannot be frozen if dexterity is higher than intelligence`:                                             modList{flag("FreezeImmune", &CondTag{Var: "DexHigherThanInt"})},
	`cannot be frozen if energy shield recharge has started recently`:                                       modList{flag("FreezeImmune", &CondTag{Var: "EnergyShieldRechargeRecently"})},
	`cannot be ignited if strength is higher than dexterity`:                                                modList{flag("IgniteImmune", &CondTag{Var: "StrHigherThanDex"})},
	`cannot be chilled while burning`:                                                                       modList{flag("ChillImmune", &CondTag{Var: "Burning"})},
	`cannot be chilled while you have onslaught`:                                                            modList{flag("ChillImmune", &CondTag{Var: "Onslaught"})},
	`cannot be chilled during onslaught`:                                                                    modList{flag("ChillImmune", &CondTag{Var: "Onslaught"})},
	`cannot be frozen or chilled if you've used a fire skill recently`:                                      modList{flag("FreezeImmune", &CondTag{Var: "UsedFireSkillRecently"}), flag("ChillImmune", &CondTag{Var: "UsedFireSkillRecently"})},
	`cannot be inflicted with bleeding`:                                                                     modList{flag("BleedImmune")},
	`bleeding cannot be inflicted on you`:                                                                   modList{flag("BleedImmune")},
	`you are immune to bleeding`:                                                                            modList{flag("BleedImmune")},
	`immune to bleeding if equipped helmet has higher armour than evasion rating`:                           modList{flag("BleedImmune", &CondTag{Var: "HelmetArmourHigherThanEvasion"})},
	`immune to poison if equipped helmet has higher evasion rating than armour`:                             modList{flag("PoisonImmune", &CondTag{Var: "HelmetEvasionHigherThanArmour"})},
	`immun[ei]t?y? to bleeding and corrupted blood during f?l?a?s?k? ?effect`:                               modList{flag("BleedImmune", &CondTag{Var: "UsingFlask"}), flag("CorruptedBloodImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to poison`:                                                                               modList{flag("PoisonImmune")},
	`cannot be poisoned`:                                                                                    modList{flag("PoisonImmune")},
	`cannot be poisoned while bleeding`:                                                                     modList{flag("PoisonImmune", &CondTag{Var: "Bleeding"})},
	`immun[ei]t?y? to poison during f?l?a?s?k? ?effect`:                                                     modList{flag("PoisonImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to shock during f?l?a?s?k? ?effect`:                                                      modList{flag("ShockImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to freeze and chill during f?l?a?s?k? ?effect`:                                           modList{flag("FreezeImmune", &CondTag{Var: "UsingFlask"}), flag("ChillImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to freeze and chill while ignited`:                                                       modList{flag("FreezeImmune", &CondTag{Var: "Ignited"}), flag("ChillImmune", &CondTag{Var: "Ignited"})},
	`immun[ei]t?y? to ignite during f?l?a?s?k? ?effect`:                                                     modList{flag("IgniteImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to bleeding during f?l?a?s?k? ?effect`:                                                   modList{flag("BleedImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to curses during f?l?a?s?k? ?effect`:                                                     modList{flag("CurseImmune", &CondTag{Var: "UsingFlask"})},
	`immun[ei]t?y? to curses while channelling`:                                                             modList{flag("CurseImmune", &CondTag{Var: "Channelling"})},
	`when you kill an enemy affected by a non-aura hex, become immune to curses for remaining hex duration`: modList{flag("Condition:CanBeCurseImmune")},
	`when you kill an enemy cursed with a non-aura hex, become immune to curses for remaining hex duration`: modList{flag("Condition:CanBeCurseImmune")},
	`immun[ei]t?y? to freeze, chill, curses and stuns during f?l?a?s?k? ?effect`:                            modList{flag("FreezeImmune", &CondTag{Var: "UsingFlask"}), flag("ChillImmune", &CondTag{Var: "UsingFlask"}), flag("CurseImmune", &CondTag{Var: "UsingFlask"}), flag("StunImmune", &CondTag{Var: "UsingFlask"})},
	// This mod doesn't work the way it should. It prevents self-chill among other issues.
	// Since we don't currently really do anything with enemy ailment infliction, this should probably be removed
	// ["cursed enemies cannot inflict elemental ailments on you"] = {
	// mod("AvoidElementalAilments", Base, 100, { type = "ActorCondition", actor = "enemy", var = "Cursed" }, { type = "GlobalEffect", effectType = "Global", unscalable = true }),
	// },
	`enemies inflict elemental ailments on you instead of nearby allies`: modList{mod("ExtraAura", List, ModRef{Mod: flag("ElementalAilmentImmune"), OnlyAllies: true})},
	`unaffected by curses`:                                                  modList{mod("CurseEffectOnSelf", More, Num(-100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`you are immune to curses`:                                              modList{flag("CurseImmune")},
	`unaffected by curses while affected by zealotry`:                       modList{mod("CurseEffectOnSelf", More, Num(-100), &CondTag{Var: "AffectedByZealotry"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by vulnerability while affected by determination`:           modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Vulnerability"}, &CondTag{Var: "AffectedByDetermination"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by enfeeble while affected by grace`:                        modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Enfeeble"}, &CondTag{Var: "AffectedByGrace"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by temporal chains while affected by haste`:                 modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Temporal Chains"}, &CondTag{Var: "AffectedByHaste"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by elemental weakness while affected by purity of elements`: modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Elemental Weakness"}, &CondTag{Var: "AffectedByPurityofElements"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by flammability while affected by purity of fire`:           modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Flammability"}, &CondTag{Var: "AffectedByPurityofFire"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by frostbite while affected by purity of ice`:               modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Frostbite"}, &CondTag{Var: "AffectedByPurityofIce"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by conductivity while affected by purity of lightning`:      modList{mod("CurseEffectOnSelf", More, Num(-100), &SkillNameTag{SkillName: "Conductivity"}, &CondTag{Var: "AffectedByPurityofLightning"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`immun[ei]t?y? to curses while you have at least ([0-9]+) rage`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("CurseImmune", &MultiplierTag{IsThreshold: true, Var: "Rage", Threshold: opt(c.n(1))})}
	}),
	`you cannot be cursed with silence`:                    modList{flag("SilenceImmune")},
	`unaffected by ignite`:                                 modList{mod("SelfIgniteEffect", More, Num(-100))},
	`unaffected by chill`:                                  modList{mod("SelfChillEffect", More, Num(-100))},
	`unaffected by chill while leeching mana`:              modList{mod("SelfChillEffect", More, Num(-100), &CondTag{Var: "LeechingMana"})},
	`unaffected by chill while channelling`:                modList{mod("SelfChillEffect", More, Num(-100), &CondTag{Var: "Channelling"})},
	`unaffected by freeze`:                                 modList{mod("SelfFreezeEffect", More, Num(-100))},
	`unaffected by shock`:                                  modList{mod("SelfShockEffect", More, Num(-100))},
	`unaffected by shock while leeching energy shield`:     modList{mod("SelfShockEffect", More, Num(-100), &CondTag{Var: "LeechingEnergyShield"})},
	`unaffected by shock while channelling`:                modList{mod("SelfShockEffect", More, Num(-100), &CondTag{Var: "Channelling"})},
	`unaffected by scorch`:                                 modList{mod("SelfScorchEffect", More, Num(-100))},
	`unaffected by brittle`:                                modList{mod("SelfBrittleEffect", More, Num(-100))},
	`unaffected by sap`:                                    modList{mod("SelfSapEffect", More, Num(-100))},
	`unaffected by damaging ailments`:                      modList{mod("SelfBleedEffect", More, Num(-100)), mod("SelfIgniteEffect", More, Num(-100)), mod("SelfPoisonEffect", More, Num(-100))},
	`unaffected by bleeding while affected by malevolence`: modList{mod("SelfBleedEffect", More, Num(-100), &CondTag{Var: "AffectedByMalevolence"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by poison while affected by malevolence`:   modList{mod("SelfPoisonEffect", More, Num(-100), &CondTag{Var: "AffectedByMalevolence"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`unaffected by (.+) if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Self"+firstToUpper(c.s(1))+"Effect", More, Num(-100), &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(3)) + "Item", Threshold: opt(c.n(2))})}
	}),
	`the effect of chill on you is reversed`:                             modList{flag("SelfChillEffectIsReversed"), mods("Dummy", Dummy, Num(1), "", &CondTag{Var: "Chilled"})},
	`your movement speed is ([0-9]+)% of its base value`:                 modFn(func(c caps) []*Mod { return []*Mod{mod("MovementSpeed", Override, Num(c.n(1)/100))} }),
	`action speed cannot be modified to below ([0-9]+)% base value`:      modFn(func(c caps) []*Mod { return []*Mod{mod("MinimumActionSpeed", Max, Num(c.n(1)))} }),
	`action speed cannot be modified to below base value while ignited`:  modList{mod("MinimumActionSpeed", Max, Num(100), &CondTag{Var: "Ignited"})},
	`nearby allies' action speed cannot be modified to below base value`: modList{mod("ExtraAura", List, ModRef{Mod: mod("MinimumActionSpeed", Max, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true}), OnlyAllies: true})},
	`armour also applies to lightning damage taken from hits`:            modList{mod("ArmourAppliesToLightningDamageTaken", Base, Num(100))},
	`lightning resistance does not affect lightning damage taken`:        modList{flag("SelfIgnoreLightningResistance")},
	`([0-9]+)% increased maximum life and reduced fire resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Life", Inc, Num(c.n(1))), mod("FireResist", Inc, Num(-c.n(1)))}
	}),
	`([0-9]+)% increased maximum mana and reduced cold resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Mana", Inc, Num(c.n(1))), mod("ColdResist", Inc, Num(-c.n(1)))}
	}),
	`([0-9]+)% increased global maximum energy shield and reduced lightning resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShield", Inc, Num(c.n(1)), &MarkerTag{Marker: TagGlobal}), mod("LightningResist", Inc, Num(-c.n(1)))}
	}),
	`phasing while on low life`:                                modList{flag("Condition:Phasing", &CondTag{Var: "LowLife"})},
	`cannot be ignited while on low life`:                      modList{flag("IgniteImmune", &CondTag{Var: "LowLife"})},
	`ward does not break during f?l?a?s?k? ?effect`:            modList{flag("Condition:WardNotBreak", &CondTag{Var: "UsingFlask"}), flag("Condition:UnbrokenWard", &CondTag{Var: "UsingFlask"})},
	`stun threshold is based on energy shield instead of life`: modList{flag("StunThresholdBasedOnEnergyShieldInsteadOfLife"), mod("StunThresholdEnergyShieldPercent", Base, Num(100))},
	`stun threshold is based on ([0-9]+)% of your energy shield instead of life`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("StunThresholdBasedOnEnergyShieldInsteadOfLife"), mod("StunThresholdEnergyShieldPercent", Base, Num(c.n(1)))}
	}),
	`stun threshold is based on ([0-9]+)% of your mana instead of life`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("StunThresholdBasedOnManaInsteadOfLife"), mod("StunThresholdManaPercent", Base, Num(c.n(1)))}
	}),
	`([0-9]+)% of your energy shield is added to your stun threshold`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("AddESToStunThreshold"), mod("ESToStunThresholdPercent", Base, Num(c.n(1)))}
	}),
	`([0-9]+)% of damage from hits is taken from your spectres' life before you`: modFn(func(c caps) []*Mod { return []*Mod{mod("takenFromSpectresBeforeYou", Base, Num(c.n(1)))} }),
	`([0-9]+)% of damage from hits is taken from your nearest totem's life before you`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("takenFromTotemsBeforeYou", Base, Num(c.n(1)), &CondTag{Var: "HaveTotem"})}
	}),
	`([0-9]+)% of damage from hits is taken from void spawns' life before you per void spawn`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("takenFromVoidSpawnBeforeYou", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "ActiveVoidSpawnLimit"})}
	}),
	`([a-zA-Z]+) resistance cannot be penetrated`: modFn(func(c caps) []*Mod { return []*Mod{flag("EnemyCannotPen" + firstToUpper(c.s(1)) + "Resistance")} }),
	// Knockback
	`cannot knock enemies back`:                                     modList{flag("CannotKnockback")},
	`knocks back enemies if you get a critical strike with a staff`: modList{modf("EnemyKnockbackChance", Base, Num(100), FlagStaff, KeywordNone, &CondTag{Var: "CriticalStrike"})},
	`knocks back enemies if you get a critical strike with a bow`:   modList{modf("EnemyKnockbackChance", Base, Num(100), FlagBow, KeywordNone, &CondTag{Var: "CriticalStrike"})},
	`bow knockback at close range`:                                  modList{modf("EnemyKnockbackChance", Base, Num(100), FlagBow, KeywordNone, &CondTag{Var: "AtCloseRange"})},
	`adds knockback during f?l?a?s?k? ?effect`:                      modList{mod("EnemyKnockbackChance", Base, Num(100), &CondTag{Var: "UsingFlask"})},
	`adds knockback to melee attacks during f?l?a?s?k? ?effect`:     modList{modf("EnemyKnockbackChance", Base, Num(100), FlagMelee, KeywordNone, &CondTag{Var: "UsingFlask"})},
	`knockback direction is reversed`:                               modList{mod("EnemyKnockbackDistance", More, Num(-200))},
	// Culling
	`culling strike`:                                                     modList{mod("CullPercent", Max, Num(10), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`culling strike with melee weapons`:                                  modList{modf("CullPercent", Max, Num(10), FlagWeaponMelee, KeywordNone, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`melee weapon attacks have culling strike`:                           modList{modf("CullPercent", Max, Num(10), FlagAttack|FlagWeaponMelee, KeywordNone, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`culling strike during f?l?a?s?k? ?effect`:                           modList{mod("CullPercent", Max, Num(10), &CondTag{Var: "UsingFlask"}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`hits with this weapon have culling strike against bleeding enemies`: modList{mod("CullPercent", Max, Num(10), &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"})},
	`you have culling strike against cursed enemies`:                     modList{mod("CullPercent", Max, Num(10), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})},
	`critical strikes have culling strike`:                               modList{mod("CriticalCullPercent", Max, Num(10))},
	`your critical strikes have culling strike`:                          modList{mod("CriticalCullPercent", Max, Num(10))},
	`your spells have culling strike`:                                    modList{modf("CullPercent", Max, Num(10), FlagSpell, KeywordNone)},
	`bow attacks have culling strike`:                                    modList{modf("CullPercent", Max, Num(10), FlagAttack|FlagBow, KeywordNone)},
	`culling strike against burning enemies`:                             modList{mod("CullPercent", Max, Num(10), &CondTag{IsActor: true, Actor: "enemy", Var: "Burning"})},
	`culling strike against frozen enemies`:                              modList{mod("CullPercent", Max, Num(10), &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"})},
	`culling strike against marked enemy`:                                modList{mod("CullPercent", Max, Num(10), &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"})},
	`nearby allies have culling strike`:                                  modList{mod("ExtraAura", List, ModRef{Mod: mod("CullPercent", Max, Num(10)), OnlyAllies: true})},
	`hits that stun enemies have culling strike`:                         modList{flag("Condition:maceMasteryStunCullSpecced")},
	// Intimidate
	`permanently intimidate enemies on block`:                                                         modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &CondTag{Var: "BlockedRecently"})},
	`with a murderous eye jewel socketed, intimidate enemies for ([0-9]) seconds on hit with attacks`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &CondTag{Var: "HaveMurderousEyeJewelIn{SlotName}"})},
	`enemies taunted by your warcries are intimidated`:                                                modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated", &CondTag{Var: "Taunted"})}, &CondTag{Var: "UsedWarcryRecently"})},
	`intimidate enemies for ([0-9]+) seconds on block while holding a shield`:                         modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &CondTag{Var: "BlockedRecently"}, &CondTag{Var: "UsingShield"})},
	`intimidate enemies for ([0-9]+) seconds on hit with attacks while at maximum endurance charges`:  modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMax"}, &CondTag{Var: "HitRecently"})},
	`your hits intimidate enemies for ([0-9]+) seconds while you are using pride`:                     modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &CondTag{Var: "AffectedByPride"})},
	// Flasks
	`flasks do not apply to you`:                               modList{flag("FlasksDoNotApplyToPlayer")},
	`flasks apply to your zombies and spectres`:                modList{flag("FlasksApplyToMinion", &SkillNameTag{SkillNameList: []string{"Raise Zombie", "Raise Spectre"}, IncludeTransfigured: true})},
	`flasks apply to your raised zombies and spectres`:         modList{flag("FlasksApplyToMinion", &SkillNameTag{SkillNameList: []string{"Raise Zombie", "Raise Spectre"}, IncludeTransfigured: true})},
	`flasks you use apply to your raised zombies and spectres`: modList{flag("FlasksApplyToMinion", &SkillNameTag{SkillNameList: []string{"Raise Zombie", "Raise Spectre"}, IncludeTransfigured: true})},
	`your minions use your flasks when summoned`:               modList{flag("FlasksApplyToMinion")},
	`recover an additional ([0-9]+)% of flask's life recovery amount over 10 seconds if used while not on full life`: modFn(func(c caps) []*Mod { return []*Mod{mod("FlaskAdditionalLifeRecovery", Base, Num(c.n(1)))} }),
	`creates a smoke cloud on use`:                     modList{},
	`creates chilled ground on use`:                    modList{},
	`creates consecrated ground on use`:                modList{},
	`removes bleeding on use`:                          modList{},
	`removes burning on use`:                           modList{},
	`removes all burning when used`:                    modList{},
	`removes curses on use`:                            modList{},
	`removes freeze and chill on use`:                  modList{},
	`removes poison on use`:                            modList{},
	`removes shock on use`:                             modList{},
	`g?a?i?n? ?unholy might during f?l?a?s?k? ?effect`: modList{flag("Condition:UnholyMight", &CondTag{Var: "UsingFlask"}), flag("Condition:CanWither", &CondTag{Var: "UsingFlask"})},
	`zealot's oath during f?l?a?s?k? ?effect`:          modList{flag("ZealotsOath", &CondTag{Var: "UsingFlask"})},
	`shocks nearby enemies during f?l?a?s?k? ?effect, causing ([0-9]+)% increased damage taken`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ShockOverride", Base, Num(c.n(1)), &CondTag{Var: "UsingFlask"})}
	}),
	`during f?l?a?s?k? ?effect, ([0-9]+)% reduced damage taken of each element for which your uncapped elemental resistance is lowest`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningDamageTaken", Inc, Num(-c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "LightningResistTotal", ThresholdStat: "ColdResistTotal", Upper: true}, &StatTag{StatKind: TagStatThreshold, Stat: "LightningResistTotal", ThresholdStat: "FireResistTotal", Upper: true}), mod("ColdDamageTaken", Inc, Num(-c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "ColdResistTotal", ThresholdStat: "LightningResistTotal", Upper: true}, &StatTag{StatKind: TagStatThreshold, Stat: "ColdResistTotal", ThresholdStat: "FireResistTotal", Upper: true}), mod("FireDamageTaken", Inc, Num(-c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "FireResistTotal", ThresholdStat: "LightningResistTotal", Upper: true}, &StatTag{StatKind: TagStatThreshold, Stat: "FireResistTotal", ThresholdStat: "ColdResistTotal", Upper: true})}
	}),
	`during f?l?a?s?k? ?effect, damage penetrates ([0-9]+)% o?f? ?resistance of each element for which your uncapped elemental resistance is highest`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningPenetration", Base, Num(c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "LightningResistTotal", ThresholdStat: "ColdResistTotal"}, &StatTag{StatKind: TagStatThreshold, Stat: "LightningResistTotal", ThresholdStat: "FireResistTotal"}), mod("ColdPenetration", Base, Num(c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "ColdResistTotal", ThresholdStat: "LightningResistTotal"}, &StatTag{StatKind: TagStatThreshold, Stat: "ColdResistTotal", ThresholdStat: "FireResistTotal"}), mod("FirePenetration", Base, Num(c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "FireResistTotal", ThresholdStat: "LightningResistTotal"}, &StatTag{StatKind: TagStatThreshold, Stat: "FireResistTotal", ThresholdStat: "ColdResistTotal"})}
	}),
	`damage penetrates fire resistance equal to your overcapped fire resistance`: modList{flag("FirePenIncreasedByUncappedFireRes")},
	`damage penetrates fire resistance equal to your overcapped fire resistance, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FirePenetration", Base, Num(1), &StatTag{StatKind: TagPerStat, Stat: "FireResistOverCap", Limit: opt(c.n(1)), LimitTotal: true})}
	}),
	`recover ([0-9]+)% of life when you kill an enemy during f?l?a?s?k? ?effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &CondTag{Var: "UsingFlask"})}
	}),
	`recover ([0-9]+)% of mana when you kill an enemy during f?l?a?s?k? ?effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &CondTag{Var: "UsingFlask"})}
	}),
	`recover ([0-9]+)% of energy shield when you kill an enemy during f?l?a?s?k? ?effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShieldOnKill", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{Var: "UsingFlask"})}
	}),
	`([0-9]+)% of maximum life taken as chaos damage per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}
	}),
	`your critical strikes do not deal extra damage during f?l?a?s?k? ?effect`: modList{flag("NoCritMultiplier", &CondTag{Var: "UsingFlask"})},
	`grants perfect agony during f?l?a?s?k? ?effect`:                           modList{mod("Keystone", List, Str("Perfect Agony"), &CondTag{Var: "UsingFlask"})},
	`grants eldritch battery during f?l?a?s?k? ?effect`:                        modList{mod("Keystone", List, Str("Eldritch Battery"), &CondTag{Var: "UsingFlask"})},
	`eldritch battery during f?l?a?s?k? ?effect`:                               modList{mod("Keystone", List, Str("Eldritch Battery"), &CondTag{Var: "UsingFlask"})},
	`chaos damage t?a?k?e?n? ?does not bypass energy shield during effect`:     modList{flag("ChaosNotBypassEnergyShield")},
	`when hit during effect, ([0-9]+)% of life loss from damage taken occurs over 4 seconds instead`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeLossPrevented", Base, Num(c.n(1)), &CondTag{Var: "UsingFlask"})}
	}),
	`y?o?u?r? ?skills [ch][oa][sv][te] no mana c?o?s?t? ?during f?l?a?s?k? ?effect`:      modList{mod("ManaCost", More, Num(-100), &CondTag{Var: "UsingFlask"})},
	`life recovery from flasks also applies to energy shield during f?l?a?s?k? ?effect`:  modList{flag("LifeFlaskAppliesToEnergyShield", &CondTag{Var: "UsingFlask"})},
	`profane ground you create inflicts malediction on enemies`:                          modList{mod("EnemyModifier", List, ModRef{Mod: flag("HasMalediction", &CondTag{Var: "OnProfaneGround"})})},
	`profane ground you create also affects you and your allies, granting chaotic might`: modList{mod("ExtraAura", List, ModRef{Mod: flag("Condition:ChaoticMight", &CondTag{Var: "OnProfaneGround"})})},
	`raised beast spectres have farrul's farric presence`:                                modList{mod("ExtraAura", List, ModRef{Mod: mod("Accuracy", Inc, Num(80)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"}), mod("ExtraAura", List, ModRef{Mod: mod("CritChance", Inc, Num(120)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"}), mod("ExtraAura", List, ModRef{Mod: mod("ReduceCritExtraDamage", Base, Num(100)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"})},
	`raised beast spectres have farrul's fertile presence`:                               modList{mod("ExtraAura", List, ModRef{Mod: mod("Damage", Inc, Num(100)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"}), mod("ExtraAura", List, ModRef{Mod: mod("LifeRegenPercent", Base, Num(3)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"}), mod("ExtraAura", List, ModRef{Mod: flag("StunImmune"), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"})},
	`raised beast spectres have farrul's wild presence`:                                  modList{mod("ExtraAura", List, ModRef{Mod: mod("Speed", Inc, Num(20)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"}), mod("ExtraAura", List, ModRef{Mod: mod("MovementSpeed", Inc, Num(20)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"}), mod("ExtraAura", List, ModRef{Mod: mod("MinimumActionSpeed", Max, Num(100)), FromAllies: true}, &CondTag{Var: "HaveBeastSpectre"})},
	`gain alchemist's genius when you use a flask`:                                       modList{flag("Condition:CanHaveAlchemistGenius")},
	`([0-9]+)% chance to gain alchemist's genius when you use a flask`:                   modList{flag("Condition:CanHaveAlchemistGenius")},
	`([0-9]+)% less flask charges gained from kills`:                                     modFn(func(c caps) []*Mod { return []*Mod{mods("FlaskChargesGained", More, Num(-c.n(1)), "from Kills")} }),
	`flasks gain ([0-9]+) charges? every ([0-9]+) seconds`:                               modFn(func(c caps) []*Mod { return []*Mod{mod("FlaskChargesGenerated", Base, Num(c.n(1)/c.n(2)))} }),
	`flasks gain a charge every ([0-9]+) seconds`:                                        modFn(func(c caps) []*Mod { return []*Mod{mod("FlaskChargesGenerated", Base, Num(1/c.n(1)))} }),
	`while a unique enemy is in your presence, flasks gain a charge every ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskChargesGenerated", Base, Num(1/c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})}
	}),
	`while a pinnacle atlas boss is in your presence, flasks gain a charge every ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskChargesGenerated", Base, Num(1/c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"})}
	}),
	`utility flasks gain ([0-9]+) charges? every ([0-9]+) seconds`: modFn(func(c caps) []*Mod { return []*Mod{mod("UtilityFlaskChargesGenerated", Base, Num(c.n(1)/c.n(2)))} }),
	`iron flasks gain ([0-9]+) charges? when your ward breaks`:     modFn(func(c caps) []*Mod { return []*Mod{mod("IronFlaskChargesGeneratedOnWardBreak", Base, Num(c.n(1)))} }),
	`life flasks gain ([0-9]+) charges? every ([0-9]+) seconds`:    modFn(func(c caps) []*Mod { return []*Mod{mod("LifeFlaskChargesGenerated", Base, Num(c.n(1)/c.n(2)))} }),
	`while on low life, life flasks gain ([0-9]+) charges? every ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeFlaskChargesGenerated", Base, Num(c.n(1)/c.n(2)), &CondTag{Var: "LowLife"})}
	}),
	`mana flasks gain ([0-9]+) charges? every ([0-9]+) seconds`: modFn(func(c caps) []*Mod { return []*Mod{mod("ManaFlaskChargesGenerated", Base, Num(c.n(1)/c.n(2)))} }),
	`flasks gain ([0-9]+) charges? per empty flask slot every ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskChargesGeneratedPerEmptyFlask", Base, Num(c.n(1)/c.n(2)))}
	}),
	`flasks gain ([0-9]+) charges? per second if you've hit a unique enemy recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskChargesGenerated", Base, Num(c.n(1)), &CondTag{Var: "HitRecently"}, &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"})}
	}),
	`effect is not removed when unreserved mana is filled`:              modList{flag("ManaFlaskEffectNotRemoved")},
	`life flask effects are not removed when unreserved life is filled`: modList{flag("LifeFlaskEffectNotRemoved")},
	`mana flask effects are not removed when unreserved mana is filled`: modList{flag("ManaFlaskEffectNotRemoved")},
	// Jewels
	`passives? ?s?k?i?l?l?s? in radius of ([a-zA-Z \t\n\v\f\r']+) can be allocated without being connected to your tree`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "impossibleEscapeKeystone", Value: c.str(1)}), mod("ImpossibleEscapeKeystones", List, DataRef{Key: c.s(1), Value: Bool(true)})}
	}),
	`passives? ?s?k?i?l?l?s? in radius can be allocated without being connected to your tree`: modList{mod("JewelData", List, DataRef{Key: "intuitiveLeapLike", Value: Bool(true)})},
	`keystone passive skills in radius can be allocated without being connected to your tree`: modList{mod("JewelData", List, DataRef{Key: "intuitiveLeapLike", Value: Bool(true)}), mod("JewelData", List, DataRef{Key: "intuitiveLeapKeystoneOnly", Value: Bool(true)})},
	`affects passives in small ring`:                    modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(6)})},
	`affects passives in medium ring`:                   modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(7)})},
	`affects passives in large ring`:                    modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(8)})},
	`affects passives in very large ring`:               modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(9)})},
	`affects passives in massive ring`:                  modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(10)})},
	`only affects passives in small ring`:               modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(6)})},
	`only affects passives in medium ring`:              modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(7)})},
	`only affects passives in large ring`:               modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(8)})},
	`only affects passives in very large ring`:          modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(9)})},
	`only affects passives in massive ring`:             modList{mod("JewelData", List, DataRef{Key: "radiusIndex", Value: Num(10)})},
	`primordial`:                                        modList{mod("Multiplier:PrimordialItem", Base, Num(1))},
	`spectres have a base duration of ([0-9]+) seconds`: modList{mod("SkillData", List, DataRef{Key: "duration", Value: Num(6)}, &SkillNameTag{SkillName: "Raise Spectre", IncludeTransfigured: true})},
	`flasks applied to you have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`flasks applied to you have ([0-9]+)% increased effect per level`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "player"}, &MultiplierTag{Var: "Level"})}
	}),
	`equipped magic flasks have ([0-9]+)% increased effect on you if no flasks are adjacent to them`: modFn(func(c caps) []*Mod { return []*Mod{mod("MagicFlaskNoAdjacentEffect", Inc, Num(c.n(1)))} }),
	`while a unique enemy is in your presence, flasks applied to you have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"}, &CondTag{IsActor: true, Actor: "player"})}
	}),
	`while a pinnacle atlas boss is in your presence, flasks applied to you have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "PinnacleBoss"}, &CondTag{IsActor: true, Actor: "player"})}
	}),
	`magic utility flasks applied to you have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MagicUtilityFlaskEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`flasks applied to you have ([0-9]+)% reduced effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FlaskEffect", Inc, Num(-c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`tinctures applied to you have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TinctureEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`tinctures applied to you have ([0-9]+)% increased effect if you've used a life flask recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TinctureEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "player"}, &CondTag{Var: "UsingLifeFlask"})}
	}),
	`tinctures applied to you have ([0-9]+)% increased effect while affected by no flasks`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TinctureEffect", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "player"}, &CondTag{Var: "UsingFlask", Neg: true})}
	}),
	`tinctures have ([0-9]+)% increased effect while at or above ([0-9]+) stacks of mana burn`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TinctureEffect", Inc, Num(c.n(1)), &MultiplierTag{IsThreshold: true, VarList: []string{"ManaBurnStacks", "WeepingWoundsStacks"}, Threshold: opt(c.n(2))})}
	}),
	`tinctures applied to you have ([0-9]+)% reduced mana burn rate`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TinctureManaBurnRate", Inc, Num(-c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`tinctures applied to you have ([0-9]+)% less mana burn rate`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TinctureManaBurnRate", More, Num(-c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`the first ([0-9]+) mana burn applied to you have no effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EffectiveManaBurnStacks", Base, Num(-c.n(1)), &CondTag{IsActor: true, Actor: "player"})}
	}),
	`tinctures deactivate when you have ([0-9]+) or more mana burn`: modFn(func(c caps) []*Mod { return []*Mod{mod("MaxManaBurnStacks", Base, Num(c.n(1)))} }),
	`tinctures inflict weeping wounds instead of mana burn`:         modList{flag("Condition:WeepingWoundsInsteadOfManaBurn")},
	`tincture effects also apply to ranged weapons`:                 modList{flag("TinctureRangedWeapons")},
	`you can have an additional tincture active`:                    modList{mod("TinctureLimit", Base, Num(1))},
	`([0-9]+)% increased tincture cooldown recovery rate`:           modFn(func(c caps) []*Mod { return []*Mod{mod("TinctureCooldownRecovery", Inc, Num(c.n(1)))} }),
	`adds ([0-9]+) passive skills`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelNodeCount", Value: Num(c.n(1))})}
	}),
	`1 added passive skill is a jewel socket`: modList{mod("JewelData", List, DataRef{Key: "clusterJewelSocketCount", Value: Num(1)})},
	`([0-9]+) added passive skills are jewel sockets`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelSocketCount", Value: Num(c.n(1))})}
	}),
	`adds ([0-9]+) jewel socket passive skills`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelSocketCountOverride", Value: Num(c.n(1))})}
	}),
	`adds ([0-9]+) small passive skills? which grants? nothing`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelNothingnessCount", Value: Num(c.n(1))})}
	}),
	`added small passive skills grant nothing`: modList{mod("JewelData", List, DataRef{Key: "clusterJewelSmallsAreNothingness", Value: Bool(true)})},
	`added small passive skills have ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelIncEffect", Value: Num(c.n(1))})}
	}),
	`this jewel's socket has ([0-9]+)% increased effect per allocated passive skill between it and your class' starting location`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "jewelIncEffectFromClassStart", Value: Num(c.n(1))})}
	}),
	`([0-9]+)% increased effect of jewel socket passive skills containing corrupted (m?r?ag?r?i?e?c?) jewels, if not from cluster jewels`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "corrupted" + firstToUpper(c.s(2)) + "JewelIncEffect", Value: Num(c.n(1))})}
	}),
	`([0-9]+)% increased effect of jewel socket passive skills containing corrupted (m?r?ag?r?i?e?c?) jewels`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("JewelData", List, DataRef{Key: "corrupted" + firstToUpper(c.s(2)) + "JewelIncEffect", Value: Num(c.n(1))})}
	}),
	// Misc
	`can't use chest armour`:        modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Body Armour"})},
	`can't use helmets?`:            modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Helmet"})},
	`can't use other rings`:         modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Ring 2"}, &SlotTag{SlotKind: TagSlotNumber, Num: 1}), mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Ring 1"}, &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`uses both hand slots`:          modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Weapon 2"}, &SlotTag{SlotKind: TagSlotNumber, Num: 1}), mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Weapon 1"}, &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`can't use flask in fifth slot`: modList{mod("CanNotUseItem", FlagTypo, Num(1), &SlotTag{SlotKind: TagDisablesItem, SlotName: "Flask 5", ExcludeItemType: "Tincture"})},
	`boneshatter has ([0-9]+)% chance to grant \+1 trauma`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraTrauma", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Boneshatter", IncludeTransfigured: true})}
	}),
	`your minimum frenzy, endurance and power charges are equal to your maximum while you are stationary`: modList{flag("MinimumFrenzyChargesIsMaximumFrenzyCharges", &CondTag{Var: "Stationary"}), flag("MinimumEnduranceChargesIsMaximumEnduranceCharges", &CondTag{Var: "Stationary"}), flag("MinimumPowerChargesIsMaximumPowerCharges", &CondTag{Var: "Stationary"})},
	`minimum power charges equal to maximum while stationary`:                                             modList{flag("MinimumPowerChargesIsMaximumPowerCharges", &CondTag{Var: "Stationary"})},
	`minimum frenzy charges equal to maximum while stationary`:                                            modList{flag("MinimumFrenzyChargesIsMaximumFrenzyCharges", &CondTag{Var: "Stationary"})},
	`minimum endurance charges equal to maximum while stationary`:                                         modList{flag("MinimumEnduranceChargesIsMaximumEnduranceCharges", &CondTag{Var: "Stationary"})},
	`count as having maximum number of power charges`:                                                     modList{flag("HaveMaximumPowerCharges")},
	`count as having maximum number of frenzy charges`:                                                    modList{flag("HaveMaximumFrenzyCharges")},
	`count as having maximum number of endurance charges`:                                                 modList{flag("HaveMaximumEnduranceCharges")},
	`leftmost ([0-9]+) magic utility flasks constantly apply their flask effects to you`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("LeftActiveMagicUtilityFlasks", Base, Num(c.n(1)))} }),
	`rightmost ([0-9]+) magic utility flasks constantly apply their flask effects to you`:                 modFn(func(c caps) []*Mod { return []*Mod{mod("RightActiveMagicUtilityFlasks", Base, Num(c.n(1)))} }),
	`marauder: melee skills have ([0-9]+)% increased area of effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &CondTag{Var: "ConnectedToMarauderStart"}, &SkillTypeTag{SkillType: SkillTypeMelee})}
	}),
	`intelligence provides no bonus to energy shield`:                                       modList{flag("NoIntBonusToES")},
	`intelligence provides no inherent bonus to energy shield`:                              modList{flag("NoIntBonusToES")},
	`gain accuracy rating equal to your intelligence`:                                       modList{mod("Accuracy", Base, Num(1), &StatTag{StatKind: TagPerStat, Stat: "Int"})},
	`intelligence is added to accuracy rating with wands`:                                   modList{modf("Accuracy", Base, Num(1), FlagWand, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "Int"})},
	`dexterity's accuracy bonus instead grants \+([0-9]+) to accuracy rating per dexterity`: modFn(func(c caps) []*Mod { return []*Mod{mod("DexAccBonusOverride", Override, Num(c.n(1)))} }),
	`([0-9]+)% increased accuracy rating against marked enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", Inc, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"})}
	}),
	`([0-9]+)% more accuracy rating against marked enemy`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", More, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"})}
	}),
	`\+([0-9]+) to accuracy against bleeding enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AccuracyVsEnemy", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"})}
	}),
	`cannot recover energy shield to above armour`:                         modList{flag("ArmourESRecoveryCap")},
	`cannot recover energy shield to above evasion rating`:                 modList{flag("EvasionESRecoveryCap")},
	`warcries exert ([0-9]+) additional attacks?`:                          modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraExertedAttacks", Base, Num(c.n(1)))} }),
	`battlemage's cry exerts ([0-9]+) additional attack`:                   modFn(func(c caps) []*Mod { return []*Mod{mod("BattlemageExertedAttacks", Base, Num(c.n(1)))} }),
	`rallying cry exerts ([0-9]+) additional attack`:                       modFn(func(c caps) []*Mod { return []*Mod{mod("RallyingExertedAttacks", Base, Num(c.n(1)))} }),
	`warcries have ([0-9]+)% chance to exert ([0-9]+) additional attacks?`: modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraExertedAttacks", Base, Num((c.n(1) * c.n(2) / 100)))} }),
	`skills deal ([0-9]+)% more damage for each warcry exerting them`:      modFn(func(c caps) []*Mod { return []*Mod{mod("EchoesExertAverageIncrease", More, Num(c.n(1)))} }),
	`iron will`:                      modList{flag("IronWill")},
	`iron reflexes while stationary`: modList{mod("Keystone", List, Str("Iron Reflexes"), &CondTag{Var: "Stationary"})},
	`you have iron reflexes while at maximum frenzy charges`:  modList{mod("Keystone", List, Str("Iron Reflexes"), &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"})},
	`you have zealot's oath if you haven't been hit recently`: modList{mod("Keystone", List, Str("Zealot's Oath"), &CondTag{Var: "BeenHitRecently", Neg: true})},
	`deal no physical damage`:                                 modList{flag("DealNoPhysical")},
	`deal no cold damage`:                                     modList{flag("DealNoCold")},
	`deal no fire damage`:                                     modList{flag("DealNoFire")},
	`deal no lightning damage`:                                modList{flag("DealNoLightning")},
	`deal no elemental damage`:                                modList{flag("DealNoLightning"), flag("DealNoCold"), flag("DealNoFire")},
	`deal no chaos damage`:                                    modList{flag("DealNoChaos")},
	`deal no damage`:                                          modList{flag("DealNoDamage")},
	`you can't deal damage with skills yourself`:              modList{flag("DealNoDamage", &SkillTypeTag{SkillTypeList: []SkillTypeID{SkillTypeSummonsTotem, SkillTypeRemoteMined, SkillTypeTrapped}, Neg: true}, &CondTag{Var: "usedByMirage", Neg: true})},
	`deal no non-elemental damage`:                            modList{flag("DealNoPhysical"), flag("DealNoChaos")},
	`deal no non-([a-zA-Z]+) damage`:                          modFn(func(c caps) []*Mod { return dealNoNonDamageType(c.s(1)) }),
	`cannot deal non-([a-zA-Z]+) damage`:                      modFn(func(c caps) []*Mod { return dealNoNonDamageType(c.s(1)) }),
	`deal no physical or elemental damage`:                    modList{flag("DealNoPhysical"), flag("DealNoCold"), flag("DealNoFire"), flag("DealNoLightning")},
	`deal no damage when not on low life`:                     modList{flag("DealNoDamage", &CondTag{Var: "LowLife", Neg: true})},
	`spell skills deal no damage`:                             modList{flag("DealNoDamage", &SkillTypeTag{SkillType: SkillTypeSpell})},
	`attacks have blood magic`:                                modList{flagf("CostLifeInsteadOfMana", FlagAttack, KeywordNone)},
	`attacks cost life instead of mana`:                       modList{flagf("CostLifeInsteadOfMana", FlagAttack, KeywordNone)},
	`attack skills cost life instead of ([0-9]+)% of mana cost`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("HybridManaAndLifeCost_Life", Base, Num(c.n(1)), FlagAttack, KeywordNone)}
	}),
	`skills cost energy shield instead of mana or life`: modList{flag("CostESInsteadOfManaOrLife")},
	`spells have an additional life cost equal to ([0-9]+)% of your maximum life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeCostBase", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1)), Floor: true}, &SkillTypeTag{SkillType: SkillTypeSpell})}
	}),
	`spells have added spell damage equal to ([0-9]+)% of physical damage of your equipped two handed weapon`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("WeaponPhysAppliesToSpells"), mod("WeaponPhysAppliesToSpellsPercent", Base, Num(c.n(1)), &CondTag{Var: "UsingTwoHandedWeapon"})}
	}),
	`spells cost \+([0-9]+)% of life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeCostBase", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1)), Floor: true}, &SkillTypeTag{SkillType: SkillTypeSpell})}
	}),
	`trigger a socketed elemental spell on block, with a ([0-9.]+) second cooldown`:                                 modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerElementalSpellOnBlock", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when you block, with a ([0-9.]+) second cooldown`:                                     modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueVirulenceSpellsCastOnBlock", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`([0-9]+)% chance to cast a? ?socketed lightning spells? on hit`:                                                modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`cast a socketed lightning spell on hit`:                                                                        modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed lightning spell on hit`:                                                                     modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed lightning spell on hit, with a ([0-9.]+) second cooldown`:                                   modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell when a hit from this weapon freezes a target, with a ([0-9.]+) second cooldown`:       modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnBowAttackFreezeHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`trigger a socketed spell on unarmed melee critical strike, with a ([0-9.]+) second cooldown`:                   modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportTriggerSpellOnUnarmedMeleeCriticalHit", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike`:                                   modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike, with a ([0-9.]+) second cooldown`: modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`you cannot cast socketed hex curse skills inflict socketed hexes on enemies that trigger your traps`:           modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportCurseOnTrapTriggered", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`your curses can apply to hexproof enemies`:                                                                     modList{flag("CursesIgnoreHexproof")},
	`your hexes can affect hexproof enemies`:                                                                        modList{flag("CursesIgnoreHexproof")},
	`([a-zA-Z \t\n\v\f\r]+) can affect hexproof enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "ignoreHexproof", Value: Bool(true)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))})}
	}),
	`hexes from socketed skills can apply ([0-9]) additional curses`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketedCursesHexLimitValue", Base, Num(c.n(1))), flag("SocketedCursesAdditionalLimit", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	// This is being changed from ignoreHexLimit to SocketedCursesAdditionalLimit due to patch 3.16.0, which states that legacy versions "will be affected by this Curse Limit change,
	// though they will only have 20% less Curse Effect of Curses triggered with Summon Doedre’s Effigy."
	// Legacy versions will still show that "Hexes from Socketed Skills ignore Curse limit", but will instead have an internal limit of 5 to match the current functionality.
	`hexes from socketed skills ignore curse limit`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SocketedCursesHexLimitValue", Base, Num(5)), flag("SocketedCursesAdditionalLimit", &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`reserves ([0-9]+)% of life`:                  modFn(func(c caps) []*Mod { return []*Mod{mod("ExtraLifeReserved", Base, Num(c.n(1)))} }),
	`([0-9]+)% of cold damage taken as lightning`: modFn(func(c caps) []*Mod { return []*Mod{mod("ColdDamageTakenAsLightning", Base, Num(c.n(1)))} }),
	`([0-9]+)% of fire damage taken as lightning`: modFn(func(c caps) []*Mod { return []*Mod{mod("FireDamageTakenAsLightning", Base, Num(c.n(1)))} }),
	`([0-9]+)% of fire and lightning damage from hits taken as cold damage during effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireDamageFromHitsTakenAsCold", Base, Num(c.n(1)), &CondTag{Var: "UsingFlask"}), mod("LightningDamageFromHitsTakenAsCold", Base, Num(c.n(1)), &CondTag{Var: "UsingFlask"})}
	}),
	`items and gems have ([0-9]+)% reduced attribute requirements`:   modFn(func(c caps) []*Mod { return []*Mod{mod("GlobalAttributeRequirements", Inc, Num(-c.n(1)))} }),
	`items and gems have ([0-9]+)% increased attribute requirements`: modFn(func(c caps) []*Mod { return []*Mod{mod("GlobalAttributeRequirements", Inc, Num(c.n(1)))} }),
	`mana reservation of herald skills is always ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "ManaReservationPercentForced", Value: Num(c.n(1))}, &SkillTypeTag{SkillType: SkillTypeHerald})}
	}),
	`([a-zA-Z \t\n\v\f\r]+) reserves no mana`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "manaReservationFlat", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationFlat", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "manaReservationPercent", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationPercent", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true})}
	}),
	`([a-zA-Z \t\n\v\f\r]+) has no reservation`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "manaReservationFlat", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationFlat", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "manaReservationPercent", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationPercent", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true})}
	}),
	`([a-zA-Z \t\n\v\f\r]+) has no reservation if cast as an aura`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SkillData", List, DataRef{Key: "manaReservationFlat", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationFlat", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "manaReservationPercent", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationPercent", Value: Num(0)}, &SkillIDTag{SkillID: gemIdOrNil(c.s(1))}, &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true})}
	}),
	`banner skills reserve no mana`:     modList{mod("SkillData", List, DataRef{Key: "manaReservationPercent", Value: Num(0)}, &SkillTypeTag{SkillType: SkillTypeBanner}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationPercent", Value: Num(0)}, &SkillTypeTag{SkillType: SkillTypeBanner}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true})},
	`banner skills have no reservation`: modList{mod("SkillData", List, DataRef{Key: "manaReservationPercent", Value: Num(0)}, &SkillTypeTag{SkillType: SkillTypeBanner}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true}), mod("SkillData", List, DataRef{Key: "lifeReservationPercent", Value: Num(0)}, &SkillTypeTag{SkillType: SkillTypeBanner}, &SkillTypeTag{SkillType: SkillTypeBlessing, Neg: true})},
	`placed banners also grant ([0-9]+)% increased attack damage to you and allies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAuraEffect", List, ModRef{Mod: modf("Damage", Inc, Num(c.n(1)), FlagAttack, KeywordNone)}, &CondTag{Var: "BannerPlanted"}, &SkillTypeTag{SkillType: SkillTypeBanner})}
	}),
	`banners also cause enemies to take ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAuraDebuffEffect", List, ModRef{Mod: mod("DamageTaken", Inc, Num(c.n(1)), &GlobalEffectTag{EffectType: "AuraDebuff", Unscalable: true})}, &CondTag{Var: "BannerPlanted"}, &SkillTypeTag{SkillType: SkillTypeBanner})}
	}),
	`dread banner grants an additional \+([0-9]+) to maximum fortification when placing the banner`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("MaximumFortification", Base, Num(c.n(1)), &GlobalEffectTag{EffectType: "Buff"})}, &CondTag{Var: "BannerPlanted"}, &SkillNameTag{SkillName: "Dread Banner"})}
	}),
	`your aura skills are disabled`:     modList{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeAura})},
	`your blessing skills are disabled`: modList{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeBlessing})},
	`your spells are disabled`:          modList{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeSpell}), flag("ForceEnableCurseApplication")},
	`your warcries are disabled`:        modList{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeWarcry})},
	`your travel skills are disabled`:   modList{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeTravel})},
	`aura skills other than ([a-zA-Z \t\n\v\f\r]+) are disabled`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeAura}, &SkillTypeTag{SkillType: SkillTypeRemoteMined, Neg: true}), flag("EnableSkill", &SkillNameTag{SkillName: c.s(1)})}
	}),
	`travel skills other than ([a-zA-Z \t\n\v\f\r]+) are disabled`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("DisableSkill", &SkillTypeTag{SkillType: SkillTypeTravel}), flag("EnableSkill", &SkillIDTag{SkillID: gemIdOrNil(c.s(1))})}
	}),
	`strength's damage bonus instead grants ([0-9]+)% increased melee physical damage per ([0-9]+) strength`: modFn(func(c caps) []*Mod { return []*Mod{mod("StrDmgBonusRatioOverride", Base, Num(c.n(1)/c.n(2)))} }),
	`while in her embrace, take ([0-9.]+)% of your total maximum life and energy shield as fire damage per second per level`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &MultiplierTag{Var: "Level"}, &CondTag{Var: "HerEmbrace"}), mod("FireDegen", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &MultiplierTag{Var: "Level"}, &CondTag{Var: "HerEmbrace"})}
	}),
	`gain her embrace for [0-9]+ seconds when you ignite an enemy`: modList{flag("Condition:CanGainHerEmbrace")},
	`when you cast a spell, sacrifice all mana to gain added maximum lightning damage equal to ([0-9]+)% of sacrificed mana for 4 seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Condition:HaveManaStorm"), mod("LightningMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaUnreserved", Percent: opt(c.n(1))}, &CondTag{Var: "SacrificeManaForLightning"})}
	}),
	`attacks with this weapon have added maximum lightning damage equal to ([0-9]+)% of your energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`attacks with this weapon have added maximum lightning damage equal to ([0-9]+)% of your maximum energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`attacks with this weapon have added fire damage equal to ([0-9]+)% of player's maximum life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1)), Actor: "parent"}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), mod("FireMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1)), Actor: "parent"}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`adds ([0-9]+)% of your maximum energy shield as cold damage to attacks with this weapon`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ColdMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), mod("ColdMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1))}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`attacks with this weapon have added maximum lightning damage equal to ([0-9]+)% of player'?s? maximum energy shield`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "EnergyShield", Percent: opt(c.n(1)), Actor: "parent"}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`adds ([0-9]+)% of your maximum mana as fire damage to attacks with this weapon`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}), mod("FireMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`gain added chaos damage equal to ([0-9]+)% of ward`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Ward", Percent: opt(c.n(1))}), mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Ward", Percent: opt(c.n(1))})}
	}),
	`while you have unbroken ward, your next non-channelling attack you use yourself breaks your ward to gain added cold damage equal to ([0-9]+)% of ward`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ColdMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Ward", Percent: opt(c.n(1))}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &SkillTypeTag{SkillType: SkillTypeAttack}, &CondTag{Var: "WardNotBreak", Neg: true}, &CondTag{Var: "UnbrokenWard"}), mod("ColdMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Ward", Percent: opt(c.n(1))}, &SkillTypeTag{SkillType: SkillTypeChannel, Neg: true}, &SkillTypeTag{SkillType: SkillTypeAttack}, &CondTag{Var: "WardNotBreak", Neg: true}, &CondTag{Var: "UnbrokenWard"})}
	}),
	`spells deal added chaos damage equal to ([0-9]+)% of your maximum life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &SkillTypeTag{SkillType: SkillTypeSpell}), mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))}, &SkillTypeTag{SkillType: SkillTypeSpell})}
	}),
	`every 16 seconds you gain iron reflexes for 8 seconds`:                                              modList{flag("Condition:HaveArborix")},
	`every 16 seconds you gain elemental overload for 8 seconds`:                                         modList{flag("Condition:HaveAugyre")},
	`every 8 seconds, gain avatar of fire for 4 seconds`:                                                 modList{flag("Condition:HaveVulconus")},
	`when hit, gain a random movement speed modifier from 40% reduced to 100% increased until hit again`: modList{flag("Condition:HaveGamblesprint")},
	`trigger socketed curse spell when you cast a curse spell, with a ([0-9.]+) second cooldown`:         modList{mod("ExtraSupport", List, SkillRef{SkillID: "SupportUniqueCastCurseOnCurse", Level: opt(1)}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})},
	`modifiers to attributes instead apply to omniscience`:                                               modList{flag("Omniscience")},
	`attribute requirements can be satisfied by ([0-9]+)% of omniscience`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("OmniAttributeRequirements", Inc, Num(c.n(1))), flag("OmniscienceRequirements")}
	}),
	`you have far shot while you do not have iron reflexes`:                modList{flag("FarShot", &CondTag{Neg: true, Var: "HaveIronReflexes"})},
	`you have resolute technique while you do not have elemental overload`: modList{mod("Keystone", List, Str("Resolute Technique"), &CondTag{Neg: true, Var: "HaveElementalOverload"})},
	`hits ignore enemy monster fire resistance while you are ignited`:      modList{flag("IgnoreFireResistance", &CondTag{Var: "Ignited"})},
	`your hits can't be evaded by blinded enemies`:                         modList{flag("CannotBeEvaded", &CondTag{IsActor: true, Actor: "enemy", Var: "Blinded"})},
	`blind does not affect your chance to hit`:                             modList{flag("UnaffectedByBlind")},
	`unaffected by blind`: modList{flag("UnaffectedByBlind")},
	`enemies blinded by you while you are blinded have malediction`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("HasMalediction", &CondTag{Var: "Blinded"})}, &CondTag{Var: "Blinded"}, &CondTag{Var: "CannotBeBlinded", Neg: true})},
	`enemies blinded by you have malediction`:                       modList{mod("EnemyModifier", List, ModRef{Mod: flag("HasMalediction", &CondTag{Var: "Blinded"})})},
	`enemies ignited by you during effect have malediction`:         modList{mod("EnemyModifier", List, ModRef{Mod: flag("HasMalediction", &CondTag{Var: "Ignited"})})},
	`skills which throw traps have blood magic`:                     modList{flag("CostLifeInsteadOfMana", &SkillTypeTag{SkillType: SkillTypeTrapped})},
	`skills which throw traps cost life instead of mana`:            modList{flag("CostLifeInsteadOfMana", &SkillTypeTag{SkillType: SkillTypeTrapped})},
	`strength provides no bonus to maximum life`:                    modList{flag("NoStrBonusToLife")},
	`strength provides no inherent bonus to maximum life`:           modList{flag("NoStrBonusToLife")},
	`intelligence provides no bonus to maximum mana`:                modList{flag("NoIntBonusToMana")},
	`intelligence provides no inherent bonus to maximum mana`:       modList{flag("NoIntBonusToMana")},
	`with a ghastly eye jewel socketed, minions have \+([0-9]+) to accuracy rating`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("Accuracy", Base, Num(c.n(1)))}, &CondTag{Var: "HaveGhastlyEyeJewelIn{SlotName}"})}
	}),
	`with a ghastly eye jewel socketed, minions have ([0-9]+)% chance to gain unholy might on hit with spells`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: flag("Condition:UnholyMight", &CondTag{Var: "HitSpellRecently"})}, &CondTag{Var: "HaveGhastlyEyeJewelIn{SlotName}"})}
	}),
	`with a hypnotic eye jewel socketed, gain arcane surge on hit with spells`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Condition:ArcaneSurge", &CondTag{Var: "HitSpellRecently"}, &CondTag{Var: "HaveHypnoticEyeJewelIn{SlotName}"})}
	}),
	`hits ignore enemy monster chaos resistance if all equipped items are shaper items`: modList{flag("IgnoreChaosResistance", &MultiplierTag{IsThreshold: true, Var: "NonShaperItem", Upper: true, Threshold: opt(0)})},
	`hits ignore enemy monster chaos resistance if all equipped items are elder items`:  modList{flag("IgnoreChaosResistance", &MultiplierTag{IsThreshold: true, Var: "NonElderItem", Upper: true, Threshold: opt(0)})},
	`your hits ignore enemy monster ([a-zA-Z]+) resistances? if all equipped rings are ([a-zA-Z]+) rings`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Ignore"+firstToUpper(c.s(1))+"Resistance", &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "RingEquipped", Threshold: opt(2)})}
	}),
	`the stars are aligned if you have 6 influence types among other equipped items`:         modList{flag("Condition:StarsAreAligned", &MultiplierTag{IsThreshold: true, Var: "ShaperItem", Threshold: opt(2)}, &MultiplierTag{IsThreshold: true, Var: "ElderItem", Threshold: opt(2)}, &MultiplierTag{IsThreshold: true, Var: "WarlordItem", Threshold: opt(2)}, &MultiplierTag{IsThreshold: true, Var: "HunterItem", Threshold: opt(2)}, &MultiplierTag{IsThreshold: true, Var: "CrusaderItem", Threshold: opt(2)}, &MultiplierTag{IsThreshold: true, Var: "RedeemerItem", Threshold: opt(2)})},
	`gain [0-9]+ rage on critical hit with attacks, no more than once every [0-9.]+ seconds`: modList{flag("Condition:CanGainRage")},
	`gain [0-9]+ rage on critical hit with attacks`:                                          modList{flag("Condition:CanGainRage")},
	`warcry skills' cooldown time is ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CooldownRecovery", Override, Num(c.n(1)), FlagNone, KeywordWarcry)}
	}),
	`non-instant warcries you use yourself have no cooldown`: modFn(func(c caps) []*Mod {
		// Archive parity: the reference writes SkillTypeTotem here,
		// which does not exist in Global.lua, so the list carries nil.
		return []*Mod{modf("CooldownRecovery", Override, Num(0), FlagNone, KeywordWarcry, &SkillTypeTag{SkillTypeList: []SkillTypeID{SkillTypeInstant, 0, SkillTypeTriggered}, Neg: true})}
	}),
	`non-instant warcries ignore their cooldown when used`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CooldownRecovery", Override, Num(0), FlagNone, KeywordWarcry, &SkillTypeTag{SkillType: SkillTypeInstant, Neg: true})}
	}),
	`warcry skills have (\+[0-9]+) seconds to cooldown`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CooldownRecovery", Base, Num(c.n(1)), FlagNone, KeywordWarcry)}
	}),
	`([0-9]+)% increased total power counted by warcries`: modFn(func(c caps) []*Mod { return []*Mod{mod("WarcryPower", Inc, Num(c.n(1)))} }),
	`warcries have a minimum of ([0-9]+) power`:           modFn(func(c caps) []*Mod { return []*Mod{mod("MinimumWarcryPower", Base, Num(c.n(1)))} }),
	`stance skills have (\+[0-9]+) seconds to cooldown`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CooldownRecovery", Base, Num(c.n(1)), &SkillTypeTag{SkillType: SkillTypeStance})}
	}),
	`using warcries is instant`: modList{flag("InstantWarcry")},
	`attacks with axes or swords grant ([0-9]+) rage on hit, no more than once every second`:                        modList{flag("Condition:CanGainRage", &CondTag{VarList: []string{"UsingAxe", "UsingSword"}})},
	`when you lose temporal chains you gain maximum rage`:                                                           modList{flag("Condition:CanGainRage")},
	`with a murderous eye jewel socketed, melee attacks grant ([0-9]+) rage on hit, no more than once every second`: modList{flag("Condition:CanGainRage", &CondTag{Var: "HaveMurderousEyeJewelIn{SlotName}"})},
	`gain [0-9]+ rage after spending a total of [0-9]+ mana`:                                                        modList{flag("Condition:CanGainRage")},
	`rage grants cast speed instead of attack speed`:                                                                modList{flag("Condition:RageCastSpeed")},
	`rage grants spell damage instead of attack damage`:                                                             modList{flag("Condition:RageSpellDamage")},
	`inherent loss of rage is ([0-9]+)% slower`:                                                                     modFn(func(c caps) []*Mod { return []*Mod{mod("InherentRageLoss", Inc, Num(-c.n(1)))} }),
	`inherent loss of rage is ([0-9]+)% faster`:                                                                     modFn(func(c caps) []*Mod { return []*Mod{mod("InherentRageLoss", Inc, Num(c.n(1)))} }),
	`inherent rage loss starts ([0-9]+) seconds? later`:                                                             modFn(func(c caps) []*Mod { return []*Mod{mod("InherentRageLossDelay", Base, Num(c.n(1)))} }),
	`your critical strike multiplier is ([0-9]+)%`:                                                                  modFn(func(c caps) []*Mod { return []*Mod{mod("CritMultiplier", Override, Num(c.n(1)))} }),
	`base critical strike chance for attacks with weapons is ([0-9.]+)%`:                                            modFn(func(c caps) []*Mod { return []*Mod{mod("WeaponBaseCritChance", Override, Num(c.n(1)))} }),
	`base critical strike chance of spells is the critical strike chance of y?o?u?r? ?main hand weapon`:             modList{flagf("BaseCritFromMainHand", FlagSpell, KeywordNone)},
	`base spell critical strike chance of spells is equal to that of main hand weapon`:                              modList{flagf("BaseCritFromMainHand", FlagSpell, KeywordNone)},
	`critical strike chance is ([0-9]+)% for hits with this weapon`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CritChance", Override, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`hits with this weapon have \+([0-9]+)% to critical strike multiplier per enemy power`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CritMultiplier", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}, &MultiplierTag{Var: "EnemyPower"})}
	}),
	`maximum critical strike chance is ([0-9]+)%`: modFn(func(c caps) []*Mod { return []*Mod{mod("CritChanceCap", Override, Num(c.n(1)))} }),
	`allocates (.+) if you have the matching modifiers? on forbidden (.+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("GrantedAscendancyNode", List, AscendancyNodeRef{Name: c.s(1), Side: c.s(2)})}
	}),
	`allocates (.+)`:          modFn(func(c caps) []*Mod { return []*Mod{mod("GrantedPassive", List, c.str(1))} }),
	`battlemage`:              modList{flag("Battlemage"), mod("MainHandWeaponDamageAppliesToSpells", Max, Num(100))},
	`transfiguration of body`: modList{flag("TransfigurationOfBody")},
	`transfiguration of mind`: modList{flag("TransfigurationOfMind")},
	`transfiguration of soul`: modList{flag("TransfigurationOfSoul")},
	`offering skills have ([0-9]+)% increased duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Duration", Inc, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`offering skills have ([0-9]+)% reduced duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Duration", Inc, Num(-c.n(1)), &SkillNameTag{SkillNameList: []string{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"}})}
	}),
	`enemies have -([0-9]+)% to total physical damage reduction against your hits`:              modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyPhysicalDamageReduction", Base, Num(-c.n(1)))} }),
	`enemies you impale have -([0-9]+)% to total physical damage reduction against impale hits`: modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyImpalePhysicalDamageReduction", Base, Num(-c.n(1)))} }),
	`hits with this weapon overwhelm ([0-9]+)% physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("EnemyPhysicalDamageReduction", Base, Num(-c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`overwhelm ([0-9]+)% physical damage reduction`: modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyPhysicalDamageReduction", Base, Num(-c.n(1)))} }),
	`hits have ([0-9]+)% chance to ignore enemy physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", Base, Num(c.n(1)))}
	}),
	`hits with this weapon have ([0-9]+)% chance to ignore enemy physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ChanceToIgnoreEnemyPhysicalDamageReduction", Base, Num(c.n(1)), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`hits with this weapon ignore enemy physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("ChanceToIgnoreEnemyPhysicalDamageReduction", Base, Num(100), FlagHit, KeywordNone, &CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack})}
	}),
	`hits against you overwhelm ([0-9]+)% of physical damage reduction`:                            modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyPhysicalOverwhelm", Base, Num(c.n(1)))} }),
	`impale damage dealt to enemies impaled by you overwhelms ([0-9]+)% physical damage reduction`: modFn(func(c caps) []*Mod { return []*Mod{mod("EnemyImpalePhysicalDamageReduction", Base, Num(-c.n(1)))} }),
	`impale damage dealt to enemies impaled by you ignores enemy physical damage reduction`:        modList{flag("IgnoreEnemyImpalePhysicalDamageReduction")},
	`you are crushed`:                                     modList{flag("Condition:Crushed")},
	`nearby enemies are crushed`:                          modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Crushed")})},
	`crush enemies on hit with maces and sceptres`:        modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Crushed")}, &CondTag{Var: "UsingMace"})},
	`you have fungal ground around you while stationary`:  modList{mod("ExtraAura", List, ModRef{Mod: mod("ChaosResist", Base, Num(25))}, &CondTag{VarList: []string{"OnFungalGround", "Stationary"}}), mod("EnemyModifier", List, ModRef{Mod: mod("ChaosResist", Base, Num(-10))}, &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"OnFungalGround", "Stationary"}}), mod("EnemyModifier", List, ModRef{Mod: mod("ElementalResist", Base, Num(-10))}, &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"OnFungalGround", "Stationary"}})},
	`create fungal ground instead of consecrated ground`:  modList{flag("Condition:CreateFungalGround")},
	`create profane ground instead of consecrated ground`: modList{flag("Condition:CreateProfaneGround")},
	`([0-9]+)% chance to create profane ground on critical strike if intelligence is your highest attribute`:                modList{flag("Condition:CreateProfaneGround", &CondTag{Var: "IntHighestAttribute"})},
	`consecrated path and purifying flame create profane ground instead of consecrated ground`:                              modList{flag("Condition:CreateProfaneGround")},
	`you gain added cold damage instead of added damage of other types if dexterity exceeds both other attributes`:          modList{flag("AllAddedDamageAsCold", &CondTag{Var: "DexSingleHighestAttribute"})},
	`you gain added lightn?ing damage instead of added damage of other types if intelligence exceeds both other attributes`: modList{flag("AllAddedDamageAsLightning", &CondTag{Var: "IntSingleHighestAttribute"})},
	`elemental hit's added damage cannot be replaced this way`:                                                              modList{},
	`you have consecrated ground around you while stationary if strength is your highest attribute`:                         modList{flag("Condition:OnConsecratedGround", &CondTag{Var: "StrHighestAttribute"}, &CondTag{Var: "Stationary"})},
	`consecrated ground around you while stationary if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Condition:OnConsecratedGround", &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "Item", Threshold: opt(c.n(1))})}
	}),
	`you count as dual wielding while you are unencumbered`:                              modList{flag("Condition:DualWielding", &CondTag{Var: "Unencumbered"})},
	`dual wielding does not inherently grant chance to block attack damage`:              modList{flag("Condition:NoInherentBlock")},
	`inherent attack speed bonus from dual wielding is doubled while wielding two claws`: modList{flag("Condition:DoubledInherentDualWieldingSpeed", &CondTag{Var: "DualWieldingClaws"})},
	`inherent bonuses from dual wielding are doubled`:                                    modList{flag("Condition:DoubledInherentDualWieldingSpeed"), flag("Condition:DoubledInherentDualWieldingBlock")},
	`([0-9]+)% reduced enemy chance to block sword attacks`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("reduceEnemyBlock", Base, Num(c.n(1)), FlagSword, KeywordNone)}
	}),
	`you do not inherently take less damage for having fortification`:     modList{flag("Condition:NoFortificationMitigation")},
	`skills supported by intensify have \+([0-9]) to maximum intensity`:   modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:IntensityLimit", Base, Num(c.n(1)))} }),
	`spells which can gain intensity have \+([0-9]) to maximum intensity`: modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:IntensityLimit", Base, Num(c.n(1)))} }),
	`final repeat of spells has ([0-9]+)% increased area of effect`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("RepeatFinalAreaOfEffect", Inc, Num(c.n(1)), FlagSpell, KeywordNone, &CondTag{Var: "CastOnFrostbolt", Neg: true}, &CondTag{VarList: []string{"averageRepeat", "alwaysFinalRepeat"}})}
	}),
	`hexes you inflict have ([+\-][0-9]+) to maximum doom`: modFn(func(c caps) []*Mod { return []*Mod{mod("MaxDoom", Base, Num(c.n(1)))} }),
	`while stationary, gain ([0-9]+)% increased area of effect every second, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &MultiplierTag{Var: "StationarySeconds", GlobalLimit: opt(c.n(2)), GlobalLimitKey: "ExpansiveMight"}, &CondTag{Var: "Stationary"})}
	}),
	`fireball and rolling magma have ([0-9]+)% more area of effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", More, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Fireball", "Rolling Magma"}})}
	}),
	`attack skills have added lightning damage equal to ([0-9]+)% of maximum mana`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("LightningMin", Base, Num(1), FlagAttack, KeywordNone, &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))}), modf("LightningMax", Base, Num(1), FlagAttack, KeywordNone, &StatTag{StatKind: TagPercentStat, Stat: "Mana", Percent: opt(c.n(1))})}
	}),
	`arc and crackling lance gains added cold damage equal to ([0-9]+)% of mana cost, if mana cost is not higher than the maximum you could spend`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ColdMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaCost", Percent: opt(c.n(1))}, &SkillNameTag{SkillNameList: []string{"Arc", "Crackling Lance"}, IncludeTransfigured: true}), mod("ColdMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaCost", Percent: opt(c.n(1))}, &SkillNameTag{SkillNameList: []string{"Arc", "Crackling Lance"}, IncludeTransfigured: true})}
	}),
	`forbidden rite and dark pact gains added chaos damage equal to ([0-9]+)% of mana cost, if mana cost is not higher than the maximum you could spend`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaCost", Percent: opt(c.n(1))}, &SkillNameTag{SkillNameList: []string{"Forbidden Rite", "Dark Bargain"}, IncludeTransfigured: true}), mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaCost", Percent: opt(c.n(1))}, &SkillNameTag{SkillNameList: []string{"Forbidden Rite", "Dark Bargain"}, IncludeTransfigured: true})}
	}),
	`skills gain added chaos damage equal to ([0-9]+)% of mana cost, if mana cost is not higher than the maximum you could spend`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChaosMin", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaCost", Percent: opt(c.n(1))}), mod("ChaosMax", Base, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "ManaCost", Percent: opt(c.n(1))})}
	}),
	`herald of thunder's storms hit enemies with ([0-9]+)% increased frequency`: modFn(func(c caps) []*Mod { return []*Mod{mod("HeraldStormFrequency", Inc, Num(c.n(1)))} }),
	`storms hit enemies with ([0-9]+)% increased frequency`:                     modFn(func(c caps) []*Mod { return []*Mod{mod("HeraldStormFrequency", Inc, Num(c.n(1)))} }),
	`your critical strikes have a ([0-9]+)% chance to deal double damage`:       modFn(func(c caps) []*Mod { return []*Mod{mod("DoubleDamageChanceOnCrit", Base, Num(c.n(1)))} }),
	`elemental skills deal triple damage`:                                       modList{mod("TripleDamageChance", Base, Num(100), &SkillTypeTag{SkillTypeList: []SkillTypeID{SkillTypeCold, SkillTypeFire, SkillTypeLightning}})},
	`deal triple damage with elemental skills`:                                  modList{mod("TripleDamageChance", Base, Num(100), &SkillTypeTag{SkillTypeList: []SkillTypeID{SkillTypeCold, SkillTypeFire, SkillTypeLightning}})},
	`skills supported by unleash have \+([0-9]) to maximum number of seals`:     modFn(func(c caps) []*Mod { return []*Mod{mod("SealCount", Base, Num(c.n(1)))} }),
	`left ring slot: skills supported by unleash have \+([0-9]) to maximum number of seals`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SealCount", Base, Num(c.n(1)), &SlotTag{SlotKind: TagSlotNumber, Num: 1})}
	}),
	`skills supported by unleash have ([0-9]+)% increased seal gain frequency`:                        modFn(func(c caps) []*Mod { return []*Mod{mod("SealGainFrequency", Inc, Num(c.n(1)))} }),
	`([0-9]+)% increased critical strike chance with spells which remove the maximum number of seals`: modFn(func(c caps) []*Mod { return []*Mod{mod("MaxSealCrit", Inc, Num(c.n(1)))} }),
	`gain elusive on critical strike`:                                                     modList{flag("Condition:CanBeElusive")},
	`gain a random shrine buff every ([0-9]+) seconds`:                                    modList{flag("Condition:CanHaveRegularShrines")},
	`gain a random shrine buff for ([0-9]+) seconds when you kill a rare or unique enemy`: modList{flag("Condition:CanHaveRegularShrines")},
	`([0-9]+)% chance to gain elusive when you block while dual wielding`:                 modList{flag("Condition:CanBeElusive", &CondTag{Var: "DualWielding"})},
	`elusive on you reduces in effect ([0-9]+)% slower`:                                   modFn(func(c caps) []*Mod { return []*Mod{mod("ElusiveEffectLossSlower", Inc, Num(c.n(1)))} }),
	`elusive's effect on you is increased instead for the first ([0-9]+) seconds`:         modFn(func(c caps) []*Mod { return []*Mod{mod("ElusiveEffectIncreaseDuration", Base, Num(c.n(1)))} }),
	`elusive is removed from you at ([0-9]+)% effect`:                                     modFn(func(c caps) []*Mod { return []*Mod{mod("ElusiveEffectMinThreshold", Override, Num(c.n(1)))} }),
	`nearby enemies have ([a-zA-Z]+) resistance equal to yours`:                           modFn(func(c caps) []*Mod { return []*Mod{flag("Enemy" + firstToUpper(c.s(1)) + "ResistEqualToYours")} }),
	`for each nearby corpse, regenerate ([0-9.]+)% life per second, up to ([0-9.]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(1)), &MultiplierTag{Var: "NearbyCorpse", Limit: opt(c.n(2)), LimitTotal: true})}
	}),
	`gain sacrificial zeal when you use a skill, dealing you [0-9]+% of the skill's mana cost as physical damage per second`: modList{flag("SacrificialZeal")},
	`skills gain a base life cost equal to ([0-9]+)% of base mana cost`:                                                      modFn(func(c caps) []*Mod { return []*Mod{mod("ManaCostAsLifeCost", Base, Num(c.n(1)))} }),
	`skills gain a base energy shield cost equal to ([0-9]+)% of base mana cost`:                                             modFn(func(c caps) []*Mod { return []*Mod{mod("ManaCostAsEnergyShieldCost", Base, Num(c.n(1)))} }),
	`skills cost life instead of ([0-9]+)% of mana cost`:                                                                     modFn(func(c caps) []*Mod { return []*Mod{mod("HybridManaAndLifeCost_Life", Base, Num(c.n(1)))} }),
	`([0-9]+)% increased cost of arc and crackling lance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Cost", Inc, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Arc", "Crackling Lance"}, IncludeTransfigured: true})}
	}),
	`hits overwhelm ([0-9]+)% of physical damage reduction while you have sacrificial zeal`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyPhysicalDamageReduction", Base, Num(-c.n(1)), nil, &CondTag{Var: "SacrificialZeal"})}
	}),
	`([0-9]+)% chance for hits to ignore enemy physical damage reduction while you have sacrificial zeal`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", Base, Num(c.n(1)), nil, &CondTag{Var: "SacrificialZeal"})}
	}),
	`hits have ([0-9]+)% chance to ignore enemy physical damage reduction while you have sacrificial zeal`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ChanceToIgnoreEnemyPhysicalDamageReduction", Base, Num(c.n(1)), nil, &CondTag{Var: "SacrificialZeal"})}
	}),
	`minions attacks overwhelm ([0-9]+)% physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("EnemyPhysicalDamageReduction", Base, Num(-c.n(1)), &SkillTypeTag{SkillType: SkillTypeAttack})})}
	}),
	`minions hits have ([0-9]+)% chance to ignore enemy physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod("ChanceToIgnoreEnemyPhysicalDamageReduction", Base, Num(c.n(1)))})}
	}),
	`focus has ([0-9]+)% increased cooldown recovery rate`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FocusCooldownRecovery", Inc, Num(c.n(1)), &CondTag{Var: "Focused"})}
	}),
	`focus has ([0-9]+)% reduced cooldown recovery rate`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FocusCooldownRecovery", Inc, Num(-c.n(1)), &CondTag{Var: "Focused"})}
	}),
	`([0-9]+)% more frozen legion and general's cry cooldown recovery rate`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CooldownRecovery", More, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Frozen Legion", "General's Cry"}, IncludeTransfigured: true})}
	}),
	`flamethrower, seismic and lightning spire trap have ([0-9]+)% increased cooldown recovery rate`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CooldownRecovery", Inc, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Flamethrower Trap", "Seismic Trap", "Lightning Spire Trap"}, IncludeTransfigured: true})}
	}),
	`flamethrower, seismic and lightning spire trap have -([0-9]+) cooldown uses?`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AdditionalCooldownUses", Base, Num(-c.n(1)), &SkillNameTag{SkillNameList: []string{"Flamethrower Trap", "Seismic Trap", "Lightning Spire Trap"}, IncludeTransfigured: true})}
	}),
	`right ring slot: shockwave has \+([0-9]+) to cooldown uses?`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AdditionalCooldownUses", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Shockwave"}, &SlotTag{SlotKind: TagSlotNumber, Num: 2})}
	}),
	`flameblast starts with ([0-9]+) additional stages`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Multiplier:FlameblastMinimumStage", Base, Num(c.n(1)), FlagNone, KeywordNone, &GlobalEffectTag{EffectType: "Buff", Unscalable: true})}
	}),
	`incinerate starts with ([0-9]+) additional stages`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Multiplier:IncinerateMinimumStage", Base, Num(c.n(1)), FlagNone, KeywordNone, &GlobalEffectTag{EffectType: "Buff", Unscalable: true})}
	}),
	`\+([0-9.]+) seconds to flameblast and incinerate cooldown`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("SkillData", List, DataRef{Key: "cooldown", Value: Num(0)})}, &SkillNameTag{SkillNameList: []string{"Incinerate", "Flameblast"}, IncludeTransfigured: true}), mod("CooldownRecovery", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Incinerate", "Flameblast"}, IncludeTransfigured: true})}
	}),
	`([0-9]+)% chance to deal double damage with attacks if attack time is longer than 1 second`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("DoubleDamageChance", Base, Num(c.n(1)), FlagNone, KeywordNone, &CondTag{Var: "OneSecondAttackTime"})}
	}),
	`elusive also grants \+([0-9]+)% to critical strike multiplier for skills supported by nightblade`: modFn(func(c caps) []*Mod { return []*Mod{mod("NightbladeElusiveCritMultiplier", Base, Num(c.n(1)))} }),
	`skills supported by nightblade have ([0-9]+)% increased effect of elusive`:                        modFn(func(c caps) []*Mod { return []*Mod{mod("NightbladeSupportedElusiveEffect", Inc, Num(c.n(1)))} }),
	`nearby enemies are scorched`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Scorched")}), mod("ScorchBase", Base, Num(10))},
	`hits have ([0-9]+)% chance to ignore enemy monster physical damage reduction`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PartialIgnoreEnemyPhysicalDamageReduction", Base, Num(c.n(1)))}
	}),
	`attacks you use yourself have ([0-9]+)% more attack speed`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Speed", More, Num(c.n(1)), FlagAttack, KeywordNone, &SkillTypeTag{Neg: true, SkillTypeList: []SkillTypeID{SkillTypeSummonsTotem, SkillTypeRemoteMined, SkillTypeTrapped, SkillTypeTriggered}}, &CondTag{Neg: true, Var: "usedByMirage"})}
	}),
	`attacks you use yourself repeat an additional time`: modList{modf("RepeatCount", Base, Num(1), FlagAttack, KeywordNone, &SkillTypeTag{Neg: true, SkillTypeList: []SkillTypeID{SkillTypeSummonsTotem, SkillTypeRemoteMined, SkillTypeTrapped, SkillTypeTriggered}}, &CondTag{Neg: true, Var: "usedByMirage"}, &CondTag{VarList: []string{"averageRepeat", "alwaysFinalRepeat"}})},
	`final repeat of attack skills deals ([0-9]+)% more damage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("RepeatFinalDamage", More, Num(c.n(1)), FlagNone, KeywordAttack)}
	}),
	`non-travel attack skills repeat an additional time`: modList{modf("RepeatCount", Base, Num(1), FlagNone, KeywordAttack, &SkillTypeTag{SkillType: SkillTypeTravel, Neg: true}, &CondTag{VarList: []string{"averageRepeat", "alwaysFinalRepeat"}})},
	`viper strike and pestilent strike deal ([0-9]+)% increased attack damage per frenzy charge`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagAttack, KeywordNone, &MultiplierTag{Var: "FrenzyCharge"}, &SkillNameTag{SkillNameList: []string{"Viper Strike", "Pestilent Strike"}, IncludeTransfigured: true})}
	}),
	`shield charge and chain hook have ([0-9]+)% increased attack speed per ([0-9]+) rampage kills`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Speed", Inc, Num(c.n(1)), FlagAttack, KeywordNone, &MultiplierTag{Var: "Rampage", Div: opt(c.n(2)), Limit: opt(1000 / c.n(2)), LimitTotal: true}, &SkillNameTag{SkillNameList: []string{"Shield Charge", "Chain Hook"}, IncludeTransfigured: true})}
	}),
	`tectonic slam and infernal blow deal ([0-9]+)% increased attack damage per ([0-9]+) armour`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Damage", Inc, Num(c.n(1)), FlagAttack, KeywordNone, &StatTag{StatKind: TagPerStat, Stat: "Armour", Div: opt(c.n(2))}, &SkillNameTag{SkillNameList: []string{"Tectonic Slam", "Infernal Blow"}, IncludeTransfigured: true})}
	}),
	`frozen sweep deals ([0-9]+)% less damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", More, Num(-c.n(1)), &SkillNameTag{SkillName: "Frozen Sweep", IncludeTransfigured: true})}
	}),
	`ice trap and lightning trap damage penetrates ([0-9]+)% of enemy elemental resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningPenetration", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Ice Trap", "Lightning Trap"}, IncludeTransfigured: true}), mod("ColdPenetration", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Ice Trap", "Lightning Trap"}, IncludeTransfigured: true}), mod("FirePenetration", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Ice Trap", "Lightning Trap"}, IncludeTransfigured: true})}
	}),
	`volatile dead and cremation penetrate ([0-9]+)% fire resistance per ([0-9]+) dexterity`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FirePenetration", Base, Num(c.n(1)), &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(2))}, &SkillNameTag{SkillNameList: []string{"Volatile Dead", "Cremation"}, IncludeTransfigured: true})}
	}),
	`regenerate ([0-9]+) mana per second while any enemy is in your righteous fire or scorching ray`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ManaRegen", Base, Num(c.n(1)), &CondTag{Var: "InRFOrScorchingRay"})}
	}),
	`\+([0-9]+)% to wave of conviction damage over time multiplier per ([0-9.]+) seconds of duration expired`:               modFn(func(c caps) []*Mod { return []*Mod{mod("WaveOfConvictionDurationDotMulti", Inc, Num(c.n(1)))} }),
	`when an enemy hit deals elemental damage to you, their resistance to those elements becomes zero for ([0-9]+) seconds`: modList{flag("Condition:HaveTrickstersSmile")},
	// Conditional Player Quantity / Rarity
	`([0-9]+)% increased quantity of items dropped by slain normal enemies`: modFn(func(c caps) []*Mod { return []*Mod{mod("LootQuantityNormalEnemies", Inc, Num(c.n(1)))} }),
	`([0-9]+)% increased rarity of items dropped by slain magic enemies`:    modFn(func(c caps) []*Mod { return []*Mod{mod("LootRarityMagicEnemies", Inc, Num(c.n(1)))} }),
	// Pantheon: Soul of Tukohama support
	`while stationary, gain ([0-9.]+)% of life regenerated per second every second, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(1)), &MultiplierTag{Var: "StationarySeconds", Limit: opt(c.n(2)), LimitTotal: true}, &CondTag{Var: "Stationary"})}
	}),
	// Pantheon: Soul of Ryslatha support
	`life flasks gain ([0-9]+) charges? every ([0-9]+) seconds if you haven't used a life flask recently`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeFlaskChargesGenerated", Base, Num(c.n(1)/c.n(2)), &CondTag{Var: "UsingLifeFlask", Neg: true})}
	}),
	// Skill-specific enchantment modifiers
	`([0-9]+)% increased decoy totem life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("TotemLife", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Decoy Totem"})}
	}),
	`([0-9]+)% increased ice spear critical strike chance in second form`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritChance", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Ice Spear", IncludeTransfigured: true}, &SkillPartTag{PartList: []float64{2, 4}})}
	}),
	`shock nova ring deals ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Shock Nova", IncludeTransfigured: true}, &SkillPartTag{Part: opt(1)})}
	}),
	`enemies affected by bear trap take ([0-9]+)% increased damage from trap or mine hits`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("EnemyModifier", List, ModRef{Mod: mod("TrapMineDamageTaken", Inc, Num(c.n(1)), &GlobalEffectTag{EffectType: "Debuff"})})}, &SkillNameTag{SkillName: "Bear Trap", IncludeTransfigured: true})}
	}),
	`blade vortex has \+([0-9]+)% to critical strike multiplier for each blade`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CritMultiplier", Base, Num(c.n(1)), &MultiplierTag{Var: "BladeVortexBlade"}, &SkillNameTag{SkillName: "Blade Vortex", IncludeTransfigured: true})}
	}),
	`burning arrow has ([0-9]+)% increased debuff effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DebuffEffect", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Burning Arrow"})}
	}),
	`double strike has a ([0-9]+)% chance to deal double damage to bleeding enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DoubleDamageChance", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}, &SkillNameTag{SkillName: "Double Strike", IncludeTransfigured: true})}
	}),
	`frost bomb has ([0-9]+)% increased debuff duration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SecondaryDuration", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Frost Bomb"})}
	}),
	`incinerate has \+([0-9]+) to maximum stages`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Multiplier:IncinerateMaxStages", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Incinerate"})}
	}),
	`perforate creates \+([0-9]+) spikes?`: modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:PerforateMaxSpikes", Base, Num(c.n(1)))} }),
	`scourge arrow has ([0-9]+)% chance to poison per stage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PoisonChance", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Scourge Arrow", IncludeTransfigured: true}, &MultiplierTag{Var: "ScourgeArrowStage"})}
	}),
	`winter orb has \+([0-9]+) maximum stages`: modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:WinterOrbMaxStages", Base, Num(c.n(1)))} }),
	`summoned holy relics have ([0-9]+)% increased buff effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("BuffEffect", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Summon Holy Relic", IncludeTransfigured: true})}
	}),
	`\+([0-9]+) to maximum virulence`: modFn(func(c caps) []*Mod { return []*Mod{mod("Multiplier:VirulenceStacksMax", Base, Num(c.n(1)))} }),
	`winter orb has ([0-9]+)% increased area of effect per stage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Winter Orb"}, &MultiplierTag{Var: "WinterOrbStage"})}
	}),
	`wintertide brand has \+([0-9]+) to maximum stages`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Multiplier:WintertideBrandMaxStages", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Wintertide Brand"})}
	}),
	`wave of conviction's exposure applies (-[0-9]+)% elemental resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "purge_expose_resist_%_matching_highest_element_damage", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Wave of Conviction"})}
	}),
	`wave of conviction's exposure applies an extra (-[0-9]+)% to elemental resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "purge_expose_resist_%_matching_highest_element_damage", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Wave of Conviction"})}
	}),
	`arcane cloak spends an additional ([0-9]+)% of current mana`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "arcane_cloak_consume_%_of_mana", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Arcane Cloak"})}
	}),
	`arcane cloak grants life regeneration equal to ([0-9]+)% of mana spent per second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: modf("LifeRegen", Base, Num(c.n(1)/100), FlagNone, KeywordNone, &MultiplierTag{Var: "ArcaneCloakConsumedMana"}, &GlobalEffectTag{EffectType: "Buff"})}, &SkillNameTag{SkillName: "Arcane Cloak"})}
	}),
	`caustic arrow has ([0-9]+)% chance to inflict withered on hit for ([0-9]+) seconds base duration`: modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("Condition:CanWither")}, &SkillNameTag{SkillName: "Caustic Arrow", IncludeTransfigured: true})},
	`venom gyre has a ([0-9]+)% chance to inflict withered for ([0-9]+) seconds on hit`:                modList{mod("ExtraSkillMod", List, ModRef{Mod: flag("Condition:CanWither")}, &SkillNameTag{SkillName: "Venom Gyre"})},
	`sigil of power's buff also grants ([0-9]+)% increased critical strike chance per stage`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("CritChance", Inc, Num(c.n(1)), FlagNone, KeywordNone, &MultiplierTag{Var: "SigilOfPowerStage", Limit: opt(4)}, &GlobalEffectTag{EffectType: "Buff", EffectName: "Sigil of Power"})}
	}),
	`cobra lash chains ([0-9]+) additional times`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ChainCountMax", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Cobra Lash"})}
	}),
	`general's cry has ([+\-][0-9]) to maximum number of mirage warriors`: modFn(func(c caps) []*Mod { return []*Mod{mod("GeneralsCryDoubleMaxCount", Base, Num(c.n(1)))} }),
	`([+\-][0-9]) to maximum blade flurry stages`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Multiplier:BladeFlurryMaxStages", Base, Num(c.n(1))), mod("Multiplier:BladeFlurryofIncisionMaxStages", Base, Num(c.n(1)))}
	}),
	`steelskin buff can take ([0-9]+)% increased amount of damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "steelskin_damage_limit_+%", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Steelskin"})}
	}),
	`hydrosphere has ([0-9]+)% increased pulse frequency`: modFn(func(c caps) []*Mod { return []*Mod{mod("HydroSphereFrequency", Inc, Num(c.n(1)))} }),
	`void sphere has ([0-9]+)% increased pulse frequency`: modFn(func(c caps) []*Mod { return []*Mod{mod("VoidSphereFrequency", Inc, Num(c.n(1)))} }),
	`shield crush central wave has ([0-9]+)% more area of effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", More, Num(c.n(1)), &SkillNameTag{SkillName: "Shield Crush", IncludeTransfigured: true}, &SkillPartTag{Part: opt(2)})}
	}),
	`storm rain has ([0-9]+)% increased beam frequency`:                                 modFn(func(c caps) []*Mod { return []*Mod{mod("StormRainBeamFrequency", Inc, Num(c.n(1)))} }),
	`voltaxic burst deals ([0-9]+)% increased damage per ([0-9.]+) seconds of duration`: modFn(func(c caps) []*Mod { return []*Mod{mod("VoltaxicDurationIncDamage", Inc, Num(c.n(1)))} }),
	`earthquake deals ([0-9]+)% increased damage per ([0-9.]+) seconds duration`:        modFn(func(c caps) []*Mod { return []*Mod{mod("EarthquakeDurationIncDamage", Inc, Num(c.n(1)))} }),
	`consecrated ground from holy flame totem applies ([0-9]+)% increased damage taken to enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod("DamageTakenConsecratedGround", Inc, Num(c.n(1)), &CondTag{Var: "OnConsecratedGround"})})}
	}),
	`consecrated ground from purifying flame applies ([0-9]+)% increased damage taken to enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "consecrated_ground_enemy_damage_taken_+%", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Purifying Flame", IncludeTransfigured: true})}
	}),
	`enemies drenched by hydrosphere have cold and lightning exposure, applying (-[0-9]+)% to resistances`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "water_sphere_cold_lightning_exposure_%", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Hydrosphere"})}
	}),
	`frost shield has \+([0-9]+) to maximum life per stage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "frost_globe_health_per_stage", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Frost Shield"})}
	}),
	`flame wall grants ([0-9]+) to ([0-9]+) added fire damage to projectiles`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "flame_wall_minimum_added_fire_damage", Value: c.v(1)}, &SkillNameTag{SkillName: "Flame Wall"}), mod("ExtraSkillStat", List, DataRef{Key: "flame_wall_maximum_added_fire_damage", Value: c.v(2)}, &SkillNameTag{SkillName: "Flame Wall"})}
	}),
	`plague bearer buff grants \+([0-9]+)% to poison damage over time multiplier while infecting`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "corrosive_shroud_poison_dot_multiplier_+_while_aura_active", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Plague Bearer"})}
	}),
	`([0-9]+)% increased lightning trap lightning ailment effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillStat", List, DataRef{Key: "shock_effect_+%", Value: Num(c.n(1))}, &SkillNameTag{SkillName: "Lightning Trap", IncludeTransfigured: true})}
	}),
	`wild strike's beam chains an additional ([0-9]+) times`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ChainCountMax", Base, Num(c.n(1)))}, &SkillNameTag{SkillName: "Wild Strike", IncludeTransfigured: true}, &SkillPartTag{Part: opt(4)})}
	}),
	`energy blades have ([0-9]+)% increased attack speed`: modFn(func(c caps) []*Mod { return []*Mod{mod("EnergyBladeAttackSpeed", Inc, Num(c.n(1)))} }),
	`ensnaring arrow has ([0-9]+)% increased debuff effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DebuffEffect", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Ensnaring Arrow"})}
	}),
	`unearth spawns corpses with ([+\-][0-9]) level`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("CorpseLevel", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Unearth"})}
	}),
	`seismic trap releases an additional wave`:        modList{mod("MaximumWaves", Base, Num(1), &SkillNameTag{SkillName: "Seismic Trap", IncludeTransfigured: true})},
	`lightning spire trap strikes an additional area`: modList{mod("MaximumWaves", Base, Num(1), &SkillNameTag{SkillName: "Lightning Spire Trap", IncludeTransfigured: true})},
	`explosive trap causes ([0-9]+) additional smaller explosions`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("SmallExplosions", Base, Num(c.n(1)), &SkillNameTag{SkillNameList: []string{"Explosive Trap", "Explosive Trap of Swells"}})}
	}),
	`frozen sweep deals ([0-9]+)% increased damage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Damage", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Frozen Sweep", IncludeTransfigured: true})}
	}),
	`([0-9]+)% increased attack speed with snipe`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Speed", Inc, Num(c.n(1)), FlagAttack, KeywordNone, &SkillNameTag{SkillName: "Snipe"})}
	}),
	`\+([0-9]+) to maximum snipe stages`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Multiplier:SnipeStagesMax", Base, Num(c.n(1)), FlagNone, KeywordNone, &GlobalEffectTag{EffectType: "Buff", Unscalable: true})}
	}),
	`chain hook has \+([0-9.]+) metres? to radius per ([0-9]+) rage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Base, Num(c.n(1)*10), &StatTag{StatKind: TagPerStat, Stat: "Rage", Div: opt(c.n(2))}, &SkillNameTag{SkillName: "Chain Hook", IncludeTransfigured: true})}
	}),
	`\+([0-9.]+) metres? to discharge radius`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Base, Num(c.n(1)*10), &SkillNameTag{SkillName: "Discharge", IncludeTransfigured: true})}
	}),
	// Alternate Quality
	`quality does not increase physical damage`: modList{mod("AlternateQualityWeapon", Base, Num(1))},
	`([0-9]+)% increased critical strike chance per 4% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AlternateQualityLocalCritChancePer4Quality", Inc, Num(c.n(1)))}
	}),
	`grants ([0-9]+)% increased accuracy per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Accuracy", Inc, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`([0-9]+)% increased attack speed per 8% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AlternateQualityLocalAttackSpeedPer8Quality", Inc, Num(c.n(1)))}
	}),
	`\+([0-9]+) weapon range per 10% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AlternateQualityLocalWeaponRangePer10Quality", Base, Num(c.n(1)))}
	}),
	`\+([0-9.]+) metres? to weapon range per 10% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AlternateQualityLocalWeaponRangePer10Quality", Base, Num(c.n(1)*10))}
	}),
	`grants ([0-9]+)% increased elemental damage per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ElementalDamage", Inc, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants ([0-9]+)% increased area of effect per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("AreaOfEffect", Inc, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`quality does not increase defences`: modList{mod("AlternateQualityArmour", Base, Num(1))},
	`grants \+([0-9]+) to maximum life per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Life", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+) to maximum mana per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Mana", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+) to strength per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Str", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+) to dexterity per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Dex", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+) to intelligence per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Int", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+)% to fire resistance per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("FireResist", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+)% to cold resistance per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ColdResist", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`grants \+([0-9]+)% to lightning resistance per ([0-9]+)% quality`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LightningResist", Base, Num(c.n(1)), &MultiplierTag{Var: "QualityOn{SlotName}", Div: opt(c.n(2))})}
	}),
	`\+([0-9]+)% to quality`: modFn(func(c caps) []*Mod { return []*Mod{mod("Quality", Base, Num(c.n(1)))} }),
	`infernal blow debuff deals an additional ([0-9]+)% of damage per charge`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("DebuffEffect", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Infernal Blow", IncludeTransfigured: true})}
	}),
	// Legion modifiers
	`passives in radius are conquered by the ([^0-9]+)`: modList{},
	`passives affected are conquered by the abyssal`:    modList{},
	`historic`: modList{},
	// Tattoos
	`\+([0-9]+) to maximum life per allocated journey tattoo of the body`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Life", Base, Num(c.n(1)), &MultiplierTag{Var: "JourneyTattooBody"}), mod("Multiplier:JourneyTattooBody", Base, Num(1))}
	}),
	`\+([0-9]+) to maximum energy shield per allocated journey tattoo of the soul`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShield", Base, Num(c.n(1)), &MultiplierTag{Var: "JourneyTattooSoul"}), mod("Multiplier:JourneyTattooSoul", Base, Num(1))}
	}),
	`\+([0-9]+) to maximum mana per allocated journey tattoo of the mind`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Mana", Base, Num(c.n(1)), &MultiplierTag{Var: "JourneyTattooMind"}), mod("Multiplier:JourneyTattooMind", Base, Num(1))}
	}),
	// Display-only modifiers
	`extra gore`: modList{},
	`prefixes:`:  modList{},
	`suffixes:`:  modList{},
	`while your passive skill tree connects to a class' starting location, you gain:`:          modList{},
	`socketed lightning spells [hd][ae][va][el] ([0-9]+)% increased spell damage if triggered`: modList{},
	`manifeste?d? dancing dervishe?s? disables both weapon slots`:                              modList{},
	`manifeste?d? dancing dervishe?s? dies? when rampage ends`:                                 modList{},
	`survival`: modList{},
	`you can have two different banners at the same time`:                modList{},
	`[+\-]([0-9]+) prefix modifiers? allowed`:                            modList{},
	`[+\-]([0-9]+) suffix modifiers? allowed`:                            modList{},
	`can have a second enchantment modifier`:                             modList{},
	`can have ([0-9]+) additional enchantment modifiers`:                 modList{},
	`this item can be anointed by cassia`:                                modList{},
	`can be anointed`:                                                    modList{},
	`implicit modifiers cannot be changed`:                               modList{},
	`has a crucible passive skill tree`:                                  modList{},
	`has elder, shaper and all conqueror influences`:                     modList{},
	`has a two handed sword crucible passive skill tree`:                 modList{},
	`has a crucible passive skill tree with only support passive skills`: modList{},
	`crucible passive skill tree is removed if this modifier is removed`: modList{},
	`all sockets are white`:                                              modList{},
	`cannot roll ([a-zA-Z]+) modifiers`:                                  modList{},
	`cannot roll modifiers of non-([a-zA-Z]+) damage types`:              modList{},
	`every ([0-9]+) seconds, regenerate ([0-9]+)% of life over one second`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(2)), &CondTag{Var: "LifeRegenBurstFull"}), mod("LifeRegenPercent", Base, Num(c.n(2)/c.n(1)), &CondTag{Var: "LifeRegenBurstAvg"})}
	}),
	`every ([0-9]+) seconds, regenerate ([0-9]+)% of life over one second if ([0-9]+) ([a-zA-Z]+) items are equipped`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("LifeRegenPercent", Base, Num(c.n(2)), &CondTag{Var: "LifeRegenBurstFull"}, &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(4)) + "Item", Threshold: opt(c.n(3))}), mod("LifeRegenPercent", Base, Num(c.n(2)/c.n(1)), &CondTag{Var: "LifeRegenBurstAvg"}, &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(4)) + "Item", Threshold: opt(c.n(3))})}
	}),
	`take no extra damage from critical strikes`:                                            modList{mod("ReduceCritExtraDamage", Base, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true})},
	`take no extra damage from critical strikes if you have a magic ring in left slot`:      modList{mod("ReduceCritExtraDamage", Base, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &CondTag{Var: "MagicItemInRing 1"})},
	`take no extra damage from critical strikes if energy shield recharge started recently`: modList{mod("ReduceCritExtraDamage", Base, Num(100), &CondTag{Var: "EnergyShieldRechargeRecently"})},
	`you take ([0-9]+)% reduced extra damage from critical strikes while affected by determination`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReduceCritExtraDamage", Base, Num(c.n(1)), &CondTag{Var: "AffectedByDetermination"})}
	}),
	`you take ([0-9]+)% reduced extra damage from critical strikes`:   modFn(func(c caps) []*Mod { return []*Mod{mod("ReduceCritExtraDamage", Base, Num(c.n(1)))} }),
	`you take ([0-9]+)% increased extra damage from critical strikes`: modFn(func(c caps) []*Mod { return []*Mod{mod("ReduceCritExtraDamage", Base, Num(-c.n(1)))} }),
	`you take ([0-9]+)% reduced extra damage from critical strikes while you have no power charges`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReduceCritExtraDamage", Base, Num(c.n(1)), &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", Threshold: opt(0), Upper: true})}
	}),
	`you take ([0-9]+)% reduced extra damage from critical strikes by poisoned enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReduceCritExtraDamage", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"})}
	}),
	`you take ([0-9]+)% reduced extra damage from critical strikes by cursed enemies`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReduceCritExtraDamage", Base, Num(c.n(1)), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})}
	}),
	`nearby allies have ([0-9]+)% chance to block attack damage per ([0-9]+) strength you have`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraAura", List, ModRef{Mod: mod("BlockChance", Base, Num(c.n(1))), OnlyAllies: true}, &StatTag{StatKind: TagPerStat, Stat: "Str", Div: opt(c.n(2))})}
	}),
	`physical skills have ([0-9]+)% increased duration per ([0-9]+) intelligence`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("Duration", Inc, Num(c.n(1)), FlagNone, KeywordPhysical, &StatTag{StatKind: TagPerStat, Stat: "Int", Div: opt(c.n(2))})}
	}),
	`y?o?u?r? ?maximum energy shield is equal to ([0-9]+)% of y?o?u?r? ?maximum life`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnergyShield", Override, Num(1), &StatTag{StatKind: TagPercentStat, Stat: "Life", Percent: opt(c.n(1))})}
	}),
	`immun[ei]t?y? to elemental ailments while bleeding`: modList{flag("ElementalAilmentImmune", &CondTag{Var: "Bleeding"})},
	`mana is increased by ([0-9]+)% of overcapped lightning resistance`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("ManaIncreasedByOvercappedLightningRes"), mod("Mana", Inc, Num(c.n(1)/100), &StatTag{StatKind: TagPerStat, Stat: "LightningResistOverCap"})}
	}),
	// handled in item parsing
	`[0-9]+% [ir][ne][cd][ru][ec][ae][sd]e?d? ?[a-zA-Z \t\n\v\f\r]* modifier magnitudes`: modList{},
	`[0-9]+% [ir][ne][cd][ru][ec][ae][sd]e?d? effect of [sp][ur][fe]fixes`:               modList{},
	`[a-zA-Z \t\n\v\f\r]* modifier magnitudes are doubled`:                               modList{},
}
