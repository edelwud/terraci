package summaryengine

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tfplan "github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestResolveLabels(t *testing.T) {
	workDir := string(filepath.Separator) + "work"
	parser := &fakePlanParser{
		plans: map[string]*tfplan.ParsedPlan{
			filepath.Join(workDir, "svc", "dev", "us", "vpc", pipeline.PlanJSONFilename): {
				Resources: []tfplan.ResourceChange{
					{Address: "aws_vpc.main", Type: "aws_vpc", Name: "main", Action: tfplan.ActionCreate},
					{Address: "aws_subnet.private", Type: "aws_subnet", Name: "private", Action: tfplan.ActionUpdate},
					{Address: "aws_iam_role.noop", Type: "aws_iam_role", Name: "noop", Action: tfplan.ActionNoOp},
				},
			},
		},
	}

	result := ResolveLabels(LabelRequest{
		WorkDir:  workDir,
		Segments: []string{"service", "environment", "region", "module"},
		Plans: []ci.PlanResult{
			{
				ModuleID:   "svc/dev/us/vpc",
				ModulePath: "svc/dev/us/vpc",
				Components: map[string]string{
					"service":     "svc",
					"environment": "dev",
					"region":      "us",
					"module":      "vpc",
				},
				Status: ci.PlanStatusChanges,
			},
			{
				ModuleID:   "svc/prod/eu/app",
				ModulePath: "svc/prod/eu/app",
				Components: map[string]string{
					"service":     "svc",
					"environment": "prod",
					"region":      "eu",
					"module":      "app",
				},
				Status: ci.PlanStatusNoChanges,
			},
			{
				ModuleID:   "svc/dev/us/db",
				ModulePath: "svc/dev/us/db",
				Components: map[string]string{
					"environment": "dev",
					"module":      "db",
				},
				Status: ci.PlanStatusFailed,
			},
		},
		Templates: []string{
			"terraform",
			"{environment}",
			"{module}",
			"resource:{resource_type}",
			"action:{resource_action}:{resource_name}",
			"{missing}",
			"",
			"terraform",
		},
		Parser: parser,
	})

	wantLabels := []string{
		"action:create:main",
		"action:update:private",
		"db",
		"dev",
		"resource:aws_subnet",
		"resource:aws_vpc",
		"terraform",
		"vpc",
	}
	if !reflect.DeepEqual(result.Labels, wantLabels) {
		t.Fatalf("Labels = %v, want %v", result.Labels, wantLabels)
	}
	if len(parser.paths) != 1 {
		t.Fatalf("expected resource parser to run once, got %d", len(parser.paths))
	}
	messages := result.Diagnostics.Messages()
	if len(messages) != 3 {
		t.Fatalf("Diagnostics = %v, want three warnings", messages)
	}
	assertDiagnosticContains(t, messages, "unresolved placeholders {missing}")
	assertDiagnosticContains(t, messages, "summary label is empty")
}

func TestResolveLabels_FallsBackToSegmentsForMissingComponents(t *testing.T) {
	result := ResolveLabels(LabelRequest{
		Segments:  []string{"service", "environment", "region", "module"},
		Plans:     []ci.PlanResult{{ModuleID: "svc/dev/us/vpc", ModulePath: "svc/dev/us/vpc", Status: ci.PlanStatusChanges}},
		Templates: []string{"{service}:{environment}:{module}"},
	})

	if got, want := strings.Join(result.Labels, ","), "svc:dev:vpc"; got != want {
		t.Fatalf("Labels = %v, want %s", result.Labels, want)
	}
}

func TestResolveLabels_ResourceParserErrorsAreWarnings(t *testing.T) {
	parser := &fakePlanParser{err: errors.New("bad json")}

	result := ResolveLabels(LabelRequest{
		WorkDir:   "/work",
		Plans:     []ci.PlanResult{{ModuleID: "svc/dev/us/vpc", ModulePath: "svc/dev/us/vpc", Status: ci.PlanStatusChanges}},
		Templates: []string{"resource:{resource_type}"},
		Parser:    parser,
	})

	if len(result.Labels) != 0 {
		t.Fatalf("Labels = %v, want none", result.Labels)
	}
	assertDiagnosticContains(t, result.Diagnostics.Messages(), "bad json")
}

func assertDiagnosticContains(t *testing.T, messages []string, want string) {
	t.Helper()
	for _, message := range messages {
		if strings.Contains(message, want) {
			return
		}
	}
	t.Fatalf("diagnostics %v do not contain %q", messages, want)
}

type fakePlanParser struct {
	plans map[string]*tfplan.ParsedPlan
	err   error
	paths []string
}

func (f *fakePlanParser) ParsePlan(path string) (*tfplan.ParsedPlan, error) {
	f.paths = append(f.paths, path)
	if f.err != nil {
		return nil, f.err
	}
	if plan := f.plans[path]; plan != nil {
		return plan, nil
	}
	return &tfplan.ParsedPlan{}, nil
}
