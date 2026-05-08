package config

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Structure.Pattern == "" {
		return errors.New("structure.pattern is required")
	}

	if _, err := ParsePattern(c.Structure.Pattern); err != nil {
		return fmt.Errorf("structure.pattern: %w", err)
	}

	if err := validateWorkspaceRelativePath(c.ServiceDir); err != nil {
		return fmt.Errorf("service_dir: %w", err)
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

	// parallelism == 0 used to silently make the executor stall instead of using
	// a worker. Treat <1 as an explicit user error; callers that want the
	// default should leave the field unset and rely on DefaultConfig().
	if c.Execution.Parallelism < 1 {
		return errors.New("execution.parallelism: must be >= 1 (omit to use default)")
	}

	return nil
}

func validateWorkspaceRelativePath(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	if normalized == "" {
		return nil
	}
	if path.IsAbs(normalized) || hasWindowsDrivePrefix(normalized) {
		return errors.New("must be workspace-relative")
	}
	if slices.Contains(strings.Split(normalized, "/"), "..") {
		return errors.New("must not contain parent directory segments")
	}
	return nil
}

func hasWindowsDrivePrefix(value string) bool {
	if len(value) < 2 || value[1] != ':' {
		return false
	}
	return (value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')
}
