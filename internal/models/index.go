package models

// ProjectIndex is the in-memory index for a single project root.
//
// Files maps relative paths to per-file structural metadata.
// SymbolTable maps qualified names to file paths (one path per name; collisions
// across files are recorded as the first-seen winner — query layer handles
// disambiguation).
// DepGraph maps a caller-qualified-name to the set of callees it references.
// ImportGraph maps a file path to the paths of files it imports.
// BasenameMap is `basename(path) -> []path` for fast basename lookups.
// SortedPaths is the lexicographically sorted slice of Files keys, used for
// deterministic iteration.
type ProjectIndex struct {
	Root        string                         `json:"root"`
	Files       map[string]*StructuralMetadata `json:"files"`
	SymbolTable map[string]string              `json:"symbol_table"`
	DepGraph    map[string]map[string]struct{} `json:"-"`
	ImportGraph map[string]map[string]struct{} `json:"-"`
	BasenameMap map[string][]string            `json:"-"`
	SortedPaths []string                       `json:"-"`
}

// NewProjectIndex returns an empty index with all maps initialised.
func NewProjectIndex(root string) *ProjectIndex {
	return &ProjectIndex{
		Root:        root,
		Files:       make(map[string]*StructuralMetadata),
		SymbolTable: make(map[string]string),
		DepGraph:    make(map[string]map[string]struct{}),
		ImportGraph: make(map[string]map[string]struct{}),
		BasenameMap: make(map[string][]string),
	}
}
