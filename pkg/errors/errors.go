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

// NoModulesError indicates no modules were found.
type NoModulesError struct {
	Dir string
}

func (e *NoModulesError) Error() string {
	return "no modules found in " + e.Dir
}
