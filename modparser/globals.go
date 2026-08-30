package modparser

// Transcribed from src/Data/Global.lua: the modifier bit flags and the
// active skill type ids, as named types so the two flag sets cannot be
// mixed by accident.

// ModFlag classifies which damage calculations a modifier participates in.
type ModFlag uint64

const (
	FlagNone ModFlag = 0

	// Damage modes
	FlagAttack ModFlag = 0x00000001
	FlagSpell  ModFlag = 0x00000002
	FlagHit    ModFlag = 0x00000004
	FlagDot    ModFlag = 0x00000008
	FlagCast   ModFlag = 0x00000010

	// Damage sources
	FlagMelee      ModFlag = 0x00000100
	FlagArea       ModFlag = 0x00000200
	FlagProjectile ModFlag = 0x00000400
	FlagSourceMask ModFlag = 0x00000600
	FlagAilment    ModFlag = 0x00000800
	FlagMeleeHit   ModFlag = 0x00001000
	FlagWeapon     ModFlag = 0x00002000

	// Weapon types
	FlagAxe     ModFlag = 0x00010000
	FlagBow     ModFlag = 0x00020000
	FlagClaw    ModFlag = 0x00040000
	FlagDagger  ModFlag = 0x00080000
	FlagMace    ModFlag = 0x00100000
	FlagStaff   ModFlag = 0x00200000
	FlagSword   ModFlag = 0x00400000
	FlagWand    ModFlag = 0x00800000
	FlagUnarmed ModFlag = 0x01000000
	FlagFishing ModFlag = 0x02000000

	// Weapon classes
	FlagWeaponMelee  ModFlag = 0x04000000
	FlagWeaponRanged ModFlag = 0x08000000
	FlagWeapon1H     ModFlag = 0x10000000
	FlagWeapon2H     ModFlag = 0x20000000
	FlagWeaponMask   ModFlag = 0x2FFF0000
)

// KeywordFlag classifies modifiers by skill keyword.
type KeywordFlag uint64

const (
	KeywordNone KeywordFlag = 0

	// Skill keywords
	KeywordAura      KeywordFlag = 0x00000001
	KeywordCurse     KeywordFlag = 0x00000002
	KeywordWarcry    KeywordFlag = 0x00000004
	KeywordMovement  KeywordFlag = 0x00000008
	KeywordPhysical  KeywordFlag = 0x00000010
	KeywordFire      KeywordFlag = 0x00000020
	KeywordCold      KeywordFlag = 0x00000040
	KeywordLightning KeywordFlag = 0x00000080
	KeywordChaos     KeywordFlag = 0x00000100
	KeywordVaal      KeywordFlag = 0x00000200
	KeywordBow       KeywordFlag = 0x00000400
	KeywordArrow     KeywordFlag = 0x00000800

	// Skill types
	KeywordTrap    KeywordFlag = 0x00001000
	KeywordMine    KeywordFlag = 0x00002000
	KeywordTotem   KeywordFlag = 0x00004000
	KeywordMinion  KeywordFlag = 0x00008000
	KeywordAttack  KeywordFlag = 0x00010000
	KeywordSpell   KeywordFlag = 0x00020000
	KeywordHit     KeywordFlag = 0x00040000
	KeywordAilment KeywordFlag = 0x00080000
	KeywordBrand   KeywordFlag = 0x00100000

	// Other effects
	KeywordPoison KeywordFlag = 0x00200000
	KeywordBleed  KeywordFlag = 0x00400000
	KeywordIgnite KeywordFlag = 0x00800000

	// Damage over Time types
	KeywordPhysicalDot  KeywordFlag = 0x01000000
	KeywordLightningDot KeywordFlag = 0x02000000
	KeywordColdDot      KeywordFlag = 0x04000000
	KeywordFireDot      KeywordFlag = 0x08000000
	KeywordChaosDot     KeywordFlag = 0x10000000

	// Match *all* flags instead of any
	KeywordMatchAll KeywordFlag = 0x40000000
)

// SkillTypeID is an active skill type from ActiveSkillType.dat.
type SkillTypeID int64

