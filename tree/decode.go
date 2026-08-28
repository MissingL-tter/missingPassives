// Decoding helpers for data/raw/tree_<version>.json (conventional JSON;
// arrays are arrays).
package tree

import "strconv"

// canonArray returns v as a slice; nil when absent or empty.
func canonArray(v any) []any {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	return arr
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
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(arr))
	for _, e := range arr {
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
