// Package tools defines the MCP tool registry. Each ToolSchema is the single
// source of truth for one tool's wire description (name + JSON Schema) and
// for the profile bitmask that decides which profiles advertise it.
package tools

import (
	"encoding/json"
)

// ProfileSet is a bitmask of profiles in which a tool is advertised.
type ProfileSet uint8

const (
	ProfileFull     ProfileSet = 1 << iota // every tool
	ProfileCore                            // daily coding, no memory engine
	ProfileNav                             // read-only exploration
	ProfileLean                            // memory off (matches Python bench v2)
	ProfileUltra                           // hot tools + ts_extended proxy
	ProfileTiny                            // 6 tools, defer-loading via ts_search
	ProfileTinyPlus                        // tiny + 9 hot tools
)

// AllProfiles is the union — convenience constant for tools every profile sees.
const AllProfiles = ProfileFull | ProfileCore | ProfileNav | ProfileLean |
	ProfileUltra | ProfileTiny | ProfileTinyPlus

// Has returns true when target is part of the set.
func (p ProfileSet) Has(target ProfileSet) bool { return p&target != 0 }

// ToolSchema describes one MCP tool.
type ToolSchema struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	Profiles    ProfileSet
}

// Registry is the in-memory tool catalog.
type Registry struct {
	byName map[string]ToolSchema
}

// DefaultRegistry returns the M1 tool catalog. Add tools by editing the
// schemas slice below; the registry is built once at package init and is
// otherwise immutable.
func DefaultRegistry() *Registry {
	r := &Registry{byName: make(map[string]ToolSchema, len(m1Schemas))}
	for _, s := range m1Schemas {
		r.byName[s.Name] = s
	}
	return r
}

// Lookup returns the schema for name, or (zero, false) when absent.
func (r *Registry) Lookup(name string) (ToolSchema, bool) {
	s, ok := r.byName[name]
	return s, ok
}

// All returns every registered schema. Order is unspecified.
func (r *Registry) All() []ToolSchema {
	out := make([]ToolSchema, 0, len(r.byName))
	for _, s := range r.byName {
		out = append(out, s)
	}
	return out
}

// m1Schemas is the catalog. Every M1 tool is in every profile.
var m1Schemas = []ToolSchema{
	{
		Name:        "find_symbol",
		Description: "Locate a symbol by exact qualified name. Returns []SymbolHit (file, line, kind, signature). Use INSTEAD of grep when you know the name.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string", "description": "Exact qualified name. Methods use 'Receiver.Method'."}
			},
			"required": ["name"]
		}`),
		Profiles: AllProfiles,
	},
	{
		Name:        "get_functions",
		Description: "List every function in the project, optionally filtered by file or path prefix.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "File path or directory prefix; empty for the whole project."}
			}
		}`),
		Profiles: AllProfiles,
	},
	{
		Name:        "get_classes",
		Description: "List every type (struct/interface/alias) in the project, optionally filtered by file or path prefix.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string"}
			}
		}`),
		Profiles: AllProfiles,
	},
	{
		Name:        "get_imports",
		Description: "List every import declaration in the project, optionally filtered by file or path prefix.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string"}
			}
		}`),
		Profiles: AllProfiles,
	},
	{
		Name:        "search_codebase",
		Description: "Literal or regex line-anchored search across all indexed files. Capped at 500 hits.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"pattern": {"type": "string", "description": "Literal string or Go regex."},
				"regex":   {"type": "boolean", "description": "True to interpret pattern as a regex."}
			},
			"required": ["pattern"]
		}`),
		Profiles: AllProfiles,
	},
	{
		Name:        "switch_project",
		Description: "Set the active project root. Idempotent. Returns the active root after switching.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"root": {"type": "string", "description": "Absolute path of an already-registered root."}
			},
			"required": ["root"]
		}`),
		Profiles: AllProfiles,
	},
	{
		Name:        "list_workspace_roots",
		Description: "Return the registered workspace roots and the currently-active one.",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
		Profiles:    AllProfiles,
	},
	{
		Name:        "get_stats",
		Description: "Return session counters: tool call counts, total chars returned, session start/duration.",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
		Profiles:    AllProfiles,
	},
}
