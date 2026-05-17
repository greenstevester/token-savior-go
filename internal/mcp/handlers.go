package mcp

import (
	"encoding/json"
	"fmt"

	"token-savior-go/internal/query"
)

// RegisterHandlers wires every M1 tool handler into dispatcher. Call once
// at startup after the dispatcher and ToolContext are constructed.
func RegisterHandlers(d *Dispatcher) {
	d.Register("find_symbol", findSymbolHandler)
	d.Register("get_functions", getFunctionsHandler)
	d.Register("get_classes", getClassesHandler)
	d.Register("get_imports", getImportsHandler)
	d.Register("search_codebase", searchCodebaseHandler)
	d.Register("switch_project", switchProjectHandler)
	d.Register("list_workspace_roots", listWorkspaceRootsHandler)
	d.Register("get_stats", getStatsHandler)
}

// activeSlot returns the active slot view or an error.
func activeSlot(ctx *ToolContext) (*query.SlotView, error) {
	s := ctx.SlotManager.Active()
	if s == nil {
		return nil, fmt.Errorf("no workspace root registered")
	}
	return &query.SlotView{Root: s.Root, Index: s.Index}, nil
}

func findSymbolHandler(ctx *ToolContext, raw json.RawMessage) (any, error) {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	s, err := activeSlot(ctx)
	if err != nil {
		return nil, err
	}
	return query.FindSymbol(s.Index, args.Name)
}

func getFunctionsHandler(ctx *ToolContext, raw json.RawMessage) (any, error) {
	var args struct {
		Path string `json:"path"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
	}
	s, err := activeSlot(ctx)
	if err != nil {
		return nil, err
	}
	return query.GetFunctions(s.Index, args.Path)
}

func getClassesHandler(ctx *ToolContext, raw json.RawMessage) (any, error) {
	var args struct {
		Path string `json:"path"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
	}
	s, err := activeSlot(ctx)
	if err != nil {
		return nil, err
	}
	return query.GetClasses(s.Index, args.Path)
}

func getImportsHandler(ctx *ToolContext, raw json.RawMessage) (any, error) {
	var args struct {
		Path string `json:"path"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
	}
	s, err := activeSlot(ctx)
	if err != nil {
		return nil, err
	}
	return query.GetImports(s.Index, args.Path)
}

func searchCodebaseHandler(ctx *ToolContext, raw json.RawMessage) (any, error) {
	var args struct {
		Pattern string `json:"pattern"`
		Regex   bool   `json:"regex"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	s, err := activeSlot(ctx)
	if err != nil {
		return nil, err
	}
	return query.SearchCodebase(s.Index, args.Pattern, args.Regex)
}

func switchProjectHandler(ctx *ToolContext, raw json.RawMessage) (any, error) {
	var args struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	s, err := ctx.SlotManager.Switch(args.Root)
	if err != nil {
		return nil, err
	}
	return map[string]string{"active": s.Root}, nil
}

func listWorkspaceRootsHandler(ctx *ToolContext, _ json.RawMessage) (any, error) {
	out := map[string]any{
		"roots":  ctx.SlotManager.Roots(),
		"active": "",
	}
	if a := ctx.SlotManager.Active(); a != nil {
		out["active"] = a.Root
	}
	return out, nil
}

func getStatsHandler(ctx *ToolContext, _ json.RawMessage) (any, error) {
	return ctx.Stats.Snapshot(), nil
}
