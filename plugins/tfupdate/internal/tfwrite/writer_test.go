package tfwrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func writeTF(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWriteModuleVersion(t *testing.T) {
	t.Run("updates_version_attribute", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 3.0"
}
`)
		if err := WriteModuleVersion(path, "vpc", "~> 4.0"); err != nil {
			t.Fatalf("WriteModuleVersion() error = %v", err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "~> 4.0") {
			t.Errorf("file content does not contain new version:\n%s", data)
		}
	})

	t.Run("module_not_found", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
module "other" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 1.0"
}
`)
		err := WriteModuleVersion(path, "vpc", "~> 4.0")
		if err == nil {
			t.Fatal("expected error for missing module")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want to contain 'not found'", err.Error())
		}
	})

	t.Run("file_read_error", func(t *testing.T) {
		err := WriteModuleVersion("/nonexistent/path.tf", "vpc", "~> 4.0")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("parse_error", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "bad.tf", `this is not valid HCL {{{`)
		err := WriteModuleVersion(path, "vpc", "~> 4.0")
		if err == nil {
			t.Fatal("expected error for invalid HCL")
		}
		if !strings.Contains(err.Error(), "parse") {
			t.Errorf("error = %q, want to contain 'parse'", err.Error())
		}
	})
}

func TestWriteProviderVersion(t *testing.T) {
	t.Run("updates_version_constraint", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)
		if err := WriteProviderVersion(path, "aws", "~> 5.3"); err != nil {
			t.Fatalf("WriteProviderVersion() error = %v", err)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Error("file should still exist after write")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read updated file: %v", err)
		}
		if !strings.Contains(string(data), "~> 5.3") {
			t.Fatalf("file content does not contain new version:\n%s", data)
		}
	})

	t.Run("provider_not_found", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    gcp = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
  }
}
`)
		err := WriteProviderVersion(path, "aws", "~> 5.3")
		if err == nil {
			t.Fatal("expected error for missing provider")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want to contain 'not found'", err.Error())
		}
	})

	t.Run("file_read_error", func(t *testing.T) {
		err := WriteProviderVersion("/nonexistent/path.tf", "aws", "~> 5.3")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("parse_error", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "bad.tf", `not valid HCL content {{{`)
		err := WriteProviderVersion(path, "aws", "~> 5.3")
		if err == nil {
			t.Fatal("expected error for invalid HCL")
		}
	})

	t.Run("no_terraform_block", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`)
		err := WriteProviderVersion(path, "aws", "~> 5.3")
		if err == nil {
			t.Fatal("expected error when no terraform block")
		}
	})
}

func TestReplaceVersionInTokens_Empty(t *testing.T) {
	result := replaceVersionInTokens(nil, "~> 5.3")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestReplaceVersionInTokens_EmptySlice(t *testing.T) {
	tokens := hclwrite.Tokens{}
	result := replaceVersionInTokens(tokens, "~> 5.3")
	if len(result) != 0 {
		t.Errorf("expected empty tokens, got %d", len(result))
	}
}

func TestReplaceVersionInTokens_WithMatchingPattern(t *testing.T) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte("source")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte("=")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("version")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte("=")},
		{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(`~> 5.0`)},
	}

	result := replaceVersionInTokens(tokens, "~> 5.3")
	if string(result[4].Bytes) != `~> 5.3` {
		t.Errorf("token[4].Bytes = %q, want '\"~> 5.3\"'", result[4].Bytes)
	}
}

func TestReplaceVersionInTokens_NoVersionKey(t *testing.T) {
	tokens := hclwrite.Tokens{
		{Type: 9, Bytes: []byte("source")},
		{Type: 9, Bytes: []byte("hashicorp/aws")},
	}

	result := replaceVersionInTokens(tokens, "~> 5.3")
	if len(result) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(result))
	}
	if string(result[1].Bytes) != "hashicorp/aws" {
		t.Errorf("tokens should be unchanged")
	}
}

func TestReplaceVersionInTokens_WithRealTokens(t *testing.T) {
	dir := t.TempDir()
	content := `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`
	path := filepath.Join(dir, "versions.tf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	src, _ := os.ReadFile(path)
	file, diags := hclwrite.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	for _, block := range file.Body().Blocks() {
		if block.Type() != "terraform" {
			continue
		}
		for _, sub := range block.Body().Blocks() {
			if sub.Type() != "required_providers" {
				continue
			}
			attr := sub.Body().GetAttribute("aws")
			if attr == nil {
				t.Fatal("aws attribute not found")
			}
			tokens := attr.Expr().BuildTokens(nil)
			result := replaceVersionInTokens(tokens, "~> 5.3")
			if len(result) == 0 {
				t.Error("expected non-empty tokens")
			}
			return
		}
	}
	t.Fatal("required_providers block not found")
}

func TestWriteProviderLock(t *testing.T) {
	t.Run("creates_lock_file", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := filepath.Join(dir, ".terraform.lock.hcl")

		err := WriteProviderLock(lockPath, "registry.terraform.io/hashicorp/aws", "5.2.0", "~> 5.2", []string{"h1:test", "zh:test"})
		if err != nil {
			t.Fatalf("WriteProviderLock() error = %v", err)
		}

		data, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("read lock file: %v", err)
		}
		text := string(data)
		for _, want := range []string{
			`provider "registry.terraform.io/hashicorp/aws"`,
			`version     = "5.2.0"`,
			`constraints = "~> 5.2"`,
			`"h1:test"`,
			`"zh:test"`,
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("lock file missing %q:\n%s", want, text)
			}
		}
		// Verify multi-line hash formatting (one hash per line).
		if !strings.Contains(text, "    \"h1:test\",\n") {
			t.Fatalf("hashes should be formatted one-per-line:\n%s", text)
		}
		if !strings.Contains(text, "    \"zh:test\",\n") {
			t.Fatalf("hashes should be formatted one-per-line:\n%s", text)
		}
	})

	t.Run("updates_existing_provider_block", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := writeTF(t, dir, ".terraform.lock.hcl", `
provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.0.0"
  constraints = "~> 5.0"
  hashes      = ["zh:old"]
}
`)

		err := WriteProviderLock(lockPath, "registry.terraform.io/hashicorp/aws", "5.2.0", "~> 5.2", []string{"h1:new", "zh:new"})
		if err != nil {
			t.Fatalf("WriteProviderLock() error = %v", err)
		}

		data, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("read lock file: %v", err)
		}
		text := string(data)
		if strings.Contains(text, "zh:old") {
			t.Fatalf("old hash still present:\n%s", text)
		}
		if !strings.Contains(text, `version     = "5.2.0"`) {
			t.Fatalf("updated version missing:\n%s", text)
		}
	})
}
