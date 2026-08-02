package a

import (
	"errors"
	"strings"
	"testing"

	errs "github.com/gomatic/go-error"
	"github.com/stretchr/testify/assert"
)

// ErrBoom is a sentinel the fixtures wrap and match against.
const ErrBoom errs.Const = "boom"

func doWork() error { return ErrBoom.With(nil) }

// TestCompliantShape is the sanctioned pattern: wantErr error + errors.Is.
func TestCompliantShape(t *testing.T) {
	tests := []struct {
		wantErr error // the one sanctioned expectation shape
		name    string
	}{
		{name: "fails", wantErr: ErrBoom},
		{name: "succeeds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := doWork()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			_ = err
		})
	}
}

// ptrErr implements error on the pointer receiver only, pinning the
// pointer-implements branch of the classifier.
type ptrErr struct{}

func (*ptrErr) Error() string { return "ptr" }

// TestSentinelTypedExpectations pins the banned expectation field types.
func TestSentinelTypedExpectations(t *testing.T) {
	tests := []struct {
		want     errs.Const // want `test error expectations must be typed error \(wantErr error\), not a concrete or custom error type`
		wantErr  errs.Const // want `test error expectations must be typed error \(wantErr error\), not a concrete or custom error type`
		wantPtr  ptrErr     // want `test error expectations must be typed error \(wantErr error\), not a concrete or custom error type`
		sentinel errs.Const // input, not an expectation: typed inputs stay sanctioned (calling .With needs the sentinel)
		name     string
	}{}
	_ = tests
	_ = t
}

// TestBoolAndMessageExpectations pins the banned bool/message expectation shapes.
func TestBoolAndMessageExpectations(t *testing.T) {
	tests := []struct {
		wantErr    bool   // want `expect errors as wantErr error matched with errors.Is, not a bool`
		wantErrMsg string // want `expect errors as wantErr error matched with errors.Is, not a message string`
		wantMsg    string // no err in the name: plain message expectations are out of scope
		name       string
	}{}
	_ = tests
	_ = t
}

// TestStringMatching pins the banned testify message-matching assertions, in
// both the package-function and assert.New receiver forms; ErrorIs stays clean.
func TestStringMatching(t *testing.T) {
	err := doWork()
	assert.EqualError(t, err, "boom")       // want `match errors with errors.Is, never message strings`
	assert.ErrorContains(t, err, "boo")     // want `match errors with errors.Is, never message strings`
	assert.ErrorIs(t, err, ErrBoom)         // sanctioned
	assert.ErrorIs(t, err, errors.New("x")) // non-testify-matching call shapes stay out of scope for other analyzers
	want := assert.New(t)
	want.EqualError(err, "boom")   // want `match errors with errors.Is, never message strings`
	want.ErrorContains(err, "boo") // want `match errors with errors.Is, never message strings`
	want.ErrorIs(err, ErrBoom)     // sanctioned
}

// TestErrorTextMatching pins the hand-rolled spelling of message matching:
// err.Error() fed to a strings matcher or compared against a literal. Printing
// the text in a failure message stays sanctioned — rendering is not matching.
func TestErrorTextMatching(t *testing.T) {
	err := doWork()
	if strings.Contains(err.Error(), "boom") { // want `never match on err.Error\(\) text`
		t.Log("matched")
	}
	if strings.HasPrefix(err.Error(), "b") { // want `never match on err.Error\(\) text`
		t.Log("matched")
	}
	if err.Error() == "boom" { // want `never match on err.Error\(\) text`
		t.Log("matched")
	}
	if "boom" != err.Error() { // want `never match on err.Error\(\) text`
		t.Log("matched")
	}
	t.Log(err.Error())                        // rendering, not matching: sanctioned
	_ = strings.Contains("boom", "oo")        // strings matcher without error text: sanctioned
	_ = stringify(err) == "boom"              // a non-Error method rendering is out of scope
	_ = strings.Contains(stringify(err), "x") // likewise as a matcher argument
}

// stringify renders an error through something other than Error().
func stringify(err error) string {
	if err == nil {
		return ""
	}
	return "wrapped: " + err.Error()
}

// TestErrorTextRenderingContracts pins the two sanctioned rendering shapes:
// pinning a SENTINEL CONSTANT's message text (the message is its declared
// contract), and asserting rendered context on an error the same function
// already discriminates with an Is-style matcher.
func TestErrorTextRenderingContracts(t *testing.T) {
	if ErrBoom.Error() != "boom" { // a sentinel's own text is its contract: sanctioned
		t.Fatal("sentinel text")
	}
	if strings.Contains(ErrBoom.Error(), "oom") { // likewise through a matcher
		t.Log("contains")
	}
	err := doWork()
	assert.ErrorIs(t, err, ErrBoom)
	if !strings.Contains(err.Error(), "boom") { // rendering asserted AFTER Is: sanctioned
		t.Fatal("context rendering")
	}
}

// pkgErr and pkgMatched exercise the package-scope shape: no enclosing
// function means no Is-exemption is possible, and a non-identifier receiver
// can carry no object to exempt.
var (
	pkgErr     = doWork()
	pkgMatched = pkgErr.Error() == "boom" // want `never match on err.Error\(\) text`
)

// coder has an Error method whose ARITY disqualifies it as the error
// interface's rendering.
type coder struct{}

// Error renders a specific code; the parameter means this is not error.Error.
func (coder) Error(code int) string {
	if code == 0 {
		return ""
	}
	return "code"
}

// fake carries a zero-arg Error method whose return type keeps it OFF the
// error interface.
type fake struct{}

// Error returns a count, not a message.
func (fake) Error() int { return 0 }

// TestErrorTextEdgeShapes pins the remaining discrimination shapes: a fresh
// call result matched by text, a struct-field receiver, an Is-style match of
// a DIFFERENT error (which exempts nothing), a closure-local exemption, and
// an arity-mismatched Error method that is no rendering at all.
func TestErrorTextEdgeShapes(t *testing.T) {
	if doWork().Error() == "boom" { // want `never match on err.Error\(\) text`
		t.Log("fresh result")
	}

	holder := struct{ err error }{err: doWork()}
	if strings.Contains(holder.err.Error(), "boom") { // want `never match on err.Error\(\) text`
		t.Log("field receiver")
	}

	err, other := doWork(), doWork()
	assert.ErrorIs(t, other, ErrBoom)
	if err.Error() == "boom" { // want `never match on err.Error\(\) text`
		t.Log("a different error's Is exempts nothing")
	}

	func() {
		inner := doWork()
		if !errors.Is(inner, ErrBoom) {
			t.Fatal("closure discriminates properly")
		}
		if strings.Contains(inner.Error(), "boom") { // exempt: Is-asserted in the SAME closure
			t.Log("closure rendering")
		}
	}()

	if (coder{}).Error(1) == "code" { // an arity-mismatched Error is not the error interface
		t.Log("not a rendering")
	}
	if (fake{}).Error() == 0 { // a zero-arg Error returning non-string implements nothing
		t.Log("not an error either")
	}
	_ = pkgMatched
}
