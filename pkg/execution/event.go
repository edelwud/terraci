package execution

import (
	"time"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// JobEvent is immutable context for an execution event.
type JobEvent struct {
	name      string
	kind      pipeline.JobKind
	moduleID  string
	operation pipeline.OperationType
	at        time.Time
}

// NewJobEvent constructs event context from an immutable pipeline job value.
func NewJobEvent(job pipeline.Job, at time.Time) JobEvent {
	moduleID := ""
	if module := job.Module(); module != nil {
		moduleID = module.ID()
	}
	return JobEvent{
		name:      job.Name(),
		kind:      job.Kind(),
		moduleID:  moduleID,
		operation: job.Operation().Type(),
		at:        at,
	}
}

// Name returns the execution job name.
func (e JobEvent) Name() string { return e.name }

// Kind returns the canonical pipeline job kind.
func (e JobEvent) Kind() pipeline.JobKind { return e.kind }

// ModuleID returns the TerraCi module ID for module jobs.
func (e JobEvent) ModuleID() string { return e.moduleID }

// Operation returns the operation type being executed.
func (e JobEvent) Operation() pipeline.OperationType { return e.operation }

// At returns the event timestamp.
func (e JobEvent) At() time.Time { return e.at }
