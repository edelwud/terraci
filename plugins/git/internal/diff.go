package gitclient

import (
	"fmt"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// GetChangedFiles returns files changed between base ref and HEAD.
func (c *Client) GetChangedFiles(baseRef string) ([]string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}

	if baseRef == "" {
		baseRef = "HEAD~1"
	}

	baseHash, err := c.getMergeBase(baseRef, "HEAD")
	if err != nil {
		baseHash, err = c.resolveRef(baseRef)
		if err != nil {
			return nil, fmt.Errorf("resolve base ref %s: %w", baseRef, err)
		}
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("get HEAD: %w", err)
	}

	return c.diffCommits(repo, baseHash, headRef.Hash())
}

// GetChangedFilesFromCommit returns files changed in a specific commit.
func (c *Client) GetChangedFilesFromCommit(commitHash string) ([]string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}

	hash := plumbing.NewHash(commitHash)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("get commit %s: %w", commitHash, err)
	}

	var parentHash plumbing.Hash
	if commit.NumParents() > 0 {
		parent, parentErr := commit.Parent(0)
		if parentErr != nil {
			return nil, fmt.Errorf("get parent commit: %w", parentErr)
		}
		parentHash = parent.Hash
	}

	return c.diffCommits(repo, parentHash, hash)
}

// GetUncommittedChanges returns uncommitted changed files.
func (c *Client) GetUncommittedChanges() ([]string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}

	var files []string
	for path, fs := range status {
		if fs.Staging != gogit.Unmodified || fs.Worktree != gogit.Unmodified {
			files = append(files, path)
		}
	}
	return files, nil
}

// diffCommits computes changed file paths between two commits.
func (c *Client) diffCommits(repo *gogit.Repository, baseHash, headHash plumbing.Hash) ([]string, error) {
	baseTree, err := commitTree(repo, baseHash)
	if err != nil {
		return nil, fmt.Errorf("get base tree: %w", err)
	}

	headTree, err := commitTree(repo, headHash)
	if err != nil {
		return nil, fmt.Errorf("get head tree: %w", err)
	}

	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("compute diff: %w", err)
	}

	return extractPaths(changes), nil
}

// commitTree returns the tree for a commit hash (empty tree for zero hash).
func commitTree(repo *gogit.Repository, hash plumbing.Hash) (*object.Tree, error) {
	if hash.IsZero() {
		return &object.Tree{}, nil
	}
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, err
	}
	return commit.Tree()
}

// extractPaths collects file paths from a set of changes.
func extractPaths(changes object.Changes) []string {
	var files []string
	for _, change := range changes {
		path := change.To.Name
		if path == "" {
			path = change.From.Name
		}
		if path != "" {
			files = append(files, path)
		}
	}
	return files
}
