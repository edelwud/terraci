package execution

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type boundedWorkerPool struct {
	parallelism int
}

func (p boundedWorkerPool) Run(ctx context.Context, jobs []*pipeline.Job, fn func(context.Context, *pipeline.Job) error) error {
	if len(jobs) == 0 {
		return nil
	}

	limit := p.parallelism
	if limit <= 0 || limit > len(jobs) {
		limit = len(jobs)
	}

	var (
		group, runCtx = errgroup.WithContext(ctx)
		sem           = make(chan struct{}, limit)
	)

	for _, job := range jobs {
		group.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-runCtx.Done():
				return runCtx.Err()
			}

			defer func() { <-sem }()
			return fn(runCtx, job)
		})
	}

	return group.Wait()
}

var _ WorkerPool = boundedWorkerPool{}
