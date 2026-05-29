package domain

type WorkflowTrigger struct {
	Push             *PushTrigger `yaml:"push,omitempty"`
	PullRequest      *PRTrigger   `yaml:"pull_request,omitempty"`
	WorkflowDispatch any          `yaml:"workflow_dispatch,omitempty"`
}

type PushTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
}

type PRTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
}

type Concurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type Container struct {
	Image string            `yaml:"image"`
	Env   map[string]string `yaml:"env,omitempty"`
}
