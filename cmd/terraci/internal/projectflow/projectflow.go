// Package projectflow adapts runflow.Prepared to canonical workflow project planning.
package projectflow

import (
	"context"
	"errors"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/workflow"
)

var errPreparedRequired = errors.New("projectflow prepared state is required")

type Result = workflow.ProjectResult
type LibraryUsage = workflow.LibraryUsage
type LibrarySummary = workflow.LibrarySummary

// Runtime contains immutable command state needed to discover a Terraform project.
type Runtime struct {
	prepared *runflow.Prepared
}

// NewRuntime creates a project discovery runtime from prepared command state.
func NewRuntime(prepared *runflow.Prepared) Runtime {
	return Runtime{prepared: prepared}
}

// Request describes one project discovery request.
type Request struct {
	Filters       filter.Flags
	SelectTargets bool
	ChangedOnly   bool
	BaseRef       string
}

// Run adapts command prepared state to workflow.PlanProject.
func Run(ctx context.Context, runtime Runtime, req Request) (*Result, error) {
	if runtime.prepared == nil {
		return nil, errPreparedRequired
	}
	appCtx := runtime.prepared.AppContext()
	return workflow.PlanProject(ctx, workflow.ProjectRequest{
		WorkDir: runtime.prepared.WorkDir(),
		Config:  runtime.prepared.Config(),
		Filters: req.Filters,
		Targeting: workflow.TargetRequest{
			Enabled:     req.SelectTargets,
			ChangedOnly: req.ChangedOnly,
			BaseRef:     req.BaseRef,
			ChangeDetectorResolver: func() (workflow.ChangeDetector, error) {
				return appCtx.ChangeDetectorResolver().ResolveChangeDetector()
			},
		},
	})
}
