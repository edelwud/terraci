package runner

import (
	"os"
	"os/exec"

	"github.com/edelwud/terraci/pkg/execution"
)

type RuntimeOptions struct {
	WorkDir         string
	ServiceDir      string
	PlanParallelism int
}

type Factory interface {
	Build(opts RuntimeOptions) (*Runtime, error)
}

type Runtime struct {
	Workspace execution.Workspace
	JobRunner execution.JobRunner
}

type binaryResolver interface {
	Resolve(binary string) (string, error)
}

type defaultFactory struct {
	binaryResolver binaryResolver
}

func NewFactory() Factory {
	return defaultFactory{
		binaryResolver: defaultBinaryResolver{},
	}
}

func (f defaultFactory) Build(opts RuntimeOptions) (*Runtime, error) {
	selfPath, err := os.Executable()
	if err != nil {
		selfPath = ""
	}

	workspace := execution.NewWorkspace(opts.WorkDir, opts.ServiceDir)
	commandRunner := &shellCommandRunner{
		workspace: workspace,
		selfPath:  selfPath,
	}
	terraformRunner := &terraformOperationRunner{
		workspace:       workspace,
		binaryResolver:  f.binaryResolver,
		planParallelism: opts.PlanParallelism,
	}

	return &Runtime{
		Workspace: workspace,
		JobRunner: &jobRunner{
			main: operationDispatcher{
				terraform: terraformRunner,
				commands:  commandRunner,
			},
		},
	}, nil
}

type defaultBinaryResolver struct{}

func (defaultBinaryResolver) Resolve(binary string) (string, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", err
	}
	return path, nil
}
