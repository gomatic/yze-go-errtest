// The mechanism's EXTERNAL test package (import path go-error_test) is a test
// variant of the mechanism itself and stays exempt. It mirrors the real
// go-error's errs_test.go, whose message-rendering assertions are the contract
// under test there. No diagnostic is expected anywhere in this file.
package errs_test

import (
	"testing"

	errs "github.com/gomatic/go-error"
)

// spec carries the very shapes the analyzer bans elsewhere.
type spec struct {
	wantErr bool
	errMsg  string
}

func TestRenderedText(t *testing.T) {
	s := spec{wantErr: true, errMsg: "boom"}
	if got := errs.Const("boom").Error(); got != s.errMsg && s.wantErr {
		t.Fatalf("Error() = %q, want %q", got, s.errMsg)
	}
}
