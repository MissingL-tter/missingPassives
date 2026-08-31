// CalcTriggers.lua L908-1585: the configTable dispatch. Every key the
// reference defines is present so the lookup order is faithful; the entries
// no corpus build reaches are nil and panic on match, rather than silently
// behaving like "no trigger config at all".
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modstore"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// triggerConfigBuilder returns nil when the entry inspects the env and
// decides it does not apply (several reference entries do).
type triggerConfigBuilder func(env *Env, actor *performActor) *triggerConfig

// Populated in init: several entries reference findTriggerSkill, whose call
// graph reaches RunTriggers and so this table, which Go's initialization
// analysis reports as a cycle for a composite-literal var.
var triggerConfigTable map[string]triggerConfigBuilder

func init() {
	triggerConfigTable = map[string]triggerConfigBuilder{
		// Unported (its comparer overloads the rate argument as totem
		// life); nil = the documented loud panic on match, never a silent
		// no-config (lua-residue.md T2).
		"avenging flame":                     nil,
		"law of the wilds":                   lawOfTheWildsConfig,
		"the rippling thoughts":              ripplingThoughtsConfig(false),
		"the surging thoughts":               ripplingThoughtsConfig(true),
		"the hidden blade":                   hiddenBladeConfig,
		"replica eternity shroud":            globalTriggerSelfConfig,
		"shroud of the lightless":            globalTriggerSelfConfig,
		"limbsplit":                          namedMeleeTriggerConfig("Gore Shockwave"),
		"the cauteriser":                     namedMeleeTriggerConfig("Gore Shockwave"),
		"duskblight":                         namedDamageTriggerConfig("Stalking Pustule"),
		"lioneye's paws":                     lioneyesPawsConfig,
		"replica lioneye's paws":             lioneyesPawsConfig,
		"moonbender's wing":                  moonbendersWingConfig,
		"ngamahu's flame":                    namedMeleeTriggerConfig("Molten Burst"),
		"cameria's avarice":                  namedDamageTriggerConfig("Icicle Burst"),
		"starcaller":                         namedMeleeTriggerConfig("Starfall"),
		"uul-netol's embrace":                namedDamageTriggerConfig("Bone Nova"),
		"rigwald's crest":                    killFinalRateConfig,
		"jorrhast's blacksteel":              killFinalRateConfig,
		"ashcaller":                          killFinalRateConfig,
		"arakaali's fang":                    killTriggerConfig,
		"sporeguard":                         killTriggerConfig,
		"mark of the elder":                  markRingConfig,
		"mark of the shaper":                 markRingConfig,
		"poet's pen":                         poetsPenConfig,
		"maloney's mechanism":                maloneysMechanismConfig,
		"asenath's chant":                    asenathsChantConfig,
		"vixen's entrapment":                 vixensEntrapmentConfig,
		"flames of judgement":                judgementConfig,
		"storm of judgement":                 judgementConfig,
		"trigger craft":                      triggerCraftConfig,
		"kitava's thirst":                    kitavasThirstConfig,
		"foulborn kitava's thirst":           foulbornKitavasThirstConfig,
		"mjolner":                            mjolnerConfig,
		"wing of the wyvern":                 wingOfTheWyvernConfig,
		"cospri's malice":                    cosprisMaliceConfig,
		"seven teachings":                    sevenTeachingsConfig,
		"squirming terror":                   squirmingTerrorConfig,
		"kinetic flux":                       kineticFluxConfig,
		"cast on critical strike":            castOnCriticalStrikeConfig,
		"cast on melee kill":                 castOnMeleeKillConfig,
		"nova":                               novaConfig,
		"cast when damage taken":             castWhenDamageTakenConfig,
		"cast when stunned":                  castWhenStunnedConfig,
		"cast on ward break":                 castOnWardBreakConfig,
		"spellslinger":                       spellslingerConfig,
		"call to arms":                       callToArmsConfig,
		"automation":                         automationConfig,
		"autoexertion":                       autoexertionConfig,
		"seize the flesh":                    seizeTheFleshConfig,
		"fissure":                            fissureConfig,
		"mark on hit":                        markOnHitConfig,
		"hextouch":                           hextouchConfig,
		"oskarm":                             oskarmConfig,
		"tempest shield":                     tempestShieldConfig,
		"shattershard":                       shattershardConfig,
		"battlemage's cry":                   battlemagesCryConfig,
		"arcanist brand":                     arcanistBrandConfig,
		"cast on death":                      castOnDeathConfig,
		"combust":                            combustConfig,
		"prismatic burst":                    prismaticBurstConfig,
		"voidstorm":                          voidstormConfig,
		"shockwave":                          shockwaveConfig,
		"void shockwave":                     shockwaveConfig,
		"falling crystal":                    shockwaveConfig,
		"call the pyre":                      callThePyreConfig,
		"manaforged arrows":                  manaforgedArrowsConfig,
		"doom blast":                         doomBlastConfig,
		"cast while channelling":             cwcConfig,
		"focus":                              focusConfig,
		"snipe":                              snipeConfig,
		"intuitive link":                     intuitiveLinkConfig,
		"svalinn cast on block":              castOnBlockConfig,
		"festering resentment cast on block": castOnBlockConfig,
		"hex on trap":                        hexOnTrapConfig,
		"supporttriggerfirespellonhit":       settlersEnchantConfig,
		"ghostly artillery":                  ghostlyArtilleryConfig,
		"replica gifts from above":           replicaGiftsFromAboveConfig,
		"bursting toad":                      burstingToadConfig,
		"TriggeredMoltenStrike":              triggeredMoltenStrikeConfig,
		"FieryImpactHeistMaceImplicit":       fieryImpactConfig,
	}
}

