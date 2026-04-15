package tfupdateengine

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

type ModuleVersionUpdate = domain.ModuleVersionUpdate
type ProviderVersionUpdate = domain.ProviderVersionUpdate
type ModuleDependency = domain.ModuleDependency
type ProviderDependency = domain.ProviderDependency
type LockSyncPlan = domain.LockSyncPlan
type LockProviderSync = domain.LockProviderSync

const (
	StatusUpToDate        = domain.StatusUpToDate
	StatusUpdateAvailable = domain.StatusUpdateAvailable
	StatusApplied         = domain.StatusApplied
	StatusSkipped         = domain.StatusSkipped
	StatusError           = domain.StatusError
)

type mockPackageDownloader struct {
	payloads map[string][]byte
	err      error
	mu       sync.Mutex
	urls     []string
}

func (d *mockPackageDownloader) Download(_ context.Context, url, destPath string) error {
	d.mu.Lock()
	d.urls = append(d.urls, url)
	d.mu.Unlock()
	if d.err != nil {
		return d.err
	}
	return os.WriteFile(destPath, d.payloads[url], 0o600)
}

type mockApplyRegistry struct {
	platforms map[string][]string
	packages  map[string]*registrymeta.ProviderPackage
}

func (r *mockApplyRegistry) ModuleVersions(_ context.Context, _ sourceaddr.ModuleAddress) ([]string, error) {
	return nil, nil
}

func (r *mockApplyRegistry) ModuleProviderDeps(_ context.Context, _ sourceaddr.ModuleAddress, _ string) ([]registrymeta.ModuleProviderDep, error) {
	return nil, nil
}

func (r *mockApplyRegistry) ProviderVersions(_ context.Context, _ sourceaddr.ProviderAddress) ([]string, error) {
	return nil, nil
}

func (r *mockApplyRegistry) ProviderPlatforms(_ context.Context, address sourceaddr.ProviderAddress, version string) ([]string, error) {
	return append([]string(nil), r.platforms[address.Namespace+"/"+address.Type+"@"+version]...), nil
}

func (r *mockApplyRegistry) ProviderPackage(_ context.Context, address sourceaddr.ProviderAddress, version, platform string) (*registrymeta.ProviderPackage, error) {
	pkg := r.packages[address.Namespace+"/"+address.Type+"@"+version+"/"+platform]
	if pkg == nil {
		return nil, nil
	}
	copyPkg := *pkg
	return &copyPkg, nil
}

