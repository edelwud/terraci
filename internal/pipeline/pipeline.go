package pipeline

import "github.com/edelwud/terraci/internal/discovery"

// DryRunResult returns a summary of what would be generated
type DryRunResult struct {
	TotalModules    int
	AffectedModules int
	Stages          int
	Jobs            int
	ExecutionOrder  [][]string
}

// GeneratedPipeline represents a generated CI pipeline
type GeneratedPipeline interface {
	ToYAML() ([]byte, error)
}

// Generator defines the interface for CI pipeline generators
type Generator interface {
	Generate(targetModules []*discovery.Module) (GeneratedPipeline, error)
	GenerateForChangedModules(changedModuleIDs []string) (GeneratedPipeline, error)
	DryRun(targetModules []*discovery.Module) (*DryRunResult, error)
}
