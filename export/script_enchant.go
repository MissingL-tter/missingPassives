// Port of .archive/src/Export/Scripts/enchant.lua. The "Craft" generation
// branch of doOtherEnchantment is dead in the reference (no caller passes a
// Craft key) and is not ported.

package export

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "enchant", Build: buildEnchant})
}

var enchantLab = map[int64]string{
	32: "NORMAL",
	53: "CRUEL",
	66: "MERCILESS",
	75: "ENDGAME",
	83: "DEDICATION",
}

var enchantSourceOrder = []string{"NORMAL", "CRUEL", "MERCILESS", "ENDGAME", "DEDICATION", "ENKINDLING", "INSTILLING", "HARVEST", "HEIST"}

// enchantSkillMap: Lua-pattern keys; all of them are valid Go regexes as
// written.
var enchantSkillMap = map[string]string{
	"Summone?d?RagingSpirit": "Summon Raging Spirit",
	"SpiritOffering":         "Spirit Offering",
	"Discharge":              "Discharge",
	"AncestorTotem[^S][^l]":  "Ancestral Protector",
	"AncestorTotemSlamMelee": "Ancestral Warchief",
	"AnimateGuardian":        "Animate Guardian",
	"AnimateWeapon":          "Animate Weapon",
	"BlinkArrow":             "Blink Arrow",
	"ConversionTrap":         "Conversion Trap",
	"MirrorArrow":            "Mirror Arrow",
	"Spectre":                "Raise Spectre",
	"Zombie":                 "Raise Zombie",
	"ChaosGolem":             "Summon Chaos Golem",
	"FlameGolem":             "Summon Flame Golem",
	"IceGolem":               "Summon Ice Golem",
	"LightningGolem":         "Summon Lightning Golem",
	"StoneGolem":             "Summon Stone Golem",
	"Skeleton":               "Summon Skeletons",
	"Bladefall":              "Bladefall",
	"BlastRain":              "Blast Rain",
	"ChargedAttack":          "Blade Flurry",
	"Desecrate":              "Desecrate",
	"DetonateDead":           "Detonate Dead",
	"DevouringTotem":         "Devouring Totem",
	"DominatingBlow":         "Dominating Blow",
	"FireBeam":               "Scorching Ray",
	"Firestorm":              "Firestorm",
	"FreezeMine":             "Freeze Mine",
	"EnchantmentFrenzy":      "Frenzy",
	"GroundSlam":             "Ground Slam",
	"HeavyStrike":            "Heavy Strike",
	"IceSpear":               "Ice Spear",
	"ImmortalCall":           "Immortal Call",
	"Incinerate":             "Incinerate",
	"KineticBlast":           "Kinetic Blast",
	"LightningArrow":         "Lightning Arrow",
	"ChargedDash":            "Charged Dash",
	"PhaseRun":               "Phase Run",
	"Puncture":               "Puncture",
	"RejuvinationTotem":      "Rejuvenation Totem",
	"ShockNova":              "Shock Nova",
	"SpectralThrow":          "Spectral Throw",
	"TectonicSlam":           "Tectonic Slam",
	"VolatileDead":           "Volatile Dead",
	"BoneLance":              "Unearth",
	"CorpseEruption":         "Cremation",
	"PowerSiphon":            "Power Siphon",
	"Smite":                  "Smite",
	"ConsecratedPath":        "Consecrated Path",
	"ScourgeArrow":           "Scourge Arrow",
	"HolyRelic":              "Summon Holy Relic",
	"HeraldOfAgony":          "Herald of Agony",
	"HeraldOfPurity":         "Herald of Purity",
	"Bane":                   "Bane",
	"DivineIre":              "Divine Ire",
	"PurifyingFlame":         "Purifying Flame",
	"Soulrend":               "Soulrend",
	"StormBurst":             "Storm Burst",
	"CarrionGolem":           "Summon Carrion Golem",
	"Steelskin":              "Steelskin",
	"[^d]Dash":               "Dash",
	"Bladestorm":             "Bladestorm",
	"Perforate":              "Perforate",
	"Frostblink":             "Frostblink",
	"ChainHook":              "Chain Hook",
	"Berserk":                "Berserk",
	"WitheringStep":          "Withering Step",
	"SnappingAdder":          "Venom Gyre",
	"PlagueBearer":           "Plague Bearer",
	"SummonSkitterbots":      "Summon Skitterbots",
	"ArtilleryBallista":      "Artillery Ballista",
	"ArcaneCloak":            "Arcane Cloak",
	"KineticBolt":            "Kinetic Bolt",
	"BladeBlast":             "Blade Blast",
	"RuneBlast":              "Stormbind",
	"Spellslinger":           "Spellslinger",
	"AncestralCry":           "Ancestral Cry",
	"EnduringCry":            "Enduring Cry",
	"SeismicCry":             "Seismic Cry",
	"Sunder":                 "Sunder",
	"Earthshatter":           "Earthshatter",
	"ArcanistBrand":          "Arcanist Brand",
	"BlazingSalvo":           "Blazing Salvo",
	"Anger":                  "Anger",
	"Clarity":                "Clarity",
	"Determination":          "Determination",
	"Discipline":             "Discipline",
	"Grace":                  "Grace",
	"Haste":                  "Haste",
	"Hatred":                 "Hatred",
	"Malevolence":            "Malevolence",
	"Precision":              "Precision",
	"Pride":                  "Pride",
	"Vitality":               "Vitality",
	"Wrath":                  "Wrath",
	"Zealotry":               "Zealotry",
	"PurityOfElements":       "Purity of Elements",
	"PurityOfFire":           "Purity of Fire",
	"PurityOfIce":            "Purity of Ice",
	"PurityOfLightning":      "Purity of Lightning",
	"MortarBarrageMine":      "Pyroclast Mine",
	"ColdProjectileMine":     "Icicle Mine",
	"LightningExplosionMine": "Stormblast Mine",
	"FleshAndStone":          "Flesh and Stone",
	"DreadBanner":            "Dread Banner",
	"WarBanner":              "War Banner",
	"FrostShield":            "Frost Shield",
	"VoidSphere":             "Void Sphere",
	"CracklingLance":         "Crackling Lance",
	"SigilOfPower":           "Sigil of Power",
	"Hexblast":               "Hexblast",
	"FlameWall":              "Flame Wall",
	"WaterSphere":            "Hydrosphere",
	"CorruptingFever":        "Corrupting Fever",
	"Bloodreap":              "Reap",
	"BladeTrap":              "Blade Trap",
	"EyeOfWinter":            "Eye of Winter",
	"StormRain":              "Storm Rain",
	"RageVortex":             "Rage Vortex",
	"ShieldCrush":            "Shield Crush",
	"SummonedReaper":         "Summon Reaper",
	"Boneshatter":            "Boneshatter",
	"SpectralHelix":          "Spectral Helix",
	"DefianceBanner":         "Defiance Banner",
	"EnergyBlade":            "Energy Blade",
	"TornadoShot":            "Tornado Shot",
	"Tornado":                "Tornado",
	"VolcanicFissure":        "Volcanic Fissure",
	"Table Charge":           "Shield Charge",
	"Flame Dash":             "Flame Dash",
}

