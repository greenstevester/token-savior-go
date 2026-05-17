package compat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiff_Matching(t *testing.T) {
	pyOut := json.RawMessage(`[{"file":"a.go","line":3,"qualified":"DoThing"}]`)
	goOut := json.RawMessage(`[{"file":"a.go","line":3,"qualified":"DoThing"}]`)
	diffs, err := DiffJSON(pyOut, goOut)
	require.NoError(t, err)
	require.Empty(t, diffs)
}

func TestDiff_FieldMismatch(t *testing.T) {
	pyOut := json.RawMessage(`[{"file":"a.go","line":3,"qualified":"DoThing"}]`)
	goOut := json.RawMessage(`[{"file":"a.go","line":4,"qualified":"DoThing"}]`)
	diffs, err := DiffJSON(pyOut, goOut)
	require.NoError(t, err)
	require.NotEmpty(t, diffs)
}

func TestDiff_LengthMismatch(t *testing.T) {
	pyOut := json.RawMessage(`[{"qualified":"A"}, {"qualified":"B"}]`)
	goOut := json.RawMessage(`[{"qualified":"A"}]`)
	diffs, err := DiffJSON(pyOut, goOut)
	require.NoError(t, err)
	require.NotEmpty(t, diffs)
}

func TestDiff_OrderIndependent(t *testing.T) {
	pyOut := json.RawMessage(`[{"qualified":"A"},{"qualified":"B"}]`)
	goOut := json.RawMessage(`[{"qualified":"B"},{"qualified":"A"}]`)
	diffs, err := DiffJSON(pyOut, goOut)
	require.NoError(t, err)
	require.Empty(t, diffs)
}
