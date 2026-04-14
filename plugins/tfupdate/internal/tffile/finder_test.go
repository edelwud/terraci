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

func TestBuildIndex(t *testing.T) {
	t.Run("finds module and provider files across multiple lookups", func(t *testing.T) {
		dir := t.TempDir()
		moduleFile := writeTF(t, dir, "main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)
		providerFile := writeTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

		index, err := BuildIndex(dir)
		if err != nil {
			t.Fatalf("BuildIndex() error = %v", err)
		}
		if got := index.FindModuleBlockFile("vpc"); got != moduleFile {
			t.Errorf("FindModuleBlockFile(vpc) = %q, want %q", got, moduleFile)
		}
		if got := index.FindProviderBlockFile("aws"); got != providerFile {
			t.Errorf("FindProviderBlockFile(aws) = %q, want %q", got, providerFile)
		}
		if got := index.FindModuleBlockFile("vpc"); got != moduleFile {
			t.Errorf("second FindModuleBlockFile(vpc) = %q, want %q", got, moduleFile)
		}
	})

	t.Run("invalid files are ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, "bad.tf", `module not valid HCL {{{`)

		index, err := BuildIndex(dir)
		if err != nil {
			t.Fatalf("BuildIndex() error = %v", err)
		}
		if got := index.FindModuleBlockFile("vpc"); got != "" {
			t.Errorf("FindModuleBlockFile(vpc) = %q, want empty", got)
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
		facts := inspectFile(path)
		if len(facts.moduleNames) != 1 || facts.moduleNames[0] != "vpc" {
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
		facts := inspectFile(path)
		if len(facts.moduleNames) != 1 || facts.moduleNames[0] != "other" {
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
		if len(inspectFile(path).moduleNames) != 0 {
			t.Error("expected false when no module keyword")
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		if len(inspectFile("/nonexistent/file.tf").moduleNames) != 0 {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("invalid_hcl", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "bad.tf", `module not valid HCL {{{`)
		if len(inspectFile(path).moduleNames) != 0 {
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
		facts := inspectFile(path)
		if len(facts.providerNames) != 1 || facts.providerNames[0] != "aws" {
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
		facts := inspectFile(path)
		if len(facts.providerNames) != 1 || facts.providerNames[0] != "gcp" {
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
		if len(inspectFile(path).providerNames) != 0 {
			t.Error("expected false when no required_providers keyword")
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		if len(inspectFile("/nonexistent/file.tf").providerNames) != 0 {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("invalid_hcl", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTF(t, dir, "bad.tf", `required_providers not valid HCL {{{`)
		if len(inspectFile(path).providerNames) != 0 {
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
		if len(inspectFile(path).providerNames) != 0 {
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
		if len(inspectFile(path).providerNames) != 0 {
			t.Error("expected false for required_providers outside terraform block")
		}
	})
}
