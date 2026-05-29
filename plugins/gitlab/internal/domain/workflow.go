package domain

// Workflow controls when pipelines are created.
type Workflow struct {
	rules []Rule
}

func NewWorkflowRules(rules []Rule) *Workflow {
	if len(rules) == 0 {
		return nil
	}
	return &Workflow{rules: cloneRules(rules)}
}

func (w *Workflow) Rules() []Rule {
	if w == nil {
		return nil
	}
	return cloneRules(w.rules)
}
