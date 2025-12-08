// Package git provides Git integration for detecting changed files
package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// Client provides Git operations using go-git
type Client struct {
	// WorkDir is the working directory for git commands
	WorkDir string
	repo    *git.Repository
}

// NewClient creates a new Git client
func NewClient(workDir string) *Client {
	return &Client{WorkDir: workDir}
}

// openRepo opens the git repository lazily
func (c *Client) openRepo() (*git.Repository, error) {
	if c.repo != nil {
		return c.repo, nil
	}

	repo, err := git.PlainOpenWithOptions(c.WorkDir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}

	c.repo = repo
	return repo, nil
}

// IsGitRepo checks if the directory is a git repository
func (c *Client) IsGitRepo() bool {
	_, err := c.openRepo()
	return err == nil
}

// GetChangedFiles returns files changed between base ref and HEAD
func (c *Client) GetChangedFiles(baseRef string) ([]string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// If no base ref specified, compare against HEAD~1
	if baseRef == "" {
		baseRef = "HEAD~1"
	}

	// Get merge-base for better branch comparison
	mergeBaseHash, err := c.getMergeBase(baseRef, "HEAD")
	if err != nil {
		// Fall back to direct ref resolution
		mergeBaseHash, err = c.resolveRef(baseRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base ref %s: %w", baseRef, err)
		}
	}

	// Get HEAD commit
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	baseCommit, err := repo.CommitObject(mergeBaseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit: %w", err)
	}

	// Get trees
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get base tree: %w", err)
	}

	// Get diff
	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	// Collect changed file paths
	var files []string
	for _, change := range changes {
		// Get the path (use To for added/modified, From for deleted)
		path := change.To.Name
		if path == "" {
			path = change.From.Name
		}
		if path != "" {
			files = append(files, path)
		}
	}

	return files, nil
}

// GetChangedFilesFromCommit returns files changed in a specific commit
func (c *Client) GetChangedFilesFromCommit(commitHash string) ([]string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	hash := plumbing.NewHash(commitHash)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s: %w", commitHash, err)
	}

	// Get parent commit (if exists)
	var parentTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent commit: %w", err)
		}
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree: %w", err)
		}
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit tree: %w", err)
	}

	var changes object.Changes
	if parentTree != nil {
		changes, err = parentTree.Diff(commitTree)
	} else {
		// Initial commit - all files are new
		changes, err = (&object.Tree{}).Diff(commitTree)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

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

	return files, nil
}

// GetUncommittedChanges returns uncommitted changed files
func (c *Client) GetUncommittedChanges() ([]string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var files []string
	for path, fileStatus := range status {
		// Include any file that has changes (staged or unstaged)
		if fileStatus.Staging != git.Unmodified || fileStatus.Worktree != git.Unmodified {
			files = append(files, path)
		}
	}

	return files, nil
}

// getMergeBase finds the common ancestor of two refs
func (c *Client) getMergeBase(ref1, ref2 string) (plumbing.Hash, error) {
	repo, err := c.openRepo()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	hash1, err := c.resolveRef(ref1)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to resolve %s: %w", ref1, err)
	}

	hash2, err := c.resolveRef(ref2)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to resolve %s: %w", ref2, err)
	}

	commit1, err := repo.CommitObject(hash1)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	commit2, err := repo.CommitObject(hash2)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	// Find merge base using ancestor traversal
	bases, err := commit1.MergeBase(commit2)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to find merge base: %w", err)
	}

	if len(bases) == 0 {
		return plumbing.ZeroHash, fmt.Errorf("no common ancestor found")
	}

	return bases[0].Hash, nil
}

// resolveRef resolves a ref string to a commit hash
func (c *Client) resolveRef(refStr string) (plumbing.Hash, error) {
	repo, err := c.openRepo()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	// Try as a hash first
	if plumbing.IsHash(refStr) {
		return plumbing.NewHash(refStr), nil
	}

	// Handle HEAD~N notation
	if strings.HasPrefix(refStr, "HEAD~") || strings.HasPrefix(refStr, "HEAD^") {
		headRef, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, err
		}

		commit, err := repo.CommitObject(headRef.Hash())
		if err != nil {
			return plumbing.ZeroHash, err
		}

		// Parse the number of parents to traverse
		n := 1
		if len(refStr) > 5 {
			fmt.Sscanf(refStr[5:], "%d", &n)
		}

		// Walk back n commits
		for i := 0; i < n && commit.NumParents() > 0; i++ {
			commit, err = commit.Parent(0)
			if err != nil {
				return plumbing.ZeroHash, err
			}
		}

		return commit.Hash, nil
	}

	// Try as branch name (local)
	ref, err := repo.Reference(plumbing.NewBranchReferenceName(refStr), true)
	if err == nil {
		return ref.Hash(), nil
	}

	// Try as remote branch (origin/branch)
	if strings.HasPrefix(refStr, "origin/") {
		branchName := strings.TrimPrefix(refStr, "origin/")
		ref, err = repo.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
		if err == nil {
			return ref.Hash(), nil
		}
	} else {
		// If not explicitly prefixed with origin/, try as remote branch anyway
		// This handles CI environments where only remote refs exist (shallow clone, detached HEAD)
		ref, err = repo.Reference(plumbing.NewRemoteReferenceName("origin", refStr), true)
		if err == nil {
			return ref.Hash(), nil
		}
	}

	// Try as tag
	ref, err = repo.Reference(plumbing.NewTagReferenceName(refStr), true)
	if err == nil {
		return ref.Hash(), nil
	}

	// Try as full reference
	ref, err = repo.Reference(plumbing.ReferenceName(refStr), true)
	if err == nil {
		return ref.Hash(), nil
	}

	return plumbing.ZeroHash, fmt.Errorf("cannot resolve reference: %s", refStr)
}

