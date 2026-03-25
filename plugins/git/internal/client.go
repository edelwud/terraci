// Package git provides Git integration for detecting changed files.
package gitclient

import (
	"errors"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
)

const defaultRemoteBranch = "origin/main"

// Client provides Git operations using go-git.
type Client struct {
	WorkDir string
	repo    *gogit.Repository
	fetched bool
}

// NewClient creates a new Git client.
func NewClient(workDir string) *Client {
	return &Client{WorkDir: workDir}
}

// IsGitRepo checks if the directory is a git repository.
func (c *Client) IsGitRepo() bool {
	_, err := c.openRepo()
	return err == nil
}

// Fetch fetches all refs from the origin remote.
func (c *Client) Fetch() error {
	if c.fetched {
		return nil
	}

	repo, err := c.openRepo()
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}

	err = repo.Fetch(&gogit.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
		Tags:       gogit.AllTags,
		Force:      true,
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch: %w", err)
	}

	c.fetched = true
	return nil
}

// GetDefaultBranch attempts to determine the default branch.
func (c *Client) GetDefaultBranch() string {
	repo, err := c.openRepo()
	if err != nil {
		return defaultRemoteBranch
	}

	for _, branch := range []string{"main", "master"} {
		ref, refErr := repo.Reference(plumbing.NewRemoteReferenceName("origin", branch), true)
		if refErr == nil && ref != nil {
			return "origin/" + branch
		}
	}

	ref, err := repo.Reference("refs/remotes/origin/HEAD", false)
	if err == nil && ref != nil {
		return ref.Target().Short()
	}

	return defaultRemoteBranch
}

func (c *Client) openRepo() (*gogit.Repository, error) {
	if c.repo != nil {
		return c.repo, nil
	}
	repo, err := gogit.PlainOpenWithOptions(c.WorkDir, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}
	c.repo = repo
	return repo, nil
}

// resolveRef resolves a ref string to a commit hash, fetching if needed.
func (c *Client) resolveRef(refStr string) (plumbing.Hash, error) {
	hash, err := c.resolveRefDirect(refStr)
	if err == nil {
		return hash, nil
	}

	if !c.fetched {
		if fetchErr := c.Fetch(); fetchErr == nil {
			if hash, err = c.resolveRefDirect(refStr); err == nil {
				return hash, nil
			}
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("cannot resolve reference: %s", refStr)
}

// resolveRefDirect resolves a ref without fetching.
func (c *Client) resolveRefDirect(refStr string) (plumbing.Hash, error) {
	repo, err := c.openRepo()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	if plumbing.IsHash(refStr) {
		return plumbing.NewHash(refStr), nil
	}

	if strings.HasPrefix(refStr, "HEAD~") || strings.HasPrefix(refStr, "HEAD^") {
		return c.resolveHEADRelative(refStr)
	}

	// Try: local branch → remote branch → remote branch (implicit origin) → tag → full ref
	resolvers := []plumbing.ReferenceName{
		plumbing.NewBranchReferenceName(refStr),
	}
	if branchName, ok := strings.CutPrefix(refStr, "origin/"); ok {
		resolvers = append(resolvers, plumbing.NewRemoteReferenceName("origin", branchName))
	} else {
		resolvers = append(resolvers, plumbing.NewRemoteReferenceName("origin", refStr))
	}
	resolvers = append(resolvers,
		plumbing.NewTagReferenceName(refStr),
		plumbing.ReferenceName(refStr),
	)

	for _, refName := range resolvers {
		if ref, refErr := repo.Reference(refName, true); refErr == nil {
			return ref.Hash(), nil
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("cannot resolve reference: %s", refStr)
}

func (c *Client) resolveHEADRelative(refStr string) (plumbing.Hash, error) {
	repo, err := c.openRepo()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	headRef, err := repo.Head()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	commit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return plumbing.ZeroHash, err
	}

	n := 1
	if len(refStr) > 5 {
		//nolint:errcheck // default n=1 is fine on parse failure
		fmt.Sscanf(refStr[5:], "%d", &n)
	}

	for range n {
		if commit.NumParents() == 0 {
			break
		}
		commit, err = commit.Parent(0)
		if err != nil {
			return plumbing.ZeroHash, err
		}
	}

	return commit.Hash, nil
}

// getMergeBase finds the common ancestor of two refs.
func (c *Client) getMergeBase(ref1, ref2 string) (plumbing.Hash, error) {
	repo, err := c.openRepo()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	hash1, err := c.resolveRef(ref1)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("resolve %s: %w", ref1, err)
	}
	hash2, err := c.resolveRef(ref2)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("resolve %s: %w", ref2, err)
	}

	commit1, err := repo.CommitObject(hash1)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	commit2, err := repo.CommitObject(hash2)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	bases, err := commit1.MergeBase(commit2)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("find merge base: %w", err)
	}
	if len(bases) == 0 {
		return plumbing.ZeroHash, fmt.Errorf("no common ancestor found")
	}

	return bases[0].Hash, nil
}