func writeApplyTF(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func providerLockPlan(providers ...LockProviderSync) []LockSyncPlan {
	if len(providers) == 0 {
		return nil
	}
	return []LockSyncPlan{{
		Providers: providers,
	}}
}

func providerZipFixture(t *testing.T, name, content string) (zipData []byte, checksum string) { //nolint:unparam // name kept as parameter for test readability
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	file, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

func TestApplyUpdates_ErrorSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Dependency: ModuleDependency{CallName: "vpc", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
		Providers: []ProviderVersionUpdate{
			{Dependency: ProviderDependency{ProviderName: "aws", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusError {
		t.Errorf("Module.Status = %q, want %q", result.Modules[0].Status, StatusError)
	}
	if result.Modules[0].Issue == "" {
		t.Error("Module.Issue should be set after write error")
	}
	if result.Providers[0].Status != StatusError {
		t.Errorf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if result.Providers[0].Issue == "" {
		t.Error("Provider.Issue should be set after write error")
	}
}

func TestApplyUpdates_SkipsNotUpdated(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpToDate, File: "some.tf"},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusSkipped, File: "some.tf", Issue: "ignored"},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusUpToDate {
		t.Errorf("Module.Status = %q, want %q", result.Modules[0].Status, StatusUpToDate)
	}
	if result.Providers[0].Status != StatusSkipped {
		t.Errorf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusSkipped)
	}
}

func TestApplyUpdates_ModuleSuccess(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)

	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{
				Dependency:    ModuleDependency{ModulePath: "test", CallName: "vpc", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusApplied {
		t.Fatalf("Module.Status = %q, want %q", result.Modules[0].Status, StatusApplied)
	}
	if result.Modules[0].Issue != "" {
		t.Fatalf("Module.Issue = %q, want empty", result.Modules[0].Issue)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), "~> 5.2") {
		t.Fatalf("updated file does not contain bumped constraint: %s", data)
	}
}

func TestApplyUpdates_ProviderSuccess(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService().Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}
	if result.Providers[0].Issue != "" {
		t.Fatalf("Provider.Issue = %q, want empty", result.Providers[0].Issue)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), "~> 5.2") {
		t.Fatalf("updated file does not contain bumped constraint: %s", data)
	}
}

func TestApplyUpdates_ProviderSuccessUpdatesLockFile(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	darwinZip, darwinSHA := providerZipFixture(t, "terraform-provider-aws", "darwin package")
	downloader := &mockPackageDownloader{
		payloads: map[string][]byte{
			"https://example.test/aws_linux_amd64.zip":  linuxZip,
			"https://example.test/aws_darwin_arm64.zip": darwinZip,
		},
	}
	registry := &mockApplyRegistry{
		platforms: map[string][]string{
			"hashicorp/aws@5.2.0": {"linux_amd64", "darwin_arm64", "linux_amd64"},
		},
		packages: map[string]*registrymeta.ProviderPackage{
			"hashicorp/aws@5.2.0/linux_amd64": {
				Platform:    "linux_amd64",
				DownloadURL: "https://example.test/aws_linux_amd64.zip",
				Shasum:      linuxSHA,
			},
			"hashicorp/aws@5.2.0/darwin_arm64": {
				Platform:    "darwin_arm64",
				DownloadURL: "https://example.test/aws_darwin_arm64.zip",
				Shasum:      darwinSHA,
			},
		},
	}
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(registry),
		WithPackageDownloader(downloader),
	).Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}
	if len(downloader.urls) != 2 {
		t.Fatalf("downloader calls = %d, want 2", len(downloader.urls))
	}

	lockData, err := os.ReadFile(filepath.Join(dir, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	lockText := string(lockData)
	for _, expected := range []string{
		`provider "registry.terraform.io/hashicorp/aws"`,
		`version     = "5.2.0"`,
		`constraints = "~> 5.2"`,
		"zh:" + linuxSHA,
		"zh:" + darwinSHA,
		"h1:",
	} {
		if !strings.Contains(lockText, expected) {
			t.Fatalf("lock file missing %q:\n%s", expected, lockText)
		}
	}
}

func TestApplyUpdates_ProviderSuccessUpdatesLockFile_OpenTofuSource(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "registry.opentofu.org/hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	downloader := &mockPackageDownloader{
		payloads: map[string][]byte{
			"https://example.test/opentofu_aws_linux_amd64.zip": linuxZip,
		},
	}
	registry := &mockApplyRegistry{
		platforms: map[string][]string{
			"hashicorp/aws@5.2.0": {"linux_amd64"},
		},
		packages: map[string]*registrymeta.ProviderPackage{
			"hashicorp/aws@5.2.0/linux_amd64": {
				Platform:    "linux_amd64",
				DownloadURL: "https://example.test/opentofu_aws_linux_amd64.zip",
				Shasum:      linuxSHA,
			},
		},
	}
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency: ProviderDependency{
					ModulePath:     "test",
					ProviderName:   "aws",
					ProviderSource: "registry.opentofu.org/hashicorp/aws",
					Constraint:     "~> 5.0",
				},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "registry.opentofu.org/hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(registry),
		WithPackageDownloader(downloader),
	).Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}

	lockData, err := os.ReadFile(filepath.Join(dir, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	lockText := string(lockData)
	if !strings.Contains(lockText, `provider "registry.opentofu.org/hashicorp/aws"`) {
		t.Fatalf("lock file did not preserve OpenTofu hostname:\n%s", lockText)
	}
	if !strings.Contains(lockText, "zh:"+linuxSHA) {
		t.Fatalf("lock file missing shasum hash:\n%s", lockText)
	}
}

func TestApplyUpdates_ProviderSuccessUpdatesLockFile_ShortSourceDocumentsTerraformDefault(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	downloader := &mockPackageDownloader{
		payloads: map[string][]byte{
			"https://example.test/aws_linux_amd64.zip": linuxZip,
		},
	}
	registry := &mockApplyRegistry{
		platforms: map[string][]string{
			"hashicorp/aws@5.2.0": {"linux_amd64"},
		},
		packages: map[string]*registrymeta.ProviderPackage{
			"hashicorp/aws@5.2.0/linux_amd64": {
				Platform:    "linux_amd64",
				DownloadURL: "https://example.test/aws_linux_amd64.zip",
				Shasum:      linuxSHA,
			},
		},
	}
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency: ProviderDependency{
					ModulePath:     "test",
					ProviderName:   "aws",
					ProviderSource: "hashicorp/aws",
					Constraint:     "~> 5.0",
				},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(registry),
		WithPackageDownloader(downloader),
	).Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}

	lockData, err := os.ReadFile(filepath.Join(dir, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	lockText := string(lockData)
	if !strings.Contains(lockText, `provider "registry.terraform.io/hashicorp/aws"`) {
		t.Fatalf("lock file did not use current Terraform registry default for short source:\n%s", lockText)
	}
}

func TestApplyUpdates_ProviderSuccessUpdatesLockFile_PreservesOpenTofuHostnameForShortSource(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)
	writeApplyTF(t, dir, ".terraform.lock.hcl", `
provider "registry.opentofu.org/hashicorp/aws" {
  version     = "5.0.0"
  constraints = "~> 5.0"
  hashes      = ["zh:old"]
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	downloader := &mockPackageDownloader{
		payloads: map[string][]byte{
			"https://example.test/aws_linux_amd64.zip": linuxZip,
		},
	}
	registry := &mockApplyRegistry{
		platforms: map[string][]string{
			"hashicorp/aws@5.2.0": {"linux_amd64"},
		},
		packages: map[string]*registrymeta.ProviderPackage{
			"hashicorp/aws@5.2.0/linux_amd64": {
				Platform:    "linux_amd64",
				DownloadURL: "https://example.test/aws_linux_amd64.zip",
				Shasum:      linuxSHA,
			},
		},
	}
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency: ProviderDependency{
					ModulePath:     "test",
					ProviderName:   "aws",
					ProviderSource: "hashicorp/aws",
					Constraint:     "~> 5.0",
				},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(registry),
		WithPackageDownloader(downloader),
	).Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}

	lockData, err := os.ReadFile(filepath.Join(dir, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	lockText := string(lockData)
	if !strings.Contains(lockText, `provider "registry.opentofu.org/hashicorp/aws"`) {
		t.Fatalf("expected updated lock file to preserve OpenTofu hostname for short provider source:\n%s", lockText)
	}
	if !strings.Contains(lockText, `provider "registry.opentofu.org/hashicorp/aws" {
  version     = "5.2.0"`) {
		t.Fatalf("expected OpenTofu lock entry to be updated in place:\n%s", lockText)
	}
	if strings.Contains(lockText, `provider "registry.terraform.io/hashicorp/aws"`) {
		t.Fatalf("did not expect terraform registry lock entry to be introduced:\n%s", lockText)
	}
}

func TestApplyUpdates_ProviderLockSyncReusesPackageHashesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	firstPath := writeApplyTF(t, dir, "first.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)
	secondDir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatal(err)
	}
	secondPath := writeApplyTF(t, secondDir, "second.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	downloader := &mockPackageDownloader{
		payloads: map[string][]byte{
			"https://example.test/aws_linux_amd64.zip": linuxZip,
		},
	}
	registry := &mockApplyRegistry{
		platforms: map[string][]string{
			"hashicorp/aws@5.2.0": {"linux_amd64"},
		},
		packages: map[string]*registrymeta.ProviderPackage{
			"hashicorp/aws@5.2.0/linux_amd64": {
				Platform:    "linux_amd64",
				DownloadURL: "https://example.test/aws_linux_amd64.zip",
				Shasum:      linuxSHA,
			},
		},
	}
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          firstPath,
				BumpedVersion: "5.2.0",
			},
			{
				Dependency:    ProviderDependency{ModulePath: "test/nested", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          secondPath,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(
			LockProviderSync{
				ProviderSource: "hashicorp/aws",
				Version:        "5.2.0",
				Constraint:     "~> 5.2",
				TerraformFile:  firstPath,
			},
			LockProviderSync{
				ProviderSource: "hashicorp/aws",
				Version:        "5.2.0",
				Constraint:     "~> 5.2",
				TerraformFile:  secondPath,
			},
		),
	}

	NewApplyService(
		WithRegistryClient(registry),
		WithPackageDownloader(downloader),
	).Apply(result)

	if len(downloader.urls) != 1 {
		t.Fatalf("downloader calls = %d, want 1", len(downloader.urls))
	}
	for i := range result.Providers {
		if result.Providers[i].Status != StatusApplied {
			t.Fatalf("Provider[%d].Status = %q, want %q", i, result.Providers[i].Status, StatusApplied)
		}
	}
}

func TestApplyUpdates_StopsAfterContextCancellation(t *testing.T) {
	dir := t.TempDir()
	firstPath := writeApplyTF(t, dir, "first.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)
	secondPath := writeApplyTF(t, dir, "second.tf", `
terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}
`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          firstPath,
				BumpedVersion: "5.2.0",
			},
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "random", ProviderSource: "hashicorp/random", Constraint: "~> 3.0"},
				Status:        StatusUpdateAvailable,
				File:          secondPath,
				BumpedVersion: "3.2.0",
			},
		},
	}

	NewApplyService(WithApplyContext(ctx)).Apply(result)

	for i := range result.Providers {
		if result.Providers[i].Status != StatusError {
			t.Fatalf("Provider[%d].Status = %q, want %q", i, result.Providers[i].Status, StatusError)
		}
		if !strings.Contains(result.Providers[i].Issue, "apply canceled: context canceled") {
			t.Fatalf("Provider[%d].Issue = %q", i, result.Providers[i].Issue)
		}
	}
}

func TestApplyUpdates_ProviderLockFailureSetsError(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(&mockApplyRegistry{
			platforms: map[string][]string{
				"hashicorp/aws@5.2.0": {"linux_amd64"},
			},
			packages: map[string]*registrymeta.ProviderPackage{
				"hashicorp/aws@5.2.0/linux_amd64": {
					Platform:    "linux_amd64",
					DownloadURL: "https://example.test/aws_linux_amd64.zip",
					Shasum:      "abc123",
				},
			},
		}),
		WithPackageDownloader(&mockPackageDownloader{err: errors.New("download failed")}),
	).Apply(result)

	if result.Providers[0].Status != StatusError {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if !strings.Contains(result.Providers[0].Issue, "update provider lock file") {
		t.Fatalf("Provider.Issue = %q, want lock file error", result.Providers[0].Issue)
	}
}

func TestApplyUpdates_InvalidBumpedVersionSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{
				Dependency:    ModuleDependency{ModulePath: "test", CallName: "vpc", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          "main.tf",
				BumpedVersion: "bad",
			},
		},
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          "versions.tf",
				BumpedVersion: "bad",
			},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusError {
		t.Fatalf("Module.Status = %q, want %q", result.Modules[0].Status, StatusError)
	}
	if !strings.Contains(result.Modules[0].Issue, "failed to build") {
		t.Fatalf("Module.Issue = %q, want apply build error", result.Modules[0].Issue)
	}
	if result.Providers[0].Status != StatusError {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if !strings.Contains(result.Providers[0].Issue, "failed to build") {
		t.Fatalf("Provider.Issue = %q, want apply build error", result.Providers[0].Issue)
	}
}

func TestApplyUpdates_ModuleSuccess_PinExactVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)

	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{
				Dependency:    ModuleDependency{ModulePath: "test", CallName: "vpc", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
	}

	NewApplyService(WithPinDependencies(true)).Apply(result)

	if result.Modules[0].Status != StatusApplied {
		t.Fatalf("Module.Status = %q, want %q", result.Modules[0].Status, StatusApplied)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), `version = "5.2.0"`) {
		t.Fatalf("updated file does not contain exact pinned version:\n%s", data)
	}
}

func TestApplyUpdates_ModuleUpToDate_PinExactVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "main.tf", `
module "resolver_endpoint" {
  source  = "terraform-aws-modules/route53/aws//modules/resolver-endpoint"
  version = "~> 6"
}
`)

	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{
				Dependency:     ModuleDependency{ModulePath: "test", CallName: "resolver_endpoint", Constraint: "~> 6"},
				Status:         StatusUpToDate,
				File:           path,
				CurrentVersion: "6.0.0",
				BumpedVersion:  "6.3.0",
			},
		},
	}

	NewApplyService(WithPinDependencies(true)).Apply(result)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), `version = "6.3.0"`) {
		t.Fatalf("updated file does not contain exact pinned version:\n%s", data)
	}
}

func TestApplyUpdates_ProviderSuccess_PinExactVersionAndLockConstraint(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	downloader := &mockPackageDownloader{
		payloads: map[string][]byte{
			"https://example.test/aws_linux_amd64.zip": linuxZip,
		},
	}
	registry := &mockApplyRegistry{
		platforms: map[string][]string{
			"hashicorp/aws@5.2.0": {"linux_amd64"},
		},
		packages: map[string]*registrymeta.ProviderPackage{
			"hashicorp/aws@5.2.0/linux_amd64": {
				Platform:    "linux_amd64",
				DownloadURL: "https://example.test/aws_linux_amd64.zip",
				Shasum:      linuxSHA,
			},
		},
	}
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "5.2.0",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(registry),
		WithPackageDownloader(downloader),
		WithPinDependencies(true),
	).Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), `version = "5.2.0"`) {
		t.Fatalf("updated provider constraint is not pinned:\n%s", data)
	}

	lockData, err := os.ReadFile(filepath.Join(dir, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if !strings.Contains(string(lockData), `constraints = "5.2.0"`) {
		t.Fatalf("updated lock constraint is not pinned:\n%s", lockData)
	}
}

func TestApplyUpdates_FilelessProviderUpdateSyncsLockWithoutWriteError(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	linuxZip, linuxSHA := providerZipFixture(t, "terraform-provider-aws", "linux package")
	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{{
			Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "local", ProviderSource: "hashicorp/local", Constraint: "~> 2.0"},
			Status:        StatusUpdateAvailable,
			File:          "",
			BumpedVersion: "2.5.0",
		}},
		LockSync: providerLockPlan(LockProviderSync{
			ProviderSource: "hashicorp/aws",
			Version:        "5.2.0",
			Constraint:     "~> 5.2",
			TerraformFile:  path,
		}),
	}

	NewApplyService(
		WithRegistryClient(&mockApplyRegistry{
			platforms: map[string][]string{
				"hashicorp/aws@5.2.0": {"linux_amd64"},
			},
			packages: map[string]*registrymeta.ProviderPackage{
				"hashicorp/aws@5.2.0/linux_amd64": {
					Platform:    "linux_amd64",
					DownloadURL: "https://example.test/aws_linux_amd64.zip",
					Shasum:      linuxSHA,
				},
			},
		}),
		WithPackageDownloader(&mockPackageDownloader{
			payloads: map[string][]byte{
				"https://example.test/aws_linux_amd64.zip": linuxZip,
			},
		}),
	).Apply(result)

	if result.Providers[0].Status != StatusUpdateAvailable {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusUpdateAvailable)
	}
	if result.Providers[0].Issue != "" {
		t.Fatalf("Provider.Issue = %q, want empty", result.Providers[0].Issue)
	}
	if _, err := os.Stat(filepath.Join(dir, ".terraform.lock.hcl")); err != nil {
		t.Fatalf("lock file should be created: %v", err)
	}
}

func TestParseVersionOrZero(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got := parseVersionOrZero("1.2.3")
		if got != (v(1, 2, 3, "")) {
			t.Errorf("parseVersionOrZero(1.2.3) = %v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		got := parseVersionOrZero("bad")
		if !got.IsZero() {
			t.Errorf("parseVersionOrZero(bad) = %v, want zero", got)
		}
	})
}

func TestBuildAppliedConstraint(t *testing.T) {
	t.Run("range style", func(t *testing.T) {
		got, ok := buildAppliedConstraint("5.2.0", "~> 5.0", false)
		if !ok {
			t.Fatal("buildAppliedConstraint() ok = false")
		}
		if got != "~> 5.2" {
			t.Fatalf("buildAppliedConstraint() = %q, want %q", got, "~> 5.2")
		}
	})

	t.Run("pinned style", func(t *testing.T) {
		got, ok := buildAppliedConstraint("5.2.0", "~> 5.0", true)
		if !ok {
			t.Fatal("buildAppliedConstraint() ok = false")
		}
		if got != "5.2.0" {
			t.Fatalf("buildAppliedConstraint() = %q, want %q", got, "5.2.0")
		}
	})
}

func TestComputeSummary(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpdateAvailable},
			{Status: StatusSkipped},
			{Status: StatusUpToDate},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusApplied},
			{Status: StatusError},
		},
	}

	s := BuildUpdateSummary(result)
	if s.TotalChecked != 5 {
		t.Errorf("TotalChecked = %d, want 5", s.TotalChecked)
	}
	if s.UpdatesAvailable != 1 {
		t.Errorf("UpdatesAvailable = %d, want 1", s.UpdatesAvailable)
	}
	if s.UpdatesApplied != 1 {
		t.Errorf("UpdatesApplied = %d, want 1", s.UpdatesApplied)
	}
	if s.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", s.Skipped)
	}
	if s.Errors != 1 {
		t.Errorf("Errors = %d, want 1", s.Errors)
	}
}
