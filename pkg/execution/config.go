package execution

import (
	"maps"

	"github.com/edelwud/terraci/pkg/config"
)

// PlanMode controls which plan artifacts TerraCi produces.
// It does not change dependency ordering or the high-level execution flow.
type PlanMode string

const (
	PlanModeStandard PlanMode = "standard"
	PlanModeDetailed PlanMode = "detailed"
)

// Config is the normalized execution configuration shared by CI generation and local execution.
type Config struct {
	Binary      string
	InitEnabled bool
	PlanEnabled bool
	PlanMode    PlanMode
	Parallelism int
	Env         map[string]string
}

// ConfigFromProject normalizes execution config from project config.
func ConfigFromProject(cfg *config.Config) Config {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	result := Config{
		Binary:      cfg.Execution.Binary,
		InitEnabled: cfg.Execution.InitEnabled,
		PlanEnabled: cfg.Execution.PlanEnabled,
		PlanMode:    PlanMode(cfg.Execution.PlanMode),
		Parallelism: cfg.Execution.Parallelism,
	}
	if result.Binary == "" {
		result.Binary = "terraform"
	}
	if result.PlanMode == "" {
		result.PlanMode = PlanModeStandard
	}
	if result.Parallelism <= 0 {
		result.Parallelism = 4
	}
	if len(cfg.Execution.Env) != 0 {
		result.Env = make(map[string]string, len(cfg.Execution.Env))
		maps.Copy(result.Env, cfg.Execution.Env)
	}

	return result
}
