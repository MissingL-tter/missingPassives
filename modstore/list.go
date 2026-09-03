// Port of .archive/src/Classes/ModList.lua — modifiers in a flat list.

package modstore

import "github.com/MissingL-tter/missingPassives/modparser"

type List struct {
	ModStore
	Mods []*modparser.Mod
}

func NewList(parent Store) *List {
	l := &List{}
	l.ModStore = newModStore(l, parent)
	return l
}

func (l *List) AddMod(mod *modparser.Mod) {
	l.Mods = append(l.Mods, mod)
}

func (l *List) ReplaceModInternal(mod *modparser.Mod) bool {
	for i, curMod := range l.Mods {
		if mod.Name == curMod.Name && mod.Type == curMod.Type && mod.Flags == curMod.Flags &&
			mod.KeywordFlags == curMod.KeywordFlags && sameSource(mod, curMod) {
			l.Mods[i] = mod
			return true
		}
	}
	if l.Parent != nil {
		return l.Parent.ReplaceModInternal(mod)
	}
	return false
}

func (l *List) ConvertModInternal(oldName string, mod *modparser.Mod) bool {
	for i, curMod := range l.Mods {
		if oldName == curMod.Name && mod.Type == curMod.Type && mod.Flags == curMod.Flags &&
			mod.KeywordFlags == curMod.KeywordFlags && sameSource(mod, curMod) {
			l.Mods[i] = mod
			return true
		}
	}
	if l.Parent != nil {
		return l.Parent.ConvertModInternal(oldName, mod)
	}
	return false
}

// MergeMod ports ModList:MergeMod.
func (l *List) MergeMod(mod *modparser.Mod, skipNonAdditive bool) {
	if mod.Type == modparser.Base || mod.Type == modparser.Inc || mod.Type == modparser.More {
		for i, curMod := range l.Mods {
			if modparser.CompareModParams(curMod, mod) {
				// Archive parity: copyTable(self[i], true) is SHALLOW, so
				// the merged copy shares tag tables (and their mutations) with
				// the original.
				cp := *curMod
				cp.Value = modparser.Num(valueNum(curMod.Value) + valueNum(mod.Value))
				l.Mods[i] = &cp
				return
			}
		}
	}
	if !skipNonAdditive {
		l.AddMod(mod)
	}
}

func (l *List) AddList(list []*modparser.Mod) {
	l.Mods = append(l.Mods, list...)
}

func (l *List) sumInternal(ctx Store, modType modparser.ModType, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) float64 {
	result := 0.0
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == modType && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					if v := evalMod(ctx, mod, cfg, nil); v != nil {
						result += valueNum(v)
					}
				} else {
					result += valueNum(mod.Value)
				}
			}
		}
	}
	if l.Parent != nil {
		result += l.Parent.sumInternal(ctx, modType, cfg, flags, keywordFlags, source, names...)
	}
	return result
}

func (l *List) moreInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) float64 {
	result := 1.0
	var modPrecision int
	hasPrecision := false
	for _, name := range names {
		modResult := 1.0
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == modparser.More && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					var value float64
					if v := evalMod(ctx, mod, cfg, nil); v != nil {
						value = valueNum(v)
					}
					modResult *= 1 + value/100
				} else {
					modResult *= 1 + valueNum(mod.Value)/100
				}
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
	if l.Parent != nil {
		result *= l.Parent.moreInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return result
}

func (l *List) flagInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) bool {
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == modparser.Flag && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
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
	if l.Parent != nil {
		return l.Parent.flagInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return false
}

func (l *List) overrideInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) modparser.Value {
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == modparser.Override && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
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
	if l.Parent != nil {
		return l.Parent.overrideInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return nil
}

func (l *List) listInternal(ctx Store, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) []modparser.Value {
	var result []modparser.Value
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == modparser.List && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					// Archive parity: `or nullValue` reads an undefined
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
	if l.Parent != nil {
		result = append(result, l.Parent.listInternal(ctx, cfg, flags, keywordFlags, source, names...)...)
	}
	return result
}

func (l *List) tabulateInternal(ctx Store, modType modparser.ModType, hasType bool, cfg *Cfg, flags modparser.ModFlag, keywordFlags modparser.KeywordFlag, source string, names ...string) []TabEntry {
	var result []TabEntry
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && (!hasType || mod.Type == modType) && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				var value modparser.Value
				if hasTags(mod) {
					value = evalMod(ctx, mod, cfg, nil)
				} else {
					value = mod.Value
				}
				if tabValueKept(value, mod.Type) {
					result = append(result, TabEntry{Value: value, Mod: mod})
				}
			}
		}
	}
	if l.Parent != nil {
		result = append(result, l.Parent.tabulateInternal(ctx, modType, hasType, cfg, flags, keywordFlags, source, names...)...)
	}
	return result
}
