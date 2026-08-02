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
	"go/token"
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
	messageErrText  = "never match on err.Error() text; assert the sentinel with errors.Is"
)

// testifyPath is the module whose message-matching assertions are banned.
const testifyPath = "github.com/stretchr/testify"

// mechanismPath is the package implementing the sentinel mechanism itself. Its
// own tests MUST assert rendered message text — the rendering is the contract
// under test there, and no amount of errors.Is matching can verify what an
// error's Error() string says. It is exempt for the same reason it is the one
// sanctioned fmt.Errorf call site in yze/errconst.
const mechanismPath = "github.com/gomatic/go-error"

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
	if isMechanism(pass) {
		return nil, nil
	}
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	types := []ast.Node{(*ast.StructType)(nil), (*ast.CallExpr)(nil), (*ast.BinaryExpr)(nil)}
	ins.Preorder(types, func(n ast.Node) {
		if !isTestFile(pass, n) {
			return
		}
		switch node := n.(type) {
		case *ast.StructType:
			checkFields(pass, node)
		case *ast.CallExpr:
			checkCall(pass, node)
		case *ast.BinaryExpr:
			checkComparison(pass, node)
		}
	})
	return nil, nil
}

// isMechanism reports whether the package under analysis is the sentinel
// mechanism itself, whose message rendering is the contract its tests exist to
// verify. The test variant of that package carries a `.test` suffix, so the
// prefix comparison covers both.
func isMechanism(pass *analysis.Pass) bool {
	return pass.Pkg != nil && strings.HasPrefix(pass.Pkg.Path(), mechanismPath)
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
	checkErrTextArgs(pass, call)
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !isMessageMatcher(assertionName(sel.Sel.Name)) {
		return
	}
	if isTestifyFunc(pass.TypesInfo.ObjectOf(sel.Sel)) {
		pass.Reportf(call.Pos(), "%s", messageMatch)
	}
}

// stringMatchers are the strings-package predicates that turn an error's
// rendered text into a match — the hand-rolled spelling of the same defect
// EqualError/ErrorContains commit.
var stringMatchers = map[string]bool{
	"Contains": true, "HasPrefix": true, "HasSuffix": true, "EqualFold": true,
}

// checkErrTextArgs reports a strings-package matcher fed an error's rendered
// text.
func checkErrTextArgs(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !stringMatchers[sel.Sel.Name] || !isStringsFunc(pass.TypesInfo.ObjectOf(sel.Sel)) {
		return
	}
	for _, arg := range call.Args {
		if isErrorText(pass, arg) {
			pass.Reportf(call.Pos(), "%s", messageErrText)
			return
		}
	}
}

// checkComparison reports an ==/!= comparison against an error's rendered text.
func checkComparison(pass *analysis.Pass, cmp *ast.BinaryExpr) {
	if cmp.Op != token.EQL && cmp.Op != token.NEQ {
		return
	}
	if isErrorText(pass, cmp.X) || isErrorText(pass, cmp.Y) {
		pass.Reportf(cmp.Pos(), "%s", messageErrText)
	}
}

// isErrorText reports whether expr is a call of the error interface's Error
// method — an error's message rendered for matching.
func isErrorText(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Error" || len(call.Args) != 0 {
		return false
	}
	recv := pass.TypesInfo.TypeOf(sel.X)
	return recv != nil && types.Implements(recv, errorInterface())
}

// errorInterface is the built-in error interface type.
func errorInterface() *types.Interface {
	iface, _ := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return iface
}

// isStringsFunc reports whether obj is a function declared by the standard
// strings package.
func isStringsFunc(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == "strings"
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
