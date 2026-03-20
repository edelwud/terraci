package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

var defaultSegments = []string{"service", "environment", "region", "module"}

func TestScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/stage/eu-central-1/ec2",
		"platform/prod/eu-central-1/vpc",
		"platform/stage/eu-central-1/ec2/rabbitmq",
		"platform/stage/eu-central-1/ec2/redis",
	})

	// Wrong depth — should be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "platform", "stage", "main.tf"), []byte("# wrong"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(tmpDir, 4, 5, defaultSegments)
	found, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(found) != 6 {
		t.Errorf("modules = %d, want 6", len(found))
		for _, m := range found {
			t.Logf("  %s (sub: %v)", m.ID(), m.IsSubmodule())
		}
	}

	base, sub := 0, 0
	for _, m := range found {
		if m.IsSubmodule() {
			sub++
		} else {
			base++
		}
	}
	if base != 4 {
		t.Errorf("base modules = %d, want 4", base)
	}
	if sub != 2 {
		t.Errorf("submodules = %d, want 2", sub)
	}
}

func TestScanner_Scan_NoSubmodules(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/ec2",
		"platform/stage/eu-central-1/ec2/rabbitmq",
	})

	scanner := NewScanner(tmpDir, 4, 4, defaultSegments) // MaxDepth=4 disables submodules

	found, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(found) != 2 {
		t.Errorf("modules = %d, want 2 (submodules disabled)", len(found))
	}
}

func TestScanner_ParentChildLinks(t *testing.T) {
	tmpDir := t.TempDir()

	createModuleTree(t, tmpDir, []string{
		"platform/stage/eu-central-1/ec2",
		"platform/stage/eu-central-1/ec2/rabbitmq",
		"platform/stage/eu-central-1/ec2/redis",
	})

	found, err := NewScanner(tmpDir, 4, 5, defaultSegments).Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var parent *Module
	for _, m := range found {
		if m.ID() == "platform/stage/eu-central-1/ec2" {
			parent = m
			break
		}
	}

	if parent == nil {
		t.Fatal("parent not found")
	}
	if len(parent.Children) != 2 {
		t.Errorf("children = %d, want 2", len(parent.Children))
	}
	for _, child := range parent.Children {
		if child.Parent != parent {
			t.Errorf("child %s missing parent link", child.ID())
		}
	}
}
