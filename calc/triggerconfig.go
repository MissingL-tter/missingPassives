// CalcTriggers.lua L908-1585: the configTable dispatch. Every key the
// reference defines is present so the lookup order is faithful; the entries
// no corpus build reaches are nil and panic on match, rather than silently
// behaving like "no trigger config at all".
package calc

import (
	"math"

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
		"law of the wilds":                   nil,
		"the rippling thoughts":              nil,
		"the surging thoughts":               nil,
		"the hidden blade":                   nil,
		"replica eternity shroud":            nil,
		"shroud of the lightless":            nil,
		"limbsplit":                          nil,
		"the cauteriser":                     nil,
		"duskblight":                         nil,
		"lioneye's paws":                     nil,
		"replica lioneye's paws":             nil,
		"moonbender's wing":                  nil,
		"ngamahu's flame":                    nil,
		"cameria's avarice":                  nil,
		"starcaller":                         nil,
		"uul-netol's embrace":                nil,
		"rigwald's crest":                    nil,
		"jorrhast's blacksteel":              nil,
		"ashcaller":                          nil,
		"arakaali's fang":                    nil,
		"sporeguard":                         nil,
		"mark of the elder":                  nil,
		"mark of the shaper":                 nil,
		"poet's pen":                         nil,
		"maloney's mechanism":                nil,
		"asenath's chant":                    nil,
		"vixen's entrapment":                 vixensEntrapmentConfig,
		"flames of judgement":                nil,
		"storm of judgement":                 nil,
		"trigger craft":                      triggerCraftConfig,
		"kitava's thirst":                    nil,
		"foulborn kitava's thirst":           nil,
		"mjolner":                            nil,
		"wing of the wyvern":                 nil,
		"cospri's malice":                    cosprisMaliceConfig,
		"seven teachings":                    nil,
		"squirming terror":                   nil,
		"kinetic flux":                       nil,
		"cast on critical strike":            castOnCriticalStrikeConfig,
		"cast on melee kill":                 nil,
		"nova":                               novaConfig,
		"cast when damage taken":             castWhenDamageTakenConfig,
		"cast when stunned":                  nil,
		"cast on ward break":                 nil,
		"spellslinger":                       spellslingerConfig,
		"call to arms":                       nil,
		"automation":                         automationConfig,
		"autoexertion":                       autoexertionConfig,
		"seize the flesh":                    nil,
		"fissure":                            nil,
		"mark on hit":                        markOnHitConfig,
		"hextouch":                           hextouchConfig,
		"oskarm":                             nil,
		"tempest shield":                     tempestShieldConfig,
		"shattershard":                       nil,
		"battlemage's cry":                   battlemagesCryConfig,
		"arcanist brand":                     arcanistBrandConfig,
		"cast on death":                      nil,
		"combust":                            combustConfig,
		"prismatic burst":                    nil,
		"voidstorm":                          voidstormConfig,
		"shockwave":                          shockwaveConfig,
		"void shockwave":                     shockwaveConfig,
		"falling crystal":                    nil,
		"call the pyre":                      callThePyreConfig,
		"manaforged arrows":                  manaforgedArrowsConfig,
		"doom blast":                         doomBlastConfig,
		"cast while channelling":             cwcConfig,
		"focus":                              nil,
		"snipe":                              nil,
		"intuitive link":                     nil,
		"svalinn cast on block":              castOnBlockConfig,
		"festering resentment cast on block": castOnBlockConfig,
		"hex on trap":                        nil,
		"supporttriggerfirespellonhit":       nil,
		"ghostly artillery":                  nil,
		"replica gifts from above":           nil,
		"bursting toad":                      burstingToadConfig,
		"TriggeredMoltenStrike":              triggeredMoltenStrikeConfig,
		"FieryImpactHeistMaceImplicit":       nil,
	}
}

// castOnCriticalStrikeConfig ports the "cast on critical strike" entry (L1162).
func castOnCriticalStrikeConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Attack] && env.slotMatch(skill)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return truthy(skill.SkillData["triggeredByCoc"]) && env.slotMatch(skill)
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
	if v, ok := env.Minion.MainSkill.SkillData["cooldown"]; ok && truthy(v) {
		n := anyNum(v)
		sim.cd = &n
	}
	return &triggerConfig{
		triggerName:     "Summon Holy Relic",
		actor:           env.minionPA,
		triggeredSkills: []*simSkill{sim},
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Attack]
		},
	}
}

// castWhenDamageTakenConfig ports the "cast when damage taken" entry
// (L1186): the skill triggers off damage rather than off another skill, so
// it is its own source and the rate comes from the trigger cap.
func castWhenDamageTakenConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if !truthy(main.SkillData["triggeredByDamageTaken"]) {
		return nil
	}
	thresholdMod := Mod(main.SkillModList, nil, "CWDTThreshold")
	env.Player.Output["CWDTThreshold"] = anyNum(main.SkillData["triggeredByDamageTaken"]) * thresholdMod
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
	return skill.SkillTypes[modparser.SkillType.Melee] || skill.SkillTypes[modparser.SkillType.Attack]
}

