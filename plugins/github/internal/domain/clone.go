package domain

import "maps"

func cloneWorkflowTrigger(in WorkflowTrigger) WorkflowTrigger {
	out := in
	if in.Push != nil {
		out.Push = &PushTrigger{Branches: append([]string(nil), in.Push.Branches...)}
	}
	if in.PullRequest != nil {
		out.PullRequest = &PRTrigger{Branches: append([]string(nil), in.PullRequest.Branches...)}
	}
	return out
}

func cloneConcurrency(in *Concurrency) *Concurrency {
	if in == nil {
		return nil
	}
	return &Concurrency{Group: in.Group, CancelInProgress: in.CancelInProgress}
}

func cloneContainer(in *Container) *Container {
	if in == nil {
		return nil
	}
	return &Container{Image: in.Image, Env: cloneStringMap(in.Env)}
}

func cloneSteps(in []Step) []Step {
	if len(in) == 0 {
		return nil
	}
	out := make([]Step, len(in))
	for i, step := range in {
		out[i] = step.clone()
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
