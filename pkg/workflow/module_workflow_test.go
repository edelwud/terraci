package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	terrierrors "github.com/edelwud/terraci/pkg/errors"
)

var defaultSegments = []string{"service", "environment", "region", "module"}

// createModuleTree creates a directory tree with main.tf files.
func createModuleTree(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, p := range paths {
		dir := filepath.Join(root, p)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# test"), 0o644); err != nil {
			t.Fatalf("write %s/main.tf: %v", dir, err)
		}
	}
}

// createModuleWithContent creates a module with specific .tf content.
func createModuleWithContent(t *testing.T, root, path, content string) {
	t.Helper()
	dir := filepath.Join(root, path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s/main.tf: %v", dir, err)
	}
}

func defaultOptions(dir string) Options {
	return Options{
		WorkDir:  dir,
		Segments: defaultSegments,
	}
}

func TestRun_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/prod/eu-central-1/vpc",
	})

	result, err := Run(context.Background(), defaultOptions(tmpDir))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.AllModules) != 3 {
		t.Errorf("AllModules = %d, want 3", len(result.AllModules))
	}
	if len(result.FilteredModules) != 3 {
		t.Errorf("FilteredModules = %d, want 3", len(result.FilteredModules))
	}
	if result.FullIndex == nil {
		t.Error("FullIndex is nil")
	}
	if result.FilteredIndex == nil {
		t.Error("FilteredIndex is nil")
	}
	if result.Graph == nil {
		t.Error("Graph is nil")
	}
	if result.Dependencies == nil {
		t.Error("Dependencies is nil")
	}
}

func TestRun_NoModules(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Run(context.Background(), defaultOptions(tmpDir))
	if err == nil {
		t.Fatal("expected error for empty directory")
	}

	var noModErr *terrierrors.NoModulesError
	if !errors.As(err, &noModErr) {
		t.Errorf("expected NoModulesError, got %T: %v", err, err)
	}
	if noModErr.Dir != tmpDir {
		t.Errorf("NoModulesError.Dir = %q, want %q", noModErr.Dir, tmpDir)
	}
}

func TestRun_InvalidDir(t *testing.T) {
	_, err := Run(context.Background(), Options{
		WorkDir:  "/nonexistent/path/that/does/not/exist",
		Segments: defaultSegments,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}

	var scanErr *terrierrors.ScanError
	if !errors.As(err, &scanErr) {
		t.Errorf("expected ScanError, got %T: %v", err, err)
	}
}

func TestRun_ExcludeFilter(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/prod/eu-central-1/vpc",
	})

	opts := defaultOptions(tmpDir)
	opts.Excludes = []string{"**/prod/**"}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.AllModules) != 3 {
		t.Errorf("AllModules = %d, want 3 (all discovered)", len(result.AllModules))
	}
	if len(result.FilteredModules) != 2 {
		t.Errorf("FilteredModules = %d, want 2 (prod excluded)", len(result.FilteredModules))
	}
}

func TestRun_IncludeFilter(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/prod/eu-central-1/vpc",
	})

	opts := defaultOptions(tmpDir)
	opts.Includes = []string{"**/prod/**"}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.FilteredModules) != 1 {
		t.Errorf("FilteredModules = %d, want 1 (only prod)", len(result.FilteredModules))
	}
}

func TestRun_SegmentFilter(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/prod/eu-central-1/vpc",
		"payments/stage/eu-central-1/vpc",
	})

	opts := defaultOptions(tmpDir)
	opts.SegmentFilters = map[string][]string{
		"service": {"platform"},
	}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.AllModules) != 3 {
		t.Errorf("AllModules = %d, want 3", len(result.AllModules))
	}
	if len(result.FilteredModules) != 2 {
		t.Errorf("FilteredModules = %d, want 2 (only platform)", len(result.FilteredModules))
	}
}

