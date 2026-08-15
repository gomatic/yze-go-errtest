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
	ins.WithStack(types, func(n ast.Node, isPush bool, stack []ast.Node) bool {
		if !isPush || !isTestFile(pass, n) {
			return true
		}
		switch node := n.(type) {
		case *ast.StructType:
			checkFields(pass, node)
		case *ast.CallExpr:
			checkCall(pass, node)
			checkErrTextArgs(pass, node, stack)
		case *ast.BinaryExpr:
			checkComparison(pass, node, stack)
		}
		return true
	})
	return nil, nil
}

// isMechanism reports whether the package under analysis is the sentinel
// mechanism itself, whose message rendering is the contract its tests exist to
// verify.
func isMechanism(pass *analysis.Pass) bool {
	return pass.Pkg != nil && isMechanismPath(packagePath(pass.Pkg.Path()))
}

// packagePath is the import path of the package under analysis.
type packagePath string

// isMechanismPath reports whether path is the mechanism module at a path
// BOUNDARY: the module itself, one of its subpackages ("/"), or one of its
// test variants — the external test package ("_test") and the synthesized
// test main (".test"). A bare prefix comparison is NOT enough: it would let
// any module prefix-extending the path (github.com/gomatic/go-error-extra, a
// go-errors fork, …) escape the entire analyzer.
func isMechanismPath(path packagePath) bool {
	switch path {
	case mechanismPath, mechanismPath + "_test", mechanismPath + ".test":
		return true
	}
	return strings.HasPrefix(string(path), mechanismPath+"/")
}

// isTestFile reports whether the node lives in a _test.go file.
//
// The name comes from the FileSet's own entry, never from a Position: Position
// applies //line directives, so a file could rename itself into or out of this
// analyzer's scope with one comment line while the go tool went on compiling
// and running it unchanged. A decision ABOUT a file must read something that
// file cannot rewrite.
func isTestFile(pass *analysis.Pass, n ast.Node) bool {
	return strings.HasSuffix(pass.Fset.File(n.Pos()).Name(), "_test.go")
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
