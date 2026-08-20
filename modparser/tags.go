package modparser

// List of modifier tags — ModParser.lua:1319.
var modTagList = map[string]any{
	`on enemies`:                               d(),
	`while active`:                             d(),
	`for ([0-9]+) seconds`:                     d(),
	`when you hit a unique enemy`:              Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"}},
	` on critical strike`:                      Tag{"tag": Tag{"type": "Condition", "var": "CriticalStrike"}},
	`from critical strikes`:                    Tag{"tag": Tag{"type": "Condition", "var": "CriticalStrike"}},
	`with critical strikes`:                    Tag{"tag": Tag{"type": "Condition", "var": "CriticalStrike"}},
	`by enemies killed with a critical strike`: Tag{"tagList": []any{Tag{"type": "Condition", "var": "CritRecently"}, Tag{"type": "Condition", "var": "KilledRecently"}}},
	`while affected by auras you cast`:         Tag{"tag": Tag{"type": "Condition", "var": "AffectedByAura"}},
	`for you and nearby allies`:                Tag{"newAura": true},
	`to you and allies`:                        Tag{"newAura": true},
	// Multipliers
	`per power charge`:         Tag{"tag": Tag{"type": "Multiplier", "var": "PowerCharge"}},
	`per frenzy charge`:        Tag{"tag": Tag{"type": "Multiplier", "var": "FrenzyCharge"}},
	`per endurance charge`:     Tag{"tag": Tag{"type": "Multiplier", "var": "EnduranceCharge"}},
	`per brine charge`:         Tag{"tag": Tag{"type": "Multiplier", "var": "BrineCharge"}},
	`per siphoning charge`:     Tag{"tag": Tag{"type": "Multiplier", "var": "SiphoningCharge"}},
	`per spirit charge`:        Tag{"tag": Tag{"type": "Multiplier", "var": "SpiritCharge"}},
	`per challenger charge`:    Tag{"tag": Tag{"type": "Multiplier", "var": "ChallengerCharge"}},
	`per maximum power charge`: Tag{"tag": Tag{"type": "Multiplier", "var": "PowerChargeMax"}},
	`per gale force`:           Tag{"tag": Tag{"type": "Multiplier", "var": "GaleForce"}},
	`per intensity`:            Tag{"tag": Tag{"type": "Multiplier", "var": "Intensity"}},
	`per brand`:                Tag{"tag": Tag{"type": "Multiplier", "var": "ActiveBrand"}},
	`per brand, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "ActiveBrand", "globalLimit": c.n(1), "globalLimitKey": "ChipAway"}}
	}),
	`per blitz charge`:                       Tag{"tag": Tag{"type": "Multiplier", "var": "BlitzCharge"}},
	`per ghost shroud`:                       Tag{"tag": Tag{"type": "Multiplier", "var": "GhostShroud"}},
	`per crab barrier`:                       Tag{"tag": Tag{"type": "Multiplier", "var": "CrabBarrier"}},
	`per rage`:                               Tag{"tag": Tag{"type": "Multiplier", "var": "Rage"}},
	`per rage while you are not losing rage`: Tag{"tag": Tag{"type": "Multiplier", "var": "Rage"}},
	`per ([0-9]+) rage`:                      fn(func(c caps) any { return Tag{"tag": Tag{"type": "Multiplier", "var": "Rage", "div": c.n(1)}} }),
	`per mana burn`:                          Tag{"tag": Tag{"type": "Multiplier", "var": "ManaBurnStacks"}},
	`per mana burn on you`:                   Tag{"tag": Tag{"type": "Multiplier", "var": "ManaBurnStacks"}},
	`per mana burn, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "ManaBurnStacks", "limit": c.n(1), "limitTotal": true}}
	}),
	`per level`:                  Tag{"tag": Tag{"type": "Multiplier", "var": "Level"}},
	`per ([0-9]+) player levels`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "Multiplier", "var": "Level", "div": c.n(1)}} }),
	`per defiance`:               Tag{"tag": Tag{"type": "Multiplier", "var": "Defiance"}},
	`per ([0-9]+)% ([a-zA-Z]+) effect on enemy`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": firstToUpper(c.s(2)) + "Effect", "div": c.n(1), "actor": "enemy"}}
	}),
	`for each equipped normal item`:          Tag{"tag": Tag{"type": "Multiplier", "var": "NormalItem"}},
	`for each normal item equipped`:          Tag{"tag": Tag{"type": "Multiplier", "var": "NormalItem"}},
	`for each normal item you have equipped`: Tag{"tag": Tag{"type": "Multiplier", "var": "NormalItem"}},
	`for each equipped magic item`:           Tag{"tag": Tag{"type": "Multiplier", "var": "MagicItem"}},
	`for each magic item equipped`:           Tag{"tag": Tag{"type": "Multiplier", "var": "MagicItem"}},
	`for each magic item you have equipped`:  Tag{"tag": Tag{"type": "Multiplier", "var": "MagicItem"}},
	`for each equipped rare item`:            Tag{"tag": Tag{"type": "Multiplier", "var": "RareItem"}},
	`for each rare item equipped`:            Tag{"tag": Tag{"type": "Multiplier", "var": "RareItem"}},
	`for each rare item you have equipped`:   Tag{"tag": Tag{"type": "Multiplier", "var": "RareItem"}},
	`for each equipped unique item`:          Tag{"tag": Tag{"type": "Multiplier", "var": "UniqueItem"}},
	`for each unique item equipped`:          Tag{"tag": Tag{"type": "Multiplier", "var": "UniqueItem"}},
	`for each unique item you have equipped`: Tag{"tag": Tag{"type": "Multiplier", "var": "UniqueItem"}},
	`per elder item equipped`:                Tag{"tag": Tag{"type": "Multiplier", "var": "ElderItem"}},
	`per shaper item equipped`:               Tag{"tag": Tag{"type": "Multiplier", "var": "ShaperItem"}},
	`per elder or shaper item equipped`:      Tag{"tag": Tag{"type": "Multiplier", "var": "ShaperOrElderItem"}},
	`if ([0-9]+) ([a-zA-Z]+) items are equipped`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": firstToUpper(c.s(2)) + "Item", "threshold": c.n(1)}}
	}),
	`for each corrupted item equipped`:   Tag{"tag": Tag{"type": "Multiplier", "var": "CorruptedItem"}},
	`for each equipped corrupted item`:   Tag{"tag": Tag{"type": "Multiplier", "var": "CorruptedItem"}},
	`for each uncorrupted item equipped`: Tag{"tag": Tag{"type": "Multiplier", "var": "NonCorruptedItem"}},
	`per equipped claw`:                  Tag{"tag": Tag{"type": "Multiplier", "var": "ClawItem"}},
	`per equipped dagger`:                Tag{"tag": Tag{"type": "Multiplier", "var": "DaggerItem"}},
	`per equipped axe`:                   Tag{"tag": Tag{"type": "Multiplier", "var": "AxeItem"}},
	`per equipped ring`:                  Tag{"tag": Tag{"type": "Multiplier", "var": "RingItem"}},
	`per equipped flask`:                 Tag{"tag": Tag{"type": "Multiplier", "var": "FlaskItem"}},
	`per equipped sword`:                 Tag{"tag": Tag{"type": "Multiplier", "var": "SwordItem"}},
	`per equipped jewel`:                 Tag{"tag": Tag{"type": "Multiplier", "var": "JewelItem"}},
	`per equipped mace`:                  Tag{"tag": Tag{"type": "Multiplier", "var": "MaceItem"}},
	`per equipped sceptre`:               Tag{"tag": Tag{"type": "Multiplier", "var": "SceptreItem"}},
	`per equipped wand`:                  Tag{"tag": Tag{"type": "Multiplier", "var": "WandItem"}},
	`per claw`:                           Tag{"tag": Tag{"type": "Multiplier", "var": "ClawItem"}},
	`per dagger`:                         Tag{"tag": Tag{"type": "Multiplier", "var": "DaggerItem"}},
	`per axe`:                            Tag{"tag": Tag{"type": "Multiplier", "var": "AxeItem"}},
	`per ring`:                           Tag{"tag": Tag{"type": "Multiplier", "var": "RingItem"}},
	`per flask`:                          Tag{"tag": Tag{"type": "Multiplier", "var": "FlaskItem"}},
	`per sword`:                          Tag{"tag": Tag{"type": "Multiplier", "var": "SwordItem"}},
	`per jewel`:                          Tag{"tag": Tag{"type": "Multiplier", "var": "JewelItem"}},
	`per mace`:                           Tag{"tag": Tag{"type": "Multiplier", "var": "MaceItem"}},
	`per sceptre`:                        Tag{"tag": Tag{"type": "Multiplier", "var": "SceptreItem"}},
	`per wand`:                           Tag{"tag": Tag{"type": "Multiplier", "var": "WandItem"}},
	`per abyssa?l? jewel affecting you`:  Tag{"tag": Tag{"type": "Multiplier", "var": "AbyssJewel"}},
	`for each herald b?u?f?f?s?k?i?l?l? ?affecting you`:    Tag{"tag": Tag{"type": "Multiplier", "var": "Herald"}},
	`for each of your aura or herald skills affecting you`: Tag{"tag": Tag{"type": "Multiplier", "varList": []any{"Herald", "AuraAffectingSelf"}}},
	`for each type of abyssa?l? jewel affecting you`:       Tag{"tag": Tag{"type": "Multiplier", "var": "AbyssJewelType"}},
	`per (.+) eye jewel affecting you, up to a maximum of \+?([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": firstToUpper(c.s(1)) + "EyeJewel", "limit": c.n(2), "limitTotal": true}}
	}),
	`per sextant affecting the area`: Tag{"tag": Tag{"type": "Multiplier", "var": "Sextant"}},
	`per buff on you`:                Tag{"tag": Tag{"type": "Multiplier", "var": "BuffOnSelf"}},
	`per hit suppressed recently`:    Tag{"tag": Tag{"type": "Multiplier", "var": "HitsSuppressedRecently"}},
	`per curse on enemy`:             Tag{"tag": Tag{"type": "Multiplier", "var": "CurseOnEnemy"}},
	`for each curse on enemy`:        Tag{"tag": Tag{"type": "Multiplier", "var": "CurseOnEnemy"}},
	`for each curse on the enemy`:    Tag{"tag": Tag{"type": "Multiplier", "var": "CurseOnEnemy"}},
	`per curse on you`:               Tag{"tag": Tag{"type": "Multiplier", "var": "CurseOnSelf"}},
	`per poison on you`:              Tag{"tag": Tag{"type": "Multiplier", "var": "PoisonStack"}},
	`for each poison on you`:         Tag{"tag": Tag{"type": "Multiplier", "var": "PoisonStack"}},
	`for each poison on you up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "PoisonStack", "limit": c.n(1), "limitTotal": true}}
	}),
	`per poison on you, up to ([0-9]+) per second`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "PoisonStack", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each poison you have inflicted recently`: Tag{"tag": Tag{"type": "Multiplier", "var": "PoisonAppliedRecently"}},
	`per withered debuff on enemy`:                Tag{"tag": Tag{"type": "Multiplier", "var": "WitheredStack", "actor": "enemy", "limit": 15}},
	`for each poison you have inflicted recently, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "PoisonAppliedRecently", "globalLimit": c.n(1), "globalLimitKey": "DurationPerPoisonRecently"}}
	}),
	`for each time you have shocked a non-shocked enemy recently, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "ShockedNonShockedEnemyRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each shocked enemy you've killed recently`: Tag{"tag": Tag{"type": "Multiplier", "var": "ShockedEnemyKilledRecently"}},
	`per enemy killed recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "EnemyKilledRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`per ([0-9]+) rampage kills`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "Rampage", "div": c.n(1), "limit": 1000 / c.n(1), "limitTotal": true}}
	}),
	`per minion, up to ?a? ?m?a?x?i?m?u?m? ?o?f? ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "SummonedMinion", "limit": c.n(1), "limitTotal": true}}
	}),
	`per minion from your non-vaal skills`: Tag{"tag": Tag{"type": "Multiplier", "var": "NonVaalSummonedMinion"}},
	`per minion`:                           Tag{"tag": Tag{"type": "Multiplier", "var": "SummonedMinion"}},
	`for each enemy you or your minions have killed recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "varList": []any{"EnemyKilledRecently", "EnemyKilledByMinionsRecently"}, "limit": c.n(1), "limitTotal": true}}
	}),
	`for each enemy you or your minions have killed recently, up to ([0-9]+)% per second`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "varList": []any{"EnemyKilledRecently", "EnemyKilledByMinionsRecently"}, "limit": c.n(1), "limitTotal": true}}
	}),
	`for each ([0-9]+) total mana y?o?u? ?h?a?v?e? ?spent recently`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "ManaSpentRecently", "div": c.n(1)}}
	}),
	`for each ([0-9]+) total mana you have spent recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "ManaSpentRecently", "div": c.n(1), "limit": c.n(2), "limitTotal": true}}
	}),
	`per ([0-9]+) mana spent recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "ManaSpentRecently", "div": c.n(1), "limit": c.n(2), "limitTotal": true}}
	}),
	`for each time you've blocked in the past 10 seconds`: Tag{"tag": Tag{"type": "Multiplier", "var": "BlockedPast10Sec"}},
	`per enemy killed by you or your totems recently`:     Tag{"tag": Tag{"type": "Multiplier", "varList": []any{"EnemyKilledRecently", "EnemyKilledByTotemsRecently"}}},
	`per nearby enemy, up to \+?([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "NearbyEnemies", "limit": c.n(1), "limitTotal": true}}
	}),
	`per enemy in close range`:                               Tag{"tagList": []any{Tag{"type": "Condition", "var": "AtCloseRange"}, Tag{"type": "Multiplier", "var": "NearbyEnemies"}}},
	`per red socket`:                                         Tag{"tag": Tag{"type": "Multiplier", "var": "RedSocketIn{SlotName}"}},
	`per green socket on main hand weapon`:                   Tag{"tag": Tag{"type": "Multiplier", "var": "GreenSocketInWeapon 1"}},
	`per green socket on`:                                    Tag{"tag": Tag{"type": "Multiplier", "var": "GreenSocketInWeapon 1"}},
	`per red socket on main hand weapon`:                     Tag{"tag": Tag{"type": "Multiplier", "var": "RedSocketInWeapon 1"}},
	`per red socket on equipped staff`:                       Tag{"tagList": []any{Tag{"type": "Multiplier", "var": "RedSocketInWeapon 1"}, Tag{"type": "Condition", "var": "UsingStaff"}}},
	`per blue socket on equipped staff`:                      Tag{"tagList": []any{Tag{"type": "Multiplier", "var": "BlueSocketInWeapon 1"}, Tag{"type": "Condition", "var": "UsingStaff"}}},
	`per green socket`:                                       Tag{"tag": Tag{"type": "Multiplier", "var": "GreenSocketIn{SlotName}"}},
	`per blue socket`:                                        Tag{"tag": Tag{"type": "Multiplier", "var": "BlueSocketIn{SlotName}"}},
	`per white socket`:                                       Tag{"tag": Tag{"type": "Multiplier", "var": "WhiteSocketIn{SlotName}"}},
	`for each unlinked socket in equipped two handed weapon`: Tag{"tagList": []any{Tag{"type": "Multiplier", "var": "UnlinkedSocketInWeapon 1"}, Tag{"type": "Condition", "var": "UsingTwoHandedWeapon"}}},
	`for each empty red socket on any equipped item`:         Tag{"tag": Tag{"type": "Multiplier", "var": "EmptyRedSocketsInAnySlot"}},
	`for each empty green socket on any equipped item`:       Tag{"tag": Tag{"type": "Multiplier", "var": "EmptyGreenSocketsInAnySlot"}},
	`for each empty blue socket on any equipped item`:        Tag{"tag": Tag{"type": "Multiplier", "var": "EmptyBlueSocketsInAnySlot"}},
	`for each empty white socket on any equipped item`:       Tag{"tag": Tag{"type": "Multiplier", "var": "EmptyWhiteSocketsInAnySlot"}},
	`per socketed gem`:                                       Tag{"tag": Tag{"type": "Multiplier", "var": "SocketedGemsIn{SlotName}"}},
	`per socketed red gem`:                                   Tag{"tag": Tag{"type": "Multiplier", "var": "SocketedRedGemsIn{SlotName}"}},
	`per socketed green gem`:                                 Tag{"tag": Tag{"type": "Multiplier", "var": "SocketedGreenGemsIn{SlotName}"}},
	`per socketed blue gem`:                                  Tag{"tag": Tag{"type": "Multiplier", "var": "SocketedBlueGemsIn{SlotName}"}},
	`per socketed murderous eye jewel`:                       Tag{"tag": Tag{"type": "Multiplier", "var": "MurderousEyeJewelIn{SlotName}"}},
	`per socketed searching eye jewel`:                       Tag{"tag": Tag{"type": "Multiplier", "var": "SearchingEyeJewelIn{SlotName}"}},
	`per socketed hypnotic eye jewel`:                        Tag{"tag": Tag{"type": "Multiplier", "var": "HypnoticEyeJewelIn{SlotName}"}},
	`per socketed ghastly eye jewel`:                         Tag{"tag": Tag{"type": "Multiplier", "var": "GhastlyEyeJewelIn{SlotName}"}},
	`for each impale on enemy`:                               Tag{"tag": Tag{"type": "Multiplier", "var": "ImpaleStacks", "actor": "enemy"}},
	`per impale on enemy`:                                    Tag{"tag": Tag{"type": "Multiplier", "var": "ImpaleStacks", "actor": "enemy"}},
	`per grasping vine`:                                      Tag{"tag": Tag{"type": "Multiplier", "var": "GraspingVinesCount"}},
	`per fragile regrowth`:                                   Tag{"tag": Tag{"type": "Multiplier", "var": "FragileRegrowthCount"}},
	`per bark`:                                               Tag{"tag": Tag{"type": "Multiplier", "var": "BarkskinStacks"}},
	`per bark below maximum`:                                 Tag{"tag": Tag{"type": "Multiplier", "var": "MissingBarkskinStacks"}},
	`per allocated mastery passive skill`:                    Tag{"tag": Tag{"type": "Multiplier", "var": "AllocatedMastery"}},
	`per allocated notable passive skill`:                    Tag{"tag": Tag{"type": "Multiplier", "var": "AllocatedNotable"}},
	`for each different type of mastery you have allocated`:  Tag{"tag": Tag{"type": "Multiplier", "var": "AllocatedMasteryType"}},
	`per grand spectrum`:                                     Tag{"tag": Tag{"type": "Multiplier", "var": "GrandSpectrum"}},
	`per second you've been stationary, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "StationarySeconds", "limit": c.n(1), "limitTotal": true}}
	}),
	`per elemental ailment you've inflicted recently`: Tag{"tag": Tag{"type": "Multiplier", "var": "AppliedAilmentsRecently"}},
	// Per stat
	`per ([0-9]+)% of maximum mana they reserve`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "ManaReservedPercent", "div": c.n(1)}}
	}),
	`for each ([0-9]+)% of life reserved`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "LifeReservedPercent", "div": c.n(1)}}
	}),
	`per ([0-9]+) strength`:     fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Str", "div": c.n(1)}} }),
	`per dexterity`:             Tag{"tag": Tag{"type": "PerStat", "stat": "Dex"}},
	`per ([0-9]+) dexterity`:    fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Dex", "div": c.n(1)}} }),
	`per ([0-9]+) intelligence`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Int", "div": c.n(1)}} }),
	`per ([0-9]+) omniscience`:  fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Omni", "div": c.n(1)}} }),
	`per ([0-9]+) total attributes`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "statList": []any{"Str", "Dex", "Int"}, "div": c.n(1)}}
	}),
	`per ([0-9]+) of your lowest attribute`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "LowestAttribute", "div": c.n(1)}} }),
	`per ([0-9]+) reserved life`:            fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "LifeReserved", "div": c.n(1)}} }),
	`per ([0-9]+) unreserved maximum mana`:  fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ManaUnreserved", "div": c.n(1)}} }),
	`per ([0-9]+) unreserved maximum mana, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "ManaUnreserved", "div": c.n(1), "limit": c.n(2), "limitTotal": true}}
	}),
	`per ([0-9]+) armour`:         fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Armour", "div": c.n(1)}} }),
	`per ([0-9]+) evasion rating`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Evasion", "div": c.n(1)}} }),
	`per ([0-9]+) evasion rating, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "Evasion", "div": c.n(1), "limit": c.n(2), "limitTotal": true}}
	}),
	`per ([0-9]+) maximum energy shield`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShield", "div": c.n(1)}} }),
	`per ([0-9]+) player maximum energy shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShield", "div": c.n(1), "actor": "player"}}
	}),
	`per ([0-9]+) maximum life`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Life", "div": c.n(1)}} }),
	`per ([0-9]+) of maximum life or maximum mana, whichever is lower`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "LowestOfMaximumLifeAndMaximumMana", "div": c.n(1)}}
	}),
	`per ([0-9]+) player maximum life`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "Life", "div": c.n(1), "actor": "player"}}
	}),
	`per ([0-9]+) maximum mana`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Mana", "div": c.n(1)}} }),
	`per ([0-9]+) maximum mana, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "Mana", "div": c.n(1), "limit": c.n(2), "limitTotal": true}}
	}),
	`per ([0-9]+) maximum mana, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "Mana", "div": c.n(1), "limit": c.n(2), "limitTotal": true}}
	}),
	`per soul required`:            Tag{"tag": Tag{"type": "PerStat", "stat": "SoulCost"}},
	`per ([0-9]+) accuracy rating`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "Accuracy", "div": c.n(1)}} }),
	`per ([0-9]+)% block chance`:   fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "BlockChance", "div": c.n(1)}} }),
	`per ([0-9]+)% chance to block on equipped shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "ShieldBlockChance", "div": c.n(1)}}
	}),
	`per ([0-9]+)% chance to block attack damage`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "BlockChance", "div": c.n(1)}} }),
	`per ([0-9]+)% chance to block spell damage`:  fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "SpellBlockChance", "div": c.n(1)}} }),
	`per ([0-9]+) of the lowest of armour and evasion rating`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "LowestOfArmourAndEvasion", "div": c.n(1)}}
	}),
	`per ([0-9]+) energy shield on equipped gloves`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnGloves", "div": c.n(1)}}
	}),
	`per ([0-9]+) maximum energy shield on helmet`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnHelmet", "div": c.n(1)}}
	}),
	`per ([0-9]+) maximum energy shield on equipped helmet`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnHelmet", "div": c.n(1)}}
	}),
	`per ([0-9]+) energy shield on equipped helmet`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnHelmet", "div": c.n(1)}}
	}),
	`per ([0-9]+) energy shield on equipped boots`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnBoots", "div": c.n(1)}}
	}),
	`per ([0-9]+) energy shield on equipped body armour`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnBody Armour", "div": c.n(1)}}
	}),
	`per ([0-9]+) maximum energy shield on equipped shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnWeapon 2", "div": c.n(1)}}
	}),
	`per ([0-9]+) maximum energy shield on shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EnergyShieldOnWeapon 2", "div": c.n(1)}}
	}),
	`per ([0-9]+) evasion rating on equipped gloves`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnGloves", "div": c.n(1)}} }),
	`per ([0-9]+) evasion rating on equipped helmet`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnHelmet", "div": c.n(1)}} }),
	`per ([0-9]+) evasion on equipped boots`:         fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnBoots", "div": c.n(1)}} }),
	`per ([0-9]+) evasion on boots`:                  fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnBoots", "div": c.n(1)}} }),
	`per ([0-9]+) evasion rating on equipped boots`:  fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnBoots", "div": c.n(1)}} }),
	`per ([0-9]+) evasion rating on body armour`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnBody Armour", "div": c.n(1)}}
	}),
	`per ([0-9]+) evasion rating on equipped body armour`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnBody Armour", "div": c.n(1)}}
	}),
	`per ([0-9]+) evasion rating on equipped shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "EvasionOnWeapon 2", "div": c.n(1)}}
	}),
	`per ([0-9]+) armour on gloves`:          fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ArmourOnGloves", "div": c.n(1)}} }),
	`per ([0-9]+) armour on equipped gloves`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ArmourOnGloves", "div": c.n(1)}} }),
	`per ([0-9]+) armour on equipped helmet`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ArmourOnHelmet", "div": c.n(1)}} }),
	`per ([0-9]+) armour on equipped boots`:  fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ArmourOnBoots", "div": c.n(1)}} }),
	`per ([0-9]+) armour on equipped body armour`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "ArmourOnBody Armour", "div": c.n(1)}}
	}),
	`per ([0-9]+) armour on equipped shield`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ArmourOnWeapon 2", "div": c.n(1)}} }),
	`per ([0-9]+) armour or evasion rating on shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "statList": []any{"ArmourOnWeapon 2", "EvasionOnWeapon 2"}, "div": c.n(1)}}
	}),
	`per ([0-9]+) armour or evasion rating on equipped shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "statList": []any{"ArmourOnWeapon 2", "EvasionOnWeapon 2"}, "div": c.n(1)}}
	}),
	`per ([0-9]+)% cold resistance`:           fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ColdResist", "div": c.n(1)}} }),
	`per ([0-9]+)% fire resistance`:           fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "FireResist", "div": c.n(1)}} }),
	`per ([0-9]+)% lightning resistance`:      fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "LightningResist", "div": c.n(1)}} }),
	`per ([0-9]+)% chaos resistance`:          fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ChaosResist", "div": c.n(1)}} }),
	`per ([0-9]+)% cold resistance above 75%`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "ColdResistOver75", "div": c.n(1)}} }),
	`per ([0-9]+)% lightning resistance above 75%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "LightningResistOver75", "div": c.n(1)}}
	}),
	`per ([0-9]+)% fire resistance above 75%`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "PerStat", "stat": "FireResistOver75", "div": c.n(1)}} }),
	`per ([0-9]+)% fire, cold, or lightning resistance above 75%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "statList": []any{"FireResistOver75", "ColdResistOver75", "LightningResistOver75"}, "div": c.n(1)}}
	}),
	`per ([0-9]+) devotion`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "Devotion", "actor": "parent", "div": c.n(1)}}
	}),
	`per ([0-9]+)% missing fire resistance, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "MissingFireResist", "div": c.n(1), "globalLimit": c.n(2), "globalLimitKey": "ReplicaNebulisFire"}}
	}),
	`per ([0-9]+)% missing cold resistance, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "MissingColdResist", "div": c.n(1), "globalLimit": c.n(2), "globalLimitKey": "ReplicaNebulisCold"}}
	}),
	`per ([0-9]+)% missing fire, cold, or lightning resistance, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "statList": []any{"MissingFireResist", "MissingColdResist", "MissingLightningResist"}, "div": c.n(1), "globalLimit": c.n(2), "globalLimitKey": "ReplicaNebulisCold"}}
	}),
	`per endurance, frenzy or power charge`: Tag{"tag": Tag{"type": "PerStat", "stat": "TotalCharges"}},
	`per fortification`:                     Tag{"tag": Tag{"type": "PerStat", "stat": "FortificationStacks"}},
	`per two fortification on you`:          Tag{"tag": Tag{"type": "PerStat", "stat": "FortificationStacks", "div": 2, "actor": "player"}},
	`per fortification above 20`:            Tag{"tag": Tag{"type": "PerStat", "stat": "FortificationStacksOver20"}},
	`per totem`:                             Tag{"tag": Tag{"type": "PerStat", "stat": "TotemsSummoned"}},
	`per summoned totem`:                    Tag{"tag": Tag{"type": "PerStat", "stat": "TotemsSummoned"}},
	`for each summoned totem`:               Tag{"tag": Tag{"type": "PerStat", "stat": "TotemsSummoned"}},
	`per maximum number of summoned totems`: Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveTotemLimit"}},
	`for each time they have chained`:       Tag{"tag": Tag{"type": "PerStat", "stat": "Chain"}},
	`for each time it has chained`:          Tag{"tag": Tag{"type": "PerStat", "stat": "Chain"}},
	`for each summoned golem`:               Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveGolemLimit"}},
	`for each golem you have summoned`:      Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveGolemLimit"}},
	`per summoned golem`:                    Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveGolemLimit"}},
	`per summoned sentinel of purity`:       Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveSentinelOfPurityLimit"}},
	`per summoned void spawn`:               Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveVoidSpawnLimit"}},
	`per summoned skeleton`:                 Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveSkeletonLimit"}},
	`per skeleton you own`:                  Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveSkeletonLimit", "actor": "parent"}},
	`per summoned raging spirit`:            Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveRagingSpiritLimit"}},
	`per summoned phantasm`:                 Tag{"tag": Tag{"type": "PerStat", "stat": "ActivePhantasmLimit"}},
	`per animated weapon`:                   Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveAnimatedWeaponLimit", "actor": "parent"}},
	`for each raised zombie`:                Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveZombieLimit"}},
	`per zombie you own`:                    Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveZombieLimit", "actor": "parent"}},
	`per raised zombie`:                     Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveZombieLimit"}},
	`per raised spectre`:                    Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveSpectreLimit"}},
	`per spectre you own`:                   Tag{"tag": Tag{"type": "PerStat", "stat": "ActiveSpectreLimit", "actor": "parent"}},
	`for each remaining chain`:              Tag{"tag": Tag{"type": "PerStat", "stat": "ChainRemaining"}},
	`for each remaining chain, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "PerStat", "stat": "ChainRemaining", "globalLimit": c.n(1), "globalLimitKey": "FollowThrough"}}
	}),
	`for each enemy pierced`:        Tag{"tag": Tag{"type": "PerStat", "stat": "PiercedCount"}},
	`for each time they've pierced`: Tag{"tag": Tag{"type": "PerStat", "stat": "PiercedCount"}},
	// Stat conditions
	`with ([0-9]+) or more strength`:                      fn(func(c caps) any { return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Str", "threshold": c.n(1)}} }),
	`with at least ([0-9]+) strength`:                     fn(func(c caps) any { return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Str", "threshold": c.n(1)}} }),
	`w?h?i[lf]e? you have at least ([0-9]+) strength`:     fn(func(c caps) any { return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Str", "threshold": c.n(1)}} }),
	`w?h?i[lf]e? you have at least ([0-9]+) dexterity`:    fn(func(c caps) any { return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Dex", "threshold": c.n(1)}} }),
	`w?h?i[lf]e? you have at least ([0-9]+) intelligence`: fn(func(c caps) any { return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Int", "threshold": c.n(1)}} }),
	`w?h?i[lf]e? strength is below ([0-9]+)`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Str", "threshold": c.n(1) - 1, "upper": true}}
	}),
	`w?h?i[lf]e? dexterity is below ([0-9]+)`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Dex", "threshold": c.n(1) - 1, "upper": true}}
	}),
	`w?h?i[lf]e? intelligence is below ([0-9]+)`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Int", "threshold": c.n(1) - 1, "upper": true}}
	}),
	`at least ([0-9]+) intelligence`:           fn(func(c caps) any { return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Int", "threshold": c.n(1)}} }),
	`if dexterity is higher than intelligence`: Tag{"tag": Tag{"type": "Condition", "var": "DexHigherThanInt"}},
	`if strength is higher than intelligence`:  Tag{"tag": Tag{"type": "Condition", "var": "StrHigherThanInt"}},
	`w?h?i[lf]e? you have at least ([0-9]+) maximum energy shield`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "EnergyShield", "threshold": c.n(1)}}
	}),
	`against targets they pierce`:   Tag{"tag": Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1}},
	`against pierced targets`:       Tag{"tag": Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1}},
	`to targets they pierce`:        Tag{"tag": Tag{"type": "StatThreshold", "stat": "PierceCount", "threshold": 1}},
	`that fire a single projectile`: Tag{"tag": Tag{"type": "StatThreshold", "stat": "ProjectileCount", "threshold": 1, "upper": true}},
	`w?h?i[lf]e? you have at least ([0-9]+) devotion`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "Devotion", "threshold": c.n(1)}}
	}),
	`while you have at least ([0-9]+) rage`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "Rage", "threshold": c.n(1)}}
	}),
	`while affected by a unique abyss jewel`: Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "UniqueAbyssJewels", "threshold": 1}},
	`while affected by a rare abyss jewel`:   Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "RareAbyssJewels", "threshold": 1}},
	`while affected by a magic abyss jewel`:  Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "MagicAbyssJewels", "threshold": 1}},
	`while affected by a normal abyss jewel`: Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NormalAbyssJewels", "threshold": 1}},
	`while you have at least ([0-9]+) nearby all[yi]e?s?`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NearbyAlly", "threshold": c.n(1)}}
	}),
	// Slot conditions
	`when in main hand`:     Tag{"tag": Tag{"type": "SlotNumber", "num": 1}},
	`whi?l?en? in off hand`: Tag{"tag": Tag{"type": "SlotNumber", "num": 2}},
	`in main hand`:          Tag{"tag": Tag{"type": "InSlot", "num": 1}},
	`in off hand`:           Tag{"tag": Tag{"type": "InSlot", "num": 2}},
	`w?i?t?h? main hand`:    Tag{"tagList": []any{Tag{"type": "Condition", "var": "MainHandAttack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}}},
	`w?i?t?h? off ?hand`:    Tag{"tagList": []any{Tag{"type": "Condition", "var": "OffHandAttack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}}},
	`[fi]?[rn]?[of]?[ml]?[ i]?[hc]?[it]?[te]?[sd]? ? with this weapon`: Tag{"tagList": []any{Tag{"type": "Condition", "var": "{Hand}Attack"}, Tag{"type": "SkillType", "skillType": SkillType.Attack}}},
	`if your o[tp][hp][eo][rs]i?t?e? ring is a shaper item`:            Tag{"tag": Tag{"type": "ItemCondition", "itemSlot": "Ring {OtherSlotNum}", "shaperCond": true}},
	`if your o[tp][hp][eo][rs]i?t?e? ring is an elder item`:            Tag{"tag": Tag{"type": "ItemCondition", "itemSlot": "Ring {OtherSlotNum}", "elderCond": true}},
	`of skills supported by spellslinger`:                              Tag{"tag": Tag{"type": "Condition", "var": "SupportedBySpellslinger"}},
	// Equipment conditions
	`while holding a fishing rod`:               Tag{"tag": Tag{"type": "Condition", "var": "UsingFishing"}},
	`while your off hand is empty`:              Tag{"tag": Tag{"type": "Condition", "var": "OffHandIsEmpty"}},
	`with shields`:                              Tag{"tag": Tag{"type": "Condition", "var": "UsingShield"}},
	`while dual wielding`:                       Tag{"tag": Tag{"type": "Condition", "var": "DualWielding"}},
	`while dual wielding claws`:                 Tag{"tag": Tag{"type": "Condition", "var": "DualWieldingClaws"}},
	`while dual wielding or holding a shield`:   Tag{"tag": Tag{"type": "Condition", "varList": []any{"DualWielding", "UsingShield"}}},
	`while wielding an axe`:                     Tag{"tag": Tag{"type": "Condition", "var": "UsingAxe"}},
	`while wielding an axe or sword`:            Tag{"tag": Tag{"type": "Condition", "varList": []any{"UsingAxe", "UsingSword"}}},
	`while wielding a bow`:                      Tag{"tag": Tag{"type": "Condition", "var": "UsingBow"}},
	`while wielding a claw`:                     Tag{"tag": Tag{"type": "Condition", "var": "UsingClaw"}},
	`while wielding a dagger`:                   Tag{"tag": Tag{"type": "Condition", "var": "UsingDagger"}},
	`while wielding a claw or dagger`:           Tag{"tag": Tag{"type": "Condition", "varList": []any{"UsingClaw", "UsingDagger"}}},
	`while wielding a mace`:                     Tag{"tag": Tag{"type": "Condition", "var": "UsingMace"}},
	`while wielding a mace or sceptre`:          Tag{"tag": Tag{"type": "Condition", "var": "UsingMace"}},
	`while wielding a mace, sceptre or staff`:   Tag{"tag": Tag{"type": "Condition", "varList": []any{"UsingMace", "UsingStaff"}}},
	`while wielding a staff`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsingStaff"}},
	`while wielding a sword`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsingSword"}},
	`while wielding a melee weapon`:             Tag{"tag": Tag{"type": "Condition", "var": "UsingMeleeWeapon"}},
	`while wielding a one handed weapon`:        Tag{"tag": Tag{"type": "Condition", "var": "UsingOneHandedWeapon"}},
	`while wielding a two handed weapon`:        Tag{"tag": Tag{"type": "Condition", "var": "UsingTwoHandedWeapon"}},
	`while wielding a two handed melee weapon`:  Tag{"tagList": []any{Tag{"type": "Condition", "var": "UsingTwoHandedWeapon"}, Tag{"type": "Condition", "var": "UsingMeleeWeapon"}}},
	`while wielding a wand`:                     Tag{"tag": Tag{"type": "Condition", "var": "UsingWand"}},
	`while wielding two different weapon types`: Tag{"tag": Tag{"type": "Condition", "var": "WieldingDifferentWeaponTypes"}},
	`while unarmed`:                             Tag{"tag": Tag{"type": "Condition", "var": "Unarmed"}},
	`while you are unencumbered`:                Tag{"tag": Tag{"type": "Condition", "var": "Unencumbered"}},
	`equipped bow`:                              Tag{"tag": Tag{"type": "Condition", "var": "UsingBow"}},
	`if equipped ([a-zA-Z \t\n\v\f\r]+) has an ([a-zA-Z \t\n\v\f\r]+) modifier`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "ItemCondition", "searchCond": c.s(2), "itemSlot": c.s(1)}}
	}),
	`if your equipped staff has a red and blue socket`: Tag{"tagList": []any{Tag{"type": "MultiplierThreshold", "var": "RedSocketInWeapon 1", "threshold": 1}, Tag{"type": "MultiplierThreshold", "var": "BlueSocketInWeapon 1", "threshold": 1}, Tag{"type": "Condition", "var": "UsingStaff"}}},
	`if there are no ([a-zA-Z \t\n\v\f\r]+) modifiers on equipped ([a-zA-Z \t\n\v\f\r]+)`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "ItemCondition", "searchCond": c.s(1), "itemSlot": c.s(2), "neg": true}}
	}),
	`if there are no ([a-zA-Z]+) modifiers on other equipped items`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "ItemCondition", "searchCond": c.s(1), "itemSlot": "{SlotName}", "allSlots": true, "excludeSelf": true, "neg": true}}
	}),
	`if corrupted`:                        Tag{"tag": Tag{"type": "ItemCondition", "itemSlot": "{SlotName}", "corruptedCond": true}},
	`with a normal item equipped`:         Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NormalItem", "threshold": 1}},
	`with a magic item equipped`:          Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "MagicItem", "threshold": 1}},
	`with a rare item equipped`:           Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "RareItem", "threshold": 1}},
	`with a unique item equipped`:         Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "UniqueItem", "threshold": 1}},
	`if you wear no corrupted items`:      Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "CorruptedItem", "threshold": 0, "upper": true}},
	`if no worn items are corrupted`:      Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "CorruptedItem", "threshold": 0, "upper": true}},
	`if no equipped items are corrupted`:  Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "CorruptedItem", "threshold": 0, "upper": true}},
	`if all worn items are corrupted`:     Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NonCorruptedItem", "threshold": 0, "upper": true}},
	`if all equipped items are corrupted`: Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NonCorruptedItem", "threshold": 0, "upper": true}},
	`if equipped shield has at least ([0-9]+)% chance to block`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "ShieldBlockChance", "threshold": c.n(1)}}
	}),
	`if you have ([0-9]+) primordial items socketed or equipped`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "PrimordialItem", "threshold": c.n(1)}}
	}),
	// Player status conditions
	`if used while on low life`:              Tag{"tag": Tag{"type": "Condition", "var": "LowLife"}},
	`wh[ie][ln]e? on low life`:               Tag{"tag": Tag{"type": "Condition", "var": "LowLife"}},
	`on reaching low life`:                   Tag{"tag": Tag{"type": "Condition", "var": "LowLife"}},
	`wh[ie][ln]e? not on low life`:           Tag{"tag": Tag{"type": "Condition", "var": "LowLife", "neg": true}},
	`wh[ie][ln]e? on low mana`:               Tag{"tag": Tag{"type": "Condition", "var": "LowMana"}},
	`wh[ie][ln]e? not on low mana`:           Tag{"tag": Tag{"type": "Condition", "var": "LowMana", "neg": true}},
	`wh[ie][ln]e? on full life`:              Tag{"tag": Tag{"type": "Condition", "var": "FullLife"}},
	`wh[ie][ln]e? not on full life`:          Tag{"tag": Tag{"type": "Condition", "var": "FullLife", "neg": true}},
	`wh[ie][ln]e? no life is reserved`:       Tag{"tag": Tag{"type": "StatThreshold", "stat": "LifeReserved", "threshold": 0, "upper": true}},
	`wh[ie][ln]e? no mana is reserved`:       Tag{"tag": Tag{"type": "StatThreshold", "stat": "ManaReserved", "threshold": 0, "upper": true}},
	`wh[ie][ln]e? on full energy shield`:     Tag{"tag": Tag{"type": "Condition", "var": "FullEnergyShield"}},
	`wh[ie][ln]e? not on full energy shield`: Tag{"tag": Tag{"type": "Condition", "var": "FullEnergyShield", "neg": true}},
	`wh[ie][ln]e? you have energy shield`:    Tag{"tag": Tag{"type": "Condition", "var": "HaveEnergyShield"}},
	`wh[ie][ln]e? you have no energy shield`: Tag{"tag": Tag{"type": "Condition", "var": "HaveEnergyShield", "neg": true}},
	`if you have energy shield`:              Tag{"tag": Tag{"type": "Condition", "var": "HaveEnergyShield"}},
	`while stationary`:                       Tag{"tag": Tag{"type": "Condition", "var": "Stationary"}},
	`while you are stationary`:               Tag{"tag": Tag{"type": "ActorCondition", "actor": "player", "var": "Stationary"}},
	`while moving`:                           Tag{"tag": Tag{"type": "Condition", "var": "Moving"}},
	`while channelling`:                      Tag{"tag": Tag{"type": "Condition", "var": "Channelling"}},
	`while channelling snipe`:                Tag{"tag": Tag{"type": "Condition", "var": "Channelling"}},
	`after channelling for ([0-9]+) seconds?`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "ChannellingTime", "threshold": c.n(1)}}
	}),
	`if you've been channelling for at least ([0-9]+) seconds?`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "ChannellingTime", "threshold": c.n(1)}}
	}),
	`if you've inflicted exposure recently`: Tag{"tag": Tag{"type": "Condition", "var": "AppliedExposureRecently"}},
	`while you have no power charges`:       Tag{"tag": Tag{"type": "StatThreshold", "stat": "PowerCharges", "threshold": 0, "upper": true}},
	`while you have no frenzy charges`:      Tag{"tag": Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "threshold": 0, "upper": true}},
	`while you have no endurance charges`:   Tag{"tag": Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "threshold": 0, "upper": true}},
	`while you have a power charge`:         Tag{"tag": Tag{"type": "StatThreshold", "stat": "PowerCharges", "threshold": 1}},
	`while you have a frenzy charge`:        Tag{"tag": Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "threshold": 1}},
	`while you have an endurance charge`:    Tag{"tag": Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "threshold": 1}},
	`while at maximum power charges`:        Tag{"tag": Tag{"type": "StatThreshold", "stat": "PowerCharges", "thresholdStat": "PowerChargesMax"}},
	`while at maximum frenzy charges`:       Tag{"tag": Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"}},
	`while on full frenzy charges`:          Tag{"tag": Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMax"}},
	`while at maximum endurance charges`:    Tag{"tag": Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMax"}},
	`while at minimum endurance charges`:    Tag{"tag": Tag{"type": "StatThreshold", "stat": "EnduranceCharges", "thresholdStat": "EnduranceChargesMin", "upper": true}},
	`while at minimum power charges`:        Tag{"tag": Tag{"type": "StatThreshold", "stat": "PowerCharges", "thresholdStat": "PowerChargesMin", "upper": true}},
	`while at minimum frenzy charges`:       Tag{"tag": Tag{"type": "StatThreshold", "stat": "FrenzyCharges", "thresholdStat": "FrenzyChargesMin", "upper": true}},
	`while at maximum rage`:                 Tag{"tag": Tag{"type": "Condition", "var": "HaveMaximumRage"}},
	`while at maximum fortification`:        Tag{"tag": Tag{"type": "Condition", "var": "HaveMaximumFortification"}},
	`while you have at least ([0-9]+) crab barriers`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "CrabBarriers", "threshold": c.n(1)}}
	}),
	`while you have at least ([0-9]+) fortification`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "StatThreshold", "stat": "FortificationStacks", "threshold": c.n(1)}}
	}),
	`while you have at least ([0-9]+) total endurance, frenzy and power charges`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "TotalCharges", "threshold": c.n(1)}}
	}),
	`while you have a totem`:                             Tag{"tag": Tag{"type": "Condition", "var": "HaveTotem"}},
	`while you have at least one nearby ally`:            Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NearbyAlly", "threshold": 1}},
	`while you have a linked target`:                     Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "LinkedTargets", "threshold": 1}},
	`while you have fortify`:                             Tag{"tag": Tag{"type": "Condition", "var": "Fortified"}},
	`while you have phasing`:                             Tag{"tag": Tag{"type": "Condition", "var": "Phasing"}},
	`while you have unbroken ward`:                       Tag{"tag": Tag{"type": "Condition", "var": "UnbrokenWard"}},
	`while your ward is broken`:                          Tag{"tag": Tag{"type": "Condition", "var": "UnbrokenWard", "neg": true}},
	`if you[' ]h?a?ve suppressed spell damage recently`:  Tag{"tag": Tag{"type": "Condition", "var": "SuppressedRecently"}},
	`while you have elusive`:                             Tag{"tag": Tag{"type": "Condition", "var": "Elusive"}},
	`while physical aegis is depleted`:                   Tag{"tag": Tag{"type": "Condition", "var": "PhysicalAegisDepleted"}},
	`during onslaught`:                                   Tag{"tag": Tag{"type": "Condition", "var": "Onslaught"}},
	`while you have onslaught`:                           Tag{"tag": Tag{"type": "Condition", "var": "Onslaught"}},
	`while phasing`:                                      Tag{"tag": Tag{"type": "Condition", "var": "Phasing"}},
	`while you have tailwind`:                            Tag{"tag": Tag{"type": "Condition", "var": "Tailwind"}},
	`while elusive`:                                      Tag{"tag": Tag{"type": "Condition", "var": "Elusive"}},
	`gain elusive`:                                       Tag{"tag": Tag{"type": "Condition", "varList": []any{"CanBeElusive", "Elusive"}}},
	`while you have arcane surge`:                        Tag{"tag": Tag{"type": "Condition", "var": "AffectedByArcaneSurge"}},
	`while you have cat's stealth`:                       Tag{"tag": Tag{"type": "Condition", "var": "AffectedByCat'sStealth"}},
	`while you have cat's agility`:                       Tag{"tag": Tag{"type": "Condition", "var": "AffectedByCat'sAgility"}},
	`while you have avian's might`:                       Tag{"tag": Tag{"type": "Condition", "var": "AffectedByAvian'sMight"}},
	`while you have avian's flight`:                      Tag{"tag": Tag{"type": "Condition", "var": "AffectedByAvian'sFlight"}},
	`while affected by aspect of the cat`:                Tag{"tag": Tag{"type": "Condition", "varList": []any{"AffectedByCat'sStealth", "AffectedByCat'sAgility"}}},
	`while affected by a non-vaal guard skill`:           Tag{"tag": Tag{"type": "Condition", "var": "AffectedByNonVaalGuardSkill"}},
	`if a non-vaal guard buff was lost recently`:         Tag{"tag": Tag{"type": "Condition", "var": "LostNonVaalBuffRecently"}},
	`while affected by a guard skill buff`:               Tag{"tag": Tag{"type": "Condition", "var": "AffectedByGuardSkill"}},
	`while affected by a herald`:                         Tag{"tag": Tag{"type": "Condition", "var": "AffectedByHerald"}},
	`while fortified`:                                    Tag{"tag": Tag{"type": "Condition", "var": "Fortified"}},
	`while in blood stance`:                              Tag{"tag": Tag{"type": "Condition", "var": "BloodStance"}},
	`while in sand stance`:                               Tag{"tag": Tag{"type": "Condition", "var": "SandStance"}},
	`while you have a bestial minion`:                    Tag{"tag": Tag{"type": "Condition", "var": "HaveBestialMinion"}},
	`while you have infusion`:                            Tag{"tag": Tag{"type": "Condition", "var": "InfusionActive"}},
	`while focus?sed`:                                    Tag{"tag": Tag{"type": "Condition", "var": "Focused"}},
	`while leeching`:                                     Tag{"tag": Tag{"type": "Condition", "var": "Leeching"}},
	`while leeching life`:                                Tag{"tag": Tag{"type": "Condition", "var": "LeechingLife"}},
	`while leeching energy shield`:                       Tag{"tag": Tag{"type": "Condition", "var": "LeechingEnergyShield"}},
	`while leeching mana`:                                Tag{"tag": Tag{"type": "Condition", "var": "LeechingMana"}},
	`while using a flask`:                                Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask"}},
	`during effect`:                                      Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask"}},
	`during flask effect`:                                Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask"}},
	`during any flask effect`:                            Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask"}},
	`while under no flask effects`:                       Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask", "neg": true}},
	`while affected by no flasks`:                        Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask", "neg": true}},
	`during effect of any mana flask`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsingManaFlask"}},
	`during effect of any life flask`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsingLifeFlask"}},
	`if you've used a life flask in the past 10 seconds`: Tag{"tag": Tag{"type": "Condition", "var": "UsingLifeFlask"}},
	`if you've used a mana flask in the past 10 seconds`: Tag{"tag": Tag{"type": "Condition", "var": "UsingManaFlask"}},
	`during effect of any life or mana flask`:            Tag{"tag": Tag{"type": "Condition", "varList": []any{"UsingManaFlask", "UsingLifeFlask"}}},
	`while you have an active tincture`:                  Tag{"tag": Tag{"type": "Condition", "var": "UsingTincture"}},
	`while you have a tincture active`:                   Tag{"tag": Tag{"type": "Condition", "var": "UsingTincture"}},
	`with at least one ([0-9a-zA-Z]+) grafted to you`:    fn(func(c caps) any { return Tag{"tag": Tag{"type": "Condition", "var": "Using" + firstToUpper(c.s(1))}} }),
	`while on consecrated ground`:                        Tag{"tag": Tag{"type": "Condition", "var": "OnConsecratedGround"}},
	`while on caustic ground`:                            Tag{"tag": Tag{"type": "Condition", "var": "OnCausticGround"}},
	`when you create consecrated ground`:                 d(),
	`on burning ground`:                                  Tag{"tag": Tag{"type": "Condition", "var": "OnBurningGround"}},
	`while on burning ground`:                            Tag{"tag": Tag{"type": "Condition", "var": "OnBurningGround"}},
	`on chilled ground`:                                  Tag{"tag": Tag{"type": "Condition", "var": "OnChilledGround"}},
	`on shocked ground`:                                  Tag{"tag": Tag{"type": "Condition", "var": "OnShockedGround"}},
	`while in a caustic cloud`:                           Tag{"tag": Tag{"type": "Condition", "var": "OnCausticCloud"}},
	`while blinded`:                                      Tag{"tagList": []any{Tag{"type": "Condition", "var": "Blinded"}, Tag{"type": "Condition", "var": "CannotBeBlinded", "neg": true}}},
	`while burning`:                                      Tag{"tag": Tag{"type": "Condition", "var": "Burning"}},
	`while ignited`:                                      Tag{"tag": Tag{"type": "Condition", "var": "Ignited"}},
	`while you are ignited`:                              Tag{"tag": Tag{"type": "Condition", "var": "Ignited"}},
	`while chilled`:                                      Tag{"tag": Tag{"type": "Condition", "var": "Chilled"}},
	`while you are chilled`:                              Tag{"tag": Tag{"type": "Condition", "var": "Chilled"}},
	`while frozen`:                                       Tag{"tag": Tag{"type": "Condition", "var": "Frozen"}},
	`while shocked`:                                      Tag{"tag": Tag{"type": "Condition", "var": "Shocked"}},
	`while you are shocked`:                              Tag{"tag": Tag{"type": "Condition", "var": "Shocked"}},
	`while you are bleeding`:                             Tag{"tag": Tag{"type": "Condition", "var": "Bleeding"}},
	`while not ignited, frozen or shocked`:               Tag{"tag": Tag{"type": "Condition", "varList": []any{"Ignited", "Frozen", "Shocked"}, "neg": true}},
	`while bleeding`:                                     Tag{"tag": Tag{"type": "Condition", "var": "Bleeding"}},
	`while poisoned`:                                     Tag{"tag": Tag{"type": "Condition", "var": "Poisoned"}},
	`wh[ei][nl][ e] ?you are poisoned`:                   Tag{"tag": Tag{"type": "Condition", "var": "Poisoned"}},
	`while cursed`:                                       Tag{"tag": Tag{"type": "Condition", "var": "Cursed"}},
	`while not cursed`:                                   Tag{"tag": Tag{"type": "Condition", "var": "Cursed", "neg": true}},
	`while there is only one nearby enemy`:               Tag{"tagList": []any{Tag{"type": "Multiplier", "var": "NearbyEnemies", "limit": 1}, Tag{"type": "Condition", "var": "OnlyOneNearbyEnemy"}}},
	`while at least ([0-9]+) enemies are nearby`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "NearbyEnemies", "threshold": c.n(1)}}
	}),
	`while t?h?e?r?e? ?i?s? ?a rare or unique enemy i?s? ?nearby`:                      Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"NearbyRareOrUniqueEnemy", "RareOrUnique"}}},
	`if you[' ]h?a?ve hit recently`:                                                    Tag{"tag": Tag{"type": "Condition", "var": "HitRecently"}},
	`if you[' ]h?a?ve hit an enemy recently`:                                           Tag{"tag": Tag{"type": "Condition", "var": "HitRecently"}},
	`if you[' ]h?a?ve hit with your main hand weapon recently`:                         Tag{"tag": Tag{"type": "Condition", "var": "HitRecentlyWithWeapon"}},
	`if you[' ]h?a?ve hit with your off hand weapon recently`:                          Tag{"tagList": []any{Tag{"type": "Condition", "var": "HitRecentlyWithWeapon"}, Tag{"type": "Condition", "var": "DualWielding"}}},
	`if you[' ]h?a?ve hit a cursed enemy recently`:                                     Tag{"tagList": []any{Tag{"type": "Condition", "var": "HitRecently"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}}},
	`when you or your totems hit an enemy with a spell`:                                Tag{"tag": Tag{"type": "Condition", "varList": []any{"HitSpellRecently", "TotemsHitSpellRecently"}}},
	`on hit with spells`:                                                               Tag{"tag": Tag{"type": "Condition", "var": "HitSpellRecently"}},
	`if you[' ]h?a?ve crit recently`:                                                   Tag{"tag": Tag{"type": "Condition", "var": "CritRecently"}},
	`if you[' ]h?a?ve dealt a critical strike recently`:                                Tag{"tag": Tag{"type": "Condition", "var": "CritRecently"}},
	`when you deal a critical strike`:                                                  Tag{"tag": Tag{"type": "Condition", "var": "CritRecently"}},
	`if you[' ]h?a?ve dealt a critical strike with this weapon recently`:               Tag{"tag": Tag{"type": "Condition", "var": "CritRecently"}},
	`if you[' ]h?a?ve crit in the past 8 seconds`:                                      Tag{"tag": Tag{"type": "Condition", "var": "CritInPast8Sec"}},
	`if you[' ]h?a?ve dealt a crit in the past 8 seconds`:                              Tag{"tag": Tag{"type": "Condition", "var": "CritInPast8Sec"}},
	`if you[' ]h?a?ve dealt a critical strike in the past 8 seconds`:                   Tag{"tag": Tag{"type": "Condition", "var": "CritInPast8Sec"}},
	`if you haven't crit recently`:                                                     Tag{"tag": Tag{"type": "Condition", "var": "CritRecently", "neg": true}},
	`if you haven't dealt a critical strike recently`:                                  Tag{"tag": Tag{"type": "Condition", "var": "CritRecently", "neg": true}},
	`if you[' ]h?a?ve dealt a non-critical strike recently`:                            Tag{"tag": Tag{"type": "Condition", "var": "NonCritRecently"}},
	`if your skills have dealt a critical strike recently`:                             Tag{"tag": Tag{"type": "Condition", "var": "SkillCritRecently"}},
	`if you dealt a critical strike with a herald skill recently`:                      Tag{"tag": Tag{"type": "Condition", "var": "CritWithHeraldSkillRecently"}},
	`if you[' ]h?a?ve dealt a critical strike with a two handed melee weapon recently`: Tag{"flags": ModFlag.Weapon2H | ModFlag.WeaponMelee, "tag": Tag{"type": "Condition", "var": "CritRecently"}},
	`if you[' ]h?a?ve killed recently`:                                                 Tag{"tag": Tag{"type": "Condition", "var": "KilledRecently"}},
	`on killing taunted enemies`:                                                       Tag{"tag": Tag{"type": "Condition", "var": "KilledTauntedEnemyRecently"}},
	`on kill`:                                                                          Tag{"tag": Tag{"type": "Condition", "var": "KilledRecently"}},
	`on melee kill`:                                                                    Tag{"flags": ModFlag.WeaponMelee, "tag": Tag{"type": "Condition", "var": "KilledRecently"}},
	`when you kill an enemy`:                                                           Tag{"tag": Tag{"type": "Condition", "var": "KilledRecently"}},
	`if you[' ]h?a?ve killed an enemy recently`:                                        Tag{"tag": Tag{"type": "Condition", "var": "KilledRecently"}},
	`if you[' ]h?a?ve killed at least ([0-9]) enemies recently`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "EnemyKilledRecently", "threshold": c.n(1)}}
	}),
	`if you haven't killed recently`:                                              Tag{"tag": Tag{"type": "Condition", "var": "KilledRecently", "neg": true}},
	`if you or your totems have killed recently`:                                  Tag{"tag": Tag{"type": "Condition", "varList": []any{"KilledRecently", "TotemsKilledRecently"}}},
	`if you[' ]h?a?ve thrown a trap or mine recently`:                             Tag{"tag": Tag{"type": "Condition", "var": "TrapOrMineThrownRecently"}},
	`on throwing a trap`:                                                          Tag{"tag": Tag{"type": "Condition", "var": "TrapOrMineThrownRecently"}},
	`if you[' ]h?a?ve killed a maimed enemy recently`:                             Tag{"tagList": []any{Tag{"type": "Condition", "var": "KilledRecently"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Maimed"}}},
	`if you[' ]h?a?ve killed a cursed enemy recently`:                             Tag{"tagList": []any{Tag{"type": "Condition", "var": "KilledRecently"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}}},
	`if you[' ]h?a?ve killed a bleeding enemy recently`:                           Tag{"tagList": []any{Tag{"type": "Condition", "var": "KilledRecently"}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}}},
	`if you[' ]h?a?ve killed an enemy affected by your damage over time recently`: Tag{"tag": Tag{"type": "Condition", "var": "KilledAffectedByDotRecently"}},
	`if you[' ]h?a?ve frozen an enemy recently`:                                   Tag{"tag": Tag{"type": "Condition", "var": "FrozenEnemyRecently"}},
	`if you[' ]h?a?ve chilled an enemy recently`:                                  Tag{"tag": Tag{"type": "Condition", "var": "ChilledEnemyRecently"}},
	`if you[' ]h?a?ve ignited an enemy recently`:                                  Tag{"tag": Tag{"type": "Condition", "var": "IgnitedEnemyRecently"}},
	`if you[' ]h?a?ve shocked an enemy recently`:                                  Tag{"tag": Tag{"type": "Condition", "var": "ShockedEnemyRecently"}},
	`if you[' ]h?a?ve stunned an enemy recently`:                                  Tag{"tag": Tag{"type": "Condition", "var": "StunnedEnemyRecently"}},
	`if you[' ]h?a?ve stunned an enemy with a two handed melee weapon recently`:   Tag{"flags": ModFlag.Weapon2H | ModFlag.WeaponMelee, "tag": Tag{"type": "Condition", "var": "StunnedEnemyRecently"}},
	`if you[' ]h?a?ve been hit recently`:                                          Tag{"tag": Tag{"type": "Condition", "var": "BeenHitRecently"}},
	`if you[' ]h?a?ve been hit by an attack recently`:                             Tag{"tag": Tag{"type": "Condition", "var": "BeenHitByAttackRecently"}},
	`if you were hit recently`:                                                    Tag{"tag": Tag{"type": "Condition", "var": "BeenHitRecently"}},
	`if you were damaged by a hit recently`:                                       Tag{"tag": Tag{"type": "Condition", "var": "BeenHitRecently"}},
	`if you[' ]h?a?ve taken a critical strike recently`:                           Tag{"tag": Tag{"type": "Condition", "var": "BeenCritRecently"}},
	`if you[' ]h?a?ve taken a savage hit recently`:                                Tag{"tag": Tag{"type": "Condition", "var": "BeenSavageHitRecently"}},
	`if you have ?n[o']t been hit recently`:                                       Tag{"tag": Tag{"type": "Condition", "var": "BeenHitRecently", "neg": true}},
	`if you have ?n[o']t been hit by an attack recently`:                          Tag{"tag": Tag{"type": "Condition", "var": "BeenHitByAttackRecently", "neg": true}},
	`if you[' ]h?a?ve taken no damage from hits recently`:                         Tag{"tag": Tag{"type": "Condition", "var": "BeenHitRecently", "neg": true}},
	`if you[' ]h?a?ve taken fire damage from a hit recently`:                      Tag{"tag": Tag{"type": "Condition", "var": "HitByFireDamageRecently"}},
	`if you[' ]h?a?ve taken fire damage from an enemy hit recently`:               Tag{"tag": Tag{"type": "Condition", "var": "TakenFireDamageFromEnemyHitRecently"}},
	`if you[' ]h?a?ve taken spell damage recently`:                                Tag{"tag": Tag{"type": "Condition", "var": "HitBySpellDamageRecently"}},
	`if you haven't taken damage recently`:                                        Tag{"tag": Tag{"type": "Condition", "var": "BeenHitRecently", "neg": true}},
	`if you[' ]h?a?ve blocked recently`:                                           Tag{"tag": Tag{"type": "Condition", "var": "BlockedRecently"}},
	`if you haven't blocked recently`:                                             Tag{"tag": Tag{"type": "Condition", "var": "BlockedRecently", "neg": true}},
	`if you[' ]h?a?ve blocked an attack recently`:                                 Tag{"tag": Tag{"type": "Condition", "var": "BlockedAttackRecently"}},
	`if you[' ]h?a?ve blocked attack damage recently`:                             Tag{"tag": Tag{"type": "Condition", "var": "BlockedAttackRecently"}},
	`if you[' ]h?a?ve blocked a spell recently`:                                   Tag{"tag": Tag{"type": "Condition", "var": "BlockedSpellRecently"}},
	`if you[' ]h?a?ve blocked spell damage recently`:                              Tag{"tag": Tag{"type": "Condition", "var": "BlockedSpellRecently"}},
	`if you[' ]h?a?ve blocked damage from a unique enemy in the past 10 seconds`:  Tag{"tag": Tag{"type": "Condition", "var": "BlockedHitFromUniqueEnemyInPast10Sec"}},
	`if you[' ]h?a?ve attacked recently`:                                          Tag{"tag": Tag{"type": "Condition", "var": "AttackedRecently"}},
	`if you[' ]h?a?ve cast a spell recently`:                                      Tag{"tag": Tag{"type": "Condition", "var": "CastSpellRecently"}},
	`if you[' ]h?a?ve been stunned while casting recently`:                        Tag{"tag": Tag{"type": "Condition", "var": "StunnedWhileCastingRecently"}},
	`if you[' ]h?a?ve consumed a corpse recently`:                                 Tag{"tag": Tag{"type": "Condition", "var": "ConsumedCorpseRecently"}},
	`if you[' ]h?a?ve cursed an enemy recently`:                                   Tag{"tag": Tag{"type": "Condition", "var": "CursedEnemyRecently"}},
	`if you[' ]h?a?ve cast a mark spell recently`:                                 Tag{"tag": Tag{"type": "Condition", "var": "CastMarkRecently"}},
	`if you have ?n[o']t consumed a corpse recently`:                              Tag{"tag": Tag{"type": "Condition", "var": "ConsumedCorpseRecently", "neg": true}},
	`for each corpse consumed recently`:                                           Tag{"tag": Tag{"type": "Multiplier", "var": "CorpseConsumedRecently"}},
	`if you[' ]h?a?ve taunted an enemy recently`:                                  Tag{"tag": Tag{"type": "Condition", "var": "TauntedEnemyRecently"}},
	`if you[' ]h?a?ve used a skill recently`:                                      Tag{"tag": Tag{"type": "Condition", "var": "UsedSkillRecently"}},
	`if you[' ]h?a?ve used a travel skill recently`:                               Tag{"tag": Tag{"type": "Condition", "var": "UsedTravelSkillRecently"}},
	`for each skill you've used recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "SkillUsedRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each different non-instant spell you[' ]h?a?ve cast recently`: Tag{"tag": Tag{"type": "Multiplier", "var": "NonInstantSpellCastRecently"}},
	`if you[' ]h?a?ve used a warcry recently`:                          Tag{"tag": Tag{"type": "Condition", "var": "UsedWarcryRecently"}},
	// "when you warcry" appears twice in the reference with the same value.
	`when you warcry`:                                 Tag{"tag": Tag{"type": "Condition", "var": "UsedWarcryRecently"}},
	`if you[' ]h?a?ve warcried recently`:              Tag{"tag": Tag{"type": "Condition", "var": "UsedWarcryRecently"}},
	`if you[' ]h?a?ve not warcried recently`:          Tag{"tag": Tag{"type": "Condition", "var": "UsedWarcryRecently", "neg": true}},
	`for each time you[' ]h?a?ve warcried recently`:   Tag{"tag": Tag{"type": "Multiplier", "var": "WarcryUsedRecently"}},
	`for each warcry exerting them`:                   Tag{"tag": Tag{"type": "Multiplier", "var": "ExertingWarcryCount"}},
	`if you[' ]h?a?ve warcried in the past 8 seconds`: Tag{"tag": Tag{"type": "Condition", "var": "UsedWarcryInPast8Seconds"}},
	`for each second you've been affected by a warcry buff, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "AffectedByWarcryBuffDuration", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each of your mines detonated recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "MineDetonatedRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`[fp][oe]r ?e?a?c?h? mine detonated recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "MineDetonatedRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`[fp][oe]r ?e?a?c?h? mine detonated recently, up to ([0-9]+)% per second`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "MineDetonatedRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`[fp][oe]r ?e?a?c?h? mine detonated recently`: Tag{"tag": Tag{"type": "Multiplier", "var": "MineDetonatedRecently"}},
	`for each of your traps triggered recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "TrapTriggeredRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each trap triggered recently, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "TrapTriggeredRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each trap triggered recently, up to ([0-9]+)% per second`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "TrapTriggeredRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`if you[' ]h?a?ve used a fire skill recently`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsedFireSkillRecently"}},
	`if you[' ]h?a?ve used a cold skill recently`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsedColdSkillRecently"}},
	`if you[' ]h?a?ve used a fire skill in the past 10 seconds`:      Tag{"tag": Tag{"type": "Condition", "var": "UsedFireSkillInPast10Sec"}},
	`if you[' ]h?a?ve used a cold skill in the past 10 seconds`:      Tag{"tag": Tag{"type": "Condition", "var": "UsedColdSkillInPast10Sec"}},
	`if you[' ]h?a?ve used a lightning skill in the past 10 seconds`: Tag{"tag": Tag{"type": "Condition", "var": "UsedLightningSkillInPast10Sec"}},
	`if you[' ]h?a?ve summoned a totem recently`:                     Tag{"tag": Tag{"type": "Condition", "var": "SummonedTotemRecently"}},
	`when you summon a totem`:                                        Tag{"tag": Tag{"type": "Condition", "var": "SummonedTotemRecently"}},
	`if you summoned a golem in the past 8 seconds`:                  Tag{"tag": Tag{"type": "Condition", "var": "SummonedGolemInPast8Sec"}},
	`if you haven't summoned a totem in the past 2 seconds`:          Tag{"tag": Tag{"type": "Condition", "var": "NoSummonedTotemsInPastTwoSeconds"}},
	`if you[' ]h?a?ve used a minion skill recently`:                  Tag{"tag": Tag{"type": "Condition", "var": "UsedMinionSkillRecently"}},
	`if you[' ]h?a?ve used a movement skill recently`:                Tag{"tag": Tag{"type": "Condition", "var": "UsedMovementSkillRecently"}},
	`when you use a movement skill`:                                  Tag{"tag": Tag{"type": "Condition", "var": "UsedMovementSkillRecently"}},
	`if you haven't cast dash recently`:                              Tag{"tag": Tag{"type": "Condition", "var": "CastDashRecently", "neg": true}},
	`if you[' ]h?a?ve cast dash recently`:                            Tag{"tag": Tag{"type": "Condition", "var": "CastDashRecently"}},
	`if you[' ]h?a?ve used a vaal skill recently`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsedVaalSkillRecently"}},
	`if you[' ]h?a?ve used a socketed vaal skill recently`:           Tag{"tag": Tag{"type": "Condition", "var": "UsedVaalSkillRecently"}},
	`when you use a vaal skill`:                                      Tag{"tag": Tag{"type": "Condition", "var": "UsedVaalSkillRecently"}},
	`if you haven't used a brand skill recently`:                     Tag{"tag": Tag{"type": "Condition", "var": "UsedBrandRecently", "neg": true}},
	`if you[' ]h?a?ve used a brand skill recently`:                   Tag{"tag": Tag{"type": "Condition", "var": "UsedBrandRecently"}},
	`if you[' ]h?a?ve used a retaliation skill recently`:             Tag{"tag": Tag{"type": "Condition", "var": "UsedRetaliationRecently"}},
	`if you[' ]h?a?ve spent ([0-9]+) total mana recently`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "ManaSpentRecently", "threshold": c.n(1)}}
	}),
	`if you[' ]h?a?ve spent life recently`: Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "LifeSpentRecently", "threshold": 1}},
	`for [0-9]+ seconds after spending a total of ([0-9]+) mana`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "ManaSpentRecently", "threshold": c.n(1)}}
	}),
	`if you've impaled an enemy recently`:                         Tag{"tag": Tag{"type": "Condition", "var": "ImpaledRecently"}},
	`if you've changed stance recently`:                           Tag{"tag": Tag{"type": "Condition", "var": "ChangedStanceRecently"}},
	`if you've gained a power charge recently`:                    Tag{"tag": Tag{"type": "Condition", "var": "GainedPowerChargeRecently"}},
	`if you haven't gained a power charge recently`:               Tag{"tag": Tag{"type": "Condition", "var": "GainedPowerChargeRecently", "neg": true}},
	`if you haven't gained a frenzy charge recently`:              Tag{"tag": Tag{"type": "Condition", "var": "GainedFrenzyChargeRecently", "neg": true}},
	`if you've stopped taking damage over time recently`:          Tag{"tag": Tag{"type": "Condition", "var": "StoppedTakingDamageOverTimeRecently"}},
	`if you've used an amethyst flask recently`:                   Tag{"tag": Tag{"type": "Condition", "var": "UsedAmethystFlaskRecently"}},
	`if you've used a ruby flask recently`:                        Tag{"tag": Tag{"type": "Condition", "var": "UsedRubyFlaskRecently"}},
	`if you've used a sapphire flask recently`:                    Tag{"tag": Tag{"type": "Condition", "var": "UsedSapphireFlaskRecently"}},
	`if you've used a topaz flask recently`:                       Tag{"tag": Tag{"type": "Condition", "var": "UsedTopazFlaskRecently"}},
	`during soul gain prevention`:                                 Tag{"tag": Tag{"type": "Condition", "var": "SoulGainPrevention"}},
	`if you detonated mines recently`:                             Tag{"tag": Tag{"type": "Condition", "var": "DetonatedMinesRecently"}},
	`if you detonated a mine recently`:                            Tag{"tag": Tag{"type": "Condition", "var": "DetonatedMinesRecently"}},
	`if you[' ]h?a?ve detonated a mine recently`:                  Tag{"tag": Tag{"type": "Condition", "var": "DetonatedMinesRecently"}},
	`when your mine is detonated targeting an enemy`:              Tag{"tag": Tag{"type": "Condition", "var": "DetonatedMinesRecently"}},
	`when your trap is triggered by an enemy`:                     Tag{"tag": Tag{"type": "Condition", "var": "TriggeredTrapsRecently"}},
	`if energy shield recharge has started recently`:              Tag{"tag": Tag{"type": "Condition", "var": "EnergyShieldRechargeRecently"}},
	`if energy shield recharge has started in the past 2 seconds`: Tag{"tag": Tag{"type": "Condition", "var": "EnergyShieldRechargePastTwoSec"}},
	`when cast on frostbolt`:                                      Tag{"tag": Tag{"type": "Condition", "var": "CastOnFrostbolt"}},
	`branded enemy's`:                                             Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "BrandsAttachedToEnemy", "threshold": 1}},
	`to enemies they're attached to`:                              Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "BrandsAttachedToEnemy", "threshold": 1}},
	`for each hit you've taken recently up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "BeenHitRecently", "limit": c.n(1), "limitTotal": true}}
	}),
	`per enemy hit taken recently`: Tag{"tag": Tag{"type": "Multiplier", "var": "BeenHitRecently"}},
	`for each nearby enemy, up to ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "NearbyEnemies", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each nearby enemy, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "var": "NearbyEnemies", "limit": c.n(1), "limitTotal": true}}
	}),
	`while you have iron reflexes`:             Tag{"tag": Tag{"type": "Condition", "var": "HaveIronReflexes"}},
	`while you do not have iron reflexes`:      Tag{"tag": Tag{"type": "Condition", "var": "HaveIronReflexes", "neg": true}},
	`while you have elemental overload`:        Tag{"tag": Tag{"type": "Condition", "var": "HaveElementalOverload"}},
	`while you do not have elemental overload`: Tag{"tag": Tag{"type": "Condition", "var": "HaveElementalOverload", "neg": true}},
	`while you have resolute technique`:        Tag{"tag": Tag{"type": "Condition", "var": "HaveResoluteTechnique"}},
	`while you do not have resolute technique`: Tag{"tag": Tag{"type": "Condition", "var": "HaveResoluteTechnique", "neg": true}},
	`while you have avatar of fire`:            Tag{"tag": Tag{"type": "Condition", "var": "HaveAvatarOfFire"}},
	`while you do not have avatar of fire`:     Tag{"tag": Tag{"type": "Condition", "var": "HaveAvatarOfFire", "neg": true}},
	`if you have a summoned golem`:             Tag{"tag": Tag{"type": "Condition", "varList": []any{"HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem"}}},
	`while you have a summoned golem`:          Tag{"tag": Tag{"type": "Condition", "varList": []any{"HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem"}}},
	`if a minion has died recently`:            Tag{"tag": Tag{"type": "Condition", "var": "MinionsDiedRecently"}},
	`if a minion has been killed recently`:     Tag{"tag": Tag{"type": "Condition", "var": "MinionsDiedRecently"}},
	`while you have sacrificial zeal`:          Tag{"tag": Tag{"type": "Condition", "var": "SacrificialZeal"}},
	`while sane`:                               Tag{"tag": Tag{"type": "Condition", "var": "Insane", "neg": true}},
	`while insane`:                             Tag{"tag": Tag{"type": "Condition", "var": "Insane"}},
	`while you have defiance`:                  Tag{"tag": Tag{"type": "MultiplierThreshold", "var": "Defiance", "threshold": 1}},
	`while affected by glorious madness`:       Tag{"tag": Tag{"type": "Condition", "var": "AffectedByGloriousMadness"}},
	`if you've shattered an enemy recently`:    Tag{"tag": Tag{"type": "Condition", "var": "ShatteredEnemyRecently"}},
	`while affected by no flasks?`:             Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask", "neg": true}},
	`while affected by flasks?`:                Tag{"tag": Tag{"type": "Condition", "var": "UsingFlask"}},
	// Enemy status conditions
	`at close range`:                               Tag{"tag": Tag{"type": "Condition", "var": "AtCloseRange"}},
	`not at close range`:                           Tag{"tag": Tag{"type": "Condition", "var": "AtCloseRange", "neg": true}},
	`against rare and unique enemies`:              Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"}},
	`by s?l?a?i?n? rare [ao][nr]d? unique enemies`: Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"}},
	`against unique enemies`:                       Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "RareOrUnique"}},
	`against enemies on full life`:                 Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "FullLife"}},
	`against enemies that are on full life`:        Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "FullLife"}},
	`against enemies on low life`:                  Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "LowLife"}},
	`against enemies that are on low life`:         Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "LowLife"}},
	`against enemies that are not on low life`:     Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "LowLife", "neg": true}},
	`to enemies which have energy shield`:          Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "HaveEnergyShield"}, "keywordFlags": KeywordFlag.Hit | KeywordFlag.Ailment},
	`against cursed enemies`:                       Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}},
	`against stunned enemies`:                      Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Stunned"}},
	`on cursed enemies`:                            Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}},
	`of cursed enemies'`:                           Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}},
	`when hitting cursed enemies`:                  Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}, "keywordFlags": KeywordFlag.Hit},
	`from cursed enemies`:                          Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"}},
	`against marked enemy`:                         Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"}},
	`when hitting marked enemy`:                    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"}, "keywordFlags": KeywordFlag.Hit},
	`from marked enemy`:                            Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Marked"}},
	`against taunted enemies`:                      Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Taunted"}},
	`against bleeding enemies`:                     Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}},
	`you inflict on bleeding enemies`:              Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}},
	`to bleeding enemies`:                          Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}},
	`from bleeding enemies`:                        Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"}},
	`against poisoned enemies`:                     Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"}},
	`you inflict on poisoned enemies`:              Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"}},
	`to poisoned enemies`:                          Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Poisoned"}},
	`against enemies affected by ([0-9]+) or more poisons`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "PoisonStack", "threshold": c.n(1)}}
	}),
	`against enemies affected by at least ([0-9]+) poisons`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "PoisonStack", "threshold": c.n(1)}}
	}),
	`against hindered enemies`:                                   Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Hindered"}},
	`against maimed enemies`:                                     Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Maimed"}},
	`you inflict on maimed enemies`:                              Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Maimed"}},
	`against blinded enemies`:                                    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Blinded"}},
	`against excommunicated enemies`:                             Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Excommunicated"}},
	`from blinded enemies`:                                       Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Blinded"}},
	`against burning enemies`:                                    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Burning"}},
	`against ignited enemies`:                                    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"}},
	`to ignited enemies`:                                         Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"}},
	`against shocked enemies`:                                    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}},
	`you inflict on shocked enemies`:                             Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}},
	`to shocked enemies`:                                         Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}},
	`inflicted on shocked enemies`:                               Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}},
	`enemies which are shocked`:                                  Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}},
	`against frozen enemies`:                                     Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"}},
	`to frozen enemies`:                                          Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"}},
	`against chilled enemies`:                                    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}},
	`you inflict on chilled enemies`:                             Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}},
	`to chilled enemies`:                                         Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}},
	`inflicted on chilled enemies`:                               Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}},
	`enemies which are chilled`:                                  Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Chilled"}},
	`against chilled or frozen enemies`:                          Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Chilled", "Frozen"}}},
	`against frozen, shocked or ignited enemies`:                 Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Frozen", "Shocked", "Ignited"}}},
	`against enemies affected by elemental ailments`:             Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}}},
	`against enemies affected by ailments`:                       Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped", "Poisoned", "Bleeding"}}},
	`against enemies that are affected by elemental ailments`:    Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}}},
	`against enemies that are affected by no elemental ailments`: Tag{"tagList": []any{Tag{"type": "ActorCondition", "actor": "enemy", "varList": []any{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}, "neg": true}, Tag{"type": "Condition", "var": "Effective"}}},
	`against enemies affected by ([0-9]+) spider's webs`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "Spider's WebStack", "threshold": c.n(1)}}
	}),
	`against enemies on consecrated ground`:                                     Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnConsecratedGround"}},
	`against enemies with a higher percentage of their life remaining than you`: Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "HigherLifePercentThanPlayer"}},
	`if ([0-9]+)% of curse duration expired`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "MultiplierThreshold", "actor": "enemy", "var": "CurseExpired", "threshold": c.n(1)}}
	}),
	`against enemies with ([0-9a-zA-Z]+) exposure`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Has" + (firstToUpper(c.s(1)) + "Exposure")}}
	}),
	`by s?l?a?i?n? ?frozen enemies`:  Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Frozen"}},
	`by s?l?a?i?n? ?shocked enemies`: Tag{"tag": Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"}},
	// Enemy multipliers
	`per freeze, shock [ao][nr]d? ignite on enemy`: Tag{"tag": Tag{"type": "Multiplier", "var": "FreezeShockIgniteOnEnemy"}},
	`per poison affecting enemy`:                   Tag{"tag": Tag{"type": "Multiplier", "actor": "enemy", "var": "PoisonStack"}},
	`per poison affecting enemy, up to \+([0-9.]+)%`: fn(func(c caps) any {
		return Tag{"tag": Tag{"type": "Multiplier", "actor": "enemy", "var": "PoisonStack", "limit": c.n(1), "limitTotal": true}}
	}),
	`for each spider's web on the enemy`: Tag{"tag": Tag{"type": "Multiplier", "actor": "enemy", "var": "Spider's WebStack"}},
	// Hand-ported entries the transform could not express — ModParser.lua:1595,1631-1632,1650-1658,1739-1746,1810-1812.
	`if you have a ([a-zA-Z]+) ([a-zA-Z]+) in ([a-zA-Z]+) slot`: fn(func(c caps) any {
		slotIndex := ""
		switch c.s(3) {
		case "right":
			slotIndex = "2"
		case "left":
			slotIndex = "1"
		}
		return d(p("tag", Tag{"type": "Condition", "var": firstToUpper(c.s(1)) + "ItemIn" + firstToUpper(c.s(2)) + " " + slotIndex}))
	}),
	`while holding a ([0-9a-zA-Z]+)`: fn(func(c caps) any {
		return d(p("tag", Tag{"type": "Condition", "varList": []any{"Using" + firstToUpper(c.s(1))}}))
	}),
	`while holding a ([0-9a-zA-Z]+) or ([0-9a-zA-Z]+)`: fn(func(c caps) any {
		return d(p("tag", Tag{"type": "Condition", "varList": []any{"Using" + firstToUpper(c.s(1)), "Using" + firstToUpper(c.s(2))}}))
	}),
	// itemSlotName:sub(1, #itemSlotName - 1) drops the plural 's'.
	`if both equipped ([a-zA-Z \t\n\v\f\r]+) have a?n? ?([a-zA-Z \t\n\v\f\r]+) modifiers?`: fn(func(c caps) any {
		slot := c.s(1)
		if len(slot) > 0 {
			slot = slot[:len(slot)-1]
		}
		return d(p("tag", Tag{"type": "ItemCondition", "searchCond": c.s(2), "itemSlot": slot, "bothSlots": true}))
	}),
	`if both equipped left and right ([a-zA-Z \t\n\v\f\r]+) have a?n? ?([a-zA-Z \t\n\v\f\r]+) modifiers?`: fn(func(c caps) any {
		slot := c.s(1)
		if len(slot) > 0 {
			slot = slot[:len(slot)-1]
		}
		return d(p("tag", Tag{"type": "ItemCondition", "searchCond": c.s(2), "itemSlot": slot, "bothSlots": true}))
	}),
	`if equipped helmet, body armour, gloves, and boots all have armour`: d(p("tagList", []any{
		Tag{"type": "StatThreshold", "stat": "ArmourOnHelmet", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "ArmourOnBody Armour", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "ArmourOnGloves", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "ArmourOnBoots", "threshold": 1},
	})),
	`if equipped helmet, body armour, gloves, and boots all have evasion rating`: d(p("tagList", []any{
		Tag{"type": "StatThreshold", "stat": "EvasionOnHelmet", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "EvasionOnBody Armour", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "EvasionOnGloves", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "EvasionOnBoots", "threshold": 1},
	})),
	`if you have reserved life and mana`: d(p("tagList", []any{
		Tag{"type": "StatThreshold", "stat": "LifeReserved", "threshold": 1},
		Tag{"type": "StatThreshold", "stat": "ManaReserved", "threshold": 1},
	})),
}
