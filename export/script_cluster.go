// Port of .archive/src/Export/Scripts/cluster.lua.

package export

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "cluster", Build: buildCluster})
}

var reDdsEnd = regexp.MustCompile(`dds$`)

func buildCluster(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("passive_skill_stat_descriptions.txt")

	jewels, err := x.Dat("PassiveTreeExpansionJewels")
	if err != nil {
		return nil, err
	}
	expansionSkills, err := x.Dat("PassiveTreeExpansionSkills")
	if err != nil {
		return nil, err
	}
	specialSkills, err := x.Dat("PassiveTreeExpansionSpecialSkills")
	if err != nil {
		return nil, err
	}
	jewelSlots, err := x.Dat("PassiveJewelSlots")
	if err != nil {
		return nil, err
	}

	var cj schema.ClusterJewels
	for jewel := range jewels.Rows() {
		size := jewel.Ref("Size")
		j := schema.ClusterJewelSize{
			Name:            jewel.Ref("BaseItemType").Str("Name"),
			Size:            size.Str("Id"),
			SizeIndex:       size.ID,
			MinNodes:        jewel.Int("MinNodes"),
			MaxNodes:        jewel.Int("MaxNodes"),
			SmallIndicies:   jewel.Ints("SmallIndicies"),
			NotableIndicies: jewel.Ints("NotableIndicies"),
			SocketIndicies:  jewel.Ints("SocketIndicies"),
			TotalIndicies:   jewel.Int("TotalIndicies"),
		}
		for _, skill := range expansionSkills.GetRowList("JewelSize", size) {
			node := skill.Ref("Node")
			tagId := skill.Ref("Tag").Str("Id")
			s := schema.ClusterSkill{
				Id:   node.Str("Id"),
				Name: node.Str("Name"),
				Icon: reDdsEnd.ReplaceAllString(node.Str("Icon"), "png"),
				Tag:  tagId,
			}
			if strings.Contains(tagId, "old_do_not_use") {
				s.Name += " (Legacy)"
			}
			if mastery := skill.Ref("Mastery"); mastery != nil {
				icon := reDdsEnd.ReplaceAllString(mastery.Str("Icon"), "png")
				s.MasteryIcon = &icon
			}
			stats := map[string]*statVal{}
			for i, stat := range node.Refs("Stats") {
				v := float64(node.Int(fmt.Sprintf("Stat%d", i+1)))
				stats[stat.Str("Id")] = &statVal{min: v, max: v}
			}
			lines, err := x.DescribeStats(stats)
			if err != nil {
				return nil, err
			}
			s.Stats = lines.Lines
			j.Skills = append(j.Skills, s)
		}
		cj.Jewels = append(cj.Jewels, j)
	}

	for skill := range specialSkills.Rows() {
		node := skill.Ref("Node")
		if node.Bool("Notable") {
			cj.NotableSortOrder = append(cj.NotableSortOrder, schema.NameOrder{
				Name:  node.Str("Name"),
				Order: skill.Ref("Stat").ID + 1, // the reference's 1-based row index
			})
		}
	}
	for skill := range specialSkills.Rows() {
		node := skill.Ref("Node")
		if node.Bool("Keystone") {
			cj.Keystones = append(cj.Keystones, node.Str("Name"))
		}
	}
	for jewelSlot := range jewelSlots.Rows() {
		if jewelSlot.Ref("ClusterSize") != nil {
			cj.OrbitOffsets = append(cj.OrbitOffsets, schema.OrbitOffset{
				NodeId: jewelSlot.Ref("Proxy").Int("PassiveSkillNodeId"),
				Starts: jewelSlot.Ints("StartIndices"),
			})
		}
	}
	return cj, nil
}