// doomBlastConfig ports the "doom blast" entry (L1394): the hexes that are
// removed drive the blast, so the source is the curse cast rate and each
// overlapping curse is another blast.
func doomBlastConfig(env *Env, actor *performActor) *triggerConfig {
	if str(env.ConfigInput["doomBlastSource"]) == "replacement" {
		env.ModDB.AddMod(newMod("UsesCurseOverlaps", "FLAG", true, "Config"))
	}
	env.PlayerMainSkill.SkillData["ignoresTickRate"] = true
	cfg := &triggerConfig{
		useCastRate:       true,
		customTriggerName: "Doom Blast triggering Hex: ",
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Hex] && env.slotMatch(skill)
		},
	}
	// `#Tabulate(...) > 0 and m_max(Sum(...), 1)`: false, not nil, when no
	// mod grants overlaps.
	if len(env.ModDB.Tabulate("BASE", nil, "Multiplier:CurseOverlaps")) > 0 {
		n := math.Max(env.ModDB.Sum("BASE", nil, "Multiplier:CurseOverlaps"), 1)
		cfg.overlaps = &n
	}
	return cfg
}

// voidstormConfig ports the "voidstorm" entry (L1369).
func voidstormConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Attack] &&
				skill.SkillTypes[modparser.SkillType.Rain] && env.slotMatch(skill)
		},
	}
}

// shockwaveConfig ports the "shockwave" entry (L1372); "void shockwave" and
// "falling crystal" are the same condition.
func shockwaveConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{triggerSkillCond: meleeAndSlot}
}

func meleeAndSlot(env *Env, skill *ActiveSkill) bool {
	return skill.SkillTypes[modparser.SkillType.Melee] && env.slotMatch(skill)
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
		if truthy(skill.SkillData["hextoadTriggerInterval"]) {
			triggerInterval = math.Min(triggerInterval, anyNum(skill.SkillData["hextoadTriggerInterval"]))
		}
	}
	if math.IsInf(triggerInterval, 1) {
		return nil
	}
	env.PlayerMainSkill.SkillFlags["globalTrigger"] = true
	env.PlayerMainSkill.SkillData["triggerRateCapOverride"] = 1 / triggerInterval
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
		return skill.SkillTypes[modparser.SkillType.Attack]
	}}
}

// hextouchConfig ports the "hextouch" entry (L1289): the curse lands at the
// rate of the attack that applies it, and that rate is already final.
func hextouchConfig(env *Env, actor *performActor) *triggerConfig {
	env.PlayerMainSkill.SkillData["sourceRateIsFinal"] = true
	return &triggerConfig{triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
		return skill.SkillTypes[modparser.SkillType.Attack] && env.slotMatch(skill)
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
	main.SkillData["sourceRateIsFinal"] = true
	return &triggerConfig{
		triggerName:  "Spellslinger",
		triggerOnUse: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			isWandProjectileAttack := skill.SkillTypes[modparser.SkillType.Attack] &&
				skill.SkillTypes[modparser.SkillType.Projectile] && cfgHasFlag(skill.SkillCfg, modparser.ModFlag.Wand)
			return isWandProjectileAttack && !truthy(skill.SkillData["triggeredBySpellSlinger"])
		},
	}
}

// manaforgedArrowsConfig ports the "manaforged arrows" entry (L1389).
func manaforgedArrowsConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerOnUse: true,
		triggerName:  "Manaforged Arrows",
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Attack] && cfgHasFlag(skill.SkillCfg, modparser.ModFlag.Bow)
		},
	}
}

// combustConfig ports the "combust" entry (L1358).
func combustConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Melee]
		},
		comparer: func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
			// Skills with no uptime ratio are not exerted by infernal cry so
			// should not be considered.
			uptimeRatio := env.GlobalCache[uuid].out("InfernalUpTimeRatio")
			return defaultComparer(env, uuid, source, triggerRate) && truthy(uptimeRatio)
		},
	}
}

// cfgHasFlag is `band(cfg.flags, ModFlag.X) > 0`.
func cfgHasFlag(cfg *modstore.Cfg, flag int64) bool {
	return cfg != nil && cfg.Flags != nil && *cfg.Flags&flag != 0
}

// vixensEntrapmentConfig ports the "vixen's entrapment" entry (L1051).
func vixensEntrapmentConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		useCastRate: true,
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Hex]
		},
	}
}

// cosprisMaliceConfig ports the "cospri's malice" entry (L1136).
func cosprisMaliceConfig(env *Env, actor *performActor) *triggerConfig {
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Melee] &&
				cfgHasFlag(skill.SkillCfg, modparser.ModFlag.Sword|modparser.ModFlag.Weapon1H)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return truthy(skill.SkillData["triggeredByCospris"]) && sameSocketSlot(env.PlayerMainSkill, skill)
		},
	}
}

