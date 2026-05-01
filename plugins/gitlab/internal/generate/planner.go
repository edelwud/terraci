package generate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type contributionIndex struct {
	hasJobs           bool
	nonFinalizeStages []string
	finalizeStages    []string
	stageByJob        map[string]string
}

func newContributionIndex(contributions []*pipeline.Contribution) contributionIndex {
	index := contributionIndex{stageByJob: make(map[string]string)}
	seen := make(map[string]bool)
	for _, contribution := range contributions {
		for _, job := range contribution.Jobs {
			index.recordJob(seen, job.Name, job.Phase)
		}
	}
	return index
}

// newContributionIndexFromIR derives the contribution index from a built IR.
// The IR's Jobs slice carries every contributed job already.
func newContributionIndexFromIR(ir *pipeline.IR) contributionIndex {
	index := contributionIndex{stageByJob: make(map[string]string)}
	if ir == nil {
		return index
	}
	seen := make(map[string]bool)
	for i := range ir.Jobs {
		job := &ir.Jobs[i]
		index.recordJob(seen, job.Name, job.Phase)
	}
	return index
}

func (i *contributionIndex) recordJob(seen map[string]bool, name string, phase pipeline.Phase) {
	i.hasJobs = true
	stage := phase.String()
	i.stageByJob[name] = stage
	if seen[stage] {
		return
	}
	seen[stage] = true
	if phase == pipeline.PhaseFinalize {
		i.finalizeStages = append(i.finalizeStages, stage)
		return
	}
	i.nonFinalizeStages = append(i.nonFinalizeStages, stage)
}

func (i *contributionIndex) stageFor(jobName string) string {
	return i.stageByJob[jobName]
}

func (i *contributionIndex) hasContributedJobs() bool {
	return i.hasJobs
}

type stagePlanner struct {
	settings      settings
	contributions contributionIndex
}

func newStagePlanner(settings settings, contributions contributionIndex) stagePlanner {
	return stagePlanner{
		settings:      settings,
		contributions: contributions,
	}
}

func (p stagePlanner) stages(ir *pipeline.IR) []string {
	stages := make([]string, 0)
	prefix := p.settings.stagesPrefix()

	for _, level := range ir.Levels {
		if p.settings.planEnabled() {
			stages = append(stages, fmt.Sprintf("%s-plan-%d", prefix, level.Index))
		}
		if !p.settings.planOnly() {
			stages = append(stages, fmt.Sprintf("%s-apply-%d", prefix, level.Index))
		}
	}

	if !p.contributions.hasContributedJobs() {
		return stages
	}

	return p.appendContributionStages(stages, prefix)
}

func (p stagePlanner) appendContributionStages(stages []string, prefix string) []string {
	if len(p.contributions.nonFinalizeStages) == 0 && len(p.contributions.finalizeStages) == 0 {
		return stages
	}

	lastPlanIdx := findLastPlanStage(stages, prefix)
	if lastPlanIdx == -1 {
		out := make([]string, 0, len(stages)+len(p.contributions.nonFinalizeStages)+len(p.contributions.finalizeStages))
		out = append(out, stages...)
		out = append(out, p.contributions.nonFinalizeStages...)
		out = append(out, p.contributions.finalizeStages...)
		return out
	}

	insertIdx := lastPlanIdx + 1
	result := make([]string, 0, len(stages)+len(p.contributions.nonFinalizeStages)+len(p.contributions.finalizeStages))
	result = append(result, stages[:insertIdx]...)
	result = append(result, p.contributions.nonFinalizeStages...)
	result = append(result, stages[insertIdx:]...)
	result = append(result, p.contributions.finalizeStages...)
	return result
}

func findLastPlanStage(stages []string, prefix string) int {
	lastPlanIdx := -1
	for i, stage := range stages {
		if strings.HasPrefix(stage, prefix+"-plan-") {
			lastPlanIdx = i
		}
	}
	return lastPlanIdx
}
