// Package compat — extension to harness.go: documents intentional Python↔Go
// wire divergences and provides per-tool normalizers so DiffJSON only surfaces
// unintended differences.
//
// Why this exists: per the M1 design spec (E1 / section 337), tool-by-tool
// parity is the gate, but "shape compare" is acceptable for tools where Go's
// idiomatic struct field names diverge from Python's. Rather than refactor Go
// to mirror Python field-by-field (and lose info like ClassHit.Kind), we
// normalize both sides into a common comparison shape and document why.
package compat

import "encoding/json"

// Expected captures parity-comparison configuration for one tool.
type Expected struct {
	// Skip = true means the tool is not in Python v3 (e.g. list_workspace_roots);
	// the harness short-circuits with a "SKIP <tool> — <Reason>" message.
	Skip bool

	// Reason documents the design intent for this divergence.
	Reason string

	// Normalize transforms both Python (want) and Go (got) into a comparable
	// shape. Nil = pass through unchanged.
	Normalize func(want, got json.RawMessage) (json.RawMessage, json.RawMessage, error)
}

// ToolExpectations is the per-tool parity rulebook. Tools not listed here
// go through DiffJSON unchanged.
var ToolExpectations = map[string]Expected{
	// list_workspace_roots is Go-only; Python v3 returns an error string for
	// unknown tools. We skip rather than parse an error string as JSON.
	"list_workspace_roots": {
		Skip:   true,
		Reason: "Python v3 does not implement list_workspace_roots; it returns a plain-text error",
	},

	// find_symbol: Python returns a single dict; Go returns []SymbolHit.
	// Python fields: {name, file, line, end_line, type, signature, source_preview, complete}
	// Go fields:     {file, line, end_line, qualified, kind, signature}
	//
	// Shape divergences normalised away:
	//   - Root shape: Python = single dict, Go = array → wrap Python in array.
	//   - name → qualified.
	//   - type/kind: Python emits "function"|"method"|"class"; Go emits
	//     "function"|"class". Drop from both sides.
	//   - signature: Python's Go annotator emits Python-style signatures
	//     (e.g. "def Hello()"); Go's annotator emits proper Go signatures
	//     (e.g. "func (g *Greeter) Hello() string"). This is a fidelity gap
	//     in Python's cross-language annotator, not a data bug — drop from
	//     both sides. (Carry-forward #1 from T24-A; will be fixed in v4 by
	//     using Go's native signature string everywhere.)
	//   - source_preview, complete: Python-only metadata fields.
	//   - end_line: dropped from both sides for simplicity (present in both but
	//     unnecessary for identity comparison).
	//
	// Retained on both sides: file, line, qualified — the identity triple.
	"find_symbol": {
		Reason: "Python returns a single dict; Go returns []SymbolHit. name→qualified. type/kind dropped (Python method/function distinction absent in Go). signature dropped (Python emits Python-style signatures for Go code — fidelity gap in Python's cross-language annotator). source_preview/complete are Python-only.",
		Normalize: func(want, got json.RawMessage) (json.RawMessage, json.RawMessage, error) {
			// Wrap Python's single dict in a one-element array.
			w, err := wrapDictAsArray(want)
			if err != nil {
				return nil, nil, err
			}
			// Python want: rename name→qualified; drop shape/metadata fields.
			w, err = renameKeys(w, map[string]string{"name": "qualified"})
			if err != nil {
				return nil, nil, err
			}
			w, err = dropKeys(w, "type", "source_preview", "complete", "end_line", "signature")
			if err != nil {
				return nil, nil, err
			}
			// Go got: drop fields absent from want.
			g, err := dropKeys(got, "end_line", "kind", "signature")
			if err != nil {
				return nil, nil, err
			}
			return w, g, nil
		},
	},

	// get_functions: Python {name, qualified_name, lines:[s,e], params, is_method, parent_class, file}
	// Go:           {file, qualified, line, end_line, signature}
	// Mapping: qualified_name→qualified, lines→line/end_line; drop name/params/
	// is_method/parent_class on want, drop signature on got (Python doesn't include it).
	"get_functions": {
		Reason: "Python uses 'qualified_name' and 'lines:[start,end]'; Go uses 'qualified', 'line', 'end_line'. Python carries params/is_method/parent_class; Go carries signature. Normalize to {file, qualified, line, end_line}.",
		Normalize: func(want, got json.RawMessage) (json.RawMessage, json.RawMessage, error) {
			w, err := renameKeys(want, map[string]string{"qualified_name": "qualified"})
			if err != nil {
				return nil, nil, err
			}
			w, err = linesToLineEndline(w)
			if err != nil {
				return nil, nil, err
			}
			w, err = dropKeys(w, "name", "params", "is_method", "parent_class")
			if err != nil {
				return nil, nil, err
			}
			g, err := dropKeys(got, "signature")
			if err != nil {
				return nil, nil, err
			}
			return w, g, nil
		},
	},

	// get_classes: Python {name, qualified_name, lines:[s,e], methods, method_signatures, bases, file}
	// Go:          {file, qualified, kind, line, end_line}
	// Mapping: qualified_name→qualified, lines→line/end_line; drop name/methods/
	// method_signatures/bases on want, drop kind on got (Python has no equivalent).
	"get_classes": {
		Reason: "Python uses 'qualified_name' and 'lines:[start,end]' with methods/bases metadata; Go uses 'qualified', 'line', 'end_line', 'kind'. Normalize to {file, qualified, line, end_line}.",
		Normalize: func(want, got json.RawMessage) (json.RawMessage, json.RawMessage, error) {
			w, err := renameKeys(want, map[string]string{"qualified_name": "qualified"})
			if err != nil {
				return nil, nil, err
			}
			w, err = linesToLineEndline(w)
			if err != nil {
				return nil, nil, err
			}
			w, err = dropKeys(w, "name", "methods", "method_signatures", "bases")
			if err != nil {
				return nil, nil, err
			}
			g, err := dropKeys(got, "kind")
			if err != nil {
				return nil, nil, err
			}
			return w, g, nil
		},
	},

	// get_imports: Python {module, names, line, is_from_import, file}
	// Go:          {file, path, alias?, line}
	// Mapping: module→path; drop names/is_from_import on want.
	// (alias is omitempty in Go; Python doesn't emit it for plain imports.)
	"get_imports": {
		Reason: "Python uses 'module' for the import path and carries 'names'/'is_from_import'; Go uses 'path' with optional 'alias'. Normalize to {file, path, line}.",
		Normalize: func(want, got json.RawMessage) (json.RawMessage, json.RawMessage, error) {
			w, err := renameKeys(want, map[string]string{"module": "path"})
			if err != nil {
				return nil, nil, err
			}
			w, err = dropKeys(w, "names", "is_from_import")
			if err != nil {
				return nil, nil, err
			}
			return w, got, nil
		},
	},

	// search_codebase: Python {content, file, line_number}
	// Go:              {file, line, text}
	// Mapping: content→text, line_number→line.
	"search_codebase": {
		Reason: "Python uses 'content' and 'line_number'; Go uses 'text' and 'line'. Field rename only — data is identical.",
		Normalize: func(want, got json.RawMessage) (json.RawMessage, json.RawMessage, error) {
			w, err := renameKeys(want, map[string]string{
				"content":     "text",
				"line_number": "line",
			})
			if err != nil {
				return nil, nil, err
			}
			return w, got, nil
		},
	},
}

