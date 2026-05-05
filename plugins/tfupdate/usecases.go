package tfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	tfupdateusecase "github.com/edelwud/terraci/plugins/tfupdate/internal/usecase"
)

func parseRuntimeOptions(cmd *cobra.Command) runtimeOptions {
	opts := runtimeOptions{}
	if flag := cmd.Flags().Lookup("write"); flag != nil {
		opts.write = flag.Value.String() == "true"
	}
	if flag := cmd.Flags().Lookup("module"); flag != nil {
		opts.modulePath = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("output"); flag != nil {
		opts.outputFmt = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("target"); flag != nil {
		opts.target = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("bump"); flag != nil {
		opts.bump = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("pin"); flag != nil {
		opts.pin = flag.Value.String() == "true"
	}
	if flag := cmd.Flags().Lookup("timeout"); flag != nil {
		opts.timeout = flag.Value.String()
	}
	if vals, err := cmd.Flags().GetStringSlice("lock-platforms"); err == nil && len(vals) > 0 {
		opts.lockPlatforms = vals
	}
	return opts
}

func runUpdateCheck(ctx context.Context, appCtx *plugin.AppContext, runtime *updateRuntime, w io.Writer) error {
	modules, err := discoverUpdateModules(ctx, appCtx, runtime.options.modulePath)
	if err != nil {
		return err
	}

	log.WithField("count", len(modules)).Info("modules to check")

	result, err := executeUpdateCheck(ctx, appCtx, runtime, modules)
	if err != nil {
		return err
	}

	emitUpdateArtifacts(appCtx.ServiceDir(), result)
	return finalizeUpdateCheck(w, runtime.options.outputFmt, result)
}

func filterModules(modules []*discovery.Module, modulePath string) []*discovery.Module {
	if modulePath == "" {
		return modules
	}

	filtered := modules[:0]
	for _, module := range modules {
		if module.RelativePath == modulePath || strings.HasSuffix(module.RelativePath, modulePath) {
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

	wfResult, err := workflow.Run(ctx, workflow.Options{
		WorkDir:  appCtx.WorkDir(),
		Segments: baseCfg.Structure.Segments,
		Excludes: baseCfg.Exclude,
		Includes: baseCfg.Include,
	})
	if err != nil {
		return nil, fmt.Errorf("discover modules: %w", err)
	}

	modules := filterModules(wfResult.Filtered.Modules, modulePath)
	if len(modules) == 0 {
		return nil, errors.New("no modules found")
	}

	return modules, nil
}

func executeUpdateCheck(
	ctx context.Context,
	appCtx *plugin.AppContext,
	runtime *updateRuntime,
	modules []*discovery.Module,
) (*tfupdateengine.UpdateResult, error) {
	tfParser := parser.NewParser(appCtx.Config().Structure.Segments)
	service := tfupdateusecase.New(runtime.config, tfParser, runtime.registry, runtime.downloader, runtime.options.write)

	result, err := service.Run(ctx, modules)
	if err != nil {
		return nil, fmt.Errorf("check versions: %w", err)
	}

	return result, nil
}

func emitUpdateArtifacts(serviceDir string, result *tfupdateengine.UpdateResult) {
	if serviceDir == "" {
		return
	}

	report, buildErr := buildUpdateReport(result)
	if buildErr != nil {
		log.WithError(buildErr).Warn("failed to build tfupdate report")
		report = nil
	}
	if saveErr := ci.SaveResultsAndReport(serviceDir, resultsFile, result, report); saveErr != nil {
		log.WithError(saveErr).Warn("failed to persist tfupdate artifacts")
	}
}

func finalizeUpdateCheck(w io.Writer, outputFmt string, result *tfupdateengine.UpdateResult) error {
	if outputErr := outputResult(w, outputFmt, result); outputErr != nil {
		return outputErr
	}
	if result.Summary.Errors > 0 {
		return fmt.Errorf("update check completed with %d errors", result.Summary.Errors)
	}
	return nil
}

func (p *Plugin) runCheck(ctx context.Context, appCtx *plugin.AppContext, cmd *cobra.Command) error {
	opts := parseRuntimeOptions(cmd)
	runtime, err := p.runtime(ctx, appCtx, &opts)
	if err != nil {
		return err
	}
	return runUpdateCheck(ctx, appCtx, runtime, os.Stdout)
}
