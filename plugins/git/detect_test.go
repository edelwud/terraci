package git

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"

	"github.com/edelwud/terraci/pkg/discovery"
	pluginpkg "github.com/edelwud/terraci/pkg/plugin"
)

func TestDetectChanges_RelativeWorkDirAndLibraryPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(t.TempDir(), "repo")
	if mkdirErr := os.MkdirAll(repoDir, 0o755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	repo, err := gogit.PlainInit(repoDir, false)
	if err != nil {
		t.Fatal(err)
	}
	disableTestCommitSigning(t, repo)
	addTestCommit(t, repoDir, repo, map[string]string{"README.md": "initial"}, "initial")

	modPath := "svc/stage/eu/app"
	addTestCommit(t, repoDir, repo, map[string]string{
		filepath.Join(modPath, "main.tf"): "# app",
		"_modules/network/main.tf":        "# lib",
	}, "change terraform")

	relWorkDir, err := filepath.Rel(cwd, repoDir)
	if err != nil {
		t.Fatal(err)
	}
	mod := discovery.NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"svc", "stage", "eu", "app"},
		filepath.Join(repoDir, filepath.FromSlash(modPath)),
		modPath,
	)

	result, err := (&Plugin{}).DetectChanges(context.Background(), pluginpkg.ChangeDetectionRequest{
		WorkDir:      relWorkDir,
		BaseRef:      "HEAD~1",
		ModuleIndex:  discovery.NewModuleIndex([]*discovery.Module{mod}),
		LibraryPaths: []string{"_modules"},
	})
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}

	if got := moduleIDs(result.Modules); !reflect.DeepEqual(got, []string{modPath}) {
		t.Fatalf("module ids = %v, want [%s]", got, modPath)
	}
	wantFiles := []string{"_modules/network/main.tf", "svc/stage/eu/app/main.tf"}
	if !reflect.DeepEqual(result.Files, wantFiles) {
		t.Fatalf("files = %v, want %v", result.Files, wantFiles)
	}
	wantLib := filepath.ToSlash(filepath.Join(repoDir, "_modules", "network"))
	if !reflect.DeepEqual(result.LibraryPaths, []string{wantLib}) {
		t.Fatalf("library paths = %v, want [%s]", result.LibraryPaths, wantLib)
	}
}

func disableTestCommitSigning(t *testing.T, repo *gogit.Repository) {
	t.Helper()
	cfg, err := repo.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Raw.SetOption("commit", "", "gpgsign", "false")
	cfg.Raw.SetOption("tag", "", "gpgsign", "false")
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func addTestCommit(t *testing.T, dir string, repo *gogit.Repository, files map[string]string, msg string) {
	t.Helper()
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		fullPath := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := worktree.Add(filepath.ToSlash(name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := worktree.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}
}

func moduleIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
