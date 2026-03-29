package costengine

import (
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/provider"
)

const defaultCacheTTL = 24 * time.Hour

// newDefaultRegistry creates a provider registry with all AWS handlers.
// When GCP/Azure support is added, their handlers register here too.
func newDefaultRegistry() *provider.Registry {
	return aws.NewRegistry()
}

// NewEstimatorFromConfig creates an Estimator using CostConfig settings.
func NewEstimatorFromConfig(cfg *CostConfig) *Estimator {
	cacheDir := ""
	cacheTTL := defaultCacheTTL

	if cfg != nil {
		if cfg.CacheDir != "" {
			cacheDir = cfg.CacheDir
		}
		if cfg.CacheTTL != "" {
			if d, err := time.ParseDuration(cfg.CacheTTL); err == nil {
				cacheTTL = d
			}
		}
	}

	return NewEstimator(cacheDir, cacheTTL, aws.NewFetcher())
}
