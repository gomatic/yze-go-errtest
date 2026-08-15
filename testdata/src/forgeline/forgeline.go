//line zz_test.go:1
package forgeline

// Source is ordinary compiled source — go list reports forgeline.go in GoFiles
// — and the directive above is the only thing claiming a test name for it. The
// analyzer inspects test files only, so reading the claimed name here does not
// silence a rule: it points the rule at production code, where the shape it
// bans is not a defect and the author has no test to fix.
type Source struct {
	name    string
	wantErr bool
}

var _ = Source{}
