package pipeline

import (
	"errors"
	"fmt"
	"slices"
)

// Validate verifies that the IR contains a closed, addressable job graph.
func (ir *IR) Validate() error {
	if ir == nil {
		return errors.New("pipeline IR is nil")
	}

	refs := ir.JobRefs()
	byName := make(map[string]*Job, len(refs))
	for _, ref := range refs {
		job := ref.Job
		if job == nil {
			return errors.New("pipeline IR contains nil job")
		}
		if job.Name == "" {
			return errors.New("pipeline IR contains unnamed job")
		}
		if _, exists := byName[job.Name]; exists {
			return fmt.Errorf("pipeline IR contains duplicate job name %q", job.Name)
		}
		if err := validateArtifact(job); err != nil {
			return err
		}
		byName[job.Name] = job
	}

	for _, ref := range refs {
		job := ref.Job
		for _, dep := range job.Dependencies {
			if dep == "" {
				return fmt.Errorf("pipeline job %q has empty dependency", job.Name)
			}
			if byName[dep] == nil {
				return fmt.Errorf("pipeline job %q depends on unknown job %q", job.Name, dep)
			}
		}
	}

	return validateAcyclicJobs(refs, byName)
}

func validateArtifact(job *Job) error {
	if job.Artifact.Name == "" && len(job.Artifact.Paths) == 0 {
		return nil
	}
	if job.Artifact.Name == "" {
		return fmt.Errorf("pipeline job %q has artifact paths without artifact name", job.Name)
	}
	if len(job.Artifact.Paths) == 0 {
		return fmt.Errorf("pipeline job %q has artifact %q without paths", job.Name, job.Artifact.Name)
	}
	if slices.Contains(job.Artifact.Paths, "") {
		return fmt.Errorf("pipeline job %q has artifact %q with empty path", job.Name, job.Artifact.Name)
	}
	return nil
}

func validateAcyclicJobs(refs []JobRef, byName map[string]*Job) error {
	const (
		visiting = 1
		visited  = 2
	)

	state := make(map[string]int, len(refs))
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

	for _, ref := range refs {
		if err := visit(ref.Job); err != nil {
			return err
		}
	}
	return nil
}
