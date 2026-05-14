package query

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"

	"token-savior-go/internal/models"
)

// SearchHit is the unit return type for search_codebase.
type SearchHit struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// SearchCodebase scans every indexed file for pattern. When asRegex is true,
// pattern is compiled with regexp.Compile; otherwise it's matched as a fixed
// byte string. Results are line-anchored and limited to 500 hits to keep MCP
// payloads bounded.
func SearchCodebase(idx *models.ProjectIndex, pattern string, asRegex bool) ([]SearchHit, error) {
	const maxHits = 500
	var re *regexp.Regexp
	var literal []byte
	if asRegex {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		re = r
	} else {
		literal = []byte(pattern)
	}

	var hits []SearchHit
	for _, rel := range idx.SortedPaths {
		full := filepath.Join(idx.Root, rel)
		f, err := os.Open(full) //nolint:gosec // rel is from indexed SortedPaths under caller-controlled Root.
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		ln := 0
		for scanner.Scan() {
			ln++
			line := scanner.Bytes()
			matched := false
			if asRegex {
				matched = re.Match(line)
			} else {
				matched = bytes.Contains(line, literal)
			}
			if matched {
				hits = append(hits, SearchHit{File: rel, Line: ln, Text: string(line)})
				if len(hits) >= maxHits {
					_ = f.Close()
					return hits, nil
				}
			}
		}
		_ = f.Close()
	}
	return hits, nil
}
