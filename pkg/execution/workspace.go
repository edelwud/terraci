package execution

import "path/filepath"

// Workspace centralizes canonical execution-domain paths.
type Workspace struct {
	workDir    string
	serviceDir string
}

// NewWorkspace constructs a Workspace from absolute work and service directories.
func NewWorkspace(workDir, serviceDir string) Workspace {
	return Workspace{workDir: workDir, serviceDir: serviceDir}
}

func (w Workspace) WorkDir() string    { return w.workDir }
func (w Workspace) ServiceDir() string { return w.serviceDir }

// ModuleDir returns the absolute module directory for a relative module path.
func (w Workspace) ModuleDir(rel string) string {
	return filepath.Join(w.workDir, filepath.Clean(rel))
}

// PlanFile returns the absolute plan.tfplan path for a module.
func (w Workspace) PlanFile(rel string) string {
	return filepath.Join(w.ModuleDir(rel), "plan.tfplan")
}

// PlanJSONFile returns the absolute plan.json path for a module.
func (w Workspace) PlanJSONFile(rel string) string {
	return filepath.Join(w.ModuleDir(rel), "plan.json")
}

// PlanTextFile returns the absolute plan.txt path for a module.
func (w Workspace) PlanTextFile(rel string) string {
	return filepath.Join(w.ModuleDir(rel), "plan.txt")
}

// ServiceFile returns the absolute path to a file under the service directory.
func (w Workspace) ServiceFile(name string) string {
	return filepath.Join(w.serviceDir, filepath.Clean(name))
}
