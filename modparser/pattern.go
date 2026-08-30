package modparser

// PatternEntry is one pattern-table value: what a matched span of a mod
// line contributes to the modifiers being built — names (the array part of
// a modNameList entry), tags, flags, and the control keys parseMod acts on.
// Exported only so the modtables differential can render the tables.
type PatternEntry struct {
	Names             []string
	PerModTags        [][]Tag // tags per generated modifier (the DOUBLED form)
	Tag               Tag
	TagList           []Tag
	Flags             ModFlag
	KeywordFlags      KeywordFlag
	AddToMinion       bool
	AddToMinionTag    Tag
	AddToSkill        Tag
	AddToAura         bool
	OnlyAddToBanners  bool
	NewAura           bool
	NewAuraOnlyAllies bool
	ApplyToEnemy      bool
	ActorEnemy        bool
	PlayerTag         Tag
	PlayerTagList     []Tag
	ModSuffix         string
}

// merge folds another entry's controls in (later entries win, as the
// reference's key-by-key copy did).
func (e *PatternEntry) merge(o *PatternEntry) {
	e.Flags |= o.Flags
	e.KeywordFlags |= o.KeywordFlags
	e.AddToMinion = e.AddToMinion || o.AddToMinion
	e.AddToAura = e.AddToAura || o.AddToAura
	e.OnlyAddToBanners = e.OnlyAddToBanners || o.OnlyAddToBanners
	e.NewAura = e.NewAura || o.NewAura
	e.NewAuraOnlyAllies = e.NewAuraOnlyAllies || o.NewAuraOnlyAllies
	e.ApplyToEnemy = e.ApplyToEnemy || o.ApplyToEnemy
	e.ActorEnemy = e.ActorEnemy || o.ActorEnemy
	if o.AddToMinionTag != nil {
		e.AddToMinionTag = o.AddToMinionTag
	}
	if o.AddToSkill != nil {
		e.AddToSkill = o.AddToSkill
	}
	if o.PlayerTag != nil {
		e.PlayerTag = o.PlayerTag
	}
	if o.PlayerTagList != nil {
		e.PlayerTagList = o.PlayerTagList
	}
	if o.ModSuffix != "" {
		e.ModSuffix = o.ModSuffix
	}
}

// nameValue is a modNameList-family value: a bare name, a name list, or
// an entry carrying names plus controls.
type nameValue interface{ nameEntry() *PatternEntry }

type name string
type nameList []string

func (n name) nameEntry() *PatternEntry     { return &PatternEntry{Names: []string{string(n)}} }
func (n nameList) nameEntry() *PatternEntry { return &PatternEntry{Names: n} }
func (e *PatternEntry) nameEntry() *PatternEntry {
	return e
}

// entryValue is a modTagList/preFlagList value: an entry, or a closure
// building one from the captures.
type entryValue interface{ entryFor(c caps) *PatternEntry }

type entryFn func(c caps) *PatternEntry

func (e *PatternEntry) entryFor(caps) *PatternEntry { return e }
func (f entryFn) entryFor(c caps) *PatternEntry     { return f(c) }

// modsValue is a specialModList value: a modifier list, or a closure
// building one. A nil list from a closure means the line is not
// understood; an empty non-nil list is understood and contributes nothing.
type modsValue interface{ modsFor(c caps) []*Mod }

type modList []*Mod
type modFn func(c caps) []*Mod

func (l modList) modsFor(caps) []*Mod { return append([]*Mod{}, l...) }
func (f modFn) modsFor(c caps) []*Mod { return f(c) }

// flagTypeValue is a flagTypes value: a FLAG modifier name, or a fully
// specified modifier ("hexproof").
type flagTypeValue interface{ isFlagType() }

type flagName string

// FlagTypeMod is a flagTypes entry that names the whole modifier.
type FlagTypeMod struct {
	Name  string
	Type  ModType
	Value Value
}

func (flagName) isFlagType()    {}
func (FlagTypeMod) isFlagType() {}

// TableFn stands for a pattern-table closure in Tables()' view.
type TableFn struct{}
