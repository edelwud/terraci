package pipeline

import (
	"testing"

	"github.com/edelwud/terraci/pkg/terraformrun"
)

func TestIRTerraformRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ir        *IR
		wantFound bool
		want      string
		wantMixed bool
	}{
		{
			name: "nil IR",
		},
		{
			name:      "empty IR",
			ir:        &IR{},
			wantFound: false,
		},
		{
			name: "command only IR",
			ir: &IR{jobs: []Job{{
				kind:      JobKindCommand,
				operation: newCommandOperation([]string{"terraci summary"}),
			}}},
		},
		{
			name: "terraform jobs",
			ir: &IR{jobs: []Job{
				testTerraformRuntimeJob(terraformrun.BinaryTerraform),
				testTerraformRuntimeJob(terraformrun.BinaryTerraform),
			}},
			wantFound: true,
			want:      "terraform",
		},
		{
			name: "tofu jobs",
			ir: &IR{jobs: []Job{
				testTerraformRuntimeJob(terraformrun.BinaryTofu),
			}},
			wantFound: true,
			want:      "tofu",
		},
		{
			name: "mixed binaries",
			ir: &IR{jobs: []Job{
				testTerraformRuntimeJob(terraformrun.BinaryTerraform),
				testTerraformRuntimeJob(terraformrun.BinaryTofu),
			}},
			wantFound: true,
			want:      "terraform",
			wantMixed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, found := tt.ir.TerraformRuntime()
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if !found {
				return
			}
			if got.Binary() != tt.want {
				t.Fatalf("Binary() = %q, want %q", got.Binary(), tt.want)
			}
			if got.Mixed() != tt.wantMixed {
				t.Fatalf("Mixed() = %v, want %v", got.Mixed(), tt.wantMixed)
			}
		})
	}
}

func testTerraformRuntimeJob(binary terraformrun.Binary) Job {
	return Job{
		kind: JobKindPlan,
		operation: Operation{
			typ: OperationTypeTerraformPlan,
			terraform: &TerraformOperation{
				binary: binary,
				kind:   OperationTypeTerraformPlan,
			},
		},
	}
}
