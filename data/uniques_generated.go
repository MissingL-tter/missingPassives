// Port of Data/Uniques/Special/Generated.lua's load-time half: the uniques
// built programmatically from the mod pools (data.uniques.generated). The
// tree-dependent half (buildTreeDependentUniques: Forbidden Flame/Flesh,
// Skin of the Lords, Impossible Escape) needs tree data and lands with the
// tree-data module; the archive comparison excludes those four items.

package data

import (
	"regexp"
	"sort"
	"strings"
)

var (
	reDigitH      = regexp.MustCompile(`[0-9]h`)
	reLowerUpper  = regexp.MustCompile(`([a-z])([A-Z])`)
	reDigit       = regexp.MustCompile(`[0-9]`)
	reCapsDigits  = regexp.MustCompile(`[A-Z0-9]`)
	rePurityHead  = regexp.MustCompile(`^[PurityOf ]*[A-Z][a-z]+`)
	reFor4Seconds = regexp.MustCompile(`[fF]or 4 ?[Ss]econds On Hit`)
	reTypeHead    = regexp.MustCompile(`^([0-9]+)([A-Za-z]+)$`)
	reParenGroup  = regexp.MustCompile(`\(.*\)`)

	rePctNumber   = regexp.MustCompile(`[+\-]?[0-9.]*[0-9]+%`)
	reAnyNumber   = regexp.MustCompile(`[0-9.]*[0-9]+`)
	reRangePct    = regexp.MustCompile(`\(-?[0-9.]+--?[0-9.]+\)%`)
	reRangeHigher = regexp.MustCompile(`(\(-?[0-9.]+-)(-?[0-9.]+)\)`)
	reAddedRange  = regexp.MustCompile(`\([0-9]+-[0-9]+\) to \([0-9]+-[0-9]+\)`)
	reAddedParts  = regexp.MustCompile(`(\([0-9]+-)([0-9]+)(\) to \([0-9]+-)([0-9]+)\)`)
)

func parseVeiledModName(s string) string {
	s = strings.ReplaceAll(s, "JunMasterVeiled", "")
	s = strings.ReplaceAll(s, "Local", "")
	s = strings.ReplaceAll(s, "Display", "")
	s = strings.ReplaceAll(s, "Crafted", "")
	s = reDigitH.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "_", "")
	s = reLowerUpper.ReplaceAllString(s, "$1 $2")
	s = reDigit.ReplaceAllString(s, " $0 ")
	return s
}

var abbreviator = strings.NewReplacer(
	"Increased", "Inc", "Reduced", "Red.", "Critical", "Crit",
	"Physical", "Phys", "Elemental", "Ele", "Multiplier", "Mult",
	"EnergyShield", "ES",
)

func abbreviateModId(s string) string { return abbreviator.Replace(s) }

// spaceCaps is gsub("[%u%d]", " %1").
func spaceCaps(s string) string { return reCapsDigits.ReplaceAllString(s, " $0") }

type veiledMod struct {
	name  string
	lines []string
}

func isIn(list []string, v string) bool {
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}

func getVeiledMods(pool, baseType, specificType1, specificType2 string) []veiledMod {
	poolKey := ""
	if pool == "catarina" {
		poolKey = "catarina_veiled_prefix"
	}
	var out []veiledMod
	for id, mod := range VeiledMods {
		findWeight := func(key string) (float64, bool) {
			if key == "" {
				return 0, false
			}
			for i, k := range mod.WeightKey {
				if k == key {
					return mod.WeightVal[i], true
				}
			}
			return 0, false
		}
		t1, has1 := findWeight(specificType1)
		t2, has2 := findWeight(specificType2)
		base, hasBase := findWeight(baseType)
		poolW, hasPool := findWeight(poolKey)
		active := (has1 && t1 > 0) || (has2 && t2 > 0) ||
			(!has1 && !has2 && hasBase && base > 0) ||
			(!has1 && !has2 && !hasBase && hasPool && poolW > 0)
		if !active {
			continue
		}
		name := "(" + mod.Type + ") " + parseVeiledModName(id)
		keep := false
		switch pool {
		case "base":
			keep = mod.Affix == "Chosen" || mod.Affix == "of the Order"
		case "catarina":
			keep = mod.Affix == "Catarina's" || mod.Affix == "Chosen" || mod.Affix == "of the Order"
		case "all":
			keep = true
		}
		if keep {
			out = append(out, veiledMod{name: name, lines: mod.Lines})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].name < out[b].name })
	return out
}

func getVeiledModsByName(modNames []string) []veiledMod {
	var out []veiledMod
	for id, mod := range VeiledMods {
		plain := parseVeiledModName(id)
		if isIn(modNames, plain) || isIn(modNames, id) {
			out = append(out, veiledMod{name: "(" + mod.Type + ") " + plain, lines: mod.Lines})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].name < out[b].name })
	return out
}