// sameSocketSlot is `mainSkill.socketGroup.slot == skill.socketGroup.slot`.
func sameSocketSlot(a, b *ActiveSkill) bool {
	slot := func(s *ActiveSkill) (string, bool) {
		if s.SocketGroup == nil {
			return "", false
		}
		v, ok := s.SocketGroup.KV["slot"].(string)
		return v, ok
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
	main.SkillData["sourceRateIsFinal"] = true
	for _, skill := range env.PlayerActiveSkills {
		if skill.ActiveEffect.GrantedEffect.Name == "Arcanist Brand" {
			main.TriggeredBy.MainSkill = skill
			break
		}
	}
	brand := main.TriggeredBy.MainSkill
	activationFreqInc := (100 + brand.SkillModList.Sum("INC", brand.SkillCfg, "Speed", "BrandActivationFrequency")) / 100
	activationFreqMore := brand.SkillModList.More(brand.SkillCfg, "BrandActivationFrequency")
	main.TriggeredBy.ActivationFreqInc = activationFreqInc
	main.TriggeredBy.ActivationFreqMore = activationFreqMore
	main.TriggeredBy.AttachedBrandCount = anyNum(brand.SkillData["attachedBrandCount"])
	main.TriggeredBy.IgnoresTickRate = true
	trigRate := anyNum(brand.SkillData["repeatFrequency"]) * activationFreqInc * activationFreqMore * main.TriggeredBy.AttachedBrandCount
	return &triggerConfig{
		trigRate: &trigRate,
		source:   brand,
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return truthy(skill.SkillData["triggeredByBrand"]) && env.slotMatch(skill)
		},
	}
}

// triggerCraftConfig ports the "trigger craft" entry (L1069): the crafted
// "trigger a socketed spell" mod. The config finds the source itself so it
// can special-case totem-like sources onto their placement rate.
func triggerCraftConfig(env *Env, actor *performActor) *triggerConfig {
	main := env.PlayerMainSkill
	if !truthy(main.SkillData["triggeredByCraft"]) {
		return nil
	}
	var trigRate *float64
	var source *ActiveSkill
	var uuid string
	useCastRate := false
	var triggeredSkills []*simSkill
	for _, skill := range env.PlayerActiveSkills {
		if (skill.SkillTypes[modparser.SkillType.Damage] || skill.SkillTypes[modparser.SkillType.Attack] || skill.SkillTypes[modparser.SkillType.Spell]) &&
			!skill.SkillFlags["aura"] && skill != main && !truthy(skill.SkillData["triggeredByCraft"]) &&
			!env.geFromItem(skill.ActiveEffect.GrantedEffect) && !isTriggered(skill) {
			source, trigRate, uuid = env.findTriggerSkill(skill, source, trigRate, nil)
			if (skill.SkillFlags["totem"] || skill.SkillFlags["golem"] || skill.SkillFlags["banner"] || skill.SkillFlags["ballista"]) &&
				skill.ActiveEffect.GrantedEffect.CastTime != nil {
				// A totem-like source triggers per placement, not per hit.
				rate := 1 / *skill.ActiveEffect.GrantedEffect.CastTime
				if skill.ActiveEffect.GrantedEffect.Levels != nil {
					cd := 0.0
					if lvl := skill.ActiveEffect.GrantedEffect.Levels[skill.ActiveEffect.Level]; lvl != nil {
						cd = lvl.Extra["cooldown"]
					}
					rate = 1 / (*skill.ActiveEffect.GrantedEffect.CastTime + cd)
				}
				trigRate = &rate
				useCastRate = true
			}
		}
		if truthy(skill.SkillData["triggeredByCraft"]) && sameSocketSlot(main, skill) {
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
	main.SkillData["sourceRateIsFinal"] = true
	main.SkillData["ignoresTickRate"] = true
	return &triggerConfig{triggerOnUse: true, useCastRate: true, source: main}
}

// battlemagesCryConfig ports the "battlemage's cry" entry (L1321).
func battlemagesCryConfig(env *Env, actor *performActor) *triggerConfig {
	if env.PlayerMainSkill.ActiveEffect.GrantedEffect.Name == "Battlemage's Cry" {
		return nil
	}
	return &triggerConfig{
		triggerSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return skill.SkillTypes[modparser.SkillType.Melee]
		},
		comparer: func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
			// Skills with no uptime ratio are not exerted by battlemage so
			// should not be considered.
			uptimeRatio := env.GlobalCache[uuid].out("BattlemageUpTimeRatio")
			return defaultComparer(env, uuid, source, triggerRate) && truthy(uptimeRatio)
		},
		triggeredSkillCond: func(env *Env, skill *ActiveSkill) bool {
			return truthy(skill.SkillData["triggeredByBattleMageCry"]) && env.slotMatch(skill)
		},
	}
}
