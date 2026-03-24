package policy

import (
	"context"
	"fmt"
	"os"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// GitSource represents a git repository source
type GitSource struct {
	URL string
	Ref string // branch, tag, or commit SHA
}

// Pull clones the git repository to the destination directory
func (s *GitSource) Pull(ctx context.Context, dest string) error {
	// Remove existing directory if it exists
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to clean destination: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:      s.URL,
		Depth:    1,
		Progress: nil, // Could add progress writer for verbose mode
	}

	// Set reference if specified
	if s.Ref != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(s.Ref)
		cloneOpts.SingleBranch = true
	}

	repo, err := git.PlainCloneContext(ctx, dest, cloneOpts)
	if err != nil {
		// Try as tag if branch clone failed
		if s.Ref != "" {
			cloneOpts.ReferenceName = plumbing.NewTagReferenceName(s.Ref)
			repo, err = git.PlainCloneContext(ctx, dest, cloneOpts)
		}
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// If ref is a commit SHA, checkout that commit
	if s.Ref != "" && len(s.Ref) == 40 {
		wt, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		err = wt.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(s.Ref),
		})
		if err != nil {
			return fmt.Errorf("failed to checkout commit: %w", err)
		}
	}

	return nil
}

// String returns a human-readable description
func (s *GitSource) String() string {
	if s.Ref != "" {
		return fmt.Sprintf("git:%s@%s", s.URL, s.Ref)
	}
	return fmt.Sprintf("git:%s", s.URL)
}
