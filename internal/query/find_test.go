package query

import (
	"os"
	"path/filepath"
	"testing"

	_ "token-savior-go/internal/annotator/golang"
	"token-savior-go/internal/indexer"

	"github.com/stretchr/testify/require"
)

func newTestIndex(t *testing.T, files map[string]string) *indexer.ProjectIndexer {
	t.Helper()
	tmp := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(tmp, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}
	return indexer.NewProjectIndexer(tmp)
}

func TestFindSymbol_ByQualifiedName(t *testing.T) {
	ix := newTestIndex(t, map[string]string{
		"main.go":  "package main\nfunc main(){}\nfunc helper(){}\n",
		"sub/x.go": "package sub\ntype Thing struct{}\nfunc (t *Thing) Do(){}\n",
	})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := FindSymbol(idx, "helper")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "main.go", got[0].File)
	require.Equal(t, "helper", got[0].Qualified)
	require.Equal(t, "function", got[0].Kind)

	got, err = FindSymbol(idx, "Thing.Do")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "sub/x.go", got[0].File)
	require.Equal(t, "function", got[0].Kind)

	got, err = FindSymbol(idx, "Thing")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "class", got[0].Kind)
}

func TestFindSymbol_NotFound(t *testing.T) {
	ix := newTestIndex(t, map[string]string{"main.go": "package main"})
	idx, err := ix.Build()
	require.NoError(t, err)

	got, err := FindSymbol(idx, "nonexistent")
	require.NoError(t, err)
	require.Empty(t, got)
}
