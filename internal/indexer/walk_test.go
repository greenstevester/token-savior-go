package indexer

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWalk_SkipsExcludedDirs(t *testing.T) {
	tmp := t.TempDir()
	mustWrite(t, tmp, "main.go", "package main\n")
	mustWrite(t, tmp, "sub/foo.go", "package sub\n")
	mustWrite(t, tmp, "node_modules/lib/index.js", "")
	mustWrite(t, tmp, ".git/HEAD", "ref: refs/heads/main\n")

	paths, err := Walk(tmp)
	require.NoError(t, err)

	sort.Strings(paths)
	require.Equal(t, []string{"main.go", "sub/foo.go"}, paths)
}

func TestWalk_OnlyReturnsAnnotatableExtensions(t *testing.T) {
	tmp := t.TempDir()
	mustWrite(t, tmp, "a.go", "package a")
	mustWrite(t, tmp, "b.ts", "")
	mustWrite(t, tmp, "c.unknown", "")
	mustWrite(t, tmp, "d.md", "")

	paths, err := Walk(tmp)
	require.NoError(t, err)

	sort.Strings(paths)
	require.Equal(t, []string{"a.go", "b.ts"}, paths)
}

// mustWrite writes content to root/rel, creating intermediate dirs.
func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}
