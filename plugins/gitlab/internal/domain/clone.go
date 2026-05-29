package domain

import "maps"

func cloneDefaultConfig(in *DefaultConfig) *DefaultConfig {
	if in == nil {
		return nil
	}
	return &DefaultConfig{Image: cloneImageConfig(in.Image)}
}

func cloneImageConfig(in *ImageConfig) *ImageConfig {
	if in == nil {
		return nil
	}
	return &ImageConfig{Name: in.Name, Entrypoint: append([]string(nil), in.Entrypoint...)}
}

func cloneWorkflow(in *Workflow) *Workflow {
	if in == nil {
		return nil
	}
	return &Workflow{rules: cloneRules(in.rules)}
}

func cloneCache(in *Cache) *Cache {
	if in == nil {
		return nil
	}
	return &Cache{Key: in.Key, Paths: append([]string(nil), in.Paths...), Policy: in.Policy}
}

func cloneNeeds(in []JobNeed) []JobNeed {
	return append([]JobNeed(nil), in...)
}

func cloneRules(in []Rule) []Rule {
	if len(in) == 0 {
		return nil
	}
	out := make([]Rule, len(in))
	for i, rule := range in {
		out[i] = rule
		out[i].Changes = append([]string(nil), rule.Changes...)
	}
	return out
}

func cloneArtifacts(in *Artifacts) *Artifacts {
	if in == nil {
		return nil
	}
	out := *in
	out.Paths = append([]string(nil), in.Paths...)
	if in.Reports != nil {
		out.Reports = &Reports{
			Terraform: append([]string(nil), in.Reports.Terraform...),
			JUnit:     append([]string(nil), in.Reports.JUnit...),
			Cobertura: append([]string(nil), in.Reports.Cobertura...),
		}
	}
	return &out
}

func cloneSecrets(in map[string]*Secret) map[string]*Secret {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]*Secret, len(in))
	for name, secret := range in {
		if secret == nil {
			continue
		}
		clone := *secret
		if secret.Vault != nil {
			clone.Vault = &VaultSecret{
				Path:  secret.Vault.Path,
				Field: secret.Vault.Field,
			}
			if secret.Vault.Engine != nil {
				clone.Vault.Engine = &VaultEngine{Name: secret.Vault.Engine.Name, Path: secret.Vault.Engine.Path}
			}
		}
		out[name] = &clone
	}
	return out
}

func cloneIDTokens(in map[string]*IDToken) map[string]*IDToken {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]*IDToken, len(in))
	for name, token := range in {
		if token == nil {
			continue
		}
		clone := *token
		out[name] = &clone
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
