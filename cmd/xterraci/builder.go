package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Builder configures and executes a custom TerraCi build.
type Builder struct {
	TerraciVersion string
	WithPlugins    []string // "module[@version][=replacement]"
	WithoutPlugins []string // built-in plugin names to exclude
	Output         string
	SkipCleanup    bool
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

// Build executes the build.
func (b *Builder) Build() error {
	// Resolve output to absolute path (we'll cd to temp dir)
	output, err := filepath.Abs(b.Output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("xterraci-%s-", time.Now().Format("20060102-1504")))
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	if !b.SkipCleanup {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Printf("build directory: %s\n", tmpDir)
	}

	// 1. Determine plugins
	builtinImports, externalSpecs := b.resolvePlugins()

	var externalImports []string
	for _, spec := range externalSpecs {
		externalImports = append(externalImports, spec.Module)
	}

	// 2. Generate main.go
	mainGo := GenerateMainGo(builtinImports, externalImports)
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGo), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// 3. Init go module
	fmt.Println("initializing build module...")
	if err := b.runCmd(tmpDir, "go", "mod", "init", "xterraci_build"); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// 4. Add terraci dependency
	terraciModule := "github.com/edelwud/terraci"
	if b.TerraciVersion != "" {
		terraciModule += "@" + b.TerraciVersion
	}
	fmt.Printf("adding terraci %s...\n", terraciModule)
	if err := b.runCmd(tmpDir, "go", "get", terraciModule); err != nil {
		return fmt.Errorf("go get terraci: %w", err)
	}

	// 5. Add external plugins
	for _, spec := range externalSpecs {
		mod := spec.Module
		if spec.Version != "" {
			mod += "@" + spec.Version
		}
		fmt.Printf("adding plugin %s...\n", mod)
		if err := b.runCmd(tmpDir, "go", "get", mod); err != nil {
			return fmt.Errorf("go get %s: %w", spec.Module, err)
		}

		// Add replace directive if needed
		if spec.Replacement != "" {
			absReplacement, absErr := filepath.Abs(spec.Replacement)
			if absErr != nil {
				absReplacement = spec.Replacement
			}
			editArgs := []string{"mod", "edit", "-replace", spec.Module + "=" + absReplacement}
			if err := b.runCmd(tmpDir, "go", editArgs...); err != nil {
				return fmt.Errorf("go mod edit replace %s: %w", spec.Module, err)
			}
		}
	}

	// 6. Tidy
	fmt.Println("resolving dependencies...")
	if err := b.runCmd(tmpDir, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// 7. Build
	fmt.Printf("building %s...\n", output)
	buildArgs := []string{"build", "-o", output}

	// Add ldflags for version info
	ldflags := fmt.Sprintf("-X main.version=%s -X main.commit=xterraci -X main.date=%s",
		b.effectiveVersion(), time.Now().UTC().Format(time.RFC3339))
	buildArgs = append(buildArgs, "-ldflags", ldflags, ".")

	if err := b.runCmd(tmpDir, "go", buildArgs...); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	fmt.Printf("successfully built: %s\n", output)
	return nil
}

func (b *Builder) resolvePlugins() (builtinImports []string, externalSpecs []pluginSpec) {
	// Build set of excluded plugins
	excluded := make(map[string]bool)
	for _, name := range b.WithoutPlugins {
		excluded[name] = true
	}

	// Include built-in plugins (unless excluded)
	for name, importPath := range BuiltinPlugins {
		if !excluded[name] {
			builtinImports = append(builtinImports, importPath)
		}
	}

	// Parse external plugins
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

func (b *Builder) runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	return cmd.Run()
}
