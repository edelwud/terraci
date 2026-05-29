package execution

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
)

type recordingRunner struct {
	active    atomic.Int32
	maxActive atomic.Int32
	delay     time.Duration
}

func (r *recordingRunner) Run(ctx context.Context, _ pipeline.Job) error {
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

	ir := pipelinetest.MustCommandIR(t,
		testJob("a"),
		testJob("b"),
		testJob("c"),
	)
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

func (r *orderRunner) Run(_ context.Context, job pipeline.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, job.Name())
	return nil
}

func TestDefaultSchedulerPreservesDAGOrder(t *testing.T) {
	t.Parallel()

	ir := pipelinetest.MustCommandIR(t,
		testJob("summary", "policy", "apply"),
		testJob("plan"),
		testJob("policy", "plan"),
		testJob("apply", "plan"),
	)

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

func TestExecutorRejectsInvalidIR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ir      *pipeline.IR
		wantErr string
	}{
		{
			name:    "nil",
			ir:      nil,
			wantErr: "execution IR is nil",
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

func TestExecutorPropagatesSchedulerError(t *testing.T) {
	t.Parallel()

	want := errors.New("scheduler failed")
	ir := pipelinetest.MustCommandIR(t, testJob("plan"))
	_, err := NewExecutor(&orderRunner{}, WithScheduler(errorScheduler{err: want})).Execute(context.Background(), ir)
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want scheduler error", err)
	}
}

func TestExecutorReturnsPartialResultAndTypedExecutionError(t *testing.T) {
	t.Parallel()

	want := errors.New("terraform failed")
	ir := pipelinetest.MustCommandIR(t, testJob("plan"))
	result, err := NewExecutor(failingRunner{err: want}).Execute(context.Background(), ir)
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
	var execErr *ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("Execute() error %T does not wrap ExecutionError", err)
	}
	if execErr.JobName != "plan" {
		t.Fatalf("ExecutionError.JobName = %q, want plan", execErr.JobName)
	}
	if result == nil {
		t.Fatal("Execute() result = nil, want partial result")
	}
	failed, ok := result.Failed()
	if !ok || failed.Name() != "plan" || !errors.Is(failed.Err(), want) {
		t.Fatalf("Failed() = %#v, %v; want plan failure", failed, ok)
	}
}

type delayedRunner struct {
	delays map[string]time.Duration
}

func (r delayedRunner) Run(ctx context.Context, job pipeline.Job) error {
	if delay := r.delays[job.Name()]; delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil
}

func TestExecutorResultsKeepScheduleOrderUnderParallelism(t *testing.T) {
	t.Parallel()

	ir := pipelinetest.MustCommandIR(t,
		testJob("a"),
		testJob("b"),
	)
	result, err := NewExecutor(delayedRunner{delays: map[string]time.Duration{"a": 40 * time.Millisecond}}, WithParallelism(2)).Execute(context.Background(), ir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	jobs := result.Jobs()
	if got := []string{jobs[0].Name(), jobs[1].Name()}; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("result job order = %v, want [a b]", got)
	}
}

func TestExecutorRecordsProducedArtifacts(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	ir := pipelinetest.MustSingleModuleIR(t, module)
	result, err := NewExecutor(&orderRunner{}, WithParallelism(1)).Execute(context.Background(), ir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	planJob := pipelinetest.MustJobByKind(t, ir, pipeline.JobKindPlan)
	planResult := findJobResult(t, result, planJob.Name())
	artifact, ok := planResult.ProducedArtifact()
	if !ok {
		t.Fatal("ProducedArtifact() ok = false, want plan artifact")
	}
	if artifact.Name != planJob.OutputArtifact().Name {
		t.Fatalf("artifact name = %q, want %q", artifact.Name, planJob.OutputArtifact().Name)
	}
}

func testJob(name string, deps ...string) pipeline.ContributedJobOptions {
	return pipeline.ContributedJobOptions{
		Name:         name,
		Dependencies: testDependencies(deps...),
		Commands:     []string{"true"},
	}
}

func testDependencies(names ...string) []pipeline.JobDependency {
	deps := make([]pipeline.JobDependency, 0, len(names))
	for _, name := range names {
		deps = append(deps, pipeline.JobDependency{Job: name})
	}
	return deps
}

type errorScheduler struct {
	err error
}

func (s errorScheduler) Schedule(*pipeline.IR) ([]pipeline.JobGroup, error) {
	return nil, s.err
}

type failingRunner struct {
	err error
}

func (r failingRunner) Run(context.Context, pipeline.Job) error {
	return r.err
}

func findJobResult(tb testing.TB, result *Result, name string) JobResult {
	tb.Helper()
	for _, job := range result.Jobs() {
		if job.Name() == name {
			return job
		}
	}
	tb.Fatalf("job result %q not found", name)
	return JobResult{}
}
