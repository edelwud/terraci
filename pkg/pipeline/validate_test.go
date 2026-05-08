package pipeline

import (
	"strings"
	"testing"
)

func TestIRValidateRejectsInvalidJobKind(t *testing.T) {
	t.Parallel()

	err := (&IR{Jobs: []Job{{Name: "job"}}}).Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid kind")
	}
	if !strings.Contains(err.Error(), `invalid kind ""`) {
		t.Fatalf("Validate() error = %q", err)
	}
}

func TestIRValidateRejectsInvalidResourceRefScope(t *testing.T) {
	t.Parallel()

	ir := &IR{Jobs: []Job{{
		Name: "summary",
		Kind: JobKindCommand,
		Operation: Operation{
			Type:     OperationTypeCommands,
			Commands: []string{"summary"},
		},
		OutputArtifact: ResultArtifact("summary", ".terraci/summary-report.json"),
		Produces: []ResourceSpec{{
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

	ir := &IR{Jobs: []Job{
		{
			Name:           "producer",
			Kind:           JobKindCommand,
			OutputArtifact: ResultArtifact("producer", ".terraci/report.json"),
			Operation: Operation{
				Type:     OperationTypeCommands,
				Commands: []string{"producer"},
			},
		},
		{
			Name: "consumer",
			Kind: JobKindCommand,
			InputArtifacts: []InputArtifact{{
				Artifact:    ResultArtifact("producer", ".terraci/report.json"),
				ProducerJob: "producer",
			}},
			Operation: Operation{
				Type:     OperationTypeCommands,
				Commands: []string{"consumer"},
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
	ir := &IR{Jobs: []Job{
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

	ir := &IR{Jobs: []Job{{
		Name: "policy",
		Kind: JobKindCommand,
		Operation: Operation{
			Type:     OperationTypeCommands,
			Commands: []string{"policy"},
		},
		Produces: []ResourceSpec{
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

	ir := &IR{Jobs: []Job{{
		Name:           "policy",
		Kind:           JobKindCommand,
		OutputArtifact: ResultArtifact("policy", ".terraci/other.json"),
		Operation: Operation{
			Type:     OperationTypeCommands,
			Commands: []string{"policy"},
		},
		Produces: []ResourceSpec{
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

	ir := &IR{Jobs: []Job{{
		Name: "summary",
		Kind: JobKindCommand,
		Operation: Operation{
			Type:     OperationTypeCommands,
			Commands: []string{"summary"},
		},
		Consumes: []ResourceSpec{
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
	ir := &IR{Jobs: []Job{
		testCommandJobWithResources("policy", []ResourceSpec{resource}),
		{
			Name:         "summary",
			Kind:         JobKindCommand,
			Dependencies: []JobDependency{{Job: "policy"}},
			Operation: Operation{
				Type:     OperationTypeCommands,
				Commands: []string{"summary"},
			},
			Consumes: []ResourceSpec{resource},
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
	ir := &IR{Jobs: []Job{
		testCommandJobWithResources("policy", []ResourceSpec{resource}),
		{
			Name:         "summary",
			Kind:         JobKindCommand,
			Dependencies: []JobDependency{{Job: "policy"}},
			InputArtifacts: []InputArtifact{{
				Artifact:    ResultArtifact("other", ".terraci/policy-report.json"),
				ProducerJob: "policy",
			}},
			Operation: Operation{
				Type:     OperationTypeCommands,
				Commands: []string{"summary"},
			},
			Consumes: []ResourceSpec{resource},
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
		Name:           name,
		Kind:           JobKindCommand,
		OutputArtifact: resultArtifactFromResources(name, produces),
		Produces:       produces,
		Operation: Operation{
			Type:     OperationTypeCommands,
			Commands: []string{name},
		},
	}
}
