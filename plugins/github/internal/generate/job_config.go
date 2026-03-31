package generate

import (
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

func stepsBefore(cfg *configpkg.Config, jobType configpkg.JobOverwriteType) []domainpkg.Step {
	var steps []domainpkg.Step

	if cfg != nil && cfg.JobDefaults != nil {
		for _, step := range cfg.JobDefaults.StepsBefore {
			steps = append(steps, convertConfigStep(step))
		}
	}

	if cfg == nil {
		return steps
	}

	for _, overwrite := range cfg.Overwrites {
		if overwrite.Type != jobType {
			continue
		}
		for _, step := range overwrite.StepsBefore {
			steps = append(steps, convertConfigStep(step))
		}
	}

	return steps
}

func stepsAfter(cfg *configpkg.Config, jobType configpkg.JobOverwriteType) []domainpkg.Step {
	var steps []domainpkg.Step

	if cfg != nil && cfg.JobDefaults != nil {
		for _, step := range cfg.JobDefaults.StepsAfter {
			steps = append(steps, convertConfigStep(step))
		}
	}

	if cfg == nil {
		return steps
	}

	for _, overwrite := range cfg.Overwrites {
		if overwrite.Type != jobType {
			continue
		}
		for _, step := range overwrite.StepsAfter {
			steps = append(steps, convertConfigStep(step))
		}
	}

	return steps
}

func convertConfigStep(step configpkg.ConfigStep) domainpkg.Step {
	return domainpkg.Step{
		Name: step.Name,
		Uses: step.Uses,
		With: step.With,
		Run:  step.Run,
		Env:  step.Env,
	}
}
