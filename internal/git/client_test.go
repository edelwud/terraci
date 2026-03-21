package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func initTestRepo(t *testing.T) (string, *gogit.Repository) {
	t.Helper()
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Add("README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir, repo
}

func TestNewClient(t *testing.T) {
	client := NewClient("/some/dir")
	if client.WorkDir != "/some/dir" {
		t.Errorf("WorkDir = %q, want /some/dir", client.WorkDir)
	}
	if client.repo != nil {
		t.Error("repo should be nil initially")
	}
	if client.fetched {
		t.Error("fetched should be false initially")
	}
}

func TestIsGitRepo(t *testing.T) {
	t.Run("valid git repo", func(t *testing.T) {
		dir, _ := initTestRepo(t)
		client := NewClient(dir)
		if !client.IsGitRepo() {
			t.Error("expected IsGitRepo() = true for git directory")
		}
	})

	t.Run("non-git directory", func(t *testing.T) {
		dir := t.TempDir()
		client := NewClient(dir)
		if client.IsGitRepo() {
			t.Error("expected IsGitRepo() = false for non-git directory")
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		client := NewClient("/nonexistent/path/xyz")
		if client.IsGitRepo() {
			t.Error("expected IsGitRepo() = false for nonexistent directory")
		}
	})
}

func TestGetDefaultBranch(t *testing.T) {
	t.Run("no remotes returns origin/main", func(t *testing.T) {
		dir, _ := initTestRepo(t)
		client := NewClient(dir)
		got := client.GetDefaultBranch()
		if got != "origin/main" {
			t.Errorf("GetDefaultBranch() = %q, want %q", got, "origin/main")
		}
	})

	t.Run("non-git directory returns origin/main", func(t *testing.T) {
		client := NewClient(t.TempDir())
		got := client.GetDefaultBranch()
		if got != "origin/main" {
			t.Errorf("GetDefaultBranch() = %q, want %q", got, "origin/main")
		}
	})
}

func TestResolveRef_Hash(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	// Force open the repo
	client.repo = repo

	headRef, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	hashStr := headRef.Hash().String()

	got, err := client.resolveRefDirect(hashStr)
	if err != nil {
		t.Fatalf("resolveRefDirect(%q) error: %v", hashStr, err)
	}
	if got != headRef.Hash() {
		t.Errorf("resolveRefDirect() = %v, want %v", got, headRef.Hash())
	}
}

func TestResolveRef_Branch(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	client.repo = repo

	headRef, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.resolveRefDirect("master")
	if err != nil {
		t.Fatalf("resolveRefDirect(master) error: %v", err)
	}
	if got != headRef.Hash() {
		t.Errorf("resolveRefDirect(master) = %v, want %v", got, headRef.Hash())
	}
}

func TestResolveRef_HeadRelative(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Create a second commit
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "second.txt"), []byte("second"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Add("second.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Commit("second commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get first commit hash via log
	headRef, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		t.Fatal(err)
	}
	parentCommit, err := headCommit.Parent(0)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	client.repo = repo

	got, err := client.resolveRefDirect("HEAD~1")
	if err != nil {
		t.Fatalf("resolveRefDirect(HEAD~1) error: %v", err)
	}
	if got != parentCommit.Hash {
		t.Errorf("resolveRefDirect(HEAD~1) = %v, want %v", got, parentCommit.Hash)
	}
}

func TestResolveRef_Invalid(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	client.repo = repo

	_, err := client.resolveRefDirect("nonexistent-branch")
	if err == nil {
		t.Error("expected error for nonexistent ref")
	}
}

func TestResolveRef_WithFetch(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	client.repo = repo

	// resolveRef on a nonexistent ref should fail even after trying to fetch
	// (no remote configured, so fetch will fail silently)
	_, err := client.resolveRef("nonexistent-ref-xyz")
	if err == nil {
		t.Error("expected error for nonexistent ref")
	}
}

func TestOpenRepo_Caching(t *testing.T) {
	dir, _ := initTestRepo(t)
	client := NewClient(dir)

	repo1, err := client.openRepo()
	if err != nil {
		t.Fatal(err)
	}
	repo2, err := client.openRepo()
	if err != nil {
		t.Fatal(err)
	}
	if repo1 != repo2 {
		t.Error("openRepo should return cached repository")
	}
}

func TestGetMergeBase(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Create a second commit on master
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("data"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Add("file2.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Commit("second commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	client.repo = repo

	// Merge base of HEAD~1 and HEAD should be HEAD~1
	headRef, headErr := repo.Head()
	if headErr != nil {
		t.Fatal(headErr)
	}
	headCommit, commitErr := repo.CommitObject(headRef.Hash())
	if commitErr != nil {
		t.Fatal(commitErr)
	}
	parent, parentErr := headCommit.Parent(0)
	if parentErr != nil {
		t.Fatal(parentErr)
	}

	got, err := client.getMergeBase("HEAD~1", "master")
	if err != nil {
		t.Fatalf("getMergeBase error: %v", err)
	}
	if got != parent.Hash {
		t.Errorf("getMergeBase = %v, want %v", got, parent.Hash)
	}
}

func TestFetch_NoRemote(t *testing.T) {
	dir, _ := initTestRepo(t)
	client := NewClient(dir)

	// Fetch should return an error when there's no remote
	err := client.Fetch()
	if err == nil {
		t.Error("expected error when fetching without remote")
	}
}

func TestFetch_AlreadyFetched(t *testing.T) {
	client := NewClient(t.TempDir())
	client.fetched = true

	// Should return nil immediately without trying to open repo
	err := client.Fetch()
	if err != nil {
		t.Errorf("Fetch() with fetched=true error: %v", err)
	}
}

func TestResolveHEADRelative_SingleCommit(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	client.repo = repo

	headRef, headErr := repo.Head()
	if headErr != nil {
		t.Fatal(headErr)
	}

	// HEAD~1 on a single commit should return the same commit (no parent to walk to)
	got, err := client.resolveHEADRelative("HEAD~1")
	if err != nil {
		t.Fatalf("resolveHEADRelative error: %v", err)
	}
	if got != headRef.Hash() {
		t.Errorf("resolveHEADRelative(HEAD~1) on single commit = %v, want %v (HEAD)", got, headRef.Hash())
	}
}

func TestResolveRefDirect_RemoteBranch(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	client.repo = repo

	// Test that "origin/somebranch" format is handled
	_, err := client.resolveRefDirect("origin/nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent remote branch")
	}

	// Verify the hash-based resolution with zero hash prefix
	hash := plumbing.ZeroHash.String()
	got, err := client.resolveRefDirect(hash)
	if err != nil {
		t.Fatalf("resolveRefDirect(zero hash) error: %v", err)
	}
	if got != plumbing.ZeroHash {
		t.Errorf("got %v, want zero hash", got)
	}
}
