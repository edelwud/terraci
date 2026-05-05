// Package ciplugin hosts shared helpers for CI provider plugins (gitlab,
// github, and any future Bitbucket / Jenkins / Azure DevOps plugin).
//
// The package lives under plugins/internal/ rather than pkg/ to keep core
// plugin-agnostic. External CI provider plugins authored against the public
// SDK can copy these helpers into their own internal/ package — they are
// small enough to inline.
package ciplugin

import "github.com/edelwud/terraci/pkg/ci"

// CommentBlockSource is the minimal contract a provider's MR/PR config block
// needs in order to use CommentEnabled. Both gitlab.MRConfig and
// github.PRConfig satisfy it via their Comment field.
type CommentBlockSource interface {
	// CommentBlock returns the *ci.MRCommentConfig pointer (may be nil).
	CommentBlock() *ci.MRCommentConfig
}

// CommentEnabled returns whether posting an MR/PR comment is permitted for
// the given provider config block. A nil block (provider didn't configure
// any comment integration) defaults to enabled, matching the previous
// per-provider commentPolicy.enabled() behavior.
func CommentEnabled(src CommentBlockSource) bool {
	if src == nil {
		return true
	}
	return ci.CommentEnabled(src.CommentBlock())
}
