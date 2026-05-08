package execution

import "github.com/edelwud/terraci/pkg/pipeline"

// DefaultScheduler schedules jobs from the pipeline DAG.
type DefaultScheduler struct{}

func (DefaultScheduler) Schedule(ir *pipeline.IR) ([]pipeline.JobGroup, error) {
	return pipeline.Schedule(ir)
}