// GetCurrentBranch returns the current branch name
func (c *Client) GetCurrentBranch() (string, error) {
	repo, err := c.openRepo()
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if headRef.Name().IsBranch() {
		return headRef.Name().Short(), nil
	}

	// Detached HEAD
	return headRef.Hash().String()[:7], nil
}

// GetDefaultBranch attempts to determine the default branch
func (c *Client) GetDefaultBranch() string {
	repo, err := c.openRepo()
	if err != nil {
		return "origin/main"
	}

	// Try common default branch names
	for _, branch := range []string{"main", "master"} {
		ref, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", branch), true)
		if err == nil && ref != nil {
			return "origin/" + branch
		}
	}

	// Try to get from origin/HEAD
	ref, err := repo.Reference("refs/remotes/origin/HEAD", false)
	if err == nil && ref != nil {
		target := ref.Target().Short()
		return target
	}

	return "origin/main"
}

// ChangedModulesDetector detects which modules have changed
type ChangedModulesDetector struct {
	gitClient *Client
	index     *discovery.ModuleIndex
	rootDir   string
}

// NewChangedModulesDetector creates a new detector
func NewChangedModulesDetector(gitClient *Client, index *discovery.ModuleIndex, rootDir string) *ChangedModulesDetector {
	return &ChangedModulesDetector{
		gitClient: gitClient,
		index:     index,
		rootDir:   rootDir,
	}
}

// DetectChangedModules returns modules affected by changed files
func (d *ChangedModulesDetector) DetectChangedModules(baseRef string) ([]*discovery.Module, error) {
	changedFiles, err := d.gitClient.GetChangedFiles(baseRef)
	if err != nil {
		return nil, err
	}

	return d.filesToModules(changedFiles), nil
}

// DetectChangedModulesVerbose returns modules affected by changed files with debug info
func (d *ChangedModulesDetector) DetectChangedModulesVerbose(baseRef string) ([]*discovery.Module, []string, error) {
	changedFiles, err := d.gitClient.GetChangedFiles(baseRef)
	if err != nil {
		return nil, nil, err
	}

	return d.filesToModules(changedFiles), changedFiles, nil
}

// DetectUncommittedModules returns modules with uncommitted changes
func (d *ChangedModulesDetector) DetectUncommittedModules() ([]*discovery.Module, error) {
	changedFiles, err := d.gitClient.GetUncommittedChanges()
	if err != nil {
		return nil, err
	}

	return d.filesToModules(changedFiles), nil
}

// filesToModules maps changed files to their modules
func (d *ChangedModulesDetector) filesToModules(files []string) []*discovery.Module {
	moduleSet := make(map[string]*discovery.Module)

	for _, file := range files {
		// Skip non-terraform related files
		if !isTerraformRelatedFile(file) {
			continue
		}

		// Get the directory containing the file
		dir := filepath.Dir(file)

		// Try to find the module this file belongs to
		// Walk up the directory tree until we find a module
		for dir != "." && dir != "/" && dir != "" {
			// Try relative path first (as returned by git)
			if module := d.index.ByPath(dir); module != nil {
				moduleSet[module.ID()] = module
				break
			}

			// Try absolute path
			absDir := filepath.Join(d.rootDir, dir)
			if module := d.index.ByPath(absDir); module != nil {
				moduleSet[module.ID()] = module
				break
			}

			// Try matching by RelativePath directly
			if module := d.findModuleByRelativePath(dir); module != nil {
				moduleSet[module.ID()] = module
				break
			}

			dir = filepath.Dir(dir)
		}
	}

	// Convert to slice
	modules := make([]*discovery.Module, 0, len(moduleSet))
	for _, m := range moduleSet {
		modules = append(modules, m)
	}

	return modules
}

// isTerraformRelatedFile checks if a file is terraform-related
func isTerraformRelatedFile(file string) bool {
	// .tf files
	if strings.HasSuffix(file, ".tf") {
		return true
	}
	// .tfvars files
	if strings.HasSuffix(file, ".tfvars") {
		return true
	}
	// .terraform.lock.hcl
	if strings.HasSuffix(file, ".terraform.lock.hcl") {
		return true
	}
	// .tf.json files
	if strings.HasSuffix(file, ".tf.json") {
		return true
	}
	return false
}

// findModuleByRelativePath searches for a module by checking if the path matches any module's RelativePath
func (d *ChangedModulesDetector) findModuleByRelativePath(path string) *discovery.Module {
	// Normalize path separators
	normalizedPath := filepath.Clean(path)

	for _, m := range d.index.All() {
		if m.RelativePath == normalizedPath {
			return m
		}
	}
	return nil
}

// GetChangedModuleIDs returns IDs of changed modules
func (d *ChangedModulesDetector) GetChangedModuleIDs(baseRef string) ([]string, error) {
	modules, err := d.DetectChangedModules(baseRef)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(modules))
	for i, m := range modules {
		ids[i] = m.ID()
	}

	return ids, nil
}
