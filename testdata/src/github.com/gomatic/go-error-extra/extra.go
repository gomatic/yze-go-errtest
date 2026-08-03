// Package extra sits at an import path that PREFIX-EXTENDS the mechanism's
// (github.com/gomatic/go-error-extra). It is NOT the mechanism, not a
// subpackage, and not a test variant, so the analyzer must inspect it in
// full — the exemption stops at a path boundary, never at a string prefix.
package extra

// works reports success; the test table wires it up.
func works() error { return nil }
