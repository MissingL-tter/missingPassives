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

func buildCluster(x *Ctx) (any, error) {
	x.LoadStatFile("passive_skill_stat_descriptions.txt")

	var cj schema.ClusterJewels
	x.Dat("PassiveTreeExpansionJewels").Rows(func(jewel *Row) bool {
		size := jewel.Get("Size").(*Row)
		j := schema.ClusterJewelSize{
			Name:            luaStr(jewel.Get("BaseItemType").(*Row).Get("Name")),
			Size:            luaStr(size.Get("Id")),
			SizeIndex:       size.Index - 1,
			MinNodes:        jewel.Get("MinNodes").(int64),
			MaxNodes:        jewel.Get("MaxNodes").(int64),
			SmallIndicies:   intCells(jewel.Get("SmallIndicies").([]any)),
			NotableIndicies: intCells(jewel.Get("NotableIndicies").([]any)),
			SocketIndicies:  intCells(jewel.Get("SocketIndicies").([]any)),
			TotalIndicies:   jewel.Get("TotalIndicies").(int64),
		}
		for _, skill := range x.Dat("PassiveTreeExpansionSkills").GetRowList("JewelSize", size) {
			node := skill.Get("Node").(*Row)
			tagId := luaStr(skill.Get("Tag").(*Row).Get("Id"))
			s := schema.ClusterSkill{
				Id:   luaStr(node.Get("Id")),
				Name: luaStr(node.Get("Name")),
				Icon: reDdsEnd.ReplaceAllString(luaStr(node.Get("Icon")), "png"),
				Tag:  tagId,
			}
			if strings.Contains(tagId, "old_do_not_use") {
				s.Name += " (Legacy)"
			}
			if mastery, ok := skill.Get("Mastery").(*Row); ok {
				icon := reDdsEnd.ReplaceAllString(luaStr(mastery.Get("Icon")), "png")
				s.MasteryIcon = &icon
			}
			stats := map[string]*statVal{}
			for i, stat := range node.Get("Stats").([]any) {
				v := float64(node.Get(fmt.Sprintf("Stat%d", i+1)).(int64))
				stats[luaStr(stat.(*Row).Get("Id"))] = &statVal{min: v, max: v}
			}
			s.Stats = x.DescribeStats(stats).Lines
			j.Skills = append(j.Skills, s)
		}
		cj.Jewels = append(cj.Jewels, j)
		return true
	})

	x.Dat("PassiveTreeExpansionSpecialSkills").Rows(func(skill *Row) bool {
		node := skill.Get("Node").(*Row)
		if node.Get("Notable").(bool) {
			cj.NotableSortOrder = append(cj.NotableSortOrder, schema.NameOrder{
				Name:  luaStr(node.Get("Name")),
				Order: skill.Get("Stat").(*Row).Index,
			})
		}
		return true
	})
	x.Dat("PassiveTreeExpansionSpecialSkills").Rows(func(skill *Row) bool {
		node := skill.Get("Node").(*Row)
		if node.Get("Keystone").(bool) {
			cj.Keystones = append(cj.Keystones, luaStr(node.Get("Name")))
		}
		return true
	})
	x.Dat("PassiveJewelSlots").Rows(func(jewelSlot *Row) bool {
		if _, ok := jewelSlot.Get("ClusterSize").(*Row); ok {
			cj.OrbitOffsets = append(cj.OrbitOffsets, schema.OrbitOffset{
				NodeId: jewelSlot.Get("Proxy").(*Row).Get("PassiveSkillNodeId").(int64),
				Starts: intCells(jewelSlot.Get("StartIndices").([]any)),
			})
		}
		return true
	})
	return cj, nil
}
