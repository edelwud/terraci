package parser

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
)

func TestExtractDependencies_SingleRemoteState(t *testing.T) {
	tmpDir := t.TempDir()

	eksPath := createTestModuleDir(t, tmpDir, "platform", "stage", "eu-central-1", "eks")
	vpcPath := createTestModuleDir(t, tmpDir, "platform", "stage", "eu-central-1", "vpc")

	writeTestFile(t, eksPath, "data.tf", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "b", key = "platform/stage/eu-central-1/vpc/terraform.tfstate" }
}
`)
	writeTestFile(t, vpcPath, "main.tf", "# VPC")

	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	eks.Path = eksPath
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	vpc.Path = vpcPath

	extractor := NewDependencyExtractor(NewParser(), discovery.NewModuleIndex([]*discovery.Module{eks, vpc}))

	deps, err := extractor.ExtractDependencies(context.Background(), eks)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if len(deps.Dependencies) != 1 {
		t.Fatalf("deps: got %d, want 1", len(deps.Dependencies))
	}
	if deps.Dependencies[0].To.ID() != "platform/stage/eu-central-1/vpc" {
		t.Errorf("dep target = %q, want vpc", deps.Dependencies[0].To.ID())
	}
	if deps.Dependencies[0].Type != "remote_state" {
		t.Errorf("dep type = %q, want remote_state", deps.Dependencies[0].Type)
	}
}

func TestExtractDependencies_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	names := []string{"app", "vpc", "rds"}
	modules := make([]*discovery.Module, 0, len(names))
	for _, name := range names {
		path := createTestModuleDir(t, tmpDir, "platform", "stage", "eu-central-1", name)
		m := discovery.TestModule("platform", "stage", "eu-central-1", name)
		m.Path = path
		modules = append(modules, m)
		writeTestFile(t, path, "main.tf", "# Module")
	}

	writeTestFile(t, filepath.Join(tmpDir, "platform", "stage", "eu-central-1", "app"), "data.tf", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "b", key = "platform/stage/eu-central-1/vpc/terraform.tfstate" }
}
data "terraform_remote_state" "rds" {
  backend = "s3"
  config = { bucket = "b", key = "platform/stage/eu-central-1/rds/terraform.tfstate" }
}
`)

	extractor := NewDependencyExtractor(NewParser(), discovery.NewModuleIndex(modules))

	deps, err := extractor.ExtractDependencies(context.Background(), modules[0]) // app
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if len(deps.DependsOn) != 2 {
		t.Fatalf("DependsOn: got %d, want 2", len(deps.DependsOn))
	}
}

func TestExtractDependencies_ForEach(t *testing.T) {
	tmpDir := t.TempDir()

	names := []string{"app", "vpc", "rds"}
	modules := make([]*discovery.Module, 0, len(names))
	for _, name := range names {
		path := createTestModuleDir(t, tmpDir, "platform", "stage", "eu-central-1", name)
		m := discovery.TestModule("platform", "stage", "eu-central-1", name)
		m.Path = path
		modules = append(modules, m)
		writeTestFile(t, path, "main.tf", "# Module")
	}

	writeTestFile(t, filepath.Join(tmpDir, "platform", "stage", "eu-central-1", "app"), "main.tf", `
locals { deps = { vpc = "platform/stage/eu-central-1/vpc", rds = "platform/stage/eu-central-1/rds" } }
data "terraform_remote_state" "deps" {
  for_each = local.deps
  backend  = "s3"
  config   = { bucket = "b", key = "${each.value}/terraform.tfstate" }
}
`)

	extractor := NewDependencyExtractor(NewParser(), discovery.NewModuleIndex(modules))

	deps, err := extractor.ExtractDependencies(context.Background(), modules[0]) // app
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if len(deps.Dependencies) != 2 {
		t.Fatalf("deps: got %d, want 2 (errors: %v)", len(deps.Dependencies), deps.Errors)
	}
}