// castOnCriticalStrikeConfig ports the "cast on critical strike" entry (L1162).
func castOnCriticalStrikeConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] && env.slotMatch(skill)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByCoc") && env.slotMatch(skill)
		},
	}
}

// novaConfig ports the "nova" entry (L1178): the Holy Relic minion's
// triggered nova, driven by the player's attacks.
func novaConfig(env *Env, actor *performActor) *triggerConfig {
	if env.Minion == nil || env.Minion.MainSkill == nil {
		return nil
	}
	sim := &simSkill{uuid: env.cacheSkillUUID(env.Minion.MainSkill)}
	if sd := env.Minion.MainSkill.SkillData; sd.Flag("cooldown") {
		n := sd.N("cooldown")
		sim.cd = &n
	}
	return &triggerConfig{
		triggerName:     "Summon Holy Relic",
		actor:           env.minionPA,
		triggeredSkills: []*simSkill{sim},
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack]
		},
	}
}

// castWhenDamageTakenConfig ports the "cast when damage taken" entry
// (L1186): the skill triggers off damage rather than off another skill, so
// it is its own source and the rate comes from the trigger cap.
func castWhenDamageTakenConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if !main.SkillData.Flag("triggeredByDamageTaken") {
		return nil
	}
	thresholdMod := Mod(main.SkillModList, nil, "CWDTThreshold")
	env.Player.Output.SetN("CWDTThreshold", main.SkillData.N("triggeredByDamageTaken")*thresholdMod)
	main.SkillFlags["globalTrigger"] = true
	return &triggerConfig{source: main}
}

// triggeredMoltenStrikeConfig ports the "TriggeredMoltenStrike" entry
// (L1577): any melee or attack skill sets it off.
func triggeredMoltenStrikeConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: meleeOrAttack}
}

// meleeOrAttack is the trigger condition several unique-item entries share.
func meleeOrAttack(env *Env, skill *ActiveSkill) bool {
	return skill.SkillTypes[modparser.SkillTypeMelee] || skill.SkillTypes[modparser.SkillTypeAttack]
}

// doomBlastConfig ports the "doom blast" entry (L1394): the hexes that are
// removed drive the blast, so the source is the curse cast rate and each
// overlapping curse is another blast.
func doomBlastConfig(env *Env, actor *performActor) *triggerConfig {
	if env.ConfigInput.DoomBlastSource == "replacement" {
		env.ModDB.AddMod(newModS("UsesCurseOverlaps", modparser.Flag, modparser.Bool(true), "Config"))
	}
	env.PlayerMainSkill.SkillData.SetFlag("ignoresTickRate", true)
	cfg := &triggerConfig{
		useCastRate:       true,
		customTriggerName: "Doom Blast triggering Hex: ",
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeHex] && env.slotMatch(skill)
		},
	}
	// `#Tabulate(...) > 0 and m_max(Sum(...), 1)`: false, not nil, when no
	// mod grants overlaps.
	if len(env.ModDB.Tabulate(modparser.Base, nil, "Multiplier:CurseOverlaps")) > 0 {
		n := math.Max(env.ModDB.Sum(modparser.Base, nil, "Multiplier:CurseOverlaps"), 1)
		cfg.overlaps = &n
	}
	return cfg
}

// voidstormConfig ports the "voidstorm" entry (L1369).
func voidstormConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] &&
				skill.SkillTypes[modparser.SkillTypeRain] && env.slotMatch(skill)
		},
	}
}

// shockwaveConfig ports the "shockwave" entry (L1372); "void shockwave" and
// "falling crystal" are the same condition.
func shockwaveConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: meleeAndSlot}
}

func meleeAndSlot(env *Env, skill *ActiveSkill) bool {
	return skill.SkillTypes[modparser.SkillTypeMelee] && env.slotMatch(skill)
}

// tempestShieldConfig ports the "tempest shield" entry (L1301): it triggers
// off being hit, so it is its own source.
func tempestShieldConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillFlags["globalTrigger"] = true
	return &triggerConfig{source: env.PlayerMainSkill}
}

// callThePyreConfig ports the "call the pyre" entry (L1381).
func callThePyreConfig(env *Env, actor *performActor) *triggerConfig {
	if !env.EnemyDB.Flag(nil, "Condition:Ignited") {
		env.PlayerMainSkill.InfoMessage = "Call the Pyre requires Ignited enemies"
		return nil
	}
	// too much of a pain to pull this from the triggering skill
	chance := 50.0
	return &triggerConfig{triggerChance: &chance, triggerSkillCond: meleeAndSlot}
}

// burstingToadConfig ports the "bursting toad" entry (L1557): the toad bursts
// on its own interval, so the rate cap is that interval.
func burstingToadConfig(env *Env, actor *performActor) *triggerConfig {
	triggerInterval := math.Inf(1)
	// All gems in the socket group should return the same HexToadCooldown
	// even when there are multiple hextoad support gems slotted
	for _, skill := range env.PlayerActiveSkills {
		if skill.SkillData.Flag("hextoadTriggerInterval") {
			triggerInterval = math.Min(triggerInterval, skill.SkillData.N("hextoadTriggerInterval"))
		}
	}
	if math.IsInf(triggerInterval, 1) {
		return nil
	}
	env.PlayerMainSkill.SkillFlags["globalTrigger"] = true
	env.PlayerMainSkill.SkillData.SetN("triggerRateCapOverride", 1/triggerInterval)
	return &triggerConfig{source: env.PlayerMainSkill}
}

