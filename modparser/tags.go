package modparser

// List of modifier tags — ModParser.lua:1319.
var modTagList = map[string]entryValue{
	`on enemies`:                               &PatternEntry{},
	`while active`:                             &PatternEntry{},
	`for ([0-9]+) seconds`:                     &PatternEntry{},
	`when you hit a unique enemy`:              &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"}},
	` on critical strike`:                      &PatternEntry{Tag: &CondTag{Var: "CriticalStrike"}},
	`from critical strikes`:                    &PatternEntry{Tag: &CondTag{Var: "CriticalStrike"}},
	`with critical strikes`:                    &PatternEntry{Tag: &CondTag{Var: "CriticalStrike"}},
	`by enemies killed with a critical strike`: &PatternEntry{TagList: []Tag{&CondTag{Var: "CritRecently"}, &CondTag{Var: "KilledRecently"}}},
	`while affected by auras you cast`:         &PatternEntry{Tag: &CondTag{Var: "AffectedByAura"}},
	`for you and nearby allies`:                &PatternEntry{NewAura: true},
	`to you and allies`:                        &PatternEntry{NewAura: true},
	// Multipliers
	`per power charge`:         &PatternEntry{Tag: &MultiplierTag{Var: "PowerCharge"}},
	`per frenzy charge`:        &PatternEntry{Tag: &MultiplierTag{Var: "FrenzyCharge"}},
	`per endurance charge`:     &PatternEntry{Tag: &MultiplierTag{Var: "EnduranceCharge"}},
	`per brine charge`:         &PatternEntry{Tag: &MultiplierTag{Var: "BrineCharge"}},
	`per siphoning charge`:     &PatternEntry{Tag: &MultiplierTag{Var: "SiphoningCharge"}},
	`per spirit charge`:        &PatternEntry{Tag: &MultiplierTag{Var: "SpiritCharge"}},
	`per challenger charge`:    &PatternEntry{Tag: &MultiplierTag{Var: "ChallengerCharge"}},
	`per maximum power charge`: &PatternEntry{Tag: &MultiplierTag{Var: "PowerChargeMax"}},
	`per gale force`:           &PatternEntry{Tag: &MultiplierTag{Var: "GaleForce"}},
	`per intensity`:            &PatternEntry{Tag: &MultiplierTag{Var: "Intensity"}},
	`per brand`:                &PatternEntry{Tag: &MultiplierTag{Var: "ActiveBrand"}},
	`per brand, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "ActiveBrand", GlobalLimit: opt(c.n(1)), GlobalLimitKey: "ChipAway"}}
	}),
	`per blitz charge`:                       &PatternEntry{Tag: &MultiplierTag{Var: "BlitzCharge"}},
	`per ghost shroud`:                       &PatternEntry{Tag: &MultiplierTag{Var: "GhostShroud"}},
	`per crab barrier`:                       &PatternEntry{Tag: &MultiplierTag{Var: "CrabBarrier"}},
	`per rage`:                               &PatternEntry{Tag: &MultiplierTag{Var: "Rage"}},
	`per rage while you are not losing rage`: &PatternEntry{Tag: &MultiplierTag{Var: "Rage"}},
	`per ([0-9]+) rage`:                      entryFn(func(c caps) *PatternEntry { return &PatternEntry{Tag: &MultiplierTag{Var: "Rage", Div: opt(c.n(1))}} }),
	`per mana burn`:                          &PatternEntry{Tag: &MultiplierTag{Var: "ManaBurnStacks"}},
	`per mana burn on you`:                   &PatternEntry{Tag: &MultiplierTag{Var: "ManaBurnStacks"}},
	`per mana burn, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "ManaBurnStacks", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per level`:                  &PatternEntry{Tag: &MultiplierTag{Var: "Level"}},
	`per ([0-9]+) player levels`: entryFn(func(c caps) *PatternEntry { return &PatternEntry{Tag: &MultiplierTag{Var: "Level", Div: opt(c.n(1))}} }),
	`per defiance`:               &PatternEntry{Tag: &MultiplierTag{Var: "Defiance"}},
	`per ([0-9]+)% ([a-zA-Z]+) effect on enemy`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: firstToUpper(c.s(2)) + "Effect", Div: opt(c.n(1)), Actor: "enemy"}}
	}),
	`for each equipped normal item`:          &PatternEntry{Tag: &MultiplierTag{Var: "NormalItem"}},
	`for each normal item equipped`:          &PatternEntry{Tag: &MultiplierTag{Var: "NormalItem"}},
	`for each normal item you have equipped`: &PatternEntry{Tag: &MultiplierTag{Var: "NormalItem"}},
	`for each equipped magic item`:           &PatternEntry{Tag: &MultiplierTag{Var: "MagicItem"}},
	`for each magic item equipped`:           &PatternEntry{Tag: &MultiplierTag{Var: "MagicItem"}},
	`for each magic item you have equipped`:  &PatternEntry{Tag: &MultiplierTag{Var: "MagicItem"}},
	`for each equipped rare item`:            &PatternEntry{Tag: &MultiplierTag{Var: "RareItem"}},
	`for each rare item equipped`:            &PatternEntry{Tag: &MultiplierTag{Var: "RareItem"}},
	`for each rare item you have equipped`:   &PatternEntry{Tag: &MultiplierTag{Var: "RareItem"}},
	`for each equipped unique item`:          &PatternEntry{Tag: &MultiplierTag{Var: "UniqueItem"}},
	`for each unique item equipped`:          &PatternEntry{Tag: &MultiplierTag{Var: "UniqueItem"}},
	`for each unique item you have equipped`: &PatternEntry{Tag: &MultiplierTag{Var: "UniqueItem"}},
	`per elder item equipped`:                &PatternEntry{Tag: &MultiplierTag{Var: "ElderItem"}},
	`per shaper item equipped`:               &PatternEntry{Tag: &MultiplierTag{Var: "ShaperItem"}},
	`per elder or shaper item equipped`:      &PatternEntry{Tag: &MultiplierTag{Var: "ShaperOrElderItem"}},
	`if ([0-9]+) ([a-zA-Z]+) items are equipped`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: firstToUpper(c.s(2)) + "Item", Threshold: opt(c.n(1))}}
	}),
	`for each corrupted item equipped`:   &PatternEntry{Tag: &MultiplierTag{Var: "CorruptedItem"}},
	`for each equipped corrupted item`:   &PatternEntry{Tag: &MultiplierTag{Var: "CorruptedItem"}},
	`for each uncorrupted item equipped`: &PatternEntry{Tag: &MultiplierTag{Var: "NonCorruptedItem"}},
	`per equipped claw`:                  &PatternEntry{Tag: &MultiplierTag{Var: "ClawItem"}},
	`per equipped dagger`:                &PatternEntry{Tag: &MultiplierTag{Var: "DaggerItem"}},
	`per equipped axe`:                   &PatternEntry{Tag: &MultiplierTag{Var: "AxeItem"}},
	`per equipped ring`:                  &PatternEntry{Tag: &MultiplierTag{Var: "RingItem"}},
	`per equipped flask`:                 &PatternEntry{Tag: &MultiplierTag{Var: "FlaskItem"}},
	`per equipped sword`:                 &PatternEntry{Tag: &MultiplierTag{Var: "SwordItem"}},
	`per equipped jewel`:                 &PatternEntry{Tag: &MultiplierTag{Var: "JewelItem"}},
	`per equipped mace`:                  &PatternEntry{Tag: &MultiplierTag{Var: "MaceItem"}},
	`per equipped sceptre`:               &PatternEntry{Tag: &MultiplierTag{Var: "SceptreItem"}},
	`per equipped wand`:                  &PatternEntry{Tag: &MultiplierTag{Var: "WandItem"}},
	`per claw`:                           &PatternEntry{Tag: &MultiplierTag{Var: "ClawItem"}},
	`per dagger`:                         &PatternEntry{Tag: &MultiplierTag{Var: "DaggerItem"}},
	`per axe`:                            &PatternEntry{Tag: &MultiplierTag{Var: "AxeItem"}},
	`per ring`:                           &PatternEntry{Tag: &MultiplierTag{Var: "RingItem"}},
	`per flask`:                          &PatternEntry{Tag: &MultiplierTag{Var: "FlaskItem"}},
	`per sword`:                          &PatternEntry{Tag: &MultiplierTag{Var: "SwordItem"}},
	`per jewel`:                          &PatternEntry{Tag: &MultiplierTag{Var: "JewelItem"}},
	`per mace`:                           &PatternEntry{Tag: &MultiplierTag{Var: "MaceItem"}},
	`per sceptre`:                        &PatternEntry{Tag: &MultiplierTag{Var: "SceptreItem"}},
	`per wand`:                           &PatternEntry{Tag: &MultiplierTag{Var: "WandItem"}},
	`per abyssa?l? jewel affecting you`:  &PatternEntry{Tag: &MultiplierTag{Var: "AbyssJewel"}},
	`for each herald b?u?f?f?s?k?i?l?l? ?affecting you`:    &PatternEntry{Tag: &MultiplierTag{Var: "Herald"}},
	`for each of your aura or herald skills affecting you`: &PatternEntry{Tag: &MultiplierTag{VarList: []string{"Herald", "AuraAffectingSelf"}}},
	`for each type of abyssa?l? jewel affecting you`:       &PatternEntry{Tag: &MultiplierTag{Var: "AbyssJewelType"}},
	`per (.+) eye jewel affecting you, up to a maximum of \+?([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: firstToUpper(c.s(1)) + "EyeJewel", Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`per sextant affecting the area`: &PatternEntry{Tag: &MultiplierTag{Var: "Sextant"}},
	`per buff on you`:                &PatternEntry{Tag: &MultiplierTag{Var: "BuffOnSelf"}},
	`per hit suppressed recently`:    &PatternEntry{Tag: &MultiplierTag{Var: "HitsSuppressedRecently"}},
	`per curse on enemy`:             &PatternEntry{Tag: &MultiplierTag{Var: "CurseOnEnemy"}},
	`for each curse on enemy`:        &PatternEntry{Tag: &MultiplierTag{Var: "CurseOnEnemy"}},
	`for each curse on the enemy`:    &PatternEntry{Tag: &MultiplierTag{Var: "CurseOnEnemy"}},
	`per curse on you`:               &PatternEntry{Tag: &MultiplierTag{Var: "CurseOnSelf"}},
	`per poison on you`:              &PatternEntry{Tag: &MultiplierTag{Var: "PoisonStack"}},
	`for each poison on you`:         &PatternEntry{Tag: &MultiplierTag{Var: "PoisonStack"}},
	`for each poison on you up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "PoisonStack", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per poison on you, up to ([0-9]+) per second`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "PoisonStack", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each poison you have inflicted recently`: &PatternEntry{Tag: &MultiplierTag{Var: "PoisonAppliedRecently"}},
	`per withered debuff on enemy`:                &PatternEntry{Tag: &MultiplierTag{Var: "WitheredStack", Actor: "enemy", Limit: opt(15)}},
	`for each poison you have inflicted recently, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "PoisonAppliedRecently", GlobalLimit: opt(c.n(1)), GlobalLimitKey: "DurationPerPoisonRecently"}}
	}),
	`for each time you have shocked a non-shocked enemy recently, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "ShockedNonShockedEnemyRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each shocked enemy you've killed recently`: &PatternEntry{Tag: &MultiplierTag{Var: "ShockedEnemyKilledRecently"}},
	`per enemy killed recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "EnemyKilledRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per ([0-9]+) rampage kills`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "Rampage", Div: opt(c.n(1)), Limit: opt(1000 / c.n(1)), LimitTotal: true}}
	}),
	`per minion, up to ?a? ?m?a?x?i?m?u?m? ?o?f? ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "SummonedMinion", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per minion from your non-vaal skills`: &PatternEntry{Tag: &MultiplierTag{Var: "NonVaalSummonedMinion"}},
	`per minion`:                           &PatternEntry{Tag: &MultiplierTag{Var: "SummonedMinion"}},
	`for each enemy you or your minions have killed recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{VarList: []string{"EnemyKilledRecently", "EnemyKilledByMinionsRecently"}, Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each enemy you or your minions have killed recently, up to ([0-9]+)% per second`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{VarList: []string{"EnemyKilledRecently", "EnemyKilledByMinionsRecently"}, Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each ([0-9]+) total mana y?o?u? ?h?a?v?e? ?spent recently`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "ManaSpentRecently", Div: opt(c.n(1))}}
	}),
	`for each ([0-9]+) total mana you have spent recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "ManaSpentRecently", Div: opt(c.n(1)), Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`per ([0-9]+) mana spent recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "ManaSpentRecently", Div: opt(c.n(1)), Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`for each time you've blocked in the past 10 seconds`: &PatternEntry{Tag: &MultiplierTag{Var: "BlockedPast10Sec"}},
	`per enemy killed by you or your totems recently`:     &PatternEntry{Tag: &MultiplierTag{VarList: []string{"EnemyKilledRecently", "EnemyKilledByTotemsRecently"}}},
	`per nearby enemy, up to \+?([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "NearbyEnemies", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per enemy in close range`:                               &PatternEntry{TagList: []Tag{&CondTag{Var: "AtCloseRange"}, &MultiplierTag{Var: "NearbyEnemies"}}},
	`per red socket`:                                         &PatternEntry{Tag: &MultiplierTag{Var: "RedSocketIn{SlotName}"}},
	`per green socket on main hand weapon`:                   &PatternEntry{Tag: &MultiplierTag{Var: "GreenSocketInWeapon 1"}},
	`per green socket on`:                                    &PatternEntry{Tag: &MultiplierTag{Var: "GreenSocketInWeapon 1"}},
	`per red socket on main hand weapon`:                     &PatternEntry{Tag: &MultiplierTag{Var: "RedSocketInWeapon 1"}},
	`per red socket on equipped staff`:                       &PatternEntry{TagList: []Tag{&MultiplierTag{Var: "RedSocketInWeapon 1"}, &CondTag{Var: "UsingStaff"}}},
	`per blue socket on equipped staff`:                      &PatternEntry{TagList: []Tag{&MultiplierTag{Var: "BlueSocketInWeapon 1"}, &CondTag{Var: "UsingStaff"}}},
	`per green socket`:                                       &PatternEntry{Tag: &MultiplierTag{Var: "GreenSocketIn{SlotName}"}},
	`per blue socket`:                                        &PatternEntry{Tag: &MultiplierTag{Var: "BlueSocketIn{SlotName}"}},
	`per white socket`:                                       &PatternEntry{Tag: &MultiplierTag{Var: "WhiteSocketIn{SlotName}"}},
	`for each unlinked socket in equipped two handed weapon`: &PatternEntry{TagList: []Tag{&MultiplierTag{Var: "UnlinkedSocketInWeapon 1"}, &CondTag{Var: "UsingTwoHandedWeapon"}}},
	`for each empty red socket on any equipped item`:         &PatternEntry{Tag: &MultiplierTag{Var: "EmptyRedSocketsInAnySlot"}},
	`for each empty green socket on any equipped item`:       &PatternEntry{Tag: &MultiplierTag{Var: "EmptyGreenSocketsInAnySlot"}},
	`for each empty blue socket on any equipped item`:        &PatternEntry{Tag: &MultiplierTag{Var: "EmptyBlueSocketsInAnySlot"}},
	`for each empty white socket on any equipped item`:       &PatternEntry{Tag: &MultiplierTag{Var: "EmptyWhiteSocketsInAnySlot"}},
	`per socketed gem`:                                       &PatternEntry{Tag: &MultiplierTag{Var: "SocketedGemsIn{SlotName}"}},
	`per socketed red gem`:                                   &PatternEntry{Tag: &MultiplierTag{Var: "SocketedRedGemsIn{SlotName}"}},
	`per socketed green gem`:                                 &PatternEntry{Tag: &MultiplierTag{Var: "SocketedGreenGemsIn{SlotName}"}},
	`per socketed blue gem`:                                  &PatternEntry{Tag: &MultiplierTag{Var: "SocketedBlueGemsIn{SlotName}"}},
	`per socketed murderous eye jewel`:                       &PatternEntry{Tag: &MultiplierTag{Var: "MurderousEyeJewelIn{SlotName}"}},
	`per socketed searching eye jewel`:                       &PatternEntry{Tag: &MultiplierTag{Var: "SearchingEyeJewelIn{SlotName}"}},
	`per socketed hypnotic eye jewel`:                        &PatternEntry{Tag: &MultiplierTag{Var: "HypnoticEyeJewelIn{SlotName}"}},
	`per socketed ghastly eye jewel`:                         &PatternEntry{Tag: &MultiplierTag{Var: "GhastlyEyeJewelIn{SlotName}"}},
	`for each impale on enemy`:                               &PatternEntry{Tag: &MultiplierTag{Var: "ImpaleStacks", Actor: "enemy"}},
	`per impale on enemy`:                                    &PatternEntry{Tag: &MultiplierTag{Var: "ImpaleStacks", Actor: "enemy"}},
	`per grasping vine`:                                      &PatternEntry{Tag: &MultiplierTag{Var: "GraspingVinesCount"}},
	`per fragile regrowth`:                                   &PatternEntry{Tag: &MultiplierTag{Var: "FragileRegrowthCount"}},
	`per bark`:                                               &PatternEntry{Tag: &MultiplierTag{Var: "BarkskinStacks"}},
	`per bark below maximum`:                                 &PatternEntry{Tag: &MultiplierTag{Var: "MissingBarkskinStacks"}},
	`per allocated mastery passive skill`:                    &PatternEntry{Tag: &MultiplierTag{Var: "AllocatedMastery"}},
	`per allocated notable passive skill`:                    &PatternEntry{Tag: &MultiplierTag{Var: "AllocatedNotable"}},
	`for each different type of mastery you have allocated`:  &PatternEntry{Tag: &MultiplierTag{Var: "AllocatedMasteryType"}},
	`per grand spectrum`:                                     &PatternEntry{Tag: &MultiplierTag{Var: "GrandSpectrum"}},
	`per second you've been stationary, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "StationarySeconds", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per elemental ailment you've inflicted recently`: &PatternEntry{Tag: &MultiplierTag{Var: "AppliedAilmentsRecently"}},
	// Per stat
	`per ([0-9]+)% of maximum mana they reserve`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ManaReservedPercent", Div: opt(c.n(1))}}
	}),
	`for each ([0-9]+)% of life reserved`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LifeReservedPercent", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) strength`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Str", Div: opt(c.n(1))}}
	}),
	`per dexterity`: &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Dex"}},
	`per ([0-9]+) dexterity`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Dex", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) intelligence`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Int", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) omniscience`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Omni", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) total attributes`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, StatList: []string{"Str", "Dex", "Int"}, Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) of your lowest attribute`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LowestAttribute", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) reserved life`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LifeReserved", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) unreserved maximum mana`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ManaUnreserved", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) unreserved maximum mana, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ManaUnreserved", Div: opt(c.n(1)), Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`per ([0-9]+) armour`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Armour", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Evasion", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Evasion", Div: opt(c.n(1)), Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`per ([0-9]+) maximum energy shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShield", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) player maximum energy shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShield", Div: opt(c.n(1)), Actor: "player"}}
	}),
	`per ([0-9]+) maximum life`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Life", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) of maximum life or maximum mana, whichever is lower`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LowestOfMaximumLifeAndMaximumMana", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) player maximum life`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Life", Div: opt(c.n(1)), Actor: "player"}}
	}),
	`per ([0-9]+) maximum mana`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Mana", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) maximum mana, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Mana", Div: opt(c.n(1)), Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`per ([0-9]+) maximum mana, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Mana", Div: opt(c.n(1)), Limit: opt(c.n(2)), LimitTotal: true}}
	}),
	`per soul required`: &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "SoulCost"}},
	`per ([0-9]+) accuracy rating`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Accuracy", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% block chance`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "BlockChance", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% chance to block on equipped shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ShieldBlockChance", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% chance to block attack damage`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "BlockChance", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% chance to block spell damage`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "SpellBlockChance", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) of the lowest of armour and evasion rating`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LowestOfArmourAndEvasion", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) energy shield on equipped gloves`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnGloves", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) maximum energy shield on helmet`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnHelmet", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) maximum energy shield on equipped helmet`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnHelmet", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) energy shield on equipped helmet`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnHelmet", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) energy shield on equipped boots`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnBoots", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) energy shield on equipped body armour`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnBody Armour", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) maximum energy shield on equipped shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnWeapon 2", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) maximum energy shield on shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EnergyShieldOnWeapon 2", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating on equipped gloves`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnGloves", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating on equipped helmet`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnHelmet", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion on equipped boots`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnBoots", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion on boots`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnBoots", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating on equipped boots`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnBoots", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating on body armour`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnBody Armour", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating on equipped body armour`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnBody Armour", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) evasion rating on equipped shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "EvasionOnWeapon 2", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour on gloves`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ArmourOnGloves", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour on equipped gloves`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ArmourOnGloves", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour on equipped helmet`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ArmourOnHelmet", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour on equipped boots`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ArmourOnBoots", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour on equipped body armour`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ArmourOnBody Armour", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour on equipped shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ArmourOnWeapon 2", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour or evasion rating on shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, StatList: []string{"ArmourOnWeapon 2", "EvasionOnWeapon 2"}, Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) armour or evasion rating on equipped shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, StatList: []string{"ArmourOnWeapon 2", "EvasionOnWeapon 2"}, Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% cold resistance`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ColdResist", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% fire resistance`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "FireResist", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% lightning resistance`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LightningResist", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% chaos resistance`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ChaosResist", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% cold resistance above 75%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ColdResistOver75", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% lightning resistance above 75%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "LightningResistOver75", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% fire resistance above 75%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "FireResistOver75", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% fire, cold, or lightning resistance above 75%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, StatList: []string{"FireResistOver75", "ColdResistOver75", "LightningResistOver75"}, Div: opt(c.n(1))}}
	}),
	`per ([0-9]+) devotion`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Devotion", Actor: "parent", Div: opt(c.n(1))}}
	}),
	`per ([0-9]+)% missing fire resistance, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "MissingFireResist", Div: opt(c.n(1)), GlobalLimit: opt(c.n(2)), GlobalLimitKey: "ReplicaNebulisFire"}}
	}),
	`per ([0-9]+)% missing cold resistance, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "MissingColdResist", Div: opt(c.n(1)), GlobalLimit: opt(c.n(2)), GlobalLimitKey: "ReplicaNebulisCold"}}
	}),
	`per ([0-9]+)% missing fire, cold, or lightning resistance, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, StatList: []string{"MissingFireResist", "MissingColdResist", "MissingLightningResist"}, Div: opt(c.n(1)), GlobalLimit: opt(c.n(2)), GlobalLimitKey: "ReplicaNebulisCold"}}
	}),
	`per endurance, frenzy or power charge`: &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "TotalCharges"}},
	`per fortification`:                     &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "FortificationStacks"}},
	`per two fortification on you`:          &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "FortificationStacks", Div: opt(2), Actor: "player"}},
	`per fortification above 20`:            &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "FortificationStacksOver20"}},
	`per totem`:                             &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "TotemsSummoned"}},
	`per summoned totem`:                    &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "TotemsSummoned"}},
	`for each summoned totem`:               &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "TotemsSummoned"}},
	`per maximum number of summoned totems`: &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveTotemLimit"}},
	`for each time they have chained`:       &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Chain"}},
	`for each time it has chained`:          &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "Chain"}},
	`for each summoned golem`:               &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveGolemLimit"}},
	`for each golem you have summoned`:      &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveGolemLimit"}},
	`per summoned golem`:                    &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveGolemLimit"}},
	`per summoned sentinel of purity`:       &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveSentinelOfPurityLimit"}},
	`per summoned void spawn`:               &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveVoidSpawnLimit"}},
	`per summoned skeleton`:                 &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveSkeletonLimit"}},
	`per skeleton you own`:                  &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveSkeletonLimit", Actor: "parent"}},
	`per summoned raging spirit`:            &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveRagingSpiritLimit"}},
	`per summoned phantasm`:                 &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActivePhantasmLimit"}},
	`per animated weapon`:                   &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveAnimatedWeaponLimit", Actor: "parent"}},
	`for each raised zombie`:                &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveZombieLimit"}},
	`per zombie you own`:                    &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveZombieLimit", Actor: "parent"}},
	`per raised zombie`:                     &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveZombieLimit"}},
	`per raised spectre`:                    &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveSpectreLimit"}},
	`per spectre you own`:                   &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ActiveSpectreLimit", Actor: "parent"}},
	`for each remaining chain`:              &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ChainRemaining"}},
	`for each remaining chain, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "ChainRemaining", GlobalLimit: opt(c.n(1)), GlobalLimitKey: "FollowThrough"}}
	}),
	`for each enemy pierced`:        &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "PiercedCount"}},
	`for each time they've pierced`: &PatternEntry{Tag: &StatTag{StatKind: TagPerStat, Stat: "PiercedCount"}},
	// Stat conditions
	`with ([0-9]+) or more strength`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Str", Threshold: opt(c.n(1))}}
	}),
	`with at least ([0-9]+) strength`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Str", Threshold: opt(c.n(1))}}
	}),
	`w?h?i[lf]e? you have at least ([0-9]+) strength`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Str", Threshold: opt(c.n(1))}}
	}),
	`w?h?i[lf]e? you have at least ([0-9]+) dexterity`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Dex", Threshold: opt(c.n(1))}}
	}),
	`w?h?i[lf]e? you have at least ([0-9]+) intelligence`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Int", Threshold: opt(c.n(1))}}
	}),
	`w?h?i[lf]e? strength is below ([0-9]+)`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Str", Threshold: opt(c.n(1) - 1), Upper: true}}
	}),
	`w?h?i[lf]e? dexterity is below ([0-9]+)`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Dex", Threshold: opt(c.n(1) - 1), Upper: true}}
	}),
	`w?h?i[lf]e? intelligence is below ([0-9]+)`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Int", Threshold: opt(c.n(1) - 1), Upper: true}}
	}),
	`at least ([0-9]+) intelligence`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Int", Threshold: opt(c.n(1))}}
	}),
	`if dexterity is higher than intelligence`: &PatternEntry{Tag: &CondTag{Var: "DexHigherThanInt"}},
	`if strength is higher than intelligence`:  &PatternEntry{Tag: &CondTag{Var: "StrHigherThanInt"}},
	`w?h?i[lf]e? you have at least ([0-9]+) maximum energy shield`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "EnergyShield", Threshold: opt(c.n(1))}}
	}),
	`against targets they pierce`:   &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)}},
	`against pierced targets`:       &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)}},
	`to targets they pierce`:        &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PierceCount", Threshold: opt(1)}},
	`that fire a single projectile`: &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "ProjectileCount", Threshold: opt(1), Upper: true}},
	`w?h?i[lf]e? you have at least ([0-9]+) devotion`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "Devotion", Threshold: opt(c.n(1))}}
	}),
	`while you have at least ([0-9]+) rage`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "Rage", Threshold: opt(c.n(1))}}
	}),
	`while affected by a unique abyss jewel`: &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "UniqueAbyssJewels", Threshold: opt(1)}},
	`while affected by a rare abyss jewel`:   &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "RareAbyssJewels", Threshold: opt(1)}},
	`while affected by a magic abyss jewel`:  &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "MagicAbyssJewels", Threshold: opt(1)}},
	`while affected by a normal abyss jewel`: &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NormalAbyssJewels", Threshold: opt(1)}},
	`while you have at least ([0-9]+) nearby all[yi]e?s?`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NearbyAlly", Threshold: opt(c.n(1))}}
	}),
	// Slot conditions
	`when in main hand`:     &PatternEntry{Tag: &SlotTag{SlotKind: TagSlotNumber, Num: 1}},
	`whi?l?en? in off hand`: &PatternEntry{Tag: &SlotTag{SlotKind: TagSlotNumber, Num: 2}},
	`in main hand`:          &PatternEntry{Tag: &SlotTag{SlotKind: TagInSlot, Num: 1}},
	`in off hand`:           &PatternEntry{Tag: &SlotTag{SlotKind: TagInSlot, Num: 2}},
	`w?i?t?h? main hand`:    &PatternEntry{TagList: []Tag{&CondTag{Var: "MainHandAttack"}, &SkillTypeTag{SkillType: SkillTypeAttack}}},
	`w?i?t?h? off ?hand`:    &PatternEntry{TagList: []Tag{&CondTag{Var: "OffHandAttack"}, &SkillTypeTag{SkillType: SkillTypeAttack}}},
	`[fi]?[rn]?[of]?[ml]?[ i]?[hc]?[it]?[te]?[sd]? ? with this weapon`: &PatternEntry{TagList: []Tag{&CondTag{Var: "{Hand}Attack"}, &SkillTypeTag{SkillType: SkillTypeAttack}}},
	`if your o[tp][hp][eo][rs]i?t?e? ring is a shaper item`:            &PatternEntry{Tag: &ItemCondTag{ItemSlot: "Ring {OtherSlotNum}", ShaperCond: true}},
	`if your o[tp][hp][eo][rs]i?t?e? ring is an elder item`:            &PatternEntry{Tag: &ItemCondTag{ItemSlot: "Ring {OtherSlotNum}", ElderCond: true}},
	`of skills supported by spellslinger`:                              &PatternEntry{Tag: &CondTag{Var: "SupportedBySpellslinger"}},
	// Equipment conditions
	`while holding a fishing rod`:               &PatternEntry{Tag: &CondTag{Var: "UsingFishing"}},
	`while your off hand is empty`:              &PatternEntry{Tag: &CondTag{Var: "OffHandIsEmpty"}},
	`with shields`:                              &PatternEntry{Tag: &CondTag{Var: "UsingShield"}},
	`while dual wielding`:                       &PatternEntry{Tag: &CondTag{Var: "DualWielding"}},
	`while dual wielding claws`:                 &PatternEntry{Tag: &CondTag{Var: "DualWieldingClaws"}},
	`while dual wielding or holding a shield`:   &PatternEntry{Tag: &CondTag{VarList: []string{"DualWielding", "UsingShield"}}},
	`while wielding an axe`:                     &PatternEntry{Tag: &CondTag{Var: "UsingAxe"}},
	`while wielding an axe or sword`:            &PatternEntry{Tag: &CondTag{VarList: []string{"UsingAxe", "UsingSword"}}},
	`while wielding a bow`:                      &PatternEntry{Tag: &CondTag{Var: "UsingBow"}},
	`while wielding a claw`:                     &PatternEntry{Tag: &CondTag{Var: "UsingClaw"}},
	`while wielding a dagger`:                   &PatternEntry{Tag: &CondTag{Var: "UsingDagger"}},
	`while wielding a claw or dagger`:           &PatternEntry{Tag: &CondTag{VarList: []string{"UsingClaw", "UsingDagger"}}},
	`while wielding a mace`:                     &PatternEntry{Tag: &CondTag{Var: "UsingMace"}},
	`while wielding a mace or sceptre`:          &PatternEntry{Tag: &CondTag{Var: "UsingMace"}},
	`while wielding a mace, sceptre or staff`:   &PatternEntry{Tag: &CondTag{VarList: []string{"UsingMace", "UsingStaff"}}},
	`while wielding a staff`:                    &PatternEntry{Tag: &CondTag{Var: "UsingStaff"}},
	`while wielding a sword`:                    &PatternEntry{Tag: &CondTag{Var: "UsingSword"}},
	`while wielding a melee weapon`:             &PatternEntry{Tag: &CondTag{Var: "UsingMeleeWeapon"}},
	`while wielding a one handed weapon`:        &PatternEntry{Tag: &CondTag{Var: "UsingOneHandedWeapon"}},
	`while wielding a two handed weapon`:        &PatternEntry{Tag: &CondTag{Var: "UsingTwoHandedWeapon"}},
	`while wielding a two handed melee weapon`:  &PatternEntry{TagList: []Tag{&CondTag{Var: "UsingTwoHandedWeapon"}, &CondTag{Var: "UsingMeleeWeapon"}}},
	`while wielding a wand`:                     &PatternEntry{Tag: &CondTag{Var: "UsingWand"}},
	`while wielding two different weapon types`: &PatternEntry{Tag: &CondTag{Var: "WieldingDifferentWeaponTypes"}},
	`while unarmed`:                             &PatternEntry{Tag: &CondTag{Var: "Unarmed"}},
	`while you are unencumbered`:                &PatternEntry{Tag: &CondTag{Var: "Unencumbered"}},
	`equipped bow`:                              &PatternEntry{Tag: &CondTag{Var: "UsingBow"}},
	`if equipped ([a-zA-Z \t\n\v\f\r]+) has an ([a-zA-Z \t\n\v\f\r]+) modifier`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &ItemCondTag{SearchCond: c.s(2), ItemSlot: c.s(1)}}
	}),
	`if your equipped staff has a red and blue socket`: &PatternEntry{TagList: []Tag{&MultiplierTag{IsThreshold: true, Var: "RedSocketInWeapon 1", Threshold: opt(1)}, &MultiplierTag{IsThreshold: true, Var: "BlueSocketInWeapon 1", Threshold: opt(1)}, &CondTag{Var: "UsingStaff"}}},
	`if there are no ([a-zA-Z \t\n\v\f\r]+) modifiers on equipped ([a-zA-Z \t\n\v\f\r]+)`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &ItemCondTag{SearchCond: c.s(1), ItemSlot: c.s(2), Neg: true}}
	}),
	`if there are no ([a-zA-Z]+) modifiers on other equipped items`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &ItemCondTag{SearchCond: c.s(1), ItemSlot: "{SlotName}", AllSlots: true, ExcludeSelf: true, Neg: true}}
	}),
	`if corrupted`:                        &PatternEntry{Tag: &ItemCondTag{ItemSlot: "{SlotName}", CorruptedCond: true}},
	`with a normal item equipped`:         &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NormalItem", Threshold: opt(1)}},
	`with a magic item equipped`:          &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "MagicItem", Threshold: opt(1)}},
	`with a rare item equipped`:           &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "RareItem", Threshold: opt(1)}},
	`with a unique item equipped`:         &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "UniqueItem", Threshold: opt(1)}},
	`if you wear no corrupted items`:      &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "CorruptedItem", Threshold: opt(0), Upper: true}},
	`if no worn items are corrupted`:      &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "CorruptedItem", Threshold: opt(0), Upper: true}},
	`if no equipped items are corrupted`:  &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "CorruptedItem", Threshold: opt(0), Upper: true}},
	`if all worn items are corrupted`:     &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NonCorruptedItem", Threshold: opt(0), Upper: true}},
	`if all equipped items are corrupted`: &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NonCorruptedItem", Threshold: opt(0), Upper: true}},
	`if equipped shield has at least ([0-9]+)% chance to block`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "ShieldBlockChance", Threshold: opt(c.n(1))}}
	}),
	`if you have ([0-9]+) primordial items socketed or equipped`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "PrimordialItem", Threshold: opt(c.n(1))}}
	}),
	// Player status conditions
	`if used while on low life`:              &PatternEntry{Tag: &CondTag{Var: "LowLife"}},
	`wh[ie][ln]e? on low life`:               &PatternEntry{Tag: &CondTag{Var: "LowLife"}},
	`on reaching low life`:                   &PatternEntry{Tag: &CondTag{Var: "LowLife"}},
	`wh[ie][ln]e? not on low life`:           &PatternEntry{Tag: &CondTag{Var: "LowLife", Neg: true}},
	`wh[ie][ln]e? on low mana`:               &PatternEntry{Tag: &CondTag{Var: "LowMana"}},
	`wh[ie][ln]e? not on low mana`:           &PatternEntry{Tag: &CondTag{Var: "LowMana", Neg: true}},
	`wh[ie][ln]e? on full life`:              &PatternEntry{Tag: &CondTag{Var: "FullLife"}},
	`wh[ie][ln]e? not on full life`:          &PatternEntry{Tag: &CondTag{Var: "FullLife", Neg: true}},
	`wh[ie][ln]e? no life is reserved`:       &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "LifeReserved", Threshold: opt(0), Upper: true}},
	`wh[ie][ln]e? no mana is reserved`:       &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "ManaReserved", Threshold: opt(0), Upper: true}},
	`wh[ie][ln]e? on full energy shield`:     &PatternEntry{Tag: &CondTag{Var: "FullEnergyShield"}},
	`wh[ie][ln]e? not on full energy shield`: &PatternEntry{Tag: &CondTag{Var: "FullEnergyShield", Neg: true}},
	`wh[ie][ln]e? you have energy shield`:    &PatternEntry{Tag: &CondTag{Var: "HaveEnergyShield"}},
	`wh[ie][ln]e? you have no energy shield`: &PatternEntry{Tag: &CondTag{Var: "HaveEnergyShield", Neg: true}},
	`if you have energy shield`:              &PatternEntry{Tag: &CondTag{Var: "HaveEnergyShield"}},
	`while stationary`:                       &PatternEntry{Tag: &CondTag{Var: "Stationary"}},
	`while you are stationary`:               &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "player", Var: "Stationary"}},
	`while moving`:                           &PatternEntry{Tag: &CondTag{Var: "Moving"}},
	`while channelling`:                      &PatternEntry{Tag: &CondTag{Var: "Channelling"}},
	`while channelling snipe`:                &PatternEntry{Tag: &CondTag{Var: "Channelling"}},
	`after channelling for ([0-9]+) seconds?`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "ChannellingTime", Threshold: opt(c.n(1))}}
	}),
	`if you've been channelling for at least ([0-9]+) seconds?`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "ChannellingTime", Threshold: opt(c.n(1))}}
	}),
	`if you've inflicted exposure recently`: &PatternEntry{Tag: &CondTag{Var: "AppliedExposureRecently"}},
	`while you have no power charges`:       &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", Threshold: opt(0), Upper: true}},
	`while you have no frenzy charges`:      &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", Threshold: opt(0), Upper: true}},
	`while you have no endurance charges`:   &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", Threshold: opt(0), Upper: true}},
	`while you have a power charge`:         &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", Threshold: opt(1)}},
	`while you have a frenzy charge`:        &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", Threshold: opt(1)}},
	`while you have an endurance charge`:    &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", Threshold: opt(1)}},
	`while at maximum power charges`:        &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", ThresholdStat: "PowerChargesMax"}},
	`while at maximum frenzy charges`:       &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"}},
	`while on full frenzy charges`:          &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMax"}},
	`while at maximum endurance charges`:    &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMax"}},
	`while at minimum endurance charges`:    &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "EnduranceCharges", ThresholdStat: "EnduranceChargesMin", Upper: true}},
	`while at minimum power charges`:        &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "PowerCharges", ThresholdStat: "PowerChargesMin", Upper: true}},
	`while at minimum frenzy charges`:       &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "FrenzyCharges", ThresholdStat: "FrenzyChargesMin", Upper: true}},
	`while at maximum rage`:                 &PatternEntry{Tag: &CondTag{Var: "HaveMaximumRage"}},
	`while at maximum fortification`:        &PatternEntry{Tag: &CondTag{Var: "HaveMaximumFortification"}},
	`while you have at least ([0-9]+) crab barriers`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "CrabBarriers", Threshold: opt(c.n(1))}}
	}),
	`while you have at least ([0-9]+) fortification`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &StatTag{StatKind: TagStatThreshold, Stat: "FortificationStacks", Threshold: opt(c.n(1))}}
	}),
	`while you have at least ([0-9]+) total endurance, frenzy and power charges`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "TotalCharges", Threshold: opt(c.n(1))}}
	}),
	`while you have a totem`:                             &PatternEntry{Tag: &CondTag{Var: "HaveTotem"}},
	`while you have at least one nearby ally`:            &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NearbyAlly", Threshold: opt(1)}},
	`while you have a linked target`:                     &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "LinkedTargets", Threshold: opt(1)}},
	`while you have fortify`:                             &PatternEntry{Tag: &CondTag{Var: "Fortified"}},
	`while you have phasing`:                             &PatternEntry{Tag: &CondTag{Var: "Phasing"}},
	`while you have unbroken ward`:                       &PatternEntry{Tag: &CondTag{Var: "UnbrokenWard"}},
	`while your ward is broken`:                          &PatternEntry{Tag: &CondTag{Var: "UnbrokenWard", Neg: true}},
	`if you[' ]h?a?ve suppressed spell damage recently`:  &PatternEntry{Tag: &CondTag{Var: "SuppressedRecently"}},
	`while you have elusive`:                             &PatternEntry{Tag: &CondTag{Var: "Elusive"}},
	`while physical aegis is depleted`:                   &PatternEntry{Tag: &CondTag{Var: "PhysicalAegisDepleted"}},
	`during onslaught`:                                   &PatternEntry{Tag: &CondTag{Var: "Onslaught"}},
	`while you have onslaught`:                           &PatternEntry{Tag: &CondTag{Var: "Onslaught"}},
	`while phasing`:                                      &PatternEntry{Tag: &CondTag{Var: "Phasing"}},
	`while you have tailwind`:                            &PatternEntry{Tag: &CondTag{Var: "Tailwind"}},
	`while elusive`:                                      &PatternEntry{Tag: &CondTag{Var: "Elusive"}},
	`gain elusive`:                                       &PatternEntry{Tag: &CondTag{VarList: []string{"CanBeElusive", "Elusive"}}},
	`while you have arcane surge`:                        &PatternEntry{Tag: &CondTag{Var: "AffectedByArcaneSurge"}},
	`while you have cat's stealth`:                       &PatternEntry{Tag: &CondTag{Var: "AffectedByCat'sStealth"}},
	`while you have cat's agility`:                       &PatternEntry{Tag: &CondTag{Var: "AffectedByCat'sAgility"}},
	`while you have avian's might`:                       &PatternEntry{Tag: &CondTag{Var: "AffectedByAvian'sMight"}},
	`while you have avian's flight`:                      &PatternEntry{Tag: &CondTag{Var: "AffectedByAvian'sFlight"}},
	`while affected by aspect of the cat`:                &PatternEntry{Tag: &CondTag{VarList: []string{"AffectedByCat'sStealth", "AffectedByCat'sAgility"}}},
	`while affected by a non-vaal guard skill`:           &PatternEntry{Tag: &CondTag{Var: "AffectedByNonVaalGuardSkill"}},
	`if a non-vaal guard buff was lost recently`:         &PatternEntry{Tag: &CondTag{Var: "LostNonVaalBuffRecently"}},
	`while affected by a guard skill buff`:               &PatternEntry{Tag: &CondTag{Var: "AffectedByGuardSkill"}},
	`while affected by a herald`:                         &PatternEntry{Tag: &CondTag{Var: "AffectedByHerald"}},
	`while fortified`:                                    &PatternEntry{Tag: &CondTag{Var: "Fortified"}},
	`while in blood stance`:                              &PatternEntry{Tag: &CondTag{Var: "BloodStance"}},
	`while in sand stance`:                               &PatternEntry{Tag: &CondTag{Var: "SandStance"}},
	`while you have a bestial minion`:                    &PatternEntry{Tag: &CondTag{Var: "HaveBestialMinion"}},
	`while you have infusion`:                            &PatternEntry{Tag: &CondTag{Var: "InfusionActive"}},
	`while focus?sed`:                                    &PatternEntry{Tag: &CondTag{Var: "Focused"}},
	`while leeching`:                                     &PatternEntry{Tag: &CondTag{Var: "Leeching"}},
	`while leeching life`:                                &PatternEntry{Tag: &CondTag{Var: "LeechingLife"}},
	`while leeching energy shield`:                       &PatternEntry{Tag: &CondTag{Var: "LeechingEnergyShield"}},
	`while leeching mana`:                                &PatternEntry{Tag: &CondTag{Var: "LeechingMana"}},
	`while using a flask`:                                &PatternEntry{Tag: &CondTag{Var: "UsingFlask"}},
	`during effect`:                                      &PatternEntry{Tag: &CondTag{Var: "UsingFlask"}},
	`during flask effect`:                                &PatternEntry{Tag: &CondTag{Var: "UsingFlask"}},
	`during any flask effect`:                            &PatternEntry{Tag: &CondTag{Var: "UsingFlask"}},
	`while under no flask effects`:                       &PatternEntry{Tag: &CondTag{Var: "UsingFlask", Neg: true}},
	`while affected by no flasks`:                        &PatternEntry{Tag: &CondTag{Var: "UsingFlask", Neg: true}},
	`during effect of any mana flask`:                    &PatternEntry{Tag: &CondTag{Var: "UsingManaFlask"}},
	`during effect of any life flask`:                    &PatternEntry{Tag: &CondTag{Var: "UsingLifeFlask"}},
	`if you've used a life flask in the past 10 seconds`: &PatternEntry{Tag: &CondTag{Var: "UsingLifeFlask"}},
	`if you've used a mana flask in the past 10 seconds`: &PatternEntry{Tag: &CondTag{Var: "UsingManaFlask"}},
	`during effect of any life or mana flask`:            &PatternEntry{Tag: &CondTag{VarList: []string{"UsingManaFlask", "UsingLifeFlask"}}},
	`while you have an active tincture`:                  &PatternEntry{Tag: &CondTag{Var: "UsingTincture"}},
	`while you have a tincture active`:                   &PatternEntry{Tag: &CondTag{Var: "UsingTincture"}},
	`with at least one ([0-9a-zA-Z]+) grafted to you`:    entryFn(func(c caps) *PatternEntry { return &PatternEntry{Tag: &CondTag{Var: "Using" + firstToUpper(c.s(1))}} }),
	`while on consecrated ground`:                        &PatternEntry{Tag: &CondTag{Var: "OnConsecratedGround"}},
	`while on caustic ground`:                            &PatternEntry{Tag: &CondTag{Var: "OnCausticGround"}},
	`when you create consecrated ground`:                 &PatternEntry{},
	`on burning ground`:                                  &PatternEntry{Tag: &CondTag{Var: "OnBurningGround"}},
	`while on burning ground`:                            &PatternEntry{Tag: &CondTag{Var: "OnBurningGround"}},
	`on chilled ground`:                                  &PatternEntry{Tag: &CondTag{Var: "OnChilledGround"}},
	`on shocked ground`:                                  &PatternEntry{Tag: &CondTag{Var: "OnShockedGround"}},
	`while in a caustic cloud`:                           &PatternEntry{Tag: &CondTag{Var: "OnCausticCloud"}},
	`while blinded`:                                      &PatternEntry{TagList: []Tag{&CondTag{Var: "Blinded"}, &CondTag{Var: "CannotBeBlinded", Neg: true}}},
	`while burning`:                                      &PatternEntry{Tag: &CondTag{Var: "Burning"}},
	`while ignited`:                                      &PatternEntry{Tag: &CondTag{Var: "Ignited"}},
	`while you are ignited`:                              &PatternEntry{Tag: &CondTag{Var: "Ignited"}},
	`while chilled`:                                      &PatternEntry{Tag: &CondTag{Var: "Chilled"}},
	`while you are chilled`:                              &PatternEntry{Tag: &CondTag{Var: "Chilled"}},
	`while frozen`:                                       &PatternEntry{Tag: &CondTag{Var: "Frozen"}},
	`while shocked`:                                      &PatternEntry{Tag: &CondTag{Var: "Shocked"}},
	`while you are shocked`:                              &PatternEntry{Tag: &CondTag{Var: "Shocked"}},
	`while you are bleeding`:                             &PatternEntry{Tag: &CondTag{Var: "Bleeding"}},
	`while not ignited, frozen or shocked`:               &PatternEntry{Tag: &CondTag{VarList: []string{"Ignited", "Frozen", "Shocked"}, Neg: true}},
	`while bleeding`:                                     &PatternEntry{Tag: &CondTag{Var: "Bleeding"}},
	`while poisoned`:                                     &PatternEntry{Tag: &CondTag{Var: "Poisoned"}},
	`wh[ei][nl][ e] ?you are poisoned`:                   &PatternEntry{Tag: &CondTag{Var: "Poisoned"}},
	`while cursed`:                                       &PatternEntry{Tag: &CondTag{Var: "Cursed"}},
	`while not cursed`:                                   &PatternEntry{Tag: &CondTag{Var: "Cursed", Neg: true}},
	`while there is only one nearby enemy`:               &PatternEntry{TagList: []Tag{&MultiplierTag{Var: "NearbyEnemies", Limit: opt(1)}, &CondTag{Var: "OnlyOneNearbyEnemy"}}},
	`while at least ([0-9]+) enemies are nearby`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "NearbyEnemies", Threshold: opt(c.n(1))}}
	}),
	`while t?h?e?r?e? ?i?s? ?a rare or unique enemy i?s? ?nearby`:                      &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"NearbyRareOrUniqueEnemy", "RareOrUnique"}}},
	`if you[' ]h?a?ve hit recently`:                                                    &PatternEntry{Tag: &CondTag{Var: "HitRecently"}},
	`if you[' ]h?a?ve hit an enemy recently`:                                           &PatternEntry{Tag: &CondTag{Var: "HitRecently"}},
	`if you[' ]h?a?ve hit with your main hand weapon recently`:                         &PatternEntry{Tag: &CondTag{Var: "HitRecentlyWithWeapon"}},
	`if you[' ]h?a?ve hit with your off hand weapon recently`:                          &PatternEntry{TagList: []Tag{&CondTag{Var: "HitRecentlyWithWeapon"}, &CondTag{Var: "DualWielding"}}},
	`if you[' ]h?a?ve hit a cursed enemy recently`:                                     &PatternEntry{TagList: []Tag{&CondTag{Var: "HitRecently"}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}}},
	`when you or your totems hit an enemy with a spell`:                                &PatternEntry{Tag: &CondTag{VarList: []string{"HitSpellRecently", "TotemsHitSpellRecently"}}},
	`on hit with spells`:                                                               &PatternEntry{Tag: &CondTag{Var: "HitSpellRecently"}},
	`if you[' ]h?a?ve crit recently`:                                                   &PatternEntry{Tag: &CondTag{Var: "CritRecently"}},
	`if you[' ]h?a?ve dealt a critical strike recently`:                                &PatternEntry{Tag: &CondTag{Var: "CritRecently"}},
	`when you deal a critical strike`:                                                  &PatternEntry{Tag: &CondTag{Var: "CritRecently"}},
	`if you[' ]h?a?ve dealt a critical strike with this weapon recently`:               &PatternEntry{Tag: &CondTag{Var: "CritRecently"}},
	`if you[' ]h?a?ve crit in the past 8 seconds`:                                      &PatternEntry{Tag: &CondTag{Var: "CritInPast8Sec"}},
	`if you[' ]h?a?ve dealt a crit in the past 8 seconds`:                              &PatternEntry{Tag: &CondTag{Var: "CritInPast8Sec"}},
	`if you[' ]h?a?ve dealt a critical strike in the past 8 seconds`:                   &PatternEntry{Tag: &CondTag{Var: "CritInPast8Sec"}},
	`if you haven't crit recently`:                                                     &PatternEntry{Tag: &CondTag{Var: "CritRecently", Neg: true}},
	`if you haven't dealt a critical strike recently`:                                  &PatternEntry{Tag: &CondTag{Var: "CritRecently", Neg: true}},
	`if you[' ]h?a?ve dealt a non-critical strike recently`:                            &PatternEntry{Tag: &CondTag{Var: "NonCritRecently"}},
	`if your skills have dealt a critical strike recently`:                             &PatternEntry{Tag: &CondTag{Var: "SkillCritRecently"}},
	`if you dealt a critical strike with a herald skill recently`:                      &PatternEntry{Tag: &CondTag{Var: "CritWithHeraldSkillRecently"}},
	`if you[' ]h?a?ve dealt a critical strike with a two handed melee weapon recently`: &PatternEntry{Flags: FlagWeapon2H | FlagWeaponMelee, Tag: &CondTag{Var: "CritRecently"}},
	`if you[' ]h?a?ve killed recently`:                                                 &PatternEntry{Tag: &CondTag{Var: "KilledRecently"}},
	`on killing taunted enemies`:                                                       &PatternEntry{Tag: &CondTag{Var: "KilledTauntedEnemyRecently"}},
	`on kill`:                                                                          &PatternEntry{Tag: &CondTag{Var: "KilledRecently"}},
	`on melee kill`:                                                                    &PatternEntry{Flags: FlagWeaponMelee, Tag: &CondTag{Var: "KilledRecently"}},
	`when you kill an enemy`:                                                           &PatternEntry{Tag: &CondTag{Var: "KilledRecently"}},
	`if you[' ]h?a?ve killed an enemy recently`:                                        &PatternEntry{Tag: &CondTag{Var: "KilledRecently"}},
	`if you[' ]h?a?ve killed at least ([0-9]) enemies recently`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "EnemyKilledRecently", Threshold: opt(c.n(1))}}
	}),
	`if you haven't killed recently`:                                              &PatternEntry{Tag: &CondTag{Var: "KilledRecently", Neg: true}},
	`if you or your totems have killed recently`:                                  &PatternEntry{Tag: &CondTag{VarList: []string{"KilledRecently", "TotemsKilledRecently"}}},
	`if you[' ]h?a?ve thrown a trap or mine recently`:                             &PatternEntry{Tag: &CondTag{Var: "TrapOrMineThrownRecently"}},
	`on throwing a trap`:                                                          &PatternEntry{Tag: &CondTag{Var: "TrapOrMineThrownRecently"}},
	`if you[' ]h?a?ve killed a maimed enemy recently`:                             &PatternEntry{TagList: []Tag{&CondTag{Var: "KilledRecently"}, &CondTag{IsActor: true, Actor: "enemy", Var: "Maimed"}}},
	`if you[' ]h?a?ve killed a cursed enemy recently`:                             &PatternEntry{TagList: []Tag{&CondTag{Var: "KilledRecently"}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}}},
	`if you[' ]h?a?ve killed a bleeding enemy recently`:                           &PatternEntry{TagList: []Tag{&CondTag{Var: "KilledRecently"}, &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}}},
	`if you[' ]h?a?ve killed an enemy affected by your damage over time recently`: &PatternEntry{Tag: &CondTag{Var: "KilledAffectedByDotRecently"}},
	`if you[' ]h?a?ve frozen an enemy recently`:                                   &PatternEntry{Tag: &CondTag{Var: "FrozenEnemyRecently"}},
	`if you[' ]h?a?ve chilled an enemy recently`:                                  &PatternEntry{Tag: &CondTag{Var: "ChilledEnemyRecently"}},
	`if you[' ]h?a?ve ignited an enemy recently`:                                  &PatternEntry{Tag: &CondTag{Var: "IgnitedEnemyRecently"}},
	`if you[' ]h?a?ve shocked an enemy recently`:                                  &PatternEntry{Tag: &CondTag{Var: "ShockedEnemyRecently"}},
	`if you[' ]h?a?ve stunned an enemy recently`:                                  &PatternEntry{Tag: &CondTag{Var: "StunnedEnemyRecently"}},
	`if you[' ]h?a?ve stunned an enemy with a two handed melee weapon recently`:   &PatternEntry{Flags: FlagWeapon2H | FlagWeaponMelee, Tag: &CondTag{Var: "StunnedEnemyRecently"}},
	`if you[' ]h?a?ve been hit recently`:                                          &PatternEntry{Tag: &CondTag{Var: "BeenHitRecently"}},
	`if you[' ]h?a?ve been hit by an attack recently`:                             &PatternEntry{Tag: &CondTag{Var: "BeenHitByAttackRecently"}},
	`if you were hit recently`:                                                    &PatternEntry{Tag: &CondTag{Var: "BeenHitRecently"}},
	`if you were damaged by a hit recently`:                                       &PatternEntry{Tag: &CondTag{Var: "BeenHitRecently"}},
	`if you[' ]h?a?ve taken a critical strike recently`:                           &PatternEntry{Tag: &CondTag{Var: "BeenCritRecently"}},
	`if you[' ]h?a?ve taken a savage hit recently`:                                &PatternEntry{Tag: &CondTag{Var: "BeenSavageHitRecently"}},
	`if you have ?n[o']t been hit recently`:                                       &PatternEntry{Tag: &CondTag{Var: "BeenHitRecently", Neg: true}},
	`if you have ?n[o']t been hit by an attack recently`:                          &PatternEntry{Tag: &CondTag{Var: "BeenHitByAttackRecently", Neg: true}},
	`if you[' ]h?a?ve taken no damage from hits recently`:                         &PatternEntry{Tag: &CondTag{Var: "BeenHitRecently", Neg: true}},
	`if you[' ]h?a?ve taken fire damage from a hit recently`:                      &PatternEntry{Tag: &CondTag{Var: "HitByFireDamageRecently"}},
	`if you[' ]h?a?ve taken fire damage from an enemy hit recently`:               &PatternEntry{Tag: &CondTag{Var: "TakenFireDamageFromEnemyHitRecently"}},
	`if you[' ]h?a?ve taken spell damage recently`:                                &PatternEntry{Tag: &CondTag{Var: "HitBySpellDamageRecently"}},
	`if you haven't taken damage recently`:                                        &PatternEntry{Tag: &CondTag{Var: "BeenHitRecently", Neg: true}},
	`if you[' ]h?a?ve blocked recently`:                                           &PatternEntry{Tag: &CondTag{Var: "BlockedRecently"}},
	`if you haven't blocked recently`:                                             &PatternEntry{Tag: &CondTag{Var: "BlockedRecently", Neg: true}},
	`if you[' ]h?a?ve blocked an attack recently`:                                 &PatternEntry{Tag: &CondTag{Var: "BlockedAttackRecently"}},
	`if you[' ]h?a?ve blocked attack damage recently`:                             &PatternEntry{Tag: &CondTag{Var: "BlockedAttackRecently"}},
	`if you[' ]h?a?ve blocked a spell recently`:                                   &PatternEntry{Tag: &CondTag{Var: "BlockedSpellRecently"}},
	`if you[' ]h?a?ve blocked spell damage recently`:                              &PatternEntry{Tag: &CondTag{Var: "BlockedSpellRecently"}},
	`if you[' ]h?a?ve blocked damage from a unique enemy in the past 10 seconds`:  &PatternEntry{Tag: &CondTag{Var: "BlockedHitFromUniqueEnemyInPast10Sec"}},
	`if you[' ]h?a?ve attacked recently`:                                          &PatternEntry{Tag: &CondTag{Var: "AttackedRecently"}},
	`if you[' ]h?a?ve cast a spell recently`:                                      &PatternEntry{Tag: &CondTag{Var: "CastSpellRecently"}},
	`if you[' ]h?a?ve been stunned while casting recently`:                        &PatternEntry{Tag: &CondTag{Var: "StunnedWhileCastingRecently"}},
	`if you[' ]h?a?ve consumed a corpse recently`:                                 &PatternEntry{Tag: &CondTag{Var: "ConsumedCorpseRecently"}},
	`if you[' ]h?a?ve cursed an enemy recently`:                                   &PatternEntry{Tag: &CondTag{Var: "CursedEnemyRecently"}},
	`if you[' ]h?a?ve cast a mark spell recently`:                                 &PatternEntry{Tag: &CondTag{Var: "CastMarkRecently"}},
	`if you have ?n[o']t consumed a corpse recently`:                              &PatternEntry{Tag: &CondTag{Var: "ConsumedCorpseRecently", Neg: true}},
	`for each corpse consumed recently`:                                           &PatternEntry{Tag: &MultiplierTag{Var: "CorpseConsumedRecently"}},
	`if you[' ]h?a?ve taunted an enemy recently`:                                  &PatternEntry{Tag: &CondTag{Var: "TauntedEnemyRecently"}},
	`if you[' ]h?a?ve used a skill recently`:                                      &PatternEntry{Tag: &CondTag{Var: "UsedSkillRecently"}},
	`if you[' ]h?a?ve used a travel skill recently`:                               &PatternEntry{Tag: &CondTag{Var: "UsedTravelSkillRecently"}},
	`for each skill you've used recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "SkillUsedRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each different non-instant spell you[' ]h?a?ve cast recently`: &PatternEntry{Tag: &MultiplierTag{Var: "NonInstantSpellCastRecently"}},
	`if you[' ]h?a?ve used a warcry recently`:                          &PatternEntry{Tag: &CondTag{Var: "UsedWarcryRecently"}},
	// Archive parity: "when you warcry" appears twice in the
	// reference with the same value (duplicate table key).
	`when you warcry`:                                 &PatternEntry{Tag: &CondTag{Var: "UsedWarcryRecently"}},
	`if you[' ]h?a?ve warcried recently`:              &PatternEntry{Tag: &CondTag{Var: "UsedWarcryRecently"}},
	`if you[' ]h?a?ve not warcried recently`:          &PatternEntry{Tag: &CondTag{Var: "UsedWarcryRecently", Neg: true}},
	`for each time you[' ]h?a?ve warcried recently`:   &PatternEntry{Tag: &MultiplierTag{Var: "WarcryUsedRecently"}},
	`for each warcry exerting them`:                   &PatternEntry{Tag: &MultiplierTag{Var: "ExertingWarcryCount"}},
	`if you[' ]h?a?ve warcried in the past 8 seconds`: &PatternEntry{Tag: &CondTag{Var: "UsedWarcryInPast8Seconds"}},
	`for each second you've been affected by a warcry buff, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "AffectedByWarcryBuffDuration", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each of your mines detonated recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "MineDetonatedRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`[fp][oe]r ?e?a?c?h? mine detonated recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "MineDetonatedRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`[fp][oe]r ?e?a?c?h? mine detonated recently, up to ([0-9]+)% per second`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "MineDetonatedRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`[fp][oe]r ?e?a?c?h? mine detonated recently`: &PatternEntry{Tag: &MultiplierTag{Var: "MineDetonatedRecently"}},
	`for each of your traps triggered recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "TrapTriggeredRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each trap triggered recently, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "TrapTriggeredRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each trap triggered recently, up to ([0-9]+)% per second`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "TrapTriggeredRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`if you[' ]h?a?ve used a fire skill recently`:                    &PatternEntry{Tag: &CondTag{Var: "UsedFireSkillRecently"}},
	`if you[' ]h?a?ve used a cold skill recently`:                    &PatternEntry{Tag: &CondTag{Var: "UsedColdSkillRecently"}},
	`if you[' ]h?a?ve used a fire skill in the past 10 seconds`:      &PatternEntry{Tag: &CondTag{Var: "UsedFireSkillInPast10Sec"}},
	`if you[' ]h?a?ve used a cold skill in the past 10 seconds`:      &PatternEntry{Tag: &CondTag{Var: "UsedColdSkillInPast10Sec"}},
	`if you[' ]h?a?ve used a lightning skill in the past 10 seconds`: &PatternEntry{Tag: &CondTag{Var: "UsedLightningSkillInPast10Sec"}},
	`if you[' ]h?a?ve summoned a totem recently`:                     &PatternEntry{Tag: &CondTag{Var: "SummonedTotemRecently"}},
	`when you summon a totem`:                                        &PatternEntry{Tag: &CondTag{Var: "SummonedTotemRecently"}},
	`if you summoned a golem in the past 8 seconds`:                  &PatternEntry{Tag: &CondTag{Var: "SummonedGolemInPast8Sec"}},
	`if you haven't summoned a totem in the past 2 seconds`:          &PatternEntry{Tag: &CondTag{Var: "NoSummonedTotemsInPastTwoSeconds"}},
	`if you[' ]h?a?ve used a minion skill recently`:                  &PatternEntry{Tag: &CondTag{Var: "UsedMinionSkillRecently"}},
	`if you[' ]h?a?ve used a movement skill recently`:                &PatternEntry{Tag: &CondTag{Var: "UsedMovementSkillRecently"}},
	`when you use a movement skill`:                                  &PatternEntry{Tag: &CondTag{Var: "UsedMovementSkillRecently"}},
	`if you haven't cast dash recently`:                              &PatternEntry{Tag: &CondTag{Var: "CastDashRecently", Neg: true}},
	`if you[' ]h?a?ve cast dash recently`:                            &PatternEntry{Tag: &CondTag{Var: "CastDashRecently"}},
	`if you[' ]h?a?ve used a vaal skill recently`:                    &PatternEntry{Tag: &CondTag{Var: "UsedVaalSkillRecently"}},
	`if you[' ]h?a?ve used a socketed vaal skill recently`:           &PatternEntry{Tag: &CondTag{Var: "UsedVaalSkillRecently"}},
	`when you use a vaal skill`:                                      &PatternEntry{Tag: &CondTag{Var: "UsedVaalSkillRecently"}},
	`if you haven't used a brand skill recently`:                     &PatternEntry{Tag: &CondTag{Var: "UsedBrandRecently", Neg: true}},
	`if you[' ]h?a?ve used a brand skill recently`:                   &PatternEntry{Tag: &CondTag{Var: "UsedBrandRecently"}},
	`if you[' ]h?a?ve used a retaliation skill recently`:             &PatternEntry{Tag: &CondTag{Var: "UsedRetaliationRecently"}},
	`if you[' ]h?a?ve spent ([0-9]+) total mana recently`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "ManaSpentRecently", Threshold: opt(c.n(1))}}
	}),
	`if you[' ]h?a?ve spent life recently`: &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "LifeSpentRecently", Threshold: opt(1)}},
	`for [0-9]+ seconds after spending a total of ([0-9]+) mana`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "ManaSpentRecently", Threshold: opt(c.n(1))}}
	}),
	`if you've impaled an enemy recently`:                         &PatternEntry{Tag: &CondTag{Var: "ImpaledRecently"}},
	`if you've changed stance recently`:                           &PatternEntry{Tag: &CondTag{Var: "ChangedStanceRecently"}},
	`if you've gained a power charge recently`:                    &PatternEntry{Tag: &CondTag{Var: "GainedPowerChargeRecently"}},
	`if you haven't gained a power charge recently`:               &PatternEntry{Tag: &CondTag{Var: "GainedPowerChargeRecently", Neg: true}},
	`if you haven't gained a frenzy charge recently`:              &PatternEntry{Tag: &CondTag{Var: "GainedFrenzyChargeRecently", Neg: true}},
	`if you've stopped taking damage over time recently`:          &PatternEntry{Tag: &CondTag{Var: "StoppedTakingDamageOverTimeRecently"}},
	`if you've used an amethyst flask recently`:                   &PatternEntry{Tag: &CondTag{Var: "UsedAmethystFlaskRecently"}},
	`if you've used a ruby flask recently`:                        &PatternEntry{Tag: &CondTag{Var: "UsedRubyFlaskRecently"}},
	`if you've used a sapphire flask recently`:                    &PatternEntry{Tag: &CondTag{Var: "UsedSapphireFlaskRecently"}},
	`if you've used a topaz flask recently`:                       &PatternEntry{Tag: &CondTag{Var: "UsedTopazFlaskRecently"}},
	`during soul gain prevention`:                                 &PatternEntry{Tag: &CondTag{Var: "SoulGainPrevention"}},
	`if you detonated mines recently`:                             &PatternEntry{Tag: &CondTag{Var: "DetonatedMinesRecently"}},
	`if you detonated a mine recently`:                            &PatternEntry{Tag: &CondTag{Var: "DetonatedMinesRecently"}},
	`if you[' ]h?a?ve detonated a mine recently`:                  &PatternEntry{Tag: &CondTag{Var: "DetonatedMinesRecently"}},
	`when your mine is detonated targeting an enemy`:              &PatternEntry{Tag: &CondTag{Var: "DetonatedMinesRecently"}},
	`when your trap is triggered by an enemy`:                     &PatternEntry{Tag: &CondTag{Var: "TriggeredTrapsRecently"}},
	`if energy shield recharge has started recently`:              &PatternEntry{Tag: &CondTag{Var: "EnergyShieldRechargeRecently"}},
	`if energy shield recharge has started in the past 2 seconds`: &PatternEntry{Tag: &CondTag{Var: "EnergyShieldRechargePastTwoSec"}},
	`when cast on frostbolt`:                                      &PatternEntry{Tag: &CondTag{Var: "CastOnFrostbolt"}},
	`branded enemy's`:                                             &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "BrandsAttachedToEnemy", Threshold: opt(1)}},
	`to enemies they're attached to`:                              &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "BrandsAttachedToEnemy", Threshold: opt(1)}},
	`for each hit you've taken recently up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "BeenHitRecently", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`per enemy hit taken recently`: &PatternEntry{Tag: &MultiplierTag{Var: "BeenHitRecently"}},
	`for each nearby enemy, up to ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "NearbyEnemies", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each nearby enemy, up to a maximum of ([0-9]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Var: "NearbyEnemies", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`while you have iron reflexes`:             &PatternEntry{Tag: &CondTag{Var: "HaveIronReflexes"}},
	`while you do not have iron reflexes`:      &PatternEntry{Tag: &CondTag{Var: "HaveIronReflexes", Neg: true}},
	`while you have elemental overload`:        &PatternEntry{Tag: &CondTag{Var: "HaveElementalOverload"}},
	`while you do not have elemental overload`: &PatternEntry{Tag: &CondTag{Var: "HaveElementalOverload", Neg: true}},
	`while you have resolute technique`:        &PatternEntry{Tag: &CondTag{Var: "HaveResoluteTechnique"}},
	`while you do not have resolute technique`: &PatternEntry{Tag: &CondTag{Var: "HaveResoluteTechnique", Neg: true}},
	`while you have avatar of fire`:            &PatternEntry{Tag: &CondTag{Var: "HaveAvatarOfFire"}},
	`while you do not have avatar of fire`:     &PatternEntry{Tag: &CondTag{Var: "HaveAvatarOfFire", Neg: true}},
	`if you have a summoned golem`:             &PatternEntry{Tag: &CondTag{VarList: []string{"HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem"}}},
	`while you have a summoned golem`:          &PatternEntry{Tag: &CondTag{VarList: []string{"HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem"}}},
	`if a minion has died recently`:            &PatternEntry{Tag: &CondTag{Var: "MinionsDiedRecently"}},
	`if a minion has been killed recently`:     &PatternEntry{Tag: &CondTag{Var: "MinionsDiedRecently"}},
	`while you have sacrificial zeal`:          &PatternEntry{Tag: &CondTag{Var: "SacrificialZeal"}},
	`while sane`:                               &PatternEntry{Tag: &CondTag{Var: "Insane", Neg: true}},
	`while insane`:                             &PatternEntry{Tag: &CondTag{Var: "Insane"}},
	`while you have defiance`:                  &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Var: "Defiance", Threshold: opt(1)}},
	`while affected by glorious madness`:       &PatternEntry{Tag: &CondTag{Var: "AffectedByGloriousMadness"}},
	`if you've shattered an enemy recently`:    &PatternEntry{Tag: &CondTag{Var: "ShatteredEnemyRecently"}},
	`while affected by no flasks?`:             &PatternEntry{Tag: &CondTag{Var: "UsingFlask", Neg: true}},
	`while affected by flasks?`:                &PatternEntry{Tag: &CondTag{Var: "UsingFlask"}},
	// Enemy status conditions
	`at close range`:                               &PatternEntry{Tag: &CondTag{Var: "AtCloseRange"}},
	`not at close range`:                           &PatternEntry{Tag: &CondTag{Var: "AtCloseRange", Neg: true}},
	`against rare and unique enemies`:              &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"}},
	`by s?l?a?i?n? rare [ao][nr]d? unique enemies`: &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"}},
	`against unique enemies`:                       &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "RareOrUnique"}},
	`against enemies on full life`:                 &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "FullLife"}},
	`against enemies that are on full life`:        &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "FullLife"}},
	`against enemies on low life`:                  &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "LowLife"}},
	`against enemies that are on low life`:         &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "LowLife"}},
	`against enemies that are not on low life`:     &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "LowLife", Neg: true}},
	`to enemies which have energy shield`:          &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "HaveEnergyShield"}, KeywordFlags: KeywordHit | KeywordAilment},
	`against cursed enemies`:                       &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}},
	`against stunned enemies`:                      &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Stunned"}},
	`on cursed enemies`:                            &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}},
	`of cursed enemies'`:                           &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}},
	`when hitting cursed enemies`:                  &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}, KeywordFlags: KeywordHit},
	`from cursed enemies`:                          &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"}},
	`against marked enemy`:                         &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"}},
	`when hitting marked enemy`:                    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"}, KeywordFlags: KeywordHit},
	`from marked enemy`:                            &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Marked"}},
	`against taunted enemies`:                      &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Taunted"}},
	`against bleeding enemies`:                     &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}},
	`you inflict on bleeding enemies`:              &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}},
	`to bleeding enemies`:                          &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}},
	`from bleeding enemies`:                        &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"}},
	`against poisoned enemies`:                     &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"}},
	`you inflict on poisoned enemies`:              &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"}},
	`to poisoned enemies`:                          &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Poisoned"}},
	`against enemies affected by ([0-9]+) or more poisons`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "PoisonStack", Threshold: opt(c.n(1))}}
	}),
	`against enemies affected by at least ([0-9]+) poisons`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "PoisonStack", Threshold: opt(c.n(1))}}
	}),
	`against hindered enemies`:                                   &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Hindered"}},
	`against maimed enemies`:                                     &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Maimed"}},
	`you inflict on maimed enemies`:                              &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Maimed"}},
	`against blinded enemies`:                                    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Blinded"}},
	`against excommunicated enemies`:                             &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Excommunicated"}},
	`from blinded enemies`:                                       &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Blinded"}},
	`against burning enemies`:                                    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Burning"}},
	`against ignited enemies`:                                    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"}},
	`to ignited enemies`:                                         &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"}},
	`against shocked enemies`:                                    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}},
	`you inflict on shocked enemies`:                             &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}},
	`to shocked enemies`:                                         &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}},
	`inflicted on shocked enemies`:                               &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}},
	`enemies which are shocked`:                                  &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}},
	`against frozen enemies`:                                     &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"}},
	`to frozen enemies`:                                          &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"}},
	`against chilled enemies`:                                    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}},
	`you inflict on chilled enemies`:                             &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}},
	`to chilled enemies`:                                         &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}},
	`inflicted on chilled enemies`:                               &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}},
	`enemies which are chilled`:                                  &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Chilled"}},
	`against chilled or frozen enemies`:                          &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Chilled", "Frozen"}}},
	`against frozen, shocked or ignited enemies`:                 &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Frozen", "Shocked", "Ignited"}}},
	`against enemies affected by elemental ailments`:             &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}}},
	`against enemies affected by ailments`:                       &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped", "Poisoned", "Bleeding"}}},
	`against enemies that are affected by elemental ailments`:    &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}}},
	`against enemies that are affected by no elemental ailments`: &PatternEntry{TagList: []Tag{&CondTag{IsActor: true, Actor: "enemy", VarList: []string{"Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"}, Neg: true}, &CondTag{Var: "Effective"}}},
	`against enemies affected by ([0-9]+) spider's webs`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "Spider's WebStack", Threshold: opt(c.n(1))}}
	}),
	`against enemies on consecrated ground`:                                     &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "OnConsecratedGround"}},
	`against enemies with a higher percentage of their life remaining than you`: &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "HigherLifePercentThanPlayer"}},
	`if ([0-9]+)% of curse duration expired`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{IsThreshold: true, Actor: "enemy", Var: "CurseExpired", Threshold: opt(c.n(1))}}
	}),
	`against enemies with ([0-9a-zA-Z]+) exposure`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Has" + (firstToUpper(c.s(1)) + "Exposure")}}
	}),
	`by s?l?a?i?n? ?frozen enemies`:  &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Frozen"}},
	`by s?l?a?i?n? ?shocked enemies`: &PatternEntry{Tag: &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"}},
	// Enemy multipliers
	`per freeze, shock [ao][nr]d? ignite on enemy`: &PatternEntry{Tag: &MultiplierTag{Var: "FreezeShockIgniteOnEnemy"}},
	`per poison affecting enemy`:                   &PatternEntry{Tag: &MultiplierTag{Actor: "enemy", Var: "PoisonStack"}},
	`per poison affecting enemy, up to \+([0-9.]+)%`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &MultiplierTag{Actor: "enemy", Var: "PoisonStack", Limit: opt(c.n(1)), LimitTotal: true}}
	}),
	`for each spider's web on the enemy`: &PatternEntry{Tag: &MultiplierTag{Actor: "enemy", Var: "Spider's WebStack"}},
	// Hand-ported entries the transform could not express — ModParser.lua:1595,1631-1632,1650-1658,1739-1746,1810-1812.
	`if you have a ([a-zA-Z]+) ([a-zA-Z]+) in ([a-zA-Z]+) slot`: entryFn(func(c caps) *PatternEntry {
		slotIndex := ""
		switch c.s(3) {
		case "right":
			slotIndex = "2"
		case "left":
			slotIndex = "1"
		}
		return &PatternEntry{Tag: &CondTag{Var: firstToUpper(c.s(1)) + "ItemIn" + firstToUpper(c.s(2)) + " " + slotIndex}}
	}),
	`while holding a ([0-9a-zA-Z]+)`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &CondTag{VarList: []string{"Using" + firstToUpper(c.s(1))}}}
	}),
	`while holding a ([0-9a-zA-Z]+) or ([0-9a-zA-Z]+)`: entryFn(func(c caps) *PatternEntry {
		return &PatternEntry{Tag: &CondTag{VarList: []string{"Using" + firstToUpper(c.s(1)), "Using" + firstToUpper(c.s(2))}}}
	}),
	// itemSlotName:sub(1, #itemSlotName - 1) drops the plural 's'.
	`if both equipped ([a-zA-Z \t\n\v\f\r]+) have a?n? ?([a-zA-Z \t\n\v\f\r]+) modifiers?`: entryFn(func(c caps) *PatternEntry {
		slot := c.s(1)
		if len(slot) > 0 {
			slot = slot[:len(slot)-1]
		}
		return &PatternEntry{Tag: &ItemCondTag{SearchCond: c.s(2), ItemSlot: slot, BothSlots: true}}
	}),
	`if both equipped left and right ([a-zA-Z \t\n\v\f\r]+) have a?n? ?([a-zA-Z \t\n\v\f\r]+) modifiers?`: entryFn(func(c caps) *PatternEntry {
		slot := c.s(1)
		if len(slot) > 0 {
			slot = slot[:len(slot)-1]
		}
		return &PatternEntry{Tag: &ItemCondTag{SearchCond: c.s(2), ItemSlot: slot, BothSlots: true}}
	}),
	`if equipped helmet, body armour, gloves, and boots all have armour`:         &PatternEntry{TagList: []Tag{&StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnHelmet", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnBody Armour", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnGloves", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "ArmourOnBoots", Threshold: opt(1)}}},
	`if equipped helmet, body armour, gloves, and boots all have evasion rating`: &PatternEntry{TagList: []Tag{&StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnHelmet", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnBody Armour", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnGloves", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "EvasionOnBoots", Threshold: opt(1)}}},
	`if you have reserved life and mana`:                                         &PatternEntry{TagList: []Tag{&StatTag{StatKind: TagStatThreshold, Stat: "LifeReserved", Threshold: opt(1)}, &StatTag{StatKind: TagStatThreshold, Stat: "ManaReserved", Threshold: opt(1)}}},
}
