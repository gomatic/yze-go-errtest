package sub

import "testing"

// spec carries the banned shapes deliberately: the mechanism exemption covers
// its subpackages, so no diagnostic is expected anywhere in this file.
type spec struct {
	wantErr bool
	errMsg  string
}

func TestRendering(t *testing.T) {
	s := spec{wantErr: true, errMsg: "boom"}
	if got := Render(); got != s.errMsg && s.wantErr {
		t.Fatalf("Render() = %q, want %q", got, s.errMsg)
	}
}
