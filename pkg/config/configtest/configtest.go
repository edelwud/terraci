// Package configtest provides test-only helpers for constructing immutable
// TerraCi config values without relying on struct literals.
package configtest

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

// Options describes a compact test config fixture.
type Options struct {
	ServiceDir    string
	ServiceDirSet bool
	Pattern       string
	Binary        string
	InitEnabled   *bool
	Parallelism   int
	Env           map[string]string
	Exclude       []string
	Include       []string
	LibraryPaths  []string
	Extensions    config.ExtensionValueSet
}

// Build returns an immutable config fixture.
func Build(tb testing.TB, opts Options) config.Config {
	tb.Helper()

	var execution *config.ExecutionConfig
	if opts.Binary != "" || opts.InitEnabled != nil || opts.Parallelism != 0 || len(opts.Env) > 0 {
		cfg, err := config.NewExecutionConfig(config.ExecutionConfigOptions{
			Binary:      opts.Binary,
			InitEnabled: opts.InitEnabled,
			Parallelism: opts.Parallelism,
			Env:         opts.Env,
		})
		if err != nil {
			tb.Fatalf("NewExecutionConfig() error = %v", err)
		}
		execution = &cfg
	}

	var libraryModules *config.LibraryModulesConfig
	if len(opts.LibraryPaths) > 0 {
		cfg, err := config.NewLibraryModulesConfig(config.LibraryModulesConfigOptions{Paths: opts.LibraryPaths})
		if err != nil {
			tb.Fatalf("NewLibraryModulesConfig() error = %v", err)
		}
		libraryModules = &cfg
	}

	cfg, err := config.Build(config.BuildOptions{
		ServiceDir:     opts.ServiceDir,
		ServiceDirSet:  opts.ServiceDirSet,
		Pattern:        opts.Pattern,
		Execution:      execution,
		Exclude:        opts.Exclude,
		Include:        opts.Include,
		LibraryModules: libraryModules,
		Extensions:     opts.Extensions,
	})
	if err != nil {
		tb.Fatalf("Build() error = %v", err)
	}
	return cfg
}
