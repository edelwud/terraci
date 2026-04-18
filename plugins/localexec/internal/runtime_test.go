package internal

import (
	"context"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
)

type recordCommandRunner struct {
	commands []string
}

func (r *recordCommandRunner) Run(_ context.Context, spec commandSpec) error {
	r.commands = append(r.commands, spec.Command)
	return nil
}

type recordOperationRunner struct {
	operations []string
}

func (r *recordOperationRunner) Run(_ context.Context, job *pipeline.Job) error {
	r.operations = append(r.operations, job.Name)
	return nil
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
		Type:   pipeline.JobTypePlan,
		Module: discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
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
}
