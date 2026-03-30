package deps

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func TestMatchPathToModule(t *testing.T) {
	submod := discovery.TestModule("platform", "stage", "eu-central-1", "ec2")
	submod.SetComponent("submodule", "rabbitmq")
	submod.RelativePath = "platform/stage/eu-central-1/ec2/rabbitmq"

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		submod,
		discovery.TestModule("platform", "prod", "us-east-1", "api"),
	}

	index := discovery.NewModuleIndex(modules)

	tests := []struct {
		name      string
		statePath string
		from      *discovery.Module
		wantID    string
	}{
		{"full path with tfstate", "platform/stage/eu-central-1/vpc/terraform.tfstate", modules[1], "platform/stage/eu-central-1/vpc"},
		{"short context match", "vpc", modules[1], "platform/stage/eu-central-1/vpc"},
		{"submodule path", "platform/stage/eu-central-1/ec2/rabbitmq/terraform.tfstate", modules[0], "platform/stage/eu-central-1/ec2/rabbitmq"},
		{"env prefix", "env:/stage/platform/stage/eu-central-1/vpc/terraform.tfstate", modules[1], "platform/stage/eu-central-1/vpc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchPathToModule(index, tt.statePath, tt.from)
			if got == nil || got.ID() != tt.wantID {
				if got == nil {
					t.Fatalf("got nil, want %s", tt.wantID)
				}
				t.Fatalf("got %s, want %s", got.ID(), tt.wantID)
			}
		})
	}
}

func TestContainsDynamicPattern(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"platform/stage/eu-central-1/vpc/terraform.tfstate", false},
		{"${var.environment}/vpc/terraform.tfstate", true},
		{"${each.key}/terraform.tfstate", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := ContainsDynamicPattern(tt.path); got != tt.want {
				t.Fatalf("ContainsDynamicPattern(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestBackendIndexKeyAndMatchByBackend(t *testing.T) {
	target := discovery.TestModule("team-b", "prod", "eu-central-1", "vpc")
	backendIndex := map[string]*discovery.Module{
		BackendIndexKey("s3", "team-b-state", "prod/eu-central-1/vpc/terraform.tfstate", ""): target,
	}

	got := MatchByBackend(backendIndex, "s3", "team-b-state", "prod/eu-central-1/vpc/terraform.tfstate")
	if got == nil || got.ID() != target.ID() {
		if got == nil {
			t.Fatalf("got nil, want %s", target.ID())
		}
		t.Fatalf("got %s, want %s", got.ID(), target.ID())
	}
}
