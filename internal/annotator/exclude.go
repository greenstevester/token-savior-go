package annotator

import "strings"

// excludedDirs are directory names that, when seen at any depth, cause the
// indexer to skip the entire subtree. Mirrors Python's `EXCLUDED_DIRS`.
var excludedDirs = map[string]struct{}{
	".token-savior-checkpoints": {},
	".git":                      {},
	"__pycache__":               {},
	"node_modules":              {},
}

// IsPathExcludedFromScans returns true when path lies under any excluded
// directory. Forward and back slashes are both accepted.
func IsPathExcludedFromScans(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, part := range strings.Split(normalized, "/") {
		if _, ok := excludedDirs[part]; ok {
			return true
		}
	}
	return false
}
