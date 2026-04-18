package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

type runtimeOptions struct {
	parallelism int
}

type executionConfigResolver interface {
	Resolve(appCtx *plugin.AppContext, opts runtimeOptions) execution.Config
}

type binaryResolver interface {
	Resolve(binary string) (string, error)
}

type runtimeFactory interface {
	Build(appCtx *plugin.AppContext, opts runtimeOptions) (*runtime, error)
}

type defaultRuntimeFactory struct {
	configResolver executionConfigResolver
	binaryResolver binaryResolver
}

func newRuntimeFactory() runtimeFactory {
	return defaultRuntimeFactory{
		configResolver: defaultExecutionConfigResolver{},
		binaryResolver: defaultBinaryResolver{},
	}
}

func (f defaultRuntimeFactory) Build(appCtx *plugin.AppContext, opts runtimeOptions) (*runtime, error) {
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

	return &runtime{
		execConfig: execCfg,
		workspace:  workspace,
		jobRunner: &jobRunner{
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

func (defaultExecutionConfigResolver) Resolve(appCtx *plugin.AppContext, opts runtimeOptions) execution.Config {
	execCfg := execution.ConfigFromProject(appCtx.Config())
	if opts.parallelism > 0 {
		execCfg.Parallelism = opts.parallelism
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

type runtime struct {
	execConfig execution.Config
	workspace  execution.Workspace
	jobRunner  execution.JobRunner
}

func (r *runtime) JobRunner() execution.JobRunner {
	return r.jobRunner
}

type terraformRunner interface {
	RunPlan(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error
	RunApply(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error
}

type commandRunner interface {
	Run(ctx context.Context, spec commandSpec) error
}

type operationRunner interface {
	Run(ctx context.Context, job *pipeline.Job) error
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

type terraformOperationRunner struct {
	binaryPath string
	workspace  execution.Workspace
	execConfig execution.Config
}

func (r *terraformOperationRunner) RunPlan(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error {
	tf, err := tfexec.NewTerraform(r.workspace.ModuleDir(op.ModulePath), r.binaryPath)
	if err != nil {
		return fmt.Errorf("%s: create terraform runner: %w", job.Name, err)
	}
	if err = tf.SetEnv(mergeEnv(environMap(), r.execConfig.Env, job.Env)); err != nil {
		return fmt.Errorf("%s: set env: %w", job.Name, err)
	}

	if op.InitEnabled {
		if err = tf.Init(ctx); err != nil {
			return fmt.Errorf("%s: init: %w", job.Name, err)
		}
	}

	opts := []tfexec.PlanOption{tfexec.Out(filepath.Base(op.PlanFile))}
	if r.execConfig.Parallelism > 0 {
		opts = append(opts, tfexec.Parallelism(r.execConfig.Parallelism))
	}
	if _, err = tf.Plan(ctx, opts...); err != nil {
		return fmt.Errorf("%s: plan: %w", job.Name, err)
	}

	if !op.DetailedPlan {
		return nil
	}

	var raw string
	raw, err = tf.ShowPlanFileRaw(ctx, filepath.Base(op.PlanFile))
	if err != nil {
		return fmt.Errorf("%s: show plan text: %w", job.Name, err)
	}
	if err = os.WriteFile(r.workspace.PlanTextFile(op.ModulePath), []byte(raw), 0o600); err != nil {
		return fmt.Errorf("%s: write plan.txt: %w", job.Name, err)
	}

	planJSON, err := tf.ShowPlanFile(ctx, filepath.Base(op.PlanFile))
	if err != nil {
		return fmt.Errorf("%s: show plan json: %w", job.Name, err)
	}
	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: marshal plan.json: %w", job.Name, err)
	}
	if err = os.WriteFile(r.workspace.PlanJSONFile(op.ModulePath), data, 0o600); err != nil {
		return fmt.Errorf("%s: write plan.json: %w", job.Name, err)
	}

	return nil
}

func (r *terraformOperationRunner) RunApply(ctx context.Context, job *pipeline.Job, op *pipeline.TerraformOperation) error {
	tf, err := tfexec.NewTerraform(r.workspace.ModuleDir(op.ModulePath), r.binaryPath)
	if err != nil {
		return fmt.Errorf("%s: create terraform runner: %w", job.Name, err)
	}
	if err = tf.SetEnv(mergeEnv(environMap(), r.execConfig.Env, job.Env)); err != nil {
		return fmt.Errorf("%s: set env: %w", job.Name, err)
	}

	if op.InitEnabled {
		if err = tf.Init(ctx); err != nil {
			return fmt.Errorf("%s: init: %w", job.Name, err)
		}
	}

	var opts []tfexec.ApplyOption
	if op.UsePlanFile {
		opts = append(opts, tfexec.DirOrPlan(filepath.Base(op.PlanFile)))
	}
	if err := tf.Apply(ctx, opts...); err != nil {
		return fmt.Errorf("%s: apply: %w", job.Name, err)
	}
	return nil
}

type operationDispatcher struct {
	terraform terraformRunner
	commands  commandRunner
}

func (r operationDispatcher) Run(ctx context.Context, job *pipeline.Job) error {
	switch job.Operation.Type {
	case pipeline.OperationTypeTerraformPlan:
		return r.terraform.RunPlan(ctx, job, job.Operation.Terraform)
	case pipeline.OperationTypeTerraformApply:
		return r.terraform.RunApply(ctx, job, job.Operation.Terraform)
	case pipeline.OperationTypeCommands:
		for _, command := range job.Operation.Commands {
			if err := r.commands.Run(ctx, commandSpec{
				JobName:      job.Name,
				Command:      command,
				Env:          job.Env,
				AllowFailure: job.AllowFailure,
			}); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported operation type %q", job.Operation.Type)
	}
}

type phaseRunner struct {
	commands commandRunner
	main     operationRunner
}

func (r phaseRunner) Run(ctx context.Context, job *pipeline.Job) error {
	prePhase, postPhase := phasesForJob(job)
	for _, step := range job.Steps {
		if step.Phase != prePhase {
			continue
		}
		if err := r.commands.Run(ctx, commandSpec{
			JobName:    job.Name,
			ModulePath: modulePath(job),
			Command:    step.Command,
			Env:        job.Env,
		}); err != nil {
			return err
		}
	}
	if err := r.main.Run(ctx, job); err != nil {
		return err
	}
	for _, step := range job.Steps {
		if step.Phase != postPhase {
			continue
		}
		if err := r.commands.Run(ctx, commandSpec{
			JobName:    job.Name,
			ModulePath: modulePath(job),
			Command:    step.Command,
			Env:        job.Env,
		}); err != nil {
			return err
		}
	}
	return nil
}

type jobRunner struct {
	phaseRunner phaseRunner
	commands    commandRunner
}

func (r *jobRunner) Run(ctx context.Context, job *pipeline.Job) error {
	if job == nil {
		return errors.New("job is nil")
	}
	if job.Module == nil {
		return r.runStandaloneJob(ctx, job)
	}
	return r.phaseRunner.Run(ctx, job)
}

func (r *jobRunner) runStandaloneJob(ctx context.Context, job *pipeline.Job) error {
	for _, command := range job.Operation.Commands {
		if err := r.commands.Run(ctx, commandSpec{
			JobName:      job.Name,
			Command:      command,
			Env:          job.Env,
			AllowFailure: job.AllowFailure,
		}); err != nil {
			return err
		}
	}
	return nil
}

func phasesForJob(job *pipeline.Job) (pre, post pipeline.Phase) {
	if job != nil && job.Type == pipeline.JobTypeApply {
		return pipeline.PhasePreApply, pipeline.PhasePostApply
	}
	return pipeline.PhasePrePlan, pipeline.PhasePostPlan
}

func modulePath(job *pipeline.Job) string {
	if job == nil || job.Module == nil {
		return ""
	}
	return job.Module.RelativePath
}

func mergeEnv(values ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, env := range values {
		maps.Copy(result, env)
	}
	return result
}

func environMap() map[string]string {
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		if k, v, ok := strings.Cut(entry, "="); ok {
			result[k] = v
		}
	}
	return result
}

func rewriteTerraciCommand(command, selfPath string) string {
	if selfPath == "" {
		return command
	}
	if command == "terraci" {
		return selfPath
	}
	const prefix = "terraci "
	if len(command) > len(prefix) && command[:len(prefix)] == prefix {
		return selfPath + command[len("terraci"):]
	}
	return command
}
