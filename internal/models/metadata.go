// Package models defines the structural data types shared across the indexer,
// query, and MCP layers. Names mirror the Python `models.StructuralMetadata`
// dataclass so the compat harness can diff outputs field-by-field.
package models

// StructuralMetadata is the per-file annotation produced by language annotators.
type StructuralMetadata struct {
	Path       string      `json:"path"`
	Language   string      `json:"language"`
	Functions  []Function  `json:"functions"`
	Classes    []Class     `json:"classes"`
	Imports    []Import    `json:"imports"`
	Calls      []SymbolRef `json:"calls"`
	References []SymbolRef `json:"references"`
}

// Function describes a function or method declaration.
//
// Receiver is empty for top-level functions and holds the receiver type name
// (e.g. "Thing" for `func (t *Thing) Do()`) for methods. Qualified is the
// canonical lookup key: bare name for funcs, "Receiver.Name" for methods.
type Function struct {
	Name      string `json:"name"`
	Receiver  string `json:"receiver,omitempty"`
	Qualified string `json:"qualified"`
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line"`
	Signature string `json:"signature"`
}

// Class describes a type declaration: struct, interface, or alias.
type Class struct {
	Name      string `json:"name"`
	Qualified string `json:"qualified"`
	Kind      string `json:"kind"` // "struct" | "interface" | "alias"
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line"`
}

// Import describes a single import declaration.
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
	Line  int    `json:"line"`
}

// SymbolRef records a directed edge between two symbols (call or reference).
type SymbolRef struct {
	From string `json:"from"` // caller qualified name
	To   string `json:"to"`   // callee qualified name (may be unresolved)
	Line int    `json:"line"`
}
