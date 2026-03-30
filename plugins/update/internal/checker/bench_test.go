package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

var (
	benchVersionAnalysis versionAnalysis
	benchCheckResult     *updateengine.UpdateResult
)

type benchRegistry struct {
	moduleVersions   map[string][]string
	providerVersions map[string][]string
}

func (r *benchRegistry) ModuleVersions(_ context.Context, ns, name, provider string) ([]string, error) {
	return r.moduleVersions[ns+"/"+name+"/"+provider], nil
}

func (r *benchRegistry) ProviderVersions(_ context.Context, ns, typeName string) ([]string, error) {
	return r.providerVersions[ns+"/"+typeName], nil
}

func BenchmarkAnalyzeModuleVersions(b *testing.B) {
	for _, size := range []int{20, 100, 500} {
		versions := benchVersionList(size)

		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchVersionAnalysis = analyzeModuleVersions("~> 5.0", versions, updateengine.BumpMinor)
			}
		})
	}
}

func BenchmarkAnalyzeProviderVersions(b *testing.B) {
	for _, tc := range []struct {
		name           string
		size           int
		currentVersion string
	}{
		{name: "constraint_only", size: 100},
		{name: "locked_current", size: 100, currentVersion: "5.67.0"},
		{name: "large_locked_current", size: 500, currentVersion: "5.67.0"},
	} {
		versions := benchVersionList(tc.size)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchVersionAnalysis = analyzeProviderVersions("~> 5.0", tc.currentVersion, versions, updateengine.BumpMinor)
			}
		})
	}
}

func BenchmarkCheckerCheck(b *testing.B) {
	for _, moduleCount := range []int{5, 20, 50} {
		modules := buildCheckerBenchModules(b, moduleCount)
		registry := &benchRegistry{
			moduleVersions: map[string][]string{
				"terraform-aws-modules/vpc/aws": benchRegistryVersions(),
				"terraform-aws-modules/eks/aws": benchRegistryVersions(),
			},
			providerVersions: map[string][]string{
				"hashicorp/aws":    benchRegistryVersions(),
				"hashicorp/random": {"3.0.0", "3.1.0", "3.2.0"},
			},
		}

		checker := NewChecker(
			&updateengine.UpdateConfig{Target: updateengine.TargetAll, Bump: updateengine.BumpMinor},
			parser.NewParser(nil),
			registry,
			false,
		)

		b.Run(fmt.Sprintf("modules=%d", moduleCount), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result, err := checker.Check(context.Background(), modules)
				if err != nil {
					b.Fatalf("Check() error = %v", err)
				}
				benchCheckResult = result
			}
		})
	}
}

func buildCheckerBenchModules(tb testing.TB, moduleCount int) []*discovery.Module {
	tb.Helper()

	root := tb.TempDir()
	modules := make([]*discovery.Module, 0, moduleCount)

	for idx := range moduleCount {
		relativePath := fmt.Sprintf("platform/prod/us-east-1/module-%02d", idx)
		moduleDir := filepath.Join(root, relativePath)
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", moduleDir, err)
		}

		content := fmt.Sprintf(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

module "vpc_%02d" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}

module "eks_%02d" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"
}
`, idx, idx)

		if err := os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte(content), 0o600); err != nil {
			tb.Fatalf("write module fixture: %v", err)
		}

		modules = append(modules, &discovery.Module{
			Path:         moduleDir,
			RelativePath: relativePath,
		})
	}

	return modules
}

func benchVersionList(size int) []updateengine.Version {
	versions := make([]updateengine.Version, 0, size)
	for i := range size {
		version, err := updateengine.ParseVersion(fmt.Sprintf("5.%d.%d", i/10, i%10))
		if err != nil {
			panic(err)
		}
		versions = append(versions, version)
	}
	prerelease, err := updateengine.ParseVersion("6.0.0-beta")
	if err != nil {
		panic(err)
	}
	versions = append(versions, prerelease)

	latest, err := updateengine.ParseVersion("6.0.0")
	if err != nil {
		panic(err)
	}
	versions = append(versions, latest)
	return versions
}

func benchRegistryVersions() []string {
	return []string{
		"5.0.0",
		"5.1.0",
		"5.2.0",
		"5.10.0",
		"5.20.0",
		"5.67.0",
		"5.68.0",
		"5.69.0",
		"6.0.0-beta",
		"6.0.0",
	}
}
