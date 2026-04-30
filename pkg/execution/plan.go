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

	for _, ref := range ir.JobRefs() {
		if ref.Job == nil {
			continue
		}
		switch ref.Kind {
		case pipeline.JobKindContributed:
			index.phaseJobs[ref.Job.Phase] = append(index.phaseJobs[ref.Job.Phase], ref.Job)
		case pipeline.JobKindPlan:
			index.planJobs[ref.Level] = append(index.planJobs[ref.Level], ref.Job)
		case pipeline.JobKindApply:
			index.applyJobs[ref.Level] = append(index.applyJobs[ref.Level], ref.Job)
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
