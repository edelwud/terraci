package pipeline

import (
	"strings"
	"testing"
)

func TestIRValidateRejectsInvalidJobKind(t *testing.T) {
	t.Parallel()

	err := (&IR{jobs: []Job{{name: "job"}}}).Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid kind")
	}
	if !strings.Contains(err.Error(), `invalid kind ""`) {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsInvalidResourceRefScope(t *testing.T) {
	t.Parallel()

	ir := &IR{jobs: []Job{{
		name: "summary",
		kind: JobKindCommand,
		operation: Operation{
			typ:      OperationTypeCommands,
			commands: []string{"summary"},
		},
		outputArtifact: ResultArtifact("summary", ".terraci/summary-report.json"),
		produces: []ResourceSpec{{
			Ref:  ResourceRef{Kind: ResourceKindPluginReport, ModulePath: "svc/prod/eu/vpc"},
			Path: ".terraci/summary-report.json",
		}},
	}}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid resource ref")
	}
	if !strings.Contains(err.Error(), "plugin_report requires producer") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsInputArtifactWithoutDependency(t *testing.T) {
	t.Parallel()

	ir := &IR{jobs: []Job{
		{
			name:           "producer",
			kind:           JobKindCommand,
			outputArtifact: ResultArtifact("producer", ".terraci/report.json"),
			operation: Operation{
				typ:      OperationTypeCommands,
				commands: []string{"producer"},
			},
		},
		{
			name: "consumer",
			kind: JobKindCommand,
			inputArtifacts: []InputArtifact{{
				Artifact:    ResultArtifact("producer", ".terraci/report.json"),
				ProducerJob: "producer",
			}},
			operation: Operation{
				typ:      OperationTypeCommands,
				commands: []string{"consumer"},
			},
		},
	}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing dependency error")
	}
	if !strings.Contains(err.Error(), `without dependency`) {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsDuplicateProducedResource(t *testing.T) {
	t.Parallel()

	resource := PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json")
	ir := &IR{jobs: []Job{
		testCommandJobWithResources("policy-a", []ResourceSpec{resource}),
		testCommandJobWithResources("policy-b", []ResourceSpec{resource}),
	}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want duplicate resource error")
	}
	if !strings.Contains(err.Error(), "produced by both") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsProducedResourceWithoutOutputArtifact(t *testing.T) {
	t.Parallel()

	ir := &IR{jobs: []Job{{
		name: "policy",
		kind: JobKindCommand,
		operation: Operation{
			typ:      OperationTypeCommands,
			commands: []string{"policy"},
		},
		produces: []ResourceSpec{
			PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
		},
	}}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing artifact error")
	}
	if !strings.Contains(err.Error(), "without output artifact") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsProducedResourceMissingFromArtifact(t *testing.T) {
	t.Parallel()

	ir := &IR{jobs: []Job{{
		name:           "policy",
		kind:           JobKindCommand,
		outputArtifact: ResultArtifact("policy", ".terraci/other.json"),
		operation: Operation{
			typ:      OperationTypeCommands,
			commands: []string{"policy"},
		},
		produces: []ResourceSpec{
			PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
		},
	}}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want artifact path error")
	}
	if !strings.Contains(err.Error(), "missing from output artifact") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsConsumedResourceWithoutProducer(t *testing.T) {
	t.Parallel()

	ir := &IR{jobs: []Job{{
		name: "summary",
		kind: JobKindCommand,
		operation: Operation{
			typ:      OperationTypeCommands,
			commands: []string{"summary"},
		},
		consumes: []ResourceSpec{
			PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
		},
	}}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want unavailable resource error")
	}
	if !strings.Contains(err.Error(), "consumes unavailable plugin_report from producer \"policy\"") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsConsumedResourceWithoutInputArtifact(t *testing.T) {
	t.Parallel()

	resource := PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json")
	ir := &IR{jobs: []Job{
		testCommandJobWithResources("policy", []ResourceSpec{resource}),
		{
			name:         "summary",
			kind:         JobKindCommand,
			dependencies: []JobDependency{{Job: "policy"}},
			operation: Operation{
				typ:      OperationTypeCommands,
				commands: []string{"summary"},
			},
			consumes: []ResourceSpec{resource},
		},
	}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing input artifact error")
	}
	if !strings.Contains(err.Error(), "without input artifact") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsInputArtifactMismatch(t *testing.T) {
	t.Parallel()

	resource := PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json")
	ir := &IR{jobs: []Job{
		testCommandJobWithResources("policy", []ResourceSpec{resource}),
		{
			name:         "summary",
			kind:         JobKindCommand,
			dependencies: []JobDependency{{Job: "policy"}},
			inputArtifacts: []InputArtifact{{
				Artifact:    ResultArtifact("other", ".terraci/policy-report.json"),
				ProducerJob: "policy",
			}},
			operation: Operation{
				typ:      OperationTypeCommands,
				commands: []string{"summary"},
			},
			consumes: []ResourceSpec{resource},
		},
	}}

	err := ir.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want artifact mismatch error")
	}
	if !strings.Contains(err.Error(), "want exact producer artifact") {
		t.Fatalf("Validate() error = %q", err)
	}
}

func testCommandJobWithResources(name string, produces []ResourceSpec) Job {
	return Job{
		name:           name,
		kind:           JobKindCommand,
		outputArtifact: resultArtifactFromResources(name, produces),
		produces:       produces,
		operation: Operation{
			typ:      OperationTypeCommands,
			commands: []string{name},
		},
	}
}
