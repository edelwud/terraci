package cost

import (
	"time"

	"github.com/edelwud/terraci/pkg/config"
)

const defaultCacheTTL = 24 * time.Hour

// NewEstimatorFromConfig creates an Estimator using CostConfig settings.
func NewEstimatorFromConfig(cfg *config.CostConfig) *Estimator {
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

	return NewEstimator(cacheDir, cacheTTL)
}
