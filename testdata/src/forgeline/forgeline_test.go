//line nottest.go:1
package forgeline

// Judged is the direction that disables the rule: a real test file — go list
// reports forgeline_test.go in TestGoFiles, and `go test` compiles and runs it
// — whose directive claims a source name. Reading the claimed name would take
// the whole file out of scope, so the bool expectation below is reported here
// exactly as it would be without the line above.
type Judged struct {
	name    string
	wantErr bool // want "expect errors as wantErr error matched with errors.Is, not a bool"
}

var _ = Judged{}
