package indexer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSymbolHash_Deterministic(t *testing.T) {
	body := "func DoThing(ctx context.Context) error { return nil }"
	a := SymbolHash(body)
	b := SymbolHash(body)
	require.Equal(t, a, b)
	require.Len(t, a, 16) // first 8 bytes hex-encoded
}

func TestSymbolHash_DifferentOnChange(t *testing.T) {
	a := SymbolHash("func A() {}")
	b := SymbolHash("func B() {}")
	require.NotEqual(t, a, b)
}
