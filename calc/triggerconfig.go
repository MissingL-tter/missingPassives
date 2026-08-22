// CalcTriggers.lua L908-1585: the configTable dispatch. Every key the
// reference defines is present so the lookup order is faithful; the entries
// no corpus build reaches are nil and panic on match, rather than silently
// behaving like "no trigger config at all".
package calc

import "github.com/MissingL-tter/missingPassives/modparser"

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
	"cast when damage taken":             nil,
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
	"tempest shield":                     nil,
	"shattershard":                       nil,
	"battlemage's cry":                   nil,
	"arcanist brand":                     nil,
	"cast on death":                      nil,
	"combust":                            nil,
	"prismatic burst":                    nil,
	"voidstorm":                          nil,
	"shockwave":                          nil,
	"void shockwave":                     nil,
	"falling crystal":                    nil,
	"call the pyre":                      nil,
	"manaforged arrows":                  nil,
	"doom blast":                         nil,
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
	"bursting toad":                      nil,
	"TriggeredMoltenStrike":              nil,
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
