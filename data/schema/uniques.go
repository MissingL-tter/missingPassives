package schema

// Uniques holds the unique item database, keyed by item type ("axe", ...).
// Each item is its final rewritten text lines — the mod-id templates
// resolved to mod text, ordered by stat order.
type Uniques map[string]UniqueFile

type UniqueFile struct {
	Sections []UniqueSection `json:"sections"`
}

// UniqueSection is a run of items (the template groups them under category
// comments; the render test reconstructs that passthrough text from the
// archive template).
type UniqueSection struct {
	Items [][]string `json:"items"`
}
