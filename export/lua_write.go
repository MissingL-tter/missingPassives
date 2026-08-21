// Lua-shaped value trees (map[any]any tables with int/string keys), used by
// the generated skillStatMap and the minion mod serializer.

package export

// luaTable is a Lua table under construction: int and string keys.
type luaTable map[any]any

func keyNum(k any) float64 {
	switch n := k.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	}
	return 0
}
