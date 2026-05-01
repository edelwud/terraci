package pipeline

// DryRunResult returns a summary of what would be generated.
type DryRunResult struct {
	TotalModules    int
	AffectedModules int
	Stages          int
	Jobs            int
	ExecutionOrder  [][]string
}

// GeneratedPipeline represents a generated CI pipeline.
type GeneratedPipeline interface {
	ToYAML() ([]byte, error)
}

// Generator transforms a pipeline IR into a provider-specific pipeline. The
// IR is bound at construction time; callers do not pass modules to Generate
// or DryRun because the IR already encodes the canonical execution plan.
type Generator interface {
	Generate() (GeneratedPipeline, error)
	DryRun() (*DryRunResult, error)
}
