// Package git provides Git integration for detecting changed files
package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
)

// Client provides Git operations
type Client struct {
	// WorkDir is the working directory for git commands
	WorkDir string
}

// NewClient creates a new Git client
func NewClient(workDir string) *Client {
	return &Client{WorkDir: workDir}
}

// IsGitRepo checks if the directory is a git repository
func (c *Client) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = c.WorkDir
	return cmd.Run() == nil
}

// GetChangedFiles returns files changed between base ref and HEAD
func (c *Client) GetChangedFiles(baseRef string) ([]string, error) {
	// If no base ref specified, compare against HEAD~1
	if baseRef == "" {
		baseRef = "HEAD~1"
	}

	// Get diff against merge-base for better branch comparison
	mergeBase, err := c.getMergeBase(baseRef, "HEAD")
	if err != nil {
		// Fall back to direct comparison if merge-base fails
		mergeBase = baseRef
	}

	cmd := exec.Command("git", "diff", "--name-only", mergeBase, "HEAD")
	cmd.Dir = c.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	return parseLines(output), nil
}

// GetChangedFilesFromCommit returns files changed in a specific commit
func (c *Client) GetChangedFilesFromCommit(commitHash string) ([]string, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", commitHash)
	cmd.Dir = c.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree failed: %w", err)
	}

	return parseLines(output), nil
}

// GetUncommittedChanges returns uncommitted changed files
func (c *Client) GetUncommittedChanges() ([]string, error) {
	// Get both staged and unstaged changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = c.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 3 {
			// Format: XY filename or XY orig -> renamed
			file := strings.TrimSpace(line[3:])
			// Handle renames (A -> B format)
			if idx := strings.Index(file, " -> "); idx != -1 {
				file = file[idx+4:]
			}
			files = append(files, file)
		}
	}

	return files, scanner.Err()
}

// getMergeBase finds the common ancestor of two refs
func (c *Client) getMergeBase(ref1, ref2 string) (string, error) {
	cmd := exec.Command("git", "merge-base", ref1, ref2)
	cmd.Dir = c.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git merge-base failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the current branch name
func (c *Client) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = c.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch attempts to determine the default branch
func (c *Client) GetDefaultBranch() string {
	// Try common default branch names
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", "origin/"+branch)
		cmd.Dir = c.WorkDir
		if cmd.Run() == nil {
			return "origin/" + branch
		}
	}

	// Try to get from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = c.WorkDir
	if output, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(output))
		return strings.TrimPrefix(ref, "refs/remotes/")
	}

	return "origin/main"
}

// parseLines splits output into lines, removing empty ones
func parseLines(data []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
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
