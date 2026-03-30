package tffile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var benchIndexResult *Index
var benchLookupFile string

func BenchmarkBuildIndex(b *testing.B) {
	for _, tc := range []struct {
		name           string
		fileCount      int
		modulesPerTF   int
		providersPerTF int
	}{
		{name: "files=5", fileCount: 5, modulesPerTF: 2, providersPerTF: 2},
		{name: "files=25", fileCount: 25, modulesPerTF: 2, providersPerTF: 2},
		{name: "files=50", fileCount: 50, modulesPerTF: 3, providersPerTF: 3},
	} {
		dir := b.TempDir()
		buildBenchTFModule(b, dir, tc.fileCount, tc.modulesPerTF, tc.providersPerTF)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				index, err := BuildIndex(dir)
				if err != nil {
					b.Fatalf("BuildIndex() error = %v", err)
				}
				benchIndexResult = index
			}
		})
	}
}

func BenchmarkIndexLookups(b *testing.B) {
	dir := b.TempDir()
	buildBenchTFModule(b, dir, 25, 3, 3)

	index, err := BuildIndex(dir)
	if err != nil {
		b.Fatalf("BuildIndex() error = %v", err)
	}

	moduleNames := []string{"module_00_00", "module_10_01", "module_20_02"}
	providerNames := []string{"provider_00_00", "provider_10_01", "provider_20_02"}

	b.Run("module", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, name := range moduleNames {
				benchLookupFile = index.FindModuleBlockFile(name)
			}
		}
	})

	b.Run("provider", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, name := range providerNames {
				benchLookupFile = index.FindProviderBlockFile(name)
			}
		}
	})
}

func buildBenchTFModule(tb testing.TB, dir string, fileCount, modulesPerTF, providersPerTF int) {
	tb.Helper()

	for fileIdx := range fileCount {
		path := filepath.Join(dir, fmt.Sprintf("bench_%02d.tf", fileIdx))
		var content strings.Builder

		for providerIdx := range providersPerTF {
			_, _ = fmt.Fprintf(&content,
				`terraform {
  required_providers {
    provider_%02d_%02d = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

`,
				fileIdx,
				providerIdx,
			)
		}

		for moduleIdx := range modulesPerTF {
			_, _ = fmt.Fprintf(&content,
				`module "module_%02d_%02d" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}

`,
				fileIdx,
				moduleIdx,
			)
		}

		if err := os.WriteFile(path, []byte(content.String()), 0o600); err != nil {
			tb.Fatalf("write %s: %v", path, err)
		}
	}
}