// automationConfig ports the "automation" entry (L1242): Automation triggers
// its linked skills on their own cooldowns.
func automationConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if main.ActiveEffect.GrantedEffect.Name == "Automation" {
		main.SkillFlags["globalTrigger"] = true
		main.SkillFlags["skipEffectiveRate"] = true
	} else if main.TriggeredBy != nil {
		// Needed to get the cooldown from the active part
		for _, skill := range env.PlayerActiveSkills {
			if skill.ActiveEffect.GrantedEffect.Name == "Automation" {
				main.TriggeredBy.GrantedEffect = skill.ActiveEffect.GrantedEffect
				break
			}
		}
	}
	return &triggerConfig{triggerOnUse: true, useCastRate: true, source: main}
}

// markOnHitConfig ports the "mark on hit" entry (L1286).
func markOnHitConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeAttack]
	}}
}

// hextouchConfig ports the "hextouch" entry (L1289): the curse lands at the
// rate of the attack that applies it, and that rate is already final.
func hextouchConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillData.SetFlag("sourceRateIsFinal", true)
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeAttack] && env.slotMatch(skill)
	}}
}

// castOnBlockConfig ports the two identical cast-on-block item entries
// (Svalinn L1509, Festering Resentment L1516).
func castOnBlockConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillFlags["globalTrigger"] = true
	return &triggerConfig{
		source: env.PlayerMainSkill,
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return env.slotMatch(skill) && skill.TriggeredBy != nil &&
				env.canGrantedEffectSupportActiveSkill(skill.TriggeredBy.GrantedEffect, skill, false)
		},
	}
}

// spellslingerConfig ports the "spellslinger" entry (L1211): the spell goes
// off with the wand attack that slings it.
func spellslingerConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if main.ActiveEffect.GrantedEffect.Name == "Spellslinger" {
		main.SkillFlags["skipEffectiveRate"] = true
	}
	// Spell slinger adds a cooldown with its support part
	main.SkillData.SetFlag("sourceRateIsFinal", true)
	return &triggerConfig{
		triggerName:  "Spellslinger",
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			isWandProjectileAttack := skill.SkillTypes[modparser.SkillTypeAttack] &&
				skill.SkillTypes[modparser.SkillTypeProjectile] && cfgHasFlag(skill.SkillCfg, modparser.FlagWand)
			return isWandProjectileAttack && !skill.SkillData.Flag("triggeredBySpellSlinger")
		},
	}
}

// manaforgedArrowsConfig ports the "manaforged arrows" entry (L1389).
func manaforgedArrowsConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerOnUse: true,
		triggerName:  "Manaforged Arrows",
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] && cfgHasFlag(skill.SkillCfg, modparser.FlagBow)
		},
	}
}

// combustConfig ports the "combust" entry (L1358).
func combustConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeMelee]
		},
		comparer: func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
			// Skills with no uptime ratio are not exerted by infernal cry so
			// should not be considered.
			uptimeRatio := env.GlobalCache[uuid].out("InfernalUpTimeRatio")
			return defaultComparer(env, uuid, source, triggerRate) && uptimeRatio.Truthy()
		},
	}
}

// cfgHasFlag is `band(cfg.flags, ModFlag.X) > 0`.
func cfgHasFlag(cfg *modstore.Cfg, flag modparser.ModFlag) bool {
	return cfg != nil && cfg.Flags != nil && *cfg.Flags&flag != 0
}

// vixensEntrapmentConfig ports the "vixen's entrapment" entry (L1051).
func vixensEntrapmentConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		useCastRate: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeHex]
		},
	}
}

// cosprisMaliceConfig ports the "cospri's malice" entry (L1136).
func cosprisMaliceConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeMelee] &&
				cfgHasFlag(skill.SkillCfg, modparser.FlagSword|modparser.FlagWeapon1H)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByCospris") && sameSocketSlot(env.PlayerMainSkill, skill)
		},
	}
}

// sameSocketSlot is `mainSkill.socketGroup.slot == skill.socketGroup.slot`.
func sameSocketSlot(a, b *ActiveSkill) bool {
	slot := func(s *ActiveSkill) (string, bool) {
		if s.SocketGroup == nil {
			return "", false
		}
		return s.SocketGroup.Slot, s.SocketGroup.Slot != ""
	}
	sa, oka := slot(a)
	sb, okb := slot(b)
	// Lua nil == nil is true only when both groups exist with no slot;
	// a missing group makes the index error, but every caller guards on
	// socketGroup being present in practice.
	return oka == okb && sa == sb
}

// cwcConfig ports the "cast while channelling" entry (L1404).
func cwcConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{customHandler: func(env *Env) { env.cwcHandler() }}
}

