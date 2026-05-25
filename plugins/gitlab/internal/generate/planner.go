package generate

import (
	"strings"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type stagePlan struct {
	stages     []string
	stageByJob map[string]string
}

type stagePlanner struct {
	settings settings
}

func newStagePlanner(settings settings) stagePlanner {
	return stagePlanner{settings: settings}
}

func (p stagePlanner) plan(ir *pipeline.IR) (stagePlan, error) {
	groups, err := pipeline.Schedule(ir)
	if err != nil {
		return stagePlan{}, err
	}

	result := stagePlan{
		stages:     make([]string, 0, len(groups)),
		stageByJob: make(map[string]string),
	}
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		stage := p.gitlabStageName(group.Name)
		if _, ok := seen[stage]; !ok {
			seen[stage] = struct{}{}
			result.stages = append(result.stages, stage)
		}
		for _, job := range group.Jobs {
			if job != nil {
				result.stageByJob[job.Name()] = stage
			}
		}
	}
	return result, nil
}

func (p stagePlanner) gitlabStageName(groupName string) string {
	prefix := p.settings.stagesPrefix()
	if level, ok := strings.CutPrefix(groupName, "dag-level-"); ok {
		return prefix + "-" + level
	}
	return groupName
}
