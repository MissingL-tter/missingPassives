package schema

// Pantheons holds the Pantheon panel data, in PantheonPanelLayout row order
// (disabled rows excluded).
type Pantheons []Pantheon

func (Pantheons) isDocument() {}

type Pantheon struct {
	Id         string        `json:"id"`
	IsMajorGod bool          `json:"isMajorGod"`
	Gods       []PantheonGod `json:"gods"`
}

// PantheonGod is one soul slot; Index is its 1-based position in the panel
// (slots without stats are omitted from the list).
type PantheonGod struct {
	Index int           `json:"index"`
	Name  string        `json:"name"`
	Mods  []PantheonMod `json:"mods"`
}

type PantheonMod struct {
	StatId string `json:"statId"`
	Line   string `json:"line"`
	Value  int64  `json:"value"`
}
