// Package indexer builds and maintains the per-project structural index.
package indexer

import (
	"io/fs"
	"path/filepath"

	"token-savior-go/internal/annotator"
)

// Walk traverses root and returns project-relative paths of files that are
// annotatable (have a registered language) and not under an excluded dir.
// Paths use forward slashes regardless of OS.
func Walk(root string) ([]string, error) {
	var paths []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if annotator.IsPathExcludedFromScans(rel + "/") {
				return fs.SkipDir
			}
			return nil
		}
		if annotator.LanguageForPath(rel) == "" {
			return nil
		}
		if annotator.IsPathExcludedFromScans(rel) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	return paths, walkErr
}
