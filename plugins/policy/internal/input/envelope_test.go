package input

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestBuild_EnvelopeUsesTerraciContextAndRawPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planPath, []byte(`{"format_version":"1.0","resource_changes":[{"type":"aws_s3_bucket"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	envelope, err := Build(Request{
		PlanJSONPath:    planPath,
		PlanDisplayPath: "platform/prod/app/plan.json",
		ModulePath:      "platform/prod/app",
		Components:      map[string]string{"environment": "prod"},
		Namespaces:      policyengine.NewNamespaces([]string{"terraform", "audit"}),
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	value := envelope.OPAValue()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	terraci := got["terraci"].(map[string]any)
	module := terraci["module"].(map[string]any)
	plan := got["plan"].(map[string]any)
	if module["path"] != "platform/prod/app" {
		t.Fatalf("module.path = %v", module["path"])
	}
	if plan["resource_changes"] == nil {
		t.Fatal("plan.resource_changes missing")
	}
}

func TestBuild_InvalidPlanJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planPath, []byte(`{"resource_changes":`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(Request{PlanJSONPath: planPath})
	if err == nil {
		t.Fatal("Build() error = nil, want parse error")
	}
}

func TestPlanDocument_OPAValueIsDefensive(t *testing.T) {
	t.Parallel()

	plan, err := NewPlanDocument([]byte(`{"resource_changes":[{"type":"aws_s3_bucket"}]}`))
	if err != nil {
		t.Fatalf("NewPlanDocument() error = %v", err)
	}
	first := plan.OPAValue()
	first["resource_changes"] = "mutated"

	second := plan.OPAValue()
	if _, ok := second["resource_changes"].([]any); !ok {
		t.Fatalf("OPAValue leaked mutation: %#v", second["resource_changes"])
	}
}

func TestEnvelope_OPAValueIsDefensive(t *testing.T) {
	t.Parallel()

	plan, err := NewPlanDocument([]byte(`{"resource_changes":[]}`))
	if err != nil {
		t.Fatalf("NewPlanDocument() error = %v", err)
	}
	components := map[string]string{"environment": "prod"}
	namespaces := policyengine.NewNamespaces([]string{"terraform"})
	envelope := NewEnvelope(NewTerraCiContext(
		NewModuleContext("platform/prod/app", components),
		NewPolicyContext(namespaces),
		NewPlanContext("platform/prod/app/plan.json"),
	), plan)

	components["environment"] = "sandbox"
	namespaces[0] = "mutated"
	value := envelope.OPAValue()
	terraci := value["terraci"].(map[string]any)
	module := terraci["module"].(map[string]any)
	policy := terraci["policy"].(map[string]any)
	if module["components"].(map[string]any)["environment"] != "prod" {
		t.Fatalf("module components leaked mutation: %#v", module["components"])
	}
	if got := policy["namespaces"].([]string)[0]; got != "terraform" {
		t.Fatalf("namespaces leaked mutation: %q", got)
	}
}