// arcanistBrandConfig ports the "arcanist brand" entry (L1332): the brand
// activates on its own frequency and fires every linked skill per activation,
// per attached brand.
func arcanistBrandConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if main.ActiveEffect.GrantedEffect.Name == "Arcanist Brand" || main.TriggeredBy == nil {
		return nil
	}
	main.SkillData.SetFlag("sourceRateIsFinal", true)
	for _, skill := range env.PlayerActiveSkills {
		if skill.ActiveEffect.GrantedEffect.Name == "Arcanist Brand" {
			main.TriggeredBy.MainSkill = skill
			break
		}
	}
	brand := main.TriggeredBy.MainSkill
	activationFreqInc := (100 + brand.SkillModList.Sum(modparser.Inc, brand.SkillCfg, "Speed", "BrandActivationFrequency")) / 100
	activationFreqMore := brand.SkillModList.More(brand.SkillCfg, "BrandActivationFrequency")
	main.TriggeredBy.ActivationFreqInc = activationFreqInc
	main.TriggeredBy.ActivationFreqMore = activationFreqMore
	main.TriggeredBy.AttachedBrandCount = brand.SkillData.N("attachedBrandCount")
	main.TriggeredBy.IgnoresTickRate = true
	trigRate := brand.SkillData.N("repeatFrequency") * activationFreqInc * activationFreqMore * main.TriggeredBy.AttachedBrandCount
	return &triggerConfig{
		trigRate: &trigRate,
		source:   brand,
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByBrand") && env.slotMatch(skill)
		},
	}
}

// triggerCraftConfig ports the "trigger craft" entry (L1069): the crafted
// "trigger a socketed spell" mod. The config finds the source itself so it
// can special-case totem-like sources onto their placement rate.
func triggerCraftConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if !main.SkillData.Flag("triggeredByCraft") {
		return nil
	}
	var trigRate *float64
	var source *ActiveSkill
	var uuid string
	useCastRate := false
	var triggeredSkills []*simSkill
	for _, skill := range env.PlayerActiveSkills {
		if (skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack] || skill.SkillTypes[modparser.SkillTypeSpell]) &&
			!skill.SkillFlags["aura"] && skill != main && !skill.SkillData.Flag("triggeredByCraft") &&
			!env.geFromItem(skill.ActiveEffect.GrantedEffect) && !isTriggered(skill) {
			source, trigRate, uuid = env.findTriggerSkill(skill, source, trigRate, nil)
			if (skill.SkillFlags["totem"] || skill.SkillFlags["golem"] || skill.SkillFlags["banner"] || skill.SkillFlags["ballista"]) &&
				skill.ActiveEffect.GrantedEffect.CastTime != nil {
				// A totem-like source triggers per placement, not per hit.
				rate := 1 / *skill.ActiveEffect.GrantedEffect.CastTime
				if skill.ActiveEffect.GrantedEffect.Levels != nil {
					cd := 0.0
					if lvl := skill.ActiveEffect.GrantedEffect.LevelData(skill.ActiveEffect.Level); lvl != nil {
						cd = lvl.Extra["cooldown"]
					}
					rate = 1 / (*skill.ActiveEffect.GrantedEffect.CastTime + cd)
				}
				trigRate = &rate
				useCastRate = true
			}
		}
		if skill.SkillData.Flag("triggeredByCraft") && sameSocketSlot(main, skill) {
			triggeredSkills = append(triggeredSkills, env.packageSkillDataForSimulation(skill))
		}
	}
	return &triggerConfig{trigRate: trigRate, source: source, uuid: uuid, useCastRate: useCastRate, triggeredSkills: triggeredSkills}
}

// autoexertionConfig ports the "autoexertion" entry (L1258): warcries exerted
// automatically on the support's own cooldown.
func autoexertionConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if main.ActiveEffect.GrantedEffect.Name == "Autoexertion" {
		main.SkillFlags["globalTrigger"] = true
		main.SkillFlags["skipEffectiveRate"] = true
	} else if main.TriggeredBy != nil {
		// Needed to get the cooldown from the active part: Autoexertion has
		// cooldown as part of its support part; applying the one from the
		// active part to be consistent with Automation and Spellslinger.
		for _, skill := range env.PlayerActiveSkills {
			if skill.ActiveEffect.GrantedEffect.Name == "Autoexertion" {
				main.TriggeredBy.GrantedEffect = skill.ActiveEffect.GrantedEffect
				break
			}
		}
	}
	main.SkillData.SetFlag("sourceRateIsFinal", true)
	main.SkillData.SetFlag("ignoresTickRate", true)
	return &triggerConfig{triggerOnUse: true, useCastRate: true, source: main}
}

// battlemagesCryConfig ports the "battlemage's cry" entry (L1321).
func battlemagesCryConfig(env *Env, actor *performActor) *triggerConfig {
	if env.PlayerMainSkill.ActiveEffect.GrantedEffect.Name == "Battlemage's Cry" {
		return nil
	}
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeMelee]
		},
		comparer: func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
			// Skills with no uptime ratio are not exerted by battlemage so
			// should not be considered.
			uptimeRatio := env.GlobalCache[uuid].out("BattlemageUpTimeRatio")
			return defaultComparer(env, uuid, source, triggerRate) && uptimeRatio.Truthy()
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByBattleMageCry") && env.slotMatch(skill)
		},
	}
}

// castOnMeleeKillConfig ports the "cast on melee kill" entry (L1166).
func castOnMeleeKillConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if !env.ModDB.Flag(nil, "Condition:KilledRecently") {
		main.InfoMessage2 = "DPS reported assuming Self-Cast"
		main.InfoMessage = "Cast on Melee Kill requires recent kills"
		return nil
	}
	return &triggerConfig{
		assumingEveryHitKills: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] && skill.SkillTypes[modparser.SkillTypeMelee] && env.slotMatch(skill)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByMeleeKill") && env.slotMatch(skill)
		},
	}
}