const (
	SkillTypeAttack                               SkillTypeID = 1
	SkillTypeSpell                                SkillTypeID = 2
	SkillTypeProjectile                           SkillTypeID = 3
	SkillTypeDualWieldOnly                        SkillTypeID = 4
	SkillTypeBuff                                 SkillTypeID = 5
	SkillTypeMinion                               SkillTypeID = 6
	SkillTypeDamage                               SkillTypeID = 7
	SkillTypeArea                                 SkillTypeID = 8
	SkillTypeDuration                             SkillTypeID = 9
	SkillTypeRequiresShield                       SkillTypeID = 10
	SkillTypeProjectileSpeed                      SkillTypeID = 11
	SkillTypeHasReservation                       SkillTypeID = 12
	SkillTypeReservationBecomesCost               SkillTypeID = 13
	SkillTypeTrappable                            SkillTypeID = 14
	SkillTypeTotemable                            SkillTypeID = 15
	SkillTypeMineable                             SkillTypeID = 16
	SkillTypeElementalStatus                      SkillTypeID = 17
	SkillTypeMinionsCanExplode                    SkillTypeID = 18
	SkillTypeChains                               SkillTypeID = 19
	SkillTypeMelee                                SkillTypeID = 20
	SkillTypeMeleeSingleTarget                    SkillTypeID = 21
	SkillTypeMulticastable                        SkillTypeID = 22
	SkillTypeTotemCastsAlone                      SkillTypeID = 23
	SkillTypeMultistrikeable                      SkillTypeID = 24
	SkillTypeCausesBurning                        SkillTypeID = 25
	SkillTypeSummonsTotem                         SkillTypeID = 26
	SkillTypeTotemCastsWhenNotDetached            SkillTypeID = 27
	SkillTypePhysical                             SkillTypeID = 28
	SkillTypeFire                                 SkillTypeID = 29
	SkillTypeCold                                 SkillTypeID = 30
	SkillTypeLightning                            SkillTypeID = 31
	SkillTypeTriggerable                          SkillTypeID = 32
	SkillTypeTrapped                              SkillTypeID = 33
	SkillTypeMovement                             SkillTypeID = 34
	SkillTypeDamageOverTime                       SkillTypeID = 35
	SkillTypeRemoteMined                          SkillTypeID = 36
	SkillTypeTriggered                            SkillTypeID = 37
	SkillTypeVaal                                 SkillTypeID = 38
	SkillTypeAura                                 SkillTypeID = 39
	SkillTypeCanTargetUnusableCorpse              SkillTypeID = 40
	SkillTypeRangedAttack                         SkillTypeID = 41
	SkillTypeChaos                                SkillTypeID = 42
	SkillTypeFixedSpeedProjectile                 SkillTypeID = 43
	SkillTypeThresholdJewelArea                   SkillTypeID = 44
	SkillTypeThresholdJewelProjectile             SkillTypeID = 45
	SkillTypeThresholdJewelDuration               SkillTypeID = 46
	SkillTypeThresholdJewelRangedAttack           SkillTypeID = 47
	SkillTypeChannel                              SkillTypeID = 48
	SkillTypeDegenOnlySpellDamage                 SkillTypeID = 49
	SkillTypeInbuiltTrigger                       SkillTypeID = 50
	SkillTypeGolem                                SkillTypeID = 51
	SkillTypeHerald                               SkillTypeID = 52
	SkillTypeAuraAffectsEnemies                   SkillTypeID = 53
	SkillTypeNoRuthless                           SkillTypeID = 54
	SkillTypeThresholdJewelSpellDamage            SkillTypeID = 55
	SkillTypeCascadable                           SkillTypeID = 56
	SkillTypeProjectilesFromUser                  SkillTypeID = 57
	SkillTypeMirageArcherCanUse                   SkillTypeID = 58
	SkillTypeProjectileSpiral                     SkillTypeID = 59
	SkillTypeSingleMainProjectile                 SkillTypeID = 60
	SkillTypeMinionsPersistWhenSkillRemoved       SkillTypeID = 61
	SkillTypeProjectileNumber                     SkillTypeID = 62
	SkillTypeWarcry                               SkillTypeID = 63
	SkillTypeInstant                              SkillTypeID = 64
	SkillTypeBrand                                SkillTypeID = 65
	SkillTypeDestroysCorpse                       SkillTypeID = 66
	SkillTypeNonHitChill                          SkillTypeID = 67
	SkillTypeChillingArea                         SkillTypeID = 68
	SkillTypeAppliesCurse                         SkillTypeID = 69
	SkillTypeCanRapidFire                         SkillTypeID = 70
	SkillTypeAuraDuration                         SkillTypeID = 71
	SkillTypeAreaSpell                            SkillTypeID = 72
	SkillTypeOR                                   SkillTypeID = 73
	SkillTypeAND                                  SkillTypeID = 74
	SkillTypeNOT                                  SkillTypeID = 75
	SkillTypeAppliesMaim                          SkillTypeID = 76
	SkillTypeCreatesMinion                        SkillTypeID = 77
	SkillTypeGuard                                SkillTypeID = 78
	SkillTypeTravel                               SkillTypeID = 79
	SkillTypeBlink                                SkillTypeID = 80
	SkillTypeCanHaveBlessing                      SkillTypeID = 81
	SkillTypeProjectilesNotFromUser               SkillTypeID = 82
	SkillTypeAttackInPlaceIsDefault               SkillTypeID = 83
	SkillTypeNova                                 SkillTypeID = 84
	SkillTypeInstantNoRepeatWhenHeld              SkillTypeID = 85
	SkillTypeInstantShiftAttackForLeftMouse       SkillTypeID = 86
	SkillTypeAuraNotOnCaster                      SkillTypeID = 87
	SkillTypeBanner                               SkillTypeID = 88
	SkillTypeRain                                 SkillTypeID = 89
	SkillTypeCooldown                             SkillTypeID = 90
	SkillTypeThresholdJewelChaining               SkillTypeID = 91
	SkillTypeSlam                                 SkillTypeID = 92
	SkillTypeStance                               SkillTypeID = 93
	SkillTypeNonRepeatable                        SkillTypeID = 94
	SkillTypeOtherThingUsesSkill                  SkillTypeID = 95
	SkillTypeSteel                                SkillTypeID = 96
	SkillTypeHex                                  SkillTypeID = 97
	SkillTypeMark                                 SkillTypeID = 98
	SkillTypeAegis                                SkillTypeID = 99
	SkillTypeOrb                                  SkillTypeID = 100
	SkillTypeKillNoDamageModifiers                SkillTypeID = 101
	SkillTypeRandomElement                        SkillTypeID = 102
	SkillTypeLateConsumeCooldown                  SkillTypeID = 103
	SkillTypeArcane                               SkillTypeID = 104
	SkillTypeFixedCastTime                        SkillTypeID = 105
	SkillTypeRequiresOffHandNotWeapon             SkillTypeID = 106
	SkillTypeLink                                 SkillTypeID = 107
	SkillTypeBlessing                             SkillTypeID = 108
	SkillTypeZeroReservation                      SkillTypeID = 109
	SkillTypeDynamicCooldown                      SkillTypeID = 110
	SkillTypeMicrotransaction                     SkillTypeID = 111
	SkillTypeOwnerCannotUse                       SkillTypeID = 112
	SkillTypeProjectilesNumberModifiersNotApplied SkillTypeID = 113
	SkillTypeTotemsAreBallistae                   SkillTypeID = 114
	SkillTypeSkillGrantedBySupport                SkillTypeID = 115
	SkillTypePreventHexTransfer                   SkillTypeID = 116
	SkillTypeMinionsAreUndamagable                SkillTypeID = 117
	SkillTypeInnateTrauma                         SkillTypeID = 118
	SkillTypeDualWieldRequiresDifferentTypes      SkillTypeID = 119
	SkillTypeNoVolley                             SkillTypeID = 120
	SkillTypeRetaliation                          SkillTypeID = 121
	SkillTypeNeverExertable                       SkillTypeID = 122
	SkillTypeDisallowTriggerSupports              SkillTypeID = 123
	SkillTypeProjectileCannotReturn               SkillTypeID = 124
	SkillTypeOffering                             SkillTypeID = 125
	SkillTypeSupportedByBane                      SkillTypeID = 126
	SkillTypeWandAttack                           SkillTypeID = 127
	SkillTypeGainsIntensity                       SkillTypeID = 128
	SkillTypeCreatesSentinelMinion                SkillTypeID = 129
	SkillTypeSupportedByAutoExertion              SkillTypeID = 130
	SkillTypeSupportedByCrabTotem                 SkillTypeID = 131
	SkillTypeSupportedBySpellTotem                SkillTypeID = 132
	SkillTypeCreatesCorpse                        SkillTypeID = 133
	SkillTypeRequiresStaff                        SkillTypeID = 134
	SkillTypePact                                 SkillTypeID = 135
)

