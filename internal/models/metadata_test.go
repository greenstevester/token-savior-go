package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructuralMetadata_JSONRoundTrip(t *testing.T) {
	original := StructuralMetadata{
		Path:     "internal/foo/bar.go",
		Language: "go",
		Functions: []Function{
			{Name: "DoThing", Line: 12, EndLine: 24, Signature: "func DoThing(ctx context.Context) error", Qualified: "DoThing"},
		},
		Classes: []Class{
			{Name: "Thing", Line: 30, EndLine: 60, Kind: "struct", Qualified: "Thing"},
		},
		Imports: []Import{
			{Path: "context", Line: 3},
		},
		Calls:      []SymbolRef{{From: "DoThing", To: "context.Background", Line: 14}},
		References: []SymbolRef{},
	}

	bytes, err := json.Marshal(original)
	require.NoError(t, err)

	var roundtripped StructuralMetadata
	require.NoError(t, json.Unmarshal(bytes, &roundtripped))
	require.Equal(t, original, roundtripped)
}

func TestSymbol_QualifiedName(t *testing.T) {
	f := Function{Name: "DoThing", Receiver: "", Qualified: "DoThing"}
	require.Equal(t, "DoThing", f.Qualified)

	m := Function{Name: "Do", Receiver: "Thing", Qualified: "Thing.Do"}
	require.Equal(t, "Thing.Do", m.Qualified)
}

func TestNewProjectIndex_InitialisesAllMaps(t *testing.T) {
	idx := NewProjectIndex("/tmp/foo")

	require.NotNil(t, idx)
	require.Equal(t, "/tmp/foo", idx.Root)
	require.NotNil(t, idx.Files)
	require.NotNil(t, idx.SymbolTable)
	require.NotNil(t, idx.DepGraph)
	require.NotNil(t, idx.ImportGraph)
	require.NotNil(t, idx.BasenameMap)
	require.Nil(t, idx.SortedPaths) // lazy-populated, nil is correct

	// Writes to nil maps panic — confirm none of them are nil.
	idx.Files["a.go"] = &StructuralMetadata{}
	idx.SymbolTable["X"] = "a.go"
	idx.DepGraph["X"] = map[string]struct{}{}
	idx.ImportGraph["a.go"] = map[string]struct{}{}
	idx.BasenameMap["a.go"] = []string{"a.go"}
}
