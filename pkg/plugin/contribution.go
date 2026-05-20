package plugin

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// PipelineContributor plugins add provider-independent jobs to the pipeline DAG.
type PipelineContributor interface {
	Plugin
	PipelineContribution(ctx *AppContext) (*pipeline.Contribution, error)
}

// PipelineContributionGate optionally controls whether an enabled plugin
// should contribute to the current pipeline.
type PipelineContributionGate interface {
	Plugin
	PipelineContributionEnabled(ctx *AppContext) (bool, error)
}

// PipelineContributionPhase identifies which contribution hook failed.
type PipelineContributionPhase string

const (
	PipelineContributionPhaseGate         PipelineContributionPhase = "gate"
	PipelineContributionPhaseContribution PipelineContributionPhase = "contribution"
)

// ErrNilPipelineContribution is returned when a contributor opts out by
// returning nil instead of using PipelineContributionGate.
var ErrNilPipelineContribution = errors.New("pipeline contribution is nil; use PipelineContributionGate to opt out")

// PipelineContributionError wraps failures from a plugin's pipeline
// contribution hooks with stable plugin and phase metadata.
type PipelineContributionError struct {
	Plugin string
	Phase  PipelineContributionPhase
	Err    error
}

func (e *PipelineContributionError) Error() string {
	if e == nil {
		return "pipeline contribution error"
	}
	subject := "pipeline contribution"
	if e.Plugin != "" {
		subject = fmt.Sprintf("pipeline contribution from plugin %q", e.Plugin)
	}
	if e.Phase != "" {
		subject += fmt.Sprintf(" during %s", e.Phase)
	}
	if e.Err != nil {
		return subject + ": " + e.Err.Error()
	}
	return subject + " failed"
}

func (e *PipelineContributionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
