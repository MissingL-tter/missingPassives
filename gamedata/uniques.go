package gamedata

// Uniques holds the unique item database, keyed by item type ("axe", ...).
// Each item is its final rewritten text lines — the mod-id templates
// resolved to mod text, ordered by stat order.
type Uniques map[string]UniqueFile

type UniqueFile struct {
	Sections []UniqueSection `json:"sections"`
	Post     []string        `json:"post"` // trailing lines after the last item
}

// UniqueSection is a run of items sharing a passthrough preamble (the
// header, or a "-- Weapon: ..." category comment). Closer is the literal
// closing line the template used ("]],", "]]" or "]],}").
type UniqueSection struct {
	Pre    []string   `json:"pre"`
	Items  [][]string `json:"items"`
	Closer string     `json:"closer"`
}
