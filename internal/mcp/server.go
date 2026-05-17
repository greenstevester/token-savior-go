package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"token-savior-go/internal/tools"
)

// Serve registers every advertised tool from registry against srv, wiring
// each one through dispatcher. profile decides which tools are advertised
// in tools/list; hidden tools remain dispatchable by name.
//
// Caller wires srv to a stdio transport before/after this call.
func Serve(srv *server.MCPServer, dispatcher *Dispatcher, registry *tools.Registry, profile tools.ProfileSet) error {
	visible := tools.VisibleTools(registry, profile)
	for _, schema := range visible {
		schema := schema // capture for closure
		// Validate that InputSchema is parseable JSON — defense in depth on
		// top of the registry-level test.
		var probe any
		if err := json.Unmarshal(schema.InputSchema, &probe); err != nil {
			return fmt.Errorf("invalid input schema for %s: %w", schema.Name, err)
		}
		srv.AddTool(
			mcp.NewTool(schema.Name,
				mcp.WithDescription(schema.Description),
				mcp.WithRawInputSchema(schema.InputSchema),
			),
			func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				raw, err := json.Marshal(req.Params.Arguments)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal args: %v", err)), nil
				}
				result, dispErr := dispatcher.Dispatch(schema.Name, raw)
				if dispErr != nil {
					return mcp.NewToolResultError(dispErr.Error()), nil
				}
				resultRaw, jerr := json.Marshal(result)
				if jerr != nil {
					return mcp.NewToolResultError(jerr.Error()), nil
				}
				return mcp.NewToolResultText(string(resultRaw)), nil
			},
		)
	}
	return nil
}
