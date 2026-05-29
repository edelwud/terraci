package generate

import (
	"maps"

	"github.com/edelwud/terraci/pkg/config/overwrite"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

type jobProfile struct {
	runsOn      string
	container   *domainpkg.Container
	env         map[string]string
	ifExpr      string
	environment string
	stepsBefore []domainpkg.Step
	stepsAfter  []domainpkg.Step
}

func (s settings) jobProfile(jobType configpkg.JobOverwriteType) (jobProfile, error) {
	cfg := s.configOrDefault()
	profile := jobProfile{
		runsOn:    cfg.RunsOn,
		container: convertContainer(cfg.Container),
	}
	if profile.runsOn == "" {
		profile.runsOn = "ubuntu-latest"
	}

	if cfg.JobDefaults != nil {
		applyJobDefaults(&profile, cfg.JobDefaults)
	}

	err := overwrite.ApplyMatching(
		&profile,
		jobType,
		cfg.Overwrites,
		overwrite.ByKey(func(ow *configpkg.JobOverwrite) configpkg.JobOverwriteType { return ow.Type }),
		applyJobOverwrite,
	)
	if err != nil {
		return jobProfile{}, err
	}
	return profile, nil
}

func applyJobDefaults(profile *jobProfile, defaults *configpkg.JobDefaults) {
	if defaults.RunsOn != "" {
		profile.runsOn = defaults.RunsOn
	}
	if defaults.Container != nil {
		profile.container = convertContainer(defaults.Container)
	}
	mergeProfileEnv(profile, defaults.Env)
	if defaults.If != "" {
		profile.ifExpr = defaults.If
	}
	if defaults.Environment != "" {
		profile.environment = defaults.Environment
	}
	profile.stepsBefore = appendConfigSteps(profile.stepsBefore, defaults.StepsBefore)
	profile.stepsAfter = appendConfigSteps(profile.stepsAfter, defaults.StepsAfter)
}

func applyJobOverwrite(profile *jobProfile, ow *configpkg.JobOverwrite) {
	if ow.RunsOn != "" {
		profile.runsOn = ow.RunsOn
	}
	if ow.Container != nil {
		profile.container = convertContainer(ow.Container)
	}
	mergeProfileEnv(profile, ow.Env)
	if ow.If != "" {
		profile.ifExpr = ow.If
	}
	if ow.Environment != "" {
		profile.environment = ow.Environment
	}
	profile.stepsBefore = appendConfigSteps(profile.stepsBefore, ow.StepsBefore)
	profile.stepsAfter = appendConfigSteps(profile.stepsAfter, ow.StepsAfter)
}

func mergeProfileEnv(profile *jobProfile, env map[string]string) {
	if len(env) == 0 {
		return
	}
	if profile.env == nil {
		profile.env = make(map[string]string, len(env))
	}
	maps.Copy(profile.env, env)
}

func appendConfigSteps(steps []domainpkg.Step, configs []configpkg.ConfigStep) []domainpkg.Step {
	for _, step := range configs {
		steps = append(steps, convertConfigStep(step))
	}
	return steps
}

func convertContainer(image *configpkg.Image) *domainpkg.Container {
	if image == nil {
		return nil
	}
	return &domainpkg.Container{Image: image.Name}
}

func mergeJobEnv(base, overrides map[string]string) map[string]string {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	result := make(map[string]string, len(base)+len(overrides))
	maps.Copy(result, base)
	maps.Copy(result, overrides)
	return result
}

func convertConfigStep(step configpkg.ConfigStep) domainpkg.Step {
	return domainpkg.NewStep(domainpkg.StepOptions{
		Name: step.Name,
		Uses: step.Uses,
		With: step.With,
		Run:  step.Run,
		Env:  step.Env,
	})
}
