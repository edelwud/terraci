package update

import (
	"errors"
	"fmt"

	updateengine "github.com/edelwud/terraci/plugins/update/internal"
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

func newRuntime(cfg *updateengine.UpdateConfig, registry updateengine.RegistryClient, opts runtimeOptions) (*updateRuntime, error) {
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
	if registry == nil {
		registry = updateengine.NewRegistryClient()
	}

	return &updateRuntime{
		config:   &runtimeConfig,
		registry: registry,
		options:  opts,
	}, nil
}
