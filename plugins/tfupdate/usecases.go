package tfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/cliout"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/pkg/workspacepath"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	tfupdateusecase "github.com/edelwud/terraci/plugins/tfupdate/internal/usecase"
)

type CheckRequest struct {
	Write         bool
	ModulePath    string
	OutputFormat  cliout.Format
	Target        string
	Bump          string
	Pin           bool
	Timeout       string
	LockPlatforms []string
}

type checkResult struct {
	Result *tfupdateengine.UpdateResult
}

func runUpdateCheck(ctx context.Context, appCtx *plugin.AppContext, runtime *updateRuntime, req CheckRequest) (*checkResult, error) {
	modules, err := discoverUpdateModules(ctx, appCtx, req.ModulePath)
	if err != nil {
		return nil, err
	}

	log.WithField("count", len(modules)).Info("modules to check")

	config, err := effectiveUpdateConfig(runtime.config, req)
	if err != nil {
		return nil, err
	}

	result, err := executeUpdateCheck(ctx, appCtx, runtime, config, req.Write, modules)
	if err != nil {
		return nil, err
	}

	return &checkResult{Result: result}, nil
}

func filterModules(modules []*discovery.Module, modulePath string) []*discovery.Module {
	if modulePath == "" {
		return modules
	}
	modulePath = workspacepath.Join(modulePath)

	filtered := modules[:0]
	for _, module := range modules {
		moduleID := module.ID()
		if moduleID == modulePath || strings.HasSuffix(moduleID, modulePath) {
			filtered = append(filtered, module)
		}
	}
	return filtered
}

func discoverUpdateModules(
	ctx context.Context,
	appCtx *plugin.AppContext,
	modulePath string,
) ([]*discovery.Module, error) {
	baseCfg := appCtx.Config()

	project, err := workflow.PlanProject(ctx, workflow.ProjectRequest{
		WorkDir: appCtx.WorkDir(),
		Config:  baseCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("discover modules: %w", err)
	}

	modules := filterModules(project.Workflow.Filtered.Modules, modulePath)
	if len(modules) == 0 {
		return nil, errors.New("no modules found")
	}

	return modules, nil
}

func executeUpdateCheck(
	ctx context.Context,
	appCtx *plugin.AppContext,
	runtime *updateRuntime,
	config *tfupdateengine.UpdateConfig,
	write bool,
	modules []*discovery.Module,
) (*tfupdateengine.UpdateResult, error) {
	structure := appCtx.Config().Structure()
	tfParser := parser.NewParser(structure.Segments)
	service := tfupdateusecase.New(config, tfParser, runtime.registry, runtime.downloader, write)

	result, err := service.Run(ctx, modules)
	if err != nil {
		return nil, fmt.Errorf("check versions: %w", err)
	}

	return result, nil
}

func effectiveUpdateConfig(base *tfupdateengine.UpdateConfig, req CheckRequest) (*tfupdateengine.UpdateConfig, error) {
	if base == nil {
		return nil, errors.New("update configuration is not set")
	}

	config := base.Clone()
	if req.Target != "" {
		config.Target = req.Target
	}
	if req.Bump != "" {
		config.Policy.Bump = req.Bump
	}
	if req.Pin {
		config.Policy.Pin = true
	}
	if req.Timeout != "" {
		config.Timeout = req.Timeout
	}
	if len(req.LockPlatforms) > 0 {
		config.Lock.Platforms = append([]string(nil), req.LockPlatforms...)
	}
	if config.Target == "" {
		config.Target = tfupdateengine.TargetAll
	}
	if err := config.ValidateRuntime(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	return config, nil
}

func emitUpdateArtifacts(ctx context.Context, appCtx *plugin.AppContext, result *tfupdateengine.UpdateResult) {
	if appCtx == nil || appCtx.Reports() == nil {
		return
	}

	publication, err := ci.NewArtifactPublication(ci.ArtifactPublicationOptions{
		Producer: pluginName,
		Writer:   appCtx.Reports(),
		Results:  result,
		BuildReport: func() (*ci.Report, error) {
			run, err := plugin.NewArtifactRun(appCtx, plugin.ArtifactRunOptions{Producer: pluginName})
			if err != nil {
				return nil, fmt.Errorf("artifact run: %w", err)
			}
			return buildUpdateReport(updateReportRequest{Result: result, Run: run})
		},
	})
	if err != nil {
		log.WithError(err).Warn("failed to persist tfupdate artifacts")
		return
	}
	if saveErr := ci.PublishArtifacts(ctx, publication); saveErr != nil {
		log.WithError(saveErr).Warn("failed to persist tfupdate artifacts")
	}
}

func finishUpdateCheck(w io.Writer, outputFmt cliout.Format, result *tfupdateengine.UpdateResult) error {
	if outputErr := outputResult(w, outputFmt, result); outputErr != nil {
		return outputErr
	}
	if result.Summary.Errors > 0 {
		return fmt.Errorf("update check completed with %d errors", result.Summary.Errors)
	}
	return nil
}

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, req CheckRequest, w io.Writer) error {
	runtime, err := p.runtime(ctx, appCtx)
	if err != nil {
		return err
	}
	result, err := runUpdateCheck(ctx, appCtx, runtime, req)
	if err != nil {
		return err
	}
	emitUpdateArtifacts(ctx, appCtx, result.Result)
	return finishUpdateCheck(w, req.OutputFormat, result.Result)
}
