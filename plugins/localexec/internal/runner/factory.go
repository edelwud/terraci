package runner

import (
	"os"
	"os/exec"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/terraformrun"
)

type Options struct {
	Parallelism int
}

type Factory interface {
	Build(appCtx *plugin.AppContext, opts Options) (*Runtime, error)
}

type Runtime struct {
	Profile   terraformrun.Profile
	Workspace execution.Workspace
	JobRunner execution.JobRunner
}

type executionConfigResolver interface {
	Resolve(appCtx *plugin.AppContext, opts Options) (terraformrun.Profile, error)
}

type binaryResolver interface {
	Resolve(binary string) (string, error)
}

type defaultFactory struct {
	configResolver executionConfigResolver
	binaryResolver binaryResolver
}

func NewFactory() Factory {
	return defaultFactory{
		configResolver: defaultExecutionConfigResolver{},
		binaryResolver: defaultBinaryResolver{},
	}
}

func (f defaultFactory) Build(appCtx *plugin.AppContext, opts Options) (*Runtime, error) {
	profile, err := f.configResolver.Resolve(appCtx, opts)
	if err != nil {
		return nil, err
	}

	selfPath, err := os.Executable()
	if err != nil {
		selfPath = ""
	}

	workspace := execution.NewWorkspace(appCtx.WorkDir(), appCtx.ServiceDir())
	commandRunner := &shellCommandRunner{
		workspace: workspace,
		selfPath:  selfPath,
	}
	terraformRunner := &terraformOperationRunner{
		workspace:       workspace,
		binaryResolver:  f.binaryResolver,
		planParallelism: profile.Parallelism(),
	}

	return &Runtime{
		Profile:   profile,
		Workspace: workspace,
		JobRunner: &jobRunner{
			main: operationDispatcher{
				terraform: terraformRunner,
				commands:  commandRunner,
			},
		},
	}, nil
}

type defaultExecutionConfigResolver struct{}

func (defaultExecutionConfigResolver) Resolve(appCtx *plugin.AppContext, opts Options) (terraformrun.Profile, error) {
	profile, err := terraformrun.ProfileFromConfig(appCtx.Config())
	if err != nil {
		return terraformrun.Profile{}, err
	}
	if opts.Parallelism > 0 {
		return profile.WithParallelism(opts.Parallelism)
	}
	return profile, nil
}

type defaultBinaryResolver struct{}

func (defaultBinaryResolver) Resolve(binary string) (string, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", err
	}
	return path, nil
}