// castWhenStunnedConfig ports the "cast when stunned" entry (L1201).
func castWhenStunnedConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	main.SkillFlags["globalTrigger"] = true
	cfg := &triggerConfig{source: main}
	if main.SkillData.Has("chanceToTriggerOnStun") {
		n := main.SkillData.N("chanceToTriggerOnStun")
		cfg.triggerChance = &n
	}
	return cfg
}

// castOnWardBreakConfig ports the "cast on ward break" entry (L1206).
func castOnWardBreakConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	main.SkillFlags["globalTrigger"] = true
	cfg := &triggerConfig{source: main}
	if main.SkillData.Has("chanceToTriggerOnWardBreak") {
		n := main.SkillData.N("chanceToTriggerOnWardBreak")
		cfg.triggerChance = &n
	}
	return cfg
}

// castOnDeathConfig ports the "cast on death" entry (L1354): it sets its
// flags and returns NO config, so the dispatch's else-arm clears
// skillData.triggered again right after.
func castOnDeathConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	main.SkillFlags["globalTrigger"] = true
	main.SkillData.SetFlag("triggered", true)
	return nil
}

// prismaticBurstConfig ports the "prismatic burst" entry (L1366).
func prismaticBurstConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeAttack] && env.slotMatch(skill)
	}}
}

// intuitiveLinkConfig ports the "intuitive link" entry (L1494): the linked
// target's casts trigger the spells, at a config-provided rate.
func intuitiveLinkConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if main.ActiveEffect.GrantedEffect.Name == "Intuitive Link" || main.TriggeredBy == nil {
		return nil
	}
	for _, skill := range env.PlayerActiveSkills {
		if skill.ActiveEffect.GrantedEffect.Name == "Intuitive Link" {
			main.TriggeredBy.MainSkill = skill
			break
		}
	}
	trigRate := env.ModDB.Sum(modparser.Base, nil, "IntuitiveLinkSourceRate")
	return &triggerConfig{
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeSpell] && env.slotMatch(skill) && skill != main.TriggeredBy.MainSkill
		},
		trigRate:    &trigRate,
		source:      main.TriggeredBy.MainSkill,
		sourceName:  "Custom source",
		useCastRate: true,
	}
}

// snipeConfig ports the "snipe" entry (L1410): the channel builds stages and
// either releases itself or triggers the supported skills in socket order,
// one per stage.
func snipeConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	snipeStages := math.Min(env.ModDB.Sum(modparser.Base, nil, "Multiplier:SnipeStage"), env.ModDB.Sum(modparser.Base, nil, "Multiplier:SnipeStagesMax"))
	snipeHitMulti := main.SkillModList.Sum(modparser.Base, main.SkillCfg, "snipeHitMulti")
	snipeAilmentMulti := main.SkillModList.Sum(modparser.Base, main.SkillCfg, "snipeAilmentMulti")
	var triggeredSkills []*ActiveSkill
	for _, skill := range env.PlayerActiveSkills {
		if skill.SkillData.Flag("triggeredBySnipe") && skill.SocketGroup != nil && sameSocketSlot(skill, main) {
			triggeredSkills = append(triggeredSkills, skill)
		}
	}

	if main.ActiveEffect.GrantedEffect.Name == "Snipe" {
		if env.LimitedSkills[env.cacheSkillUUID(main)] {
			// Snipe is being used by some other skill; it does not get the
			// more-damage mods then.
			snipeStages = 0
		} else {
			// max(1, stages) keeps it consistent with other channelled
			// ranged skills; the first stage takes 0.5x time to channel.
			main.SkillData.SetN("hitTimeMultiplier", math.Max(1, snipeStages)-0.5)
		}
		if len(triggeredSkills) < 1 {
			// Snipe is being used as a standalone skill. `if snipeStages`: a
			// bare number is always truthy.
			main.SkillModList.AddMod(newModS("Multiplier:SnipeStages", modparser.Base, modparser.Num(snipeStages), "Snipe"))
			main.SkillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(snipeHitMulti), "Snipe", modparser.FlagHit, modparser.KeywordNone, &modparser.MultiplierTag{Var: "SnipeStages"}))
			main.SkillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(snipeAilmentMulti), "Snipe", modparser.FlagAilment, modparser.KeywordNone, &modparser.MultiplierTag{Var: "SnipeStages"}))
		} else {
			// Snipe is a trigger source: it triggers the others and deals no
			// damage itself.
			for _, dmg := range []string{"Lightning", "Cold", "Fire", "Chaos", "Physical"} {
				main.SkillModList.AddMod(newMod("DealNo"+dmg, modparser.Flag, modparser.Bool(true), &modparser.SkillNameTag{SkillName: "Snipe", IncludeTransfigured: true}))
			}
		}
		return nil
	}

	currentSkillSnipeIndex := 0
	for index, skill := range triggeredSkills {
		if skill == main {
			currentSkillSnipeIndex = index + 1
			break
		}
	}

	// Does snipe have enough stages to trigger this skill?
	if currentSkillSnipeIndex == 0 || float64(currentSkillSnipeIndex) > snipeStages {
		main.SkillData.Del("triggered")
		main.InfoMessage2 = "DPS reported assuming Self-Cast"
		main.InfoMessage = "Not enough Snipe stages to trigger this skill"
		return nil
	}
	var source *ActiveSkill
	var trigRate *float64
	main.SkillModList.AddMod(newModS("Multiplier:SnipeStages", modparser.Base, modparser.Num(snipeStages), "Snipe"))
	main.SkillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(snipeAilmentMulti), "Snipe", modparser.FlagAilment, modparser.KeywordNone, &modparser.MultiplierTag{Var: "SnipeStages"}))
	main.SkillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(snipeHitMulti), "Snipe", modparser.FlagHit, modparser.KeywordNone, &modparser.MultiplierTag{Var: "SnipeStages"}))
	for _, skill := range env.PlayerActiveSkills {
		if skill.ActiveEffect.GrantedEffect.Name == "Snipe" && skill.SocketGroup != nil && sameSocketSlot(skill, main) {
			skill.SkillData.SetN("hitTimeMultiplier", snipeStages-0.5)
			uuid := env.cacheSkillUUID(skill)
			if env.GlobalCache[uuid] == nil || env.Mode == ModeCalculator {
				env.BuildActiveSkill(env.Mode, skill, uuid)
			}
			cachedSpeed := env.GlobalCache[uuid].out("HitSpeed")
			usedByMirage := skill.SkillCfg != nil && skill.SkillCfg.SkillCond != nil && skill.SkillCfg.SkillCond["usedByMirage"]
			if !skill.SkillFlags["disable"] && skill.SkillCfg != nil && !usedByMirage &&
				!skill.SkillTypes[modparser.SkillTypeOtherThingUsesSkill] &&
				cachedSpeed.Truthy() && (source == nil || cachedSpeed.Num() > num64(trigRate)) {
				n := cachedSpeed.Num()
				trigRate = &n
				env.Player.Output.Set("ChannelTimeToTrigger", env.GlobalCache[uuid].out("HitTime"))
				source = skill
			}
		}
	}
	return &triggerConfig{trigRate: trigRate, source: source}
}

