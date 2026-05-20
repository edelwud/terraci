package execution

import (
	"maps"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
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

// ConfigFromProject normalizes execution config from a project config snapshot.
func ConfigFromProject(cfg config.Snapshot) Config {
	if !cfg.Present() {
		cfg = config.DefaultConfig().Snapshot()
	}
	execution := cfg.Execution()

	result := Config{
		Binary:      execution.Binary,
		InitEnabled: execution.InitEnabled,
		PlanEnabled: execution.PlanEnabled,
		PlanMode:    PlanMode(execution.PlanMode),
		Parallelism: execution.Parallelism,
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
	if len(execution.Env) != 0 {
		result.Env = make(map[string]string, len(execution.Env))
		maps.Copy(result.Env, execution.Env)
	}

	return result
}

// BuildRequirements converts execution-mode choices into pipeline IR
// requirements. Provider plugins should only add provider/config requirements;
// runtime execution requirements are derived here.
func (c Config) BuildRequirements() pipeline.BuildRequirements {
	if c.PlanMode != PlanModeDetailed {
		return pipeline.BuildRequirements{}
	}
	return pipeline.RequirementsForDetailedPlans()
}
