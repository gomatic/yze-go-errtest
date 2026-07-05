// Package a pins the errtest contract: the analyzer inspects ONLY test files,
// so nothing in this non-test file is reported — not even the exact shapes the
// analyzer bans in _test.go files.
package a

import errs "github.com/gomatic/go-error"

// spec uses a sentinel-typed expectation field in production code; wiring
// tables in non-test code are out of scope (unflagged control).
type spec struct {
	want    errs.Const
	wantErr bool
	errMsg  string
}

// use keeps the control fields referenced so the fixture compiles cleanly.
func use(s spec) (errs.Const, bool, string) { return s.want, s.wantErr, s.errMsg }
