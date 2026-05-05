package policy

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

type runtimeOptions struct {
	modulePath string
	outputDir  string
	outputFmt  string
}

type policyRuntime struct {
	config     *policyengine.Config
	puller     *policyengine.Puller
	workDir    string
	serviceDir string
	options    runtimeOptions
}

func newRuntime(appCtx *plugin.AppContext, cfg *policyengine.Config, opts runtimeOptions) (*policyRuntime, error) {
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

	puller, err := policyengine.NewPuller(&runtimeConfig, appCtx.WorkDir(), appCtx.ServiceDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create puller: %w", err)
	}

	return &policyRuntime{
		config:     &runtimeConfig,
		puller:     puller,
		workDir:    appCtx.WorkDir(),
		serviceDir: appCtx.ServiceDir(),
		options:    opts,
	}, nil
}

func (p *Plugin) Runtime(_ context.Context, appCtx *plugin.AppContext) (any, error) {
	return newRuntime(appCtx, p.Config(), runtimeOptions{})
}

// runtime returns the typed plugin runtime. Pass opts == nil to reuse the
// framework-cached runtime created from p.Runtime; pass a non-nil pointer to
// build a fresh runtime with command-specific overrides. Using a pointer
// discriminator avoids the option-by-option zero-check predicate that
// previously needed to be updated whenever a new option was introduced.
func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext, opts *runtimeOptions) (*policyRuntime, error) {
	if opts == nil {
		return plugin.BuildRuntime[*policyRuntime](ctx, p, appCtx)
	}
	return newRuntime(appCtx, p.Config(), *opts)
}
