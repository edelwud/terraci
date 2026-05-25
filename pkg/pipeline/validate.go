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

	byName := make(map[string]*Job, len(ir.jobs))
	for i := range ir.jobs {
		job := &ir.jobs[i]
		if _, exists := byName[job.name]; exists {
			return fmt.Errorf("pipeline IR contains duplicate job name %q", job.name)
		}
		if err := validateJob(job); err != nil {
			return err
		}
		byName[job.name] = job
	}

	if err := validateJobEdges(ir.jobs, byName); err != nil {
		return err
	}
	if err := validateResourceClosure(ir, byName); err != nil {
		return err
	}

	return validateAcyclicJobs(ir.jobs, byName)
}

func validateJob(job *Job) error {
	if job.name == "" {
		return errors.New("pipeline IR contains unnamed job")
	}
	if !job.kind.valid() {
		return fmt.Errorf("pipeline job %q has invalid kind %q", job.name, job.kind)
	}
	if err := validateOperation(job); err != nil {
		return err
	}
	if err := validateArtifact(job, job.outputArtifact); err != nil {
		return err
	}
	for _, input := range job.inputArtifacts {
		if err := validateInputArtifact(job, input); err != nil {
			return err
		}
	}
	for _, resource := range job.produces {
		if err := validateResource(job, resource, "produces"); err != nil {
			return err
		}
	}
	for _, resource := range job.consumes {
		if err := validateResource(job, resource, "consumes"); err != nil {
			return err
		}
	}
	return nil
}

func validateJobEdges(jobs []Job, byName map[string]*Job) error {
	for i := range jobs {
		job := &jobs[i]
		dependencies := make(map[string]struct{}, len(job.dependencies))
		for _, dep := range job.dependencies {
			if dep.Job == "" {
				return fmt.Errorf("pipeline job %q has empty dependency", job.name)
			}
			if byName[dep.Job] == nil {
				return fmt.Errorf("pipeline job %q depends on unknown job %q", job.name, dep.Job)
			}
			dependencies[dep.Job] = struct{}{}
		}
		for _, input := range job.inputArtifacts {
			if !input.Configured() {
				continue
			}
			if byName[input.ProducerJob] == nil {
				return fmt.Errorf("pipeline job %q restores artifact %q from unknown job %q", job.name, input.Artifact.Name, input.ProducerJob)
			}
			if _, ok := dependencies[input.ProducerJob]; !ok {
				return fmt.Errorf("pipeline job %q restores artifact %q from job %q without dependency", job.name, input.Artifact.Name, input.ProducerJob)
			}
			producer := byName[input.ProducerJob]
			if producer != nil && !sameArtifact(input.Artifact, producer.outputArtifact) {
				return fmt.Errorf("pipeline job %q restores artifact %q from job %q, want exact producer artifact %q", job.name, input.Artifact.Name, input.ProducerJob, producer.outputArtifact.Name)
			}
		}
	}
	return nil
}

func validateOperation(job *Job) error {
	switch job.kind {
	case JobKindPlan:
		if job.module == nil {
			return fmt.Errorf("pipeline plan job %q has no module", job.name)
		}
		if job.operation.typ != OperationTypeTerraformPlan || job.operation.terraform == nil {
			return fmt.Errorf("pipeline plan job %q must carry terraform plan operation", job.name)
		}
	case JobKindApply:
		if job.module == nil {
			return fmt.Errorf("pipeline apply job %q has no module", job.name)
		}
		if job.operation.typ != OperationTypeTerraformApply || job.operation.terraform == nil {
			return fmt.Errorf("pipeline apply job %q must carry terraform apply operation", job.name)
		}
	case JobKindCommand:
		if job.operation.typ != OperationTypeCommands {
			return fmt.Errorf("pipeline command job %q must carry command operation", job.name)
		}
	}
	return nil
}

func validateArtifact(job *Job, artifact Artifact) error {
	if artifact.Name == "" && len(artifact.Paths) == 0 {
		return nil
	}
	if artifact.Name == "" {
		return fmt.Errorf("pipeline job %q has artifact paths without artifact name", job.name)
	}
	if len(artifact.Paths) == 0 {
		return fmt.Errorf("pipeline job %q has artifact %q without paths", job.name, artifact.Name)
	}
	if slices.Contains(artifact.Paths, "") {
		return fmt.Errorf("pipeline job %q has artifact %q with empty path", job.name, artifact.Name)
	}
	for _, path := range artifact.Paths {
		if err := ValidateWorkspacePath(path); err != nil {
			return fmt.Errorf("pipeline job %q has artifact %q with invalid path: %w", job.name, artifact.Name, err)
		}
	}
	return nil
}

