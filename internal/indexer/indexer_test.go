package indexer

import (
	"os"
	"path/filepath"
	"testing"

	_ "token-savior-go/internal/annotator/golang" // register Go annotator

	"github.com/stretchr/testify/require"
)

func TestProjectIndexer_BuildsIndex(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.go"), []byte(`package main

func main() { helper() }

func helper() {}
`), 0o600))

	idx, err := NewProjectIndexer(tmp).Build()
	require.NoError(t, err)

	require.Equal(t, tmp, idx.Root)
	require.Contains(t, idx.Files, "main.go")

	md := idx.Files["main.go"]
	require.Equal(t, "go", md.Language)
	require.Len(t, md.Functions, 2)

	require.Equal(t, "main.go", idx.SymbolTable["main"])
	require.Equal(t, "main.go", idx.SymbolTable["helper"])

	require.Contains(t, idx.DepGraph["main"], "helper")

	require.Contains(t, idx.BasenameMap["main.go"], "main.go")
}
