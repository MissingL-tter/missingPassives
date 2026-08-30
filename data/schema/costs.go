package schema

// Costs holds the skill cost types, in CostTypes row order, with the
// exporter's synthetic "Soul" entry appended last.
type Costs []CostType

func (Costs) isDocument() {}

type CostType struct {
	Resource       string  `json:"resource"`
	Stat           *string `json:"stat"` // nil when the row carries no stat
	ResourceString string  `json:"resourceString"`
	Divisor        int64   `json:"divisor"`
}
