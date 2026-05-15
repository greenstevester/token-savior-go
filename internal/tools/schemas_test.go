package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_M1ToolsPresent(t *testing.T) {
	r := DefaultRegistry()
	for _, name := range []string{
		"find_symbol", "get_functions", "get_classes", "get_imports",
		"search_codebase", "switch_project", "list_workspace_roots", "get_stats",
	} {
		_, ok := r.Lookup(name)
		require.True(t, ok, "tool %q missing from registry", name)
	}
}

func TestRegistry_NoDuplicateNames(t *testing.T) {
	r := DefaultRegistry()
	seen := map[string]struct{}{}
	for _, schema := range r.All() {
		_, dup := seen[schema.Name]
		require.False(t, dup, "duplicate tool name: %s", schema.Name)
		seen[schema.Name] = struct{}{}
	}
}

func TestSchema_InputSchemaIsValidJSON(t *testing.T) {
	r := DefaultRegistry()
	for _, s := range r.All() {
		var v any
		require.NoError(t, json.Unmarshal(s.InputSchema, &v), "tool %s has invalid input schema JSON", s.Name)
	}
}
