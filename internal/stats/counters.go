// Package stats tracks per-session tool-call counters and char totals.
// Output is exposed via the get_stats MCP tool.
package stats

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Snapshot is the immutable read-out of counter state.
type Snapshot struct {
	SessionID    string           `json:"session_id"`
	SessionStart time.Time        `json:"session_start"`
	ToolCalls    map[string]int64 `json:"tool_calls"`
	TotalChars   int64            `json:"total_chars"`
}

// Counters is the in-memory session counter store. Safe for concurrent use.
type Counters struct {
	mu         sync.Mutex
	sessionID  string
	startedAt  time.Time
	toolCalls  map[string]int64
	totalChars int64
}

// NewCounters returns a fresh counter set with a random 12-hex session ID.
func NewCounters() *Counters {
	return &Counters{
		sessionID: randHex(6),
		startedAt: time.Now().UTC(),
		toolCalls: make(map[string]int64),
	}
}

// RecordCall increments the call counter for tool and adds chars to the
// running total. chars may be negative if a tool returns nothing useful;
// callers normally pass the byte length of the wrapped result.
func (c *Counters) RecordCall(tool string, chars int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.toolCalls[tool]++
	c.totalChars += chars
}

// Snapshot returns a deep-copied view of current state.
func (c *Counters) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	calls := make(map[string]int64, len(c.toolCalls))
	for k, v := range c.toolCalls {
		calls[k] = v
	}
	return Snapshot{
		SessionID:    c.sessionID,
		SessionStart: c.startedAt,
		ToolCalls:    calls,
		TotalChars:   c.totalChars,
	}
}

func randHex(byteLen int) string {
	buf := make([]byte, byteLen)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
