package gitclient

import (
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
)

// terraform-related file extensions
var terraformExtensions = []string{".tf", ".tfvars", ".terraform.lock.hcl", ".tf.json"}

// ChangedModulesDetector detects which modules have changed.
type ChangedModulesDetector struct {
	gitClient *Client
	index     *discovery.ModuleIndex
	rootDir   string
}

// NewChangedModulesDetector creates a new detector.
func NewChangedModulesDetector(gitClient *Client, index *discovery.ModuleIndex, rootDir string) *ChangedModulesDetector {
	return &ChangedModulesDetector{
		gitClient: gitClient,
		index:     index,
		rootDir:   rootDir,
	}
}

// DetectChanges returns changed modules, raw files, and changed library paths
// from one git diff.
func (d *ChangedModulesDetector) DetectChanges(baseRef string, libraryPaths []string) (modules []*discovery.Module, files, changedLibraries []string, err error) {
	files, err = d.gitClient.GetChangedFiles(baseRef)
	if err != nil {
		return nil, nil, nil, err
	}
	return d.filesToModules(files), files, d.filesToLibraryPaths(files, libraryPaths), nil
}

// DetectUncommittedModules returns modules with uncommitted changes.
func (d *ChangedModulesDetector) DetectUncommittedModules() ([]*discovery.Module, error) {
	files, err := d.gitClient.GetUncommittedChanges()
	if err != nil {
		return nil, err
	}
	return d.filesToModules(files), nil
}

// filesToModules maps changed files to their containing modules.
func (d *ChangedModulesDetector) filesToModules(files []string) []*discovery.Module {
	if d.index == nil {
		return nil
	}

	seen := make(map[string]bool)

	for _, file := range files {
		if !isTerraformRelated(file) {
			continue
		}
		if mod := d.findOwningModule(pathpkg.Dir(cleanWorkspacePath(file))); mod != nil {
			seen[mod.ID()] = true
		}
	}

	modules := make([]*discovery.Module, 0, len(seen))
	for _, module := range d.index.All() {
		if seen[module.ID()] {
			modules = append(modules, module)
		}
	}
	return modules
}

// findOwningModule walks up from dir until it finds a known module.
func (d *ChangedModulesDetector) findOwningModule(dir string) *discovery.Module {
	if d.index == nil {
		return nil
	}
	dir = cleanWorkspacePath(dir)
	for dir != "." && dir != "/" && dir != "" {
		if m := d.index.ByPath(dir); m != nil {
			return m
		}
		if d.rootDir != "" {
			absPath := filepath.Join(d.rootDir, filepath.FromSlash(dir))
			if m := d.index.ByPath(absPath); m != nil {
				return m
			}
			if m := d.index.ByPath(filepath.ToSlash(absPath)); m != nil {
				return m
			}
		}
		if m := d.findByRelativePath(dir); m != nil {
			return m
		}
		dir = pathpkg.Dir(dir)
	}
	return nil
}

func (d *ChangedModulesDetector) findByRelativePath(path string) *discovery.Module {
	if d.index == nil {
		return nil
	}
	normalized := cleanWorkspacePath(path)
	for _, m := range d.index.All() {
		if m.ID() == normalized {
			return m
		}
	}
	return nil
}

// filesToLibraryPaths maps changed files to library module paths.
func (d *ChangedModulesDetector) filesToLibraryPaths(files, libraryPaths []string) []string {
	libs := make(map[string]bool)

	for _, file := range files {
		if !isTerraformRelated(file) {
			continue
		}
		file = cleanWorkspacePath(file)
		for _, libPath := range libraryPaths {
			root := cleanWorkspacePath(libPath)
			if root == "" {
				continue
			}
			if file != root && !strings.HasPrefix(file, root+"/") {
				continue
			}
			libs[d.absoluteLibraryPath(changedLibraryPath(root, file))] = true
		}
	}

	result := make([]string, 0, len(libs))
	for path := range libs {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func changedLibraryPath(root, file string) string {
	rel := strings.TrimPrefix(file, root)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return root
	}
	name, _, ok := strings.Cut(rel, "/")
	if !ok {
		return root
	}
	return pathpkg.Join(root, name)
}

func (d *ChangedModulesDetector) absoluteLibraryPath(relPath string) string {
	if d.rootDir == "" {
		return relPath
	}
	return filepath.ToSlash(filepath.Join(d.rootDir, filepath.FromSlash(relPath)))
}

// isTerraformRelated checks if a file is terraform-related.
func isTerraformRelated(file string) bool {
	file = cleanWorkspacePath(file)
	for _, ext := range terraformExtensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}
	return false
}

func cleanWorkspacePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = pathpkg.Clean(value)
	if value == "." {
		return ""
	}
	return strings.TrimPrefix(value, "./")
}
