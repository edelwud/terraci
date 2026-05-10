package source

import (
	"context"
	"fmt"
	"os"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

type GitSource struct {
	URL string
	Ref string
}

func (s *GitSource) Materialize(ctx context.Context, dest string) (string, error) {
	if err := os.RemoveAll(dest); err != nil {
		return "", fmt.Errorf("clean destination: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:   s.URL,
		Depth: 1,
	}
	if s.Ref != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(s.Ref)
		cloneOpts.SingleBranch = true
	}

	repo, err := git.PlainCloneContext(ctx, dest, cloneOpts)
	if err != nil && s.Ref != "" {
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(s.Ref)
		repo, err = git.PlainCloneContext(ctx, dest, cloneOpts)
	}
	if err != nil {
		return "", fmt.Errorf("clone repository: %w", err)
	}

	if s.Ref != "" && len(s.Ref) == 40 {
		wt, worktreeErr := repo.Worktree()
		if worktreeErr != nil {
			return "", fmt.Errorf("get worktree: %w", worktreeErr)
		}
		if checkoutErr := wt.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(s.Ref)}); checkoutErr != nil {
			return "", fmt.Errorf("checkout commit: %w", checkoutErr)
		}
	}

	return dest, nil
}

func (s *GitSource) String() string {
	if s.Ref != "" {
		return fmt.Sprintf("git:%s@%s", s.URL, s.Ref)
	}
	return "git:" + s.URL
}
