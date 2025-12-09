package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModule_ID(t *testing.T) {
	tests := []struct {
		name     string
		module   *Module
		expected string
	}{
		{
			name: "base module",
			module: &Module{
				Service:     "cdp",
				Environment: "stage",
				Region:      "eu-central-1",
				Module:      "vpc",
			},
			expected: "cdp/stage/eu-central-1/vpc",
		},
		{
			name: "submodule",
			module: &Module{
				Service:     "cdp",
				Environment: "stage",
				Region:      "eu-central-1",
				Module:      "ec2",
				Submodule:   "rabbitmq",
			},
			expected: "cdp/stage/eu-central-1/ec2/rabbitmq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.module.ID(); got != tt.expected {
				t.Errorf("Module.ID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModule_Name(t *testing.T) {
	tests := []struct {
		name     string
		module   *Module
		expected string
	}{
		{
			name: "base module",
			module: &Module{
				Module: "vpc",
			},
			expected: "vpc",
		},
		{
			name: "submodule",
			module: &Module{
				Module:    "ec2",
				Submodule: "rabbitmq",
			},
			expected: "ec2/rabbitmq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.module.Name(); got != tt.expected {
				t.Errorf("Module.Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModule_IsSubmodule(t *testing.T) {
	baseModule := &Module{Module: "vpc"}
	submodule := &Module{Module: "ec2", Submodule: "rabbitmq"}

	if baseModule.IsSubmodule() {
		t.Error("Base module should not be a submodule")
	}

	if !submodule.IsSubmodule() {
		t.Error("Submodule should be identified as submodule")
	}
}

func TestScanner_Scan(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "terraci-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create module directories (base modules)
	baseModules := []string{
		"cdp/stage/eu-central-1/vpc",
		"cdp/stage/eu-central-1/eks",
		"cdp/stage/eu-central-1/ec2",
		"cdp/prod/eu-central-1/vpc",
	}

	// Create submodule directories
	subModules := []string{
		"cdp/stage/eu-central-1/ec2/rabbitmq",
		"cdp/stage/eu-central-1/ec2/redis",
	}

	allModules := make([]string, 0, len(baseModules)+len(subModules))
	allModules = append(allModules, baseModules...)
	allModules = append(allModules, subModules...)

	for _, m := range allModules {
		dir := filepath.Join(tmpDir, m)
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, mkErr)
		}
		// Create a dummy .tf file
		tfFile := filepath.Join(dir, "main.tf")
		if writeErr := os.WriteFile(tfFile, []byte("# test"), 0o644); writeErr != nil {
			t.Fatalf("Failed to create .tf file: %v", writeErr)
		}
	}

	// Create directory at wrong depth (should be ignored)
	wrongDepth := filepath.Join(tmpDir, "cdp", "stage")
	tfFile := filepath.Join(wrongDepth, "main.tf")
	if writeErr := os.WriteFile(tfFile, []byte("# wrong depth"), 0o644); writeErr != nil {
		t.Fatalf("Failed to create .tf file: %v", writeErr)
	}

	// Scan with submodules enabled
	scanner := NewScanner(tmpDir)
	found, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find 4 base modules + 2 submodules = 6 total
	if len(found) != 6 {
		t.Errorf("Expected 6 modules, found %d", len(found))
		for _, m := range found {
			t.Logf("  Found: %s (submodule: %v)", m.ID(), m.IsSubmodule())
		}
	}

	// Count base modules and submodules
	baseCount := 0
	subCount := 0
	for _, m := range found {
		if m.IsSubmodule() {
			subCount++
		} else {
			baseCount++
		}
	}

	if baseCount != 4 {
		t.Errorf("Expected 4 base modules, found %d", baseCount)
	}

	if subCount != 2 {
		t.Errorf("Expected 2 submodules, found %d", subCount)
	}

	// Test scanning with submodules disabled
	scanner.MaxDepth = 4
	found, err = scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(found) != 4 {
		t.Errorf("Expected 4 modules (submodules disabled), found %d", len(found))
	}
}

func TestScanner_ParentChildLinks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "terraci-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create parent and child modules
	modules := []string{
		"cdp/stage/eu-central-1/ec2",
		"cdp/stage/eu-central-1/ec2/rabbitmq",
		"cdp/stage/eu-central-1/ec2/redis",
	}

	for _, m := range modules {
		dir := filepath.Join(tmpDir, m)
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			t.Fatalf("Failed to create dir: %v", mkErr)
		}
		if writeErr := os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# test"), 0o644); writeErr != nil {
			t.Fatalf("Failed to create .tf file: %v", writeErr)
		}
	}

	scanner := NewScanner(tmpDir)
	found, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Find parent module
	var parent *Module
	for _, m := range found {
		if m.ID() == "cdp/stage/eu-central-1/ec2" {
			parent = m
			break
		}
	}

	if parent == nil {
		t.Fatal("Parent module not found")
	}

	// Check parent has children
	if len(parent.Children) != 2 {
		t.Errorf("Expected parent to have 2 children, got %d", len(parent.Children))
	}

	// Check children have parent link
	for _, child := range parent.Children {
		if child.Parent != parent {
			t.Errorf("Child %s should have parent link", child.ID())
		}
	}
}

func TestModuleIndex(t *testing.T) {
	modules := []*Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc", Path: "/test/cdp/stage/eu-central-1/vpc", RelativePath: "cdp/stage/eu-central-1/vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "ec2", Path: "/test/cdp/stage/eu-central-1/ec2", RelativePath: "cdp/stage/eu-central-1/ec2"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "ec2", Submodule: "rabbitmq", Path: "/test/cdp/stage/eu-central-1/ec2/rabbitmq", RelativePath: "cdp/stage/eu-central-1/ec2/rabbitmq"},
		{Service: "cdp", Environment: "prod", Region: "eu-central-1", Module: "vpc", Path: "/test/cdp/prod/eu-central-1/vpc", RelativePath: "cdp/prod/eu-central-1/vpc"},
	}

	idx := NewModuleIndex(modules)

	// Test All
	if len(idx.All()) != 4 {
		t.Errorf("Expected 4 modules, got %d", len(idx.All()))
	}

	// Test ByID for base module
	m := idx.ByID("cdp/stage/eu-central-1/vpc")
	if m == nil {
		t.Error("ByID returned nil for existing base module")
	}

	// Test ByID for submodule
	m = idx.ByID("cdp/stage/eu-central-1/ec2/rabbitmq")
	if m == nil {
		t.Error("ByID returned nil for existing submodule")
	} else if !m.IsSubmodule() {
		t.Error("Expected submodule")
	}

	// Test BaseModules
	baseModules := idx.BaseModules()
	if len(baseModules) != 3 {
		t.Errorf("Expected 3 base modules, got %d", len(baseModules))
	}

	// Test Submodules
	submodules := idx.Submodules()
	if len(submodules) != 1 {
		t.Errorf("Expected 1 submodule, got %d", len(submodules))
	}

	// Test ByName
	ec2Modules := idx.ByName("ec2")
	if len(ec2Modules) != 2 { // ec2 base + ec2/rabbitmq indexed by base name
		t.Errorf("Expected 2 modules named 'ec2', got %d", len(ec2Modules))
	}

	// Test FindInContext
	context := modules[0] // cdp/stage/eu-central-1/vpc
	found := idx.FindInContext("ec2", context)
	if found == nil {
		t.Error("FindInContext should find ec2 in same context")
	} else if found.ID() != "cdp/stage/eu-central-1/ec2" {
		t.Errorf("FindInContext returned wrong module: %s", found.ID())
	}
}

func TestContainsTerraformFiles(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "terraci-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Directory without .tf files
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	if containsTerraformFiles(emptyDir) {
		t.Error("containsTerraformFiles should return false for empty directory")
	}

	// Directory with .tf file
	tfDir := filepath.Join(tmpDir, "terraform")
	if err := os.MkdirAll(tfDir, 0o755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("Failed to create .tf file: %v", err)
	}

	if !containsTerraformFiles(tfDir) {
		t.Error("containsTerraformFiles should return true for directory with .tf files")
	}
}
