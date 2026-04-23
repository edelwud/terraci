package runner

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/plugin"
)

type Options struct {
	Parallelism int
}

type Factory interface {
	Build(appCtx *plugin.AppContext, opts Options) (*Runtime, error)
}

type Runtime struct {
	ExecConfig execution.Config
	Workspace  execution.Workspace
	JobRunner  execution.JobRunner
}

type executionConfigResolver interface {
	Resolve(appCtx *plugin.AppContext, opts Options) execution.Config
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
	execCfg := f.configResolver.Resolve(appCtx, opts)
	binaryPath, err := f.binaryResolver.Resolve(execCfg.Binary)
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
		execEnv:   execCfg.Env,
	}
	terraformRunner := &terraformOperationRunner{
		binaryPath: binaryPath,
		workspace:  workspace,
		execConfig: execCfg,
	}

	return &Runtime{
		ExecConfig: execCfg,
		Workspace:  workspace,
		JobRunner: &jobRunner{
			phaseRunner: phaseRunner{
				commands: commandRunner,
				main: operationDispatcher{
					terraform: terraformRunner,
					commands:  commandRunner,
				},
			},
			commands: commandRunner,
		},
	}, nil
}

type defaultExecutionConfigResolver struct{}

func (defaultExecutionConfigResolver) Resolve(appCtx *plugin.AppContext, opts Options) execution.Config {
	execCfg := execution.ConfigFromProject(appCtx.Config())
	if opts.Parallelism > 0 {
		execCfg.Parallelism = opts.Parallelism
	}
	return execCfg
}

type defaultBinaryResolver struct{}

func (defaultBinaryResolver) Resolve(binary string) (string, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("resolve %s binary: %w", binary, err)
	}
	return path, nil
}
