// Package errors provides structured error types for TerraCi.
package errors

import "fmt"

// ConfigError represents a configuration loading or validation error.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("config error (%s): %s", e.Path, e.Err)
	}
	return fmt.Sprintf("config error: %s", e.Err)
}

func (e *ConfigError) Unwrap() error { return e.Err }

// ScanError represents an error during module discovery.
type ScanError struct {
	Dir string
	Err error
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("scan %s: %s", e.Dir, e.Err)
}

func (e *ScanError) Unwrap() error { return e.Err }

// ParseError represents an error during HCL parsing.
type ParseError struct {
	Module string
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse %s: %s", e.Module, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// PolicyError represents a policy check failure.
type PolicyError struct {
	Module     string
	Violations []string
}

func (e *PolicyError) Error() string {
	return fmt.Sprintf("policy check failed for %s: %d violation(s)", e.Module, len(e.Violations))
}

// CostError represents a cost estimation error.
type CostError struct {
	Module string
	Err    error
}

func (e *CostError) Error() string {
	return fmt.Sprintf("cost estimation %s: %s", e.Module, e.Err)
}

func (e *CostError) Unwrap() error { return e.Err }

// GraphError represents a dependency graph error (e.g., cycles).
type GraphError struct {
	Cycles [][]string
}

func (e *GraphError) Error() string {
	return fmt.Sprintf("dependency graph has %d cycle(s)", len(e.Cycles))
}

// NoModulesError indicates no modules were found.
type NoModulesError struct {
	Dir string
}

func (e *NoModulesError) Error() string {
	return fmt.Sprintf("no modules found in %s", e.Dir)
}