func TestRun_WithDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	// VPC module (no deps)
	createModuleWithContent(t, tmpDir, "platform/stage/eu-central-1/vpc", `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

output "vpc_id" {
  value = aws_vpc.main.id
}
`)

	// EKS module depends on VPC via remote state
	createModuleWithContent(t, tmpDir, "platform/stage/eu-central-1/eks", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_eks_cluster" "main" {
  name = "cluster"
  vpc_config {
    subnet_ids = data.terraform_remote_state.vpc.outputs.subnet_ids
  }
}
`)

	result, err := Run(context.Background(), defaultOptions(tmpDir))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.FilteredModules) != 2 {
		t.Fatalf("FilteredModules = %d, want 2", len(result.FilteredModules))
	}

	// Graph should have an edge from eks to vpc
	eksID := "platform/stage/eu-central-1/eks"
	deps := result.Graph.GetDependencies(eksID)
	if len(deps) == 0 {
		t.Error("expected eks to have dependencies on vpc")
	}

	vpcID := "platform/stage/eu-central-1/vpc"
	found := slices.Contains(deps, vpcID)
	if !found {
		t.Errorf("expected eks to depend on vpc, got deps: %v", deps)
	}
}

func TestRun_Indexes(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/prod/eu-central-1/vpc",
	})

	opts := defaultOptions(tmpDir)
	opts.Excludes = []string{"**/prod/**"}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// FullIndex should contain all modules
	if m := result.FullIndex.ByID("platform/prod/eu-central-1/vpc"); m == nil {
		t.Error("FullIndex should contain prod/vpc")
	}

	// FilteredIndex should only contain non-excluded modules
	if m := result.FilteredIndex.ByID("platform/prod/eu-central-1/vpc"); m != nil {
		t.Error("FilteredIndex should NOT contain prod/vpc (excluded)")
	}
	if m := result.FilteredIndex.ByID("platform/stage/eu-central-1/vpc"); m == nil {
		t.Error("FilteredIndex should contain stage/vpc")
	}
}

func TestRun_ContextCanceled(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := Run(ctx, defaultOptions(tmpDir))
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestRun_CustomSegments(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"team-a/projectX/compute",
		"team-b/projectY/network",
	})

	opts := Options{
		WorkDir:  tmpDir,
		Segments: []string{"team", "project", "component"},
	}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.AllModules) != 2 {
		t.Errorf("AllModules = %d, want 2", len(result.AllModules))
	}

	// Verify segments are set correctly
	for _, m := range result.AllModules {
		if m.Get("team") == "" {
			t.Errorf("module %s: team segment is empty", m.ID())
		}
		if m.Get("project") == "" {
			t.Errorf("module %s: project segment is empty", m.ID())
		}
		if m.Get("component") == "" {
			t.Errorf("module %s: component segment is empty", m.ID())
		}
	}
}

func TestRun_Submodules(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/ec2",
		"platform/stage/eu-central-1/ec2/rabbitmq",
	})

	opts := Options{
		WorkDir:  tmpDir,
		Segments: defaultSegments,
	}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.AllModules) != 2 {
		t.Errorf("AllModules = %d, want 2", len(result.AllModules))
	}

	subCount := 0
	for _, m := range result.AllModules {
		if m.IsSubmodule() {
			subCount++
		}
	}
	if subCount != 1 {
		t.Errorf("submodules = %d, want 1", subCount)
	}
}

func TestRun_CombinedFilters(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/prod/eu-central-1/vpc",
		"payments/stage/eu-central-1/vpc",
	})

	opts := defaultOptions(tmpDir)
	opts.Excludes = []string{"**/prod/**"}
	opts.SegmentFilters = map[string][]string{
		"service": {"platform"},
	}

	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should exclude prod AND filter to platform only
	if len(result.FilteredModules) != 2 {
		t.Errorf("FilteredModules = %d, want 2 (platform/stage only)", len(result.FilteredModules))
		for _, m := range result.FilteredModules {
			t.Logf("  %s", m.ID())
		}
	}
}

func TestRun_WarningsReturned(t *testing.T) {
	tmpDir := t.TempDir()

	// Module with an unresolvable remote state reference
	createModuleWithContent(t, tmpDir, "platform/stage/eu-central-1/vpc", `
data "terraform_remote_state" "missing" {
  backend = "s3"
  config = {
    bucket = "state"
    key    = "nonexistent/path/terraform.tfstate"
    region = "eu-central-1"
  }
}
`)

	result, err := Run(context.Background(), defaultOptions(tmpDir))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Unresolvable remote state should produce a warning, not an error
	if result == nil {
		t.Fatal("result is nil")
	}
	// The workflow should complete — warnings are non-fatal
	if len(result.FilteredModules) != 1 {
		t.Errorf("FilteredModules = %d, want 1", len(result.FilteredModules))
	}
}

func TestRun_GraphBuilt(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/stage/eu-central-1/rds",
	})

	result, err := Run(context.Background(), defaultOptions(tmpDir))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Graph should have all modules as nodes
	for _, m := range result.FilteredModules {
		if result.Graph.GetNode(m.ID()) == nil {
			t.Errorf("graph missing node for %s", m.ID())
		}
	}
}
