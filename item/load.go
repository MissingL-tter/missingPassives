// The ItemsTab:Load protocol for one saved <Item> node (ItemsTab.lua
// L1215-1261): attribute pre-seeding, ParseRaw, legacy ModRange overrides,
// then the second BuildModList.
package item

// SavedModRange is one legacy <ModRange> child.
type SavedModRange struct {
	ID    int
	Range float64
}

// SavedItem is the decoded <Item> node.
type SavedItem struct {
	ID          int
	Variant     *int
	VariantAlt  *int
	VariantAlt2 *int
	VariantAlt3 *int
	VariantAlt4 *int
	VariantAlt5 *int
	Raw         string
	ModRanges   []SavedModRange
}

// LoadSaved builds the Item the way ItemsTab:Load does. Returns nil when
// the raw text resolves no base (the loader drops such items).
func LoadSaved(saved *SavedItem) *Item {
	it := &Item{ID: saved.ID}
	it.Variant = saved.Variant
	if saved.VariantAlt != nil {
		it.HasAltVariant = true
		it.VariantAlt = saved.VariantAlt
	}
	if saved.VariantAlt2 != nil {
		it.HasAltVariant2 = true
		it.VariantAlt2 = saved.VariantAlt2
	}
	if saved.VariantAlt3 != nil {
		it.HasAltVariant3 = true
		it.VariantAlt3 = saved.VariantAlt3
	}
	if saved.VariantAlt4 != nil {
		it.HasAltVariant4 = true
		it.VariantAlt4 = saved.VariantAlt4
	}
	if saved.VariantAlt5 != nil {
		it.HasAltVariant5 = true
		it.VariantAlt5 = saved.VariantAlt5
	}
	it.ParseRaw(saved.Raw, "", false)
	for _, mr := range saved.ModRanges {
		id := mr.ID
		for _, list := range [][]*ModLine{it.BuffModLines, it.EnchantModLines, it.ScourgeModLines, it.ImplicitModLines, it.ExplicitModLines, it.CrucibleModLines} {
			if id <= len(list) {
				if id >= 1 {
					r := mr.Range
					list[id-1].Range = &r
				}
				break
			}
			id -= len(list)
		}
	}
	if it.Base == nil {
		return nil
	}
	it.BuildModList()
	return it
}