// ModFlagByName is ModFlag[name]: the flag under its Global.lua key.
var ModFlagByName = map[string]ModFlag{
	"Attack": FlagAttack, "Spell": FlagSpell, "Hit": FlagHit, "Dot": FlagDot, "Cast": FlagCast,
	"Melee": FlagMelee, "Area": FlagArea, "Projectile": FlagProjectile, "SourceMask": FlagSourceMask,
	"Ailment": FlagAilment, "MeleeHit": FlagMeleeHit, "Weapon": FlagWeapon,
	"Axe": FlagAxe, "Bow": FlagBow, "Claw": FlagClaw, "Dagger": FlagDagger, "Mace": FlagMace,
	"Staff": FlagStaff, "Sword": FlagSword, "Wand": FlagWand, "Unarmed": FlagUnarmed, "Fishing": FlagFishing,
	"WeaponMelee": FlagWeaponMelee, "WeaponRanged": FlagWeaponRanged, "Weapon1H": FlagWeapon1H,
	"Weapon2H": FlagWeapon2H, "WeaponMask": FlagWeaponMask,
}

// KeywordFlagByName is KeywordFlag[name].
var KeywordFlagByName = map[string]KeywordFlag{
	"Aura": KeywordAura, "Curse": KeywordCurse, "Warcry": KeywordWarcry, "Movement": KeywordMovement,
	"Physical": KeywordPhysical, "Fire": KeywordFire, "Cold": KeywordCold, "Lightning": KeywordLightning,
	"Chaos": KeywordChaos, "Vaal": KeywordVaal, "Bow": KeywordBow, "Arrow": KeywordArrow,
	"Trap": KeywordTrap, "Mine": KeywordMine, "Totem": KeywordTotem, "Minion": KeywordMinion,
	"Attack": KeywordAttack, "Spell": KeywordSpell, "Hit": KeywordHit, "Ailment": KeywordAilment,
	"Brand": KeywordBrand, "Poison": KeywordPoison, "Bleed": KeywordBleed, "Ignite": KeywordIgnite,
	"PhysicalDot": KeywordPhysicalDot, "LightningDot": KeywordLightningDot, "ColdDot": KeywordColdDot,
	"FireDot": KeywordFireDot, "ChaosDot": KeywordChaosDot, "MatchAll": KeywordMatchAll,
}