// ── helpers ──────────────────────────────────────────────────────────────────

// dropKeys removes the named keys from every object in an array, or directly
// from a single object, returning the modified JSON.
func dropKeys(raw json.RawMessage, keys ...string) (json.RawMessage, error) {
	return transformElements(raw, func(m map[string]any) {
		for _, k := range keys {
			delete(m, k)
		}
	})
}

// renameKeys applies oldKey→newKey renames on every object in an array, or
// directly on a single object, returning the modified JSON.
func renameKeys(raw json.RawMessage, mapping map[string]string) (json.RawMessage, error) {
	return transformElements(raw, func(m map[string]any) {
		for old, new := range mapping {
			if v, ok := m[old]; ok {
				m[new] = v
				delete(m, old)
			}
		}
	})
}

// linesToLineEndline expands {"lines":[start,end]} into {"line":start,"end_line":end}
// on every object in an array (or a single object), returning modified JSON.
func linesToLineEndline(raw json.RawMessage) (json.RawMessage, error) {
	return transformElements(raw, func(m map[string]any) {
		lines, ok := m["lines"].([]any)
		if !ok || len(lines) < 2 {
			return
		}
		m["line"] = lines[0]
		m["end_line"] = lines[1]
		delete(m, "lines")
	})
}

// wrapDictAsArray wraps a single JSON object in a one-element array.
// If raw is already an array, it is returned unchanged.
func wrapDictAsArray(raw json.RawMessage) (json.RawMessage, error) {
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if _, isSlice := probe.([]any); isSlice {
		return raw, nil
	}
	out, err := json.Marshal([]any{probe})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// transformElements applies fn to each object element of raw (which may be an
// array of objects or a single object) and returns the re-marshalled result.
func transformElements(raw json.RawMessage, fn func(map[string]any)) (json.RawMessage, error) {
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	switch v := probe.(type) {
	case []any:
		for _, elem := range v {
			if m, ok := elem.(map[string]any); ok {
				fn(m)
			}
		}
	case map[string]any:
		fn(v)
	}
	out, err := json.Marshal(probe)
	if err != nil {
		return nil, err
	}
	return out, nil
}
