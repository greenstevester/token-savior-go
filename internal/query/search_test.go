package query

import (
	"testing"

	_ "token-savior-go/internal/annotator/golang"

	"github.com/stretchr/testify/require"
)

func TestSearchCodebase_Literal(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"a.go": "package a\nfunc DoThing() {\n\thelper()\n}\nfunc helper() {}\n",
		"b.go": "package b\nfunc Other() {}\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := SearchCodebase(idx, "helper", false)
	require.NoError(t, err)
	require.Len(t, got, 2) // call line + def line
	require.Equal(t, "a.go", got[0].File)
}

func TestSearchCodebase_Regex(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"a.go": "package a\nfunc DoThing(){}\nfunc DoOther(){}\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := SearchCodebase(idx, `^func Do\w+`, true)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestSearchCodebase_NoMatch(t *testing.T) {
	ix := newTestIndex(t, map[string]string{"a.go": "package a"})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := SearchCodebase(idx, "nothing", false)
	require.NoError(t, err)
	require.Empty(t, got)
}
