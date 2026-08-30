package modparser

// modForm is the shape a modifier line takes (the formList value), which
// picks the value/type handling in parseMod.
type modForm uint8

const (
	formInc modForm = iota + 1
	formRed
	formMore
	formLess
	formBase
	formGain
	formLose
	formGrants
	formGrantsGlobal
	formRemoves
	formChance
	formFlag
	formTotalCost
	formBaseCost
	formPen
	formRegenFlat
	formRegenPercent
	formDegenFlat
	formDegenPercent
	formDegen
	formDmg
	formDmgAttacks
	formDmgSpells
	formDmgBoth
	formOverride
	formDoubled
)

var modFormNames = [...]string{"", "INC", "RED", "MORE", "LESS", "BASE", "GAIN", "LOSE", "GRANTS", "GRANTS_GLOBAL",
	"REMOVES", "CHANCE", "FLAG", "TOTALCOST", "BASECOST", "PEN", "REGENFLAT", "REGENPERCENT", "DEGENFLAT",
	"DEGENPERCENT", "DEGEN", "DMG", "DMGATTACKS", "DMGSPELLS", "DMGBOTH", "OVERRIDE", "DOUBLED"}

// String is the reference's formList value text.
func (f modForm) String() string { return modFormNames[f] }

// List of modifier forms — ModParser.lua:72.
var formList = map[string]modForm{
	`^([0-9]+)% increased`:                                               formInc,
	`^([0-9]+)% faster`:                                                  formInc,
	`^([0-9]+)% reduced`:                                                 formRed,
	`^([0-9]+)% slower`:                                                  formRed,
	`^([0-9]+)% more`:                                                    formMore,
	`^([0-9]+)% less`:                                                    formLess,
	`^([+\-][0-9.]+)%?`:                                                  formBase,
	`^([+\-][0-9.]+)%? to`:                                               formBase,
	`^([+\-]?[0-9.]+)%? of`:                                              formBase,
	`^([+\-][0-9.]+)%? base`:                                             formBase,
	`^([+\-]?[0-9.]+)%? additional`:                                      formBase,
	`([0-9]+) additional hits?`:                                          formBase,
	`([0-9]+) additional times?`:                                         formBase,
	`^throw up to ([0-9]+)`:                                              formBase,
	`^you gain ([0-9.]+)`:                                                formGain,
	`^gains? ([0-9.]+)% of`:                                              formGain,
	`^gain ([0-9.]+)`:                                                    formGain,
	`^gain \+([0-9]+)% to`:                                               formGain,
	`^you lose ([0-9.]+)`:                                                formLose,
	`^loses? ([0-9.]+)% of`:                                              formLose,
	`^lose ([0-9.]+)`:                                                    formLose,
	`^lose \+([0-9]+)% to`:                                               formLose,
	`^grants ([0-9.]+)`:                                                  formGrants,  // local
	`^removes? ([0-9.]+) ?o?f? ?y?o?u?r?`:                                formRemoves, // local
	`^([0-9]+)`:                                                          formBase,
	`^([+\-]?[0-9]+)% chance`:                                            formChance,
	`^([+\-]?[0-9]+)% chance to gain `:                                   formFlag,
	`^([+\-]?[0-9]+)% additional chance`:                                 formChance,
	`costs? ([+\-]?[0-9]+)`:                                              formTotalCost,
	`skills cost ([+\-]?[0-9]+)`:                                         formBaseCost,
	`penetrates? ([0-9]+)%`:                                              formPen,
	`penetrates ([0-9]+)% of`:                                            formPen,
	`penetrates ([0-9]+)% of enemy`:                                      formPen,
	`^([0-9.]+) (.+) regenerated per second`:                             formRegenFlat,
	`^([0-9.]+)% (.+) regenerated per second`:                            formRegenPercent,
	`^([0-9.]+)% of (.+) regenerated per second`:                         formRegenPercent,
	`^regenerate ([0-9.]+) (.*?) per second`:                             formRegenFlat,
	`^regenerate ([0-9.]+)% (.*?) per second`:                            formRegenPercent,
	`^regenerate ([0-9.]+)% of (.*?) per second`:                         formRegenPercent,
	`^regenerate ([0-9.]+)% of your (.*?) per second`:                    formRegenPercent,
	`^you regenerate ([0-9.]+)% of (.*?) per second`:                     formRegenPercent,
	`^([0-9.]+) (.+) lost per second`:                                    formDegenFlat,
	`^([0-9.]+)% (.+) lost per second`:                                   formDegenPercent,
	`^([0-9.]+)% of (.+) lost per second`:                                formDegenPercent,
	`^lose ([0-9.]+) (.*?) per second`:                                   formDegenFlat,
	`^lose ([0-9.]+)% (.*?) per second`:                                  formDegenPercent,
	`^lose ([0-9.]+)% of (.*?) per second`:                               formDegenPercent,
	`^lose ([0-9.]+)% of your (.*?) per second`:                          formDegenPercent,
	`^you lose ([0-9.]+)% of (.*?) per second`:                           formDegenPercent,
	`^([0-9.]+) ([a-zA-Z]+) damage taken per second`:                     formDegen,
	`^([0-9.]+) ([a-zA-Z]+) damage per second`:                           formDegen,
	`([0-9]+) to ([0-9]+) added ([a-zA-Z]+) damage`:                      formDmg,
	`([0-9]+)-([0-9]+) added ([a-zA-Z]+) damage`:                         formDmg,
	`([0-9]+) to ([0-9]+) additional ([a-zA-Z]+) damage`:                 formDmg,
	`([0-9]+)-([0-9]+) additional ([a-zA-Z]+) damage`:                    formDmg,
	`^([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage`:                           formDmg,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage`:                       formDmg,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) damage`:                          formDmg,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage to attacks`:            formDmgAttacks,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) damage to attacks`:               formDmgAttacks,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) attack damage`:                formDmgAttacks,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) attack damage`:                   formDmgAttacks,
	`([0-9]+) to ([0-9]+) added attack ([a-zA-Z]+) damage`:               formDmgAttacks,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage to spells`:             formDmgSpells,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) damage to spells`:                formDmgSpells,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) spell damage`:                 formDmgSpells,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) spell damage`:                    formDmgSpells,
	`([0-9]+) to ([0-9]+) added spell ([a-zA-Z]+) damage`:                formDmgSpells,
	`([0-9]+) to ([0-9]+) spell ([a-zA-Z]+) damage`:                      formDmgSpells,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage to attacks and spells`: formDmgBoth,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) damage to attacks and spells`:    formDmgBoth,
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage to spells and attacks`: formDmgBoth, // o_O
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) damage to spells and attacks`:    formDmgBoth, // o_O
	`adds ([0-9]+) to ([0-9]+) ([a-zA-Z]+) damage to hits`:               formDmgBoth,
	`adds ([0-9]+)-([0-9]+) ([a-zA-Z]+) damage to hits`:                  formDmgBoth,
	`^you have `:       formFlag,
	`^have `:           formFlag,
	`^you are `:        formFlag,
	`^are `:            formFlag,
	`^gain `:           formFlag,
	`^you gain `:       formFlag,
	`is (-?[0-9]+)%? `: formOverride,
	`is doubled`:       formDoubled,
	`doubles?`:         formDoubled,
	`causes? double`:   formDoubled,
}
