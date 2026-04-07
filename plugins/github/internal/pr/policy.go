package pr

import (
	"github.com/edelwud/terraci/pkg/ci"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

type commentPolicy struct {
	config *configpkg.PRConfig
}

func newCommentPolicy(cfg *configpkg.PRConfig) commentPolicy {
	return commentPolicy{config: cfg}
}

func (p commentPolicy) enabled() bool {
	if p.config == nil {
		return true
	}
	return ci.CommentEnabled(p.config.Comment)
}