// num64 is `p or 0`.
func num64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// mjolnerConfig ports the "mjolner" entry (L1129-region): mace hits trigger
// the socketed lightning spells.
func mjolnerConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return (skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]) &&
				cfgHasFlag(skill.SkillCfg, modparser.FlagMace|modparser.FlagWeapon1H) && !env.slotMatch(skill)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByMjolner") && env.slotMatch(skill)
		},
	}
}

// oskarmConfig ports the "oskarm" entry (L1295).
func oskarmConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillData.SetFlag("sourceRateIsFinal", true)
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeAttack]
	}}
}

// markRingConfig ports the identical "mark of the elder" / "mark of the
// shaper" entries (L1014, L1017).
func markRingConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		assumingEveryHitKills: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]
		},
	}
}

// kitavasThirstConfig ports the "kitava's thirst" entry (L1092): the source
// is the fastest skill whose mana cost clears the helmet's threshold.
func kitavasThirstConfig(env *Env, actor *performActor) *triggerConfig {
	requiredManaCost := env.ModDB.Sum(modparser.Base, nil, "KitavaRequiredManaCost")
	chance := env.ModDB.Sum(modparser.Base, nil, "KitavaTriggerChance")
	return &triggerConfig{
		triggerChance: &chance,
		triggerName:   "Kitava's Thirst",
		comparer: func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
			cached := env.GlobalCache[uuid]
			speed, hasSpeed := cached.speedOrHitSpeed()
			manaCost := 0.0
			if cached.ManaCost != nil {
				manaCost = *cached.ManaCost
			}
			return ((source == nil && hasSpeed) || (hasSpeed && speed > num64(triggerRate))) && manaCost >= requiredManaCost
		},
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			// Filtering done by skill() in SkillStatMap, comparer and
			// default excludes
			return true
		},
	}
}

// asenathsChantConfig ports the "asenath's chant" entry (L1042).
func asenathsChantConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return (skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]) &&
				cfgHasFlag(skill.SkillCfg, modparser.FlagBow)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByUnique") && sameSocketSlot(env.PlayerMainSkill, skill) &&
				skill.SkillTypes[modparser.SkillTypeSpell]
		},
	}
}

// maloneysMechanismConfig ports the "maloney's mechanism" entry (L1029): the
// quiver name is fished out of the item's mod source (`.*:.*:(.*),.*` --
// greedy, so the text between the LAST colon and the LAST comma), and the
// Replica variant triggers spells at cast rate instead of bow skills.
func maloneysMechanismConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	item, _ := env.Player.ItemList[main.SkillCfg.SlotName].(*Item)
	modSource := ""
	if item != nil && item.In.ModSource != nil {
		modSource = *item.In.ModSource
	}
	uniqueTriggerName := ""
	if i := strings.LastIndex(modSource, ":"); i >= 0 {
		if j := strings.LastIndex(modSource, ","); j > i {
			uniqueTriggerName = modSource[i+1 : j]
		}
	}
	// `uniqueTriggerName:match("Replica.")`
	ridx := strings.Index(uniqueTriggerName, "Replica")
	isReplica := ridx >= 0 && ridx+len("Replica") < len(uniqueTriggerName)
	return &triggerConfig{
		triggerOnUse: true,
		triggerName:  uniqueTriggerName,
		useCastRate:  isReplica,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			attack := skill.SkillTypes[modparser.SkillTypeAttack] && cfgHasFlag(skill.SkillCfg, modparser.FlagBow) && !isReplica
			spell := skill.SkillTypes[modparser.SkillTypeSpell] && isReplica
			return attack || spell
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByUnique") && sameSocketSlot(env.PlayerMainSkill, skill) &&
				skill.SkillTypes[modparser.SkillTypeRangedAttack]
		},
	}
}

