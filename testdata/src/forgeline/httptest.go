package forgeline

// Helper sits at the matcher's LEFT edge — the underscore that separates a base
// name from the "_test.go" suffix. Its name ends in "test.go" and not in
// "_test.go", so the go tool compiles it as ordinary source and this analyzer,
// which inspects test files only, must stay silent here.
//
// This edge is not hypothetical and not latent: `find ~/src/github.com -name
// '*test.go' -not -name '*_test.go'` returns 39 files, among them
// net/http/httptest/httptest.go, gomatic/go-wofl/internal/pgtest/pgtest.go and
// this repository's own errtest.go. A matcher that dropped the underscore would
// point an error-TESTING rule at production source, where the shape it bans is
// not a defect and the author has no test to fix.
type Helper struct {
	name    string
	wantErr bool
}

var _ = Helper{}
