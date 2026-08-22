// CalcTriggers.lua L908-1585: the configTable dispatch. Every key the
// reference defines is present so the lookup order is faithful; the entries
// no corpus build reaches are nil and panic on match, rather than silently
// behaving like "no trigger config at all".
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// triggerConfigBuilder returns nil when the entry inspects the env and
// decides it does not apply (several reference entries do).
type triggerConfigBuilder func(env *Env, actor *performActor) *triggerConfig

var triggerConfigTable = map[string]triggerConfigBuilder{
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
	"vixen's entrapment":                 nil,
	"flames of judgement":                nil,
	"storm of judgement":                 nil,
	"trigger craft":                      nil,
	"kitava's thirst":                    nil,
	"foulborn kitava's thirst":           nil,
	"mjolner":                            nil,
	"wing of the wyvern":                 nil,
	"cospri's malice":                    nil,
	"seven teachings":                    nil,
	"squirming terror":                   nil,
	"kinetic flux":                       nil,
	"cast on critical strike":            castOnCriticalStrikeConfig,
	"cast on melee kill":                 nil,
	"nova":                               novaConfig,
	"cast when damage taken":             castWhenDamageTakenConfig,
	"cast when stunned":                  nil,
	"cast on ward break":                 nil,
	"spellslinger":                       nil,
	"call to arms":                       nil,
	"automation":                         nil,
	"autoexertion":                       nil,
	"seize the flesh":                    nil,
	"fissure":                            nil,
	"mark on hit":                        nil,
	"hextouch":                           nil,
	"oskarm":                             nil,
	"tempest shield":                     tempestShieldConfig,
	"shattershard":                       nil,
	"battlemage's cry":                   nil,
	"arcanist brand":                     nil,
	"cast on death":                      nil,
	"combust":                            nil,
	"prismatic burst":                    nil,
	"voidstorm":                          voidstormConfig,
	"shockwave":                          shockwaveConfig,
	"void shockwave":                     nil,
	"falling crystal":                    nil,
	"call the pyre":                      callThePyreConfig,
	"manaforged arrows":                  nil,
	"doom blast":                         doomBlastConfig,
	"cast while channelling":             nil,
	"focus":                              nil,
	"snipe":                              nil,
	"intuitive link":                     nil,
	"svalinn cast on block":              nil,
	"festering resentment cast on block": nil,
	"hex on trap":                        nil,
	"supporttriggerfirespellonhit":       nil,
	"ghostly artillery":                  nil,
	"replica gifts from above":           nil,
	"bursting toad":                      burstingToadConfig,
	"TriggeredMoltenStrike":              triggeredMoltenStrikeConfig,
	"FieryImpactHeistMaceImplicit":       nil,
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
