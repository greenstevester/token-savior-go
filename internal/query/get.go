package query

import (
	"strings"

	"token-savior-go/internal/models"
)

// FunctionHit is the unit return type for get_functions.
type FunctionHit struct {
	File      string `json:"file"`
	Qualified string `json:"qualified"`
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line"`
	Signature string `json:"signature,omitempty"`
}

// ClassHit is the unit return type for get_classes.
type ClassHit struct {
	File      string `json:"file"`
	Qualified string `json:"qualified"`
	Kind      string `json:"kind"`
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line"`
}

// ImportHit is the unit return type for get_imports.
type ImportHit struct {
	File  string `json:"file"`
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
	Line  int    `json:"line"`
}

// GetFunctions returns every function in the index. When pathFilter is
// non-empty, results are restricted to files whose path equals or has
// pathFilter as a prefix (so "sub/" filters to a subdirectory).
func GetFunctions(idx *models.ProjectIndex, pathFilter string) ([]FunctionHit, error) {
	var out []FunctionHit
	for _, p := range idx.SortedPaths {
		if !pathMatches(p, pathFilter) {
			continue
		}
		md := idx.Files[p]
		for _, f := range md.Functions {
			out = append(out, FunctionHit{
				File: p, Qualified: f.Qualified, Line: f.Line, EndLine: f.EndLine,
				Signature: f.Signature,
			})
		}
	}
	return out, nil
}

// GetClasses returns every class (struct/interface/alias). pathFilter as above.
func GetClasses(idx *models.ProjectIndex, pathFilter string) ([]ClassHit, error) {
	var out []ClassHit
	for _, p := range idx.SortedPaths {
		if !pathMatches(p, pathFilter) {
			continue
		}
		md := idx.Files[p]
		for _, c := range md.Classes {
			out = append(out, ClassHit{
				File: p, Qualified: c.Qualified, Kind: c.Kind, Line: c.Line, EndLine: c.EndLine,
			})
		}
	}
	return out, nil
}

// GetImports returns every import in the index. pathFilter as above.
func GetImports(idx *models.ProjectIndex, pathFilter string) ([]ImportHit, error) {
	var out []ImportHit
	for _, p := range idx.SortedPaths {
		if !pathMatches(p, pathFilter) {
			continue
		}
		md := idx.Files[p]
		for _, im := range md.Imports {
			out = append(out, ImportHit{
				File: p, Path: im.Path, Alias: im.Alias, Line: im.Line,
			})
		}
	}
	return out, nil
}

func pathMatches(path, filter string) bool {
	if filter == "" {
		return true
	}
	if path == filter {
		return true
	}
	if strings.HasSuffix(filter, "/") {
		return strings.HasPrefix(path, filter)
	}
	return strings.HasPrefix(path, filter+"/") || path == filter
}
