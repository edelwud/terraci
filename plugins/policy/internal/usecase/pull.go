package usecase

import (
	"context"
	"errors"

	"github.com/edelwud/terraci/plugins/policy/internal/source"
)

type PullRequest struct{}

type PullResult struct {
	PolicyDirs []string
	CacheDir   string
}

func Pull(ctx context.Context, materializer *source.Materializer, _ PullRequest) (*PullResult, error) {
	if materializer == nil {
		return nil, errors.New("policy source materializer is nil")
	}

	dirs, err := materializer.Materialize(ctx)
	if err != nil {
		return nil, err
	}
	return &PullResult{
		PolicyDirs: dirs,
		CacheDir:   materializer.CacheDir(),
	}, nil
}