func buildGeneratedUniques() {
	add := func(lines []string) {
		Uniques["generated"] = append(Uniques["generated"], strings.Join(lines, "\n"))
	}
	Uniques["generated"] = []string{}

	// ------------------------------------------------------- Paradoxica --
	paradoxicaMods := getVeiledMods("base", "weapon", "one_hand_weapon", "")
	paradoxica := []string{
		"Paradoxica",
		"Vaal Rapier",
		"League: Betrayal",
		"Source: Drops from unique{Intervention Leaders} in normal{Safehouses}",
		"Has Alt Variant: true",
		"Selected Variant: 4",
		"Selected Alt Variant: 16",
	}
	for i, mod := range paradoxicaMods {
		if mod.name == "(Suffix) Double Damage Chance" {
			paradoxicaMods = append(paradoxicaMods[:i], paradoxicaMods[i+1:]...)
			break
		}
	}
	for _, mod := range paradoxicaMods {
		paradoxica = append(paradoxica, "Variant: "+mod.name)
	}
	paradoxica = append(paradoxica, "Requires Level 66, 212 Dex", "Implicits: 1", "+25% to Global Critical Strike Multiplier")
	for i, mod := range paradoxicaMods {
		for _, line := range mod.lines {
			paradoxica = append(paradoxica, "{variant:"+itoa(i+1)+"}"+line)
		}
	}
	paradoxica = append(paradoxica, "Attacks with this Weapon deal Double Damage")
	add(paradoxica)

	// -------------------------------------------------- Cane of Kulemak --
	caneMods := getVeiledMods("catarina", "weapon", "staff", "two_hand_weapon")
	cane := []string{
		"Cane of Kulemak",
		"Serpentine Staff",
		"Source: Drops from unique{Catarina, Master of Undeath}",
		"Has Alt Variant: true",
		"Has Alt Variant Two: true",
		"Has Alt Variant Three: true",
		"Selected Variant: 1",
		"Selected Alt Variant: 3",
		"Selected Alt Variant Two: 25",
		"Selected Alt Variant Three: 26",
	}
	for _, mod := range caneMods {
		cane = append(cane, "Variant: "+mod.name)
	}
	cane = append(cane, "Requires Level 68, 85 Str, 85 Int", "Implicits: 1")
	cane = append(cane, strings.Join(ItemMods["ItemExclusive"]["StaffBlockPercentImplicitStaff2"].Lines, ""))
	cane = append(cane, strings.Join(ItemMods["ItemExclusive"]["LocalVeiledModEffectUnique__1"].Lines, ""))
	for i, mod := range caneMods {
		for _, line := range mod.lines {
			cane = append(cane, "{variant:"+itoa(i+1)+"}"+line)
		}
	}
	add(cane)

	// --------------------------------------------- Replica Paradoxica --
	replicaMods := getVeiledMods("all", "weapon", "one_hand_weapon", "")
	replica := []string{
		"Replica Paradoxica",
		"Vaal Rapier",
		"League: Heist",
		"Source: Steal from a unique{Curio Display} during a Grand Heist",
		"Has Alt Variant: true",
		"Has Alt Variant Two: true",
		"Has Alt Variant Three: true",
		"Has Alt Variant Four: true",
		"Has Alt Variant Five: true",
		"Selected Variant: 1",
		"Selected Alt Variant: 2",
		"Selected Alt Variant Two: 3",
		"Selected Alt Variant Three: 25",
		"Selected Alt Variant Four: 27",
		"Selected Alt Variant Five: 34",
	}
	for _, mod := range replicaMods {
		replica = append(replica, "Variant: "+mod.name)
	}
	replica = append(replica, "Requires Level 66, 212 Dex", "Implicits: 1", "+25% to Global Critical Strike Multiplier")
	for i, mod := range replicaMods {
		for _, line := range mod.lines {
			replica = append(replica, "{variant:"+itoa(i+1)+"}"+line)
		}
	}
	add(replica)

	// ----------------------------------------------- The Queen's Hunger --
	queensMods := getVeiledModsByName([]string{
		"JunMasterVeiledLocalIncreasedEnergyShieldAndLifeHigh",
		"JunMasterVeiledPhysicalDamageReductionRatingDuringSoulGainPrevention",
		"JunMasterVeiledPercentageLifeAndMana",
		"JunMasterVeiledBlockPercent",
		"JunMasterVeiledAvoidStunAndElementalStatusAilments",
		"JunMasterVeiledSpellBlockPercent____",
		"JunMasterVeiledOfferingEffect",
		"JunMasterVeiledLifeRegenerationRatePercentageIfCorpseConsumedRecently",
		"JunMasterVeiledManaRegenerationRatePercentageIfCorpseConsumedRecently",
		"JunMasterVeiledEnergyShieldRegenerationRatePercentageIfCorpseConsumedRecently",
		"JunMasterVeiledAllow2Offerings",
		"JunMasterVeiledOfferingDuration",
		"JunMasterVeiledStrengthAndDexterity",
		"JunMasterVeiledDexterityAndIntelligence",
		"JunMasterVeiledStrengthAndIntelligence",
		"JunMasterVeiledAvoidElementalDamageChanceDuringSoulGainPrevention",
		"JunMasterVeiledEnergyShieldRegenerationRatePerMinuteIfRareOrUniqueEnemyNearby",
		"JunMasterVeiledLifeRegenerationPerEvasionDuringFocus",
		"JunMasterVeiledRestoreManaAndEnergyShieldOnFocus",
		"JunMasterVeiledFortifyEffectWhileFocused_",
		"JunMasterVeiledDamageRemovedFromManaBeforeLifeWhileFocused",
		"JunMasterVeiledFireAndChaosDamageResistance",
		"JunMasterVeiledLightningAndChaosDamageResistance",
		"JunMasterVeiledColdAndChaosDamageResistance",
		"JunMasterVeiledStrengthAndAvoidIgnite",
		"JunMasterVeiledDexterityAndAvoidFreeze",
		"JunMasterVeiledIntelligenceAndAvoidShock",
	})
	queens := []string{
		"The Queen's Hunger",
		"Vaal Regalia",
		"League: Betrayal",
		"Source: Drops from unique{Catarina, Master of Undeath}",
		"Has Alt Variant: true",
		"Selected Variant: 1",
		"Selected Alt Variant: 24",
	}
	for _, mod := range queensMods {
		queens = append(queens, "Variant: "+mod.name)
	}
	queens = append(queens,
		"Requires Level 68, 194 Int",
		"Trigger Level 20 Bone Offering, Flesh Offering or Spirit Offering every 5 seconds",
		"Offering Skills Triggered this way also affect you",
		"(5-10)% increased Cast Speed",
		"(100-130)% increased Energy Shield",
		"(6-10)% increased maximum Life",
	)
	for i, mod := range queensMods {
		for _, line := range mod.lines {
			queens = append(queens, "{variant:"+itoa(i+1)+"}{crafted}"+line)
		}
	}
	add(queens)

	// ---------------------------------------------------- Megalomaniac --
	megalomaniac := []string{
		"Megalomaniac",
		"Medium Cluster Jewel",
		"League: Delirium",
		"Source: Drops from the Simulacrum Encounter",
		"Has Alt Variant: true",
		"Has Alt Variant Two: true",
	}
	var notables []string
	for name := range ClusterJewels.NotableSortOrder {
		notables = append(notables, name)
	}
	sort.Strings(notables)
	for _, name := range notables {
		megalomaniac = append(megalomaniac, "Variant: "+name)
	}
	megalomaniac = append(megalomaniac, "Adds 4 Passive Skills", "Added Small Passive Skills grant Nothing")
	for i, name := range notables {
		megalomaniac = append(megalomaniac, "{variant:"+itoa(i+1)+"}1 Added Passive Skill is "+name)
	}
	add(megalomaniac)

	// -------------------------------------------------- Forbidden Shako --
	shako := []string{
		"Forbidden Shako",
		"Great Crown",
		"League: Harvest",
		"Source: Drops from unique{Oshabi, Avatar of the Grove}",
		"Requires Level 68, 59 Str, 59 Int",
		"Has Alt Variant: true",
	}
	replicaShako := []string{
		"Replica Forbidden Shako",
		"Great Crown",
		"League: Heist",
		"Source: Steal from a unique{Curio Display} during a Grand Heist",
		"Requires Level 68, 59 Str, 59 Int",
		"Has Alt Variant: true",
	}
	excludedGems := []string{"Block Chance Reduction", "Empower", "Enhance", "Enlighten", "Item Quantity"}
	var supportGems []string
	for _, gem := range Gems {
		ge := gem.GrantedEffect
		if ge.Support && ge.PlusVersionOf == nil && !isIn(excludedGems, ge.Name) {
			supportGems = append(supportGems, ge.Name)
		}
	}
	sort.Strings(supportGems)
	for _, name := range supportGems {
		shako = append(shako, "Variant: "+name+" (Low Level)", "Variant: "+name+" (High Level)")
		replicaShako = append(replicaShako, "Variant: "+name+" (Low Level)", "Variant: "+name+" (High Level)")
	}
	for i, name := range supportGems {
		lo, hi := itoa(2*i+1), itoa(2*i+2)
		shako = append(shako,
			"{variant:"+lo+"}Socketed Gems are Supported by Level (1-10) "+name,
			"{variant:"+hi+"}Socketed Gems are Supported by Level (25-35) "+name)
		replicaShako = append(replicaShako,
			"{variant:"+lo+"}Socketed Gems are Supported by Level (1-10) "+name,
			"{variant:"+hi+"}Socketed Gems are Supported by Level (25-35) "+name)
	}
	shako = append(shako, "+(25-30) to all Attributes")
	replicaShako = append(replicaShako, "+(25-30) to all Attributes")
	add(shako)
	add(replicaShako)

	// ----------------------------------------------- Precursor's Emblem --
	add(buildPrecursorsEmblem())

	// -------------------------------------------- The Balance of Terror --
	add(buildBalanceOfTerror())

	// Watcher's Eye / Sublime Vision / Vorana's March / Bound by Destiny.
	we, sv, vm, bbd := buildEyeFamily()
	add(we)
	add(sv)
	add(vm)
	add(bbd)

	add(buildThatWhichWasTaken())
	add(buildReplicaDragonfangsFlight())
}
