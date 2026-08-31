package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/item"
)

// energyBladeFor synthesizes the weapon that replaces prev while the Energy
// Blade buff is active (CalcSetup.lua L1016-1048): a bare base of the
// matching hand count, run through the item machinery, then handed the
// replaced weapon's sockets. Returns nil for a slot holding something with
// no weapon data, or a bow - those keep their own modifiers.
func energyBladeFor(prev *Item) *Item {
	if prev.In.WeaponData == nil {
		return nil
	}
	side := prev.In.WeaponData[1]
	if side == nil {
		panic("calc: weapon-data item with no side 1 (the Lua errors indexing it)")
	}
	info, ok := data.WeaponTypeInfo[side.Type]
	if !ok || side.Type == "Bow" {
		return nil
	}
	name := "Energy Blade Two Handed"
	if info.OneHand {
		name = "Energy Blade One Handed"
	}
	quality := 0.0
	eb := &item.Item{
		Name:     name,
		BaseName: name,
		Base:     data.ItemBases[name],
		Rarity:   "NORMAL",
		Quality:  &quality,
	}
	// #EVAL: the reference parses the base's implicit lines here, but
	// looks them up on item.baseName - a string, where the lookup is
	// always nil - so the block never runs and the blade has no
	// implicits. Reproduced by omitting it.
	eb.NormaliseQuality()
	eb.BuildAndParseRaw()
	// The replaced weapon's sockets land after the mod list is built, so
	// the socket multipliers on it count the base's own default sockets,
	// not these.
	eb.Sockets = nil
	for _, s := range prev.In.Sockets {
		eb.Sockets = append(eb.Sockets, &item.Socket{Color: s.Color, Group: s.Group})
	}
	if n := prev.In.AbyssalSocketCount; n != nil {
		eb.AbyssalSocketCount = *n
	}
	return &Item{In: ItemInputOf(eb)}
}
