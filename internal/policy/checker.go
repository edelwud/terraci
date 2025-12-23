package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
)

// Checker runs policy checks against Terraform plans
type Checker struct {
	config     *config.PolicyConfig
	policyDirs []string
	rootDir    string
}

// NewChecker creates a new policy checker
func NewChecker(cfg *config.PolicyConfig, policyDirs []string, rootDir string) *Checker {
	return &Checker{
		config:     cfg,
		policyDirs: policyDirs,
		rootDir:    rootDir,
	}
}

// CheckModule runs policy checks for a single module
func (c *Checker) CheckModule(ctx context.Context, modulePath string) (*Result, error) {
	// Get effective config for this module (with overwrites applied)
	effectiveCfg := c.config.GetEffectiveConfig(modulePath)

	// Skip if disabled for this module
	if effectiveCfg == nil || !effectiveCfg.Enabled {
		return &Result{
			Module:  modulePath,
			Skipped: 1,
		}, nil
	}

	// Find plan.json in module directory
	planJSONPath := filepath.Join(c.rootDir, modulePath, "plan.json")
	if _, err := os.Stat(planJSONPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plan.json not found in %s", modulePath)
	}

	// Get namespaces to evaluate
	namespaces := effectiveCfg.Namespaces
	if len(namespaces) == 0 {
		// Default namespace
		namespaces = []string{"terraform"}
	}

	// Create and run engine
	engine := NewEngine(c.policyDirs, namespaces)
	result, err := engine.Evaluate(ctx, planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	result.Module = modulePath
	return result, nil
}

// CheckAll runs policy checks for all modules with plan.json files
func (c *Checker) CheckAll(ctx context.Context) (*Summary, error) {
	var results []Result

	// Find all plan.json files
	err := filepath.Walk(c.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "plan.json" && !info.IsDir() {
			// Get module path relative to root
			modulePath, err := filepath.Rel(c.rootDir, filepath.Dir(path))
			if err != nil {
				return err
			}

			result, err := c.CheckModule(ctx, modulePath)
			if err != nil {
				// Log error but continue with other modules
				results = append(results, Result{
					Module: modulePath,
					Failures: []Violation{{
						Message:   fmt.Sprintf("check failed: %v", err),
						Namespace: "terraci",
					}},
				})
				return nil
			}

			results = append(results, *result)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return NewSummary(results), nil
}

// ShouldBlock returns true if the results should block the pipeline
func (c *Checker) ShouldBlock(summary *Summary) bool {
	if c.config.OnFailure == config.PolicyActionBlock {
		return summary.HasFailures()
	}
	return false
}
