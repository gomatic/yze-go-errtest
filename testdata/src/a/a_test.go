package a

import (
	"errors"
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
