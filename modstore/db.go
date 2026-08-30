// Port of .archive/src/Classes/ModDB.lua — modifiers bucketed by name.

package modstore

import "github.com/MissingL-tter/missingPassives/modparser"

type DB struct {
	ModStore
	Mods map[string][]*modparser.Mod
}

func NewDB(parent Store) *DB {
	db := &DB{Mods: map[string][]*modparser.Mod{}}
	db.ModStore = newModStore(db, parent)
	return db
}

func (db *DB) AddMod(mod *modparser.Mod) {
	db.Mods[mod.Name] = append(db.Mods[mod.Name], mod)
}

// sourceOK applies the `mod.source:match("[^:]+") == source` filter.
// #EVAL: archive parity — only ModDB.SumInternal guards source-less mods
// (its extra `mod.source and` check); every other aggregation errors on
// them, so guardNil=false panics.
func sourceOK(mod *modparser.Mod, source string, guardNil bool) bool {
	if source == "" {
		return true
	}
	if !mod.SourceSet {
		if guardNil {
			return false
		}
		panic("modstore: source filter against a mod without source (the Lua errors)")
	}
	return sourceMatch(mod.Source, source)
}

func modMatches(mod *modparser.Mod, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag) bool {
	return flags&mod.Flags == mod.Flags && matchKeywordFlags(keywordFlags, mod.KeywordFlags)
}

// hasTags is the `mod[1]` check: an evaluated mod (tag in slot 1).
func hasTags(mod *modparser.Mod) bool {
	return len(mod.Tags) > 0 && mod.Tags[0] != nil
}

func (db *DB) ReplaceModInternal(mod *modparser.Mod) bool {
	name := mod.Name
	if db.Mods[name] == nil {
		db.Mods[name] = []*modparser.Mod{}
	}
	modList := db.Mods[name]
	modIndex := -1
	for i, curMod := range modList {
		if mod.Name == curMod.Name && mod.Type == curMod.Type && mod.Flags == curMod.Flags &&
			mod.KeywordFlags == curMod.KeywordFlags && sameSource(mod, curMod) && !curMod.Replaced {
			modIndex = i
			mod.Replaced = true
			break
		}
	}
	if modIndex >= 0 {
		modList[modIndex] = mod
		return true
	}
	if db.Parent != nil {
		return db.Parent.ReplaceModInternal(mod)
	}
	return false
}

func sameSource(a, b *modparser.Mod) bool {
	return a.SourceSet == b.SourceSet && a.Source == b.Source
}

func (db *DB) ConvertModInternal(oldName string, mod *modparser.Mod) bool {
	oldList, ok := db.Mods[oldName]
	if !ok {
		if db.Parent != nil {
			return db.Parent.ConvertModInternal(oldName, mod)
		}
		return false
	}
	for i, curMod := range oldList {
		if oldName == curMod.Name && mod.Type == curMod.Type && mod.Flags == curMod.Flags &&
			mod.KeywordFlags == curMod.KeywordFlags && sameSource(mod, curMod) && !curMod.Converted {
			db.Mods[oldName] = append(oldList[:i], oldList[i+1:]...)
			mod.Converted = true
			db.Mods[mod.Name] = append(db.Mods[mod.Name], mod)
			return true
		}
	}
	if db.Parent != nil {
		return db.Parent.ConvertModInternal(oldName, mod)
	}
	return false
}

func (db *DB) AddList(list []*modparser.Mod) {
	for _, mod := range list {
		db.Mods[mod.Name] = append(db.Mods[mod.Name], mod)
	}
}

// AddDB ports ModDB:AddDB.
func (db *DB) AddDB(other *DB) {
	for modName, modList := range other.Mods {
		db.Mods[modName] = append(db.Mods[modName], modList...)
	}
}

func (db *DB) SumInternal(ctx Store, modType modparser.ModType, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) float64 {
	result := 0.0
	globalLimits := map[string]float64{}
	for _, name := range names {
		for _, mod := range db.Mods[name] {
			if mod.Type == modType && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, true) {
				if hasTags(mod) {
					if v := evalMod(ctx, mod, cfg, globalLimits); v != nil {
						result += valueNum(v)
					}
				} else {
					result += valueNum(mod.Value)
				}
			}
		}
	}
	if db.Parent != nil {
		result += db.Parent.SumInternal(ctx, modType, cfg, flags, keywordFlags, source, names...)
	}
	return result
}

