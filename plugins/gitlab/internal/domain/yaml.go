package domain

import "go.yaml.in/yaml/v4"

// MarshalYAML emits the short string form when entrypoint is empty.
func (img ImageConfig) MarshalYAML() (any, error) {
	if len(img.Entrypoint) == 0 {
		return img.Name, nil
	}

	type imageAlias ImageConfig
	return imageAlias(img), nil
}

// MarshalYAML emits the short vault syntax when configured.
func (s Secret) MarshalYAML() (any, error) {
	if s.VaultPath != "" {
		type secretShorthand struct {
			Vault string `yaml:"vault"`
			File  bool   `yaml:"file,omitempty"`
		}

		return secretShorthand{
			Vault: s.VaultPath,
			File:  s.File,
		}, nil
	}

	type secretAlias Secret
	return secretAlias(s), nil
}

func (w *Workflow) MarshalYAML() (any, error) {
	return struct {
		Rules []Rule `yaml:"rules,omitempty"`
	}{
		Rules: cloneRules(w.rules),
	}, nil
}

// ToYAML converts the pipeline to YAML.
func (p *Pipeline) ToYAML() ([]byte, error) {
	result := make(map[string]any)
	if p == nil {
		result["stages"] = []string(nil)
		return yaml.Marshal(result)
	}
	result["stages"] = p.stages

	if len(p.variables) > 0 {
		result["variables"] = p.variables
	}

	if p.defaults != nil {
		result["default"] = p.defaults
	}

	if p.workflow != nil {
		result["workflow"] = p.workflow
	}

	for _, name := range p.JobNames() {
		result[name] = p.jobs[name]
	}

	return yaml.Marshal(result)
}

func (j Job) MarshalYAML() (any, error) {
	return struct {
		Stage         string              `yaml:"stage"`
		Image         *ImageConfig        `yaml:"image,omitempty"`
		Script        []string            `yaml:"script"`
		BeforeScript  []string            `yaml:"before_script,omitempty"`
		AfterScript   []string            `yaml:"after_script,omitempty"`
		Variables     map[string]string   `yaml:"variables,omitempty"`
		Needs         []JobNeed           `yaml:"needs,omitempty"`
		Rules         []Rule              `yaml:"rules,omitempty"`
		Artifacts     *Artifacts          `yaml:"artifacts,omitempty"`
		Cache         *Cache              `yaml:"cache,omitempty"`
		Secrets       map[string]*Secret  `yaml:"secrets,omitempty"`
		IDTokens      map[string]*IDToken `yaml:"id_tokens,omitempty"`
		When          string              `yaml:"when,omitempty"`
		AllowFailure  bool                `yaml:"allow_failure,omitempty"`
		Tags          []string            `yaml:"tags,omitempty"`
		ResourceGroup string              `yaml:"resource_group,omitempty"`
	}{
		Stage:         j.stage,
		Image:         cloneImageConfig(j.image),
		Script:        append([]string(nil), j.script...),
		BeforeScript:  append([]string(nil), j.beforeScript...),
		AfterScript:   append([]string(nil), j.afterScript...),
		Variables:     cloneStringMap(j.variables),
		Needs:         cloneNeeds(j.needs),
		Rules:         cloneRules(j.rules),
		Artifacts:     cloneArtifacts(j.artifacts),
		Cache:         cloneCache(j.cache),
		Secrets:       cloneSecrets(j.secrets),
		IDTokens:      cloneIDTokens(j.idTokens),
		When:          j.when,
		AllowFailure:  j.allowFailure,
		Tags:          append([]string(nil), j.tags...),
		ResourceGroup: j.resourceGroup,
	}, nil
}
