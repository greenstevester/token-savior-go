package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProfile(t *testing.T) {
	cases := []struct {
		input string
		want  ProfileSet
	}{
		{"", ProfileFull},
		{"full", ProfileFull},
		{"core", ProfileCore},
		{"nav", ProfileNav},
		{"lean", ProfileLean},
		{"ultra", ProfileUltra},
		{"tiny", ProfileTiny},
		{"tiny_plus", ProfileTinyPlus},
		{"FULL", ProfileFull},    // case-insensitive
		{"unknown", ProfileFull}, // unknown falls back, warning logged by caller
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			require.Equal(t, tc.want, ParseProfile(tc.input))
		})
	}
}

func TestVisibleTools_AllVisibleInFull(t *testing.T) {
	r := DefaultRegistry()
	got := VisibleTools(r, ProfileFull)
	require.Len(t, got, len(r.All()))
}
