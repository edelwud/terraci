package pipeline

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

const (
	PlanBinaryFilename = "plan.tfplan"
	PlanTextFilename   = "plan.txt"
	PlanJSONFilename   = "plan.json"
)

// WorkspacePath joins workspace-relative path components with POSIX
// separators, independent of the host OS.
func WorkspacePath(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		normalized := strings.Trim(strings.ReplaceAll(part, "\\", "/"), "/")
		if normalized == "" {
			continue
		}
		cleaned = append(cleaned, normalized)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return strings.Join(cleaned, "/")
}

func ValidateWorkspacePath(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	if normalized == "" {
		return errors.New("path is empty")
	}
	if path.IsAbs(normalized) || hasWindowsDrivePrefix(normalized) {
		return fmt.Errorf("path %q must be workspace-relative", value)
	}
	if slices.Contains(strings.Split(normalized, "/"), "..") {
		return fmt.Errorf("path %q must not contain parent directory segments", value)
	}
	return nil
}

func hasWindowsDrivePrefix(value string) bool {
	if len(value) < 2 || value[1] != ':' {
		return false
	}
	return (value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')
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
