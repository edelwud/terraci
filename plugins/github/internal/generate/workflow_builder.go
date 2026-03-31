package generate

import domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"

type workflowBuilder struct {
	settings settings
}

func newWorkflowBuilder(settings settings) workflowBuilder {
	return workflowBuilder{settings: settings}
}

func (b workflowBuilder) baseWorkflow() *domainpkg.Workflow {
	return &domainpkg.Workflow{
		Name: "Terraform",
		On: domainpkg.WorkflowTrigger{
			Push:        &domainpkg.PushTrigger{Branches: []string{"main"}},
			PullRequest: &domainpkg.PRTrigger{Branches: []string{"main"}},
		},
		Permissions: b.settings.permissions(),
		Env:         b.settings.env(),
		Jobs:        make(map[string]*domainpkg.Job),
	}
}
