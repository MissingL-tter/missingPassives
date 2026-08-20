package modparser

// Transcribed from src/Data/Global.lua. Kept as fields on package-level structs
// so hand-ported parser code reads exactly like the reference: ModFlag.Attack,
// KeywordFlag.Aura, SkillType.Banner.

// ModFlag classifies which damage calculations a modifier participates in.
var ModFlag = struct {
	// Damage modes
	Attack, Spell, Hit, Dot, Cast int64
	// Damage sources
	Melee, Area, Projectile, SourceMask, Ailment, MeleeHit, Weapon int64
	// Weapon types
	Axe, Bow, Claw, Dagger, Mace, Staff, Sword, Wand, Unarmed, Fishing int64
	// Weapon classes
	WeaponMelee, WeaponRanged, Weapon1H, Weapon2H, WeaponMask int64
}{
	Attack: 0x00000001,
	Spell:  0x00000002,
	Hit:    0x00000004,
	Dot:    0x00000008,
	Cast:   0x00000010,

	Melee:      0x00000100,
	Area:       0x00000200,
	Projectile: 0x00000400,
	SourceMask: 0x00000600,
	Ailment:    0x00000800,
	MeleeHit:   0x00001000,
	Weapon:     0x00002000,

	Axe:     0x00010000,
	Bow:     0x00020000,
	Claw:    0x00040000,
	Dagger:  0x00080000,
	Mace:    0x00100000,
	Staff:   0x00200000,
	Sword:   0x00400000,
	Wand:    0x00800000,
	Unarmed: 0x01000000,
	Fishing: 0x02000000,

	WeaponMelee:  0x04000000,
	WeaponRanged: 0x08000000,
	Weapon1H:     0x10000000,
	Weapon2H:     0x20000000,
	WeaponMask:   0x2FFF0000,
}

// KeywordFlag classifies modifiers by skill keyword.
var KeywordFlag = struct {
	// Skill keywords
	Aura, Curse, Warcry, Movement, Physical, Fire, Cold, Lightning, Chaos, Vaal, Bow, Arrow int64
	// Skill types
	Trap, Mine, Totem, Minion, Attack, Spell, Hit, Ailment, Brand int64
	// Other effects
	Poison, Bleed, Ignite int64
	// Damage over Time types
	PhysicalDot, LightningDot, ColdDot, FireDot, ChaosDot int64
	// Match *all* flags instead of any
	MatchAll int64
}{
	Aura:      0x00000001,
	Curse:     0x00000002,
	Warcry:    0x00000004,
	Movement:  0x00000008,
	Physical:  0x00000010,
	Fire:      0x00000020,
	Cold:      0x00000040,
	Lightning: 0x00000080,
	Chaos:     0x00000100,
	Vaal:      0x00000200,
	Bow:       0x00000400,
	Arrow:     0x00000800,

	Trap:    0x00001000,
	Mine:    0x00002000,
	Totem:   0x00004000,
	Minion:  0x00008000,
	Attack:  0x00010000,
	Spell:   0x00020000,
	Hit:     0x00040000,
	Ailment: 0x00080000,
	Brand:   0x00100000,

	Poison: 0x00200000,
	Bleed:  0x00400000,
	Ignite: 0x00800000,

	PhysicalDot:  0x01000000,
	LightningDot: 0x02000000,
	ColdDot:      0x04000000,
	FireDot:      0x08000000,
	ChaosDot:     0x10000000,

	MatchAll: 0x40000000,
}

