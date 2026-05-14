package indexer

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"token-savior-go/internal/annotator"
	"token-savior-go/internal/models"
)

// ProjectIndexer builds a ProjectIndex for a single root directory.
type ProjectIndexer struct {
	Root    string
	Workers int
}

// NewProjectIndexer returns an indexer with worker count defaulted to NumCPU.
func NewProjectIndexer(root string) *ProjectIndexer {
	return &ProjectIndexer{Root: root, Workers: runtime.NumCPU()}
}

// Build walks the root, annotates every annotatable file in a worker pool,
// and assembles a ProjectIndex with symbol table, dep graph, import graph,
// and basename map. Per-file annotation errors are collected and returned
// as a joined error; the index is still returned with the files that did
// annotate successfully.
func (p *ProjectIndexer) Build() (*models.ProjectIndex, error) {
	paths, err := Walk(p.Root)
	if err != nil {
		return nil, err
	}

	idx := models.NewProjectIndex(p.Root)

	jobs := make(chan string, len(paths))
	type result struct {
		path string
		md   *models.StructuralMetadata
		err  error
	}
	results := make(chan result, len(paths))

	var wg sync.WaitGroup
	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				a := annotator.AnnotatorFor(rel)
				if a == nil {
					continue
				}
				source, readErr := os.ReadFile(filepath.Join(p.Root, rel)) //nolint:gosec // rel comes from Walk which already filtered the root; p.Root is caller-controlled.
				if readErr != nil {
					results <- result{path: rel, err: readErr}
					continue
				}
				md, annErr := a.Annotate(rel, source)
				results <- result{path: rel, md: md, err: annErr}
			}
		}()
	}

	for _, rel := range paths {
		jobs <- rel
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		idx.Files[r.path] = r.md
		populateLookups(idx, r.path, r.md)
	}

	rebuildSortedAndBasename(idx)

	if len(errs) > 0 {
		return idx, errors.Join(errs...)
	}
	return idx, nil
}

// populateLookups updates SymbolTable, DepGraph, and ImportGraph for one
// annotated file. First-seen wins on SymbolTable collisions.
func populateLookups(idx *models.ProjectIndex, path string, md *models.StructuralMetadata) {
	for _, f := range md.Functions {
		if _, exists := idx.SymbolTable[f.Qualified]; !exists {
			idx.SymbolTable[f.Qualified] = path
		}
	}
	for _, c := range md.Classes {
		if _, exists := idx.SymbolTable[c.Qualified]; !exists {
			idx.SymbolTable[c.Qualified] = path
		}
	}
	for _, call := range md.Calls {
		if idx.DepGraph[call.From] == nil {
			idx.DepGraph[call.From] = make(map[string]struct{})
		}
		idx.DepGraph[call.From][call.To] = struct{}{}
	}
	for _, im := range md.Imports {
		if idx.ImportGraph[path] == nil {
			idx.ImportGraph[path] = make(map[string]struct{})
		}
		idx.ImportGraph[path][im.Path] = struct{}{}
	}
}

func rebuildSortedAndBasename(idx *models.ProjectIndex) {
	idx.SortedPaths = make([]string, 0, len(idx.Files))
	for p := range idx.Files {
		idx.SortedPaths = append(idx.SortedPaths, p)
	}
	sort.Strings(idx.SortedPaths)

	idx.BasenameMap = make(map[string][]string)
	for _, p := range idx.SortedPaths {
		base := filepath.Base(p)
		idx.BasenameMap[base] = append(idx.BasenameMap[base], p)
	}
}