// SkillTypeByName is SkillType[name].
var SkillTypeByName = map[string]SkillTypeID{
	"Attack": SkillTypeAttack, "Spell": SkillTypeSpell, "Projectile": SkillTypeProjectile,
	"DualWieldOnly": SkillTypeDualWieldOnly, "Buff": SkillTypeBuff, "Minion": SkillTypeMinion,
	"Damage": SkillTypeDamage, "Area": SkillTypeArea, "Duration": SkillTypeDuration,
	"RequiresShield": SkillTypeRequiresShield, "ProjectileSpeed": SkillTypeProjectileSpeed,
	"HasReservation": SkillTypeHasReservation, "ReservationBecomesCost": SkillTypeReservationBecomesCost,
	"Trappable": SkillTypeTrappable, "Totemable": SkillTypeTotemable, "Mineable": SkillTypeMineable,
	"ElementalStatus": SkillTypeElementalStatus, "MinionsCanExplode": SkillTypeMinionsCanExplode,
	"Chains": SkillTypeChains, "Melee": SkillTypeMelee, "MeleeSingleTarget": SkillTypeMeleeSingleTarget,
	"Multicastable": SkillTypeMulticastable, "TotemCastsAlone": SkillTypeTotemCastsAlone,
	"Multistrikeable": SkillTypeMultistrikeable, "CausesBurning": SkillTypeCausesBurning,
	"SummonsTotem": SkillTypeSummonsTotem, "TotemCastsWhenNotDetached": SkillTypeTotemCastsWhenNotDetached,
	"Physical": SkillTypePhysical, "Fire": SkillTypeFire, "Cold": SkillTypeCold, "Lightning": SkillTypeLightning,
	"Triggerable": SkillTypeTriggerable, "Trapped": SkillTypeTrapped, "Movement": SkillTypeMovement,
	"DamageOverTime": SkillTypeDamageOverTime, "RemoteMined": SkillTypeRemoteMined, "Triggered": SkillTypeTriggered,
	"Vaal": SkillTypeVaal, "Aura": SkillTypeAura, "CanTargetUnusableCorpse": SkillTypeCanTargetUnusableCorpse,
	"RangedAttack": SkillTypeRangedAttack, "Chaos": SkillTypeChaos, "FixedSpeedProjectile": SkillTypeFixedSpeedProjectile,
	"ThresholdJewelArea": SkillTypeThresholdJewelArea, "ThresholdJewelProjectile": SkillTypeThresholdJewelProjectile,
	"ThresholdJewelDuration": SkillTypeThresholdJewelDuration, "ThresholdJewelRangedAttack": SkillTypeThresholdJewelRangedAttack,
	"Channel": SkillTypeChannel, "DegenOnlySpellDamage": SkillTypeDegenOnlySpellDamage, "InbuiltTrigger": SkillTypeInbuiltTrigger,
	"Golem": SkillTypeGolem, "Herald": SkillTypeHerald, "AuraAffectsEnemies": SkillTypeAuraAffectsEnemies,
	"NoRuthless": SkillTypeNoRuthless, "ThresholdJewelSpellDamage": SkillTypeThresholdJewelSpellDamage,
	"Cascadable": SkillTypeCascadable, "ProjectilesFromUser": SkillTypeProjectilesFromUser,
	"MirageArcherCanUse": SkillTypeMirageArcherCanUse, "ProjectileSpiral": SkillTypeProjectileSpiral,
	"SingleMainProjectile": SkillTypeSingleMainProjectile, "MinionsPersistWhenSkillRemoved": SkillTypeMinionsPersistWhenSkillRemoved,
	"ProjectileNumber": SkillTypeProjectileNumber, "Warcry": SkillTypeWarcry, "Instant": SkillTypeInstant,
	"Brand": SkillTypeBrand, "DestroysCorpse": SkillTypeDestroysCorpse, "NonHitChill": SkillTypeNonHitChill,
	"ChillingArea": SkillTypeChillingArea, "AppliesCurse": SkillTypeAppliesCurse, "CanRapidFire": SkillTypeCanRapidFire,
	"AuraDuration": SkillTypeAuraDuration, "AreaSpell": SkillTypeAreaSpell, "OR": SkillTypeOR, "AND": SkillTypeAND,
	"NOT": SkillTypeNOT, "AppliesMaim": SkillTypeAppliesMaim, "CreatesMinion": SkillTypeCreatesMinion,
	"Guard": SkillTypeGuard, "Travel": SkillTypeTravel, "Blink": SkillTypeBlink, "CanHaveBlessing": SkillTypeCanHaveBlessing,
	"ProjectilesNotFromUser": SkillTypeProjectilesNotFromUser, "AttackInPlaceIsDefault": SkillTypeAttackInPlaceIsDefault,
	"Nova": SkillTypeNova, "InstantNoRepeatWhenHeld": SkillTypeInstantNoRepeatWhenHeld,
	"InstantShiftAttackForLeftMouse": SkillTypeInstantShiftAttackForLeftMouse, "AuraNotOnCaster": SkillTypeAuraNotOnCaster,
	"Banner": SkillTypeBanner, "Rain": SkillTypeRain, "Cooldown": SkillTypeCooldown,
	"ThresholdJewelChaining": SkillTypeThresholdJewelChaining, "Slam": SkillTypeSlam, "Stance": SkillTypeStance,
	"NonRepeatable": SkillTypeNonRepeatable, "OtherThingUsesSkill": SkillTypeOtherThingUsesSkill, "Steel": SkillTypeSteel,
	"Hex": SkillTypeHex, "Mark": SkillTypeMark, "Aegis": SkillTypeAegis, "Orb": SkillTypeOrb,
	"KillNoDamageModifiers": SkillTypeKillNoDamageModifiers, "RandomElement": SkillTypeRandomElement,
	"LateConsumeCooldown": SkillTypeLateConsumeCooldown, "Arcane": SkillTypeArcane, "FixedCastTime": SkillTypeFixedCastTime,
	"RequiresOffHandNotWeapon": SkillTypeRequiresOffHandNotWeapon, "Link": SkillTypeLink, "Blessing": SkillTypeBlessing,
	"ZeroReservation": SkillTypeZeroReservation, "DynamicCooldown": SkillTypeDynamicCooldown,
	"Microtransaction": SkillTypeMicrotransaction, "OwnerCannotUse": SkillTypeOwnerCannotUse,
	"ProjectilesNumberModifiersNotApplied": SkillTypeProjectilesNumberModifiersNotApplied,
	"TotemsAreBallistae":                   SkillTypeTotemsAreBallistae, "SkillGrantedBySupport": SkillTypeSkillGrantedBySupport,
	"PreventHexTransfer": SkillTypePreventHexTransfer, "MinionsAreUndamagable": SkillTypeMinionsAreUndamagable,
	"InnateTrauma": SkillTypeInnateTrauma, "DualWieldRequiresDifferentTypes": SkillTypeDualWieldRequiresDifferentTypes,
	"NoVolley": SkillTypeNoVolley, "Retaliation": SkillTypeRetaliation, "NeverExertable": SkillTypeNeverExertable,
	"DisallowTriggerSupports": SkillTypeDisallowTriggerSupports, "ProjectileCannotReturn": SkillTypeProjectileCannotReturn,
	"Offering": SkillTypeOffering, "SupportedByBane": SkillTypeSupportedByBane, "WandAttack": SkillTypeWandAttack,
	"GainsIntensity": SkillTypeGainsIntensity, "CreatesSentinelMinion": SkillTypeCreatesSentinelMinion,
	"SupportedByAutoExertion": SkillTypeSupportedByAutoExertion, "SupportedByCrabTotem": SkillTypeSupportedByCrabTotem,
	"SupportedBySpellTotem": SkillTypeSupportedBySpellTotem, "CreatesCorpse": SkillTypeCreatesCorpse,
	"RequiresStaff": SkillTypeRequiresStaff, "Pact": SkillTypePact,
}

