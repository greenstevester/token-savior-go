// Package golang annotates Go source files using the stdlib AST.
package golang

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"token-savior-go/internal/models"
)

// Annotator is the Go-language annotator. Safe for concurrent use.
type Annotator struct{}

// New returns a Go annotator.
func New() *Annotator { return &Annotator{} }

// Annotate parses Go source and emits structural metadata. Parse errors are
// returned to the caller; the indexer is expected to log them per-file and
// continue.
func (a *Annotator) Annotate(path string, source []byte) (*models.StructuralMetadata, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	md := &models.StructuralMetadata{
		Path:       path,
		Language:   "go",
		Functions:  []models.Function{},
		Classes:    []models.Class{},
		Imports:    []models.Import{},
		Calls:      []models.SymbolRef{},
		References: []models.SymbolRef{},
	}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			md.Functions = append(md.Functions, funcFromDecl(fset, fn))
		}
	}
	return md, nil
}

// funcFromDecl translates an *ast.FuncDecl into a models.Function.
//
// The signature is rendered with go/printer so it matches what a developer
// would see, including parameter names. Multi-line signatures are flattened.
func funcFromDecl(fset *token.FileSet, fn *ast.FuncDecl) models.Function {
	name := fn.Name.Name
	receiver := ""
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		receiver = receiverTypeName(fn.Recv.List[0].Type)
	}
	qualified := name
	if receiver != "" {
		qualified = receiver + "." + name
	}

	return models.Function{
		Name:      name,
		Receiver:  receiver,
		Qualified: qualified,
		Line:      fset.Position(fn.Pos()).Line,
		EndLine:   fset.Position(fn.End()).Line,
		Signature: renderFuncSignature(fset, fn),
	}
}

// receiverTypeName extracts the bare type name from a method receiver.
// Handles both `func (t Thing)` and `func (t *Thing)`.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// renderFuncSignature returns a single-line signature like
// "func (t *Thing) Name() string" using go/printer for fidelity.
func renderFuncSignature(fset *token.FileSet, fn *ast.FuncDecl) string {
	// Build a synthetic decl with an empty body so we only print the signature.
	// Clear Doc too — go/printer would otherwise emit the attached doc comment
	// before the signature, polluting Function.Signature.
	stub := *fn
	stub.Body = nil
	stub.Doc = nil

	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	if err := cfg.Fprint(&buf, fset, &stub); err != nil {
		return fn.Name.Name
	}
	// Collapse any newlines to spaces.
	return string(bytes.Join(bytes.Fields(buf.Bytes()), []byte(" ")))
}
