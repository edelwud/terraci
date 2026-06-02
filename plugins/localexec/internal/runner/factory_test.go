package runner

import "testing"

type stubBinaryResolver struct {
	path string
	err  error
}

func (r stubBinaryResolver) Resolve(string) (string, error) {
	return r.path, r.err
}

func TestDefaultFactoryBuildWiresRuntime(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	serviceDir := t.TempDir()
	factory := defaultFactory{
		binaryResolver: stubBinaryResolver{path: "/bin/terraform"},
	}

	runtime, err := factory.Build(RuntimeOptions{
		WorkDir:         workDir,
		ServiceDir:      serviceDir,
		PlanParallelism: 4,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("Build() runtime = nil")
	}
	if runtime.Workspace.WorkDir() != workDir {
		t.Fatalf("workspace work dir = %q, want %q", runtime.Workspace.WorkDir(), workDir)
	}
	if runtime.Workspace.ServiceDir() != serviceDir {
		t.Fatalf("workspace service dir = %q, want %q", runtime.Workspace.ServiceDir(), serviceDir)
	}
	if runtime.JobRunner == nil {
		t.Fatal("job runner = nil")
	}

	jobRunner, ok := runtime.JobRunner.(*jobRunner)
	if !ok {
		t.Fatalf("job runner type = %T, want *jobRunner", runtime.JobRunner)
	}
	dispatcher, ok := jobRunner.main.(operationDispatcher)
	if !ok {
		t.Fatalf("operation runner type = %T, want operationDispatcher", jobRunner.main)
	}
	terraformRunner, ok := dispatcher.terraform.(*terraformOperationRunner)
	if !ok {
		t.Fatalf("terraform runner type = %T, want *terraformOperationRunner", dispatcher.terraform)
	}
	if terraformRunner.planParallelism != 4 {
		t.Fatalf("plan parallelism = %d, want 4", terraformRunner.planParallelism)
	}
	if terraformRunner.workspace.WorkDir() != workDir {
		t.Fatalf("terraform workspace work dir = %q, want %q", terraformRunner.workspace.WorkDir(), workDir)
	}
	if dispatcher.commands == nil {
		t.Fatal("command runner = nil")
	}
}
