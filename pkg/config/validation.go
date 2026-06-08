package config

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/workspacepath"
)

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.structure.Pattern() == "" {
		return errors.New("structure.pattern is required")
	}

	if _, err := ParsePattern(c.structure.Pattern()); err != nil {
		return fmt.Errorf("structure.pattern: %w", err)
	}

	if err := workspacepath.ValidateOptional(c.serviceDir); err != nil {
		return fmt.Errorf("service_dir: %w", err)
	}

	switch c.execution.Binary() {
	case "", ExecutionBinaryTerraform, ExecutionBinaryTofu:
	default:
		return unsupportedExecutionBinaryError(c.execution.Binary())
	}

	if c.execution.Parallelism() < 1 {
		return invalidParallelismError()
	}

	return nil
}

func unsupportedExecutionBinaryError(binary string) error {
	return fmt.Errorf("execution.binary: unsupported value %q", binary)
}

func invalidParallelismError() error {
	return errors.New("execution.parallelism: must be >= 1 (omit to use default)")
}
