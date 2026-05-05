package config

import (
	"errors"
	"fmt"
)

// Validate runs the GitHub plugin's config-shape sanity checks. Called from
// the plugin's Preflight after DecodeAndSet — fails fast with a descriptive
// error rather than emitting YAML-shaped garbage in the generated workflow.
//
// Catches obvious shape mistakes (missing runs_on without a container,
// blank JobOverwrite.Type) and conflicting flags. Skips semantic checks that
// need network or filesystem access.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if c.RunsOn == "" && c.Container == nil {
		errs = append(errs, errors.New("either runs_on or container must be set"))
	}
	for i := range c.Overwrites {
		if err := c.Overwrites[i].Type.validate(); err != nil {
			errs = append(errs, fmt.Errorf("overwrites[%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func (t JobOverwriteType) validate() error {
	if t == "" {
		return errors.New("type must be set (plan, apply, or a contributed job name)")
	}
	return nil
}
