package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "token-savior-go/internal/annotator/golang"
	"token-savior-go/internal/slot"
	"token-savior-go/internal/stats"

	"github.com/stretchr/testify/require"
)

func setupCtx(t *testing.T, files map[string]string) *ToolContext {
	t.Helper()
	tmp := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(tmp, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}
	mgr := slot.NewManager()
	require.NoError(t, mgr.RegisterRoot(tmp))
	return &ToolContext{SlotManager: mgr, Stats: stats.NewCounters()}
}

func TestHandler_FindSymbol(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"a.go": "package a\nfunc DoThing() {}\n",
	})
	d := NewDispatcher(ctx)
	RegisterHandlers(d)

	result, err := d.Dispatch("find_symbol", json.RawMessage(`{"name": "DoThing"}`))
	require.NoError(t, err)
	raw, merr := json.Marshal(result)
	require.NoError(t, merr)
	require.Contains(t, string(raw), `"DoThing"`)
	require.Contains(t, string(raw), `"a.go"`)
}

func TestHandler_GetFunctions(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"a.go": "package a\nfunc A(){}\nfunc B(){}\n",
	})
	d := NewDispatcher(ctx)
	RegisterHandlers(d)

	result, err := d.Dispatch("get_functions", json.RawMessage(`{}`))
	require.NoError(t, err)
	raw, merr := json.Marshal(result)
	require.NoError(t, merr)
	require.Contains(t, string(raw), `"A"`)
	require.Contains(t, string(raw), `"B"`)
}

func TestHandler_SwitchProject(t *testing.T) {
	tmp1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp1, "x.go"), []byte("package x"), 0o600))
	tmp2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp2, "y.go"), []byte("package y"), 0o600))

	mgr := slot.NewManager()
	require.NoError(t, mgr.RegisterRoot(tmp1))
	require.NoError(t, mgr.RegisterRoot(tmp2))

	d := NewDispatcher(&ToolContext{SlotManager: mgr, Stats: stats.NewCounters()})
	RegisterHandlers(d)

	args := json.RawMessage(`{"root":"` + tmp2 + `"}`)
	_, err := d.Dispatch("switch_project", args)
	require.NoError(t, err)
	require.Equal(t, tmp2, mgr.Active().Root)
}

func TestHandler_ListWorkspaceRoots(t *testing.T) {
	ctx := setupCtx(t, map[string]string{"a.go": "package a"})
	d := NewDispatcher(ctx)
	RegisterHandlers(d)

	result, err := d.Dispatch("list_workspace_roots", json.RawMessage(`{}`))
	require.NoError(t, err)
	raw, merr := json.Marshal(result)
	require.NoError(t, merr)
	require.Contains(t, string(raw), `"active"`)
	require.Contains(t, string(raw), `"roots"`)
}

func TestHandler_GetStats(t *testing.T) {
	ctx := setupCtx(t, map[string]string{"a.go": "package a"})
	d := NewDispatcher(ctx)
	RegisterHandlers(d)

	_, err := d.Dispatch("find_symbol", json.RawMessage(`{"name":"none"}`))
	require.NoError(t, err)
	result, err := d.Dispatch("get_stats", json.RawMessage(`{}`))
	require.NoError(t, err)
	raw, merr := json.Marshal(result)
	require.NoError(t, merr)
	require.Contains(t, string(raw), `"tool_calls"`)
}

func TestHandler_SearchCodebase(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"a.go": "package a\nfunc DoThing(){}\n",
	})
	d := NewDispatcher(ctx)
	RegisterHandlers(d)

	result, err := d.Dispatch("search_codebase", json.RawMessage(`{"pattern":"DoThing","regex":false}`))
	require.NoError(t, err)
	raw, merr := json.Marshal(result)
	require.NoError(t, merr)
	require.Contains(t, string(raw), `"a.go"`)
}
