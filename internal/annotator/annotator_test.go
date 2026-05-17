package annotator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLanguageForPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"foo.go", "go"},
		{"foo.ts", "typescript"},
		{"foo.tsx", "typescript"},
		{"foo.js", "javascript"},
		{"foo.jsx", "javascript"},
		{"foo.java", "java"},
		{"foo.rs", "rust"},
		{"foo.sh", "shell"},
		{"foo.bash", "shell"},
		{"foo.zsh", "shell"},
		{"foo.unknown", ""},
		{"Dockerfile", ""}, // out of M1 scope
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, tc.want, LanguageForPath(tc.path))
		})
	}
}

func TestIsPathExcludedFromScans(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{".git/config", true},
		{"src/foo/.git/HEAD", true},
		{"node_modules/foo/index.js", true},
		{"src/__pycache__/foo.pyc", true},
		{".token-savior-checkpoints/abc/foo.go", true},
		{"src/foo.go", false},
		{"vendor/foo.go", false}, // vendor is allowed; matches Python
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, tc.want, IsPathExcludedFromScans(tc.path))
		})
	}
}
