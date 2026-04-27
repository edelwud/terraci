package execution

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// Plan is an immutable executable view over Pipeline IR.
type Plan struct {
	IR    *pipeline.IR
	index JobIndex
}

// NewPlan wraps a pipeline IR into an execution plan.
func NewPlan(ir *pipeline.IR) *Plan {
	plan := &Plan{IR: ir}
	plan.index = newJobIndex(ir)
	return plan
}

// Validate verifies that the plan is a closed, addressable job graph.
func (p *Plan) Validate() error {
	if p == nil || p.IR == nil {
		return errors.New("execution plan is nil")
	}
	if err := p.IR.Validate(); err != nil {
		return fmt.Errorf("invalid execution plan: %w", err)
	}
	return nil
}

// JobsByPhase returns standalone contributed jobs for a phase.
func (p *Plan) JobsByPhase(phase pipeline.Phase) []*pipeline.Job {
	if p == nil || p.IR == nil {
		return nil
	}
	return p.index.jobsByPhase(phase)
}

// PlanJobsForLevel returns plan jobs for a level.
func (p *Plan) PlanJobsForLevel(levelIdx int) []*pipeline.Job {
	if p == nil || p.IR == nil {
		return nil
	}
	return p.index.planJobsForLevel(levelIdx)
}

// ApplyJobsForLevel returns apply jobs for a level.
func (p *Plan) ApplyJobsForLevel(levelIdx int) []*pipeline.Job {
	if p == nil || p.IR == nil {
		return nil
	}
	return p.index.applyJobsForLevel(levelIdx)
}

// JobIndex provides indexed access to execution jobs.
type JobIndex struct {
	phaseJobs map[pipeline.Phase][]*pipeline.Job
	planJobs  map[int][]*pipeline.Job
	applyJobs map[int][]*pipeline.Job
}

func newJobIndex(ir *pipeline.IR) JobIndex {
	index := JobIndex{
		phaseJobs: make(map[pipeline.Phase][]*pipeline.Job),
		planJobs:  make(map[int][]*pipeline.Job),
		applyJobs: make(map[int][]*pipeline.Job),
	}
	if ir == nil {
		return index
	}

	for i := range ir.Jobs {
		job := &ir.Jobs[i]
		index.phaseJobs[job.Phase] = append(index.phaseJobs[job.Phase], job)
	}
	for levelIdx := range ir.Levels {
		level := &ir.Levels[levelIdx]
		for moduleIdx := range level.Modules {
			moduleJobs := &level.Modules[moduleIdx]
			if moduleJobs.Plan != nil {
				index.planJobs[level.Index] = append(index.planJobs[level.Index], moduleJobs.Plan)
			}
			if moduleJobs.Apply != nil {
				index.applyJobs[level.Index] = append(index.applyJobs[level.Index], moduleJobs.Apply)
			}
		}
	}

	return index
}

func (i JobIndex) jobsByPhase(phase pipeline.Phase) []*pipeline.Job {
	return append([]*pipeline.Job(nil), i.phaseJobs[phase]...)
}

func (i JobIndex) planJobsForLevel(levelIdx int) []*pipeline.Job {
	return append([]*pipeline.Job(nil), i.planJobs[levelIdx]...)
}

func (i JobIndex) applyJobsForLevel(levelIdx int) []*pipeline.Job {
	return append([]*pipeline.Job(nil), i.applyJobs[levelIdx]...)
}
