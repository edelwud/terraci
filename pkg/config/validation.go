package config

import (
	"errors"
	"fmt"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Structure.Pattern == "" {
		return errors.New("structure.pattern is required")
	}

	if _, err := ParsePattern(c.Structure.Pattern); err != nil {
		return fmt.Errorf("structure.pattern: %w", err)
	}

	switch c.Execution.Binary {
	case "", ExecutionBinaryTerraform, ExecutionBinaryTofu:
	default:
		return fmt.Errorf("execution.binary: unsupported value %q", c.Execution.Binary)
	}

	switch c.Execution.PlanMode {
	case "", "standard", "detailed":
	default:
		return fmt.Errorf("execution.plan_mode: unsupported value %q", c.Execution.PlanMode)
	}

	if c.Execution.Parallelism < 0 {
		return errors.New("execution.parallelism: must be >= 0")
	}

	return nil
}
