package forgeline

// Cased differs from a test file's name only in letter case. The go tool's own
// check is `strings.HasSuffix(name, "_test.go")` and is case-sensitive, so this
// is ordinary compiled source — verified on a case-INSENSITIVE darwin
// filesystem, where `go list` still reports it in GoFiles and never in
// TestGoFiles. Nothing here is a test, so nothing here is reported.
//
// Case is the third dimension this package's names would otherwise hold
// constant. Folding the name before matching is the ordinary instinct of anyone
// who has been bitten by a Windows or macOS path.
type Cased struct {
	name    string
	wantErr bool
}

var _ = Cased{}
