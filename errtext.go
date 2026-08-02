package errtest

// The err.Error() text-matching rule: the hand-rolled spelling of the same
// defect EqualError/ErrorContains commit, with the two rendering-contract
// exemptions (a sentinel constant's own text; text asserted alongside an
// Is-style match of the same error).

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// stringMatchers are the strings-package predicates that turn an error's
// rendered text into a match — the hand-rolled spelling of the same defect
// EqualError/ErrorContains commit.
var stringMatchers = map[string]bool{
	"Contains": true, "HasPrefix": true, "HasSuffix": true, "EqualFold": true,
}

// checkErrTextArgs reports a strings-package matcher fed an error's rendered
// text, unless the shape is a rendering-contract assertion (see reportErrText).
func checkErrTextArgs(pass *analysis.Pass, call *ast.CallExpr, stack []ast.Node) {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !stringMatchers[sel.Sel.Name] || !isStringsFunc(pass.TypesInfo.ObjectOf(sel.Sel)) {
		return
	}
	for _, arg := range call.Args {
		if reportErrText(pass, call.Pos(), arg, stack) {
			return
		}
	}
}

// checkComparison reports an ==/!= comparison against an error's rendered
// text, under the same rendering-contract exemptions.
func checkComparison(pass *analysis.Pass, cmp *ast.BinaryExpr, stack []ast.Node) {
	if cmp.Op != token.EQL && cmp.Op != token.NEQ {
		return
	}
	if !reportErrText(pass, cmp.Pos(), cmp.X, stack) {
		reportErrText(pass, cmp.Pos(), cmp.Y, stack)
	}
}

// reportErrText reports expr when it renders an error's text for MATCHING, and
// stays silent for the two rendering-contract shapes: the text of a directly
// referenced sentinel constant (its message is part of its declared contract,
// which an errors.Is match says nothing about), and the text of an error the
// SAME function already asserts with an Is-style matcher (the sentinel is
// discriminated properly; the text assertion verifies rendering on top).
func reportErrText(pass *analysis.Pass, at token.Pos, expr ast.Expr, stack []ast.Node) bool {
	receiver, ok := errorTextReceiver(pass, expr)
	if !ok || isSentinelConst(pass, receiver) || isAssertedInFunc(pass, receiver, stack) {
		return false
	}
	pass.Reportf(at, "%s", messageErrText)
	return true
}

// errorTextReceiver returns the receiver expression of an (error).Error() call,
// or false when expr is no such call.
func errorTextReceiver(pass *analysis.Pass, expr ast.Expr) (ast.Expr, bool) {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Error" || len(call.Args) != 0 {
		return nil, false
	}
	recv := pass.TypesInfo.TypeOf(sel.X)
	if recv == nil || !types.Implements(recv, errorInterface()) {
		return nil, false
	}
	return sel.X, true
}

// isSentinelConst reports whether the receiver is a sentinel whose rendered
// message is its own declared contract: a directly referenced constant, or any
// expression TYPED by the sentinel mechanism (a table field of errs.Const
// pinning each sentinel's message is the same contract, table-driven).
func isSentinelConst(pass *analysis.Pass, receiver ast.Expr) bool {
	if _, isConst := referencedObject(pass, receiver).(*types.Const); isConst {
		return true
	}
	return isMechanismType(pass.TypesInfo.TypeOf(receiver))
}

// isMechanismType reports whether t is declared by the sentinel mechanism
// package.
func isMechanismType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return strings.HasPrefix(named.Obj().Pkg().Path(), mechanismPath)
}

// isAssertedInFunc reports whether the innermost enclosing function also
// asserts the same error object with an Is-style matcher (errors.Is, or
// testify's ErrorIs/ErrorIsf in function or method form).
func isAssertedInFunc(pass *analysis.Pass, receiver ast.Expr, stack []ast.Node) bool {
	obj := referencedObject(pass, receiver)
	if obj == nil {
		return false
	}
	body := enclosingBody(stack)
	if body == nil {
		return false
	}
	return containsIsAssertion(pass, body, obj)
}

// referencedObject resolves the object a plain identifier or selector names.
func referencedObject(pass *analysis.Pass, expr ast.Expr) types.Object {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return pass.TypesInfo.ObjectOf(e)
	case *ast.SelectorExpr:
		return pass.TypesInfo.ObjectOf(e.Sel)
	}
	return nil
}

// enclosingBody is the body of the innermost function in the stack.
func enclosingBody(stack []ast.Node) *ast.BlockStmt {
	for i := len(stack) - 1; i >= 0; i-- {
		switch fn := stack[i].(type) {
		case *ast.FuncDecl:
			return fn.Body
		case *ast.FuncLit:
			return fn.Body
		}
	}
	return nil
}

// isMatcherName reports an Is-style matcher's name.
func isMatcherName(name assertionName) bool {
	return name == "Is" || name == "ErrorIs" || name == "ErrorIsf"
}

// containsIsAssertion reports whether body calls an Is-style matcher with obj
// among its arguments.
func containsIsAssertion(pass *analysis.Pass, body *ast.BlockStmt, obj types.Object) bool {
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok && isMatcherName(assertionName(sel.Sel.Name)) {
			found = callMentions(pass, call, obj)
		}
		return !found
	})
	return found
}

// callMentions reports whether any argument of call names obj.
func callMentions(pass *analysis.Pass, call *ast.CallExpr, obj types.Object) bool {
	for _, arg := range call.Args {
		if ident, ok := ast.Unparen(arg).(*ast.Ident); ok && pass.TypesInfo.ObjectOf(ident) == obj {
			return true
		}
	}
	return false
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
