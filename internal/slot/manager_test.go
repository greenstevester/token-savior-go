package slot

import (
	"os"
	"path/filepath"
	"testing"

	_ "token-savior-go/internal/annotator/golang"

	"github.com/stretchr/testify/require"
)

func TestManager_RegisterAndLookup(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "a.go"), []byte("package a\nfunc A(){}\n"), 0o600))

	m := NewManager()
	require.NoError(t, m.RegisterRoot(tmp))

	got, ok := m.Get(tmp)
	require.True(t, ok)
	require.Equal(t, tmp, got.Root)
	require.NotNil(t, got.Index)
	require.Contains(t, got.Index.Files, "a.go")
}

func TestManager_SwitchActive(t *testing.T) {
	tmp1 := t.TempDir()
	tmp2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp1, "x.go"), []byte("package x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tmp2, "y.go"), []byte("package y"), 0o600))

	m := NewManager()
	require.NoError(t, m.RegisterRoot(tmp1))
	require.NoError(t, m.RegisterRoot(tmp2))

	// Default active is the first registered.
	require.Equal(t, tmp1, m.Active().Root)

	// Switching is idempotent and returns the slot.
	s, err := m.Switch(tmp2)
	require.NoError(t, err)
	require.Equal(t, tmp2, s.Root)
	require.Equal(t, tmp2, m.Active().Root)

	// Switching to current active is a no-op.
	s2, err := m.Switch(tmp2)
	require.NoError(t, err)
	require.Same(t, s, s2)
}

func TestParseWorkspaceRoots(t *testing.T) {
	require.Equal(t, []string{"/a"}, ParseWorkspaceRoots("/a"))
	require.Equal(t, []string{"/a", "/b"}, ParseWorkspaceRoots("/a,/b"))
	require.Equal(t, []string{"/a", "/b"}, ParseWorkspaceRoots(" /a , /b "))
	require.Empty(t, ParseWorkspaceRoots(""))
}
