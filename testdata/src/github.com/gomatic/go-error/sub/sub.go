// Package sub is a genuine subpackage of the sentinel mechanism; like its
// parent, its tests may assert rendered message text — the "/" boundary of
// the exemption covers it.
package sub

// Render returns the rendered message under test.
func Render() string { return "boom" }
