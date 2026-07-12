package st

import (
	"encoding/json"
	"strconv"
)

// asFloat coerces an arbitrary value (typically decoded from JSON, where all
// numbers are float64) into a float64, returning def when coercion is not
// possible.
func asFloat(v any, def float64) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f
		}
	}
	return def
}

// asInt coerces an arbitrary value into an int, returning def on failure.
func asInt(v any, def int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return def
}

// asBool coerces an arbitrary value into a bool, returning def on failure.
func asBool(v any, def bool) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		if pb, err := strconv.ParseBool(b); err == nil {
			return pb
		}
	}
	return def
}

// asString coerces an arbitrary value into a string, returning def on failure.
func asString(v any, def string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

// asStringSlice coerces an arbitrary value (typically a []any decoded from
// JSON) into a []string, returning def on failure.
func asStringSlice(v any, def []string) []string {
	switch xs := v.(type) {
	case []string:
		return xs
	case []any:
		out := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return def
}

// clampFloat constrains x to the inclusive range [min, max].
func clampFloat(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// containsString reports whether xs contains s.
func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