func TestExtractDependencies_Library(t *testing.T) {
	tmpDir := t.TempDir()

	modPath := createTestModuleDir(t, tmpDir, "platform", "stage", "eu-central-1", "kafka")
	libPath := createTestModuleDir(t, tmpDir, "_modules", "kafka")

	writeTestFile(t, modPath, "main.tf", `module "kafka" { source = "../../../../_modules/kafka" }`)
	writeTestFile(t, libPath, "main.tf", "# Kafka lib")

	m := discovery.TestModule("platform", "stage", "eu-central-1", "kafka")
	m.Path = modPath

	extractor := NewDependencyExtractor(NewParser(), discovery.NewModuleIndex([]*discovery.Module{m}))

	deps, err := extractor.ExtractDependencies(context.Background(), m)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if len(deps.LibraryDependencies) != 1 {
		t.Fatalf("lib deps: got %d, want 1", len(deps.LibraryDependencies))
	}
	if deps.LibraryDependencies[0].ModuleCall.Name != "kafka" {
		t.Errorf("lib call name = %q, want kafka", deps.LibraryDependencies[0].ModuleCall.Name)
	}
}

func TestExtractAllDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	names := []string{"vpc", "rds", "app"}
	modules := make([]*discovery.Module, 0, len(names))
	for _, name := range names {
		path := createTestModuleDir(t, tmpDir, "platform", "stage", "eu-central-1", name)
		m := discovery.TestModule("platform", "stage", "eu-central-1", name)
		m.Path = path
		modules = append(modules, m)
	}

	writeTestFile(t, modules[0].Path, "main.tf", "# VPC")
	writeTestFile(t, modules[1].Path, "main.tf", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "b", key = "platform/stage/eu-central-1/vpc/terraform.tfstate" }
}
`)
	writeTestFile(t, modules[2].Path, "main.tf", `
data "terraform_remote_state" "rds" {
  backend = "s3"
  config = { bucket = "b", key = "platform/stage/eu-central-1/rds/terraform.tfstate" }
}
`)

	extractor := NewDependencyExtractor(NewParser(), discovery.NewModuleIndex(modules))
	allDeps, _ := extractor.ExtractAllDependencies(context.Background())

	if len(allDeps) != 3 {
		t.Fatalf("modules: got %d, want 3", len(allDeps))
	}

	if n := len(allDeps["platform/stage/eu-central-1/vpc"].Dependencies); n != 0 {
		t.Errorf("vpc deps = %d, want 0", n)
	}
	if n := len(allDeps["platform/stage/eu-central-1/rds"].Dependencies); n != 1 {
		t.Errorf("rds deps = %d, want 1", n)
	}
	if n := len(allDeps["platform/stage/eu-central-1/app"].Dependencies); n != 1 {
		t.Errorf("app deps = %d, want 1", n)
	}
}

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

	extractor := NewDependencyExtractor(NewParser(), discovery.NewModuleIndex(modules))

	tests := []struct {
		name      string
		statePath string
		from      *discovery.Module
		wantID    string
	}{
		{"full path with tfstate", "platform/stage/eu-central-1/vpc/terraform.tfstate", modules[1], "platform/stage/eu-central-1/vpc"},
		{"full path bare", "platform/stage/eu-central-1/vpc", modules[1], "platform/stage/eu-central-1/vpc"},
		{"dotTfstate suffix", "platform/stage/eu-central-1/eks.tfstate", modules[0], "platform/stage/eu-central-1/eks"},
		{"short context match", "vpc", modules[1], "platform/stage/eu-central-1/vpc"},
		{"submodule path", "platform/stage/eu-central-1/ec2/rabbitmq/terraform.tfstate", modules[0], "platform/stage/eu-central-1/ec2/rabbitmq"},
		{"different context", "platform/prod/us-east-1/api/terraform.tfstate", modules[0], "platform/prod/us-east-1/api"},
		{"nonexistent", "platform/stage/eu-central-1/nonexistent/terraform.tfstate", modules[0], ""},
		{"env prefix", "env:/stage/platform/stage/eu-central-1/vpc/terraform.tfstate", modules[1], "platform/stage/eu-central-1/vpc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.matchPathToModule(tt.statePath, tt.from)
			if tt.wantID == "" {
				if got != nil {
					t.Errorf("got %s, want nil", got.ID())
				}
				return
			}
			if got == nil {
				t.Fatalf("got nil, want %s", tt.wantID)
			}
			if got.ID() != tt.wantID {
				t.Errorf("got %s, want %s", got.ID(), tt.wantID)
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
		{"${lookup(local.envs, var.env)}/vpc/terraform.tfstate", true},
		{`path/"}/something`, true},
		{"simple/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := containsDynamicPattern(tt.path); got != tt.want {
				t.Errorf("containsDynamicPattern(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
