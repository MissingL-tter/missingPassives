// Port of modLib.mergeKeystones (ModTools.lua:226), deferred here from
// mod-parser because it needs a live ModDB and the tree keystone map.

package modstore

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// KeystoneEnv is the slice of env/spec that mergeKeystones touches:
// env.keystonesAdded plus env.spec.tree.keystoneMap[name].modList.
type KeystoneEnv struct {
	KeystonesAdded map[string]bool
	KeystoneMods   map[string][]*modparser.Mod
}

// MergeKeystones ports modLib.mergeKeystones.
// #EVAL: archive parity — the reference mutates the keystone map's own mods
// through setSource when the granting mod's source is not tree-flavoured, so
// the tree's shared modList carries the last granter's source.
// The reference calls it on a ModDB (perform) and on the tree's ModList
// (initEnv), so any Store is accepted.
func MergeKeystones(env *KeystoneEnv, modDB Store) {
	if env.KeystonesAdded == nil {
		env.KeystonesAdded = map[string]bool{}
	}
	for _, modObj := range modDB.Tabulate("LIST", nil, "Keystone") {
		name, ok := modObj.Value.(string)
		if !ok {
			continue // keystoneMap lookup with a non-string key finds nothing
		}
		modList, known := env.KeystoneMods[name]
		if env.KeystonesAdded[name] || !known {
			continue
		}
		env.KeystonesAdded[name] = true
		fromTree := modObj.Mod.SourceSet && !strings.Contains(strings.ToLower(modObj.Mod.Source), "tree")
		for _, mod := range modList {
			if fromTree {
				modDB.AddMod(modparser.SetSource(mod, modObj.Mod.Source))
			} else {
				modDB.AddMod(mod)
			}
		}
	}
}
