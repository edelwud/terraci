package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainsTerraformFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty dir
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if containsTerraformFiles(emptyDir) {
		t.Error("should be false for empty dir")
	}

	// Dir with .tf file
	tfDir := filepath.Join(tmpDir, "terraform")
	if err := os.MkdirAll(tfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !containsTerraformFiles(tfDir) {
		t.Error("should be true for dir with .tf files")
	}

	// Nonexistent dir
	if containsTerraformFiles("/nonexistent/path") {
		t.Error("should be false for nonexistent dir")
	}
}

func TestShouldSkipDir(t *testing.T) {
	tmpDir := t.TempDir()

	hidden := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(hidden)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldSkipDir(info) {
		t.Error("should skip hidden dir")
	}

	normal := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(normal, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(normal)
	if err != nil {
		t.Fatal(err)
	}
	if shouldSkipDir(info) {
		t.Error("should not skip normal dir")
	}
}

func TestIsTerraformDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Dir with .tf
	tfDir := filepath.Join(tmpDir, "mod")
	if err := os.MkdirAll(tfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte("# tf"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(tfDir)
	if err != nil {
		t.Fatal(err)
	}
	if !isTerraformDir(info, tfDir) {
		t.Error("should be true for dir with .tf files")
	}

	// File, not dir
	filePath := filepath.Join(tmpDir, "file.txt")
	err = os.WriteFile(filePath, []byte("text"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if isTerraformDir(info, filePath) {
		t.Error("should be false for file")
	}
}
