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

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, errtest.Registration.Validate())
	assert.Equal(t, "yze/errtest", errtest.Registration.RuleID())
	assert.Same(t, errtest.Analyzer, errtest.Registration.Analyzer)
}
