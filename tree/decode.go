// Decoding helpers for data/raw/tree_<version>.json, which is canon-encoded:
// every Lua table is a JSON object, arrays included ({"1":..., "2":...}).
package tree

import "strconv"

// canonArray converts a numeric-keyed object to a slice; returns nil when v
// is not array-shaped.
func canonArray(v any) []any {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	out := make([]any, len(m))
	for i := 1; i <= len(m); i++ {
		e, ok := m[strconv.Itoa(i)]
		if !ok {
			return nil
		}
		out[i-1] = e
	}
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) float64 {
	n, _ := v.(float64)
	return n
}

func numPtr(v any) *float64 {
	if n, ok := v.(float64); ok {
		return &n
	}
	return nil
}

func boolean(v any) bool {
	b, _ := v.(bool)
	return b
}

func strList(v any) []string {
	arr := canonArray(v)
	out := make([]string, len(arr))
	for i, e := range arr {
		out[i] = str(e)
	}
	return out
}

// idList decodes out/in link lists: ids arrive as strings in the GGG data.
func idList(v any) []int64 {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(m))
	for i := 1; i <= len(m); i++ {
		e, ok := m[strconv.Itoa(i)]
		if !ok {
			break
		}
		switch t := e.(type) {
		case string:
			n, err := strconv.ParseInt(t, 10, 64)
			if err != nil {
				panic("tree: non-numeric link id " + t)
			}
			out = append(out, n)
		case float64:
			out = append(out, int64(t))
		}
	}
	return out
}
