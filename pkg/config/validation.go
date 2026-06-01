package config

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/workspacepath"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Structure.Pattern == "" {
		return errors.New("structure.pattern is required")
	}

	if _, err := ParsePattern(c.Structure.Pattern); err != nil {
		return fmt.Errorf("structure.pattern: %w", err)
	}

	if err := workspacepath.ValidateOptional(c.ServiceDir); err != nil {
		return fmt.Errorf("service_dir: %w", err)
	}

	switch c.Execution.Binary {
	case "", ExecutionBinaryTerraform, ExecutionBinaryTofu:
	default:
		return fmt.Errorf("execution.binary: unsupported value %q", c.Execution.Binary)
	}

	// parallelism == 0 used to silently make the executor stall instead of using
	// a worker. Treat <1 as an explicit user error; callers that want the
	// default should leave the field unset and rely on DefaultConfig().
	if c.Execution.Parallelism < 1 {
		return errors.New("execution.parallelism: must be >= 1 (omit to use default)")
	}

	return nil
}
