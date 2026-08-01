package errtest

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcreteErrorTypeGuards pins the type-classifier's guard branches that
// valid loaded packages cannot reach through the fixtures: a missing type (nil)
// and a type parameter are never expectation violations.
func TestConcreteErrorTypeGuards(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.False(isConcreteErrorType(nil))
	param := types.NewTypeParam(types.NewTypeName(0, nil, "T", nil), types.NewInterfaceType(nil, nil))
	want.False(isConcreteErrorType(param))
	want.False(isConcreteErrorType(types.Universe.Lookup("error").Type()))
}

// TestBasicKindGuards pins isBasic's nil guard and non-basic rejection.
func TestBasicKindGuards(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.False(isBasic(nil, types.Bool))
	want.False(isBasic(types.NewSlice(types.Typ[types.String]), types.String))
	want.True(isBasic(types.Typ[types.Bool], types.Bool))
}

// TestTestifyFuncGuards pins the callee classifier's guard branches: a nil
// object, a non-function object, a package-less builtin function, and a
// same-named function from a different module are never testify's.
func TestTestifyFuncGuards(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	want.False(isTestifyFunc(nil))
	want.False(isTestifyFunc(types.NewVar(0, nil, "EqualError", sig)))
	want.False(isTestifyFunc(types.NewFunc(0, nil, "EqualError", sig)))
	other := types.NewPackage("example.com/other/assert", "assert")
	want.False(isTestifyFunc(types.NewFunc(0, other, "EqualError", sig)))
	testify := types.NewPackage("github.com/stretchr/testify/assert", "assert")
	want.True(isTestifyFunc(types.NewFunc(0, testify, "EqualError", sig)))
}

// TestMessageMatcherNames pins the banned-assertion name set including the
// formatted variants.
func TestMessageMatcherNames(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.True(isMessageMatcher(assertionName("EqualError")))
	want.True(isMessageMatcher(assertionName("EqualErrorf")))
	want.True(isMessageMatcher(assertionName("ErrorContains")))
	want.True(isMessageMatcher(assertionName("ErrorContainsf")))
	want.False(isMessageMatcher(assertionName("ErrorIs")))
	want.False(isMessageMatcher(assertionName("Errorf")))
}

// TestIsConcreteErrorTypeExcludesTheBuiltinInterfaceItself names
// isConcreteErrorType's claim. The rule it serves is "declare a table's error
// expectation as the builtin error, not as a concrete type" — so this predicate
// must say NO to `error` itself and YES to everything that implements it. Both
// directions are load-bearing: saying yes to `error` would report every
// correctly-written table, and saying no to a sentinel type would let the shape
// the rule exists to ban pass unnoticed.
//
// The alias and type-parameter cases are the ones that catch a naive
// implementation. `type myError = error` IS the builtin under another name, and
// a generic constraint is not a concrete type at all.
func TestIsConcreteErrorTypeExcludesTheBuiltinInterfaceItself(t *testing.T) {
	t.Parallel()
	pkg := typesOf(t, `package p

import "errors"

type Sentinel string

func (Sentinel) Error() string { return "" }

type Wrapped struct{ err error }

func (*Wrapped) Error() string { return "" }

type Aliased = error

var _ = errors.New

type NotAnError struct{}
`)
	builtinError := types.Universe.Lookup("error").Type()

	assert.False(t, isConcreteErrorType(nil), "a nil type is not an error type")
	assert.False(t, isConcreteErrorType(builtinError), "the builtin interface is the expected shape, not a banned one")
	assert.False(t, isConcreteErrorType(lookup(t, pkg, "Aliased")), "an alias of error IS error")
	assert.False(t, isConcreteErrorType(lookup(t, pkg, "NotAnError")), "a type that does not implement error")

	assert.True(t, isConcreteErrorType(lookup(t, pkg, "Sentinel")), "a value-receiver sentinel type")
	assert.True(t, isConcreteErrorType(lookup(t, pkg, "Wrapped")),
		"a pointer-receiver error type is still a concrete error type")
}

// TestIsTestifyFuncRecognisesOnlyTestifyDeclarations names isTestifyFunc's
// claim. The banned message-matching assertions are testify's, so a locally
// declared helper called EqualError must NOT be treated as one — reporting it
// would flag a project's own function for a rule about someone else's library.
// A nil object, a non-function and a package-less builtin all reach this
// predicate from ordinary code and must each answer no rather than panic.
func TestIsTestifyFuncRecognisesOnlyTestifyDeclarations(t *testing.T) {
	t.Parallel()

	assert.False(t, isTestifyFunc(nil), "a nil object")
	assert.False(t, isTestifyFunc(types.Universe.Lookup("len")), "a package-less builtin")
	assert.False(t, isTestifyFunc(types.Universe.Lookup("error")), "a type name is not a function")

	local := typesOf(t, "package p\n\nfunc EqualError(a, b any) {}\n")
	assert.False(t, isTestifyFunc(local.Scope().Lookup("EqualError")),
		"a locally declared helper sharing testify's name is not testify's")
}

// typesOf type-checks src and returns its package.
func typesOf(t *testing.T, src string) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	require.NoError(t, err)
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.test/p", fset, []*ast.File{file}, nil)
	require.NoError(t, err)
	return pkg
}

// lookup returns the named type declared in pkg.
func lookup(t *testing.T, pkg *types.Package, name string) types.Type {
	t.Helper()
	obj := pkg.Scope().Lookup(name)
	require.NotNil(t, obj, "no type %s in the fixture", name)
	return obj.Type()
}
