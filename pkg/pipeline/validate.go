package pipeline

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Validate verifies that the IR contains a closed, addressable job graph.
func (ir *IR) Validate() error {
	if ir == nil {
		return errors.New("pipeline IR is nil")
	}

	byName := make(map[string]*Job, len(ir.Jobs))
	for i := range ir.Jobs {
		job := &ir.Jobs[i]
		if _, exists := byName[job.Name]; exists {
			return fmt.Errorf("pipeline IR contains duplicate job name %q", job.Name)
		}
		if err := validateJob(job); err != nil {
			return err
		}
		byName[job.Name] = job
	}

	if err := validateJobEdges(ir.Jobs, byName); err != nil {
		return err
	}
	if err := validateResourceClosure(ir, byName); err != nil {
		return err
	}

	return validateAcyclicJobs(ir.Jobs, byName)
}

func validateJob(job *Job) error {
	if job.Name == "" {
		return errors.New("pipeline IR contains unnamed job")
	}
	if !job.Kind.valid() {
		return fmt.Errorf("pipeline job %q has invalid kind %q", job.Name, job.Kind)
	}
	if err := validateOperation(job); err != nil {
		return err
	}
	if err := validateArtifact(job, job.OutputArtifact); err != nil {
		return err
	}
	for _, input := range job.InputArtifacts {
		if err := validateInputArtifact(job, input); err != nil {
			return err
		}
	}
	for _, resource := range job.Produces {
		if err := validateResource(job, resource, "produces"); err != nil {
			return err
		}
	}
	for _, resource := range job.Consumes {
		if err := validateResource(job, resource, "consumes"); err != nil {
			return err
		}
	}
	return nil
}

func validateJobEdges(jobs []Job, byName map[string]*Job) error {
	for i := range jobs {
		job := &jobs[i]
		dependencies := make(map[string]struct{}, len(job.Dependencies))
		for _, dep := range job.Dependencies {
			if dep.Job == "" {
				return fmt.Errorf("pipeline job %q has empty dependency", job.Name)
			}
			if byName[dep.Job] == nil {
				return fmt.Errorf("pipeline job %q depends on unknown job %q", job.Name, dep.Job)
			}
			dependencies[dep.Job] = struct{}{}
		}
		for _, input := range job.InputArtifacts {
			if !input.Configured() {
				continue
			}
			if byName[input.ProducerJob] == nil {
				return fmt.Errorf("pipeline job %q restores artifact %q from unknown job %q", job.Name, input.Artifact.Name, input.ProducerJob)
			}
			if _, ok := dependencies[input.ProducerJob]; !ok {
				return fmt.Errorf("pipeline job %q restores artifact %q from job %q without dependency", job.Name, input.Artifact.Name, input.ProducerJob)
			}
			producer := byName[input.ProducerJob]
			if producer != nil && !sameArtifact(input.Artifact, producer.OutputArtifact) {
				return fmt.Errorf("pipeline job %q restores artifact %q from job %q, want exact producer artifact %q", job.Name, input.Artifact.Name, input.ProducerJob, producer.OutputArtifact.Name)
			}
		}
	}
	return nil
}

func validateOperation(job *Job) error {
	switch job.Kind {
	case JobKindPlan:
		if job.Module == nil {
			return fmt.Errorf("pipeline plan job %q has no module", job.Name)
		}
		if job.Operation.Type != OperationTypeTerraformPlan || job.Operation.Terraform == nil {
			return fmt.Errorf("pipeline plan job %q must carry terraform plan operation", job.Name)
		}
	case JobKindApply:
		if job.Module == nil {
			return fmt.Errorf("pipeline apply job %q has no module", job.Name)
		}
		if job.Operation.Type != OperationTypeTerraformApply || job.Operation.Terraform == nil {
			return fmt.Errorf("pipeline apply job %q must carry terraform apply operation", job.Name)
		}
	case JobKindCommand:
		if job.Operation.Type != OperationTypeCommands {
			return fmt.Errorf("pipeline command job %q must carry command operation", job.Name)
		}
	}
	return nil
}

func validateArtifact(job *Job, artifact Artifact) error {
	if artifact.Name == "" && len(artifact.Paths) == 0 {
		return nil
	}
	if artifact.Name == "" {
		return fmt.Errorf("pipeline job %q has artifact paths without artifact name", job.Name)
	}
	if len(artifact.Paths) == 0 {
		return fmt.Errorf("pipeline job %q has artifact %q without paths", job.Name, artifact.Name)
	}
	if slices.Contains(artifact.Paths, "") {
		return fmt.Errorf("pipeline job %q has artifact %q with empty path", job.Name, artifact.Name)
	}
	for _, path := range artifact.Paths {
		if err := ValidateWorkspacePath(path); err != nil {
			return fmt.Errorf("pipeline job %q has artifact %q with invalid path: %w", job.Name, artifact.Name, err)
		}
	}
	return nil
}

func validateInputArtifact(job *Job, input InputArtifact) error {
	if input.Artifact.Name == "" && len(input.Artifact.Paths) == 0 && input.ProducerJob == "" {
		return nil
	}
	if input.ProducerJob == "" {
		return fmt.Errorf("pipeline job %q has input artifact without producer job", job.Name)
	}
	return validateArtifact(job, input.Artifact)
}

