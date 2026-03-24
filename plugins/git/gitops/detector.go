package gitops

import (
	"path/filepath"
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

// DetectChangedModules returns modules affected by changed files.
func (d *ChangedModulesDetector) DetectChangedModules(baseRef string) ([]*discovery.Module, error) {
	files, err := d.gitClient.GetChangedFiles(baseRef)
	if err != nil {
		return nil, err
	}
	return d.filesToModules(files), nil
}

// DetectChangedModulesVerbose returns modules and the raw changed file list.
func (d *ChangedModulesDetector) DetectChangedModulesVerbose(baseRef string) ([]*discovery.Module, []string, error) {
	files, err := d.gitClient.GetChangedFiles(baseRef)
	if err != nil {
		return nil, nil, err
	}
	return d.filesToModules(files), files, nil
}

// DetectUncommittedModules returns modules with uncommitted changes.
func (d *ChangedModulesDetector) DetectUncommittedModules() ([]*discovery.Module, error) {
	files, err := d.gitClient.GetUncommittedChanges()
	if err != nil {
		return nil, err
	}
	return d.filesToModules(files), nil
}

// GetChangedModuleIDs returns IDs of changed modules.
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

// DetectChangedLibraryModules returns library module paths that have changed files.
func (d *ChangedModulesDetector) DetectChangedLibraryModules(baseRef string, libraryPaths []string) ([]string, error) {
	files, err := d.gitClient.GetChangedFiles(baseRef)
	if err != nil {
		return nil, err
	}
	return d.filesToLibraryPaths(files, libraryPaths), nil
}

// filesToModules maps changed files to their containing modules.
func (d *ChangedModulesDetector) filesToModules(files []string) []*discovery.Module {
	seen := make(map[string]*discovery.Module)

	for _, file := range files {
		if !isTerraformRelated(file) {
			continue
		}
		if mod := d.findOwningModule(filepath.Dir(file)); mod != nil {
			seen[mod.ID()] = mod
		}
	}

	modules := make([]*discovery.Module, 0, len(seen))
	for _, m := range seen {
		modules = append(modules, m)
	}
	return modules
}

// findOwningModule walks up from dir until it finds a known module.
func (d *ChangedModulesDetector) findOwningModule(dir string) *discovery.Module {
	for dir != "." && dir != "/" && dir != "" {
		if m := d.index.ByPath(dir); m != nil {
			return m
		}
		if m := d.index.ByPath(filepath.Join(d.rootDir, dir)); m != nil {
			return m
		}
		if m := d.findByRelativePath(dir); m != nil {
			return m
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

func (d *ChangedModulesDetector) findByRelativePath(path string) *discovery.Module {
	normalized := filepath.Clean(path)
	for _, m := range d.index.All() {
		if m.RelativePath == normalized {
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
		for _, libPath := range libraryPaths {
			prefix := libPath + "/"
			if !strings.HasPrefix(file, prefix) {
				continue
			}
			// Extract first directory component after libPath
			rel := strings.TrimPrefix(file, prefix)
			if name, _, ok := strings.Cut(rel, "/"); ok && name != "" {
				libs[filepath.Join(d.rootDir, libPath, name)] = true
			} else if rel != "" {
				libs[filepath.Join(d.rootDir, libPath, rel)] = true
			}
		}
	}

	result := make([]string, 0, len(libs))
	for path := range libs {
		result = append(result, path)
	}
	return result
}

// isTerraformRelated checks if a file is terraform-related.
func isTerraformRelated(file string) bool {
	for _, ext := range terraformExtensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}
	return false
}
