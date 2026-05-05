package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/caarlos0/log"
)

// Builder configures and executes a custom TerraCi build.
type Builder struct {
	TerraciVersion string
	WithPlugins    []string // "module[@version][=replacement]"
	WithoutPlugins []string // built-in plugin names to exclude
	Output         string
	SkipCleanup    bool
	SkipSmoke      bool // skip post-build `<output> version` invocation
}

// pluginSpec represents a parsed --with value.
type pluginSpec struct {
	Module      string
	Version     string
	Replacement string
}

func parsePluginSpec(s string) pluginSpec {
	spec := pluginSpec{}

	// Check for replacement: module=replacement
	if parts := strings.SplitN(s, "=", 2); len(parts) == 2 {
		s = parts[0]
		spec.Replacement = parts[1]
	}

	// Check for version: module@version
	if parts := strings.SplitN(s, "@", 2); len(parts) == 2 {
		spec.Module = parts[0]
		spec.Version = parts[1]
	} else {
		spec.Module = s
	}

	return spec
}

// validate checks that --without and --with flags reference valid plugins/modules.
func (b *Builder) validate() error {
	if err := validateWithout(b.WithoutPlugins); err != nil {
		return err
	}
	return validateWith(b.WithPlugins)
}

// Build executes the build.
func (b *Builder) Build(ctx context.Context) error {
	if err := b.validate(); err != nil {
		return err
	}

	output, err := filepath.Abs(b.Output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("xterraci-%s-", time.Now().Format("20060102-1504")))
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	if !b.SkipCleanup {
		defer os.RemoveAll(tmpDir)
	} else {
		log.WithField("dir", tmpDir).Info("build directory (kept)")
	}

	// 1. Determine plugins
	builtinImports, externalSpecs := b.resolvePlugins()

	var externalImports []string
	for _, spec := range externalSpecs {
		externalImports = append(externalImports, spec.Module)
	}

	log.WithField("builtin", len(builtinImports)).
		WithField("external", len(externalImports)).
		Debug("resolved plugins")

	// 2. Generate main.go
	mainGo := GenerateMainGo(builtinImports, externalImports)
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGo), 0o600); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// 3. Init go module
	log.Info("initializing build module")
	if err := b.runCmd(ctx, tmpDir, "go", "mod", "init", "xterraci_build"); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// 4. Add terraci dependency
	terraciModule := "github.com/edelwud/terraci"
	if b.TerraciVersion != "" {
		terraciModule += "@" + b.TerraciVersion
	}
	log.WithField("module", terraciModule).Info("adding terraci")
	if err := b.runCmd(ctx, tmpDir, "go", "get", terraciModule); err != nil {
		return fmt.Errorf("go get terraci: %w", err)
	}

	// 5. Add external plugins
	for _, spec := range externalSpecs {
		mod := spec.Module
		if spec.Version != "" {
			mod += "@" + spec.Version
		}
		log.WithField("plugin", mod).Info("adding plugin")
		if err := b.runCmd(ctx, tmpDir, "go", "get", mod); err != nil {
			return fmt.Errorf("go get %s: %w", spec.Module, err)
		}

		if spec.Replacement != "" {
			absReplacement, absErr := filepath.Abs(spec.Replacement)
			if absErr != nil {
				absReplacement = spec.Replacement
			}
			log.WithField("module", spec.Module).
				WithField("replacement", absReplacement).
				Debug("adding replace directive")
			editArgs := []string{"mod", "edit", "-replace", spec.Module + "=" + absReplacement}
			if err := b.runCmd(ctx, tmpDir, "go", editArgs...); err != nil {
				return fmt.Errorf("go mod edit replace %s: %w", spec.Module, err)
			}
		}
	}

	// 6. Tidy
	log.Info("resolving dependencies")
	if err := b.runCmd(ctx, tmpDir, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// 7. Build
	log.WithField("output", output).Info("building binary")
	ldflags := fmt.Sprintf("-X main.version=%s -X main.commit=xterraci -X main.date=%s",
		b.effectiveVersion(), time.Now().UTC().Format(time.RFC3339))
	buildArgs := []string{"build", "-o", output, "-ldflags", ldflags, "."}

	if err := b.runCmd(ctx, tmpDir, "go", buildArgs...); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	log.WithField("path", output).Info("build successful")

	if !b.SkipSmoke {
		if err := b.smokeTest(ctx, output); err != nil {
			return fmt.Errorf("smoke test: %w", err)
		}
	}
	return nil
}

// smokeTest runs `<output> version` and confirms the binary starts. Catches
// the silent-failure case where a misconfigured --with module compiled fine
// but its init() didn't call registry.RegisterFactory — the resulting binary
// would just be missing the plugin without any compile-time signal.
func (b *Builder) smokeTest(ctx context.Context, output string) error {
	log.WithField("path", output).Debug("smoke test: running `version`")
	cmd := exec.CommandContext(ctx, output, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("`%s version` failed: %w\noutput:\n%s", output, err, string(out))
	}
	if len(out) == 0 {
		return fmt.Errorf("`%s version` returned empty output — binary may be broken", output)
	}
	log.WithField("path", output).Info("smoke test passed")
	return nil
}

func (b *Builder) resolvePlugins() (builtinImports []string, externalSpecs []pluginSpec) {
	excluded := make(map[string]bool)
	for _, name := range b.WithoutPlugins {
		excluded[name] = true
	}

	for name, importPath := range BuiltinPlugins {
		if !excluded[name] {
			builtinImports = append(builtinImports, importPath)
		} else {
			log.WithField("plugin", name).Debug("excluded")
		}
	}

	for _, withStr := range b.WithPlugins {
		externalSpecs = append(externalSpecs, parsePluginSpec(withStr))
	}

	return
}

func (b *Builder) effectiveVersion() string {
	if b.TerraciVersion != "" {
		return b.TerraciVersion
	}
	return "custom"
}

func (b *Builder) runCmd(ctx context.Context, dir, name string, args ...string) error { //nolint:unparam // name is intentionally a parameter for generality
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	log.WithField("cmd", name+" "+strings.Join(args, " ")).Debug("executing")

	if IsDebug() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}
