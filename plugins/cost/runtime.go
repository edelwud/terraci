package cost

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

type costRuntime struct {
	estimator *engine.Estimator
}

func newRuntime(cfg *model.CostConfig) (*costRuntime, error) {
	if err := validateRuntimeConfig(cfg); err != nil {
		return nil, err
	}

	estimator, err := engine.NewEstimatorFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create cost estimator: %w", err)
	}

	return &costRuntime{estimator: estimator}, nil
}

func newRuntimeWithEstimator(estimator *engine.Estimator) *costRuntime {
	return &costRuntime{estimator: estimator}
}

// Runtime implements plugin.RuntimeProvider and serves as the reference lazy
// runtime pattern for runtime-heavy plugins in TerraCi.
func (p *Plugin) Runtime(_ context.Context, _ *plugin.AppContext) (any, error) {
	return newRuntime(p.Config())
}

func (p *Plugin) runtime(ctx context.Context, appCtx *plugin.AppContext) (*costRuntime, error) {
	return plugin.BuildRuntime[*costRuntime](ctx, p, appCtx)
}

func validateRuntimeConfig(cfg *model.CostConfig) error {
	if cfg == nil {
		return errors.New("cost estimation is not configured")
	}

	if cfg.LegacyEnabled != nil {
		log.Warn("cost: plugins.cost.enabled is deprecated; use plugins.cost.providers.<provider>.enabled")
	}

	if cfg.CacheTTL != "" {
		if _, err := time.ParseDuration(cfg.CacheTTL); err != nil {
			return fmt.Errorf("invalid cost configuration: %w", err)
		}
	}

	return nil
}
