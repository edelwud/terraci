package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/edelwud/terraci/pkg/execution"
)

type commandRunner interface {
	Run(ctx context.Context, spec commandSpec) error
}

type commandSpec struct {
	JobName      string
	ModulePath   string
	Command      string
	Env          map[string]string
	AllowFailure bool
}

type shellCommandRunner struct {
	workspace execution.Workspace
	selfPath  string
	execEnv   map[string]string
}

func (r *shellCommandRunner) Run(ctx context.Context, spec commandSpec) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-lc", rewriteTerraciCommand(spec.Command, r.selfPath)) //nolint:gosec // user-controlled commands are intentional
	if spec.ModulePath != "" {
		cmd.Dir = r.workspace.ModuleDir(spec.ModulePath)
	} else {
		cmd.Dir = r.workspace.WorkDir()
	}

	cmd.Env = os.Environ()
	for key, value := range mergeEnv(r.execEnv, spec.Env) {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && spec.AllowFailure {
		fmt.Fprintln(os.Stderr, stderr.String())
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: run %q: %w: %s", spec.JobName, spec.Command, err, stderr.String())
	}
	return nil
}