func (db *DB) MoreInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) float64 {
	result := 1.0
	var modPrecision int
	hasPrecision := false
	globalLimits := map[string]float64{}
	for _, name := range names {
		modResult := 1.0
		for _, mod := range db.Mods[name] {
			if mod.Type == modparser.More && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				var value float64
				if hasTags(mod) {
					if v := evalMod(ctx, mod, cfg, globalLimits); v != nil {
						value = valueNum(v)
					}
				} else if modparser.Truthy(mod.Value) {
					value = valueNum(mod.Value)
				}
				modResult *= 1 + value/100
				if p, ok := highPrecision(mod.Name, mod.Type); ok {
					if !hasPrecision || p > modPrecision {
						modPrecision = p
					}
					hasPrecision = true
				}
			}
		}
		result = applyMorePrecision(result, modResult, modPrecision, hasPrecision)
	}
	if db.Parent != nil {
		result *= db.Parent.MoreInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return result
}

func applyMorePrecision(result, modResult float64, modPrecision int, hasPrecision bool) float64 {
	if hasPrecision {
		power := pow10(modPrecision)
		return floorf(result*modResult*power) / power
	}
	return result * round(modResult, 2)
}

func (db *DB) FlagInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) bool {
	for _, name := range names {
		for _, mod := range db.Mods[name] {
			if mod.Type == modparser.Flag && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					if modparser.Truthy(evalMod(ctx, mod, cfg, nil)) {
						return true
					}
				} else if modparser.Truthy(mod.Value) {
					return true
				}
			}
		}
	}
	if db.Parent != nil {
		return db.Parent.FlagInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return false
}

func (db *DB) OverrideInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) modparser.Value {
	for _, name := range names {
		for _, mod := range db.Mods[name] {
			if mod.Type == modparser.Override && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					if v := evalMod(ctx, mod, cfg, nil); modparser.Truthy(v) {
						return v
					}
				} else if modparser.Truthy(mod.Value) {
					return mod.Value
				}
			}
		}
	}
	if db.Parent != nil {
		return db.Parent.OverrideInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return nil
}

func (db *DB) ListInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) []modparser.Value {
	var result []modparser.Value
	for _, name := range names {
		for _, mod := range db.Mods[name] {
			if mod.Type == modparser.List && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					// #EVAL: archive parity — `or nullValue` reads an undefined
					// global (nil), so failed evaluations are dropped silently.
					if v := evalMod(ctx, mod, cfg, nil); v != nil {
						result = append(result, v)
					}
				} else if modparser.Truthy(mod.Value) {
					result = append(result, mod.Value)
				}
			}
		}
	}
	if db.Parent != nil {
		result = append(result, db.Parent.ListInternal(ctx, cfg, flags, keywordFlags, source, names...)...)
	}
	return result
}

func (db *DB) TabulateInternal(ctx Store, modType modparser.ModType, hasType bool, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) []TabEntry {
	var result []TabEntry
	globalLimits := map[string]float64{}
	for _, name := range names {
		for _, mod := range db.Mods[name] {
			if (!hasType || mod.Type == modType) && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				var value modparser.Value
				if hasTags(mod) {
					value = evalMod(ctx, mod, cfg, globalLimits)
				} else {
					value = mod.Value
				}
				if tabValueKept(value, mod.Type) {
					result = append(result, TabEntry{Value: value, Mod: mod})
				}
			}
		}
	}
	if db.Parent != nil {
		result = append(result, db.Parent.TabulateInternal(ctx, modType, hasType, cfg, flags, keywordFlags, source, names...)...)
	}
	return result
}

// tabValueKept is `value and (value ~= 0 or mod.type == "OVERRIDE")`.
func tabValueKept(value modparser.Value, modType modparser.ModType) bool {
	if !modparser.Truthy(value) {
		return false
	}
	if n, ok := numValue(value); ok && n == 0 {
		return modType == modparser.Override
	}
	return true
}

// HasModInternal ports ModDB:HasModInternal.
func (db *DB) HasModInternal(modType modparser.ModType, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) bool {
	for _, name := range names {
		for _, mod := range db.Mods[name] {
			if mod.Type == modType && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				return true
			}
		}
	}
	if db.Parent != nil {
		if parentDB, ok := db.Parent.(*DB); ok {
			return parentDB.HasModInternal(modType, flags, keywordFlags, source, names...)
		}
		// #EVAL: archive parity — see HasMod.
		panic("modstore: HasModInternal through a non-DB parent (the Lua errors)")
	}
	return false
}