func validateInputArtifact(job *Job, input InputArtifact) error {
	if input.Artifact.Name == "" && len(input.Artifact.Paths) == 0 && input.ProducerJob == "" {
		return nil
	}
	if input.ProducerJob == "" {
		return fmt.Errorf("pipeline job %q has input artifact without producer job", job.name)
	}
	return validateArtifact(job, input.Artifact)
}

func validateResource(job *Job, resource ResourceSpec, direction string) error {
	if resource.Ref.Kind == "" {
		return fmt.Errorf("pipeline job %q %s resource without kind", job.name, direction)
	}
	if err := validateResourceRef(resource.Ref); err != nil {
		return fmt.Errorf("pipeline job %q %s invalid resource: %w", job.name, direction, err)
	}
	if resource.Path == "" {
		return fmt.Errorf("pipeline job %q %s %s without path", job.name, direction, resource.Ref.Kind)
	}
	if err := ValidateWorkspacePath(resource.Path); err != nil {
		return fmt.Errorf("pipeline job %q %s %s with invalid path: %w", job.name, direction, resource.Ref.Kind, err)
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

	for i := range ir.jobs {
		job := &ir.jobs[i]
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
	if len(job.produces) == 0 {
		return nil
	}
	artifactPaths := make(map[string]struct{}, len(job.outputArtifact.Paths))
	for _, path := range job.outputArtifact.Paths {
		artifactPaths[path] = struct{}{}
	}
	for _, resource := range job.produces {
		if _, ok := artifactPaths[resource.Path]; !ok {
			return fmt.Errorf("pipeline job %q produces %s at %q missing from output artifact %q", job.name, resource.Ref.Kind, resource.Path, job.outputArtifact.Name)
		}
	}
	return nil
}

func validateConsumedResources(job *Job, byName map[string]*Job, resources *resourceIndex) error {
	if len(job.consumes) == 0 {
		return nil
	}

	dependencies := make(map[string]struct{}, len(job.dependencies))
	for _, dep := range job.dependencies {
		dependencies[dep.Job] = struct{}{}
	}
	inputsByProducer := make(map[string]InputArtifact, len(job.inputArtifacts))
	for _, input := range job.inputArtifacts {
		if input.Configured() {
			inputsByProducer[input.ProducerJob] = input
		}
	}

	for _, consume := range job.consumes {
		produced, ok := resources.byRef[consume.Ref]
		if !ok {
			return fmt.Errorf("pipeline job %q consumes unavailable %s", job.name, resourceRefLabel(consume.Ref))
		}
		if produced.spec.Path != consume.Path {
			return fmt.Errorf("pipeline job %q consumes %s at %q, producer %q writes %q", job.name, resourceRefLabel(consume.Ref), consume.Path, produced.jobName, produced.spec.Path)
		}
		if produced.jobName == job.name {
			continue
		}
		producer := byName[produced.jobName]
		if producer == nil {
			return fmt.Errorf("pipeline job %q consumes %s from unknown job %q", job.name, resourceRefLabel(consume.Ref), produced.jobName)
		}
		if _, depOK := dependencies[produced.jobName]; !depOK {
			return fmt.Errorf("pipeline job %q consumes %s from job %q without dependency", job.name, resourceRefLabel(consume.Ref), produced.jobName)
		}
		input, ok := inputsByProducer[produced.jobName]
		if !ok {
			return fmt.Errorf("pipeline job %q consumes %s from job %q without input artifact", job.name, resourceRefLabel(consume.Ref), produced.jobName)
		}
		if !sameArtifact(input.Artifact, producer.outputArtifact) {
			return fmt.Errorf("pipeline job %q consumes %s from job %q with input artifact %q, want %q", job.name, resourceRefLabel(consume.Ref), produced.jobName, input.Artifact.Name, producer.outputArtifact.Name)
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
		switch state[job.name] {
		case visiting:
			return fmt.Errorf("pipeline IR contains dependency cycle at job %q", job.name)
		case visited:
			return nil
		}

		state[job.name] = visiting
		for _, dep := range job.dependencies {
			if err := visit(byName[dep.Job]); err != nil {
				return err
			}
		}
		state[job.name] = visited
		return nil
	}

	for i := range jobs {
		if err := visit(&jobs[i]); err != nil {
			return err
		}
	}
	return nil
}
