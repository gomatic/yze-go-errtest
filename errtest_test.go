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
