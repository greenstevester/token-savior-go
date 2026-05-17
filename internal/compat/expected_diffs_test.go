package compat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── search_codebase ───────────────────────────────────────────────────────────

func TestNormalizeSearchCodebase_Match(t *testing.T) {
	want := json.RawMessage(`[{"content":"foo bar","file":"a.go","line_number":1}]`)
	got := json.RawMessage(`[{"file":"a.go","line":1,"text":"foo bar"}]`)
	w, g, err := ToolExpectations["search_codebase"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.Empty(t, diffs, "should have no diffs after normalisation")
}

func TestNormalizeSearchCodebase_RealDiffSurfaces(t *testing.T) {
	// Different text content — real data difference, not shape difference.
	want := json.RawMessage(`[{"content":"something else","file":"a.go","line_number":1}]`)
	got := json.RawMessage(`[{"file":"a.go","line":1,"text":"foo bar"}]`)
	w, g, err := ToolExpectations["search_codebase"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "data divergence should still surface after normalisation")
}

// ── get_imports ───────────────────────────────────────────────────────────────

func TestNormalizeGetImports_Match(t *testing.T) {
	want := json.RawMessage(`[{"file":"main.go","is_from_import":false,"line":3,"module":"fmt","names":["fmt"]}]`)
	got := json.RawMessage(`[{"file":"main.go","line":3,"path":"fmt"}]`)
	w, g, err := ToolExpectations["get_imports"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.Empty(t, diffs, "should have no diffs after normalisation")
}

func TestNormalizeGetImports_RealDiffSurfaces(t *testing.T) {
	// Different module/path — real data difference.
	want := json.RawMessage(`[{"file":"main.go","is_from_import":false,"line":3,"module":"fmt","names":["fmt"]}]`)
	got := json.RawMessage(`[{"file":"main.go","line":3,"path":"os"}]`)
	w, g, err := ToolExpectations["get_imports"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "data divergence should still surface after normalisation")
}

// ── get_classes ───────────────────────────────────────────────────────────────

func TestNormalizeGetClasses_Match(t *testing.T) {
	want := json.RawMessage(`[{"bases":[],"file":"main.go","lines":[5,7],"method_signatures":["Greeter.Hello"],"methods":["Hello"],"name":"Greeter","qualified_name":"Greeter"}]`)
	got := json.RawMessage(`[{"end_line":7,"file":"main.go","kind":"struct","line":5,"qualified":"Greeter"}]`)
	w, g, err := ToolExpectations["get_classes"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.Empty(t, diffs, "should have no diffs after normalisation")
}

func TestNormalizeGetClasses_RealDiffSurfaces(t *testing.T) {
	// Different line range — real data difference.
	want := json.RawMessage(`[{"bases":[],"file":"main.go","lines":[5,99],"method_signatures":[],"methods":[],"name":"Greeter","qualified_name":"Greeter"}]`)
	got := json.RawMessage(`[{"end_line":7,"file":"main.go","kind":"struct","line":5,"qualified":"Greeter"}]`)
	w, g, err := ToolExpectations["get_classes"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "data divergence should still surface after normalisation")
}

// ── get_functions ─────────────────────────────────────────────────────────────

func TestNormalizeGetFunctions_Match(t *testing.T) {
	want := json.RawMessage(`[{"file":"main.go","is_method":true,"lines":[9,11],"name":"Hello","params":[],"parent_class":"Greeter","qualified_name":"Greeter.Hello"}]`)
	got := json.RawMessage(`[{"end_line":11,"file":"main.go","line":9,"qualified":"Greeter.Hello","signature":"func (g *Greeter) Hello() string"}]`)
	w, g, err := ToolExpectations["get_functions"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.Empty(t, diffs, "should have no diffs after normalisation")
}

func TestNormalizeGetFunctions_RealDiffSurfaces(t *testing.T) {
	// Different qualified name — real data difference.
	want := json.RawMessage(`[{"file":"main.go","is_method":true,"lines":[9,11],"name":"Hello","params":[],"parent_class":"Greeter","qualified_name":"Greeter.Hello"}]`)
	got := json.RawMessage(`[{"end_line":11,"file":"main.go","line":9,"qualified":"Greeter.Goodbye","signature":"func (g *Greeter) Goodbye() string"}]`)
	w, g, err := ToolExpectations["get_functions"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "data divergence should still surface after normalisation")
}

// ── find_symbol ───────────────────────────────────────────────────────────────

func TestNormalizeFindSymbol_Match(t *testing.T) {
	// Python returns a single dict at level=0; Go returns []SymbolHit.
	// After normalisation both sides are reduced to {file, line, qualified}.
	// signature is dropped because Python emits Python-style sigs for Go code.
	want := json.RawMessage(`{"complete":true,"end_line":11,"file":"main.go","line":9,"name":"Greeter.Hello","signature":"def Hello()","source_preview":"func (g *Greeter) Hello() string {\n\treturn fmt.Sprintf(\"hi %s\", g.name)\n}","type":"method"}`)
	got := json.RawMessage(`[{"end_line":11,"file":"main.go","kind":"function","line":9,"qualified":"Greeter.Hello","signature":"func (g *Greeter) Hello() string"}]`)
	w, g, err := ToolExpectations["find_symbol"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.Empty(t, diffs, "should have no diffs after normalisation")
}

func TestNormalizeFindSymbol_RealDiffSurfaces(t *testing.T) {
	// Different file — real data difference must survive normalisation.
	want := json.RawMessage(`{"complete":true,"end_line":11,"file":"other.go","line":9,"name":"Greeter.Hello","signature":"def Hello()","source_preview":"...","type":"method"}`)
	got := json.RawMessage(`[{"end_line":11,"file":"main.go","kind":"function","line":9,"qualified":"Greeter.Hello","signature":"func (g *Greeter) Hello() string"}]`)
	w, g, err := ToolExpectations["find_symbol"].Normalize(want, got)
	require.NoError(t, err)
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "data divergence should still surface after normalisation")
}

func TestNormalizeFindSymbol_ErrorDict(t *testing.T) {
	// Python error path: single dict with "error" key.
	// After wrapping, it becomes a one-element array.
	want := json.RawMessage(`{"error":"symbol 'Foo' not found","scanned_files":1,"complete":true}`)
	got := json.RawMessage(`[]`)
	w, g, err := ToolExpectations["find_symbol"].Normalize(want, got)
	require.NoError(t, err)
	// Both sides should now be arrays; they'll differ in content (that's expected —
	// this verifies the normaliser doesn't crash on error dicts).
	diffs, err := DiffJSON(w, g)
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "error vs empty-array should still show a diff")
}

// ── list_workspace_roots (skip) ───────────────────────────────────────────────

func TestListWorkspaceRootsIsSkipped(t *testing.T) {
	exp := ToolExpectations["list_workspace_roots"]
	require.True(t, exp.Skip, "list_workspace_roots should be marked Skip")
	require.NotEmpty(t, exp.Reason, "Skip entry must have a Reason")
	require.Nil(t, exp.Normalize, "Skip entry should have no Normalize function")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func TestWrapDictAsArray_SingleObject(t *testing.T) {
	raw := json.RawMessage(`{"a":1}`)
	out, err := wrapDictAsArray(raw)
	require.NoError(t, err)
	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	require.Len(t, result, 1)
	require.Equal(t, float64(1), result[0]["a"])
}

func TestWrapDictAsArray_ArrayPassthrough(t *testing.T) {
	raw := json.RawMessage(`[{"a":1},{"a":2}]`)
	out, err := wrapDictAsArray(raw)
	require.NoError(t, err)
	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	require.Len(t, result, 2)
}

func TestLinesToLineEndline(t *testing.T) {
	raw := json.RawMessage(`[{"lines":[5,7],"file":"a.go"}]`)
	out, err := linesToLineEndline(raw)
	require.NoError(t, err)
	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	require.Equal(t, float64(5), result[0]["line"])
	require.Equal(t, float64(7), result[0]["end_line"])
	_, hasLines := result[0]["lines"]
	require.False(t, hasLines, "'lines' key should be removed")
}

func TestDropKeys(t *testing.T) {
	raw := json.RawMessage(`[{"a":1,"b":2,"c":3}]`)
	out, err := dropKeys(raw, "b", "c")
	require.NoError(t, err)
	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	require.Equal(t, float64(1), result[0]["a"])
	_, hasB := result[0]["b"]
	require.False(t, hasB)
	_, hasC := result[0]["c"]
	require.False(t, hasC)
}

func TestRenameKeys(t *testing.T) {
	raw := json.RawMessage(`[{"old_name":42,"keep":1}]`)
	out, err := renameKeys(raw, map[string]string{"old_name": "new_name"})
	require.NoError(t, err)
	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	require.Equal(t, float64(42), result[0]["new_name"])
	_, hasOld := result[0]["old_name"]
	require.False(t, hasOld)
	require.Equal(t, float64(1), result[0]["keep"])
}
