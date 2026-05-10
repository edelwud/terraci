package input

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
		Namespaces:      []string{"terraform", "audit"},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	data, err := json.Marshal(envelope)
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
