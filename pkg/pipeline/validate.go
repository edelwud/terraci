package pipeline

import (
	"errors"
	"fmt"
)

// Validate verifies that the IR contains a closed, addressable job graph.
func (ir *IR) Validate() error {
	if ir == nil {
		return errors.New("pipeline IR is nil")
	}

	jobs := ir.allJobs()
	byName := make(map[string]*Job, len(jobs))
	for _, job := range jobs {
		if job == nil {
			return errors.New("pipeline IR contains nil job")
		}
		if job.Name == "" {
			return errors.New("pipeline IR contains unnamed job")
		}
		if _, exists := byName[job.Name]; exists {
			return fmt.Errorf("pipeline IR contains duplicate job name %q", job.Name)
		}
		byName[job.Name] = job
	}

	for _, job := range jobs {
		for _, dep := range job.Dependencies {
			if dep == "" {
				return fmt.Errorf("pipeline job %q has empty dependency", job.Name)
			}
			if byName[dep] == nil {
				return fmt.Errorf("pipeline job %q depends on unknown job %q", job.Name, dep)
			}
		}
	}

	return validateAcyclicJobs(jobs, byName)
}

func (ir *IR) allJobs() []*Job {
	if ir == nil {
		return nil
	}

	jobs := make([]*Job, 0, len(ir.Jobs))
	for i := range ir.Jobs {
		jobs = append(jobs, &ir.Jobs[i])
	}
	for levelIdx := range ir.Levels {
		level := &ir.Levels[levelIdx]
		for moduleIdx := range level.Modules {
			moduleJobs := &level.Modules[moduleIdx]
			if moduleJobs.Plan != nil {
				jobs = append(jobs, moduleJobs.Plan)
			}
			if moduleJobs.Apply != nil {
				jobs = append(jobs, moduleJobs.Apply)
			}
		}
	}

	return jobs
}

func validateAcyclicJobs(jobs []*Job, byName map[string]*Job) error {
	const (
		visiting = 1
		visited  = 2
	)

	state := make(map[string]int, len(jobs))
	var visit func(*Job) error
	visit = func(job *Job) error {
		switch state[job.Name] {
		case visiting:
			return fmt.Errorf("pipeline IR contains dependency cycle at job %q", job.Name)
		case visited:
			return nil
		}

		state[job.Name] = visiting
		for _, dep := range job.Dependencies {
			if err := visit(byName[dep]); err != nil {
				return err
			}
		}
		state[job.Name] = visited
		return nil
	}

	for _, job := range jobs {
		if err := visit(job); err != nil {
			return err
		}
	}
	return nil
}
