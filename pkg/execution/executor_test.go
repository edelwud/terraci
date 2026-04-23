package execution

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type recordingRunner struct {
	active    atomic.Int32
	maxActive atomic.Int32
	delay     time.Duration
}

func (r *recordingRunner) Run(ctx context.Context, _ *pipeline.Job) error {
	current := r.active.Add(1)
	defer r.active.Add(-1)

	for {
		maxVal := r.maxActive.Load()
		if current <= maxVal || r.maxActive.CompareAndSwap(maxVal, current) {
			break
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(r.delay):
		return nil
	}
}

func TestExecutorHonorsParallelism(t *testing.T) {
	t.Parallel()

	plan := NewPlan(&pipeline.IR{
		Jobs: []pipeline.Job{
			{Name: "pre-a", Phase: pipeline.PhasePrePlan},
			{Name: "pre-b", Phase: pipeline.PhasePrePlan},
			{Name: "pre-c", Phase: pipeline.PhasePrePlan},
		},
	})

	runner := &recordingRunner{delay: 20 * time.Millisecond}
	_, err := NewExecutor(runner, WithParallelism(1)).Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := runner.maxActive.Load(); got != 1 {
		t.Fatalf("max concurrency = %d, want 1", got)
	}
}

type orderRunner struct {
	mu    sync.Mutex
	order []string
}

func (r *orderRunner) Run(_ context.Context, job *pipeline.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, job.Name)
	return nil
}

func TestDefaultSchedulerPreservesGroupOrder(t *testing.T) {
	t.Parallel()

	plan := NewPlan(&pipeline.IR{
		Levels: []pipeline.Level{
			{Index: 0, Modules: []pipeline.ModuleJobs{{Plan: &pipeline.Job{Name: "plan-0"}, Apply: &pipeline.Job{Name: "apply-0"}}}},
			{Index: 1, Modules: []pipeline.ModuleJobs{{Plan: &pipeline.Job{Name: "plan-1"}, Apply: &pipeline.Job{Name: "apply-1"}}}},
		},
		Jobs: []pipeline.Job{
			{Name: "pre", Phase: pipeline.PhasePrePlan},
			{Name: "post", Phase: pipeline.PhasePostPlan},
			{Name: "final", Phase: pipeline.PhaseFinalize},
		},
	})

	runner := &orderRunner{}
	_, err := NewExecutor(runner, WithParallelism(1)).Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := []string{"pre", "plan-0", "plan-1", "post", "apply-0", "apply-1", "final"}
	if len(runner.order) != len(want) {
		t.Fatalf("order len = %d, want %d (%v)", len(runner.order), len(want), runner.order)
	}
	for i := range want {
		if runner.order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q (%v)", i, runner.order[i], want[i], runner.order)
		}
	}
}

func TestDefaultSchedulerAlwaysIncludesFinalizeStage(t *testing.T) {
	t.Parallel()

	plan := NewPlan(&pipeline.IR{
		Levels: []pipeline.Level{
			{Index: 0, Modules: []pipeline.ModuleJobs{{Plan: &pipeline.Job{Name: "plan-0"}}}},
		},
	})

	groups := DefaultScheduler{}.Schedule(plan)
	if len(groups) == 0 {
		t.Fatal("expected scheduled groups")
	}

	last := groups[len(groups)-1]
	if last.Name != "finalize" {
		t.Fatalf("last group = %q, want finalize", last.Name)
	}
	if len(last.Jobs) != 0 {
		t.Fatalf("finalize jobs = %d, want 0", len(last.Jobs))
	}
}

func TestExecutorRecordsEmptyFinalizeStage(t *testing.T) {
	t.Parallel()

	plan := NewPlan(&pipeline.IR{
		Levels: []pipeline.Level{
			{Index: 0, Modules: []pipeline.ModuleJobs{{Plan: &pipeline.Job{Name: "plan-0"}}}},
		},
	})

	runner := &orderRunner{}
	result, err := NewExecutor(runner, WithParallelism(1)).Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Groups) == 0 {
		t.Fatal("expected execution groups to be recorded")
	}

	last := result.Groups[len(result.Groups)-1]
	if last.Name != "finalize" {
		t.Fatalf("last recorded group = %q, want finalize", last.Name)
	}
	if last.JobCount != 0 {
		t.Fatalf("finalize job count = %d, want 0", last.JobCount)
	}
}
