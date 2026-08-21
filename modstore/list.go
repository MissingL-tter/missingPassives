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
	if mod.Type == "BASE" || mod.Type == "INC" || mod.Type == "MORE" {
		for i, curMod := range l.Mods {
			if modparser.CompareModParams(curMod, mod) {
				// #EVAL: archive parity — copyTable(self[i], true) is SHALLOW, so
				// the merged copy shares tag tables (and their mutations) with
				// the original.
				cp := *curMod
				cp.Value = arithNum(curMod.Value) + arithNum(mod.Value)
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

func (l *List) SumInternal(ctx Store, modType string, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) float64 {
	result := 0.0
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == modType && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					if v := evalMod(ctx, mod, cfg, nil); v != nil {
						result += arithNum(v)
					}
				} else {
					result += arithNum(mod.Value)
				}
			}
		}
	}
	if l.Parent != nil {
		result += l.Parent.SumInternal(ctx, modType, cfg, flags, keywordFlags, source, names...)
	}
	return result
}

func (l *List) MoreInternal(ctx Store, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) float64 {
	result := 1.0
	var modPrecision int
	hasPrecision := false
	for _, name := range names {
		modResult := 1.0
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == "MORE" && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					var value float64
					if v := evalMod(ctx, mod, cfg, nil); v != nil {
						value = arithNum(v)
					}
					modResult *= 1 + value/100
				} else {
					modResult *= 1 + arithNum(mod.Value)/100
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
		result *= l.Parent.MoreInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return result
}

func (l *List) FlagInternal(ctx Store, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) bool {
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == "FLAG" && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					if truthy(evalMod(ctx, mod, cfg, nil)) {
						return true
					}
				} else if truthy(mod.Value) {
					return true
				}
			}
		}
	}
	if l.Parent != nil {
		return l.Parent.FlagInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return false
}

func (l *List) OverrideInternal(ctx Store, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) any {
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == "OVERRIDE" && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					if v := evalMod(ctx, mod, cfg, nil); truthy(v) {
						return v
					}
				} else if truthy(mod.Value) {
					return mod.Value
				}
			}
		}
	}
	if l.Parent != nil {
		return l.Parent.OverrideInternal(ctx, cfg, flags, keywordFlags, source, names...)
	}
	return nil
}

func (l *List) ListInternal(ctx Store, result *[]any, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) {
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && mod.Type == "LIST" && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				if hasTags(mod) {
					// #EVAL: archive parity — `or nullValue` reads an undefined
					// global (nil), so failed evaluations are dropped silently.
					if v := evalMod(ctx, mod, cfg, nil); v != nil {
						*result = append(*result, v)
					}
				} else if truthy(mod.Value) {
					*result = append(*result, mod.Value)
				}
			}
		}
	}
	if l.Parent != nil {
		l.Parent.ListInternal(ctx, result, cfg, flags, keywordFlags, source, names...)
	}
}

func (l *List) TabulateInternal(ctx Store, result *[]TabEntry, modType string, hasType bool, cfg *Cfg, flags, keywordFlags int64, source string, names ...string) {
	for _, name := range names {
		for _, mod := range l.Mods {
			if mod.Name == name && (!hasType || mod.Type == modType) && modMatches(mod, flags, keywordFlags) && sourceOK(mod, source, false) {
				var value any
				if hasTags(mod) {
					value = evalMod(ctx, mod, cfg, nil)
				} else {
					value = mod.Value
				}
				if tabValueKept(value, mod.Type) {
					*result = append(*result, TabEntry{Value: value, Mod: mod})
				}
			}
		}
	}
	if l.Parent != nil {
		l.Parent.TabulateInternal(ctx, result, modType, hasType, cfg, flags, keywordFlags, source, names...)
	}
}
