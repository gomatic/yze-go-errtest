// Package errtest provides a go/analysis analyzer enforcing the gomatic
// error-testing shape in _test.go files: a table's error expectation is
// declared as the builtin error interface (wantErr error) and matched with
// errors.Is, never as a concrete sentinel type, a bool, or a message string,
// and never via testify's EqualError/ErrorContains message matching.
//
// Scope: only test files are inspected. Expectation fields are struct fields
// whose name starts with want/expect (any case); typed sentinel INPUTS (e.g. a
// `sentinel errs.Const` field the test calls .With on) stay sanctioned, as do
// message expectations that do not name an error (plain `wantMsg string`).
package errtest

import (
	"go/ast"
	"go/types"
	"strings"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Diagnostic messages, one per banned expectation shape.
const (
	messageSentinel = "test error expectations must be typed error (wantErr error), not a concrete or custom error type"
	messageBool     = "expect errors as wantErr error matched with errors.Is, not a bool"
	messageMessage  = "expect errors as wantErr error matched with errors.Is, not a message string"
	messageMatch    = "match errors with errors.Is, never message strings"
)

// testifyPath is the module whose message-matching assertions are banned.
const testifyPath = "github.com/stretchr/testify"

// Analyzer reports error expectations in test files that bypass the
// wantErr-error-plus-errors.Is shape.
var Analyzer = &analysis.Analyzer{
	Name:     "errtest",
	Doc:      "reports test error expectations that are not wantErr error matched with errors.Is",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "errtest",
	Categories: []goyze.Category{"errors", "tests"},
	URL:        "https://docs.gomatic.dev/yze/errtest",
	Analyzer:   Analyzer,
}

// run reports banned error-expectation shapes in the pass's test files.
func run(pass *analysis.Pass) (any, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	types := []ast.Node{(*ast.StructType)(nil), (*ast.CallExpr)(nil)}
	ins.Preorder(types, func(n ast.Node) {
		if !isTestFile(pass, n) {
			return
		}
		switch node := n.(type) {
		case *ast.StructType:
			checkFields(pass, node)
		case *ast.CallExpr:
			checkCall(pass, node)
		}
	})
	return nil, nil
}

// isTestFile reports whether the node lives in a _test.go file.
func isTestFile(pass *analysis.Pass, n ast.Node) bool {
	return strings.HasSuffix(pass.Fset.Position(n.Pos()).Filename, "_test.go")
}

// checkFields reports each expectation-named struct field whose type bypasses
// the wantErr-error shape.
func checkFields(pass *analysis.Pass, structType *ast.StructType) {
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			checkExpectation(pass, name, pass.TypesInfo.TypeOf(field.Type))
		}
	}
}

// checkExpectation reports a single want/expect-prefixed field whose type is a
// concrete error type, a bool naming an error, or an error-message string.
func checkExpectation(pass *analysis.Pass, name *ast.Ident, fieldType types.Type) {
	lower := strings.ToLower(name.Name)
	if !strings.HasPrefix(lower, "want") && !strings.HasPrefix(lower, "expect") {
		return
	}
	switch {
	case isConcreteErrorType(fieldType):
		pass.Reportf(name.Pos(), "%s", messageSentinel)
	case strings.Contains(lower, "err") && isBasic(fieldType, types.Bool):
		pass.Reportf(name.Pos(), "%s", messageBool)
	case strings.Contains(lower, "err") && isBasic(fieldType, types.String):
		pass.Reportf(name.Pos(), "%s", messageMessage)
	}
}

// isConcreteErrorType reports whether t implements error without being the
// builtin error interface itself (aliases of it included); such a type is a
// sentinel or custom error and must not be an expectation's type.
func isConcreteErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	u := types.Unalias(t)
	if _, isParam := u.(*types.TypeParam); isParam {
		return false
	}
	if types.Identical(u, types.Universe.Lookup("error").Type()) {
		return false
	}
	iface := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return types.Implements(u, iface) || types.Implements(types.NewPointer(u), iface)
}

// isBasic reports whether t's core type is the given basic kind.
func isBasic(t types.Type, kind types.BasicKind) bool {
	if t == nil {
		return false
	}
	basic, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && basic.Kind() == kind
}

// checkCall reports testify message-matching assertions (EqualError and
// ErrorContains, function and method forms alike).
func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !isMessageMatcher(assertionName(sel.Sel.Name)) {
		return
	}
	if isTestifyFunc(pass.TypesInfo.ObjectOf(sel.Sel)) {
		pass.Reportf(call.Pos(), "%s", messageMatch)
	}
}

// isTestifyFunc reports whether obj is a function or method declared by the
// testify module (a nil object, a non-function, or a package-less builtin is
// never testify's).
func isTestifyFunc(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	path := fn.Pkg().Path()
	return path == testifyPath || strings.HasPrefix(path, testifyPath+"/")
}

// assertionName is a testify assertion's method or function name.
type assertionName string

// isMessageMatcher reports whether name is a banned testify assertion,
// including the formatted (…f) variants.
func isMessageMatcher(name assertionName) bool {
	switch strings.TrimSuffix(string(name), "f") {
	case "EqualError", "ErrorContains":
		return true
	}
	return false
}
