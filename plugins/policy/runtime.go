package policy

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	policyconfig "github.com/edelwud/terraci/plugins/policy/internal/config"
	"github.com/edelwud/terraci/plugins/policy/internal/source"
)

type runtimeOptions struct {
	modulePath string
	outputDir  string
	outputFmt  string
}

type policyRuntime struct {
	config       *policyconfig.Config
	sources      *source.Materializer
	workDir      string
	serviceDir   string
	planSegments []string
	options      runtimeOptions
}

func newRuntime(appCtx *plugin.AppContext, cfg *policyconfig.Config, opts runtimeOptions) (*policyRuntime, error) {
	if cfg == nil {
		return nil, errors.New("policy checks are not configured")
	}
	if appCtx == nil {
		return nil, errors.New("policy runtime requires app context")
	}

	runtimeConfig := *cfg
	if opts.outputDir != "" {
		runtimeConfig.CacheDir = opts.outputDir
	}
	if err := runtimeConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy configuration: %w", err)
	}

	materializer, err := source.NewMaterializer(&runtimeConfig, appCtx.WorkDir(), appCtx.ServiceDir())
	if err != nil {
		return nil, fmt.Errorf("create policy source materializer: %w", err)
	}

	var segments []string
	if baseCfg := appCtx.Config(); baseCfg != nil {
		segments = append([]string(nil), baseCfg.Structure.Segments...)
	}

	return &policyRuntime{
		config:       &runtimeConfig,
		sources:      materializer,
		workDir:      appCtx.WorkDir(),
		serviceDir:   appCtx.ServiceDir(),
		planSegments: segments,
		options:      opts,
	}, nil
}

func (p *Plugin) Runtime(_ context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(appCtx, p.Config(), runtimeOptions{})
}

// runtime returns the typed plugin runtime. Pass opts == nil to reuse the
// RuntimeProvider path; pass opts to build a command-specific runtime.
func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext, opts *runtimeOptions) (*policyRuntime, error) {
	if opts == nil {
		return plugin.BuildRuntime[*policyRuntime](ctx, p, appCtx)
	}
	return newRuntime(appCtx, p.Config(), *opts)
}
