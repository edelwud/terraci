package policy

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
	"github.com/edelwud/terraci/plugins/policy/internal/source"
)

type policyRuntime struct {
	config       policyengine.Config
	sources      *source.Materializer
	workDir      string
	serviceDir   string
	planSegments []string
}

func newRuntime(appCtx *plugin.AppContext, cfg *policyengine.Config) (*policyRuntime, error) {
	if cfg == nil {
		return nil, errors.New("policy checks are not configured")
	}
	if appCtx == nil {
		return nil, errors.New("policy runtime requires app context")
	}

	runtimeConfig := cfg.Normalized()
	if err := runtimeConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy configuration: %w", err)
	}

	materializer, err := source.NewMaterializer(&runtimeConfig, appCtx.WorkDir(), appCtx.ServiceDir())
	if err != nil {
		return nil, fmt.Errorf("create policy source materializer: %w", err)
	}

	structure := appCtx.Config().Structure()
	segments := structure.Segments()

	return &policyRuntime{
		config:       runtimeConfig,
		sources:      materializer,
		workDir:      appCtx.WorkDir(),
		serviceDir:   appCtx.ServiceDir(),
		planSegments: segments,
	}, nil
}

// runtime returns the typed plugin runtime used by policy use-cases.
func (p *Plugin) runtime(_ context.Context, appCtx *plugin.AppContext) (*policyRuntime, error) {
	return newRuntime(appCtx, p.Config())
}
