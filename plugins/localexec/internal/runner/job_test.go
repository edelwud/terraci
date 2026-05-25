package runner

import (
	"context"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
)

type recordCommandRunner struct {
	commands []string
	specs    []commandSpec
	err      error
}

func (r *recordCommandRunner) Run(_ context.Context, spec commandSpec) error {
	r.commands = append(r.commands, spec.Command)
	r.specs = append(r.specs, spec)
	return r.err
}

type recordTerraformRunner struct {
	plans   []string
	applies []string
	err     error
}

func (r *recordTerraformRunner) RunPlan(_ context.Context, job *pipeline.Job, _ *pipeline.TerraformOperation) error {
	r.plans = append(r.plans, job.Name())
	return r.err
}

func (r *recordTerraformRunner) RunApply(_ context.Context, job *pipeline.Job, _ *pipeline.TerraformOperation) error {
	r.applies = append(r.applies, job.Name())
	return r.err
}

func TestOperationDispatcherRoutesTerraformOperations(t *testing.T) {
	t.Parallel()

	terraform := &recordTerraformRunner{}
	commands := &recordCommandRunner{}
	dispatcher := operationDispatcher{
		terraform: terraform,
		commands:  commands,
	}

	ir := pipelinetest.MustSingleModuleIR(t, discovery.TestModule("platform", "stage", "eu-central-1", "vpc"))
	plan := pipelinetest.MustJobByKind(t, ir, pipeline.JobKindPlan)
	apply := pipelinetest.MustJobByKind(t, ir, pipeline.JobKindApply)
	planJob := &plan
	applyJob := &apply

	if err := dispatcher.Run(context.Background(), planJob); err != nil {
		t.Fatalf("Run(plan) error = %v", err)
	}
	if err := dispatcher.Run(context.Background(), applyJob); err != nil {
		t.Fatalf("Run(apply) error = %v", err)
	}

	if !reflect.DeepEqual(terraform.plans, []string{planJob.Name()}) {
		t.Fatalf("plans = %v, want [%s]", terraform.plans, planJob.Name())
	}
	if !reflect.DeepEqual(terraform.applies, []string{applyJob.Name()}) {
		t.Fatalf("applies = %v, want [%s]", terraform.applies, applyJob.Name())
	}
	if len(commands.commands) != 0 {
		t.Fatalf("commands = %v, want none", commands.commands)
	}
}

func TestOperationDispatcherRunsCommandOperationsWithJobMetadata(t *testing.T) {
	t.Parallel()

	commands := &recordCommandRunner{}
	dispatcher := operationDispatcher{
		terraform: &recordTerraformRunner{},
		commands:  commands,
	}

	commandJob := pipelinetest.MustCommandJob(t, pipeline.ContributedJobOptions{
		Name:         "policy-check",
		Commands:     []string{"terraci policy check", "terraci summary"},
		AllowFailure: true,
	})
	job := &commandJob
	if err := dispatcher.Run(context.Background(), job); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := commands.commands, job.Operation().Commands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
	for _, spec := range commands.specs {
		if spec.JobName != job.Name() {
			t.Fatalf("job name = %q, want %q", spec.JobName, job.Name())
		}
		if !spec.AllowFailure {
			t.Fatal("allow failure = false, want true")
		}
		if len(spec.Env) != 0 {
			t.Fatalf("env = %#v, want empty", spec.Env)
		}
	}
}

func TestJobRunnerStandaloneCommandsPropagateAllowFailure(t *testing.T) {
	t.Parallel()

	commands := &recordCommandRunner{}
	runner := &jobRunner{main: operationDispatcher{commands: commands}}
	commandJob := pipelinetest.MustCommandJob(t, pipeline.ContributedJobOptions{Name: "summary", Commands: []string{"terraci summary"}, AllowFailure: true})
	job := &commandJob

	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(commands.specs) != 1 {
		t.Fatalf("command specs = %d, want 1", len(commands.specs))
	}
	if commands.specs[0].JobName != job.Name() {
		t.Fatalf("job name = %q, want %q", commands.specs[0].JobName, job.Name())
	}
	if !commands.specs[0].AllowFailure {
		t.Fatal("allow failure = false, want true")
	}
}

func TestOperationDispatcherRejectsMissingCollaborators(t *testing.T) {
	t.Parallel()
	ir := pipelinetest.MustSingleModuleIR(t, discovery.TestModule("platform", "stage", "eu-central-1", "vpc"))
	plan := pipelinetest.MustJobByKind(t, ir, pipeline.JobKindPlan)
	command := pipelinetest.MustCommandJob(t, pipeline.ContributedJobOptions{Name: "summary", Commands: []string{"terraci summary"}})

	tests := []struct {
		name string
		job  *pipeline.Job
	}{
		{
			name: "missing terraform runner",
			job:  &plan,
		},
		{
			name: "missing command runner",
			job:  &command,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := (operationDispatcher{}).Run(context.Background(), tt.job); err == nil {
				t.Fatal("Run() error = nil, want missing collaborator error")
			}
		})
	}
}

func TestOperationDispatcherRejectsNilJob(t *testing.T) {
	t.Parallel()

	err := (operationDispatcher{}).Run(context.Background(), nil)
	if err == nil {
		t.Fatal("Run() error = nil, want nil job error")
	}
}
