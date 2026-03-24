package gitops

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func addCommit(t *testing.T, dir string, repo *gogit.Repository, filename, content, msg string) plumbing.Hash {
	t.Helper()
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create parent directories if needed
	fullPath := filepath.Join(dir, filename)
	err = os.MkdirAll(filepath.Dir(fullPath), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(fullPath, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Add(filename)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := w.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func TestGetChangedFiles(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Add a second commit with a new file
	addCommit(t, dir, repo, "new_file.tf", "resource {}", "add terraform file")

	client := NewClient(dir)

	files, err := client.GetChangedFiles("HEAD~1")
	if err != nil {
		t.Fatalf("GetChangedFiles error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "new_file.tf" {
		t.Errorf("files[0] = %q, want %q", files[0], "new_file.tf")
	}
}

func TestGetChangedFiles_EmptyBaseRef(t *testing.T) {
	dir, repo := initTestRepo(t)
	addCommit(t, dir, repo, "second.txt", "data", "second commit")

	client := NewClient(dir)

	// Empty base ref defaults to HEAD~1
	files, err := client.GetChangedFiles("")
	if err != nil {
		t.Fatalf("GetChangedFiles(\"\") error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "second.txt" {
		t.Errorf("files[0] = %q, want %q", files[0], "second.txt")
	}
}

func TestGetChangedFiles_MultipleFiles(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Add commit with multiple files
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.tf", "b.tf", "c.txt"} {
		err = os.WriteFile(filepath.Join(dir, f), []byte("content"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
		_, err = w.Add(f)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = w.Commit("add multiple files", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	files, err := client.GetChangedFiles("HEAD~1")
	if err != nil {
		t.Fatalf("GetChangedFiles error: %v", err)
	}

	sort.Strings(files)
	want := []string{"a.tf", "b.tf", "c.txt"}
	if len(files) != len(want) {
		t.Fatalf("got %v, want %v", files, want)
	}
	for i := range files {
		if files[i] != want[i] {
			t.Errorf("files[%d] = %q, want %q", i, files[i], want[i])
		}
	}
}

func TestGetChangedFilesFromCommit(t *testing.T) {
	dir, repo := initTestRepo(t)

	commitHash := addCommit(t, dir, repo, "feature.tf", "resource {}", "add feature")

	client := NewClient(dir)
	files, err := client.GetChangedFilesFromCommit(commitHash.String())
	if err != nil {
		t.Fatalf("GetChangedFilesFromCommit error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "feature.tf" {
		t.Errorf("files[0] = %q, want %q", files[0], "feature.tf")
	}
}

func TestGetChangedFilesFromCommit_InitialCommit(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Get the initial commit hash
	headRef, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	files, err := client.GetChangedFilesFromCommit(headRef.Hash().String())
	if err != nil {
		t.Fatalf("GetChangedFilesFromCommit error: %v", err)
	}

	// Initial commit should show README.md
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "README.md" {
		t.Errorf("files[0] = %q, want %q", files[0], "README.md")
	}
}

func TestGetChangedFilesFromCommit_InvalidHash(t *testing.T) {
	dir, _ := initTestRepo(t)
	client := NewClient(dir)

	_, err := client.GetChangedFilesFromCommit("0000000000000000000000000000000000000001")
	if err == nil {
		t.Error("expected error for invalid commit hash")
	}
}

func TestGetUncommittedChanges(t *testing.T) {
	dir, _ := initTestRepo(t)

	// Modify a file without committing
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	files, err := client.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("GetUncommittedChanges error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "README.md" {
		t.Errorf("files[0] = %q, want %q", files[0], "README.md")
	}
}

func TestGetUncommittedChanges_NoChanges(t *testing.T) {
	dir, _ := initTestRepo(t)

	client := NewClient(dir)
	files, err := client.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("GetUncommittedChanges error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("got %d files, want 0: %v", len(files), files)
	}
}

func TestGetUncommittedChanges_NewFile(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Add a new untracked file and stage it
	if err := os.WriteFile(filepath.Join(dir, "new.tf"), []byte("resource {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, wtErr := repo.Worktree()
	if wtErr != nil {
		t.Fatal(wtErr)
	}
	if _, err := w.Add("new.tf"); err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	files, err := client.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("GetUncommittedChanges error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "new.tf" {
		t.Errorf("files[0] = %q, want %q", files[0], "new.tf")
	}
}

func TestGetUncommittedChanges_NotGitRepo(t *testing.T) {
	client := NewClient(t.TempDir())
	_, err := client.GetUncommittedChanges()
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestExtractPaths(t *testing.T) {
	t.Run("nil changes", func(t *testing.T) {
		files := extractPaths(nil)
		if len(files) != 0 {
			t.Errorf("got %v, want empty", files)
		}
	})

	t.Run("empty changes", func(t *testing.T) {
		files := extractPaths(object.Changes{})
		if len(files) != 0 {
			t.Errorf("got %v, want empty", files)
		}
	})
}

func TestCommitTree_ZeroHash(t *testing.T) {
	dir, repo := initTestRepo(t)
	_ = dir

	tree, err := commitTree(repo, plumbing.ZeroHash)
	if err != nil {
		t.Fatalf("commitTree(zero hash) error: %v", err)
	}
	if tree == nil {
		t.Fatal("commitTree(zero hash) returned nil tree")
	}
	// Zero hash should produce empty tree
	if len(tree.Entries) != 0 {
		t.Errorf("expected empty tree, got %d entries", len(tree.Entries))
	}
}

func TestCommitTree_ValidHash(t *testing.T) {
	dir, repo := initTestRepo(t)
	_ = dir

	headRef, headErr := repo.Head()
	if headErr != nil {
		t.Fatal(headErr)
	}
	tree, err := commitTree(repo, headRef.Hash())
	if err != nil {
		t.Fatalf("commitTree error: %v", err)
	}
	if tree == nil {
		t.Fatal("commitTree returned nil")
	}
	// Should have at least README.md
	if len(tree.Entries) == 0 {
		t.Error("expected non-empty tree")
	}
}

func TestCommitTree_InvalidHash(t *testing.T) {
	_, repo := initTestRepo(t)

	_, err := commitTree(repo, plumbing.NewHash("0000000000000000000000000000000000000001"))
	if err == nil {
		t.Error("expected error for invalid hash")
	}
}

func TestGetChangedFiles_NestedPaths(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Add files in nested directories
	addCommit(t, dir, repo, "myapp/prod/us-east-1/vpc/main.tf", "resource {}", "add vpc module")

	client := NewClient(dir)
	files, err := client.GetChangedFiles("HEAD~1")
	if err != nil {
		t.Fatalf("GetChangedFiles error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "myapp/prod/us-east-1/vpc/main.tf" {
		t.Errorf("files[0] = %q, want nested path", files[0])
	}
}

func TestDiffCommits_SameCommit(t *testing.T) {
	dir, repo := initTestRepo(t)
	client := NewClient(dir)
	client.repo = repo

	headRef, headErr := repo.Head()
	if headErr != nil {
		t.Fatal(headErr)
	}

	files, err := client.diffCommits(repo, headRef.Hash(), headRef.Hash())
	if err != nil {
		t.Fatalf("diffCommits same commit error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no changes for same commit, got %v", files)
	}
}
