package errtest

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
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
