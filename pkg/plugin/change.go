package plugin

import "github.com/edelwud/terraci/pkg/workflow"

// ChangeDetectionProvider detects changed modules from git (or other VCS).
type ChangeDetectionProvider interface {
	Plugin
	workflow.ChangeDetector
}
