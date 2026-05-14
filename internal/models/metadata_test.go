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
