package execution

import (
	"context"
	"strings"
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

	ir := &pipeline.IR{Jobs: []pipeline.Job{testJob("a"), testJob("b"), testJob("c")}}
	runner := &recordingRunner{delay: 20 * time.Millisecond}
	_, err := NewExecutor(runner, WithParallelism(1)).Execute(context.Background(), ir)
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

func TestDefaultSchedulerPreservesDAGOrder(t *testing.T) {
	t.Parallel()

	ir := &pipeline.IR{Jobs: []pipeline.Job{
		testJob("summary", "policy", "apply"),
		testJob("plan"),
		testJob("policy", "plan"),
		testJob("apply", "plan"),
	}}

	runner := &orderRunner{}
	_, err := NewExecutor(runner, WithParallelism(1)).Execute(context.Background(), ir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(runner.order) != 4 {
		t.Fatalf("order len = %d, want 4 (%v)", len(runner.order), runner.order)
	}
	if runner.order[0] != "plan" {
		t.Fatalf("first job = %q, want plan (%v)", runner.order[0], runner.order)
	}
	if runner.order[len(runner.order)-1] != "summary" {
		t.Fatalf("last job = %q, want summary (%v)", runner.order[len(runner.order)-1], runner.order)
	}
}

func TestExecutorRejectsInvalidPlanGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ir      *pipeline.IR
		wantErr string
	}{
		{
			name:    "duplicate names",
			ir:      &pipeline.IR{Jobs: []pipeline.Job{testJob("policy-check"), testJob("policy-check")}},
			wantErr: `duplicate job name "policy-check"`,
		},
		{
			name:    "unknown dependency",
			ir:      &pipeline.IR{Jobs: []pipeline.Job{testJob("summary", "policy-check")}},
			wantErr: `depends on unknown job "policy-check"`,
		},
		{
			name: "dependency cycle",
			ir: &pipeline.IR{Jobs: []pipeline.Job{
				testJob("summary", "policy-check"),
				testJob("policy-check", "summary"),
			}},
			wantErr: "dependency cycle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := &orderRunner{}
			_, err := NewExecutor(runner).Execute(context.Background(), tt.ir)
			if err == nil {
				t.Fatal("Execute() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Execute() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
			if len(runner.order) != 0 {
				t.Fatalf("runner executed %v, want no jobs", runner.order)
			}
		})
	}
}

func testJob(name string, deps ...string) pipeline.Job {
	return pipeline.Job{
		Name:         name,
		Kind:         pipeline.JobKindCommand,
		Dependencies: testDependencies(deps...),
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"true"},
		},
	}
}

func testDependencies(names ...string) []pipeline.JobDependency {
	deps := make([]pipeline.JobDependency, 0, len(names))
	for _, name := range names {
		deps = append(deps, pipeline.JobDependency{Job: name})
	}
	return deps
}
