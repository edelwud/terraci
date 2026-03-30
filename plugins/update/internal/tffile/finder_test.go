package tffile

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTF(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindModuleBlockFile(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, "main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)
		got := FindModuleBlockFile(dir, "vpc")
		if got == "" {
			t.Error("expected to find file for module 'vpc'")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, "main.tf", `
module "other" {
  source = "terraform-aws-modules/eks/aws"
}
`)
		got := FindModuleBlockFile(dir, "vpc")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no_tf_files", func(t *testing.T) {
		dir := t.TempDir()
		got := FindModuleBlockFile(dir, "vpc")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestFindProviderBlockFile(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)
		got := FindProviderBlockFile(dir, "aws")
		if got == "" {
			t.Error("expected to find file for provider 'aws'")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    gcp = {
      source = "hashicorp/google"
    }
  }
}
`)
		got := FindProviderBlockFile(dir, "aws")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no_tf_files", func(t *testing.T) {
		dir := t.TempDir()
		got := FindProviderBlockFile(dir, "aws")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
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
