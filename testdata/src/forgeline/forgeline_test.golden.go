package forgeline

// Golden sits INSIDE the matcher's literal. Its name contains "_test.go" and
// does not end in it, so `go list` reports it in GoFiles and this analyzer must
// stay silent here. A matcher widened from a suffix to a substring would judge
// it.
//
// The fleet holds no file of this shape — the same find that returns 39 files
// for the left edge returns none for this one — and that absence is the reason
// the case is here rather than the reason to skip it: a widening nothing
// exercises is a widening nothing can fail on.
//
// The sibling escape, a package DIRECTORY named "*_test.go", is declined: it
// kills the same one widening and costs a second package in this corpus.
type Golden struct {
	name    string
	wantErr bool
}

var _ = Golden{}