var skillTypeNames = func() map[SkillTypeID]string {
	m := make(map[SkillTypeID]string, len(SkillTypeByName))
	for name, id := range SkillTypeByName {
		m[id] = name
	}
	return m
}()

// SkillTypeName is the inverse of SkillTypeByName; ok is false for an id
// without a named constant.
func SkillTypeName(id SkillTypeID) (name string, ok bool) {
	name, ok = skillTypeNames[id]
	return
}

// ModType is a modifier's aggregation kind (mod.type). The zero value is
// "no type set" and formats as "".
type ModType uint8

const (
	Base ModType = iota + 1
	Inc
	More
	Flag
	Override
	List
	Max
	Min
	Chance   // HitsInvert*ResChance mods (special.go)
	Dummy    // the "Dummy" carrier mod (special.go)
	FlagTypo // the reference's mixed-case "Flag": CanNotUseItem mods and their item-disabler query
)

var modTypeNames = [...]string{"", "BASE", "INC", "MORE", "FLAG", "OVERRIDE", "LIST", "MAX", "MIN", "CHANCE", "DUMMY", "Flag"}

// String is the reference's mod.type text.
func (t ModType) String() string {
	if int(t) < len(modTypeNames) {
		return modTypeNames[t]
	}
	return ""
}

// ModTypeByName resolves the reference's mod.type text (codec, ModTools).
var ModTypeByName = map[string]ModType{
	"BASE": Base, "INC": Inc, "MORE": More, "FLAG": Flag, "OVERRIDE": Override,
	"LIST": List, "MAX": Max, "MIN": Min, "CHANCE": Chance, "DUMMY": Dummy, "Flag": FlagTypo,
}
