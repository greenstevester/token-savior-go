package query

import (
	"testing"

	_ "token-savior-go/internal/annotator/golang"

	"github.com/stretchr/testify/require"
)

func TestGetFunctions_All(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"a.go": "package a\nfunc A(){}\nfunc B(){}\n",
		"b.go": "package b\nfunc C(){}\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := GetFunctions(idx, "")
	require.NoError(t, err)
	require.Len(t, got, 3)
}

func TestGetFunctions_FilterByPath(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"a.go": "package a\nfunc A(){}\n",
		"b.go": "package b\nfunc B(){}\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := GetFunctions(idx, "a.go")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "A", got[0].Qualified)
}

func TestGetClasses(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"a.go": "package a\ntype X struct{}\ntype Y interface{Do()}\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := GetClasses(idx, "")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestGetImports(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"a.go": "package a\nimport \"context\"\nimport \"fmt\"\n",
		"b.go": "package b\nimport \"errors\"\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := GetImports(idx, "")
	require.NoError(t, err)
	require.Len(t, got, 3)
}
