package runner

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
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

type recordOperationRunner struct {
	operations []string
	err        error
}

func (r *recordOperationRunner) Run(_ context.Context, job *pipeline.Job) error {
	r.operations = append(r.operations, job.Name)
	return r.err
}

type recordTerraformRunner struct {
	plans   []string
	applies []string
	err     error
}

func (r *recordTerraformRunner) RunPlan(_ context.Context, job *pipeline.Job, _ *pipeline.TerraformOperation) error {
	r.plans = append(r.plans, job.Name)
	return r.err
}

func (r *recordTerraformRunner) RunApply(_ context.Context, job *pipeline.Job, _ *pipeline.TerraformOperation) error {
	r.applies = append(r.applies, job.Name)
	return r.err
}

func TestPhaseRunnerOrdersPreMainPost(t *testing.T) {
	t.Parallel()

	commands := &recordCommandRunner{}
	operations := &recordOperationRunner{}
	runner := phaseRunner{
		commands: commands,
		main:     operations,
	}

	job := &pipeline.Job{
		Name:   "plan-platform-stage-eu-central-1-vpc",
		Module: discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		Env:    map[string]string{"TF_WORKSPACE": "stage"},
		Steps: []pipeline.Step{
			{Phase: pipeline.PhasePrePlan, Command: "echo pre"},
			{Phase: pipeline.PhasePostPlan, Command: "echo post"},
		},
	}

	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := []string{commands.commands[0], operations.operations[0], commands.commands[1]}
	want := []string{"echo pre", job.Name, "echo post"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("execution order = %v, want %v", got, want)
	}
	for _, spec := range commands.specs {
		if spec.ModulePath != job.Module.RelativePath {
			t.Fatalf("module path = %q, want %q", spec.ModulePath, job.Module.RelativePath)
		}
		if !reflect.DeepEqual(spec.Env, job.Env) {
			t.Fatalf("env = %#v, want %#v", spec.Env, job.Env)
		}
	}
}

func TestPhaseRunnerUsesApplyPhasesForApplyJobs(t *testing.T) {
	t.Parallel()

	commands := &recordCommandRunner{}
	operations := &recordOperationRunner{}
	runner := phaseRunner{
		commands: commands,
		main:     operations,
	}

	job := &pipeline.Job{
		Name:      "apply-platform-stage-eu-central-1-vpc",
		Module:    discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		Operation: pipeline.Operation{Type: pipeline.OperationTypeTerraformApply},
		Steps: []pipeline.Step{
			{Phase: pipeline.PhasePrePlan, Command: "echo wrong-pre"},
			{Phase: pipeline.PhasePreApply, Command: "echo pre-apply"},
			{Phase: pipeline.PhasePostApply, Command: "echo post-apply"},
			{Phase: pipeline.PhasePostPlan, Command: "echo wrong-post"},
		},
	}

	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := []string{commands.commands[0], operations.operations[0], commands.commands[1]}
	want := []string{"echo pre-apply", job.Name, "echo post-apply"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("execution order = %v, want %v", got, want)
	}
}