// lioneyesPawsConfig ports the identical "lioneye's paws" / "replica
// lioneye's paws" entries (L967, L973). The granted skill is the normal Rain
// of Arrows (the mod parser's triggerExtraSkill cannot attach the custom
// no-cooldown version), so the cooldown is written here instead.
func lioneyesPawsConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillData.SetN("cooldown", 1.0)
	return &triggerConfig{
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] && cfgHasFlag(skill.SkillCfg, modparser.FlagBow)
		},
	}
}

// The shared shapes below cover the many one-line unique-item entries.

// namedMeleeTriggerConfig is the shape of Limbsplit, The Cauteriser,
// Ngamahu's Flame and Starcaller: any melee hit fires the named skill.
func namedMeleeTriggerConfig(name string) triggerConfigBuilder {
	return func(env *Env, actor *performActor) *triggerConfig {
		return &triggerConfig{triggerName: name, triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeMelee] || skill.SkillTypes[modparser.SkillTypeAttack]
		}}
	}
}

// namedDamageTriggerConfig is the same with the Damage-or-Attack condition
// (Duskblight, Cameria's Avarice, Uul-Netol's Embrace).
func namedDamageTriggerConfig(name string) triggerConfigBuilder {
	return func(env *Env, actor *performActor) *triggerConfig {
		return &triggerConfig{triggerName: name, triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]
		}}
	}
}

// killTriggerConfig: on-kill triggers assuming every hit kills (Arakaali's
// Fang, Sporeguard).
func killTriggerConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		assumingEveryHitKills: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]
		},
	}
}

// killFinalRateConfig: the same plus sourceRateIsFinal (Rigwald's Crest,
// Jorrhast's Blacksteel, Ashcaller).
func killFinalRateConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillData.SetFlag("sourceRateIsFinal", true)
	return killTriggerConfig(env, actor)
}

// globalTriggerSelfConfig: the skill is its own source and triggers globally
// (Replica Eternity Shroud, Shroud of the Lightless).
func globalTriggerSelfConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillFlags["globalTrigger"] = true
	return &triggerConfig{source: env.PlayerMainSkill}
}

// lawOfTheWildsConfig ports the "law of the wilds" entry (L909).
func lawOfTheWildsConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return !skill.SkillTypes[modparser.SkillTypeSummonsTotem] &&
			(skill.SkillTypes[modparser.SkillTypeMelee] || skill.SkillTypes[modparser.SkillTypeAttack]) &&
			cfgHasFlag(skill.SkillCfg, modparser.FlagClaw)
	}}
}

// ripplingThoughtsConfig ports "the rippling thoughts" (L916) and "the
// surging thoughts" (L925) -- identical but for triggerOnUse.
func ripplingThoughtsConfig(onUse bool) triggerConfigBuilder {
	return func(env *Env, actor *performActor) *triggerConfig {
		if env.PlayerMainSkill.ActiveEffect.GrantedEffect.Name != "Storm Cascade" {
			return nil
		}
		return &triggerConfig{
			triggerOnUse: onUse,
			triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
				return skill.SkillTypes[modparser.SkillTypeMelee] || skill.SkillTypes[modparser.SkillTypeAttack]
			},
		}
	}
}

// hiddenBladeConfig ports "the hidden blade" entry (L935): Unseen Strike
// fires on its own two-per-second clock, but only while phasing.
func hiddenBladeConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	main.SkillFlags["globalTrigger"] = true
	main.SkillData.SetN("triggerRateCapOverride", 2.0)
	if env.ModDB.Flag(nil, "Condition:Phasing") {
		return &triggerConfig{source: main}
	}
	main.SkillFlags["disable"] = true
	main.DisableReason = "This skill is requires you to be phasing"
	return nil
}

// moonbendersWingConfig ports the "moonbender's wing" entry (L979); like
// Lioneye's Paws the granted skill is the ordinary Lightning Warp, so the
// cooldown is written here.
func moonbendersWingConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillData.SetN("cooldown", 1.0)
	return &triggerConfig{triggerName: "Lightning Warp", triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeMelee] || skill.SkillTypes[modparser.SkillTypeAttack]
	}}
}

// poetsPenConfig ports the "poet's pen" entry (L1020).
func poetsPenConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return (skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]) &&
				cfgHasFlag(skill.SkillCfg, modparser.FlagWand)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByUnique") && sameSocketSlot(env.PlayerMainSkill, skill) &&
				skill.SkillTypes[modparser.SkillTypeSpell]
		},
	}
}

// judgementConfig ports the identical "flames of judgement" / "storm of
// judgement" entries (L1057, L1063): Queen's Demand casts them.
func judgementConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	main.SkillData.SetFlag("sourceRateIsFinal", true)
	return &triggerConfig{
		triggerName: main.ActiveEffect.GrantedEffect.Name,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.ActiveEffect.GrantedEffect.Name == "Queen's Demand"
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByUnique") && sameSocketSlot(env.PlayerMainSkill, skill)
		},
	}
}

