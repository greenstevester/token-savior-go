package stats

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCounters_Increment(t *testing.T) {
	c := NewCounters()
	c.RecordCall("find_symbol", 120)
	c.RecordCall("find_symbol", 200)
	c.RecordCall("get_functions", 80)

	snap := c.Snapshot()
	require.Equal(t, int64(2), snap.ToolCalls["find_symbol"])
	require.Equal(t, int64(1), snap.ToolCalls["get_functions"])
	require.Equal(t, int64(400), snap.TotalChars)
}

func TestCounters_ConcurrentSafe(t *testing.T) {
	c := NewCounters()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.RecordCall("find_symbol", 1)
		}()
	}
	wg.Wait()
	require.Equal(t, int64(100), c.Snapshot().ToolCalls["find_symbol"])
}

func TestSnapshot_JSONShape(t *testing.T) {
	c := NewCounters()
	c.RecordCall("find_symbol", 50)
	raw, err := json.Marshal(c.Snapshot())
	require.NoError(t, err)
	require.Contains(t, string(raw), `"tool_calls"`)
	require.Contains(t, string(raw), `"total_chars"`)
	require.Contains(t, string(raw), `"session_id"`)
}
