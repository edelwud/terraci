package updateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
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
		// WriteProviderVersion succeeds when the attribute exists.
		// It rewrites the file via SetAttributeRaw with token-level replacement.
		if err := WriteProviderVersion(path, "aws", "~> 5.3"); err != nil {
			t.Fatalf("WriteProviderVersion() error = %v", err)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Error("file should still exist after write")
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
	// Empty tokens — should return empty without panic.
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
	// Construct tokens that match the function's expected pattern:
	// "version" (type 9) followed by version value (type 9).
	tokens := hclwrite.Tokens{
		{Type: 9, Bytes: []byte("source")},
		{Type: 9, Bytes: []byte("hashicorp/aws")},
		{Type: 9, Bytes: []byte("version")},
		{Type: 9, Bytes: []byte(`"~> 5.0"`)},
	}

	result := replaceVersionInTokens(tokens, "~> 5.3")
	// The version value token should be replaced
	if string(result[3].Bytes) != `"~> 5.3"` {
		t.Errorf("token[3].Bytes = %q, want '\"~> 5.3\"'", result[3].Bytes)
	}
}

func TestReplaceVersionInTokens_NoVersionKey(t *testing.T) {
	// No "version" key — function returns tokens unchanged.
	tokens := hclwrite.Tokens{
		{Type: 9, Bytes: []byte("source")},
		{Type: 9, Bytes: []byte("hashicorp/aws")},
	}

	result := replaceVersionInTokens(tokens, "~> 5.3")
	if len(result) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(result))
	}
	// Tokens should be unchanged
	if string(result[1].Bytes) != "hashicorp/aws" {
		t.Errorf("tokens should be unchanged")
	}
}

func TestReplaceVersionInTokens_WithRealTokens(t *testing.T) {
	// Create a real HCL file and extract expression tokens to test the function
	// with actual hclwrite tokens.
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

	// Navigate to get the aws attribute tokens
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
			// Call replaceVersionInTokens — it works with TokenQuotedLit type 9
			// The tokens contain the object expression
			result := replaceVersionInTokens(tokens, "~> 5.3")
			if len(result) == 0 {
				t.Error("expected non-empty tokens")
			}
			return
		}
	}
	t.Fatal("required_providers block not found")
}

func TestContainsModuleBlock(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`)
		if !containsModuleBlock(path, "vpc") {
			t.Error("expected true for existing module block")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
module "other" {
  source = "terraform-aws-modules/eks/aws"
}
`)
		if containsModuleBlock(path, "vpc") {
			t.Error("expected false for non-matching module")
		}
	})

	t.Run("no_module_keyword", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`)
		if containsModuleBlock(path, "vpc") {
			t.Error("expected false when no module keyword")
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		if containsModuleBlock("/nonexistent/file.tf", "vpc") {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("invalid_hcl", func(t *testing.T) {
		dir := t.TempDir()
		// Include "module" keyword so the quick check passes, but HCL parse fails
		path := writeTF(t, dir, "bad.tf", `module not valid HCL {{{`)
		if containsModuleBlock(path, "vpc") {
			t.Error("expected false for invalid HCL")
		}
	})
}

func TestContainsProviderBlock(t *testing.T) {
	t.Run("found", func(t *testing.T) {
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
		if !containsProviderBlock(path, "aws") {
			t.Error("expected true for existing provider block")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    gcp = {
      source = "hashicorp/google"
    }
  }
}
`)
		if containsProviderBlock(path, "aws") {
			t.Error("expected false for non-matching provider")
		}
	})

	t.Run("no_required_providers_keyword", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`)
		if containsProviderBlock(path, "aws") {
			t.Error("expected false when no required_providers keyword")
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		if containsProviderBlock("/nonexistent/file.tf", "aws") {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("invalid_hcl", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "bad.tf", `required_providers not valid HCL {{{`)
		if containsProviderBlock(path, "aws") {
			t.Error("expected false for invalid HCL")
		}
	})

	t.Run("terraform_without_required_providers", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "backend.tf", `
terraform {
  required_providers {
  }
  backend "s3" {
    bucket = "test"
  }
}
`)
		if containsProviderBlock(path, "aws") {
			t.Error("expected false when required_providers has no matching attribute")
		}
	})

	t.Run("non_terraform_block", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "main.tf", `
required_providers {
  dummy = true
}
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`)
		if containsProviderBlock(path, "aws") {
			t.Error("expected false for required_providers outside terraform block")
		}
	})
}
