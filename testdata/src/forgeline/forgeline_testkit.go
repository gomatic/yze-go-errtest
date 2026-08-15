package forgeline

// Kit sits at the matcher's boundary. Its name CONTAINS "_test" and does not
// END in "_test.go", so the go tool compiles it as ordinary source and this
// analyzer, which inspects test files only, must stay silent here. A matcher
// widened from a suffix to a substring would judge it.
type Kit struct {
	name    string
	wantErr bool
}

var _ = Kit{}
