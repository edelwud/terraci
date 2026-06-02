package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestSettingsDefaultImageDerivedFromExecutionBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		binary string
		want   string
	}{
		{name: "terraform", binary: "terraform", want: defaultTerraformImage},
		{name: "tofu", binary: "tofu", want: defaultTofuImage},
		{name: "empty", binary: "", want: defaultTerraformImage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ir := testRuntimeIR(t, tt.binary)
			got, err := newSettings(&configpkg.Config{}).defaultImage(ir)
			if err != nil {
				t.Fatalf("defaultImage() error = %v", err)
			}
			if got.Name != tt.want {
				t.Fatalf("defaultImage().Name = %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestSettingsConfiguredImageOverridesDerivedDefault(t *testing.T) {
	t.Parallel()

	image := configpkg.Image{Name: "registry.example.com/terraform:custom"}
	ir := testRuntimeIR(t, DefaultTofuBinary)
	got, err := newSettings(&configpkg.Config{Image: &image}).defaultImage(ir)
	if err != nil {
		t.Fatalf("defaultImage() error = %v", err)
	}
	if got.Name != image.Name {
		t.Fatalf("defaultImage().Name = %q, want configured image", got.Name)
	}
}

func testRuntimeIR(tb testing.TB, binary string) *pipeline.IR {
	tb.Helper()
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	depGraph := graph.NewDependencyGraph()
	depGraph.AddNode(module)
	return mustBuildIR(tb, &configpkg.Config{}, pipeline.TerraformJobConfigOptions{
		Binary:      binary,
		InitEnabled: true,
	}, nil, depGraph, []*discovery.Module{module}, nil)
}
