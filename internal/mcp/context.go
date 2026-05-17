// Package mcp wires the MCP stdio server to the token-savior tool catalog.
package mcp

import (
	"encoding/json"

	"token-savior-go/internal/slot"
	"token-savior-go/internal/stats"
)

// ToolContext is the dependency bundle passed to every tool handler.
//
// Handlers reach into it for what they need: SlotManager for slot/index
// access, Stats for counter updates. Future milestones add Memory engine
// and OptEngines fields.
type ToolContext struct {
	SlotManager *slot.Manager
	Stats       *stats.Counters
}

// Handler is the unified tool-handler signature.
//
// Returns any JSON-serialisable value (or a struct with json tags) plus an
// error. The dispatcher wraps the value into the MCP TextContent envelope.
type Handler func(ctx *ToolContext, args json.RawMessage) (any, error)