func validateResource(job *Job, resource ResourceSpec, direction string) error {
	if resource.Ref.Kind == "" {
		return fmt.Errorf("pipeline job %q %s resource without kind", job.Name, direction)
	}
	if err := validateResourceRef(resource.Ref); err != nil {
		return fmt.Errorf("pipeline job %q %s invalid resource: %w", job.Name, direction, err)
	}
	if resource.Path == "" {
		return fmt.Errorf("pipeline job %q %s %s without path", job.Name, direction, resource.Ref.Kind)
	}
	if err := ValidateWorkspacePath(resource.Path); err != nil {
		return fmt.Errorf("pipeline job %q %s %s with invalid path: %w", job.Name, direction, resource.Ref.Kind, err)
	}
	return nil
}

func validateResourceRef(ref ResourceRef) error {
	switch {
	case isPlanResourceKind(ref.Kind):
		if ref.ModulePath == "" {
			return fmt.Errorf("%s requires module path", ref.Kind)
		}
		if err := ValidateWorkspacePath(ref.ModulePath); err != nil {
			return fmt.Errorf("%s module path invalid: %w", ref.Kind, err)
		}
		if ref.Producer != "" {
			return fmt.Errorf("%s must not set producer", ref.Kind)
		}
	case isPluginResourceKind(ref.Kind):
		if ref.Producer == "" {
			return fmt.Errorf("%s requires producer", ref.Kind)
		}
		if err := validateProducerName(ref.Producer); err != nil {
			return fmt.Errorf("%s producer invalid: %w", ref.Kind, err)
		}
		if ref.ModulePath != "" {
			return fmt.Errorf("%s must not set module path", ref.Kind)
		}
	default:
		return fmt.Errorf("unknown resource kind %q", ref.Kind)
	}
	return nil
}

func validateProducerName(producer string) error {
	if strings.TrimSpace(producer) == "" {
		return errors.New("producer is required")
	}
	if strings.ContainsAny(producer, `/\`) {
		return fmt.Errorf("%q is not a safe artifact producer name", producer)
	}
	return nil
}

func validateResourceClosure(ir *IR, byName map[string]*Job) error {
	resources, err := buildResourceIndex(ir)
	if err != nil {
		return err
	}

	for i := range ir.Jobs {
		job := &ir.Jobs[i]
		if err := validateProducedResourceArtifacts(job); err != nil {
			return err
		}
		if err := validateConsumedResources(job, byName, resources); err != nil {
			return err
		}
	}
	return nil
}

func validateProducedResourceArtifacts(job *Job) error {
	if len(job.Produces) == 0 {
		return nil
	}
	artifactPaths := make(map[string]struct{}, len(job.OutputArtifact.Paths))
	for _, path := range job.OutputArtifact.Paths {
		artifactPaths[path] = struct{}{}
	}
	for _, resource := range job.Produces {
		if _, ok := artifactPaths[resource.Path]; !ok {
			return fmt.Errorf("pipeline job %q produces %s at %q missing from output artifact %q", job.Name, resource.Ref.Kind, resource.Path, job.OutputArtifact.Name)
		}
	}
	return nil
}

func validateConsumedResources(job *Job, byName map[string]*Job, resources *resourceIndex) error {
	if len(job.Consumes) == 0 {
		return nil
	}

	dependencies := make(map[string]struct{}, len(job.Dependencies))
	for _, dep := range job.Dependencies {
		dependencies[dep.Job] = struct{}{}
	}
	inputsByProducer := make(map[string]InputArtifact, len(job.InputArtifacts))
	for _, input := range job.InputArtifacts {
		if input.Configured() {
			inputsByProducer[input.ProducerJob] = input
		}
	}

	for _, consume := range job.Consumes {
		produced, ok := resources.byRef[consume.Ref]
		if !ok {
			return fmt.Errorf("pipeline job %q consumes unavailable %s", job.Name, resourceRefLabel(consume.Ref))
		}
		if produced.spec.Path != consume.Path {
			return fmt.Errorf("pipeline job %q consumes %s at %q, producer %q writes %q", job.Name, resourceRefLabel(consume.Ref), consume.Path, produced.jobName, produced.spec.Path)
		}
		if produced.jobName == job.Name {
			continue
		}
		producer := byName[produced.jobName]
		if producer == nil {
			return fmt.Errorf("pipeline job %q consumes %s from unknown job %q", job.Name, resourceRefLabel(consume.Ref), produced.jobName)
		}
		if _, depOK := dependencies[produced.jobName]; !depOK {
			return fmt.Errorf("pipeline job %q consumes %s from job %q without dependency", job.Name, resourceRefLabel(consume.Ref), produced.jobName)
		}
		input, ok := inputsByProducer[produced.jobName]
		if !ok {
			return fmt.Errorf("pipeline job %q consumes %s from job %q without input artifact", job.Name, resourceRefLabel(consume.Ref), produced.jobName)
		}
		if !sameArtifact(input.Artifact, producer.OutputArtifact) {
			return fmt.Errorf("pipeline job %q consumes %s from job %q with input artifact %q, want %q", job.Name, resourceRefLabel(consume.Ref), produced.jobName, input.Artifact.Name, producer.OutputArtifact.Name)
		}
	}
	return nil
}

func sameArtifact(a, b Artifact) bool {
	return a.Name == b.Name && slices.Equal(a.Paths, b.Paths)
}

func validateAcyclicJobs(jobs []Job, byName map[string]*Job) error {
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
			if err := visit(byName[dep.Job]); err != nil {
				return err
			}
		}
		state[job.Name] = visited
		return nil
	}

	for i := range jobs {
		if err := visit(&jobs[i]); err != nil {
			return err
		}
	}
	return nil
}
