package golang

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	bytes, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return bytes
}

func TestAnnotate_Functions(t *testing.T) {
	src := loadFixture(t, "funcs.go.txt")
	a := New()
	md, err := a.Annotate("funcs.go", src)
	require.NoError(t, err)
	require.Equal(t, "go", md.Language)
	require.Equal(t, "funcs.go", md.Path)

	require.Len(t, md.Functions, 4)

	byQualified := map[string]int{}
	for i, f := range md.Functions {
		byQualified[f.Qualified] = i
	}

	require.Contains(t, byQualified, "DoThing")
	require.Contains(t, byQualified, "helper")
	require.Contains(t, byQualified, "Thing.Name")
	require.Contains(t, byQualified, "Thing.Rename")

	doThing := md.Functions[byQualified["DoThing"]]
	require.Equal(t, "DoThing", doThing.Name)
	require.Equal(t, "", doThing.Receiver)
	require.Equal(t, 6, doThing.Line) // 1-indexed line number of `func DoThing`
	require.Contains(t, doThing.Signature, "ctx context.Context")
	require.Contains(t, doThing.Signature, "n int")
	require.Contains(t, doThing.Signature, "error")

	method := md.Functions[byQualified["Thing.Name"]]
	require.Equal(t, "Name", method.Name)
	require.Equal(t, "Thing", method.Receiver)
}
