package pr

import (
	"github.com/edelwud/terraci/pkg/ciprovider"
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
	return ciprovider.CommentEnabled(p.config.Comment)
}
