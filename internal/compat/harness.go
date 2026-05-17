// Package compat compares Python v3 and Go v4 tool outputs during the port.
// Deleted at v4.0 alongside the Python source.
package compat

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// DiffJSON returns a slice of human-readable diff descriptions between two
// JSON payloads. Empty slice means equivalent.
//
// Equivalence rules:
//   - Arrays of objects are compared after sorting on a stable serialisation
//     of each element, so result ordering doesn't cause false positives.
//   - Maps and primitives are compared with reflect.DeepEqual.
//   - Numbers come through as float64 from json.Unmarshal; integer values
//     compare cleanly.
func DiffJSON(want, got json.RawMessage) ([]string, error) {
	var w, g any
	if err := json.Unmarshal(want, &w); err != nil {
		return nil, fmt.Errorf("unmarshal want: %w", err)
	}
	if err := json.Unmarshal(got, &g); err != nil {
		return nil, fmt.Errorf("unmarshal got: %w", err)
	}
	return diffAny("$", w, g), nil
}

func diffAny(path string, want, got any) []string {
	switch w := want.(type) {
	case []any:
		gSlice, ok := got.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s: want []any, got %T", path, got)}
		}
		return diffSlice(path, w, gSlice)
	case map[string]any:
		gMap, ok := got.(map[string]any)
		if !ok {
			return []string{fmt.Sprintf("%s: want map, got %T", path, got)}
		}
		return diffMap(path, w, gMap)
	default:
		if !reflect.DeepEqual(want, got) {
			return []string{fmt.Sprintf("%s: want %v, got %v", path, want, got)}
		}
		return nil
	}
}

func diffSlice(path string, want, got []any) []string {
	if len(want) != len(got) {
		return []string{fmt.Sprintf("%s: length want=%d got=%d", path, len(want), len(got))}
	}
	canon := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("<marshal-error: %v>", err)
		}
		return string(b)
	}
	w := make([]string, len(want))
	g := make([]string, len(got))
	for i := range want {
		w[i] = canon(want[i])
		g[i] = canon(got[i])
	}
	sort.Strings(w)
	sort.Strings(g)
	var diffs []string
	for i := range w {
		if w[i] != g[i] {
			diffs = append(diffs, fmt.Sprintf("%s[%d]: %s vs %s", path, i, w[i], g[i]))
		}
	}
	return diffs
}

func diffMap(path string, want, got map[string]any) []string {
	var diffs []string
	for k, vWant := range want {
		vGot, ok := got[k]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("%s.%s: missing in got", path, k))
			continue
		}
		diffs = append(diffs, diffAny(path+"."+k, vWant, vGot)...)
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			diffs = append(diffs, fmt.Sprintf("%s.%s: unexpected in got", path, k))
		}
	}
	return diffs
}
