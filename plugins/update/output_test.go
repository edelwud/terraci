package update

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func TestOutputResult_JSON(t *testing.T) {
	result := &updateengine.UpdateResult{
		Modules: []updateengine.ModuleVersionUpdate{
			{
				ModulePath:    "platform/prod/vpc",
				CallName:      "vpc",
				Source:        "terraform-aws-modules/vpc/aws",
				LatestVersion: "5.2.0",
			},
		},
		Summary: updateengine.UpdateSummary{TotalChecked: 1},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, "json", result)
	if err != nil {
		t.Fatalf("outputResult(json) error = %v", err)
	}

	var parsed updateengine.UpdateResult
	if jsonErr := json.Unmarshal(buf.Bytes(), &parsed); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", jsonErr, buf.String())
	}
	if len(parsed.Modules) != 1 {
		t.Errorf("modules = %d, want 1", len(parsed.Modules))
	}
	if parsed.Summary.TotalChecked != 1 {
		t.Errorf("TotalChecked = %d, want 1", parsed.Summary.TotalChecked)
	}
}

func TestOutputResult_Text(t *testing.T) {
	result := &updateengine.UpdateResult{
		Providers: []updateengine.ProviderVersionUpdate{
			{
				ModulePath:      "test",
				ProviderName:    "aws",
				ProviderSource:  "hashicorp/aws",
				BumpedVersion:   "5.3.0",
				UpdateAvailable: true,
			},
		},
		Summary: updateengine.UpdateSummary{TotalChecked: 1, UpdatesAvailable: 1},
	}

	output := captureUpdateTextOutput(t, func() {
		if err := outputResult(&bytes.Buffer{}, "text", result); err != nil {
			t.Fatalf("outputResult(text) error = %v", err)
		}
	})
	if !strings.Contains(output, "summary") {
		t.Fatalf("output = %q, want summary block", output)
	}
	if !strings.Contains(output, "updates available") {
		t.Fatalf("output = %q, want summary line", output)
	}
}

func TestOutputResult_TextNoUpdates(t *testing.T) {
	result := &updateengine.UpdateResult{
		Summary: updateengine.UpdateSummary{TotalChecked: 3},
	}

	output := captureUpdateTextOutput(t, func() {
		if err := outputResult(&bytes.Buffer{}, "text", result); err != nil {
			t.Fatalf("outputResult(text) error = %v", err)
		}
	})
	if !strings.Contains(output, "summary") {
		t.Fatalf("output = %q, want summary block", output)
	}
	if !strings.Contains(output, "all dependencies are up to date") {
		t.Fatalf("output = %q, want no-updates message", output)
	}
}

func TestOutputResult_TextWithModuleUpdates(t *testing.T) {
	result := &updateengine.UpdateResult{
		Modules: []updateengine.ModuleVersionUpdate{
			{
				ModulePath:      "platform/prod/vpc",
				CallName:        "vpc",
				Source:          "terraform-aws-modules/vpc/aws",
				Constraint:      "~> 5.0",
				CurrentVersion:  "5.0.0",
				BumpedVersion:   "5.2.0",
				LatestVersion:   "6.0.0",
				UpdateAvailable: true,
			},
			{
				ModulePath:      "platform/prod/vpc",
				CallName:        "eks",
				Source:          "terraform-aws-modules/eks/aws",
				Constraint:      "~> 20.0",
				CurrentVersion:  "20.0.0",
				BumpedVersion:   "20.1.0",
				UpdateAvailable: true,
			},
		},
		Providers: []updateengine.ProviderVersionUpdate{
			{
				ModulePath:      "platform/prod/vpc",
				ProviderName:    "aws",
				ProviderSource:  "hashicorp/aws",
				Constraint:      "~> 5.0",
				CurrentVersion:  "5.67.0",
				BumpedVersion:   "5.69.0",
				LatestVersion:   "6.0.0",
				UpdateAvailable: true,
			},
		},
		Summary: updateengine.UpdateSummary{TotalChecked: 3, UpdatesAvailable: 3, Skipped: 1},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, "text", result)
	if err != nil {
		t.Fatalf("outputResult(text) error = %v", err)
	}
}

func TestOutputResult_TextSkippedOnly(t *testing.T) {
	// All items skipped or not updated — should print "all up to date"
	result := &updateengine.UpdateResult{
		Providers: []updateengine.ProviderVersionUpdate{
			{ModulePath: "test", Skipped: true, SkipReason: "ignored"},
		},
		Modules: []updateengine.ModuleVersionUpdate{
			{ModulePath: "test"},
		},
		Summary: updateengine.UpdateSummary{TotalChecked: 2, Skipped: 1},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, "text", result)
	if err != nil {
		t.Fatalf("outputResult(text) error = %v", err)
	}
}

func TestOutputResult_TextModuleWithSameBumpedLatest(t *testing.T) {
	// When LatestVersion == BumpedVersion, the "latest" field should be omitted from log.
	result := &updateengine.UpdateResult{
		Modules: []updateengine.ModuleVersionUpdate{
			{
				ModulePath:      "platform/prod/vpc",
				CallName:        "vpc",
				Source:          "terraform-aws-modules/vpc/aws",
				Constraint:      "~> 5.0",
				CurrentVersion:  "5.0.0",
				BumpedVersion:   "5.2.0",
				LatestVersion:   "5.2.0",
				UpdateAvailable: true,
			},
		},
		Summary: updateengine.UpdateSummary{TotalChecked: 1, UpdatesAvailable: 1},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, "text", result)
	if err != nil {
		t.Fatalf("outputResult(text) error = %v", err)
	}
}

func TestOutputResult_TextProviderWithSameBumpedLatest(t *testing.T) {
	result := &updateengine.UpdateResult{
		Providers: []updateengine.ProviderVersionUpdate{
			{
				ModulePath:      "test",
				ProviderName:    "aws",
				ProviderSource:  "hashicorp/aws",
				Constraint:      "~> 5.0",
				CurrentVersion:  "5.0.0",
				BumpedVersion:   "5.69.0",
				LatestVersion:   "5.69.0",
				UpdateAvailable: true,
			},
		},
		Summary: updateengine.UpdateSummary{TotalChecked: 1, UpdatesAvailable: 1},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, "text", result)
	if err != nil {
		t.Fatalf("outputResult(text) error = %v", err)
	}
}

func TestFormatCurrent(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		resolved   string
		want       string
	}{
		{"both different", "~> 5.0", "5.84.0", "~> 5.0 (5.84.0)"},
		{"no resolved", "~> 5.0", "", "~> 5.0"},
		{"no constraint", "", "5.84.0", "5.84.0"},
		{"same value", "5.0.0", "5.0.0", "5.0.0"},
		{"both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCurrent(tt.constraint, tt.resolved)
			if got != tt.want {
				t.Errorf("formatCurrent(%q, %q) = %q, want %q", tt.constraint, tt.resolved, got, tt.want)
			}
		})
	}
}
