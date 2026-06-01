package execution

import (
	"maps"

	"github.com/edelwud/terraci/pkg/config"
)

// Config is the normalized execution configuration shared by CI generation and local execution.
type Config struct {
	Binary      string
	InitEnabled bool
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
		Parallelism: execution.Parallelism,
	}
	if result.Binary == "" {
		result.Binary = "terraform"
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
