package pipeline

import "fmt"

// JobGroup is a barriered set of jobs that may run in parallel.
type JobGroup struct {
	Name string
	Jobs []*Job
}

// Schedule groups an IR into deterministic topological execution barriers.
func Schedule(ir *IR) ([]JobGroup, error) {
	if ir == nil {
		return nil, nil
	}

	pending := make(map[string]*Job, len(ir.jobs))
	dependents := make(map[string][]string, len(ir.jobs))
	indegree := make(map[string]int, len(ir.jobs))
	order := make([]string, 0, len(ir.jobs))

	for i := range ir.jobs {
		job := &ir.jobs[i]
		name := job.name
		if _, exists := pending[name]; exists {
			return nil, fmt.Errorf("duplicate job name %q in schedule", name)
		}
		pending[name] = job
		indegree[name] = 0
		order = append(order, name)
	}

	for i := range ir.jobs {
		job := &ir.jobs[i]
		for _, dep := range job.dependencies {
			if pending[dep.Job] == nil {
				return nil, fmt.Errorf("job %q depends on unknown job %q", job.name, dep.Job)
			}
			dependents[dep.Job] = append(dependents[dep.Job], job.name)
			indegree[job.name]++
		}
	}

	var groups []JobGroup
	scheduled := 0
	for {
		layer := make([]*Job, 0)
		for _, name := range order {
			job := pending[name]
			if job == nil || indegree[name] != 0 {
				continue
			}
			layer = append(layer, job)
		}
		if len(layer) == 0 {
			break
		}

		groups = append(groups, JobGroup{
			Name: dagGroupName(len(groups)),
			Jobs: layer,
		})
		for _, job := range layer {
			delete(pending, job.name)
			scheduled++
			for _, dependent := range dependents[job.name] {
				indegree[dependent]--
			}
		}
	}

	if scheduled != len(order) {
		remaining := make([]string, 0, len(pending))
		for _, name := range order {
			if pending[name] != nil {
				remaining = append(remaining, name)
			}
		}
		return nil, fmt.Errorf("cycle or unresolvable dependency among jobs: %v", remaining)
	}

	return groups, nil
}

func dagGroupName(index int) string {
	return fmt.Sprintf("dag-level-%d", index)
}
