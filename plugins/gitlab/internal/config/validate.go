package config

import (
	"errors"
	"fmt"
)

// Validate runs the GitLab plugin's config-shape sanity checks. Called from
// the plugin's Preflight after DecodeAndSet — fails fast with a descriptive
// error rather than emitting YAML-shaped garbage in the generated pipeline.
//
// Catches misconfigured enums (cache.policy, JobOverwrite.Type) and obvious
// missing fields. Avoids semantic checks that need network or filesystem.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if err := c.validateImage(); err != nil {
		errs = append(errs, err)
	}
	if err := c.Cache.validate(); err != nil {
		errs = append(errs, fmt.Errorf("cache: %w", err))
	}
	for i := range c.Overwrites {
		if err := c.Overwrites[i].Type.validate(); err != nil {
			errs = append(errs, fmt.Errorf("overwrites[%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func (c *Config) validateImage() error {
	if c.Image.Name == "" {
		return errors.New("image.name must not be empty")
	}
	return nil
}

func (c *CacheConfig) validate() error {
	if c == nil || c.Policy == "" {
		return nil
	}
	switch c.Policy {
	case "pull", "push", "pull-push":
		return nil
	default:
		return fmt.Errorf("invalid policy %q (want one of pull, push, pull-push)", c.Policy)
	}
}

func (t JobOverwriteType) validate() error {
	if t == "" {
		return errors.New("type must be set (plan, apply, or a contributed job name)")
	}
	// Plan / Apply are the canonical values; other strings are treated as a
	// contributed job name and validated at IR-resolution time.
	return nil
}
