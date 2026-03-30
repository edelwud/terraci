package update

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
	"github.com/edelwud/terraci/plugins/update/internal/registryclient"
)

type runtimeOptions struct {
	write      bool
	modulePath string
	outputFmt  string
	target     string
	bump       string
}

type updateRuntime struct {
	config   *updateengine.UpdateConfig
	registry updateengine.RegistryClient
	options  runtimeOptions
}

func newRuntime(cfg *updateengine.UpdateConfig, registryFactory func() updateengine.RegistryClient, opts runtimeOptions) (*updateRuntime, error) {
	if cfg == nil {
		return nil, errors.New("update configuration is not set")
	}

	runtimeConfig := *cfg
	if opts.target != "" {
		runtimeConfig.Target = opts.target
	}
	if opts.bump != "" {
		runtimeConfig.Bump = opts.bump
	}
	if runtimeConfig.Target == "" {
		runtimeConfig.Target = updateengine.TargetAll
	}
	if runtimeConfig.Bump == "" {
		runtimeConfig.Bump = updateengine.BumpMinor
	}
	if err := runtimeConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	if registryFactory == nil {
		registryFactory = func() updateengine.RegistryClient {
			return registryclient.New()
		}
	}

	return &updateRuntime{
		config:   &runtimeConfig,
		registry: registryFactory(),
		options:  opts,
	}, nil
}

func (p *Plugin) Runtime(_ context.Context, _ *plugin.AppContext) (any, error) {
	return newRuntime(p.Config(), p.registryFactory, runtimeOptions{})
}

func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext, opts runtimeOptions) (*updateRuntime, error) {
	if opts == (runtimeOptions{}) {
		rawRuntime, err := p.Runtime(ctx, appCtx)
		if err != nil {
			return nil, err
		}
		return plugin.RuntimeAs[*updateRuntime](rawRuntime)
	}

	return newRuntime(p.Config(), p.registryFactory, opts)
}