func buildEnchant(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("stat_descriptions.txt")

	mods, err := x.Dat("Mods")
	if err != nil {
		return nil, err
	}
	activeSkills, err := x.Dat("ActiveSkills")
	if err != nil {
		return nil, err
	}

	doLabEnchantment := func(group string) (map[string][][]string, error) {
		byDiff := map[string][][]string{}
		for _, mod := range mods.RowsByInt("GenerationType", 10) {
			family := mod.Refs("Family")
			if len(family) > 0 && family[0].Str("Id") == group &&
				mod.Ints("SpawnWeights")[0] > 0 {
				stats, err := x.DescribeMod(mod)
				if err != nil {
					return nil, err
				}
				diff := enchantLab[mod.Int("Level")]
				byDiff[diff] = append(byDiff[diff], stats.Lines)
			}
		}
		return byDiff, nil
	}

	doOtherEnchantment := func(groupsList map[int64]map[string]string) (map[string][][]string, error) {
		byDiff := map[string][][]string{}
		gens := make([]int64, 0, len(groupsList))
		for g := range groupsList {
			gens = append(gens, g)
		}
		sort.Slice(gens, func(a, b int) bool { return gens[a] < gens[b] })
		for _, generation := range gens {
			for _, mod := range mods.RowsByInt("GenerationType", generation) {
				family := mod.Refs("Family")
				if len(family) > 0 {
					if diff, ok := groupsList[generation][family[0].Str("Id")]; ok {
						stats, err := x.DescribeMod(mod)
						if err != nil {
							return nil, err
						}
						byDiff[diff] = append(byDiff[diff], stats.Lines)
					}
				}
			}
		}
		return byDiff, nil
	}

	var d schema.Enchants
	if d.Boots, err = doLabEnchantment("ConditionalBuffEnchantment"); err != nil {
		return nil, err
	}
	if d.Gloves, err = doLabEnchantment("TriggerEnchantment"); err != nil {
		return nil, err
	}
	if d.Belt, err = doLabEnchantment("BuffEnchantment"); err != nil {
		return nil, err
	}
	// Harvest flask enchants stat descriptions don't read properly yet
	if d.Flask, err = doOtherEnchantment(map[int64]map[string]string{
		21: {"FlaskEnchantment": "ENKINDLING"},
		22: {"FlaskEnchantment": "INSTILLING"},
	}); err != nil {
		return nil, err
	}
	if d.Body, err = doOtherEnchantment(map[int64]map[string]string{
		3: {"AlternateArmourQuality": "HARVEST", "EnchantmentHeistArmour": "HEIST"},
	}); err != nil {
		return nil, err
	}
	if d.Weapon, err = doOtherEnchantment(map[int64]map[string]string{
		3: {"AlternateWeaponQuality": "HARVEST", "EnchantmentHeistWeapon": "HEIST"},
	}); err != nil {
		return nil, err
	}

	// Helmet skill enchantments.
	skillPatterns := make([]string, 0, len(enchantSkillMap))
	for p := range enchantSkillMap {
		skillPatterns = append(skillPatterns, p)
	}
	sort.Strings(skillPatterns)
	patternRe := map[string]*regexp.Regexp{}
	for _, p := range skillPatterns {
		patternRe[p] = regexp.MustCompile(p)
	}

	bySkill := map[string]map[string][][]string{}
	for _, mod := range mods.RowsByInt("GenerationType", 10) {
		family := mod.Refs("Family")
		if !(len(family) > 0 && family[0].Str("Id") == "SkillEnchantment" &&
			mod.Ints("SpawnWeights")[0] > 0) {
			continue
		}
		skill := ""
		// Each stat's searches can overwrite an earlier stat's finding, as in
		// the Lua (no outer break).
		findSkill := func(col string, stat *Row) {
			for _, as := range activeSkills.RowsByRef(col, stat) {
				// Archive parity: the Lua compares SkillTypes ROWS
				// against the number 39, which is always false; only the id
				// substring check can set isVaal.
				isVaal := strings.Contains(as.Str("Id"), "vaal")
				if dn := as.Str("DisplayName"); !isVaal && dn != "" {
					skill = dn
					return
				}
			}
		}
		for i := 1; i <= 6; i++ {
			stat := mod.Ref("Stat" + strconv.Itoa(i))
			if stat == nil {
				continue
			}
			findSkill("SkillSpecificStat", stat)
			findSkill("SecondarySkillSpecificStat", stat)
		}
		modId := mod.Str("Id")
		for _, p := range skillPatterns {
			if loc := patternRe[p].FindStringIndex(modId); loc != nil {
				// Lua string.find spans: j - i with 1-based inclusive ends.
				span := loc[1] - loc[0] - 1
				if len(skill) < span {
					skill = enchantSkillMap[p]
				}
			}
		}
		if mapped, ok := enchantSkillMap[skill]; ok {
			skill = mapped
		}

		stats, err := x.DescribeMod(mod)
		if err != nil {
			return nil, err
		}
		if len(stats.Lines) == 0 {
			continue // the Lua printfs the mod id
		}
		if bySkill[skill] == nil {
			bySkill[skill] = map[string][][]string{}
		}
		diff := enchantLab[mod.Int("Level")]
		bySkill[skill][diff] = append(bySkill[skill][diff], stats.Lines)
	}
	d.Helmet = bySkill
	return d, nil
}