// foulbornKitavasThirstConfig ports the "foulborn kitava's thirst" entry
// (L1106): the life-cost twin of Kitava's Thirst.
func foulbornKitavasThirstConfig(env *Env, actor *performActor) *triggerConfig {
	requiredLifeCost := env.ModDB.Sum(modparser.Base, nil, "FoulbornKitavaRequiredLifeCost")
	chance := env.ModDB.Sum(modparser.Base, nil, "FoulbornKitavaTriggerChance")
	return &triggerConfig{
		triggerChance: &chance,
		triggerName:   "Foulborn Kitava's Thirst",
		comparer: func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
			cached := env.GlobalCache[uuid]
			speed, hasSpeed := cached.speedOrHitSpeed()
			lifeCost := 0.0
			if cached.LifeCost != nil {
				lifeCost = *cached.LifeCost
			}
			return ((source == nil && hasSpeed) || (hasSpeed && speed > num64(triggerRate))) && lifeCost >= requiredLifeCost
		},
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool { return true },
	}
}

// wingOfTheWyvernConfig ports the "wing of the wyvern" entry (L1128).
func wingOfTheWyvernConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return (skill.SkillTypes[modparser.SkillTypeDamage] || skill.SkillTypes[modparser.SkillTypeAttack]) &&
				cfgHasFlag(skill.SkillCfg, modparser.FlagBow) && !env.slotMatch(skill)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByUnique") && env.slotMatch(skill)
		},
	}
}

// sevenTeachingsConfig ports the "seven teachings" entry (L1142): unarmed
// melee hits trigger it.
func sevenTeachingsConfig(env *Env, actor *performActor) *triggerConfig {
	unarmedMelee := modparser.FlagUnarmed | modparser.FlagMelee
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeMelee] && skill.SkillCfg != nil &&
				skill.SkillCfg.Flags != nil && *skill.SkillCfg.Flags&unarmedMelee == unarmedMelee
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredBySevenTeachings") && env.slotMatch(skill)
		},
	}
}

// squirmingTerrorConfig ports the "squirming terror" entry (L1146).
func squirmingTerrorConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if !env.ModDB.Flag(nil, "Condition:KilledRecently") {
		main.InfoMessage2 = "DPS reported assuming Self-Cast"
		main.InfoMessage = "Squirming Terror requires recent kills"
		return nil
	}
	return &triggerConfig{
		assumingEveryHitKills: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] && skill.SkillTypes[modparser.SkillTypeMelee]
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredBySquirmingTerror") && env.slotMatch(skill)
		},
	}
}

// kineticFluxConfig ports the "kinetic flux" entry (L1158).
func kineticFluxConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack] && env.slotMatch(skill)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByKineticFlux") && env.slotMatch(skill)
		},
	}
}

// callToArmsConfig ports the "call to arms" entry (L1224), kept for backwards
// compatibility; the shape of Automation.
func callToArmsConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if main.ActiveEffect.GrantedEffect.Name == "Call to Arms" {
		main.SkillFlags["globalTrigger"] = true
		main.SkillFlags["skipEffectiveRate"] = true
	} else if main.TriggeredBy != nil {
		// Needed to get the cooldown from the active part
		for _, skill := range env.PlayerActiveSkills {
			if skill.ActiveEffect.GrantedEffect.Name == "Call to Arms" {
				main.TriggeredBy.GrantedEffect = skill.ActiveEffect.GrantedEffect
				break
			}
		}
	}
	return &triggerConfig{triggerOnUse: true, useCastRate: true, source: main}
}

// seizeTheFleshConfig ports the "seize the flesh" entry (L1280).
func seizeTheFleshConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeWarcry] && env.slotMatch(skill)
	}}
}

// fissureConfig ports the "fissure" entry (L1283).
func fissureConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerOnUse: true, triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillTypeSlam] && env.slotMatch(skill)
	}}
}

// shattershardConfig ports the "shattershard" entry (L1305): the crystal's
// duration is its pseudo cooldown, read from the skill's own cached build.
func shattershardConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	main.SkillFlags["globalTrigger"] = true
	uuid := env.cacheSkillUUID(main)
	if env.GlobalCache[uuid] == nil || env.Mode == ModeCalculator {
		env.BuildActiveSkill(env.Mode, main, uuid, uuid)
	}
	main.SkillData.SetN("triggerRateCapOverride", 1/env.GlobalCache[uuid].out("Duration").Num())
	return &triggerConfig{source: main}
}

// hexOnTrapConfig ports the "hex on trap" entry (L1523, Hand of the Lords):
// the hex is triggered only when a trap is triggered.
func hexOnTrapConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeTrapped]
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredByTrapTrigger") && env.slotMatch(skill)
		},
	}
}

// settlersEnchantConfig ports the "supporttriggerfirespellonhit" entry
// (L1532): the skill is triggered only when the weapon with the enchant on
// it hits.
func settlersEnchantConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeMelee]
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillData.Flag("triggeredBySettlersEnchantTrigger") && env.slotMatch(skill)
		},
	}
}

// ghostlyArtilleryConfig ports the "ghostly artillery" entry (L1541).
func ghostlyArtilleryConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		sourceWeapon: true,
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillTypeAttack]
		},
	}
}

// replicaGiftsFromAboveConfig ports the "replica gifts from above" entry
// (L1549).
func replicaGiftsFromAboveConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return !skill.SkillTypes[modparser.SkillTypeSummonsTotem] && skill.SkillTypes[modparser.SkillTypeAttack]
	}}
}

// fieryImpactConfig ports the "FieryImpactHeistMaceImplicit" entry (L1580).
func fieryImpactConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{sourceWeapon: true, triggerSkillCond: meleeOrAttack}
}

// focusConfig ports the "focus" entry (L1407): helmetFocusHandler replaces
// the default handler outright.
func focusConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{customHandler: func(env *Env) { env.helmetFocusHandler() }}
}
