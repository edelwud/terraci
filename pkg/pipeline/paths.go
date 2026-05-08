package pipeline

import "github.com/edelwud/terraci/pkg/workspacepath"

const (
	PlanBinaryFilename = "plan.tfplan"
	PlanTextFilename   = "plan.txt"
	PlanJSONFilename   = "plan.json"
)

// WorkspacePath joins workspace-relative path components with POSIX
// separators, independent of the host OS.
func WorkspacePath(parts ...string) string {
	return workspacepath.Join(parts...)
}

func ValidateWorkspacePath(value string) error {
	return workspacepath.Validate(value)
}

func PlanBinaryPath(modulePath string) string {
	return WorkspacePath(modulePath, PlanBinaryFilename)
}

func PlanTextPath(modulePath string) string {
	return WorkspacePath(modulePath, PlanTextFilename)
}

func PlanJSONPath(modulePath string) string {
	return WorkspacePath(modulePath, PlanJSONFilename)
}
