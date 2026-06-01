package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/terraformrun"
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

			got := newSettings(&configpkg.Config{}, mustProfile(terraformrun.ProfileOptions{Binary: tt.binary})).defaultImage()
			if got.Name != tt.want {
				t.Fatalf("defaultImage().Name = %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestSettingsConfiguredImageOverridesDerivedDefault(t *testing.T) {
	t.Parallel()

	image := configpkg.Image{Name: "registry.example.com/terraform:custom"}
	got := newSettings(&configpkg.Config{Image: &image}, mustProfile(terraformrun.ProfileOptions{Binary: "tofu"})).defaultImage()
	if got.Name != image.Name {
		t.Fatalf("defaultImage().Name = %q, want configured image", got.Name)
	}
}