// SkillType enumerates active skill types from ActiveSkillType.dat.
var SkillType = struct {
	Attack, Spell, Projectile, DualWieldOnly, Buff, Minion, Damage, Area, Duration,
	RequiresShield, ProjectileSpeed, HasReservation, ReservationBecomesCost, Trappable,
	Totemable, Mineable, ElementalStatus, MinionsCanExplode, Chains, Melee,
	MeleeSingleTarget, Multicastable, TotemCastsAlone, Multistrikeable, CausesBurning,
	SummonsTotem, TotemCastsWhenNotDetached, Physical, Fire, Cold, Lightning,
	Triggerable, Trapped, Movement, DamageOverTime, RemoteMined, Triggered, Vaal, Aura,
	CanTargetUnusableCorpse, RangedAttack, Chaos, FixedSpeedProjectile,
	ThresholdJewelArea, ThresholdJewelProjectile, ThresholdJewelDuration,
	ThresholdJewelRangedAttack, Channel, DegenOnlySpellDamage, InbuiltTrigger, Golem,
	Herald, AuraAffectsEnemies, NoRuthless, ThresholdJewelSpellDamage, Cascadable,
	ProjectilesFromUser, MirageArcherCanUse, ProjectileSpiral, SingleMainProjectile,
	MinionsPersistWhenSkillRemoved, ProjectileNumber, Warcry, Instant, Brand,
	DestroysCorpse, NonHitChill, ChillingArea, AppliesCurse, CanRapidFire, AuraDuration,
	AreaSpell, OR, AND, NOT, AppliesMaim, CreatesMinion, Guard, Travel, Blink,
	CanHaveBlessing, ProjectilesNotFromUser, AttackInPlaceIsDefault, Nova,
	InstantNoRepeatWhenHeld, InstantShiftAttackForLeftMouse, AuraNotOnCaster, Banner,
	Rain, Cooldown, ThresholdJewelChaining, Slam, Stance, NonRepeatable,
	OtherThingUsesSkill, Steel, Hex, Mark, Aegis, Orb, KillNoDamageModifiers,
	RandomElement, LateConsumeCooldown, Arcane, FixedCastTime, RequiresOffHandNotWeapon,
	Link, Blessing, ZeroReservation, DynamicCooldown, Microtransaction, OwnerCannotUse,
	ProjectilesNumberModifiersNotApplied, TotemsAreBallistae, SkillGrantedBySupport,
	PreventHexTransfer, MinionsAreUndamagable, InnateTrauma,
	DualWieldRequiresDifferentTypes, NoVolley, Retaliation, NeverExertable,
	DisallowTriggerSupports, ProjectileCannotReturn, Offering, SupportedByBane,
	WandAttack, GainsIntensity, CreatesSentinelMinion, SupportedByAutoExertion,
	SupportedByCrabTotem, SupportedBySpellTotem, CreatesCorpse, RequiresStaff, Pact int64
}{
	Attack: 1, Spell: 2, Projectile: 3, DualWieldOnly: 4, Buff: 5, Minion: 6,
	Damage: 7, Area: 8, Duration: 9, RequiresShield: 10, ProjectileSpeed: 11,
	HasReservation: 12, ReservationBecomesCost: 13, Trappable: 14, Totemable: 15,
	Mineable: 16, ElementalStatus: 17, MinionsCanExplode: 18, Chains: 19, Melee: 20,
	MeleeSingleTarget: 21, Multicastable: 22, TotemCastsAlone: 23, Multistrikeable: 24,
	CausesBurning: 25, SummonsTotem: 26, TotemCastsWhenNotDetached: 27, Physical: 28,
	Fire: 29, Cold: 30, Lightning: 31, Triggerable: 32, Trapped: 33, Movement: 34,
	DamageOverTime: 35, RemoteMined: 36, Triggered: 37, Vaal: 38, Aura: 39,
	CanTargetUnusableCorpse: 40, RangedAttack: 41, Chaos: 42, FixedSpeedProjectile: 43,
	ThresholdJewelArea: 44, ThresholdJewelProjectile: 45, ThresholdJewelDuration: 46,
	ThresholdJewelRangedAttack: 47, Channel: 48, DegenOnlySpellDamage: 49,
	InbuiltTrigger: 50, Golem: 51, Herald: 52, AuraAffectsEnemies: 53, NoRuthless: 54,
	ThresholdJewelSpellDamage: 55, Cascadable: 56, ProjectilesFromUser: 57,
	MirageArcherCanUse: 58, ProjectileSpiral: 59, SingleMainProjectile: 60,
	MinionsPersistWhenSkillRemoved: 61, ProjectileNumber: 62, Warcry: 63, Instant: 64,
	Brand: 65, DestroysCorpse: 66, NonHitChill: 67, ChillingArea: 68, AppliesCurse: 69,
	CanRapidFire: 70, AuraDuration: 71, AreaSpell: 72, OR: 73, AND: 74, NOT: 75,
	AppliesMaim: 76, CreatesMinion: 77, Guard: 78, Travel: 79, Blink: 80,
	CanHaveBlessing: 81, ProjectilesNotFromUser: 82, AttackInPlaceIsDefault: 83,
	Nova: 84, InstantNoRepeatWhenHeld: 85, InstantShiftAttackForLeftMouse: 86,
	AuraNotOnCaster: 87, Banner: 88, Rain: 89, Cooldown: 90, ThresholdJewelChaining: 91,
	Slam: 92, Stance: 93, NonRepeatable: 94, OtherThingUsesSkill: 95, Steel: 96,
	Hex: 97, Mark: 98, Aegis: 99, Orb: 100, KillNoDamageModifiers: 101,
	RandomElement: 102, LateConsumeCooldown: 103, Arcane: 104, FixedCastTime: 105,
	RequiresOffHandNotWeapon: 106, Link: 107, Blessing: 108, ZeroReservation: 109,
	DynamicCooldown: 110, Microtransaction: 111, OwnerCannotUse: 112,
	ProjectilesNumberModifiersNotApplied: 113, TotemsAreBallistae: 114,
	SkillGrantedBySupport: 115, PreventHexTransfer: 116, MinionsAreUndamagable: 117,
	InnateTrauma: 118, DualWieldRequiresDifferentTypes: 119, NoVolley: 120,
	Retaliation: 121, NeverExertable: 122, DisallowTriggerSupports: 123,
	ProjectileCannotReturn: 124, Offering: 125, SupportedByBane: 126, WandAttack: 127,
	GainsIntensity: 128, CreatesSentinelMinion: 129, SupportedByAutoExertion: 130,
	SupportedByCrabTotem: 131, SupportedBySpellTotem: 132, CreatesCorpse: 133,
	RequiresStaff: 134, Pact: 135,
}
