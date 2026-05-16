// Package query implements the read-only structural queries served by MCP
// tool handlers. All functions are pure: ProjectIndex in, results out.
package query

import (
	"token-savior-go/internal/models"
)

// SymbolHit is the unit return type for find_symbol.
type SymbolHit struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line"`
	Qualified string `json:"qualified"`
	Kind      string `json:"kind"` // "function" | "class"
	Signature string `json:"signature,omitempty"`
}

// SlotView is the read-only handler-facing view of a slot. Avoids a
// dependency from internal/mcp on internal/slot's concrete Slot type.
type SlotView struct {
	Root  string
	Index *models.ProjectIndex
}

// FindSymbol returns all symbols whose qualified name equals name. Empty
// slice when nothing matches; error reserved for invalid input (none today).
func FindSymbol(idx *models.ProjectIndex, name string) ([]SymbolHit, error) {
	var hits []SymbolHit
	for _, path := range idx.SortedPaths {
		md := idx.Files[path]
		for _, f := range md.Functions {
			if f.Qualified == name {
				hits = append(hits, SymbolHit{
					File: path, Line: f.Line, EndLine: f.EndLine,
					Qualified: f.Qualified, Kind: "function",
					Signature: f.Signature,
				})
			}
		}
		for _, c := range md.Classes {
			if c.Qualified == name {
				hits = append(hits, SymbolHit{
					File: path, Line: c.Line, EndLine: c.EndLine,
					Qualified: c.Qualified, Kind: "class",
				})
			}
		}
	}
	return hits, nil
}
