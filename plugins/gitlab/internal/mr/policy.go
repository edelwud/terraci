package mr

import (
	"github.com/edelwud/terraci/pkg/ciprovider"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

type commentPolicy struct {
	config *configpkg.MRConfig
}

func newCommentPolicy(cfg *configpkg.MRConfig) commentPolicy {
	return commentPolicy{config: cfg}
}

func (p commentPolicy) enabled() bool {
	if p.config == nil {
		return true
	}
	return ciprovider.CommentEnabled(p.config.Comment)
}
