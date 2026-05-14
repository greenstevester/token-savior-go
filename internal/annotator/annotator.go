// Package annotator defines the per-language annotation interface and
// dispatches on file extension. Mirrors Python's `annotator._EXTENSION_MAP`.
package annotator

import (
	"path/filepath"
	"strings"

	"token-savior-go/internal/models"
)

// Annotator parses a single source file and produces structural metadata.
//
// Implementations live in internal/annotator/<lang>/. They MUST be safe for
// concurrent use — the indexer runs annotators in a worker pool.
type Annotator interface {
	Annotate(path string, source []byte) (*models.StructuralMetadata, error)
}

// extensionMap is the source of truth for path → language dispatch.
// Update this when a new annotator lands in a milestone.
var extensionMap = map[string]string{
	".go":   "go",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".java": "java",
	".rs":   "rust",
	".sh":   "shell",
	".bash": "shell",
	".zsh":  "shell",
}

// LanguageForPath returns the language identifier for the file at path, or
// the empty string if no annotator is registered for that extension.
func LanguageForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return extensionMap[ext]
}
