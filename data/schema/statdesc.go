package schema

// StatDescs holds the parsed stat description files, keyed by file basename
// (e.g. "stat_descriptions").
type StatDescs map[string]*StatDescFile

type StatDescFile struct {
	Parent      string           `json:"parent,omitempty"`
	Descriptors []StatDescriptor `json:"descriptors"`
}

// StatDescriptor is one entry. NoDesc entries carry a single stat and no
// language block. For description entries, Stats is nil when the file never
// supplied a stats line, and Lang holds the English lines (always non-nil,
// possibly empty).
type StatDescriptor struct {
	Name   string     `json:"name,omitempty"`
	NoDesc bool       `json:"noDesc,omitempty"`
	Stats  []string   `json:"stats"`
	Lang   []DescLine `json:"lang"`
}

type DescLine struct {
	Text     string        `json:"text"`
	Limits   []DescLimit   `json:"limits"`
	Specials []DescSpecial `json:"specials,omitempty"`
	Quality  string        `json:"quality,omitempty"` // raw gem_quality token
}

// DescLimit is one value's limit; nil ends mean an unrecognized token
// (empty table in the Lua).
type DescLimit struct {
	Min *NumOrStr `json:"min,omitempty"`
	Max *NumOrStr `json:"max,omitempty"`
}

// NumOrStr is a Lua number-or-string value.
type NumOrStr struct {
	Num *float64 `json:"num,omitempty"`
	Str *string  `json:"str,omitempty"`
}

// DescSpecial is one k/v special; exactly one V field is set
// (canonical_line carries VBool=true).
type DescSpecial struct {
	K     string   `json:"k"`
	VNum  *float64 `json:"vNum,omitempty"`
	VStr  *string  `json:"vStr,omitempty"`
	VBool *bool    `json:"vBool,omitempty"`
}