func TestPhaseRunnerStopsWhenPreStepFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("pre failed")
	commands := &recordCommandRunner{err: wantErr}
	operations := &recordOperationRunner{}
	runner := phaseRunner{
		commands: commands,
		main:     operations,
	}

	err := runner.Run(context.Background(), &pipeline.Job{
		Name:   "plan-platform-stage-eu-central-1-vpc",
		Module: discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		Steps: []pipeline.Step{
			{Phase: pipeline.PhasePrePlan, Command: "echo pre"},
			{Phase: pipeline.PhasePostPlan, Command: "echo post"},
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if len(operations.operations) != 0 {
		t.Fatalf("operations = %v, want none", operations.operations)
	}
	if got := commands.commands; !reflect.DeepEqual(got, []string{"echo pre"}) {
		t.Fatalf("commands = %v, want pre only", got)
	}
}

func TestPhaseRunnerRejectsMissingCollaborators(t *testing.T) {
	t.Parallel()

	job := &pipeline.Job{
		Name:   "plan-platform-stage-eu-central-1-vpc",
		Module: discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	tests := []struct {
		name   string
		runner phaseRunner
	}{
		{name: "missing commands", runner: phaseRunner{main: &recordOperationRunner{}}},
		{name: "missing main", runner: phaseRunner{commands: &recordCommandRunner{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.runner.Run(context.Background(), job); err == nil {
				t.Fatal("Run() error = nil, want missing collaborator error")
			}
		})
	}
}

func TestOperationDispatcherRoutesTerraformOperations(t *testing.T) {
	t.Parallel()

	terraform := &recordTerraformRunner{}
	commands := &recordCommandRunner{}
	dispatcher := operationDispatcher{
		terraform: terraform,
		commands:  commands,
	}

	planJob := &pipeline.Job{
		Name: "plan-platform-stage-eu-central-1-vpc",
		Operation: pipeline.Operation{
			Type:      pipeline.OperationTypeTerraformPlan,
			Terraform: &pipeline.TerraformOperation{ModulePath: "platform/stage/eu-central-1/vpc"},
		},
	}
	applyJob := &pipeline.Job{
		Name: "apply-platform-stage-eu-central-1-vpc",
		Operation: pipeline.Operation{
			Type:      pipeline.OperationTypeTerraformApply,
			Terraform: &pipeline.TerraformOperation{ModulePath: "platform/stage/eu-central-1/vpc"},
		},
	}

	if err := dispatcher.Run(context.Background(), planJob); err != nil {
		t.Fatalf("Run(plan) error = %v", err)
	}
	if err := dispatcher.Run(context.Background(), applyJob); err != nil {
		t.Fatalf("Run(apply) error = %v", err)
	}

	if !reflect.DeepEqual(terraform.plans, []string{planJob.Name}) {
		t.Fatalf("plans = %v, want [%s]", terraform.plans, planJob.Name)
	}
	if !reflect.DeepEqual(terraform.applies, []string{applyJob.Name}) {
		t.Fatalf("applies = %v, want [%s]", terraform.applies, applyJob.Name)
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

	job := &pipeline.Job{
		Name:         "policy-check",
		Env:          map[string]string{"A": "B"},
		AllowFailure: true,
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"terraci policy check", "terraci summary"},
		},
	}
	if err := dispatcher.Run(context.Background(), job); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := commands.commands; !reflect.DeepEqual(got, job.Operation.Commands) {
		t.Fatalf("commands = %v, want %v", got, job.Operation.Commands)
	}
	for _, spec := range commands.specs {
		if spec.JobName != job.Name {
			t.Fatalf("job name = %q, want %q", spec.JobName, job.Name)
		}
		if !spec.AllowFailure {
			t.Fatal("allow failure = false, want true")
		}
		if !reflect.DeepEqual(spec.Env, job.Env) {
			t.Fatalf("env = %#v, want %#v", spec.Env, job.Env)
		}
	}
}

func TestJobRunnerStandaloneCommandsPropagateAllowFailure(t *testing.T) {
	t.Parallel()

	commands := &recordCommandRunner{}
	runner := &jobRunner{commands: commands}
	job := &pipeline.Job{
		Name:         "summary",
		AllowFailure: true,
		Operation: pipeline.Operation{
			Commands: []string{"terraci summary"},
		},
	}

	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(commands.specs) != 1 {
		t.Fatalf("command specs = %d, want 1", len(commands.specs))
	}
	if commands.specs[0].JobName != job.Name {
		t.Fatalf("job name = %q, want %q", commands.specs[0].JobName, job.Name)
	}
	if !commands.specs[0].AllowFailure {
		t.Fatal("allow failure = false, want true")
	}
}

func TestOperationDispatcherRejectsUnsupportedOperation(t *testing.T) {
	t.Parallel()

	err := operationDispatcher{}.Run(context.Background(), &pipeline.Job{
		Name: "unsupported",
		Operation: pipeline.Operation{
			Type: pipeline.OperationType("unsupported"),
		},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want unsupported operation error")
	}
}

func TestOperationDispatcherRejectsMissingCollaborators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *pipeline.Job
	}{
		{
			name: "missing terraform runner",
			job: &pipeline.Job{
				Name: "plan-platform-stage-eu-central-1-vpc",
				Operation: pipeline.Operation{
					Type:      pipeline.OperationTypeTerraformPlan,
					Terraform: &pipeline.TerraformOperation{ModulePath: "platform/stage/eu-central-1/vpc"},
				},
			},
		},
		{
			name: "missing command runner",
			job: &pipeline.Job{
				Name: "summary",
				Operation: pipeline.Operation{
					Type:     pipeline.OperationTypeCommands,
					Commands: []string{"terraci summary"},
				},
			},
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

func TestOperationDispatcherRejectsNilTerraformOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  pipeline.OperationType
	}{
		{name: "plan", typ: pipeline.OperationTypeTerraformPlan},
		{name: "apply", typ: pipeline.OperationTypeTerraformApply},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (operationDispatcher{}).Run(context.Background(), &pipeline.Job{
				Name: "terraform-" + tt.name,
				Operation: pipeline.Operation{
					Type: tt.typ,
				},
			})
			if err == nil {
				t.Fatal("Run() error = nil, want nil terraform operation error")
			}
		})
	}
}
