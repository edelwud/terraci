package generate

import (
	"maps"

	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

func applyJobConfig(job *domain.Job, cfg configpkg.JobConfig) {
	if img := cfg.GetImage(); img != nil && img.Name != "" {
		job.Image = &domain.ImageConfig{
			Name:       img.Name,
			Entrypoint: img.Entrypoint,
		}
	}

	if tokens := cfg.GetIDTokens(); len(tokens) > 0 {
		job.IDTokens = make(map[string]*domain.IDToken)
		for name, token := range tokens {
			job.IDTokens[name] = &domain.IDToken{Aud: token.Aud}
		}
	}

	if secrets := cfg.GetSecrets(); len(secrets) > 0 {
		job.Secrets = convertSecrets(secrets)
	}

	if bs := cfg.GetBeforeScript(); len(bs) > 0 {
		job.BeforeScript = bs
	}
	if as := cfg.GetAfterScript(); len(as) > 0 {
		job.AfterScript = as
	}

	if artifacts := cfg.GetArtifacts(); artifacts != nil {
		job.Artifacts = convertArtifacts(artifacts)
	}

	if tags := cfg.GetTags(); len(tags) > 0 {
		job.Tags = tags
	}

	if rules := cfg.GetRules(); len(rules) > 0 {
		job.Rules = make([]domain.Rule, len(rules))
		for i, r := range rules {
			job.Rules[i] = domain.Rule{
				If:      r.If,
				When:    r.When,
				Changes: r.Changes,
			}
		}
	}

	if vars := cfg.GetVariables(); len(vars) > 0 {
		if job.Variables == nil {
			job.Variables = make(map[string]string)
		}
		maps.Copy(job.Variables, vars)
	}
}

func applyResolvedJobConfig(settings settings, job *domain.Job, jobType configpkg.JobOverwriteType) {
	applyJobDefaults(settings, job)
	applyOverwrites(settings, job, jobType)
}

func applyJobDefaults(settings settings, job *domain.Job) {
	defaults := settings.jobDefaults()
	if defaults == nil {
		return
	}
	applyJobConfig(job, defaults)
}

func applyOverwrites(settings settings, job *domain.Job, jobType configpkg.JobOverwriteType) {
	overwrites := settings.overwrites()
	for i := range overwrites {
		ow := &overwrites[i]
		if ow.Type != jobType {
			continue
		}
		applyJobConfig(job, ow)
	}
}

func convertSecrets(secrets map[string]configpkg.CfgSecret) map[string]*domain.Secret {
	result := make(map[string]*domain.Secret, len(secrets))
	for name, secret := range secrets {
		out := &domain.Secret{File: secret.File}
		if secret.Vault == nil {
			result[name] = out
			continue
		}

		if secret.Vault.Shorthand != "" {
			out.VaultPath = secret.Vault.Shorthand
			result[name] = out
			continue
		}

		out.Vault = &domain.VaultSecret{
			Path:  secret.Vault.Path,
			Field: secret.Vault.Field,
		}
		if secret.Vault.Engine != nil {
			out.Vault.Engine = &domain.VaultEngine{
				Name: secret.Vault.Engine.Name,
				Path: secret.Vault.Engine.Path,
			}
		}
		result[name] = out
	}

	return result
}

func convertArtifacts(cfg *configpkg.ArtifactsConfig) *domain.Artifacts {
	if cfg == nil {
		return nil
	}

	artifacts := &domain.Artifacts{
		Paths:     cfg.Paths,
		ExpireIn:  cfg.ExpireIn,
		Name:      cfg.Name,
		Untracked: cfg.Untracked,
		When:      cfg.When,
		ExposeAs:  cfg.ExposeAs,
	}
	if cfg.Reports != nil {
		artifacts.Reports = &domain.Reports{
			Terraform: cfg.Reports.Terraform,
			JUnit:     cfg.Reports.JUnit,
			Cobertura: cfg.Reports.Cobertura,
		}
	}

	return artifacts
}

func (g *Generator) generateWorkflow() *domain.Workflow {
	ruleConfigs := g.settings.workflowRules()
	if len(ruleConfigs) == 0 {
		return nil
	}

	rules := make([]domain.Rule, len(ruleConfigs))
	for i, r := range ruleConfigs {
		rules[i] = domain.Rule{
			If:      r.If,
			When:    r.When,
			Changes: r.Changes,
		}
	}

	return &domain.Workflow{Rules: rules}
}
