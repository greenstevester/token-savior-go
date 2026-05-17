package mcp

import (
	"encoding/json"
	"testing"

	"token-savior-go/internal/slot"
	"token-savior-go/internal/stats"

	"github.com/stretchr/testify/require"
)

func TestDispatcher_RoutesToHandler(t *testing.T) {
	d := NewDispatcher(&ToolContext{
		SlotManager: slot.NewManager(),
		Stats:       stats.NewCounters(),
	})
	d.Register("echo", func(ctx *ToolContext, args json.RawMessage) (any, error) {
		return map[string]any{"echo": string(args)}, nil
	})

	got, err := d.Dispatch("echo", json.RawMessage(`{"x": 1}`))
	require.NoError(t, err)
	gotMap, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, `{"x": 1}`, gotMap["echo"])
}

func TestDispatcher_UnknownToolErrors(t *testing.T) {
	d := NewDispatcher(&ToolContext{
		SlotManager: slot.NewManager(),
		Stats:       stats.NewCounters(),
	})
	_, err := d.Dispatch("nonexistent", nil)
	require.Error(t, err)
}

func TestDispatcher_RecordsCallInStats(t *testing.T) {
	counters := stats.NewCounters()
	d := NewDispatcher(&ToolContext{
		SlotManager: slot.NewManager(),
		Stats:       counters,
	})
	d.Register("noop", func(ctx *ToolContext, args json.RawMessage) (any, error) {
		return "ok", nil
	})
	_, err := d.Dispatch("noop", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), counters.Snapshot().ToolCalls["noop"])
}
