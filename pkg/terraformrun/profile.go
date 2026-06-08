// Package terraformrun owns Terraform/OpenTofu runtime intent shared by
// pipeline planning, CI providers, and local execution.
package terraformrun

import (
	"fmt"
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/config"
)

const (
	DefaultParallelism = 4
)

// Binary identifies the Terraform-compatible executable TerraCi should run.
type Binary string

const (
	BinaryTerraform Binary = config.ExecutionBinaryTerraform
	BinaryTofu      Binary = config.ExecutionBinaryTofu
)

func (b Binary) String() string {
	if b == "" {
		return string(BinaryTerraform)
	}
	return string(b)
}

// ParseBinary normalizes and validates a Terraform-compatible binary name.
func ParseBinary(raw string) (Binary, error) {
	switch normalized := Binary(strings.TrimSpace(raw)); normalized {
	case "":
		return BinaryTerraform, nil
	case BinaryTerraform, BinaryTofu:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported terraform binary %q", raw)
	}
}

// ProfileOptions configures a Terraform/OpenTofu runtime profile.
type ProfileOptions struct {
	Binary      string
	InitEnabled *bool
	Parallelism int
	Env         map[string]string
}

// Profile is the immutable Terraform/OpenTofu runtime intent.
type Profile struct {
	binary      Binary
	initEnabled bool
	parallelism int
	env         map[string]string
}

// NewProfile creates a normalized Terraform/OpenTofu runtime profile.
func NewProfile(opts ProfileOptions) (Profile, error) {
	binary, err := ParseBinary(opts.Binary)
	if err != nil {
		return Profile{}, err
	}

	initEnabled := true
	if opts.InitEnabled != nil {
		initEnabled = *opts.InitEnabled
	}

	parallelism := opts.Parallelism
	if parallelism == 0 {
		parallelism = DefaultParallelism
	}
	if parallelism < 0 {
		return Profile{}, fmt.Errorf("parallelism must be >= 1 or omitted, got %d", opts.Parallelism)
	}

	return Profile{
		binary:      binary,
		initEnabled: initEnabled,
		parallelism: parallelism,
		env:         maps.Clone(opts.Env),
	}, nil
}

// ProfileFromConfig normalizes Terraform/OpenTofu runtime settings from config.
func ProfileFromConfig(cfg config.Config) (Profile, error) {
	if !cfg.Present() {
		return NewProfile(ProfileOptions{})
	}
	execution := cfg.Execution()
	return NewProfile(ProfileOptions{
		Binary:      execution.Binary(),
		InitEnabled: boolPtr(execution.InitEnabled()),
		Parallelism: execution.Parallelism(),
		Env:         execution.Env(),
	})
}

func boolPtr(v bool) *bool { return &v }

func (p Profile) Binary() Binary { return p.binary }

func (p Profile) InitEnabled() bool { return p.initEnabled }

func (p Profile) Parallelism() int { return p.parallelism }

func (p Profile) Env() map[string]string { return maps.Clone(p.env) }

// WithParallelism returns a copy with local-execution parallelism overridden.
func (p Profile) WithParallelism(parallelism int) (Profile, error) {
	if parallelism <= 0 {
		return p, nil
	}
	return NewProfile(ProfileOptions{
		Binary:      p.binary.String(),
		InitEnabled: &p.initEnabled,
		Parallelism: parallelism,
		Env:         p.env,
	})
}
