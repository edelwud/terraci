// Package reportrender renders producer-agnostic ci.Report payloads for
// markdown comments and terminal output.
//
// Producers publish render-ready sections through pkg/ci. This package owns
// presentation only and must not import producer plugin domain models.
package reportrender
