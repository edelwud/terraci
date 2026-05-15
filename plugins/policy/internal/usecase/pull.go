package usecase

import (
	"context"
	"errors"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func Pull(ctx context.Context, materializer SourceMaterializer, req policyengine.PullRequest) (*policyengine.PullResult, error) {
	if materializer == nil {
		return nil, errors.New("policy source materializer is nil")
	}

	dirs, err := materializer.Materialize(ctx, req.CacheDir)
	if err != nil {
		return nil, err
	}
	return &policyengine.PullResult{
		PolicyDirs: dirs,
		CacheDir:   materializer.CacheDir(req.CacheDir),
	}, nil
}
