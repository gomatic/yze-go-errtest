package extra

import "testing"

// TestBoolExpectation carries the textbook banned shape; this package's
// import path merely prefix-extends the mechanism's, which exempts nothing.
func TestBoolExpectation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		wantErr bool // want `expect errors as wantErr error matched with errors.Is, not a bool`
	}{{name: "succeeds"}} {
		t.Run(tt.name, func(t *testing.T) {
			if err := works(); (err != nil) != tt.wantErr {
				t.Fatalf("works() = %v", err)
			}
		})
	}
}
