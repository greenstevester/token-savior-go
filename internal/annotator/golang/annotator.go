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
		switch d := decl.(type) {
		case *ast.FuncDecl:
			f := funcFromDecl(fset, d)
			md.Functions = append(md.Functions, f)
			walkCalls(fset, d, f.Qualified, &md.Calls)
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if cls, ok := classFromSpec(fset, ts); ok {
							md.Classes = append(md.Classes, cls)
						}
					}
				}
			case token.IMPORT:
				for _, spec := range d.Specs {
					if is, ok := spec.(*ast.ImportSpec); ok {
						md.Imports = append(md.Imports, importFromSpec(fset, is))
					}
				}
			}
		}
	}
	return md, nil
}

// classFromSpec translates a *ast.TypeSpec into a models.Class. Returns
// (Class, true) on success and (zero, false) when the spec is not a kind we
// model (e.g. function-type aliases would still classify as alias).
func classFromSpec(fset *token.FileSet, spec *ast.TypeSpec) (models.Class, bool) {
	kind := "alias"
	switch spec.Type.(type) {
	case *ast.StructType:
		kind = "struct"
	case *ast.InterfaceType:
		kind = "interface"
	}
	return models.Class{
		Name:      spec.Name.Name,
		Qualified: spec.Name.Name,
		Kind:      kind,
		Line:      fset.Position(spec.Pos()).Line,
		EndLine:   fset.Position(spec.End()).Line,
	}, true
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
// Handles six shapes:
//   - func (t Thing)            -> *ast.Ident
//   - func (t *Thing)           -> *ast.StarExpr wrapping *ast.Ident
//   - func (t Thing[T])         -> *ast.IndexExpr
//   - func (t *Thing[T])        -> *ast.StarExpr wrapping *ast.IndexExpr
//   - func (t Thing[T, U])      -> *ast.IndexListExpr
//   - func (t *Thing[T, U])     -> *ast.StarExpr wrapping *ast.IndexListExpr
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		switch x := t.X.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.IndexExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				return id.Name
			}
		case *ast.IndexListExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				return id.Name
			}
		}
	case *ast.IndexExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.IndexListExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// importFromSpec converts an *ast.ImportSpec into a models.Import. The Path
// is unquoted; Alias is empty unless the import is renamed.
func importFromSpec(fset *token.FileSet, spec *ast.ImportSpec) models.Import {
	path := spec.Path.Value
	// Path is a quoted string literal; strip the quotes.
	if len(path) >= 2 && path[0] == '"' && path[len(path)-1] == '"' {
		path = path[1 : len(path)-1]
	}
	alias := ""
	if spec.Name != nil {
		alias = spec.Name.Name
	}
	return models.Import{
		Path:  path,
		Alias: alias,
		Line:  fset.Position(spec.Pos()).Line,
	}
}

// walkCalls populates md.Calls with edges from each function/method body's
// CallExpr nodes. The To field carries:
//   - "ident"           for bare calls (helper())
//   - "pkg.Ident"       for qualified calls (fmt.Println())
//   - "recv.Method"     for method calls on identifier receivers (t.Do())
//
// Method calls on non-identifier receivers (e.g. m().Foo()) are skipped to
// avoid an avalanche of unresolved edges; the query layer treats To as a
// best-effort hint.
func walkCalls(fset *token.FileSet, fn *ast.FuncDecl, from string, out *[]models.SymbolRef) {
	if fn.Body == nil {
		return
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		to := callTargetName(call.Fun)
		if to == "" {
			return true
		}
		*out = append(*out, models.SymbolRef{
			From: from,
			To:   to,
			Line: fset.Position(call.Pos()).Line,
		})
		return true
	})
}

// callTargetName extracts a best-effort name for a CallExpr.Fun expression.
func callTargetName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
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
