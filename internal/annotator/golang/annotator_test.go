package golang

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"token-savior-go/internal/models"
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
	require.False(t, strings.HasPrefix(doThing.Signature, "//"),
		"signature should not contain leading doc comment: %q", doThing.Signature)

	method := md.Functions[byQualified["Thing.Name"]]
	require.Equal(t, "Name", method.Name)
	require.Equal(t, "Thing", method.Receiver)
}

func TestAnnotate_GenericReceivers(t *testing.T) {
	src := loadFixture(t, "generics.go.txt")
	md, err := New().Annotate("generics.go", src)
	require.NoError(t, err)
	require.Len(t, md.Functions, 3)

	byQualified := map[string]models.Function{}
	for _, f := range md.Functions {
		byQualified[f.Qualified] = f
	}

	// Pointer-receiver, single type param.
	require.Contains(t, byQualified, "Container.Get")
	require.Equal(t, "Container", byQualified["Container.Get"].Receiver)

	// Value-receiver, single type param.
	require.Contains(t, byQualified, "Container.Set")
	require.Equal(t, "Container", byQualified["Container.Set"].Receiver)

	// Pointer-receiver, multi type param.
	require.Contains(t, byQualified, "Pair.Key")
	require.Equal(t, "Pair", byQualified["Pair.Key"].Receiver)
}

func TestAnnotate_Types(t *testing.T) {
	src := loadFixture(t, "types.go.txt")
	md, err := New().Annotate("types.go", src)
	require.NoError(t, err)

	require.Len(t, md.Classes, 4)

	byName := map[string]int{}
	for i, c := range md.Classes {
		byName[c.Name] = i
	}

	require.Equal(t, "struct", md.Classes[byName["Thing"]].Kind)
	require.Equal(t, "interface", md.Classes[byName["Greeter"]].Kind)
	require.Equal(t, "alias", md.Classes[byName["StringList"]].Kind)
	require.Equal(t, "alias", md.Classes[byName["Counter"]].Kind)

	require.Equal(t, "Thing", md.Classes[byName["Thing"]].Qualified)
}

func TestAnnotate_Imports(t *testing.T) {
	src := loadFixture(t, "imports.go.txt")
	md, err := New().Annotate("imports.go", src)
	require.NoError(t, err)

	require.Len(t, md.Imports, 4)

	byPath := map[string]models.Import{}
	for _, im := range md.Imports {
		byPath[im.Path] = im
	}

	require.Contains(t, byPath, "context")
	require.Contains(t, byPath, "fmt")
	require.Contains(t, byPath, "net/http")
	require.Contains(t, byPath, "errors")

	require.Equal(t, "myhttp", byPath["net/http"].Alias)
	require.Equal(t, "", byPath["context"].Alias)
}

func TestAnnotate_Calls(t *testing.T) {
	src := loadFixture(t, "calls.go.txt")
	md, err := New().Annotate("calls.go", src)
	require.NoError(t, err)

	// caller -> {Println, helper, another}; helper -> {Sprintf}
	got := map[string]map[string]struct{}{}
	for _, c := range md.Calls {
		if got[c.From] == nil {
			got[c.From] = map[string]struct{}{}
		}
		got[c.From][c.To] = struct{}{}
	}

	require.Contains(t, got["caller"], "fmt.Println")
	require.Contains(t, got["caller"], "helper")
	require.Contains(t, got["caller"], "another")
	require.Contains(t, got["helper"], "fmt.Sprintf")
	require.NotContains(t, got, "another") // empty body -> no entry
}
