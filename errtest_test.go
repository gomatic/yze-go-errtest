package errtest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	errtest "github.com/gomatic/yze-go-errtest"
)

func TestErrorExpectationShape(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errtest.Analyzer, "a")
}

// TestTheSentinelMechanismIsExempt pins that the package implementing the
// mechanism may assert rendered message text: its own tests carry the banned
// shapes deliberately, because the rendering is what they exist to verify.
func TestTheSentinelMechanismIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errtest.Analyzer, "github.com/gomatic/go-error")
}

// TestTheMechanismSubpackageIsExempt pins the "/" side of the exemption's
// boundary: a genuine subpackage of the mechanism inherits the exemption.
func TestTheMechanismSubpackageIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errtest.Analyzer, "github.com/gomatic/go-error/sub")
}

// TestAPathPrefixExtensionIsNotExempt pins the other side of the boundary: a
// package whose import path merely PREFIX-EXTENDS the mechanism's
// (go-error-extra) is not the mechanism and is analyzed in full. A bare
// prefix comparison would silently exempt it — escaping the entire analyzer.
func TestAPathPrefixExtensionIsNotExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errtest.Analyzer, "github.com/gomatic/go-error-extra")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, errtest.Registration.Validate())
	assert.Equal(t, "yze/errtest", errtest.Registration.RuleID())
	assert.Same(t, errtest.Analyzer, errtest.Registration.Analyzer)
}

// TestADirectiveDoesNotRenameAFile pins the test-file scope to the name the
// FileSet holds, which the judged file cannot rewrite. A `//line` directive is a
// compiler feature for generated code: it changes what fset.Position reports and
// nothing else — the go tool still compiles the file, go list still names it in
// GoFiles or TestGoFiles, and `go test` still runs it. So a scope resolved
// through Position is decided by the very file being judged.
//
// This analyzer's matcher INCLUDES rather than exempts, so both directions are
// defects and both are pinned here: package forgeline's real test file claims a
// source name and is judged anyway, while its ordinary source file claims a test
// name and is left alone anyway.
//
// The rest of the package sits on the matcher's literal, because the identity
// is only as good as the comparison that reads it and every widening below
// survived this suite before its fixture existed. Kit's name contains "_test"
// and does not end in "_test.go"; Helper's ends in "test.go" with no underscore
// — the left edge, and the shape of this repository's own errtest.go; Golden's
// contains "_test.go" without ending in it; Cased's differs from "_test.go"
// only in letter case. All four are in GoFiles, so all four carry the banned
// bool expectation and none is reported, and each kills one widening that would
// point an error-testing rule at production source.
func TestADirectiveDoesNotRenameAFile(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errtest.Analyzer, "forgeline")
}
